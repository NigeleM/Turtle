// Copyright 2017-2026 Nigele McCoy
// Licensed under the Apache License, Version 2.0;
// see LICENSE and NOTICE.

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
	Plain          Class = iota
	Keyword              // if, loop, def, return, ... and a library's words (random, check, log)
	Constant             // numbers, true, false, none
	String               // "text"
	Comment              // // ... and //* ... *//
	Definition           // the name after def
	Call                 // name[ ... ]: a function call
	Builtin              // a library function: sql_open[ ... ], min_sort[ ... ]
	Import               // an import line, all of it: import time [now]
	Show                 // show, warn: where a program speaks
	Data                 // list, set, map, matrix, assemble, and assembled types: Order
	DataDefinition       // the name after assemble
	Method               // the name after at: x at upper
	Theory               // theory, its sections, its [end], and a theory's word where it's used
	TestDefinition       // the name after def of a test: test_total
)

// Span is a colored stretch of the source, [Start, End) in bytes.
type Span struct {
	Start, End int
	Class      Class
}

// Words are the names colored specially: the words of the libraries
// imported (random, check, log ...) and the library functions.
type Words struct {
	Context  map[string]bool
	Builtin  map[string]bool
	Theories map[string]bool // the theories' words: tally
	Types    map[string]bool // the assembled types: Order
}

// theorySections are a theory's section words, special at a line's start
// inside a theory.
var theorySections = map[string]bool{"abstract": true, "notation": true, "definition": true, "theorem": true, "proof": true}

// LibraryWords are the words a library's import turns on.
var LibraryWords = map[string][]string{
	"random": {"random"},
	"test":   {"check", "verify", "validate", "each", "any", "not", "exactly", "least", "most", "pair", "that", "with", "as", "matches", "fails", "close", "within"},
	"log":    {"log", "debug", "info", "error"},
	"linear": {"matrix"},
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
	inTheory := false
	inAbstract := false // a theory's abstract is prose, not code
	importLine := 0     // the line of an import being colored, all of it
	for i, t := range toks {
		out = append(out, comments(src, prevEnd, t.Pos)...)
		prevEnd = t.End
		next := token.Token{}
		if i+1 < len(toks) {
			next = toks[i+1]
		}
		prev := token.Token{}
		if i > 0 {
			prev = toks[i-1]
		}
		lineStart := i == 0 || prev.Line < t.Line
		if inAbstract && lineStart && (theorySections[t.Literal] || t.Literal == "theory") {
			inAbstract = false
		}
		if inAbstract {
			continue
		}
		c := Plain
		switch {
		case importLine == t.Line:
			c = Import
		case t.Type == token.IMPORT:
			c, importLine = Import, t.Line
		case t.Type == token.STRING || t.Type == token.RAWSTRING:
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
		case t.Type == token.SHOW || t.Type == token.WARN:
			c = Show
		case t.Type == token.LIST || t.Type == token.SET || t.Type == token.MAP || t.Type == token.ASSEMBLE:
			c = Data
		case t.Type == token.END && prev.Type == token.LBRACKET && i >= 2 && toks[i-2].Type == token.IDENT && toks[i-2].Literal == "theory" && toks[i-2].Line == t.Line:
			// theory [end]: one span, from theory to the ].
			inTheory = false
			if n := len(out); n > 0 && out[n-1].Class == Theory && out[n-1].Start == toks[i-2].Pos {
				end := t.End
				if next.Type == token.RBRACKET && next.Line == t.Line {
					end = next.End
				}
				out[n-1].End = end
			}
			continue
		case token.IsKeyword(t.Type):
			c = Keyword
		case t.Type == token.IDENT:
			switch {
			case t.Literal == "theory" && lineStart && next.Line == t.Line && (next.Type == token.IDENT || next.Type == token.LBRACKET):
				c = Theory
				if next.Type == token.IDENT {
					inTheory = true
				}
			case prev.Type == token.IDENT && prev.Literal == "theory" && prev.Line == t.Line && (i < 2 || toks[i-2].Line < prev.Line):
				c = Theory // the theory's name
			case inTheory && lineStart && theorySections[t.Literal]:
				c = Theory
				inAbstract = t.Literal == "abstract" && next.Line > t.Line
			case prev.Type == token.DEF:
				c = Definition
				if strings.HasPrefix(t.Literal, "test_") {
					c = TestDefinition
				}
			case prev.Type == token.ASSEMBLE:
				c = DataDefinition
			case w.Theories[t.Literal]:
				c = Theory
			case w.Types[t.Literal]:
				c = Data
			case t.Literal == "matrix" && w.Context[t.Literal]:
				c = Data
			case w.Context[t.Literal]:
				c = Keyword
			case prev.Type == token.AT && prev.Line == t.Line:
				c = Method
			case next.Type == token.LBRACKET && next.Pos == t.End:
				c = Call
				if w.Builtin[t.Literal] {
					c = Builtin
				}
			case w.Builtin[t.Literal] && sentenceAfter(prev, t):
				// nums process x give x + 1: a library function in a sentence.
				c = Builtin
			}
		}
		if c != Plain {
			out = append(out, Span{t.Pos, t.End, c})
		}
	}
	return append(out, comments(src, prevEnd, len(src))...)
}

// sentenceAfter tells whether prev is a value a sentence call could
// follow on the same line: a name, a literal, or a closing ] or ).
func sentenceAfter(prev, t token.Token) bool {
	if prev.Line != t.Line {
		return false
	}
	switch prev.Type {
	case token.IDENT, token.INT, token.FLOAT, token.STRING, token.RAWSTRING, token.RBRACKET, token.RPAREN:
		return true
	}
	return false
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
	"find", "floor", "get", "getkeys", "getvalues", "index", "indexof", "insert",
	"intersection", "invert", "isempty", "isnumber", "len", "length", "lower",
	"pop", "pow", "put", "random", "remove", "replace", "reverse", "round",
	"slice", "sort", "split", "sqrt", "subset", "superset", "tostring", "trim",
	"union", "upper", "fixed", "commas", "padleft", "padright",
	"rows", "columns", "shape", "row", "column",
}

// ShapeWords are the kinds of value that can follow random (and "as" in
// validate): random list of 5 integers.
var ShapeWords = []string{
	"integer", "integers", "float", "floats", "string", "strings", "digit", "digits",
	"letter", "letters", "boolean", "booleans", "date", "dates", "time", "times",
	"list", "lists", "set", "sets", "map", "maps",
}
