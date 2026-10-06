package sqlite

import (
	"encoding/binary"
	"slices"
	"strings"
)

// Result is what Exec did.
type Result struct {
	Changes   int64 // rows inserted, updated or deleted by the last statement
	LastRowid int64 // rowid of the last row inserted
	Cols      []string
	Rows      [][]Value // rows from RETURNING
}

// Exec runs statements that change the database: INSERT, UPDATE, DELETE,
// CREATE TABLE/INDEX/VIEW, DROP, ALTER TABLE, and BEGIN/COMMIT/ROLLBACK.
// Several statements can be given at once, separated by ";", when no ?
// values are passed.
//
// Each statement is all-or-nothing: if it fails, none of its changes are
// kept. Outside BEGIN ... COMMIT each statement is also its own
// transaction, saved to the file (through the journal) before Exec
// returns.
func (db *DB) Exec(sql string, params []Value) (res Result, err error) {
	defer catch(&err)
	if db.closed {
		fail("the database is closed")
	}
	p := newParser(sql)
	var stmts []any
	for {
		for p.acceptOp(";") {
		}
		if p.peek().kind == tEOF {
			break
		}
		stmts = append(stmts, p.parseStatement())
		if !p.acceptOp(";") && p.peek().kind != tEOF {
			fail("SQL: unexpected %q after the end of the statement", p.peek().text)
		}
	}
	if len(stmts) == 0 {
		fail("SQL: the statement is empty")
	}
	if len(stmts) > 1 && len(params) > 0 {
		fail("SQL: ? values can only be given with a single statement")
	}
	p.checkParams(params)
	for _, st := range stmts {
		res, err = db.execOne(st, params)
		if err != nil {
			return res, err
		}
	}
	return res, nil
}

// ExecRows runs one changing statement once per row of values, as a
// single statement: it is parsed once, and if any row fails, none of the
// rows' changes are kept (even inside BEGIN ... COMMIT). Outside a
// transaction everything is saved in one write. Changes is the total.
func (db *DB) ExecRows(sql string, rows [][]Value) (res Result, err error) {
	defer catch(&err)
	if db.closed {
		fail("the database is closed")
	}
	p := newParser(sql)
	st := p.parseStatement()
	p.endStatement()
	switch st.(type) {
	case *insertStmt, *updateStmt, *deleteStmt:
	default:
		fail("SQL: ExecRows runs INSERT, UPDATE or DELETE")
	}
	for _, r := range rows {
		if len(r) != p.params {
			fail("SQL: the statement has %d ? placeholder(s) but a row has %d value(s)", p.params, len(r))
		}
	}
	return db.execOneRows(st, rows)
}

// execOne runs one statement in its own statement transaction.
func (db *DB) execOne(st any, params []Value) (res Result, err error) {
	f := db.dbFile
	switch s := st.(type) {
	case *txStmt:
		switch s.op {
		case "BEGIN":
			if f.explicit {
				fail("cannot start a transaction within a transaction")
			}
			if err := f.beginWrite(); err != nil {
				return res, err
			}
			f.explicit = true
		case "COMMIT":
			if !f.explicit {
				fail("cannot commit - no transaction is active")
			}
			if f.fkDeferred > 0 {
				// The transaction stays open: fix the rows, or ROLLBACK.
				return res, errorf("FOREIGN KEY constraint failed")
			}
			if err := f.commit(); err != nil {
				// If nothing was written the transaction is still open:
				// COMMIT again, or ROLLBACK.
				return res, err
			}
		case "ROLLBACK":
			if !f.explicit {
				fail("cannot rollback - no transaction is active")
			}
			f.rollbackTx()
		case "SAVEPOINT":
			started := false
			if !f.explicit {
				if err := f.beginWrite(); err != nil {
					return res, err
				}
				f.explicit, started = true, true
			}
			f.savepoints = append(f.savepoints, &savepoint{name: s.name, undo: f.newUndo(), startedTx: started})
		case "RELEASE", "ROLLBACK TO":
			i := len(f.savepoints) - 1
			for ; i >= 0 && !strings.EqualFold(f.savepoints[i].name, s.name); i-- {
			}
			if i < 0 {
				fail("no such savepoint: %s", s.name)
			}
			sp := f.savepoints[i]
			if s.op == "ROLLBACK TO" {
				// Back to the savepoint, which stays.
				f.savepoints = f.savepoints[:i+1]
				f.undo(sp.undo)
				f.stmt = nil
				sp.undo = f.newUndo()
				break
			}
			f.savepoints = f.savepoints[:i]
			if sp.startedTx {
				if f.fkDeferred > 0 {
					f.savepoints = append(f.savepoints, sp)
					return res, errorf("FOREIGN KEY constraint failed")
				}
				if err := f.commit(); err != nil {
					return res, err
				}
			}
		}
		return res, nil
	case *selectStmt:
		fail("SQL: SELECT reads the database; run it with Query, not Exec")
	case *pragmaStmt:
		_, _, err := db.runPragma(s)
		return res, err
	case *vacuumStmt:
		return res, db.runVacuum(s, params)
	case *attachStmt:
		return res, db.runAttach(s, params)
	case *detachStmt:
		return res, db.runDetach(s)
	}
	return db.execOneRows(st, [][]Value{params})
}

// execOneRows runs a changing statement once per row of ? values, all as
// one statement.
func (db *DB) execOneRows(st any, rows [][]Value) (res Result, err error) {
	f := db.dbFile
	auto := !f.inWrite
	if err := f.beginWrite(); err != nil {
		return res, err
	}
	release, aerr := f.readAttached()
	if aerr != nil {
		if auto {
			f.rollbackTx()
		}
		return res, aerr
	}
	defer release()
	f.startStatement()
	func() {
		defer func() {
			if r := recover(); r != nil {
				e, ok := r.(*Error)
				if !ok {
					f.undoStatement()
					if auto {
						f.rollbackTx()
					}
					panic(r)
				}
				err = e
			}
		}()
		for _, params := range rows {
			r := f.run(st, params)
			res.Changes += r.Changes
			if r.LastRowid != 0 {
				res.LastRowid = r.LastRowid
			}
			res.Cols = r.Cols
			res.Rows = append(res.Rows, r.Rows...)
		}
	}()
	if err != nil {
		e := err.(*Error)
		switch {
		case strings.HasPrefix(e.Msg, rollbackMark):
			// ON CONFLICT ROLLBACK ends the whole transaction.
			f.rollbackTx()
			return res, &Error{Msg: strings.TrimPrefix(e.Msg, rollbackMark)}
		case strings.HasPrefix(e.Msg, failMark):
			// ON CONFLICT FAIL keeps the statement's earlier changes.
			err = &Error{Msg: strings.TrimPrefix(e.Msg, failMark)}
			f.stmt = nil
		default:
			f.undoStatement()
		}
		if auto {
			if cerr := f.commit(); cerr != nil {
				f.rollbackTx()
				return res, cerr
			}
		}
		return res, err
	}
	if f.fkStmt > 0 || (auto && f.fkDeferred > 0) {
		f.undoStatement()
		if auto {
			f.rollbackTx()
		}
		return Result{}, errorf("FOREIGN KEY constraint failed")
	}
	f.stmt = nil
	switch st.(type) {
	case *insertStmt, *updateStmt, *deleteStmt:
		f.changes = res.Changes
		f.totalChanges += res.Changes
		if res.LastRowid != 0 {
			f.lastRowid = res.LastRowid
		}
	}
	if auto {
		if err := f.commit(); err != nil {
			f.rollbackTx()
			return res, err
		}
	}
	return res, nil
}

const (
	rollbackMark = "\x00rollback\x00"
	failMark     = "\x00fail\x00"
)

// run dispatches one changing statement.
func (f *dbFile) run(st any, params []Value) Result {
	switch s := st.(type) {
	case *insertStmt:
		return f.runInsert(s, params)
	case *updateStmt:
		return f.runUpdate(s, params)
	case *deleteStmt:
		return f.runDelete(s, params)
	case *createTableStmt:
		f.runCreateTable(s, params)
	case *createIndexStmt:
		f.runCreateIndex(s)
	case *createViewStmt:
		f.runCreateView(s)
	case *createTriggerStmt:
		f.runCreateTrigger(s)
	case *dropStmt:
		f.runDrop(s)
	case *alterStmt:
		f.runAlter(s)
	case *headerWrite:
		binary.BigEndian.PutUint32(f.writable(1)[s.off:], s.v)
	case *byteWrite:
		copy(f.writable(1)[s.off:], s.v)
	default:
		fail("internal error: unknown statement")
	}
	return Result{}
}

// ---- statements ----

type txStmt struct{ op, name string }

type setClause struct {
	col string
	e   expr
}

type upsert struct {
	target    []string
	targetWhr expr
	nothing   bool
	sets      []setClause
	where     expr
}

type insertStmt struct {
	with      []*cteDef
	recursive bool
	or        string
	table     string
	alias     string
	cols      []string
	src       *selectStmt // VALUES or SELECT; nil for DEFAULT VALUES
	upserts   []*upsert
	returning []resultCol
}

type updateStmt struct {
	with      []*cteDef
	recursive bool
	or        string
	table     string
	alias     string
	sets      []setClause
	from      []fromItem // UPDATE ... FROM: other tables joined in
	where     expr
	returning []resultCol
}

type deleteStmt struct {
	with      []*cteDef
	recursive bool
	table     string
	alias     string
	where     expr
	returning []resultCol
}

type createTableStmt struct {
	sql         string // as stored in sqlite_schema
	name        string
	ifNotExists bool
	as          *selectStmt
}

type createIndexStmt struct {
	sql         string
	name        string
	table       string
	ifNotExists bool
	src         string
}

type createViewStmt struct {
	sql         string
	name        string
	ifNotExists bool
	sel         *selectStmt
}

type createTriggerStmt struct {
	tr          *Trigger
	sql         string
	ifNotExists bool
}

type dropStmt struct {
	kind     string
	name     string
	ifExists bool
}

type alterStmt struct {
	table   string
	op      string // RENAME, ADD, RENAME COLUMN, DROP COLUMN
	newName string
	col     string
	colDef  string
}

