package repl

import (
	"strings"

	"Turtle/syntax"
)

// Terminal colors for Turtle code (the coloring itself is package
// syntax). The scheme is IDLE's: keywords orange, strings green, comments
// red, definitions blue, library functions magenta.

// ---- colors ----

// The colors, from the 256-color palette most terminals have, chosen to
// read on dark and light backgrounds alike.
var colorCodes = map[syntax.Class]string{
	syntax.Keyword:    "\x1b[38;5;208m",  // orange
	syntax.Constant:   "\x1b[38;5;135m",  // purple
	syntax.String:     "\x1b[38;5;34m",   // green
	syntax.Comment:    "\x1b[38;5;167m",  // red
	syntax.Definition: "\x1b[1;38;5;33m", // bold blue
	syntax.Call:       "\x1b[38;5;33m",   // blue
	syntax.Builtin:    "\x1b[38;5;133m",  // magenta
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
