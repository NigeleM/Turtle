// Copyright 2017-2026 Nigele McCoy
// Licensed under the Apache License, Version 2.0;
// see LICENSE and NOTICE.

package repl

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

// The line editor: the keys a terminal sends, what they do to the line
// being typed, and redrawing the line (colored) after each key. None of
// it touches the terminal itself, so it can be tested.

type keyKind int

const (
	keyRune keyKind = iota
	keyEnter
	keyBackspace
	keyDelete
	keyLeft
	keyRight
	keyUp
	keyDown
	keyHome
	keyEnd
	keyWordLeft
	keyWordRight
	keyTab
	keyCancel    // Ctrl-C
	keyEOF       // Ctrl-D
	keyClear     // Ctrl-L
	keyKillEnd   // Ctrl-K
	keyKillStart // Ctrl-U
	keyKillWord  // Ctrl-W
	keyEscape    // Esc alone
	keyIgnored
)

type key struct {
	kind keyKind
	r    rune
}

// decodeKeys turns what the terminal sent into keys. Arrows and the like
// come as escape sequences: ESC [ A is up, ESC [ 1 ; 5 C is Ctrl-right,
// ESC b is Alt-b (word left). Text is UTF-8.
func decodeKeys(b []byte) []key {
	var out []key
	for len(b) > 0 {
		c := b[0]
		switch {
		case c == 27: // ESC
			k, n := decodeEscape(b)
			out = append(out, k)
			b = b[n:]
			continue
		case c == '\r' || c == '\n':
			out = append(out, key{kind: keyEnter})
		case c == 127 || c == 8:
			out = append(out, key{kind: keyBackspace})
		case c == 1:
			out = append(out, key{kind: keyHome})
		case c == 2:
			out = append(out, key{kind: keyLeft})
		case c == 3:
			out = append(out, key{kind: keyCancel})
		case c == 4:
			out = append(out, key{kind: keyEOF})
		case c == 5:
			out = append(out, key{kind: keyEnd})
		case c == 6:
			out = append(out, key{kind: keyRight})
		case c == 9:
			out = append(out, key{kind: keyTab})
		case c == 11:
			out = append(out, key{kind: keyKillEnd})
		case c == 12:
			out = append(out, key{kind: keyClear})
		case c == 14:
			out = append(out, key{kind: keyDown})
		case c == 16:
			out = append(out, key{kind: keyUp})
		case c == 21:
			out = append(out, key{kind: keyKillStart})
		case c == 23:
			out = append(out, key{kind: keyKillWord})
		case c < 32:
			out = append(out, key{kind: keyIgnored})
		default:
			r, n := utf8.DecodeRune(b)
			if r == utf8.RuneError && n <= 1 {
				if !utf8.FullRune(b) {
					return out // the rest of a character comes in the next read
				}
				n = 1
			}
			out = append(out, key{kind: keyRune, r: r})
			b = b[n:]
			continue
		}
		b = b[1:]
	}
	return out
}

// decodeEscape reads one escape sequence at the start of b.
func decodeEscape(b []byte) (key, int) {
	if len(b) == 1 {
		return key{kind: keyEscape}, 1
	}
	switch b[1] {
	case 'b':
		return key{kind: keyWordLeft}, 2
	case 'f':
		return key{kind: keyWordRight}, 2
	case 127, 8:
		return key{kind: keyKillWord}, 2
	case 'O': // ESC O H / F (Home / End on some terminals)
		if len(b) >= 3 {
			switch b[2] {
			case 'H':
				return key{kind: keyHome}, 3
			case 'F':
				return key{kind: keyEnd}, 3
			}
			return key{kind: keyIgnored}, 3
		}
		return key{kind: keyIgnored}, 2
	case '[':
	default:
		return key{kind: keyIgnored}, 2
	}
	// ESC [ params final
	i := 2
	for i < len(b) && (b[i] >= '0' && b[i] <= '9' || b[i] == ';') {
		i++
	}
	if i >= len(b) {
		return key{kind: keyIgnored}, len(b)
	}
	params, final := string(b[2:i]), b[i]
	n := i + 1
	ctrl := strings.HasSuffix(params, ";5") || strings.HasSuffix(params, ";3")
	switch final {
	case 'A':
		return key{kind: keyUp}, n
	case 'B':
		return key{kind: keyDown}, n
	case 'C':
		if ctrl {
			return key{kind: keyWordRight}, n
		}
		return key{kind: keyRight}, n
	case 'D':
		if ctrl {
			return key{kind: keyWordLeft}, n
		}
		return key{kind: keyLeft}, n
	case 'H':
		return key{kind: keyHome}, n
	case 'F':
		return key{kind: keyEnd}, n
	case '~':
		switch params {
		case "1", "7":
			return key{kind: keyHome}, n
		case "4", "8":
			return key{kind: keyEnd}, n
		case "3":
			return key{kind: keyDelete}, n
		}
	}
	return key{kind: keyIgnored}, n
}

