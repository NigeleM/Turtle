// Copyright 2017-2026 Nigele McCoy
// Licensed under the Apache License, Version 2.0;
// see LICENSE and NOTICE.

package syntax

import (
	"os"
	"strings"
	"testing"

	"Turtle/token"
)

func noWords() Words {
	return Words{Context: map[string]bool{}, Builtin: map[string]bool{"sql_open": true}}
}

// colored names each colored piece of src: "def:keyword double:definition ...".
func colored(src string, w Words) string {
	names := map[Class]string{Keyword: "keyword", Constant: "constant", String: "string", Comment: "comment", Definition: "definition", Call: "call", Builtin: "builtin",
		Import: "import", Show: "show", Data: "data", DataDefinition: "datadef", Method: "method", Theory: "theory", TestDefinition: "testdef"}
	var parts []string
	for _, s := range Highlight(src, w) {
		parts = append(parts, src[s.Start:s.End]+":"+names[s.Class])
	}
	return strings.Join(parts, " ")
}

func TestHighlight(t *testing.T) {
	cases := map[string]string{
		`def double[x]`:                    "def:keyword double:definition",
		`show "hi", 42, 2.5, true, none .`: `show:show "hi":string 42:constant 2.5:constant true:constant none:constant`,
		`x = 1 // note`:                    "1:constant // note:comment",
		`//* block *// y = 2`:              "//* block *//:comment 2:constant",
		`total = add_tax[5]`:               "add_tax:call 5:constant",
		`db = sql_open["a.db"]`:            `sql_open:builtin "a.db":string`,
		`if ] x > 1 [`:                     "if:keyword 1:constant",
		`assemble Order [item]`:            "assemble:data Order:datadef",
		`nums at get[0]`:                   "at:keyword get:method 0:constant",
		`sys ls -la`:                       "sys:keyword",
		`7 div 2`:                          "7:constant div:keyword 2:constant",
		`s = "unfinished`:                  `"unfinished:string`,
		`show "a // not a comment" .`:      `show:show "a // not a comment":string`,
		`warn "careful" .`:                 `warn:show "careful":string`,
		`import time [now] // clock`:       "import:import time:import [:import now:import ]:import // clock:comment",
		`m = map ["a": list [1]]`:          `map:data "a":string list:data 1:constant`,
		`def test_total[]`:                 "def:keyword test_total:testdef",
	}
	for src, want := range cases {
		if got := colored(src, noWords()); got != want {
			t.Errorf("%s\n got  %s\n want %s", src, got, want)
		}
	}
	// A library function written as a sentence, after a value.
	w := noWords()
	w.Builtin["process"] = true
	sentences := map[string]string{
		"a process x give x + 1": "process:builtin give:keyword 1:constant",
		"b = a process x give x": "process:builtin give:keyword",
		"process = 2":            "2:constant",
		"show process .":         "show:show",
		"p = `\\d{3}`":           "`\\d{3}`:string",
	}
	for src, want := range sentences {
		if got := colored(src, w); got != want {
			t.Errorf("%s\n got  %s\n want %s", src, got, want)
		}
	}
	// A library's words color once it's imported.
	w = noWords()
	if got := colored(`check x is integer .`, w); got != "is:keyword" {
		t.Errorf("before import test: %s", got)
	}
	w.Context["check"] = true
	if got := colored(`check x is integer .`, w); got != "check:keyword is:keyword" {
		t.Errorf("after import test: %s", got)
	}
}

func TestFunctionDoc(t *testing.T) {
	src := `// above, line one
// above, line two
def a[x]
    return x
def [end]

def b[x]
    // inside, line one
    // inside, line two
    return x
def [end]

//*
  above as a block,
    over two lines
*//
def c[x]
    //* inside as a block *//
    return x
def [end]

// above too
def d[x]

    //* inside after a blank line,
        ending here *//
    // and a // line right after

    // not this one: after a blank line
    return x
def [end]

def e[x]
    x = 1 // not a description: code comes first
    // nor this
def [end]

x = 1
def f[x]
    return x
def [end]`
	lines := strings.Split(src, "\n")
	defAt := func(name string) int {
		for i, l := range lines {
			if strings.HasPrefix(l, "def "+name+"[") {
				return i
			}
		}
		t.Fatalf("no def %s", name)
		return -1
	}
	cases := map[string]string{
		"a": "above, line one\nabove, line two",
		"b": "inside, line one\ninside, line two",
		"c": "above as a block,\nover two lines\ninside as a block",
		"d": "above too\ninside after a blank line,\nending here\nand a // line right after",
		"e": "",
		"f": "",
	}
	for name, want := range cases {
		if got := FunctionDoc(lines, defAt(name)); got != want {
			t.Errorf("%s: got %q, want %q", name, got, want)
		}
	}
	if got := CommentAbove(lines, defAt("c")); got != "above as a block,\nover two lines" {
		t.Errorf("CommentAbove c: %q", got)
	}
}

// TestKeywordsAreDocumented: the reference's Keywords section names every
// reserved word and every word an import turns on.
func TestKeywordsAreDocumented(t *testing.T) {
	data, err := os.ReadFile("../docs/reference.md")
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	start := strings.Index(text, "## Keywords")
	end := strings.Index(text, "## Statement terminators")
	if start < 0 || end < start {
		t.Fatal("reference.md has no Keywords section before Statement terminators")
	}
	section := text[start:end]
	for _, k := range token.Keywords() {
		if !strings.Contains(section, "| `"+k+"` |") {
			t.Errorf("reserved word %q isn't in the Keywords table", k)
		}
	}
	for lib, words := range LibraryWords {
		for _, w := range words {
			if !strings.Contains(section, "`"+w+"`") {
				t.Errorf("%q (import %s) isn't in the Keywords section", w, lib)
			}
		}
	}
}

// A theory: its head, sections and [end] are theory-colored, its abstract
// is prose and stays plain, and its word is theory-colored where used.
func TestHighlightTheories(t *testing.T) {
	src := "theory tally\n    abstract\n        tally is how many b are in a.\n    notation tally b in a .\n    definition\n        return 0\ntheory [end]\nn = tally 1 in nums ."
	w := noWords()
	w.Theories = map[string]bool{"tally": true}
	want := "theory:theory tally:theory abstract:theory notation:theory tally:theory in:keyword definition:theory return:keyword 0:constant theory [end]:theory tally:theory 1:constant in:keyword"
	if got := colored(src, w); got != want {
		t.Errorf("\n got  %s\n want %s", got, want)
	}
	// Without the theory known (another file's, not yet read), its word is plain.
	if got := colored("n = tally 1 in nums .", noWords()); got != "1:constant in:keyword" {
		t.Errorf("unknown theory: %s", got)
	}
	// An assembled type where it's used.
	w.Types = map[string]bool{"Order": true}
	if got := colored(`o = Order["tea"]`, w); got != `Order:data "tea":string` {
		t.Errorf("type: %s", got)
	}
}
