// Copyright 2017-2026 Nigele McCoy
// Licensed under the Apache License, Version 2.0;
// see LICENSE and NOTICE.

package sqlite

import (
	"strings"
)

// Table is one table from the schema.
type Table struct {
	Name        string
	Root        uint32 // root page of its B-tree
	Columns     []Column
	RowidCol    int // index of the INTEGER PRIMARY KEY column (which holds the rowid), or -1
	SQL         string
	Indexes     []*Index
	Autoinc     bool   // INTEGER PRIMARY KEY AUTOINCREMENT: rowids are never reused
	Strict      bool   // STRICT: values must match the column types
	Checks      []expr // CHECK constraints
	FKs         []*ForeignKey
	PK          []int // the PRIMARY KEY's columns, in order
	CheckText   []string
	pkConf      string // ON CONFLICT of the INTEGER PRIMARY KEY
	unsupported string // why this table can't be written, or ""
	readErr     string // why this table can't be read, or ""
	hasVirtual  bool   // has VIRTUAL generated columns, which the record leaves out

	// WITHOUT ROWID: stored as an index on the primary key (pkIdx), each
	// record in recOrder: the key's columns, then the rest.
	WithoutRowid bool
	pkIdx        *Index
	recOrder     []int
	pkDesc       bool // a column PRIMARY KEY DESC
}

// ForeignKey is a REFERENCES constraint: Cols of this table hold a key
// of Parent's ParentCols (its PRIMARY KEY when none are named).
type ForeignKey struct {
	Cols       []int
	Parent     string
	ParentCols []string
	OnDelete   string // NO ACTION, RESTRICT, CASCADE, SET NULL, SET DEFAULT
	OnUpdate   string
	Deferred   bool // DEFERRABLE INITIALLY DEFERRED: checked at COMMIT
	Match      string
}

// Column is one column of a table.
type Column struct {
	Name       string
	Type       string // as declared: "INTEGER", "VARCHAR(20)", "" ...
	Affinity   affinity
	Collate    string // BINARY, NOCASE, RTRIM, or "" (BINARY)
	NotNull    bool
	notNullOn  string // its ON CONFLICT, or ""
	Default    expr   // DEFAULT value, or nil
	DefaultSQL string // its text, as written
	Generated  bool
	GenExpr    expr // AS (expr)
	Stored     bool // STORED (in the record), else VIRTUAL (worked out when read)
}

// Index is one index from the schema: an ordered copy of some columns of
// a table, plus the rowid, used to find rows fast and to keep UNIQUE
// values unique.
type Index struct {
	Name    string
	Table   string
	Root    uint32
	Cols    []IndexCol
	Unique  bool
	Partial expr   // WHERE of a partial index, or nil
	HasExpr bool   // indexes an expression, not a column
	Auto    bool   // made for a UNIQUE or PRIMARY KEY constraint
	SQL     string // CREATE INDEX text, "" for an automatic index
	onConf  string
	Origin  string // c (CREATE INDEX), u (UNIQUE) or pk (PRIMARY KEY)

	namePos, tablePos, tableEnd int
	tableName                   string
	ifNotExists                 bool
	refs                        []token
}

type IndexCol struct {
	Expr expr // an expression, when Col is -1
	Col  int  // column of the table
	Coll string
	Desc bool
}

// View is a stored SELECT.
type View struct {
	Name string
	SQL  string
}

// Column affinity: how SQLite treats values compared with a column.
type affinity int

const (
	affBlob affinity = iota // no preference (also: no declared type)
	affText
	affNumeric
	affInteger
	affReal
)

// affinityOf applies SQLite's rules to a declared column type.
func affinityOf(declared string) affinity {
	t := strings.ToUpper(declared)
	switch {
	case strings.Contains(t, "INT"):
		return affInteger
	case strings.Contains(t, "CHAR"), strings.Contains(t, "CLOB"), strings.Contains(t, "TEXT"):
		return affText
	case strings.Contains(t, "BLOB"), t == "":
		return affBlob
	case strings.Contains(t, "REAL"), strings.Contains(t, "FLOA"), strings.Contains(t, "DOUB"):
		return affReal
	}
	return affNumeric
}

