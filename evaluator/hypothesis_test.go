package evaluator

import (
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"Turtle/lexer"
	"Turtle/parser"
)

// A theory for the hypotheses about theories: share has a proof and a
// theorem; plain has neither.
const hypothesisTheories = `theory share
    abstract
        share is the part a of the whole b, as a percent.
    notation share a of b .
    definition
        if ] b == 0 [
            fail "share: the whole can't be 0"
        if [end]
        return a * 100 / b
    theorem result >= -1000000
    proof
        share 1 of 4 . is 25.0
        share -1 of 4 . is -25.0
theory [end]

theory plain
    abstract
        plain gives n back.
    notation plain n .
    definition
        return n
theory [end]
`

func TestHypothesisReports(t *testing.T) {
	cases := []struct {
		name, src string
		want      []string
	}{
		{"a fact that holds", "xs = list [1, 2]\nhypothesis[xs at len == 2]", []string{
			"hypothesis xs at len == 2 (line 3)", "  holds     yes\n", "  took "}},
		{"a fact that doesn't", "xs = list [0.5, 0.25]\nhypothesis[xs at get[0] + xs at get[1] == 1]", []string{
			"  holds     no\n", "  where     got 0.75, want 1", "  xs        list of 2: [ 0.5, 0.25 ]"}},
		{"a check form", "n = 4\nhypothesis[n is close to 4.2 within 0.1]\nhypothesis[n is integer]", []string{
			"hypothesis n is close to 4.2 within 0.1 (line 3)\n  holds     no\n  where     got 4, want 4.2 (off by -0.2; allowed: 0.1)",
			"hypothesis n is integer (line 4)\n  holds     yes"}},
		{"an expected error", "hypothesis[1 / 0 fails [math]]", []string{"  holds     yes"}},
		{"each item", `assemble Sale [item, qty]
rows = list [Sale["tea", 2], Sale["pie", 0], Sale["bun", -2], Sale["jam", 1]]
hypothesis[rows each r give qty of r > 0]`, []string{
			"  holds     no: 2 of 4 items (50%) break it",
			"  rows      list of 4: [ Sale { item: \"tea\", qty: 2 }",
			"  where     at 1  Sale { item: \"pie\", qty: 0 }   (qty of r is 0, which isn't > 0)",
			"            at 2  Sale { item: \"bun\", qty: -2 }   (qty of r is -2, which isn't > 0)"}},
		{"kinds", `assemble Sale [item, qty]
mixed = list [Sale["tea", 2], map ["a": 1], "x", Sale["pie", 1]]
hypothesis[mixed each r give r type Sale]`, []string{
			"  holds     no: 2 of 4 items (50%) break it",
			`at 2  "x"   (r is "x", a string, not a Sale)`,
			"  types     2 Sale, 1 map, 1 string"}},
		{"any, not and at least", `xs = list [1, 2, -3]
hypothesis[xs any x give x > 5]
hypothesis[xs not x give x < 0]
hypothesis[xs at least 3 x give x > 0]
hypothesis[xs each x give x < 10]`, []string{
			"  holds     no: none of the 3 items follows it",
			"  holds     no: 1 of 3 items (33.3%) follows it, and none should\n  xs        list of 3: [ 1, 2, -3 ]\n  where     at 2  -3\n",
			"  holds     no: 2 of 3 items (66.7%) follow it; at least 3 should",
			"  holds     yes: all 3 items follow it"}},
		{"many breaks are cut short", "xs = list [1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12]\nhypothesis[xs each x give x > 100]", []string{
			"at 9  10", "            ... and 2 more"}},
		{"a function, on random inputs", `seed = 7
def bad_evens[nums]
    return nums keep x give x % 2 != 1
def [end]
def evens[nums]
    return nums keep x give x % 2 == 0
def [end]
hypothesis[bad_evens[nums] with nums as list of integer that result each x give x % 2 == 0]
hypothesis[evens[nums] with nums as list of integer that result each x give x % 2 == 0]`, []string{
			"  holds     no: failed on case ", "(smallest found; first failed on ", "  result    list of 1: [ -", "  why       1 of 1 item breaks the rule:",
			"  seed      7   (seed = 7 repeats this run)",
			"  holds     yes: on 100 random inputs\n  seed      7\n"}},
		{"an error in the function says its line", `def boom[n]
    if ] n > 5 [
        return n / 0
    if [end]
    return n
def [end]
hypothesis[boom[n] with n as integer that result <= n]`, []string{
			"  why       it stopped with an error: line 4: division by zero"}},
		{"a theory", hypothesisTheories + "hypothesis[share]", []string{
			"hypothesis share (line 24)\n  holds     yes: proof 2 of 2, 1 of 1 theorem\n  proof     ✓ share 1 of 4 . is 25.0\n            ✓ share -1 of 4 . is -25.0\n  theorems  ✓ result >= -1000000\n  tried     the proof cases and 100 random inputs"}},
		{"a theorem tried on it", hypothesisTheories + "hypothesis[share, theorem result <= 100]\nhypothesis[share, theorem result >= -1000000]\nhypothesis[plain, theorem result == n]", []string{
			"  holds     no: it fails on a = ", ": result is 200.0, which isn't <= 100\n  proof     2 of 2 hold",
			"  holds     yes: on the 2 proof cases and 100 random inputs",
			"  holds     no: not tried: plain has no proof cases to learn its inputs from"}},
		{"a proof case", hypothesisTheories + "hypothesis[share 1 of 4 . is 25.0]\nhypothesis[share 1 of 8 . is 12.0]", []string{
			"hypothesis share 1 of 4 . is 25.0 (line 24)\n  holds     yes\n  a         1\n  b         4",
			"  holds     no\n  a         1\n  b         8\n  where     got 12.5, want 12.0"}},
		{"one use", hypothesisTheories + "hypothesis[share 2 of 1]\nhypothesis[share 2 of 0]\nhypothesis[plain 3]", []string{
			"  returned  200.0\n  theorems  ✓ result >= -1000000",
			"  holds     no: share stopped with a custom error\n  a         2\n  b         0\n  where     line 8: share: the whole can't be 0",
			"  holds     no: plain has no theorems to try it on"}},
		{"an error is a no, and the program goes on", "hypothesis[10 / 0 == 1]\nshow \"after\" .", []string{
			"  holds     no: trying it stopped with a math error\n  where     line 2: division by zero", "after\n"}},
		{"a value only answers", "xs = list [1, -1]\nok = hypothesis[xs each x give x > 0]\nshow ok, \" \", hypothesis[xs any x give x > 0] .\nif ] hypothesis[xs any x give x > 0] [\n    show \"some\" .\nif [end]", []string{
			"false true\nsome\n"}},
		{"a variable can be called hypothesis", "hypothesis = 5\nshow hypothesis + 1 .", []string{"6\n"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			out, err := run(t, "import data\n"+c.src, "")
			if err != nil {
				t.Fatalf("error: %v\n%s", err, out)
			}
			for _, w := range c.want {
				if !strings.Contains(out, w) {
					t.Errorf("missing %q in:\n%s", w, out)
				}
			}
		})
	}
	// Only the value: nothing shown.
	out, err := run(t, "ok = hypothesis[1 == 2]\nshow ok .", "")
	if err != nil || out != "false\n" {
		t.Errorf("a value shows nothing: got %q, %v", out, err)
	}
}

