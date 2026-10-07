package evaluator

import (
	"bufio"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"

	"Turtle/ast"
	"Turtle/lexer"
	"Turtle/object"
	"Turtle/parser"
)

// turtle debug: run a program a line at a time. It stops before the first
// line and waits for a command; anything that isn't a command is Turtle,
// run right there (show x ., x = 5, total + 1):
//
//	→ line 4      total = total + x
//	(debug) p total
//	5
//
// Its messages go to standard error; commands come from standard input,
// as the program's ? prompts do.

const debugHelp = `Commands:
  Enter or s   step: run this line, stop at the next (into functions)
  n            next: run this line and any functions it calls
  o            out: run to the end of this function
  c            continue to the next breakpoint (or the end)
  b 12         stop at line 12 (b utils.trt:4 in an imported file); b alone lists them
  d 12         remove that breakpoint
  p <value>    show a value: p total, p nums at len
  v            the variables here, then the globals
  w            where: the functions called, outermost first
  l            the lines around this one
  q            quit the program
  h            this help
Anything else is run as Turtle, here: x = 5, show x .`

// Debugger holds turtle debug's state.
type Debugger struct {
	out    io.Writer
	in     *bufio.Scanner
	src    map[string][]string // each file's lines ("" for the main one)
	breaks map[string]bool     // "file:line"
	mode   string              // "step", "next", "out", "continue"
	depth  int                 // the call depth "next" and "out" were given at
	frames []debugFrame        // functions called, outermost first
	quiet  bool                // the input ended: run on without stopping
}

type debugFrame struct {
	name string
	file string
	line int // the line it was called from
}

// Debug turns on turtle debug for this interpreter: messages go to out,
// commands are read as the program's ? input is.
func (it *Interpreter) Debug(out io.Writer, src string) {
	it.debug = &Debugger{out: out, in: it.stdin, src: map[string][]string{}, breaks: map[string]bool{}, mode: "step"}
	it.debug.src[""] = splitLines(src)
	fmt.Fprintln(out, "turtle debug: stopped before the first line. Enter steps; h shows the commands.")
}

func splitLines(src string) []string {
	return strings.Split(strings.ReplaceAll(src, "\r\n", "\n"), "\n")
}

// debugLeave ends a function's place in the list of those being run
// (callFunction adds it).
func (it *Interpreter) debugLeave() {
	it.debug.frames = it.debug.frames[:len(it.debug.frames)-1]
}

// debugStop runs before each statement: it stops when the mode or a
// breakpoint says so, and takes commands until one runs the program on.
func (it *Interpreter) debugStop(stmt ast.Statement, env *object.Environment) {
	d := it.debug
	if d.quiet {
		return
	}
	if _, ok := stmt.(*ast.FunctionDefStatement); ok {
		return
	}
	line := stmt.Line()
	depth := len(d.frames)
	stop := d.breaks[fmt.Sprintf("%s:%d", currentFile, line)]
	switch d.mode {
	case "step":
		stop = true
	case "next":
		stop = stop || depth <= d.depth
	case "out":
		stop = stop || depth < d.depth
	}
	if !stop {
		return
	}
	d.show(currentFile, line)
	for {
		fmt.Fprint(d.out, "(debug) ")
		if !d.in.Scan() {
			fmt.Fprintln(d.out, "\n(no more input: running to the end)")
			d.quiet = true
			return
		}
		cmd := strings.TrimSpace(d.in.Text())
		word, rest, _ := strings.Cut(cmd, " ")
		rest = strings.TrimSpace(rest)
		// A command is a word alone (b, d and p take what follows);
		// anything else is Turtle: "n = 100" sets n.
		switch {
		case word == "b" || word == "break" || word == "d" || word == "delete":
		case (word == "p" || word == "print") && rest != "":
		case rest == "":
		default:
			word = "turtle"
		}
		switch word {
		case "", "s", "step":
			d.mode = "step"
			return
		case "n", "next":
			d.mode, d.depth = "next", depth
			return
		case "o", "out":
			d.mode, d.depth = "out", depth
			if depth == 0 {
				d.mode = "continue"
			}
			return
		case "c", "continue":
			d.mode = "continue"
			return
		case "q", "quit":
			panic(ExitRequest{Code: 1})
		case "h", "help", "?":
			fmt.Fprintln(d.out, debugHelp)
		case "b", "break":
			d.breakpoint(rest, true)
		case "d", "delete":
			d.breakpoint(rest, false)
		case "l", "list":
			d.list(currentFile, line)
		case "w", "where":
			d.where(line)
		case "v", "vars":
			d.vars(env)
		case "p", "print":
			it.debugRun(rest, env)
		default:
			if cmd != "" {
				it.debugRun(cmd, env)
			}
		}
	}
}

// show writes where the program stopped: → line 4   total = total + x
func (d *Debugger) show(file string, line int) {
	where := fmt.Sprintf("line %d", line)
	if file != "" {
		where = fmt.Sprintf("%s:%d", file, line)
	}
	text := ""
	if lines := d.src[file]; line >= 1 && line <= len(lines) {
		text = strings.TrimRight(lines[line-1], " \t")
	}
	fmt.Fprintf(d.out, "→ %s  %s\n", where, text)
}

