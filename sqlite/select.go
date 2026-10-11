// Copyright 2017-2026 Nigele McCoy
// Licensed under the Apache License, Version 2.0;
// see LICENSE and NOTICE.

package sqlite

import (
	"math"
	"sort"
	"strconv"
	"strings"
)

// Running a SELECT happens in two steps. Planning binds every column name
// to the FROM item (source) it reads, expands *, plans subqueries and WITH
// tables, and finds the aggregates. Running then loops over the sources
// (nested loops, one per FROM item: a join), filters with WHERE, groups,
// filters groups with HAVING, and sorts.

// source is one FROM item: a stored table, a subquery, or a WITH table.
type source struct {
	name     string // what qualifies its columns: the alias, or the table name
	table    *Table // a stored table, or nil
	cols     []Column
	rowidCol int             // the INTEGER PRIMARY KEY column, or -1
	hidden   map[string]bool // USING and NATURAL columns, read from the left table instead
	join     string          // "" for the first, else INNER, LEFT, RIGHT, FULL or CROSS
	on       expr
	sub      *queryPlan // FROM (SELECT ...)
	cte      *ctePlan   // a WITH table
	self     bool       // the WITH table's own name, inside its recursive part
	rows     []*srcRow
	loaded   bool
	matched  []bool // RIGHT and FULL joins: rows that found a partner
	lookup   *lookup
	tvf      *tvfPlan // a table-valued function: rows worked out per use
	file     *dbFile  // the file a table is in (an attached database's), or nil: the main one
	// qualifiedOnly: its columns are named only as name.column (an
	// upsert's excluded row).
	qualifiedOnly bool
}

// srcRow is one row of a source. vals is nil for a missing (NULL) row.
type srcRow struct {
	rowid int64
	vals  []Value
}

// scope is the names a query's expressions can see: its sources, then
// the scopes of the queries around it (for correlated subqueries).
type scope struct {
	sources []*source
	parent  *scope
	owner   *queryPlan
	ctes    *cteEnv
}

type cteEnv struct {
	defs   map[string]*ctePlan
	parent *cteEnv
}

func (e *cteEnv) find(name string) *ctePlan {
	for ; e != nil; e = e.parent {
		if c, ok := e.defs[strings.ToLower(name)]; ok {
			return c
		}
	}
	return nil
}

// ctePlan is one WITH table.
type ctePlan struct {
	def       *cteDef
	names     []string
	affs      []affinity
	colls     []string
	q         *queryPlan
	recursive bool
	selfCore  int // which core of q names the table itself
	rows      []*srcRow
	done      bool
	working   []*srcRow // the rows the recursive part reads, step by step
}

// queryPlan is a whole SELECT: one or more cores joined by UNION,
// INTERSECT or EXCEPT, then ORDER BY and LIMIT.
type queryPlan struct {
	db         *dbFile
	cores      []*corePlan
	ops        []string
	order      []orderTerm // compound ORDER BY, by position
	limit      expr
	offset     expr
	names      []string
	affs       []affinity
	colls      []string
	correlated bool // reads a column of a query around it, so it can't be cached
	cache      [][]Value
	cached     bool
	sc         *scope // where LIMIT and OFFSET are evaluated
	cte        *ctePlan
}

// corePlan is one SELECT ... FROM ... WHERE ... GROUP BY ... HAVING.
type corePlan struct {
	q        *queryPlan
	s        *selectStmt
	sc       *scope
	cols     []resultCol
	aggs     []*callExpr
	grouped  bool
	minmax   int // the lone min() or max() aggregate, whose row bare columns read, or -1
	order    []orderTerm
	ordColls []string
	wins     []*winPlan
}

type outRow struct {
	vals []Value
	keys []Value
}

// planQuery plans s. parent is the scope of the query around it (for a
// subquery), env the WITH tables in reach, and cte the WITH table s
// defines, if any.
func (db *dbFile) planQuery(s *selectStmt, parent *scope, env *cteEnv, cte *ctePlan) *queryPlan {
	q := &queryPlan{db: db, cte: cte}
	if len(s.with) > 0 {
		env = db.planWithIn(s.with, s.recursive, parent, env)
	}
	q.sc = &scope{parent: parent, owner: q, ctes: env}
	cores := []*selectStmt{s}
	for _, part := range s.compound {
		cores = append(cores, part.core)
		q.ops = append(q.ops, part.op)
	}
	for i, cs := range cores {
		var order []orderTerm
		if len(cores) == 1 {
			order = s.order
		}
		core := db.planCore(q, cs, parent, env, order)
		q.cores = append(q.cores, core)
		if i == 0 {
			for _, rc := range core.cols {
				q.names = append(q.names, rc.name)
				q.affs = append(q.affs, exprAffinity(rc.e))
				coll, _ := exprCollation(rc.e)
				q.colls = append(q.colls, coll)
			}
			if cte != nil {
				cte.names = q.names
				if len(cte.def.cols) > 0 {
					if len(cte.def.cols) != len(q.names) {
						fail("SQL: table %s has %d values for %d columns", cte.def.name, len(q.names), len(cte.def.cols))
					}
					cte.names = cte.def.cols
				}
				cte.affs, cte.colls = q.affs, q.colls
			}
		} else if len(core.cols) != len(q.names) {
			fail("SQL: SELECTs to the left and right of %s do not have the same number of result columns", q.ops[i-1])
		}
	}
	if cte != nil && cte.recursive && cte.selfCore != len(cores)-1 {
		fail("SQL: in WITH %s, only the last SELECT (after UNION or UNION ALL) can name %s", cte.def.name, cte.def.name)
	}
	if len(cores) > 1 {
		for _, t := range s.order {
			if t.pos == 0 {
				if c, ok := t.e.(*colExpr); ok && c.table == "" {
					for i, n := range q.names {
						if strings.EqualFold(n, c.name) {
							t.pos = i + 1
							break
						}
					}
				}
				if t.pos == 0 {
					fail("SQL: ORDER BY term %d does not match any column in the result set", len(q.order)+1)
				}
			}
			if t.pos < 1 || t.pos > len(q.names) {
				fail("SQL: ORDER BY term out of range - should be between 1 and %d", len(q.names))
			}
			q.order = append(q.order, t)
		}
	}
	q.limit, q.offset = s.limit, s.offset
	lim := &corePlan{q: q, sc: q.sc}
	lim.resolve(q.limit, false)
	lim.resolve(q.offset, false)
	return q
}