// parseStatement parses one statement of any kind.
func (p *sqlParser) parseStatement() any {
	start := p.peek().pos
	switch {
	case p.isWord("SELECT", "VALUES"):
		return p.parseSelect()
	case p.isWord("WITH"):
		at := p.pos
		p.next()
		with, rec := p.parseWithList()
		switch {
		case p.isWord("INSERT", "REPLACE"):
			s := p.parseInsert()
			s.with, s.recursive = with, rec
			return s
		case p.acceptWord("UPDATE"):
			s := p.parseUpdate()
			s.with, s.recursive = with, rec
			return s
		case p.acceptWord("DELETE"):
			s := p.parseDelete()
			s.with, s.recursive = with, rec
			return s
		}
		p.pos = at
		return p.parseSelect()
	case p.acceptWord("BEGIN"):
		if !p.acceptWord("DEFERRED") && !p.acceptWord("IMMEDIATE") {
			p.acceptWord("EXCLUSIVE")
		}
		if p.acceptWord("TRANSACTION") && p.peek().kind == tIdent {
			p.next()
		}
		return &txStmt{op: "BEGIN"}
	case p.isWord("COMMIT", "END"):
		p.next()
		p.acceptWord("TRANSACTION")
		return &txStmt{op: "COMMIT"}
	case p.acceptWord("ROLLBACK"):
		p.acceptWord("TRANSACTION")
		if p.acceptWord("TO") {
			p.acceptWord("SAVEPOINT")
			return &txStmt{op: "ROLLBACK TO", name: p.expectName()}
		}
		return &txStmt{op: "ROLLBACK"}
	case p.acceptWord("SAVEPOINT"):
		return &txStmt{op: "SAVEPOINT", name: p.expectName()}
	case p.acceptWord("RELEASE"):
		p.acceptWord("SAVEPOINT")
		return &txStmt{op: "RELEASE", name: p.expectName()}
	case p.isWord("INSERT", "REPLACE"):
		return p.parseInsert()
	case p.acceptWord("UPDATE"):
		return p.parseUpdate()
	case p.acceptWord("DELETE"):
		return p.parseDelete()
	case p.acceptWord("CREATE"):
		return p.parseCreate(start)
	case p.acceptWord("DROP"):
		d := &dropStmt{}
		if !p.isWord("TABLE", "INDEX", "VIEW", "TRIGGER") {
			p.near("expected TABLE, INDEX, VIEW or TRIGGER")
		}
		d.kind = strings.ToUpper(p.next().text)
		if p.acceptWord("IF") {
			p.expectWord("EXISTS")
			d.ifExists = true
		}
		d.name = p.expectName()
		if p.acceptOp(".") {
			d.name = p.expectName()
		}
		return d
	case p.acceptWord("ALTER"):
		p.expectWord("TABLE")
		a := &alterStmt{table: p.expectName()}
		if p.acceptOp(".") {
			a.table = p.expectName()
		}
		switch {
		case p.acceptWord("RENAME"):
			if p.acceptWord("TO") {
				a.op, a.newName = "RENAME", p.expectName()
				break
			}
			p.acceptWord("COLUMN")
			a.op, a.col = "RENAME COLUMN", p.expectName()
			p.expectWord("TO")
			a.newName = p.expectName()
		case p.acceptWord("ADD"):
			p.acceptWord("COLUMN")
			a.op = "ADD"
			s := p.peek().pos
			for !p.atStatementEnd() {
				p.next()
			}
			a.colDef = strings.TrimSpace(p.src[s:p.toks[p.pos-1].end])
			if a.colDef == "" {
				p.near("expected a column definition")
			}
		case p.acceptWord("DROP"):
			p.acceptWord("COLUMN")
			a.op, a.col = "DROP COLUMN", p.expectName()
		default:
			p.near("expected RENAME, ADD or DROP")
		}
		return a
	case p.acceptWord("PRAGMA"):
		return p.parsePragma()
	case p.acceptWord("VACUUM"):
		s := &vacuumStmt{}
		if p.peek().kind == tIdent && !p.isWord("INTO") {
			p.next() // schema name
		}
		if p.acceptWord("INTO") {
			s.into = p.parseExpr()
		}
		return s
	case p.acceptWord("ATTACH"):
		return p.parseAttach()
	case p.acceptWord("DETACH"):
		p.acceptWord("DATABASE")
		return &detachStmt{name: p.expectName()}
	case p.isWord("ANALYZE", "REINDEX"):
		fail("SQL: %s isn't supported", strings.ToUpper(p.peek().text))
	case p.peek().kind == tEOF:
		fail("SQL: the statement is empty")
	}
	p.near("unknown statement")
	return nil
}

// tableName reads [schema.]table, the table a statement changes.
func (p *sqlParser) tableName() string {
	t := p.peek()
	name := p.expectName()
	if p.acceptOp(".") {
		if !strings.EqualFold(name, "main") {
			fail("attached databases are read-only in Turtle: change tables in the main database (%s.%s)", name, p.peek().text)
		}
		t = p.peek()
		name = p.expectName()
	}
	if p.onTbl != nil {
		p.onTbl(t)
	}
	return name
}

func (p *sqlParser) parseConflictOr() string {
	if !p.acceptWord("OR") {
		return ""
	}
	w := strings.ToUpper(p.expectName())
	switch w {
	case "ROLLBACK", "ABORT", "FAIL", "IGNORE", "REPLACE":
		return w
	}
	fail("SQL: unknown conflict resolution %s", w)
	return ""
}

func (p *sqlParser) parseInsert() *insertStmt {
	s := &insertStmt{}
	if p.acceptWord("REPLACE") {
		s.or = "REPLACE"
	} else {
		p.expectWord("INSERT")
		s.or = p.parseConflictOr()
	}
	p.expectWord("INTO")
	s.table = p.tableName()
	if p.acceptWord("AS") {
		s.alias = p.expectName()
	}
	if p.acceptOp("(") {
		for {
			s.cols = append(s.cols, p.expectName())
			if !p.acceptOp(",") {
				break
			}
		}
		p.expectOp(")")
	}
	switch {
	case p.acceptWord("DEFAULT"):
		p.expectWord("VALUES")
	case p.isWord("VALUES", "SELECT", "WITH"):
		s.src = p.parseSelect()
	default:
		p.near("expected VALUES, SELECT or DEFAULT VALUES")
	}
	for p.isWord("ON") && isWord(p.peekAt(1), "CONFLICT") {
		if s.src == nil {
			fail("SQL: ON CONFLICT can't follow DEFAULT VALUES")
		}
		p.next()
		p.next()
		u := &upsert{}
		if p.acceptOp("(") {
			for {
				e := p.parseExpr()
				if c, ok := e.(*collateExpr); ok {
					e = c.x
				}
				col, ok := e.(*colExpr)
				if !ok {
					fail("SQL: ON CONFLICT ( ) lists column names")
				}
				u.target = append(u.target, col.name)
				p.acceptWord("ASC")
				p.acceptWord("DESC")
				if !p.acceptOp(",") {
					break
				}
			}
			p.expectOp(")")
			if p.acceptWord("WHERE") {
				u.targetWhr = p.parseExpr()
			}
		}
		p.expectWord("DO")
		if p.acceptWord("NOTHING") {
			u.nothing = true
		} else {
			p.expectWord("UPDATE")
			p.expectWord("SET")
			u.sets = p.parseSets()
			if p.acceptWord("WHERE") {
				u.where = p.parseExpr()
			}
		}
		s.upserts = append(s.upserts, u)
	}
	s.returning = p.parseReturning()
	return s
}

func (p *sqlParser) parseSets() []setClause {
	var sets []setClause
	for {
		if p.isOp("(") {
			fail("SQL: SET (a, b) = ... isn't supported; set each column on its own")
		}
		col := p.expectName()
		p.expectOp("=")
		sets = append(sets, setClause{col: col, e: p.parseExpr()})
		if !p.acceptOp(",") {
			break
		}
	}
	return sets
}

func (p *sqlParser) parseReturning() []resultCol {
	if !p.acceptWord("RETURNING") {
		return nil
	}
	var cols []resultCol
	for {
		if p.acceptOp("*") {
			cols = append(cols, resultCol{star: true})
		} else {
			start := p.peek().pos
			e := p.parseExpr()
			c := resultCol{e: e, name: strings.TrimSpace(p.src[start:p.toks[p.pos-1].end])}
			if col, ok := e.(*colExpr); ok {
				c.name, c.bare = col.name, true
			}
			if p.acceptWord("AS") {
				c.name = p.expectName()
			} else if p.peek().kind == tIdent && !p.isWord(clauseWords...) {
				c.name = p.next().text
			}
			cols = append(cols, c)
		}
		if !p.acceptOp(",") {
			break
		}
	}
	return cols
}

func (p *sqlParser) parseUpdate() *updateStmt {
	s := &updateStmt{or: p.parseConflictOr()}
	s.table = p.tableName()
	if p.acceptWord("AS") {
		s.alias = p.expectName()
	} else if p.peek().kind == tIdent && !p.isWord("SET", "INDEXED", "NOT") {
		s.alias = p.next().text
	}
	if p.acceptWord("INDEXED") {
		p.expectWord("BY")
		p.expectName()
	} else if p.isWord("NOT") {
		p.next()
		p.expectWord("INDEXED")
	}
	p.expectWord("SET")
	s.sets = p.parseSets()
	if p.acceptWord("FROM") {
		s.from = p.parseFromList()
	}
	if p.acceptWord("WHERE") {
		s.where = p.parseExpr()
	}
	s.returning = p.parseReturning()
	if p.isWord("ORDER", "LIMIT") {
		fail("SQL: UPDATE with ORDER BY or LIMIT isn't supported")
	}
	return s
}

func (p *sqlParser) parseDelete() *deleteStmt {
	p.expectWord("FROM")
	s := &deleteStmt{table: p.tableName()}
	if p.acceptWord("AS") {
		s.alias = p.expectName()
	} else if p.peek().kind == tIdent && !p.isWord("WHERE", "RETURNING", "INDEXED", "NOT") {
		s.alias = p.next().text
	}
	if p.acceptWord("INDEXED") {
		p.expectWord("BY")
		p.expectName()
	} else if p.isWord("NOT") && isWord(p.peekAt(1), "INDEXED") {
		p.next()
		p.next()
	}
	if p.acceptWord("WHERE") {
		s.where = p.parseExpr()
	}
	s.returning = p.parseReturning()
	if p.isWord("ORDER", "LIMIT") {
		fail("SQL: DELETE with ORDER BY or LIMIT isn't supported")
	}
	return s
}

