// Copyright 2017-2026 Nigele McCoy
// Licensed under the Apache License, Version 2.0;
// see LICENSE and NOTICE.

// Package lexer tokenizes Turtle source text.
package lexer

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"Turtle/token"
)

type Lexer struct {
	input          string
	pos            int
	readPos        int
	ch             byte
	line           int
	inBlockComment bool
	atLineStart    bool
	unclosed       bool // the last string read reached the end of the input
	names          map[string]string
}

// Source is the text being read.
func (l *Lexer) Source() string { return l.input }

func New(input string) *Lexer {
	l := &Lexer{input: input, line: 1, atLineStart: true}
	l.readChar()
	return l
}

func (l *Lexer) readChar() {
	if l.readPos >= len(l.input) {
		l.ch = 0
	} else {
		l.ch = l.input[l.readPos]
	}
	if l.ch == '\n' {
		// line is incremented when we consume the newline below, not here;
		// callers rely on l.line reflecting the line of l.ch itself.
	}
	l.pos = l.readPos
	l.readPos++
}

func (l *Lexer) peekChar() byte {
	if l.readPos >= len(l.input) {
		return 0
	}
	return l.input[l.readPos]
}

func (l *Lexer) peekChar2() byte {
	if l.readPos+1 >= len(l.input) {
		return 0
	}
	return l.input[l.readPos+1]
}

// NextToken returns the next token in the input.
func (l *Lexer) NextToken() token.Token {
	tok := l.nextToken()
	// An unfinished string ("abc with no closing quote) reads one step
	// past the end.
	tok.End = min(l.pos, len(l.input))
	if tok.Type == token.EOF {
		tok.Pos, tok.End = len(l.input), len(l.input)
	}
	return tok
}

func (l *Lexer) nextToken() token.Token {
	before := l.pos
	l.skipWhitespaceAndComments()

	tok := token.Token{Line: l.line, SpaceBefore: l.pos != before || l.pos == 0, Pos: l.pos}
	startOfLine := l.atLineStart
	l.atLineStart = false

	switch l.ch {
	case '=':
		if l.peekChar() == '=' {
			l.readChar()
			tok.Type, tok.Literal = token.EQ, "=="
		} else {
			tok.Type, tok.Literal = token.ASSIGN, "="
		}
	case '+':
		if l.peekChar() == '+' {
			l.readChar()
			tok.Type, tok.Literal = token.INCR, "++"
		} else {
			tok.Type, tok.Literal = token.PLUS, "+"
		}
	case '-':
		if l.peekChar() == '-' {
			l.readChar()
			tok.Type, tok.Literal = token.DECR, "--"
		} else {
			tok.Type, tok.Literal = token.MINUS, "-"
		}
	case '*':
		tok.Type, tok.Literal = token.ASTERISK, "*"
	case '/':
		tok.Type, tok.Literal = token.SLASH, "/"
	case '%':
		tok.Type, tok.Literal = token.PERCENT, "%"
	case '<':
		if l.peekChar() == '=' {
			l.readChar()
			tok.Type, tok.Literal = token.LE, "<="
		} else {
			tok.Type, tok.Literal = token.LT, "<"
		}
	case '>':
		if l.peekChar() == '=' {
			l.readChar()
			tok.Type, tok.Literal = token.GE, ">="
		} else {
			tok.Type, tok.Literal = token.GT, ">"
		}
	case '!':
		if l.peekChar() == '=' {
			l.readChar()
			tok.Type, tok.Literal = token.NOT_EQ, "!="
		} else {
			tok.Type, tok.Literal = token.BANG, "!"
		}
	case '&':
		if l.peekChar() == '&' {
			l.readChar()
			tok.Type, tok.Literal = token.AND, "&&"
		} else {
			tok.Type, tok.Literal = token.ILLEGAL, string(l.ch)
		}
	case '|':
		if l.peekChar() == '|' {
			l.readChar()
			tok.Type, tok.Literal = token.OR, "||"
		} else {
			tok.Type, tok.Literal = token.ILLEGAL, string(l.ch)
		}
	case ',':
		tok.Type, tok.Literal = token.COMMA, ","
	case ':':
		tok.Type, tok.Literal = token.COLON, ":"
	case ';':
		tok.Type, tok.Literal = token.SEMI, ";"
	case '[':
		tok.Type, tok.Literal = token.LBRACKET, "["
	case ']':
		tok.Type, tok.Literal = token.RBRACKET, "]"
	case '(':
		tok.Type, tok.Literal = token.LPAREN, "("
	case ')':
		tok.Type, tok.Literal = token.RPAREN, ")"
	case '?':
		tok.Type, tok.Literal = token.QUESTION, "?"
	case '"', '\'':
		// 'single quotes' are the same string, so text full of double
		// quotes (JSON, speech) needs no escapes: '{"name": "Ann"}'.
		tok.Type = token.STRING
		tok.Literal = l.readString(l.ch)
		tok.Unclosed = l.unclosed
		return tok
	case '`':
		tok.Type = token.RAWSTRING
		tok.Literal = l.readRawString()
		tok.Unclosed = l.unclosed
		return tok
	case 0:
		tok.Type, tok.Literal = token.EOF, ""
	case '.':
		if isDigit(l.peekChar()) {
			tok.Type, tok.Literal = token.FLOAT, l.readNumberStartingWithDot()
			return tok
		}
		tok.Type, tok.Literal = token.PERIOD, "."
	default:
		// ~ before a name makes it a private function's: ~limit.
		if isLetter(l.ch) || l.ch == '~' && isLetter(l.peekChar()) || l.wideLetter(l.pos, true) > 0 || l.ch == '~' && l.wideLetter(l.pos+1, true) > 0 {
			tok.Literal = l.readIdentifier()
			// `sys` is only ever a leading statement keyword: when it starts a line, the rest of that line is
			// captured raw as an arbitrary shell command rather than
			// tokenized, since shell syntax isn't valid Turtle syntax.
			if tok.Literal == "sys" && startOfLine {
				tok.Type = token.SYS
				tok.Literal = l.captureRestOfLine()
				return tok
			}
			tok.Type = token.LookupIdent(tok.Literal)
			return tok
		} else if isDigit(l.ch) {
			litType, lit := l.readNumber()
			tok.Type, tok.Literal = litType, lit
			return tok
		}
		tok.Type, tok.Literal = token.ILLEGAL, string(l.ch)
		if l.ch >= utf8.RuneSelf { // the whole character, not its first byte: 🐢
			r, n := utf8.DecodeRuneInString(l.input[l.pos:])
			if r != utf8.RuneError {
				tok.Literal = string(r)
				for range n - 1 {
					l.readChar()
				}
			}
		}
	}

	l.readChar()
	return tok
}