// The schema table itself, queryable as sqlite_schema or sqlite_master.
var schemaTable = &Table{
	Name: "sqlite_schema",
	Root: 1,
	Columns: []Column{
		{Name: "type", Type: "text", Affinity: affText},
		{Name: "name", Type: "text", Affinity: affText},
		{Name: "tbl_name", Type: "text", Affinity: affText},
		{Name: "rootpage", Type: "int", Affinity: affInteger},
		{Name: "sql", Type: "text", Affinity: affText},
	},
	RowidCol:    -1,
	unsupported: "sqlite_schema can't be changed directly",
}

// schemaRow is one row of sqlite_schema.
type schemaRow struct {
	rowid                int64
	kind, name, tbl, sql string
	root                 uint32
}

// loadSchema reads the tables, indexes and views listed in sqlite_schema
// (page 1).
func (db *dbFile) loadSchema() error {
	db.tables = map[string]*Table{}
	db.views = map[string]*View{}
	db.triggers = map[string][]*Trigger{}
	db.names = nil
	var rows []schemaRow
	err := db.scanTable(1, func(rowid int64, payload []byte) error {
		vals, err := db.decodeRecord(payload)
		if err != nil {
			return err
		}
		if len(vals) < 5 {
			return nil
		}
		r := schemaRow{rowid: rowid}
		r.kind, _ = vals[0].(string)
		r.name, _ = vals[1].(string)
		r.tbl, _ = vals[2].(string)
		root, _ := vals[3].(int64)
		r.root = uint32(root)
		r.sql, _ = vals[4].(string)
		rows = append(rows, r)
		return nil
	})
	if err != nil {
		return err
	}
	db.schemaRows = rows
	for _, r := range rows {
		if r.kind != "table" {
			continue
		}
		if strings.HasPrefix(strings.ToLower(r.name), "sqlite_") && !strings.EqualFold(r.name, "sqlite_sequence") && !strings.EqualFold(r.name, "sqlite_stat1") {
			continue
		}
		t, err := parseCreateTable(r.sql)
		if err != nil {
			return errorf("table %s: %v", r.name, err)
		}
		t.Name, t.Root, t.SQL = r.name, r.root, r.sql
		if t.pkIdx != nil {
			t.pkIdx.Root, t.pkIdx.Table = r.root, r.name
		}
		db.tables[strings.ToLower(r.name)] = t
		if !strings.HasPrefix(strings.ToLower(r.name), "sqlite_") {
			db.names = append(db.names, r.name)
		}
	}
	for _, r := range rows {
		switch r.kind {
		case "index":
			t := db.tables[strings.ToLower(r.tbl)]
			if t == nil {
				continue
			}
			var ix *Index
			if r.sql == "" {
				ix = t.autoIndex(r.name)
			} else {
				ix = parseCreateIndex(r.sql, t)
			}
			if ix == nil {
				t.unsupported = "its index " + r.name + " isn't understood"
				continue
			}
			ix.Name, ix.Table, ix.Root = r.name, t.Name, r.root
			t.Indexes = append(t.Indexes, ix)
		case "view":
			db.views[strings.ToLower(r.name)] = &View{Name: r.name, SQL: r.sql}
		case "trigger":
			tr, err := parseTriggerSQL(r.sql)
			if err != nil {
				if t := db.tables[strings.ToLower(r.tbl)]; t != nil {
					t.unsupported = "its trigger " + r.name + " isn't understood"
				}
				continue
			}
			tr.Name, tr.SQL = r.name, r.sql
			key := strings.ToLower(r.tbl)
			db.triggers[key] = append(db.triggers[key], tr)
		}
	}
	return nil
}

// Tables lists the database's tables, in the order they were created.
func (db *dbFile) Tables() []string { return append([]string{}, db.names...) }

// table finds a table by name, ignoring case like SQLite.
func (db *dbFile) table(name string) (*Table, error) {
	lower := strings.ToLower(name)
	if lower == "sqlite_schema" || lower == "sqlite_master" {
		return schemaTable, nil
	}
	if t, ok := db.tables[lower]; ok {
		if t.readErr != "" {
			return nil, errorf("%s", t.readErr)
		}
		return t, nil
	}
	if _, ok := db.views[lower]; ok {
		return nil, errorf("%s is a view, not a table", name)
	}
	return nil, errorf("no such table: %s", name)
}

