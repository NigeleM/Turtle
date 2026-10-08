package evaluator

import (
	"strings"
	"testing"
)

// Raw strings, the pattern library, number and text formatting, data's
// range, reduce and sum, typeof and type, and calling a function before its def.

func TestRawStringsAndPatterns(t *testing.T) {
	cases := []struct{ src, want string }{
		{"show `\\d{3}-x` .", "\\d{3}-x\n"},
		{"n = 5\nshow `{n} stays` .", "{n} stays\n"},
		{"show `C:\\new\\table` .", "C:\\new\\table\n"},
		{"show `a\nb` .", "a\nb\n"},
		{"import pattern\nshow matches[\"ab-1234\", `^[a-z]{2}-\\d{4}$`] .", "true\n"},
		{"import pattern\nshow matches[\"ab\", `\\d`] .", "false\n"},
		{"import pattern\nshow findall[\"a1 b22\", `\\d+`] .", "[ \"1\", \"22\" ]\n"},
		{"import pattern\nshow findall[\"ab\", `\\d+`] .", "[  ]\n"},
		{"import pattern\nshow replaceall[\"a1 b2\", `\\d`, \"#\"] .", "a# b#\n"},
		{"import pattern\nshow replaceall[\"2026-10-06\", `(\\d+)-(\\d+)-(\\d+)`, `$3/$2/$1`] .", "06/10/2026\n"},
		{"import pattern\nshow splitby[\"a, b;c\", `[,;]\\s*`] .", "[ \"a\", \"b\", \"c\" ]\n"},
		{"import pattern\nshow groups[\"2026-10-06\", `(\\d+)-(\\d+)`] .", "[ \"2026\", \"10\" ]\n"},
		{"import pattern\nshow groups[\"x\", `(\\d+)`] .", "none\n"},
	}
	for _, c := range cases {
		got, err := run(t, c.src, "")
		if err != nil {
			t.Errorf("%q: %v", c.src, err)
		} else if got != c.want {
			t.Errorf("%q: got %q, want %q", c.src, got, c.want)
		}
	}
}

func TestBadPatternIsKindPattern(t *testing.T) {
	got, err := run(t, "import pattern\nsafe\n    x = matches[\"a\", `(`]\nhandle [pattern] e .\n    show kind of e .\nsafe [end]", "")
	if err != nil {
		t.Fatal(err)
	}
	if got != "pattern\n" {
		t.Errorf("got %q, want the kind pattern", got)
	}
}

func TestNumberAndTextFormatting(t *testing.T) {
	cases := []struct{ src, want string }{
		{"x = 3.5\nshow x at fixed[2] .", "3.50\n"},
		{"x = 2\nshow x at fixed[1] .", "2.0\n"},
		{"x = 1234567\nshow x at commas .", "1,234,567\n"},
		{"x = -1234.5\nshow x at commas .", "-1,234.5\n"},
		{"x = 999\nshow x at commas .", "999\n"},
		{"s = \"7\"\nshow s at padleft[3, \"0\"] .", "007\n"},
		{"s = \"ab\"\nshow s at padright[5, \".\"], \"|\" .", "ab...|\n"},
		{"s = \"abc\"\nshow s at padleft[2], \"|\" .", "abc|\n"},
		{"s = \"ab\"\nshow s at padleft[4], \"|\" .", "  ab|\n"},
	}
	for _, c := range cases {
		got, err := run(t, c.src, "")
		if err != nil {
			t.Errorf("%q: %v", c.src, err)
		} else if got != c.want {
			t.Errorf("%q: got %q, want %q", c.src, got, c.want)
		}
	}
}

