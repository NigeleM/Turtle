// Copyright 2017-2026 Nigele McCoy
// Licensed under the Apache License, Version 2.0;
// see LICENSE and NOTICE.

// Package toml reads and writes TOML 1.0 (https://toml.io), written from
// scratch on the standard library, for Turtle's config library.
//
// Parse gives a *Table: keys in the order the file has them, values of
// type string, int64, float64, bool, DateTime, []any or *Table. Write
// turns such a table back into TOML text.
package toml

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

// Table is a TOML table: its keys in order, and their values.
type Table struct {
	Keys   []string
	Values map[string]any

	// How the table came to be, for TOML's rules on defining one twice.
	implicit bool // made on the way to a [a.b.c] header, not yet defined itself
	dotted   bool // made by a dotted key (a.b = 1)
	inline   bool // an inline table { ... }: closed once written
	header   bool // defined by its own [header]
}

// NewTable is an empty table.
func NewTable() *Table { return &Table{Values: map[string]any{}} }

// Get is the value at key, if there is one.
func (t *Table) Get(key string) (any, bool) {
	v, ok := t.Values[key]
	return v, ok
}

// Set puts value at key, keeping the first place a key was given.
func (t *Table) Set(key string, value any) {
	if _, ok := t.Values[key]; !ok {
		t.Keys = append(t.Keys, key)
	}
	t.Values[key] = value
}

// DateTimeKind is which of TOML's four date and time forms a value is.
type DateTimeKind int

const (
	OffsetDateTime DateTimeKind = iota // 1979-05-27T07:32:00Z, ...-07:00
	LocalDateTime                      // 1979-05-27T07:32:00
	LocalDate                          // 1979-05-27
	LocalTime                          // 07:32:00
)

// DateTime is a TOML date or time. A local one's Time is in time.Local;
// a LocalTime's date part is January 1 of year 0.
type DateTime struct {
	Time time.Time
	Kind DateTimeKind
}

// Error is a problem in TOML text, with the line it's on.
type Error struct {
	Line int
	Msg  string
}

func (e *Error) Error() string { return fmt.Sprintf("line %d: %s", e.Line, e.Msg) }

// arrayOfTables marks an array made by [[header]]s, which more [[ ]]
// can add to (a value array can't be).
type arrayOfTables struct{ tables []*Table }

// Parse reads TOML text.
func Parse(text string) (t *Table, err error) {
	p := &parser{src: strings.TrimPrefix(text, "\ufeff"), line: 1}
	root := NewTable()
	defer func() {
		if r := recover(); r != nil {
			e, ok := r.(*Error)
			if !ok {
				panic(r)
			}
			t, err = nil, e
		}
	}()
	p.document(root)
	return finish(root), nil
}

// finish turns the arrays of tables into plain []any.
func finish(t *Table) *Table {
	for _, k := range t.Keys {
		t.Values[k] = finishValue(t.Values[k])
	}
	return t
}

func finishValue(v any) any {
	switch x := v.(type) {
	case *Table:
		return finish(x)
	case *arrayOfTables:
		out := make([]any, len(x.tables))
		for i, t := range x.tables {
			out[i] = finish(t)
		}
		return out
	case []any:
		for i := range x {
			x[i] = finishValue(x[i])
		}
	}
	return v
}

type parser struct {
	src  string
	pos  int
	line int
}

func (p *parser) fail(format string, args ...any) {
	panic(&Error{Line: p.line, Msg: fmt.Sprintf(format, args...)})
}

func (p *parser) eof() bool { return p.pos >= len(p.src) }

func (p *parser) peek() byte {
	if p.eof() {
		return 0
	}
	return p.src[p.pos]
}

func (p *parser) at(s string) bool { return strings.HasPrefix(p.src[p.pos:], s) }

func (p *parser) next() byte {
	c := p.src[p.pos]
	p.pos++
	if c == '\n' {
		p.line++
	}
	return c
}

// spaces skips spaces and tabs.
func (p *parser) spaces() {
	for !p.eof() && (p.peek() == ' ' || p.peek() == '\t') {
		p.pos++
	}
}

