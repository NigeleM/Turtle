package parser

import (
	"strings"
	"testing"

	"Turtle/ast"
	"Turtle/lexer"
	"Turtle/syntax"
)

// parseOK parses src and fails the test immediately if there were any
// parse errors, returning the resulting program for further assertions.
func parseOK(t *testing.T, src string) *ast.Program {
	t.Helper()
	p := New(lexer.New(src))
	program := p.ParseProgram()
	if errs := p.Errors(); len(errs) > 0 {
		t.Fatalf("unexpected parse error(s) for %q: %v", src, errs)
	}
	return program
}

func TestParseAssignmentAndShow(t *testing.T) {
	program := parseOK(t, `x = 1 + 2 * 3
show x .`)
	if len(program.Statements) != 2 {
		t.Fatalf("got %d statements, want 2", len(program.Statements))
	}
	as, ok := program.Statements[0].(*ast.AssignStatement)
	if !ok {
		t.Fatalf("statement 0 is %T, want *ast.AssignStatement", program.Statements[0])
	}
	if as.Name != "x" {
		t.Errorf("assign name = %q, want %q", as.Name, "x")
	}
	infix, ok := as.Value.(*ast.InfixExpression)
	if !ok || infix.Operator != "+" {
		t.Fatalf("value is %#v, want a top-level '+' infix (precedence: * binds tighter)", as.Value)
	}
	if _, ok := program.Statements[1].(*ast.ShowStatement); !ok {
		t.Errorf("statement 1 is %T, want *ast.ShowStatement", program.Statements[1])
	}
}

func TestParseIfElseReversedBrackets(t *testing.T) {
	program := parseOK(t, `if ] x > 10 [
    show "big" .
else if ] x > 0 [
    show "small" .
else ]
    show "non-positive" .
if [end]`)
	if len(program.Statements) != 1 {
		t.Fatalf("got %d statements, want 1", len(program.Statements))
	}
	ifs, ok := program.Statements[0].(*ast.IfStatement)
	if !ok {
		t.Fatalf("statement is %T, want *ast.IfStatement", program.Statements[0])
	}
	if len(ifs.Clauses) != 3 {
		t.Fatalf("got %d clauses, want 3 (if / else-if / else)", len(ifs.Clauses))
	}
	if ifs.Clauses[0].Condition == nil {
		t.Error("clause 0 (if) should have a condition")
	}
	if ifs.Clauses[1].Condition == nil {
		t.Error("clause 1 (else if) should have a condition")
	}
	if ifs.Clauses[2].Condition != nil {
		t.Error("clause 2 (trailing else) should have a nil condition")
	}
}

func TestParseLoopHeaders(t *testing.T) {
	program := parseOK(t, `[loop][i = 0; i < 5; i++]
    show i .
[loop][end]`)
	ls, ok := program.Statements[0].(*ast.LoopStatement)
	if !ok {
		t.Fatalf("statement is %T, want *ast.LoopStatement", program.Statements[0])
	}
	if ls.Kind != ast.LoopCStyle {
		t.Errorf("got loop kind %v, want LoopCStyle", ls.Kind)
	}
	if ls.Init == nil || ls.Condition == nil || ls.Post == nil {
		t.Error("C-style loop should have Init, Condition, and Post all set")
	}

	program2 := parseOK(t, `[loop][count < 3]
    count = count + 1
[loop][end]`)
	ls2, ok := program2.Statements[0].(*ast.LoopStatement)
	if !ok {
		t.Fatalf("statement is %T, want *ast.LoopStatement", program2.Statements[0])
	}
	if ls2.Kind != ast.LoopWhile {
		t.Errorf("got loop kind %v, want LoopWhile", ls2.Kind)
	}
	if ls2.Init != nil || ls2.Post != nil {
		t.Error("while-style loop should have nil Init and Post")
	}
}