func TestRangeReduceSum(t *testing.T) {
	cases := []struct{ src, want string }{
		{"import data\nshow range[5] .", "[ 0, 1, 2, 3, 4 ]\n"},
		{"import data\nshow range[1, 5] .", "[ 1, 2, 3, 4 ]\n"},
		{"import data\nr = 1 range 5\nshow r .", "[ 1, 2, 3, 4 ]\n"},
		{"import data\nshow range[0, 10, 2] .", "[ 0, 2, 4, 6, 8 ]\n"},
		{"import data\nshow range[0, 9, 3] .", "[ 0, 3, 6 ]\n"},
		{"import data\nshow range[5, 0, -1] .", "[ 5, 4, 3, 2, 1 ]\n"},
		{"import data\nshow range[10, 0, -3] .", "[ 10, 7, 4, 1 ]\n"},
		{"import data\nshow range[5, 1], range[0], range[3, 3] .", "[  ][  ][  ]\n"},
		{"import data\nshow range[-2, 2] .", "[ -2, -1, 0, 1 ]\n"},
		{"import data\n[loop][i in 1 range 4]\n    show i .\n[loop][end]", "1\n2\n3\n"},
		{"import data\nnums = list [1, 2, 3]\nshow reduce[nums, 0, [t, x] give t + x] .", "6\n"},
		{"import data\nnums = list [1, 2, 3]\ntotal = nums reduce 0, [t, x] give t + x\nshow total .", "6\n"},
		{"import data\nc = list [\"t\", \"u\"]\nw = c reduce \"\", [w, x] give w + x\nshow w .", "tu\n"},
		{"import data\nnums = list [1, 2, 3]\nshow sum[nums] .", "6\n"},
		{"import data\nnums = list [1, 2.5]\ns = nums sum\nshow s .", "3.5\n"},
		{"import data\nnums = list []\nshow sum[nums] .", "0\n"},
	}
	for _, c := range cases {
		got, err := run(t, c.src, "")
		if err != nil {
			t.Errorf("%q: %v", c.src, err)
		} else if got != c.want {
			t.Errorf("%q: got %q, want %q", c.src, got, c.want)
		}
	}
	for _, src := range []string{
		"import data\nnums = list [1, \"a\"]\nshow sum[nums] .",
		"import data\nshow range[1, 5, 0] .",
		"import data\nnums = list [1]\nshow reduce[nums, 0, [x] give x] .",
	} {
		if _, err := run(t, src, ""); err == nil {
			t.Errorf("%q: want an error", src)
		}
	}
}

func TestTypeofAndType(t *testing.T) {
	src := `assemble order [item]
o = order ["tea"]
n = 3
s = "a"
l = list [1]
show typeof[n], " ", typeof[l], " ", typeof[s], " ", typeof[o] .
show o type order, " ", n type integer, " ", s type integer, " ", l type list .
if ] n type integer && s type string [
    show "both" .
if [end]
type = 5
show type .`
	got, err := run(t, src, "")
	if err != nil {
		t.Fatal(err)
	}
	want := "integer list string order\ntrue true false true\nboth\n5\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	if _, err := run(t, "n = 3\nshow n type ordr .", ""); err == nil || !strings.Contains(err.Error(), "isn't a kind") {
		t.Errorf("unknown kind: got %v", err)
	}
}

func TestCallBeforeDef(t *testing.T) {
	cases := []struct{ src, want string }{
		// Main code first, functions below.
		{"show double[2] .\ndef double[x]\n    return x * 2\ndef [end]", "4\n"},
		// Functions calling each other in any order.
		{"show a[1] .\ndef a[x]\n    return b[x] + 1\ndef [end]\ndef b[x]\n    return x * 10\ndef [end]", "11\n"},
		// A sentence call before the def.
		{"n = 3\nshow n twice .\ndef twice[x]\n    return x * 2\ndef [end]", "6\n"},
		// Def'd twice: the first until the second's line runs.
		{"show f[] .\ndef f[]\n    return 1\ndef [end]\nshow f[] .\ndef f[]\n    return 2\ndef [end]\nshow f[] .", "1\n1\n2\n"},
	}
	for _, c := range cases {
		got, err := run(t, c.src, "")
		if err != nil {
			t.Errorf("%q: %v", c.src, err)
		} else if got != c.want {
			t.Errorf("%q: got %q, want %q", c.src, got, c.want)
		}
	}
	// Variables still run in order: a function called early can't read a
	// global that isn't set yet.
	if _, err := run(t, "show f[] .\nlimit = 5\ndef f[]\n    return limit\ndef [end]", ""); err == nil || !strings.Contains(err.Error(), "limit") {
		t.Errorf("global before it's set: got %v", err)
	}
	// Only functions: an assembled type is made at its line.
	if _, err := run(t, "o = order [1]\nassemble order [x]", ""); err == nil {
		t.Errorf("assembled type before its line: want an error")
	}
	// A def inside a function is a local, made when its line runs.
	if _, err := run(t, "def outer[]\n    show inner[] .\n    def inner[]\n        return 1\n    def [end]\ndef [end]\nouter[]", ""); err == nil {
		t.Errorf("nested def before its line: want an error")
	}
}

