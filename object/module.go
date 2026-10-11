// Copyright 2017-2026 Nigele McCoy
// Licensed under the Apache License, Version 2.0;
// see LICENSE and NOTICE.

package object

// Module is something "import" can load: a .t file, run once in its own
// global scope (Env), or a builtin like math/time. A module's exports are
// its top-level functions only — its top-level variables stay private to
// it, though its own functions can read them. A builtin module exports
// Funcs (called as name[...]) and Methods (called as x at name).
//
// A builtin can also be partly written in Turtle (Hybrid): then Env holds
// its Turtle functions, and it exports just Funcs, whether each is
// written in Go or in Turtle; its other Turtle functions are helpers.
type Module struct {
	Name    string
	Env     *Environment // nil for a builtin written only in Go
	Funcs   []string     // builtin only
	Methods []string     // builtin only
	Hybrid  bool         // a builtin with Turtle functions in Env
	// Aliases are a builtin's old function names, each to its current
	// one: scriptFolder still means scriptfolder.
	Aliases map[string]string
}

// Canonical is name's current spelling (name itself unless it's an old
// name of one of this module's functions).
func (m *Module) Canonical(name string) string {
	if n, ok := m.Aliases[name]; ok {
		return n
	}
	return name
}

// IsPrivate reports whether name is a private function's, ~limit: it can
// be used only in the file that defines it, never by one that imports it.
func IsPrivate(name string) bool {
	return len(name) > 1 && name[0] == '~'
}

// HasPrivate reports whether this module has the private function name,
// for an error that says so rather than that it doesn't exist.
func (m *Module) HasPrivate(name string) bool {
	if m.Env == nil || !IsPrivate(name) {
		return false
	}
	_, ok := m.Env.functions[name]
	return ok
}

// Reachable reports whether code may use this module's function name: an
// exported one, or, for a test file (fromTest), a private one too.
func (m *Module) Reachable(name string, fromTest bool) bool {
	if IsPrivate(name) {
		return fromTest && m.HasPrivate(name)
	}
	return m.ExportsFunction(name)
}

// Exports reports whether name is something this module provides — what
// an "import m [name]" list may name.
func (m *Module) Exports(name string) bool {
	return m.ExportsFunction(name) || contains(m.Methods, name)
}

// ExportsFunction reports whether name is a function this module
// provides, i.e. what "name[...]" or "m name[...]" can resolve to.
func (m *Module) ExportsFunction(name string) bool {
	if IsPrivate(name) {
		return false
	}
	if m.Env != nil && !m.Hybrid {
		_, ok := m.Env.functions[name]
		return ok
	}
	return contains(m.Funcs, m.Canonical(name))
}

// Function returns the user-defined function name from a .t module (or
// an exported Turtle function of a hybrid builtin).
// Whether the caller may have a private one is decided before, by
// Reachable.
func (m *Module) Function(name string) (*Function, bool) {
	if m.Env == nil || m.Hybrid && !contains(m.Funcs, name) && !IsPrivate(name) {
		return nil, false
	}
	fn, ok := m.Env.functions[name]
	return fn, ok
}

// Import is one module as imported into one scope: either the whole
// module (Names nil) or only the names listed in "import m [a, b]".
type Import struct {
	Module *Module
	Names  map[string]bool
}

// Allows reports whether this import makes name available.
func (im *Import) Allows(name string) bool {
	return im.Names == nil || im.Names[im.Module.Canonical(name)]
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}
