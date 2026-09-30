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

// evalImport parses and runs another .t file directly into the global
// environment, matching the legacy interpreter's flatten-into-globals
// behavior (there is no module namespacing in Turtle). Recognized builtin
// module names ("math", "time" — see builtinModules) are handled natively
// instead: no file is read, "import <name>" just flips that module on.
func (it *Interpreter) evalImport(s *ast.ImportStatement, env *object.Environment) {
	if builtinModules[s.Path] {
		it.modules[s.Path] = true
		return
	}
	path := it.resolvePath(s.Path + ".t")
	data, err := os.ReadFile(path)
	if err != nil {
		fatalf("import %s: %v", s.Path, err)
	}
	p := parser.New(lexer.New(string(data)))
	program := p.ParseProgram()
	if errs := p.Errors(); len(errs) > 0 {
		fatalf("import %s: %s", s.Path, errs[0])
	}
	it.evalStatements(program.Statements, it.Global)
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
