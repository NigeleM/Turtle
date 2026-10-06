package sqlite

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// ---- syntax tree --------------------------------------------------------

type expr interface{}

type (
	litExpr   struct{ v Value }
	paramExpr struct{ idx int }
	colExpr   struct {
		table, name string
		src         int  // which source of its scope (FROM item) it reads
		idx         int  // column index in that source
		up          int  // how many scopes out: 0 is its own query, 1 the query around a subquery, ...
		rowid       bool // the rowid itself (rowid, oid, _rowid_, or an INTEGER PRIMARY KEY)
		aff         affinity
		coll        string // the column's declared collation, or ""
		alias       expr   // a result column this name refers to (SELECT a + 1 AS b ... ORDER BY b)
		col         *Column
		bound       bool
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
		sub  *selectStmt
		plan *queryPlan
		not  bool
	}
	betweenExpr struct {
		x, lo, hi expr
		not       bool
	}
	likeExpr struct {
		x, pat, esc expr
		not         bool
		glob        bool
	}
	callExpr struct {
		name     string // lowercase
		args     []expr
		star     bool // count(*)
		distinct bool
		filter   expr // aggregate FILTER (WHERE ...)
		agg      int  // index into the aggregate states, or -1
		over     *windowSpec
		win      int // 1 + index into the core's windows; 0: not a window function
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
	collateExpr struct {
		x    expr
		coll string // BINARY, NOCASE or RTRIM
	}
	subqueryExpr struct { // (SELECT ...) as a value: its first row's first column
		sel  *selectStmt
		plan *queryPlan
	}
	existsExpr struct {
		sel  *selectStmt
		plan *queryPlan
	}
)

// windowSpec is OVER (...): which rows a window function sees.
type windowSpec struct {
	name      string // OVER name, or the base window in OVER (name ...)
	partition []expr
	order     []orderTerm
	frame     *frameSpec
}

type frameSpec struct {
	unit       string // ROWS, RANGE or GROUPS
	start, end frameBound
	exclude    string // "", CURRENT ROW, GROUP, TIES
}

type frameBound struct {
	kind string // UNBOUNDED PRECEDING, PRECEDING, CURRENT ROW, FOLLOWING, UNBOUNDED FOLLOWING
	n    expr
}

type resultCol struct {
	e     expr
	star  bool
	table string // t.*: star for one table
	name  string
	bare  bool // a plain column, named after the column
}

type orderTerm struct {
	e     expr
	desc  bool
	nulls int // 0 default (first ascending, last descending), 1 first, 2 last
	pos   int // 1-based result column, or 0
}

type fromItem struct {
	name, alias string
	schema      string // main, or an attached database's name
	args        []expr // a table-valued function: FROM json_each(x)
	isFunc      bool
	sub         *selectStmt // FROM (SELECT ...)
	join        string      // "" for the first item, else INNER, LEFT, RIGHT, FULL or CROSS
	natural     bool
	on          expr
	using       []string
}

type cteDef struct {
	name string
	cols []string
	sel  *selectStmt
}

type compoundPart struct {
	op   string // UNION, UNION ALL, INTERSECT, EXCEPT
	core *selectStmt
}

// selectStmt is a SELECT, or a VALUES list, with what can follow it.
type selectStmt struct {
	with      []*cteDef
	recursive bool
	distinct  bool
	cols      []resultCol
	from      []fromItem
	where     expr
	groupBy   []expr
	having    expr
	values    [][]expr // VALUES (...), (...) instead of SELECT
	windows   map[string]*windowSpec
	compound  []compoundPart
	order     []orderTerm
	limit     expr
	offset    expr
}

// ---- parser -------------------------------------------------------------

type sqlParser struct {
	toks   []token
	pos    int
	src    string
	params int         // ? placeholders seen
	onCol  func(token) // called with each name that may name a column (ALTER TABLE uses it)
	onTbl  func(token) // called with each name that names a table
}

func (p *sqlParser) peek() token { return p.toks[p.pos] }
func (p *sqlParser) peekAt(n int) token {
	if p.pos+n < len(p.toks) {
		return p.toks[p.pos+n]
	}
	return p.toks[len(p.toks)-1]
}
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
		p.near("expected %s", w)
	}
}
func (p *sqlParser) expectOp(op string) {
	if !p.acceptOp(op) {
		p.near("expected %q", op)
	}
}

