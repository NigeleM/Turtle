package object

// Environment holds variable bindings for one scope. Function definitions
// are only ever stored on the global environment: a function body has no
// closure over its caller's locals, but can read top-level (global)
// variables and call any top-level function regardless of which scope
// it's running in.
type Environment struct {
	vars      map[string]Object
	global    *Environment
	functions map[string]*Function
}

func NewGlobalEnvironment() *Environment {
	return &Environment{vars: map[string]Object{}, functions: map[string]*Function{}}
}

// NewCallEnvironment returns a fresh, isolated scope for one function
// call, so recursive/re-entrant calls to the same function never share
// mutable state (the legacy interpreter's bug: one shared map per
// function definition, reused by every call).
func (e *Environment) NewCallEnvironment() *Environment {
	global := e
	if e.global != nil {
		global = e.global
	}
	return &Environment{vars: map[string]Object{}, global: global}
}

// Get looks up name in this scope, falling back to the global scope if
// it's not a local (a parameter, loop variable, or local assignment) —
// so a function can read a top-level variable without it being passed as
// a parameter. A parameter/local of the same name always shadows the
// global.
func (e *Environment) Get(name string) (Object, bool) {
	if v, ok := e.vars[name]; ok {
		return v, ok
	}
	if e.global != nil {
		return e.global.Get(name)
	}
	return nil, false
}

// Set always writes to this scope's own locals, never the global scope,
// even if a global of the same name exists — so a plain assignment inside
// a function can't silently mutate global state through a name collision;
// it just shadows the global for the rest of that call. (A data-structure
// operation or method call on a global list/set/map still mutates it in
// place, since that goes through the same *object.List/Set/Map value
// Get() returned, not through Set.)
func (e *Environment) Set(name string, val Object) {
	e.vars[name] = val
}

func (e *Environment) Delete(name string) {
	delete(e.vars, name)
}

func (e *Environment) GetFunction(name string) (*Function, bool) {
	if e.global != nil {
		return e.global.GetFunction(name)
	}
	fn, ok := e.functions[name]
	return fn, ok
}

func (e *Environment) DefineFunction(fn *Function) {
	if e.global != nil {
		e.global.DefineFunction(fn)
		return
	}
	e.functions[fn.Name] = fn
}
