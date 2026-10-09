package evaluator

import (
	"fmt"
	"strings"

	"Turtle/ast"
)

// turtle trace: each line is written to Trace (stderr) as it runs, and an
// assignment adds the value it gave the name:
//
//	line 1  total = 0                     total = 0
//	line 2  [loop][x in nums]
//	line 3      total = total + x         total = 5
//
// A line from an imported file is named with its file (utils.turtle:4).
// Function defs are left out: they're defined before the file runs.

// TraceSource gives the tracer the text of a file ("" for the main one),
// to show its lines.
func (it *Interpreter) TraceSource(file, src string) {
	if it.traceSrc == nil {
		it.traceSrc = map[string][]string{}
	}
	it.traceSrc[file] = strings.Split(strings.ReplaceAll(src, "\r\n", "\n"), "\n")
}

// traceLine is an assignment's line, written but waiting for its value.
type traceLine struct {
	where string // "line 6"
	width int    // how much of the line is written
	open  bool   // nothing else written since: the value goes on it
}

const traceWidth = 40

// traceStart writes the statement's line, unless it's a def. For an
// assignment the line stays open for its value (traceAssign).
func (it *Interpreter) traceStart(stmt ast.Statement) *traceLine {
	if _, ok := stmt.(*ast.FunctionDefStatement); ok {
		return nil
	}
	it.traceClose()
	line := stmt.Line()
	lines := it.traceSrc[currentFile]
	where := fmt.Sprintf("line %-*d", len(fmt.Sprint(len(lines))), line)
	if currentFile != "" {
		where = fmt.Sprintf("%s:%d", currentFile, line)
	}
	text := where
	if line >= 1 && line <= len(lines) {
		text += "  " + strings.TrimRight(lines[line-1], " \t")
	}
	if _, ok := stmt.(*ast.AssignStatement); !ok {
		fmt.Fprintln(it.Trace, text)
		return nil
	}
	fmt.Fprint(it.Trace, text)
	t := &traceLine{where: where, width: len(text), open: true}
	it.traceOpen = t
	return t
}

// traceClose ends an open line: something else is about to be traced (a
// line of a function the assignment calls).
func (it *Interpreter) traceClose() {
	if it.traceOpen != nil {
		fmt.Fprintln(it.Trace)
		it.traceOpen.open = false
		it.traceOpen = nil
	}
}

// traceAssign finishes an assignment's line with the value it gave, or,
// when other lines came in between, says it on a line of its own.
func (it *Interpreter) traceAssign(t *traceLine, name, value string) {
	if t == nil {
		return
	}
	if len(value) > 60 {
		value = value[:57] + "..."
	}
	if t.open {
		fmt.Fprintf(it.Trace, "%s  %s = %s\n", strings.Repeat(" ", max(traceWidth-t.width, 0)), name, value)
		it.traceOpen = nil
		return
	}
	it.traceClose()
	fmt.Fprintf(it.Trace, "%-*s  %s = %s\n", traceWidth, t.where+"  ...", name, value)
}

// tracePass writes a loop pass's line under turtle trace (and in a
// diagnose block): its number, and the loop's names.
func (it *Interpreter) tracePass(n int, names string) {
	it.traceClose()
	if names != "" {
		names = ": " + names
	}
	fmt.Fprintf(it.Trace, "%*s  pass %d%s\n", traceWidth, "", n, names)
}
