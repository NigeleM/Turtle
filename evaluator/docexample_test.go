// Copyright 2017-2026 Nigele McCoy
// Licensed under the Apache License, Version 2.0;
// see LICENSE and NOTICE.

package evaluator

import (
	"os"
	"strings"
	"testing"
)

// The documented examples are kept true by running them: this one is the
// generator from the Theories section of the reference.
func TestDocumentedGeneratorExample(t *testing.T) {
	doc, err := os.ReadFile("../docs/reference.md")
	if err != nil {
		t.Fatal(err)
	}
	text := string(doc)
	theories := strings.Index(text, "## Theories")
	if theories < 0 {
		t.Fatal("docs/reference.md has no Theories section")
	}
	start := strings.Index(text[theories:], "def make_counter[]")
	if start < 0 {
		t.Fatal("the generator example is missing from the Theories section")
	}
	start += theories
	end := strings.Index(text[start:], "\n```")
	if end < 0 {
		t.Fatal("the generator example in docs/reference.md has no closing fence")
	}
	src := text[start : start+end]
	got, err := run(t, src, "")
	if err != nil {
		t.Fatalf("the documented example fails: %v", err)
	}
	if want := "1\n2\n3\n"; got != want {
		t.Errorf("the documented example prints %q, the docs say %q", got, want)
	}
}
