package evaluator

import (
	"regexp"
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
		// Inside a saved scroll used as a step: the step, and the step it's in.
		{"s = scroll add1, here / 0 .\nx is scroll 1 into double, s .", "math", "scroll step 2 of 2 (here / 0), in step 2 of 2 (s): division by zero"},
		{"s = scroll add1, here / 0 .\nt = scroll s .\nx is scroll 1 into double, t .", "math", "scroll step 2 of 2 (here / 0), in step 1 of 1 (s), in step 2 of 2 (t): division by zero"},
		{"total = 5\ns = scroll add1, total .\nx is scroll 1 into s .", "scroll", "scroll step 2 (total) is a number, not a function or scroll, in step 1 of 1 (s)"},
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

// memoryLines are reports' memory lines, whose numbers vary run to run.
var memoryLines = regexp.MustCompile(`(?m)^  memory +[0-9.]+ (B|KB|MB|GB) allocated\n`)

// withoutMemory is a report without its memory lines, to compare.
func withoutMemory(s string) string { return memoryLines.ReplaceAllString(s, "") }

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
	if n := len(memoryLines.FindAllString(got, -1)); n != 4 {
		t.Errorf("want a memory line in each of the 4 reports, got %d:\n%s", n, got)
	}
	got = withoutMemory(got)
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
	got = withoutMemory(got)
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
	got = withoutMemory(got)
	for _, want := range []string{"(it was given none, from the start)", "2.1 double ✗ failed:", "(it was given none, from step 1)"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
	// none as the answer: shown, marked, given back.
	got, err = run(t, scrollDefs+"x is diagnose[scroll 3 into add1, shownumber .] .\nshow x .", "")
	got = withoutMemory(got)
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

func TestDiagnoseBlock(t *testing.T) {
	src := `nums = list [4, 7]
diagnose
    total = 0
    [loop][x in nums]
        total = total + x
    [loop][end]
    avg = total / 0
    show "never" .
diagnose [end]
show "after: ", total .
diagnose
    [loop][i = 0; i < 2; i++]
        y = i
    [loop][end]
diagnose [end]`
	got, err := run(t, src, "")
	if err != nil {
		t.Fatalf("diagnose stopped the program: %v", err)
	}
	if n := len(memoryLines.FindAllString(got, -1)); n != 2 {
		t.Errorf("want a memory line in each block's report, got %d:\n%s", n, got)
	}
	got = withoutMemory(got)
	for _, want := range []string{
		"diagnose (lines 3-8)\n  line 3       total = 0",
		"pass 1: x = 4\n  line 5           total = total + x        total = 4",
		"pass 2: x = 7",
		"  ✗ failed: math error: line 7: division by zero\nafter: 11",
		"pass 2: i = 1",
		"  finished",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
	if strings.Contains(got, "never") || strings.Contains(got, "i++]  i =") {
		t.Errorf("unexpected output:\n%s", got)
	}
	// A long loop: the first lines, then a count.
	got, _ = run(t, "diagnose\n    [loop][i = 0; i < 500; i++]\n        y = i\n    [loop][end]\ndiagnose [end]", "")
	got = withoutMemory(got)
	if !strings.Contains(got, "more lines\n  finished") || strings.Count(got, "\n") > 70 {
		t.Errorf("long loop not cut short: %d lines", strings.Count(got, "\n"))
	}
	// Without its end.
	p := parser.New(lexer.New("diagnose\n    x = 1\n"))
	p.ParseProgram()
	if errs := p.Errors(); len(errs) == 0 || !strings.Contains(errs[0], "diagnose [end]") {
		t.Errorf("unclosed: %v", errs)
	}
}

// An operator right after a text over several lines belongs to the same
// expression (it used to be read as the start of a new line).
func TestOperatorAfterMultilineText(t *testing.T) {
	got, err := run(t, "name = \"Ann\"\nr = \"Dear\n\" + name + \",\nthanks.\" + \"!\"\nshow r .", "")
	if err != nil || got != "Dear\nAnn,\nthanks.!\n" {
		t.Errorf("got %q, %v", got, err)
	}
}

// Methods in function form (trim[s]) and as sentences (s trim), besides
// s at trim; the program's own and library functions keep their names.
func TestMethodsAsFunctions(t *testing.T) {
	cases := []struct{ src, want string }{
		{"s = \"  Hi  \"\nshow trim[s], \"|\", upper[trim[s]] .", "Hi|HI"},
		{"s = \"  Hi  \"\nt = s trim\nshow t .", "Hi"},
		{"import math\nshow round[3.14159, 2] .", "3.14"},
		{"import math\nx = 3.14159\nr = x round 2\nshow r .", "3.14"},
		{"nums = list [3, 1, 2]\nshow get[nums, 0], len[nums], contains[nums, 2] .", "33true"},
		{"m = map [\"a\": 1]\nshow getkeys[m] .", `[ "a" ]`},
		{"def trim[x]\n    return \"mine\"\ndef [end]\nshow trim[\" a \"] .", "mine"},
		{"show round[2.5] .", ""}, // needs import math, as x at round does
	}
	for _, c := range cases {
		got, err := run(t, c.src, "")
		if c.want == "" {
			if err == nil || !strings.Contains(err.Error(), "import math") {
				t.Errorf("%s: want the import math error, got %v", c.src, err)
			}
			continue
		}
		if err != nil || strings.TrimSpace(got) != c.want {
			t.Errorf("%s: got %q, %v; want %q", c.src, got, err, c.want)
		}
	}
}

// The hint on "params of req at get[...]", where get works on req.
func TestOfGetHint(t *testing.T) {
	_, err := run(t, "req = map [\"params\": map [\"id\": \"7\"]]\nshow params of req at get[\"id\"] .", "")
	if err == nil || !strings.Contains(err.Error(), "(hint: in params of req at get[...], the get works on req first; to take part of params of req, name it first: v = params of req, then v at get[...])") {
		t.Errorf("got %v", err)
	}
}

// add ... to a map at a key; the older form still works.
func TestAddToMapAtKey(t *testing.T) {
	got, err := run(t, "ages = map [\"Ann\": 30]\nadd 12 to ages at \"Cy\" .\nadd 31 to ages at \"Ann\" .\nages is ages at add[\"Bo\", 7] .\nshow ages .", "")
	if err != nil || strings.TrimSpace(got) != `{ "Ann": 31, "Cy": 12, "Bo": 7 }` {
		t.Errorf("got %q, %v", got, err)
	}
	_, err = run(t, "nums = list [1]\nadd 2 to nums at 0 .", "")
	if err == nil || !strings.Contains(err.Error(), "insert ... to nums at ...") {
		t.Errorf("list: %v", err)
	}
}

// && and || stop as soon as the answer is known.
func TestShortCircuit(t *testing.T) {
	src := `x = none
show x != none && x > 5, x == none || x < 5 .
def boom[]
    show "ran" .
    return true
def [end]
show false && boom[], true || boom[] .
show true && boom[] .`
	got, err := run(t, src, "")
	if err != nil || got != "falsetrue\nfalsetrue\nran\ntrue\n" {
		t.Errorf("got %q, %v", got, err)
	}
}

// sort compares whole numbers exactly, past 2^53 too, and text by its
// characters; mixed numbers still sort by value.
func TestSortExact(t *testing.T) {
	got, err := run(t, "big = list [9007199254740993, 9007199254740992, 3]\nsort big .\nshow big .\nw = list [\"pear\", \"Apple\", \"apple\"]\nsort w .\nshow w .\nm = list [2, 1.5, 1]\nsort m .\nshow m .", "")
	want := "[ 3, 9007199254740992, 9007199254740993 ]\n[ \"Apple\", \"apple\", \"pear\" ]\n[ 1, 1.5, 2 ]\n"
	if err != nil || got != want {
		t.Errorf("got %q, %v", got, err)
	}
}

// diagnose shows each step's value as it was when the step returned, even
// when a later step changes the same map or list in place, and shows what
// is inside.
func TestDiagnoseShowsEachStepAsItWas(t *testing.T) {
	src := `def add_fee[m]
    m at add "fee", 50 .
    return m
def [end]
def add_tax[m]
    m at add "tax", 8 .
    return m
def [end]
price = scroll add_fee, add_tax .
bill = map ["total": 100]
done is diagnose[price, bill] .
nums = list [3, 1, 2]
def sorted_copy[xs]
    xs at sort
    return xs
def [end]
x is diagnose[sorted_copy, nums] .`
	got, err := run(t, src, "")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"start       map of 1: { \"total\": 100 }",
		"1 add_fee → returned map of 2: { \"total\": 100, \"fee\": 50 }",
		"2 add_tax → returned map of 3: { \"total\": 100, \"fee\": 50, \"tax\": 8 }",
		"given     list of 3: [ 3, 1, 2 ]",
		"returned  list of 3: [ 1, 2, 3 ]",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
}
