package format

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
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

// Every .trt file in the repository formats, and formatting it again
// changes nothing.
func TestFormatEveryFile(t *testing.T) {
	var files []string
	filepath.WalkDir("..", func(p string, d os.DirEntry, err error) error {
		if err == nil && !d.IsDir() && strings.HasSuffix(p, ".trt") {
			files = append(files, p)
		}
		return nil
	})
	if len(files) < 10 {
		t.Fatalf("found only %d .trt files", len(files))
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
