package sqlite

import (
	"bytes"
	"math"
	"sort"
	"strconv"
	"strings"
)

// Query runs one SELECT statement with ? parameters and returns the
// column names and rows.
//
// Supported: SELECT [DISTINCT] columns or expressions [AS name], or *,
// FROM one table (or none), WHERE, ORDER BY (expressions, result names,
// or positions; ASC/DESC), LIMIT/OFFSET, and the aggregates count, sum,
// total, avg, min, max, group_concat over the whole result. Expressions
// follow SQLite: comparisons apply column affinity, NULL propagates,
// integer division truncates, x/0 is NULL.
func (db *DB) Query(sql string, params []Value) (cols []string, rows [][]Value, err error) {
	defer func() {
		if r := recover(); r != nil {
			e, ok := r.(*Error)
			if !ok {
				panic(r)
			}
			cols, rows, err = nil, nil, e
		}
	}()
	toks, err := lexSQL(sql)
	if err != nil {
		return nil, nil, err
	}
	p := &sqlParser{toks: toks, src: sql}
	if !p.isWord("SELECT") {
		word := p.peek().text
		if p.peek().kind == tEOF {
			fail("SQL: the statement is empty")
		}
		fail("SQL: only SELECT is supported so far, not %s (writing comes in the next version)", strings.ToUpper(word))
	}
	stmt := p.parseSelect()
	if p.peek().kind == tOp && p.peek().text == ";" {
		p.next()
	}
	if p.peek().kind != tEOF {
		fail("SQL: unexpected %q after the end of the statement", p.peek().text)
	}
	if p.params != len(params) {
		fail("SQL: the query has %d ? placeholder(s) but %d value(s) were given", p.params, len(params))
	}
	return db.run(stmt, params)
}

// fail stops the query with an error (recovered in Query).
func fail(format string, args ...any) {
	panic(errorf(format, args...).(*Error))
}

// ---- syntax tree --------------------------------------------------------

type expr interface{}

type (
	litExpr   struct{ v Value }
	paramExpr struct{ idx int }
	colExpr   struct {
		table, name string
		idx         int  // column index, resolved against the FROM table
		rowid       bool // the rowid itself (rowid, oid, _rowid_, or an INTEGER PRIMARY KEY)
		aff         affinity
	}
	unaryExpr struct {
		op string
		x  expr
	}
	binExpr struct {
		op   string
		l, r expr
	}
	isExpr struct {
		l, r expr
		not  bool
	}
	isNullExpr struct {
		x   expr
		not bool
	}
	inExpr struct {
		x    expr
		list []expr
		not  bool
	}
	betweenExpr struct {
		x, lo, hi expr
		not       bool
	}
	likeExpr struct {
		x, pat, esc expr
		not         bool
	}
	callExpr struct {
		name     string // lowercase
		args     []expr
		star     bool // count(*)
		distinct bool
		agg      int // index into the aggregate states, or -1
	}
	caseExpr struct {
		base  expr
		whens [][2]expr
		els   expr
	}
	castExpr struct {
		x   expr
		typ string
	}
)

type resultCol struct {
	e    expr
	star bool
	name string
	bare bool // a plain column, named after the column
}

type orderTerm struct {
	e    expr
	desc bool
	pos  int // 1-based result column, or 0
}

type selectStmt struct {
	distinct  bool
	cols      []resultCol
	fromName  string
	fromAlias string
	where     expr
	order     []orderTerm
	limit     expr
	offset    expr
}

// ---- parser -------------------------------------------------------------

type sqlParser struct {
	toks   []token
	pos    int
	src    string
	params int // ? placeholders seen
}

func (p *sqlParser) peek() token { return p.toks[p.pos] }
func (p *sqlParser) next() token {
	t := p.toks[p.pos]
	if p.pos < len(p.toks)-1 {
		p.pos++
	}
	return t
}
func (p *sqlParser) isWord(words ...string) bool { return isWord(p.peek(), words...) }
func (p *sqlParser) isOp(op string) bool {
	return p.peek().kind == tOp && p.peek().text == op
}
func (p *sqlParser) acceptWord(w string) bool {
	if p.isWord(w) {
		p.next()
		return true
	}
	return false
}
func (p *sqlParser) acceptOp(op string) bool {
	if p.isOp(op) {
		p.next()
		return true
	}
	return false
}
func (p *sqlParser) expectWord(w string) {
	if !p.acceptWord(w) {
		fail("SQL: expected %s near %q", w, p.peek().text)
	}
}
func (p *sqlParser) expectOp(op string) {
	if !p.acceptOp(op) {
		fail("SQL: expected %q near %q", op, p.peek().text)
	}
}

// clauseWords end an expression or a result column's bare alias.
var clauseWords = []string{"FROM", "WHERE", "ORDER", "LIMIT", "OFFSET", "GROUP", "HAVING", "UNION", "EXCEPT", "INTERSECT", "AS", "ASC", "DESC", "AND", "OR", "NOT", "ON", "JOIN"}