// comment skips a # comment to the end of its line.
func (p *parser) comment() {
	if p.peek() != '#' {
		return
	}
	for !p.eof() && p.peek() != '\n' {
		c := p.peek()
		if c < 0x20 && c != '\t' && !(c == '\r' && p.at("\r\n")) || c == 0x7f {
			p.fail("a comment can't hold control characters")
		}
		p.pos++
	}
}

// lineEnd expects the rest of a line to be empty or a comment.
func (p *parser) lineEnd(after string) {
	p.spaces()
	p.comment()
	switch {
	case p.eof():
	case p.at("\r\n"):
		p.pos++
		p.next()
	case p.peek() == '\n':
		p.next()
	default:
		p.fail("unexpected %s after %s (one thing per line; # starts a comment)", p.describe(), after)
	}
}

func (p *parser) describe() string {
	if p.eof() {
		return "end of file"
	}
	r, _ := utf8.DecodeRuneInString(p.src[p.pos:])
	return strconv.QuoteRune(r)
}

// blank skips spaces, newlines and comments between values in an array.
func (p *parser) blank() {
	for !p.eof() {
		switch p.peek() {
		case ' ', '\t', '\n':
			p.next()
		case '\r':
			if !p.at("\r\n") {
				p.fail("a carriage return must be followed by a newline")
			}
			p.next()
		case '#':
			p.comment()
		default:
			return
		}
	}
}

func (p *parser) document(root *Table) {
	current := root
	for {
		p.blank()
		if p.eof() {
			return
		}
		switch {
		case p.at("[["):
			p.pos += 2
			keys := p.key()
			if !p.at("]]") {
				p.fail("a [[table]] header ends with ]]")
			}
			p.pos += 2
			current = p.arrayTable(root, keys)
			p.lineEnd("a [[table]] header")
		case p.peek() == '[':
			p.pos++
			keys := p.key()
			if p.peek() != ']' {
				p.fail("a [table] header ends with ]")
			}
			p.pos++
			current = p.headerTable(root, keys)
			p.lineEnd("a [table] header")
		default:
			p.keyValue(current)
			p.lineEnd("a value")
		}
	}
}

// key reads a dotted key: bare-key, "quoted", 'literal', joined by dots.
func (p *parser) key() []string {
	var keys []string
	for {
		p.spaces()
		switch c := p.peek(); {
		case c == '"':
			if p.at(`"""`) {
				p.fail("a key can't be a multi-line string")
			}
			keys = append(keys, p.basicString())
		case c == '\'':
			if p.at("'''") {
				p.fail("a key can't be a multi-line string")
			}
			keys = append(keys, p.literalString())
		case isBare(c):
			start := p.pos
			for !p.eof() && isBare(p.peek()) {
				p.pos++
			}
			keys = append(keys, p.src[start:p.pos])
		default:
			p.fail("expected a key, found %s", p.describe())
		}
		p.spaces()
		if p.peek() != '.' {
			return keys
		}
		p.pos++
	}
}

func isBare(c byte) bool {
	return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '_' || c == '-'
}

func dottedName(keys []string) string {
	parts := make([]string, len(keys))
	for i, k := range keys {
		parts[i] = quoteKey(k)
	}
	return strings.Join(parts, ".")
}

// descend walks into table t's sub-table key, making it if it's missing
// (marked by mark), through an array of tables to its last table.
func (p *parser) descend(t *Table, key string, path []string, mark func(*Table)) *Table {
	v, ok := t.Get(key)
	if !ok {
		sub := NewTable()
		mark(sub)
		t.Set(key, sub)
		return sub
	}
	switch x := v.(type) {
	case *Table:
		if x.inline {
			p.fail("%s is an inline table { ... }, which can't be added to later", dottedName(path))
		}
		return x
	case *arrayOfTables:
		return x.tables[len(x.tables)-1]
	}
	p.fail("%s is already a value, not a table", dottedName(path))
	return nil
}

func (p *parser) headerTable(root *Table, keys []string) *Table {
	t := root
	for i, k := range keys[:len(keys)-1] {
		t = p.descend(t, k, keys[:i+1], func(s *Table) { s.implicit = true })
	}
	last := keys[len(keys)-1]
	v, ok := t.Get(last)
	if !ok {
		sub := NewTable()
		sub.header = true
		t.Set(last, sub)
		return sub
	}
	sub, isTable := v.(*Table)
	switch {
	case !isTable:
		if _, aot := v.(*arrayOfTables); aot {
			p.fail("[%s] is an array of tables ([[%s]]), so it can't also be a [table]", dottedName(keys), dottedName(keys))
		}
		p.fail("[%s]: %s is already a value, not a table", dottedName(keys), dottedName(keys))
	case sub.inline:
		p.fail("[%s] is an inline table { ... } already", dottedName(keys))
	case sub.header || sub.dotted:
		p.fail("table [%s] is defined twice", dottedName(keys))
	}
	sub.implicit, sub.header = false, true
	return sub
}

