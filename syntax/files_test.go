// Copyright 2017-2026 Nigele McCoy
// Licensed under the Apache License, Version 2.0;
// see LICENSE and NOTICE.

package syntax

import (
	"strings"
	"testing"
)

func TestTurtleFileEndings(t *testing.T) {
	for name, want := range map[string]bool{
		"a.turtle": true, "a.trt": true, "test_x.turtle": true, "lib/u.trt": true,
		"a.txt": false, ".turtle": false, ".trt": false, "turtle": false, "a.turtle.bak": false,
	} {
		if got := IsTurtleFile(name); got != want {
			t.Errorf("IsTurtleFile(%q) = %v", name, got)
		}
	}
	if TrimExtension("lib/u.turtle") != "lib/u" || TrimExtension("u.trt") != "u" || TrimExtension("u.txt") != "u.txt" {
		t.Error("TrimExtension")
	}
	has := func(exts ...string) func(string) bool {
		return func(e string) bool { return strings.Contains(strings.Join(exts, " "), e) }
	}
	if ext, err := FindModule("u", has(".turtle")); ext != ".turtle" || err != nil {
		t.Errorf("only .turtle: %q %v", ext, err)
	}
	if ext, err := FindModule("u", has(".trt")); ext != ".trt" || err != nil {
		t.Errorf("only .trt: %q %v", ext, err)
	}
	if ext, err := FindModule("u", has()); ext != "" || err != nil {
		t.Errorf("neither: %q %v", ext, err)
	}
	if _, err := FindModule("lib/u", has(".turtle", ".trt")); err == nil || !strings.Contains(err.Error(), "both lib/u.turtle and lib/u.trt") {
		t.Errorf("both: %v", err)
	}
}