func (p *sqlParser) parseSelect() *selectStmt {
	p.expectWord("SELECT")
	s := &selectStmt{}
	if p.acceptWord("DISTINCT") {
		s.distinct = true
	} else {
		p.acceptWord("ALL")
	}
	for {
		if p.acceptOp("*") {
			s.cols = append(s.cols, resultCol{star: true})
		} else {
			start := p.peek().pos
			e := p.parseExpr()
			end := p.toks[p.pos-1].end
			c := resultCol{e: e, name: strings.TrimSpace(p.src[start:end])}
			if col, ok := e.(*colExpr); ok {
				c.name = col.name
				c.bare = true
			}
			if p.acceptWord("AS") {
				c.name = p.expectName()
			} else if p.peek().kind == tIdent && (p.peek().quoted || !p.isWord(clauseWords...)) {
				c.name = p.next().text
			}
			s.cols = append(s.cols, c)
		}
		if !p.acceptOp(",") {
			break
		}
	}
	if p.acceptWord("FROM") {
		s.fromName = p.expectName()
		if p.acceptOp(".") { // schema.table: main.books
			s.fromName = p.expectName()
		}
		if p.acceptWord("AS") {
			s.fromAlias = p.expectName()
		} else if p.peek().kind == tIdent && (p.peek().quoted || !p.isWord(clauseWords...)) {
			s.fromAlias = p.next().text
		}
		if p.isOp(",") || p.isWord("JOIN") {
			fail("SQL: joins aren't supported yet")
		}
	}
	if p.acceptWord("WHERE") {
		s.where = p.parseExpr()
	}
	if p.isWord("GROUP") {
		fail("SQL: GROUP BY isn't supported yet")
	}
	if p.acceptWord("ORDER") {
		p.expectWord("BY")
		for {
			t := orderTerm{e: p.parseExpr()}
			if lit, ok := t.e.(*litExpr); ok {
				if n, ok := lit.v.(int64); ok {
					t.pos = int(n)
				}
			}
			if p.acceptWord("DESC") {
				t.desc = true
			} else {
				p.acceptWord("ASC")
			}
			s.order = append(s.order, t)
			if !p.acceptOp(",") {
				break
			}
		}
	}
	if p.acceptWord("LIMIT") {
		s.limit = p.parseExpr()
		if p.acceptWord("OFFSET") {
			s.offset = p.parseExpr()
		} else if p.acceptOp(",") { // LIMIT offset, count
			s.offset, s.limit = s.limit, p.parseExpr()
		}
	}
	if p.isWord("UNION", "EXCEPT", "INTERSECT") {
		fail("SQL: %s isn't supported yet", strings.ToUpper(p.peek().text))
	}
	return s
}

func (p *sqlParser) expectName() string {
	t := p.next()
	if t.kind != tIdent {
		fail("SQL: expected a name near %q", t.text)
	}
	return t.text
}

func (p *sqlParser) parseExpr() expr { return p.parseOr() }

func (p *sqlParser) parseOr() expr {
	l := p.parseAnd()
	for p.acceptWord("OR") {
		l = &binExpr{op: "OR", l: l, r: p.parseAnd()}
	}
	return l
}

func (p *sqlParser) parseAnd() expr {
	l := p.parseNot()
	for p.acceptWord("AND") {
		l = &binExpr{op: "AND", l: l, r: p.parseNot()}
	}
	return l
}

func (p *sqlParser) parseNot() expr {
	if p.acceptWord("NOT") {
		return &unaryExpr{op: "NOT", x: p.parseNot()}
	}
	return p.parseEquality()
}

func (p *sqlParser) parseEquality() expr {
	l := p.parseComparison()
	for {
		switch {
		case p.isOp("=") || p.isOp("==") || p.isOp("!=") || p.isOp("<>"):
			op := p.next().text
			if op == "==" {
				op = "="
			} else if op == "<>" {
				op = "!="
			}
			l = &binExpr{op: op, l: l, r: p.parseComparison()}
		case p.isWord("IS"):
			p.next()
			not := p.acceptWord("NOT")
			if p.acceptWord("NULL") {
				l = &isNullExpr{x: l, not: not}
			} else {
				l = &isExpr{l: l, r: p.parseComparison(), not: not}
			}
		case p.isWord("ISNULL"):
			p.next()
			l = &isNullExpr{x: l}
		case p.isWord("NOTNULL"):
			p.next()
			l = &isNullExpr{x: l, not: true}
		case p.isWord("NOT") && p.pos+1 < len(p.toks) && isWord(p.toks[p.pos+1], "IN", "LIKE", "BETWEEN", "NULL"):
			p.next()
			l = p.parseNegatable(l, true)
		case p.isWord("IN", "LIKE", "BETWEEN"):
			l = p.parseNegatable(l, false)
		default:
			return l
		}
	}
}

// parseNegatable parses [NOT] IN / LIKE / BETWEEN / NULL after l.
func (p *sqlParser) parseNegatable(l expr, not bool) expr {
	switch {
	case p.acceptWord("NULL"):
		return &isNullExpr{x: l, not: true}
	case p.acceptWord("IN"):
		p.expectOp("(")
		in := &inExpr{x: l, not: not}
		if p.isWord("SELECT") {
			fail("SQL: IN (SELECT ...) isn't supported yet")
		}
		if !p.isOp(")") {
			for {
				in.list = append(in.list, p.parseExpr())
				if !p.acceptOp(",") {
					break
				}
			}
		}
		p.expectOp(")")
		return in
	case p.acceptWord("LIKE"):
		lk := &likeExpr{x: l, pat: p.parseComparison(), not: not}
		if p.acceptWord("ESCAPE") {
			lk.esc = p.parseComparison()
		}
		return lk
	}
	p.expectWord("BETWEEN")
	lo := p.parseComparison()
	p.expectWord("AND")
	return &betweenExpr{x: l, lo: lo, hi: p.parseComparison(), not: not}
}

