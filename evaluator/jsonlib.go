package evaluator

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"math"
	"os"
	"strconv"
	"strings"

	"Turtle/object"
)

// The "json" builtin module: JSON text and files to Turtle values and back.
//
//	data = load[text]                       // JSON text -> value
//	text = json_text[data]                  // value -> JSON text (one line)
//	cfg = json_read["config.json"]          // read and load a file
//	json_write["out.json", data]            // write a value, indented
//	name = json_get[data, "users", 0, "name"] // dig in; none if missing
//
// JSON objects become maps (keys in file order), arrays lists, whole
// numbers integers, other numbers floats, null none. Going back, sets
// are arrays, assembled values objects of their fields, and number map
// keys become text keys. Bad JSON is an error of kind json.
func (it *Interpreter) callJSON(name string, args []object.Object) object.Object {
	switch name {
	case "load":
		requireFuncArgs(name, args, 1)
		return parseJSON("load", asStringArg(name, args[0]))
	case "json_text":
		requireFuncArgs(name, args, 1)
		return &object.String{Value: string(toJSON("json_text", args[0], ""))}
	case "json_read":
		requireFuncArgs(name, args, 1)
		path := asStringArg(name, args[0])
		data, err := os.ReadFile(it.resolvePath(path))
		if err != nil {
			fatalKind(kindFile, "json_read %s: %s", path, fileProblem(err))
		}
		return parseJSON("json_read "+path, string(data))
	case "json_write":
		requireFuncArgs(name, args, 2)
		path := asStringArg(name, args[0])
		text := append(toJSON("json_write", args[1], "  "), '\n')
		if err := os.WriteFile(it.resolvePath(path), text, 0o644); err != nil {
			fatalKind(kindFile, "json_write %s: %s", path, fileProblem(err))
		}
		return object.NoneValue
	case "json_get":
		if len(args) < 2 {
			fatalf("'json_get' expects a value and at least one key or index, e.g. json_get[data, \"users\", 0], got %d argument(s)", len(args))
		}
		return jsonGet(args[0], args[1:])
	}
	fatalKind(kindName, "no json function %q", name)
	return nil
}

// ---- JSON text -> Turtle ------------------------------------------------

// parseJSON loads text, reporting bad JSON as "<who>: invalid JSON at
// line L, column C: ...".
func parseJSON(who, text string) object.Object {
	dec := json.NewDecoder(strings.NewReader(text))
	dec.UseNumber()
	val, err := decodeJSON(dec)
	if err == nil {
		// Only whitespace may follow the value.
		if _, extra := dec.Token(); extra != io.EOF {
			err = jsonProblem{offset: dec.InputOffset(), msg: "extra text after the JSON value"}
		}
	}
	if err != nil {
		fatalKind(kindJSON, "%s: invalid JSON %s", who, describeJSONError(text, dec, err))
	}
	return val
}

// jsonProblem is a decoding error with the byte offset it happened at.
type jsonProblem struct {
	offset int64
	msg    string
}

func (p jsonProblem) Error() string { return p.msg }

func decodeJSON(dec *json.Decoder) (object.Object, error) {
	tok, err := dec.Token()
	if err != nil {
		return nil, err
	}
	switch t := tok.(type) {
	case json.Delim:
		switch t {
		case '{':
			m := object.NewMap()
			for dec.More() {
				keyTok, err := dec.Token()
				if err != nil {
					return nil, err
				}
				key, _ := keyTok.(string) // the decoder only allows string keys
				val, err := decodeJSON(dec)
				if err != nil {
					return nil, err
				}
				m.Put(&object.String{Value: key}, val)
			}
			if _, err := dec.Token(); err != nil { // '}'
				return nil, err
			}
			return m, nil
		case '[':
			l := &object.List{}
			for dec.More() {
				val, err := decodeJSON(dec)
				if err != nil {
					return nil, err
				}
				l.Elements = append(l.Elements, val)
			}
			if _, err := dec.Token(); err != nil { // ']'
				return nil, err
			}
			return l, nil
		}
		return nil, jsonProblem{offset: dec.InputOffset(), msg: "unexpected " + strconv.Quote(t.String())}
	case string:
		return &object.String{Value: t}, nil
	case json.Number:
		return jsonNumber(dec, t)
	case bool:
		return &object.Boolean{Value: t}, nil
	case nil:
		return object.NoneValue, nil
	}
	return nil, jsonProblem{offset: dec.InputOffset(), msg: "unexpected value"}
}

