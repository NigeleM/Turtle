// Package format lays Turtle code out one way: turtle fmt and an
// editor's Format Document.
package format

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"Turtle/lexer"
	"Turtle/parser"
	"Turtle/syntax"
	"Turtle/token"
)

// Format lays Turtle code out: each block's lines indented four spaces deeper than
// its opening line, no spaces at line ends, at most two blank lines in a
// row, none at the start, and one newline at the end.
//
// It changes only the space at the start and end of lines, never what's
// on them: in Turtle a space can matter ("nums get -1" isn't "a - 1").
// Lines inside a `raw string` or a //* *// comment are left exactly as
// they are. Code that doesn't parse isn't formatted: fix it first.
func Format(src string) (string, error) {
	return FormatFile(src, "")
}

// FormatFile is Format for the file at path, so its imports are found: the
// phrases of an imported file's theories are read as phrases, private ones
// too in a test file (test_*.turtle), which may test them.
func FormatFile(src, path string) (string, error) {
	p := parser.New(lexer.New(src))
	dir := ""
	if path != "" {
		dir = filepath.Dir(path)
		p.TestFile = strings.HasPrefix(filepath.Base(path), "test_")
	}
	p.ModuleDir = dir
	p.ParseProgram()
	if len(p.ErrorList()) > 0 {
		// Perhaps a library's own code, using its words without importing
		// itself.
		lib := parser.New(lexer.New(src))
		lib.ModuleDir = dir
		lib.TestFile = p.TestFile
		for _, name := range []string{"random", "test", "log", "linear"} {
			lib.Enable(name)
		}
		lib.ParseProgram()
		if len(lib.ErrorList()) == 0 {
			p = lib
		}
	}
	if errs := p.ErrorList(); len(errs) > 0 {
		return "", fmt.Errorf("it doesn't parse yet, so it isn't formatted: %s", errs[0])
	}
	crlf := strings.Contains(src, "\r\n")
	text := strings.ReplaceAll(src, "\r\n", "\n")
	lines := strings.Split(text, "\n")

	// Which lines start inside a string or a comment (kept as they are),
	// and each line's tokens.
	starts := make([]int, len(lines)) // each line's offset in text
	for i, off := 1, 0; i < len(lines); i++ {
		off += len(lines[i-1]) + 1
		starts[i] = off
	}
	verbatim := make([]bool, len(lines))
	for _, s := range syntax.Highlight(text, syntax.Words{}) {
		if s.Class != syntax.String && s.Class != syntax.Comment {
			continue
		}
		for i, st := range starts {
			if st > s.Start && st < s.End {
				verbatim[i] = true
			}
		}
	}
	toks := make([][]token.Token, len(lines))
	l := lexer.New(text)
	for t := l.NextToken(); t.Type != token.EOF; t = l.NextToken() {
		if n := t.Line - 1; n >= 0 && n < len(lines) {
			toks[n] = append(toks[n], t)
		}
	}

	var stack []string // the blocks open: def, if, nestedif, loop, safe, give, block, scroll, bracket
	var out []string
	blank := 0
	for i, line := range lines {
		if verbatim[i] {
			out = append(out, line)
			blank = 0
			continue
		}
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			blank++
			if blank <= 2 && len(out) > 0 {
				out = append(out, "")
			}
			continue
		}
		blank = 0
		depth, next, err := lineDepth(toks[i], stack)
		if err != nil {
			return "", fmt.Errorf("line %d: %v", i+1, err)
		}
		stack = next
		out = append(out, strings.Repeat("    ", depth)+trimmed)
	}
	for len(out) > 0 && out[len(out)-1] == "" {
		out = out[:len(out)-1]
	}
	result := strings.Join(out, "\n") + "\n"
	if len(out) == 0 {
		result = ""
	}
	if !sameCode(text, result) {
		return "", errors.New("formatting would change the code; left as it is")
	}
	if crlf {
		result = strings.ReplaceAll(result, "\n", "\r\n")
	}
	return result, nil
}

// bracketBalance is how many more [ than ] the tokens have.
func bracketBalance(t []token.Token) int {
	n := 0
	for _, tok := range t {
		switch tok.Type {
		case token.LBRACKET:
			n++
		case token.RBRACKET:
			n--
		}
	}
	return n
}

