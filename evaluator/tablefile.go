package evaluator

import (
	"encoding/csv"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"

	"Turtle/object"
)

// Table files: the one reader and writer behind data's table_read and
// table_write and sql's sql_load, sql_save, sql_update, sql_delete and
// sql_upsert. The file name's extension picks the format:
//
//	.csv (and anything else)   comma-separated values, RFC 4180
//	.tsv                       tab-separated values, quoted the same way
//	.txt                       the aligned table show table[x] prints (writing only)
//	.json                      a list of objects, one per row
//
// In .csv and .tsv files the first line names the columns. A value with
// the separator, a quote or a line break is put in quotes, with quotes
// doubled. none is an empty cell, and an empty cell reads back as none.
// Every value reads back as text.
//
// A .json file keeps values' kinds: numbers, true/false, null (none),
// text, and lists or objects inside a row. Dates are written as text.

func tableFormat(path string) string {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".tsv":
		return "tsv"
	case ".txt":
		return "txt"
	case ".json":
		return "json"
	}
	return "csv"
}

// writeTableFile writes t to path and returns how many rows it wrote. It
// replaces the file if it's there.
func (it *Interpreter) writeTableFile(fn, path string, t textTable) int {
	var data string
	switch tableFormat(path) {
	case "txt":
		data = t.render(-1) + "\n"
	case "json":
		list := &object.List{}
		for _, r := range t.rows {
			m := object.NewMap()
			for i, h := range t.header {
				var v object.Object = object.NoneValue
				if i < len(r) && r[i] != nil {
					v = r[i]
				}
				m.Put(&object.String{Value: h}, v)
			}
			list.Elements = append(list.Elements, m)
		}
		data = string(toJSON(fn, list, "  ")) + "\n"
	default:
		var b strings.Builder
		w := csv.NewWriter(&b)
		if tableFormat(path) == "tsv" {
			w.Comma = '\t'
		}
		if len(t.header) > 0 {
			w.Write(t.header)
		}
		for _, r := range t.rows {
			line := make([]string, len(t.header))
			for i := range line {
				if i < len(r) {
					line[i] = fileCell(r[i])
				}
			}
			w.Write(line)
		}
		w.Flush()
		data = b.String()
	}
	if err := os.WriteFile(it.resolvePath(path), []byte(data), 0o644); err != nil {
		fatalKind(kindFile, "%s %s: %s", fn, path, fileProblem(err))
	}
	return len(t.rows)
}

// fileCell is a value as a file cell: text as it is, none empty,
// anything else as show prints it.
func fileCell(v object.Object) string {
	switch c := v.(type) {
	case nil, *object.None:
		return ""
	case *object.String:
		return c.Value
	}
	return v.Inspect()
}

// readTableFile reads a .csv, .tsv or .json file: its header and its
// rows, an empty cell as none.
func (it *Interpreter) readTableFile(fn, path string) ([]string, [][]object.Object) {
	format := tableFormat(path)
	if format == "txt" {
		fatalKind(kindFile, "%s %s: reads .csv, .tsv and .json files (a .txt table is for people to read)", fn, path)
	}
	data, err := os.ReadFile(it.resolvePath(path))
	if err != nil {
		fatalKind(kindFile, "%s %s: %s", fn, path, fileProblem(err))
	}
	if format == "json" {
		return readJSONTable(fn, path, strings.TrimPrefix(string(data), "\ufeff"))
	}
	r := csv.NewReader(strings.NewReader(strings.TrimPrefix(string(data), "\ufeff")))
	r.FieldsPerRecord = -1
	if format == "tsv" {
		r.Comma = '\t'
	}
	header, err := r.Read()
	if err == io.EOF {
		return nil, nil
	}
	if err != nil {
		tableFileFail(fn, path, err)
	}
	seen := map[string]bool{}
	for _, h := range header {
		if seen[h] {
			fatalKind(kindCSV, "%s %s: the header names %q twice", fn, path, h)
		}
		seen[h] = true
	}
	var rows [][]object.Object
	for {
		rec, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			tableFileFail(fn, path, err)
		}
		if len(rec) == 1 && rec[0] == "" && len(header) > 1 {
			continue // a blank line
		}
		if len(rec) > len(header) {
			line, _ := r.FieldPos(0)
			fatalKind(kindCSV, "%s %s: line %d has %d values but the header has %d names", fn, path, line, len(rec), len(header))
		}
		row := make([]object.Object, len(header))
		for i := range header {
			row[i] = object.NoneValue
			if i < len(rec) && rec[i] != "" {
				row[i] = &object.String{Value: rec[i]}
			}
		}
		rows = append(rows, row)
	}
	return header, rows
}