// planView plans a view's SELECT afresh for each use.
func (db *dbFile) planView(v *View, env *cteEnv) *queryPlan {
	p := newParser(v.SQL)
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
	var cols []string
	if p.acceptOp("(") {
		for {
			cols = append(cols, p.expectName())
			if !p.acceptOp(",") {
				break
			}
		}
		p.expectOp(")")
	}
	p.expectWord("AS")
	q := db.planQuery(p.parseSelect(), nil, nil, nil)
	if len(cols) > 0 {
		if len(cols) != len(q.names) {
			fail("SQL: view %s has %d columns but its SELECT gives %d", v.Name, len(cols), len(q.names))
		}
		q.names = cols
	}
	return q
}

// tableDef is what parseTableDef learns from CREATE TABLE beyond the
// columns: the UNIQUE and PRIMARY KEY constraints, in order, which each
// have an automatic index.
type tableDef struct {
	t           *Table
	uniques     []*Index
	namePos     int // where the table's name starts in the SQL
	nameEnd     int // and ends
	listEnd     int // the ) closing the column list
	ifNotExists bool
	colSpans    [][2]int // each column definition's text
	refs        []token  // every name in the SQL that names one of its columns
}

// parseCreateTable reads a CREATE TABLE statement: the columns with
// their types, collations, defaults and NOT NULL, which column (if any)
// is an INTEGER PRIMARY KEY (stored as the rowid instead of in the
// record), CHECK constraints, and the UNIQUE and PRIMARY KEY constraints.
func parseCreateTable(src string) (t *Table, err error) {
	defer catch(&err)
	d := parseTableDef(newParser(src))
	return d.t, nil
}