func (p *sqlParser) parseComparison() expr {
	l := p.parseBits()
	for p.isOp("<") || p.isOp("<=") || p.isOp(">") || p.isOp(">=") {
		op := p.next().text
		l = &binExpr{op: op, l: l, r: p.parseBits()}
	}
	return l
}

func (p *sqlParser) parseBits() expr {
	l := p.parseAdd()
	for p.isOp("&") || p.isOp("|") || p.isOp("<<") || p.isOp(">>") {
		op := p.next().text
		l = &binExpr{op: op, l: l, r: p.parseAdd()}
	}
	return l
}

func (p *sqlParser) parseAdd() expr {
	l := p.parseMul()
	for p.isOp("+") || p.isOp("-") {
		op := p.next().text
		l = &binExpr{op: op, l: l, r: p.parseMul()}
	}
	return l
}

func (p *sqlParser) parseMul() expr {
	l := p.parseConcat()
	for p.isOp("*") || p.isOp("/") || p.isOp("%") {
		op := p.next().text
		l = &binExpr{op: op, l: l, r: p.parseConcat()}
	}
	return l
}

func (p *sqlParser) parseConcat() expr {
	l := p.parseUnary()
	for p.acceptOp("||") {
		l = &binExpr{op: "||", l: l, r: p.parseUnary()}
	}
	return l
}

func (p *sqlParser) parseUnary() expr {
	if p.isOp("-") || p.isOp("+") || p.isOp("~") {
		op := p.next().text
		x := p.parseUnary()
		// Fold -5 into a literal, so ORDER BY -1 isn't a position and
		// LIMIT -1 reads naturally.
		if lit, ok := x.(*litExpr); ok && op == "-" {
			switch v := lit.v.(type) {
			case int64:
				if v != math.MinInt64 {
					return &litExpr{v: -v}
				}
			case float64:
				return &litExpr{v: -v}
			}
		}
		return &unaryExpr{op: op, x: x}
	}
	return p.parsePrimary()
}

func (p *sqlParser) parsePrimary() expr {
	t := p.peek()
	switch t.kind {
	case tNumber:
		p.next()
		return &litExpr{v: parseNumberLiteral(t.text)}
	case tString:
		p.next()
		return &litExpr{v: t.text}
	case tBlob:
		p.next()
		b, ok := decodeHex(t.text)
		if !ok {
			fail("SQL: bad blob literal X'%s'", t.text)
		}
		return &litExpr{v: b}
	case tParam:
		p.next()
		if len(t.text) > 1 {
			n, err := strconv.Atoi(t.text[1:])
			if err != nil || n < 1 {
				fail("SQL: bad placeholder %s", t.text)
			}
			if n > p.params {
				p.params = n
			}
			return &paramExpr{idx: n - 1}
		}
		p.params++
		return &paramExpr{idx: p.params - 1}
	case tOp:
		if p.acceptOp("(") {
			if p.isWord("SELECT") {
				fail("SQL: subqueries aren't supported yet")
			}
			e := p.parseExpr()
			p.expectOp(")")
			return e
		}
	case tIdent:
		if !t.quoted {
			switch strings.ToUpper(t.text) {
			case "NULL":
				p.next()
				return &litExpr{v: nil}
			case "TRUE":
				p.next()
				return &litExpr{v: int64(1)}
			case "FALSE":
				p.next()
				return &litExpr{v: int64(0)}
			case "CASE":
				return p.parseCase()
			case "CAST":
				p.next()
				p.expectOp("(")
				x := p.parseExpr()
				p.expectWord("AS")
				var typ []string
				for !p.isOp(")") && p.peek().kind != tEOF {
					typ = append(typ, p.next().text)
				}
				p.expectOp(")")
				return &castExpr{x: x, typ: strings.Join(typ, " ")}
			}
		}
		p.next()
		if !t.quoted && p.isOp("(") {
			return p.parseCall(strings.ToLower(t.text))
		}
		if p.acceptOp(".") {
			return &colExpr{table: t.text, name: p.expectName(), idx: -1}
		}
		return &colExpr{name: t.text, idx: -1}
	case tEOF:
		fail("SQL: the statement ends too soon")
	}
	fail("SQL: unexpected %q", t.text)
	return nil
}

func (p *sqlParser) parseCall(name string) expr {
	p.expectOp("(")
	c := &callExpr{name: name, agg: -1}
	if p.acceptOp("*") {
		c.star = true
	} else if !p.isOp(")") {
		c.distinct = p.acceptWord("DISTINCT")
		for {
			c.args = append(c.args, p.parseExpr())
			if !p.acceptOp(",") {
				break
			}
		}
	}
	p.expectOp(")")
	return c
}

