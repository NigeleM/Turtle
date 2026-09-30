package lexer

import (
	"testing"

	"Turtle/token"
)

func TestNextTokenCoreSyntax(t *testing.T) {
	input := `x = 5
y = 3.14
name = "Nigele\n"
a = 1 + 2 - 3 * 4 / 5 % 6
if ] x > 10 [
show x .
if [end]
[loop][i = 0; i < 5; i++]
[loop][end]
change x to integer .
def addition[a, b]
def [end]
// a comment
//* block
   comment *//
`
	want := []struct {
		typ     token.Type
		literal string
	}{
		{token.IDENT, "x"}, {token.ASSIGN, "="}, {token.INT, "5"},
		{token.IDENT, "y"}, {token.ASSIGN, "="}, {token.FLOAT, "3.14"},
		{token.IDENT, "name"}, {token.ASSIGN, "="}, {token.STRING, "Nigele\n"},
		{token.IDENT, "a"}, {token.ASSIGN, "="},
		{token.INT, "1"}, {token.PLUS, "+"}, {token.INT, "2"},
		{token.MINUS, "-"}, {token.INT, "3"}, {token.ASTERISK, "*"},
		{token.INT, "4"}, {token.SLASH, "/"}, {token.INT, "5"},
		{token.PERCENT, "%"}, {token.INT, "6"},
		{token.IF, "if"}, {token.RBRACKET, "]"}, {token.IDENT, "x"},
		{token.GT, ">"}, {token.INT, "10"}, {token.LBRACKET, "["},
		{token.SHOW, "show"}, {token.IDENT, "x"}, {token.PERIOD, "."},
		{token.IF, "if"}, {token.LBRACKET, "["}, {token.END, "end"}, {token.RBRACKET, "]"},
		{token.LBRACKET, "["}, {token.LOOP, "loop"}, {token.RBRACKET, "]"},
		{token.LBRACKET, "["}, {token.IDENT, "i"}, {token.ASSIGN, "="}, {token.INT, "0"},
		{token.SEMI, ";"}, {token.IDENT, "i"}, {token.LT, "<"}, {token.INT, "5"},
		{token.SEMI, ";"}, {token.IDENT, "i"}, {token.INCR, "++"}, {token.RBRACKET, "]"},
		{token.LBRACKET, "["}, {token.LOOP, "loop"}, {token.RBRACKET, "]"},
		{token.LBRACKET, "["}, {token.END, "end"}, {token.RBRACKET, "]"},
		{token.CHANGE, "change"}, {token.IDENT, "x"}, {token.TO, "to"},
		{token.IDENT, "integer"}, {token.PERIOD, "."},
		{token.DEF, "def"}, {token.IDENT, "addition"}, {token.LBRACKET, "["},
		{token.IDENT, "a"}, {token.COMMA, ","}, {token.IDENT, "b"}, {token.RBRACKET, "]"},
		{token.DEF, "def"}, {token.LBRACKET, "["}, {token.END, "end"}, {token.RBRACKET, "]"},
		{token.EOF, ""},
	}

	l := New(input)
	for i, tt := range want {
		tok := l.NextToken()
		if tok.Type != tt.typ || tok.Literal != tt.literal {
			t.Fatalf("token %d: got {%s %q}, want {%s %q}", i, tok.Type, tok.Literal, tt.typ, tt.literal)
		}
	}
}

func TestLineTracking(t *testing.T) {
	input := "a = 1\nb = 2\n\nc = 3"
	l := New(input)
	var lines []int
	for {
		tok := l.NextToken()
		if tok.Type == token.EOF {
			break
		}
		if tok.Type == token.IDENT {
			lines = append(lines, tok.Line)
		}
	}
	want := []int{1, 2, 4}
	if len(lines) != len(want) {
		t.Fatalf("got %d identifier lines %v, want %v", len(lines), lines, want)
	}
	for i, l := range want {
		if lines[i] != l {
			t.Errorf("identifier %d: got line %d, want %d", i, lines[i], l)
		}
	}
}

func TestSysCapturesRestOfLineVerbatim(t *testing.T) {
	l := New("sys echo hello[world], \"quoted\"\nshow 1 .")
	tok := l.NextToken()
	if tok.Type != token.SYS {
		t.Fatalf("got token type %s, want SYS", tok.Type)
	}
	want := `echo hello[world], "quoted"`
	if tok.Literal != want {
		t.Errorf("got sys literal %q, want %q", tok.Literal, want)
	}
	next := l.NextToken()
	if next.Type != token.SHOW {
		t.Errorf("got %s after sys line, want SHOW (lexer should resume normally)", next.Type)
	}
}

func TestReservedWordsAreNotIdentifiers(t *testing.T) {
	for word, want := range map[string]token.Type{
		"show": token.SHOW, "if": token.IF, "else": token.ELSE, "def": token.DEF,
		"end": token.END, "loop": token.LOOP, "return": token.RETURN,
		"list": token.LIST, "set": token.SET, "map": token.MAP,
		"add": token.ADD, "change": token.CHANGE, "break": token.BREAK,
		"continue": token.CONTINUE, "true": token.TRUE, "false": token.FALSE,
	} {
		l := New(word)
		tok := l.NextToken()
		if tok.Type != want {
			t.Errorf("keyword %q: got token type %s, want %s", word, tok.Type, want)
		}
	}
}
