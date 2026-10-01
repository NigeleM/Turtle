// Package evaluator tree-walks a Turtle AST.
package evaluator

import (
	"bufio"
	"fmt"
	"io"
	"os"

	"Turtle/ast"
	"Turtle/object"
)

// Signal is a control-flow effect that a statement can produce, which
// propagates up through enclosing blocks until something (a loop, for
// Break/Continue; a function call, for Return) consumes it.
type Signal int

const (
	SigNone Signal = iota
	SigReturn
	SigBreak
	SigContinue
)

type ExecResult struct {
	Signal Signal
	Value  object.Object
}

var noneResult = ExecResult{Signal: SigNone}

// Interpreter holds everything shared across a whole program run: global
// scope and a single stdin reader (fixing the legacy interpreter's habit
// of allocating a fresh bufio.Scanner per input prompt).
type Interpreter struct {
	Global  *object.Environment
	Dir     string   // the script's directory: imports resolve relative to it
	WorkDir string   // where turtle was run from: file paths resolve relative to it
	Args    []string // command-line arguments after the script path (system's args[])
	stdin   *bufio.Scanner
	modules map[string]*object.Module // loaded .t modules, by resolved path
	loading []string                  // .t modules mid-import, outermost first
}

func New(dir string) *Interpreter {
	return NewWithStdin(dir, os.Stdin)
}

// NewWithStdin is like New but reads "?" input from r instead of
// os.Stdin — used by tests to feed a program's input prompts without
// touching the real stdin.
func NewWithStdin(dir string, r io.Reader) *Interpreter {
	wd, _ := os.Getwd()
	return &Interpreter{
		Global:  object.NewGlobalEnvironment(),
		Dir:     dir,
		WorkDir: wd,
		stdin:   bufio.NewScanner(r),
		modules: map[string]*object.Module{},
	}
}

// builtinModules maps a name recognized by "import <name>" to a native
// capability instead of a <name>.t file on disk: "math" provides the
// sqrt/abs/round/floor/ceil/pow/random number methods, "time" provides
// the now[]/sleep[ms] builtin functions, "data" provides process/keep/copy
// (see datalib.go), "system" provides command-line and filesystem functions
// (see systemlib.go), "strings" provides find/substring/isinstring/join
// (see stringslib.go). Anything else falls through to
// the file-based import.
var builtinModules = map[string]*object.Module{
	"math":    {Name: "math", Methods: []string{"sqrt", "abs", "round", "floor", "ceil", "pow", "random"}},
	"time":    {Name: "time", Funcs: []string{"now", "sleep"}},
	"data":    {Name: "data", Funcs: []string{"process", "keep", "copy"}},
	"system":  {Name: "system", Funcs: []string{"args", "exists", "isFile", "isFolder", "exit", "env", "scriptFolder", "contents"}},
	"strings": {Name: "strings", Funcs: []string{"find", "substring", "isinstring", "join"}},
}

// requireModule fails with a clear message naming the missing import,
// rather than "unknown method" — the whole point of gating these behind
// import is that the error tells you exactly what to add. Imports are
// per file, so env is whichever file's code is running.
func requireModule(env *object.Environment, module, what string) {
	im, ok := env.FindImport(module)
	if !ok {
		fatalf("%q needs \"import %s\" first", what, module)
	}
	if !im.Allows(what) {
		fatalf("%q isn't imported — add it to \"import %s [...]\"", what, module)
	}
}

// currentLine is the source line of whatever statement is being evaluated
// right now, kept up to date by evalStatement (the single dispatch point
// every statement, top-level or nested, passes through). Every fatalf call
// anywhere in the evaluator reads it, so runtime errors get a real
// location without threading a line parameter through every function.
// Turtle runs one program per process and exits on the first fatal error,
// so a package-level variable is safe here — there's never more than one
// evaluation in flight.
var currentLine int

// fatalError is what fatalf panics with. Run recovers exactly this type at
// the top of a program's evaluation and turns it into a returned error —
// any other panic (a genuine interpreter bug, not a deliberate runtime
// error) is left to propagate and crash normally, so a real bug is never
// silently swallowed.
type fatalError struct{ msg string }

func (e fatalError) Error() string { return e.msg }

// fatalf reports a runtime error and unwinds the current evaluation via
// panic/recover (see Run) rather than calling os.Exit directly — that's
// what lets tests exercise an error path (e.g. division by zero, a failed
// change conversion) without killing the test binary, while the CLI
// (cmd/turtle/main.go) still exits 1 with this exact message, since it's
// the only thing at the top that doesn't recover.
func fatalf(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	if currentLine > 0 {
		msg = fmt.Sprintf("line %d: %s", currentLine, msg)
	}
	panic(fatalError{msg})
}

