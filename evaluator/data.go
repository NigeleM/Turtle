package evaluator

import (
	"fmt"
	"math"
	"math/rand"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"Turtle/ast"
	"Turtle/object"
)

func (it *Interpreter) getVar(env *object.Environment, name string) object.Object {
	v, ok := env.Get(name)
	if !ok {
		fatalKind(kindName, "undefined variable %q", name)
	}
	return v
}

func (it *Interpreter) evalDataOp(s *ast.DataOpStatement, env *object.Environment) {
	target := it.getVar(env, s.Target)
	switch s.Kind {
	case ast.OpAdd:
		val := it.evalExpression(s.Value, env)
		switch t := target.(type) {
		case *object.List:
			t.Elements = append(t.Elements, val)
		case *object.Set:
			t.Add(val)
		default:
			fatalf("'add ... to %s' needs a list or set, got %s", s.Target, target.Type())
		}
	case ast.OpRemove:
		val := it.evalExpression(s.Value, env)
		switch t := target.(type) {
		case *object.List:
			t.Elements = removeFirst(t.Elements, val)
		case *object.Set:
			t.Elements = removeFirst(t.Elements, val)
			t.Changed()
		default:
			fatalf("'remove ... from %s' needs a list or set, got %s", s.Target, target.Type())
		}
	case ast.OpDelete:
		key := it.evalExpression(s.Value, env)
		m, ok := target.(*object.Map)
		if !ok {
			fatalf("'delete ... from %s' needs a map, got %s", s.Target, target.Type())
		}
		if !m.Delete(key) {
			fatalKind(kindKey, "key %s not found in map %s", showKey(key), s.Target)
		}
	case ast.OpSort:
		switch t := target.(type) {
		case *object.List:
			sortElements(t.Elements)
		case *object.Set:
			sortElements(t.Elements)
		default:
			fatalf("'sort %s' needs a list or set, got %s", s.Target, target.Type())
		}
	case ast.OpReverse:
		switch t := target.(type) {
		case *object.List:
			reverseElements(t.Elements)
		case *object.Set:
			reverseElements(t.Elements)
		default:
			fatalf("'reverse %s' needs a list or set, got %s", s.Target, target.Type())
		}
	case ast.OpPut:
		val := it.evalExpression(s.Value, env)
		idxObj, ok := it.evalExpression(s.Index, env).(*object.Integer)
		if !ok {
			fatalf("'put ... at ...' needs an integer index")
		}
		list, ok := target.(*object.List)
		if !ok {
			fatalf("'put ... to %s' needs a list, got %s", s.Target, target.Type())
		}
		list.Elements[listIndex("list "+s.Target, int(idxObj.Value), len(list.Elements))] = val
	case ast.OpInsert:
		val := it.evalExpression(s.Value, env)
		idxObj, ok := it.evalExpression(s.Index, env).(*object.Integer)
		if !ok {
			fatalf("'insert ... at ...' needs an integer index")
		}
		list, ok := target.(*object.List)
		if !ok {
			fatalf("'insert ... to %s' needs a list, got %s", s.Target, target.Type())
		}
		idx := int(idxObj.Value)
		if idx < 0 || idx > len(list.Elements) {
			fatalKind(kindIndex, "index %d out of range for list %s (length %d)", idx, s.Target, len(list.Elements))
		}
		list.Elements = append(list.Elements[:idx:idx], append([]object.Object{val}, list.Elements[idx:]...)...)
	}
}

func removeFirst(elems []object.Object, val object.Object) []object.Object {
	for i, e := range elems {
		if object.Equal(e, val) {
			return append(elems[:i], elems[i+1:]...)
		}
	}
	return elems
}

func reverseElements(elems []object.Object) {
	for i, j := 0, len(elems)-1; i < j; i, j = i+1, j-1 {
		elems[i], elems[j] = elems[j], elems[i]
	}
}

func sortElements(elems []object.Object) {
	sort.SliceStable(elems, func(i, j int) bool {
		fi, _, iok := numeric(elems[i])
		fj, _, jok := numeric(elems[j])
		if iok && jok {
			return fi < fj
		}
		return elems[i].Inspect() < elems[j].Inspect()
	})
}

