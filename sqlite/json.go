// Copyright 2017-2026 Nigele McCoy
// Licensed under the Apache License, Version 2.0;
// see LICENSE and NOTICE.

package sqlite

import (
	"math"
	"strconv"
	"strings"
	"unicode/utf8"
)

// JSON functions, as SQLite has them built in. JSON is text; the
// functions read it, find values by path ($.a.b[0], $[#-1]), and build
// and change it. Object keys keep their order, and numbers keep the text
// they were written with.
//
// A value that comes straight from a function giving JSON (json(),
// json_object(), json_array(), json_set(), ->, ...) is put into another
// as JSON; any other text is put in as a JSON string. That's SQLite's
// "JSON subtype", decided here by looking at the expression.

type jkind int

const (
	jNull jkind = iota
	jTrue
	jFalse
	jInt
	jReal
	jString
	jArray
	jObject
)

type jnode struct {
	kind  jkind
	raw   string // numbers: as written; strings: the text between the quotes, escapes kept
	elems []*jnode
	keys  []*jnode // object keys (strings), beside elems
}

// ---- reading ----

type jparser struct {
	s   string
	pos int
}

func parseJSON(text string) (*jnode, bool) {
	p := &jparser{s: text}
	p.space()
	n, ok := p.value(0)
	if !ok {
		return nil, false
	}
	p.space()
	if p.pos != len(p.s) {
		return nil, false
	}
	return n, true
}

func mustJSON(v Value) *jnode {
	n, ok := parseJSON(textValue(v))
	if !ok {
		fail("malformed JSON")
	}
	return n
}

func (p *jparser) space() {
	for p.pos < len(p.s) {
		switch p.s[p.pos] {
		case ' ', '\t', '\n', '\r':
			p.pos++
		default:
			return
		}
	}
}

func (p *jparser) value(depth int) (*jnode, bool) {
	if depth > 1000 || p.pos >= len(p.s) {
		return nil, false
	}
	switch c := p.s[p.pos]; {
	case c == '{':
		p.pos++
		n := &jnode{kind: jObject}
		p.space()
		if p.pos < len(p.s) && p.s[p.pos] == '}' {
			p.pos++
			return n, true
		}
		for {
			p.space()
			if p.pos >= len(p.s) || p.s[p.pos] != '"' {
				return nil, false
			}
			k, ok := p.str()
			if !ok {
				return nil, false
			}
			p.space()
			if p.pos >= len(p.s) || p.s[p.pos] != ':' {
				return nil, false
			}
			p.pos++
			p.space()
			v, ok := p.value(depth + 1)
			if !ok {
				return nil, false
			}
			n.keys = append(n.keys, k)
			n.elems = append(n.elems, v)
			p.space()
			if p.pos < len(p.s) && p.s[p.pos] == ',' {
				p.pos++
				continue
			}
			if p.pos < len(p.s) && p.s[p.pos] == '}' {
				p.pos++
				return n, true
			}
			return nil, false
		}
	case c == '[':
		p.pos++
		n := &jnode{kind: jArray}
		p.space()
		if p.pos < len(p.s) && p.s[p.pos] == ']' {
			p.pos++
			return n, true
		}
		for {
			p.space()
			v, ok := p.value(depth + 1)
			if !ok {
				return nil, false
			}
			n.elems = append(n.elems, v)
			p.space()
			if p.pos < len(p.s) && p.s[p.pos] == ',' {
				p.pos++
				continue
			}
			if p.pos < len(p.s) && p.s[p.pos] == ']' {
				p.pos++
				return n, true
			}
			return nil, false
		}
	case c == '"':
		return p.str()
	case strings.HasPrefix(p.s[p.pos:], "true"):
		p.pos += 4
		return &jnode{kind: jTrue}, true
	case strings.HasPrefix(p.s[p.pos:], "false"):
		p.pos += 5
		return &jnode{kind: jFalse}, true
	case strings.HasPrefix(p.s[p.pos:], "null"):
		p.pos += 4
		return &jnode{kind: jNull}, true
	case c == '-' || c >= '0' && c <= '9':
		start := p.pos
		if c == '-' {
			p.pos++
		}
		digits := func() int {
			n := 0
			for p.pos < len(p.s) && p.s[p.pos] >= '0' && p.s[p.pos] <= '9' {
				p.pos++
				n++
			}
			return n
		}
		if p.pos < len(p.s) && p.s[p.pos] == '0' {
			p.pos++
		} else if digits() == 0 {
			return nil, false
		}
		kind := jInt
		if p.pos < len(p.s) && p.s[p.pos] == '.' {
			p.pos++
			if digits() == 0 {
				return nil, false
			}
			kind = jReal
		}
		if p.pos < len(p.s) && (p.s[p.pos] == 'e' || p.s[p.pos] == 'E') {
			p.pos++
			if p.pos < len(p.s) && (p.s[p.pos] == '+' || p.s[p.pos] == '-') {
				p.pos++
			}
			if digits() == 0 {
				return nil, false
			}
			kind = jReal
		}
		return &jnode{kind: kind, raw: p.s[start:p.pos]}, true
	}
	return nil, false
}

