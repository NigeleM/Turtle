// Copyright 2017-2026 Nigele McCoy
// Licensed under the Apache License, Version 2.0;
// see LICENSE and NOTICE.

package object

import (
	"sort"
	"unsafe"
)

// Environment holds variable bindings for one scope. Scopes form a chain:
// each function call's scope encloses the scope its function was defined
// in (the global scope for a top-level def, or the enclosing call's scope
// for a nested def — that's what makes a nested def a closure). Lookups
// walk the chain outward, so every function can read top-level (global)
// variables and any variables of the functions it's nested inside.
//
// Top-level function definitions live in a separate table on the global
// environment, so any scope can call any top-level function by name.
// Nested defs are instead ordinary local variables holding a *Function.
//
// The main program and every imported .t module each have their own
// global (root) environment, which also records what that file imported.
type Environment struct {
	// vars are this scope's own names and values. Most scopes (a call, a
	// loop pass) hold a few, which a short list finds fastest; index
	// takes over once a scope has more than smallScope names.
	vars      []binding
	small     [4]binding // vars' first home: most scopes need no more, so no second allocation
	index     map[string]int
	outer     *Environment
	functions map[string]*Function
	imports   []*Import
	loop      bool   // a loop's scope: see NewLoopEnvironment
	file      string // root only: the module's file as errors show it, "" for the main script
	testFile  bool   // root only: a test file run by turtle test, which may use its imports' ~ functions
	// captured is set once a function (a give, a nested def, a saved
	// scroll, an assembled type) keeps this scope; a scope nobody kept
	// can be reused for the next loop pass or call (see Reuse).
	captured bool
}

// Capture marks e and every scope around it as kept by a function made
// in it, so none of them is ever reused.
func (e *Environment) Capture() {
	for s := e; s != nil && !s.captured; s = s.outer {
		s.captured = true
	}
}

// Captured reports whether a function kept e (see Capture).
func (e *Environment) Captured() bool { return e.captured }

// Reuse empties a scope nobody kept, to stand for a new loop pass or
// call inside outer: what a new scope would be, without making one.
func (e *Environment) Reuse(outer *Environment) {
	clear(e.vars) // drop the old values, so they can be collected
	if len(e.vars) <= len(e.small) {
		e.vars = e.small[:0]
	} else {
		e.vars = e.vars[:0]
	}
	e.index = nil
	e.outer = outer
}

// Release empties a finished call's scope as it's put aside for reuse,
// so the values its call made can be collected now, not when the next
// call takes the scope.
func (e *Environment) Release() {
	e.Reuse(nil)
}

// SetFile records which file a global environment belongs to, as errors
// name it ("lib/utils.turtle"). The main script's is "".
func (e *Environment) SetFile(name string) { e.root().file = name }

// File is the file the code running in e comes from (see SetFile).
func (e *Environment) File() string { return e.root().file }

// MarkTestFile marks a global environment as a test file's, run by
// turtle test: its code may use the private (~) functions of the modules
// it imports, so a test can test them. Those modules stay as private to
// each other, and to programs, as ever.
func (e *Environment) MarkTestFile() { e.root().testFile = true }

// IsTestFile reports whether the code running in e is a test file's.
func (e *Environment) IsTestFile() bool { return e.root().testFile }

// binding is one name and its value.
type binding struct {
	name string
	val  Object
}

const smallScope = 8

// find is where name is among this scope's own vars, or -1.
func (e *Environment) find(name string) int {
	if e.index != nil {
		if i, ok := e.index[name]; ok {
			return i
		}
		return -1
	}
	for i := range e.vars {
		// Names are shared strings (the lexer makes one per name), so the
		// same name is usually the same address; a different first letter
		// rules a name out without comparing the rest.
		n := e.vars[i].name
		if len(n) == len(name) && (unsafe.StringData(n) == unsafe.StringData(name) || n[0] == name[0] && n == name) {
			return i
		}
	}
	return -1
}

// put sets name in this exact scope.
func (e *Environment) put(name string, val Object) {
	if i := e.find(name); i >= 0 {
		e.vars[i].val = val
		return
	}
	if e.vars == nil {
		e.vars = e.small[:0]
	}
	e.vars = append(e.vars, binding{name, val})
	switch {
	case e.index != nil:
		e.index[name] = len(e.vars) - 1
	case len(e.vars) > smallScope:
		e.index = make(map[string]int, len(e.vars)*2)
		for i, b := range e.vars {
			e.index[b.name] = i
		}
	}
}

func NewGlobalEnvironment() *Environment {
	return &Environment{functions: map[string]*Function{}}
}

// NewEnclosedEnvironment returns a fresh scope for one function call,
// enclosing outer (the called function's defining scope). Every call gets
// its own, so recursive/re-entrant calls to the same function never share
// mutable state.
func NewEnclosedEnvironment(outer *Environment) *Environment {
	return &Environment{outer: outer}
}

