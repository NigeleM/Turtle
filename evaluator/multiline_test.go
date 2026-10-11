// Copyright 2017-2026 Nigele McCoy
// Licensed under the Apache License, Version 2.0;
// see LICENSE and NOTICE.

package evaluator

import "testing"

// Anything in brackets may go over several lines, with a comma before the
// closing ']' allowed; in an if-header a '[' that ends its line still
// opens the body.
func TestBracketsOverLines(t *testing.T) {
	cases := []struct{ name, src, want string }{
		{"list", "x = list [\n    1,\n    2\n]\nshow x .", "[ 1, 2 ]\n"},
		{"list, trailing comma", "x = list [\n    1,\n    2,\n]\nshow x .", "[ 1, 2 ]\n"},
		{"set, trailing comma on one line", "x = set [1, 2,]\nshow x .", "{ 1, 2 }\n"},
		{"map, trailing comma", "m = map [\n    \"a\": 1,\n    \"b\": 2,\n]\nshow m .", "{ \"a\": 1, \"b\": 2 }\n"},
		{"nested, comments and blank lines", "x = list [\n\n    map [\"a\": list [\n        1, // one\n        2,\n    ]],\n    // a note\n    list [3],\n]\nshow x .", "[ { \"a\": [ 1, 2 ] }, [ 3 ] ]\n"},
		{"call", "def f[a, b, c]\n    return a + b + c\ndef [end]\ny = f[\n    1,\n    2,\n    3,\n]\nshow y .", "6\n"},
		{"call, options", "def f[o]\n    return o at get[\"b\"]\ndef [end]\nshow f[\n    \"a\": 1,\n    \"b\": 2,\n] .", "2\n"},
		{"call inside a list", "def f[a]\n    return a * 2\ndef [end]\nx = list [\n    f[1],\n    f[\n        2],\n]\nshow x .", "[ 2, 4 ]\n"},
		{"assembled value", "assemble Order [item, qty]\no = Order[\n    \"pen\",\n    3,\n]\nshow o .", "Order { item: \"pen\", qty: 3 }\n"},
		{"assembled fields", "assemble Order [\n    item,\n    qty\n]\nshow Order[\"pen\", 3] .", "Order { item: \"pen\", qty: 3 }\n"},
		{"function parameters", "def f[\n    a,\n    b\n]\n    return a + b\ndef [end]\nshow f[1, 2] .", "3\n"},
		{"library call", "import data\ny = range[\n    0, 3,\n]\nshow y .", "[ 0, 1, 2 ]\n"},
		{"method", "x = list [3, 1]\ny = x at get[\n    0\n]\nshow y .", "3\n"},
		{"method in an is statement", "m = map []\nm is m at add[\n    \"a\", 1\n] .\nshow m .", "{ \"a\": 1 }\n"},
		{"import names, trailing comma", "import time [\n    now,\n    today,\n]\nshow today[] != none .", "true\n"},
		{"if ] name [", "ready = true\nif ] ready [\n    show \"go\" .\nif [end]", "go\n"},
		{"if ] x at isempty [", "x = list []\nif ] x at isempty [\n    show \"empty\" .\nif [end]", "empty\n"},
		{"if with a call in its header", "def ok[a]\n    return a > 0\ndef [end]\nif ] ok[1] [\n    show \"yes\" .\nif [end]", "yes\n"},
		{"else if ] name [", "a = false\nb = true\nif ] a [\n    show \"a\" .\nelse if ] b [\n    show \"b\" .\nif [end]", "b\n"},
		{"nested if ] name [", "a = true\nb = true\nif ] a [\n    [if ] b [\n        show \"both\" .\nif [end]", "both\n"},
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
