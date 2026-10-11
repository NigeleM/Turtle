// Copyright 2017-2026 Nigele McCoy
// Licensed under the Apache License, Version 2.0;
// see LICENSE and NOTICE.

package evaluator

import (
	"os"
	"path/filepath"
	"testing"
)

// A call keeps the function it found only while nothing could change
// which function its name reaches.
func TestCallsReachTheRightFunction(t *testing.T) {
	cases := []struct{ name, src, want string }{
		{"a variable holding a function wins over a def of that name",
			"def f[x]\n    return \"def\"\ndef [end]\ndef run[]\n    f = x give \"variable\"\n    return f[1]\ndef [end]\nshow f[1], \" \", run[], \" \", f[1] .",
			"def variable def\n"},
		{"the same call with and without such a variable, in recursion",
			"def f[x]\n    return \"def\"\ndef [end]\ndef walk[n]\n    if ] n == 1 [\n        f = x give \"local\"\n    if [end]\n    out = f[0]\n    if ] n < 3 [\n        out = out + \",\" + walk[n + 1]\n    if [end]\n    return out\ndef [end]\nshow walk[0] .",
			"def,local,def,def\n"},
		{"a function defined again",
			"def g[]\n    return 1\ndef [end]\ndef use[]\n    return g[]\ndef [end]\na = use[]\ndef g[]\n    return 2\ndef [end]\nshow a, use[] .",
			"12\n"},
		{"a parameter holding a function",
			"def twice[f, x]\n    return f[f[x]]\ndef [end]\ndef inc[x]\n    return x + 1\ndef [end]\nshow twice[inc, 1], \" \", twice[x give x * 3, 1] .",
			"3 9\n"},
		{"recursion", "def fib[n]\n    if ] n < 2 [\n        return n\n    if [end]\n    return fib[n - 1] + fib[n - 2]\ndef [end]\nshow fib[15] .", "610\n"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			out, err := run(t, c.src, "")
			if err != nil || out != c.want {
				t.Errorf("got %q, %v; want %q", out, err, c.want)
			}
		})
	}
}

// A module's function and the main file's function of the same name stay
// apart, though their calls share one name.
func TestCallsKeepModulesApart(t *testing.T) {
	dir := t.TempDir()
	lib := "def name[]\n    return \"lib\"\ndef [end]\ndef ask[]\n    return name[]\ndef [end]\n"
	if err := os.WriteFile(filepath.Join(dir, "lib.turtle"), []byte(lib), 0o644); err != nil {
		t.Fatal(err)
	}
	src := "import lib [ask]\ndef name[]\n    return \"main\"\ndef [end]\nshow name[], \" \", ask[], \" \", name[], \" \", ask[] ."
	out, err := runIn(t, dir, src, "")
	if err != nil || out != "main lib main lib\n" {
		t.Errorf("got %q, %v", out, err)
	}
}
