// Copyright 2017-2026 Nigele McCoy
// Licensed under the Apache License, Version 2.0;
// see LICENSE and NOTICE.

package sqlite

import (
	"bytes"
	"fmt"
	"math"
	"strconv"
	"strings"
)

// Query runs one SELECT (or WITH ... SELECT, or VALUES) with ?
// parameters and returns the column names and rows.
//
// Supported: SELECT [DISTINCT] columns or expressions [AS name], *, t.*;
// FROM tables, subqueries, views and WITH tables, joined with JOIN, LEFT,
// RIGHT, FULL, CROSS and NATURAL joins, ON or USING; WHERE; GROUP BY and
// HAVING with the aggregates count, sum, total, avg, min, max,
// group_concat, string_agg (DISTINCT and FILTER too); UNION [ALL],
// INTERSECT, EXCEPT; ORDER BY (expressions, result names, positions,
// ASC/DESC, NULLS FIRST/LAST); LIMIT/OFFSET; subqueries in expressions,
// IN and EXISTS. Expressions follow SQLite: comparisons apply column
// affinity and collation, NULL propagates, integer division truncates,
// x/0 is NULL.
func (db *DB) Query(sql string, params []Value) (cols []string, rows [][]Value, err error) {
	defer catch(&err)
	p := newParser(sql)
	if p.acceptWord("PRAGMA") {
		st := p.parsePragma()
		p.endStatement()
		return db.runPragma(st)
	}
	if p.isWord("INSERT", "REPLACE", "UPDATE", "DELETE") {
		// With RETURNING, a change gives back rows.
		st := p.parseStatement()
		p.endStatement()
		p.checkParams(params)
		has := false
		switch s := st.(type) {
		case *insertStmt:
			has = s.returning != nil
		case *updateStmt:
			has = s.returning != nil
		case *deleteStmt:
			has = s.returning != nil
		}
		if !has {
			fail("SQL: %s changes the database; run it with sql_run, or add RETURNING to get rows back", strings.ToUpper(p.toks[0].text))
		}
		res, err := db.execOne(st, params)
		if err != nil {
			return nil, nil, err
		}
		return res.Cols, res.Rows, nil
	}
	if !p.isWord("SELECT", "WITH", "VALUES") {
		if p.peek().kind == tEOF {
			fail("SQL: the statement is empty")
		}
		fail("SQL: %s changes the database; run it with sql_run", strings.ToUpper(p.peek().text))
	}
	stmt := p.parseSelect()
	p.endStatement()
	p.checkParams(params)
	done, err := db.beginRead()
	if err != nil {
		return nil, nil, err
	}
	defer done()
	release, err := db.readAttached()
	if err != nil {
		return nil, nil, err
	}
	defer release()
	q := db.planQuery(stmt, nil, nil, nil)
	return q.names, q.rows(params, nil), nil
}

// catch turns a fail() panic into the returned error.
func catch(err *error) {
	if r := recover(); r != nil {
		e, ok := r.(*Error)
		if !ok {
			panic(r)
		}
		*err = e
	}
}

func newParser(sql string) *sqlParser {
	toks, err := lexSQL(sql)
	if err != nil {
		panic(err)
	}
	return &sqlParser{toks: toks, src: sql}
}

// endStatement checks that nothing follows the statement but a ;.
func (p *sqlParser) endStatement() {
	p.acceptOp(";")
	if p.peek().kind != tEOF {
		fail("SQL: unexpected %q after the end of the statement", p.peek().text)
	}
}

func (p *sqlParser) checkParams(params []Value) {
	if p.params != len(params) {
		fail("SQL: the query has %d ? placeholder(s) but %d value(s) were given", p.params, len(params))
	}
}

// fail stops the query with an error (recovered in Query).
func fail(format string, args ...any) {
	panic(errorf(format, args...).(*Error))
}

// ---- evaluating expressions ----------------------------------------------