func (it *Interpreter) lengthOf(obj object.Object) int {
	switch v := obj.(type) {
	case *object.List:
		return len(v.Elements)
	case *object.Set:
		return len(v.Elements)
	case *object.Map:
		return len(v.Keys)
	case *object.String:
		return len([]rune(v.Value))
	default:
		fatalf("'length of' needs a list, set, map, or string, got %s", obj.Type())
		return 0
	}
}

func (it *Interpreter) reduceExtreme(argExpr ast.Expression, env *object.Environment, wantMin bool) object.Object {
	obj := it.evalExpression(argExpr, env)
	var candidates []object.Object
	switch v := obj.(type) {
	case *object.List:
		candidates = v.Elements
	case *object.Set:
		candidates = v.Elements
	case *object.Map:
		for _, k := range v.Keys {
			candidates = append(candidates, v.KeyOf(k))
		}
	default:
		fatalf("'min/max of' needs a list, set, or map, got %s", obj.Type())
	}
	if len(candidates) == 0 {
		fatalKind(kindIndex, "'min/max of' called on an empty collection")
	}
	best := candidates[0]
	for _, c := range candidates[1:] {
		if better(c, best, wantMin) {
			best = c
		}
	}
	return best
}

func better(a, b object.Object, wantMin bool) bool {
	af, _, aok := numeric(a)
	bf, _, bok := numeric(b)
	if aok && bok {
		if wantMin {
			return af < bf
		}
		return af > bf
	}
	if wantMin {
		return a.Inspect() < b.Inspect()
	}
	return a.Inspect() > b.Inspect()
}

// ---- method-call form: "result is receiver at method arg, arg ." -------

func (it *Interpreter) evalMethodCall(mc *ast.MethodCallExpression, env *object.Environment) object.Object {
	receiver := it.evalExpression(mc.Receiver, env)
	args := make([]object.Object, len(mc.Arguments))
	for i, a := range mc.Arguments {
		args[i] = it.evalExpression(a, env)
	}
	return it.applyMethod(mc, receiver, args, env)
}

// oldMethodNames are the methods' names from before every name was
// lowercase (2026-10-06). They still work; the docs show the new ones.
var oldMethodNames = map[string]string{
	"isEmpty": "isempty", "isNumber": "isnumber", "getKeys": "getkeys",
	"getValues": "getvalues", "indexOf": "indexof", "toString": "tostring",
}

// applyMethod runs mc's method on values already worked out.
func (it *Interpreter) applyMethod(mc *ast.MethodCallExpression, receiver object.Object, args []object.Object, env *object.Environment) object.Object {
	if newName, ok := oldMethodNames[mc.Method]; ok {
		c := *mc
		c.Method = newName
		mc = &c
	}
	switch r := receiver.(type) {
	case *object.List:
		return listMethod(r, mc.Method, args)
	case *object.Set:
		return setMethod(r, mc.Method, args)
	case *object.Map:
		return mapMethod(r, mc.Method, args)
	case *object.String:
		return stringMethod(r, mc.Method, args)
	case *object.Integer, *object.Float:
		return it.numberMethod(r, mc.Method, args, env)
	case *object.Assembly:
		if mc.Method == "get" || mc.Method == "slice" {
			// "tags of book at get[0]": the get goes with book, not tags.
			fatalf("%s is one assembled value, so it has no %s; to take part of a field, name it first: t = tags of book, then t at %s[...]", r.Shape.Name, mc.Method, mc.Method)
		}
		fatalf("method %q needs a list, set, map, string, or number receiver, got %s", mc.Method, receiver.Type())
		return nil
	default:
		fatalf("method %q needs a list, set, map, string, or number receiver, got %s", mc.Method, receiver.Type())
		return nil
	}
}

// showKey formats a map key for an error message: strings quoted, so
// key "1" and key 1 read differently.
func showKey(k object.Object) string {
	if s, ok := k.(*object.String); ok {
		return fmt.Sprintf("%q", s.Value)
	}
	return k.Inspect()
}

