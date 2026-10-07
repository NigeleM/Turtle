package repl

import (
	"strings"

	"Turtle/lexer"
	"Turtle/token"
)

// Coloring Turtle code from its real tokens, so the colors always match
// what the parser sees. The scheme is IDLE's: keywords orange, strings
// green, comments red, definitions blue, library functions purple.

// class is what a piece of code is, for its color.
type class int

const (
	plain      class = iota
	keyword          // if, def, show, import, ... and a library's words (random, check, log)
	constant         // numbers, true, false, none
	stringLit        // "text"
	comment          // // ... and //* ... *//
	definition       // the name after def or assemble
	call             // name[ ... ]: a function call
	builtin          // a library function: sql_open[ ... ], min_sort[ ... ]
)

// span is a colored stretch of the source, [start, end) in bytes.
type span struct {
	start, end int
	c          class
}

// words are the names colored specially in this session: the words of
// the libraries imported so far (random, check, log ...) and the library
// functions.
type words struct {
	context map[string]bool
	builtin map[string]bool
}

// libraryWords are the words a library's import turns on.
var libraryWords = map[string][]string{
	"random": {"random"},
	"test":   {"check", "verify", "validate", "each", "any", "not", "exactly", "least", "most", "pair", "that", "with", "as", "matches", "fails", "close", "within"},
	"log":    {"log", "debug", "info", "error"},
}

// highlight finds the colored spans of src.
func highlight(src string, w words) []span {
	l := lexer.New(src)
	var toks []token.Token
	for {
		t := l.NextToken()
		if t.Type == token.EOF {
			break
		}
		toks = append(toks, t)
	}
	var out []span
	prevEnd := 0
	for i, t := range toks {
		out = append(out, comments(src, prevEnd, t.Pos)...)
		prevEnd = t.End
		c := plain
		switch {
		case t.Type == token.STRING:
			c = stringLit
		case t.Type == token.INT || t.Type == token.FLOAT || t.Type == token.TRUE || t.Type == token.FALSE || t.Type == token.NONE:
			c = constant
		case t.Type == token.SYS:
			// The lexer keeps the rest of the line as the command: color
			// just the word.
			start := strings.Index(src[t.Pos:], "sys")
			if start >= 0 {
				out = append(out, span{t.Pos + start, t.Pos + start + 3, keyword})
			}
			continue
		case token.IsKeyword(t.Type):
			c = keyword
		case t.Type == token.IDENT:
			next := token.Token{}
			if i+1 < len(toks) {
				next = toks[i+1]
			}
			prev := token.Token{}
			if i > 0 {
				prev = toks[i-1]
			}
			switch {
			case prev.Type == token.DEF || prev.Type == token.ASSEMBLE:
				c = definition
			case w.context[t.Literal]:
				c = keyword
			case next.Type == token.LBRACKET && next.Pos == t.End:
				c = call
				if w.builtin[t.Literal] {
					c = builtin
				}
			}
		}
		if c != plain {
			out = append(out, span{t.Pos, t.End, c})
		}
	}
	return append(out, comments(src, prevEnd, len(src))...)
}

// comments finds the comments in src[from:to], the space between two
// tokens.
func comments(src string, from, to int) []span {
	var out []span
	for from < to {
		i := strings.Index(src[from:to], "//")
		if i < 0 {
			break
		}
		start := from + i
		end := to
		if strings.HasPrefix(src[start:], "//*") {
			if j := strings.Index(src[start+3:to], "*//"); j >= 0 {
				end = start + 3 + j + 3
			}
		} else if j := strings.IndexByte(src[start:to], '\n'); j >= 0 {
			end = start + j
		}
		out = append(out, span{start, end, comment})
		from = end
	}
	return out
}

// ---- colors ----

// The colors, from the 256-color palette most terminals have, chosen to
// read on dark and light backgrounds alike.
var colorCodes = map[class]string{
	keyword:    "\x1b[38;5;208m",  // orange
	constant:   "\x1b[38;5;135m",  // purple
	stringLit:  "\x1b[38;5;34m",   // green
	comment:    "\x1b[38;5;167m",  // red
	definition: "\x1b[1;38;5;33m", // bold blue
	call:       "\x1b[38;5;33m",   // blue
	builtin:    "\x1b[38;5;133m",  // magenta
}

const (
	reset       = "\x1b[0m"
	promptColor = "\x1b[38;5;244m" // gray
	errorColor  = "\x1b[38;5;196m" // red
	resultColor = "\x1b[38;5;33m"  // blue
)

// colorize writes src with its colors as terminal codes.
func colorize(src string, w words) string {
	var b strings.Builder
	at := 0
	for _, s := range highlight(src, w) {
		if s.start < at || s.end > len(src) {
			continue
		}
		b.WriteString(src[at:s.start])
		b.WriteString(colorCodes[s.c])
		b.WriteString(src[s.start:s.end])
		b.WriteString(reset)
		at = s.end
	}
	b.WriteString(src[at:])
	return b.String()
}