// NewLoopEnvironment returns the scope for one pass of a loop (for-each)
// or one whole loop (C-style). It holds only the loop's own names (the
// counter, or the for-each names), defined with Define. Every other
// assignment in the body passes through to the enclosing scope, exactly
// as if the loop had no scope of its own. Because each pass has its own
// scope, a closure created in the loop keeps the loop name it saw, even
// after the loop ends.
func NewLoopEnvironment(outer *Environment) *Environment {
	return &Environment{outer: outer, loop: true}
}

// Define binds name in this exact scope (a loop's own names).
func (e *Environment) Define(name string, val Object) {
	e.put(name, val)
}

// IsTopLevel reports whether code running in e is at a file's top level:
// e is the global scope, or only loop scopes sit between e and it. A def
// or assemble there is a top-level one.
func (e *Environment) IsTopLevel() bool {
	for e.loop {
		e = e.outer
	}
	return e.outer == nil
}

// NewCallEnvironment returns a fresh call scope enclosing e.
func (e *Environment) NewCallEnvironment() *Environment {
	return NewEnclosedEnvironment(e)
}

func (e *Environment) root() *Environment {
	for e.outer != nil {
		e = e.outer
	}
	return e
}

// IsRoot reports whether e is a global environment (the main program's
// or an imported module's) rather than a function call's scope.
func (e *Environment) IsRoot() bool { return e.outer == nil }

// Outer is the scope around this one (nil for a global one).
func (e *Environment) Outer() *Environment { return e.outer }

// Get looks up name in this scope, then each enclosing scope out to the
// global scope — so a function can read a top-level variable, or a
// closure its enclosing function's locals, without them being passed as
// parameters. A parameter/local of the same name always shadows the
// outer one.
func (e *Environment) Get(name string) (Object, bool) {
	for s := e; s != nil; s = s.outer {
		if i := s.find(name); i >= 0 {
			return s.vars[i].val, true
		}
	}
	return nil, false
}

// Set always writes to this scope's own locals, never an enclosing scope,
// even if an outer variable of the same name exists — so a plain
// assignment inside a function can't silently mutate global state (or a
// closure its enclosing function's state) through a name collision; it
// just shadows the outer variable for the rest of that call. (A
// data-structure operation or method call on an outer list/set/map still
// mutates it in place, since that goes through the same
// *object.List/Set/Map value Get() returned, not through Set.)
func (e *Environment) Set(name string, val Object) {
	for e.loop {
		if e.find(name) >= 0 {
			break
		}
		e = e.outer
	}
	e.put(name, val)
}

// Shadows reports whether a plain assignment to name here would make a
// new local that hides a variable of an enclosing scope, and whether that
// variable is a global (else it's an enclosing function's).
func (e *Environment) Shadows(name string) (shadows, global bool) {
	for e.loop {
		if e.find(name) >= 0 {
			return false, false
		}
		e = e.outer
	}
	if e.outer == nil || e.find(name) >= 0 {
		return false, false
	}
	for s := e.outer; s != nil; s = s.outer {
		if s.find(name) >= 0 {
			return true, s.outer == nil
		}
	}
	return false, false
}

func (e *Environment) Delete(name string) {
	i := e.find(name)
	if i < 0 {
		return
	}
	e.vars = append(e.vars[:i], e.vars[i+1:]...)
	if e.index != nil {
		e.index = make(map[string]int, len(e.vars)*2)
		for j, b := range e.vars {
			e.index[b.name] = j
		}
	}
}

func (e *Environment) GetFunction(name string) (*Function, bool) {
	fn, ok := e.root().functions[name]
	return fn, ok
}

func (e *Environment) DefineFunction(fn *Function) {
	e.root().functions[fn.Name] = fn
}

// AddImport records that this file imported mod — the whole module when
// names is nil, otherwise just those names. Importing the same module
// again merges: a full import wins over any list, and lists accumulate.
func (e *Environment) AddImport(mod *Module, names []string) {
	r := e.root()
	for _, im := range r.imports {
		if im.Module != mod {
			continue
		}
		if names == nil {
			im.Names = nil
		} else if im.Names != nil {
			for _, n := range names {
				im.Names[mod.Canonical(n)] = true
			}
		}
		return
	}
	im := &Import{Module: mod}
	if names != nil {
		im.Names = map[string]bool{}
		for _, n := range names {
			im.Names[mod.Canonical(n)] = true
		}
	}
	r.imports = append(r.imports, im)
}

// Imports returns this file's imports, in import order.
func (e *Environment) Imports() []*Import {
	return e.root().imports
}

// FindImport returns this file's import of the module named name.
func (e *Environment) FindImport(name string) (*Import, bool) {
	for _, im := range e.root().imports {
		if im.Module.Name == name {
			return im, true
		}
	}
	return nil, false
}

// Names lists this scope's own variables and functions, sorted.
func (e *Environment) Names() (vars, funcs []string) {
	for _, b := range e.vars {
		vars = append(vars, b.name)
	}
	for n := range e.functions {
		funcs = append(funcs, n)
	}
	sort.Strings(vars)
	sort.Strings(funcs)
	return vars, funcs
}

// Functions are the top-level functions of e's file.
func (e *Environment) Functions() []*Function {
	r := e.root()
	out := make([]*Function, 0, len(r.functions))
	for _, fn := range r.functions {
		out = append(out, fn)
	}
	return out
}
