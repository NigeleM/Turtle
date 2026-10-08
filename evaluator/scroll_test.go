package evaluator

import (
	"strings"
	"testing"

	"Turtle/lexer"
	"Turtle/parser"
)

// Scrolls: steps in order, here, saved scrolls running in place, the
// errors, and diagnose.

const scrollDefs = `def add1[n]
    return n + 1
def [end]
def double[n]
    return n * 2
def [end]
def half[n]
    return n * 0.5
def [end]
def between[low, n, high]
    return n >= low && n <= high
def [end]
def shownumber[n]
    show "number ", n .
def [end]
`

func TestScroll(t *testing.T) {
	cases := []struct{ src, want string }{
		{"x is scroll 3 into add1, double, half .\nshow x .", "4.0"},
		{"x = scroll 3 into add1, double, half .\nshow x .", "4.0"},
		{"show scroll 3 into add1, double .", "8"},
		// Saved scrolls: values, named after their variable, run in place.
		{"s = scroll add1, double, half .\nshow s, \"|\", typeof[s] .", "scroll s (3 steps)|scroll"},
		{"s = scroll add1, double, half .\nx is scroll 10 into s .\nshow x .", "11.0"},
		{"s = scroll add1, double .\nt = scroll s, s .\nshow scroll 1 into t, add1 .", "11"},
		{"s = scroll add1, double .\nshow s[4] .", "10"},
		{"import data\ns = scroll add1, double .\nshow list [1, 2] process s .", "[ 4, 6 ]"},
		// The three kinds of step, and here.
		{"show scroll 7 into between[1, here, 10] .", "true"},
		{"show scroll 3 into here + 1, here * 2, here * 0.5 .", "4.0"},
		{"show scroll 3 into n give n * 5 .", "15"},
		{"import strings\nimport pattern\nshow scroll \"a,b,c\" into splitby \",\", join[\" - \"] .", "a - b - c"},
		{"import pattern\nshow scroll \"  Hello World  \" into at trim, at lower, replaceall[here, \" \", \"-\"] .", "hello-world"},
		{"import math\nshow scroll 3.14159 into at round[2] .", "3.14"},
		{"show scroll \"abc\" into at upper at len .", "3"},
		{"import data\nshow scroll list [\" a\", \"b \"] into process s give s at trim .", `[ "a", "b" ]`},
		// The whole list goes to each step.
		{"import data\nshow scroll list [1, 2, 3] into sum, here * 2 .", "12"},
		// A module's function by its qualified name.
		{"import json\nshow scroll \"[1, 2]\" into json load, here at len .", "2"},
		// Over several lines, ending at the period.
		{"import strings\nimport data\nn is scroll list [\"  ann\", \"\", \"bo  \"] into\n    keep s give s != \"\",\n    process s give s at trim,\n    join[\", \"] .\nshow n .", "ann, bo"},
		// Inside brackets, the period before the ].
		{"show typeof[scroll 1 into add1 .] .", "integer"},
		// A scroll as a statement, run for its effect.
		{"scroll 4 into here + 1 .", ""},
		// here is the nearest scroll's value; outside scrolls, a plain name.
		{"here = \"outside\"\nx is scroll 2 into here * 10 .\nshow here, \" \", x .", "outside 20"},
		{"show scroll 2 into scroll here + 1 into here * 10 ., here + 1 .", "31"},
		// none goes on like any value.
		{"def describe[v]\n    if ] v == none [\n        return \"nothing\"\n    if [end]\n    return \"something\"\ndef [end]\nshow scroll 3 into shownumber, describe .", "nothing"},
		{"x is scroll 3 into shownumber .\nshow x .", "none"},
		// return of a scroll from a function.
		{"def f[v]\n    return scroll v into add1, double .\ndef [end]\nshow f[2] .", "6"},
	}
	for _, c := range cases {
		got, err := run(t, scrollDefs+c.src, "")
		if err != nil {
			t.Errorf("%s: %v", c.src, err)
			continue
		}
		lines := strings.Split(strings.TrimSpace(got), "\n")
		if last := lines[len(lines)-1]; c.want != "" && last != c.want {
			t.Errorf("%s:\n got  %q\n want %q", c.src, got, c.want)
		}
	}
}

