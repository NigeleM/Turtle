// Copyright 2017-2026 Nigele McCoy
// Licensed under the Apache License, Version 2.0;
// see LICENSE and NOTICE.

package repl

import (
	"regexp"
	"strings"

	"Turtle/lexer"
	"Turtle/token"
)

// When an entry is finished: the REPL runs an entry once its blocks are
// all closed, and otherwise asks for another line, with "...".

// openBlocks counts the blocks src leaves open: def ... def [end],
// if ] ... [ ... if [end], [loop] ... [loop][end], safe ... safe [end],
// a give function whose body starts on the next line ... give [end],
// and [read] / [write] / [append] / [directory] ... [end].
func openBlocks(src string) int {
	l := lexer.New(src)
	var t []token.Token
	for {
		tok := l.NextToken()
		t = append(t, tok)
		if tok.Type == token.EOF {
			break
		}
	}
	at := func(i int) token.Type {
		if i < 0 || i >= len(t) {
			return token.EOF
		}
		return t[i].Type
	}
	closes := func(i int) bool { // "<keyword> [end]"
		return at(i+1) == token.LBRACKET && at(i+2) == token.END
	}
	depth := 0
	for i, tok := range t {
		if tok.Type == token.IDENT && tok.Literal == "diagnose" && (i == 0 || t[i-1].Line < tok.Line) {
			switch {
			case closes(i):
				depth--
			case at(i+1) == token.EOF || t[i+1].Line > tok.Line:
				depth++
			}
			continue
		}
		switch tok.Type {
		case token.DEF, token.SAFE:
			if closes(i) {
				depth--
			} else {
				depth++
			}
		case token.IF:
			switch {
			case closes(i):
				depth--
			case at(i+1) == token.RBRACKET && !(at(i-1) == token.LBRACKET && t[i-1].Line == tok.Line):
				// "if ] ..." opens a chain; a nested "[if ] ..." chain
				// has no end of its own.
				depth++
			}
		case token.GIVES:
			if closes(i) {
				depth--
			} else if at(i+1) == token.EOF || t[i+1].Line != tok.Line {
				depth++ // "x give" at the end of a line: a body follows
			}
		case token.LOOP:
			if at(i+1) == token.RBRACKET && at(i+2) == token.LBRACKET && at(i+3) == token.END {
				depth--
			} else {
				depth++
			}
		case token.READ, token.WRITE, token.APPEND, token.DIR:
			if at(i-1) == token.LBRACKET {
				depth++
			}
		case token.END:
			// A bare "[end]" closes [read], [write], [append] or [directory].
			if at(i-1) == token.LBRACKET {
				owner := at(i - 2)
				loopEnd := owner == token.RBRACKET && at(i-3) == token.LOOP
				if owner != token.DEF && owner != token.IF && owner != token.SAFE && owner != token.GIVES && !loopEnd {
					depth--
				}
			}
		}
	}
	return max(depth, 0)
}

// needsMore reports whether src is unfinished: a block still open, or a
// validate sentence or a scroll (which can run over several lines) still
// waiting for its period.
func needsMore(src string) bool {
	if openBlocks(src) > 0 || theoryOpen(src) || scrollOpen(src) || matrixOpen(src) {
		return true
	}
	l := lexer.New(src)
	first := l.NextToken()
	if first.Type != token.IDENT || first.Literal != "validate" {
		return false
	}
	last := first
	for tok := l.NextToken(); tok.Type != token.EOF; tok = l.NextToken() {
		last = tok
	}
	return last.Type != token.PERIOD
}

// indentFor is how far to indent the next line of an unfinished entry:
// four spaces per open block, and four more for a theory's lines or a
// scroll's steps.
func indentFor(src string) int {
	n := openBlocks(src)
	if theoryOpen(src) || scrollOpen(src) || matrixOpen(src) {
		n++
	}
	return 4 * n
}

var (
	theoryStart = regexp.MustCompile(`^theory[ \t]+~?[A-Za-z_][A-Za-z0-9_]*[ \t]*(//.*)?$`)
	theoryEnd   = regexp.MustCompile(`^theory[ \t]*\[[ \t]*end[ \t]*\]`)
)

// theoryOpen reports whether src is inside a theory ... theory [end].
// Its lines all go one level in: a section word any deeper would be read
// as part of the section before it. Theories are read by their lines, as
// the parser reads them, not by tokens.
func theoryOpen(src string) bool {
	open := false
	for _, line := range strings.Split(src, "\n") {
		text := strings.TrimSpace(line)
		switch {
		case theoryEnd.MatchString(text):
			open = false
		case theoryStart.MatchString(text):
			open = true
		}
	}
	return open
}

// matrixOpen reports whether src has a matrix [ ... still waiting for its
// closing bracket: its rows go on the lines that follow.
func matrixOpen(src string) bool {
	l := lexer.New(src)
	depth := 0
	prev := token.Token{}
	for tok := l.NextToken(); tok.Type != token.EOF; tok = l.NextToken() {
		switch {
		case tok.Type == token.LBRACKET && (depth > 0 || prev.Type == token.IDENT && prev.Literal == "matrix"):
			depth++
		case tok.Type == token.RBRACKET && depth > 0:
			depth--
		}
		prev = tok
	}
	return depth > 0
}

// scrollOpen reports whether src's last line is a scroll still waiting
// for its period.
func scrollOpen(src string) bool {
	l := lexer.New(src)
	open := false
	for tok := l.NextToken(); tok.Type != token.EOF; tok = l.NextToken() {
		switch tok.Type {
		case token.SCROLL:
			open = true
		case token.PERIOD:
			open = false
		}
	}
	return open
}
