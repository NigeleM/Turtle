package evaluator

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"Turtle/lexer"
	"Turtle/parser"
)

// caught runs src and returns the message of the test error it stops
// with ("" if none), by wrapping it in safe ... handle [test].
func caught(t *testing.T, setup, stmt string) string {
	t.Helper()
	src := "import test\n" + setup + "\nmsg = \"\"\nsafe\n    " + stmt + "\nhandle [test] e .\n    msg = message of e\nsafe [end]\nshow msg ."
	out, err := run(t, src, "")
	if err != nil {
		t.Fatalf("unexpected error: %v (output %q)", err, out)
	}
	return strings.TrimSuffix(out, "\n")
}

func TestCheck(t *testing.T) {
	setup := `assemble Order [item, qty]
def evens[nums]
    return nums keep x gives x % 2 == 0
def [end]
nums = list [1, 2, 3, 4]`
	setup = "import data\n" + setup
	passes := []string{
		`check evens[nums] == list [2, 4] .`,
		`check nums at contains[3] .`,
		`check nums at find[3] .`,
		`check nums at index[3] == 2 .`,
		`check set [1] at subset[set [1, 2]] .`,
		`check set [1, 2] at superset[set [1]] .`,
		`check map ["a": 1] at invert == map [1: "a"] .`,
		`check map ["a": 1] at invert == map ["a": 1] at invert .`,
		`check "" at isEmpty .`,
		`check "42" at isNumber .`,
		`check !nums at isEmpty .`,
		`check length of nums > 3 && nums at contains[1] .`,
		`check length of nums > 30 || nums at contains[1] .`,
		`check 5 is integer .`,
		`check 5.0 is float .`,
		`check 5 is number .`,
		`check 5.5 is number .`,
		`check "5" is string .`,
		`check "5" is not number .`,
		`check true is boolean .`,
		`check nums is list .`,
		`check set [1] is set .`,
		`check map [] is map .`,
		`check none is none .`,
		`check nums is not none .`,
		`check evens is function .`,
		`check Order["pen", 1] is Order .`,
		`check list [] is empty .`,
		`check "" is empty .`,
		`check nums is not empty .`,
		`check 0.1 + 0.2 is close to 0.3 .`,
		`check 10 / 3 is close to 3.33 within 0.01 .`,
		`check 0.5 is not close to 0.3 .`,
		`check 1 div 0 fails .`,
		`check 1 div 0 fails [math] .`,
		`check nums at get[9] fails [index] .`,
		`check "a" + list [1] fails [math, type] .`,
	}
	for _, p := range passes {
		if msg := caught(t, setup, p); msg != "" {
			t.Errorf("%s: should pass, failed with %q", p, msg)
		}
	}
	fails := map[string][]string{
		`check evens[nums] == list [2, 4, 6] .`:                    {"failed: check evens[nums] == list [2, 4, 6] .", "got 2 items, want 3", "at 2: missing (want 6)"},
		`check list [1, 2, 3] == list [1, 5, 3] .`:                 {"at 1: got 2, want 5 (3 less)"},
		`check list [3, 1, 2] == list [1, 2, 3] .`:                 {"the same items, in a different order"},
		`check map ["a": 1, "b": 2] == map ["a": 1, "c": 3] .`:     {`key "c": missing (want 3)`, `key "b": not wanted (got 2)`},
		`check map ["a": list [1, 2]] == map ["a": list [1, 3]] .`: {`key "a", at 1: got 2, want 3 (1 less)`},
		`check set [1, 2] == set [2, 3] .`:                         {"missing 3", "not wanted: 1"},
		`check "abd" == "abc" .`:                                   {`got "abd", want "abc"`, `first difference at position 2: "d", want "c"`},
		`check "ab" == "abc" .`:                                    {`got is shorter`},
		`check Order["pen", 1] == Order["pen", 2] .`:               {"field qty: got 1, want 2 (1 less)"},
		`check 5 == "5" .`:                                         {`got 5 (an integer), want "5" (a string)`},
		`check 2.5 == 3 .`:                                         {"got 2.5, want 3 (0.5 less)"},
		`check 4 != 4 .`:                                           {"both are 4"},
		`check length of nums > 5 .`:                               {"length of nums is 4, which isn't > 5"},
		`check nums at contains[7] .`:                              {"nums is [ 1, 2, 3, 4 ]"},
		`check length of nums > 5 && nums at contains[1] .`:        {"length of nums > 5 is false"},
		`check "x" is integer .`:                                   {`"x" is a string, not an integer`},
		`check nums is map .`:                                      {"nums is [ 1, 2, 3, 4 ], a list, not a map"},
		`check none is not none .`:                                 {"none is none"},
		`check 0.5 is close to 0.3 .`:                              {"got 0.5, want 0.3 (off by 0.2; allowed: a tiny rounding difference)"},
		`check 0.31 is not close to 0.3 within 0.1 .`:              {"which is within 0.1 of 0.3"},
		`check 1 + 1 fails .`:                                      {"expected an error, but it gave 2"},
		`check "a" + list [1] fails [math] .`:                      {"expected a math error, got a type error"},
	}
	for stmt, wants := range fails {
		msg := caught(t, setup, stmt)
		for _, w := range wants {
			if !strings.Contains(msg, w) {
				t.Errorf("%s: want %q in the message, got %q", stmt, w, msg)
			}
		}
	}
}

