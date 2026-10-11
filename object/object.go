// Copyright 2017-2026 Nigele McCoy
// Licensed under the Apache License, Version 2.0;
// see LICENSE and NOTICE.

// Package object defines Turtle's runtime values.
package object

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"Turtle/ast"
)

type Type string

const (
	INTEGER  Type = "INTEGER"
	FLOAT    Type = "FLOAT"
	STRING   Type = "STRING"
	BOOLEAN  Type = "BOOLEAN"
	LIST     Type = "LIST"
	SET      Type = "SET"
	MAP      Type = "MAP"
	FUNCTION Type = "FUNCTION"
	NONE     Type = "NONE"
	ERROR    Type = "ERROR"
	DATE     Type = "DATE"
	DATABASE Type = "DATABASE"
)

type Object interface {
	Type() Type
	Inspect() string
}

type Integer struct{ Value int64 }

func (i *Integer) Type() Type      { return INTEGER }
func (i *Integer) Inspect() string { return strconv.FormatInt(i.Value, 10) }

type Float struct{ Value float64 }

func (f *Float) Type() Type { return FLOAT }

// Inspect always shows a float as a float: 4.0, not 4, so it's never
// mistaken for an integer.
// Inspect is how a float shows: to 15 significant digits, as Excel and
// SQLite show numbers, so binary leftovers don't: 0.1 + 0.2 shows 0.3
// (it's 0.30000000000000004 inside, and == still sees that). Files and
// databases get every digit: see Exact.
func (f *Float) Inspect() string {
	v := f.Value
	if !math.IsInf(v, 0) && !math.IsNaN(v) {
		v, _ = strconv.ParseFloat(strconv.FormatFloat(v, 'g', 15, 64), 64)
	}
	return floatText(v)
}

// Exact is every digit of the float: what JSON, CSV and test failures
// write, so nothing is lost.
func (f *Float) Exact() string { return floatText(f.Value) }

func floatText(v float64) string {
	// Very big and very small numbers in scientific notation, where
	// Python switches too: 1e+16 and up, below 0.0001 (1e-05).
	if a := math.Abs(v); a != 0 && !math.IsInf(v, 0) && !math.IsNaN(v) && (a >= 1e16 || a < 1e-4) {
		return strconv.FormatFloat(v, 'g', -1, 64)
	}
	s := strconv.FormatFloat(v, 'f', -1, 64)
	if !strings.ContainsAny(s, ".eEIN") { // integral and finite (not Inf/NaN)
		s += ".0"
	}
	return s
}

// Exact is a value's text with floats in full (Float.Exact); anything
// else as it shows.
func Exact(o Object) string {
	switch v := o.(type) {
	case *Float:
		return v.Exact()
	case *Date:
		return v.Text()
	}
	return o.Inspect()
}

// ShownExact is Shown, with a float in full.
func ShownExact(o Object) string {
	if f, ok := o.(*Float); ok {
		return f.Exact()
	}
	return Shown(o)
}

type String struct{ Value string }

// Shown is how a value appears inside a list, set, map or assembled value:
// like Inspect, except that a string is quoted, so you can see its type —
// list [1, "1"] shows as [ 1, "1" ]. A string shown on its own (show "hi")
// isn't quoted.
func Shown(o Object) string {
	switch v := o.(type) {
	case *String:
		return strconv.Quote(v.Value)
	case *Matrix:
		return v.Line()
	}
	return o.Inspect()
}

func (s *String) Type() Type      { return STRING }
func (s *String) Inspect() string { return s.Value }

type Boolean struct{ Value bool }

func (b *Boolean) Type() Type { return BOOLEAN }
func (b *Boolean) Inspect() string {
	if b.Value {
		return "true"
	}
	return "false"
}

// List is Turtle's `list [...]` — ordered, duplicates allowed.
type List struct{ Elements []Object }

func (l *List) Type() Type { return LIST }
func (l *List) Inspect() string {
	parts := make([]string, len(l.Elements))
	for i, e := range l.Elements {
		parts[i] = Shown(e)
	}
	return "[ " + strings.Join(parts, ", ") + " ]"
}

// Set is Turtle's `set [...]` — insertion-ordered, deduplicated by Equal
// (so set [1, 2] and set [2, 1] are equal, and 1 and "1" are distinct).
//
// index makes membership checks fast: elements grouped by Key, confirmed
// with Equal. It's built on first use and kept up to date by Add; any
// other change to Elements must call Changed so it's rebuilt. Reordering
// in place (sort, reverse) needs no call, since membership is the same.
type Set struct {
	Elements []Object
	index    map[string][]Object
}