func (p *sqlParser) parseCase() expr {
	p.expectWord("CASE")
	c := &caseExpr{}
	if !p.isWord("WHEN") {
		c.base = p.parseExpr()
	}
	for p.acceptWord("WHEN") {
		w := p.parseExpr()
		p.expectWord("THEN")
		c.whens = append(c.whens, [2]expr{w, p.parseExpr()})
	}
	if len(c.whens) == 0 {
		fail("SQL: CASE needs at least one WHEN")
	}
	if p.acceptWord("ELSE") {
		c.els = p.parseExpr()
	}
	p.expectWord("END")
	return c
}

func parseNumberLiteral(s string) Value {
	if strings.HasPrefix(s, "0x") || strings.HasPrefix(s, "0X") {
		u, err := strconv.ParseUint(s[2:], 16, 64)
		if err != nil {
			fail("SQL: hex number %s is too big", s)
		}
		return int64(u)
	}
	if !strings.ContainsAny(s, ".eE") {
		if n, err := strconv.ParseInt(s, 10, 64); err == nil {
			return n
		}
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		fail("SQL: bad number %s", s)
	}
	return f
}

func decodeHex(s string) ([]byte, bool) {
	if len(s)%2 != 0 {
		return nil, false
	}
	out := make([]byte, len(s)/2)
	for i := range out {
		n, err := strconv.ParseUint(s[2*i:2*i+2], 16, 8)
		if err != nil {
			return nil, false
		}
		out[i] = byte(n)
	}
	return out, true
}

// ---- running a SELECT -------------------------------------------------------

type aggState struct {
	call  *callExpr
	count int64
	sum   Value // int64 while every value is an integer, else float64
	best  Value
	seen  bool
	parts []string
	dist  map[string]bool
}

type runner struct {
	params []Value
	table  *Table
	row    []Value
	rowid  int64
	aggs   []*aggState
	final  bool // aggregates finished: callExprs read their result
}

func (db *DB) run(s *selectStmt, params []Value) ([]string, [][]Value, error) {
	r := &runner{params: params}
	if s.fromName != "" {
		t, err := db.table(s.fromName)
		if err != nil {
			return nil, nil, err
		}
		r.table = t
	}
	// Expand * and resolve column names.
	var cols []resultCol
	for _, c := range s.cols {
		if c.star {
			if r.table == nil {
				fail("SQL: SELECT * needs a FROM table")
			}
			for i, tc := range r.table.Columns {
				cols = append(cols, resultCol{e: &colExpr{name: tc.Name, idx: i, rowid: i == r.table.RowidCol, aff: tc.Affinity}, name: tc.Name})
			}
			continue
		}
		r.resolve(c.e, s)
		// SELECT rowid names the column after an INTEGER PRIMARY KEY, as
		// SQLite does, since that column is the rowid.
		if col, ok := c.e.(*colExpr); ok && c.bare && col.rowid && r.table.RowidCol >= 0 {
			c.name = r.table.Columns[r.table.RowidCol].Name
		}
		cols = append(cols, c)
	}
	r.resolve(s.where, s)
	for i := range s.order {
		t := &s.order[i]
		if t.pos != 0 {
			if t.pos < 1 || t.pos > len(cols) {
				fail("SQL: ORDER BY %d is outside the %d result column(s)", t.pos, len(cols))
			}
			continue
		}
		// A bare name matching a result column's name sorts by it.
		if c, ok := t.e.(*colExpr); ok && c.table == "" {
			for ci, rc := range cols {
				if strings.EqualFold(rc.name, c.name) && !r.isTableColumn(c.name) {
					t.pos = ci + 1
				}
			}
			if t.pos != 0 {
				continue
			}
		}
		r.resolve(t.e, s)
	}
	for _, c := range cols {
		r.findAggs(c.e)
	}
	if s.where != nil && r.hasAgg(s.where) {
		fail("SQL: aggregate functions like count() can't be used in WHERE")
	}

	names := make([]string, len(cols))
	for i, c := range cols {
		names[i] = c.name
	}
	type outRow struct {
		vals []Value
		keys []Value
	}
	var out []outRow
	emit := func() {
		vals := make([]Value, len(cols))
		for i, c := range cols {
			vals[i] = r.eval(c.e)
		}
		keys := make([]Value, len(s.order))
		for i, t := range s.order {
			if t.pos != 0 {
				keys[i] = vals[t.pos-1]
			} else {
				keys[i] = r.eval(t.e)
			}
		}
		out = append(out, outRow{vals, keys})
	}
	visit := func(rowid int64, vals []Value) {
		r.row, r.rowid = vals, rowid
		if s.where != nil && !truthy(r.eval(s.where)) {
			return
		}
		if len(r.aggs) > 0 {
			r.accumulate()
			return
		}
		emit()
	}
	if r.table == nil {
		visit(0, nil)
	} else {
		err := db.scanTable(r.table.Root, func(rowid int64, payload []byte) error {
			vals, err := db.decodeRecord(payload)
			if err != nil {
				return err
			}
			visit(rowid, vals)
			return nil
		})
		if err != nil {
			return nil, nil, err
		}
	}
	if len(r.aggs) > 0 {
		r.final = true
		emit() // one row, even over no rows: count(*) is 0
	}
	if s.distinct {
		seen := map[string]bool{}
		kept := out[:0]
		for _, o := range out {
			k := rowKey(o.vals)
			if !seen[k] {
				seen[k] = true
				kept = append(kept, o)
			}
		}
		out = kept
	}
	if len(s.order) > 0 {
		sort.SliceStable(out, func(i, j int) bool {
			for k, t := range s.order {
				c := compare(out[i].keys[k], out[j].keys[k])
				if c != 0 {
					return (c < 0) != t.desc
				}
			}
			return false
		})
	}
	if s.offset != nil {
		n := r.intValue(s.offset, "OFFSET")
		if n > int64(len(out)) {
			n = int64(len(out))
		}
		if n > 0 {
			out = out[n:]
		}
	}
	if s.limit != nil {
		if n := r.intValue(s.limit, "LIMIT"); n >= 0 && n < int64(len(out)) {
			out = out[:n]
		}
	}
	rows := make([][]Value, len(out))
	for i, o := range out {
		rows[i] = o.vals
	}
	return names, rows, nil
}