// near fails with msg and where the parser is.
func (p *sqlParser) near(format string, args ...any) {
	msg := "SQL: " + strings.TrimPrefix(fmt.Sprintf(format, args...), "SQL: ")
	if p.peek().kind == tEOF {
		fail("%s, but the statement ends", msg)
	}
	fail("%s near %q", msg, p.peek().text)
}

// clauseWords end an expression or a result column's bare alias.
var clauseWords = []string{"FROM", "WHERE", "ORDER", "LIMIT", "OFFSET", "GROUP", "HAVING", "UNION", "EXCEPT", "INTERSECT",
	"AS", "ASC", "DESC", "AND", "OR", "NOT", "ON", "JOIN", "LEFT", "RIGHT", "FULL", "INNER", "CROSS", "NATURAL", "OUTER",
	"USING", "WINDOW", "SET", "VALUES", "RETURNING", "DO", "NULLS", "COLLATE"}

// atStatementEnd reports whether the statement is over: a ; or the end.
func (p *sqlParser) atStatementEnd() bool {
	return p.peek().kind == tEOF || p.isOp(";")
}

// parseSelect parses [WITH ...] a SELECT or VALUES, compound parts, then
// ORDER BY and LIMIT for the whole.
func (p *sqlParser) parseSelect() *selectStmt {
	var with []*cteDef
	recursive := false
	if p.acceptWord("WITH") {
		with, recursive = p.parseWithList()
	}
	s := p.parseCore()
	s.with, s.recursive = with, recursive
	for p.isWord("UNION", "INTERSECT", "EXCEPT") {
		op := strings.ToUpper(p.next().text)
		if op == "UNION" && p.acceptWord("ALL") {
			op = "UNION ALL"
		}
		s.compound = append(s.compound, compoundPart{op: op, core: p.parseCore()})
	}
	if p.acceptWord("ORDER") {
		p.expectWord("BY")
		s.order = p.parseOrderTerms()
	}
	if p.acceptWord("LIMIT") {
		s.limit = p.parseExpr()
		if p.acceptWord("OFFSET") {
			s.offset = p.parseExpr()
		} else if p.acceptOp(",") { // LIMIT offset, count
			s.offset, s.limit = s.limit, p.parseExpr()
		}
	}
	return s
}

// parseWithList parses the tables of a WITH clause (after WITH).
func (p *sqlParser) parseWithList() ([]*cteDef, bool) {
	recursive := p.acceptWord("RECURSIVE")
	var with []*cteDef
	for {
		c := &cteDef{name: p.expectName()}
		if p.acceptOp("(") {
			for {
				c.cols = append(c.cols, p.expectName())
				if !p.acceptOp(",") {
					break
				}
			}
			p.expectOp(")")
		}
		p.expectWord("AS")
		if p.acceptWord("NOT") {
			p.expectWord("MATERIALIZED")
		} else {
			p.acceptWord("MATERIALIZED")
		}
		p.expectOp("(")
		c.sel = p.parseSelect()
		p.expectOp(")")
		with = append(with, c)
		if !p.acceptOp(",") {
			break
		}
	}
	return with, recursive
}

