// Copyright 2017-2026 Nigele McCoy
// Licensed under the Apache License, Version 2.0;
// see LICENSE and NOTICE.

package evaluator

import (
	"fmt"
	"math/rand"
	"strings"
	"testing"
)

// Arithmetic worked out as plain numbers (evalNumeric) gives exactly what
// the general path does: the same values, and the same errors (overflow,
// division by zero, text, too big for div), on random expressions.
func TestNumericPathsAgree(t *testing.T) {
	rng := rand.New(rand.NewSource(11))
	leaves := []string{"0", "1", "2", "-3", "7", "65535", "65536", "1000000007",
		"9223372036854775807", "-9223372036854775807", "4611686018427387904",
		"0.0", "2.5", "-1.25", "1e300", "-1e300", "0.1", "a", "b", "c", "s"}
	ops := []string{"+", "-", "*", "/", "div", "%"}
	var gen func(depth int) string
	gen = func(depth int) string {
		if depth == 0 || rng.Intn(4) == 0 {
			return leaves[rng.Intn(len(leaves))]
		}
		return gen(depth-1) + " " + ops[rng.Intn(len(ops))] + " " + gen(depth-1)
	}
	var src strings.Builder
	src.WriteString("a = 12\nb = 0\nc = 2.5\ns = \"x\"\n")
	for i := 0; i < 3000; i++ {
		fmt.Fprintf(&src, "safe\n    show %s .\nhandle [] e .\n    show \"error: \", message of e .\nsafe [end]\n", gen(4))
	}
	numericPaths = true
	fast, err1 := run(t, src.String(), "")
	numericPaths = false
	general, err2 := run(t, src.String(), "")
	numericPaths = true
	if err1 != nil || err2 != nil {
		t.Fatalf("errors: %v / %v", err1, err2)
	}
	if fast != general {
		f, g := strings.Split(fast, "\n"), strings.Split(general, "\n")
		for i := range f {
			if i >= len(g) || f[i] != g[i] {
				t.Fatalf("line %d differs:\n  numeric: %s\n  general: %s", i, f[i], g[i])
			}
		}
		t.Fatal("outputs differ in length")
	}
	if !strings.Contains(fast, "error: ") || !strings.Contains(fast, "overflow") {
		t.Error("the expressions should reach errors and overflow too")
	}
}