func (p *parser) arrayTable(root *Table, keys []string) *Table {
	t := root
	for i, k := range keys[:len(keys)-1] {
		t = p.descend(t, k, keys[:i+1], func(s *Table) { s.implicit = true })
	}
	last := keys[len(keys)-1]
	sub := NewTable()
	sub.header = true
	v, ok := t.Get(last)
	if !ok {
		t.Set(last, &arrayOfTables{tables: []*Table{sub}})
		return sub
	}
	a, isArray := v.(*arrayOfTables)
	if !isArray {
		p.fail("[[%s]]: %s is already a table or a value, not an array of tables", dottedName(keys), dottedName(keys))
	}
	a.tables = append(a.tables, sub)
	return sub
}

func (p *parser) keyValue(t *Table) {
	line := p.line
	keys := p.key()
	if p.peek() != '=' {
		p.fail("expected = after the key %s, found %s", dottedName(keys), p.describe())
	}
	p.pos++
	p.spaces()
	if p.eof() || p.peek() == '\n' || p.peek() == '#' || p.at("\r\n") {
		p.fail("%s = needs a value", dottedName(keys))
	}
	value := p.value()
	for i, k := range keys[:len(keys)-1] {
		v, ok := t.Get(k)
		if ok {
			sub, isTable := v.(*Table)
			if !isTable {
				p.line = line
				p.fail("%s is already a value, not a table", dottedName(keys[:i+1]))
			}
			if sub.inline || sub.header || (!sub.dotted && !sub.implicit) {
				p.line = line
				p.fail("%s is a table defined elsewhere; add to it under its own [header]", dottedName(keys[:i+1]))
			}
			t = sub
			continue
		}
		sub := NewTable()
		sub.dotted = true
		t.Set(k, sub)
		t = sub
	}
	last := keys[len(keys)-1]
	if _, exists := t.Get(last); exists {
		p.line = line
		p.fail("%s is already set (each key once)", dottedName(keys))
	}
	t.Set(last, value)
}

// value reads any value; it ends right after the value.
func (p *parser) value() any {
	switch c := p.peek(); {
	case p.at(`"""`):
		return p.multiBasicString()
	case c == '"':
		return p.basicString()
	case p.at("'''"):
		return p.multiLiteralString()
	case c == '\'':
		return p.literalString()
	case c == '[':
		return p.array()
	case c == '{':
		return p.inlineTable()
	case p.at("true") && !p.bareAfter(4):
		p.pos += 4
		return true
	case p.at("false") && !p.bareAfter(5):
		p.pos += 5
		return false
	}
	return p.scalar()
}

func (p *parser) bareAfter(n int) bool {
	return p.pos+n < len(p.src) && isBare(p.src[p.pos+n])
}

func (p *parser) basicString() string {
	p.pos++ // "
	var b strings.Builder
	for {
		if p.eof() || p.peek() == '\n' {
			p.fail(`a "string" isn't closed on its line`)
		}
		c := p.next()
		switch {
		case c == '"':
			return b.String()
		case c == '\\':
			p.escape(&b)
		case c < 0x20 && c != '\t' || c == 0x7f:
			p.fail("a string can't hold control characters; write them as escapes like \\n")
		default:
			b.WriteByte(c)
		}
	}
}

