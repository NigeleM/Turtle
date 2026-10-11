// Copyright 2017-2026 Nigele McCoy
// Licensed under the Apache License, Version 2.0;
// see LICENSE and NOTICE.

package sqlite

import (
	"strings"
)

// Triggers: statements the database runs by itself when rows change.
//
//	CREATE TRIGGER stock_log AFTER UPDATE OF stock ON books
//	WHEN NEW.stock < OLD.stock
//	BEGIN
//	    INSERT INTO log (sku, sold) VALUES (NEW.sku, OLD.stock - NEW.stock);
//	END
//
// BEFORE, AFTER, or INSTEAD OF (on a view, which makes it writable);
// INSERT, UPDATE [OF columns] or DELETE; FOR EACH ROW (the only kind);
// an optional WHEN. NEW is the row as it will be (INSERT, UPDATE), OLD as
// it was (UPDATE, DELETE). RAISE(IGNORE) skips the row; RAISE(ABORT,
// 'message'), RAISE(FAIL, ...) and RAISE(ROLLBACK, ...) stop with an
// error. A trigger doesn't set itself off again (recursive_triggers is
// off, as by default in SQLite).

type Trigger struct {
	Name    string
	Table   string
	Timing  string // BEFORE, AFTER, INSTEAD OF
	Event   string // INSERT, UPDATE, DELETE
	Cols    []string
	When    expr
	Body    []any
	SQL     string
	tblPos  int // where the table's name is in SQL, for RENAME
	tblEnd  int
	namePos int
	refs    []token // names of tables in the body
}

type raiseExpr struct {
	kind string // IGNORE, ROLLBACK, ABORT, FAIL
	msg  expr
}

// raiseIgnore unwinds a trigger that said RAISE(IGNORE).
type raiseIgnore struct{}

func (p *sqlParser) parseRaise() expr {
	p.expectOp("(")
	r := &raiseExpr{kind: strings.ToUpper(p.expectName())}
	switch r.kind {
	case "IGNORE":
	case "ROLLBACK", "ABORT", "FAIL":
		p.expectOp(",")
		r.msg = p.parseExpr()
	default:
		fail("SQL: RAISE takes IGNORE, ROLLBACK, ABORT or FAIL")
	}
	p.expectOp(")")
	return r
}

func (r *runner) evalRaise(x *raiseExpr) Value {
	if r.db.trig == nil {
		fail("RAISE() may only be used within a trigger-program")
	}
	if x.kind == "IGNORE" {
		panic(raiseIgnore{})
	}
	msg := textValue(r.eval(x.msg))
	switch x.kind {
	case "ROLLBACK":
		panic(&Error{Msg: rollbackMark + msg})
	case "FAIL":
		panic(&Error{Msg: failMark + msg})
	}
	panic(&Error{Msg: msg})
}

// parseTrigger reads CREATE TRIGGER, after CREATE.
func (p *sqlParser) parseTrigger() *Trigger {
	tr := &Trigger{Timing: "BEFORE"}
	p.expectWord("TRIGGER")
	if p.acceptWord("IF") {
		p.expectWord("NOT")
		p.expectWord("EXISTS")
	}
	tr.namePos = p.peek().pos
	tr.Name = p.expectName()
	if p.acceptOp(".") {
		tr.namePos = p.peek().pos
		tr.Name = p.expectName()
	}
	switch {
	case p.acceptWord("BEFORE"):
	case p.acceptWord("AFTER"):
		tr.Timing = "AFTER"
	case p.acceptWord("INSTEAD"):
		p.expectWord("OF")
		tr.Timing = "INSTEAD OF"
	}
	switch {
	case p.acceptWord("INSERT"):
		tr.Event = "INSERT"
	case p.acceptWord("DELETE"):
		tr.Event = "DELETE"
	case p.acceptWord("UPDATE"):
		tr.Event = "UPDATE"
		if p.acceptWord("OF") {
			for {
				tr.Cols = append(tr.Cols, p.expectName())
				if !p.acceptOp(",") {
					break
				}
			}
		}
	default:
		p.near("expected INSERT, UPDATE or DELETE")
	}
	p.expectWord("ON")
	tr.tblPos = p.peek().pos
	tr.Table = p.expectName()
	if p.acceptOp(".") {
		tr.tblPos = p.peek().pos
		tr.Table = p.expectName()
	}
	tr.tblEnd = p.toks[p.pos-1].end
	if p.acceptWord("FOR") {
		p.expectWord("EACH")
		p.expectWord("ROW")
	}
	if p.acceptWord("WHEN") {
		tr.When = p.parseExpr()
	}
	p.expectWord("BEGIN")
	p.onTbl = func(tk token) { tr.refs = append(tr.refs, tk) }
	defer func() { p.onTbl = nil }()
	for !p.acceptWord("END") {
		if p.peek().kind == tEOF {
			fail("SQL: a trigger's BEGIN needs an END")
		}
		st := p.parseStatement()
		switch st.(type) {
		case *insertStmt, *updateStmt, *deleteStmt, *selectStmt:
		default:
			fail("SQL: a trigger can only run INSERT, UPDATE, DELETE and SELECT")
		}
		tr.Body = append(tr.Body, st)
		p.expectOp(";")
	}
	return tr
}

