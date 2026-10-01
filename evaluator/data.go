package evaluator

import (
	"math"
	"math/rand"
	"sort"
	"strconv"
	"strings"

	"Turtle/ast"
	"Turtle/object"
)

func (it *Interpreter) getVar(env *object.Environment, name string) object.Object {
	v, ok := env.Get(name)
	if !ok {
		fatalf("undefined variable %q", name)
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
			if !t.Contains(val) {
				t.Elements = append(t.Elements, val)
			}
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
		default:
			fatalf("'remove ... from %s' needs a list or set, got %s", s.Target, target.Type())
		}
	case ast.OpDelete:
		key := object.Key(it.evalExpression(s.Value, env))
		m, ok := target.(*object.Map)
		if !ok {
			fatalf("'delete ... from %s' needs a map, got %s", s.Target, target.Type())
		}
		if !m.Delete(key) {
			fatalf("key %q not found in map %s", key, s.Target)
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
			fatalf("index %d out of range for list %s (length %d)", idx, s.Target, len(list.Elements))
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
			candidates = append(candidates, &object.String{Value: k})
		}
	default:
		fatalf("'min/max of' needs a list, set, or map, got %s", obj.Type())
	}
	if len(candidates) == 0 {
		fatalf("'min/max of' called on an empty collection")
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
	default:
		fatalf("method %q needs a list, set, map, string, or number receiver, got %s", mc.Method, receiver.Type())
		return nil
	}
}

func requireArgs(method string, args []object.Object, n int) {
	if len(args) != n {
		fatalf("method %q expects %d argument(s), got %d", method, n, len(args))
	}
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
	case "add":
		requireArgs(method, args, 1)
		l.Elements = append(l.Elements, args[0])
		return l
	case "len", "length":
		return &object.Integer{Value: int64(len(l.Elements))}
	case "toString":
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
			fatalf("'pop' called on an empty list")
		}
		last := l.Elements[len(l.Elements)-1]
		l.Elements = l.Elements[:len(l.Elements)-1]
		return last
	case "find":
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
			fatalf("index %d out of range for list (length %d)", idx, len(l.Elements))
		}
		l.Elements = append(l.Elements[:idx:idx], append([]object.Object{args[0]}, l.Elements[idx:]...)...)
		return l
	case "get":
		requireArgs(method, args, 1)
		idx := asIndex(method, args[0])
		if idx < 0 || idx >= len(l.Elements) {
			fatalf("index %d out of range for list (length %d)", idx, len(l.Elements))
		}
		return l.Elements[idx]
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
	fatalf("unknown list method %q", method)
	return nil
}

func setMethod(s *object.Set, method string, args []object.Object) object.Object {
	switch method {
	case "add":
		requireArgs(method, args, 1)
		if !s.Contains(args[0]) {
			s.Elements = append(s.Elements, args[0])
		}
		return s
	case "len", "length":
		return &object.Integer{Value: int64(len(s.Elements))}
	case "toString":
		return &object.String{Value: s.Inspect()}
	case "clear":
		s.Elements = nil
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
		return s
	case "reverse":
		reverseElements(s.Elements)
		return s
	case "pop":
		if len(s.Elements) == 0 {
			fatalf("'pop' called on an empty set")
		}
		last := s.Elements[len(s.Elements)-1]
		s.Elements = s.Elements[:len(s.Elements)-1]
		return last
	case "find":
		requireArgs(method, args, 1)
		return &object.Boolean{Value: s.Contains(args[0])}
	case "insert":
		requireArgs(method, args, 2)
		if s.Contains(args[0]) {
			return s
		}
		idx := asIndex(method, args[1])
		if idx < 0 || idx > len(s.Elements) {
			fatalf("index %d out of range for set (length %d)", idx, len(s.Elements))
		}
		s.Elements = append(s.Elements[:idx:idx], append([]object.Object{args[0]}, s.Elements[idx:]...)...)
		return s
	case "get":
		requireArgs(method, args, 1)
		idx := asIndex(method, args[0])
		if idx < 0 || idx >= len(s.Elements) {
			fatalf("index %d out of range for set (length %d)", idx, len(s.Elements))
		}
		return s.Elements[idx]
	case "union":
		requireArgs(method, args, 1)
		other, ok := args[0].(*object.Set)
		if !ok {
			fatalf("'union' argument must be a set")
		}
		result := &object.Set{Elements: append([]object.Object{}, s.Elements...)}
		for _, e := range other.Elements {
			if !result.Contains(e) {
				result.Elements = append(result.Elements, e)
			}
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
	fatalf("unknown set method %q", method)
	return nil
}

func mapMethod(m *object.Map, method string, args []object.Object) object.Object {
	switch method {
	case "get":
		requireArgs(method, args, 1)
		key := object.Key(args[0])
		v, ok := m.Values[key]
		if !ok {
			fatalf("key %q not found in map", key)
		}
		return v
	case "getValues":
		list := &object.List{}
		for _, k := range m.Keys {
			list.Elements = append(list.Elements, m.Values[k])
		}
		return list
	case "getKeys":
		list := &object.List{}
		for _, k := range m.Keys {
			list.Elements = append(list.Elements, &object.String{Value: k})
		}
		return list
	case "add":
		requireArgs(method, args, 2)
		m.Set(object.Key(args[0]), args[1])
		return m
	case "delete":
		requireArgs(method, args, 1)
		key := object.Key(args[0])
		if !m.Delete(key) {
			fatalf("key %q not found in map", key)
		}
		return m
	case "invert":
		result := object.NewMap()
		for _, k := range m.Keys {
			result.Set(object.Key(m.Values[k]), &object.String{Value: k})
		}
		return result
	case "toString":
		return &object.String{Value: m.Inspect()}
	}
	fatalf("unknown map method %q", method)
	return nil
}

func stringMethod(s *object.String, method string, args []object.Object) object.Object {
	switch method {
	case "isNumber":
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
		i := asIndex(method, args[0])
		if i < 0 || i >= len(runes) {
			fatalf("%q index %d out of range (length %d)", method, i, len(runes))
		}
		return &object.String{Value: string(runes[i])}
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
	case "indexOf":
		requireArgs(method, args, 1)
		return &object.Integer{Value: int64(runeIndexOf(s.Value, asStringArg(method, args[0])))}
	case "replace":
		requireArgs(method, args, 2)
		old := asStringArg(method, args[0])
		new_ := asStringArg(method, args[1])
		return &object.String{Value: strings.ReplaceAll(s.Value, old, new_)}
	case "toString":
		return s
	}
	fatalf("unknown string method %q", method)
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
	case "sqrt":
		requireModule(env, "math", "sqrt")
		requireArgs(method, args, 0)
		f, _, _ := numeric(receiver)
		if f < 0 {
			fatalf("'sqrt' of a negative number (%v)", f)
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
		base, _, _ := numeric(receiver)
		exp, _, ok := numeric(args[0])
		if !ok {
			fatalf("%q argument must be a number, got %s", method, args[0].Type())
		}
		return &object.Float{Value: math.Pow(base, exp)}
	case "random":
		requireModule(env, "math", "random")
		requireArgs(method, args, 0)
		n, ok := receiver.(*object.Integer)
		if !ok {
			fatalf("%q needs an integer receiver (the exclusive upper bound), got %s", method, receiver.Type())
		}
		if n.Value <= 0 {
			fatalf("%q needs a positive upper bound, got %d", method, n.Value)
		}
		return &object.Integer{Value: rand.Int63n(n.Value)}
	}
	fatalf("unknown number method %q", method)
	return nil
}