// Run evaluates program to completion, or returns the fatalf error that
// stopped it (formatted exactly as the CLI has always printed it, minus
// the "turtle: " prefix the caller adds). A panic that isn't a fatalError
// is a real bug, not a Turtle runtime error, and is re-panicked rather
// than swallowed.
// ExitRequest is what Run returns when the program called system's
// exit[code]: not a failure to report, just the exit code the CLI should
// end the process with.
type ExitRequest struct{ Code int }

func (e ExitRequest) Error() string { return fmt.Sprintf("exit %d", e.Code) }

func (it *Interpreter) Run(program *ast.Program) (err error) {
	defer func() {
		if r := recover(); r != nil {
			if fe, ok := r.(fatalError); ok {
				err = fe
				return
			}
			if ex, ok := r.(ExitRequest); ok {
				err = ex
				return
			}
			panic(r)
		}
	}()
	it.evalStatements(program.Statements, it.Global)
	return nil
}

func (it *Interpreter) evalStatements(stmts []ast.Statement, env *object.Environment) ExecResult {
	for _, stmt := range stmts {
		res := it.evalStatement(stmt, env)
		if res.Signal != SigNone {
			return res
		}
	}
	return noneResult
}

func (it *Interpreter) evalBlock(block *ast.BlockStatement, env *object.Environment) ExecResult {
	if block == nil {
		return noneResult
	}
	return it.evalStatements(block.Statements, env)
}

func (it *Interpreter) evalStatement(stmt ast.Statement, env *object.Environment) ExecResult {
	currentLine = stmt.Line()
	switch s := stmt.(type) {
	case *ast.AssignStatement:
		env.Set(s.Name, it.evalExpression(s.Value, env))
		return noneResult

	case *ast.InputStatement:
		fmt.Print(s.Prompt)
		if !it.stdin.Scan() {
			// EOF (or a read error) on stdin: there's nothing left to
			// read, ever, so returning "" here would make any loop that
			// re-prompts on bad input (e.g. retrying invalid numeric
			// input) spin forever re-reading empty strings instead of
			// terminating.
			fatalf("unexpected end of input reading %q", s.Name)
		}
		env.Set(s.Name, &object.String{Value: it.stdin.Text()})
		return noneResult

	case *ast.ShowStatement:
		out := ""
		for _, e := range s.Expressions {
			out += it.evalExpression(e, env).Inspect()
		}
		fmt.Println(out)
		return noneResult

	case *ast.ExpressionStatement:
		val := it.evalExpression(s.Expression, env)
		if s.Print {
			fmt.Println(val.Inspect())
		}
		return noneResult

	case *ast.CallStatement:
		it.evalExpression(s.Call, env)
		return noneResult

	case *ast.ReturnStatement:
		if s.Value == nil {
			return ExecResult{Signal: SigReturn, Value: object.NoneValue}
		}
		return ExecResult{Signal: SigReturn, Value: it.evalExpression(s.Value, env)}

	case *ast.BreakStatement:
		return ExecResult{Signal: SigBreak}

	case *ast.ContinueStatement:
		return ExecResult{Signal: SigContinue}

	case *ast.FunctionDefStatement:
		// A top-level def goes in its file's global function table,
		// callable by name from anywhere in that file (and exported to
		// files that import it). A def nested inside a function body is a
		// closure: a local variable holding a function that captures the
		// enclosing call's scope, so it can be returned or passed along
		// and still read that call's locals after it has finished.
		fn := &object.Function{Name: s.Name, Parameters: s.Parameters, Body: s.Body, Env: env}
		if env.IsRoot() {
			env.DefineFunction(fn)
		} else {
			env.Set(s.Name, fn)
		}
		return noneResult

	case *ast.AssembleStatement:
		// Like a def: the constructor goes in the file's function table at
		// top level (so it's exported and importable), or is a local
		// inside a function body.
		shape := &object.Shape{Name: s.Name, Fields: s.Fields}
		fn := &object.Function{Name: s.Name, Parameters: s.Fields, Shape: shape, Env: env}
		if env.IsRoot() {
			env.DefineFunction(fn)
		} else {
			env.Set(s.Name, fn)
		}
		return noneResult

	case *ast.FieldAssignStatement:
		a, i := it.assemblyField(s.Target, env)
		a.Values[i] = it.evalExpression(s.Value, env)
		return noneResult

	case *ast.IfStatement:
		return it.evalIf(s, env)

	case *ast.LoopStatement:
		return it.evalLoop(s, env)

	case *ast.DataOpStatement:
		it.evalDataOp(s, env)
		return noneResult

	case *ast.ImportStatement:
		it.evalImport(s, env)
		return noneResult

	case *ast.SysStatement:
		it.evalSys(s)
		return noneResult

	case *ast.FileReadStatement:
		it.evalFileRead(s, env)
		return noneResult

	case *ast.FileWriteStatement:
		it.evalFileWrite(s, env)
		return noneResult

	case *ast.DirectoryStatement:
		it.evalDirectory(s, env)
		return noneResult

	default:
		fatalf("no evaluator for statement type %T", stmt)
		return noneResult
	}
}

