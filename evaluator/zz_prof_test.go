package evaluator

import (
	"os"
	"testing"

	"Turtle/lexer"
	"Turtle/parser"
)

func TestProfScript(t *testing.T) {
	name := os.Getenv("PROF_SCRIPT")
	if name == "" {
		t.Skip()
	}
	src, _ := os.ReadFile(name)
	p := parser.New(lexer.New(string(src)))
	prog := p.ParseProgram()
	it := New(".")
	if err := it.Run(prog); err != nil {
		t.Fatal(err)
	}
}
