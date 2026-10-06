package evaluator

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	pathpkg "path"
	"path/filepath"
	"strings"

	"Turtle/ast"
	"Turtle/lexer"
	"Turtle/object"
	"Turtle/parser"
)

// resolvePath resolves a file path used by [read]/[write]/[append]/
// [directory] and system's exists/isFile/isFolder: relative to the folder
// turtle was run from, like any command-line tool — so
// "turtle ~/tools/count.t notes.txt" finds ./notes.txt. Scripts that want
// files next to themselves can build the path from system's
// scriptFolder[].
func (it *Interpreter) resolvePath(p string) string {
	if filepath.IsAbs(p) {
		return p
	}
	return filepath.Join(it.WorkDir, p)
}

// evalPath evaluates a file statement's path. A path written as a single
// bare word ("[read] name to lines") is the value of the variable of that
// name if one exists, so a path from args[] or a computed string works;
// otherwise it's that literal filename, as before. Anything with a "."
// or "/" in it ("notes.txt", "data/in.csv") is always literal.
func (it *Interpreter) evalPath(e ast.Expression, env *object.Environment) string {
	if id, ok := e.(*ast.Identifier); ok {
		if v, ok := env.Get(id.Value); ok {
			return v.Inspect()
		}
		return id.Value
	}
	return it.evalExpression(e, env).Inspect()
}

// resolveImportPath resolves "import name" relative to the script's own
// folder, so a program and its .t libraries can be moved together and run
// from anywhere.
func (it *Interpreter) resolveImportPath(p string) string {
	if filepath.IsAbs(p) {
		return p
	}
	return filepath.Join(it.Dir, p)
}

// fileProblem says briefly what went wrong with a file, without the full
// resolved path Go's own message repeats.
func fileProblem(err error) string {
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return "no such file or folder"
	case errors.Is(err, fs.ErrPermission):
		return "permission denied"
	}
	var pe *fs.PathError
	if errors.As(err, &pe) {
		return pe.Err.Error()
	}
	return err.Error()
}

func (it *Interpreter) evalFileRead(s *ast.FileReadStatement, env *object.Environment) {
	name := it.evalPath(s.File, env)
	data, err := os.ReadFile(it.resolvePath(name))
	if err != nil {
		fatalKind(kindFile, "[read] %s: %s", name, fileProblem(err))
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
	name := it.evalPath(s.File, env)
	path := it.resolvePath(name)
	var lines []string
	for _, item := range s.Content {
		if item.Expr != nil {
			lines = append(lines, it.evalExpression(item.Expr, env).Inspect())
		} else if item.IsVar {
			v, ok := env.Get(item.Name)
			if !ok {
				fatalKind(kindName, "[write]/[append]: undefined variable %q", item.Name)
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
		fatalKind(kindFile, "[write]/[append] %s: %s", name, fileProblem(err))
	}
}

func (it *Interpreter) evalDirectory(s *ast.DirectoryStatement, env *object.Environment) {
	name := it.evalPath(s.Path, env)
	entries, err := os.ReadDir(it.resolvePath(name))
	if err != nil {
		fatalKind(kindFile, "[directory] %s: %s", name, fileProblem(err))
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
	if other, ok := env.FindImport(mod.Name); ok && other.Module != mod {
		fatalKind(kindName, "import %s: this file already imports another module called %q — rename one of the files", s.Path, mod.Name)
	}
	for _, n := range s.Names {
		if !mod.Exports(n) {
			fatalKind(kindName, "import %s: module %q has no %q", s.Path, s.Path, n)
		}
	}
	env.AddImport(mod, s.Names)
	if im, _ := env.FindImport("data"); im != nil && im.Module == builtinModules["data"] && im.Allows("table") {
		defineTableRows(env)
	}
}

func (it *Interpreter) loadModule(name string) *object.Module {
	if mod, ok := builtinModules[name]; ok {
		return mod
	}
	if _, ok := builtinModules[pathpkg.Base(name)]; ok {
		fatalKind(kindName, "import %s: %q is the name of a builtin module — rename the file", name, pathpkg.Base(name))
	}
	path := it.resolveImportPath(filepath.FromSlash(name) + ".t")
	if mod, ok := it.modules[path]; ok {
		return mod
	}
	for i, p := range it.loading {
		if p == path {
			chain := append(append([]string{}, it.loading[i:]...), path)
			for j := range chain {
				chain[j] = strings.TrimSuffix(filepath.Base(chain[j]), ".t")
			}
			fatalKind(kindName, "import %s: circular import (%s)", name, strings.Join(chain, " -> "))
		}
	}
	file := name + ".t"
	if rel, err := filepath.Rel(it.Dir, path); err == nil {
		file = filepath.ToSlash(rel)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		fatalKind(kindFile, "import %s: %s: %s", name, file, fileProblem(err))
	}
	p := parser.New(lexer.New(string(data)))
	program := p.ParseProgram()
	if errs := p.Errors(); len(errs) > 0 {
		// errs[0] is "line N: ...": name the module's file and line.
		var line int
		text := errs[0]
		if _, err := fmt.Sscanf(text, "line %d: ", &line); err == nil {
			text = strings.TrimPrefix(text, fmt.Sprintf("line %d: ", line))
		}
		panic(fatalError{msg: place(file, line) + text, text: text, kind: kindFile, line: line, file: file, parse: true})
	}
	mod := &object.Module{Name: pathpkg.Base(name), Env: object.NewGlobalEnvironment()}
	mod.Env.SetFile(file)
	it.loading = append(it.loading, path)
	prevFile, prevLine := currentFile, currentLine
	currentFile = file
	defer func() { // also when a safe block handles an error from the module
		it.loading = it.loading[:len(it.loading)-1]
		currentFile, currentLine = prevFile, prevLine
	}()
	it.evalStatements(program.Statements, mod.Env)
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