// parseCore parses one SELECT ... or VALUES ... without ORDER BY/LIMIT.
func (p *sqlParser) parseCore() *selectStmt {
	s := &selectStmt{}
	if p.acceptWord("VALUES") {
		for {
			p.expectOp("(")
			var row []expr
			for {
				row = append(row, p.parseExpr())
				if !p.acceptOp(",") {
					break
				}
			}
			p.expectOp(")")
			if len(s.values) > 0 && len(row) != len(s.values[0]) {
				fail("SQL: all VALUES must have the same number of terms")
			}
			s.values = append(s.values, row)
			if !p.acceptOp(",") {
				break
			}
		}
		return s
	}
	if !p.isWord("SELECT") {
		p.near("expected SELECT")
	}
	p.next()
	if p.acceptWord("DISTINCT") {
		s.distinct = true
	} else {
		p.acceptWord("ALL")
	}
	for {
		switch {
		case p.acceptOp("*"):
			s.cols = append(s.cols, resultCol{star: true})
		case p.peek().kind == tIdent && p.peekAt(1).kind == tOp && p.peekAt(1).text == "." && p.peekAt(2).kind == tOp && p.peekAt(2).text == "*":
			name := p.next().text
			p.next()
			p.next()
			s.cols = append(s.cols, resultCol{star: true, table: name})
		default:
			start := p.peek().pos
			e := p.parseExpr()
			end := p.toks[p.pos-1].end
			c := resultCol{e: e, name: strings.TrimSpace(p.src[start:end])}
			if col, ok := e.(*colExpr); ok {
				c.name = col.name
				c.bare = true
			}
			if p.acceptWord("AS") {
				c.name, c.bare = p.expectName(), false
			} else if (p.peek().kind == tIdent && (p.peek().quoted || !p.isWord(clauseWords...))) || p.peek().kind == tString {
				c.name, c.bare = p.next().text, false
			}
			s.cols = append(s.cols, c)
		}
		if !p.acceptOp(",") {
			break
		}
	}
	if p.acceptWord("FROM") {
		s.from = p.parseFromList()
	}
	if p.acceptWord("WHERE") {
		s.where = p.parseExpr()
	}
	if p.acceptWord("GROUP") {
		p.expectWord("BY")
		for {
			s.groupBy = append(s.groupBy, p.parseExpr())
			if !p.acceptOp(",") {
				break
			}
		}
	}
	if p.acceptWord("HAVING") {
		s.having = p.parseExpr()
	}
	if p.acceptWord("WINDOW") {
		s.windows = map[string]*windowSpec{}
		for {
			name := p.expectName()
			p.expectWord("AS")
			p.expectOp("(")
			s.windows[strings.ToLower(name)] = p.parseWindowSpec()
			p.expectOp(")")
			if !p.acceptOp(",") {
				break
			}
		}
	}
	return s
}

// parseFromList reads what follows FROM: tables, subqueries and joins.
func (p *sqlParser) parseFromList() []fromItem {
	var from []fromItem
	from = append(from, p.parseFromItem(""))
	for {
		if p.acceptOp(",") {
			from = append(from, p.parseFromItem("INNER"))
			continue
		}
		natural := p.acceptWord("NATURAL")
		join := ""
		switch {
		case p.acceptWord("LEFT"):
			p.acceptWord("OUTER")
			join = "LEFT"
		case p.acceptWord("RIGHT"):
			p.acceptWord("OUTER")
			join = "RIGHT"
		case p.acceptWord("FULL"):
			p.acceptWord("OUTER")
			join = "FULL"
		case p.acceptWord("INNER"):
			join = "INNER"
		case p.acceptWord("CROSS"):
			join = "CROSS"
		}
		if join == "" && !natural && !p.isWord("JOIN") {
			break
		}
		p.expectWord("JOIN")
		if join == "" {
			join = "INNER"
		}
		f := p.parseFromItem(join)
		f.natural = natural
		if p.acceptWord("ON") {
			if natural {
				fail("SQL: a NATURAL join can't have an ON clause")
			}
			f.on = p.parseExpr()
		} else if p.acceptWord("USING") {
			if natural {
				fail("SQL: a NATURAL join can't have a USING clause")
			}
			p.expectOp("(")
			for {
				f.using = append(f.using, p.expectName())
				if !p.acceptOp(",") {
					break
				}
			}
			p.expectOp(")")
		}
		from = append(from, f)
	}
	return from
}

func (p *sqlParser) parseFromItem(join string) fromItem {
	f := fromItem{join: join}
	if p.acceptOp("(") {
		if !p.isWord("SELECT", "WITH", "VALUES") {
			fail("SQL: parenthesized joins aren't supported; write the joins without ( )")
		}
		f.sub = p.parseSelect()
		p.expectOp(")")
	} else {
		nt := p.peek()
		f.name = p.expectName()
		if p.acceptOp(".") { // schema.table: main.books
			nt = p.peek()
			f.schema = f.name
			f.name = p.expectName()
		}
		if p.onTbl != nil {
			p.onTbl(nt)
		}
		if p.acceptOp("(") {
			f.isFunc = true
			if !p.isOp(")") {
				for {
					f.args = append(f.args, p.parseExpr())
					if !p.acceptOp(",") {
						break
					}
				}
			}
			p.expectOp(")")
		}
	}
	if p.acceptWord("AS") {
		f.alias = p.expectName()
	} else if p.peek().kind == tIdent && (p.peek().quoted || !p.isWord(clauseWords...)) {
		f.alias = p.next().text
	}
	if p.acceptWord("INDEXED") {
		p.expectWord("BY")
		p.expectName()
	} else if p.isWord("NOT") && isWord(p.peekAt(1), "INDEXED") {
		p.next()
		p.next()
	}
	return f
}

