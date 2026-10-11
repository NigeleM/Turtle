// Copyright 2017-2026 Nigele McCoy
// Licensed under the Apache License, Version 2.0;
// see LICENSE and NOTICE.

package parser

import (
	"os"
	"path/filepath"
	"testing"

	"Turtle/lexer"
)

// Any text at all parses to a program or to errors, never a crash. The
// seeds are every Turtle program in the repository; go test runs them,
// go test -fuzz=FuzzParse mutates them.
func FuzzParse(f *testing.F) {
	for _, pattern := range []string{"../testdata/*.turtle", "../testdata/*/*.turtle", "../bench/*.turtle", "../evaluator/lib/*.turtle"} {
		files, _ := filepath.Glob(pattern)
		for _, file := range files {
			if src, err := os.ReadFile(file); err == nil {
				f.Add(string(src))
			}
		}
	}
	f.Add("def f[x]\n    return x\ndef [end]")
	f.Add("if ] 1 < 2 [\nshow 1 .\nif [end]")
	f.Fuzz(func(t *testing.T, src string) {
		p := New(lexer.New(src))
		p.ModuleDir = t.TempDir()
		p.ParseProgram()
		p.Errors()
	})
}
