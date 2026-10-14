// Copyright 2017-2026 Nigele McCoy
// Licensed under the Apache License, Version 2.0;
// see LICENSE and NOTICE.

package evaluator

import (
	"strings"
	"testing"
)

// import m as d: the module is called d in this file, and d f[...] calls
// its functions. import m [f as g]: the listed name is g here, and f isn't
// available under its own name.
func TestImportAliases(t *testing.T) {
	cases := []struct{ name, src, want string }{
		{"module alias", "import data as d\nshow d sum[list [1, 2, 3]] .\nshow d mean[list [2, 4]] .\n", "6\n3.0\n"},
		{"listed alias", "import data [mean as average, sum]\nshow average[list [2, 4]] .\nshow sum[list [1, 2]] .\n", "3.0\n3\n"},
		{"theory uses the freed name", "import data [mean as average]\ntheory mean\n    abstract\n        mean is a test.\n    notation mean a .\n    definition\n        return a\ntheory [end]\nshow mean list [1] .\n", "[ 1 ]\n"},
		{"original name refused", "import data [mean as average]\nshow mean[list [2, 4]] .\n", "is imported as \"average\" — use that name"},
		{"same module, two names", "import data as d\nimport data [sum]\n", "already imports data as \"d\""},
		{"two modules, one name", "import data as d\nimport math as d\n", "already imports another module called \"d\""},
	}
	for _, c := range cases {
		got, err := run(t, c.src, "")
		text := got
		if err != nil {
			text = err.Error()
		}
		if !strings.Contains(text, c.want) && got != c.want {
			t.Errorf("%s: got %q (err %v), want %q", c.name, got, err, c.want)
		}
	}
}