func (s *Set) Type() Type { return SET }
func (s *Set) Inspect() string {
	parts := make([]string, len(s.Elements))
	for i, e := range s.Elements {
		parts[i] = Shown(e)
	}
	return "{ " + strings.Join(parts, ", ") + " }"
}

func (s *Set) Contains(v Object) bool {
	if s.index == nil {
		s.index = map[string][]Object{}
		for _, e := range s.Elements {
			k := entryKey(e)
			s.index[k] = append(s.index[k], e)
		}
	}
	for _, e := range s.index[entryKey(v)] {
		if Equal(e, v) {
			return true
		}
	}
	return false
}

// Add appends v unless an equal element is already present, reporting
// whether it was added.
func (s *Set) Add(v Object) bool {
	if s.Contains(v) {
		return false
	}
	s.Elements = append(s.Elements, v)
	k := entryKey(v)
	s.index[k] = append(s.index[k], v)
	return true
}

// Changed must be called after any change to Elements other than Add or
// reordering.
func (s *Set) Changed() { s.index = nil }

// Map is Turtle's `map [...]`, insertion-ordered. Keys can be any value
// and keep their type: map [1: "a"] has the integer key 1, distinct from
// the string "1"; keys are one key when Key says so (2 and 2.0 are). The
// entries are one list in order; a map of more than smallMap entries also
// keeps an index from entryKey to position. Most maps are records and
// rows, which a short scan finds as fast, without a hash table each.
type Map struct {
	entries []MapEntry
	index   map[string]int
}

// MapEntry is one key and its value.
type MapEntry struct {
	Key Object
	Val Object
}

const smallMap = 32

func NewMap() *Map { return &Map{} }

func (m *Map) Type() Type { return MAP }
func (m *Map) Inspect() string {
	parts := make([]string, len(m.entries))
	for i, e := range m.entries {
		parts[i] = Shown(e.Key) + ": " + Shown(e.Val)
	}
	return "{ " + strings.Join(parts, ", ") + " }"
}

// Len is how many entries the map has.
func (m *Map) Len() int { return len(m.entries) }

// Entries are the map's entries in order. Read them; change a value with
// SetAt, and keys only with Put and Delete.
func (m *Map) Entries() []MapEntry { return m.entries }

// find is where key is, or -1.
func (m *Map) find(key Object) int {
	if m.index != nil {
		if i, ok := m.index[entryKey(key)]; ok {
			return i
		}
		return -1
	}
	if ks, ok := key.(*String); ok { // the usual key: text against text
		for i := range m.entries {
			if s, ok := m.entries[i].Key.(*String); ok && s.Value == ks.Value {
				return i
			}
		}
		return -1
	}
	for i := range m.entries {
		if sameKey(m.entries[i].Key, key) {
			return i
		}
	}
	return -1
}

// Put sets key to val, adding key at the end if it's new.
func (m *Map) Put(key, val Object) {
	if i := m.find(key); i >= 0 {
		m.entries[i].Val = val
		return
	}
	m.entries = append(m.entries, MapEntry{Key: key, Val: val})
	switch {
	case m.index != nil:
		m.index[entryKey(key)] = len(m.entries) - 1
	case len(m.entries) > smallMap:
		m.reindex()
	}
}

func (m *Map) reindex() {
	m.index = make(map[string]int, len(m.entries)*2)
	for i, e := range m.entries {
		m.index[entryKey(e.Key)] = i
	}
}

// SetAt changes the value of the i-th entry.
func (m *Map) SetAt(i int, val Object) { m.entries[i].Val = val }

// Get returns the value stored for key.
func (m *Map) Get(key Object) (Object, bool) {
	if i := m.find(key); i >= 0 {
		return m.entries[i].Val, true
	}
	return nil, false
}

// KeyList is the keys in order, as a new list of values.
func (m *Map) KeyList() []Object {
	out := make([]Object, len(m.entries))
	for i, e := range m.entries {
		out[i] = e.Key
	}
	return out
}

// ValueList is the values in order, as a new list.
func (m *Map) ValueList() []Object {
	out := make([]Object, len(m.entries))
	for i, e := range m.entries {
		out[i] = e.Val
	}
	return out
}