func (p *parser) escape(b *strings.Builder) {
	if p.eof() {
		p.fail("a \\ at the end of the file")
	}
	switch c := p.next(); c {
	case 'b':
		b.WriteByte('\b')
	case 't':
		b.WriteByte('\t')
	case 'n':
		b.WriteByte('\n')
	case 'f':
		b.WriteByte('\f')
	case 'r':
		b.WriteByte('\r')
	case 'e':
		b.WriteByte(0x1b)
	case '"':
		b.WriteByte('"')
	case '\\':
		b.WriteByte('\\')
	case 'u', 'U':
		n := 4
		if c == 'U' {
			n = 8
		}
		if p.pos+n > len(p.src) {
			p.fail("\\%c needs %d hex digits", c, n)
		}
		code, err := strconv.ParseUint(p.src[p.pos:p.pos+n], 16, 32)
		if err != nil || !utf8.ValidRune(rune(code)) {
			p.fail("\\%c%s isn't a Unicode character", c, p.src[p.pos:p.pos+n])
		}
		p.pos += n
		b.WriteRune(rune(code))
	default:
		p.fail("\\%c isn't an escape in a string; use \\\\ for a backslash", c)
	}
}

func (p *parser) multiBasicString() string {
	p.pos += 3
	p.trimFirstNewline()
	var b strings.Builder
	for {
		if p.eof() {
			p.fail(`a """string""" isn't closed`)
		}
		if p.at(`"""`) {
			// Up to two more quotes belong to the string: """a""""" is a"".
			n := 3
			for n < 5 && p.pos+n < len(p.src) && p.src[p.pos+n] == '"' {
				n++
			}
			b.WriteString(strings.Repeat(`"`, n-3))
			p.pos += n
			return b.String()
		}
		c := p.next()
		switch {
		case c == '\\':
			// A \ at a line's end drops it and the space up to the next text.
			rest := p.pos
			for rest < len(p.src) && (p.src[rest] == ' ' || p.src[rest] == '\t') {
				rest++
			}
			if rest < len(p.src) && (p.src[rest] == '\n' || p.src[rest] == '\r') {
				p.pos = rest
				for !p.eof() && strings.IndexByte(" \t\r\n", p.peek()) >= 0 {
					p.next()
				}
				continue
			}
			p.escape(&b)
		case c == '\r' && p.peek() == '\n':
		case c < 0x20 && c != '\t' && c != '\n' || c == 0x7f:
			p.fail("a string can't hold control characters; write them as escapes like \\n")
		default:
			b.WriteByte(c)
		}
	}
}

func (p *parser) trimFirstNewline() {
	if p.at("\r\n") {
		p.pos++
	}
	if p.peek() == '\n' {
		p.next()
	}
}

func (p *parser) literalString() string {
	p.pos++ // '
	start := p.pos
	for {
		if p.eof() || p.peek() == '\n' {
			p.fail("a 'string' isn't closed on its line")
		}
		c := p.next()
		if c == '\'' {
			return p.src[start : p.pos-1]
		}
		if c < 0x20 && c != '\t' || c == 0x7f {
			p.fail("a 'string' can't hold control characters")
		}
	}
}

func (p *parser) multiLiteralString() string {
	p.pos += 3
	p.trimFirstNewline()
	var b strings.Builder
	for {
		if p.eof() {
			p.fail("a '''string''' isn't closed")
		}
		if p.at("'''") {
			n := 3
			for n < 5 && p.pos+n < len(p.src) && p.src[p.pos+n] == '\'' {
				n++
			}
			b.WriteString(strings.Repeat("'", n-3))
			p.pos += n
			return b.String()
		}
		c := p.next()
		if c == '\r' && p.peek() == '\n' {
			continue
		}
		if c < 0x20 && c != '\t' && c != '\n' || c == 0x7f {
			p.fail("a string can't hold control characters")
		}
		b.WriteByte(c)
	}
}

func (p *parser) array() []any {
	p.pos++ // [
	out := []any{}
	for {
		p.blank()
		if p.peek() == ']' {
			p.pos++
			return out
		}
		if p.eof() {
			p.fail("an array [ ... ] isn't closed")
		}
		out = append(out, p.value())
		p.blank()
		switch p.peek() {
		case ',':
			p.pos++
		case ']':
			p.pos++
			return out
		default:
			p.fail("expected , or ] in an array, found %s", p.describe())
		}
	}
}

func (p *parser) inlineTable() *Table {
	p.pos++ // {
	t := NewTable()
	p.spaces()
	if p.peek() == '}' {
		p.pos++
		t.inline = true
		return t
	}
	for {
		p.spaces()
		if p.peek() == '\n' || p.eof() {
			p.fail("an inline table { ... } goes on one line")
		}
		p.keyValue(t)
		p.spaces()
		switch p.peek() {
		case ',':
			p.pos++
			p.spaces()
			if p.peek() == '}' {
				p.fail("an inline table can't end with a comma")
			}
		case '}':
			p.pos++
			freeze(t)
			return t
		default:
			p.fail("expected , or } in an inline table, found %s", p.describe())
		}
	}
}

