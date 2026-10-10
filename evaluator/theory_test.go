package evaluator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"Turtle/lexer"
	"Turtle/parser"
)

// The theories the tests use: a slot before a fixed word that's an
// ordinary name, a fixed keyword, a type, and a phrase inside a phrase.
const theoryDefs = `assemble Item [name, qty, cents]

theory tally
    abstract
        tally says how many times b appears in the list a.
    notation tally b in a .
    definition
        n = 0
        [loop][x in a]
            if ] x == b [
                n = n + 1
            if [end]
        [loop][end]
        return n
    theorem result >= 0
    theorem result <= length of a
    proof
        tally 1 in list [1, 1, 0, 3] . is 2
theory [end]

theory price
    abstract
        price is what an item costs, in cents, every twelfth one free.
    notation price i .
    definition
        free = qty of i div 12
        return qty of i * cents of i - free * cents of i
theory [end]

theory discount
    abstract
        discount takes p percent off c.
    notation discount p off c .
    definition
        return c - c * p div 100
    theorem result <= c
theory [end]
`

func TestTheories(t *testing.T) {
	cases := []struct{ src, want string }{
		{"nums = list [1, 1, 0, 3]\ntally 1 in nums .\nshow tally 1 in nums .", "2"},
		{"c is tally 1 in list [1, 1] .\nshow c .", "2"},
		{"c = tally 1 in list [1, 2]\nshow c .", "1"},
		{"show \"ones: \", tally 1 in list [1, 1], \"!\" .", "ones: 2!"},
		// A value reaches through arithmetic, not past a comparison.
		{"show tally 1 in list [1] + list [1] == 2 .", "true"},
		// A fixed word that's an ordinary name ends the value before it.
		{"rate = 25\nshow discount rate off 2000 .", "1500"},
		// A phrase inside a phrase.
		{"show discount 10 off price Item[\"Cookie\", 12, 150] .", "1485"},
		{"basket = list [Item[\"Cookie\", 12, 150], Item[\"Loaf\", 2, 650]]\ns = 0\n[loop][i in basket]\n    s = s + price i\n[loop][end]\nshow discount 10 off s .", "2655"},
		// Text that matches a fixed word is a value, not the word.
		{"show tally \"in\" in list [\"in\", \"out\", \"in\"] .", "2"},
		// Where the notation wants a value, a word that's fixed elsewhere
		// is the value: a variable called off.
		{"off = 2000\nshow discount 10 off off .", "1800"},
		// The word alone is the theory, a value.
		{"show typeof[tally], \" \", tally type theory .", "theory true"},
	}
	for _, c := range cases {
		got, err := run(t, theoryDefs+c.src, "")
		if err != nil {
			t.Errorf("%s: %v", c.src, err)
		} else if strings.TrimSpace(got) != c.want {
			t.Errorf("%s:\n got  %q\n want %q", c.src, strings.TrimSpace(got), c.want)
		}
	}
}

func TestTheoryHelpAndDiagnose(t *testing.T) {
	got, err := run(t, theoryDefs+"help[tally]\nhelp[\"discount\"]\nx is diagnose[discount 10 off 2655] .\nshow x .", "")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"tally        (a theory, line 3)\n  tally says how many times b appears in the list a.\n  Written: tally b in a .\n  Theorem: result >= 0\n  Theorem: result <= length of a",
		"discount        (a theory, line 30)",
		"diagnose theory discount (line 40)\n  notation  discount p off c .\n  p         10\n  c         2655\n  returned  2390\n  theorem   result <= c  holds",
		"2390",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
	// A theorem that fails for this use says why; the program goes on.
	got, err = run(t, `theory keep_off
    abstract
        keep_off keeps p percent of c.
    notation keep_off p off c .
    definition
        return c * p div 100
    theorem result >= c div 2
theory [end]
x is diagnose[keep_off 10 off 2000] .
show "still running" .`, "")
	if err != nil || !strings.Contains(got, "result >= c div 2  ✗ fails: result is 200, which isn't >= 1000; c div 2 is 1000") || !strings.Contains(got, "still running") {
		t.Errorf("%v\n%s", err, got)
	}
}

