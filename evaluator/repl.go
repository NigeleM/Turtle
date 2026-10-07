package evaluator

import (
	"errors"
	"sort"

	"Turtle/ast"
	"Turtle/object"
)

// For the REPL (package repl): running one entry at a time in a lasting
// session, and the facts about errors and names it shows.

// interruptRequest stops running code when the REPL's Ctrl-C asks; no
// safe block handles it.
type interruptRequest struct{}

// ErrInterrupted is what RunEntry gives back after Interrupt.
var ErrInterrupted = errors.New("interrupted")

// Interrupt stops the code now running, at its next statement. It's safe
// to call from another goroutine.
func (it *Interpreter) Interrupt() { it.interrupt.Store(true) }

// RunEntry runs one REPL entry in the session's global scope. Unlike Run
// it keeps databases open for the next entry. For a lone call
// ("double[21]") it gives back the call's value, to show.
func (it *Interpreter) RunEntry(program *ast.Program) (shown object.Object, err error) {
	defer it.recoverEntry(&err)
	it.interrupt.Store(false)
	if len(program.Statements) == 1 {
		switch st := program.Statements[0].(type) {
		case *ast.CallStatement:
			currentLine = st.Line()
			return it.evalExpression(st.Call, it.Global), nil
		case *ast.ExpressionStatement:
			currentLine = st.Line()
			return it.evalExpression(st.Expression, it.Global), nil
		}
	}
	it.evalStatements(program.Statements, it.Global)
	return nil, nil
}

// RunExpression works out one expression in the session's global scope.
func (it *Interpreter) RunExpression(e ast.Expression) (v object.Object, err error) {
	defer it.recoverEntry(&err)
	it.interrupt.Store(false)
	currentLine = 1
	return it.evalExpression(e, it.Global), nil
}

func (it *Interpreter) recoverEntry(err *error) {
	if r := recover(); r != nil {
		switch e := r.(type) {
		case fatalError:
			*err = e
		case ExitRequest:
			*err = e
		case interruptRequest:
			*err = ErrInterrupted
		default:
			panic(r)
		}
	}
	it.depth = 0
	currentFile = ""
}

// Close ends the session: databases still open are closed.
func (it *Interpreter) Close() { it.closeDatabases() }

// GlobalNames lists the session's variables and functions, sorted,
// leaving out the settings libraries make (tablerows, seed, ...).
func (it *Interpreter) GlobalNames() (vars, funcs []string) {
	allVars, funcs := it.Global.Names()
	for _, v := range allVars {
		if !librarySettings[v] {
			vars = append(vars, v)
		}
	}
	return vars, funcs
}

var librarySettings = map[string]bool{
	tableRowsName: true, seedName: true, suiteName: true, benchmarkName: true, runsName: true,
	benchtimeName: true, casesName: true, logLevelName: true, logFileName: true, logConsoleName: true,
	logTimeName: true, logPartsName: true, logFormatName: true, logMaxSizeName: true, logKeepName: true,
	outputFileName: true,
}

// ErrorDetails takes apart an error from RunEntry: its kind ("math"), its
// message without the line, the line and the file ("" for the session).
func ErrorDetails(err error) (kind, text string, line int, file string, ok bool) {
	var fe fatalError
	if errors.As(err, &fe) {
		return fe.kind, fe.text, fe.line, fe.file, true
	}
	return "", "", 0, "", false
}

// LibraryFunctions are the builtin libraries' function names, for the
// REPL to color as builtins.
func LibraryFunctions() []string {
	var out []string
	for _, m := range builtinModules {
		out = append(out, m.Funcs...)
	}
	sort.Strings(out)
	return out
}