// freeze marks an inline table and the tables made inside it as closed.
func freeze(t *Table) {
	t.inline = true
	for _, v := range t.Values {
		if sub, ok := v.(*Table); ok {
			freeze(sub)
		}
	}
}

// scalar reads a number, a date or a time.
func (p *parser) scalar() any {
	start := p.pos
	for !p.eof() && strings.IndexByte("0123456789abcdefABCDEFxonitTZ_+-.:", p.peek()) >= 0 {
		p.pos++
	}
	// "1979-05-27 07:32:00": a date, a space, then a time.
	if p.pos-start == 10 && p.peek() == ' ' && p.pos+3 < len(p.src) && isDigit(p.src[p.pos+1]) && isDigit(p.src[p.pos+2]) && p.src[p.pos+3] == ':' {
		p.pos++
		for !p.eof() && strings.IndexByte("0123456789Z+-.:", p.peek()) >= 0 {
			p.pos++
		}
	}
	tok := p.src[start:p.pos]
	if tok == "" && isBare(p.peek()) {
		for !p.eof() && isBare(p.peek()) {
			p.pos++
		}
		tok = p.src[start:p.pos]
		p.fail("%q isn't a value: text needs quotes (\"%s\"), and numbers, true, false and dates are written plainly", tok, tok)
	}
	if tok == "" {
		p.fail("expected a value, found %s", p.describe())
	}
	if dt, ok := parseDateTime(tok); ok {
		return dt
	}
	if strings.ContainsAny(tok, ":") || len(tok) >= 10 && tok[4] == '-' && tok[7] == '-' {
		p.fail("%q isn't a date or time TOML knows (like 1979-05-27, 07:32:00 or 1979-05-27T07:32:00Z)", tok)
	}
	if n, ok := parseInt(tok); ok {
		return n
	}
	if f, ok := parseFloat(tok); ok {
		return f
	}
	p.fail("%q isn't a value: text needs quotes (\"%s\"), and numbers, true, false and dates are written plainly", tok, tok)
	return nil
}

func isDigit(c byte) bool { return c >= '0' && c <= '9' }

// underscores checks TOML's rule (each _ between two digits) and drops them.
func underscores(s string, digit func(byte) bool) (string, bool) {
	if !strings.Contains(s, "_") {
		return s, true
	}
	for i := 0; i < len(s); i++ {
		if s[i] == '_' && (i == 0 || i == len(s)-1 || !digit(s[i-1]) || !digit(s[i+1])) {
			return "", false
		}
	}
	return strings.ReplaceAll(s, "_", ""), true
}

func parseInt(tok string) (int64, bool) {
	isHex := func(c byte) bool { return isDigit(c) || c >= 'a' && c <= 'f' || c >= 'A' && c <= 'F' }
	for _, pre := range []struct {
		p     string
		base  int
		digit func(byte) bool
	}{{"0x", 16, isHex}, {"0o", 8, func(c byte) bool { return c >= '0' && c <= '7' }}, {"0b", 2, func(c byte) bool { return c == '0' || c == '1' }}} {
		if strings.HasPrefix(tok, pre.p) {
			s, ok := underscores(tok[2:], pre.digit)
			if !ok || s == "" {
				return 0, false
			}
			for i := 0; i < len(s); i++ {
				if !pre.digit(s[i]) {
					return 0, false
				}
			}
			n, err := strconv.ParseUint(s, pre.base, 64)
			if err != nil || n > math.MaxInt64 {
				return 0, false
			}
			return int64(n), true
		}
	}
	s := tok
	sign := ""
	if s != "" && (s[0] == '+' || s[0] == '-') {
		sign, s = s[:1], s[1:]
	}
	s, ok := underscores(s, isDigit)
	if !ok || s == "" || len(s) > 1 && s[0] == '0' {
		return 0, false
	}
	for i := 0; i < len(s); i++ {
		if !isDigit(s[i]) {
			return 0, false
		}
	}
	n, err := strconv.ParseInt(sign+s, 10, 64)
	return n, err == nil
}

