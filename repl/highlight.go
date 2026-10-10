package repl

import (
	"strings"

	"Turtle/syntax"
)

// Terminal colors for Turtle code (the coloring itself is package
// syntax), grouped by what a word does, as in the VS Code extension's
// Turtle scheme: keywords green, imports teal, functions blue (bold where
// defined), data structures amber, show violet, text red, numbers pink,
// theories gold, bold and italic.

// ---- colors ----

// The colors, from the 256-color palette most terminals have, chosen to
// read on dark and light backgrounds alike.
var colorCodes = map[syntax.Class]string{
	syntax.Keyword:        "\x1b[38;5;71m",      // green
	syntax.Import:         "\x1b[38;5;37m",      // teal
	syntax.Definition:     "\x1b[1;38;5;33m",    // bold blue
	syntax.TestDefinition: "\x1b[1;3;38;5;33m",  // bold italic blue
	syntax.Call:           "\x1b[38;5;33m",      // blue
	syntax.Builtin:        "\x1b[38;5;33m",      // blue
	syntax.Method:         "\x1b[38;5;33m",      // blue
	syntax.Data:           "\x1b[38;5;172m",     // amber
	syntax.DataDefinition: "\x1b[1;38;5;172m",   // bold amber
	syntax.Show:           "\x1b[1;38;5;135m",   // bold violet
	syntax.String:         "\x1b[38;5;167m",     // red
	syntax.Constant:       "\x1b[38;5;169m",     // pink
	syntax.Theory:         "\x1b[1;3;38;5;178m", // bold italic gold
	syntax.Comment:        "\x1b[3;38;5;244m",   // italic gray
}

const (
	reset       = "\x1b[0m"
	promptColor = "\x1b[38;5;244m" // gray
	errorColor  = "\x1b[38;5;196m" // red
	resultColor = "\x1b[38;5;33m"  // blue
)

// colorize writes src with its colors as terminal codes.
func colorize(src string, w syntax.Words) string {
	var b strings.Builder
	at := 0
	for _, s := range syntax.Highlight(src, w) {
		if s.Start < at || s.End > len(src) {
			continue
		}
		b.WriteString(src[at:s.Start])
		b.WriteString(colorCodes[s.Class])
		b.WriteString(src[s.Start:s.End])
		b.WriteString(reset)
		at = s.End
	}
	b.WriteString(src[at:])
	return b.String()
}