func requireArgs(method string, args []object.Object, n int) {
	if len(args) != n {
		fatalf("method %q expects %d argument(s), got %d", method, n, len(args))
	}
}

// requireFuncArgs is requireArgs for library functions (find[...]), so
// the message doesn't call them methods.
func requireFuncArgs(name string, args []object.Object, n int) {
	if len(args) != n {
		fatalf("function %q expects %d argument(s), got %d", name, n, len(args))
	}
}

// listIndex checks a get index: 0 up to length-1. Unlike slice, get
// doesn't count from the end — a negative index is out of range.
func listIndex(kind string, idx, n int) int {
	if idx < 0 || idx >= n {
		fatalKind(kindIndex, "index %d out of range for %s (length %d)", idx, kind, n)
	}
	return idx
}

func asIndex(method string, obj object.Object) int {
	i, ok := obj.(*object.Integer)
	if !ok {
		fatalf("%q argument must be an integer index", method)
	}
	return int(i.Value)
}

func listMethod(l *object.List, method string, args []object.Object) object.Object {
	switch method {
	case "isempty":
		requireArgs(method, args, 0)
		return &object.Boolean{Value: len(l.Elements) == 0}
	case "add":
		requireArgs(method, args, 1)
		l.Elements = append(l.Elements, args[0])
		return l
	case "len", "length":
		return &object.Integer{Value: int64(len(l.Elements))}
	case "tostring":
		return &object.String{Value: l.Inspect()}
	case "clear":
		l.Elements = nil
		return l
	case "count":
		requireArgs(method, args, 1)
		n := 0
		for _, e := range l.Elements {
			if object.Equal(e, args[0]) {
				n++
			}
		}
		return &object.Integer{Value: int64(n)}
	case "index":
		requireArgs(method, args, 1)
		for i, e := range l.Elements {
			if object.Equal(e, args[0]) {
				return &object.Integer{Value: int64(i)}
			}
		}
		return &object.Integer{Value: -1}
	case "sort":
		sortElements(l.Elements)
		return l
	case "remove":
		requireArgs(method, args, 1)
		l.Elements = removeFirst(l.Elements, args[0])
		return l
	case "reverse":
		reverseElements(l.Elements)
		return l
	case "pop":
		if len(l.Elements) == 0 {
			fatalKind(kindIndex, "'pop' called on an empty list")
		}
		last := l.Elements[len(l.Elements)-1]
		l.Elements = l.Elements[:len(l.Elements)-1]
		return last
	case "find", "contains":
		requireArgs(method, args, 1)
		for _, e := range l.Elements {
			if object.Equal(e, args[0]) {
				return &object.Boolean{Value: true}
			}
		}
		return &object.Boolean{Value: false}
	case "insert":
		requireArgs(method, args, 2)
		idx := asIndex(method, args[1])
		if idx < 0 || idx > len(l.Elements) {
			fatalKind(kindIndex, "index %d out of range for list (length %d)", idx, len(l.Elements))
		}
		l.Elements = append(l.Elements[:idx:idx], append([]object.Object{args[0]}, l.Elements[idx:]...)...)
		return l
	case "put":
		// put[v, i]: the item at i becomes v, in insert's order (insert
		// pushes the rest along; put replaces). From 0; like get, a
		// negative index is out of range.
		requireArgs(method, args, 2)
		l.Elements[listIndex("list", asIndex(method, args[1]), len(l.Elements))] = args[0]
		return l
	case "get":
		requireArgs(method, args, 1)
		return l.Elements[listIndex("list", asIndex(method, args[0]), len(l.Elements))]
	case "slice":
		if len(args) < 1 || len(args) > 2 {
			fatalf("%q expects 1 or 2 argument(s), got %d", method, len(args))
		}
		n := len(l.Elements)
		start := normalizeSliceIndex(asIndex(method, args[0]), n)
		end := n
		if len(args) == 2 {
			end = normalizeSliceIndex(asIndex(method, args[1]), n)
		}
		if start > end {
			start = end
		}
		out := &object.List{}
		out.Elements = append(out.Elements, l.Elements[start:end]...)
		return out
	}
	fatalKind(kindName, "unknown list method %q", method)
	return nil
}

