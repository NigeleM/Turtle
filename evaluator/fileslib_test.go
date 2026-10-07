package evaluator

import (
	"archive/zip"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"Turtle/lexer"
	"Turtle/parser"
)

// system's copyto, moveto, makefolder, walk, pack, unpack, loadenv and
// options, and turtle trace.

func writeFiles(t *testing.T, dir string, files map[string]string) {
	t.Helper()
	for name, text := range files {
		path := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestFoldersAndArchives(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{"src/a.txt": "hi", "src/sub/b.txt": "there"})
	src := `import system
show walk["src"] .
copyto["src", "copy"]
show walk["copy"] .
copyto["src/a.txt", "out/deep/a.txt"]
moveto["out/deep/a.txt", "out/renamed.txt"]
show walk["out"] .
makefolder["x/y/z"]
show isfolder["x/y/z"] .
[loop][kind in list ["zip", "tar", "tar.gz", "tgz"]]
    pack["src", "src." + kind]
    unpack["src." + kind, "back_" + kind]
    show walk["back_" + kind] .
[loop][end]
pack["src/a.txt", "one.zip"]
unpack["one.zip", "one"]
show walk["one"] .
[read] back_tgz/src/sub/b.txt to lines [end]
show lines .
safe
    copyto["src", "copy"]
handle [file] e .
    show message of e .
safe [end]
copyto["src", "copy", true]
safe
    copyto["src", "src/inner"]
handle [file] e .
    show message of e .
safe [end]
safe
    pack["src", "src.rar"]
handle [file] e .
    show message of e .
safe [end]
safe
    unpack["src.zip", "back_zip"]
handle [file] e .
    show message of e .
safe [end]
unpack["src.zip", "back_zip", true]
safe
    moveto[".", "elsewhere"]
handle [file] e .
    show message of e .
safe [end]`
	got, err := runIn(t, dir, src, "")
	if err != nil {
		t.Fatal(err)
	}
	want := `[ "src/a.txt", "src/sub/b.txt" ]
[ "copy/a.txt", "copy/sub/b.txt" ]
[ "out/renamed.txt" ]
true
[ "back_zip/src/a.txt", "back_zip/src/sub/b.txt" ]
[ "back_tar/src/a.txt", "back_tar/src/sub/b.txt" ]
[ "back_tar.gz/src/a.txt", "back_tar.gz/src/sub/b.txt" ]
[ "back_tgz/src/a.txt", "back_tgz/src/sub/b.txt" ]
[ "one/a.txt" ]
[ "there" ]
copyto: copy is already there; add true as the last argument to replace it
copyto: can't copy the folder src into itself (src/inner)
pack src.rar: an archive name ends in .zip, .tar, .tar.gz or .tgz
unpack: back_zip/src/a.txt is already there; add true as the last argument to replace it
moveto .: won't change the folder turtle is running in, or a folder above it
`
	if got != want {
		t.Errorf("got\n%s\nwant\n%s", got, want)
	}
	if _, err := os.Stat(filepath.Join(dir, "out", "deep", "a.txt")); err == nil {
		t.Error("moveto left the original")
	}
}

func TestUnpackRefusesEntriesOutside(t *testing.T) {
	dir := t.TempDir()
	f, err := os.Create(filepath.Join(dir, "evil.zip"))
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(f)
	w, _ := zw.Create("../escaped.txt")
	w.Write([]byte("x"))
	zw.Close()
	f.Close()
	_, err = runIn(t, dir, `import system
unpack["evil.zip", "inside"]`, "")
	if err == nil || !strings.Contains(err.Error(), "would land outside") {
		t.Errorf("got %v, want a refusal", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "escaped.txt")); err == nil {
		t.Error("the entry was written outside the folder")
	}
}

func TestLoadenv(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{".env": "# settings\n\nTT_KEY=abc123\nexport TT_NAME=\"Ann \\\"L\\\"\"\nTT_RAW='a $b # c'\nTT_COUNT=5 # five\nTT_SET=from file\n"})
	t.Setenv("TT_SET", "from outside")
	for _, k := range []string{"TT_KEY", "TT_NAME", "TT_RAW", "TT_COUNT"} {
		t.Setenv(k, "") // restored after the test
		os.Unsetenv(k)
	}
	got, err := runIn(t, dir, `import system
s = loadenv[]
show s .
show env["TT_KEY"], "|", env["TT_SET"] .`, "")
	if err != nil {
		t.Fatal(err)
	}
	want := `{ "TT_KEY": "abc123", "TT_NAME": "Ann \"L\"", "TT_RAW": "a $b # c", "TT_COUNT": "5", "TT_SET": "from file" }` + "\nabc123|from outside\n"
	if got != want {
		t.Errorf("got  %q\nwant %q", got, want)
	}
	writeFiles(t, dir, map[string]string{"bad.env": "no equals here\n"})
	if _, err := runIn(t, dir, "import system\nloadenv[\"bad.env\"]", ""); err == nil || !strings.Contains(err.Error(), "line 1 isn't KEY=value") {
		t.Errorf("bad line: got %v", err)
	}
}