func (p *jparser) str() (*jnode, bool) {
	p.pos++ // the opening quote
	start := p.pos
	for p.pos < len(p.s) {
		c := p.s[p.pos]
		switch {
		case c == '"':
			n := &jnode{kind: jString, raw: p.s[start:p.pos]}
			p.pos++
			return n, true
		case c == '\\':
			p.pos++
			if p.pos >= len(p.s) {
				return nil, false
			}
			switch p.s[p.pos] {
			case '"', '\\', '/', 'b', 'f', 'n', 'r', 't':
				p.pos++
			case 'u':
				if p.pos+5 > len(p.s) {
					return nil, false
				}
				if _, err := strconv.ParseUint(p.s[p.pos+1:p.pos+5], 16, 32); err != nil {
					return nil, false
				}
				p.pos += 5
			default:
				return nil, false
			}
		case c < 0x20:
			return nil, false
		default:
			p.pos++
		}
	}
	return nil, false
}

// text decodes a string node's escapes.
func (n *jnode) text() string {
	if !strings.Contains(n.raw, "\\") {
		return n.raw
	}
	s, err := strconv.Unquote(`"` + strings.ReplaceAll(n.raw, `\/`, `/`) + `"`)
	if err != nil {
		var sb strings.Builder
		for i := 0; i < len(n.raw); i++ {
			if n.raw[i] == '\\' && i+5 < len(n.raw) && n.raw[i+1] == 'u' {
				r, _ := strconv.ParseUint(n.raw[i+2:i+6], 16, 32)
				sb.WriteRune(rune(r))
				i += 5
				continue
			}
			sb.WriteByte(n.raw[i])
		}
		return sb.String()
	}
	return s
}

// ---- writing ----

func (n *jnode) String() string {
	var sb strings.Builder
	n.write(&sb)
	return sb.String()
}

func (n *jnode) write(sb *strings.Builder) {
	switch n.kind {
	case jNull:
		sb.WriteString("null")
	case jTrue:
		sb.WriteString("true")
	case jFalse:
		sb.WriteString("false")
	case jInt, jReal:
		sb.WriteString(n.raw)
	case jString:
		sb.WriteByte('"')
		sb.WriteString(n.raw)
		sb.WriteByte('"')
	case jArray:
		sb.WriteByte('[')
		for i, e := range n.elems {
			if i > 0 {
				sb.WriteByte(',')
			}
			e.write(sb)
		}
		sb.WriteByte(']')
	case jObject:
		sb.WriteByte('{')
		for i, e := range n.elems {
			if i > 0 {
				sb.WriteByte(',')
			}
			n.keys[i].write(sb)
			sb.WriteByte(':')
			e.write(sb)
		}
		sb.WriteByte('}')
	}
}

// jsonString makes a string node from text, escaping it.
func jsonString(s string) *jnode {
	var sb strings.Builder
	for _, r := range s {
		switch r {
		case '"':
			sb.WriteString(`\"`)
		case '\\':
			sb.WriteString(`\\`)
		case '\b':
			sb.WriteString(`\b`)
		case '\f':
			sb.WriteString(`\f`)
		case '\n':
			sb.WriteString(`\n`)
		case '\r':
			sb.WriteString(`\r`)
		case '\t':
			sb.WriteString(`\t`)
		default:
			if r < 0x20 {
				sb.WriteString(`\u00`)
				sb.WriteByte("0123456789abcdef"[r>>4])
				sb.WriteByte("0123456789abcdef"[r&15])
			} else if r == utf8.RuneError {
				sb.WriteRune(r)
			} else {
				sb.WriteRune(r)
			}
		}
	}
	return &jnode{kind: jString, raw: sb.String()}
}

