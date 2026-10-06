package sqlite

import (
	"encoding/binary"
	"strconv"
	"strings"
	"time"
)

// PRAGMA statements read and change settings and describe the schema:
//
//	PRAGMA table_info(books)        the columns
//	PRAGMA foreign_keys = ON        enforce REFERENCES
//	PRAGMA user_version = 3         a number for the program's own use
//
// The ones that describe the schema also work as tables in FROM:
// SELECT name FROM pragma_table_info('books').

type pragmaStmt struct {
	name  string
	arg   Value
	set   bool // PRAGMA name = value (vs PRAGMA name(value), which reads)
	hasAr bool
}

func (p *sqlParser) parsePragma() *pragmaStmt {
	s := &pragmaStmt{name: strings.ToLower(p.expectName())}
	if p.acceptOp(".") {
		s.name = strings.ToLower(p.expectName())
	}
	value := func() Value {
		neg := p.acceptOp("-")
		p.acceptOp("+")
		t := p.next()
		switch t.kind {
		case tNumber:
			v := parseNumberLiteral(t.text)
			if neg {
				return arith("-", int64(0), v)
			}
			return v
		case tString, tIdent:
			return t.text
		}
		fail("SQL: bad value for PRAGMA %s", s.name)
		return nil
	}
	switch {
	case p.acceptOp("="):
		s.arg, s.set, s.hasAr = value(), true, true
	case p.acceptOp("("):
		s.arg, s.hasAr = value(), true
		p.expectOp(")")
	}
	return s
}

// pragmaBool reads ON/OFF, TRUE/FALSE, YES/NO, 1/0.
func pragmaBool(v Value) bool {
	switch strings.ToLower(textValue(v)) {
	case "on", "true", "yes", "1":
		return true
	case "off", "false", "no", "0":
		return false
	}
	fail("SQL: expected ON or OFF, got %v", textValue(v))
	return false
}