func (src *source) fileOr(def *dbFile) *dbFile {
	if src.file != nil {
		return src.file
	}
	return def
}

// schemaFile is the file a schema-qualified name is in: the main one, or
// an attached database. Without a schema, the main file, unless only an
// attached database has the name.
func (db *dbFile) schemaFile(schema, name string) *dbFile {
	low := strings.ToLower(schema)
	switch low {
	case "", "main":
		if low == "" {
			lname := strings.ToLower(name)
			_, t := db.tables[lname]
			_, v := db.views[lname]
			if !t && !v && lname != "sqlite_schema" && lname != "sqlite_master" {
				for _, a := range db.attached {
					if _, ok := a.db.tables[lname]; ok {
						return a.db.dbFile
					}
					if _, ok := a.db.views[lname]; ok {
						return a.db.dbFile
					}
				}
			}
		}
		return db
	case "temp":
		fail("SQL: temp tables aren't supported")
	}
	for _, a := range db.attached {
		if strings.EqualFold(a.name, schema) {
			return a.db.dbFile
		}
	}
	fail("SQL: unknown database %s", schema)
	return nil
}

// isBareFunction reports whether a FROM name with no ( ) is a
// table-valued function (FROM pragma_table_list), not a table.
func (db *dbFile) isBareFunction(name string, env *cteEnv) bool {
	low := strings.ToLower(name)
	if _, ok := tvfColumns[low]; !ok || env.find(name) != nil {
		return false
	}
	_, isTable := db.tables[low]
	_, isView := db.views[low]
	return !isTable && !isView
}

// planWith plans WITH tables for a statement that isn't a SELECT.
func (db *dbFile) planWith(with []*cteDef, recursive bool) *cteEnv {
	if len(with) == 0 {
		return nil
	}
	return db.planWithIn(with, recursive, nil, nil)
}

func (db *dbFile) planWithIn(with []*cteDef, recursive bool, parent *scope, env *cteEnv) *cteEnv {
	env = &cteEnv{defs: map[string]*ctePlan{}, parent: env}
	for _, d := range with {
		key := strings.ToLower(d.name)
		if _, dup := env.defs[key]; dup {
			fail("SQL: duplicate WITH table name: %s", d.name)
		}
		c := &ctePlan{def: d, selfCore: -1}
		if recursive {
			env.defs[key] = c // visible to its own query
		}
		c.q = db.planQuery(d.sel, parent, env, c)
		env.defs[key] = c
	}
	return env
}

func (db *dbFile) planCore(q *queryPlan, s *selectStmt, parent *scope, env *cteEnv, order []orderTerm) *corePlan {
	c := &corePlan{q: q, s: s, minmax: -1}
	c.sc = &scope{parent: parent, owner: q, ctes: env}
	if s.values != nil {
		for _, row := range s.values {
			for _, e := range row {
				c.resolve(e, false)
				if c.hasAgg(e) {
					fail("SQL: aggregate functions can't be used in VALUES")
				}
			}
		}
		for i, e := range s.values[0] {
			c.cols = append(c.cols, resultCol{e: e, name: "column" + itoa(i+1)})
		}
		c.planOrder(order)
		return c
	}
	for _, f := range s.from {
		c.addSource(db, f, parent, env)
	}
	for _, src := range c.sc.sources {
		c.resolve(src.on, false)
		if c.hasAgg(src.on) {
			fail("SQL: aggregate functions can't be used in ON")
		}
	}
	for _, rc := range s.cols {
		if rc.star {
			found := false
			for si, src := range c.sc.sources {
				if rc.table != "" && !strings.EqualFold(src.name, rc.table) {
					continue
				}
				found = true
				for ci, col := range src.cols {
					if rc.table == "" && src.hidden[strings.ToLower(col.Name)] {
						continue
					}
					c.cols = append(c.cols, resultCol{e: c.colRef(si, ci), name: col.Name, bare: true})
				}
			}
			if !found {
				if rc.table != "" {
					fail("SQL: no such table: %s", rc.table)
				}
				fail("SQL: SELECT * needs a FROM table")
			}
			continue
		}
		c.resolve(rc.e, false)
		// SELECT rowid names the column after an INTEGER PRIMARY KEY, as
		// SQLite does, since that column is the rowid.
		if col, ok := rc.e.(*colExpr); ok && rc.bare && col.rowid && col.up == 0 && col.idx < 0 {
			if src := c.sc.sources[col.src]; src.rowidCol >= 0 {
				rc.name = src.cols[src.rowidCol].Name
			}
		}
		c.cols = append(c.cols, rc)
	}
	c.resolve(s.where, true)
	if c.hasAgg(s.where) {
		fail("SQL: aggregate functions like count() can't be used in WHERE")
	}
	if hasWindow(s.where) || hasWindow(s.having) {
		fail("SQL: window functions can't be used in WHERE or HAVING")
	}
	var groupBy []expr
	for _, g := range s.groupBy {
		if lit, ok := g.(*litExpr); ok {
			if n, ok := lit.v.(int64); ok {
				if n < 1 || n > int64(len(c.cols)) {
					fail("SQL: GROUP BY term out of range - should be between 1 and %d", len(c.cols))
				}
				g = c.cols[n-1].e
			}
		}
		c.resolve(g, true)
		if hasWindow(g) {
			fail("SQL: window functions can't be used in GROUP BY")
		}
		if c.hasAgg(g) {
			fail("SQL: aggregate functions can't be used in GROUP BY")
		}
		groupBy = append(groupBy, g)
	}
	s.groupBy = groupBy
	c.resolve(s.having, true)
	for _, rc := range c.cols {
		c.findAggs(rc.e)
	}
	c.findAggs(s.having)
	c.planOrder(order)
	for _, t := range c.order {
		if t.pos == 0 {
			c.findAggs(t.e)
		}
	}
	c.planWindows()
	c.grouped = len(c.aggs) > 0 || len(s.groupBy) > 0 || s.having != nil
	for i, a := range c.aggs {
		if a.name == "min" || a.name == "max" {
			if c.minmax == -1 {
				c.minmax = i
			} else {
				c.minmax = -2
			}
		} else {
			c.minmax = -2
		}
	}
	if c.minmax < 0 {
		c.minmax = -1
	}
	c.planLookups()
	return c
}

