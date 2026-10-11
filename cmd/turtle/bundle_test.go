// Copyright 2017-2026 Nigele McCoy
// Licensed under the Apache License, Version 2.0;
// see LICENSE and NOTICE.

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
	// The library has a theory: its phrase reads in the built program too.
	os.WriteFile(filepath.Join(src, "lib", "greet.trt"), []byte("def greet[n]\n    return \"hello \" + n\ndef [end]\ntheory shout\n    abstract\n        shout is the text t in capitals.\n    notation shout t .\n    definition\n        return t at upper\ntheory [end]\n"), 0o644)
	os.WriteFile(filepath.Join(src, "app.trt"), []byte("import lib/greet\nimport system\nshow greet[args[] at get[0]], \" \", args[] at len, \" \", shout \"hi\" .\n[write] out.txt\n\"made here\"\n[end]\n"), 0o644)
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
	if err != nil || strings.TrimSpace(string(out)) != "hello Ann 2 HI" {
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
		{[]string{"build"}, "usage: turtle build script.turtle"},
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
	// Windows won't replace a running program: building over one says so,
	// and leaves no half-made file behind.
	if runtime.GOOS == "windows" {
		slow := filepath.Join(dir, "slow.trt")
		os.WriteFile(slow, []byte("import time\nsleep[30]\n"), 0o644)
		slowExe := filepath.Join(dir, "slow.exe")
		if out, err := exec.Command(turtle, "build", slow, "-o", slowExe).CombinedOutput(); err != nil {
			t.Fatalf("turtle build: %v\n%s", err, out)
		}
		running := exec.Command(slowExe)
		if err := running.Start(); err != nil {
			t.Fatal(err)
		}
		defer func() { running.Process.Kill(); running.Wait() }()
		if out, _ := exec.Command(turtle, "build", slow, "-o", slowExe).CombinedOutput(); !strings.Contains(string(out), "it's running") {
			t.Errorf("building over a running program: %s", out)
		}
		if _, err := os.Stat(slowExe + ".building"); err == nil {
			t.Error("left slow.exe.building behind")
		}
	}
}
