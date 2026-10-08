package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestBuild compiles turtle, makes a program with turtle build (a script
// importing a .trt file), and runs it from another folder: on Windows,
// macOS and Linux, as CI runs this on each.
func TestBuild(t *testing.T) {
	if testing.Short() {
		t.Skip("compiles turtle")
	}
	dir := t.TempDir()
	exe := ".exe"
	if runtime.GOOS != "windows" {
		exe = ""
	}
	turtle := filepath.Join(dir, "turtle"+exe)
	if out, err := exec.Command("go", "build", "-o", turtle, ".").CombinedOutput(); err != nil {
		t.Fatalf("go build: %v\n%s", err, out)
	}
	src := filepath.Join(dir, "src")
	os.MkdirAll(filepath.Join(src, "lib"), 0o755)
	os.WriteFile(filepath.Join(src, "lib", "greet.trt"), []byte("def greet[n]\n    return \"hello \" + n\ndef [end]\n"), 0o644)
	os.WriteFile(filepath.Join(src, "app.trt"), []byte("import lib/greet\nimport system\nshow greet[args[] at get[0]], \" \", args[] at len .\n[write] out.txt\n\"made here\"\n[end]\n"), 0o644)
	cmd := exec.Command(turtle, "build", "app.trt", "-o", filepath.Join(dir, "app"))
	cmd.Dir = src
	if out, err := cmd.CombinedOutput(); err != nil || !strings.Contains(string(out), "built ") {
		t.Fatalf("turtle build: %v\n%s", err, out)
	}
	app := filepath.Join(dir, "app"+exe)
	// Run from another folder, with the source gone: everything it needs
	// is packed in it.
	os.RemoveAll(src)
	work := filepath.Join(dir, "work")
	os.Mkdir(work, 0o755)
	run := exec.Command(app, "Ann", "two words")
	run.Dir = work
	out, err := run.CombinedOutput()
	if err != nil || strings.TrimSpace(string(out)) != "hello Ann 2" {
		t.Fatalf("running the built program: %v\n%s", err, out)
	}
	if data, err := os.ReadFile(filepath.Join(work, "out.txt")); err != nil || strings.TrimSpace(string(data)) != "made here" {
		t.Errorf("files go where it's run: %v %q", err, data)
	}
	// A turtle that's a built program won't build more.
	again := exec.Command(app, "x")
	again.Dir = work
	if out, _ := again.CombinedOutput(); strings.Contains(string(out), "usage") {
		t.Errorf("a built program answered as turtle: %s", out)
	}
	// Mistakes: no script, a script that doesn't parse, a missing import.
	for _, c := range []struct {
		args []string
		want string
	}{
		{[]string{"build"}, "usage: turtle build script.trt"},
		{[]string{"build", "nope.trt"}, "nope.trt"},
	} {
		out, _ := exec.Command(turtle, c.args...).CombinedOutput()
		if !strings.Contains(string(out), c.want) {
			t.Errorf("%v: got %s", c.args, out)
		}
	}
	bad := filepath.Join(dir, "bad.trt")
	os.WriteFile(bad, []byte("import missinglib\nshow 1 .\n"), 0o644)
	if out, _ := exec.Command(turtle, "build", bad).CombinedOutput(); !strings.Contains(string(out), "imports missinglib") {
		t.Errorf("missing import: %s", out)
	}
}