func TestParseFunctionDefAndCall(t *testing.T) {
	// "add" itself is a reserved word (the data-op statement), so the
	// function here is named "addition" — same workaround the language's
	// own testdata/calculator.turtle needs.
	program := parseOK(t, `def addition[a, b]
    return a + b
def [end]

sum = addition[1, 2]`)
	fd, ok := program.Statements[0].(*ast.FunctionDefStatement)
	if !ok {
		t.Fatalf("statement 0 is %T, want *ast.FunctionDefStatement", program.Statements[0])
	}
	if fd.Name != "addition" || len(fd.Parameters) != 2 {
		t.Errorf("got name=%q params=%v, want name=addition params=[a b]", fd.Name, fd.Parameters)
	}

	as, ok := program.Statements[1].(*ast.AssignStatement)
	if !ok {
		t.Fatalf("statement 1 is %T, want *ast.AssignStatement", program.Statements[1])
	}
	call, ok := as.Value.(*ast.CallExpression)
	if !ok || call.Name != "addition" || len(call.Arguments) != 2 {
		t.Errorf("value is %#v, want a call to addition with 2 arguments", as.Value)
	}
}

func TestParseIsAtMethodCall(t *testing.T) {
	program := parseOK(t, `r is nums at get 0 .`)
	as, ok := program.Statements[0].(*ast.AssignStatement)
	if !ok {
		t.Fatalf("statement is %T, want *ast.AssignStatement (is-statements lower to assignment)", program.Statements[0])
	}
	mc, ok := as.Value.(*ast.MethodCallExpression)
	if !ok {
		t.Fatalf("value is %T, want *ast.MethodCallExpression", as.Value)
	}
	if mc.Method != "get" || len(mc.Arguments) != 1 {
		t.Errorf("got method=%q args=%d, want method=get args=1", mc.Method, len(mc.Arguments))
	}
}

func TestParseChangeStatementRequiresIdentifier(t *testing.T) {
	program := parseOK(t, `change a to integer .`)
	as, ok := program.Statements[0].(*ast.AssignStatement)
	if !ok {
		t.Fatalf("statement is %T, want *ast.AssignStatement (change-statement lowers to self-assignment)", program.Statements[0])
	}
	if as.Name != "a" {
		t.Errorf("assign target = %q, want %q", as.Name, "a")
	}
	ce, ok := as.Value.(*ast.ChangeExpression)
	if !ok {
		t.Fatalf("value is %T, want *ast.ChangeExpression", as.Value)
	}
	if ce.TypeName != "integer" {
		t.Errorf("got type name %q, want %q", ce.TypeName, "integer")
	}
}

func TestParseAdditionKeywordDataOp(t *testing.T) {
	program := parseOK(t, `add 4 to nums .`)
	dop, ok := program.Statements[0].(*ast.DataOpStatement)
	if !ok {
		t.Fatalf("statement is %T, want *ast.DataOpStatement", program.Statements[0])
	}
	if dop.Kind != ast.OpAdd || dop.Target != "nums" {
		t.Errorf("got kind=%v target=%q, want OpAdd target=nums", dop.Kind, dop.Target)
	}
}

func TestParseErrorsReportLineNumbers(t *testing.T) {
	p := New(lexer.New("x = 1\nshow x\nb = 2"))
	p.ParseProgram()
	errs := p.Errors()
	if len(errs) == 0 {
		t.Fatal("expected a parse error for a missing 'show' period, got none")
	}
	if !strings.Contains(errs[0], "line 2") {
		t.Errorf("error %q does not mention line 2", errs[0])
	}
}

func TestImportList(t *testing.T) {
	p := New(lexer.New("import time [sleep, now]\n"))
	prog := p.ParseProgram()
	if errs := p.Errors(); len(errs) > 0 {
		t.Fatalf("parse errors: %v", errs)
	}
	is, ok := prog.Statements[0].(*ast.ImportStatement)
	if !ok || is.Path != "time" || len(is.Names) != 2 || is.Names[0] != "sleep" || is.Names[1] != "now" {
		t.Fatalf("got %#v", prog.Statements[0])
	}
}