// fromSQL turns an SQL value into JSON: text as a string, unless isJSON
// (it came from a JSON function), when it's parsed as JSON.
func fromSQLValue(v Value, isJSON bool) *jnode {
	switch x := v.(type) {
	case nil:
		return &jnode{kind: jNull}
	case int64:
		return &jnode{kind: jInt, raw: strconv.FormatInt(x, 10)}
	case float64:
		if math.IsInf(x, 0) || math.IsNaN(x) {
			if math.IsInf(x, -1) {
				return &jnode{kind: jReal, raw: "-9e999"}
			}
			if math.IsInf(x, 1) {
				return &jnode{kind: jReal, raw: "9e999"}
			}
			return &jnode{kind: jNull}
		}
		return &jnode{kind: jReal, raw: formatReal(x)}
	case []byte:
		fail("JSON cannot hold BLOB values")
	}
	if isJSON {
		return mustJSON(v)
	}
	return jsonString(textValue(v))
}

// sqlValue is a JSON value as an SQL value: numbers and text as
// themselves, true/false as 1/0, null as NULL, arrays and objects as
// JSON text.
func (n *jnode) sqlValue() Value {
	switch n.kind {
	case jNull:
		return nil
	case jTrue:
		return int64(1)
	case jFalse:
		return int64(0)
	case jInt:
		if i, err := strconv.ParseInt(n.raw, 10, 64); err == nil {
			return i
		}
		f, _ := strconv.ParseFloat(n.raw, 64)
		return f
	case jReal:
		f, _ := strconv.ParseFloat(n.raw, 64)
		return f
	case jString:
		return n.text()
	}
	return n.String()
}

func (n *jnode) typeName() string {
	return []string{"null", "true", "false", "integer", "real", "text", "array", "object"}[n.kind]
}

func (n *jnode) clone() *jnode {
	c := *n
	c.elems = make([]*jnode, len(n.elems))
	for i, e := range n.elems {
		c.elems[i] = e.clone()
	}
	c.keys = append([]*jnode{}, n.keys...)
	return &c
}

// ---- paths ----

type jstep struct {
	key     string
	isIndex bool
	index   int64
	fromEnd bool // [#-n]; [#] is fromEnd with index 0
}

func parsePath(path string) []jstep {
	if !strings.HasPrefix(path, "$") {
		fail("bad JSON path: '%s'", path)
	}
	var steps []jstep
	s := path[1:]
	for len(s) > 0 {
		switch s[0] {
		case '.':
			s = s[1:]
			if strings.HasPrefix(s, `"`) {
				end := strings.IndexByte(s[1:], '"')
				if end < 0 {
					fail("bad JSON path: '%s'", path)
				}
				steps = append(steps, jstep{key: s[1 : 1+end]})
				s = s[end+2:]
				continue
			}
			end := strings.IndexAny(s, ".[")
			if end < 0 {
				end = len(s)
			}
			if end == 0 {
				fail("bad JSON path: '%s'", path)
			}
			steps = append(steps, jstep{key: s[:end]})
			s = s[end:]
		case '[':
			end := strings.IndexByte(s, ']')
			if end < 0 {
				fail("bad JSON path: '%s'", path)
			}
			inside := strings.TrimSpace(s[1:end])
			st := jstep{isIndex: true}
			if strings.HasPrefix(inside, "#") {
				st.fromEnd = true
				rest := strings.TrimSpace(inside[1:])
				if rest != "" {
					if !strings.HasPrefix(rest, "-") {
						fail("bad JSON path: '%s'", path)
					}
					n, err := strconv.ParseInt(strings.TrimSpace(rest[1:]), 10, 64)
					if err != nil {
						fail("bad JSON path: '%s'", path)
					}
					st.index = n
				}
			} else {
				n, err := strconv.ParseInt(inside, 10, 64)
				if err != nil || n < 0 {
					fail("bad JSON path: '%s'", path)
				}
				st.index = n
			}
			steps = append(steps, st)
			s = s[end+1:]
		default:
			fail("bad JSON path: '%s'", path)
		}
	}
	return steps
}