// A theory that takes few of the numbers either side of 0 is still tried
// on many: half the random inputs stay within its proof cases' range, and
// the first of those try the edges (smallest, largest, 0).
func TestTheoryInputsStayWhereTheTheoryWorks(t *testing.T) {
	src := `seed = 1
theory cut
    abstract
        cut takes p percent off c.
    notation cut p off c .
    definition
        if ] p < 0 || p > 100 || c < 0 [
            fail "cut: p is a percent from 0 to 100, c an amount of 0 or more"
        if [end]
        return c - c * p div 100
    proof
        cut 10 off 2000 . is 1800
        cut 100 off 7 . is 0
theory [end]
hypothesis[cut, theorem result < c]
hypothesis[cut, theorem result <= c]`
	out, err := run(t, src, "")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "hypothesis cut, theorem result < c (line 15)\n  holds     no: it fails on ") {
		t.Errorf("want the false theorem caught (p = 0 or c = 0):\n%s", out)
	}
	refused := regexp.MustCompile(`100 random inputs, (\d+) of them refused`).FindStringSubmatch(out)
	if refused == nil {
		t.Fatalf("want a refused count:\n%s", out)
	}
	if n, _ := strconv.Atoi(refused[1]); n > 60 {
		t.Errorf("want most random inputs taken:\n%s", out)
	}
}

