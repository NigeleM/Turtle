package evaluator

import (
	"embed"
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strings"

	"Turtle/ast"
	"Turtle/lexer"
	"Turtle/object"
	"Turtle/parser"
	"Turtle/syntax"
)

// The standard library written in Turtle: each lib/<name>.turtle is built
// into turtle and runs, once per program, when a file imports <name>.
//
//   - A file named after a library written in Go (data.turtle) adds its
//     functions to that library: import data gives both, alike.
//   - Any other name (geometry.turtle) is a new library of its own.
//   - Its public functions are its top-level defs; ~ functions are its
//     private helpers. A public function can't have the name of one the
//     library has in Go.
//   - Each function's documentation (turtle doc, help, the editor) is the
//     comment above its def, // lines or a //* *// block; a library of its
//     own is described by the comment at the top of its file.
//   - Inside the file, "import <name>" is the library's Go half.
//
// Errors from its code point at the caller's line, as a Go builtin's do.
// Nothing else needs changing to add one: a rebuild picks it up.

//go:embed lib/*.turtle
var turtleLibFiles embed.FS

// libFiles is where the libraries in Turtle come from (tests swap it).
var libFiles fs.FS = turtleLibFiles

// turtleLibs are the builtins with a lib/<name>.turtle.
var turtleLibs = map[string]bool{}

func init() {
	if err := registerTurtleLibs(libFiles, builtinModules, moduleDocs, turtleLibs); err != nil {
		panic(err)
	}
}

// registerTurtleLibs adds each lib/<name>.turtle in files to the
// libraries (modules), their documentation (docs) and the set of
// libraries with Turtle code (libs).
func registerTurtleLibs(files fs.FS, modules map[string]*object.Module, docs map[string]string, libs map[string]bool) error {
	names, err := fs.Glob(files, "lib/*.turtle")
	if err != nil {
		return err
	}
	sort.Strings(names)
	for _, file := range names {
		name := strings.TrimSuffix(path.Base(file), ".turtle")
		data, err := fs.ReadFile(files, file)
		if err != nil {
			return err
		}
		src := strings.ReplaceAll(string(data), "\r\n", "\n")
		p := parser.New(lexer.New(src))
		p.Enable(name)
		program := p.ParseProgram()
		if errs := p.Errors(); len(errs) > 0 {
			return fmt.Errorf("builtin library %s: %s", file, strings.Join(errs, "; "))
		}
		mod, ok := modules[name]
		if !ok {
			mod = &object.Module{Name: name}
			modules[name] = mod
		}
		lines := strings.Split(src, "\n")
		var entries []string
		for _, st := range program.Statements {
			// Its public functions, and its assembled types, which are
			// exported as a file's are.
			var fn, call, doc string
			switch d := st.(type) {
			case *ast.FunctionDefStatement:
				if object.IsPrivate(d.Name) {
					continue
				}
				fn, call = d.Name, d.Name+"["+strings.Join(d.Parameters, ", ")+"]"
				doc = syntax.FunctionDoc(lines, d.Token.Line-1)
			case *ast.AssembleStatement:
				fn, call = d.Name, d.Name+"["+strings.Join(d.Fields, ", ")+"]"
				doc = syntax.CommentAbove(lines, d.Token.Line-1)
			default:
				continue
			}
			if containsName(mod.Funcs, fn) || containsName(mod.Methods, fn) {
				return fmt.Errorf("builtin library %s: %s is already a function of %s written in Go; a function in Turtle needs a name of its own", file, fn, name)
			}
			mod.Funcs = append(mod.Funcs, fn)
			if documented(docs[name], fn) {
				continue
			}
			if doc == "" {
				doc = "(no description yet: write // lines just above it)"
			}
			entries = append(entries, "### "+call+"\n  "+strings.ReplaceAll(doc, "\n", "\n  "))
		}
		if _, ok := docs[name]; !ok {
			intro, _ := syntax.FileDoc(lines)
			if intro == "" {
				intro = "The " + name + " library."
			}
			docs[name] = intro
		}
		if len(entries) > 0 {
			docs[name] = strings.TrimRight(docs[name], "\n") + "\n\n" + strings.Join(entries, "\n\n")
		}
		libs[name] = true
	}
	return nil
}

// documented reports whether a library's documentation has an entry for
// the function name already.
func documented(text, name string) bool {
	_, entries := parseModuleDoc("", text)
	for _, e := range entries {
		if e.name() == name {
			return true
		}
	}
	return false
}

func containsName(list []string, name string) bool {
	for _, s := range list {
		if s == name {
			return true
		}
	}
	return false
}

const builtinFilePrefix = "builtin:"

// loadTurtleLib runs lib/<name>.turtle for the builtin mod, once.
func (it *Interpreter) loadTurtleLib(base *object.Module) *object.Module {
	key := builtinFilePrefix + base.Name
	if mod, ok := it.modules[key]; ok {
		return mod
	}
	src, err := fs.ReadFile(libFiles, "lib/"+base.Name+".turtle")
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