func parseFloat(tok string) (float64, bool) {
	s := tok
	sign := 1.0
	if s != "" && (s[0] == '+' || s[0] == '-') {
		if s[0] == '-' {
			sign = -1
		}
		s = s[1:]
	}
	switch s {
	case "inf":
		return sign * math.Inf(1), true
	case "nan":
		return math.NaN(), true
	}
	mant, exp, hasExp := strings.Cut(strings.ToLower(s), "e")
	whole, frac, hasFrac := strings.Cut(mant, ".")
	if !hasExp && !hasFrac {
		return 0, false
	}
	whole, ok1 := underscores(whole, isDigit)
	frac, ok2 := underscores(frac, isDigit)
	if !ok1 || !ok2 || whole == "" || hasFrac && frac == "" || len(whole) > 1 && whole[0] == '0' || !allDigits(whole) || !allDigits(frac) {
		return 0, false
	}
	if hasExp {
		e := exp
		if e != "" && (e[0] == '+' || e[0] == '-') {
			e = e[1:]
		}
		e, ok := underscores(e, isDigit)
		if !ok || e == "" || !allDigits(e) {
			return 0, false
		}
	}
	f, err := strconv.ParseFloat(strings.ReplaceAll(s, "_", ""), 64)
	if err != nil && !strings.Contains(err.Error(), "range") {
		return 0, false
	}
	return sign * f, true
}

func allDigits(s string) bool {
	for i := 0; i < len(s); i++ {
		if !isDigit(s[i]) {
			return false
		}
	}
	return true
}

func parseDateTime(tok string) (DateTime, bool) {
	s := strings.Replace(tok, " ", "T", 1)
	s = strings.Replace(s, "t", "T", 1)
	s = strings.Replace(s, "z", "Z", 1)
	if len(s) >= 11 && s[10] == 'T' {
		date, clock := s[:10], s[11:]
		zone := ""
		if i := strings.IndexAny(clock, "Z+-"); i >= 0 {
			clock, zone = clock[:i], clock[i:]
		}
		if !validDate(date) || !validClock(clock) {
			return DateTime{}, false
		}
		if zone == "" {
			t, err := time.ParseInLocation("2006-01-02T15:04:05.999999999", date+"T"+fullClock(clock), time.Local)
			return DateTime{Time: t, Kind: LocalDateTime}, err == nil
		}
		t, err := time.Parse(time.RFC3339Nano, date+"T"+fullClock(clock)+zone)
		return DateTime{Time: t, Kind: OffsetDateTime}, err == nil
	}
	if len(s) == 10 && validDate(s) {
		t, err := time.ParseInLocation("2006-01-02", s, time.Local)
		return DateTime{Time: t, Kind: LocalDate}, err == nil
	}
	if validClock(s) {
		t, err := time.Parse("15:04:05.999999999", fullClock(s))
		return DateTime{Time: t, Kind: LocalTime}, err == nil
	}
	return DateTime{}, false
}

func validDate(s string) bool {
	if len(s) != 10 || s[4] != '-' || s[7] != '-' {
		return false
	}
	_, err := time.Parse("2006-01-02", s)
	return err == nil
}

// validClock: hh:mm:ss with optional fraction (hh:mm too, as TOML 1.1 allows).
func validClock(s string) bool {
	if len(s) < 5 || s[2] != ':' {
		return false
	}
	_, err := time.Parse("15:04:05.999999999", fullClock(s))
	return err == nil
}

func fullClock(s string) string {
	if len(s) == 5 {
		return s + ":00"
	}
	return s
}

// ---- writing ----------------------------------------------------------------

// Write turns a table into TOML text: plain keys first, then [tables],
// then [[arrays of tables]], each with its own header.
func Write(t *Table) (string, error) {
	var b strings.Builder
	if err := writeTable(&b, t, nil); err != nil {
		return "", err
	}
	return strings.TrimLeft(b.String(), "\n"), nil
}

