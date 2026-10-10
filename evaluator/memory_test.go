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