// planOrder resolves a single core's ORDER BY: positions, result column
// names, then expressions.
func (c *corePlan) planOrder(order []orderTerm) {
	for _, t := range order {
		if t.pos != 0 {
			if t.pos < 1 || t.pos > len(c.cols) {
				fail("SQL: ORDER BY %d is outside the %d result column(s)", t.pos, len(c.cols))
			}
		} else if col, ok := t.e.(*colExpr); ok && col.table == "" && !c.isSourceColumn(col.name) {
			// A bare name matching a result column's name sorts by it.
			for ci, rc := range c.cols {
				if strings.EqualFold(rc.name, col.name) {
					t.pos = ci + 1
					break
				}
			}
		}
		coll := ""
		if t.pos != 0 {
			coll, _ = exprCollation(c.cols[t.pos-1].e)
		} else {
			c.resolve(t.e, true)
			coll, _ = exprCollation(t.e)
		}
		c.order = append(c.order, t)
		c.ordColls = append(c.ordColls, coll)
	}
}

func (c *corePlan) isSourceColumn(name string) bool {
	for _, src := range c.sc.sources {
		if colIndex(src.cols, name) >= 0 {
			return true
		}
	}
	return false
}

func colIndex(cols []Column, name string) int {
	for i, col := range cols {
		if strings.EqualFold(col.Name, name) {
			return i
		}
	}
	return -1
}

func isRowidName(name string) bool {
	switch strings.ToLower(name) {
	case "rowid", "oid", "_rowid_":
		return true
	}
	return false
}

func (c *corePlan) addSource(db *dbFile, f fromItem, parent *scope, env *cteEnv) {
	src := &source{join: f.join, on: f.on, rowidCol: -1, hidden: map[string]bool{}}
	switch {
	case f.isFunc || f.sub == nil && db.isBareFunction(f.name, env):
		t := c.tvfSource(f)
		t.join, t.on = f.join, f.on
		src = t
		if src.join == "RIGHT" || src.join == "FULL" {
			fail("SQL: a table-valued function can't be on the right of a RIGHT or FULL join")
		}
	case f.sub != nil:
		src.sub = db.planQuery(f.sub, parent, env, nil)
		src.name = f.alias
		for i, n := range src.sub.names {
			src.cols = append(src.cols, Column{Name: n, Affinity: src.sub.affs[i], Collate: src.sub.colls[i]})
		}
	default:
		src.name = f.name
		if f.alias != "" {
			src.name = f.alias
		}
		if cte := env.find(f.name); cte != nil {
			if cte.names == nil {
				if cte.q != nil || cte.def == nil {
					fail("SQL: circular reference: %s", f.name)
				}
				fail("SQL: in WITH %s, the first SELECT can't name %s itself", f.name, f.name)
			}
			src.cte = cte
			if cte.q == nil { // inside its own definition: the recursive part
				src.self = true
				cte.recursive = true
				cte.selfCore = len(c.q.cores)
			}
			for i, n := range cte.names {
				src.cols = append(src.cols, Column{Name: n, Affinity: cte.affs[i], Collate: cte.colls[i]})
			}
			break
		}
		home := db.schemaFile(f.schema, f.name)
		if v, ok := home.views[strings.ToLower(f.name)]; ok {
			src.sub = home.planView(v, env)
			for i, n := range src.sub.names {
				src.cols = append(src.cols, Column{Name: n, Affinity: src.sub.affs[i], Collate: src.sub.colls[i]})
			}
			break
		}
		t, err := home.table(f.name)
		if err != nil {
			panic(err)
		}
		if home != db {
			src.file = home
		}
		src.table, src.cols, src.rowidCol = t, t.Columns, t.RowidCol
		if f.alias == "" {
			src.name = t.Name
		}
	}
	k := len(c.sc.sources)
	names := f.using
	if f.natural {
		for _, col := range src.cols {
			for _, left := range c.sc.sources {
				if !left.hidden[strings.ToLower(col.Name)] && colIndex(left.cols, col.Name) >= 0 {
					names = append(names, col.Name)
					break
				}
			}
		}
	}
	c.sc.sources = append(c.sc.sources, src)
	for _, n := range names {
		ri := colIndex(src.cols, n)
		li, lc := -1, -1
		for j := 0; j < k && li < 0; j++ {
			left := c.sc.sources[j]
			if left.hidden[strings.ToLower(n)] {
				continue
			}
			if x := colIndex(left.cols, n); x >= 0 {
				li, lc = j, x
			}
		}
		if ri < 0 || li < 0 {
			fail("SQL: cannot join using column %s - column not present in both tables", n)
		}
		eq := &binExpr{op: "=", l: c.colRef(li, lc), r: c.colRef(k, ri)}
		if src.on == nil {
			src.on = eq
		} else {
			src.on = &binExpr{op: "AND", l: src.on, r: eq}
		}
		src.hidden[strings.ToLower(n)] = true
	}
}