func TestCheckMistakes(t *testing.T) {
	cases := map[string]string{
		"import test\ncheck 5 .":                   "check needs something true or false, but 5 is 5",
		"import test\ncheck 5 is gizmo .":          "gizmo isn't a kind",
		"import test\ncheck \"a\" is close to 1 .": "needs two numbers",
		"import test\ncheck 5 == 6 .":              "line 2: failed: check 5 == 6 .",
	}
	for src, want := range cases {
		_, err := run(t, src, "")
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%q: want error containing %q, got %v", src, want, err)
		}
	}
}

func TestVerify(t *testing.T) {
	setup := `rolls = list [6, 2, 6, 3]
ages = map ["ann": 30, "bo": 25]
def positive[x]
    return x > 0
def [end]`
	passes := []string{
		`verify rolls each r gives r >= 1 && r <= 6 .`,
		`verify rolls each positive .`,
		`verify rolls any r gives r == 6 .`,
		`verify rolls not r gives r == 7 .`,
		`verify rolls at least 2 r gives r == 6 .`,
		`verify rolls at least 0 r gives r == 7 .`,
		`verify rolls at most 2 r gives r == 6 .`,
		`verify rolls at most 0 r gives r == 7 .`,
		`verify rolls exactly 2 r gives r == 6 .`,
		`verify rolls exactly 0 r gives r == 7 .`,
		`verify list [1, 2, 2, 5] each pair [a, b] gives a <= b .`,
		`verify ages each a gives a >= 18 .`,
		`verify ages each [name, age] gives name at length <= 3 && age > 0 .`,
		`verify "abc" each c gives c != "z" .`,
		`verify set [1, 2] each x gives x > 0 .`,
		`verify list [] each x gives x > 100 .`,
		`verify list [] not x gives x > 100 .`,
	}
	for _, p := range passes {
		if msg := caught(t, setup, p); msg != "" {
			t.Errorf("%s: should pass, failed with %q", p, msg)
		}
	}
	fails := map[string][]string{
		`verify rolls each r gives r < 6 .`:                     {"2 of 4 items break the rule:", "at 0: 6", "at 2: 6"},
		`verify rolls any r gives r == 1 .`:                     {"none of the 4 items follows the rule"},
		`verify list [] any r gives r == 1 .`:                   {"there are no items"},
		`verify rolls not r gives r == 6 .`:                     {"2 of 4 items follow the rule, and none should:", "at 0: 6"},
		`verify rolls at least 3 r gives r == 6 .`:              {"2 of 4 items follow the rule; at least 3 should:"},
		`verify rolls at least 1 r gives r == 7 .`:              {"0 of 4 items follow the rule; at least 1 should"},
		`verify rolls at most 1 r gives r == 6 .`:               {"2 of 4 items follow the rule; at most 1 should:"},
		`verify rolls exactly 1 r gives r == 6 .`:               {"2 of 4 items follow the rule; exactly 1 should:"},
		`verify rolls exactly 1 r gives r == 3 && r == 2 .`:     {"0 of 4 items follow the rule; exactly 1 should"},
		`verify rolls exactly 3 r gives r == 3 .`:               {"1 of 4 items follows the rule; exactly 3 should:", "at 3: 3"},
		`verify list [3, 1, 2] each pair [a, b] gives a <= b .`: {"1 of 2 pairs breaks the rule:", "at 0 and 1: 3, 1"},
		`verify ages each a gives a > 26 .`:                     {`at key "bo": 25`},
		`verify "abc" each c gives c != "b" .`:                  {`at 1: "b"`},
	}
	for stmt, wants := range fails {
		msg := caught(t, setup, stmt)
		for _, w := range wants {
			if !strings.Contains(msg, w) {
				t.Errorf("%s: want %q in the message, got %q", stmt, w, msg)
			}
		}
	}
	mistakes := map[string]string{
		"import test\nverify list [1] each [a, b] gives a < b .":            "for neighbors write each pair",
		"import test\nverify list [1] each pair x gives x .":                "",
		"import test\nverify map [\"a\": 1] each pair [a, b] gives a < b .": "a map has no neighbors",
		"import test\nverify list [1] each x gives x + 1 .":                 "the rule must give true or false",
		"import test\nverify 5 each x gives x > 0 .":                        "needs a list, set, map or string",
		"import test\nverify list [1] each 5 .":                             "the rule must be a function",
		"import test\nverify list [1] at least \"two\" x gives x > 0 .":     "",
	}
	for src, want := range mistakes {
		p := parser.New(lexer.New(src))
		p.ParseProgram()
		if len(p.Errors()) > 0 {
			continue // a parse error is fine for these
		}
		_, err := run(t, src, "")
		if want != "" && (err == nil || !strings.Contains(err.Error(), want)) {
			t.Errorf("%q: want error containing %q, got %v", src, want, err)
		}
	}
}