// Delete removes key, reporting whether it was there.
func (m *Map) Delete(key Object) bool {
	i := m.find(key)
	if i < 0 {
		return false
	}
	m.entries = append(m.entries[:i], m.entries[i+1:]...)
	m.entries[len(m.entries):cap(m.entries)][0] = MapEntry{} // let the removed one go
	if m.index != nil {
		if len(m.entries) > smallMap {
			m.reindex()
		} else {
			m.index = nil
		}
	}
	return true
}

// None is Turtle's "no value": the result of a function that returns
// nothing, and the value of the none literal. There is exactly one,
// NoneValue, so identity and equality coincide.
type None struct{}

func (n *None) Type() Type      { return NONE }
func (n *None) Inspect() string { return "none" }

var NoneValue = &None{}

// Function is a Turtle function value (not a call frame — each call gets
// its own fresh Environment enclosing Env; see NewEnclosedEnvironment).
// Env is the scope the def ran in, which is what makes a nested def a
// closure: its body can read the enclosing call's locals even after that
// call has returned.
//
// A Function with Shape set is the constructor an "assemble" declaration
// makes: calling it builds an Assembly instead of running a body. It's a
// function so that it's called, imported, exported and clash-checked
// exactly like one.
type Function struct {
	Name       string
	Parameters []string
	Body       *ast.BlockStatement
	Env        *Environment
	Shape      *Shape
	// Scroll is set for a saved scroll ("s = scroll a, b ."): called on
	// one value, it runs the steps in order.
	Scroll *ast.ScrollExpression
	// Theory is set for a theory's word: its phrases call it, with the
	// notation's values as its parameters.
	Theory *ast.TheoryStatement
}

func (f *Function) Type() Type { return FUNCTION }
func (f *Function) Inspect() string {
	if f.Shape != nil {
		return "assemble " + f.Name
	}
	if f.Scroll != nil {
		n := len(f.Scroll.Steps)
		s := "s"
		if n == 1 {
			s = ""
		}
		if f.Name == "" {
			return fmt.Sprintf("scroll (%d step%s)", n, s)
		}
		return fmt.Sprintf("scroll %s (%d step%s)", f.Name, n, s)
	}
	if f.Name == "" {
		return "gives function"
	}
	return "def " + f.Name
}

// Shape is what "assemble Order [item, qty, price]" declares: a named set
// of fields. Each declaration is its own Shape, so two declarations that
// happen to share a name (in different modules) are different types.
type Shape struct {
	Name   string
	Fields []string
}

// Index returns the position of field, or -1.
func (s *Shape) Index(field string) int {
	for i, f := range s.Fields {
		if f == field {
			return i
		}
	}
	return -1
}

// Assembly is a value of an assembled type, e.g. Order["pen", 3, 1.5].
// Like lists and maps it's a reference: assigning it to another name
// shares it, so changing a field through either name changes both.
type Assembly struct {
	Shape  *Shape
	Values []Object // in Shape.Fields order
}

func (a *Assembly) Type() Type { return Type(a.Shape.Name) }
func (a *Assembly) Inspect() string {
	parts := make([]string, len(a.Values))
	for i, v := range a.Values {
		parts[i] = a.Shape.Fields[i] + ": " + Shown(v)
	}
	return a.Shape.Name + " { " + strings.Join(parts, ", ") + " }"
}

// Error is what a handle statement stores when a safe block stops on an
// error. It shows as its full message ("line 3: division by zero"), and
// "kind of", "file of", "line of" and "message of" read its parts.
type Error struct {
	Kind     string // file, number, math, index, key, name, type, custom
	File     string // the file it happened in: "report.turtle", "lib/utils.turtle"
	InModule bool   // File is an imported module, so the message names it
	Line     int
	Message  string // without the place: "division by zero"
	// Steps is what each step of the scroll it happened in did, nil for
	// an error outside a scroll. Not a part of the error: only diagnose[e]
	// shows it and gives it back.
	Steps *List
}

func (e *Error) Type() Type { return ERROR }

// Inspect is the full message, as the program would have stopped with:
// "line 3: division by zero", or "lib/utils.turtle line 3: ..." when it
// happened in an imported module.
func (e *Error) Inspect() string {
	where := ""
	if e.InModule {
		where = e.File + " "
	}
	if e.Line > 0 {
		return fmt.Sprintf("%sline %d: %s", where, e.Line, e.Message)
	}
	if where != "" {
		return e.File + ": " + e.Message
	}
	return e.Message
}