// colRef is an already-bound reference to column ci of source si.
func (c *corePlan) colRef(si, ci int) *colExpr {
	src := c.sc.sources[si]
	col := &src.cols[ci]
	return &colExpr{name: col.Name, src: si, idx: ci, rowid: ci == src.rowidCol, aff: col.Affinity, coll: col.Collate, col: col, bound: true}
}

// resolve binds the names in e and plans its subqueries. aliases lets a
// bare name that isn't a column fall back to a result column's name.
func (c *corePlan) resolve(e expr, aliases bool) {
	walk(e, func(e expr) {
		switch x := e.(type) {
		case *colExpr:
			c.bind(x, aliases)
		case *subqueryExpr:
			x.plan = c.q.db.planQuery(x.sel, c.sc, c.sc.ctes, nil)
			if len(x.plan.names) != 1 {
				fail("SQL: sub-select returns %d columns - expected 1", len(x.plan.names))
			}
		case *existsExpr:
			x.plan = c.q.db.planQuery(x.sel, c.sc, c.sc.ctes, nil)
		case *inExpr:
			if x.sub != nil {
				x.plan = c.q.db.planQuery(x.sub, c.sc, c.sc.ctes, nil)
				if len(x.plan.names) != 1 {
					fail("SQL: sub-select returns %d columns - expected 1", len(x.plan.names))
				}
			}
		}
	})
}

func (c *corePlan) bind(x *colExpr, aliases bool) {
	if x.bound || x.alias != nil {
		return
	}
	up := 0
	for sc := c.sc; sc != nil; sc, up = sc.parent, up+1 {
		if si, ci, ok := sc.lookup(x); ok {
			x.src, x.idx, x.up, x.bound = si, ci, up, true
			src := sc.sources[si]
			if ci < 0 {
				x.rowid, x.aff = true, affInteger
			} else {
				col := &src.cols[ci]
				x.aff, x.coll, x.col, x.rowid = col.Affinity, col.Collate, col, ci == src.rowidCol
			}
			s := c.sc
			for i := 0; i < up; i++ {
				s.owner.correlated = true
				s = s.parent
			}
			return
		}
		if up == 0 && aliases && x.table == "" {
			for _, rc := range c.cols {
				if !rc.star && strings.EqualFold(rc.name, x.name) {
					x.alias = rc.e
					return
				}
			}
		}
	}
	if x.table != "" {
		fail("SQL: no such column: %s.%s", x.table, x.name)
	}
	if len(c.sc.sources) == 0 && c.sc.parent == nil {
		fail("SQL: no such column: %s (there's no FROM table)", x.name)
	}
	fail("SQL: no such column: %s", x.name)
}

// lookup finds the column x names in this scope's sources: (source,
// column, true), with column -1 for the rowid.
func (sc *scope) lookup(x *colExpr) (int, int, bool) {
	if x.table != "" {
		for i, src := range sc.sources {
			if !strings.EqualFold(src.name, x.table) {
				continue
			}
			if ci := colIndex(src.cols, x.name); ci >= 0 {
				return i, ci, true
			}
			if src.table != nil && !src.table.WithoutRowid && isRowidName(x.name) {
				return i, -1, true
			}
			fail("SQL: no such column: %s.%s", x.table, x.name)
		}
		return 0, 0, false
	}
	found, fci := -1, -1
	for i, src := range sc.sources {
		if src.hidden[strings.ToLower(x.name)] || src.qualifiedOnly {
			continue
		}
		if ci := colIndex(src.cols, x.name); ci >= 0 {
			if found >= 0 {
				fail("SQL: ambiguous column name: %s", x.name)
			}
			found, fci = i, ci
		}
	}
	if found >= 0 {
		return found, fci, true
	}
	if isRowidName(x.name) {
		for i, src := range sc.sources {
			if src.table != nil && !src.table.WithoutRowid {
				if found >= 0 {
					fail("SQL: ambiguous column name: %s", x.name)
				}
				found = i
			}
		}
		if found >= 0 {
			return found, -1, true
		}
	}
	return 0, 0, false
}

