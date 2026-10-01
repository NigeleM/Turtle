package object

// Module is something "import" can load: a .t file, run once in its own
// global scope (Env), or a builtin like math/time. A module's exports are
// its top-level functions only — its top-level variables stay private to
// it, though its own functions can read them. A builtin module exports
// Funcs (called as name[...]) and Methods (called as x at name).
type Module struct {
	Name    string
	Env     *Environment // nil for a builtin
	Funcs   []string     // builtin only
	Methods []string     // builtin only
}

// Exports reports whether name is something this module provides — what
// an "import m [name]" list may name.
func (m *Module) Exports(name string) bool {
	return m.ExportsFunction(name) || contains(m.Methods, name)
}

// ExportsFunction reports whether name is a function this module
// provides, i.e. what "name[...]" or "m name[...]" can resolve to.
func (m *Module) ExportsFunction(name string) bool {
	if m.Env != nil {
		_, ok := m.Env.functions[name]
		return ok
	}
	return contains(m.Funcs, name)
}

// Function returns the user-defined function name from a .t module.
func (m *Module) Function(name string) (*Function, bool) {
	if m.Env == nil {
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
	return im.Names == nil || im.Names[name]
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}