func (r *runner) eval(e expr) Value {
	switch x := e.(type) {
	case *litExpr:
		return x.v
	case *paramExpr:
		return r.params[x.idx]
	case *colExpr:
		if x.alias != nil {
			return r.eval(x.alias)
		}
		rr := r
		for i := 0; i < x.up; i++ {
			rr = rr.parent
		}
		row := rr.cur[x.src]
		if row == nil {
			return nil // the missing side of an outer join
		}
		if x.rowid {
			return row.rowid
		}
		if x.idx >= len(row.vals) {
			// A column added after this row was written: its default.
			if x.col != nil && x.col.Default != nil {
				return r.eval(x.col.Default)
			}
			return nil
		}
		v := row.vals[x.idx]
		// SQLite stores a whole REAL like 5.0 as the integer 5 to save
		// space; it's still a REAL.
		if i, ok := v.(int64); ok && x.aff == affReal {
			return float64(i)
		}
		return v
	case *collateExpr:
		return r.eval(x.x)
	case *raiseExpr:
		return r.evalRaise(x)
	case *subqueryExpr:
		rows := x.plan.rows(r.params, r)
		if len(rows) == 0 {
			return nil
		}
		return rows[0][0]
	case *existsExpr:
		return boolValue(len(x.plan.rows(r.params, r)) > 0)
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
		same := (l == nil && rv == nil) || (l != nil && rv != nil && compareColl(l, rv, binaryCollation(x.l, x.r)) == 0)
		return boolValue(same != x.not)
	case *isNullExpr:
		return boolValue((r.eval(x.x) == nil) != x.not)
	case *inExpr:
		v := r.eval(x.x)
		if x.plan != nil {
			rows := x.plan.rows(r.params, r)
			if v == nil {
				if len(rows) == 0 {
					return boolValue(x.not)
				}
				return nil
			}
			sawNull := false
			sub := &colExpr{aff: x.plan.affs[0], coll: x.plan.colls[0], bound: true}
			coll := binaryCollation(x.x, sub)
			for _, row := range rows {
				iv := row[0]
				if iv == nil {
					sawNull = true
					continue
				}
				a, b := compareAffinity(x.x, sub, v, iv)
				if compareColl(a, b, coll) == 0 {
					return boolValue(!x.not)
				}
			}
			if sawNull {
				return nil
			}
			return boolValue(x.not)
		}
		if v == nil {
			if len(x.list) == 0 {
				return boolValue(x.not)
			}
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
			if compareColl(a, b, binaryCollation(x.x, item)) == 0 {
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
		in := compareColl(a, l, binaryCollation(x.x, x.lo)) >= 0 && compareColl(b, h, binaryCollation(x.x, x.hi)) <= 0
		return boolValue(in != x.not)
	case *likeExpr:
		v, pat := r.eval(x.x), r.eval(x.pat)
		if v == nil || pat == nil {
			return nil
		}
		if x.glob {
			return boolValue(glob([]rune(textValue(pat)), []rune(textValue(v))) != x.not)
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
		if x.win > 0 {
			if r.winVal == nil {
				fail("SQL: misuse of window function %s()", x.name)
			}
			return r.winVal[r.winRow][x.win-1]
		}
		if x.over == nil && rankingFuncs[x.name] && !aggNames[x.name] {
			fail("SQL: misuse of window function %s()", x.name)
		}
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
				if base != nil && wv != nil && compareColl(base, wv, binaryCollation(x.base, w[0])) == 0 {
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
	case "->", "->>":
		return r.evalArrow(x)
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
		c := compareColl(l, rv, binaryCollation(x.l, x.r))
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
	if math.IsNaN(f) {
		return "NaN"
	}
	if f == 0 {
		return "0.0"
	}
	// Like SQLite: 15 significant digits when they read back as the same
	// number, else 17; exponent form below 1e-4 and from 1e17 up.
	e := strconv.FormatFloat(f, 'e', 14, 64)
	if v, err := strconv.ParseFloat(e, 64); err != nil || v != f {
		e = strconv.FormatFloat(f, 'e', 16, 64)
	}
	neg := e[0] == '-'
	if neg {
		e = e[1:]
	}
	mant, expPart, _ := strings.Cut(e, "e")
	exp, _ := strconv.Atoi(expPart)
	digits := strings.TrimRight(strings.Replace(mant, ".", "", 1), "0")
	if digits == "" {
		digits = "0"
	}
	var out string
	if exp < -4 || exp >= 17 {
		frac := digits[1:]
		if frac == "" {
			frac = "0"
		}
		sign := "+"
		if exp < 0 {
			sign, exp = "-", -exp
		}
		out = fmt.Sprintf("%s.%se%s%02d", digits[:1], frac, sign, exp)
	} else if exp >= 0 {
		whole := digits
		frac := ""
		if len(digits) > exp+1 {
			whole, frac = digits[:exp+1], digits[exp+1:]
		} else {
			whole += strings.Repeat("0", exp+1-len(digits))
		}
		if frac == "" {
			frac = "0"
		}
		out = whole + "." + frac
	} else {
		out = "0." + strings.Repeat("0", -exp-1) + digits
	}
	if neg {
		out = "-" + out
	}
	return out
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
	case *collateExpr:
		return exprAffinity(x.x)
	}
	return affBlob
}

// exprCollation is the collation an expression brings to a comparison,
// and how strongly: 2 for an explicit COLLATE, 1 for a column (BINARY
// when it declares none), 0 for anything else.
func exprCollation(e expr) (string, int) {
	switch x := e.(type) {
	case *collateExpr:
		return x.coll, 2
	case *colExpr:
		if x.alias != nil {
			return exprCollation(x.alias)
		}
		if x.coll == "" {
			return "BINARY", 1
		}
		return x.coll, 1
	}
	return "", 0
}

// binaryCollation picks the collation for comparing l and r: an explicit
// COLLATE first (left before right), then a column's (left before right).
func binaryCollation(l, r expr) string {
	lc, lk := exprCollation(l)
	rc, rk := exprCollation(r)
	switch {
	case lk == 2:
		return lc
	case rk == 2:
		return rc
	case lk == 1:
		return lc
	}
	return rc
}

// compareColl is compare, with text compared under a collation: NOCASE
// folds ASCII letters, RTRIM ignores trailing spaces.
func compareColl(a, b Value, coll string) int {
	if coll == "NOCASE" || coll == "RTRIM" {
		as, aok := a.(string)
		bs, bok := b.(string)
		if aok && bok {
			if coll == "NOCASE" {
				return strings.Compare(strings.Map(lowerASCII, as), strings.Map(lowerASCII, bs))
			}
			return strings.Compare(strings.TrimRight(as, " "), strings.TrimRight(bs, " "))
		}
	}
	return compare(a, b)
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

// glob matches SQLite's GLOB: * any run, ? one character, [...] a set
// ([^...] not in it), case-sensitive.
func glob(pat, s []rune) bool {
	for len(pat) > 0 {
		c := pat[0]
		switch c {
		case '*':
			for len(pat) > 0 && pat[0] == '*' {
				pat = pat[1:]
			}
			if len(pat) == 0 {
				return true
			}
			for i := 0; i <= len(s); i++ {
				if glob(pat, s[i:]) {
					return true
				}
			}
			return false
		case '?':
			if len(s) == 0 {
				return false
			}
			pat, s = pat[1:], s[1:]
		case '[':
			if len(s) == 0 {
				return false
			}
			end := 1
			if end < len(pat) && pat[end] == '^' {
				end++
			}
			if end < len(pat) && pat[end] == ']' {
				end++
			}
			for end < len(pat) && pat[end] != ']' {
				end++
			}
			if end >= len(pat) {
				return false // an unclosed [ matches nothing
			}
			set := pat[1:end]
			neg := len(set) > 0 && set[0] == '^'
			if neg {
				set = set[1:]
			}
			in := false
			for i := 0; i < len(set); i++ {
				if i+2 < len(set) && set[i+1] == '-' {
					if s[0] >= set[i] && s[0] <= set[i+2] {
						in = true
					}
					i += 2
				} else if set[i] == s[0] {
					in = true
				}
			}
			if in == neg {
				return false
			}
			pat, s = pat[end+1:], s[1:]
		default:
			if len(s) == 0 || s[0] != c {
				return false
			}
			pat, s = pat[1:], s[1:]
		}
	}
	return len(s) == 0
}
