// Package object defines Turtle's runtime values.
package object

import (
	"fmt"
	"strconv"
	"strings"

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
)

type Object interface {
	Type() Type
	Inspect() string
}

type Integer struct{ Value int64 }

func (i *Integer) Type() Type      { return INTEGER }
func (i *Integer) Inspect() string { return strconv.FormatInt(i.Value, 10) }

type Float struct{ Value float64 }

func (f *Float) Type() Type      { return FLOAT }
func (f *Float) Inspect() string { return strconv.FormatFloat(f.Value, 'f', -1, 64) }

type String struct{ Value string }

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
		parts[i] = e.Inspect()
	}
	return "[ " + strings.Join(parts, ", ") + " ]"
}

// Set is Turtle's `set [...]` — insertion-ordered, deduplicated by
// Inspect() string equality (matching legacy's string-based dedup).
type Set struct{ Elements []Object }

func (s *Set) Type() Type { return SET }
func (s *Set) Inspect() string {
	parts := make([]string, len(s.Elements))
	for i, e := range s.Elements {
		parts[i] = e.Inspect()
	}
	return "{ " + strings.Join(parts, ", ") + " }"
}

func (s *Set) Contains(v Object) bool {
	for _, e := range s.Elements {
		if e.Inspect() == v.Inspect() {
			return true
		}
	}
	return false
}

// Map is Turtle's `map [...]` — insertion-ordered (fixes the legacy
// interpreter's randomized Go-map iteration order).
type Map struct {
	Keys   []string
	Values map[string]Object
}

func NewMap() *Map { return &Map{Values: map[string]Object{}} }

func (m *Map) Type() Type { return MAP }
func (m *Map) Inspect() string {
	parts := make([]string, len(m.Keys))
	for i, k := range m.Keys {
		parts[i] = fmt.Sprintf("%s: %s", k, m.Values[k].Inspect())
	}
	return "{ " + strings.Join(parts, ", ") + " }"
}

func (m *Map) Set(key string, val Object) {
	if _, exists := m.Values[key]; !exists {
		m.Keys = append(m.Keys, key)
	}
	m.Values[key] = val
}

func (m *Map) Delete(key string) bool {
	if _, exists := m.Values[key]; !exists {
		return false
	}
	delete(m.Values, key)
	for i, k := range m.Keys {
		if k == key {
			m.Keys = append(m.Keys[:i], m.Keys[i+1:]...)
			break
		}
	}
	return true
}

// Function is a Turtle function definition (not a call frame — each
// call gets its own fresh Environment; see Environment.NewCall).
type Function struct {
	Name       string
	Parameters []string
	Body       *ast.BlockStatement
}

func (f *Function) Type() Type      { return FUNCTION }
func (f *Function) Inspect() string { return "def " + f.Name }