func TestScrollErrors(t *testing.T) {
	cases := []struct{ src, kind, want string }{
		{"x is scroll 3 into add1, shownumber, double .", "type", "scroll step 3 of 3 (double):"},
		{"total = 5\nx is scroll 3 into total .", "scroll", "scroll step 1 (total) is a number, not a function or scroll"},
		{"x is scroll 3 into add1, here / 0 .", "math", "scroll step 2 of 2 (here / 0): division by zero"},
		{"x is scroll 3 into between .", "scroll", "a function of 3 values"},
		{"s = scroll add1 .\nt = scroll s, add1 .\ns = scroll t .\nx is scroll 1 into s .", "scroll", "contains itself"},
		{"x is scroll 1 into nothing_here .", "name", `undefined function "nothing_here"`},
		// Inside a saved scroll used as a step: the inner step's number.
		{"s = scroll add1, here / 0 .\nx is scroll 1 into double, s .", "math", "scroll step 2.2 of 2 (here / 0): division by zero"},
	}
	for _, c := range cases {
		src := scrollDefs + "safe\n    " + strings.ReplaceAll(c.src, "\n", "\n    ") + "\nhandle [] e .\n    show kind of e, \"|\", message of e .\nsafe [end]"
		got, err := run(t, src, "")
		if err != nil {
			t.Errorf("%s: %v", c.src, err)
			continue
		}
		lines := strings.Split(strings.TrimSpace(got), "\n")
		last := lines[len(lines)-1]
		if !strings.HasPrefix(last, c.kind+"|") || !strings.Contains(last, c.want) {
			t.Errorf("%s: got %q, want kind %s and %q", c.src, last, c.kind, c.want)
		}
	}
}

func TestScrollParseErrors(t *testing.T) {
	cases := []struct{ src, want string }{
		{"x is scroll 3 add1, double .", `a scroll with a starting value needs "into" before its steps`},
		{"x is scroll 3 into add1, double", "a scroll ends with '.'"},
		{"x is scroll 3 into add1, , double .", "scroll step 2 is empty"},
		{"x is scroll 3 into add1, double, .", "a comma after the last step"},
		{"x is scroll today[] into add_time 30, \"days\" .", `scroll step 2 is the text "days", not a step; for two or more values use brackets: add_time[here, 30, "days"]`},
		{"x is scroll .", "a scroll needs at least one step"},
		{"x is scroll 3 into .", `a scroll needs at least one step after "into"`},
		{"show typeof[scroll 1 into add1] .", "a scroll ends with '.', inside [ ] too"},
		{"scroll = 5", "reserved"},
	}
	for _, c := range cases {
		p := parser.New(lexer.New(scrollDefs + c.src))
		p.ParseProgram()
		errs := p.Errors()
		if len(errs) == 0 || !strings.Contains(errs[0], c.want) {
			t.Errorf("%s: got %v, want %q first", c.src, errs, c.want)
		}
	}
}

func TestDiagnoseErrorSteps(t *testing.T) {
	src := scrollDefs + `safe
    x is scroll 3 into add1, here / 0, double .
handle [math] e .
    d = diagnose[e]
    show kind of d .
safe [end]
safe
    y = 1 / 0
handle [] e .
    d = diagnose[e]
safe [end]`
	got, err := run(t, src, "")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"diagnose error (line 19)\n  math: scroll step 2 of 3 (here / 0): division by zero\n  1 add1     → returned 4\n  2 here / 0 ✗ failed: (see the message above)\n  3 double     not reached\nmath",
		"diagnose error (line 25)\n  math: division by zero",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing:\n%s\nin:\n%s", want, got)
		}
	}
	// steps isn't a part of an error: diagnose is the one way in.
	_, err = run(t, "safe\n    y = 1 / 0\nhandle [] e .\nsafe [end]\nshow steps of e .", "")
	if err == nil || !strings.Contains(err.Error(), `an error has no field "steps" (its fields: kind, file, line, message)`) {
		t.Errorf("got %v", err)
	}
}