func parseTableDef(p *sqlParser) *tableDef {
	p.expectWord("CREATE")
	if !p.acceptWord("TEMP") {
		p.acceptWord("TEMPORARY")
	}
	if p.acceptWord("VIRTUAL") {
		fail("virtual tables aren't supported")
	}
	p.expectWord("TABLE")
	ifNotExists := false
	if p.acceptWord("IF") {
		p.expectWord("NOT")
		p.expectWord("EXISTS")
		ifNotExists = true
	}
	t := &Table{RowidCol: -1}
	d := &tableDef{t: t}
	d.ifNotExists = ifNotExists
	d.namePos = p.peek().pos
	t.Name = p.expectName()
	if p.acceptOp(".") {
		d.namePos = p.peek().pos
		t.Name = p.expectName()
	}
	d.nameEnd = p.toks[p.pos-1].end
	if p.isWord("AS") {
		fail("CREATE TABLE ... AS SELECT isn't supported here")
	}
	p.expectOp("(")
	p.onCol = func(tk token) { d.refs = append(d.refs, tk) }
	defer func() { p.onCol = nil }()
	var pkCols []IndexCol
	pkConf := ""
	pkSeen := false
	for {
		if p.isWord("CONSTRAINT", "PRIMARY", "UNIQUE", "CHECK", "FOREIGN") {
			if p.acceptWord("CONSTRAINT") {
				p.expectName()
			}
			switch {
			case p.acceptWord("PRIMARY"):
				p.expectWord("KEY")
				if pkSeen {
					fail("table %q has more than one primary key", t.Name)
				}
				pkSeen = true
				pkCols = p.parseIndexedCols(t)
				for _, ic := range pkCols {
					t.PK = append(t.PK, ic.Col)
				}
				pkConf = p.parseConflict()
				if p.acceptWord("AUTOINCREMENT") {
					t.Autoinc = true
				}
				d.addUnique(pkCols, pkConf, true)
			case p.acceptWord("UNIQUE"):
				cols := p.parseIndexedCols(t)
				d.addUnique(cols, p.parseConflict(), false)
			case p.acceptWord("CHECK"):
				p.parseCheck(t)
			case p.acceptWord("FOREIGN"):
				p.expectWord("KEY")
				p.expectOp("(")
				var cols []int
				for {
					if p.onCol != nil {
						p.onCol(p.peek())
					}
					name := p.expectName()
					ci := colIndex(t.Columns, name)
					if ci < 0 {
						fail("unknown column \"%s\" in foreign key definition", name)
					}
					cols = append(cols, ci)
					if !p.acceptOp(",") {
						break
					}
				}
				p.expectOp(")")
				p.expectWord("REFERENCES")
				t.FKs = append(t.FKs, p.parseReferences(cols))
			}
		} else {
			start := p.peek().pos
			p.parseColumnDef(d, &pkSeen)
			d.colSpans = append(d.colSpans, [2]int{start, p.toks[p.pos-1].end})
		}
		if !p.acceptOp(",") {
			break
		}
	}
	d.listEnd = p.peek().pos
	p.expectOp(")")
	for !p.atStatementEnd() {
		switch {
		case p.acceptWord("WITHOUT"):
			p.expectWord("ROWID")
			t.WithoutRowid = true
		case p.acceptWord("STRICT"):
			t.Strict = true
		case p.acceptOp(","):
		default:
			p.near("unexpected")
		}
	}
	if len(t.Columns) == 0 {
		fail("no columns in table %s", t.Name)
	}
	// A table-level PRIMARY KEY (a) makes a the rowid when a is the only
	// key column and declared INTEGER.
	if pkSeen && !t.WithoutRowid && t.RowidCol < 0 && len(pkCols) == 1 && strings.EqualFold(t.Columns[pkCols[0].Col].Type, "INTEGER") {
		t.RowidCol = pkCols[0].Col
		t.pkConf = pkConf
		d.dropUnique(pkCols)
	}
	if t.WithoutRowid {
		if !pkSeen {
			fail("PRIMARY KEY missing on table %s", t.Name)
		}
		if t.Autoinc {
			fail("AUTOINCREMENT not allowed on WITHOUT ROWID tables")
		}
		key := pkCols
		if t.RowidCol >= 0 {
			// INTEGER PRIMARY KEY is an ordinary key here, not the rowid.
			ci := t.RowidCol
			t.RowidCol = -1
			key = []IndexCol{{Col: ci, Coll: t.Columns[ci].Collate, Desc: t.pkDesc}}
			d.addUnique(key, t.pkConf, true)
		} else if key == nil {
			for _, u := range d.uniques {
				if u.Origin == "pk" {
					key = u.Cols
				}
			}
		}
		// The table is stored as an index on its key; a record holds the
		// key's columns first, then the others.
		t.pkIdx = &Index{Name: "sqlite_autoindex_" + t.Name, Table: t.Name, Cols: key, Unique: true, Origin: "pk"}
		inKey := map[int]bool{}
		for _, ic := range key {
			t.recOrder = append(t.recOrder, ic.Col)
			inKey[ic.Col] = true
		}
		for i := range t.Columns {
			if !inKey[i] {
				t.recOrder = append(t.recOrder, i)
			}
		}
	}
	if t.Autoinc && t.RowidCol < 0 {
		fail("AUTOINCREMENT is only allowed on an INTEGER PRIMARY KEY")
	}
	return d
}

// autoIndex is the automatic index called name (sqlite_autoindex_t_N):
// the Nth UNIQUE or PRIMARY KEY constraint.
func (t *Table) autoIndex(name string) *Index {
	d := parseTableDef(newParserOrNil(t.SQL))
	n := 0
	if i := strings.LastIndexByte(name, '_'); i >= 0 {
		n = atoiOr(name[i+1:], 0)
	}
	if n < 1 || n > len(d.uniques) {
		return nil
	}
	return d.uniques[n-1]
}

func newParserOrNil(sql string) *sqlParser { return newParser(sql) }

func atoiOr(s string, def int) int {
	n := 0
	if s == "" {
		return def
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return def
		}
		n = n*10 + int(c-'0')
	}
	return n
}

// addUnique records a UNIQUE or PRIMARY KEY constraint, unless an earlier
// one covers exactly the same columns (SQLite makes one index for both).
func (d *tableDef) addUnique(cols []IndexCol, conf string, pk bool) {
	for _, u := range d.uniques {
		if sameIndexCols(u.Cols, cols) {
			return
		}
	}
	origin := "u"
	if pk {
		origin = "pk"
	}
	d.uniques = append(d.uniques, &Index{Cols: cols, Unique: true, Auto: true, onConf: conf, Origin: origin})
}

