// Copyright 2017-2026 Nigele McCoy
// Licensed under the Apache License, Version 2.0;
// see LICENSE and NOTICE.

package evaluator

import (
	"strings"
	"testing"
)

// Proofs and theorems: "fails" in a proof, setup lines shared by a proof's
// checks, an "if" condition on a theorem, and sentence theorems (each, at
// least, ...) as verify reads them. Each theory is good, or broken on purpose.
func TestProofsAndTheoremSentences(t *testing.T) {
	lib := `theory average
    abstract
        average says the average of the numbers in a list a.
    notation average a .
    definition
        if ] length of a == 0 [
            fail "average of an empty list"
        if [end]
        total = 0
        [loop][x in a]
            total = total + x
        [loop][end]
        return total / length of a
    theorem result >= min of a
    proof
        average list [2, 4, 6] . is 4
        average list [] . fails
theory [end]

theory squares
    abstract
        squares says the list of each item of a multiplied by itself.
    notation squares a .
    definition
        out = list []
        [loop][x in a]
            add x * x to out .
        [loop][end]
        return out
    theorem result each x give x >= 0
    proof
        nums = list [-3, 0, 2]
        squares nums . is list [9, 0, 4]
        big = list [10, 20]
        squares big . is list [100, 400]
theory [end]

theory strict
    abstract
        strict says the squares of a, too strong for zero.
    notation strict a .
    definition
        out = list []
        [loop][x in a]
            add x * x to out .
        [loop][end]
        return out
    theorem result each x give x > 0
    proof
        strict list [3] . is list [9]
theory [end]

theory clamp
    abstract
        clamp says n kept between lo and hi.
    notation clamp n between lo and hi .
    definition
        if ] n < lo [
            return lo
        if [end]
        if ] n > hi [
            return hi
        if [end]
        return n
    theorem result >= lo if lo <= hi
    theorem result <= hi if lo <= hi
    proof
        clamp 5 between 0 and 10 . is 5
        clamp -4 between 0 and 10 . is 0
theory [end]

theory brokenclamp
    abstract
        brokenclamp says n kept between lo and hi, but one short at lo.
    notation brokenclamp n between lo and hi .
    definition
        if ] n < lo [
            return lo
        if [end]
        if ] n > hi [
            return hi
        if [end]
        return n - 1
    theorem result >= lo if lo <= hi
    proof
        brokenclamp 5 between 0 and 10 . is 4
theory [end]

theory wrongfail
    abstract
        wrongfail says n plus one.
    notation wrongfail n .
    definition
        return n + 1
    proof
        wrongfail 1 . fails
theory [end]

theory somebig
    abstract
        somebig says the squares of a, with at least one item at least 4.
    notation somebig a .
    definition
        out = list []
        [loop][x in a]
            add x * x to out .
        [loop][end]
        return out
    theorem result at least 1 x give x >= 4
    proof
        somebig list [2, 3] . is list [4, 9]
theory [end]
`
	out, code := testRun(t, map[string]string{
		"lib/lib.turtle":  lib,
		"test_lib.turtle": "import test\nimport lib/lib\nseed = 1\n",
	})
	if code != 1 {
		t.Errorf("exit %d, want 1", code)
	}
	for _, want := range []string{
		"PASS  theory average",
		"PASS  theory squares",
		"PASS  theory clamp",
		"FAIL  theory strict",
		"theorem result each x give x > 0 fails on a = [ 0 ]: 1 of 1 item breaks the rule: at 0: 0",
		"FAIL  theory brokenclamp",
		"theorem result >= lo if lo <= hi fails on n = 0, lo = 0, hi = 0",
		"FAIL  theory wrongfail",
		"proof: wrongfail 1 . fails . gave 2, expected an error",
		"FAIL  theory somebig",
		"theorem result at least 1 x give x >= 4 fails on a = [  ]: 0 of 0 items follow the rule; at least 1 should",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
}

// A proof's setup lines must be followed by a check.
func TestProofSetupNeedsACheck(t *testing.T) {
	src := `theory dangling
    abstract
        dangling says n plus one.
    notation dangling n .
    definition
        return n + 1
    proof
        x = 3
theory [end]
`
	out, code := testRun(t, map[string]string{"lib/dangling.turtle": src, "test_d.turtle": "import test\nimport lib/dangling\nseed = 1\n"})
	if code != 1 || !strings.Contains(out, "a proof's setup lines must be followed by a check") {
		t.Errorf("exit %d, want the setup error; got:\n%s", code, out)
	}
}