// Field returns the error's part called name: kind, file, line or message.
func (e *Error) Field(name string) (Object, bool) {
	switch name {
	case "kind":
		return &String{Value: e.Kind}, true
	case "line":
		return &Integer{Value: int64(e.Line)}, true
	case "file":
		return &String{Value: e.File}, true
	case "message":
		return &String{Value: e.Message}, true
	}
	return nil, false
}

// Date is a moment in time, from the time library (today[], make_date,
// to_date, add_time, to_zone). It shows as "2026-10-03 14:05:00" in its
// own clock: local, or another zone's, then named ("2026-12-25 09:00:00
// GMT"). It compares as a moment, whatever the zone, and "year of",
// "month of", ..., "weekday of" and "zone of" read its parts. Dates are
// whole seconds.
type Date struct{ Time time.Time }

func (d *Date) Type() Type { return DATE }
func (d *Date) Inspect() string {
	s := d.Time.Format("2006-01-02 15:04:05")
	if d.Local() {
		return s
	}
	return s + " " + d.Time.Format("MST")
}

// Local reports whether the date is in the computer's own zone.
func (d *Date) Local() bool { return d.Time.Location() == time.Local }

// Text is the date as data (JSON, files, databases): "2026-10-03
// 14:05:00" for a local date, as before; with its offset for another
// zone's ("2026-12-25T09:00:00Z", "2026-12-25T18:00:00+09:00"), so it
// reads back as the same moment.
func (d *Date) Text() string {
	if d.Local() {
		return d.Time.Format("2006-01-02 15:04:05")
	}
	return d.Time.Format(time.RFC3339)
}

// Zone is the date's zone's name: "local", "UTC", "Asia/Tokyo", or an
// offset like "-04:00" for a date read with one.
func (d *Date) Zone() string {
	loc := d.Time.Location()
	switch {
	case loc == time.Local:
		return "local"
	case loc == time.UTC || loc.String() == "UTC":
		return "UTC"
	}
	if name := loc.String(); name != "" && !strings.HasPrefix(name, "+") && !strings.HasPrefix(name, "-") && strings.Contains(name, "/") {
		return name
	}
	return d.Time.Format("-07:00")
}

// Field returns the date's part called name.
func (d *Date) Field(name string) (Object, bool) {
	t := d.Time
	switch name {
	case "year":
		return &Integer{Value: int64(t.Year())}, true
	case "month":
		return &Integer{Value: int64(t.Month())}, true
	case "day":
		return &Integer{Value: int64(t.Day())}, true
	case "hour":
		return &Integer{Value: int64(t.Hour())}, true
	case "minute":
		return &Integer{Value: int64(t.Minute())}, true
	case "second":
		return &Integer{Value: int64(t.Second())}, true
	case "weekday":
		return &String{Value: t.Weekday().String()}, true
	case "zone":
		return &String{Value: d.Zone()}, true
	}
	return nil, false
}

// DateFields lists the parts Field knows, for error messages.
const DateFields = "year, month, day, hour, minute, second, weekday, zone"

// Database is an open database connection from the sql library's
// sql_open. Conn is the driver's connection (the object package doesn't
// know the drivers). It shows as "database shop.db".
type Database struct {
	Name   string
	Conn   any
	Closed bool
	// Address is a server's full address, password included, so more
	// connections can be opened (schedule's queryall); "" for SQLite.
	// Never shown: Name is what errors and show print.
	Address string
}

func (d *Database) Type() Type      { return DATABASE }
func (d *Database) Inspect() string { return "database " + d.Name }

// Values never change once made, so the common ones are made once and
// shared: Int and Bool give them out instead of making new ones, which
// saves most of a program's memory churn.
var (
	smallInts  [smallIntMax - smallIntMin + 1]Integer // one block, filled once
	trueValue  = &Boolean{Value: true}
	falseValue = &Boolean{Value: false}
)

// The shared integers cover loop counters, positions and most counts.
const (
	smallIntMin = -1024
	smallIntMax = 65535
)

func init() {
	for i := range smallInts {
		smallInts[i].Value = int64(i + smallIntMin)
	}
}

// Int is the integer n, shared when it's small.
func Int(n int64) *Integer {
	if n >= smallIntMin && n <= smallIntMax {
		return &smallInts[n-smallIntMin]
	}
	return &Integer{Value: n}
}

// Bool is true or false, shared.
func Bool(b bool) *Boolean {
	if b {
		return trueValue
	}
	return falseValue
}