func (d *tableDef) dropUnique(cols []IndexCol) {
	for i, u := range d.uniques {
		if sameIndexCols(u.Cols, cols) {
			d.uniques = append(d.uniques[:i], d.uniques[i+1:]...)
			return
		}
	}
}

func sameIndexCols(a, b []IndexCol) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].Col != b[i].Col || !strings.EqualFold(a[i].Coll, b[i].Coll) {
			return false
		}
	}
	return true
}

func (p *sqlParser) parseColumnDef(d *tableDef, pkSeen *bool) {
	t := d.t
	if p.onCol != nil {
		p.onCol(p.peek())
	}
	col := Column{Name: p.expectName()}
	if colIndex(t.Columns, col.Name) >= 0 {
		fail("duplicate column name: %s", col.Name)
	}
	ci := len(t.Columns)
	// The type is the words after the name, up to the first constraint
	// keyword, with any parenthesized size.
	var typ []string
	for p.peek().kind == tIdent && !(!p.peek().quoted && isConstraintWord(p.peek().text)) {
		typ = append(typ, p.next().text)
		if p.isOp("(") {
			start := p.peek().pos
			p.skipParens()
			typ[len(typ)-1] += p.src[start:p.toks[p.pos-1].end]
		}
	}
	col.Type = strings.Join(typ, " ")
	col.Affinity = affinityOf(col.Type)
	t.Columns = append(t.Columns, col)
	c := &t.Columns[ci]
	for {
		if p.acceptWord("CONSTRAINT") {
			p.expectName()
		}
		switch {
		case p.acceptWord("PRIMARY"):
			p.expectWord("KEY")
			if *pkSeen {
				fail("table %q has more than one primary key", t.Name)
			}
			*pkSeen = true
			desc := p.acceptWord("DESC")
			if !desc {
				p.acceptWord("ASC")
			}
			t.pkDesc = desc
			conf := p.parseConflict()
			auto := p.acceptWord("AUTOINCREMENT")
			t.PK = []int{ci}
			if strings.EqualFold(c.Type, "INTEGER") && !desc {
				// (Undone at the end for a WITHOUT ROWID table.)
				t.RowidCol = ci
				t.pkConf = conf
				t.Autoinc = auto
			} else {
				if auto {
					fail("AUTOINCREMENT is only allowed on an INTEGER PRIMARY KEY")
				}
				d.addUnique([]IndexCol{{Col: ci, Coll: c.Collate, Desc: desc}}, conf, true)
			}
		case p.acceptWord("NOT"):
			p.expectWord("NULL")
			c.NotNull = true
			c.notNullOn = p.parseConflict()
		case p.acceptWord("NULL"):
			p.parseConflict()
		case p.acceptWord("UNIQUE"):
			d.addUnique([]IndexCol{{Col: ci, Coll: c.Collate}}, p.parseConflict(), false)
		case p.acceptWord("CHECK"):
			p.parseCheck(t)
		case p.acceptWord("DEFAULT"):
			start := p.peek().pos
			if p.acceptOp("(") {
				c.Default = p.parseExpr()
				p.expectOp(")")
			} else if p.isOp("-") || p.isOp("+") {
				c.Default = p.parseUnary()
			} else {
				c.Default = p.parsePrimary()
				if col, ok := c.Default.(*colExpr); ok {
					c.Default = &litExpr{v: col.name} // DEFAULT word: the word as text
				}
			}
			c.DefaultSQL = p.src[start:p.toks[p.pos-1].end]
		case p.acceptWord("COLLATE"):
			c.Collate = collationName(p.expectName())
		case p.acceptWord("REFERENCES"):
			t.FKs = append(t.FKs, p.parseReferences([]int{ci}))
		case p.isWord("GENERATED", "AS"):
			if p.acceptWord("GENERATED") {
				p.expectWord("ALWAYS")
			}
			p.expectWord("AS")
			p.expectOp("(")
			c.GenExpr = p.parseExpr()
			p.expectOp(")")
			c.Generated = true
			if p.acceptWord("STORED") {
				c.Stored = true
			} else {
				p.acceptWord("VIRTUAL")
				t.hasVirtual = true
			}
		default:
			return
		}
	}
}