// pragmaRows runs a pragma that reads; for the schema pragmas arg names
// the table or index.
func (f *dbFile) pragmaRows(name string, arg Value, hasArg bool) ([]string, [][]Value) {
	one := func(col string, v Value) ([]string, [][]Value) { return []string{col}, [][]Value{{v}} }
	tableArg := func() *Table {
		if !hasArg {
			fail("SQL: PRAGMA %s needs a table name", name)
		}
		t, err := f.table(textValue(arg))
		if err != nil {
			return nil
		}
		return t
	}
	h := f.mustPage(1)
	switch name {
	case "table_info", "table_xinfo":
		cols := []string{"cid", "name", "type", "notnull", "dflt_value", "pk"}
		if name == "table_xinfo" {
			cols = append(cols, "hidden")
		}
		var rows [][]Value
		if v, ok := f.views[strings.ToLower(textValue(arg))]; ok && hasArg {
			q := f.planView(v, nil)
			for i, n := range q.names {
				row := []Value{int64(i), n, "", int64(0), nil, int64(0)}
				if name == "table_xinfo" {
					row = append(row, int64(0))
				}
				rows = append(rows, row)
			}
			return cols, rows
		}
		t := tableArg()
		if t == nil {
			return cols, nil
		}
		for i, c := range t.Columns {
			hidden := int64(0)
			if c.Generated {
				if name == "table_info" {
					continue
				}
				hidden = 2
				if c.Stored {
					hidden = 3
				}
			}
			pk := int64(0)
			for k, ci := range t.PK {
				if ci == i {
					pk = int64(k + 1)
				}
			}
			var dflt Value
			if c.DefaultSQL != "" {
				dflt = c.DefaultSQL
			}
			row := []Value{int64(len(rows)), c.Name, c.Type, boolValue(c.NotNull), dflt, pk}
			if name == "table_xinfo" {
				row = append(row, hidden)
			}
			rows = append(rows, row)
		}
		return cols, rows
	case "index_list":
		cols := []string{"seq", "name", "unique", "origin", "partial"}
		t := tableArg()
		if t == nil {
			return cols, nil
		}
		var rows [][]Value
		for i := len(t.Indexes) - 1; i >= 0; i-- {
			ix := t.Indexes[i]
			origin := ix.Origin
			if origin == "" {
				origin = "c"
			}
			rows = append(rows, []Value{int64(len(rows)), ix.Name, boolValue(ix.Unique), origin, boolValue(ix.Partial != nil)})
		}
		return cols, rows
	case "index_info", "index_xinfo":
		cols := []string{"seqno", "cid", "name"}
		if name == "index_xinfo" {
			cols = append(cols, "desc", "coll", "key")
		}
		if !hasArg {
			fail("SQL: PRAGMA %s needs an index name", name)
		}
		ix, t := f.indexNamed(textValue(arg))
		if ix == nil {
			return cols, nil
		}
		var rows [][]Value
		for i, ic := range ix.Cols {
			var cname Value
			cid := int64(ic.Col)
			if ic.Col >= 0 {
				cname = t.Columns[ic.Col].Name
			} else {
				cid = -2
			}
			row := []Value{int64(i), cid, cname}
			if name == "index_xinfo" {
				coll := ic.Coll
				if coll == "" {
					coll = "BINARY"
				}
				row = append(row, boolValue(ic.Desc), coll, int64(1))
			}
			rows = append(rows, row)
		}
		if name == "index_xinfo" {
			rows = append(rows, []Value{int64(len(ix.Cols)), int64(-1), nil, int64(0), "BINARY", int64(0)})
		}
		return cols, rows
	case "foreign_key_list":
		cols := []string{"id", "seq", "table", "from", "to", "on_update", "on_delete", "match"}
		t := tableArg()
		if t == nil {
			return cols, nil
		}
		var rows [][]Value
		for i := len(t.FKs) - 1; i >= 0; i-- {
			fk := t.FKs[i]
			for k, ci := range fk.Cols {
				var to Value
				if k < len(fk.ParentCols) {
					to = fk.ParentCols[k]
				}
				rows = append(rows, []Value{int64(len(t.FKs) - 1 - i), int64(k), fk.Parent, t.Columns[ci].Name, to, fk.OnUpdate, fk.OnDelete, fk.Match})
			}
		}
		return cols, rows
	case "foreign_key_check":
		cols := []string{"table", "rowid", "parent", "fkid"}
		var tables []*Table
		if hasArg {
			if t := tableArg(); t != nil {
				tables = append(tables, t)
			} else {
				fail("SQL: no such table: %s", textValue(arg))
			}
		} else {
			for _, n := range f.names {
				tables = append(tables, f.tables[strings.ToLower(n)])
			}
		}
		var rows [][]Value
		for _, t := range tables {
			for _, v := range f.fkViolations(t) {
				rows = append(rows, v)
			}
		}
		return cols, rows
	case "integrity_check", "quick_check":
		db := &DB{dbFile: f}
		if err := db.checkLocked(); err != nil {
			return one(name, strings.TrimPrefix(err.Error(), "damaged database: "))
		}
		return one(name, "ok")
	case "table_list":
		cols := []string{"schema", "name", "type", "ncol", "wr", "strict"}
		var rows [][]Value
		for _, r := range f.schemaRows {
			switch r.kind {
			case "table":
				t := f.tables[strings.ToLower(r.name)]
				n, strict := int64(0), int64(0)
				if t != nil {
					n, strict = int64(len(t.Columns)), boolValue(t.Strict).(int64)
				}
				rows = append(rows, []Value{"main", r.name, "table", n, int64(0), strict})
			case "view":
				q := f.planView(&View{Name: r.name, SQL: r.sql}, nil)
				rows = append(rows, []Value{"main", r.name, "view", int64(len(q.names)), int64(0), int64(0)})
			}
		}
		rows = append(rows, []Value{"main", "sqlite_schema", "table", int64(5), int64(0), int64(0)})
		return cols, rows
	case "database_list":
		rows := [][]Value{{int64(0), "main", fileKey(f.path)}}
		for i, a := range f.attached {
			rows = append(rows, []Value{int64(i + 2), a.name, fileKey(a.db.path)})
		}
		return []string{"seq", "name", "file"}, rows
	case "collation_list":
		return []string{"seq", "name"}, [][]Value{{int64(0), "RTRIM"}, {int64(1), "NOCASE"}, {int64(2), "BINARY"}}
	case "foreign_keys":
		return one(name, boolValue(f.foreignKeys))
	case "defer_foreign_keys":
		return one(name, boolValue(f.deferFKs))
	case "user_version":
		return one(name, int64(int32(binary.BigEndian.Uint32(h[60:]))))
	case "application_id":
		return one(name, int64(int32(binary.BigEndian.Uint32(h[68:]))))
	case "schema_version":
		return one(name, int64(int32(binary.BigEndian.Uint32(h[40:]))))
	case "page_size":
		return one(name, int64(f.pageSize))
	case "page_count":
		return one(name, int64(f.pageCount))
	case "freelist_count":
		return one(name, int64(binary.BigEndian.Uint32(h[36:])))
	case "encoding":
		return one(name, []string{"", "UTF-8", "UTF-16le", "UTF-16be"}[f.encoding])
	case "journal_mode":
		return one(name, f.journalMode())
	case "busy_timeout", "timeout":
		return one("timeout", int64(f.timeout()/time.Millisecond))
	case "synchronous":
		return one(name, int64(f.syncLevel()))
	}
	fail("SQL: unknown PRAGMA %s", name)
	return nil, nil
}