// readJSONTable reads a JSON list of objects as rows. The columns are
// every name any row has, in the order they first appear; a row without
// one has none there.
func readJSONTable(fn, path, text string) ([]string, [][]object.Object) {
	if strings.TrimSpace(text) == "" {
		return nil, nil
	}
	v := parseJSON(fn+" "+path, text)
	list, ok := v.(*object.List)
	if !ok {
		fatalKind(kindJSON, "%s %s: a table file holds a list of objects, [ {...}, {...} ], not %s", fn, path, jsonKindName(v))
	}
	var header []string
	index := map[string]int{}
	var maps []*object.Map
	for i, e := range list.Elements {
		m, ok := e.(*object.Map)
		if !ok {
			fatalKind(kindJSON, "%s %s: row %d is %s, not an object { ... }", fn, path, i+1, jsonKindName(e))
		}
		for _, k := range m.Keys {
			name := m.KeyOf(k).(*object.String).Value
			if _, seen := index[name]; !seen {
				index[name] = len(header)
				header = append(header, name)
			}
		}
		maps = append(maps, m)
	}
	rows := make([][]object.Object, len(maps))
	for r, m := range maps {
		row := make([]object.Object, len(header))
		for i := range row {
			row[i] = object.NoneValue
		}
		for _, k := range m.Keys {
			row[index[m.KeyOf(k).(*object.String).Value]] = m.Values[k]
		}
		rows[r] = row
	}
	return header, rows
}

func jsonKindName(v object.Object) string {
	switch v.(type) {
	case *object.List:
		return "a list"
	case *object.Map:
		return "an object"
	case *object.String:
		return "a text value"
	case *object.None:
		return "null"
	case *object.Boolean:
		return "a true/false value"
	}
	return "a number"
}

func tableFileFail(fn, path string, err error) {
	var pe *csv.ParseError
	if errors.As(err, &pe) {
		fatalKind(kindCSV, "%s %s: line %d: %v", fn, path, pe.Line, pe.Err)
	}
	fatalKind(kindFile, "%s %s: %v", fn, path, err)
}

// table_read[path] (data) reads a .csv, .tsv or .json file into a list of maps,
// one per line, keyed by the header's names: the shape sql_query gives,
// so the rows go straight to table[], table_write, or a database.
func (it *Interpreter) tableRead(args []object.Object) object.Object {
	requireFuncArgs("table_read", args, 1)
	path := asStringArg("table_read", args[0])
	header, rows := it.readTableFile("table_read", path)
	out := &object.List{}
	for _, r := range rows {
		m := object.NewMap()
		for i, h := range header {
			m.Put(&object.String{Value: h}, r[i])
		}
		out.Elements = append(out.Elements, m)
	}
	return out
}

// table_write[path, x] (data) writes anything table[x] can show to a
// .csv, .tsv, .txt or .json file, and returns how many rows it wrote.
func (it *Interpreter) tableWrite(args []object.Object) object.Object {
	requireFuncArgs("table_write", args, 2)
	path := asStringArg("table_write", args[0])
	return &object.Integer{Value: int64(it.writeTableFile("table_write", path, tableShape("table_write", args[1])))}
}