// parseIndexedCols reads ( col [COLLATE c] [ASC|DESC], ... ) for a
// table's PRIMARY KEY or UNIQUE constraint.
func (p *sqlParser) parseIndexedCols(t *Table) []IndexCol {
	p.expectOp("(")
	var cols []IndexCol
	for {
		if p.onCol != nil {
			p.onCol(p.peek())
		}
		name := p.expectName()
		ci := colIndex(t.Columns, name)
		if ci < 0 {
			fail("no such column: %s", name)
		}
		ic := IndexCol{Col: ci, Coll: t.Columns[ci].Collate}
		if p.acceptWord("COLLATE") {
			ic.Coll = collationName(p.expectName())
		}
		if p.acceptWord("DESC") {
			ic.Desc = true
		} else {
			p.acceptWord("ASC")
		}
		cols = append(cols, ic)
		if !p.acceptOp(",") {
			break
		}
	}
	p.expectOp(")")
	return cols
}

// parseCheck reads CHECK ( expr ), keeping its text for error messages.
func (p *sqlParser) parseCheck(t *Table) {
	p.expectOp("(")
	start := p.peek().pos
	t.Checks = append(t.Checks, p.parseExpr())
	t.CheckText = append(t.CheckText, p.src[start:p.toks[p.pos-1].end])
	p.expectOp(")")
}

// parseConflict reads ON CONFLICT ROLLBACK/ABORT/FAIL/IGNORE/REPLACE.
func (p *sqlParser) parseConflict() string {
	if !p.isWord("ON") || !isWord(p.peekAt(1), "CONFLICT") {
		return ""
	}
	p.next()
	p.next()
	w := strings.ToUpper(p.expectName())
	switch w {
	case "ROLLBACK", "ABORT", "FAIL", "IGNORE", "REPLACE":
		return w
	}
	fail("SQL: unknown conflict resolution %s", w)
	return ""
}

// skipParens skips a ( ... ) group, nested.
func (p *sqlParser) skipParens() {
	p.expectOp("(")
	depth := 1
	for depth > 0 {
		t := p.next()
		switch {
		case t.kind == tEOF:
			fail("SQL: a ( isn't closed")
		case t.kind == tOp && t.text == "(":
			depth++
		case t.kind == tOp && t.text == ")":
			depth--
		}
	}
}

// parseReferences reads what follows REFERENCES: the parent table, its
// columns, ON DELETE / ON UPDATE actions, MATCH, and DEFERRABLE.
func (p *sqlParser) parseReferences(cols []int) *ForeignKey {
	fk := &ForeignKey{Cols: cols, Parent: p.expectName(), OnDelete: "NO ACTION", OnUpdate: "NO ACTION", Match: "NONE"}
	if p.acceptOp("(") {
		for {
			fk.ParentCols = append(fk.ParentCols, p.expectName())
			if !p.acceptOp(",") {
				break
			}
		}
		p.expectOp(")")
	}
	for {
		switch {
		case p.acceptWord("ON"):
			del := p.acceptWord("DELETE")
			if !del {
				p.expectWord("UPDATE")
			}
			var action string
			switch {
			case p.acceptWord("SET"):
				if p.acceptWord("NULL") {
					action = "SET NULL"
				} else {
					p.expectWord("DEFAULT")
					action = "SET DEFAULT"
				}
			case p.acceptWord("NO"):
				p.expectWord("ACTION")
				action = "NO ACTION"
			case p.acceptWord("CASCADE"):
				action = "CASCADE"
			default:
				p.expectWord("RESTRICT")
				action = "RESTRICT"
			}
			if del {
				fk.OnDelete = action
			} else {
				fk.OnUpdate = action
			}
		case p.acceptWord("MATCH"):
			fk.Match = strings.ToUpper(p.expectName())
		case p.isWord("NOT") && isWord(p.peekAt(1), "DEFERRABLE"):
			p.next()
			p.next()
			p.skipDeferrable()
		case p.acceptWord("DEFERRABLE"):
			if p.acceptWord("INITIALLY") {
				fk.Deferred = p.acceptWord("DEFERRED")
				if !fk.Deferred {
					p.expectWord("IMMEDIATE")
				}
			}
		default:
			if len(fk.ParentCols) > 0 && len(fk.ParentCols) != len(fk.Cols) {
				fail("number of columns in foreign key does not match the number of columns in the referenced table")
			}
			return fk
		}
	}
}