// lookup follows steps from n; nil if it isn't there.
func (n *jnode) lookup(steps []jstep) *jnode {
	for _, st := range steps {
		n = n.child(st)
		if n == nil {
			return nil
		}
	}
	return n
}

func (n *jnode) childIndex(st jstep) int {
	if st.isIndex {
		if n.kind != jArray {
			return -1
		}
		i := st.index
		if st.fromEnd {
			i = int64(len(n.elems)) - st.index
		}
		if i < 0 || i >= int64(len(n.elems)) {
			return -1
		}
		return int(i)
	}
	if n.kind != jObject {
		return -1
	}
	for i, k := range n.keys {
		if k.text() == st.key {
			return i
		}
	}
	return -1
}

func (n *jnode) child(st jstep) *jnode {
	if i := n.childIndex(st); i >= 0 {
		return n.elems[i]
	}
	return nil
}

// edit puts value at the path: mode "set" (always), "insert" (only if
// missing), "replace" (only if there), "remove".
func (n *jnode) edit(steps []jstep, value *jnode, mode string) *jnode {
	if len(steps) == 0 {
		switch mode {
		case "remove":
			return nil
		case "insert":
			return n
		}
		return value
	}
	st := steps[0]
	i := n.childIndex(st)
	if i >= 0 {
		updated := n.elems[i].edit(steps[1:], value, mode)
		if updated == nil {
			n.elems = append(n.elems[:i], n.elems[i+1:]...)
			if n.kind == jObject {
				n.keys = append(n.keys[:i], n.keys[i+1:]...)
			}
		} else {
			n.elems[i] = updated
		}
		return n
	}
	if mode == "replace" || mode == "remove" {
		return n
	}
	// Missing: make it (and any containers on the way).
	build := func(rest []jstep) *jnode {
		v := value
		for k := len(rest) - 1; k >= 0; k-- {
			if rest[k].isIndex {
				if !(rest[k].fromEnd && rest[k].index == 0) {
					return nil
				}
				v = &jnode{kind: jArray, elems: []*jnode{v}}
			} else {
				v = &jnode{kind: jObject, keys: []*jnode{jsonString(rest[k].key)}, elems: []*jnode{v}}
			}
		}
		return v
	}
	v := build(steps[1:])
	if v == nil {
		return n
	}
	switch {
	case st.isIndex && n.kind == jArray && st.fromEnd && st.index == 0:
		n.elems = append(n.elems, v)
	case !st.isIndex && n.kind == jObject:
		n.keys = append(n.keys, jsonString(st.key))
		n.elems = append(n.elems, v)
	}
	return n
}

// patch is RFC 7396 merge patch.
func mergePatch(target, patch *jnode) *jnode {
	if patch.kind != jObject {
		return patch
	}
	if target == nil || target.kind != jObject {
		target = &jnode{kind: jObject}
	}
	for i, k := range patch.keys {
		v := patch.elems[i]
		idx := target.childIndex(jstep{key: k.text()})
		if v.kind == jNull {
			if idx >= 0 {
				target.elems = append(target.elems[:idx], target.elems[idx+1:]...)
				target.keys = append(target.keys[:idx], target.keys[idx+1:]...)
			}
			continue
		}
		if idx >= 0 {
			target.elems[idx] = mergePatch(target.elems[idx], v)
		} else {
			target.keys = append(target.keys, k)
			target.elems = append(target.elems, mergePatch(nil, v))
		}
	}
	return target
}

// ---- which expressions give JSON ----

var jsonResultFuncs = map[string]bool{"json": true, "json_array": true, "json_object": true, "json_set": true,
	"json_insert": true, "json_replace": true, "json_remove": true, "json_patch": true, "json_quote": true,
	"json_group_array": true, "json_group_object": true}

func isJSONExpr(e expr) bool {
	switch x := e.(type) {
	case *callExpr:
		if jsonResultFuncs[x.name] {
			return true
		}
		if x.name == "json_extract" {
			return true // its arrays and objects; scalars are checked by value
		}
	case *binExpr:
		return x.op == "->"
	case *collateExpr:
		return isJSONExpr(x.x)
	}
	return false
}

