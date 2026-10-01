package evaluator

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"Turtle/ast"
	"Turtle/lexer"
	"Turtle/object"
	"Turtle/parser"
)

func (it *Interpreter) resolvePath(p string) string {
	if filepath.IsAbs(p) {
		return p
	}
	return filepath.Join(it.Dir, p)
}

func (it *Interpreter) evalFileRead(s *ast.FileReadStatement, env *object.Environment) {
	path := it.resolvePath(it.evalExpression(s.File, env).Inspect())
	data, err := os.ReadFile(path)
	if err != nil {
		fatalf("[read] %s: %v", path, err)
	}
	text := strings.TrimRight(string(data), "\n")
	list := &object.List{}
	if text != "" {
		for _, line := range strings.Split(text, "\n") {
			list.Elements = append(list.Elements, &object.String{Value: line})
		}
	}
	env.Set(s.Var, list)
}

func (it *Interpreter) evalFileWrite(s *ast.FileWriteStatement, env *object.Environment) {
	path := it.resolvePath(it.evalExpression(s.File, env).Inspect())
	var lines []string
	for _, item := range s.Content {
		if item.IsVar {
			v, ok := env.Get(item.Name)
			if !ok {
				fatalf("[write]/[append]: undefined variable %q", item.Name)
			}
			lines = append(lines, v.Inspect())
		} else {
			lines = append(lines, item.Literal)
		}
	}
	content := strings.Join(lines, "\n")
	if len(content) > 0 {
		content += "\n"
	}

	var err error
	if s.Append {
		var f *os.File
		f, err = os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err == nil {
			_, err = f.WriteString(content)
			f.Close()
		}
	} else {
		err = os.WriteFile(path, []byte(content), 0644)
	}
	if err != nil {
		fatalf("[write]/[append] %s: %v", path, err)
	}
}

func (it *Interpreter) evalDirectory(s *ast.DirectoryStatement, env *object.Environment) {
	path := it.resolvePath(it.evalExpression(s.Path, env).Inspect())
	entries, err := os.ReadDir(path)
	if err != nil {
		fatalf("[directory] %s: %v", path, err)
	}
	list := &object.List{}
	for _, e := range entries {
		list.Elements = append(list.Elements, &object.String{Value: e.Name()})
	}
	env.Set(s.Var, list)
}

// evalImport makes a module's functions available to the importing file:
// all of them for "import m", or just the listed ones for "import m [a,
// b]". Recognized builtin module names ("math", "time" — see
// builtinModules) are handled natively; anything else is <name>.t. Each
// .t module runs once, in its own global scope, no matter how many files
// import it — its top-level variables stay private to it, and its
// functions keep reading those, not the importer's. Two imports exporting
// the same function name is fine; only calling that name unqualified is
// an error (see resolveImported).
func (it *Interpreter) evalImport(s *ast.ImportStatement, env *object.Environment) {
	mod := it.loadModule(s.Path)
	for _, n := range s.Names {
		if !mod.Exports(n) {
			fatalf("import %s: module %q has no %q", s.Path, s.Path, n)
		}
	}
	env.AddImport(mod, s.Names)
}

func (it *Interpreter) loadModule(name string) *object.Module {
	if mod, ok := builtinModules[name]; ok {
		return mod
	}
	path := it.resolvePath(name + ".t")
	if mod, ok := it.modules[path]; ok {
		return mod
	}
	for i, p := range it.loading {
		if p == path {
			chain := append(append([]string{}, it.loading[i:]...), path)
			for j := range chain {
				chain[j] = strings.TrimSuffix(filepath.Base(chain[j]), ".t")
			}
			fatalf("import %s: circular import (%s)", name, strings.Join(chain, " -> "))
		}
	}
	data, err := os.ReadFile(path)
	if err != nil {
		fatalf("import %s: %v", name, err)
	}
	p := parser.New(lexer.New(string(data)))
	program := p.ParseProgram()
	if errs := p.Errors(); len(errs) > 0 {
		fatalf("import %s: %s", name, errs[0])
	}
	mod := &object.Module{Name: name, Env: object.NewGlobalEnvironment()}
	it.loading = append(it.loading, path)
	it.evalStatements(program.Statements, mod.Env)
	it.loading = it.loading[:len(it.loading)-1]
	it.modules[path] = mod
	return mod
}

// evalSys runs the rest of the line through a shell, inheriting
// stdin/stdout/stderr — a deliberately dangerous feature kept on
// request, on par with Python's os.system. Implemented with os/exec
// instead of the legacy cgo system() call, so the interpreter no longer
// needs a C toolchain to build.
func (it *Interpreter) evalSys(s *ast.SysStatement) {
	cmd := exec.Command("sh", "-c", s.Command)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Dir = it.Dir
	_ = cmd.Run()
}
