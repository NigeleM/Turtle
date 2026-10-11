// Copyright 2017-2026 Nigele McCoy
// Licensed under the Apache License, Version 2.0;
// see LICENSE and NOTICE.

package repl

import (
	"strings"

	"Turtle/syntax"
)

// Terminal colors for Turtle code (the coloring itself is package
// syntax), grouped by what a word does, in the scheme chosen with the
// colors command (colors.go).

const (
	reset       = "\x1b[0m"
	promptColor = "\x1b[38;5;244m" // gray
	errorColor  = "\x1b[38;5;196m" // red
)

// colorize writes src with its colors as terminal codes.
func colorize(src string, w syntax.Words, codes map[syntax.Class]string) string {
	var b strings.Builder
	at := 0
	for _, s := range syntax.Highlight(src, w) {
		if s.Start < at || s.End > len(src) {
			continue
		}
		b.WriteString(src[at:s.Start])
		b.WriteString(codes[s.Class])
		b.WriteString(src[s.Start:s.End])
		b.WriteString(reset)
		at = s.End
	}
	b.WriteString(src[at:])
	return b.String()
}