// walk calls fn on e and every expression inside it (not inside
// subqueries, which are planned on their own).
func walk(e expr, fn func(expr)) {
	if e == nil {
		return
	}
	fn(e)
	switch x := e.(type) {
	case *unaryExpr:
		walk(x.x, fn)
	case *binExpr:
		walk(x.l, fn)
		walk(x.r, fn)
	case *isExpr:
		walk(x.l, fn)
		walk(x.r, fn)
	case *isNullExpr:
		walk(x.x, fn)
	case *inExpr:
		walk(x.x, fn)
		for _, a := range x.list {
			walk(a, fn)
		}
	case *betweenExpr:
		walk(x.x, fn)
		walk(x.lo, fn)
		walk(x.hi, fn)
	case *likeExpr:
		walk(x.x, fn)
		walk(x.pat, fn)
		walk(x.esc, fn)
	case *callExpr:
		for _, a := range x.args {
			walk(a, fn)
		}
		walk(x.filter, fn)
	case *caseExpr:
		walk(x.base, fn)
		for _, w := range x.whens {
			walk(w[0], fn)
			walk(w[1], fn)
		}
		walk(x.els, fn)
	case *castExpr:
		walk(x.x, fn)
	case *collateExpr:
		walk(x.x, fn)
	case *raiseExpr:
		walk(x.msg, fn)
	}
}

var aggNames = map[string]bool{"count": true, "sum": true, "total": true, "avg": true, "min": true, "max": true, "group_concat": true, "string_agg": true,
	"json_group_array": true, "json_group_object": true}

func isAggCall(c *callExpr) bool {
	if !aggNames[c.name] {
		return false
	}
	// min(a, b) and max(a, b) with two or more arguments are scalar.
	if (c.name == "min" || c.name == "max") && len(c.args) != 1 {
		return false
	}
	return true
}

func (c *corePlan) findAggs(e expr) {
	walk(e, func(e expr) {
		x, ok := e.(*callExpr)
		if !ok || !isAggCall(x) || x.agg >= 0 || x.over != nil {
			return
		}
		switch {
		case x.name == "count" && !x.star && len(x.args) != 1:
			fail("SQL: count() takes * or one value")
		case x.name == "string_agg" && len(x.args) != 2:
			fail("SQL: string_agg() takes two values")
		case x.name == "group_concat" && (len(x.args) < 1 || len(x.args) > 2):
			fail("SQL: group_concat() takes one or two values")
		case x.name == "json_group_object" && len(x.args) != 2:
			fail("SQL: json_group_object() takes a label and a value")
		case x.name != "count" && x.name != "group_concat" && x.name != "string_agg" && x.name != "json_group_object" && len(x.args) != 1:
			fail("SQL: %s() takes one value", x.name)
		}
		for _, a := range x.args {
			if c.hasAgg(a) {
				fail("SQL: misuse of aggregate function %s()", x.name)
			}
		}
		x.agg = len(c.aggs)
		c.aggs = append(c.aggs, x)
	})
}

func (c *corePlan) hasAgg(e expr) bool {
	found := false
	walk(e, func(e expr) {
		if x, ok := e.(*callExpr); ok && isAggCall(x) && x.over == nil {
			found = true
		}
	})
	return found
}

// ---- running ----------------------------------------------------------------

type runner struct {
	db     *dbFile
	params []Value
	parent *runner
	cur    []*srcRow
	aggs   []*aggState
	final  bool // in a group's result: aggregates give their totals
	core   *corePlan
	winVal [][]Value // window function results, per row of the window stage
	winRow int
}

// rows runs the query and returns its rows. A query that reads nothing
// from the queries around it runs once and keeps its answer.
func (q *queryPlan) rows(params []Value, parent *runner) [][]Value {
	if q.cached {
		return q.cache
	}
	var out [][]Value
	if q.cte != nil && q.cte.recursive {
		out = q.recursiveRows(params, parent)
	} else if len(q.cores) == 1 {
		core := q.cores[0]
		res := core.run(params, parent)
		if len(core.order) > 0 {
			sort.SliceStable(res, func(i, j int) bool {
				return orderLess(res[i].keys, res[j].keys, core.order, core.ordColls)
			})
		}
		out = make([][]Value, len(res))
		for i, o := range res {
			out[i] = o.vals
		}
		out = q.limitRows(out, params, parent)
	} else {
		out = q.compoundRows(params, parent, len(q.cores))
		if len(q.order) > 0 {
			colls := make([]string, len(q.order))
			for i, t := range q.order {
				colls[i] = q.colls[t.pos-1]
			}
			sort.SliceStable(out, func(i, j int) bool {
				ki, kj := make([]Value, len(q.order)), make([]Value, len(q.order))
				for k, t := range q.order {
					ki[k], kj[k] = out[i][t.pos-1], out[j][t.pos-1]
				}
				return orderLess(ki, kj, q.order, colls)
			})
		}
		out = q.limitRows(out, params, parent)
	}
	if !q.correlated {
		q.cache, q.cached = out, true
	}
	return out
}

// compoundRows combines the first n cores with their UNION, INTERSECT
// and EXCEPT operators, left to right.
func (q *queryPlan) compoundRows(params []Value, parent *runner, n int) [][]Value {
	acc := q.cores[0].values(params, parent)
	for i := 1; i < n; i++ {
		acc = combine(q.ops[i-1], acc, q.cores[i].values(params, parent))
	}
	return acc
}

func (c *corePlan) values(params []Value, parent *runner) [][]Value {
	res := c.run(params, parent)
	out := make([][]Value, len(res))
	for i, o := range res {
		out[i] = o.vals
	}
	return out
}

