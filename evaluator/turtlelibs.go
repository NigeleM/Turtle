package evaluator

import (
	"embed"
	"fmt"
	"strings"

	"Turtle/lexer"
	"Turtle/object"
	"Turtle/parser"
)

// Builtin libraries written partly in Turtle (the hybrid standard
// library): lib/<name>.turtle is built into turtle and runs, once per program,
// when a file imports <name>. Its functions listed in the builtin's Funcs
// are exported; the rest are its private helpers. Errors from its code
// point at the caller's line, as a Go builtin's do.

//go:embed lib/*.turtle
var turtleLibFiles embed.FS

// turtleLibs are the builtins with a lib/<name>.turtle.
var turtleLibs = map[string]bool{"random": true}

const builtinFilePrefix = "builtin:"

// loadTurtleLib runs lib/<name>.turtle for the builtin mod, once.
func (it *Interpreter) loadTurtleLib(base *object.Module) *object.Module {
	key := builtinFilePrefix + base.Name
	if mod, ok := it.modules[key]; ok {
		return mod
	}
	src, err := turtleLibFiles.ReadFile("lib/" + base.Name + ".turtle")
	if err != nil {
		panic(fmt.Sprintf("builtin library %s: %v", base.Name, err))
	}
	p := parser.New(lexer.New(string(src)))
	p.Enable(base.Name) // its own sentences, without importing itself
	program := p.ParseProgram()
	if errs := p.Errors(); len(errs) > 0 {
		panic(fmt.Sprintf("builtin library %s.turtle: %s", base.Name, strings.Join(errs, "; ")))
	}
	mod := &object.Module{Name: base.Name, Funcs: base.Funcs, Methods: base.Methods, Hybrid: true, Env: object.NewGlobalEnvironment()}
	mod.Env.SetFile(key + ".turtle")
	prevFile, prevLine := currentFile, currentLine
	prevBuiltin := it.inBuiltin
	it.inBuiltin = true
	defer func() { currentFile, currentLine, it.inBuiltin = prevFile, prevLine, prevBuiltin }()
	it.runFile(program.Statements, mod.Env)
	it.modules[key] = mod
	return mod
}

// isBuiltinEnv: env belongs to a builtin library's Turtle code.
func isBuiltinEnv(env *object.Environment) bool {
	return strings.HasPrefix(env.File(), builtinFilePrefix)
}

// isBuiltin reports whether mod is the builtin called name (Go or hybrid).
func isBuiltin(mod *object.Module, name string) bool {
	return mod.Name == name && (mod == builtinModules[name] || mod.Hybrid)
}
