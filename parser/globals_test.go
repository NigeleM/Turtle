// Copyright 2017-2026 Nigele McCoy
// Licensed under the Apache License, Version 2.0;
// see LICENSE and NOTICE.

package parser

import (
	"strings"
	"testing"

	"Turtle/lexer"
)

// A function that assigns a top-level name makes a local of it: warned, once
// per name, unless the name is its parameter or the value reads it.
func TestWarnsWhenAFunctionShadowsATopLevelName(t *testing.T) {
	cases := []struct {
		name, src string
		warnings  int
	}{
		{"assigns a global", "best = 1\ndef set_best[v]\n    if ] v < best [\n        best = v\n    if [end]\ndef [end]\n", 1},
		{"a parameter of the same name", "best = 1\ndef f[best]\n    best = 2\ndef [end]\n", 0},
		{"reads the name (already an error when it runs)", "count = 0\ndef bump[]\n    count = count + 1\ndef [end]\n", 0},
		{"only the first assignment warns", "x = 0\ndef f[]\n    x = 1\n    x = 2\ndef [end]\n", 1},
		{"no top-level name", "def f[]\n    y = 1\ndef [end]\n", 0},
	}
	for _, c := range cases {
		p := New(lexer.New(c.src))
		p.ParseProgram()
		got := 0
		for _, w := range p.Warnings() {
			if strings.Contains(w.Msg, "also a top-level variable") {
				got++
			}
		}
		if got != c.warnings {
			t.Errorf("%s: %d warnings, want %d (%v)", c.name, got, c.warnings, p.Warnings())
		}
	}
}

// A number alone in braces in text is noted, since it's read as the value;
// a variable or an expression in braces is not.
func TestNoteForANumberInBraces(t *testing.T) {
	cases := []struct {
		src  string
		want int
	}{
		{"show \"f(g(h[1]{2}))\" .\n", 1},
		{"x = 3\nshow \"{x} and {1+1}\" .\n", 0},
	}
	for _, c := range cases {
		p := New(lexer.New(c.src))
		p.ParseProgram()
		got := 0
		for _, w := range p.Warnings() {
			if strings.Contains(w.Msg, "is the value 2, not the text") {
				got++
			}
		}
		if got != c.want {
			t.Errorf("%q: %d notes, want %d (%v)", c.src, got, c.want, p.Warnings())
		}
	}
}
