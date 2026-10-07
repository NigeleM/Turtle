package repl

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func noWords() words {
	return words{context: map[string]bool{}, builtin: map[string]bool{"sql_open": true}}
}

// colored names each colored piece of src: "def:keyword double:definition ...".
func colored(src string, w words) string {
	names := map[class]string{keyword: "keyword", constant: "constant", stringLit: "string", comment: "comment", definition: "definition", call: "call", builtin: "builtin"}
	var parts []string
	for _, s := range highlight(src, w) {
		parts = append(parts, src[s.start:s.end]+":"+names[s.c])
	}
	return strings.Join(parts, " ")
}

func TestHighlight(t *testing.T) {
	cases := map[string]string{
		`def double[x]`:                    "def:keyword double:definition",
		`show "hi", 42, 2.5, true, none .`: `show:keyword "hi":string 42:constant 2.5:constant true:constant none:constant`,
		`x = 1 // note`:                    "1:constant // note:comment",
		`//* block *// y = 2`:              "//* block *//:comment 2:constant",
		`total = add_tax[5]`:               "add_tax:call 5:constant",
		`db = sql_open["a.db"]`:            `sql_open:builtin "a.db":string`,
		`if ] x > 1 [`:                     "if:keyword 1:constant",
		`assemble Order [item]`:            "assemble:keyword Order:definition",
		`nums at get[0]`:                   "at:keyword get:call 0:constant",
		`sys ls -la`:                       "sys:keyword",
		`7 div 2`:                          "7:constant div:keyword 2:constant",
		`s = "unfinished`:                  `"unfinished:string`,
		`show "a // not a comment" .`:      `show:keyword "a // not a comment":string`,
	}
	for src, want := range cases {
		if got := colored(src, noWords()); got != want {
			t.Errorf("%s\n got  %s\n want %s", src, got, want)
		}
	}
	// A library's words color once it's imported.
	w := noWords()
	if got := colored(`check x is integer .`, w); got != "is:keyword" {
		t.Errorf("before import test: %s", got)
	}
	w.context["check"] = true
	if got := colored(`check x is integer .`, w); got != "check:keyword is:keyword" {
		t.Errorf("after import test: %s", got)
	}
}

func TestColorizeKeepsTheText(t *testing.T) {
	src := `def f[x] // add one
    return x + 1 . "s"`
	out := colorize(src, noWords())
	plain := strings.NewReplacer("\x1b[0m", "").Replace(out)
	for _, code := range colorCodes {
		plain = strings.ReplaceAll(plain, code, "")
	}
	if plain != src {
		t.Errorf("colorize changed the text:\n%q\n%q", plain, src)
	}
}

func TestNeedsMore(t *testing.T) {
	cases := map[string]bool{
		"x = 1":                                false,
		"def f[x]":                             true,
		"def f[x]\n    return x\ndef [end]":    false,
		"if ] x > 1 [":                         true,
		"if ] x > 1 [\n    show 1 .\nif [end]": false,
		"if ] x [\n  [if ] y [\n  show 1 .\nif [end]": false,
		"[loop][i = 0; i < 3; i++]":                   true,
		"[loop][n in nums]\n show n .\n[loop][end]":   false,
		"safe": true,
		"safe\n x = 1 div 0\nhandle [] e .\n x = 0\nsafe [end]": false,
		"f = x give":                                           true,
		"f = x give\n  return x\ngive [end]":                   false,
		"f = x give x + 1":                                     false,
		"[write] notes.txt":                                    true,
		"[write] notes.txt\n\"hi\"\n[end]":                     false,
		"[read] notes.txt to lines [end]":                      false,
		"validate f[x] with x as integer":                      true,
		"validate f[x] with x as integer\n  that result > 0 .": false,
		"def f[]\n  [loop][true]\n    break\n  [loop][end]":    true,
	}
	for src, want := range cases {
		if got := needsMore(src); got != want {
			t.Errorf("needsMore(%q) = %v, want %v", src, got, want)
		}
	}
	if got := indentFor("def f[]\n  if ] x [\n"); got != 8 {
		t.Errorf("indentFor two open blocks: %d", got)
	}
}

