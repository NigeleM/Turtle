package evaluator

import (
	"regexp"
	"testing"
)

// A function's own values are freed when it returns: a finished call's
// scope, put aside to be reused, no longer holds them. What it returns
// and the program keeps stays.
func TestFinishedCallsLetGo(t *testing.T) {
	src := `import data
import system
def scratch[]
    b = range[0, 1000000]
    return length of b
def [end]
def made[]
    b = range[0, 1000000]
    return b
def [end]
before = memory[]
n = scratch[]
show memory[] - before < 2000000 .
kept = made[]
show memory[] - before > 10000000 .
kept = none
freed = freememory[]
show memory[] - before < 2000000 .`
	out, err := run(t, src, "")
	if err != nil || out != "true\ntrue\ntrue\n" {
		t.Errorf("got %q, %v", out, err)
	}
}

func TestSizeof(t *testing.T) {
	src := `import system
assemble Pet [name, age]
small = sizeof[list [1, 2, 3]]
big = sizeof[list ["a long piece of text, longer than the others", 2, 3]]
show small > 0, " ", big > small .
shared = "the same text"
twice = list [shared, shared]
once = list [shared, "x"]
show sizeof[twice] <= sizeof[once] .
show sizeof[Pet["Rex", 3]] > sizeof[3], " ", sizeof[map ["a": list [1, 2]]] > sizeof[map []] .`
	out, err := run(t, src, "")
	if err != nil || out != "true true\ntrue\ntrue true\n" {
		t.Errorf("got %q, %v", out, err)
	}
}

func TestMemoryInReports(t *testing.T) {
	out, err := run(t, "import data\nxs = list [1, 2]\nhypothesis[xs each x give x > 0]\ndef double[n]\n    return n * 2\ndef [end]\nd is diagnose[double, 2] .", "")
	if err != nil {
		t.Fatal(err)
	}
	if n := len(memoryLines.FindAllString(out, -1)); n != 2 {
		t.Errorf("want a memory line in each report:\n%s", out)
	}
	// Each test shows what it allocated, after its time.
	files := map[string]string{"test_m.turtle": "import test\nimport data\ndef test_big[]\n    b = range[0, 100000]\n    check length of b == 100000 .\ndef [end]\n"}
	got, code := testRun(t, files)
	if code != 0 || !regexpMatch(`PASS  test_big   [0-9.]+(µs|ms|ns) +[0-9.]+ MB\n`, got) {
		t.Errorf("exit %d:\n%s", code, got)
	}
}

func regexpMatch(pattern, s string) bool {
	return regexp.MustCompile(pattern).MatchString(s)
}

// memorylimit fails a test that allocates more; a test can set its own.
func TestMemoryLimit(t *testing.T) {
	files := map[string]string{"test_limit.turtle": `import test
import data
memorylimit = 1000000
def test_small[]
    check length of range[0, 10] == 10 .
def [end]
def test_big[]
    b = range[0, 200000]
    check length of b == 200000 .
def [end]
def test_own_limit[]
    memorylimit = none
    b = range[0, 200000]
    check length of b == 200000 .
def [end]
`}
	got, code := testRun(t, files)
	if code == 0 || !regexpMatch(`PASS  test_small `, got) || !regexpMatch(`FAIL  test_big .*\n.*it allocated [0-9.]+ MB; memorylimit is 1.0 MB \(1000000 bytes\)`, got) || !regexpMatch(`PASS  test_own_limit `, got) {
		t.Errorf("exit %d:\n%s", code, got)
	}
	files["test_limit.turtle"] = "import test\nmemorylimit = \"big\"\ndef test_a[]\n    check true .\ndef [end]\n"
	if got, _ := testRun(t, files); !regexpMatch(`memorylimit must be a number of bytes of 0 or more, or none, got "big"`, got) {
		t.Errorf("want a memorylimit error:\n%s", got)
	}
}

// A loop over range counts without its list; a user's own range is used.
func TestLoopOverRange(t *testing.T) {
	out, err := run(t, `import data
s = 0
[loop][i in range[0, 5]]
    s = s + i
[loop][end]
show s .
[loop][i, x in range[10, 0, -3]]
    show i, ":", x .
[loop][end]
fs = list []
[loop][x in data range[3]]
    g = y give y + x * 10
    fs at add g
[loop][end]
h = fs at get[2]
show h[1] .
[loop][x in range[5, 5]]
    show "never" .
[loop][end]`, "")
	if err != nil || out != "10\n0:10\n1:7\n2:4\n3:1\n21\n" {
		t.Errorf("got %q, %v", out, err)
	}
	out, err = run(t, "def range[n]\n    return list [\"mine\"]\ndef [end]\n[loop][x in range[3]]\n    show x .\n[loop][end]", "")
	if err != nil || out != "mine\n" {
		t.Errorf("a program's own range: got %q, %v", out, err)
	}
}