// lineDepth is how deep a line goes, from its tokens and the blocks open
// before it, and the blocks open after it.
func lineDepth(t []token.Token, stack []string) (int, []string, error) {
	is := func(types ...token.Type) bool {
		if len(t) < len(types) {
			return false
		}
		for i, ty := range types {
			if t[i].Type != ty {
				return false
			}
		}
		return true
	}
	// close pops nested ifs (which end with what holds them), then the
	// block kind; the line goes at that block's depth.
	close := func(kind string) (int, []string, error) {
		s := popNested(stack, kind)
		if len(s) == 0 || s[len(s)-1] != kind {
			return 0, nil, fmt.Errorf("its blocks don't line up")
		}
		return len(s) - 1, s[:len(s)-1], nil
	}
	// middle is else or handle: at its block's depth, which stays open.
	middle := func(kind string) (int, []string, error) {
		s := popNested(stack, kind)
		if len(s) == 0 || s[len(s)-1] != kind {
			return 0, nil, fmt.Errorf("its blocks don't line up")
		}
		return len(s) - 1, s, nil
	}
	open := func(kind string) (int, []string, error) {
		return len(stack), append(append([]string{}, stack...), kind), nil
	}
	if len(t) == 0 { // a comment line: at the depth of what's around it
		return len(stack), stack, nil
	}
	if d, next, ok := theoryDepth(t, stack); ok {
		return d, next, nil
	}
	last := t[len(t)-1]
	// A scroll over several lines: its steps one step in, until the
	// line ending with its period.
	//     names is scroll raw into
	//         splitby ",",
	//         join[" - "] .
	if len(stack) > 0 && stack[len(stack)-1] == "scroll" {
		if last.Type == token.PERIOD {
			return len(stack), stack[:len(stack)-1], nil
		}
		return len(stack), stack, nil
	}
	// Brackets left open over several lines (a map, a list, a matrix, a
	// call's values): what's inside goes one step in, and a line starting
	// with the closing ] goes back to the depth of the line that opened it.
	//     routes = map [
	//         "GET /": home,
	//         "GET /health": "ok"
	//     ]
	// A line opening more than one ([list [ ...) goes one step in, not two:
	// the brackets after the first are "bracket+", which add no depth.
	if len(stack) > 0 && strings.HasPrefix(stack[len(stack)-1], "bracket") {
		depth := visibleDepth(stack)
		if t[0].Type == token.RBRACKET {
			depth--
		}
		next := append([]string{}, stack...)
		n := bracketBalance(t)
		for ; n < 0 && len(next) > 0 && strings.HasPrefix(next[len(next)-1], "bracket"); n++ {
			next = next[:len(next)-1]
		}
		return depth, openBrackets(next, n), nil
	}
	// A scroll goes on to the next lines only if its period isn't on this
	// one: diagnose[scroll 3 into add1 .] is whole.
	if last.Type != token.PERIOD && last.Type != token.GIVES {
		for i, tok := range t {
			if tok.Type == token.SCROLL && !hasPeriod(t[i+1:]) {
				return len(stack), append(append([]string{}, stack...), "scroll"), nil
			}
		}
	}
	// validate's rule goes on its own line, one step in:
	//     validate evens[nums] with nums as list of integer
	//         that result each x give x % 2 == 0 .
	if t[0].Type == token.IDENT && t[0].Literal == "that" && !is(token.IDENT, token.ASSIGN) && !is(token.IDENT, token.IS) {
		return len(stack) + 1, stack, nil
	}
	switch {
	case is(token.DEF, token.LBRACKET, token.END):
		return close("def")
	case is(token.IF, token.LBRACKET, token.END):
		return close("if")
	case is(token.LBRACKET, token.LOOP, token.RBRACKET, token.LBRACKET, token.END):
		return close("loop")
	case is(token.SAFE, token.LBRACKET, token.END):
		return close("safe")
	case len(t) == 4 && t[0].Type == token.IDENT && t[0].Literal == "diagnose" && is(token.IDENT, token.LBRACKET, token.END):
		return close("diagnose")
	case len(t) == 1 && t[0].Type == token.IDENT && t[0].Literal == "diagnose":
		return open("diagnose")
	case is(token.GIVES, token.LBRACKET, token.END):
		return close("give")
	case is(token.LBRACKET, token.END, token.RBRACKET):
		return close("block")
	case is(token.ELSE):
		return middle("if")
	case is(token.LBRACKET, token.ELSE):
		return middle("nestedif")
	case is(token.HANDLE):
		return middle("safe")
	case is(token.DEF, token.IDENT, token.LBRACKET):
		return open("def")
	case is(token.IF, token.RBRACKET):
		return open("if")
	case is(token.LBRACKET, token.IF, token.RBRACKET):
		return open("nestedif")
	case is(token.LBRACKET, token.LOOP):
		return open("loop")
	case is(token.SAFE) && len(t) == 1:
		return open("safe")
	case (is(token.LBRACKET, token.WRITE) || is(token.LBRACKET, token.APPEND)) && !endsWithEnd(t):
		return open("block")
	case last.Type == token.GIVES:
		return open("give")
	}
	if n := bracketBalance(t); n > 0 {
		return len(stack), openBrackets(append([]string{}, stack...), n), nil
	}
	return len(stack), stack, nil
}