func setMethod(s *object.Set, method string, args []object.Object) object.Object {
	switch method {
	case "isempty":
		requireArgs(method, args, 0)
		return &object.Boolean{Value: len(s.Elements) == 0}
	case "add":
		requireArgs(method, args, 1)
		s.Add(args[0])
		return s
	case "len", "length":
		return &object.Integer{Value: int64(len(s.Elements))}
	case "tostring":
		return &object.String{Value: s.Inspect()}
	case "clear":
		s.Elements = nil
		s.Changed()
		return s
	case "count":
		requireArgs(method, args, 1)
		n := 0
		for _, e := range s.Elements {
			if object.Equal(e, args[0]) {
				n++
			}
		}
		return &object.Integer{Value: int64(n)}
	case "index":
		requireArgs(method, args, 1)
		for i, e := range s.Elements {
			if object.Equal(e, args[0]) {
				return &object.Integer{Value: int64(i)}
			}
		}
		return &object.Integer{Value: -1}
	case "sort":
		sortElements(s.Elements)
		return s
	case "remove":
		requireArgs(method, args, 1)
		s.Elements = removeFirst(s.Elements, args[0])
		s.Changed()
		return s
	case "reverse":
		reverseElements(s.Elements)
		return s
	case "pop":
		if len(s.Elements) == 0 {
			fatalKind(kindIndex, "'pop' called on an empty set")
		}
		last := s.Elements[len(s.Elements)-1]
		s.Elements = s.Elements[:len(s.Elements)-1]
		s.Changed()
		return last
	case "find", "contains":
		requireArgs(method, args, 1)
		return &object.Boolean{Value: s.Contains(args[0])}
	case "insert":
		requireArgs(method, args, 2)
		if s.Contains(args[0]) {
			return s
		}
		idx := asIndex(method, args[1])
		if idx < 0 || idx > len(s.Elements) {
			fatalKind(kindIndex, "index %d out of range for set (length %d)", idx, len(s.Elements))
		}
		s.Elements = append(s.Elements[:idx:idx], append([]object.Object{args[0]}, s.Elements[idx:]...)...)
		s.Changed()
		return s
	case "get":
		requireArgs(method, args, 1)
		return s.Elements[listIndex("set", asIndex(method, args[0]), len(s.Elements))]
	case "union":
		requireArgs(method, args, 1)
		other, ok := args[0].(*object.Set)
		if !ok {
			fatalf("'union' argument must be a set")
		}
		result := &object.Set{Elements: append([]object.Object{}, s.Elements...)}
		for _, e := range other.Elements {
			result.Add(e)
		}
		return result
	case "intersection":
		requireArgs(method, args, 1)
		other, ok := args[0].(*object.Set)
		if !ok {
			fatalf("'intersection' argument must be a set")
		}
		result := &object.Set{}
		for _, e := range s.Elements {
			if other.Contains(e) {
				result.Elements = append(result.Elements, e)
			}
		}
		return result
	case "difference":
		requireArgs(method, args, 1)
		other, ok := args[0].(*object.Set)
		if !ok {
			fatalf("'difference' argument must be a set")
		}
		result := &object.Set{}
		for _, e := range s.Elements {
			if !other.Contains(e) {
				result.Elements = append(result.Elements, e)
			}
		}
		return result
	case "subset":
		requireArgs(method, args, 1)
		other, ok := args[0].(*object.Set)
		if !ok {
			fatalf("'subset' argument must be a set")
		}
		for _, e := range s.Elements {
			if !other.Contains(e) {
				return &object.Boolean{Value: false}
			}
		}
		return &object.Boolean{Value: true}
	case "superset":
		requireArgs(method, args, 1)
		other, ok := args[0].(*object.Set)
		if !ok {
			fatalf("'superset' argument must be a set")
		}
		for _, e := range other.Elements {
			if !s.Contains(e) {
				return &object.Boolean{Value: false}
			}
		}
		return &object.Boolean{Value: true}
	}
	fatalKind(kindName, "unknown set method %q", method)
	return nil
}

