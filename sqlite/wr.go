// Copyright 2017-2026 Nigele McCoy
// Licensed under the Apache License, Version 2.0;
// see LICENSE and NOTICE.

package sqlite

// WITHOUT ROWID tables are stored as an index on their primary key: each
// entry is a whole row, the key's columns first.

// scanWithoutRowid reads every row, in key order.
func (f *dbFile) scanWithoutRowid(t *Table) []*srcRow {
	var out []*srcRow
	var walk func(pg uint32, depth int)
	walk = func(pg uint32, depth int) {
		if depth > 40 {
			panic(errorf("damaged database: B-tree too deep"))
		}
		n := f.readNode(pg)
		if n.table() {
			panic(errorf("damaged database: WITHOUT ROWID table %s isn't stored as an index", t.Name))
		}
		for _, c := range n.cells {
			if !n.leaf() {
				walk(c.child, depth+1)
			}
			out = append(out, &srcRow{vals: f.wrRow(t, f.cellKey(n, c))})
		}
		if !n.leaf() {
			walk(n.right, depth+1)
		}
	}
	walk(t.Root, 0)
	return out
}

// wrRow puts a WITHOUT ROWID record's values in column order.
func (f *dbFile) wrRow(t *Table, rec []Value) []Value {
	vals := make([]Value, len(t.Columns))
	for pos, ci := range t.recOrder {
		if pos < len(rec) {
			vals[ci] = rec[pos]
		}
	}
	for i, c := range t.Columns {
		if n, ok := vals[i].(int64); ok && c.Affinity == affReal {
			vals[i] = float64(n)
		}
	}
	return vals
}
