package evaluator

import (
	"strings"
	"testing"

	"Turtle/token"
)

func TestHelp(t *testing.T) {
	cases := []struct{ src, want string }{
		{`show version, version[] .`, "devdev"},
		{`help[]`, "help[\"linear\"]     a library"},
		{`help[]`, "  linear    Matrices and vectors."},
		{`help["linear"]`, "  reshape[m, rows, columns]\n      The same numbers, in the same order, as a rows x columns matrix."},
		{`help["sql_load"]`, "sql_load[db, table, path"},
		{`help["upper"]`, "string at upper        (a string method"},
		{`help["get"]`, "list at get[index]"},
		{`help["string"]`, "A string's methods, as x at upper:\n  upper, lower, trim,"},
		{`help["nothing_here"]`, `no library, function, method or keyword called "nothing_here"`},
		{`help["sql_lod"]`, "did you mean sql_load?"},
		{"import linear\nhelp[matrix [1, 2]]", "This is a matrix, 1 x 2.\nA matrix's methods, as x at rows:"},
		{`help[list [1]]`, "This is a list of 1 item."},
		{`help[map ["a": 1, "b": 2]]`, "This is a map of 2 entries."},
		{`help[2]`, "This is an integer.\nAn integer's methods"},
		{`help[none]`, "This is none, the value for nothing."},
		{"assemble Book [title, pages]\nhelp[Book[\"Dune\", 412]]", "This is an assembled Book.\nIts fields: title, pages. Read one: title of x"},
		{"assemble Book [title, pages]\nhelp[Book]", "Make one: Book[title, pages]."},
		{"def area[w, h]\n    return w * h\ndef [end]\nhelp[area]", "This is a function.\nCall it: area[w, h]"},
		{"import time\nhelp[today[]]", "This is a date.\nLibraries for it: help[\"time\"]"},
		// Your own names come first.
		{"def help[x]\n    show \"mine\" .\ndef [end]\nhelp[1]", "mine"},
		{"version = \"1.2\"\nshow version .", "1.2"},
		{"def stdlib[]\n    return 7\ndef [end]\nshow stdlib[] .", "7"},
	}
	for _, c := range cases {
		got, err := run(t, c.src, "")
		if err != nil {
			t.Errorf("%s: %v", c.src, err)
		} else if !strings.Contains(got, c.want) {
			t.Errorf("%s:\n got  %q\n want %q", c.src, got, c.want)
		}
	}
	for _, src := range []string{`help[1, 2]`, `stdlib[1]`, `version[1]`} {
		if _, err := run(t, src, ""); err == nil {
			t.Errorf("%s: no error", src)
		}
	}
}

// Every method help lists is one the value really has.
func TestHelpMethodsAreReal(t *testing.T) {
	samples := map[string]string{
		"list": "list [1]", "set": "set [1]", "map": `map ["a": 1]`, "string": `"a"`,
		"integer": "1", "float": "1.5", "matrix": "matrix [1]",
	}
	for kind, methods := range typeMethods {
		v, ok := samples[kind]
		if !ok {
			t.Errorf("no sample %s", kind)
			continue
		}
		for _, m := range methods {
			name := methodName(m)
			src := "import linear\nimport math\nv = " + v + "\nsafe\n    x = v at " + name + "\nhandle [] e .\n    show message of e .\nsafe [end]"
			got, _ := run(t, src, "")
			if strings.Contains(got, "unknown") || strings.Contains(got, "has no method") || strings.Contains(got, "needs a list, set") {
				t.Errorf("%s at %s: %s", kind, name, got)
			}
		}
	}
}

func TestHelpKeywords(t *testing.T) {
	cases := []struct{ src, want string }{
		{`help["if"]`, "if        (a keyword)\n  Runs code when a condition holds.\n  Example: if ] x > 1 [ ... if [end]\n  More: Conditionals (docs/reference.md#conditionals)"},
		{`help["matrix"]`, "matrix        (a keyword after import linear)"},
		{`help["matrix"]`, "A matrix's methods"},
		{`help["integer"]`, "integer        (a word, special only in one place)"},
		{`help["sort"]`, "sort        (a keyword)"},
		{`help["sort"]`, "import sort"},
		{`help["typeof"]`, "typeof        (no import needed)"},
		{`help["version"]`, "version (or version[])"},
		{`help["keywords"]`, "  loop       repeats code"},
		{`help["keywords"]`, "  integer, float, string, ascii, char, hex, keys, values\n             after change ... to"},
		{`help["iff"]`, "did you mean if?"},
	}
	for _, c := range cases {
		got, err := run(t, c.src, "")
		if err != nil {
			t.Errorf("%s: %v", c.src, err)
		} else if !strings.Contains(got, c.want) {
			t.Errorf("%s:\n got  %q\n want %q", c.src, got, c.want)
		}
	}
}

// Every reserved word has help, read from docs/reference.md's tables.
func TestEveryKeywordHasHelp(t *testing.T) {
	for _, w := range token.Keywords() {
		if keywordHelp(w) == "" {
			t.Errorf("no help for %q: add it to the Keywords tables in docs/reference.md", w)
		}
	}
}