// jsonArg turns a function's argument into JSON, as JSON when it came
// from a JSON function.
func jsonArg(e expr, v Value) *jnode {
	if v == nil {
		return &jnode{kind: jNull}
	}
	if isJSONExpr(e) {
		if s, ok := v.(string); ok {
			if n, ok := parseJSON(s); ok {
				return n
			}
		}
	}
	return fromSQLValue(v, false)
}

// pathArg reads a path for -> and ->>: a full path, a label, or an
// array index.
func arrowPath(v Value) []jstep {
	switch x := v.(type) {
	case int64:
		if x < 0 {
			return []jstep{{isIndex: true, fromEnd: true, index: -x}}
		}
		return []jstep{{isIndex: true, index: x}}
	}
	s := textValue(v)
	if strings.HasPrefix(s, "$") {
		return parsePath(s)
	}
	if strings.HasPrefix(s, "[") || strings.HasPrefix(s, ".") {
		return parsePath("$" + s)
	}
	return []jstep{{key: s}}
}

// evalArrow is a -> b (JSON text) and a ->> b (an SQL value).
func (r *runner) evalArrow(x *binExpr) Value {
	l, rv := r.eval(x.l), r.eval(x.r)
	if l == nil || rv == nil {
		return nil
	}
	n, ok := parseJSON(textValue(l))
	if !ok {
		fail("malformed JSON")
	}
	found := n.lookup(arrowPath(rv))
	if found == nil {
		return nil
	}
	if x.op == "->" {
		return found.String()
	}
	return found.sqlValue()
}

// callJSON runs the JSON scalar functions.
func (r *runner) callJSON(c *callExpr, args []Value) (Value, bool) {
	need := func(lo, hi int) {
		if len(args) < lo || (hi >= 0 && len(args) > hi) {
			fail("SQL: wrong number of arguments to function %s()", c.name)
		}
	}
	switch c.name {
	case "json":
		need(1, 1)
		if args[0] == nil {
			return nil, true
		}
		return mustJSON(args[0]).String(), true
	case "json_valid":
		need(1, 2)
		if args[0] == nil {
			return nil, true
		}
		_, ok := parseJSON(textValue(args[0]))
		return boolValue(ok), true
	case "json_error_position":
		need(1, 1)
		if args[0] == nil {
			return nil, true
		}
		p := &jparser{s: textValue(args[0])}
		p.space()
		if _, ok := p.value(0); ok {
			p.space()
			if p.pos == len(p.s) {
				return int64(0), true
			}
		}
		return int64(p.pos + 1), true
	case "json_quote":
		need(1, 1)
		return jsonArg(c.args[0], args[0]).String(), true
	case "json_array":
		n := &jnode{kind: jArray}
		for i, a := range args {
			n.elems = append(n.elems, jsonArg(c.args[i], a))
		}
		return n.String(), true
	case "json_object":
		if len(args)%2 != 0 {
			fail("json_object() requires an even number of arguments")
		}
		n := &jnode{kind: jObject}
		for i := 0; i < len(args); i += 2 {
			if _, ok := args[i].(string); !ok {
				fail("json_object() labels must be TEXT")
			}
			n.keys = append(n.keys, jsonString(args[i].(string)))
			n.elems = append(n.elems, jsonArg(c.args[i+1], args[i+1]))
		}
		return n.String(), true
	case "json_extract":
		need(1, -1)
		if args[0] == nil {
			return nil, true
		}
		n := mustJSON(args[0])
		if len(args) == 2 {
			if args[1] == nil {
				return nil, true
			}
			found := n.lookup(parsePath(textValue(args[1])))
			if found == nil {
				return nil, true
			}
			return found.sqlValue(), true
		}
		out := &jnode{kind: jArray}
		for _, p := range args[1:] {
			if p == nil {
				return nil, true
			}
			found := n.lookup(parsePath(textValue(p)))
			if found == nil {
				found = &jnode{kind: jNull}
			}
			out.elems = append(out.elems, found)
		}
		return out.String(), true
	case "json_type":
		need(1, 2)
		if args[0] == nil {
			return nil, true
		}
		n := mustJSON(args[0])
		if len(args) == 2 {
			if args[1] == nil {
				return nil, true
			}
			n = n.lookup(parsePath(textValue(args[1])))
			if n == nil {
				return nil, true
			}
		}
		return n.typeName(), true
	case "json_array_length":
		need(1, 2)
		if args[0] == nil {
			return nil, true
		}
		n := mustJSON(args[0])
		if len(args) == 2 {
			n = n.lookup(parsePath(textValue(args[1])))
			if n == nil {
				return nil, true
			}
		}
		if n.kind != jArray {
			return int64(0), true
		}
		return int64(len(n.elems)), true
	case "json_set", "json_insert", "json_replace":
		if len(args) < 1 || len(args)%2 != 1 {
			fail("json_%s() needs an odd number of arguments", strings.TrimPrefix(c.name, "json_"))
		}
		if args[0] == nil {
			return nil, true
		}
		n := mustJSON(args[0])
		mode := strings.TrimPrefix(c.name, "json_")
		for i := 1; i < len(args); i += 2 {
			if args[i] == nil {
				return nil, true
			}
			n = n.edit(parsePath(textValue(args[i])), jsonArg(c.args[i+1], args[i+1]), mode)
		}
		return n.String(), true
	case "json_remove":
		need(1, -1)
		if args[0] == nil {
			return nil, true
		}
		n := mustJSON(args[0])
		for _, p := range args[1:] {
			if p == nil {
				return nil, true
			}
			steps := parsePath(textValue(p))
			if len(steps) == 0 {
				return nil, true
			}
			n = n.edit(steps, nil, "remove")
		}
		return n.String(), true
	case "json_patch":
		need(2, 2)
		if args[0] == nil || args[1] == nil {
			return nil, true
		}
		return mergePatch(mustJSON(args[0]), mustJSON(args[1])).String(), true
	}
	return nil, false
}

