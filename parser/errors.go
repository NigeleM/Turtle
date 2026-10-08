package parser

import (
	"fmt"
	"strings"

	"Turtle/ast"
	"Turtle/token"
)

// Parse errors that say what's wrong in plain words, where, and, for the
// common mistakes, how to fix it:
//
//	report.trt, line 2: this line needs a '.' at the end
//	  2 | show x
//	    |       ^
//
// Only the first error is worth showing: the parser trips over what
// follows a mistake, so the rest are usually echoes of it.

// Error is one parse error: its line, the byte offset it points at in the
// source, and the message.
type Error struct {
	Line int
	Pos  int
	Msg  string
}

func (e Error) String() string { return fmt.Sprintf("line %d: %s", e.Line, e.Msg) }

// ErrorList is the errors with their positions; Errors gives the same as
// "line N: message" text.
func (p *Parser) ErrorList() []Error { return p.errs }

// errorAt records an error at a byte offset on a line.
func (p *Parser) errorAt(line, pos int, format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	if p.multilineString > 0 && line == p.multilineEnd {
		// An error right where a "..." over several lines ends: the quotes
		// most likely paired up wrong, so report the text instead.
		msg = fmt.Sprintf("this text runs over several lines: is its closing quote missing? (then line %d: %s)", line, msg)
		line, pos = p.multilineString, p.multilinePos
		p.multilineString = 0
	}
	p.errs = append(p.errs, Error{Line: line, Pos: pos, Msg: msg})
	p.errors = append(p.errors, fmt.Sprintf("line %d: %s", line, msg))
}

// endLine is the line a token ends on: a text over several lines ends
// below where it starts.
func (p *Parser) endLine(t token.Token) int {
	src := p.l.Source()
	if t.End > len(src) || t.Pos > t.End {
		return t.Line
	}
	return t.Line + strings.Count(src[t.Pos:t.End], "\n")
}

// describe names a token for a message: the end of the line or file, a
// word in quotes, a piece of text.
func describe(t token.Token) string {
	switch t.Type {
	case token.EOF:
		return "the end of the file"
	case token.STRING:
		s := t.Literal
		if len(s) > 20 {
			s = s[:17] + "..."
		}
		return fmt.Sprintf("the text %q", s)
	case token.RAWSTRING:
		return "a `text`"
	}
	return fmt.Sprintf("'%s'", t.Literal)
}

// expectError is expectPeek's error: what was wanted, in words, with a
// fix for the usual mistakes. It points at the token found, or at the end
// of the line when the line ended first.
func (p *Parser) expectError(want token.Type) {
	cur, next := p.curToken, p.peekToken
	line, pos := next.Line, next.Pos
	lineEnded := next.Type == token.EOF || next.Line != p.endLine(cur)
	if lineEnded {
		line, pos = p.endLine(cur), cur.End
	}
	found := describe(next)
	if lineEnded && next.Type != token.EOF {
		found = "the end of the line"
	}
	switch {
	case want == token.PERIOD && lineEnded:
		p.errorAt(line, pos, "this line needs a '.' at the end")
	case want == token.PERIOD && next.Type == token.LPAREN && cur.Type == token.IDENT && !next.SpaceBefore:
		p.errorAt(line, pos, "calls use [ ], not ( ): write %s[...]", cur.Literal)
	case want == token.PERIOD:
		p.errorAt(line, pos, "expected '.' to end the statement, found %s", found)
	case want == token.RBRACKET && cur.Type == token.IF:
		p.errorAt(line, pos, "an if is written: if ] condition [ (found %s after if)", found)
	case want == token.RBRACKET && cur.Type == token.ELSE:
		p.errorAt(line, pos, "else is written: else ]  (or else if ] condition [)")
	case want == token.LBRACKET && next.Type == token.ASSIGN:
		p.errorAt(line, pos, "to compare, use == (= sets a variable)")
	case want == token.RBRACKET && lineEnded:
		p.errorAt(line, pos, "a '[' on this line isn't closed: add ']'")
	case want == token.IDENT && cur.Type == token.DEF && next.Type == token.LBRACKET:
		p.errorAt(cur.Line, cur.Pos, "def [end] with no def above it to close (or a def is missing its name: def name[params])")
	default:
		p.errorAt(line, pos, "expected %s, found %s", wanted(want), found)
	}
}

// wanted names a token type a statement needed.
func wanted(t token.Type) string {
	switch t {
	case token.IDENT:
		return "a name"
	case token.PERIOD, token.LBRACKET, token.RBRACKET, token.COLON, token.COMMA, token.ASSIGN, token.SEMI:
		return "'" + string(t) + "'"
	}
	return strings.ToLower(string(t))
}

