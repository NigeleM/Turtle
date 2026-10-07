// Package syntax colors Turtle code from its real tokens, for the REPL
// (terminal colors) and the language server (an editor's colors): what
// each stretch of source is, so the colors always match what the parser
// sees.
package syntax

import (
	"strings"

	"Turtle/lexer"
	"Turtle/token"
)

// Class is what a piece of code is, for its color.
type Class int

const (
	Plain      Class = iota
	Keyword          // if, def, show, import, ... and a library's words (random, check, log)
	Constant         // numbers, true, false, none
	String           // "text"
	Comment          // // ... and //* ... *//
	Definition       // the name after def or assemble
	Call             // name[ ... ]: a function call
	Builtin          // a library function: sql_open[ ... ], min_sort[ ... ]
)

// Span is a colored stretch of the source, [Start, End) in bytes.
type Span struct {
	Start, End int
	Class      Class
}

// Words are the names colored specially: the words of the libraries
// imported (random, check, log ...) and the library functions.
type Words struct {
	Context map[string]bool
	Builtin map[string]bool
}

// LibraryWords are the words a library's import turns on.
var LibraryWords = map[string][]string{
	"random": {"random"},
	"test":   {"check", "verify", "validate", "each", "any", "not", "exactly", "least", "most", "pair", "that", "with", "as", "matches", "fails", "close", "within"},
	"log":    {"log", "debug", "info", "error"},
}

// Highlight finds the colored spans of src.
func Highlight(src string, w Words) []Span {
	l := lexer.New(src)
	var toks []token.Token
	for {
		t := l.NextToken()
		if t.Type == token.EOF {
			break
		}
		toks = append(toks, t)
	}
	var out []Span
	prevEnd := 0
	for i, t := range toks {
		out = append(out, comments(src, prevEnd, t.Pos)...)
		prevEnd = t.End
		c := Plain
		switch {
		case t.Type == token.STRING:
			c = String
		case t.Type == token.INT || t.Type == token.FLOAT || t.Type == token.TRUE || t.Type == token.FALSE || t.Type == token.NONE:
			c = Constant
		case t.Type == token.SYS:
			// The lexer keeps the rest of the line as the command: color
			// just the word.
			start := strings.Index(src[t.Pos:], "sys")
			if start >= 0 {
				out = append(out, Span{t.Pos + start, t.Pos + start + 3, Keyword})
			}
			continue
		case token.IsKeyword(t.Type):
			c = Keyword
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
				c = Definition
			case w.Context[t.Literal]:
				c = Keyword
			case next.Type == token.LBRACKET && next.Pos == t.End:
				c = Call
				if w.Builtin[t.Literal] {
					c = Builtin
				}
			}
		}
		if c != Plain {
			out = append(out, Span{t.Pos, t.End, c})
		}
	}
	return append(out, comments(src, prevEnd, len(src))...)
}

// comments finds the comments in src[from:to], the space between two
// tokens.
func comments(src string, from, to int) []Span {
	var out []Span
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
		out = append(out, Span{start, end, Comment})
		from = end
	}
	return out
}

// Methods are the names that can follow "at": x at upper, nums at get[0].
// (Library functions are called by name, sql_open[...], not with at.)
var Methods = []string{
	"abs", "add", "ceil", "clear", "contains", "count", "delete", "difference",
	"find", "floor", "get", "getKeys", "getValues", "index", "indexOf", "insert",
	"intersection", "invert", "isEmpty", "isNumber", "len", "length", "lower",
	"pop", "pow", "put", "random", "remove", "replace", "reverse", "round",
	"slice", "sort", "split", "sqrt", "subset", "superset", "toString", "trim",
	"union", "upper",
}

// ShapeWords are the kinds of value that can follow random (and "as" in
// validate): random list of 5 integers.
var ShapeWords = []string{
	"integer", "integers", "float", "floats", "string", "strings", "digit", "digits",
	"letter", "letters", "boolean", "booleans", "date", "dates", "time", "times",
	"list", "lists", "set", "sets", "map", "maps",
}
