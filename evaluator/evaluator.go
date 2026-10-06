// Package evaluator tree-walks a Turtle AST.
package evaluator

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"slices"

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
	Script  string   // the main script's file name, for "file of" an error
	stdin   *bufio.Scanner
	modules map[string]*object.Module // loaded .t modules, by resolved path
	depth   int                       // function calls in progress (see maxCallDepth)
	loading []string                  // .t modules mid-import, outermost first
	dbs     []*object.Database        // databases opened, closed when Run ends
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

// maxCallDepth caps how many function calls can be in progress at once,
// so runaway recursion is a Turtle error instead of Go running out of
// stack and crashing with a stack dump. Well below that crash point even
// for large function bodies.
const maxCallDepth = 100000

// builtinModules maps a name recognized by "import <name>" to a native
// capability instead of a <name>.t file on disk: "math" provides the
// sqrt/abs/round/floor/ceil/pow/random number methods, "time" provides
// the now[]/sleep[ms] builtin functions, "data" provides process/keep/copy
// (see datalib.go), table (see tablelib.go), table_read and table_write
// (see tablefile.go), "system" provides command-line and filesystem functions
// (see systemlib.go), "strings" provides find/substring/isinstring/join
// (see stringslib.go), "json" provides load/json_text/json_read/
// json_write/json_get (see jsonlib.go), "http" provides http_get/
// http_post/http_request (see httplib.go), "sql" provides sql_open/
// sql_create/sql_query/sql_run/sql_tables/sql_load/sql_save/sql_update/
// sql_delete/sql_upsert/sql_close (see sqllib.go), "sort" provides min_sort/
// max_sort and the classic sorting algorithms (see sortlib.go), "search"
// provides find_* and the search algorithms (see searchlib.go). Anything else falls through to
// the file-based import.
var builtinModules = map[string]*object.Module{
	"math":    {Name: "math", Methods: []string{"sqrt", "abs", "round", "floor", "ceil", "pow", "random"}},
	"time":    {Name: "time", Funcs: []string{"now", "sleep", "today", "today_utc", "make_date", "to_date", "add_time", "time_between", "format_date", "wait_until", "every"}},
	"data":    {Name: "data", Funcs: []string{"process", "keep", "copy", "table", "table_read", "table_write"}},
	"system":  {Name: "system", Funcs: []string{"args", "exists", "isFile", "isFolder", "exit", "env", "scriptFolder", "contents", "erase", "warn"}},
	"strings": {Name: "strings", Funcs: []string{"find", "substring", "isinstring", "join"}},
	"json":    {Name: "json", Funcs: []string{"load", "json_text", "json_read", "json_write", "json_get"}},
	"http":    {Name: "http", Funcs: []string{"http_get", "http_post", "http_request"}},
	"sort":    {Name: "sort", Funcs: []string{"min_sort", "max_sort", "is_sorted", "reverse_list", "bubble_sort", "insertion_sort", "selection_sort", "merge_sort", "quick_sort", "heap_sort", "shell_sort", "counting_sort", "radix_sort"}},
	"search":  {Name: "search", Funcs: []string{"find_first", "find_last", "find_all", "find_index", "count_where", "find_key", "linear_search", "binary_search", "jump_search", "exponential_search", "interpolation_search", "ternary_search", "insert_position"}},
	"sql":     {Name: "sql", Funcs: []string{"sql_open", "sql_create", "sql_query", "sql_run", "sql_tables", "sql_load", "sql_save", "sql_update", "sql_delete", "sql_upsert", "sql_close"}},
}

