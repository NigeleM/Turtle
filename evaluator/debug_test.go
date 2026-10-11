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

// debugSession runs src under turtle debug with the commands typed, and
// gives back what the debugger wrote.
func debugSession(t *testing.T, src, commands string) (string, error) {
	t.Helper()
	program := parser.New(lexer.New(src)).ParseProgram()
	var out strings.Builder
	it := NewWithStdin(".", strings.NewReader(commands))
	it.Debug(&out, src)
	err := it.Run(program)
	return out.String(), err
}

const debugProgram = `nums = list [5, 7]
total = 0
[loop][x in nums]
    total = total + x
[loop][end]
d = double[total]

def double[n]
    r = n * 2
    return r
def [end]`

func TestDebugStepsAndBreakpoints(t *testing.T) {
	got, err := debugSession(t, debugProgram, "\nb 9\nc\nw\nv\np n * 10\nn = 100\nn\nr\no\n")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"→ line 1  nums = list [5, 7]",
		"→ line 2  total = 0",
		"will stop at 9",
		"→ line 9      r = n * 2",
		"the main code, line 6\n└ double[], line 9",
		"here:\n  n = 12\nglobals:\n  nums = [ 5, 7 ]\n  total = 12",
		"120",
		"→ line 10      return r",
		"200", // n = 100 changed n before r = n * 2 ran
		"(debug) 200\n(debug) ",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
}

func TestDebugQuit(t *testing.T) {
	got, err := debugSession(t, debugProgram, "q\n")
	if _, quit := err.(ExitRequest); !quit || strings.Contains(got, "line 2") {
		t.Errorf("q should end the program at once: %v\n%s", err, got)
	}
}

func TestDebugNextStepsOverCalls(t *testing.T) {
	got, _ := debugSession(t, debugProgram, "n\nn\nn\nn\nn\nn\nn\n")
	if strings.Contains(got, "r = n * 2") {
		t.Errorf("n went into the function:\n%s", got)
	}
	if !strings.Contains(got, "→ line 6  d = double[total]") {
		t.Errorf("n didn't stop at line 6:\n%s", got)
	}
}

func TestDebugErrorsAndEndOfInput(t *testing.T) {
	got, err := debugSession(t, debugProgram, "nope\np 1 / 0\nb x\n")
	if err != nil {
		t.Fatal(err) // the input ended: it runs to the end
	}
	for _, want := range []string{
		`error (name): undefined variable "nope"`,
		"error (math): division by zero",
		`"x" isn't a line`,
		"no more input: running to the end",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
}