func writeTable(b *strings.Builder, t *Table, path []string) error {
	var tables, arrays []string
	for _, k := range t.Keys {
		v := t.Values[k]
		switch x := v.(type) {
		case *Table:
			tables = append(tables, k)
			continue
		case []any:
			if len(x) > 0 && allTables(x) {
				arrays = append(arrays, k)
				continue
			}
		}
		s, err := valueText(v, append(path, k))
		if err != nil {
			return err
		}
		fmt.Fprintf(b, "%s = %s\n", quoteKey(k), s)
	}
	for _, k := range tables {
		sub := t.Values[k].(*Table)
		p := append(append([]string{}, path...), k)
		if hasPlain(sub) || len(sub.Keys) == 0 {
			fmt.Fprintf(b, "\n[%s]\n", dottedName(p))
		}
		if err := writeTable(b, sub, p); err != nil {
			return err
		}
	}
	for _, k := range arrays {
		p := append(append([]string{}, path...), k)
		for _, e := range t.Values[k].([]any) {
			fmt.Fprintf(b, "\n[[%s]]\n", dottedName(p))
			if err := writeTable(b, e.(*Table), p); err != nil {
				return err
			}
		}
	}
	return nil
}

// hasPlain: a table with a key of its own (not only sub-tables).
func hasPlain(t *Table) bool {
	for _, k := range t.Keys {
		switch x := t.Values[k].(type) {
		case *Table:
			continue
		case []any:
			if len(x) > 0 && allTables(x) {
				continue
			}
		}
		return true
	}
	return false
}

func allTables(items []any) bool {
	for _, e := range items {
		if _, ok := e.(*Table); !ok {
			return false
		}
	}
	return true
}

func valueText(v any, path []string) (string, error) {
	switch x := v.(type) {
	case string:
		return quoteString(x), nil
	case bool:
		return strconv.FormatBool(x), nil
	case int64:
		return strconv.FormatInt(x, 10), nil
	case int:
		return strconv.Itoa(x), nil
	case float64:
		switch {
		case math.IsNaN(x):
			return "nan", nil
		case math.IsInf(x, 1):
			return "inf", nil
		case math.IsInf(x, -1):
			return "-inf", nil
		}
		s := strconv.FormatFloat(x, 'g', -1, 64)
		if !strings.ContainsAny(s, ".eEn") {
			s += ".0"
		}
		return s, nil
	case DateTime:
		switch x.Kind {
		case LocalDate:
			return x.Time.Format("2006-01-02"), nil
		case LocalTime:
			return x.Time.Format("15:04:05.999999999"), nil
		case LocalDateTime:
			return x.Time.Format("2006-01-02T15:04:05.999999999"), nil
		}
		return x.Time.Format(time.RFC3339Nano), nil
	case []any:
		parts := make([]string, len(x))
		for i, e := range x {
			s, err := valueText(e, path)
			if err != nil {
				return "", err
			}
			parts[i] = s
		}
		return "[" + strings.Join(parts, ", ") + "]", nil
	case *Table:
		parts := make([]string, len(x.Keys))
		for i, k := range x.Keys {
			s, err := valueText(x.Values[k], append(path, k))
			if err != nil {
				return "", err
			}
			parts[i] = quoteKey(k) + " = " + s
		}
		if len(parts) == 0 {
			return "{}", nil
		}
		return "{ " + strings.Join(parts, ", ") + " }", nil
	case nil:
		return "", fmt.Errorf("%s is none, and TOML has no none: leave the key out", dottedName(path))
	}
	return "", fmt.Errorf("%s can't be written as TOML (%T)", dottedName(path), v)
}

func quoteKey(k string) string {
	if k == "" {
		return `""`
	}
	for i := 0; i < len(k); i++ {
		if !isBare(k[i]) {
			return quoteString(k)
		}
	}
	return k
}

func quoteString(s string) string {
	var b strings.Builder
	b.WriteByte('"')
	for _, r := range s {
		switch r {
		case '"':
			b.WriteString(`\"`)
		case '\\':
			b.WriteString(`\\`)
		case '\n':
			b.WriteString(`\n`)
		case '\t':
			b.WriteString(`\t`)
		case '\r':
			b.WriteString(`\r`)
		case '\b':
			b.WriteString(`\b`)
		case '\f':
			b.WriteString(`\f`)
		default:
			if r < 0x20 || r == 0x7f {
				fmt.Fprintf(&b, `\u%04X`, r)
			} else {
				b.WriteRune(r)
			}
		}
	}
	b.WriteByte('"')
	return b.String()
}
