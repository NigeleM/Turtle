// Package lexer tokenizes Turtle source text.
package lexer

import (
	"strings"

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
}

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
	before := l.pos
	l.skipWhitespaceAndComments()

	tok := token.Token{Line: l.line, SpaceBefore: l.pos != before || l.pos == 0}
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
		if isLetter(l.ch) {
			tok.Literal = l.readIdentifier()
			// `sys` is only ever a leading statement keyword (see
			// SPEC.md): when it starts a line, the rest of that line is
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
	for isLetter(l.ch) || isDigit(l.ch) {
		l.readChar()
	}
	return l.input[start:l.pos]
}

func (l *Lexer) readNumber() (token.Type, string) {
	start := l.pos
	for isDigit(l.ch) {
		l.readChar()
	}
	if l.ch == '.' && isDigit(l.peekChar()) {
		l.readChar()
		for isDigit(l.ch) {
			l.readChar()
		}
		return token.FLOAT, l.input[start:l.pos]
	}
	return token.INT, l.input[start:l.pos]
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
		}
		l.readChar()
	}
	l.readChar() // skip closing quote
	return sb.String()
}

func isLetter(ch byte) bool {
	return 'a' <= ch && ch <= 'z' || 'A' <= ch && ch <= 'Z' || ch == '_'
}

func isDigit(ch byte) bool {
	return '0' <= ch && ch <= '9'
}

// Input is the source text being read, for messages that quote a line.
func (l *Lexer) Input() string { return l.input }