func (f *dbFile) indexNamed(name string) (*Index, *Table) {
	for _, t := range f.tables {
		for _, ix := range t.Indexes {
			if strings.EqualFold(ix.Name, name) {
				return ix, t
			}
		}
	}
	return nil, nil
}

// runPragma runs a PRAGMA statement: one that changes a setting, or one
// that reads (giving rows).
func (db *DB) runPragma(s *pragmaStmt) ([]string, [][]Value, error) {
	f := db.dbFile
	if !s.set {
		done, err := f.beginRead()
		if err != nil {
			return nil, nil, err
		}
		defer done()
		cols, rows := f.pragmaRows(s.name, s.arg, s.hasAr)
		return cols, rows, nil
	}
	switch s.name {
	case "foreign_keys":
		if !f.inWrite { // inside a transaction it changes nothing, as in SQLite
			f.foreignKeys = pragmaBool(s.arg)
		}
	case "defer_foreign_keys":
		f.deferFKs = pragmaBool(s.arg)
	case "busy_timeout", "timeout":
		n, ok := applyAffinity(s.arg, affInteger).(int64)
		if !ok || n < 0 {
			fail("SQL: PRAGMA busy_timeout needs a number of milliseconds")
		}
		f.busy = time.Duration(n) * time.Millisecond
		f.busySet = true
	case "synchronous":
		switch strings.ToUpper(textValue(s.arg)) {
		case "OFF", "0":
			f.sync = 1
		case "NORMAL", "1":
			f.sync = 2
		case "FULL", "2":
			f.sync = 3
		case "EXTRA", "3":
			f.sync = 4
		default:
			fail("SQL: PRAGMA synchronous takes OFF, NORMAL, FULL or EXTRA")
		}
	case "user_version", "application_id":
		n, ok := applyAffinity(s.arg, affInteger).(int64)
		if !ok {
			fail("SQL: PRAGMA %s needs a whole number", s.name)
		}
		off := 60
		if s.name == "application_id" {
			off = 68
		}
		_, err := db.execOneRows(&headerWrite{off: off, v: uint32(int32(n))}, [][]Value{nil})
		return nil, nil, err
	case "journal_mode":
		mode, err := db.setJournalMode(strings.ToLower(textValue(s.arg)))
		if err != nil {
			return nil, nil, err
		}
		return []string{"journal_mode"}, [][]Value{{mode}}, nil
	default:
		fail("SQL: PRAGMA %s can't be changed", s.name)
	}
	cols, rows := f.pragmaRows(s.name, nil, false)
	return cols, rows, nil
}

// headerWrite is a statement that sets a 4-byte header field.
type headerWrite struct {
	off int
	v   uint32
}

func (f *dbFile) timeout() time.Duration {
	if f.busySet {
		return f.busy
	}
	return busyTimeout
}

// syncLevel is PRAGMA synchronous: 0 OFF, 1 NORMAL, 2 FULL, 3 EXTRA.
func (f *dbFile) syncLevel() int {
	if f.sync == 0 {
		return 2
	}
	return f.sync - 1
}

func itoa64(n int64) string { return strconv.FormatInt(n, 10) }