func TestEmptyImportListIsAnError(t *testing.T) {
	p := New(lexer.New("import time []\n"))
	p.ParseProgram()
	if errs := p.Errors(); len(errs) != 1 || !strings.Contains(errs[0], "lists nothing to import") {
		t.Fatalf("want exactly one 'lists nothing' error, got %v", errs)
	}
}

func TestQualifiedCall(t *testing.T) {
	p := New(lexer.New("t = time now[]\n"))
	prog := p.ParseProgram()
	if errs := p.Errors(); len(errs) > 0 {
		t.Fatalf("parse errors: %v", errs)
	}
	ce, ok := prog.Statements[0].(*ast.AssignStatement).Value.(*ast.CallExpression)
	if !ok || ce.Module != "time" || ce.Name != "now" {
		t.Fatalf("got %#v", prog.Statements[0].(*ast.AssignStatement).Value)
	}
}

func TestAssembleDuplicateFieldIsAnError(t *testing.T) {
	p := New(lexer.New("assemble A [x, x]\n"))
	p.ParseProgram()
	if errs := p.Errors(); len(errs) != 1 || !strings.Contains(errs[0], `field "x" listed twice`) {
		t.Fatalf("want one duplicate-field error, got %v", errs)
	}
}

func TestIntegerLiteralPastLimit(t *testing.T) {
	p := New(lexer.New("show 99999999999999999999 .\n"))
	p.ParseProgram()
	if errs := p.Errors(); len(errs) == 0 || !strings.Contains(errs[0], "past the integer limits") {
		t.Fatalf("want a limits error, got %v", errs)
	}
}

func TestReservedWordAsNameIsOneClearError(t *testing.T) {
	cases := map[string]string{
		"max = 5\nshow 1 .\n":                             "a variable",
		"total is 5 .\nmin is 3 .\n":                      "a variable",
		"def show[a]\n    return a\ndef [end]\n":          "a function",
		"def f[list]\n    return 1\ndef [end]\n":          "a parameter",
		"assemble Order [item, length]\n":                 "a field",
		"assemble list [a]\n":                             "an assembled type",
		"[loop][max in list [1]]\n[loop][end]\n":          "a loop variable",
		"[loop][i, max in list [1]]\n[loop][end]\n":       "a loop variable",
		"safe\n    x = 1\nhandle [] list .\nsafe [end]\n": "an error variable",
	}
	for src, what := range cases {
		p := New(lexer.New(src))
		p.ParseProgram()
		errs := p.Errors()
		if len(errs) != 1 || !strings.Contains(errs[0], "is a reserved word, so it can't be used as "+what+" name") {
			t.Errorf("%q: want one %q error, got %v", src, what, errs)
		}
	}
}

func TestParseSafeHandle(t *testing.T) {
	prog := parseOK(t, `safe
    x = 1 / 0
    show x .
handle [math, file] problem .
    show problem .
safe [end]
show "after" .`)
	if len(prog.Statements) != 2 {
		t.Fatalf("want 2 statements, got %d", len(prog.Statements))
	}
	ss, ok := prog.Statements[0].(*ast.SafeStatement)
	if !ok || len(ss.Body.Statements) != 2 || len(ss.Handler.Statements) != 1 || ss.Name != "problem" ||
		len(ss.Kinds) != 2 || ss.Kinds[0] != "math" || ss.Kinds[1] != "file" {
		t.Fatalf("got %#v", prog.Statements[0])
	}
	ss = parseOK(t, "safe\n    x = 1\nhandle [] e .\nsafe [end]\n").Statements[0].(*ast.SafeStatement)
	if len(ss.Kinds) != 0 || ss.Name != "e" || len(ss.Handler.Statements) != 0 {
		t.Fatalf("got %#v", ss)
	}
	if _, ok := parseOK(t, `fail "no " + x`).Statements[0].(*ast.FailStatement); !ok {
		t.Fatal("want a FailStatement")
	}
}