// jsonNumber is an integer when the number is whole and fits, else a
// float: 3 -> 3, 3.0 -> 3.0, 1e3 -> 1000.0, 99999999999999999999 -> float.
func jsonNumber(dec *json.Decoder, n json.Number) (object.Object, error) {
	s := n.String()
	if !strings.ContainsAny(s, ".eE") {
		if i, err := strconv.ParseInt(s, 10, 64); err == nil {
			return &object.Integer{Value: i}, nil
		}
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil || math.IsInf(f, 0) {
		return nil, jsonProblem{offset: dec.InputOffset(), msg: "number " + s + " is too big"}
	}
	return &object.Float{Value: f}, nil
}

// describeJSONError turns a decoder error into "at line L, column C:
// <what>", with the place counted from the start of text.
func describeJSONError(text string, dec *json.Decoder, err error) string {
	offset := dec.InputOffset()
	msg := err.Error()
	var syn *json.SyntaxError
	var prob jsonProblem
	switch {
	case errors.As(err, &syn):
		offset = syn.Offset
		msg = strings.TrimPrefix(syn.Error(), "json: ")
	case errors.As(err, &prob):
		offset = prob.offset
		msg = prob.msg
	case errors.Is(err, io.EOF), errors.Is(err, io.ErrUnexpectedEOF):
		offset = int64(len(text))
		msg = "the text ended before the JSON did"
		if strings.TrimSpace(text) == "" {
			msg = "there's no JSON, the text is empty"
		}
	}
	if offset > int64(len(text)) {
		offset = int64(len(text))
	}
	before := text[:offset]
	line := strings.Count(before, "\n") + 1
	col := len(before) - strings.LastIndex(before, "\n")
	if col < 1 {
		col = 1
	}
	return "at line " + strconv.Itoa(line) + ", column " + strconv.Itoa(col) + ": " + msg
}

// ---- Turtle -> JSON text ------------------------------------------------

// toJSON encodes v. indent "" is one line; "  " is indented for files.
func toJSON(who string, v object.Object, indent string) []byte {
	var buf bytes.Buffer
	writeJSON(&buf, who, v, indent, "", map[object.Object]bool{})
	return buf.Bytes()
}

func writeJSON(buf *bytes.Buffer, who string, v object.Object, indent, prefix string, seen map[object.Object]bool) {
	// A list that contains itself would never finish.
	switch v.(type) {
	case *object.List, *object.Set, *object.Map, *object.Assembly:
		if seen[v] {
			fatalf("%s: the value contains itself, so it can't be written as JSON", who)
		}
		seen[v] = true
		defer delete(seen, v)
	}
	switch x := v.(type) {
	case *object.None:
		buf.WriteString("null")
	case *object.Boolean:
		buf.WriteString(strconv.FormatBool(x.Value))
	case *object.Integer:
		buf.WriteString(x.Inspect())
	case *object.Float:
		if math.IsInf(x.Value, 0) || math.IsNaN(x.Value) {
			fatalf("%s: %s can't be written as JSON", who, x.Inspect())
		}
		buf.WriteString(x.Exact()) // every digit: data, not display
	case *object.String:
		writeJSONString(buf, x.Value)
	case *object.Date:
		writeJSONString(buf, x.Text())
	case *object.List:
		writeJSONArray(buf, who, x.Elements, indent, prefix, seen)
	case *object.Set:
		writeJSONArray(buf, who, x.Elements, indent, prefix, seen)
	case *object.Map:
		keys := make([]string, len(x.Keys))
		vals := make([]object.Object, len(x.Keys))
		for i, k := range x.Keys {
			keys[i] = jsonKey(who, x.KeyOf(k))
			vals[i] = x.Values[k]
		}
		writeJSONObject(buf, who, keys, vals, indent, prefix, seen)
	case *object.Assembly:
		writeJSONObject(buf, who, x.Shape.Fields, x.Values, indent, prefix, seen)
	default:
		fatalf("%s: a %s can't be written as JSON", who, v.Type())
	}
}

// jsonKey is a map key as JSON object key text: text as is, numbers as
// their digits. JSON has no other kind of key.
func jsonKey(who string, k object.Object) string {
	switch x := k.(type) {
	case *object.String:
		return x.Value
	case *object.Integer, *object.Float:
		return x.Inspect()
	}
	fatalf("%s: a map key that's a %s can't be a JSON key (only text and numbers can)", who, k.Type())
	return ""
}

func writeJSONString(buf *bytes.Buffer, s string) {
	enc := json.NewEncoder(buf)
	enc.SetEscapeHTML(false)    // keep <, > and & readable
	_ = enc.Encode(s)           // a string always encodes
	buf.Truncate(buf.Len() - 1) // Encode adds a newline
}

func writeJSONArray(buf *bytes.Buffer, who string, elems []object.Object, indent, prefix string, seen map[object.Object]bool) {
	if len(elems) == 0 {
		buf.WriteString("[]")
		return
	}
	inner := prefix + indent
	buf.WriteByte('[')
	for i, e := range elems {
		if i > 0 {
			buf.WriteByte(',')
		}
		newline(buf, indent, inner)
		writeJSON(buf, who, e, indent, inner, seen)
	}
	newline(buf, indent, prefix)
	buf.WriteByte(']')
}

func writeJSONObject(buf *bytes.Buffer, who string, keys []string, vals []object.Object, indent, prefix string, seen map[object.Object]bool) {
	if len(keys) == 0 {
		buf.WriteString("{}")
		return
	}
	inner := prefix + indent
	buf.WriteByte('{')
	for i, k := range keys {
		if i > 0 {
			buf.WriteByte(',')
		}
		newline(buf, indent, inner)
		writeJSONString(buf, k)
		buf.WriteByte(':')
		if indent != "" {
			buf.WriteByte(' ')
		}
		writeJSON(buf, who, vals[i], indent, inner, seen)
	}
	newline(buf, indent, prefix)
	buf.WriteByte('}')
}

// newline starts a new indented line, when indenting.
func newline(buf *bytes.Buffer, indent, prefix string) {
	if indent != "" {
		buf.WriteByte('\n')
		buf.WriteString(prefix)
	}
}

// ---- json_get -----------------------------------------------------------

// jsonGet follows keys (map keys, assembled fields) and integer indexes
// (lists) from v, one step each, and returns none as soon as a step
// doesn't exist, instead of an error.
func jsonGet(v object.Object, path []object.Object) object.Object {
	for _, step := range path {
		switch x := v.(type) {
		case *object.Map:
			next, ok := x.Get(step)
			if !ok {
				return object.NoneValue
			}
			v = next
		case *object.List:
			i, ok := step.(*object.Integer)
			if !ok || i.Value < 0 || i.Value >= int64(len(x.Elements)) {
				return object.NoneValue
			}
			v = x.Elements[i.Value]
		case *object.Assembly:
			s, ok := step.(*object.String)
			if !ok || x.Shape.Index(s.Value) < 0 {
				return object.NoneValue
			}
			v = x.Values[x.Shape.Index(s.Value)]
		default:
			return object.NoneValue
		}
	}
	return v
}