func (r *runner) isTableColumn(name string) bool {
	if r.table == nil {
		return false
	}
	for _, c := range r.table.Columns {
		if strings.EqualFold(c.Name, name) {
			return true
		}
	}
	return false
}

// resolve binds column references to the FROM table's columns.
func (r *runner) resolve(e expr, s *selectStmt) {
	walk(e, func(e expr) {
		c, ok := e.(*colExpr)
		if !ok {
			return
		}
		if r.table == nil {
			fail("SQL: no such column: %s (there's no FROM table)", c.name)
		}
		// With an alias (FROM books b), only the alias names the table.
		if c.table != "" && !(s.fromAlias == "" && strings.EqualFold(c.table, r.table.Name)) && !(s.fromAlias != "" && strings.EqualFold(c.table, s.fromAlias)) {
			fail("SQL: no such column: %s.%s", c.table, c.name)
		}
		for i, tc := range r.table.Columns {
			if strings.EqualFold(tc.Name, c.name) {
				c.idx, c.aff, c.rowid = i, tc.Affinity, i == r.table.RowidCol
				return
			}
		}
		switch strings.ToLower(c.name) {
		case "rowid", "oid", "_rowid_":
			c.rowid, c.aff = true, affInteger
			return
		}
		fail("SQL: no such column: %s", c.name)
	})
}

// walk calls fn on e and every expression inside it.
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
	case *caseExpr:
		walk(x.base, fn)
		for _, w := range x.whens {
			walk(w[0], fn)
			walk(w[1], fn)
		}
		walk(x.els, fn)
	case *castExpr:
		walk(x.x, fn)
	}
}

var aggNames = map[string]bool{"count": true, "sum": true, "total": true, "avg": true, "min": true, "max": true, "group_concat": true}

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

func (r *runner) findAggs(e expr) {
	walk(e, func(e expr) {
		if c, ok := e.(*callExpr); ok && isAggCall(c) {
			if c.name == "count" && !c.star && len(c.args) != 1 {
				fail("SQL: count() takes * or one value")
			}
			if c.name != "count" && c.name != "group_concat" && len(c.args) != 1 {
				fail("SQL: %s() takes one value", c.name)
			}
			c.agg = len(r.aggs)
			r.aggs = append(r.aggs, &aggState{call: c, dist: map[string]bool{}})
		}
	})
}

func (r *runner) hasAgg(e expr) bool {
	found := false
	walk(e, func(e expr) {
		if c, ok := e.(*callExpr); ok && isAggCall(c) {
			found = true
		}
	})
	return found
}

func (r *runner) accumulate() {
	for _, a := range r.aggs {
		c := a.call
		if c.star {
			a.count++
			continue
		}
		v := r.eval(c.args[0])
		if v == nil {
			continue
		}
		if c.distinct {
			k := rowKey([]Value{v})
			if a.dist[k] {
				continue
			}
			a.dist[k] = true
		}
		a.count++
		switch c.name {
		case "sum", "total", "avg":
			n := numericValue(v)
			if a.sum == nil {
				a.sum = n
			} else {
				a.sum = arith("+", a.sum, n)
			}
		case "min", "max":
			if !a.seen || (c.name == "min" && compare(v, a.best) < 0) || (c.name == "max" && compare(v, a.best) > 0) {
				a.best = v
			}
		case "group_concat":
			a.parts = append(a.parts, textValue(v))
		}
		a.seen = true
	}
}

func (r *runner) aggResult(a *aggState) Value {
	switch a.call.name {
	case "count":
		return a.count
	case "sum":
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
	}
	// group_concat
	if len(a.parts) == 0 {
		return nil
	}
	sep := ","
	if len(a.call.args) > 1 {
		sep = textValue(r.eval(a.call.args[1]))
	}
	return strings.Join(a.parts, sep)
}