// statementEnd moves to the end of the statement and returns where it
// ends in the source.
func (p *sqlParser) statementEnd() int {
	for !p.atStatementEnd() {
		if p.isOp("(") {
			p.skipParens()
			continue
		}
		p.next()
	}
	return p.toks[p.pos-1].end
}

func (p *sqlParser) parseCreate(start int) any {
	from := p.pos - 1 // the CREATE token
	if p.isWord("TEMP", "TEMPORARY") {
		fail("SQL: TEMP tables, indexes and views aren't supported")
	}
	switch {
	case p.isWord("TABLE"):
		p.next()
		ifNot := false
		if p.acceptWord("IF") {
			p.expectWord("NOT")
			p.expectWord("EXISTS")
			ifNot = true
		}
		namePos := p.peek().pos
		name := p.expectName()
		if p.acceptOp(".") {
			namePos = p.peek().pos
			name = p.expectName()
		}
		if p.acceptWord("AS") {
			return &createTableStmt{name: name, ifNotExists: ifNot, as: p.parseSelect()}
		}
		p.pos = from
		d := parseTableDef(p)
		end := p.toks[p.pos-1].end
		_ = namePos
		return &createTableStmt{sql: "CREATE TABLE " + p.src[d.namePos:end], name: d.t.Name, ifNotExists: d.ifNotExists}
	case p.isWord("UNIQUE", "INDEX"):
		unique := p.acceptWord("UNIQUE")
		p.expectWord("INDEX")
		s := &createIndexStmt{}
		if p.acceptWord("IF") {
			p.expectWord("NOT")
			p.expectWord("EXISTS")
			s.ifNotExists = true
		}
		namePos := p.peek().pos
		s.name = p.expectName()
		if p.acceptOp(".") {
			namePos = p.peek().pos
			s.name = p.expectName()
		}
		p.expectWord("ON")
		s.table = p.expectName()
		end := p.statementEnd()
		kw := "CREATE INDEX "
		if unique {
			kw = "CREATE UNIQUE INDEX "
		}
		s.sql = kw + p.src[namePos:end]
		s.src = p.src[p.toks[from].pos:end]
		return s
	case p.isWord("VIEW"):
		p.next()
		s := &createViewStmt{}
		if p.acceptWord("IF") {
			p.expectWord("NOT")
			p.expectWord("EXISTS")
			s.ifNotExists = true
		}
		namePos := p.peek().pos
		s.name = p.expectName()
		if p.acceptOp(".") {
			namePos = p.peek().pos
			s.name = p.expectName()
		}
		if p.isOp("(") {
			p.skipParens()
		}
		p.expectWord("AS")
		s.sel = p.parseSelect()
		s.sql = "CREATE VIEW " + p.src[namePos:p.toks[p.pos-1].end]
		return s
	case p.isWord("TRIGGER"):
		tr := p.parseTrigger()
		end := p.toks[p.pos-1].end
		ifNot := strings.Contains(strings.ToUpper(p.src[p.toks[from].pos:tr.namePos]), "IF")
		return &createTriggerStmt{tr: tr, sql: "CREATE TRIGGER " + p.src[tr.namePos:end], ifNotExists: ifNot}
	case p.isWord("VIRTUAL"):
		fail("SQL: virtual tables aren't supported")
	}
	p.near("expected TABLE, INDEX or VIEW")
	return nil
}

// ---- helpers for changing rows ----

// tableFor finds a table that may be changed.
func (f *dbFile) tableFor(name string) *Table {
	t, err := f.table(name)
	if err != nil {
		panic(err)
	}
	if t == schemaTable {
		fail("table sqlite_schema may not be modified")
	}
	if t.unsupported != "" {
		fail("table %s can't be changed: %s", t.Name, t.unsupported)
	}
	if t.WithoutRowid {
		fail("table %s is a WITHOUT ROWID table, which Turtle can read but not change yet", t.Name)
	}
	return t
}

// rowPlan is a planned scope with one source: the table being changed
// (and, for an upsert, the "excluded" row after it).
func (f *dbFile) rowPlan(t *Table, alias string, excluded bool) *corePlan {
	q := &queryPlan{db: f}
	c := &corePlan{q: q, s: &selectStmt{}, minmax: -1}
	c.sc = &scope{owner: q, parent: f.trigScope()}
	q.sc = c.sc
	name := t.Name
	if alias != "" {
		name = alias
	}
	c.sc.sources = append(c.sc.sources, &source{name: name, table: t, cols: t.Columns, rowidCol: t.RowidCol, hidden: map[string]bool{}})
	if excluded {
		c.sc.sources = append(c.sc.sources, &source{name: "excluded", cols: t.Columns, rowidCol: -1, hidden: map[string]bool{}, qualifiedOnly: true})
	}
	return c
}

// resolveOn binds e in plan c, refusing aggregates.
func (c *corePlan) resolveRow(e expr, what string) {
	c.resolve(e, false)
	if c.hasAgg(e) {
		fail("SQL: aggregate functions can't be used in %s", what)
	}
}

// fullRow is a stored row's values, one per column: the rowid in the
// INTEGER PRIMARY KEY column, defaults for columns added later, and the
// VIRTUAL generated columns worked out.
func (f *dbFile) fullRow(t *Table, rowid int64, payload []byte, r *runner) []Value {
	vals, err := f.decodeRecord(payload)
	if err != nil {
		panic(err)
	}
	out := f.rowValues(t, rowid, vals)
	if t.RowidCol >= 0 {
		out[t.RowidCol] = rowid
	}
	for i, c := range t.Columns {
		if n, ok := out[i].(int64); ok && c.Affinity == affReal {
			out[i] = float64(n)
		}
	}
	return out
}

// rowValues lays a record's values out one per column. Usually that's
// the record itself; a table with VIRTUAL generated columns doesn't store
// them, and a column added by ALTER TABLE after the row was written takes
// its default.
func (f *dbFile) rowValues(t *Table, rowid int64, rec []Value) []Value {
	if !t.hasVirtual && len(rec) >= len(t.Columns) {
		return rec
	}
	out := make([]Value, len(t.Columns))
	k := 0
	var r *runner
	for i, c := range t.Columns {
		if c.Generated && !c.Stored {
			continue
		}
		if k < len(rec) {
			out[i] = rec[k]
		} else if c.Default != nil {
			if r == nil {
				r = &runner{db: f, core: f.rowPlan(t, "", false), cur: make([]*srcRow, 1)}
			}
			out[i] = r.eval(c.Default)
		}
		k++
	}
	if t.hasVirtual {
		if t.RowidCol >= 0 {
			out[t.RowidCol] = rowid
		}
		f.computeGenerated(t, rowid, out, false)
	}
	return out
}

// computeGenerated works out a row's generated columns (all, or only the
// VIRTUAL ones). One may use another, so it goes round until nothing
// changes.
func (f *dbFile) computeGenerated(t *Table, rowid int64, vals []Value, all bool) {
	var gen []int
	for i, c := range t.Columns {
		if c.Generated && (all || !c.Stored) {
			gen = append(gen, i)
		}
	}
	if len(gen) == 0 {
		return
	}
	c := f.rowPlan(t, "", false)
	r := &runner{db: f, core: c, cur: []*srcRow{{rowid: rowid, vals: vals}}}
	for pass := 0; pass <= len(gen); pass++ {
		changed := false
		for _, i := range gen {
			col := t.Columns[i]
			c.resolveRow(col.GenExpr, "a generated column")
			v := applyAffinity(r.eval(col.GenExpr), col.Affinity)
			if !sameKey([]Value{v}, []Value{vals[i]}) {
				vals[i], changed = v, true
			}
		}
		if !changed {
			return
		}
	}
}

// storedRecord is the record stored for a row: the INTEGER PRIMARY KEY
// column holds NULL (its value is the rowid), and VIRTUAL generated
// columns are left out.
func (f *dbFile) storedRecord(t *Table, vals []Value) []byte {
	rec := make([]Value, 0, len(vals))
	for i, v := range vals {
		c := t.Columns[i]
		if c.Generated && !c.Stored {
			continue
		}
		if i == t.RowidCol {
			v = nil
		}
		rec = append(rec, v)
	}
	return f.encodeRecord(rec)
}

// indexKey is a row's entry in an index: the indexed columns (or
// expressions), then the rowid.
func (f *dbFile) indexKey(t *Table, ix *Index, rowid int64, vals []Value) []Value {
	key := make([]Value, 0, len(ix.Cols)+1)
	var r *runner
	for _, ic := range ix.Cols {
		if ic.Col >= 0 {
			key = append(key, vals[ic.Col])
			continue
		}
		if r == nil {
			r = &runner{db: f, core: f.rowPlan(t, "", false), cur: []*srcRow{{rowid: rowid, vals: vals}}}
		}
		r.core.resolveRow(ic.Expr, "an index")
		key = append(key, r.eval(ic.Expr))
	}
	return append(key, rowid)
}

// inIndex reports whether a row belongs in a (partial) index.
func (f *dbFile) inIndex(t *Table, ix *Index, rowid int64, vals []Value, params []Value) bool {
	if ix.Partial == nil {
		return true
	}
	c := f.rowPlan(t, "", false)
	c.resolveRow(ix.Partial, "an index")
	r := &runner{db: f, params: nil, core: c, cur: []*srcRow{{rowid: rowid, vals: vals}}}
	return truthy(r.eval(ix.Partial))
}

func (f *dbFile) insertRow(t *Table, rowid int64, vals []Value) {
	f.rawInsert(t, rowid, vals)
	if f.foreignKeys {
		f.fkAfterInsert(t, vals)
	}
}

func (f *dbFile) rawInsert(t *Table, rowid int64, vals []Value) {
	f.tablePut(t.Root, rowid, f.storedRecord(t, vals))
	for _, ix := range t.Indexes {
		if f.inIndex(t, ix, rowid, vals, nil) {
			f.indexInsert(ix, f.indexKey(t, ix, rowid, vals))
		}
	}
}