func TestTheoryMistakes(t *testing.T) {
	theory := func(word, notation, def string) string {
		return "theory " + word + "\n    abstract\n        " + word + " is a test.\n    notation " + notation + "\n    definition\n        " + def + "\ntheory [end]\n"
	}
	cases := []struct{ src, want string }{
		{theory("count", "count b in a .", "return 1"), "count is already a word in Turtle (a method): a theory needs a new word"},
		{theory("mean", "mean a .", "return a"), "mean is already a word in Turtle (a function of the data library)"},
		{"def f[]\n    return 1\ndef [end]\n" + theory("f", "f n .", "return n"), "f is already a word in Turtle (a function in this file)"},
		{theory("list", "list a .", "return a"), "list is already a word in Turtle (a keyword)"},
		{theory("swap", "swap a b .", "return list [b, a]"), "swap a b .: two values (a, b) need a word or a comma between them"},
		{"x = twice 3\n" + theory("twice", "twice n .", "return n * 2"), "twice is defined by a theory on line 2, below this line"},
		{theory("twice", "double n .", "return n * 2"), "twice's notation starts with its word"},
		{theory("twice", "twice n at m .", "return n * m"), `a notation can't use "at"`},
		{theory("twice", "twice n", "return n * 2"), "a notation ends with a period"},
		{theory("twice", "twice n .", "return n * 2") + theory("twice", "twice n .", "return n"), "twice is already a theory (line 1)"},
		{"theory twice\n    notation twice n .\n    definition\n        return n\ntheory [end]\n", "theory twice needs an abstract"},
		{"theory twice\n    abstract\n        twice.\n    notation twice n .\ntheory [end]\n", "theory twice needs a definition"},
		{"theory twice\n    abstract\n        twice.\n    notation twice n .\n    definition\n        return n\n", "theory twice is never closed"},
		{"theory twice\n    abstract\n        twice.\n    notation twice n .\n    definition\n        return n\n    lemma\n        x\ntheory [end]\n", "expected abstract, notation, definition, theorem or proof, found"},
		{theory("twice", "twice n by m .", "return n * m") + "x = twice 3 with 4\n", "this isn't how twice is written: twice n by m ."},
	}
	for _, c := range cases {
		// A mistake the parser finds, or one found when the theory runs.
		p := parser.New(lexer.New(c.src))
		p.ParseProgram()
		got := strings.Join(p.Errors(), "; ")
		if got == "" {
			if _, err := run(t, c.src, ""); err != nil {
				got = err.Error()
			}
		}
		if !strings.Contains(got, c.want) {
			t.Errorf("%s\n got  %v\n want %q", c.src, got, c.want)
		}
	}
}