func TestHypothesisMisspelledNameStops(t *testing.T) {
	_, err := run(t, "import data\nhypothesis[nope each x give x > 0]", "")
	if err == nil || !strings.Contains(err.Error(), `nope`) {
		t.Errorf("want a name error, got %v", err)
	}
}

func TestHypothesisParseErrors(t *testing.T) {
	for src, want := range map[string]string{
		"hypothesis[]": "write the claim to try in the brackets",
		"def hypothesis[x]\n    return x\ndef [end]": "hypothesis is Turtle's own word",
		"hypothesis[f[n] that result > 0]":           "say what inputs to try, e.g. with n as list of integer",
		"hypothesis[share, result > 0]":              "write theorem and the claim to try",
		"hypothesis[xs each]":                        "give the rule after",
		"theory t\n    abstract\n        t is n.\n    notation t n .\n    definition\n        return hypothesis[n == 1]\ntheory [end]":                     "hypothesis[...] isn't written in a theory: a theory's claims are its theorems",
		"theory t\n    abstract\n        t is n.\n    notation t n .\n    definition\n        return n\n    theorem hypothesis[result == n]\ntheory [end]": "Try one from outside it: hypothesis[t, theorem ...]",
	} {
		p := parser.New(lexer.New(src))
		p.ParseProgram()
		if errs := strings.Join(p.Errors(), "; "); !strings.Contains(errs, want) {
			t.Errorf("%q: want %q in %q", src, want, errs)
		}
	}
}

func TestReportFile(t *testing.T) {
	dir := t.TempDir()
	src := `import data
reportfile = "checks.txt"
xs = list [1, 2, -3]
hypothesis[xs each x give x > 0]
def double[n]
    return n * 2
def [end]
x is diagnose[double, 21] .
diagnose
    y = x + 1
diagnose [end]
reportfile = none
hypothesis[xs any x give x < 0]
`
	out, err := runIn(t, dir, src, "")
	if err != nil {
		t.Fatal(err)
	}
	// The screen has every report.
	for _, w := range []string{"hypothesis xs each x give x > 0 (line 4)", "diagnose double (line 8)", "diagnose (lines 10-10)", "hypothesis xs any x give x < 0 (line 13)"} {
		if !strings.Contains(out, w) {
			t.Errorf("screen: missing %q in:\n%s", w, out)
		}
	}
	data, err := os.ReadFile(filepath.Join(dir, "checks.txt"))
	if err != nil {
		t.Fatal(err)
	}
	file := string(data)
	for _, w := range []string{"line 4\nhypothesis xs each x give x > 0 (line 4)\n  holds     no: 1 of 3 items (33.3%) breaks it", "line 8\ndiagnose double (line 8)\n  given     21", "line 9\ndiagnose (lines 10-10)\n", "  finished\n"} {
		if !strings.Contains(file, w) {
			t.Errorf("file: missing %q in:\n%s", w, file)
		}
	}
	if !strings.HasPrefix(file, "== ") || strings.Count(file, "\n== 20") != 2 {
		t.Errorf("want 3 reports, each under its time and line:\n%s", file)
	}
	if strings.Contains(file, "x < 0") {
		t.Errorf("reportfile = none: the screen only, got:\n%s", file)
	}

	// It adds to the file; it never starts it over.
	if _, err := runIn(t, dir, "reportfile = \"checks.txt\"\nhypothesis[1 == 1]", ""); err != nil {
		t.Fatal(err)
	}
	data, _ = os.ReadFile(filepath.Join(dir, "checks.txt"))
	if strings.Count(string(data), "\n== 20") != 3 {
		t.Errorf("want the 4th report added:\n%s", data)
	}

	if _, err := run(t, "reportfile = 3\nhypothesis[1 == 1]", ""); err == nil || !strings.Contains(err.Error(), "reportfile must be a file's path, or none") {
		t.Errorf("want a reportfile error, got %v", err)
	}
}

