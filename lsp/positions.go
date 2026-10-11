// Copyright 2017-2026 Nigele McCoy
// Licensed under the Apache License, Version 2.0;
// see LICENSE and NOTICE.

package lsp

import "strings"

// Positions: editors count a line's characters in UTF-16 units (the LSP
// default) or bytes (when both sides agree to utf-8); Turtle's lexer
// counts bytes from the start of the file. text converts between them.

type text struct {
	src   string
	lines []int // the byte offset where each line starts
	utf8  bool  // columns in bytes, not UTF-16 units
}

func newText(src string, utf8Cols bool) *text {
	t := &text{src: src, lines: []int{0}, utf8: utf8Cols}
	for i := 0; i < len(src); i++ {
		if src[i] == '\n' {
			t.lines = append(t.lines, i+1)
		}
	}
	return t
}

func (t *text) lineText(n int) string {
	if n < 0 || n >= len(t.lines) {
		return ""
	}
	end := len(t.src)
	if n+1 < len(t.lines) {
		end = t.lines[n+1] - 1
	}
	return strings.TrimSuffix(t.src[t.lines[n]:end], "\r")
}

// position turns a byte offset into a line and a column, in the editor's
// units.
func (t *text) position(offset int) position {
	offset = max(0, min(offset, len(t.src)))
	line := 0
	for line+1 < len(t.lines) && t.lines[line+1] <= offset {
		line++
	}
	return position{Line: line, Character: t.units(t.src[t.lines[line]:offset])}
}

// offset turns an editor's line and column into a byte offset.
func (t *text) offset(p position) int {
	if p.Line < 0 {
		return 0
	}
	if p.Line >= len(t.lines) {
		return len(t.src)
	}
	start := t.lines[p.Line]
	s := t.lineText(p.Line)
	if t.utf8 {
		return start + min(p.Character, len(s))
	}
	n := 0
	for i, r := range s {
		if n >= p.Character {
			return start + i
		}
		n += utf16Len(r)
	}
	return start + len(s)
}

// units is how long s is in the editor's units.
func (t *text) units(s string) int {
	if t.utf8 {
		return len(s)
	}
	n := 0
	for _, r := range s {
		n += utf16Len(r)
	}
	return n
}

// utf16Len: characters past U+FFFF take two UTF-16 units.
func utf16Len(r rune) int {
	if r > 0xFFFF {
		return 2
	}
	return 1
}

// wordAt is the identifier at (or right before) offset, and where it
// starts.
func (t *text) wordAt(offset int) (string, int) {
	isWord := func(c byte) bool {
		return c == '_' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9'
	}
	start, end := offset, offset
	for start > 0 && isWord(t.src[start-1]) {
		start--
	}
	for end < len(t.src) && isWord(t.src[end]) {
		end++
	}
	if start > 0 && t.src[start-1] == '~' { // a private function: ~limit
		start--
	}
	return t.src[start:end], start
}