func mapMethod(m *object.Map, method string, args []object.Object) object.Object {
	switch method {
	case "len", "length":
		requireArgs(method, args, 0)
		return &object.Integer{Value: int64(len(m.Keys))}
	case "contains":
		// Whether the map has this key.
		requireArgs(method, args, 1)
		_, ok := m.Get(args[0])
		return &object.Boolean{Value: ok}
	case "isempty":
		requireArgs(method, args, 0)
		return &object.Boolean{Value: len(m.Keys) == 0}
	case "get":
		requireArgs(method, args, 1)
		v, ok := m.Get(args[0])
		if !ok {
			fatalKind(kindKey, "key %s not found in map", showKey(args[0]))
		}
		return v
	case "getvalues":
		list := &object.List{}
		for _, k := range m.Keys {
			list.Elements = append(list.Elements, m.Values[k])
		}
		return list
	case "getkeys":
		list := &object.List{}
		for _, k := range m.Keys {
			list.Elements = append(list.Elements, m.KeyOf(k))
		}
		return list
	case "add":
		requireArgs(method, args, 2)
		m.Put(args[0], args[1])
		return m
	case "delete":
		requireArgs(method, args, 1)
		if !m.Delete(args[0]) {
			fatalKind(kindKey, "key %s not found in map", showKey(args[0]))
		}
		return m
	case "invert":
		result := object.NewMap()
		for _, k := range m.Keys {
			result.Put(m.Values[k], m.KeyOf(k))
		}
		return result
	case "tostring":
		return &object.String{Value: m.Inspect()}
	}
	fatalKind(kindName, "unknown map method %q", method)
	return nil
}

func stringMethod(s *object.String, method string, args []object.Object) object.Object {
	switch method {
	case "padleft":
		return padText(method, s, args, true)
	case "padright":
		return padText(method, s, args, false)
	case "len", "length":
		requireArgs(method, args, 0)
		return &object.Integer{Value: int64(utf8.RuneCountInString(s.Value))}
	case "isempty":
		requireArgs(method, args, 0)
		return &object.Boolean{Value: s.Value == ""}
	case "isnumber":
		requireArgs(method, args, 0)
		_, err := strconv.ParseFloat(strings.TrimSpace(s.Value), 64)
		return &object.Boolean{Value: err == nil}
	case "upper":
		requireArgs(method, args, 0)
		return &object.String{Value: strings.ToUpper(s.Value)}
	case "lower":
		requireArgs(method, args, 0)
		return &object.String{Value: strings.ToLower(s.Value)}
	case "trim":
		requireArgs(method, args, 0)
		return &object.String{Value: strings.TrimSpace(s.Value)}
	case "get":
		requireArgs(method, args, 1)
		runes := []rune(s.Value)
		return &object.String{Value: string(runes[listIndex("string", asIndex(method, args[0]), len(runes))])}
	case "slice":
		return stringSlice(s, method, args)
	case "split":
		requireArgs(method, args, 1)
		sep := asStringArg(method, args[0])
		list := &object.List{}
		for _, part := range strings.Split(s.Value, sep) {
			list.Elements = append(list.Elements, &object.String{Value: part})
		}
		return list
	case "contains":
		requireArgs(method, args, 1)
		return &object.Boolean{Value: strings.Contains(s.Value, asStringArg(method, args[0]))}
	case "indexof":
		requireArgs(method, args, 1)
		return &object.Integer{Value: int64(runeIndexOf(s.Value, asStringArg(method, args[0])))}
	case "replace":
		requireArgs(method, args, 2)
		old := asStringArg(method, args[0])
		new_ := asStringArg(method, args[1])
		return &object.String{Value: strings.ReplaceAll(s.Value, old, new_)}
	case "tostring":
		return s
	}
	fatalKind(kindName, "unknown string method %q", method)
	return nil
}

func asStringArg(method string, obj object.Object) string {
	s, ok := obj.(*object.String)
	if !ok {
		fatalf("%q argument must be a string, got %s", method, obj.Type())
	}
	return s.Value
}