// trigCtx is a running trigger: its NEW and OLD rows, as a scope its
// statements see and the runner that holds the rows.
type trigCtx struct {
	trig   *Trigger
	sc     *scope
	r      *runner
	parent *trigCtx
}

func (f *dbFile) trigScope() *scope {
	if f.trig == nil {
		return nil
	}
	return f.trig.sc
}

func (f *dbFile) trigRunner() *runner {
	if f.trig == nil {
		return nil
	}
	return f.trig.r
}

// triggersFor lists a table's (or view's) triggers.
func (f *dbFile) triggersFor(name string) []*Trigger {
	return f.triggers[strings.ToLower(name)]
}

// fire runs the matching triggers for one row. cols is the table's (or
// view's) columns; old and new are nil where the event has none. It
// reports false when a trigger said RAISE(IGNORE).
func (f *dbFile) fire(target string, cols []Column, rowidCol int, timing, event string, oldRowid int64, old []Value, newRowid int64, new []Value, changed map[string]bool) bool {
	for _, tr := range f.triggersFor(target) {
		if tr.Timing != timing || tr.Event != event {
			continue
		}
		if len(tr.Cols) > 0 && changed != nil {
			hit := false
			for _, c := range tr.Cols {
				if changed[strings.ToLower(c)] {
					hit = true
				}
			}
			if !hit {
				continue
			}
		}
		if f.activeTrigger(tr) {
			continue
		}
		if !f.runTrigger(tr, cols, rowidCol, oldRowid, old, newRowid, new) {
			return false
		}
	}
	return true
}

func (f *dbFile) activeTrigger(tr *Trigger) bool {
	for c := f.trig; c != nil; c = c.parent {
		if c.trig == tr {
			return true
		}
	}
	return false
}

func (f *dbFile) runTrigger(tr *Trigger, cols []Column, rowidCol int, oldRowid int64, old []Value, newRowid int64, new []Value) (ok bool) {
	q := &queryPlan{db: f}
	sc := &scope{owner: q}
	q.sc = sc
	var cur []*srcRow
	add := func(name string, rowid int64, vals []Value) {
		if vals == nil {
			return
		}
		sc.sources = append(sc.sources, &source{name: name, cols: cols, rowidCol: rowidCol, hidden: map[string]bool{}, qualifiedOnly: true, table: &Table{Name: name, Columns: cols, RowidCol: rowidCol}})
		cur = append(cur, &srcRow{rowid: rowid, vals: vals})
	}
	add("new", newRowid, new)
	add("old", oldRowid, old)
	c := &corePlan{q: q, s: &selectStmt{}, sc: sc, minmax: -1}
	r := &runner{db: f, core: c, cur: cur}
	ctx := &trigCtx{trig: tr, sc: sc, r: r, parent: f.trig}
	f.trig = ctx
	defer func() {
		f.trig = ctx.parent
		if rec := recover(); rec != nil {
			if _, ignore := rec.(raiseIgnore); ignore {
				ok = false
				return
			}
			panic(rec)
		}
	}()
	if tr.When != nil {
		c.resolve(tr.When, false)
		if !truthy(r.eval(tr.When)) {
			return true
		}
	}
	for _, st := range tr.Body {
		if s, isSelect := st.(*selectStmt); isSelect {
			f.planQuery(s, sc, nil, nil).rows(nil, r)
			continue
		}
		f.run(st, nil)
	}
	return true
}