func TestDecodeKeys(t *testing.T) {
	kinds := func(b string) []keyKind {
		var out []keyKind
		for _, k := range decodeKeys([]byte(b)) {
			out = append(out, k.kind)
		}
		return out
	}
	cases := map[string][]keyKind{
		"ab":                               {keyRune, keyRune},
		"\r":                               {keyEnter},
		"\x7f":                             {keyBackspace},
		"\x1b[A\x1b[B":                     {keyUp, keyDown},
		"\x1b[C\x1b[D":                     {keyRight, keyLeft},
		"\x1b[H\x1b[F":                     {keyHome, keyEnd},
		"\x1bOH\x1bOF":                     {keyHome, keyEnd},
		"\x1b[1~\x1b[4~\x1b[3~":            {keyHome, keyEnd, keyDelete},
		"\x1b[1;5C\x1b[1;5D":               {keyWordRight, keyWordLeft},
		"\x1bb\x1bf":                       {keyWordLeft, keyWordRight},
		"\x01\x05\x0b\x15\x17\x03\x04\x0c": {keyHome, keyEnd, keyKillEnd, keyKillStart, keyKillWord, keyCancel, keyEOF, keyClear},
		"é":                                {keyRune},
	}
	for in, want := range cases {
		got := kinds(in)
		if len(got) != len(want) {
			t.Errorf("%q: got %v, want %v", in, got, want)
			continue
		}
		for i := range got {
			if got[i] != want[i] {
				t.Errorf("%q: got %v, want %v", in, got, want)
				break
			}
		}
	}
	if k := decodeKeys([]byte("é")); k[0].r != 'é' {
		t.Errorf("é decoded as %q", k[0].r)
	}
}

func TestLineEditing(t *testing.T) {
	l := &line{}
	typeIn := func(s string) {
		for _, r := range s {
			l.edit(key{kind: keyRune, r: r})
		}
	}
	typeIn("show x .")
	l.edit(key{kind: keyHome})
	l.edit(key{kind: keyWordRight})
	typeIn(" 1 +")
	if l.String() != "show 1 + x ." {
		t.Fatalf("insert: %q", l.String())
	}
	l.edit(key{kind: keyEnd})
	l.edit(key{kind: keyBackspace})
	l.edit(key{kind: keyBackspace})
	if l.String() != "show 1 + x" || l.pos != 10 {
		t.Fatalf("backspace: %q %d", l.String(), l.pos)
	}
	l.edit(key{kind: keyKillWord})
	if l.String() != "show 1 + " {
		t.Fatalf("kill word: %q", l.String())
	}
	l.edit(key{kind: keyHome})
	l.edit(key{kind: keyDelete})
	if l.String() != "how 1 + " {
		t.Fatalf("delete: %q", l.String())
	}
	l.edit(key{kind: keyKillEnd})
	if l.String() != "" {
		t.Fatalf("kill to end: %q", l.String())
	}
	if l.edit(key{kind: keyEnter}) {
		t.Error("Enter is the session's, not an edit")
	}
}

func TestRender(t *testing.T) {
	// The cursor goes back to its place: 3 columns right of the start.
	out, row := render(">>> ", 4, "abc", "abc", 0, 80, 0)
	if !strings.HasPrefix(out, "\r\x1b[J>>> abc") || !strings.HasSuffix(out, "\r\x1b[4C") || row != 0 {
		t.Errorf("short line: %q row %d", out, row)
	}
	// A line wider than the screen wraps; the cursor at the end is on
	// the second row.
	text := strings.Repeat("x", 10)
	out, row = render("> ", 2, text, text, 10, 8, 0)
	if row != 1 || !strings.HasSuffix(out, "\r\x1b[4C") {
		t.Errorf("wrapped: %q row %d", out, row)
	}
	// Redrawing from row 1 first goes up a row.
	out, _ = render("> ", 2, text, text, 0, 8, 1)
	if !strings.HasPrefix(out, "\x1b[1A\r\x1b[J") {
		t.Errorf("redraw from row 1: %q", out)
	}
}

func session(t *testing.T) (*Session, *bytes.Buffer) {
	var out bytes.Buffer
	s := NewSession(t.TempDir(), &out, false)
	t.Cleanup(s.Close)
	return s, &out
}

func TestSession(t *testing.T) {
	s, out := session(t)
	for _, e := range []string{
		"nums = list [3, 1, 2]",
		"nums at length * 2",
		"def double[x]\n    return x * 2\ndef [end]",
		"double[21]",
		`"a" + "b"`,
		"nums",
		"none",
		"1 div 0",
		"show nums .", // show writes to the terminal itself, not to out
		"oops here",
		"x = ",
		"import random",
		"seed = 3",
		"random integer from 5 to 5",
		"length of nums .",
	} {
		s.Eval(e)
	}
	want := `6
42
"ab"
[ 3, 1, 2 ]
error (math): division by zero
error (name): unknown module or variable "oops" in front of a function name
error: no prefix parse function for EOF ("")
5
3
`
	if out.String() != want {
		t.Errorf("got:\n%s\nwant:\n%s", out.String(), want)
	}
}