func (l *Lexer) skipWhitespaceAndComments() {
	for {
		if l.inBlockComment {
			if l.ch == 0 {
				return
			}
			if l.ch == '*' && l.peekChar() == '/' && l.peekChar2() == '/' {
				l.readChar()
				l.readChar()
				l.readChar()
				l.inBlockComment = false
				continue
			}
			l.advanceRaw()
			continue
		}
		switch {
		case l.ch == ' ' || l.ch == '\t' || l.ch == '\r' || l.ch == '\n':
			l.advanceRaw()
		case l.ch == '/' && l.peekChar() == '/' && l.peekChar2() == '*':
			l.readChar()
			l.readChar()
			l.readChar()
			l.inBlockComment = true
		case l.ch == '/' && l.peekChar() == '/':
			for l.ch != '\n' && l.ch != 0 {
				l.advanceRaw()
			}
		default:
			return
		}
	}
}

// advanceRaw is readChar but also tracks line numbers; readChar itself
// stays line-agnostic so callers reading multi-char literals (numbers,
// identifiers, strings) don't have to think about it.
func (l *Lexer) advanceRaw() {
	if l.ch == '\n' {
		l.line++
		l.atLineStart = true
	}
	l.readChar()
}

// captureRestOfLine grabs everything from the current position (right
// after the "sys" identifier) to the end of the line, verbatim, and
// leaves the lexer positioned to resume normal tokenizing on the next
// line. Used only for `sys` shell-escape statements.
func (l *Lexer) captureRestOfLine() string {
	if l.ch == ' ' {
		l.readChar()
	}
	start := l.pos
	for l.ch != '\n' && l.ch != 0 {
		l.readChar()
	}
	cmd := l.input[start:l.pos]
	if l.ch == '\n' {
		l.line++
		l.atLineStart = true
		l.readChar()
	}
	return strings.TrimRight(cmd, " \t\r")
}

func (l *Lexer) readIdentifier() string {
	start := l.pos
	if l.ch == '~' {
		l.readChar()
	}
	for {
		if isLetter(l.ch) || isDigit(l.ch) {
			l.readChar()
		} else if n := l.wideLetter(l.pos, false); n > 0 {
			for range n {
				l.readChar()
			}
		} else {
			break
		}
	}
	// Every use of a name shares one string, so the interpreter, matching
	// a name against a scope's names, finds it by its first byte's address
	// instead of comparing letters.
	name := l.input[start:l.pos]
	if shared, ok := l.names[name]; ok {
		return shared
	}
	if l.names == nil {
		l.names = map[string]string{}
	}
	l.names[name] = name
	return name
}

