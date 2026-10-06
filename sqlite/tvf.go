package sqlite

import (
	"strings"
)

// Table-valued functions: functions used as tables in FROM.
//
//	SELECT value FROM generate_series(1, 10, 2)
//	SELECT name FROM pragma_table_info('books')
//	SELECT key, value FROM json_each('{"a": 1}')
//
// Their arguments may use columns of the tables before them in the join,
// so they're worked out again for each of those rows:
// FROM sqlite_schema s, pragma_table_info(s.name).

type tvfPlan struct {
	name string
	args []expr
}

// tvfColumns are each table-valued function's columns.
var tvfColumns = map[string][]string{
	"generate_series":          {"value"},
	"json_each":                {"key", "value", "type", "atom", "id", "parent", "fullkey", "path"},
	"json_tree":                {"key", "value", "type", "atom", "id", "parent", "fullkey", "path"},
	"pragma_table_info":        {"cid", "name", "type", "notnull", "dflt_value", "pk"},
	"pragma_table_xinfo":       {"cid", "name", "type", "notnull", "dflt_value", "pk", "hidden"},
	"pragma_index_list":        {"seq", "name", "unique", "origin", "partial"},
	"pragma_index_info":        {"seqno", "cid", "name"},
	"pragma_index_xinfo":       {"seqno", "cid", "name", "desc", "coll", "key"},
	"pragma_foreign_key_list":  {"id", "seq", "table", "from", "to", "on_update", "on_delete", "match"},
	"pragma_foreign_key_check": {"table", "rowid", "parent", "fkid"},
	"pragma_table_list":        {"schema", "name", "type", "ncol", "wr", "strict"},
	"pragma_database_list":     {"seq", "name", "file"},
	"pragma_collation_list":    {"seq", "name"},
}

func (c *corePlan) tvfSource(f fromItem) *source {
	name := strings.ToLower(f.name)
	cols, ok := tvfColumns[name]
	if !ok {
		fail("SQL: no such table-valued function: %s", f.name)
	}
	src := &source{name: f.name, rowidCol: -1, hidden: map[string]bool{}, tvf: &tvfPlan{name: name, args: f.args}}
	if f.alias != "" {
		src.name = f.alias
	}
	for _, n := range cols {
		src.cols = append(src.cols, Column{Name: n})
	}
	for _, a := range f.args {
		c.resolve(a, false)
	}
	return src
}

// rows works the function out for the current rows of the sources
// before it.
func (t *tvfPlan) rows(r *runner) []*srcRow {
	args := make([]Value, len(t.args))
	for i, a := range t.args {
		args[i] = r.eval(a)
	}
	var vals [][]Value
	switch {
	case t.name == "generate_series":
		vals = generateSeries(args)
	case t.name == "json_each" || t.name == "json_tree":
		vals = jsonEachRows(t.name == "json_tree", args)
	default:
		pragma := strings.TrimPrefix(t.name, "pragma_")
		var arg Value
		if len(args) > 0 {
			arg = args[0]
		}
		_, vals = r.db.pragmaRows(pragma, arg, len(args) > 0)
	}
	out := make([]*srcRow, len(vals))
	for i, v := range vals {
		out[i] = &srcRow{rowid: int64(i + 1), vals: v}
	}
	return out
}

// generateSeries is generate_series(start [, stop [, step]]): the whole
// numbers from start to stop, step apart.
func generateSeries(args []Value) [][]Value {
	if len(args) == 0 || len(args) > 3 {
		fail("SQL: generate_series takes 1 to 3 values")
	}
	num := func(v Value, def int64) int64 {
		if v == nil {
			return def
		}
		n, ok := applyAffinity(v, affInteger).(int64)
		if !ok {
			n = integerValue(v)
		}
		return n
	}
	start := num(args[0], 0)
	stop := int64(4294967295)
	step := int64(1)
	if len(args) > 1 {
		stop = num(args[1], stop)
	}
	if len(args) > 2 {
		step = num(args[2], 1)
	}
	if step == 0 {
		step = 1
	}
	var out [][]Value
	if step > 0 {
		for v := start; v <= stop; v += step {
			out = append(out, []Value{v})
			if len(out) > 10000000 {
				fail("SQL: generate_series makes over 10 million rows")
			}
			if v > stop-step {
				break
			}
		}
	} else {
		for v := start; v >= stop; v += step {
			out = append(out, []Value{v})
			if len(out) > 10000000 {
				fail("SQL: generate_series makes over 10 million rows")
			}
			if v < stop-step {
				break
			}
		}
	}
	return out
}