func (p *sqlParser) expectName() string {
	t := p.peek()
	if t.kind != tIdent && t.kind != tString {
		p.near("expected a name")
	}
	p.next()
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
				if p.acceptWord("DISTINCT") {
					p.expectWord("FROM")
					not = !not
				}
				l = &isExpr{l: l, r: p.parseComparison(), not: not}
			}
		case p.isWord("ISNULL"):
			p.next()
			l = &isNullExpr{x: l}
		case p.isWord("NOTNULL"):
			p.next()
			l = &isNullExpr{x: l, not: true}
		case p.isWord("NOT") && isWord(p.peekAt(1), "IN", "LIKE", "GLOB", "BETWEEN", "NULL"):
			p.next()
			l = p.parseNegatable(l, true)
		case p.isWord("IN", "LIKE", "GLOB", "BETWEEN"):
			l = p.parseNegatable(l, false)
		case p.isWord("REGEXP", "MATCH"):
			fail("SQL: %s isn't supported", strings.ToUpper(p.peek().text))
		default:
			return l
		}
	}
}

// parseNegatable parses [NOT] IN / LIKE / GLOB / BETWEEN / NULL after l.
func (p *sqlParser) parseNegatable(l expr, not bool) expr {
	switch {
	case p.acceptWord("NULL"):
		return &isNullExpr{x: l, not: true}
	case p.acceptWord("IN"):
		in := &inExpr{x: l, not: not}
		if !p.acceptOp("(") {
			// IN tablename: the table's only column.
			if p.onTbl != nil {
				p.onTbl(p.peek())
			}
			in.sub = &selectStmt{cols: []resultCol{{star: true}}, from: []fromItem{{name: p.expectName()}}}
			return in
		}
		if p.isWord("SELECT", "WITH", "VALUES") {
			in.sub = p.parseSelect()
		} else if !p.isOp(")") {
			for {
				in.list = append(in.list, p.parseExpr())
				if !p.acceptOp(",") {
					break
				}
			}
		}
		p.expectOp(")")
		return in
	case p.isWord("LIKE", "GLOB"):
		glob := strings.EqualFold(p.next().text, "GLOB")
		lk := &likeExpr{x: l, pat: p.parseComparison(), not: not, glob: glob}
		if !glob && p.acceptWord("ESCAPE") {
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
	for p.isOp("||") || p.isOp("->") || p.isOp("->>") {
		op := p.next().text
		l = &binExpr{op: op, l: l, r: p.parseUnary()}
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
	x := p.parsePrimary()
	for p.acceptWord("COLLATE") {
		x = &collateExpr{x: x, coll: collationName(p.expectName())}
	}
	return x
}

// collationName checks a collation name: BINARY, NOCASE or RTRIM.
func collationName(name string) string {
	n := strings.ToUpper(name)
	switch n {
	case "BINARY", "NOCASE", "RTRIM":
		return n
	}
	fail("SQL: no such collation sequence: %s", name)
	return ""
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
			if p.isWord("SELECT", "WITH", "VALUES") {
				sub := &subqueryExpr{sel: p.parseSelect()}
				p.expectOp(")")
				return sub
			}
			e := p.parseExpr()
			if p.isOp(",") {
				fail("SQL: row values like (a, b) aren't supported")
			}
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
			case "CURRENT_TIMESTAMP":
				p.next()
				return &callExpr{name: "datetime", args: []expr{&litExpr{v: "now"}}, agg: -1}
			case "CURRENT_DATE":
				p.next()
				return &callExpr{name: "date", args: []expr{&litExpr{v: "now"}}, agg: -1}
			case "CURRENT_TIME":
				p.next()
				return &callExpr{name: "time", args: []expr{&litExpr{v: "now"}}, agg: -1}
			case "CASE":
				return p.parseCase()
			case "RAISE":
				p.next()
				return p.parseRaise()
			case "EXISTS":
				p.next()
				p.expectOp("(")
				e := &existsExpr{sel: p.parseSelect()}
				p.expectOp(")")
				return e
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
			case "SELECT":
				fail("SQL: a subquery needs ( ) around it")
			}
		}
		p.next()
		if !t.quoted && p.isOp("(") {
			return p.parseCall(strings.ToLower(t.text))
		}
		if p.acceptOp(".") {
			nt := p.peek()
			name := p.expectName()
			if p.acceptOp(".") { // schema.table.column
				nt = p.peek()
				col := &colExpr{table: name, name: p.expectName(), idx: -1}
				if p.onCol != nil {
					p.onCol(nt)
				}
				return col
			}
			if p.onCol != nil {
				p.onCol(nt)
			}
			if p.onTbl != nil {
				p.onTbl(t)
			}
			return &colExpr{table: t.text, name: name, idx: -1}
		}
		if p.onCol != nil {
			p.onCol(t)
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
		if p.isWord("ORDER") {
			fail("SQL: ORDER BY inside %s() isn't supported", name)
		}
	}
	p.expectOp(")")
	if p.acceptWord("FILTER") {
		p.expectOp("(")
		p.expectWord("WHERE")
		c.filter = p.parseExpr()
		p.expectOp(")")
	}
	if p.acceptWord("OVER") {
		if p.acceptOp("(") {
			c.over = p.parseWindowSpec()
			p.expectOp(")")
		} else {
			c.over = &windowSpec{name: p.expectName()}
		}
	}
	return c
}

// parseWindowSpec parses what's inside OVER ( ... ) or WINDOW w AS ( ... ).
func (p *sqlParser) parseWindowSpec() *windowSpec {
	w := &windowSpec{}
	if p.peek().kind == tIdent && !p.isWord("PARTITION", "ORDER", "ROWS", "RANGE", "GROUPS") {
		w.name = p.next().text
	}
	if p.acceptWord("PARTITION") {
		p.expectWord("BY")
		for {
			w.partition = append(w.partition, p.parseExpr())
			if !p.acceptOp(",") {
				break
			}
		}
	}
	if p.acceptWord("ORDER") {
		p.expectWord("BY")
		w.order = p.parseOrderTerms()
	}
	if p.isWord("ROWS", "RANGE", "GROUPS") {
		f := &frameSpec{unit: strings.ToUpper(p.next().text)}
		if p.acceptWord("BETWEEN") {
			f.start = p.parseFrameBound()
			p.expectWord("AND")
			f.end = p.parseFrameBound()
		} else {
			f.start = p.parseFrameBound()
			f.end = frameBound{kind: "CURRENT ROW"}
		}
		if p.acceptWord("EXCLUDE") {
			switch {
			case p.acceptWord("NO"):
				p.expectWord("OTHERS")
			case p.acceptWord("CURRENT"):
				p.expectWord("ROW")
				f.exclude = "CURRENT ROW"
			case p.acceptWord("GROUP"):
				f.exclude = "GROUP"
			default:
				p.expectWord("TIES")
				f.exclude = "TIES"
			}
		}
		if f.start.kind == "UNBOUNDED FOLLOWING" {
			fail("SQL: unsupported frame specification")
		}
		if f.end.kind == "UNBOUNDED PRECEDING" {
			fail("SQL: unsupported frame specification")
		}
		w.frame = f
	}
	return w
}

func (p *sqlParser) parseFrameBound() frameBound {
	switch {
	case p.acceptWord("UNBOUNDED"):
		if p.acceptWord("PRECEDING") {
			return frameBound{kind: "UNBOUNDED PRECEDING"}
		}
		p.expectWord("FOLLOWING")
		return frameBound{kind: "UNBOUNDED FOLLOWING"}
	case p.acceptWord("CURRENT"):
		p.expectWord("ROW")
		return frameBound{kind: "CURRENT ROW"}
	}
	n := p.parseAdd()
	if p.acceptWord("PRECEDING") {
		return frameBound{kind: "PRECEDING", n: n}
	}
	p.expectWord("FOLLOWING")
	return frameBound{kind: "FOLLOWING", n: n}
}

// parseOrderTerms parses the terms after ORDER BY.
func (p *sqlParser) parseOrderTerms() []orderTerm {
	var terms []orderTerm
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
		if p.acceptWord("NULLS") {
			if p.acceptWord("FIRST") {
				t.nulls = 1
			} else {
				p.expectWord("LAST")
				t.nulls = 2
			}
		}
		terms = append(terms, t)
		if !p.acceptOp(",") {
			break
		}
	}
	return terms
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