// A file's theories come with it when it's imported: all of them, or the
// ones an import list names.
func TestImportedTheories(t *testing.T) {
	dir := t.TempDir()
	lib := `assemble Item [name, qty, cents]

theory price
    abstract
        price is what an item costs, in cents, every twelfth one free.
    notation price i .
    definition
        free = qty of i div 12
        return qty of i * cents of i - free * cents of i
theory [end]

theory discount
    abstract
        discount takes p percent off c.
    notation discount p off c .
    definition
        return c - c * p div 100
    theorem result <= c
theory [end]
`
	if err := os.MkdirAll(filepath.Join(dir, "lib"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "lib", "shop.turtle"), []byte(lib), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := runIn(t, dir, `import lib/shop
basket = list [Item["Cookie", 12, 150], Item["Loaf", 2, 650]]
s = 0
[loop][i in basket]
    s = s + price i
[loop][end]
total is discount 10 off s .
show total, " ", typeof[discount] .`, "")
	if err != nil || strings.TrimSpace(got) != "2655 theory" {
		t.Errorf("imported theories: %q, %v", got, err)
	}
	got, err = runIn(t, dir, "import lib/shop [price, Item]\nshow price Item[\"Tart\", 2, 525] .", "")
	if err != nil || strings.TrimSpace(got) != "1050" {
		t.Errorf("an import list: %q, %v", got, err)
	}
	for src, want := range map[string]string{
		"import lib/shop [price, Item]\nx = discount 10 off 100": `"discount" isn't imported: add it to "import lib/shop [...]"`,
		"import lib/shop\ntheory price\n    abstract\n        price again.\n    notation price i .\n    definition\n        return i\ntheory [end]\n": "price is already a theory, imported from lib/shop",
	} {
		p := parser.New(lexer.New(src))
		p.ModuleDir = dir
		p.ParseProgram()
		if errs := strings.Join(p.Errors(), "; "); !strings.Contains(errs, want) {
			t.Errorf("%s:\n got  %s\n want %q", src, errs, want)
		}
	}
}

// turtle test checks the theories of a test file and the files it
// imports: each proof case, each theorem on the proof cases and on random
// inputs (shrunk to the smallest that breaks it), and warns about a theory
// with no proof or an abstract that doesn't name its word.
func TestProvingTheories(t *testing.T) {
	shop := `theory discount
    abstract
        discount takes p percent off an amount c.
    notation discount p off c .
    definition
        if ] p < 0 || p > 100 || c < 0 [
            fail "discount: p is a percent and c an amount"
        if [end]
        return c - c * p div 100
    theorem result <= c
    theorem result >= 0
    proof
        discount 10 off 2000 . is 1800
        discount 0 off 5 . is 5
theory [end]
`
	broken := `theory keep_off
    abstract
        keep_off is meant to take p percent off c.
    notation keep_off p off c .
    definition
        return c * p div 100
    theorem result <= c
    proof
        keep_off 10 off 2000 . is 1800
theory [end]

theory loose
    abstract
        loose takes p percent off c, for any numbers at all.
    notation loose p off c .
    definition
        return c - c * p div 100
    theorem result <= c
    proof
        loose 10 off 2000 . is 1800
        loose -10 off 2000 . is 2200
theory [end]

theory twice
    abstract
        Doubles a number.
    notation twice n .
    definition
        return n * 2
theory [end]
`
	out, code := testRun(t, map[string]string{
		"lib/shop.turtle":    shop,
		"lib/broken.turtle":  broken,
		"test_shop.turtle":   "import test\nimport lib/shop\ndef test_use[]\n    check discount 50 off 10 == 5 .\ndef [end]\n",
		"test_broken.turtle": "import test\nimport lib/broken\nseed = 1\n",
	})
	if code != 1 {
		t.Errorf("exit %d, want 1", code)
	}
	for _, want := range []string{
		"  PASS  theory discount   proof 2 of 2, 2 theorems hold on 100 random inputs",
		"  FAIL  theory keep_off\n        lib/broken.turtle line 9: proof: keep_off 10 off 2000 . gave 200, expected 1800",
		"  FAIL  theory loose\n        lib/broken.turtle line 18: theorem result <= c fails on p = -",
		"  PASS  theory twice      no proof\n  WARN  theory twice      unproven: it has no proof\n  WARN  theory twice      its abstract doesn't say what twice is: name twice in it",
		"FAILED: 1 passed, 0 failed, 2 theories proven, 2 theories failed, 2 warnings (2 files,",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
}

// diagnose[theory] runs its proof and checks its theorems, and shows
// where it fails, without stopping the program.
func TestDiagnoseTheory(t *testing.T) {
	src := `theory keep_off
    abstract
        keep_off is meant to take p percent off c.
    notation keep_off p off c .
    definition
        return c * p div 100
    theorem result <= c
    proof
        keep_off 10 off 2000 . is 1800
        keep_off 50 off 10 . is 5
theory [end]
seed = 3
x is diagnose[keep_off] .
show typeof[x], " still running" .`
	got, err := run(t, src, "")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"diagnose theory keep_off (line 1)\n  notation  keep_off p off c .\n  proof\n",
		"keep_off 10 off 2000 . is 1800  ✗ gave 200, expected 1800",
		"keep_off 50 off 10 . is 5       ✓",
		"theorems, on the proof cases and 100 random inputs (seed 3)",
		"theory still running",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
}

// A private theory works in its own file; another file can't use it,
// except a test file, which can test it.
func TestPrivateTheories(t *testing.T) {
	dir := t.TempDir()
	lib := `import math
theory ~cents_of
    abstract
        ~cents_of turns dollars d into whole cents.
    notation ~cents_of d .
    definition
        hundred = d * 100
        return hundred at round
theory [end]

theory owed
    abstract
        owed is what n items cost at d dollars each, in cents.
    notation owed n at_price d .
    definition
        return n * ~cents_of d
theory [end]
`
	if err := os.WriteFile(filepath.Join(dir, "shop.turtle"), []byte(lib), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := runIn(t, dir, "import shop\nshow owed 3 at_price 1.25 .", "")
	if err != nil || strings.TrimSpace(got) != "375" {
		t.Errorf("a public theory using a private one: %q, %v", got, err)
	}
	p := parser.New(lexer.New("import shop\nx = ~cents_of 3"))
	p.ModuleDir = dir
	p.ParseProgram()
	if errs := strings.Join(p.Errors(), "; "); !strings.Contains(errs, "~cents_of is private to shop: a ~ theory is for its own file") {
		t.Errorf("from another file: %s", errs)
	}
	out, code := testRun(t, map[string]string{
		"shop.turtle":      lib,
		"test_shop.turtle": "import test\nimport shop\ndef test_both[]\n    check ~cents_of 0.1 == 10 .\n    check owed 2 at_price 0.5 == 100 .\ndef [end]\n",
	})
	if code != 0 || !strings.Contains(out, "PASS  test_both") || !strings.Contains(out, "theory ~cents_of") {
		t.Errorf("a test file: exit %d\n%s", code, out)
	}
}

// Random inputs keep to the proof cases' ranges: numbers 0 or more stay 0
// or more; a proof with a negative number gets negative ones too. A failed
// theorem's reason is one tidy line.
func TestTheoryInputsLearnRanges(t *testing.T) {
	out, code := testRun(t, map[string]string{
		"lib.turtle": `assemble Sale [qty, cents]
theory revenue
    abstract
        revenue is what the sales s brought in.
    notation revenue of_sales s .
    definition
        t = 0
        [loop][x in s]
            t = t + qty of x * cents of x
        [loop][end]
        return t
    theorem result >= 0
    proof
        revenue of_sales list [Sale[2, 300]] . is 600
theory [end]

theory shift
    abstract
        shift adds 5 to n.
    notation shift n .
    definition
        return n + 5
    theorem result > 100 || result < -100
    proof
        shift -300 . is -295
theory [end]
`,
		"test_lib.turtle": "import test\nimport lib\nseed = 3\n",
	})
	if code != 1 || !strings.Contains(out, "PASS  theory revenue") {
		t.Errorf("exit %d\n%s", code, out)
	}
	if !strings.Contains(out, "theorem result > 100 || result < -100 fails on n = ") || strings.Contains(out, ";  ") {
		t.Errorf("the shift theorem's failure:\n%s", out)
	}
}

// diagnose shows a big list's start without copying it all.
func TestDiagnosePreviewsBigValues(t *testing.T) {
	got, err := run(t, "import data\nnums = range[100000]\nx is diagnose[scroll nums into here .] .", "")
	if err != nil || !strings.Contains(got, "list of 100000: [ 0, 1, 2, 3, 4,") || strings.Contains(got, "99999") {
		t.Errorf("%v\n%s", err, got)
	}
}

// of and is can be a notation's words: inside the phrase they're its own,
// and elsewhere of still reads a field.
func TestTheoriesWithOfAndIs(t *testing.T) {
	defs := `import data
import math
assemble Item [name, qty]
theory average
    abstract
        average is the mean of xs, rounded to places decimals.
    notation average of xs to places .
    definition
        return mean[xs] at round[places]
theory [end]
theory share
    abstract
        share is the part a of the whole b, as a percent.
    notation share a of b .
    definition
        return a * 100 / b
theory [end]
theory same
    abstract
        same says whether a is b.
    notation same a is b .
    definition
        return a == b
theory [end]
`
	cases := []struct{ src, want string }{
		{"show average of list [3, 4, 4] to 1 .", "3.7"},
		{"x is average of list [1, 2] to 1 .\nshow x .", "1.5"},
		{"part = 5\nshow share part of 20 .", "25.0"},
		{"item = Item[\"tea\", 30]\nq = qty of item\nshow share q of 120, \" \", qty of item .", "25.0 30"},
		{"a = 2\nshow same a is 2, \" \", same \"x\" is \"y\" .", "true false"},
	}
	for _, c := range cases {
		got, err := run(t, defs+c.src, "")
		if err != nil {
			t.Errorf("%s: %v", c.src, err)
		} else if strings.TrimSpace(got) != c.want {
			t.Errorf("%s:\n got  %q\n want %q", c.src, strings.TrimSpace(got), c.want)
		}
	}
	for src, want := range map[string]string{
		defs + "item = Item[\"tea\", 30]\nx = share qty of item of 120":                                                          "in this phrase of is share's own word: name a value that uses of first",
		"theory same\n    abstract\n        same.\n    notation same is b .\n    definition\n        return b\ntheory [end]\n":   "a notation can't have is right after its word",
		"theory near\n    abstract\n        near.\n    notation near a at b .\n    definition\n        return a\ntheory [end]\n": `a notation can't use "at"`,
	} {
		p := parser.New(lexer.New(src))
		p.ParseProgram()
		if errs := strings.Join(p.Errors(), "; "); !strings.Contains(errs, want) {
			t.Errorf("%s\n got  %s\n want %q", src, errs, want)
		}
	}
}