func (d *Debugger) breakpoint(arg string, add bool) {
	if arg == "" {
		if !add {
			fmt.Fprintln(d.out, "say which: d 12")
			return
		}
		if len(d.breaks) == 0 {
			fmt.Fprintln(d.out, "no breakpoints: b 12 adds one")
			return
		}
		var list []string
		for b := range d.breaks {
			list = append(list, strings.TrimPrefix(b, ":"))
		}
		sort.Strings(list)
		fmt.Fprintln(d.out, "breakpoints: "+strings.Join(list, ", "))
		return
	}
	file, num := "", arg
	if i := strings.LastIndex(arg, ":"); i >= 0 {
		file, num = arg[:i], arg[i+1:]
	}
	n, err := strconv.Atoi(num)
	if err != nil || n < 1 {
		fmt.Fprintf(d.out, "%q isn't a line: b 12, or b utils.trt:4\n", arg)
		return
	}
	key := fmt.Sprintf("%s:%d", file, n)
	if add {
		d.breaks[key] = true
		fmt.Fprintf(d.out, "will stop at %s\n", arg)
	} else {
		delete(d.breaks, key)
		fmt.Fprintf(d.out, "won't stop at %s\n", arg)
	}
}

func (d *Debugger) list(file string, line int) {
	lines := d.src[file]
	for n := max(line-3, 1); n <= min(line+3, len(lines)); n++ {
		mark := "  "
		if n == line {
			mark = "→ "
		}
		fmt.Fprintf(d.out, "%s%4d  %s\n", mark, n, strings.TrimRight(lines[n-1], " \t"))
	}
}

func (d *Debugger) where(line int) {
	place := func(file string, line int) string {
		if file == "" {
			return fmt.Sprintf("line %d", line)
		}
		return fmt.Sprintf("%s:%d", file, line)
	}
	if len(d.frames) == 0 {
		fmt.Fprintf(d.out, "the main code, %s\n", place(currentFile, line))
		return
	}
	fmt.Fprintf(d.out, "the main code, %s\n", place(d.frames[0].file, d.frames[0].line))
	for i, f := range d.frames {
		at := line
		file := currentFile
		if i+1 < len(d.frames) {
			at, file = d.frames[i+1].line, d.frames[i+1].file
		}
		fmt.Fprintf(d.out, "%s└ %s[], %s\n", strings.Repeat("  ", i), f.name, place(file, at))
	}
}

// vars lists the names in scope, innermost first; a name hidden by an
// inner one isn't repeated.
func (d *Debugger) vars(env *object.Environment) {
	seen := map[string]bool{}
	for s := env; s != nil; s = s.Outer() {
		names, _ := s.Names()
		label := "here"
		if s.Outer() == nil {
			label = "globals"
		}
		var lines []string
		for _, n := range names {
			if seen[n] {
				continue
			}
			seen[n] = true
			v, _ := s.Get(n)
			text := v.Inspect()
			if len(text) > 70 {
				text = text[:67] + "..."
			}
			lines = append(lines, fmt.Sprintf("  %s = %s", n, text))
		}
		if len(lines) > 0 {
			fmt.Fprintf(d.out, "%s:\n%s\n", label, strings.Join(lines, "\n"))
		}
	}
	if len(seen) == 0 {
		fmt.Fprintln(d.out, "no variables yet")
	}
}

// debugRun runs Turtle typed at the prompt, in the stopped code's scope;
// a lone value is shown. An error is shown, not fatal.
func (it *Interpreter) debugRun(src string, env *object.Environment) {
	d := it.debug
	p := parser.New(lexer.New(src))
	program := p.ParseProgram()
	var value ast.Expression // a value on its own, to show
	if errs := p.ErrorList(); len(errs) > 0 {
		v := parser.New(lexer.New("debugvalue = " + src))
		vp := v.ParseProgram()
		a, ok := (*ast.AssignStatement)(nil), false
		if len(v.ErrorList()) == 0 && len(vp.Statements) == 1 {
			a, ok = vp.Statements[0].(*ast.AssignStatement)
		}
		if !ok {
			fmt.Fprintf(d.out, "%s (h shows the commands)\n", errs[0].Msg)
			return
		}
		value = a.Value
	} else if len(program.Statements) == 1 {
		if e, ok := program.Statements[0].(*ast.ExpressionStatement); ok {
			value = e.Expression
		}
	}
	saved, savedLine, savedMode := it.debug, currentLine, d.mode
	it.debug = nil // what's typed runs without stopping
	defer func() {
		it.debug, currentLine = saved, savedLine
		d.mode = savedMode
		if r := recover(); r != nil {
			fe, ok := r.(fatalError)
			if !ok {
				panic(r)
			}
			fmt.Fprintf(d.out, "error (%s): %s\n", fe.kind, fe.text)
		}
	}()
	if value != nil {
		fmt.Fprintln(d.out, it.evalExpression(value, env).Inspect())
		return
	}
	it.evalStatements(program.Statements, env)
}