func TestValidate(t *testing.T) {
	setup := `import data
import sort [min_sort, quick_sort]
assemble Order [item, qty]
def evens[nums]
    return nums keep x gives x % 2 == 0
def [end]
def bad_evens[nums]
    return nums keep x gives x % 2 != 1
def [end]
def plus[a, b]
    return a + b
def [end]
def total[o]
    return qty of o * 2
def [end]`
	passes := []string{
		`validate evens[xs] with xs as list of integer that result each x gives x % 2 == 0 .`,
		`validate evens[xs] to found with xs as list of integer from -50 to 50 that length of found <= length of xs .`,
		`validate plus[a, b] with a as integer, b as integer that result == plus[b, a] .`,
		`validate quick_sort[xs] with xs as list of integer matches min_sort[xs] .`,
		`validate total[o] with o as Order [string, integer from 1 to 10] that result >= 2 .`,
		`validate plus[a, b] with a as string of 3, b as string that result at length >= 4 .`,
	}
	for _, p := range passes {
		if msg := caught(t, setup, p); msg != "" {
			t.Errorf("%s: should pass, failed with %q", p, msg)
		}
	}
	// A real bug: -3 % 2 is -1, so bad_evens keeps negative odd numbers.
	// Shrinking finds the smallest: one negative odd number, -1.
	msg := caught(t, setup, `validate bad_evens[xs] with xs as list of integer that result each x gives x % 2 == 0 .`)
	for _, w := range []string{"failed on case", "xs = [ -1 ]", "result = [ -1 ]", "1 of 1 item breaks the rule", "to repeat this run: seed = "} {
		if !strings.Contains(msg, w) {
			t.Errorf("bad_evens: want %q in %q", w, msg)
		}
	}
	// The same seed finds the same failing case.
	first := caught(t, setup+"\nseed = 42", `validate bad_evens[xs] with xs as list of integer that result each x gives x % 2 == 0 .`)
	again := caught(t, setup+"\nseed = 42", `validate bad_evens[xs] with xs as list of integer that result each x gives x % 2 == 0 .`)
	if first == "" || first != again || !strings.Contains(first, "(seed 42)") {
		t.Errorf("seed 42 twice: %q and %q", first, again)
	}
	// matches shows how the two answers differ.
	msg = caught(t, setup, `validate evens[xs] with xs as list of integer matches bad_evens[xs] .`)
	if !strings.Contains(msg, "bad_evens[xs] gave") {
		t.Errorf("matches: got %q", msg)
	}
	// An error in the function is a failure, with the error.
	msg = caught(t, setup, `validate plus[a, b] with a as integer, b as list of integer that true .`)
	if !strings.Contains(msg, "it stopped with an error") {
		t.Errorf("error in the call: got %q", msg)
	}
	// cases sets how many inputs are tried.
	msg = caught(t, setup+"\ncases = 1\nseed = 1", `validate bad_evens[xs] with xs as list of 50 integers that result each x gives x % 2 == 0 .`)
	if !strings.Contains(msg, "case 1 of 1") {
		t.Errorf("cases = 1: got %q", msg)
	}
}