func hasPeriod(t []token.Token) bool {
	for _, tok := range t {
		if tok.Type == token.PERIOD {
			return true
		}
	}
	return false
}

// openBrackets adds n brackets left open by one line: one step of depth.
func openBrackets(stack []string, n int) []string {
	for i := 0; i < n; i++ {
		if i == 0 {
			stack = append(stack, "bracket")
		} else {
			stack = append(stack, "bracket+")
		}
	}
	return stack
}

// visibleDepth is how many steps in the blocks open put a line.
func visibleDepth(stack []string) int {
	n := 0
	for _, s := range stack {
		if s != "bracket+" {
			n++
		}
	}
	return n
}

// popNested drops the nested ifs ([if ] ...) above the nearest block of
// kind: they end when the block that holds them goes on.
func popNested(stack []string, kind string) []string {
	s := stack
	for len(s) > 0 && s[len(s)-1] == "nestedif" && kind != "nestedif" {
		s = s[:len(s)-1]
	}
	return s
}

// endsWithEnd: the line closes itself with [end] ([write] f "x" [end]).
func endsWithEnd(t []token.Token) bool {
	n := len(t)
	return n >= 3 && t[n-3].Type == token.LBRACKET && t[n-2].Type == token.END && t[n-1].Type == token.RBRACKET
}

// sameCode: a and b have the same tokens and comments, only laid out
// differently.
func sameCode(a, b string) bool {
	return codeOf(a) == codeOf(b)
}

func codeOf(src string) string {
	var sb strings.Builder
	l := lexer.New(src)
	for t := l.NextToken(); t.Type != token.EOF; t = l.NextToken() {
		fmt.Fprintf(&sb, "%s %q\n", t.Type, t.Literal)
	}
	for _, s := range syntax.Highlight(src, syntax.Words{}) {
		if s.Class == syntax.Comment {
			var lines []string
			for _, line := range strings.Split(src[s.Start:s.End], "\n") {
				lines = append(lines, strings.TrimSpace(line))
			}
			sb.WriteString(strings.Join(lines, "\n") + "\n")
		}
	}
	return sb.String()
}

// theoryDepth lays out a theory: its sections one step in, what they hold
// (the abstract's text, the definition's code, the proof's lines) a step
// further, and theory [end] back at theory's depth.
//
//	theory tally
//	    abstract
//	        tally says how many times b is in a.
//	    notation tally b in a .
//	    definition
//	        ...
//	theory [end]
func theoryDepth(t []token.Token, stack []string) (int, []string, bool) {
	top := ""
	if len(stack) > 0 {
		top = stack[len(stack)-1]
	}
	switch {
	case len(t) == 2 && t[0].Literal == "theory" && t[1].Type == token.IDENT:
		return len(stack), append(append([]string{}, stack...), "theory"), true
	case len(t) == 4 && t[0].Literal == "theory" && t[1].Type == token.LBRACKET && t[2].Type == token.END:
		s := closeSection(stack)
		if len(s) > 0 && s[len(s)-1] == "theory" {
			return len(s) - 1, s[:len(s)-1], true
		}
		return 0, stack, false
	case isSection(t) && inTheory(stack):
		s := closeSection(stack)
		switch t[0].Literal {
		case "abstract", "definition", "proof":
			return len(s), append(append([]string{}, s...), "section:"+t[0].Literal), true
		}
		return len(s), s, true
	case top == "section:abstract":
		return len(stack), stack, true // its text, as it is
	}
	return 0, nil, false
}

var theorySections = map[string]bool{"abstract": true, "notation": true, "definition": true, "theorem": true, "proof": true}

// isSection: the line starts a theory's section (abstract, notation,
// definition, theorem, proof), not code that uses the word as a name.
func isSection(t []token.Token) bool {
	if t[0].Type != token.IDENT || !theorySections[t[0].Literal] {
		return false
	}
	if len(t) > 1 {
		switch t[1].Type {
		case token.ASSIGN, token.IS, token.AT, token.OF, token.LBRACKET:
			return false
		}
	}
	return true
}

// inTheory: the line is in a theory, outside any block of code a section
// opened.
func inTheory(stack []string) bool {
	if len(stack) == 0 {
		return false
	}
	top := stack[len(stack)-1]
	return top == "theory" || strings.HasPrefix(top, "section:")
}

// closeSection ends the section open, if one is.
func closeSection(stack []string) []string {
	s := popNested(stack, "")
	if len(s) > 0 && strings.HasPrefix(s[len(s)-1], "section:") {
		return s[:len(s)-1]
	}
	return s
}