// ---- json_each and json_tree ----

// jsonEachRows lists the elements of a JSON value (json_each), or every
// value in it, depth first (json_tree): key, value, type, atom, id,
// parent, fullkey, path.
func jsonEachRows(tree bool, args []Value) [][]Value {
	if len(args) < 1 || len(args) > 2 {
		fail("SQL: json_each takes a JSON value and an optional path")
	}
	if args[0] == nil {
		return nil
	}
	root := mustJSON(args[0])
	base := "$"
	if len(args) == 2 && args[1] != nil {
		base = textValue(args[1])
		root = root.lookup(parsePath(base))
		if root == nil {
			return nil
		}
	}
	var out [][]Value
	id := int64(0)
	row := func(key Value, n *jnode, parent Value, fullkey, path string) int64 {
		id++
		var value, atom Value
		if n.kind == jArray || n.kind == jObject {
			value = n.String()
		} else {
			value, atom = n.sqlValue(), n.sqlValue()
		}
		out = append(out, []Value{key, value, n.typeName(), atom, id, parent, fullkey, path})
		return id
	}
	keyPath := func(parent string, n *jnode, i int) (Value, string) {
		if n.kind == jArray {
			return int64(i), parent + "[" + strconv.Itoa(i) + "]"
		}
		k := n.keys[i].text()
		if isPlainKey(k) {
			return k, parent + "." + k
		}
		return k, parent + `."` + k + `"`
	}
	if !tree {
		if root.kind != jArray && root.kind != jObject {
			row(nil, root, nil, base, base)
			return out
		}
		for i, e := range root.elems {
			k, full := keyPath(base, root, i)
			row(k, e, nil, full, base)
		}
		return out
	}
	var walkTree func(key Value, n *jnode, parent Value, full, path string)
	walkTree = func(key Value, n *jnode, parent Value, full, path string) {
		me := row(key, n, parent, full, path)
		for i, e := range n.elems {
			k, cf := keyPath(full, n, i)
			walkTree(k, e, me, cf, full)
		}
	}
	parentPath := base
	if base != "$" {
		if i := strings.LastIndexAny(base, ".["); i > 0 {
			parentPath = base[:i]
		}
	}
	var rootKey Value
	if base != "$" {
		steps := parsePath(base)
		last := steps[len(steps)-1]
		if last.isIndex {
			rootKey = last.index
		} else {
			rootKey = last.key
		}
	}
	walkTree(rootKey, root, nil, base, parentPath)
	return out
}

func isPlainKey(k string) bool {
	if k == "" {
		return false
	}
	for i, c := range k {
		if !(c == '_' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || i > 0 && c >= '0' && c <= '9') {
			return false
		}
	}
	return true
}