func (f *dbFile) deleteRow(t *Table, rowid int64, vals []Value) {
	if f.foreignKeys {
		key := t.Name + "\x00" + itoa64(rowid)
		if f.fkDeleting == nil {
			f.fkDeleting = map[string]bool{}
		}
		f.fkDeleting[key] = true
		defer delete(f.fkDeleting, key)
		f.fkBeforeDelete(t, rowid, vals)
	}
	f.rawDelete(t, rowid, vals)
}

func (f *dbFile) rawDelete(t *Table, rowid int64, vals []Value) {
	for _, ix := range t.Indexes {
		if f.inIndex(t, ix, rowid, vals, nil) {
			f.indexDelete(ix, f.indexKey(t, ix, rowid, vals))
		}
	}
	f.tableDelete(t.Root, rowid)
}

// writeInPlace rewrites a row that keeps its rowid, changing only the
// index entries whose values changed.
func (f *dbFile) writeInPlace(t *Table, rowid int64, cur, vals []Value) {
	f.tablePut(t.Root, rowid, f.storedRecord(t, vals))
	for _, ix := range t.Indexes {
		was, is := f.inIndex(t, ix, rowid, cur, nil), f.inIndex(t, ix, rowid, vals, nil)
		oldKey, newKey := f.indexKey(t, ix, rowid, cur), f.indexKey(t, ix, rowid, vals)
		if was && is && sameKey(oldKey, newKey) {
			continue
		}
		if was {
			f.indexDelete(ix, oldKey)
		}
		if is {
			f.indexInsert(ix, newKey)
		}
	}
}

// readRow reads the current row with this rowid.
func (f *dbFile) readRow(t *Table, rowid int64, r *runner) ([]Value, bool) {
	payload, ok := f.seekRow(t.Root, rowid)
	if !ok {
		return nil, false
	}
	return f.fullRow(t, rowid, payload, r), true
}

// prepareValue applies the column's affinity, and the STRICT type rules.
func prepareValue(t *Table, ci int, v Value) Value {
	col := t.Columns[ci]
	typ := strings.ToUpper(col.Type)
	if t.Strict && typ == "ANY" {
		return v
	}
	v = applyAffinity(v, col.Affinity)
	if t.Strict && v != nil {
		ok := true
		switch typ {
		case "INT", "INTEGER":
			if fv, isF := v.(float64); isF && fv == float64(int64(fv)) {
				v = int64(fv)
			}
			_, ok = v.(int64)
		case "REAL":
			if n, isI := v.(int64); isI {
				v = float64(n)
			}
			_, ok = v.(float64)
		case "TEXT":
			_, ok = v.(string)
		case "BLOB":
			_, ok = v.([]byte)
		}
		if !ok {
			fail("cannot store %s value in %s column %s.%s", strings.ToUpper(typeName(v)), typ, t.Name, col.Name)
		}
	}
	return v
}

func typeName(v Value) string {
	switch v.(type) {
	case nil:
		return "null"
	case int64:
		return "integer"
	case float64:
		return "real"
	case string:
		return "text"
	}
	return "blob"
}

// toRowid turns a value given for the rowid into one, as SQLite does.
func toRowid(v Value) (int64, bool) {
	switch x := applyAffinity(v, affInteger).(type) {
	case int64:
		return x, true
	case float64:
		if x == float64(int64(x)) {
			return int64(x), true
		}
	}
	return 0, false
}

// newRowid picks the rowid for a new row: one more than the largest
// (and, with AUTOINCREMENT, more than any ever used).
func (f *dbFile) newRowid(t *Table) int64 {
	max := f.maxRowid(t.Root)
	if t.Autoinc {
		if seq := f.sequence(t); seq > max {
			max = seq
		}
	}
	if max == 1<<63-1 {
		fail("database or disk is full (no rowid left in %s)", t.Name)
	}
	return max + 1
}

// sequence is a table's AUTOINCREMENT counter in sqlite_sequence.
func (f *dbFile) sequence(t *Table) int64 {
	seqT := f.tables["sqlite_sequence"]
	if seqT == nil {
		return 0
	}
	var seq int64
	f.scanTable(seqT.Root, func(_ int64, payload []byte) error {
		vals, err := f.decodeRecord(payload)
		if err == nil && len(vals) >= 2 {
			if name, ok := vals[0].(string); ok && strings.EqualFold(name, t.Name) {
				seq, _ = vals[1].(int64)
			}
		}
		return nil
	})
	return seq
}

// setSequence raises a table's AUTOINCREMENT counter to at least n.
func (f *dbFile) setSequence(t *Table, n int64) { f.setSequenceFor(t.Name, n) }

func (f *dbFile) setSequenceFor(tname string, n int64) {
	seqT := f.tables["sqlite_sequence"]
	if seqT == nil {
		return
	}
	var found int64 = -1
	var cur int64
	f.scanTable(seqT.Root, func(rowid int64, payload []byte) error {
		vals, err := f.decodeRecord(payload)
		if err == nil && len(vals) >= 2 {
			if name, ok := vals[0].(string); ok && strings.EqualFold(name, tname) {
				found = rowid
				cur, _ = vals[1].(int64)
			}
		}
		return nil
	})
	if found >= 0 && cur >= n {
		return
	}
	if found < 0 {
		found = f.maxRowid(seqT.Root) + 1
	}
	f.tablePut(seqT.Root, found, f.encodeRecord([]Value{tname, n}))
}

func (f *dbFile) dropSequence(t *Table) {
	seqT := f.tables["sqlite_sequence"]
	if seqT == nil {
		return
	}
	var rowids []int64
	f.scanTable(seqT.Root, func(rowid int64, payload []byte) error {
		vals, err := f.decodeRecord(payload)
		if err == nil && len(vals) >= 1 {
			if name, ok := vals[0].(string); ok && strings.EqualFold(name, t.Name) {
				rowids = append(rowids, rowid)
			}
		}
		return nil
	})
	for _, id := range rowids {
		f.tableDelete(seqT.Root, id)
	}
}

// constraint describes a UNIQUE conflict for messages and upserts.
func uniqueName(t *Table, ix *Index) string {
	var parts []string
	for _, ic := range ix.Cols {
		if ic.Col < 0 {
			return "index '" + ix.Name + "'"
		}
		parts = append(parts, t.Name+"."+t.Columns[ic.Col].Name)
	}
	return strings.Join(parts, ", ")
}

// conflictsWith returns the rowids of the rows (other than self) that
// have the same values as vals in a UNIQUE index. NULLs never conflict.
func (f *dbFile) conflictsWith(t *Table, ix *Index, rowid int64, vals []Value, self int64, hasSelf bool) []int64 {
	if !ix.Unique {
		return nil
	}
	key := f.indexKey(t, ix, rowid, vals)
	prefix := key[:len(ix.Cols)]
	for _, v := range prefix {
		if v == nil {
			return nil
		}
	}
	if !f.inIndex(t, ix, rowid, vals, nil) {
		return nil
	}
	var out []int64
	for _, e := range f.indexEqual(ix, prefix) {
		if id, ok := e[len(e)-1].(int64); ok && !(hasSelf && id == self) {
			out = append(out, id)
		}
	}
	return out
}

// checkRow applies NOT NULL, CHECK and the rowid's type. It returns false
// when the row should be skipped (OR IGNORE).
func (f *dbFile) checkRow(t *Table, rowid int64, vals []Value, or string, r *runner) bool {
	for i, col := range t.Columns {
		if vals[i] != nil || !col.NotNull || i == t.RowidCol {
			continue
		}
		mode := resolution(or, col.notNullOn)
		switch mode {
		case "IGNORE":
			return false
		case "REPLACE":
			if col.Default != nil {
				if d := r.eval(col.Default); d != nil {
					vals[i] = prepareValue(t, i, d)
					continue
				}
			}
		}
		conflictFail(mode, "NOT NULL constraint failed: %s.%s", t.Name, col.Name)
	}
	if len(t.Checks) > 0 {
		c := f.rowPlan(t, "", false)
		cr := &runner{db: f, params: r.params, core: c, cur: []*srcRow{{rowid: rowid, vals: vals}}}
		for i, chk := range t.Checks {
			c.resolveRow(chk, "CHECK")
			v := cr.eval(chk)
			if v != nil && !truthy(v) {
				mode := resolution(or, "")
				if mode == "IGNORE" {
					return false
				}
				if mode == "REPLACE" {
					mode = "ABORT"
				}
				conflictFail(mode, "CHECK constraint failed: %s", t.CheckText[i])
			}
		}
	}
	return true
}

// resolution picks the conflict resolution: the statement's OR, else the
// constraint's ON CONFLICT, else ABORT.
func resolution(or, constraint string) string {
	if or != "" {
		return or
	}
	if constraint != "" {
		return constraint
	}
	return "ABORT"
}

// conflictFail stops the statement: ABORT undoes it, FAIL keeps its
// earlier rows, ROLLBACK undoes the whole transaction.
func conflictFail(mode, format string, args ...any) {
	msg := errorf(format, args...).Error()
	switch mode {
	case "ROLLBACK":
		msg = rollbackMark + msg
	case "FAIL":
		msg = failMark + msg
	}
	panic(&Error{Msg: msg})
}

// placeRow resolves UNIQUE conflicts for a row about to be written at
// rowid. When updating, self is the row's current rowid (hasSelf), so it
// doesn't conflict with itself. It returns false to skip the row (OR
// IGNORE).
func (f *dbFile) placeRow(t *Table, rowid int64, self int64, hasSelf bool, vals []Value, or string, r *runner) bool {
	if !hasSelf || rowid != self {
		if _, exists := f.seekRow(t.Root, rowid); exists {
			mode := resolution(or, t.pkConf)
			switch mode {
			case "IGNORE":
				return false
			case "REPLACE":
				if old, ok := f.readRow(t, rowid, r); ok {
					f.deleteRow(t, rowid, old)
				}
			default:
				name := "rowid"
				if t.RowidCol >= 0 {
					name = t.Columns[t.RowidCol].Name
				}
				conflictFail(mode, "UNIQUE constraint failed: %s.%s", t.Name, name)
			}
		}
	}
	for _, ix := range t.Indexes {
		real := f.conflictsWith(t, ix, rowid, vals, self, hasSelf)
		if len(real) == 0 {
			continue
		}
		mode := resolution(or, ix.onConf)
		switch mode {
		case "IGNORE":
			return false
		case "REPLACE":
			for _, id := range real {
				if old, ok := f.readRow(t, id, r); ok {
					f.deleteRow(t, id, old)
				}
			}
		default:
			conflictFail(mode, "UNIQUE constraint failed: %s", uniqueName(t, ix))
		}
	}
	return true
}