func (it *Interpreter) evalIf(s *ast.IfStatement, env *object.Environment) ExecResult {
	for _, clause := range s.Clauses {
		if clause.Condition == nil {
			return it.evalBlock(clause.Body, env)
		}
		if isTruthy(it.evalExpression(clause.Condition, env)) {
			return it.evalBlock(clause.Body, env)
		}
	}
	return noneResult
}

// evalLoop runs a loop. For the C-style form it shadows the induction
// variable for the duration of the loop and restores whatever value (or
// absence) it had beforehand once the loop finishes — otherwise two
// loops nested with the same induction-variable name (a real pattern
// seen in historical Turtle scripts, e.g. both using "i") would clobber
// each other's iteration state, since Turtle has no block scoping and
// loop variables live in the same environment as everything else.
func (it *Interpreter) evalLoop(s *ast.LoopStatement, env *object.Environment) ExecResult {
	if s.Kind == ast.LoopEach {
		return it.evalEach(s, env)
	}
	if s.Kind == ast.LoopCStyle && s.Init != nil {
		var varName string
		if initAssign, ok := s.Init.(*ast.AssignStatement); ok {
			varName = initAssign.Name
		}
		var outerVal object.Object
		var hadOuter bool
		if varName != "" {
			outerVal, hadOuter = env.Get(varName)
			defer func() {
				if hadOuter {
					env.Set(varName, outerVal)
				} else {
					env.Delete(varName)
				}
			}()
		}
		it.evalStatement(s.Init, env)
	}
	for {
		if s.Condition != nil {
			if !isTruthy(it.evalExpression(s.Condition, env)) {
				break
			}
		}
		res := it.evalBlock(s.Body, env)
		if res.Signal == SigBreak {
			break
		}
		if res.Signal == SigReturn {
			return res
		}
		if s.Kind == ast.LoopCStyle && s.Post != nil {
			it.evalStatement(s.Post, env)
		}
	}
	return noneResult
}

// evalEach runs "[loop][x in c]" / "[loop][a, b in c]". With one name, x
// is each element of a list or set, each character of a string, or each
// key of a map. With two, they're (index, element) — or (key, value) for a
// map. It walks a snapshot, so adding to or removing from c inside the
// loop can't make it skip or repeat. Like the C-style loop, the loop
// names are restored to their previous values (or removed) afterwards.
func (it *Interpreter) evalEach(s *ast.LoopStatement, env *object.Environment) ExecResult {
	var firsts, seconds []object.Object
	isMap := false
	switch c := it.evalExpression(s.Iterable, env).(type) {
	case *object.List:
		seconds = append(seconds, c.Elements...)
	case *object.Set:
		seconds = append(seconds, c.Elements...)
	case *object.String:
		for _, r := range c.Value {
			seconds = append(seconds, &object.String{Value: string(r)})
		}
	case *object.Map:
		isMap = true
		for _, k := range c.Keys {
			firsts = append(firsts, &object.String{Value: k})
			seconds = append(seconds, c.Values[k])
		}
	default:
		fatalf("'[loop][... in ...]' needs a list, set, map, or string, got %s", c.Type())
	}
	if firsts == nil {
		for i := range seconds {
			firsts = append(firsts, &object.Integer{Value: int64(i)})
		}
	}
	if len(s.Vars) == 1 && isMap {
		seconds = firsts // one name over a map: its keys
	}

	for _, name := range s.Vars {
		outerVal, hadOuter := env.Get(name)
		defer func(name string) {
			if hadOuter {
				env.Set(name, outerVal)
			} else {
				env.Delete(name)
			}
		}(name)
	}
	for i := range seconds {
		if len(s.Vars) == 2 {
			env.Set(s.Vars[0], firsts[i])
			env.Set(s.Vars[1], seconds[i])
		} else {
			env.Set(s.Vars[0], seconds[i])
		}
		res := it.evalBlock(s.Body, env)
		if res.Signal == SigBreak {
			break
		}
		if res.Signal == SigReturn {
			return res
		}
	}
	return noneResult
}

func isTruthy(obj object.Object) bool {
	switch v := obj.(type) {
	case *object.Boolean:
		return v.Value
	case *object.Integer:
		return v.Value != 0
	case *object.Float:
		return v.Value != 0
	case *object.String:
		return v.Value != ""
	case *object.None:
		return false
	default:
		return obj != nil
	}
}
