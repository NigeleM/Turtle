package object

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
	vars      map[string]Object
	outer     *Environment
	functions map[string]*Function
	imports   []*Import
	loop      bool   // a loop's scope: see NewLoopEnvironment
	file      string // root only: the module's file as errors show it, "" for the main script
}

// SetFile records which file a global environment belongs to, as errors
// name it ("lib/utils.trt"). The main script's is "".
func (e *Environment) SetFile(name string) { e.root().file = name }

// File is the file the code running in e comes from (see SetFile).
func (e *Environment) File() string { return e.root().file }

func NewGlobalEnvironment() *Environment {
	return &Environment{vars: map[string]Object{}, functions: map[string]*Function{}}
}

// NewEnclosedEnvironment returns a fresh scope for one function call,
// enclosing outer (the called function's defining scope). Every call gets
// its own, so recursive/re-entrant calls to the same function never share
// mutable state (the legacy interpreter's bug: one shared map per
// function definition, reused by every call).
func NewEnclosedEnvironment(outer *Environment) *Environment {
	return &Environment{vars: map[string]Object{}, outer: outer}
}

// NewLoopEnvironment returns the scope for one pass of a loop (for-each)
// or one whole loop (C-style). It holds only the loop's own names (the
// counter, or the for-each names), defined with Define. Every other
// assignment in the body passes through to the enclosing scope, exactly
// as if the loop had no scope of its own. Because each pass has its own
// scope, a closure created in the loop keeps the loop name it saw, even
// after the loop ends.
func NewLoopEnvironment(outer *Environment) *Environment {
	return &Environment{vars: map[string]Object{}, outer: outer, loop: true}
}

// Define binds name in this exact scope (a loop's own names).
func (e *Environment) Define(name string, val Object) {
	e.vars[name] = val
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

// Get looks up name in this scope, then each enclosing scope out to the
// global scope — so a function can read a top-level variable, or a
// closure its enclosing function's locals, without them being passed as
// parameters. A parameter/local of the same name always shadows the
// outer one.
func (e *Environment) Get(name string) (Object, bool) {
	for s := e; s != nil; s = s.outer {
		if v, ok := s.vars[name]; ok {
			return v, true
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
		if _, own := e.vars[name]; own {
			break
		}
		e = e.outer
	}
	e.vars[name] = val
}

func (e *Environment) Delete(name string) {
	delete(e.vars, name)
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
				im.Names[n] = true
			}
		}
		return
	}
	im := &Import{Module: mod}
	if names != nil {
		im.Names = map[string]bool{}
		for _, n := range names {
			im.Names[n] = true
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
