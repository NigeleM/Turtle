// Copyright 2017-2026 Nigele McCoy
// Licensed under the Apache License, Version 2.0;
// see LICENSE and NOTICE.

package evaluator

import (
	"strings"
	"testing"

	"Turtle/lexer"
	"Turtle/parser"
)

func TestRandomShapes(t *testing.T) {
	cases := []struct{ name, src, want, wantErr string }{
		{name: "integers stay in range", src: `import random
ok = true
[loop][i in list [1, 2, 3, 4, 5, 6, 7, 8, 9, 10]]
    n = random integer from 1 to 6
    if ] n < 1 || n > 6 [
        ok = false
    if [end]
[loop][end]
show ok .`, want: "true\n"},
		{name: "a range of one", src: `import random
show random integer from 7 to 7, " ", random float from 2.5 to 2.5, " ", random date from "2026-10-06" to "2026-10-06" .`, want: "7 2.5 2026-10-06 00:00:00\n"},
		{name: "counts and lengths", src: `import random
show length of random list of 5 integers .
show length of random string of 8 .
show length of random digits of 4 .
show length of random set of 3 integers .
show length of random map of 4 integer to string .
n = 6
show length of random list of n booleans .`, want: "5\n8\n4\n3\n4\n6\n"},
		{name: "count and length ranges", src: `import random
l = length of random list of 2 to 4 floats
t = length of random string of 3 to 5
show l >= 2 && l <= 4, t >= 3 && t <= 5 .`, want: "truetrue\n"},
		{name: "text from characters", src: `import random
show random string of 6 from "a" .
show random digits of 3 at contains["x"] .`, want: "aaaaaa\nfalse\n"},
		{name: "rounded floats", src: `import random
import math
f = random float from 0 to 10 rounded to 2
cents = f * 100
whole = cents at round
diff = cents - whole
show diff < 0.000001 && diff > -0.000001 .
show random float from 1 to 1 rounded to 0 .`, want: "true\n1.0\n"},
		{name: "nested shapes", src: `import random
g = random list of 3 lists of 2 integers from 0 to 0
show g .
show random set of 1 list of 2 strings of 1 from "z" .
show random map of 1 string of 1 from "k" to list of 2 booleans at get["k"] at length .`, want: "[ [ 0, 0 ], [ 0, 0 ], [ 0, 0 ] ]\n{ [ \"z\", \"z\" ] }\n2\n"},
		{name: "assembled values", src: `import random
assemble Order [item, qty, price]
o = random Order [string of 3 from "x", integer from 5 to 5, float from 1 to 1]
show o .
show length of random list of 4 Order [string, integer, float] .`, want: "Order { item: \"xxx\", qty: 5, price: 1.0 }\n4\n"},
		{name: "the same seed gives the same values", src: `import random
seed = 7
a = random list of 10 integers
b = random string of 20
seed = 8
c = random list of 10 integers
seed = 7
show a == random list of 10 integers, b == random string of 20, a == c .`, want: "truetruefalse\n"},
		{name: "seed inside a function is the file's", src: `import random
seed = 3
def roll[]
    return random integer
def [end]
first = roll[]
seed = 4
seed = 3
show roll[] == first .`, want: "true\n"},
		{name: "comparisons apply to the value", src: `import random
show random integer from 1 to 6 == 7 .
show random integer from 1 to 2 + 4 > 0 .`, want: "false\ntrue\n"},
		{name: "random values in a list literal", src: `import random
show length of list [random integer, random string, random boolean] .`, want: "3\n"},
		{name: "pick shuffle sample chance", src: `import random
show pick[list [9]], pick[set [8]], pick[map ["k": 1]], pick["q"] .
s = shuffle[list [1, 2, 3, 4]]
show length of s, " ", min_or[s] .`, wantErr: "undefined function \"min_or\""},
		{name: "shuffle keeps the items", src: `import random
import sort [min_sort]
show min_sort[shuffle[list [3, 1, 2, 3]]], shuffle["aaa"], length of sample[list [1, 2, 3], 2], chance[1], chance[0] .`, want: "[ 1, 2, 3, 3 ]aaa2truefalse\n"},
		{name: "random qualified call", src: `import random
show random pick[list [5]] .`, want: "5\n"},
		{name: "without the import, random is a name", src: `random = 4
show random .`, want: "4\n"},
		{name: "number at random still works", src: `import math
import random
n is 3 at random .
show n < 3 .`, want: "true\n"},
		{name: "from more than to", src: `import random
n = random integer from 6 to 1`, wantErr: "random integer from 6 to 1: from is more than to"},
		{name: "set too big for its range", src: `import random
s = random set of 5 integers from 1 to 3`, wantErr: "wanted 5 different values, but found only 3"},
		{name: "no such assembled type", src: `import random
o = random Thing [string]`, wantErr: "no assembled type called \"Thing\""},
		{name: "wrong number of field kinds", src: `import random
assemble P [a, b]
o = random P [string]`, wantErr: "P has 2 fields (a, b), so give 2 kinds, got 1"},
		{name: "a seed that isn't a number", src: `import random
seed = "x"
n = random integer`, wantErr: "seed must be a whole number"},
		{name: "chance out of range", src: `import random
b = chance[2]`, wantErr: "chance: needs a number from 0 to 1"},
		{name: "sample too many", src: `import random
s = sample[list [1], 2]`, wantErr: "can't take 2 different items from 1"},
		{name: "pick from nothing", src: `import random
p = pick[list []]`, wantErr: "there's nothing to pick"},
		{name: "a bad date", src: `import random
d = random date from "soon" to "later"`, wantErr: "needs a date"},
		{name: "integers need whole limits", src: `import random
n = random integer from 1.5 to 3`, wantErr: "needs a whole number"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			out, err := run(t, c.src, "")
			if c.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), c.wantErr) {
					t.Fatalf("want error containing %q, got %v (output %q)", c.wantErr, err, out)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v (output %q)", err, out)
			}
			if out != c.want {
				t.Errorf("got %q, want %q", out, c.want)
			}
		})
	}
}