func TestHypothesisInTheREPL(t *testing.T) {
	it := New(".")
	p := parser.New(lexer.New("hypothesis[1 == 2]"))
	program := p.ParseProgram()
	out := captureStdout(t, func() {
		v, err := it.RunEntry(program)
		if err != nil || v != nil {
			t.Errorf("want the report only, got %v, %v", v, err)
		}
	})
	if !strings.Contains(out, "hypothesis 1 == 2 (line 1)\n  holds     no") {
		t.Errorf("want the report, got %q", out)
	}
}

// captureStdout is what f writes to stdout.
func captureStdout(t *testing.T, f func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	orig := os.Stdout
	os.Stdout = w
	done := make(chan string)
	go func() {
		b, _ := io.ReadAll(r)
		done <- string(b)
	}()
	f()
	w.Close()
	os.Stdout = orig
	return <-done
}

// Theories on text, maps, sets, nested lists and assembled values are
// proved like those on numbers, whatever the seed: a set's items and a
// map's keys (all different) have room to be, and edges are tried.
func TestTheoriesOfEveryKind(t *testing.T) {
	defs := `assemble Pet [name, age]
theory shout
    abstract
        shout gives the text t in capitals, with a ! after it.
    notation shout t .
    definition
        return t at upper + "!"
    theorem length of result == length of t + 1
    proof
        shout "hi" . is "HI!"
theory [end]
theory size_of
    abstract
        size_of is how many things are in the set s.
    notation size_of s .
    definition
        return length of s
    theorem result >= 0
    proof
        size_of set [1, 2] . is 2
theory [end]
theory keysum
    abstract
        keysum adds up the keys of the map m.
    notation keysum m .
    definition
        t = 0
        [loop][k in m at getkeys]
            t = t + k
        [loop][end]
        return t
    theorem result == result
    proof
        keysum map [1: "a", 2: "b"] . is 3
theory [end]
theory names_of
    abstract
        names_of gives the names of the pets ps.
    notation names_of ps .
    definition
        return ps process p give name of p
    theorem length of result == length of ps
    proof
        names_of list [Pet["Rex", 3]] . is list ["Rex"]
theory [end]
theory flat
    abstract
        flat joins the lists in xs into one list.
    notation flat xs .
    definition
        out = list []
        [loop][x in xs]
            out = out + x
        [loop][end]
        return out
    theorem length of result >= length of xs
    proof
        flat list [list [1], list [2, 3]] . is list [1, 2, 3]
theory [end]
`
	for seed := 1; seed <= 20; seed++ {
		src := "import data\nseed = " + strconv.Itoa(seed) + "\n" + defs +
			"show hypothesis[shout], hypothesis[size_of], hypothesis[keysum], hypothesis[names_of], hypothesis[flat] .\n"
		out, err := run(t, src, "")
		// flat's theorem is false (an empty list in xs adds nothing): the
		// edges (a list holding an empty list) find it on every seed.
		if err != nil || out != "truetruetruetruefalse\n" {
			t.Fatalf("seed %d: got %q, %v", seed, out, err)
		}
	}
}

// A theory runs once per random input, its theorems all checked on that
// one result: the proof case plus 100 inputs is 101 runs.
func TestTheoryRunsOncePerInput(t *testing.T) {
	src := `theory twice
    abstract
        twice doubles n.
    notation twice n .
    definition
        show "ran" .
        return n * 2
    theorem result == n + n
    theorem result * 2 == n * 4
    proof
        twice 2 . is 4
theory [end]
x = hypothesis[twice]
show x .`
	out, err := run(t, src, "")
	if err != nil || strings.Count(out, "ran\n") != 101 || !strings.HasSuffix(out, "true\n") {
		t.Errorf("want 101 runs, got %d (%v):\n%s", strings.Count(out, "ran\n"), err, out[max(0, len(out)-200):])
	}
}

// reportfile set inside a function counts for the reports made there.
func TestReportFileInAFunction(t *testing.T) {
	dir := t.TempDir()
	src := `def double[n]
    return n * 2
def [end]
def check_it[]
    reportfile = "inside.txt"
    d is diagnose[double, 4] .
    s = scroll double .
    e is s diagnose 5 .
def [end]
check_it[]`
	if _, err := runIn(t, dir, src, ""); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "inside.txt"))
	if err != nil || !strings.Contains(string(data), "diagnose double") || !strings.Contains(string(data), "diagnose s") {
		t.Errorf("want both reports in the file, got %q, %v", data, err)
	}
}