func (r *runner) intValue(e expr, what string) int64 {
	v := r.eval(e)
	n, ok := applyAffinity(v, affInteger).(int64)
	if !ok {
		fail("SQL: %s needs a whole number", what)
	}
	return n
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

// ---- evaluating expressions ----------------------------------------------

func (r *runner) eval(e expr) Value {
	switch x := e.(type) {
	case *litExpr:
		return x.v
	case *paramExpr:
		return r.params[x.idx]
	case *colExpr:
		if x.rowid {
			return r.rowid
		}
		if r.row == nil || x.idx >= len(r.row) {
			return nil // a column added after this row was written
		}
		v := r.row[x.idx]
		// SQLite stores a whole REAL like 5.0 as the integer 5 to save
		// space; it's still a REAL.
		if i, ok := v.(int64); ok && x.aff == affReal {
			return float64(i)
		}
		return v
	case *unaryExpr:
		v := r.eval(x.x)
		switch x.op {
		case "NOT":
			if v == nil {
				return nil
			}
			return boolValue(!truthy(v))
		case "-":
			if v == nil {
				return nil
			}
			return arith("-", int64(0), numericValue(v))
		case "+":
			return v
		}
		if v == nil { // ~
			return nil
		}
		return ^integerValue(v)
	case *binExpr:
		return r.evalBinary(x)
	case *isExpr:
		l, rv := r.eval(x.l), r.eval(x.r)
		l, rv = compareAffinity(x.l, x.r, l, rv)
		same := (l == nil && rv == nil) || (l != nil && rv != nil && compare(l, rv) == 0)
		return boolValue(same != x.not)
	case *isNullExpr:
		return boolValue((r.eval(x.x) == nil) != x.not)
	case *inExpr:
		v := r.eval(x.x)
		if v == nil {
			return nil
		}
		sawNull := false
		for _, item := range x.list {
			iv := r.eval(item)
			if iv == nil {
				sawNull = true
				continue
			}
			a, b := compareAffinity(x.x, item, v, iv)
			if compare(a, b) == 0 {
				return boolValue(!x.not)
			}
		}
		if sawNull {
			return nil
		}
		return boolValue(x.not)
	case *betweenExpr:
		v, lo, hi := r.eval(x.x), r.eval(x.lo), r.eval(x.hi)
		if v == nil || lo == nil || hi == nil {
			return nil
		}
		a, l := compareAffinity(x.x, x.lo, v, lo)
		b, h := compareAffinity(x.x, x.hi, v, hi)
		in := compare(a, l) >= 0 && compare(b, h) <= 0
		return boolValue(in != x.not)
	case *likeExpr:
		v, pat := r.eval(x.x), r.eval(x.pat)
		if v == nil || pat == nil {
			return nil
		}
		esc := rune(0)
		if x.esc != nil {
			e := []rune(textValue(r.eval(x.esc)))
			if len(e) != 1 {
				fail("SQL: ESCAPE needs a single character")
			}
			esc = e[0]
		}
		return boolValue(like([]rune(textValue(pat)), []rune(textValue(v)), esc) != x.not)
	case *callExpr:
		if x.agg >= 0 {
			if !r.final {
				fail("SQL: aggregate %s() used outside a result column", x.name)
			}
			return r.aggResult(r.aggs[x.agg])
		}
		return r.callFunc(x)
	case *caseExpr:
		var base Value
		if x.base != nil {
			base = r.eval(x.base)
		}
		for _, w := range x.whens {
			if x.base != nil {
				wv := r.eval(w[0])
				if base != nil && wv != nil && compare(base, wv) == 0 {
					return r.eval(w[1])
				}
			} else if truthy(r.eval(w[0])) {
				return r.eval(w[1])
			}
		}
		if x.els != nil {
			return r.eval(x.els)
		}
		return nil
	case *castExpr:
		return castValue(r.eval(x.x), x.typ)
	}
	fail("SQL: can't evaluate this expression")
	return nil
}

func (r *runner) evalBinary(x *binExpr) Value {
	switch x.op {
	case "AND":
		l := r.eval(x.l)
		if l != nil && !truthy(l) {
			return int64(0)
		}
		rv := r.eval(x.r)
		if rv != nil && !truthy(rv) {
			return int64(0)
		}
		if l == nil || rv == nil {
			return nil
		}
		return int64(1)
	case "OR":
		l := r.eval(x.l)
		if l != nil && truthy(l) {
			return int64(1)
		}
		rv := r.eval(x.r)
		if rv != nil && truthy(rv) {
			return int64(1)
		}
		if l == nil || rv == nil {
			return nil
		}
		return int64(0)
	}
	l, rv := r.eval(x.l), r.eval(x.r)
	if l == nil || rv == nil {
		return nil
	}
	switch x.op {
	case "=", "!=", "<", "<=", ">", ">=":
		l, rv = compareAffinity(x.l, x.r, l, rv)
		c := compare(l, rv)
		switch x.op {
		case "=":
			return boolValue(c == 0)
		case "!=":
			return boolValue(c != 0)
		case "<":
			return boolValue(c < 0)
		case "<=":
			return boolValue(c <= 0)
		case ">":
			return boolValue(c > 0)
		}
		return boolValue(c >= 0)
	case "||":
		return textValue(l) + textValue(rv)
	case "&", "|", "<<", ">>":
		a, b := integerValue(l), integerValue(rv)
		switch x.op {
		case "&":
			return a & b
		case "|":
			return a | b
		case "<<":
			if b >= 64 || b <= -64 {
				return int64(0)
			}
			if b < 0 {
				return a >> uint(-b)
			}
			return a << uint(b)
		}
		if b >= 64 || b <= -64 {
			if a < 0 {
				return int64(-1)
			}
			return int64(0)
		}
		if b < 0 {
			return a << uint(-b)
		}
		return a >> uint(b)
	}
	return arith(x.op, numericValue(l), numericValue(rv))
}

// arith does + - * / % on two numbers (int64 or float64), the SQLite way:
// integers stay integers unless they overflow, x/0 and x%0 are NULL.
func arith(op string, a, b Value) Value {
	ai, aInt := a.(int64)
	bi, bInt := b.(int64)
	if aInt && bInt {
		switch op {
		case "+":
			if s := ai + bi; (s > ai) == (bi > 0) {
				return s
			}
		case "-":
			if s := ai - bi; (s < ai) == (bi > 0) {
				return s
			}
		case "*":
			if ai == 0 || bi == 0 {
				return int64(0)
			}
			if s := ai * bi; s/bi == ai && !(ai == -1 && bi == math.MinInt64) && !(bi == -1 && ai == math.MinInt64) {
				return s
			}
		case "/":
			if bi == 0 {
				return nil
			}
			if !(ai == math.MinInt64 && bi == -1) {
				return ai / bi
			}
		case "%":
			if bi == 0 {
				return nil
			}
			if bi == -1 {
				return int64(0)
			}
			return ai % bi
		}
	}
	af, bf := toFloat(a), toFloat(b)
	switch op {
	case "+":
		return af + bf
	case "-":
		return af - bf
	case "*":
		return af * bf
	case "/":
		if bf == 0 {
			return nil
		}
		return af / bf
	}
	// % on floats works on their integer parts, like SQLite.
	ia, ib := int64(af), int64(bf)
	if ib == 0 {
		return nil
	}
	return float64(ia % ib)
}

// ---- values ---------------------------------------------------------------

func boolValue(b bool) Value {
	if b {
		return int64(1)
	}
	return int64(0)
}

// truthy is SQLite's test for WHERE and NOT: a nonzero number (text is
// read as a number first). NULL is false here.
func truthy(v Value) bool {
	switch x := numericValue(v).(type) {
	case int64:
		return x != 0
	case float64:
		return x != 0
	}
	return false
}

func toFloat(v Value) float64 {
	switch x := v.(type) {
	case int64:
		return float64(x)
	case float64:
		return x
	}
	return toFloat(numericValue(v))
}

func integerValue(v Value) int64 {
	switch x := numericValue(v).(type) {
	case int64:
		return x
	case float64:
		return int64(x)
	}
	return 0
}

// numericValue reads v as a number: numbers as they are, text by its
// longest numeric prefix ("12abc" is 12, "abc" is 0), NULL stays NULL.
func numericValue(v Value) Value {
	switch x := v.(type) {
	case nil, int64, float64:
		return v
	case string:
		return numericPrefix(x)
	case []byte:
		return numericPrefix(string(x))
	}
	return int64(0)
}

func numericPrefix(s string) Value {
	s = strings.TrimLeft(s, " \t\n\r")
	end := 0
	if end < len(s) && (s[end] == '+' || s[end] == '-') {
		end++
	}
	digits := 0
	for end < len(s) && isDigit(s[end]) {
		end++
		digits++
	}
	isFloat := false
	if end < len(s) && s[end] == '.' {
		j := end + 1
		for j < len(s) && isDigit(s[j]) {
			j++
			digits++
		}
		if digits > 0 {
			isFloat, end = true, j
		}
	}
	if digits == 0 {
		return int64(0)
	}
	if end < len(s) && (s[end] == 'e' || s[end] == 'E') {
		j := end + 1
		if j < len(s) && (s[j] == '+' || s[j] == '-') {
			j++
		}
		if j < len(s) && isDigit(s[j]) {
			for j < len(s) && isDigit(s[j]) {
				j++
			}
			isFloat, end = true, j
		}
	}
	if !isFloat {
		if n, err := strconv.ParseInt(s[:end], 10, 64); err == nil {
			return n
		}
	}
	f, _ := strconv.ParseFloat(s[:end], 64)
	return f
}

// textValue writes v as SQLite text: 3.0 is "3.0", NULL is "".
func textValue(v Value) string {
	switch x := v.(type) {
	case nil:
		return ""
	case int64:
		return strconv.FormatInt(x, 10)
	case float64:
		return formatReal(x)
	case string:
		return x
	case []byte:
		return string(x)
	}
	return ""
}

func formatReal(f float64) string {
	if math.IsInf(f, 1) {
		return "Inf"
	}
	if math.IsInf(f, -1) {
		return "-Inf"
	}
	s := strconv.FormatFloat(f, 'g', 15, 64)
	if !strings.ContainsAny(s, ".eEN") {
		s += ".0"
	}
	return s
}

// applyAffinity converts v the way storing it in a column of that
// affinity would: text that is a well-formed number becomes a number for
// numeric affinities; numbers become text for TEXT.
func applyAffinity(v Value, aff affinity) Value {
	switch aff {
	case affText:
		switch v.(type) {
		case int64, float64:
			return textValue(v)
		}
	case affInteger, affNumeric, affReal:
		s, ok := v.(string)
		if !ok {
			if f, isF := v.(float64); isF && aff != affReal && f == math.Trunc(f) && math.Abs(f) < 9e18 {
				return int64(f)
			}
			if i, isI := v.(int64); isI && aff == affReal {
				return float64(i)
			}
			return v
		}
		t := strings.TrimSpace(s)
		if t == "" {
			return v
		}
		n := numericPrefix(t)
		// Only a complete number converts.
		if textValue(n) != t && !looksNumeric(t) {
			return v
		}
		if aff == affReal {
			return toFloat(n)
		}
		if f, isF := n.(float64); isF && f == math.Trunc(f) && math.Abs(f) < 9e18 {
			return int64(f)
		}
		return n
	}
	return v
}

// looksNumeric reports whether all of s is a number SQLite would accept.
func looksNumeric(s string) bool {
	if s == "" {
		return false
	}
	_, err := strconv.ParseFloat(s, 64)
	return err == nil && !strings.ContainsAny(s, "xXnNiI_")
}

// exprAffinity is the affinity an expression brings to a comparison: a
// column's, a CAST's, or none.
func exprAffinity(e expr) affinity {
	switch x := e.(type) {
	case *colExpr:
		return x.aff
	case *castExpr:
		return affinityOf(x.typ)
	}
	return affBlob
}

// compareAffinity applies SQLite's comparison rules: numeric affinity on
// one side converts text on the other; text affinity converts a side
// that has no affinity.
func compareAffinity(le, re expr, l, r Value) (Value, Value) {
	la, ra := exprAffinity(le), exprAffinity(re)
	numeric := func(a affinity) bool { return a == affInteger || a == affReal || a == affNumeric }
	switch {
	case numeric(la) && !numeric(ra):
		r = applyAffinity(r, affNumeric)
	case numeric(ra) && !numeric(la):
		l = applyAffinity(l, affNumeric)
	case la == affText && ra == affBlob && !isColumn(re):
		r = applyAffinity(r, affText)
	case ra == affText && la == affBlob && !isColumn(le):
		l = applyAffinity(l, affText)
	}
	return l, r
}

func isColumn(e expr) bool {
	_, ok := e.(*colExpr)
	return ok
}

// compare orders two non-NULL... or NULL values the SQLite way: NULL <
// numbers < text < blobs; numbers by value, text and blobs by bytes.
func compare(a, b Value) int {
	ca, cb := class(a), class(b)
	if ca != cb {
		if ca < cb {
			return -1
		}
		return 1
	}
	switch ca {
	case 0:
		return 0
	case 1:
		ai, aInt := a.(int64)
		bi, bInt := b.(int64)
		if aInt && bInt {
			switch {
			case ai < bi:
				return -1
			case ai > bi:
				return 1
			}
			return 0
		}
		af, bf := toFloat(a), toFloat(b)
		switch {
		case af < bf:
			return -1
		case af > bf:
			return 1
		}
		return 0
	case 2:
		return strings.Compare(a.(string), b.(string))
	}
	return bytes.Compare(a.([]byte), b.([]byte))
}

func class(v Value) int {
	switch v.(type) {
	case nil:
		return 0
	case int64, float64:
		return 1
	case string:
		return 2
	}
	return 3
}

func castValue(v Value, typ string) Value {
	if v == nil {
		return nil
	}
	switch affinityOf(typ) {
	case affText:
		return textValue(v)
	case affInteger:
		return integerValue(v)
	case affReal:
		return toFloat(v)
	case affNumeric:
		n := numericValue(v)
		if f, ok := n.(float64); ok && f == math.Trunc(f) && math.Abs(f) < 9e18 {
			return int64(f)
		}
		return n
	}
	// BLOB
	switch x := v.(type) {
	case []byte:
		return x
	}
	return []byte(textValue(v))
}

// like matches SQLite's LIKE: % any run, _ one character, case-insensitive
// for ASCII letters.
func like(pat, s []rune, esc rune) bool {
	for len(pat) > 0 {
		c := pat[0]
		switch {
		case esc != 0 && c == esc && len(pat) > 1:
			if len(s) == 0 || lowerASCII(s[0]) != lowerASCII(pat[1]) {
				return false
			}
			pat, s = pat[2:], s[1:]
		case c == '%':
			for len(pat) > 0 && pat[0] == '%' {
				pat = pat[1:]
			}
			if len(pat) == 0 {
				return true
			}
			for i := 0; i <= len(s); i++ {
				if like(pat, s[i:], esc) {
					return true
				}
			}
			return false
		case c == '_':
			if len(s) == 0 {
				return false
			}
			pat, s = pat[1:], s[1:]
		default:
			if len(s) == 0 || lowerASCII(s[0]) != lowerASCII(c) {
				return false
			}
			pat, s = pat[1:], s[1:]
		}
	}
	return len(s) == 0
}

func lowerASCII(r rune) rune {
	if r >= 'A' && r <= 'Z' {
		return r + 32
	}
	return r
}