func TestParseSafeHandleErrors(t *testing.T) {
	cases := map[string]string{
		"handle [] e .\n":                          "'handle' needs a 'safe' above it",
		"safe\n    x = 1\nshow x .\n":              "needs a 'handle [...] error .' line to close it",
		"safe\n    x = 1\nhandle [maths] e .\n":    `"maths" isn't a kind of error`,
		"safe\n    x = 1\nhandle [] e\nshow 1 .\n": "this line needs a '.' at the end",
		"safe [end]\n":                             "'safe [end]' needs a 'safe' block",
		"fail\nshow 1 .\n":                         "'fail' needs a message",
	}
	for src, want := range cases {
		p := New(lexer.New(src))
		p.ParseProgram()
		errs := p.Errors()
		if len(errs) == 0 || !strings.Contains(errs[0], want) {
			t.Errorf("%q: want error containing %q, got %v", src, want, errs)
		}
	}
}

func TestParseSubfolderImport(t *testing.T) {
	is, ok := parseOK(t, "import lib/text/utils [a]\n").Statements[0].(*ast.ImportStatement)
	if !ok || is.Path != "lib/text/utils" || len(is.Names) != 1 {
		t.Fatalf("got %#v", is)
	}
}

func TestParseInterpolation(t *testing.T) {
	show := parseOK(t, `show "a {x + 1} b\{c" .`).Statements[0].(*ast.ShowStatement)
	is, ok := show.Expressions[0].(*ast.InterpolatedString)
	if !ok || len(is.Parts) != 3 {
		t.Fatalf("got %#v", show.Expressions[0])
	}
	if last := is.Parts[2].(*ast.StringLiteral).Value; last != " b{c" {
		t.Errorf("last part %q", last)
	}
	if _, ok := is.Parts[1].(*ast.InfixExpression); !ok {
		t.Errorf("middle part %#v", is.Parts[1])
	}
	cases := map[string]string{
		`show "a {b" .`:    "needs a closing '}'",
		`show "a {1 +}" .`: "the expression isn't finished",
		`show "a {1 2}" .`: "unexpected",
	}
	for src, want := range cases {
		p := New(lexer.New(src))
		p.ParseProgram()
		if errs := p.Errors(); len(errs) == 0 || !strings.Contains(errs[0], want) {
			t.Errorf("%q: want %q, got %v", src, want, errs)
		}
	}
}

func TestParsePlainBraces(t *testing.T) {
	cases := map[string]string{
		`show "{}" .`:                  "{}",
		`show "a { } b" .`:             "a { } b",
		`show '{"a": 1}' .`:            `{"a": 1}`,
		`show "{\"a\": {\"b\": 1}}" .`: `{"a": {"b": 1}}`,
		"show '{\n  \"a\": 1\n}' .":    "{\n  \"a\": 1\n}",
		`show "end {" .`:               "end {",
	}
	for src, want := range cases {
		show := parseOK(t, src).Statements[0].(*ast.ShowStatement)
		lit, ok := show.Expressions[0].(*ast.StringLiteral)
		if !ok || lit.Value != want {
			t.Errorf("%s: want plain %q, got %#v", src, want, show.Expressions[0])
		}
	}
	// A brace before anything else still interpolates, spaces allowed.
	show := parseOK(t, `show '{"k": {x}, "y": { y }}' .`).Statements[0].(*ast.ShowStatement)
	is, ok := show.Expressions[0].(*ast.InterpolatedString)
	if !ok || len(is.Parts) != 5 {
		t.Fatalf("got %#v", show.Expressions[0])
	}
	if first := is.Parts[0].(*ast.StringLiteral).Value; first != `{"k": ` {
		t.Errorf("first part %q", first)
	}
}

// TestGivesIsNowGive: code written with the old word says how to fix it.
func TestGivesIsNowGive(t *testing.T) {
	p := New(lexer.New("f = x gives x + 1"))
	p.ParseProgram()
	if errs := strings.Join(p.Errors(), "\n"); !strings.Contains(errs, "x gives ...: the word is give now: x give ...") {
		t.Errorf("got %q", errs)
	}
	p = New(lexer.New("f = x give x + 1"))
	p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Errorf("give: %v", p.Errors())
	}
}