func combine(op string, a, b [][]Value) [][]Value {
	switch op {
	case "UNION ALL":
		return append(append([][]Value{}, a...), b...)
	case "UNION":
		return distinctRows(append(append([][]Value{}, a...), b...))
	}
	inB := map[string]bool{}
	for _, r := range b {
		inB[rowKey(r)] = true
	}
	var out [][]Value
	for _, r := range a {
		if inB[rowKey(r)] == (op == "INTERSECT") {
			out = append(out, r)
		}
	}
	return distinctRows(out)
}

func distinctRows(rows [][]Value) [][]Value {
	seen := map[string]bool{}
	var out [][]Value
	for _, r := range rows {
		k := rowKey(r)
		if !seen[k] {
			seen[k] = true
			out = append(out, r)
		}
	}
	return out
}

func (q *queryPlan) limitRows(out [][]Value, params []Value, parent *runner) [][]Value {
	r := &runner{db: q.db, params: params, parent: parent, core: &corePlan{q: q, sc: q.sc}}
	if q.offset != nil {
		n := r.intValue(q.offset, "OFFSET")
		if n > int64(len(out)) {
			n = int64(len(out))
		}
		if n > 0 {
			out = out[n:]
		}
	}
	if q.limit != nil {
		if n := r.intValue(q.limit, "LIMIT"); n >= 0 && n < int64(len(out)) {
			out = out[:n]
		}
	}
	return out
}

// recursiveRows runs a recursive WITH table: the first SELECTs give the
// starting rows, then the last one runs again and again on the rows the
// step before added, until a step adds none.
func (q *queryPlan) recursiveRows(params []Value, parent *runner) [][]Value {
	n := len(q.cores)
	if n < 2 || (q.ops[n-2] != "UNION" && q.ops[n-2] != "UNION ALL") {
		fail("SQL: a recursive WITH needs its recursive SELECT after UNION or UNION ALL")
	}
	if len(q.order) > 0 {
		fail("SQL: ORDER BY in a recursive WITH isn't supported")
	}
	all := q.compoundRows(params, parent, n-1)
	distinct := q.ops[n-2] == "UNION"
	seen := map[string]bool{}
	if distinct {
		all = distinctRows(all)
		for _, r := range all {
			seen[rowKey(r)] = true
		}
	}
	limit := int64(-1)
	if q.limit != nil {
		r := &runner{db: q.db, params: params, parent: parent, core: &corePlan{q: q, sc: q.sc}}
		limit = r.intValue(q.limit, "LIMIT")
		if limit >= 0 && q.offset != nil {
			if off := r.intValue(q.offset, "OFFSET"); off > 0 {
				limit += off
			}
		}
	}
	work := all
	for len(work) > 0 && (limit < 0 || int64(len(all)) < limit) {
		q.cte.working = toSrcRows(work)
		var next [][]Value
		for _, r := range q.cores[n-1].values(params, parent) {
			if distinct {
				k := rowKey(r)
				if seen[k] {
					continue
				}
				seen[k] = true
			}
			next = append(next, r)
		}
		all = append(all, next...)
		work = next
		if len(all) > 10000000 {
			fail("SQL: WITH %s made over 10 million rows; is its stop condition missing?", q.cte.def.name)
		}
	}
	q.cte.working = nil
	return q.limitRows(all, params, parent)
}

func toSrcRows(vals [][]Value) []*srcRow {
	out := make([]*srcRow, len(vals))
	for i, v := range vals {
		out[i] = &srcRow{vals: v}
	}
	return out
}

// load fills a source's rows for one run.
func (src *source) load(r *runner, k int) {
	switch {
	case src.tvf != nil:
		return
	case src.self:
		src.rows = src.cte.working
	case src.lookup != nil && !src.lookup.hash && src.join != "RIGHT" && src.join != "FULL":
		return
	case src.loaded:
	case src.table != nil && src.table.WithoutRowid:
		src.rows = src.fileOr(r.db).scanWithoutRowid(src.table)
		src.loaded = true
	case src.table != nil:
		var rows []*srcRow
		fl := src.fileOr(r.db)
		err := fl.scanTable(src.table.Root, func(rowid int64, payload []byte) error {
			vals, err := fl.decodeRecord(payload)
			if err != nil {
				return err
			}
			if src.table.hasVirtual {
				vals = fl.rowValues(src.table, rowid, vals)
			}
			rows = append(rows, &srcRow{rowid: rowid, vals: vals})
			return nil
		})
		if err != nil {
			panic(err)
		}
		src.rows, src.loaded = rows, true
	case src.sub != nil:
		src.rows = toSrcRows(src.sub.rows(r.params, r.parent))
		src.loaded = !src.sub.correlated
	case src.cte != nil:
		src.rows = src.cte.load(r.params, r.parent)
		src.loaded = src.cte.done
	}
	if src.join == "RIGHT" || src.join == "FULL" {
		src.matched = make([]bool, len(src.rows))
	}
}

func (c *ctePlan) load(params []Value, parent *runner) []*srcRow {
	if c.done {
		return c.rows
	}
	rows := toSrcRows(c.q.rows(params, parent))
	if !c.q.correlated {
		c.rows, c.done = rows, true
	}
	return rows
}