// line is the text being typed and where the cursor is in it.
type line struct {
	buf []rune
	pos int
}

func (l *line) String() string { return string(l.buf) }

func (l *line) set(s string) {
	l.buf = []rune(s)
	l.pos = len(l.buf)
}

// edit applies an editing key; it reports false for keys it doesn't
// handle (Enter, history, Ctrl-C ...), which the session does.
func (l *line) edit(k key) bool {
	switch k.kind {
	case keyRune:
		l.buf = append(l.buf[:l.pos], append([]rune{k.r}, l.buf[l.pos:]...)...)
		l.pos++
	case keyTab:
		for range 4 {
			l.edit(key{kind: keyRune, r: ' '})
		}
	case keyBackspace:
		if l.pos > 0 {
			l.buf = append(l.buf[:l.pos-1], l.buf[l.pos:]...)
			l.pos--
		}
	case keyDelete:
		if l.pos < len(l.buf) {
			l.buf = append(l.buf[:l.pos], l.buf[l.pos+1:]...)
		}
	case keyLeft:
		if l.pos > 0 {
			l.pos--
		}
	case keyRight:
		if l.pos < len(l.buf) {
			l.pos++
		}
	case keyHome:
		l.pos = 0
	case keyEnd:
		l.pos = len(l.buf)
	case keyWordLeft:
		l.pos = l.wordStart()
	case keyWordRight:
		for l.pos < len(l.buf) && !isWord(l.buf[l.pos]) {
			l.pos++
		}
		for l.pos < len(l.buf) && isWord(l.buf[l.pos]) {
			l.pos++
		}
	case keyKillEnd:
		l.buf = l.buf[:l.pos]
	case keyKillStart:
		l.buf = l.buf[l.pos:]
		l.pos = 0
	case keyKillWord:
		start := l.wordStart()
		l.buf = append(l.buf[:start], l.buf[l.pos:]...)
		l.pos = start
	default:
		return false
	}
	return true
}

func (l *line) wordStart() int {
	p := l.pos
	for p > 0 && !isWord(l.buf[p-1]) {
		p--
	}
	for p > 0 && isWord(l.buf[p-1]) {
		p--
	}
	return p
}

func isWord(r rune) bool { return r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r) }

// render redraws the prompt and line: from where the last drawing left
// the cursor (row rows below the prompt's start), clear to the end of the
// screen, write it all, and put the cursor back at the line's cursor. It
// gives back the text to write and the cursor's new row. width is the
// terminal's width; a long line wraps onto more rows.
func render(prompt string, promptWidth int, text string, colored string, pos, width, row int) (string, int) {
	if width < 1 {
		width = 80
	}
	var b strings.Builder
	if row > 0 {
		fmt.Fprintf(&b, "\x1b[%dA", row)
	}
	b.WriteString("\r\x1b[J")
	b.WriteString(prompt)
	b.WriteString(colored)
	end := promptWidth + utf8.RuneCountInString(text)
	cur := promptWidth + pos
	if end > 0 && end%width == 0 {
		b.WriteString("\r\n") // past the last column: start the next row now
	}
	endRow, curRow, curCol := end/width, cur/width, cur%width
	if up := endRow - curRow; up > 0 {
		fmt.Fprintf(&b, "\x1b[%dA", up)
	}
	b.WriteString("\r")
	if curCol > 0 {
		fmt.Fprintf(&b, "\x1b[%dC", curCol)
	}
	return b.String(), curRow
}
