// Copyright 2017-2026 Nigele McCoy
// Licensed under the Apache License, Version 2.0;
// see LICENSE and NOTICE.

package repl

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"Turtle/syntax"
)

// The colors picker: colors with nothing after it lists the schemes down
// the left and a short program on the right, colored in the scheme the
// cursor is on. Up and Down pick a scheme, Left and Right the shade
// (auto, dark, light), Enter keeps it, Esc (or q, or Ctrl-C) leaves it as
// it was. Without keys to read (input piped in), the same schemes come as
// a list, each in its own colors.

// pickerSample is the program the picker colors: every group in it.
var pickerSample = []string{
	"import time [today]",
	"assemble Order [item, qty]",
	"",
	"// the items, and how many",
	"def total[orders]",
	"    sum = 0",
	"    [loop][o in orders]",
	"        sum = sum + o at qty",
	"    [loop][end]",
	`    show "total: " . sum`,
	"    return tally orders",
	"def [end]",
}

var shadeNames = []string{"auto", "dark", "light"}

// sampleWords are the words the sample's colors need: its type and a
// theory's word.
func sampleWords() syntax.Words {
	return syntax.Words{Context: map[string]bool{}, Builtin: map[string]bool{},
		Theories: map[string]bool{"tally": true}, Types: map[string]bool{"Order": true}}
}

// pickerListWidth is the list's columns.
const pickerListWidth = 22

// pickerWidth is the columns the picker needs: the list, the bar, the
// longest sample line, and room for the keys line.
func pickerWidth() int {
	w := 0
	for _, l := range pickerSample {
		w = max(w, len(l))
	}
	return max(pickerListWidth+3+w, utf8.RuneCountInString(pickerKeys))
}

const pickerKeys = "  ↑↓ scheme   ←→ shade   Enter keeps it   Esc leaves it"

// pickerLines draws the picker: scheme sel, shade sh, the scheme in use
// marked "now"; detected is the shade auto finds.
func pickerLines(sel, sh int, inUse string, detected, truecolor bool) []string {
	light := shadeNames[sh] == "light" || shadeNames[sh] == "auto" && detected
	codes, _ := palette(schemes[sel], light, truecolor)
	w := sampleWords()
	shadeText := shadeNames[sh]
	if shadeNames[sh] == "auto" {
		shadeText = "auto (dark)"
		if detected {
			shadeText = "auto (light)"
		}
	}
	lines := []string{"  shade: ← " + shadeText + " →", ""}
	for i := 0; i < max(len(schemes), len(pickerSample)); i++ {
		left := ""
		if i < len(schemes) {
			key := schemeKey(schemes[i].name)
			if key == inUse {
				key += " (now)"
			}
			if i == sel {
				left = "\x1b[1m> " + key + reset + strings.Repeat(" ", pickerListWidth-2-len(key))
			} else {
				left = "  " + key + strings.Repeat(" ", pickerListWidth-2-len(key))
			}
		} else {
			left = strings.Repeat(" ", pickerListWidth)
		}
		right := ""
		if i < len(pickerSample) {
			right = colorize(pickerSample[i], w, codes)
		}
		lines = append(lines, left+promptColor+"│"+reset+"  "+right)
	}
	return append(lines, "", promptColor+pickerKeys+reset)
}

// pickColors runs the picker on the terminal; false if it can't (not a
// terminal, too narrow), for the list instead.
func (s *Session) pickColors() bool {
	if s.in == nil || !s.color || termWidth(1) < pickerWidth() {
		return false
	}
	state, err := makeRaw(0)
	if err != nil {
		return false
	}
	defer restore(0, state)

	sel := 0
	for i, sc := range schemes {
		if schemeKey(sc.name) == s.scheme {
			sel = i
		}
	}
	sh := 0
	for i, n := range shadeNames {
		if n == s.shade {
			sh = i
		}
	}
	detected := s.detectLight != nil && s.detectLight()
	fmt.Fprint(s.out, "\x1b[?25l") // no cursor while picking
	drawn := 0
	draw := func() {
		var b strings.Builder
		if drawn > 0 {
			fmt.Fprintf(&b, "\x1b[%dA", drawn)
		}
		b.WriteString("\r\x1b[J")
		lines := pickerLines(sel, sh, s.scheme, detected, s.truecolor)
		b.WriteString(strings.Join(lines, "\r\n"))
		drawn = len(lines) - 1
		fmt.Fprint(s.out, b.String())
	}
	done := func() {
		fmt.Fprintf(s.out, "\x1b[%dA\r\x1b[J\x1b[?25h", drawn) // the picker goes
	}
	draw()
	buf := make([]byte, 64)
	for {
		n, err := s.in.Read(buf)
		if err != nil {
			done()
			return true
		}
		for _, k := range decodeKeys(buf[:n]) {
			switch {
			case k.kind == keyUp:
				sel = (sel + len(schemes) - 1) % len(schemes)
			case k.kind == keyDown:
				sel = (sel + 1) % len(schemes)
			case k.kind == keyLeft:
				sh = (sh + len(shadeNames) - 1) % len(shadeNames)
			case k.kind == keyRight:
				sh = (sh + 1) % len(shadeNames)
			case k.kind == keyEnter:
				done()
				restore(0, state)
				s.chooseColors(schemeKey(schemes[sel].name), shadeNames[sh])
				return true
			case k.kind == keyEscape || k.kind == keyCancel || k.kind == keyRune && k.r == 'q':
				done()
				restore(0, state)
				fmt.Fprintln(s.out, "colors: still "+s.scheme+", "+s.shadeText())
				return true
			}
		}
		draw()
	}
}

// listColors is the picker without keys: each scheme in its own colors,
// one word for each group.
func (s *Session) listColors() {
	fmt.Fprintf(s.out, "%s, %s\n", s.scheme, s.shadeText())
	w := sampleWords()
	frags := []string{"if", "import time", "total[x]", "list", "show", `"text"`, "42", "tally"}
	for _, sc := range schemes {
		key := schemeKey(sc.name)
		line := "  " + key + strings.Repeat(" ", 16-len(key))
		if s.color {
			codes, _ := palette(sc, s.light, s.truecolor)
			for _, f := range frags {
				line += colorize(f, w, codes) + "  "
			}
		}
		fmt.Fprintln(s.out, strings.TrimRight(line, " "))
	}
	fmt.Fprintln(s.out, "colors okabe-ito picks one; add light, dark or auto for the shade")
}