// stringSlice implements "slice <start> [, <end>]", Python-style but
// without a step: negative bounds count from the end, an omitted end
// defaults to the string's length, and out-of-range bounds clamp instead
// of erroring (matching Python's slice semantics, as opposed to "get",
// which is a strict single-index lookup that's fatal out of range).
func stringSlice(s *object.String, method string, args []object.Object) object.Object {
	if len(args) < 1 || len(args) > 2 {
		fatalf("%q expects 1 or 2 argument(s), got %d", method, len(args))
	}
	runes := []rune(s.Value)
	n := len(runes)
	start := normalizeSliceIndex(asIndex(method, args[0]), n)
	end := n
	if len(args) == 2 {
		end = normalizeSliceIndex(asIndex(method, args[1]), n)
	}
	if start > end {
		start = end
	}
	return &object.String{Value: string(runes[start:end])}
}

func normalizeSliceIndex(i, n int) int {
	if i < 0 {
		i += n
	}
	if i < 0 {
		i = 0
	}
	if i > n {
		i = n
	}
	return i
}

// runeIndexOf is like strings.Index but returns a rune offset instead of
// a byte offset, so it lines up with "get"/"slice"'s rune-based indexing.
func runeIndexOf(s, sub string) int {
	byteIdx := strings.Index(s, sub)
	if byteIdx < 0 {
		return -1
	}
	return len([]rune(s[:byteIdx]))
}

// numberMethod handles method calls on an Integer or Float receiver. All
// of these require "import math" first (see requireModule) — they're a
// bundle of library functionality on top of the core +-*/% operators,
// not core syntax.
func (it *Interpreter) numberMethod(receiver object.Object, method string, args []object.Object, env *object.Environment) object.Object {
	switch method {
	case "fixed":
		return fixedText(method, receiver, args)
	case "commas":
		return commasText(method, receiver, args)
	case "sqrt":
		requireModule(env, "math", "sqrt")
		requireArgs(method, args, 0)
		f, _, _ := numeric(receiver)
		if f < 0 {
			fatalKind(kindMath, "'sqrt' of a negative number (%v)", f)
		}
		return &object.Float{Value: math.Sqrt(f)}
	case "abs":
		requireModule(env, "math", "abs")
		requireArgs(method, args, 0)
		f, isInt, _ := numeric(receiver)
		if isInt {
			return &object.Integer{Value: int64(math.Abs(f))}
		}
		return &object.Float{Value: math.Abs(f)}
	case "round":
		requireModule(env, "math", "round")
		requireArgs(method, args, 0)
		f, _, _ := numeric(receiver)
		return &object.Integer{Value: int64(math.Round(f))}
	case "floor":
		requireModule(env, "math", "floor")
		requireArgs(method, args, 0)
		f, _, _ := numeric(receiver)
		return &object.Integer{Value: int64(math.Floor(f))}
	case "ceil":
		requireModule(env, "math", "ceil")
		requireArgs(method, args, 0)
		f, _, _ := numeric(receiver)
		return &object.Integer{Value: int64(math.Ceil(f))}
	case "pow":
		requireModule(env, "math", "pow")
		requireArgs(method, args, 1)
		base, baseInt, _ := numeric(receiver)
		exp, expInt, ok := numeric(args[0])
		if !ok {
			fatalf("%q argument must be a number, got %s", method, args[0].Type())
		}
		r := math.Pow(base, exp)
		// A whole number to a non-negative whole power is a whole number:
		// 2 at pow 10 is 1024, not 1024.0 (as long as it fits an integer).
		if baseInt && expInt && exp >= 0 && math.Abs(r) < 1<<53 {
			return &object.Integer{Value: int64(r)}
		}
		return &object.Float{Value: r}
	case "random":
		requireModule(env, "math", "random")
		requireArgs(method, args, 0)
		n, ok := receiver.(*object.Integer)
		if !ok {
			fatalf("%q needs an integer receiver (the exclusive upper bound), got %s", method, receiver.Type())
		}
		if n.Value <= 0 {
			fatalKind(kindMath, "%q needs a positive upper bound, got %d", method, n.Value)
		}
		return &object.Integer{Value: rand.Int63n(n.Value)}
	}
	fatalKind(kindName, "unknown number method %q", method)
	return nil
}