// returningRow evaluates RETURNING for one row.
func (f *dbFile) returningPlan(t *Table, alias string, cols []resultCol) (*corePlan, []resultCol, []string) {
	if len(cols) == 0 {
		return nil, nil, nil
	}
	c := f.rowPlan(t, alias, false)
	var out []resultCol
	var names []string
	for _, rc := range cols {
		if rc.star {
			for ci, col := range t.Columns {
				out = append(out, resultCol{e: c.colRef(0, ci), name: col.Name})
				names = append(names, col.Name)
			}
			continue
		}
		c.resolveRow(rc.e, "RETURNING")
		out = append(out, rc)
		names = append(names, rc.name)
	}
	return c, out, names
}

func (r *runner) returning(cols []resultCol, rowid int64, vals []Value) []Value {
	r.cur[0] = &srcRow{rowid: rowid, vals: vals}
	out := make([]Value, len(cols))
	for i, rc := range cols {
		out[i] = r.eval(rc.e)
	}
	return out
}

// ---- INSERT ----

func (f *dbFile) runInsert(s *insertStmt, params []Value) Result {
	if v, ok := f.views[strings.ToLower(s.table)]; ok {
		return f.viewInsert(v, s, params)
	}
	t := f.tableFor(s.table)
	var res Result
	// Which column each value goes to (-1: the rowid itself).
	var target []int
	if s.cols != nil {
		seen := map[int]bool{}
		for _, name := range s.cols {
			ci := colIndex(t.Columns, name)
			if ci < 0 && isRowidName(name) {
				ci = -1
				if t.RowidCol >= 0 {
					ci = t.RowidCol
				}
			} else if ci < 0 {
				fail("table %s has no column named %s", t.Name, name)
			}
			if seen[ci] {
				fail("SQL: column %s is given twice", name)
			}
			if ci >= 0 && t.Columns[ci].Generated {
				fail("cannot INSERT into generated column \"%s\"", t.Columns[ci].Name)
			}
			seen[ci] = true
			target = append(target, ci)
		}
	} else {
		for i, c := range t.Columns {
			if !c.Generated {
				target = append(target, i)
			}
		}
	}
	var rows [][]Value
	if s.src != nil && len(s.with) > 0 {
		s.src.with = append(append([]*cteDef{}, s.with...), s.src.with...)
		s.src.recursive = s.src.recursive || s.recursive
	}
	if s.src == nil {
		rows = [][]Value{{}}
		target = nil
	} else {
		q := f.planQuery(s.src, f.trigScope(), nil, nil)
		if len(q.names) != len(target) {
			if s.cols != nil {
				fail("%d values for %d columns", len(q.names), len(target))
			}
			fail("table %s has %d columns but %d values were supplied", t.Name, len(target), len(q.names))
		}
		rows = q.rows(params, f.trigRunner())
	}
	empty := f.rowPlan(t, s.alias, false)
	r := &runner{db: f, parent: f.trigRunner(), params: params, core: empty, cur: make([]*srcRow, 1)}
	// Upserts: their targets, and a plan where "excluded" is the new row.
	var up *corePlan
	var upTargets [][]*Index
	if len(s.upserts) > 0 {
		up = f.rowPlan(t, s.alias, true)
		for i, u := range s.upserts {
			var matches []*Index
			if u.target == nil {
				if i != len(s.upserts)-1 {
					fail("SQL: only the last ON CONFLICT clause may leave out its columns")
				}
			} else {
				matches = upsertIndexes(t, u.target)
				if matches == nil {
					fail("ON CONFLICT clause does not match any PRIMARY KEY or UNIQUE constraint")
				}
			}
			upTargets = append(upTargets, matches)
			for _, set := range u.sets {
				up.resolveRow(set.e, "ON CONFLICT")
			}
			if u.where != nil {
				up.resolveRow(u.where, "ON CONFLICT")
			}
		}
	}
	rc, rcols, rnames := f.returningPlan(t, s.alias, s.returning)
	res.Cols = rnames
	for _, src := range rows {
		vals := make([]Value, len(t.Columns))
		given := make([]bool, len(t.Columns))
		var rowid int64
		haveRowid := false
		for i, ci := range target {
			if ci == -1 {
				if src[i] != nil {
					id, ok := toRowid(src[i])
					if !ok {
						fail("datatype mismatch")
					}
					rowid, haveRowid = id, true
				}
				continue
			}
			vals[ci], given[ci] = src[i], true
		}
		for ci, col := range t.Columns {
			if !given[ci] && col.Default != nil {
				vals[ci] = r.eval(col.Default)
			}
		}
		if t.RowidCol >= 0 && vals[t.RowidCol] != nil {
			id, ok := toRowid(vals[t.RowidCol])
			if !ok {
				fail("datatype mismatch")
			}
			rowid, haveRowid = id, true
		}
		for ci := range t.Columns {
			if ci != t.RowidCol {
				vals[ci] = prepareValue(t, ci, vals[ci])
			}
		}
		if !haveRowid {
			rowid = f.newRowid(t)
		}
		if t.RowidCol >= 0 {
			vals[t.RowidCol] = rowid
		}
		f.computeGenerated(t, rowid, vals, true)
		if !f.fire(t.Name, t.Columns, t.RowidCol, "BEFORE", "INSERT", 0, nil, rowid, vals, nil) {
			continue // RAISE(IGNORE)
		}
		if !f.checkRow(t, rowid, vals, s.or, r) {
			continue
		}
		// An upsert takes over a conflict on its target.
		if up != nil {
			if done, changed := f.tryUpsert(t, s, up, upTargets, rowid, vals, params, r); done {
				if changed {
					res.Changes++
				}
				continue
			}
		}
		if !f.placeRow(t, rowid, 0, false, vals, s.or, r) {
			continue
		}
		f.insertRow(t, rowid, vals)
		if t.Autoinc {
			f.setSequence(t, rowid)
		}
		f.fire(t.Name, t.Columns, t.RowidCol, "AFTER", "INSERT", 0, nil, rowid, vals, nil)
		res.Changes++
		res.LastRowid = rowid
		if rc != nil {
			rr := &runner{db: f, parent: f.trigRunner(), params: params, core: rc, cur: make([]*srcRow, 1)}
			res.Rows = append(res.Rows, rr.returning(rcols, rowid, vals))
		}
	}
	return res
}

// upsertIndexes finds the UNIQUE indexes (or the INTEGER PRIMARY KEY,
// as a nil *Index) whose columns are exactly cols.
func upsertIndexes(t *Table, cols []string) []*Index {
	want := map[int]bool{}
	for _, c := range cols {
		ci := colIndex(t.Columns, c)
		if ci < 0 {
			fail("no such column: %s", c)
		}
		want[ci] = true
	}
	var out []*Index
	if len(want) == 1 && t.RowidCol >= 0 && want[t.RowidCol] {
		out = append(out, nil)
	}
	for _, ix := range t.Indexes {
		if !ix.Unique || len(ix.Cols) != len(want) {
			continue
		}
		all := true
		for _, ic := range ix.Cols {
			if !want[ic.Col] {
				all = false
			}
		}
		if all {
			out = append(out, ix)
		}
	}
	return out
}

// tryUpsert checks the new row against each ON CONFLICT clause. If one
// applies, it does what the clause says and reports done.
func (f *dbFile) tryUpsert(t *Table, s *insertStmt, up *corePlan, targets [][]*Index, rowid int64, vals []Value, params []Value, r *runner) (bool, bool) {
	for i, u := range s.upserts {
		var other int64
		found := false
		check := func(ix *Index) {
			if found {
				return
			}
			if ix == nil {
				if _, ok := f.seekRow(t.Root, rowid); ok {
					other, found = rowid, true
				}
				return
			}
			if ids := f.conflictsWith(t, ix, rowid, vals, 0, false); len(ids) > 0 {
				other, found = ids[0], true
			}
		}
		if targets[i] == nil {
			check(nil)
			for _, ix := range t.Indexes {
				check(ix)
			}
		} else {
			for _, ix := range targets[i] {
				check(ix)
			}
		}
		if !found {
			continue
		}
		if u.nothing {
			return true, false
		}
		old, ok := f.readRow(t, other, r)
		if !ok {
			return true, false
		}
		ur := &runner{db: f, parent: f.trigRunner(), params: params, core: up, cur: []*srcRow{{rowid: other, vals: old}, {vals: vals}}}
		if u.where != nil && !truthy(ur.eval(u.where)) {
			return true, false
		}
		nv := append([]Value{}, old...)
		newRowid := other
		for _, set := range u.sets {
			ci := colIndex(t.Columns, set.col)
			v := ur.eval(set.e)
			if ci < 0 {
				if !isRowidName(set.col) {
					fail("no such column: %s", set.col)
				}
				id, ok := toRowid(v)
				if !ok {
					fail("datatype mismatch")
				}
				newRowid = id
				continue
			}
			if ci == t.RowidCol {
				id, ok := toRowid(v)
				if !ok {
					fail("datatype mismatch")
				}
				newRowid = id
				nv[ci] = id
				continue
			}
			nv[ci] = prepareValue(t, ci, v)
		}
		f.computeGenerated(t, newRowid, nv, true)
		changed := setNames(u.sets)
		if !f.fire(t.Name, t.Columns, t.RowidCol, "BEFORE", "UPDATE", other, old, newRowid, nv, changed) {
			return true, false
		}
		f.updateRow(t, other, old, newRowid, nv, "ABORT", r)
		f.fire(t.Name, t.Columns, t.RowidCol, "AFTER", "UPDATE", other, old, newRowid, nv, changed)
		return true, true
	}
	return false, false
}