func TestRoundToPlaces(t *testing.T) {
	src := `import data
import math
b is list [1, 2, 3] process x give x * 0.2 .
c is b process x give x at round[2] .
show c, " ", c at get[2] == 0.6 .
show 3.14159 at round[2], " ", 3.14159 at round, " ", 1234 at round[-2], " ", 1250.0 at round[-2], " ", 7 at round[2], " ", -2.5 at round[0] .
show typeof[3.14159 at round[2]], " ", typeof[1234 at round[-2]] .`
	got, err := run(t, src, "")
	if err != nil {
		t.Fatal(err)
	}
	want := "[ 0.2, 0.4, 0.6 ] true\n3.14 3 1200 1300 7 -3\nfloat integer\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	for src, msg := range map[string]string{
		"import math\nx = 2.5 at round[2, 3]": "'round' takes nothing, or how many places",
		"import math\nx = 2.5 at round[99]":   "from -15 to 15",
		"import math\nx = 2.5 at round[1.5]":  "from -15 to 15",
		"x = fixed[2]":                        `method "fixed" expects 1 argument(s), got 0`,
		"x = trim[]":                          "trim needs a value to work on: trim[x], x trim, or x at trim",
	} {
		if _, err := run(t, src, ""); err == nil || !strings.Contains(err.Error(), msg) {
			t.Errorf("%q: got %v, want %q", src, err, msg)
		}
	}
}

func TestFloatsShowFifteenDigits(t *testing.T) {
	src := `import json
x = 0.1 + 0.2
show x, " ", x == 0.3, " ", 1/3.0, " ", 2.5, " ", 7.0, " ", 0.000001234 .
show "{x}", " ", list [0.6000000000000001] .
show json_text[list [x]] .`
	got, err := run(t, src, "")
	if err != nil {
		t.Fatal(err)
	}
	want := "0.3 true 0.333333333333333 2.5 7.0 1.234e-06\n0.3 [ 0.6 ]\n[0.30000000000000004]\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// Floats compare as they show: ==, <, sets, map keys, searches and check
// all agree, and integers stay exact.
func TestFloatsCompareAsTheyShow(t *testing.T) {
	src := `import test
import data
import math
x = 0.1 + 0.2
show x == 0.3, x != 0.3, x > 0.3, x >= 0.3, x < 0.3, x <= 0.3 .
show 1.1 * 3 == 3.3, 0.3 < 0.30000000000001, 2.0000000000000004 == 2 .
s = set [0.3, x, 0.6000000000000001, 0.6]
show length of s .
m = map [0.3: "a"]
show m at get[x] .
nums = list [0.2, 0.4, 0.6000000000000001]
show nums at contains[0.6], nums at index[0.6] .
b is list [1, 2, 3] process v give v * 0.2 .
show b == list [0.2, 0.4, 0.6] .
check 0.1 + 0.2 == 0.3 .
check x at round[2] == 0.3 .
show 9007199254740993 == 9007199254740992 .`
	got, err := run(t, src, "")
	if err != nil {
		t.Fatal(err)
	}
	want := "truefalsefalsetruefalsetrue\ntruetruetrue\n2\na\ntrue2\ntrue\nfalse\n"
	if got != want {
		t.Errorf("got %q\nwant %q", got, want)
	}
}
