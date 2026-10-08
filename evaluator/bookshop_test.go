package evaluator

import (
	"bufio"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"Turtle/lexer"
	"Turtle/parser"
)

// TestBookshopScript runs testdata/bookshop: one program using most of
// Turtle together (a module, config, scrolls, diagnose, closures, loops,
// errors, time zones, JSON, crypt, patterns, SQLite, files, log, random,
// schedule, and a web server it calls itself). It checks its own results
// and exits with how many failed.
func TestBookshopScript(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the script calls its server with sh and curl")
	}
	if _, err := exec.LookPath("curl"); err != nil {
		t.Skip("no curl")
	}
	work := t.TempDir()
	for _, f := range []string{"bookshop.trt", "shoplib.trt"} {
		data, err := os.ReadFile(filepath.Join("..", "testdata", "bookshop", f))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(work, f), data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	src, _ := os.ReadFile(filepath.Join(work, "bookshop.trt"))
	p := parser.New(lexer.New(string(src)))
	program := p.ParseProgram()
	if errs := p.Errors(); len(errs) > 0 {
		t.Fatal(errs)
	}
	r, w, _ := os.Pipe()
	orig := os.Stdout
	os.Stdout = w
	outCh := make(chan string)
	go func() {
		var b strings.Builder
		sc := bufio.NewScanner(r)
		for sc.Scan() {
			b.WriteString(sc.Text() + "\n")
		}
		outCh <- b.String()
	}()
	it := New(work)
	it.WorkDir = work
	err := it.Run(program)
	w.Close()
	os.Stdout = orig
	out := <-outCh
	if ex, ok := err.(ExitRequest); !ok || ex.Code != 0 {
		t.Fatalf("want exit 0, got %v\n%s", err, out)
	}
	for _, want := range []string{"scrolls ok", "time zones ok", "sql ok", "server ok", "0 failed"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
	left, _ := os.ReadDir(work)
	if len(left) != 2 {
		var names []string
		for _, e := range left {
			names = append(names, e.Name())
		}
		t.Errorf("files left behind: %v", names)
	}
}