// TestShapeWordsMatch: the words editors complete after random are the
// ones the parser reads.
func TestShapeWordsMatch(t *testing.T) {
	if len(syntax.ShapeWords) != len(shapeWords) {
		t.Fatalf("syntax.ShapeWords has %d words, the parser %d", len(syntax.ShapeWords), len(shapeWords))
	}
	for _, w := range syntax.ShapeWords {
		if _, ok := shapeWords[w]; !ok {
			t.Errorf("syntax.ShapeWords has %q, which the parser doesn't read", w)
		}
	}
}

func TestWarnings(t *testing.T) {
	cases := []struct {
		src  string
		want string // "" for no warning
	}{
		{"x = a / b at round[1]", "at round works on b only, not on the whole / expression"},
		{"x = a + b at fixed[2]", "at fixed works on b only"},
		{"x = a / b", ""},
		{"x = i at sqrt + i at round", ""}, // a method on each term
		{"x = \"a\" + s at upper", ""},     // not a finishing method
		{"x = a * 2 at fixed[1]", ""},      // on a plain number
		{"v = a / b\nx = v at round[1]", ""},
	}
	for _, c := range cases {
		p := New(lexer.New(c.src))
		p.ParseProgram()
		ws := p.Warnings()
		switch {
		case c.want == "" && len(ws) > 0:
			t.Errorf("%q: unexpected warning %v", c.src, ws)
		case c.want != "" && (len(ws) == 0 || !strings.Contains(ws[0].Msg, c.want)):
			t.Errorf("%q: got %v, want %q", c.src, ws, c.want)
		}
	}
}

// ~ marks a private function or type; anything else with it is an error.
func TestPrivateNames(t *testing.T) {
	good := "def ~limit[n]\n    return n\ndef [end]\nx = ~limit[1]\n"
	p := New(lexer.New(good))
	p.ParseProgram()
	if errs := p.Errors(); len(errs) > 0 {
		t.Errorf("private function and type: %v", errs)
	}
	for src, what := range map[string]string{
		"~x = 5":                                      "a variable",
		"~x is 5 .":                                   "a variable",
		"def f[~a]\n    return 1\ndef [end]":          "a parameter",
		"g = [~a] give ~a":                            "a parameter",
		"[loop][~i in list [1]]\n[loop][end]":         "a loop variable",
		"assemble P [~x]":                             "a field",
		"assemble ~Box [w]":                           "an assembled type",
		"[read] f.txt to ~lines [end]":                "a variable",
		"safe\n    x = 1\nhandle [] ~e .\nsafe [end]": "an error variable",
	} {
		p := New(lexer.New(src))
		p.ParseProgram()
		errs := p.Errors()
		if len(errs) == 0 || !strings.Contains(errs[0], "can't be "+what+" name: ~ marks a private function") {
			t.Errorf("%q: %v", src, errs)
		}
	}
}

// A theory's slots are the names its definition uses, however it uses
// them: as a value, a sentence's subject, a statement's target.
func TestTheorySlots(t *testing.T) {
	for def, want := range map[string]string{
		"return a + b":                             "a b",
		"return a process n give n + b":            "a b",
		"add b to a .\n        return a":           "a b",
		"x is a at get[0] .\n        return x + b": "a b",
	} {
		src := "theory joined\n    abstract\n        joined is a test.\n    notation joined a with b .\n    definition\n        " + def + "\ntheory [end]\n"
		p := New(lexer.New(src))
		p.ParseProgram()
		if errs := p.Errors(); len(errs) > 0 {
			t.Errorf("%s: %v", def, errs)
			continue
		}
		if got := strings.Join(p.theories["joined"].slots, " "); got != want {
			t.Errorf("%s: slots %q, want %q", def, got, want)
		}
	}
}
