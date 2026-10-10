// Package repl is Turtle's interactive prompt: type Turtle a line at a
// time, see results at once, with the code colored as it's typed.
//
//	$ turtle
//	>>> nums = list [3, 1, 2]
//	>>> nums at length * 2
//	6
//	>>> def double[x]
//	...     return x * 2
//	... def [end]
//	>>> double[21]
//	42
//
// A lone expression or call shows its value. A block (def, if, loop,
// safe ...) waits for its end. Errors are shown, and the session goes on.
package repl

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strings"

	"Turtle/ast"
	"Turtle/evaluator"
	"Turtle/lexer"
	"Turtle/object"
	"Turtle/parser"
	"Turtle/syntax"
	"Turtle/token"
)

const (
	prompt     = ">>> "
	contPrompt = "... "
	historyMax = 1000
)

// Session is one REPL: its interpreter, kept between entries, and what
// it remembers (the libraries imported, the entries run).
type Session struct {
	it      *evaluator.Interpreter
	out     io.Writer
	color   bool
	words   syntax.Words
	libs    map[string]bool // libraries imported: their words color, and parse
	entries []string        // entries that ran, for save
	dir     string
	quit    bool
	// theories are the theories entered (or imported) so far: later
	// entries read their phrases.
	theories parser.Theories
}

// NewSession makes a session whose files and imports resolve from dir.
func NewSession(dir string, out io.Writer, color bool) *Session {
	s := &Session{it: evaluator.New(dir), out: out, color: color, libs: map[string]bool{}, dir: dir}
	s.it.Script = ""
	s.words = syntax.Words{Context: map[string]bool{}, Builtin: map[string]bool{}, Theories: map[string]bool{}, Types: map[string]bool{}}
	for _, f := range evaluator.LibraryFunctions() {
		s.words.Builtin[f] = true
	}
	return s
}

// Close ends the session (databases still open are closed).
func (s *Session) Close() { s.it.Close() }

// Quit reports whether the user asked to leave (quit, or exit[]).
func (s *Session) Quit() bool { return s.quit }

func (s *Session) paint(code, text string) string {
	if !s.color {
		return text
	}
	return code + text + reset
}

// Eval runs one entry and writes its result or error. It reports false if
// the entry was a request to leave.
func (s *Session) Eval(src string) {
	trimmed := strings.TrimSpace(src)
	if trimmed == "" {
		return
	}
	if s.command(trimmed) {
		return
	}
	program, perr := s.parse(src)
	if perr == nil {
		v, err := s.it.RunEntry(program)
		s.noteImports(program)
		if s.report(err) {
			s.entries = append(s.entries, src)
			if v != nil {
				s.show(v)
			}
		}
		return
	}
	// Not statements: maybe a lone expression ("1 + 2", "nums").
	p := s.parser(src)
	e := p.ParseExpressionOnly()
	if e != nil && len(p.Errors()) == 0 {
		v, err := s.it.RunExpression(e)
		if s.report(err) {
			s.show(v)
		}
		return
	}
	// The first error is the real one; the parser trips over what
	// follows it, and those would only confuse.
	msg := perr[0]
	if e == nil && len(p.Errors()) > 0 && looksLikeExpression(src) {
		msg = p.Errors()[0] // "a at give x" reads as an expression gone wrong
	}
	fmt.Fprintln(s.out, s.paint(errorColor, "error: "+stripLine(msg, src)))
}

func (s *Session) parser(src string) *parser.Parser {
	p := parser.New(lexer.New(src))
	p.ModuleDir = s.dir
	p.UseTheories(s.theories)
	for lib := range s.libs {
		p.Enable(lib)
	}
	return p
}

func (s *Session) parse(src string) (*ast.Program, []string) {
	p := s.parser(src)
	program := p.ParseProgram()
	if errs := p.Errors(); len(errs) > 0 {
		return nil, errs
	}
	s.theories = p.Theories() // the theories entered so far, for the next entry
	for name := range s.theories {
		s.words.Theories[name] = true // their words color from now on
	}
	for _, st := range program.Statements {
		if a, ok := st.(*ast.AssembleStatement); ok {
			s.words.Types[a.Name] = true
		}
	}
	return program, nil
}

// noteImports remembers the libraries an entry imported: their words
// (random, check, log ...) parse and color from now on.
func (s *Session) noteImports(program *ast.Program) {
	for _, st := range program.Statements {
		if im, ok := st.(*ast.ImportStatement); ok {
			s.libs[im.Path] = true
			for _, w := range syntax.LibraryWords[im.Path] {
				s.words.Context[w] = true
			}
		}
	}
}

// show writes a value the way a program would see it: text in quotes.
func (s *Session) show(v object.Object) {
	if _, isNone := v.(*object.None); isNone {
		return
	}
	text := object.Shown(v)
	if m, ok := v.(*object.Matrix); ok {
		text = m.Inspect() // a matrix on its own shows as a grid, as show prints it
	}
	fmt.Fprintln(s.out, s.paint(resultColor, text))
}

