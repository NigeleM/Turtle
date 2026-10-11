// Copyright 2017-2026 Nigele McCoy
// Licensed under the Apache License, Version 2.0;
// see LICENSE and NOTICE.

package evaluator

import (
	"strconv"
	"strings"
)

// jsonSyntax finds the first mistake in text read as JSON: its byte
// offset (the wrong character itself, or the end of the text) and what
// it is, worded as encoding/json words it. Go's own messages and
// positions change between Go versions, so bad JSON is described by
// this instead, the same whatever Go builds Turtle. bad is false when
// text is well-formed (parseJSON then keeps encoding/json's
// description, e.g. a number too big).
func jsonSyntax(text string) (offset int, msg string, bad bool) {
	s := text
	i := 0
	var stack []byte // the '{' and '[' still open
	space := func() {
		for i < len(s) && (s[i] == ' ' || s[i] == '\t' || s[i] == '\n' || s[i] == '\r') {
			i++
		}
	}
	ended := func() (int, string, bool) {
		if strings.TrimSpace(text) == "" {
			return len(s), "there's no JSON, the text is empty", true
		}
		return len(s), "the text ended before the JSON did", true
	}
	wrong := func(context string) (int, string, bool) {
		return i, "invalid character " + jsonQuoteChar(s[i]) + " " + context, true
	}
	isDigit := func(c byte) bool { return c >= '0' && c <= '9' }

	const (
		wantValue = iota
		wantKey
		afterValue
	)
	state := wantValue
	for {
		space()
		switch state {
		case wantKey:
			if i >= len(s) {
				return ended()
			}
			if s[i] != '"' {
				return wrong("looking for beginning of object key string")
			}
			if o, m, b := jsonSkipString(s, &i); b {
				return o, m, b
			}
			space()
			if i >= len(s) {
				return ended()
			}
			if s[i] != ':' {
				return wrong("after object key")
			}
			i++
			state = wantValue
		case wantValue:
			if i >= len(s) {
				return ended()
			}
			switch c := s[i]; {
			case c == '{' || c == '[':
				i++
				space()
				if i < len(s) && (c == '{' && s[i] == '}' || c == '[' && s[i] == ']') {
					i++
					state = afterValue
					continue
				}
				stack = append(stack, c)
				if c == '{' {
					state = wantKey
				}
				continue
			case c == '"':
				if o, m, b := jsonSkipString(s, &i); b {
					return o, m, b
				}
			case c == 't' || c == 'f' || c == 'n':
				word := map[byte]string{'t': "true", 'f': "false", 'n': "null"}[c]
				for j := 1; j < len(word); j++ {
					i++
					if i >= len(s) {
						return ended()
					}
					if s[i] != word[j] {
						return wrong("in literal " + word + " (expecting " + jsonQuoteChar(word[j]) + ")")
					}
				}
				i++
			case c == '-' || isDigit(c):
				if c == '-' {
					i++
					if i >= len(s) {
						return ended()
					}
					if !isDigit(s[i]) {
						return wrong("in numeric literal")
					}
				}
				if s[i] == '0' {
					i++
				} else {
					for i < len(s) && isDigit(s[i]) {
						i++
					}
				}
				if i < len(s) && s[i] == '.' {
					i++
					if i >= len(s) {
						return ended()
					}
					if !isDigit(s[i]) {
						return wrong("after decimal point in numeric literal")
					}
					for i < len(s) && isDigit(s[i]) {
						i++
					}
				}
				if i < len(s) && (s[i] == 'e' || s[i] == 'E') {
					i++
					if i < len(s) && (s[i] == '+' || s[i] == '-') {
						i++
					}
					if i >= len(s) {
						return ended()
					}
					if !isDigit(s[i]) {
						return wrong("in exponent of numeric literal")
					}
					for i < len(s) && isDigit(s[i]) {
						i++
					}
				}
			default:
				return wrong("looking for beginning of value")
			}
			state = afterValue
		case afterValue:
			if len(stack) == 0 {
				if i < len(s) {
					return i, "extra text after the JSON value", true
				}
				return 0, "", false
			}
			if i >= len(s) {
				return ended()
			}
			top := stack[len(stack)-1]
			switch {
			case s[i] == ',':
				i++
				if top == '{' {
					state = wantKey
				} else {
					state = wantValue
				}
			case top == '{' && s[i] == '}', top == '[' && s[i] == ']':
				i++
				stack = stack[:len(stack)-1]
			case top == '{':
				return wrong("after object key:value pair")
			default:
				return wrong("after array element")
			}
		}
	}
}

// jsonSkipString moves *i past the string starting there, or reports
// the mistake in it as jsonSyntax does.
func jsonSkipString(s string, i *int) (offset int, msg string, bad bool) {
	wrong := func(context string) (int, string, bool) {
		return *i, "invalid character " + jsonQuoteChar(s[*i]) + " " + context, true
	}
	ended := func() (int, string, bool) {
		return len(s), "the text ended before the JSON did", true
	}
	*i++ // the opening quote
	for {
		if *i >= len(s) {
			return ended()
		}
		switch c := s[*i]; {
		case c == '"':
			*i++
			return 0, "", false
		case c == '\\':
			*i++
			if *i >= len(s) {
				return ended()
			}
			switch s[*i] {
			case '"', '\\', '/', 'b', 'f', 'n', 'r', 't':
				*i++
			case 'u':
				*i++
				for k := 0; k < 4; k++ {
					if *i >= len(s) {
						return ended()
					}
					c := s[*i]
					if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f' || c >= 'A' && c <= 'F') {
						return wrong("in \\u hexadecimal character escape")
					}
					*i++
				}
			default:
				return wrong("in string escape code")
			}
		case c < 0x20:
			return wrong("in string literal")
		default:
			*i++
		}
	}
}

// jsonQuoteChar is a character as encoding/json's messages show it.
func jsonQuoteChar(c byte) string {
	if c == '\'' {
		return `'\''`
	}
	if c == '"' {
		return `'"'`
	}
	q := strconv.Quote(string(rune(c)))
	return "'" + q[1:len(q)-1] + "'"
}
