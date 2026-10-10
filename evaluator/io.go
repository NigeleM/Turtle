package evaluator

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	pathpkg "path"
	"path/filepath"
	"strings"

	"Turtle/ast"
	"Turtle/lexer"
	"Turtle/object"
	"Turtle/parser"
	"Turtle/syntax"
)

// resolvePath resolves a file path used by [read]/[write]/[append]/
// [directory] and system's exists/isfile/isfolder: relative to the folder
// turtle was run from, like any command-line tool — so
// "turtle ~/tools/count.turtle notes.txt" finds ./notes.txt. Scripts that want
// files next to themselves can build the path from system's
// scriptfolder[].
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
// builtinModules) are handled natively; anything else is <name>.turtle
// (or <name>.trt). Each module runs once, in its own global scope, no matter how many files
// import it — its top-level variables stay private to it, and its
// functions keep reading those, not the importer's. Two imports exporting
// the same function name is fine; only calling that name unqualified is
// an error (see resolveImported).
func (it *Interpreter) evalImport(s *ast.ImportStatement, env *object.Environment) {
	var mod *object.Module
	if env.File() == builtinFilePrefix+s.Path+".turtle" {
		// lib/data.turtle's own "import data": the library's Go half.
		mod = builtinModules[s.Path]
	} else {
		mod = it.loadModule(s.Path)
	}
	if other, ok := env.FindImport(mod.Name); ok && other.Module != mod {
		fatalKind(kindName, "import %s: this file already imports another module called %q — rename one of the files", s.Path, mod.Name)
	}
	for _, n := range s.Names {
		if !mod.Exports(n) && !mod.Reachable(n, env.IsTestFile()) {
			if object.IsPrivate(n) {
				fatalKind(kindName, "import %s: %s is private to %s (a ~ function is for its own file)", s.Path, n, s.Path)
			}
			fatalKind(kindName, "import %s: module %q has no %q", s.Path, s.Path, n)
		}
	}
	env.AddImport(mod, s.Names)
	if im, _ := env.FindImport("data"); im != nil && im.Module == builtinModules["data"] && im.Allows("table") {
		defineTableRows(env)
	}
	if isBuiltin(mod, "random") {
		defineSeed(env)
	}
	if isBuiltin(mod, "test") {
		defineTestSettings(env)
	}
	if isBuiltin(mod, "log") {
		defineLogSettings(env)
	}
	if isBuiltin(mod, "schedule") {
		defineScheduleSettings(env)
	}
	if isBuiltin(mod, "server") {
		defineServerSettings(env)
	}
}

func (it *Interpreter) loadModule(name string) *object.Module {
	if mod, ok := builtinModules[name]; ok {
		if turtleLibs[name] {
			return it.loadTurtleLib(mod)
		}
		return mod
	}
	if _, ok := builtinModules[pathpkg.Base(name)]; ok {
		fatalKind(kindName, "import %s: %q is the name of a builtin module — rename the file", name, pathpkg.Base(name))
	}
	ext, err := syntax.FindModule(name, func(ext string) bool {
		return it.moduleExists(name+ext, it.resolveImportPath(filepath.FromSlash(name)+ext))
	})
	if err != nil {
		fatalKind(kindFile, "import %s: %v", name, err)
	}
	if ext == "" {
		ext = syntax.Extensions[0] // not there: the message names the official ending
	}
	path := it.resolveImportPath(filepath.FromSlash(name) + ext)
	if mod, ok := it.modules[path]; ok {
		return mod
	}
	for i, p := range it.loading {
		if p == path {
			chain := append(append([]string{}, it.loading[i:]...), path)
			for j := range chain {
				chain[j] = syntax.TrimExtension(filepath.Base(chain[j]))
			}
			fatalKind(kindName, "import %s: circular import (%s)", name, strings.Join(chain, " -> "))
		}
	}
	file := name + ext
	if rel, err := filepath.Rel(it.Dir, path); err == nil {
		file = filepath.ToSlash(rel)
	}
	data, err := it.readModule(file, path)
	if err != nil {
		fatalKind(kindFile, "import %s: %s: %s", name, file, fileProblem(err))
	}
	if it.Trace != nil {
		it.TraceSource(file, string(data))
	}
	if it.debug != nil {
		it.debug.src[file] = splitLines(string(data))
	}
	p := parser.New(lexer.New(string(data)))
	p.ModuleDir = it.Dir
	p.Modules = it.Bundle
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
	it.runFile(program.Statements, mod.Env)
	it.modules[path] = mod
	return mod
}

// evalSys runs the rest of the line through a shell (sh, or cmd on
// Windows), inheriting stdin/stdout/stderr — a deliberately dangerous
// feature kept on request, on par with Python's os.system.
func (it *Interpreter) evalSys(s *ast.SysStatement) {
	cmd := shellCommand(s.Command)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Dir = it.WorkDir // where turtle was run, as file paths resolve
	_ = cmd.Run()
}

// moduleExists reports whether an imported file is there: in the
// program's bundle (as file) or on disk (at path).
func (it *Interpreter) moduleExists(file, path string) bool {
	if it.Bundle != nil {
		if _, err := fs.Stat(it.Bundle, filepath.ToSlash(file)); err == nil {
			return true
		}
	}
	_, err := os.Stat(path)
	return err == nil
}

// readModule reads an imported file: from the program's own bundle first
// (turtle build packs a program's imports into it), then from disk.
func (it *Interpreter) readModule(file, path string) ([]byte, error) {
	if it.Bundle != nil {
		if data, err := fs.ReadFile(it.Bundle, file); err == nil {
			return data, nil
		}
	}
	return os.ReadFile(path)
}
