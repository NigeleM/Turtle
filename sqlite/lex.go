package sqlite

import (
	"strings"
)

// SQL tokens.
type tokKind int

const (
	tEOF    tokKind = iota
	tIdent          // a name: books, "my table", [col], `col`
	tNumber         // 42, 3.5, 1e3, 0x1F
	tString         // 'text' ('' is a quote inside)
	tBlob           // X'ABCD'
	tParam          // ? or ?3
	tOp             // punctuation and operators: ( ) , . ; = == != <> < <= > >= || + - * / % ~
)

type token struct {
	kind   tokKind
	text   string // the identifier, number text, string value, blob hex, operator
	quoted bool   // an identifier written in quotes (never a keyword)
	pos    int    // byte offset in the SQL, for error messages and column names
	end    int
}

// lexSQL splits a SQL statement into tokens.
func lexSQL(src string) ([]token, error) {
	var toks []token
	i := 0
	for i < len(src) {
		c := src[i]
		switch {
		case c == ' ' || c == '\t' || c == '\n' || c == '\r':
			i++
		case c == '-' && i+1 < len(src) && src[i+1] == '-':
			for i < len(src) && src[i] != '\n' {
				i++
			}
		case c == '/' && i+1 < len(src) && src[i+1] == '*':
			end := strings.Index(src[i+2:], "*/")
			if end < 0 {
				return nil, errorf("SQL: a /* comment isn't closed")
			}
			i += end + 4
		case isIdentStart(c):
			start := i
			for i < len(src) && isIdentChar(src[i]) {
				i++
			}
			word := src[start:i]
			// X'ABCD' is a blob literal.
			if (word == "x" || word == "X") && i < len(src) && src[i] == '\'' {
				end := strings.IndexByte(src[i+1:], '\'')
				if end < 0 {
					return nil, errorf("SQL: a blob literal isn't closed")
				}
				toks = append(toks, token{kind: tBlob, text: src[i+1 : i+1+end], pos: start, end: i + 2 + end})
				i += end + 2
				continue
			}
			toks = append(toks, token{kind: tIdent, text: word, pos: start, end: i})
		case c == '"' || c == '`' || c == '[':
			closer := c
			if c == '[' {
				closer = ']'
			}
			start := i
			var sb strings.Builder
			i++
			for {
				if i >= len(src) {
					return nil, errorf("SQL: a quoted name isn't closed")
				}
				if src[i] == closer {
					if closer != ']' && i+1 < len(src) && src[i+1] == closer {
						sb.WriteByte(closer)
						i += 2
						continue
					}
					i++
					break
				}
				sb.WriteByte(src[i])
				i++
			}
			toks = append(toks, token{kind: tIdent, text: sb.String(), quoted: true, pos: start, end: i})
		case c == '\'':
			start := i
			var sb strings.Builder
			i++
			for {
				if i >= len(src) {
					return nil, errorf("SQL: a 'text' value isn't closed")
				}
				if src[i] == '\'' {
					if i+1 < len(src) && src[i+1] == '\'' {
						sb.WriteByte('\'')
						i += 2
						continue
					}
					i++
					break
				}
				sb.WriteByte(src[i])
				i++
			}
			toks = append(toks, token{kind: tString, text: sb.String(), pos: start, end: i})
		case isDigit(c) || (c == '.' && i+1 < len(src) && isDigit(src[i+1])):
			start := i
			if c == '0' && i+1 < len(src) && (src[i+1] == 'x' || src[i+1] == 'X') {
				i += 2
				for i < len(src) && isHex(src[i]) {
					i++
				}
			} else {
				for i < len(src) && isDigit(src[i]) {
					i++
				}
				if i < len(src) && src[i] == '.' {
					i++
					for i < len(src) && isDigit(src[i]) {
						i++
					}
				}
				if i < len(src) && (src[i] == 'e' || src[i] == 'E') {
					j := i + 1
					if j < len(src) && (src[j] == '+' || src[j] == '-') {
						j++
					}
					if j < len(src) && isDigit(src[j]) {
						i = j
						for i < len(src) && isDigit(src[i]) {
							i++
						}
					}
				}
			}
			toks = append(toks, token{kind: tNumber, text: src[start:i], pos: start, end: i})
		case c == '?':
			start := i
			i++
			for i < len(src) && isDigit(src[i]) {
				i++
			}
			toks = append(toks, token{kind: tParam, text: src[start:i], pos: start, end: i})
		default:
			start := i
			two := ""
			if i+1 < len(src) {
				two = src[i : i+2]
			}
			switch two {
			case "==", "!=", "<>", "<=", ">=", "||", "<<", ">>":
				toks = append(toks, token{kind: tOp, text: two, pos: start, end: i + 2})
				i += 2
				continue
			}
			if strings.IndexByte("(),.;=<>+-*/%~&|", c) < 0 {
				return nil, errorf("SQL: unexpected character %q", string(c))
			}
			toks = append(toks, token{kind: tOp, text: string(c), pos: start, end: i + 1})
			i++
		}
	}
	toks = append(toks, token{kind: tEOF, pos: len(src), end: len(src)})
	return toks, nil
}

func isIdentStart(c byte) bool {
	return c == '_' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= 0x80
}
func isIdentChar(c byte) bool { return isIdentStart(c) || isDigit(c) || c == '$' }
func isDigit(c byte) bool     { return c >= '0' && c <= '9' }
func isHex(c byte) bool {
	return isDigit(c) || c >= 'a' && c <= 'f' || c >= 'A' && c <= 'F'
}