func TestValidateLearnsFromChecks(t *testing.T) {
	src := `import test
import data
def evens[nums]
    return nums keep x gives x % 2 == 0
def [end]
def bad_evens[nums]
    return nums keep x gives x % 2 != 1
def [end]
def test_learned[]
    check evens[list [1, 2, 3, 4]] == list [2, 4] .
    validate evens[nums] that result each x gives x % 2 == 0 .
def [end]
def test_learned_bug[]
    check bad_evens[list [2, 4]] == list [2, 4] .
    validate bad_evens[nums] that result each x gives x % 2 == 0 .
def [end]
def test_nothing_to_learn[]
    validate evens[nums] that result each x gives x % 2 == 0 .
def [end]
test_learned[]
msg = ""
safe
    test_learned_bug[]
handle [test] e .
    msg = message of e
safe [end]
show msg at contains["inputs made like the ones in this test's checks"] .
test_nothing_to_learn[]`
	out, err := run(t, src, "")
	if out != "true\n" || err == nil || !strings.Contains(err.Error(), "there's no check on evens in this test to learn its inputs from") {
		t.Fatalf("got %q, %v", out, err)
	}
}

func TestTestWordsNeedTheImport(t *testing.T) {
	// Without import test, check is an ordinary name.
	out, err := run(t, `def check[a, b]
    return a == b
def [end]
show check[1, 1] .`, "")
	if err != nil || out != "true\n" {
		t.Fatalf("got %q, %v", out, err)
	}
	p := parser.New(lexer.New("import test\ndef check[a]\n    return a\ndef [end]"))
	p.ParseProgram()
	if errs := strings.Join(p.Errors(), "\n"); !strings.Contains(errs, "check is a test word in a file that imports test") {
		t.Errorf("def check after import test: got %q", errs)
	}
}

