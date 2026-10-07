package parser

import (
	"strings"
	"testing"

	"Turtle/lexer"
)

// The first parse error for the usual mistakes: what it says, and the line
// and column it points at (from 1).
func TestFriendlyParseErrors(t *testing.T) {
	cases := []struct {
		name, src, want string
		line, col       int
	}{
		{"missing period", "x = 1\nshow x\nshow 2 .\n", "this line needs a '.' at the end", 2, 7},
		{"if without ]", "x = 1\nif x > 0 [\n  show x .\nif [end]\n", "an if is written: if ] condition [", 2, 4},
		{"if with ( )", "x = 1\nif (x > 0) [\n  show x .\nif [end]\n", "an if is written: if ] condition [ (found '(' after if)", 2, 4},
		{"= to compare", "x = 1\nif ] x = 1 [\n  show x .\nif [end]\n", "to compare, use ==", 2, 8},
		{"else [", "x = 1\nif ] x > 0 [\n  show 1 .\nelse [\n  show 2 .\nif [end]\n", "else is written: else ]", 4, 6},
		{"dot-call", "x = \"a\"\nshow x.upper() .\n", "Turtle has no dot-calls: write value at upper", 2, 7},
		{"( ) call", "def f[a]\n  return a\ndef [end]\nshow f(1) .\n", "calls use [ ], not ( ): write f[...]", 4, 7},
		{"( ) call statement", "f(1)\n", "calls use [ ], not ( ): write f[...]", 1, 2},
		{"for loop", "for x in list [1] [\n  show x .\n]\n", "loops are written [loop][x in nums]", 1, 1},
		{"def never closed", "def f[a]\n  return a\nshow f[1] .\n", "the function f is never closed: add def [end]", 1, 1},
		{"if never closed", "x = 1\nif ] x > 0 [\n  show x .\n", "this if is never closed: add if [end]", 2, 1},
		{"loop never closed", "[loop][x in list [1]]\n  show x .\n", "this loop is never closed: add [loop][end]", 1, 1},
		{"stray def [end]", "show 1 .\ndef [end]\n", "def [end] with no def above it", 2, 1},
		{"list not closed", "x = list [1, 2\nshow x .\n", "a '[' on this line isn't closed: add ']'", 1, 15},
		{"text not closed", "show \"hi .\n", "this text never closes: add the closing \"", 1, 6},
		{"quotes paired wrong", "show \"a .\nx = 2\nshow \"b\" .\n", "is its closing quote missing?", 1, 6},
		{"stray else", "else ]\n", "else with no if ] condition [ above it", 1, 1},
	}
	for _, c := range cases {
		p := New(lexer.New(c.src))
		p.ParseProgram()
		errs := p.ErrorList()
		if len(errs) == 0 {
			t.Errorf("%s: no error", c.name)
			continue
		}
		e := errs[0]
		if !strings.Contains(e.Msg, c.want) {
			t.Errorf("%s: got %q, want it to say %q", c.name, e.Msg, c.want)
		}
		lines := strings.Split(c.src, "\n")
		start := 0
		for i := 0; i < e.Line-1; i++ {
			start += len(lines[i]) + 1
		}
		if e.Line != c.line || e.Pos-start+1 != c.col {
			t.Errorf("%s: points at line %d column %d, want line %d column %d", c.name, e.Line, e.Pos-start+1, c.line, c.col)
		}
	}
}

func TestFormat(t *testing.T) {
	src := "x = 1\n\tshow x\nshow 2 .\n"
	p := New(lexer.New(src))
	p.ParseProgram()
	got := Format(src, p.ErrorList()[0])
	want := "line 2: this line needs a '.' at the end\n  2 | \tshow x\n    | \t      ^"
	if got != want {
		t.Errorf("got\n%s\nwant\n%s", got, want)
	}
}

// Text over several lines is fine: SQL, say. Lines after it count on, and
// a mistake further down isn't blamed on it.
func TestTextOverSeveralLines(t *testing.T) {
	src := "rows = query[db, \"SELECT item\n    FROM orders\n    WHERE qty > 1\"]\nmin = 3\n"
	p := New(lexer.New(src))
	p.ParseProgram()
	errs := p.ErrorList()
	if len(errs) == 0 || errs[0].Line != 4 || !strings.Contains(errs[0].Msg, "reserved word") {
		t.Errorf("got %v, want the reserved word on line 4", errs)
	}
	src = "show \"a .\nx = 2\nshow \"b\" .\nshow 1 .\n"
	p = New(lexer.New(src))
	p.ParseProgram()
	if e := p.ErrorList()[0]; e.Line != 1 || !strings.Contains(e.Msg, "closing quote missing? (then line 3") {
		t.Errorf("quotes paired wrong: got %v", e)
	}
}

// An is line is a sentence: it ends with a period, and the error says
// = needs none.
func TestIsNeedsPeriod(t *testing.T) {
	for _, src := range []string{"b is a process x give x * 0.2\nshow b .\n", "u is \"a\" at upper\n", "x is 1"} {
		p := New(lexer.New(src))
		p.ParseProgram()
		errs := p.ErrorList()
		if len(errs) == 0 || errs[0].Line != 1 || !strings.Contains(errs[0].Msg, "a line with is ends with '.' (or write name = value, which needs none)") {
			t.Errorf("%q: got %v", src, errs)
		}
	}
	p := New(lexer.New("x is 1 .\ny is x at upper .\n"))
	p.ParseProgram()
	if errs := p.ErrorList(); len(errs) > 0 {
		t.Errorf("got %v", errs)
	}
}