// updateRow replaces a row, checking constraints first.
func (f *dbFile) updateRow(t *Table, rowid int64, old []Value, newRowid int64, vals []Value, or string, r *runner) bool {
	if t.RowidCol >= 0 {
		vals[t.RowidCol] = newRowid
	}
	if !f.checkRow(t, newRowid, vals, or, r) {
		return false
	}
	if !f.placeRow(t, newRowid, rowid, true, vals, or, r) {
		return false
	}
	// placeRow may have deleted rows (REPLACE); the row itself is still
	// there unless it was the conflicting one.
	cur, ok := f.readRow(t, rowid, r)
	if ok && f.foreignKeys {
		f.fkBeforeUpdate(t, rowid, cur, vals)
	}
	switch {
	case ok && newRowid == rowid:
		f.writeInPlace(t, rowid, cur, vals)
	case ok:
		f.rawDelete(t, rowid, cur)
		f.rawInsert(t, newRowid, vals)
	default:
		f.insertRow(t, newRowid, vals)
	}
	if ok && f.foreignKeys {
		f.fkAfterUpdate(t, cur, vals)
	}
	if t.Autoinc {
		f.setSequence(t, newRowid)
	}
	return true
}

// sameKey reports whether two index keys are stored identically (same
// values and types, so 1 and 1.0 differ).
func sameKey(a, b []Value) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if typeName(a[i]) != typeName(b[i]) || compare(a[i], b[i]) != 0 {
			return false
		}
		if x, ok := a[i].(string); ok && x != b[i].(string) {
			return false
		}
	}
	return true
}

// ---- UPDATE and DELETE ----

// matchingRowids returns the rowids of the rows WHERE selects, in rowid
// order, read before anything changes.
func (f *dbFile) matchingRowids(t *Table, alias string, where expr, params []Value, with []*cteDef, recursive bool) []int64 {
	name := t.Name
	if alias != "" {
		name = alias
	}
	_ = name
	rowid := &colExpr{name: "rowid", idx: -1, rowid: true, aff: affInteger, bound: true}
	s := &selectStmt{cols: []resultCol{{e: rowid}}, from: []fromItem{{name: t.Name, alias: alias}}, where: where, with: with, recursive: recursive}
	q := f.planQuery(s, f.trigScope(), nil, nil)
	var ids []int64
	for _, row := range q.rows(params, f.trigRunner()) {
		ids = append(ids, row[0].(int64))
	}
	return ids
}

func (f *dbFile) runUpdate(s *updateStmt, params []Value) Result {
	if v, ok := f.views[strings.ToLower(s.table)]; ok {
		return f.viewUpdate(v, s, params)
	}
	t := f.tableFor(s.table)
	c := f.rowPlan(t, s.alias, false)
	c.sc.ctes = f.planWith(s.with, s.recursive)
	type assign struct {
		ci    int // column, or -1 for the rowid
		e     expr
		rowid bool
	}
	var sets []assign
	for _, set := range s.sets {
		ci := colIndex(t.Columns, set.col)
		a := assign{ci: ci, e: set.e}
		if ci < 0 {
			if !isRowidName(set.col) {
				fail("no such column: %s", set.col)
			}
			a.rowid = true
		} else if ci == t.RowidCol {
			a.rowid = true
		} else if t.Columns[ci].Generated {
			fail("cannot UPDATE generated column \"%s\"", t.Columns[ci].Name)
		}
		if s.from == nil {
			c.resolveRow(set.e, "SET")
		}
		sets = append(sets, a)
	}
	var ids []int64
	var joined map[int64][]Value
	if s.from == nil {
		ids = f.matchingRowids(t, s.alias, s.where, params, s.with, s.recursive)
	} else {
		ids, joined = f.updateFromRows(t, s, params)
	}
	changed := setNames(s.sets)
	r := &runner{db: f, parent: f.trigRunner(), params: params, core: c, cur: make([]*srcRow, 1)}
	rc, rcols, rnames := f.returningPlan(t, s.alias, s.returning)
	res := Result{Cols: rnames}
	for _, id := range ids {
		old, ok := f.readRow(t, id, r)
		if !ok {
			continue // removed by an earlier REPLACE
		}
		r.cur[0] = &srcRow{rowid: id, vals: old}
		nv := append([]Value{}, old...)
		newRowid := id
		for k, a := range sets {
			var v Value
			if joined != nil {
				v = joined[id][k]
			} else {
				v = r.eval(a.e)
			}
			if a.rowid {
				nid, ok := toRowid(v)
				if !ok {
					fail("datatype mismatch")
				}
				newRowid = nid
				if a.ci >= 0 {
					nv[a.ci] = nid
				}
				continue
			}
			nv[a.ci] = prepareValue(t, a.ci, v)
		}
		f.computeGenerated(t, newRowid, nv, true)
		trigs := len(f.triggersFor(t.Name)) > 0
		if trigs {
			if !f.fire(t.Name, t.Columns, t.RowidCol, "BEFORE", "UPDATE", id, old, newRowid, nv, changed) {
				continue // RAISE(IGNORE)
			}
			// A trigger may have changed or removed the row.
			if cur, ok := f.readRow(t, id, r); ok {
				old = cur
			} else {
				continue
			}
		}
		if !f.updateRow(t, id, old, newRowid, nv, s.or, r) {
			continue
		}
		if trigs {
			f.fire(t.Name, t.Columns, t.RowidCol, "AFTER", "UPDATE", id, old, newRowid, nv, changed)
		}
		res.Changes++
		if rc != nil {
			rr := &runner{db: f, parent: f.trigRunner(), params: params, core: rc, cur: make([]*srcRow, 1)}
			res.Rows = append(res.Rows, rr.returning(rcols, newRowid, nv))
		}
	}
	return res
}

// updateFromRows runs UPDATE ... FROM's join: for each row of the table
// that WHERE pairs with a row of the FROM tables, the new values, worked
// out from the first such pair (as SQLite does, other pairs are ignored).
func (f *dbFile) updateFromRows(t *Table, s *updateStmt, params []Value) ([]int64, map[int64][]Value) {
	name := t.Name
	if s.alias != "" {
		name = s.alias
	}
	cols := []resultCol{{e: &colExpr{table: name, name: "rowid", idx: -1}}}
	for _, set := range s.sets {
		cols = append(cols, resultCol{e: set.e})
	}
	from := append([]fromItem{{name: t.Name, alias: s.alias}}, s.from...)
	if from[1].join == "" {
		from[1].join = "INNER"
	}
	sel := &selectStmt{cols: cols, from: from, where: s.where, with: s.with, recursive: s.recursive}
	q := f.planQuery(sel, f.trigScope(), nil, nil)
	var ids []int64
	vals := map[int64][]Value{}
	for _, row := range q.rows(params, f.trigRunner()) {
		id, ok := row[0].(int64)
		if !ok {
			continue
		}
		if _, seen := vals[id]; !seen {
			ids = append(ids, id)
			vals[id] = row[1:]
		}
	}
	slices.Sort(ids)
	return ids, vals
}

func (f *dbFile) runDelete(s *deleteStmt, params []Value) Result {
	if v, ok := f.views[strings.ToLower(s.table)]; ok {
		return f.viewDelete(v, s, params)
	}
	t := f.tableFor(s.table)
	rc, rcols, rnames := f.returningPlan(t, s.alias, s.returning)
	res := Result{Cols: rnames}
	r := &runner{db: f, parent: f.trigRunner(), params: params, core: f.rowPlan(t, s.alias, false), cur: make([]*srcRow, 1)}
	trigs := len(f.triggersFor(t.Name)) > 0
	if s.where == nil && rc == nil && len(s.with) == 0 && !trigs && (!f.foreignKeys || len(t.FKs) == 0 && len(f.incoming(t)) == 0) {
		// Everything: empty the table and its indexes.
		var n int64
		f.scanTable(t.Root, func(int64, []byte) error { n++; return nil })
		f.clearTree(t.Root, false)
		for _, ix := range t.Indexes {
			f.clearTree(ix.Root, false)
		}
		res.Changes = n
		return res
	}
	for _, id := range f.matchingRowids(t, s.alias, s.where, params, s.with, s.recursive) {
		old, ok := f.readRow(t, id, r)
		if !ok {
			continue
		}
		if trigs {
			if !f.fire(t.Name, t.Columns, t.RowidCol, "BEFORE", "DELETE", id, old, 0, nil, nil) {
				continue // RAISE(IGNORE)
			}
			if old, ok = f.readRow(t, id, r); !ok {
				continue
			}
		}
		if rc != nil {
			rr := &runner{db: f, parent: f.trigRunner(), params: params, core: rc, cur: make([]*srcRow, 1)}
			res.Rows = append(res.Rows, rr.returning(rcols, id, old))
		}
		f.deleteRow(t, id, old)
		if trigs {
			f.fire(t.Name, t.Columns, t.RowidCol, "AFTER", "DELETE", id, old, 0, nil, nil)
		}
		res.Changes++
	}
	return res
}

// ---- the schema ----

// nameTaken reports what already has this name: "table", "index", "view".
func (f *dbFile) nameTaken(name string) string {
	for _, r := range f.schemaRows {
		if strings.EqualFold(r.name, name) {
			return r.kind
		}
	}
	return ""
}

func checkNewName(name string) {
	if strings.HasPrefix(strings.ToLower(name), "sqlite_") {
		fail("object name reserved for internal use: %s", name)
	}
}

func (f *dbFile) addSchemaRow(kind, name, tbl string, root uint32, sql Value) {
	id := f.maxRowid(1) + 1
	f.tablePut(1, id, f.encodeRecord([]Value{kind, name, tbl, int64(root), sql}))
	f.schemaChanged = true
}

func (f *dbFile) reloadSchema() {
	if err := f.loadSchema(); err != nil {
		panic(err)
	}
}