// testRun writes files into a folder and runs turtle test there.
func testRun(t *testing.T, files map[string]string, args ...string) (string, int) {
	t.Helper()
	dir := t.TempDir()
	for name, src := range files {
		path := filepath.Join(dir, name)
		os.MkdirAll(filepath.Dir(path), 0o755)
		if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	var out bytes.Buffer
	code := TestCommand(args, dir, &out)
	return out.String(), code
}

func TestTestCommand(t *testing.T) {
	tests := `import test
def test_one[]
    check 1 + 1 == 2 .
def [end]
def test_two[]
    check 2 * 2 == 5 .
def [end]
def test_three[]
    verify list [1, 2] each x gives x > 0 .
def [end]
def helper[]
    check 1 == 2 .
def [end]
`
	cases := []struct {
		name  string
		files map[string]string
		args  []string
		code  int
		want  []string
		not   []string
	}{
		{name: "independent tests", files: map[string]string{"test_a.trt": tests}, code: 1,
			want: []string{"test_a.trt\n", "PASS  test_one", "FAIL  test_two", "test_a.trt line 6: failed: check 2 * 2 == 5 .", "got 4, want 5 (1 less)", "PASS  test_three", "FAILED: 2 passed, 1 failed (1 file"},
			not:  []string{"helper", "suite"}},
		{name: "suite stop", files: map[string]string{"test_a.trt": strings.Replace(tests, "import test\n", "import test\nsuite = \"stop\"\n", 1)}, code: 1,
			want: []string{"test_a.trt (suite: stop)", "SKIP  test_three   (stopped: test_two failed)", "suite FAILED: 1 passed, 1 failed, 1 skipped"}},
		{name: "suite true is stop", files: map[string]string{"test_a.trt": strings.Replace(tests, "import test\n", "import test\nsuite = true\n", 1)}, code: 1,
			want: []string{"(suite: stop)", "SKIP  test_three"}},
		{name: "suite all", files: map[string]string{"test_a.trt": strings.Replace(tests, "import test\n", "import test\nsuite = \"all\"\n", 1)}, code: 1,
			want: []string{"(suite: all)", "PASS  test_three", "suite FAILED: 2 passed, 1 failed", "failures:\n  test_two\n      test_a.trt line 7: failed: check 2 * 2 == 5 ."}},
		{name: "all pass", files: map[string]string{"test_ok.trt": "import test\ndef test_x[]\n    check true .\ndef [end]\n"}, code: 0,
			want: []string{"PASS  test_x", "ok: 1 passed, 0 failed (1 file"}},
		{name: "folders below, sorted, hidden skipped", files: map[string]string{
			"test_b.trt":         "import test\ndef test_b[]\n    check true .\ndef [end]\n",
			"sub/test_a.trt":     "import test\ndef test_a[]\n    check true .\ndef [end]\n",
			".hidden/test_c.trt": "import test\ndef test_c[]\n    check false .\ndef [end]\n",
			"notes.trt":          "import test\ndef test_n[]\n    check false .\ndef [end]\n",
		}, code: 0, want: []string{"sub/test_a.trt", "test_b.trt", "ok: 2 passed, 0 failed (2 files"}, not: []string{"test_c", "test_n"}},
		{name: "one file", files: map[string]string{"test_a.trt": tests, "test_ok.trt": "import test\ndef test_x[]\n    check true .\ndef [end]\n"}, args: []string{"test_ok.trt"}, code: 0,
			want: []string{"ok: 1 passed"}, not: []string{"test_two"}},
		{name: "a file not named test_", files: map[string]string{"lists.trt": tests}, args: []string{"lists.trt"}, code: 1,
			want: []string{"a test file's name starts with test_"}},
		{name: "no test files", files: map[string]string{"main.trt": "show 1 ."}, code: 1, want: []string{"no test files (test_*.trt)"}},
		{name: "old .t test files aren't tests", files: map[string]string{"test_old.t": "import test\ndef test_x[]\n    check true .\ndef [end]\n"}, code: 1, want: []string{"no test files (test_*.trt)"}},
		{name: "missing import", files: map[string]string{"test_a.trt": "def test_x[]\n    show 1 .\ndef [end]\n"}, code: 1,
			want: []string{"add import test at the top", "1 file couldn't run"}},
		{name: "parse error", files: map[string]string{"test_a.trt": "import test\ncheck .\n"}, code: 1, want: []string{"test_a.trt: parse error"}},
		{name: "top-level error", files: map[string]string{"test_a.trt": "import test\nx = 1 div 0\ndef test_x[]\n    check true .\ndef [end]\n"}, code: 1,
			want: []string{"the file's own code stopped before the tests: line 2: division by zero"}},
		{name: "an error that isn't a check", files: map[string]string{"test_a.trt": "import test\ndef test_x[]\n    x = list [1] at get[5]\ndef [end]\n"}, code: 1,
			want: []string{"test_a.trt line 3: stopped with a index error: index 5 out of range"}},
		{name: "test functions take no arguments", files: map[string]string{"test_a.trt": "import test\ndef test_x[a]\n    check true .\ndef [end]\n"}, code: 1,
			want: []string{"a test function takes no arguments: write def test_x[]"}},
		{name: "top-level values are shared", files: map[string]string{"test_a.trt": "import test\nprices = list [3, 1]\ndef test_x[]\n    check length of prices == 2 .\ndef [end]\n"}, code: 0},
		{name: "benchmark with runs", files: map[string]string{"test_a.trt": "import test\ndef test_x[]\n    benchmark = true\n    runs = 25\n    check true .\ndef [end]\ndef test_y[]\n    check true .\ndef [end]\n"}, code: 0,
			want: []string{"PASS  test_x   25 runs   avg ", "fastest ", "slowest "}, not: []string{"test_y   1 run"}},
		{name: "benchmark for the whole file, one test opts out", files: map[string]string{"test_a.trt": "import test\nbenchmark = true\nruns = 3\ndef test_x[]\n    check true .\ndef [end]\ndef test_y[]\n    benchmark = false\n    check true .\ndef [end]\n"}, code: 0,
			want: []string{"test_x   3 runs"}, not: []string{"test_y   3 runs"}},
		{name: "benchmark Go-style", files: map[string]string{"test_a.trt": "import test\nbenchtime = 0.05\ndef test_x[]\n    benchmark = true\n    check true .\ndef [end]\n"}, code: 0,
			want: []string{" runs   avg "}},
		{name: "a benchmark run that fails", files: map[string]string{"test_a.trt": "import test\nimport random\nn = list []\ndef test_x[]\n    benchmark = true\n    runs = 5\n    add 1 to n .\n    check length of n < 3 .\ndef [end]\n"}, code: 1,
			want: []string{"benchmark run 2"}},
		{name: "bad settings", files: map[string]string{"test_a.trt": "import test\nsuite = \"sometimes\"\ndef test_x[]\n    check true .\ndef [end]\n"}, code: 1,
			want: []string{`suite must be false, "stop" or "all", got "sometimes"`}},
		{name: "testing your own library", files: map[string]string{
			"lib/shapes.trt":  "assemble Rect [w, h]\ndef area[r]\n    return w of r * h of r\ndef [end]\ndef broken[n]\n    return n div 0\ndef [end]\n",
			"test_shapes.trt": "import test\nimport lib/shapes\ndef test_area[]\n    check area[Rect[2, 3]] == 6 .\n    check shapes area[Rect[1, 1]] == 1 .\ndef [end]\ndef test_area_wrong[]\n    check area[Rect[2, 3]] == 7 .\ndef [end]\ndef test_broken[]\n    x = broken[1]\ndef [end]\n",
		}, code: 1, want: []string{"PASS  test_area", "test_shapes.trt line 8: failed: check area[Rect[2, 3]] == 7 .", "got 6, want 7 (1 less)", "lib/shapes.trt line 6: stopped with a math error: division by zero"}},
		{name: "a library file isn't a test file", files: map[string]string{
			"lib/test_helpers.trt": "def test_helper[]\n    show 1 .\ndef [end]\n",
			"test_a.trt":           "import test\nimport lib/test_helpers\ndef test_x[]\n    check true .\ndef [end]\n",
		}, code: 1, want: []string{"lib/test_helpers.trt: add import test at the top", "PASS  test_x"}},
		{name: "exit in a test", files: map[string]string{"test_a.trt": "import test\nimport system [exit]\ndef test_x[]\n    exit[0]\ndef [end]\n"}, code: 1,
			want: []string{"the test called exit[]"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			out, code := testRun(t, c.files, c.args...)
			if code != c.code {
				t.Errorf("exit code %d, want %d\n%s", code, c.code, out)
			}
			for _, w := range c.want {
				if !strings.Contains(out, w) {
					t.Errorf("want %q in:\n%s", w, out)
				}
			}
			for _, n := range c.not {
				if strings.Contains(out, n) {
					t.Errorf("don't want %q in:\n%s", n, out)
				}
			}
		})
	}
}

// TestTestExamples runs the example test files in testdata/testlib with
// turtle test: they all pass.
func TestTestExamples(t *testing.T) {
	src, err := filepath.Abs("../testdata/testlib")
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if code := TestCommand(nil, src, &out); code != 0 || !strings.Contains(out.String(), "ok: ") {
		t.Fatalf("exit %d:\n%s", code, out.String())
	}
}

// TestSpeedFile runs testdata/speed/test_speed.trt once, with benchmarking
// off, so the speed test keeps working (its timings are for reading, not
// checking).
func TestSpeedFile(t *testing.T) {
	src, err := os.ReadFile("../testdata/speed/test_speed.trt")
	if err != nil {
		t.Fatal(err)
	}
	// Git on Windows may check the file out with \r\n line endings.
	orig := strings.ReplaceAll(string(src), "\r\n", "\n")
	text := strings.Replace(orig, "\nbenchmark = true\n", "\nbenchmark = false\n", 1)
	if text == orig {
		t.Fatal("test_speed.trt no longer sets benchmark = true at the top")
	}
	out, code := testRun(t, map[string]string{"test_speed.trt": text})
	if code != 0 || strings.Contains(out, " runs ") {
		t.Fatalf("exit %d:\n%s", code, out)
	}
}
