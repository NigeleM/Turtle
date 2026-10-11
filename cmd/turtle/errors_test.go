// Copyright 2017-2026 Nigele McCoy
// Licensed under the Apache License, Version 2.0;
// see LICENSE and NOTICE.

package main

import (
	"io"
	"os"
	"strings"
	"testing"
)

// stderrOf runs src as the main script and gives what it wrote to stderr.
func stderrOf(t *testing.T, src string) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	stderr := os.Stderr
	os.Stderr = w
	runWatched(src, t.TempDir(), "main.turtle", nil, "")
	w.Close()
	os.Stderr = stderr
	out, _ := io.ReadAll(r)
	return string(out)
}

// A runtime error shows the line it happened on, as a parse error does.
func TestRuntimeErrorShowsItsLine(t *testing.T) {
	got := stderrOf(t, "x = 1\nshow nope .\n")
	want := "turtle: line 2: undefined variable \"nope\"\n  2 | show nope .\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestParseErrorWording(t *testing.T) {
	cases := map[string]string{
		"x = list [1, 2]]\n": "this ']' has no '[' to close",
		"[end]\n":            "this [end] closes nothing",
		"x = 🐢\n":            "found '🐢'",
	}
	for src, want := range cases {
		if got := stderrOf(t, src); !strings.Contains(got, want) {
			t.Errorf("%q: got %q, want it to say %q", src, got, want)
		}
	}
}
