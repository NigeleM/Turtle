package format

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"Turtle/syntax"
)

func TestFormat(t *testing.T) {
	src := "\n\nx = 5   \n" +
		"def f[a]\n" +
		"  // a comment\n" +
		"      if ] a > 1 [\n" +
		"  show a .\n" +
		"[if ] a > 2 [\n" +
		"show 2 .\n" +
		"[else ]\n" +
		"show 3 .\n" +
		"  else ]\n" +
		"show 4 .\n" +
		"if [end]\n" +
		"\n\n\n\n" +
		"  [loop][i in list [1]]\n" +
		"safe\n" +
		"show i .\n" +
		"handle [] e .\n" +
		"show e .\n" +
		"safe [end]\n" +
		"[loop][end]\n" +
		"g = x give\n" +
		"return x\n" +
		"give [end]\n" +
		"  return `raw\n   kept   \n  as is`\n" +
		"def [end]\n" +
		"//* block\n      inside\n*//\n" +
		"[write] out.txt\n\"a\"\n[end]\n"
	want := "x = 5\n" +
		"def f[a]\n" +
		"    // a comment\n" +
		"    if ] a > 1 [\n" +
		"        show a .\n" +
		"        [if ] a > 2 [\n" +
		"            show 2 .\n" +
		"        [else ]\n" +
		"            show 3 .\n" +
		"    else ]\n" +
		"        show 4 .\n" +
		"    if [end]\n" +
		"\n\n" +
		"    [loop][i in list [1]]\n" +
		"        safe\n" +
		"            show i .\n" +
		"        handle [] e .\n" +
		"            show e .\n" +
		"        safe [end]\n" +
		"    [loop][end]\n" +
		"    g = x give\n" +
		"        return x\n" +
		"    give [end]\n" +
		"    return `raw\n   kept   \n  as is`\n" +
		"def [end]\n" +
		"//* block\n      inside\n*//\n" +
		"[write] out.txt\n    \"a\"\n[end]\n"
	got, err := Format(src)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Errorf("got\n%s\nwant\n%s", got, want)
	}
	if again, _ := Format(got); again != got {
		t.Errorf("formatting twice changed it again:\n%s", again)
	}
	// Line endings stay as they were.
	if crlf, _ := Format("if ] 1 > 0 [\r\nshow 1 .\r\nif [end]\r\n"); crlf != "if ] 1 > 0 [\r\n    show 1 .\r\nif [end]\r\n" {
		t.Errorf("CRLF: got %q", crlf)
	}
	// validate's rule, one step in.
	test := "import test\ndef test_a[]\nvalidate f[n] with n as integer\nthat result > 0 .\ndef [end]\n"
	if got, _ := Format(test); !strings.Contains(got, "\n    validate f[n] with n as integer\n        that result > 0 .\n") {
		t.Errorf("validate: got\n%s", got)
	}
	// Code that doesn't parse is left alone.
	if _, err := Format("x = \n"); err == nil || !strings.Contains(err.Error(), "doesn't parse") {
		t.Errorf("broken code: got %v", err)
	}
}

// Every Turtle file in the repository formats, and formatting it again
// changes nothing.
func TestFormatEveryFile(t *testing.T) {
	var files []string
	filepath.WalkDir("..", func(p string, d os.DirEntry, err error) error {
		if err == nil && !d.IsDir() && syntax.IsTurtleFile(p) {
			files = append(files, p)
		}
		return nil
	})
	if len(files) < 10 {
		t.Fatalf("found only %d Turtle files", len(files))
	}
	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		once, err := Format(string(data))
		if err != nil {
			if strings.Contains(f, "parse_error") || strings.Contains(f, "bad") {
				continue
			}
			t.Errorf("%s: %v", f, err)
			continue
		}
		if twice, _ := Format(once); twice != once {
			t.Errorf("%s: formatting twice changed it again", f)
		}
	}
}

func TestFormatScroll(t *testing.T) {
	src := "def f[raw]\nnames is scroll raw into\nkeep s give s != \"\",\n      join[\", \"] .\nreturn names\ndef [end]\nx is scroll 3 into here + 1 .\n"
	want := "def f[raw]\n    names is scroll raw into\n        keep s give s != \"\",\n        join[\", \"] .\n    return names\ndef [end]\nx is scroll 3 into here + 1 .\n"
	got, err := Format(src)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Errorf("got\n%s\nwant\n%s", got, want)
	}
}

func TestFormatDiagnoseBlock(t *testing.T) {
	got, err := Format("diagnose\nx = 1\n[loop][i in list [1]]\nx = x + i\n[loop][end]\ndiagnose [end]\n")
	want := "diagnose\n    x = 1\n    [loop][i in list [1]]\n        x = x + i\n    [loop][end]\ndiagnose [end]\n"
	if err != nil || got != want {
		t.Errorf("got %v\n%s", err, got)
	}
}

func TestFormatMatrix(t *testing.T) {
	src := "import linear\ndef f[]\nm = matrix [\n1, 2\n      3, 4\n]\nn = matrix [\n1, 2\n3, 4]\nreturn m\ndef [end]\n"
	want := "import linear\ndef f[]\n    m = matrix [\n        1, 2\n        3, 4\n    ]\n    n = matrix [\n        1, 2\n        3, 4]\n    return m\ndef [end]\n"
	got, err := Format(src)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Errorf("got\n%s\nwant\n%s", got, want)
	}
}

// Maps, lists and calls left open over several lines: what's inside goes
// one step in, whatever the bracket, and however many open on one line.
func TestFormatOpenBrackets(t *testing.T) {
	src := "routes = map [\n\"GET /\": home,\n\"nested\": map [\n\"a\": 1\n],\n\"GET /health\": \"ok\"\n]\n" +
		"def f[]\nrows = list [\n1, 2,\n3]\nx = sum[list [\n1, 2\n]]\nreturn x\ndef [end]\n"
	want := "routes = map [\n    \"GET /\": home,\n    \"nested\": map [\n        \"a\": 1\n    ],\n    \"GET /health\": \"ok\"\n]\n" +
		"def f[]\n    rows = list [\n        1, 2,\n        3]\n    x = sum[list [\n        1, 2\n    ]]\n    return x\ndef [end]\n"
	got, err := Format(src)
	if err != nil || got != want {
		t.Errorf("got %q, %v\nwant %q", got, err, want)
	}
}

// A scroll finished on its own line, inside brackets, isn't one going on
// to the next lines.
func TestFormatScrollOnOneLine(t *testing.T) {
	src := "res = diagnose[scroll 3 into add1, here * 2 .]\nshow res .\n"
	if got, err := Format(src); err != nil || got != src {
		t.Errorf("got %q, %v", got, err)
	}
}