// requireModule fails with a clear message naming the missing import,
// rather than "unknown method" — the whole point of gating these behind
// import is that the error tells you exactly what to add. Imports are
// per file, so env is whichever file's code is running.
func requireModule(env *object.Environment, module, what string) {
	im, ok := env.FindImport(module)
	if !ok {
		fatalKind(kindName, "%q needs \"import %s\" first", what, module)
	}
	if !im.Allows(what) {
		fatalKind(kindName, "%q isn't imported — add it to \"import %s [...]\"", what, module)
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

// currentFile is the file whose code is running, as errors name it: ""
// for the main script, "lib/utils.t" for an imported module. Function
// calls and imports switch it (see callFunction, loadModule).
var currentFile string

// fatalError is what fatalf panics with. Run recovers exactly this type at
// the top of a program's evaluation and turns it into a returned error —
// any other panic (a genuine interpreter bug, not a deliberate runtime
// error) is left to propagate and crash normally, so a real bug is never
// silently swallowed.
//
// Every fatalError also has a kind (file, number, math, ...), which is
// what lets a safe block handle some errors and let others through.
type fatalError struct {
	msg  string // the full message, "line 3: division by zero"
	text string // the message without its line, "division by zero"
	kind string
	line int
	file string // "" for the main script
	// parse is set for a parse error in an imported module: like one in
	// the main script, no safe block handles it.
	parse bool
}

func (e fatalError) Error() string { return e.msg }

// The kinds of runtime error, as written in handle [file, math] error .
// Anything not given a kind by fatalKind is kindType: a value of the
// wrong type, or the wrong number of arguments.
const (
	kindFile   = "file"   // missing file, can't write, end of input
	kindNumber = "number" // text that isn't a number: change "abc" to integer
	kindMath   = "math"   // division by zero, overflow, sqrt of a negative
	kindIndex  = "index"  // index out of range, pop or min of an empty collection
	kindKey    = "key"    // map key not found
	kindName   = "name"   // undefined variable, function, method, module or field
	kindType   = "type"   // the wrong kind of value or number of arguments
	kindJSON   = "json"   // text that isn't valid JSON
	kindDate   = "date"   // text that isn't a date, or a date that doesn't exist
	kindHTTP   = "http"   // a web request that failed, or got a 4xx/5xx status
	kindSQL    = "sql"    // a bad query, or a database problem
	kindCSV    = "csv"    // a .csv or .tsv file that isn't well formed
	kindCustom = "custom" // the program's own, from fail "..."
)

// fatalf reports a runtime error and unwinds the current evaluation via
// panic/recover (see Run) rather than calling os.Exit directly — that's
// what lets tests exercise an error path (e.g. division by zero, a failed
// change conversion) without killing the test binary, while the CLI
// (cmd/turtle/main.go) still exits 1 with this exact message, since it's
// the only thing at the top that doesn't recover.
func fatalf(format string, args ...interface{}) {
	fatalKind(kindType, format, args...)
}

// fatalKind is fatalf for an error of a particular kind.
func fatalKind(kind, format string, args ...interface{}) {
	text := fmt.Sprintf(format, args...)
	panic(fatalError{msg: place(currentFile, currentLine) + text, text: text, kind: kind, line: currentLine, file: currentFile})
}

// place is the "where" an error message starts with: "line 3: " in the
// main script, "lib/utils.t line 3: " in an imported module.
func place(file string, line int) string {
	switch {
	case line <= 0 && file == "":
		return ""
	case line <= 0:
		return file + ": "
	case file == "":
		return fmt.Sprintf("line %d: ", line)
	}
	return fmt.Sprintf("%s line %d: ", file, line)
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
	// Databases the program didn't close are closed when it ends, however
	// it ends: an unfinished transaction is rolled back and files are let
	// go (Windows can't delete or reopen a file a program still holds).
	defer it.closeDatabases()
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

// evalSafe runs a safe block. An error of a kind its handle line lists
// (any kind, for handle [] e .) stops the block, is stored in the
// handle's variable, and runs the handle code; with no error the handle
// code is skipped and the variable is none. Other errors, and system's
// exit[code], go on up untouched. A return, break or continue inside
// either part still works.
func (it *Interpreter) evalSafe(s *ast.SafeStatement, env *object.Environment) ExecResult {
	res, fe, failed := it.evalProtected(s, env)
	if !failed {
		env.Set(s.Name, object.NoneValue)
		return res
	}
	file := fe.file
	if file == "" {
		file = it.Script
	}
	env.Set(s.Name, &object.Error{Kind: fe.kind, File: file, InModule: fe.file != "", Line: fe.line, Message: fe.text})
	// Outside evalProtected, so an error in the handle code isn't handled
	// by its own safe block.
	return it.evalBlock(s.Handler, env)
}

// evalProtected runs a safe block's body, recovering an error of a kind
// its handle line lists.
func (it *Interpreter) evalProtected(s *ast.SafeStatement, env *object.Environment) (res ExecResult, fe fatalError, failed bool) {
	defer func() {
		r := recover()
		if r == nil {
			return
		}
		e, ok := r.(fatalError)
		if !ok || e.parse || (len(s.Kinds) > 0 && !slices.Contains(s.Kinds, e.kind)) {
			panic(r)
		}
		res, fe, failed = noneResult, e, true
	}()
	return it.evalBlock(s.Body, env), fatalError{}, false
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
		fmt.Print(it.evalExpression(s.Prompt, env).Inspect())
		if !it.stdin.Scan() {
			// EOF (or a read error) on stdin: there's nothing left to
			// read, ever, so returning "" here would make any loop that
			// re-prompts on bad input (e.g. retrying invalid numeric
			// input) spin forever re-reading empty strings instead of
			// terminating.
			fatalKind(kindFile, "unexpected end of input reading %q", s.Name)
		}
		env.Set(s.Name, &object.String{Value: it.stdin.Text()})
		return noneResult

	case *ast.ShowStatement:
		if s.Stderr {
			requireModule(env, "system", "warn")
		}
		out := ""
		for _, e := range s.Expressions {
			out += it.evalExpression(e, env).Inspect()
		}
		if s.Stderr {
			fmt.Fprintln(os.Stderr, out)
		} else {
			fmt.Println(out)
		}
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
		if env.IsTopLevel() {
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
		if env.IsTopLevel() {
			env.DefineFunction(fn)
		} else {
			env.Set(s.Name, fn)
		}
		return noneResult

	case *ast.SafeStatement:
		return it.evalSafe(s, env)

	case *ast.FailStatement:
		val := it.evalExpression(s.Value, env)
		fatalKind(kindCustom, "%s", val.Inspect())
		return noneResult

	case *ast.FieldAssignStatement:
		obj := it.evalExpression(s.Target.Object, env)
		if _, ok := obj.(*object.Error); ok {
			fatalf("an error's parts can't be changed (%s of an error is read-only)", s.Target.Field)
		}
		if _, ok := obj.(*object.Date); ok {
			fatalf("a date's parts can't be changed (%s of a date is read-only); make a new one with add_time or make_date", s.Target.Field)
		}
		a, i := fieldOf(obj, s.Target.Field)
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

// evalLoop runs a loop. The C-style form's counter lives in a loop scope
// of its own (object.NewLoopEnvironment), so it never touches a variable
// of the same name outside — two nested loops both using "i" (a real
// pattern in historical Turtle scripts) don't clobber each other — and a
// closure made in the loop can still read it after the loop ends. Every
// other assignment in the body goes to the enclosing scope as usual.
func (it *Interpreter) evalLoop(s *ast.LoopStatement, env *object.Environment) ExecResult {
	if s.Kind == ast.LoopEach {
		return it.evalEach(s, env)
	}
	if s.Kind == ast.LoopCStyle && s.Init != nil {
		loopEnv := object.NewLoopEnvironment(env)
		if a, ok := s.Init.(*ast.AssignStatement); ok {
			loopEnv.Define(a.Name, it.evalExpression(a.Value, env))
		} else {
			it.evalStatement(s.Init, loopEnv)
		}
		env = loopEnv
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
// loop can't make it skip or repeat. Each pass gets its own loop scope
// holding just the loop names (see object.NewLoopEnvironment): outside
// variables of the same name are untouched, and a closure made in a pass
// keeps that pass's values.
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
			firsts = append(firsts, c.KeyOf(k))
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

	for i := range seconds {
		pass := object.NewLoopEnvironment(env)
		if len(s.Vars) == 2 {
			pass.Define(s.Vars[0], firsts[i])
			pass.Define(s.Vars[1], seconds[i])
		} else {
			pass.Define(s.Vars[0], seconds[i])
		}
		res := it.evalBlock(s.Body, pass)
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
	case *object.List:
		return len(v.Elements) > 0
	case *object.Set:
		return len(v.Elements) > 0
	case *object.Map:
		return len(v.Keys) > 0
	case *object.None:
		return false
	default:
		return obj != nil
	}
}