func (f *dbFile) runCreateTable(s *createTableStmt, params []Value) {
	if kind := f.nameTaken(s.name); kind != "" {
		if s.ifNotExists && kind == "table" {
			return
		}
		fail("%s %s already exists", kind, s.name)
	}
	checkNewName(s.name)
	var rows [][]Value
	sql := s.sql
	if s.as != nil {
		q := f.planQuery(s.as, nil, nil, nil)
		rows = q.rows(params, nil)
		var defs []string
		for i, n := range q.names {
			typ := ""
			switch q.affs[i] {
			case affInteger:
				typ = " INT"
			case affReal:
				typ = " REAL"
			case affText:
				typ = " TEXT"
			case affNumeric:
				typ = " NUM"
			}
			defs = append(defs, quoteIdent(n)+typ)
		}
		sql = "CREATE TABLE " + quoteIdent(s.name) + "(" + strings.Join(defs, ",") + ")"
	}
	d := parseTableDef(newParser(sql))
	t := d.t
	root := f.newTree(!t.WithoutRowid)
	f.addSchemaRow("table", t.Name, t.Name, root, sql)
	for i, u := range d.uniques {
		if t.WithoutRowid && u.Origin == "pk" {
			continue // the table itself is this index
		}
		ir := f.newTree(false)
		f.addSchemaRow("index", "sqlite_autoindex_"+t.Name+"_"+itoa(i+1), t.Name, ir, nil)
	}
	if t.Autoinc && f.tables["sqlite_sequence"] == nil {
		sr := f.newTree(true)
		f.addSchemaRow("table", "sqlite_sequence", "sqlite_sequence", sr, "CREATE TABLE sqlite_sequence(name,seq)")
	}
	f.reloadSchema()
	if rows != nil {
		nt := f.tables[strings.ToLower(t.Name)]
		for i, row := range rows {
			vals := make([]Value, len(nt.Columns))
			for ci := range vals {
				vals[ci] = prepareValue(nt, ci, row[ci])
			}
			f.insertRow(nt, int64(i+1), vals)
		}
	}
}

// quoteIdent writes a name so SQL reads it back as the same name.
func quoteIdent(name string) string {
	plain := name != ""
	for i, c := range name {
		if !(c == '_' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || (i > 0 && c >= '0' && c <= '9')) {
			plain = false
		}
	}
	if plain && !isKeyword(name) {
		return name
	}
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}

// sqlKeywords are SQLite's keywords: a name that is one must be quoted.
var sqlKeywords = map[string]bool{}

func init() {
	for _, w := range strings.Fields(`ABORT ACTION ADD AFTER ALL ALTER ALWAYS ANALYZE AND AS ASC ATTACH
		AUTOINCREMENT BEFORE BEGIN BETWEEN BY CASCADE CASE CAST CHECK COLLATE COLUMN COMMIT CONFLICT
		CONSTRAINT CREATE CROSS CURRENT CURRENT_DATE CURRENT_TIME CURRENT_TIMESTAMP DATABASE DEFAULT
		DEFERRABLE DEFERRED DELETE DESC DETACH DISTINCT DO DROP EACH ELSE END ESCAPE EXCEPT EXCLUDE
		EXCLUSIVE EXISTS EXPLAIN FAIL FILTER FIRST FOLLOWING FOR FOREIGN FROM FULL GENERATED GLOB GROUP
		GROUPS HAVING IF IGNORE IMMEDIATE IN INDEX INDEXED INITIALLY INNER INSERT INSTEAD INTERSECT INTO
		IS ISNULL JOIN KEY LAST LEFT LIKE LIMIT MATCH MATERIALIZED NATURAL NO NOT NOTHING NOTNULL NULL
		NULLS OF OFFSET ON OR ORDER OTHERS OUTER OVER PARTITION PLAN PRAGMA PRECEDING PRIMARY QUERY RAISE
		RANGE RECURSIVE REFERENCES REGEXP REINDEX RELEASE RENAME REPLACE RESTRICT RETURNING RIGHT ROLLBACK
		ROW ROWS SAVEPOINT SELECT SET TABLE TEMP TEMPORARY THEN TIES TO TRANSACTION TRIGGER UNBOUNDED
		UNION UNIQUE UPDATE USING VACUUM VALUES VIEW VIRTUAL WHEN WHERE WINDOW WITH WITHOUT`) {
		sqlKeywords[w] = true
	}
}

func isKeyword(w string) bool { return sqlKeywords[strings.ToUpper(w)] }

// quoteAlways writes a name in double quotes, as SQLite does when it
// renames a table.
func quoteAlways(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}

func (f *dbFile) runCreateIndex(s *createIndexStmt) {
	if kind := f.nameTaken(s.name); kind != "" {
		if s.ifNotExists && kind == "index" {
			return
		}
		fail("%s %s already exists", kind, s.name)
	}
	checkNewName(s.name)
	t, err := f.table(s.table)
	if err != nil {
		panic(err)
	}
	if t == schemaTable || strings.HasPrefix(strings.ToLower(t.Name), "sqlite_") {
		fail("table %s may not be indexed", t.Name)
	}
	ix := newParser(s.src).parseIndexDef(t)
	for _, ic := range ix.Cols {
		if ic.Expr != nil {
			r := f.rowPlan(t, "", false)
			r.resolveRow(ic.Expr, "an index")
			if hasWindow(ic.Expr) {
				fail("misuse of window function in an index")
			}
		}
	}
	ix.Name, ix.Table = s.name, t.Name
	ix.Root = f.newTree(false)
	f.addSchemaRow("index", s.name, t.Name, ix.Root, s.sql)
	// Fill it from the table's rows.
	var rows []srcRow
	r := &runner{db: f, core: f.rowPlan(t, "", false), cur: make([]*srcRow, 1)}
	f.scanTable(t.Root, func(rowid int64, payload []byte) error {
		rows = append(rows, srcRow{rowid: rowid, vals: f.fullRow(t, rowid, payload, r)})
		return nil
	})
	for _, row := range rows {
		if !f.inIndex(t, ix, row.rowid, row.vals, nil) {
			continue
		}
		if ix.Unique && len(f.conflictsWith(t, ix, row.rowid, row.vals, row.rowid, true)) > 0 {
			fail("UNIQUE constraint failed: %s", uniqueName(t, ix))
		}
		f.indexInsert(ix, f.indexKey(t, ix, row.rowid, row.vals))
	}
	f.reloadSchema()
}

func (f *dbFile) runCreateView(s *createViewStmt) {
	if kind := f.nameTaken(s.name); kind != "" {
		if s.ifNotExists && kind == "view" {
			return
		}
		fail("%s %s already exists", kind, s.name)
	}
	checkNewName(s.name)
	// Like SQLite, the tables and columns a view names are checked when
	// it's used, not now: a view may come before its tables.
	f.addSchemaRow("view", s.name, s.name, 0, s.sql)
	f.reloadSchema()
}

func (f *dbFile) runCreateTrigger(s *createTriggerStmt) {
	tr := s.tr
	if kind := f.nameTaken(tr.Name); kind != "" {
		if s.ifNotExists && kind == "trigger" {
			return
		}
		fail("%s %s already exists", kind, tr.Name)
	}
	checkNewName(tr.Name)
	low := strings.ToLower(tr.Table)
	var tbl string
	if t, ok := f.tables[low]; ok {
		if strings.HasPrefix(low, "sqlite_") {
			fail("cannot create trigger on system table")
		}
		if tr.Timing == "INSTEAD OF" {
			fail("cannot create INSTEAD OF trigger on table: %s", t.Name)
		}
		tbl = t.Name
	} else if v, ok := f.views[low]; ok {
		if tr.Timing != "INSTEAD OF" {
			fail("cannot create %s trigger on view: %s", tr.Timing, v.Name)
		}
		tbl = v.Name
	} else {
		fail("no such table: main.%s", tr.Table)
	}
	f.addSchemaRow("trigger", tr.Name, tbl, 0, s.sql)
	f.reloadSchema()
}

func (f *dbFile) runDrop(s *dropStmt) {
	kind := strings.ToLower(s.kind)
	if low := strings.ToLower(s.name); low == "sqlite_schema" || low == "sqlite_master" {
		fail("table %s may not be dropped", s.name)
	}
	var row *schemaRow
	for i := range f.schemaRows {
		if strings.EqualFold(f.schemaRows[i].name, s.name) {
			row = &f.schemaRows[i]
		}
	}
	if row == nil || row.kind != kind {
		if s.ifExists && row == nil {
			return
		}
		fail("no such %s: %s", kind, s.name)
	}
	switch kind {
	case "table":
		if strings.HasPrefix(strings.ToLower(row.name), "sqlite_") {
			fail("table %s may not be dropped", row.name)
		}
		t := f.tables[strings.ToLower(row.name)]
		if f.foreignKeys && t != nil {
			// As in SQLite: an implicit DELETE FROM first, so foreign key
			// actions and checks happen.
			r := &runner{db: f, core: f.rowPlan(t, "", false), cur: make([]*srcRow, 1)}
			var rows []srcRow
			f.scanTable(t.Root, func(rowid int64, payload []byte) error {
				rows = append(rows, srcRow{rowid, f.fullRow(t, rowid, payload, r)})
				return nil
			})
			for _, rw := range rows {
				if cur, ok := f.readRow(t, rw.rowid, r); ok {
					f.deleteRow(t, rw.rowid, cur)
				}
			}
		}
		for _, r := range append([]schemaRow{}, f.schemaRows...) {
			if strings.EqualFold(r.tbl, row.name) && r.kind != "table" {
				if r.root > 0 {
					f.clearTree(r.root, true)
				}
				f.tableDelete(1, r.rowid)
			}
		}
		f.clearTree(row.root, true)
		f.tableDelete(1, row.rowid)
		if t != nil && t.Autoinc {
			f.dropSequence(t)
		}
	case "index":
		if strings.HasPrefix(strings.ToLower(row.name), "sqlite_autoindex") {
			fail("index associated with UNIQUE or PRIMARY KEY constraint cannot be dropped")
		}
		f.clearTree(row.root, true)
		f.tableDelete(1, row.rowid)
	case "view", "trigger":
		f.tableDelete(1, row.rowid)
	}
	f.schemaChanged = true
	f.reloadSchema()
}

// putSchemaRow rewrites one row of sqlite_schema.
func (f *dbFile) putSchemaRow(r schemaRow, sql Value) {
	f.tablePut(1, r.rowid, f.encodeRecord([]Value{r.kind, r.name, r.tbl, int64(r.root), sql}))
	f.schemaChanged = true
}