func TestOptions(t *testing.T) {
	src := `import system
opts = options["--out": "result.csv", "-v": false, "--count": 1, "--rate": 0.5]
show opts .
show args[] .`
	cases := []struct {
		args []string
		want string
	}{
		{nil, "{ \"--out\": \"result.csv\", \"-v\": false, \"--count\": 1, \"--rate\": 0.5 }\n[  ]\n"},
		{[]string{"data.csv", "--out=r.csv", "-v", "--count", "3", "more"}, "{ \"--out\": \"r.csv\", \"-v\": true, \"--count\": 3, \"--rate\": 0.5 }\n[ \"data.csv\", \"more\" ]\n"},
		{[]string{"--rate", "2", "-v=false", "--", "--out"}, "{ \"--out\": \"result.csv\", \"-v\": false, \"--count\": 1, \"--rate\": 2.0 }\n[ \"--out\" ]\n"},
		{[]string{"-5"}, "{ \"--out\": \"result.csv\", \"-v\": false, \"--count\": 1, \"--rate\": 0.5 }\n[ \"-5\" ]\n"},
	}
	for _, c := range cases {
		got, err := runFull(t, ".", src, "", c.args)
		if err != nil {
			t.Errorf("%v: %v", c.args, err)
		} else if got != c.want {
			t.Errorf("%v:\n got  %q\n want %q", c.args, got, c.want)
		}
	}
	for args, wantErr := range map[string]string{
		"--bogus":   "unknown option --bogus; the options are --out, -v, --count, --rate",
		"--count x": `option --count takes a whole number; "x" isn't one`,
		"--out":     "option --out needs a value after it",
		"-v=maybe":  `option -v is on or off; "maybe" isn't true or false`,
	} {
		_, err := runFull(t, ".", src, "", strings.Fields(args))
		if err == nil || !strings.Contains(err.Error(), wantErr) {
			t.Errorf("%s: got %v, want %q", args, err, wantErr)
		}
	}
	got, err := runFull(t, ".", src, "", []string{"--help"})
	if _, exited := err.(ExitRequest); !exited || !strings.Contains(got, "--count  default 1") || !strings.Contains(got, "-v       on or off") {
		t.Errorf("--help: got %q, %v", got, err)
	}
	if _, err := run(t, "import system\nopts = options[\"out\": 1]", ""); err == nil || !strings.Contains(err.Error(), "isn't an option name") {
		t.Errorf("a name without -: got %v", err)
	}
}

func TestTrace(t *testing.T) {
	src := `nums = list [5, 7]
total = 0
[loop][x in nums]
    total = total + x
[loop][end]
d = double[total]

def double[n]
    return n * 2
def [end]`
	program := parser.New(lexer.New(src)).ParseProgram()
	var trace strings.Builder
	it := New(".")
	it.Trace = &trace
	it.TraceSource("", src)
	if err := it.Run(program); err != nil {
		t.Fatal(err)
	}
	want := `line 1   nums = list [5, 7]               nums = [ 5, 7 ]
line 2   total = 0                        total = 0
line 3   [loop][x in nums]
line 4       total = total + x            total = 5
line 4       total = total + x            total = 12
line 6   d = double[total]
line 9       return n * 2
line 6   ...                              d = 24
`
	if trace.String() != want {
		t.Errorf("got\n%s\nwant\n%s", trace.String(), want)
	}
}

func TestCallWithPairsIsOneMap(t *testing.T) {
	src := `def f[m]
    return m
def [end]
show f["a": 1, "b": 2 + 3] .
show typeof[f["k": list [1]]] .
x = 4
show f[x: "four"] .`
	got, err := run(t, src, "")
	if err != nil {
		t.Fatal(err)
	}
	if want := "{ \"a\": 1, \"b\": 5 }\nmap\n{ 4: \"four\" }\n"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