func TestDiagnose(t *testing.T) {
	src := scrollDefs + `s = scroll add1, double .
x is diagnose[s, 3] .
show x .
t = scroll add1, s .
y is t diagnose 1 .
z is diagnose[scroll "hi" into at upper, at len .] .
d is diagnose[double, 21] .
show d .`
	got, err := run(t, src, "")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"diagnose s (line 17)\n  start      3\n  1 add1   → returned 4\n  2 double → returned 8\n  result     8\n8",
		"diagnose t (line 20)\n  start          1\n  1 add1       → returned 2\n  2 s          → returned 6\n    2.1 add1   → returned 3\n    2.2 double → returned 6\n  result         6",
		"diagnose scroll (line 21)\n  start        \"hi\"\n  1 at upper → returned \"HI\"",
		"diagnose double (line 22)\n  given     21\n  returned  42\n  took",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing:\n%s\nin:\n%s", want, got)
		}
	}

	// none goes on, marked; then the error it led to ends the scroll,
	// and comes back as a value: the program goes on.
	got, err = run(t, scrollDefs+"x is diagnose[scroll 3 into add1, shownumber, double .] .\nshow typeof[x], \"|\", kind of x .\nshow \"still running\" .", "")
	if err != nil {
		t.Fatalf("diagnose stopped the program: %v", err)
	}
	for _, want := range []string{"1 add1       → returned 4", "2 shownumber → returned nothing (none)   is none expected!?", "3 double     ✗ failed:", "\n                 (it was given none, from step 2)\n", "error|type\nstill running"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
	// The none came from the start, or from a step outside a saved scroll.
	got, _ = run(t, scrollDefs+"x is diagnose[scroll none into double .] .\ns = scroll double .\ny is diagnose[scroll 1 into shownumber, s .] .", "")
	for _, want := range []string{"(it was given none, from the start)", "2.1 double ✗ failed:", "(it was given none, from step 1)"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
	// none as the answer: shown, marked, given back.
	got, err = run(t, scrollDefs+"x is diagnose[scroll 3 into add1, shownumber .] .\nshow x .", "")
	if err != nil || !strings.Contains(got, "2 shownumber → returned nothing (none)   is none expected!?\n  result         nothing (none)\nnone") {
		t.Errorf("none result: %v\n%s", err, got)
	}
	// A step failing: the scroll ends there; the error comes back as a value.
	got, err = run(t, scrollDefs+"x is diagnose[scroll 3 into add1, here / 0, double .] .\nshow typeof[x], \"|\", kind of x, \"|\", message of x .", "")
	if err != nil {
		t.Fatalf("diagnose stopped the program: %v", err)
	}
	for _, want := range []string{"2 here / 0 ✗ failed: division by zero", "3 double     not reached", "error|math|scroll step 2 of 3 (here / 0): division by zero"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
	// Inside a saved scroll used as a step: marked there too.
	got, err = run(t, scrollDefs+"s = scroll add1, shownumber .\nx is diagnose[scroll 1 into double, s .] .\nshow x .", "")
	if err != nil || !strings.HasSuffix(strings.TrimSpace(got), "none") || !strings.Contains(got, "2.2 shownumber → returned nothing (none)   is none expected!?") {
		t.Errorf("nested none: %v\n%s", err, got)
	}
	// A function failing: its error comes back.
	got, err = run(t, scrollDefs+"def bad[n]\n    return n / 0\ndef [end]\ny is diagnose[bad, 4] .\nshow kind of y .", "")
	if err != nil || !strings.Contains(got, "failed    ✗ math error: division by zero") || !strings.HasSuffix(strings.TrimSpace(got), "math") {
		t.Errorf("function error: %v\n%s", err, got)
	}

	// An error: its message and steps, and the error back. A library
	// function by name.
	got, err = run(t, scrollDefs+"import json\nsafe\n    x is scroll 3 into add1, shownumber, double .\nhandle [] e .\n    d = diagnose[e]\n    show d == e .\nsafe [end]\nv is diagnose[load, \"[1]\"] .", "")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"diagnose error", "type: scroll step 3 of 3 (double):", "2 shownumber → returned nothing (none)   is none expected!?", "\ntrue\n", "diagnose load", "given     \"[1]\"", "returned  list of 1"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}

	// Misuse.
	_, err = run(t, "x is diagnose[5] .", "")
	if err == nil || !strings.Contains(err.Error(), "diagnose needs a scroll, a function or an error, got a number") {
		t.Errorf("got %v", err)
	}
	// The program's own diagnose wins.
	got, _ = run(t, "def diagnose[x]\n    return \"mine\"\ndef [end]\nshow diagnose[1] .", "")
	if strings.TrimSpace(got) != "mine" {
		t.Errorf("own diagnose: %q", got)
	}
}