// run runs one core and returns its rows with their ORDER BY keys.
func (c *corePlan) run(params []Value, parent *runner) []outRow {
	r := &runner{db: c.q.db, params: params, parent: parent, core: c, cur: make([]*srcRow, len(c.sc.sources))}
	var out []outRow
	if c.s.values != nil {
		for _, row := range c.s.values {
			vals := make([]Value, len(row))
			for i, e := range row {
				vals[i] = r.eval(e)
			}
			out = append(out, outRow{vals: vals, keys: r.orderKeys(vals)})
		}
		return out
	}
	for k, src := range c.sc.sources {
		src.load(r, k)
	}
	type group struct {
		key  []Value
		aggs []*aggState
		rep  []*srcRow
	}
	groups := map[string]*group{}
	var order []*group
	var snaps []winSnap
	produce := func() {
		if len(c.wins) > 0 {
			snaps = append(snaps, r.snapshot())
			return
		}
		out = append(out, r.emit())
	}
	keyColls := make([]string, len(c.s.groupBy))
	for i, g := range c.s.groupBy {
		keyColls[i], _ = exprCollation(g)
	}
	visit := func() {
		if c.s.where != nil && !truthy(r.eval(c.s.where)) {
			return
		}
		if !c.grouped {
			produce()
			return
		}
		key := make([]Value, len(c.s.groupBy))
		for i, g := range c.s.groupBy {
			key[i] = r.eval(g)
		}
		ks := collatedKey(key, keyColls)
		g, ok := groups[ks]
		if !ok {
			g = &group{key: key, aggs: newAggStates(c.aggs)}
			groups[ks] = g
			order = append(order, g)
		}
		r.aggs = g.aggs
		changed := r.accumulate()
		if c.minmax < 0 || changed[c.minmax] || g.rep == nil {
			g.rep = append(g.rep[:0:0], r.cur...)
		}
	}
	r.loop(0, visit)
	for k, src := range c.sc.sources {
		if src.join != "RIGHT" && src.join != "FULL" {
			continue
		}
		for i, row := range src.rows {
			if src.matched[i] {
				continue
			}
			for j := 0; j < k; j++ {
				r.cur[j] = nil
			}
			r.cur[k] = row
			r.loop(k+1, visit)
		}
		r.cur[k] = nil
	}
	if c.grouped {
		if len(order) == 0 && len(c.s.groupBy) == 0 {
			order = append(order, &group{aggs: newAggStates(c.aggs), rep: make([]*srcRow, len(c.sc.sources))})
		}
		sort.SliceStable(order, func(i, j int) bool {
			for k := range order[i].key {
				if d := compareColl(order[i].key[k], order[j].key[k], keyColls[k]); d != 0 {
					return d < 0
				}
			}
			return false
		})
		r.final = true
		for _, g := range order {
			r.aggs = g.aggs
			copy(r.cur, g.rep)
			if c.s.having != nil && !truthy(r.eval(c.s.having)) {
				continue
			}
			produce()
		}
		r.final = false
	}
	if len(c.wins) > 0 {
		r.winVal = c.computeWindows(r, snaps)
		for i, s := range snaps {
			r.restore(s)
			r.winRow = i
			out = append(out, r.emit())
		}
		r.winVal = nil
	}
	if c.s.distinct {
		colls := make([]string, len(c.cols))
		for i, rc := range c.cols {
			colls[i], _ = exprCollation(rc.e)
		}
		seen := map[string]bool{}
		kept := out[:0]
		for _, o := range out {
			k := collatedKey(o.vals, colls)
			if !seen[k] {
				seen[k] = true
				kept = append(kept, o)
			}
		}
		out = kept
	}
	return out
}

// loop runs visit for every combination of rows of sources k and on,
// the nested loops of a join.
func (r *runner) loop(k int, visit func()) {
	srcs := r.core.sc.sources
	if k == len(srcs) {
		visit()
		return
	}
	src := srcs[k]
	rows := src.rows
	useLookup := src.lookup != nil && src.join != "RIGHT" && src.join != "FULL"
	if useLookup {
		rows = src.lookup.find(r, src)
	}
	if src.tvf != nil {
		rows = src.tvf.rows(r)
	}
	any := false
	for i, row := range rows {
		r.cur[k] = row
		if src.on != nil && !truthy(r.eval(src.on)) {
			continue
		}
		any = true
		if src.matched != nil && !useLookup {
			src.matched[i] = true
		}
		r.loop(k+1, visit)
	}
	r.cur[k] = nil
	if !any && (src.join == "LEFT" || src.join == "FULL") {
		r.loop(k+1, visit)
	}
}

func (r *runner) emit() outRow {
	c := r.core
	vals := make([]Value, len(c.cols))
	for i, rc := range c.cols {
		vals[i] = r.eval(rc.e)
	}
	return outRow{vals: vals, keys: r.orderKeys(vals)}
}

func (r *runner) orderKeys(vals []Value) []Value {
	c := r.core
	if len(c.order) == 0 {
		return nil
	}
	keys := make([]Value, len(c.order))
	for i, t := range c.order {
		if t.pos != 0 {
			keys[i] = vals[t.pos-1]
		} else {
			keys[i] = r.eval(t.e)
		}
	}
	return keys
}

// orderLess compares two rows' ORDER BY keys. NULL comes first going up
// and last going down, unless NULLS FIRST or NULLS LAST says otherwise.
func orderLess(a, b []Value, order []orderTerm, colls []string) bool {
	for k, t := range order {
		x, y := a[k], b[k]
		if x == nil || y == nil {
			if x == nil && y == nil {
				continue
			}
			nullsFirst := t.nulls == 1 || (t.nulls == 0 && !t.desc)
			return (x == nil) == nullsFirst
		}
		d := compareColl(x, y, colls[k])
		if d != 0 {
			return (d < 0) != t.desc
		}
	}
	return false
}