// statementStartError is the error for a token that can't start a
// statement, with a fix for what people bring from other languages.
func (p *Parser) statementStartError() {
	cur, prev := p.curToken, p.prevToken
	switch {
	case cur.Type == token.IDENT && prev.Type == token.PERIOD && !cur.SpaceBefore && prev.Line == cur.Line:
		p.errorAt(cur.Line, prev.Pos, "Turtle has no dot-calls: write value at %s (a method) or %s[value] (a function)", cur.Literal, cur.Literal)
	case cur.Type == token.IDENT && (cur.Literal == "for" || cur.Literal == "while" || cur.Literal == "foreach"):
		p.errorAt(cur.Line, cur.Pos, "loops are written [loop][x in nums] (or [loop][condition]) ... [loop][end]")
	case cur.Type == token.IDENT && (cur.Literal == "function" || cur.Literal == "func" || cur.Literal == "fn"):
		p.errorAt(cur.Line, cur.Pos, "functions are written def name[params] ... def [end]")
	case cur.Type == token.IDENT && p.peekTokenIs(token.LPAREN) && !p.peekToken.SpaceBefore:
		p.errorAt(cur.Line, p.peekToken.Pos, "calls use [ ], not ( ): write %s[...]", cur.Literal)
	case cur.Type == token.IDENT:
		p.errorAt(cur.Line, cur.Pos, "a line can't start with %q here: did you mean %s = ..., %s[...], or show %s .?", cur.Literal, cur.Literal, cur.Literal, cur.Literal)
	case cur.Type == token.ELSE:
		p.errorAt(cur.Line, cur.Pos, "else with no if ] condition [ above it")
	case cur.Type == token.RBRACKET || cur.Type == token.LBRACKET:
		p.errorAt(cur.Line, cur.Pos, "a line can't start with %s here", describe(cur))
	default:
		p.errorAt(cur.Line, cur.Pos, "a line can't start with %s", describe(cur))
	}
}

// noValueError: a value was needed (after =, an operator, show ...) and
// the token there can't start one.
func (p *Parser) noValueError() {
	cur, prev := p.curToken, p.prevToken
	switch {
	case (cur.Type == token.EOF || cur.Line != prev.Line) && prev.Line > 0:
		p.errorAt(prev.Line, prev.End, "a value is missing at the end of this line")
	case cur.Type == token.PERIOD || cur.Type == token.EOF:
		p.errorAt(cur.Line, cur.Pos, "a value is missing before %s", describe(cur))
	case cur.Type == token.ASSIGN:
		p.errorAt(cur.Line, cur.Pos, "'=' sets a variable and can't go here: to compare, use ==")
	case cur.Type == token.LPAREN || cur.Type == token.RPAREN:
		p.errorAt(cur.Line, cur.Pos, "a value is missing before %s", describe(cur))
	default:
		p.errorAt(cur.Line, cur.Pos, "expected a value, found %s", describe(cur))
	}
}

// unclosedError: a block that opens at open never got its closer. It
// points at the opening line, where the fix starts.
func (p *Parser) unclosedError(open token.Token, what, closer string) {
	if p.curToken.Type == token.EOF {
		p.errorAt(open.Line, open.Pos, "%s is never closed: add %s after its last line", what, closer)
		return
	}
	p.errorAt(open.Line, open.Pos, "%s is never closed: add %s before line %d (found %s there)", what, closer, p.curToken.Line, describe(p.curToken))
}

// Format writes an error the way turtle shows it: the message, then the
// line with a ^ under the spot.
//
//	line 2: this line needs a '.' at the end
//	  2 | show x
//	    |       ^
func Format(src string, e Error) string {
	lines := strings.Split(strings.ReplaceAll(src, "\r\n", "\n"), "\n")
	out := e.String()
	if e.Line < 1 || e.Line > len(lines) {
		return out
	}
	text := lines[e.Line-1]
	start := 0 // the line's offset in src
	for i := 0; i < e.Line-1; i++ {
		start += len(lines[i]) + 1
		if strings.Contains(src, "\r\n") {
			start++
		}
	}
	col := e.Pos - start
	if col < 0 || col > len(text) {
		col = len(strings.TrimRight(text, " \t"))
	}
	// The ^ lines up under the same characters, tabs included.
	var pad strings.Builder
	for _, r := range text[:col] {
		if r == '\t' {
			pad.WriteByte('\t')
		} else {
			pad.WriteByte(' ')
		}
	}
	num := fmt.Sprint(e.Line)
	gutter := strings.Repeat(" ", len(num))
	return fmt.Sprintf("%s\n  %s | %s\n  %s | %s^", out, num, text, gutter, pad.String())
}

// Warnings are spots that are valid but probably don't do what they
// seem to, like "a / b at round[1]" (which rounds only b). Programs run
// as written; turtle and editors show them as notes.
func (p *Parser) Warnings() []Error { return p.warnings }

func (p *Parser) warnAt(line, pos int, format string, args ...interface{}) {
	p.warnings = append(p.warnings, Error{Line: line, Pos: pos, Msg: fmt.Sprintf(format, args...)})
}

// finishingMethods usually finish a result (rounding, formatting); after
// an operator they apply only to the value right before them, which is
// rarely what's meant.
var finishingMethods = map[string]bool{"round": true, "floor": true, "ceil": true, "fixed": true, "commas": true}

// checkMethodOnRight warns about "a / b at round[1]": the method works on
// b only, not on a / b.
func (p *Parser) checkMethodOnRight(op token.Token, left, right ast.Expression) {
	if _, perTerm := left.(*ast.MethodCallExpression); perTerm {
		return // "x at floor + y at floor": each term on purpose
	}
	switch op.Literal {
	case "+", "-", "*", "/", "div", "%":
	default:
		return
	}
	mc, ok := right.(*ast.MethodCallExpression)
	if !ok || !finishingMethods[mc.Method] {
		return
	}
	what := "the value just before it"
	switch r := mc.Receiver.(type) {
	case *ast.Identifier:
		what = r.Value
	case *ast.IntegerLiteral, *ast.FloatLiteral:
		return // "x * 2 at fixed[1]" on a literal is clear enough
	}
	p.warnAt(mc.Token.Line, mc.Token.Pos, "at %s works on %s only, not on the whole %s; to %s the whole value, store it first: v = ... %s %s, then v at %s",
		mc.Method, what, op.Literal+" expression", mc.Method, op.Literal, what, mc.Method)
}