// replaceTokens rewrites the names at the given tokens of src.
func replaceTokens(src string, toks []token, old, new string) string {
	var sb strings.Builder
	last := 0
	for _, tk := range toks {
		if !strings.EqualFold(tk.text, old) || tk.pos < last {
			continue
		}
		sb.WriteString(src[last:tk.pos])
		sb.WriteString(quoteIdent(new))
		last = tk.end
	}
	sb.WriteString(src[last:])
	return sb.String()
}

func (f *dbFile) runAlter(s *alterStmt) {
	t := f.tableFor(s.table)
	if strings.HasPrefix(strings.ToLower(t.Name), "sqlite_") {
		fail("table %s may not be altered", t.Name)
	}
	var tableRow schemaRow
	var indexRows []schemaRow
	for _, r := range f.schemaRows {
		switch {
		case r.kind == "table" && strings.EqualFold(r.name, t.Name):
			tableRow = r
		case r.kind == "index" && strings.EqualFold(r.tbl, t.Name):
			indexRows = append(indexRows, r)
		}
	}
	d := parseTableDef(newParser(t.SQL))
	switch s.op {
	case "RENAME":
		if kind := f.nameTaken(s.newName); kind != "" && !strings.EqualFold(s.newName, t.Name) {
			fail("there is already another table or index with this name: %s", s.newName)
		}
		checkNewName(s.newName)
		// Views that name the table follow it to its new name, as in
		// SQLite.
		for _, r := range f.schemaRows {
			if r.kind != "view" {
				continue
			}
			toks := viewTableTokens(r.sql)
			if len(toks) > 0 {
				f.putSchemaRow(r, replaceTokensAlways(r.sql, toks, t.Name, s.newName))
			}
		}
		defer f.checkViews()
		// Triggers on the table, or naming it, follow it too.
		for _, r := range f.schemaRows {
			if r.kind != "trigger" {
				continue
			}
			tr, err := parseTriggerSQL(r.sql)
			if err != nil {
				continue
			}
			sql := replaceTokensAlways(r.sql, tr.refs, t.Name, s.newName)
			if strings.EqualFold(tr.Table, t.Name) {
				// The ON name comes before the body, so its position is
				// unchanged by the body's rewrite.
				sql = sql[:tr.tblPos] + quoteAlways(s.newName) + sql[tr.tblEnd:]
				r.tbl = s.newName
			}
			if sql != r.sql || strings.EqualFold(tr.Table, t.Name) {
				f.putSchemaRow(r, sql)
			}
		}
		sql := t.SQL[:d.namePos] + quoteAlways(s.newName) + t.SQL[d.nameEnd:]
		tableRow.name, tableRow.tbl = s.newName, s.newName
		f.putSchemaRow(tableRow, sql)
		for _, r := range indexRows {
			r.tbl = s.newName
			if r.sql == "" {
				suffix := r.name[strings.LastIndexByte(r.name, '_'):]
				r.name = "sqlite_autoindex_" + s.newName + suffix
				f.putSchemaRow(r, nil)
				continue
			}
			ix := newParser(r.sql).parseIndexDef(t)
			f.putSchemaRow(r, r.sql[:ix.tablePos]+quoteAlways(s.newName)+r.sql[ix.tableEnd:])
		}
		if t.Autoinc {
			seq := f.sequence(t)
			f.dropSequence(t)
			if seq > 0 {
				f.setSequenceFor(s.newName, seq)
			}
		}
	case "ADD":
		def := "CREATE TABLE x(" + s.colDef + ")"
		nd := parseTableDef(newParser(def))
		col := nd.t.Columns[0]
		switch {
		case colIndex(t.Columns, col.Name) >= 0:
			fail("duplicate column name: %s", col.Name)
		case nd.t.RowidCol >= 0 || len(nd.uniques) > 0:
			fail("Cannot add a PRIMARY KEY or UNIQUE column")
		case col.NotNull && col.Default == nil:
			fail("Cannot add a NOT NULL column with default value NULL")
		case col.Generated:
			fail("cannot add a generated column")
		}
		if col.Default != nil {
			r := &runner{db: f, core: f.rowPlan(t, "", false), cur: make([]*srcRow, 1)}
			ok := true
			walk(col.Default, func(e expr) {
				switch x := e.(type) {
				case *colExpr, *subqueryExpr, *existsExpr:
					ok = false
				case *callExpr:
					if x.name == "datetime" || x.name == "date" || x.name == "time" || x.name == "random" {
						ok = false
					}
				}
			})
			if !ok {
				fail("Cannot add a column with non-constant default")
			}
			r.core.resolveRow(col.Default, "DEFAULT")
			if col.NotNull && r.eval(col.Default) == nil {
				fail("Cannot add a NOT NULL column with default value NULL")
			}
		}
		sql := t.SQL[:d.listEnd] + ", " + s.colDef + t.SQL[d.listEnd:]
		f.putSchemaRow(tableRow, sql)
	case "RENAME COLUMN":
		ci := colIndex(t.Columns, s.col)
		if ci < 0 {
			fail("no such column: \"%s\"", s.col)
		}
		if colIndex(t.Columns, s.newName) >= 0 && !strings.EqualFold(s.col, s.newName) {
			fail("duplicate column name: %s", s.newName)
		}
		f.putSchemaRow(tableRow, replaceTokens(t.SQL, d.refs, s.col, s.newName))
		for _, r := range indexRows {
			if r.sql == "" {
				continue
			}
			ix := newParser(r.sql).parseIndexDef(t)
			f.putSchemaRow(r, replaceTokens(r.sql, ix.refs, s.col, s.newName))
		}
		for _, v := range f.views {
			if mentions(v.SQL, s.col) {
				fail("view %s may use column %s; drop the view, rename, then create it again", v.Name, s.col)
			}
		}
	case "DROP COLUMN":
		ci := colIndex(t.Columns, s.col)
		if ci < 0 {
			fail("no such column: \"%s\"", s.col)
		}
		if len(t.Columns) == 1 {
			fail("cannot drop column \"%s\": no other columns exist", s.col)
		}
		if ci == t.RowidCol {
			fail("cannot drop PRIMARY KEY column: \"%s\"", s.col)
		}
		for _, ix := range t.Indexes {
			for _, ic := range ix.Cols {
				if ic.Col == ci {
					if ix.Auto {
						fail("cannot drop UNIQUE column: \"%s\"", s.col)
					}
					fail("error in index %s after drop column: no such column: %s", ix.Name, s.col)
				}
			}
			if ix.Partial != nil && mentionsExpr(ix.Partial, s.col) {
				fail("error in index %s after drop column: no such column: %s", ix.Name, s.col)
			}
		}
		span := d.colSpans[ci]
		for i, chk := range t.Checks {
			if mentionsExpr(chk, s.col) {
				inOwn := strings.Contains(t.SQL[span[0]:span[1]], t.CheckText[i])
				if !inOwn {
					fail("error in table %s after drop column: no such column: %s", t.Name, s.col)
				}
			}
		}
		// Remove the definition and the comma before it (or after it, for
		// the first column).
		start, end := span[0], span[1]
		if ci > 0 {
			start = strings.LastIndexByte(t.SQL[:start], ',')
		} else {
			end = strings.IndexByte(t.SQL[end:], ',') + end + 1
			for end < len(t.SQL) && t.SQL[end] == ' ' {
				end++
			}
		}
		sql := t.SQL[:start] + t.SQL[end:]
		// Rewrite every row without the column.
		type row struct {
			id   int64
			vals []Value
		}
		var rows []row
		f.scanTable(t.Root, func(rowid int64, payload []byte) error {
			vals, err := f.decodeRecord(payload)
			if err != nil {
				return err
			}
			rows = append(rows, row{rowid, vals})
			return nil
		})
		for _, rw := range rows {
			if ci < len(rw.vals) {
				nv := append(append([]Value{}, rw.vals[:ci]...), rw.vals[ci+1:]...)
				f.tablePut(t.Root, rw.id, f.encodeRecord(nv))
			}
		}
		f.putSchemaRow(tableRow, sql)
	}
	f.reloadSchema()
}

// checkViews fails if a view no longer works, as SQLite checks after
// renaming a table.
func (f *dbFile) checkViews() {
	f.reloadSchema()
	for _, r := range f.schemaRows {
		if r.kind != "view" {
			continue
		}
		var err error
		func() {
			defer catch(&err)
			f.planView(&View{Name: r.name, SQL: r.sql}, nil)
		}()
		if err != nil {
			fail("error in view %s: %s", r.name, strings.TrimPrefix(err.Error(), "SQL: "))
		}
	}
}

// viewTableTokens finds the names of tables in a view's SQL.
func viewTableTokens(sql string) []token {
	var toks []token
	p := newParser(sql)
	p.onTbl = func(tk token) { toks = append(toks, tk) }
	p.expectWord("CREATE")
	if !p.acceptWord("TEMP") {
		p.acceptWord("TEMPORARY")
	}
	p.expectWord("VIEW")
	if p.acceptWord("IF") {
		p.expectWord("NOT")
		p.expectWord("EXISTS")
	}
	p.expectName()
	if p.acceptOp(".") {
		p.expectName()
	}
	if p.isOp("(") {
		p.skipParens()
	}
	p.expectWord("AS")
	p.parseSelect()
	return toks
}

// replaceTokensAlways is replaceTokens with the new name always quoted.
func replaceTokensAlways(src string, toks []token, old, new string) string {
	var sb strings.Builder
	last := 0
	for _, tk := range toks {
		if !strings.EqualFold(tk.text, old) || tk.pos < last {
			continue
		}
		sb.WriteString(src[last:tk.pos])
		sb.WriteString(quoteAlways(new))
		last = tk.end
	}
	sb.WriteString(src[last:])
	return sb.String()
}

// mentions reports whether sql has a name token equal to name.
func mentions(sql, name string) bool {
	toks, err := lexSQL(sql)
	if err != nil {
		return true
	}
	for _, tk := range toks {
		if tk.kind == tIdent && strings.EqualFold(tk.text, name) {
			return true
		}
	}
	return false
}

func mentionsExpr(e expr, name string) bool {
	found := false
	walk(e, func(e expr) {
		if c, ok := e.(*colExpr); ok && strings.EqualFold(c.name, name) {
			found = true
		}
	})
	return found
}