// report writes an error, if there is one, and reports whether the entry
// ran without one.
func (s *Session) report(err error) bool {
	if err == nil {
		return true
	}
	if ex, ok := err.(evaluator.ExitRequest); ok {
		s.quit = true
		_ = ex
		return true
	}
	if err == evaluator.ErrInterrupted {
		fmt.Fprintln(s.out, s.paint(errorColor, "stopped (Ctrl-C)"))
		return false
	}
	kind, text, line, file, ok := evaluator.ErrorDetails(err)
	if !ok {
		fmt.Fprintln(s.out, s.paint(errorColor, "error: "+err.Error()))
		return false
	}
	where := ""
	if file != "" {
		where = fmt.Sprintf(" in %s line %d", file, line)
	} else if line > 1 {
		where = fmt.Sprintf(" on line %d", line)
	}
	fmt.Fprintln(s.out, s.paint(errorColor, fmt.Sprintf("error (%s)%s: %s", kind, where, text)))
	return false
}

// looksLikeExpression: a one-line entry that doesn't start with a
// statement word (an assignment, show, def ...), so the expression
// parser's complaint is the useful one.
func looksLikeExpression(src string) bool {
	if strings.Contains(strings.TrimSpace(src), "\n") {
		return false
	}
	l := lexer.New(src)
	first, second := l.NextToken(), l.NextToken()
	if token.IsKeyword(first.Type) || second.Type == token.ASSIGN || second.Type == token.IS {
		return false
	}
	return true
}

// stripLine drops "line 1: " from a one-line entry's parse error.
func stripLine(msg, src string) string {
	if !strings.Contains(strings.TrimRight(src, "\n"), "\n") {
		return strings.TrimPrefix(msg, "line 1: ")
	}
	return msg
}

// ---- commands ----

const helpText = `Type Turtle, and press Enter to run it. A lone expression or call
shows its value; a block (def, if, [loop], safe ...) waits for its end.

Commands (a line with just the word):
  help              this
  help linear       a library: what it's for and its functions
  help reshape      one function or method, with an example
  help m            what your variable m is and what you can do with it
  help if           a keyword; help keywords lists them all
  quit              leave (or Ctrl-D)
  clear             clear the screen (or Ctrl-L)
  names             your variables and functions
  load file.turtle     run a file into this session
  save file.turtle     write what you've run in this session to a file

Keys: arrows move and go through history, Home/End (Ctrl-A/Ctrl-E),
Ctrl-K/Ctrl-U cut to the end/start, Ctrl-W cuts a word, Tab indents,
Ctrl-C cancels the line or stops running code.

In a program: help["linear"], help[m], stdlib[], version.`

// command runs a REPL command, if the entry is one. A word that's also
// one of your variables or functions is yours, not a command.
func (s *Session) command(line string) bool {
	word, arg, _ := strings.Cut(line, " ")
	arg = strings.TrimSpace(arg)
	vars, funcs := s.it.GlobalNames()
	for _, n := range append(vars, funcs...) {
		if n == word {
			return false
		}
	}
	switch {
	case (word == "help" || word == "quit" || word == "exit" || word == "clear" || word == "names") && arg == "":
	case (word == "load" || word == "save") && arg != "" && !strings.ContainsAny(arg, "[]=\""):
	case word == "help" && arg != "" && !strings.ContainsAny(arg, "[]=\" "):
	default:
		return false
	}
	switch word {
	case "help":
		if arg == "" {
			fmt.Fprintln(s.out, helpText)
			return true
		}
		text, err := s.it.HelpText(arg)
		if err != nil {
			fmt.Fprintln(s.out, s.paint(errorColor, err.Error()))
			return true
		}
		fmt.Fprintln(s.out, text)
	case "quit", "exit":
		s.quit = true
	case "clear":
		fmt.Fprint(s.out, "\x1b[2J\x1b[H")
	case "names":
		vars, funcs := s.it.GlobalNames()
		if len(vars)+len(funcs) == 0 {
			fmt.Fprintln(s.out, "nothing yet")
		}
		if len(vars) > 0 {
			fmt.Fprintln(s.out, "variables: "+strings.Join(vars, ", "))
		}
		if len(funcs) > 0 {
			fmt.Fprintln(s.out, "functions: "+strings.Join(funcs, ", "))
		}
	case "load":
		data, err := os.ReadFile(s.path(arg))
		if err != nil {
			fmt.Fprintln(s.out, s.paint(errorColor, "load: "+err.Error()))
			return true
		}
		s.Eval(string(data))
	case "save":
		text := strings.Join(s.entries, "\n")
		if text != "" {
			text += "\n"
		}
		if err := os.WriteFile(s.path(arg), []byte(text), 0o644); err != nil {
			fmt.Fprintln(s.out, s.paint(errorColor, "save: "+err.Error()))
			return true
		}
		fmt.Fprintf(s.out, "saved %d %s to %s\n", len(s.entries), plural(len(s.entries), "entry", "entries"), arg)
	}
	return true
}

func plural(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}

func (s *Session) path(p string) string {
	if filepath.IsAbs(p) {
		return p
	}
	return filepath.Join(s.dir, p)
}

// ---- the interactive loop ----