func TestSessionErrorsName(t *testing.T) {
	s, out := session(t)
	s.Eval("x = 1\ny = x div 0")
	if !strings.Contains(out.String(), "error (math) on line 2: division by zero") {
		t.Errorf("multi-line entry error: %q", out.String())
	}
}

func TestCommands(t *testing.T) {
	s, out := session(t)
	s.Eval("names")
	s.Eval("x = 1")
	s.Eval("def f[]\n    return 2\ndef [end]")
	s.Eval("names")
	s.Eval("help")
	if !strings.Contains(out.String(), "nothing yet\nvariables: x\nfunctions: f\n") || !strings.Contains(out.String(), "Commands (a line with just the word)") {
		t.Errorf("names/help: %q", out.String())
	}
	out.Reset()
	s.Eval("save session.trt")
	if !strings.Contains(out.String(), "saved 2 entries to session.trt") {
		t.Errorf("save: %q", out.String())
	}
	data, _ := os.ReadFile(filepath.Join(s.dir, "session.trt"))
	if string(data) != "x = 1\ndef f[]\n    return 2\ndef [end]\n" {
		t.Errorf("saved file: %q", data)
	}
	s2, out2 := session(t)
	os.WriteFile(filepath.Join(s2.dir, "lib.trt"), []byte("y = 41\ndef g[]\n    return y + 1\ndef [end]\n"), 0o644)
	s2.Eval("load lib.trt")
	s2.Eval("g[]")
	if out2.String() != "42\n" {
		t.Errorf("load: %q", out2.String())
	}
	// A variable called names is yours, not the command.
	s3, out3 := session(t)
	s3.Eval("names = list [1]")
	s3.Eval("names")
	if out3.String() != "[ 1 ]\n" {
		t.Errorf("names variable: %q", out3.String())
	}
	s3.Eval("quit")
	if !s3.Quit() {
		t.Error("quit didn't quit")
	}
	s4, _ := session(t)
	s4.Eval("import system\nexit[0]")
	if !s4.Quit() {
		t.Error("exit[0] didn't quit")
	}
}

func TestSessionRemembersLibraryWords(t *testing.T) {
	s, out := session(t)
	s.Eval("import test")
	s.Eval("check 1 + 1 == 2 .")
	s.Eval("check 1 == 2 .")
	s.Eval("import log")
	s.Eval("logconsole = false")
	s.Eval(`log "quiet" .`)
	if !strings.Contains(out.String(), "error (test): failed: check 1 == 2 .") || strings.Count(out.String(), "error") != 1 {
		t.Errorf("got %q", out.String())
	}
	if !s.words.context["check"] || !s.words.context["log"] {
		t.Error("imported words should color")
	}
}

func TestInterrupt(t *testing.T) {
	s, out := session(t)
	go func() {
		time.Sleep(100 * time.Millisecond)
		s.it.Interrupt()
	}()
	s.Eval("n = 0\n[loop][true]\n    n = n + 1\n[loop][end]")
	if !strings.Contains(out.String(), "stopped (Ctrl-C)") {
		t.Fatalf("got %q", out.String())
	}
	out.Reset()
	s.Eval("n > 0")
	if out.String() != "true\n" {
		t.Errorf("the session goes on after Ctrl-C: %q", out.String())
	}
}

func TestHistoryFile(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("USERPROFILE", os.Getenv("HOME"))
	if h := loadHistory(); len(h) != 0 {
		t.Fatalf("new home has history %q", h)
	}
	saveHistory([]string{"x = 1", "show x ."})
	if h := loadHistory(); strings.Join(h, "|") != "x = 1|show x ." {
		t.Errorf("got %q", h)
	}
	many := make([]string, historyMax+5)
	for i := range many {
		many[i] = "line"
	}
	many[len(many)-1] = "last"
	saveHistory(many)
	if h := loadHistory(); len(h) != historyMax || h[len(h)-1] != "last" {
		t.Errorf("kept %d lines", len(h))
	}
}

// TestOneClearParseError: a line that isn't valid Turtle gets one error,
// not the parser's follow-on complaints.
func TestOneClearParseError(t *testing.T) {
	s, out := session(t)
	s.Eval("a = list [1, 2, 3]")
	s.Eval("a at give x + 1")
	s.Eval("show 1 +")
	if strings.Count(out.String(), "error") != 2 {
		t.Errorf("want one error per entry, got:\n%s", out.String())
	}
	if !strings.Contains(out.String(), `unexpected "x" after the expression`) {
		t.Errorf("got:\n%s", out.String())
	}
}
