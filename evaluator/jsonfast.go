// Copyright 2017-2026 Nigele McCoy
// Licensed under the Apache License, Version 2.0;
// see LICENSE and NOTICE.

package evaluator

import (
	"encoding/json"
	"strings"

	"Turtle/object"
)

// fastJSON reads well-formed JSON straight into Turtle values, keeping
// object keys in order, several times faster than encoding/json's token
// stream. It's strict, and says ok only when it read the whole text
// cleanly; anything else (bad JSON, a number too big, any escape or
// non-ASCII text in a string, which encoding/json decodes) goes to the
// token-stream reader, so values and error messages stay exactly as they
// were.
func fastJSON(text string) (object.Object, bool) {
	p := jsonScan{s: text}
	p.space()
	v, ok := p.value(0)
	if !ok {
		return nil, false
	}
	p.space()
	return v, p.i == len(p.s)
}

type jsonScan struct {
	s string
	i int
}

// maxJSONDepth leaves very deep nesting to encoding/json's limits.
const maxJSONDepth = 1000

func (p *jsonScan) space() {
	for p.i < len(p.s) {
		switch p.s[p.i] {
		case ' ', '\t', '\n', '\r':
			p.i++
		default:
			return
		}
	}
}

func (p *jsonScan) value(depth int) (object.Object, bool) {
	if p.i >= len(p.s) || depth > maxJSONDepth {
		return nil, false
	}
	switch c := p.s[p.i]; {
	case c == '{':
		p.i++
		m := object.NewMap()
		p.space()
		if p.i < len(p.s) && p.s[p.i] == '}' {
			p.i++
			return m, true
		}
		for {
			p.space()
			key, ok := p.str()
			if !ok {
				return nil, false
			}
			p.space()
			if p.i >= len(p.s) || p.s[p.i] != ':' {
				return nil, false
			}
			p.i++
			p.space()
			v, ok := p.value(depth + 1)
			if !ok {
				return nil, false
			}
			m.Put(key, v)
			p.space()
			if p.i >= len(p.s) {
				return nil, false
			}
			if p.s[p.i] == ',' {
				p.i++
				continue
			}
			if p.s[p.i] == '}' {
				p.i++
				return m, true
			}
			return nil, false
		}
	case c == '[':
		p.i++
		l := &object.List{}
		p.space()
		if p.i < len(p.s) && p.s[p.i] == ']' {
			p.i++
			return l, true
		}
		for {
			p.space()
			v, ok := p.value(depth + 1)
			if !ok {
				return nil, false
			}
			l.Elements = append(l.Elements, v)
			p.space()
			if p.i >= len(p.s) {
				return nil, false
			}
			if p.s[p.i] == ',' {
				p.i++
				continue
			}
			if p.s[p.i] == ']' {
				p.i++
				return l, true
			}
			return nil, false
		}
	case c == '"':
		return p.str()
	case c == 't':
		return p.word("true", object.Bool(true))
	case c == 'f':
		return p.word("false", object.Bool(false))
	case c == 'n':
		return p.word("null", object.NoneValue)
	case c == '-' || c >= '0' && c <= '9':
		return p.number()
	}
	return nil, false
}

func (p *jsonScan) word(w string, v object.Object) (object.Object, bool) {
	if strings.HasPrefix(p.s[p.i:], w) {
		p.i += len(w)
		return v, true
	}
	return nil, false
}

// str reads a string with no escapes and only printable ASCII, the
// common case; anything else isn't ok (encoding/json decodes it).
func (p *jsonScan) str() (*object.String, bool) {
	if p.i >= len(p.s) || p.s[p.i] != '"' {
		return nil, false
	}
	start := p.i + 1
	for j := start; j < len(p.s); j++ {
		c := p.s[j]
		if c == '"' {
			p.i = j + 1
			return &object.String{Value: p.s[start:j]}, true
		}
		if c == '\\' || c < 0x20 || c >= 0x80 {
			return nil, false
		}
	}
	return nil, false
}

// number reads -?(0|[1-9][0-9]*)(.[0-9]+)?([eE][+-]?[0-9]+)?, as JSON
// writes numbers, then makes it as jsonNumber does.
func (p *jsonScan) number() (object.Object, bool) {
	start := p.i
	digits := func() int {
		n := 0
		for p.i < len(p.s) && p.s[p.i] >= '0' && p.s[p.i] <= '9' {
			p.i++
			n++
		}
		return n
	}
	if p.s[p.i] == '-' {
		p.i++
	}
	if p.i < len(p.s) && p.s[p.i] == '0' {
		p.i++
	} else if digits() == 0 {
		return nil, false
	}
	if p.i < len(p.s) && p.s[p.i] == '.' {
		p.i++
		if digits() == 0 {
			return nil, false
		}
	}
	if p.i < len(p.s) && (p.s[p.i] == 'e' || p.s[p.i] == 'E') {
		p.i++
		if p.i < len(p.s) && (p.s[p.i] == '+' || p.s[p.i] == '-') {
			p.i++
		}
		if digits() == 0 {
			return nil, false
		}
	}
	v, err := jsonNumber(nil, json.Number(p.s[start:p.i]))
	if err != nil {
		return nil, false
	}
	return v, true
}