func (l *Lexer) readNumber() (token.Type, string) {
	start := l.pos
	kind := token.INT
	for isDigit(l.ch) {
		l.readChar()
	}
	if l.ch == '.' && isDigit(l.peekChar()) {
		l.readChar()
		for isDigit(l.ch) {
			l.readChar()
		}
		kind = token.FLOAT
	}
	// Scientific notation: 1e-18, 2.5e6, 6.02E23 (a float, as in Python).
	// The e needs digits after it (a sign first is fine), so "2e" alone
	// stays a number and a name.
	if l.ch == 'e' || l.ch == 'E' {
		next := l.peekChar()
		if isDigit(next) || (next == '+' || next == '-') && isDigit(l.peekChar2()) {
			l.readChar()
			if l.ch == '+' || l.ch == '-' {
				l.readChar()
			}
			for isDigit(l.ch) {
				l.readChar()
			}
			kind = token.FLOAT
		}
	}
	return kind, l.input[start:l.pos]
}

func (l *Lexer) readNumberStartingWithDot() string {
	start := l.pos
	l.readChar() // consume '.'
	for isDigit(l.ch) {
		l.readChar()
	}
	return l.input[start:l.pos]
}

func (l *Lexer) readString(quote byte) string {
	var sb strings.Builder
	l.readChar() // skip opening quote
	for l.ch != quote && l.ch != 0 {
		if l.ch == '\\' {
			switch l.peekChar() {
			case 'n':
				sb.WriteByte('\n')
				l.readChar()
			case 't':
				sb.WriteByte('\t')
				l.readChar()
			case '"', '\'':
				sb.WriteByte(l.peekChar())
				l.readChar()
			case '\\':
				sb.WriteByte('\\')
				l.readChar()
			case '{':
				// A literal brace, not an interpolation: kept as a marker
				// the parser turns back into '{' (see token.LiteralBrace).
				sb.WriteByte(token.LiteralBrace)
				l.readChar()
			case '}':
				sb.WriteByte('}')
				l.readChar()
			default:
				sb.WriteByte(l.ch)
			}
		} else {
			sb.WriteByte(l.ch)
			if l.ch == '\n' { // text over several lines: later lines count on
				l.line++
			}
		}
		l.readChar()
	}
	l.unclosed = l.ch == 0
	l.readChar() // skip closing quote
	return sb.String()
}

// wideLetter is the size in bytes of a letter past ASCII at input[off]
// (é, ñ, 名), which names may use as Python's do; or, unless first, a
// digit or a combining mark. 0 for anything else, emoji included.
func (l *Lexer) wideLetter(off int, first bool) int {
	if off >= len(l.input) || l.input[off] < utf8.RuneSelf {
		return 0
	}
	r, n := utf8.DecodeRuneInString(l.input[off:])
	switch {
	case r == utf8.RuneError:
		return 0
	case unicode.IsLetter(r):
		return n
	case !first && (unicode.IsDigit(r) || unicode.Is(unicode.Mn, r) || unicode.Is(unicode.Mc, r)):
		return n
	}
	return 0
}

func isLetter(ch byte) bool {
	return 'a' <= ch && ch <= 'z' || 'A' <= ch && ch <= 'Z' || ch == '_'
}

func isDigit(ch byte) bool {
	return '0' <= ch && ch <= '9'
}

// Input is the source text being read, for messages that quote a line.
func (l *Lexer) Input() string { return l.input }

// readRawString reads `text` exactly as written, across lines too, up to
// the closing backtick (or the end of the source).
func (l *Lexer) readRawString() string {
	l.readChar() // the opening backtick
	start := l.pos
	for l.ch != '`' && l.ch != 0 {
		l.advanceRaw()
	}
	text := l.input[start:l.pos]
	l.unclosed = l.ch == 0
	if l.ch == '`' {
		l.readChar()
	}
	return text
}

// SeekLine moves the lexer to the start of line n (from 1), as if the
// lines before it had been read: a theory's abstract is free text, taken
// from the source as it is, and lexing goes on after it.
func (l *Lexer) SeekLine(n int) {
	off := 0
	for line := 1; line < n && off < len(l.input); line++ {
		i := strings.IndexByte(l.input[off:], '\n')
		if i < 0 {
			off = len(l.input)
			break
		}
		off += i + 1
	}
	l.readPos = off
	l.line = n
	l.atLineStart = true
	l.inBlockComment = false
	l.unclosed = false
	l.readChar()
}