func (p *sqlParser) skipDeferrable() {
	if p.acceptWord("INITIALLY") {
		p.next()
	}
}

func isConstraintWord(w string) bool {
	switch strings.ToUpper(w) {
	case "CONSTRAINT", "PRIMARY", "NOT", "NULL", "UNIQUE", "CHECK", "DEFAULT", "COLLATE", "REFERENCES", "GENERATED", "AS":
		return true
	}
	return false
}

// parseCreateIndex reads CREATE [UNIQUE] INDEX name ON t (cols) [WHERE
// ...] for table t. Expression columns mark it HasExpr.
func parseCreateIndex(src string, t *Table) (ix *Index) {
	defer func() {
		if r := recover(); r != nil {
			if _, ok := r.(*Error); !ok {
				panic(r)
			}
			ix = nil
		}
	}()
	p := newParser(src)
	return p.parseIndexDef(t)
}

func (p *sqlParser) parseIndexDef(t *Table) *Index {
	p.expectWord("CREATE")
	ix := &Index{}
	ix.Unique = p.acceptWord("UNIQUE")
	p.expectWord("INDEX")
	if p.acceptWord("IF") {
		p.expectWord("NOT")
		p.expectWord("EXISTS")
		ix.ifNotExists = true
	}
	ix.namePos = p.peek().pos
	ix.Name = p.expectName()
	if p.acceptOp(".") {
		ix.namePos = p.peek().pos
		ix.Name = p.expectName()
	}
	p.expectWord("ON")
	ix.tablePos = p.peek().pos
	ix.tableName = p.expectName()
	ix.tableEnd = p.toks[p.pos-1].end
	p.onCol = func(tk token) { ix.refs = append(ix.refs, tk) }
	defer func() { p.onCol = nil }()
	p.expectOp("(")
	for {
		start := p.pos
		e := p.parseExpr()
		ic := IndexCol{Col: -1}
		coll := ""
		if ce, ok := e.(*collateExpr); ok {
			coll, e = ce.coll, ce.x
		}
		if col, ok := e.(*colExpr); ok && col.table == "" {
			ic.Col = colIndex(t.Columns, col.name)
			if ic.Col < 0 {
				fail("no such column: %s", col.name)
			}
			ic.Coll = t.Columns[ic.Col].Collate
		} else if lit, ok := e.(*litExpr); ok && p.toks[start].kind == tString {
			// CREATE INDEX i ON t ('a'): a quoted string naming a column.
			if s, ok := lit.v.(string); ok {
				ic.Col = colIndex(t.Columns, s)
			}
			if ic.Col < 0 {
				ix.HasExpr, ic.Expr = true, e
			}
		} else {
			ix.HasExpr, ic.Expr = true, e
		}
		if coll != "" {
			ic.Coll = coll
		}
		if p.acceptWord("DESC") {
			ic.Desc = true
		} else {
			p.acceptWord("ASC")
		}
		ix.Cols = append(ix.Cols, ic)
		if !p.acceptOp(",") {
			break
		}
	}
	p.expectOp(")")
	if p.acceptWord("WHERE") {
		ix.Partial = p.parseExpr()
	}
	return ix
}

// isWord reports whether tk is an unquoted identifier equal to one of
// words, ignoring case.
func isWord(tk token, words ...string) bool {
	if tk.kind != tIdent || tk.quoted {
		return false
	}
	for _, w := range words {
		if strings.EqualFold(tk.text, w) {
			return true
		}
	}
	return false
}

// parseTriggerSQL reads a stored CREATE TRIGGER.
func parseTriggerSQL(sql string) (tr *Trigger, err error) {
	defer catch(&err)
	p := newParser(sql)
	p.expectWord("CREATE")
	if !p.acceptWord("TEMP") {
		p.acceptWord("TEMPORARY")
	}
	tr = p.parseTrigger()
	return tr, nil
}
