// Copyright 2017-2026 Nigele McCoy
// Licensed under the Apache License, Version 2.0;
// see LICENSE and NOTICE.

package evaluator

import (
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
// In .csv and .tsv files the first line names the columns. A plain
// number reads back as a number, quoted or not, and anything else as text
// (see csvfile.go); text is quoted only where it must be. none is an
// empty cell, and an empty cell reads back as none.
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
	return it.writeTableFileQuoted(fn, path, t, quoteNeeded)
}

// writeTableFileQuoted is writeTableFile with a choice of how a .csv or
// .tsv file quotes text (see csvQuoting).
func (it *Interpreter) writeTableFileQuoted(fn, path string, t textTable, q csvQuoting) int {
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
		sep := byte(',')
		if tableFormat(path) == "tsv" {
			sep = '\t'
		}
		if len(t.header) > 0 {
			names := make([]string, len(t.header))
			for i, h := range t.header {
				names[i] = csvHeader(h, sep)
			}
			b.WriteString(strings.Join(names, string(sep)) + "\n")
		}
		for _, r := range t.rows {
			line := make([]string, len(t.header))
			for i := range line {
				if i < len(r) && r[i] != nil {
					line[i] = csvField(r[i], sep, q)
				}
			}
			b.WriteString(strings.Join(line, string(sep)) + "\n")
		}
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
	return object.Exact(v) // a float with every digit: data, not display
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
	sep := byte(',')
	if format == "tsv" {
		sep = '\t'
	}
	recs, err := parseCSV(strings.TrimPrefix(string(data), "\ufeff"), sep)
	if err != nil {
		fatalKind(kindCSV, "%s %s: %v", fn, path, err)
	}
	if len(recs) == 0 {
		return nil, nil
	}
	var header []string
	seen := map[string]bool{}
	for _, c := range recs[0].cells {
		if seen[c.text] {
			fatalKind(kindCSV, "%s %s: the header names %q twice", fn, path, c.text)
		}
		seen[c.text] = true
		header = append(header, c.text)
	}
	var rows [][]object.Object
	for _, rec := range recs[1:] {
		if len(rec.cells) == 1 && rec.cells[0] == (csvCell{}) && len(header) > 1 {
			continue // a blank line
		}
		if len(rec.cells) > len(header) {
			fatalKind(kindCSV, "%s %s: line %d has %d values but the header has %d names", fn, path, rec.line, len(rec.cells), len(header))
		}
		row := make([]object.Object, len(header))
		for i := range header {
			row[i] = object.NoneValue
			if i < len(rec.cells) {
				row[i] = cellValue(rec.cells[i])
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
		for _, me := range m.Entries() {
			name := me.Key.(*object.String).Value
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
		for _, me := range m.Entries() {
			row[index[me.Key.(*object.String).Value]] = me.Val
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

// table_read[path [, types]] (data) reads a .csv, .tsv or .json file into
// a list of maps, one per line, keyed by the header's names: the shape
// sql_query gives, so the rows go straight to table[], table_write, or a
// database. types (see columntypes.go) turns columns into numbers,
// booleans or dates as they're read.
func (it *Interpreter) tableRead(args []object.Object) object.Object {
	if len(args) != 1 && len(args) != 2 {
		fatalf("'table_read' expects 1 or 2 arguments (file [, map of column types]), got %d", len(args))
	}
	path := asStringArg("table_read", args[0])
	header, rows := it.readTableFile("table_read", path)
	if len(args) == 2 {
		if ct := parseColumnTypes("table_read", args[1]); ct != nil {
			applyColumnTypes("table_read", path, header, rows, ct)
			if ct.key != "" {
				checkKeyColumn("table_read", path, header, rows, ct.key)
			}
		}
	}
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

// table_write[path, x [, options]] (data) writes anything table[x] can
// show to a .csv, .tsv, .txt or .json file, and returns how many rows it
// wrote. options is a map: "quote": "text" quotes every text value of a
// .csv or .tsv file ("007" rather than 007); the default, "needed",
// quotes text only when reading it back needs it.
func (it *Interpreter) tableWrite(args []object.Object) object.Object {
	if len(args) != 2 && len(args) != 3 {
		fatalf("'table_write' expects 2 or 3 arguments (file, rows [, map of options]), got %d", len(args))
	}
	path := asStringArg("table_write", args[0])
	q := quoteNeeded
	if len(args) == 3 {
		opts, ok := args[2].(*object.Map)
		if !ok {
			fatalf("'table_write' options must be a map, e.g. map [\"quote\": \"text\"], got %s", typeName(args[2]))
		}
		for _, me := range opts.Entries() {
			name, v := me.Key.Inspect(), me.Val
			if name != "quote" {
				fatalf("'table_write' has no option %q (it has \"quote\")", name)
			}
			s, ok := v.(*object.String)
			if !ok || s.Value != string(quoteNeeded) && s.Value != string(quoteText) {
				fatalf("'table_write' option \"quote\" is \"needed\" (the default: only where reading it back needs it) or \"text\" (every text value), got %s", object.Shown(v))
			}
			q = csvQuoting(s.Value)
		}
	}
	return object.Int(int64(it.writeTableFileQuoted("table_write", path, tableShape("table_write", args[1]), q)))
}
