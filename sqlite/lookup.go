// Copyright 2017-2026 Nigele McCoy
// Licensed under the Apache License, Version 2.0;
// see LICENSE and NOTICE.

package sqlite

import "strings"

// Fast paths: instead of reading every row of a table, a source can look
// its rows up when a condition pins them down:
//
//	rowid = value, INTEGER PRIMARY KEY = value     one B-tree search
//	rowid IN (v1, v2, ...)                         a search per value
//	column = value, with an index on the column    an index search
//
// "value" may use parameters, constants, and columns of the tables
// before this one in the join (so a join on an id looks each partner up).
// The full condition is still checked on every row found, so a lookup
// only ever skips rows that couldn't match.

type lookup struct {
	ix   *Index // nil: by rowid
	col  *colExpr
	val  expr   // col = val
	list []expr // rowid IN (list)

	// hash: no index fits, so the source's rows are grouped by the
	// column's value once, and each lookup takes one group.
	hash    bool
	coll    string
	buckets map[string][]*srcRow
	builtOn []*srcRow
}

// planLookups picks a lookup for each source that has one.
func (c *corePlan) planLookups() {
	for k, src := range c.sc.sources {
		if src.join == "RIGHT" || src.join == "FULL" || src.tvf != nil || src.self {
			continue
		}
		if src.table == nil || src.table.WithoutRowid {
			if k > 0 {
				src.lookup = c.hashLookup(k)
			}
			continue
		}
		var terms []expr
		terms = conjuncts(src.on, terms)
		if src.join == "" || src.join == "INNER" || src.join == "CROSS" {
			terms = conjuncts(c.s.where, terms)
		}
		var best *lookup
		score := 0
		for _, term := range terms {
			l, s := c.lookupFor(term, k, src)
			if l != nil && s > score {
				best, score = l, s
			}
		}
		if best == nil && k > 0 {
			best = c.hashLookup(k)
		}
		src.lookup = best
	}
}

// hashLookup finds an equality between a column of source k and values
// from the sources before it, for a join with no index to use.
func (c *corePlan) hashLookup(k int) *lookup {
	src := c.sc.sources[k]
	var terms []expr
	terms = conjuncts(src.on, terms)
	if src.join == "" || src.join == "INNER" || src.join == "CROSS" {
		terms = conjuncts(c.s.where, terms)
	}
	for _, term := range terms {
		x, ok := term.(*binExpr)
		if !ok || x.op != "=" {
			continue
		}
		coll := binaryCollation(x.l, x.r)
		for _, side := range [2][2]expr{{x.l, x.r}, {x.r, x.l}} {
			col, ok := side[0].(*colExpr)
			if !ok || col.alias != nil || col.up != 0 || col.src != k || !col.bound || col.rowid || !usableAt(side[1], k) {
				continue
			}
			// Grouping by stored value only works when comparing doesn't
			// convert the column's side.
			if numericAff(exprAffinity(side[1])) && !numericAff(col.aff) {
				continue
			}
			return &lookup{hash: true, col: col, val: side[1], coll: coll}
		}
	}
	return nil
}

func conjuncts(e expr, out []expr) []expr {
	if e == nil {
		return out
	}
	if b, ok := e.(*binExpr); ok && b.op == "AND" {
		return conjuncts(b.r, conjuncts(b.l, out))
	}
	return append(out, e)
}

// lookupFor sees whether one condition can look up source k's rows, and
// how good the lookup is (rowid 3, unique index 2, other index 1).
func (c *corePlan) lookupFor(term expr, k int, src *source) (*lookup, int) {
	switch x := term.(type) {
	case *binExpr:
		if x.op != "=" {
			return nil, 0
		}
		coll := binaryCollation(x.l, x.r)
		for _, side := range [2][2]expr{{x.l, x.r}, {x.r, x.l}} {
			col, ok := side[0].(*colExpr)
			if !ok || col.alias != nil || col.up != 0 || col.src != k || !col.bound || !usableAt(side[1], k) {
				continue
			}
			if col.rowid {
				return &lookup{col: col, val: side[1]}, 3
			}
			// Use an index only when comparing doesn't convert the
			// column's own values (the index holds them as stored).
			if numericAff(exprAffinity(side[1])) && !numericAff(col.aff) {
				continue
			}
			var pick *Index
			for _, ix := range src.table.Indexes {
				if ix.HasExpr || ix.Partial != nil || len(ix.Cols) == 0 || ix.Cols[0].Col != col.idx {
					continue
				}
				ic := ix.Cols[0].Coll
				if ic == "" {
					ic = "BINARY"
				}
				cc := coll
				if cc == "" {
					cc = "BINARY"
				}
				if !strings.EqualFold(ic, cc) {
					continue
				}
				if pick == nil || (ix.Unique && len(ix.Cols) == 1 && !(pick.Unique && len(pick.Cols) == 1)) {
					pick = ix
				}
			}
			if pick != nil {
				s := 1
				if pick.Unique && len(pick.Cols) == 1 {
					s = 2
				}
				return &lookup{ix: pick, col: col, val: side[1]}, s
			}
		}
	case *inExpr:
		col, ok := x.x.(*colExpr)
		if x.not || x.plan != nil || !ok || !col.rowid || col.alias != nil || col.up != 0 || col.src != k || !col.bound {
			return nil, 0
		}
		for _, item := range x.list {
			if !usableAt(item, k) {
				return nil, 0
			}
		}
		return &lookup{col: col, list: x.list}, 3
	}
	return nil, 0
}