// rowCols makes the change map UPDATE OF triggers look at.
func setNames(sets []setClause) map[string]bool {
	m := map[string]bool{}
	for _, s := range sets {
		m[strings.ToLower(s.col)] = true
	}
	return m
}

// ---- writing to a view through INSTEAD OF triggers ----

func (f *dbFile) viewColumns(v *View) []Column {
	q := f.planView(v, nil)
	cols := make([]Column, len(q.names))
	for i, n := range q.names {
		cols[i] = Column{Name: n}
	}
	return cols
}

func (f *dbFile) insteadOf(view *View, event string) {
	for _, tr := range f.triggersFor(view.Name) {
		if tr.Timing == "INSTEAD OF" && tr.Event == event {
			return
		}
	}
	fail("cannot modify %s because it is a view", view.Name)
}

// viewInsert runs INSTEAD OF INSERT for each row of an INSERT into a view.
func (f *dbFile) viewInsert(view *View, s *insertStmt, params []Value) Result {
	f.insteadOf(view, "INSERT")
	cols := f.viewColumns(view)
	target := make([]int, 0, len(cols))
	if s.cols != nil {
		for _, n := range s.cols {
			ci := colIndex(cols, n)
			if ci < 0 {
				fail("table %s has no column named %s", view.Name, n)
			}
			target = append(target, ci)
		}
	} else {
		for i := range cols {
			target = append(target, i)
		}
	}
	rows := [][]Value{{}}
	if s.src != nil {
		q := f.planQuery(s.src, f.trigScope(), nil, nil)
		if len(q.names) != len(target) {
			fail("%d values for %d columns", len(q.names), len(target))
		}
		rows = q.rows(params, f.trigRunner())
	}
	var res Result
	for _, src := range rows {
		vals := make([]Value, len(cols))
		for i, ci := range target {
			if i < len(src) {
				vals[ci] = src[i]
			}
		}
		if f.fire(view.Name, cols, -1, "INSTEAD OF", "INSERT", 0, nil, 0, vals, nil) {
			res.Changes++
		}
	}
	return res
}

// viewRows reads a view's rows that WHERE selects, with extra
// expressions after them (an UPDATE's new values).
func (f *dbFile) viewRows(view *View, alias string, where expr, extra []expr, params []Value) [][]Value {
	cols := []resultCol{{star: true}}
	for _, e := range extra {
		cols = append(cols, resultCol{e: e})
	}
	s := &selectStmt{cols: cols, from: []fromItem{{name: view.Name, alias: alias}}, where: where}
	return f.planQuery(s, f.trigScope(), nil, nil).rows(params, f.trigRunner())
}

func (f *dbFile) viewUpdate(view *View, s *updateStmt, params []Value) Result {
	f.insteadOf(view, "UPDATE")
	cols := f.viewColumns(view)
	var extra []expr
	var idx []int
	for _, set := range s.sets {
		ci := colIndex(cols, set.col)
		if ci < 0 {
			fail("no such column: %s", set.col)
		}
		idx = append(idx, ci)
		extra = append(extra, set.e)
	}
	var res Result
	for _, row := range f.viewRows(view, s.alias, s.where, extra, params) {
		old := row[:len(cols)]
		nv := append([]Value{}, old...)
		for i, ci := range idx {
			nv[ci] = row[len(cols)+i]
		}
		if f.fire(view.Name, cols, -1, "INSTEAD OF", "UPDATE", 0, old, 0, nv, setNames(s.sets)) {
			res.Changes++
		}
	}
	return res
}

func (f *dbFile) viewDelete(view *View, s *deleteStmt, params []Value) Result {
	f.insteadOf(view, "DELETE")
	cols := f.viewColumns(view)
	var res Result
	for _, row := range f.viewRows(view, s.alias, s.where, nil, params) {
		if f.fire(view.Name, cols, -1, "INSTEAD OF", "DELETE", 0, row, 0, nil, nil) {
			res.Changes++
		}
	}
	return res
}
