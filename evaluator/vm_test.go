// Copyright 2017-2026 Nigele McCoy
// Licensed under the Apache License, Version 2.0;
// see LICENSE and NOTICE.

package evaluator

import "testing"

// A compiled body does what the tree does: each program here runs both
// ways and must give the same output and the same error, word for word.
// Each one's function must also have compiled, so the test covers it.
func TestCompiledBodiesMatchTheTree(t *testing.T) {
	cases := map[string]string{
		"recursion": `def fib[n]
    if ] n < 2 [
        return n
    if [end]
    return fib[n - 1] + fib[n - 2]
def [end]
show fib[20] .`,
		"overflow to a big number and back": `def grow[x]
    y = x * x * x
    z = y - y + 1
    return list [y, z, y div x, -9223372036854775807 - 1 - 1]
def [end]
show grow[3000000000] .`,
		"floats, division and mixed": `def f[a, b]
    return list [a / b, a div b, a % b, a + 0.5, -a, a * 1.5 < 4, 7 % -3, -7 div 2]
def [end]
show f[7, 2] .`,
		"text and comparisons": `def f[a]
    s = "x" + a + "y"
    return list [s, s == "xay", a < "b", "n={a}", a + 1]
def [end]
show f["a"] .`,
		"maybe local: outer until set": `total = 100
def f[go]
    if ] go [
        total = 5
    if [end]
    return total
def [end]
show f[false], f[true] .`,
		"read before set in a loop": `k = "outer"
def f[]
    out = list []
    [loop][i = 0; i < 3; i++]
        add k to out .
        k = i
    [loop][end]
    return out
def [end]
show f[] .`,
		"loop counter apart from a local of the same name": `def f[]
    i = 99
    s = 0
    [loop][i = 0; i < 4; i++]
        s = s + i
    [loop][end]
    return list [i, s]
def [end]
show f[] .`,
		"break, continue, return inside loops": `def f[n]
    found = -1
    [loop][i = 0; i < n; i++]
        if ] i % 2 == 0 [
            continue
        if [end]
        if ] i > 6 [
            break
        if [end]
        [loop][j = 0; j < 10; j++]
            if ] i * j == 15 [
                return list [i, j]
            if [end]
        [loop][end]
    [loop][end]
    return found
def [end]
show f[20], f[3] .`,
		"while and else-if": `def collatz[n]
    steps = 0
    [loop][n != 1]
        if ] n % 2 == 0 [
            n = n div 2
        else if ] n % 3 == 0 [
            n = 3 * n + 1
        else ]
            n = 3 * n + 1
        if [end]
        steps = steps + 1
    [loop][end]
    return steps
def [end]
show collatz[27] .`,
		"for-each over list, map, text": `def f[]
    out = list []
    [loop][x in list [3, 1, 2]]
        add x * 2 to out .
    [loop][end]
    [loop][k, v in map ["a": 1, "b": 2]]
        add k + "=" + (change v to string) to out .
    [loop][end]
    [loop][i, c in "hi"]
        add i to out .
        add c to out .
    [loop][end]
    [loop][k in map ["z": 0]]
        add k to out .
    [loop][end]
    return out
def [end]
show f[] .`,
		"lists, maps, methods, data statements": `def f[n]
    xs = list []
    [loop][i = 0; i < n; i++]
        add i * i to xs .
    [loop][end]
    put 7 to xs at 0 .
    m = map ["a": 1]
    m at put["b", 2]
    add 5 to m at "a" .
    s = set [1, 2]
    add 3 to s .
    sort xs .
    return list [xs, xs at len, xs at get[2], m, s, length of xs, change "12" to integer]
def [end]
show f[5] .`,
		"globals and library calls": `import math
limit = 3
def f[x]
    return list [abs[x], x at abs, sqrt[16], limit * x, pi > 3]
def [end]
show f[-4] .`,
		"calls into tree functions and back": `def tree[x]
    seed = 1
    return x + 1
def [end]
def vm[x]
    return tree[x] * 2
def [end]
show vm[4] .`,
		"show inside": `def f[x]
    show "x is", x, "and", x * 2 .
    return none
def [end]
f[3]`,
		"division by zero names the line": `def f[x]
    y = 1
    return y / x
def [end]
show f[0] .`,
		"modulo by zero": `def f[x]
    return 5 % x
def [end]
show f[0] .`,
		"fail": `def f[x]
    if ] x < 0 [
        fail "negative: {x}"
    if [end]
    return x
def [end]
show f[2] .
show f[-1] .`,
		"undefined name": `def f[]
    return nothere + 1
def [end]
show f[] .`,
		"wrong argument count": `def g[a, b]
    return a
def [end]
def f[]
    return g[1]
def [end]
show f[] .`,
		"text plus number error": `def f[]
    return list [1] + 2
def [end]
show f[] .`,
		"safe block around a compiled call": `def f[x]
    return 10 / x
def [end]
safe
    show f[0] .
handle [math] e .
    show "caught", kind of e .
safe [end]
show f[5] .`,
		"index error": `def f[]
    xs = list [1]
    put 2 to xs at 3 .
    return xs
def [end]
show f[] .`,
		"too deep": `def down[n]
    return down[n + 1]
def [end]
show down[0] .`,
		"assigning a global it reads stays on the tree": `count = 1
def f[]
    count = count + 1
    return count
def [end]
show f[] .`,
	}
	for name, src := range cases {
		vmEnabled = true
		before := vmCompiledCount.Load()
		vmOut, vmErr := run(t, src, "")
		compiled := vmCompiledCount.Load() > before
		vmEnabled = false
		treeOut, treeErr := run(t, src, "")
		vmEnabled = true
		if vmOut != treeOut || errText(vmErr) != errText(treeErr) {
			t.Errorf("%s:\ncompiled: %q %v\ntree:     %q %v", name, vmOut, vmErr, treeOut, treeErr)
		}
		if !compiled && name != "assigning a global it reads stays on the tree" {
			t.Errorf("%s: nothing compiled", name)
		}
	}
}

func errText(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