// collatedKey is a string equal for values that compare equal under the
// collations: 'a' and 'A' under NOCASE.
func collatedKey(vals []Value, colls []string) string {
	if len(colls) == 0 {
		return rowKey(vals)
	}
	adj := make([]Value, len(vals))
	for i, v := range vals {
		adj[i] = v
		if s, ok := v.(string); ok && i < len(colls) {
			switch colls[i] {
			case "NOCASE":
				adj[i] = strings.Map(lowerASCII, s)
			case "RTRIM":
				adj[i] = strings.TrimRight(s, " ")
			}
		}
	}
	return rowKey(adj)
}

// ---- aggregates ---------------------------------------------------------------

type aggState struct {
	call  *callExpr
	count int64
	sum   Value // int64 while every value is an integer, else float64
	best  Value
	seen  bool
	parts []string
	seps  []string
	json  []*jnode // json_group_array / json_group_object values
	jkeys []*jnode
	dist  map[string]bool
}

func newAggStates(calls []*callExpr) []*aggState {
	out := make([]*aggState, len(calls))
	for i, c := range calls {
		out[i] = &aggState{call: c, dist: map[string]bool{}}
	}
	return out
}

// accumulate adds the current row to every aggregate, and reports which
// min/max aggregates found a new best value.
func (r *runner) accumulate() []bool {
	changed := make([]bool, len(r.aggs))
	for i, a := range r.aggs {
		c := a.call
		if c.filter != nil && !truthy(r.eval(c.filter)) {
			continue
		}
		if c.star {
			a.count++
			continue
		}
		v := r.eval(c.args[0])
		if v == nil && !strings.HasPrefix(c.name, "json_group") {
			continue
		}
		if c.distinct {
			coll, _ := exprCollation(c.args[0])
			k := collatedKey([]Value{v}, []string{coll})
			if a.dist[k] {
				continue
			}
			a.dist[k] = true
		}
		var second Value = ","
		if len(c.args) > 1 {
			second = r.eval(c.args[1])
		}
		changed[i] = a.add(v, second)
	}
	return changed
}

// add counts a non-NULL value into an aggregate, and reports whether a
// min() or max() found a new best value.
func (a *aggState) add(v Value, second Value) bool {
	c := a.call
	changed := false
	a.count++
	switch c.name {
	case "json_group_array":
		a.json = append(a.json, jsonArg(c.args[0], v))
	case "json_group_object":
		// Unlike json_object, any label works: it's used as text. A
		// NULL label leaves the row out.
		if v == nil {
			a.count--
			break
		}
		a.jkeys = append(a.jkeys, jsonString(textValue(v)))
		a.json = append(a.json, jsonArg(c.args[1], second))
	case "sum", "total", "avg":
		n := numericValue(v)
		if a.sum == nil {
			a.sum = n
		} else {
			a.sum = arith("+", a.sum, n)
		}
	case "min", "max":
		coll, _ := exprCollation(c.args[0])
		if !a.seen || (c.name == "min" && compareColl(v, a.best, coll) < 0) || (c.name == "max" && compareColl(v, a.best, coll) > 0) {
			a.best = v
			changed = true
		}
	case "group_concat", "string_agg":
		a.parts = append(a.parts, textValue(v))
		a.seps = append(a.seps, textValue(second))
	}
	a.seen = true
	return changed
}

func (r *runner) aggResult(a *aggState) Value {
	switch a.call.name {
	case "count":
		return a.count
	case "sum":
		if f, ok := a.sum.(float64); ok && f != f {
			return nil
		}
		return a.sum
	case "total":
		if a.sum == nil {
			return 0.0
		}
		return toFloat(a.sum)
	case "avg":
		if a.count == 0 {
			return nil
		}
		return toFloat(a.sum) / float64(a.count)
	case "min", "max":
		return a.best
	case "json_group_array":
		return (&jnode{kind: jArray, elems: a.json}).String()
	case "json_group_object":
		return (&jnode{kind: jObject, keys: a.jkeys, elems: a.json}).String()
	}
	// group_concat, string_agg: each separator goes before the value it
	// came with.
	if len(a.parts) == 0 {
		return nil
	}
	var sb strings.Builder
	for i, p := range a.parts {
		if i > 0 {
			sb.WriteString(a.seps[i])
		}
		sb.WriteString(p)
	}
	return sb.String()
}

func (r *runner) intValue(e expr, what string) int64 {
	v := r.eval(e)
	n, ok := applyAffinity(v, affInteger).(int64)
	if !ok {
		fail("SQL: %s needs a whole number", what)
	}
	return n
}

func itoa(n int) string {
	return textValue(int64(n))
}

// rowKey is a string that's equal for rows SQLite considers the same.
func rowKey(vals []Value) string {
	var sb strings.Builder
	for _, v := range vals {
		switch x := v.(type) {
		case nil:
			sb.WriteString("n|")
		case int64:
			sb.WriteString("i" + strconv.FormatInt(x, 10) + "|")
		case float64:
			if x == math.Trunc(x) && math.Abs(x) < 1e18 {
				sb.WriteString("i" + strconv.FormatInt(int64(x), 10) + "|")
			} else {
				sb.WriteString("f" + strconv.FormatFloat(x, 'g', -1, 64) + "|")
			}
		case string:
			sb.WriteString("t" + strconv.Quote(x) + "|")
		case []byte:
			sb.WriteString("b" + strconv.Quote(string(x)) + "|")
		}
	}
	return sb.String()
}
