// Copyright 2017-2026 Nigele McCoy
// Licensed under the Apache License, Version 2.0;
// see LICENSE and NOTICE.

package evaluator

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"Turtle/lexer"
	"Turtle/parser"
)

// fuzzSkip are programs the fuzzer doesn't run: ones that could touch
// files or the system, or ask for an enormous amount of work in a single
// step (a long number: range[0, 9999999999]).
var fuzzSkip = regexp.MustCompile(`import|write|read|append|exit|\d{6}`)

// Any program runs to its end or to a Turtle error, never a crash. The
// seeds are the repository's programs that pass fuzzSkip; go test runs
// them, go test -fuzz=FuzzRun mutates them. Each run gets 300 ms, then
// it's stopped as Ctrl-C stops the REPL.
func FuzzRun(f *testing.F) {
	for _, pattern := range []string{"../testdata/*.turtle", "../bench/*.turtle"} {
		files, _ := filepath.Glob(pattern)
		for _, file := range files {
			if src, err := os.ReadFile(file); err == nil {
				f.Add(string(src))
			}
		}
	}
	f.Add("def f[x]\n    return x * 2\ndef [end]\nshow f[21] .")
	f.Add("nums = list [3, 1, 2]\nsort nums .\nshow nums at get 0 .")
	f.Add("m = map [\"a\": 1]\nshow m at get[\"a\"] .")
	f.Add("safe\n    x = 1 div 0\nhandle [] e .\n    show message of e .\nsafe [end]")

	devnull, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		f.Fatal(err)
	}
	f.Fuzz(func(t *testing.T, src string) {
		if len(src) > 4096 || fuzzSkip.MatchString(src) {
			return
		}
		p := parser.New(lexer.New(src))
		p.ModuleDir = t.TempDir()
		program := p.ParseProgram()
		if len(p.Errors()) > 0 {
			return
		}
		stdout := os.Stdout
		os.Stdout = devnull
		defer func() { os.Stdout = stdout }()

		it := NewWithStdin(t.TempDir(), strings.NewReader(""))
		it.WorkDir = it.Dir
		done := make(chan any, 1)
		go func() {
			defer func() { done <- recover() }()
			it.Run(program) // a Turtle error is fine; a crash panics
		}()
		select {
		case r := <-done:
			fuzzCheck(t, r, src)
		case <-time.After(300 * time.Millisecond):
			it.Interrupt()
			select {
			case r := <-done:
				fuzzCheck(t, r, src)
			case <-time.After(5 * time.Second):
				t.Fatalf("didn't stop when interrupted:\n%s", src)
			}
		}
	})
}

// fuzzCheck fails on a crash: anything Run let through that isn't a
// stop from Interrupt.
func fuzzCheck(t *testing.T, r any, src string) {
	if r == nil {
		return
	}
	if _, ok := r.(interruptRequest); ok {
		return
	}
	t.Fatalf("crashed: %v\nprogram:\n%s", r, src)
}