func TestRandomParseErrors(t *testing.T) {
	cases := map[string]string{
		"import random\nm = random map of string":    "say what the keys map to",
		"import random\nf = random float rounded 2":  "", // "rounded" without a range or "to" is just a name after the float
		"import random\no = random Order []":         "give a kind for each field",
		"import random\nf = random integer from 1 3": "give the range as from A to B",
	}
	for src, want := range cases {
		p := parser.New(lexer.New(src))
		p.ParseProgram()
		errs := strings.Join(p.Errors(), "\n")
		if want == "" {
			continue
		}
		if !strings.Contains(errs, want) {
			t.Errorf("%q: want a parse error containing %q, got %q", src, want, errs)
		}
	}
}

// TestRandomLibraryInTurtle: pick, shuffle, sample and chance are written
// in Turtle (lib/random.trt, built in). The caller's seed decides their
// values, their helpers stay private, and errors point at the caller.
func TestRandomLibraryInTurtle(t *testing.T) {
	cases := []struct{ name, src, want, wantErr string }{
		{name: "the caller's seed", src: `import random
seed = 5
a = shuffle[list [1, 2, 3, 4, 5, 6, 7, 8]]
b = pick["abcdefgh"]
c = sample[list [1, 2, 3, 4, 5, 6], 3]
seed = 5
show a == shuffle[list [1, 2, 3, 4, 5, 6, 7, 8]], b == pick["abcdefgh"], c == sample[list [1, 2, 3, 4, 5, 6], 3] .`, want: "truetruetrue\n"},
		{name: "a string shuffles into a string", src: `import random
import sort [min_sort]
s = shuffle["banana"]
show s at length, " ", min_sort[change s to list] == min_sort[change "banana" to list] .`, want: "6 true\n"},
		{name: "a map's keys", src: `import random
show pick[map ["only": 1]], " ", shuffle[map ["k": 1]] .`, want: "only [ \"k\" ]\n"},
		{name: "the original is left as it was", src: `import random
nums = list [1, 2, 3]
s = shuffle[nums]
x = sample[nums, 2]
show nums .`, want: "[ 1, 2, 3 ]\n"},
		{name: "helpers are private", src: `import random
x = items_of[list [1], "x"]`, wantErr: `undefined function "items_of"`},
		{name: "errors point at the caller's line", src: `import random
n = 1
x = shuffle[n]`, wantErr: "line 3: shuffle needs a list, set, map or string, got 1"},
		{name: "importing some", src: `import random [pick]
show pick[list [4]] .
x = shuffle[list [1]]`, wantErr: `"shuffle" isn't imported`},
		{name: "qualified", src: `import random
show random pick[list [4]] .`, want: "4\n"},
		{name: "two files share one library", src: `import random
import sort [is_sorted]
seed = 1
show is_sorted[shuffle[list [1, 1, 1]]] .`, want: "true\n"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			out, err := run(t, c.src, "")
			if c.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), c.wantErr) {
					t.Fatalf("want error containing %q, got %v (output %q)", c.wantErr, err, out)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v (output %q)", err, out)
			}
			if out != c.want {
				t.Errorf("got %q, want %q", out, c.want)
			}
		})
	}
}
