package syntax

import (
	"strings"
	"testing"
)

func noWords() Words {
	return Words{Context: map[string]bool{}, Builtin: map[string]bool{"sql_open": true}}
}

// colored names each colored piece of src: "def:keyword double:definition ...".
func colored(src string, w Words) string {
	names := map[Class]string{Keyword: "keyword", Constant: "constant", String: "string", Comment: "comment", Definition: "definition", Call: "call", Builtin: "builtin"}
	var parts []string
	for _, s := range Highlight(src, w) {
		parts = append(parts, src[s.Start:s.End]+":"+names[s.Class])
	}
	return strings.Join(parts, " ")
}

func TestHighlight(t *testing.T) {
	cases := map[string]string{
		`def double[x]`:                    "def:keyword double:definition",
		`show "hi", 42, 2.5, true, none .`: `show:keyword "hi":string 42:constant 2.5:constant true:constant none:constant`,
		`x = 1 // note`:                    "1:constant // note:comment",
		`//* block *// y = 2`:              "//* block *//:comment 2:constant",
		`total = add_tax[5]`:               "add_tax:call 5:constant",
		`db = sql_open["a.db"]`:            `sql_open:builtin "a.db":string`,
		`if ] x > 1 [`:                     "if:keyword 1:constant",
		`assemble Order [item]`:            "assemble:keyword Order:definition",
		`nums at get[0]`:                   "at:keyword get:call 0:constant",
		`sys ls -la`:                       "sys:keyword",
		`7 div 2`:                          "7:constant div:keyword 2:constant",
		`s = "unfinished`:                  `"unfinished:string`,
		`show "a // not a comment" .`:      `show:keyword "a // not a comment":string`,
	}
	for src, want := range cases {
		if got := colored(src, noWords()); got != want {
			t.Errorf("%s\n got  %s\n want %s", src, got, want)
		}
	}
	// A library's words color once it's imported.
	w := noWords()
	if got := colored(`check x is integer .`, w); got != "is:keyword" {
		t.Errorf("before import test: %s", got)
	}
	w.Context["check"] = true
	if got := colored(`check x is integer .`, w); got != "check:keyword is:keyword" {
		t.Errorf("after import test: %s", got)
	}
}