func numericAff(a affinity) bool { return a == affInteger || a == affReal || a == affNumeric }

// usableAt reports whether e can be computed before source k's row is
// chosen: it reads only earlier sources, outer queries, parameters and
// constants.
func usableAt(e expr, k int) bool {
	ok := true
	walk(e, func(e expr) {
		switch x := e.(type) {
		case *colExpr:
			if x.alias != nil || !x.bound || (x.up == 0 && x.src >= k) {
				ok = false
			}
		case *subqueryExpr, *existsExpr:
			ok = false
		case *inExpr:
			if x.sub != nil {
				ok = false
			}
		case *callExpr:
			if x.agg >= 0 || isAggCall(x) || x.name == "random" || x.name == "randomblob" || x.name == "changes" {
				ok = false
			}
		}
	})
	return ok
}

// find returns the rows the lookup selects for the current row of the
// sources before it.
func (l *lookup) find(r *runner, src *source) []*srcRow {
	if l.hash {
		if l.buckets == nil || len(src.rows) != len(l.builtOn) || len(src.rows) > 0 && &src.rows[0] != &l.builtOn[0] {
			l.buckets = map[string][]*srcRow{}
			for _, row := range src.rows {
				if l.col.idx >= len(row.vals) {
					continue
				}
				v := row.vals[l.col.idx]
				if i, ok := v.(int64); ok && l.col.aff == affReal {
					v = float64(i)
				}
				if v == nil {
					continue
				}
				key := collatedKey([]Value{v}, []string{l.coll})
				l.buckets[key] = append(l.buckets[key], row)
			}
			l.builtOn = src.rows
		}
		v := r.eval(l.val)
		if v == nil {
			return nil
		}
		_, v = compareAffinity(l.col, l.val, nil, v)
		return l.buckets[collatedKey([]Value{v}, []string{l.coll})]
	}
	t := src.table
	fl := src.fileOr(r.db)
	var out []*srcRow
	fetch := func(rowid int64) {
		payload, ok := fl.seekRow(t.Root, rowid)
		if !ok {
			return
		}
		vals, err := fl.decodeRecord(payload)
		if err != nil {
			panic(err)
		}
		if t.hasVirtual {
			vals = fl.rowValues(t, rowid, vals)
		}
		out = append(out, &srcRow{rowid: rowid, vals: vals})
	}
	if l.ix == nil {
		vals := []expr{l.val}
		if l.list != nil {
			vals = l.list
		}
		seen := map[int64]bool{}
		for _, e := range vals {
			v := r.eval(e)
			if v == nil {
				continue
			}
			_, v = compareAffinity(l.col, e, nil, v)
			id, ok := exactRowid(v)
			if !ok || seen[id] {
				continue
			}
			seen[id] = true
			fetch(id)
		}
		return out
	}
	v := r.eval(l.val)
	if v == nil {
		return nil
	}
	_, v = compareAffinity(l.col, l.val, nil, v)
	for _, e := range fl.indexEqual(l.ix, []Value{v}) {
		if id, ok := e[len(e)-1].(int64); ok {
			fetch(id)
		}
	}
	return out
}

// exactRowid is v as a rowid, if a rowid can equal it.
func exactRowid(v Value) (int64, bool) {
	switch x := v.(type) {
	case int64:
		return x, true
	case float64:
		if x == float64(int64(x)) && x >= -9.2e18 && x <= 9.2e18 {
			return int64(x), true
		}
	case string:
		if n, ok := applyAffinity(x, affInteger).(int64); ok {
			return n, true
		}
	}
	return 0, false
}
