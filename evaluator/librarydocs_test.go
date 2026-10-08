package evaluator

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"Turtle/lexer"
	"Turtle/parser"
)

// The library pages in docs/library: every builtin library has one, every
// function of it appears there, and every example that starts with an
// import parses, so the docs can't fall behind the code unnoticed.
func TestLibraryPages(t *testing.T) {
	dir := filepath.Join("..", "docs", "library")
	index, err := os.ReadFile(filepath.Join(dir, "index.md"))
	if err != nil {
		t.Fatal(err)
	}
	for name, mod := range builtinModules {
		page, err := os.ReadFile(filepath.Join(dir, name+".md"))
		if err != nil {
			t.Errorf("%s has no page: %v", name, err)
			continue
		}
		if !strings.Contains(string(index), "("+name+".md)") {
			t.Errorf("index.md doesn't link %s.md", name)
		}
		for _, f := range mod.Funcs {
			if !strings.Contains(string(page), "`"+f) && !strings.Contains(string(page), f+"[") {
				t.Errorf("%s.md doesn't mention %s", name, f)
			}
		}
	}
	pages, _ := filepath.Glob(filepath.Join(dir, "*.md"))
	fence := regexp.MustCompile("(?s)```\\n(import .*?)```")
	examples := 0
	for _, p := range pages {
		text, _ := os.ReadFile(p)
		for _, m := range fence.FindAllStringSubmatch(strings.ReplaceAll(string(text), "\r\n", "\n"), -1) {
			if strings.Contains(m[1], "<") && strings.Contains(m[1], ">") && regexp.MustCompile(`<[a-z]+>`).MatchString(m[1]) {
				continue // a syntax template (import <name>), not an example
			}
			examples++
			ps := parser.New(lexer.New(m[1]))
			ps.ParseProgram()
			if errs := ps.Errors(); len(errs) > 0 {
				t.Errorf("%s: an example doesn't parse: %s\n%s", filepath.Base(p), errs[0], m[1])
			}
		}
	}
	if examples < 20 {
		t.Errorf("only %d examples found; is the pattern right?", examples)
	}
}