// Run is the REPL on the terminal (stdin and stdout). It returns the exit
// code.
func Run(version string) int {
	dir, _ := os.Getwd()
	color := os.Getenv("NO_COLOR") == "" && os.Getenv("TERM") != "dumb" && enableColors(1)
	s := NewSession(dir, os.Stdout, color)
	defer s.Close()
	fmt.Printf("Turtle %s — type help for help, quit to leave\n", version)

	history := loadHistory()
	defer func() { saveHistory(history) }() // the history as it is at the end

	// Ctrl-C while code runs stops the code, not the REPL.
	interrupts := make(chan os.Signal, 1)
	signal.Notify(interrupts, os.Interrupt)
	defer signal.Stop(interrupts)
	go func() {
		for range interrupts {
			s.it.Interrupt()
		}
	}()

	in := bufio.NewReader(os.Stdin)
	for !s.quit {
		entry, ok := s.readEntry(in, &history)
		if !ok {
			fmt.Println()
			break
		}
		s.Eval(entry)
	}
	return 0
}

// readEntry reads one entry: a line, and more lines while a block is
// open. An empty line ends an unfinished entry anyway (it then shows the
// parse error). false means the input ended (Ctrl-D).
func (s *Session) readEntry(in *bufio.Reader, history *[]string) (string, bool) {
	var lines []string
	for {
		p := prompt
		start := ""
		if len(lines) > 0 {
			p = contPrompt
			start = strings.Repeat(" ", indentFor(strings.Join(lines, "\n")))
		}
		text, ok := s.readLine(in, p, start, history)
		if !ok {
			if len(lines) > 0 {
				return "", true // Ctrl-C in a block: drop the block
			}
			return "", false
		}
		if text == "\x03" { // canceled
			return "", true
		}
		if strings.TrimSpace(text) == "" && len(lines) > 0 {
			return strings.Join(lines, "\n"), true
		}
		lines = append(lines, text)
		if strings.TrimSpace(text) != "" {
			*history = append(*history, text)
		}
		entry := strings.Join(lines, "\n")
		if !needsMore(entry) {
			return entry, true
		}
	}
}

// readLine reads one line with editing, colors and history. It returns
// "\x03" when the line is canceled (Ctrl-C), and false at the end of
// input (Ctrl-D on an empty line).
func (s *Session) readLine(in *bufio.Reader, p, start string, history *[]string) (string, bool) {
	state, err := makeRaw(0)
	if err != nil {
		// Not a terminal we can drive: plain lines.
		fmt.Print(p + start)
		text, err := in.ReadString('\n')
		if err != nil && text == "" {
			return "", false
		}
		return start + strings.TrimRight(text, "\r\n"), true
	}
	defer restore(0, state)

	l := &line{}
	l.set(start)
	hpos := len(*history)
	draft := ""
	row := 0
	width := termWidth(1)
	shownPrompt := s.paint(promptColor, p)
	redraw := func() {
		colored := l.String()
		if s.color {
			colored = colorize(colored, s.words)
		}
		out, r := render(shownPrompt, len(p), l.String(), colored, l.pos, width, row)
		row = r
		fmt.Print(out)
	}
	redraw()
	buf := make([]byte, 256)
	for {
		n, err := in.Read(buf)
		if err != nil {
			return "", false
		}
		for _, k := range decodeKeys(buf[:n]) {
			switch k.kind {
			case keyEnter:
				l.pos = len(l.buf)
				redraw()
				fmt.Print("\r\n")
				return l.String(), true
			case keyCancel:
				l.pos = len(l.buf)
				redraw()
				fmt.Print(s.paint(promptColor, "^C") + "\r\n")
				return "\x03", true
			case keyEOF:
				if len(l.buf) == 0 {
					fmt.Print("\r\n")
					return "", false
				}
				l.edit(key{kind: keyDelete})
			case keyClear:
				fmt.Print("\x1b[2J\x1b[H")
				row = 0
			case keyUp:
				if hpos > 0 {
					if hpos == len(*history) {
						draft = l.String()
					}
					hpos--
					l.set((*history)[hpos])
				}
			case keyDown:
				if hpos < len(*history) {
					hpos++
					if hpos == len(*history) {
						l.set(draft)
					} else {
						l.set((*history)[hpos])
					}
				}
			default:
				l.edit(k)
			}
		}
		width = termWidth(1)
		redraw()
	}
}

// ---- history: ~/.turtle_history ----

func historyPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".turtle_history")
}

func loadHistory() []string {
	data, err := os.ReadFile(historyPath())
	if err != nil {
		return nil
	}
	lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
	if len(lines) > historyMax {
		lines = lines[len(lines)-historyMax:]
	}
	return lines
}

func saveHistory(h []string) {
	path := historyPath()
	if path == "" {
		return
	}
	if len(h) > historyMax {
		h = h[len(h)-historyMax:]
	}
	os.WriteFile(path, []byte(strings.Join(h, "\n")+"\n"), 0o600)
}

// IsTerminal reports whether turtle's input is a terminal (so turtle with
// no file opens the REPL, rather than running piped-in code).
func IsTerminal() bool { return isTerminal(0) }
