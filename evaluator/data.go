// Copyright 2017-2026 Nigele McCoy
// Licensed under the Apache License, Version 2.0;
// see LICENSE and NOTICE.

package evaluator

import (
	"cmp"
	"fmt"
	"math"
	"math/big"
	"math/rand"
	"slices"
	"strconv"
	"strings"
	"sync"
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
	var v, idx, idx2 object.Object
	if s.Value != nil {
		v = it.evalExpression(s.Value, env)
	}
	if s.Index != nil {
		idx = it.evalExpression(s.Index, env)
	}
	if s.Index2 != nil {
		idx2 = it.evalExpression(s.Index2, env)
	}
	dataOp(s, target, v, idx, idx2)
}

// dataOp is the data statement s on target with its parts worked out:
// the value, and the index and column when s has them.
func dataOp(s *ast.DataOpStatement, target, v, idx, idx2 object.Object) {
	switch s.Kind {
	case ast.OpAdd:
		val := v
		if s.Index != nil {
			addAt(s.Target, target, idx, val)
			return
		}
		switch t := target.(type) {
		case *object.List:
			t.Elements = append(t.Elements, val)
		case *object.Set:
			t.Add(val)
		default:
			fatalf("'add ... to %s' needs a list or set, got %s", s.Target, typeName(target))
		}
	case ast.OpRemove:
		val := v
		switch t := target.(type) {
		case *object.List:
			t.Elements = removeFirst(t.Elements, val)
		case *object.Set:
			t.Elements = removeFirst(t.Elements, val)
			t.Changed()
		default:
			fatalf("'remove ... from %s' needs a list or set, got %s", s.Target, typeName(target))
		}
	case ast.OpDelete:
		key := v
		m, ok := target.(*object.Map)
		if !ok {
			fatalf("'delete ... from %s' needs a map, got %s", s.Target, typeName(target))
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
			fatalf("'sort %s' needs a list or set, got %s", s.Target, typeName(target))
		}
	case ast.OpReverse:
		switch t := target.(type) {
		case *object.List:
			reverseElements(t.Elements)
		case *object.Set:
			reverseElements(t.Elements)
		default:
			fatalf("'reverse %s' needs a list or set, got %s", s.Target, typeName(target))
		}
	case ast.OpPut:
		val := v
		if m, ok := target.(*object.Map); ok {
			if s.Index2 != nil {
				fatalf("'put ... to %s at ...' takes one key for a map: put 9 to %s at \"key\" .", s.Target, s.Target)
			}
			m.Put(idx, val)
			return
		}
		if m, ok := target.(*object.Matrix); ok {
			if s.Index2 == nil {
				fatalf("'put ... to %s' needs a row and a column for a matrix: put 9 to %s at 1, 2 .", s.Target, s.Target)
			}
			matrixMethod(m, "put", []object.Object{val, idx, idx2})
			return
		}
		if s.Index2 != nil {
			fatalf("'put ... to %s at %s' has two positions, but %s is %s, which has one: put 9 to nums at 2 .", s.Target, "row, column", s.Target, typeName(target))
		}
		idxObj, ok := idx.(*object.Integer)
		if !ok {
			fatalf("'put ... at ...' needs an integer index")
		}
		list, ok := target.(*object.List)
		if !ok {
			fatalf("'put ... to %s' needs a list, got %s", s.Target, typeName(target))
		}
		i := int(idxObj.Value)
		if i < 0 || i >= len(list.Elements) {
			listIndex("list "+s.Target, i, len(list.Elements)) // the error, worded only when needed
		}
		list.Elements[i] = val
	case ast.OpInsert:
		val := v
		idxObj, ok := idx.(*object.Integer)
		if !ok {
			fatalf("'insert ... at ...' needs an integer index")
		}
		list, ok := target.(*object.List)
		if !ok {
			fatalf("'insert ... to %s' needs a list, got %s", s.Target, typeName(target))
		}
		idx := int(idxObj.Value)
		if idx < 0 || idx > len(list.Elements) {
			fatalKind(kindIndex, "index %d out of range for list %s (length %d)", idx, s.Target, len(list.Elements))
		}
		list.Elements = append(list.Elements[:idx:idx], append([]object.Object{val}, list.Elements[idx:]...)...)
	}
}

// addAt is "add val to target at key". A map's new key starts at val
// (add 12 to ages at "Cy" gives Cy 12). An existing value changes by its
// type: a number adds up (Cy 5 becomes 17), text joins on, and a list or
// set takes the value.
func addAt(name string, target, key, val object.Object) {
	var place object.Object
	var set func(object.Object)
	switch t := target.(type) {
	case *object.Map:
		cur, ok := t.Get(key)
		if !ok { // a new key starts at the value: add 12 to ages at "Cy" gives Cy 12
			t.Put(key, val)
			return
		}
		place = cur
		set = func(v object.Object) { t.Put(key, v) }
	case *object.List:
		idx, ok := key.(*object.Integer)
		if !ok {
			fatalf("'add ... to %s at ...' needs an integer position for a list, got %s", name, typeName(key))
		}
		i := int(idx.Value)
		if i < 0 || i >= len(t.Elements) {
			listIndex("list "+name, i, len(t.Elements))
		}
		place = t.Elements[i]
		set = func(v object.Object) { t.Elements[i] = v }
	default:
		fatalf("'add ... to %s at ...' needs a map or a list, got %s", name, typeName(target))
	}
	switch p := place.(type) {
	case *object.List:
		p.Elements = append(p.Elements, val)
	case *object.Set:
		p.Add(val)
	default:
		set(evalInfix("+", place, val))
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
	// All whole numbers, or all text, the usual cases: compared exactly
	// and directly (a number past 2^53 has no exact float).
	allInts, allText := true, true
	for _, e := range elems {
		switch e.(type) {
		case *object.Integer:
			allText = false
		case *object.String:
			allInts = false
		default:
			allInts, allText = false, false
		}
		if !allInts && !allText {
			break
		}
	}
	switch {
	case len(elems) < 2:
		return
	// Equal whole numbers, or equal text, are the same value, so their
	// order can't show: the faster unstable sort does.
	case allInts:
		// The values are read once into keys, so each comparison is a plain
		// int64 compare, not a type check on two interfaces.
		keys := make([]intKey, len(elems))
		for i, e := range elems {
			keys[i] = intKey{e.(*object.Integer).Value, e}
		}
		slices.SortFunc(keys, func(a, b intKey) int { return cmp.Compare(a.v, b.v) })
		for i := range keys {
			elems[i] = keys[i].o
		}
		return
	case allText:
		keys := make([]textKey, len(elems))
		for i, e := range elems {
			keys[i] = textKey{e.(*object.String).Value, e}
		}
		slices.SortFunc(keys, func(a, b textKey) int { return strings.Compare(a.s, b.s) })
		for i := range keys {
			elems[i] = keys[i].o
		}
		return
	}
	// Mixed values compare like with like, as in Python: numbers with
	// numbers, text with text, lists element by element. Anything else
	// stops with an error rather than a guess.
	slices.SortStableFunc(elems, orderValues)
}

// orderValues orders two values for sort, element by element for lists, or
// stops with an error when they can't be compared.
func orderValues(a, b object.Object) int { return orderAt(a, b, 0) }

// orderAt is orderValues at depth levels inside lists (see
// object.MaxNesting).
func orderAt(a, b object.Object, depth int) int {
	fa, _, aNum := numeric(a)
	fb, _, bNum := numeric(b)
	ta, tb := typeName(a), typeName(b)
	switch {
	case aNum && bNum:
		return cmp.Compare(fa, fb)
	case isText(a) && isText(b):
		return strings.Compare(a.(*object.String).Value, b.(*object.String).Value)
	case isList(a) && isList(b):
		return compareLists(a.(*object.List), b.(*object.List), depth)
	case isBool(a) && isBool(b):
		return cmp.Compare(boolRank(a), boolRank(b))
	case ta == tb:
		fatalKind(kindType, "can't sort %s values with each other: sort compares numbers with numbers, text with text and lists with lists, as Python does", ta)
	}
	fatalKind(kindType, "can't sort %s and %s together: sort compares numbers with numbers, text with text and lists with lists, as Python does", ta, tb)
	return 0
}

// extremeOrder orders two values for min or max. Only values of one simple
// kind compare: numbers, text or booleans. Lists and maps don't, because a
// list of lists or a record needs a key to say what's largest, which is what
// max_sort and min_sort (import sort) are for.
func extremeOrder(verb string, a, b object.Object) int {
	fa, _, aNum := numeric(a)
	fb, _, bNum := numeric(b)
	ta, tb := typeName(a), typeName(b)
	hint := ". min and max take a list or set of numbers, text or booleans; for a list of lists, a set of sets or a record, use max_sort or min_sort from import sort, as in max_sort[rows, \"v\", \"first\"]"
	switch {
	case aNum && bNum:
		return cmp.Compare(fa, fb)
	case isText(a) && isText(b):
		return strings.Compare(a.(*object.String).Value, b.(*object.String).Value)
	case isBool(a) && isBool(b):
		return cmp.Compare(boolRank(a), boolRank(b))
	case ta == tb:
		fatalKind(kindType, "%s can't compare %s values with each other%s", verb, ta, hint)
	}
	if isList(a) || isList(b) || typeName(a) == "map" || typeName(b) == "map" {
		fatalKind(kindType, "%s can't compare %s and %s: the values must be the same kind%s", verb, ta, tb, hint)
	}
	fatalKind(kindType, "%s can't compare %s and %s: the values must be the same kind", verb, ta, tb)
	return 0
}

// compareLists orders two lists element by element, then by length, as
// Python does: [2, "c"] comes before [9, "a"].
func compareLists(a, b *object.List, depth int) int {
	if depth >= object.MaxNesting {
		object.TooDeep()
	}
	for i := 0; i < len(a.Elements) && i < len(b.Elements); i++ {
		if c := orderAt(a.Elements[i], b.Elements[i], depth+1); c != 0 {
			return c
		}
	}
	return cmp.Compare(len(a.Elements), len(b.Elements))
}

func isText(obj object.Object) bool {
	_, ok := obj.(*object.String)
	return ok
}

func isBool(obj object.Object) bool {
	_, ok := obj.(*object.Boolean)
	return ok
}

// boolRank is 0 for false and 1 for true, so false sorts first.
func boolRank(obj object.Object) int {
	if obj.(*object.Boolean).Value {
		return 1
	}
	return 0
}

func isList(obj object.Object) bool {
	_, ok := obj.(*object.List)
	return ok
}

func (it *Interpreter) lengthOf(obj object.Object) int {
	switch v := obj.(type) {
	case *object.List:
		return len(v.Elements)
	case *object.Set:
		return len(v.Elements)
	case *object.Map:
		return v.Len()
	case *object.String:
		return len([]rune(v.Value))
	default:
		fatalf("'length of' needs a list, set, map, or string, got %s", typeName(obj))
		return 0
	}
}

// extreme is min or max: "min of x" takes the collection x, and "min[a, b]"
// takes the values (min[nums] is the collection, as a single argument).
// Values compare as sort does, so mixing kinds is an error.
func (it *Interpreter) extreme(argExpr ast.Expression, argExprs []ast.Expression, env *object.Environment, wantMin bool) object.Object {
	if argExprs == nil {
		return it.reduceExtreme(argExpr, env, wantMin)
	}
	if len(argExprs) == 1 {
		return it.reduceExtreme(argExprs[0], env, wantMin)
	}
	candidates := make([]object.Object, len(argExprs))
	for i, e := range argExprs {
		candidates[i] = it.evalExpression(e, env)
	}
	return pickExtreme(candidates, wantMin)
}

func (it *Interpreter) reduceExtreme(argExpr ast.Expression, env *object.Environment, wantMin bool) object.Object {
	return extremeOf(it.evalExpression(argExpr, env), wantMin)
}

// extremeOf is min or max of a list's, set's or map's items.
func extremeOf(obj object.Object, wantMin bool) object.Object {
	switch v := obj.(type) {
	case *object.List:
		return pickExtreme(v.Elements, wantMin)
	case *object.Set:
		return pickExtreme(v.Elements, wantMin)
	case *object.Map:
		return pickExtreme(v.KeyList(), wantMin)
	}
	fatalf("'min/max of' needs a list, set, or map, got %s", typeName(obj))
	return nil
}

func pickExtreme(candidates []object.Object, wantMin bool) object.Object {
	verb := "max"
	if wantMin {
		verb = "min"
	}
	if len(candidates) == 0 {
		fatalKind(kindIndex, "'min/max' called on an empty collection")
	}
	best := candidates[0]
	for _, c := range candidates[1:] {
		if c2 := extremeOrder(verb, c, best); (wantMin && c2 < 0) || (!wantMin && c2 > 0) {
			best = c
		}
	}
	return best
}

// ---- method-call form: "result is receiver at method arg, arg ." -------

func (it *Interpreter) evalMethodCall(mc *ast.MethodCallExpression, env *object.Environment) object.Object {
	if mc.Hint != "" {
		defer func() {
			if r := recover(); r != nil {
				if fe, ok := r.(fatalError); ok && !strings.Contains(fe.text, "(hint:") {
					fe.text += " (hint: " + mc.Hint + ")"
					fe.msg = place(fe.file, fe.line) + fe.text
					panic(fe)
				}
				panic(r)
			}
		}()
	}
	receiver := it.evalExpression(mc.Receiver, env)
	// Most calls take one or two arguments: they fit in a buffer on the
	// stack, so no list of arguments is made for them.
	var buf [2]object.Object
	args := buf[:0]
	if len(mc.Arguments) > len(buf) {
		args = make([]object.Object, 0, len(mc.Arguments))
	}
	for _, a := range mc.Arguments {
		args = append(args, it.evalExpression(a, env))
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
	// Every old name has a capital letter, so a name without one skips the lookup.
	if hasUpper(mc.Method) {
		if newName, ok := oldMethodNames[mc.Method]; ok {
			c := *mc
			c.Method = newName
			mc = &c
		}
	}
	if !mc.Bracketed && len(args) == 1 {
		// One argument without brackets: if the method wanted more, say
		// how to give them ("show m at get 0, 0 ." is two things to show).
		defer func() {
			if r := recover(); r != nil {
				if fe, ok := r.(fatalError); ok && strings.Contains(fe.text, "expects") {
					hint := fmt.Sprintf(" (without brackets a method takes one argument; for more, use brackets: x at %s[a, b])", mc.Method)
					fe.text += hint
					fe.msg += hint
					panic(fe)
				}
				panic(r)
			}
		}()
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
	case *object.Integer, *object.Float, *object.BigInt:
		return it.numberMethod(r, mc.Method, args, env)
	case *object.Matrix:
		return matrixMethod(r, mc.Method, args)
	case *object.Date:
		if mc.Method == "tostring" {
			requireArgs(mc.Method, args, 0)
			return &object.String{Value: r.Inspect()} // as show writes it
		}
		fatalf("a date's only method is tostring, not %q; for more, use the time library's functions (format_date[d, pattern])", mc.Method)
		return nil
	case *object.Assembly:
		if mc.Method == "get" || mc.Method == "slice" {
			// "tags of book at get[0]": the get goes with book, not tags.
			fatalf("%s is one assembled value, so it has no %s; to take part of a field, name it first: t = tags of book, then t at %s[...]", r.Shape.Name, mc.Method, mc.Method)
		}
		fatalf("method %q needs a list, set, map, string, or number receiver, got %s", mc.Method, typeName(receiver))
		return nil
	default:
		fatalf("method %q needs a list, set, map, string, or number receiver, got %s", mc.Method, typeName(receiver))
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
		return object.Bool(len(l.Elements) == 0)
	case "add":
		requireArgs(method, args, 1)
		// A big list doubles its room when full (Go's own growth adds a
		// quarter), so a list built one add at a time is copied a few
		// times, not dozens.
		if n := len(l.Elements); n == cap(l.Elements) && n >= 1024 {
			grown := make([]object.Object, n, 2*n)
			copy(grown, l.Elements)
			l.Elements = grown
		}
		l.Elements = append(l.Elements, args[0])
		return l
	case "len", "length":
		return object.Int(int64(len(l.Elements)))
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
		return object.Int(int64(n))
	case "index":
		requireArgs(method, args, 1)
		for i, e := range l.Elements {
			if object.Equal(e, args[0]) {
				return object.Int(int64(i))
			}
		}
		return object.Int(-1)
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
		// pop removes and gives back the last item; pop[i] does the same for
		// the item at position i (pop[0] is the first).
		if len(args) > 1 {
			fatalf("'pop' takes an optional position: pop or pop[i], got %d argument(s)", len(args))
		}
		if len(l.Elements) == 0 {
			fatalKind(kindIndex, "'pop' called on an empty list")
		}
		i := len(l.Elements) - 1
		if len(args) == 1 {
			idx, ok := args[0].(*object.Integer)
			if !ok {
				fatalf("'pop' needs a whole-number position, got %s", typeName(args[0]))
			}
			i = listIndex("list", int(idx.Value), len(l.Elements))
		}
		item := l.Elements[i]
		l.Elements = append(l.Elements[:i:i], l.Elements[i+1:]...)
		return item
	case "find", "contains":
		requireArgs(method, args, 1)
		for _, e := range l.Elements {
			if object.Equal(e, args[0]) {
				return object.Bool(true)
			}
		}
		return object.Bool(false)
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
		return object.Bool(len(s.Elements) == 0)
	case "add":
		requireArgs(method, args, 1)
		s.Add(args[0])
		return s
	case "len", "length":
		return object.Int(int64(len(s.Elements)))
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
		return object.Int(int64(n))
	case "index":
		requireArgs(method, args, 1)
		for i, e := range s.Elements {
			if object.Equal(e, args[0]) {
				return object.Int(int64(i))
			}
		}
		return object.Int(-1)
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
		return object.Bool(s.Contains(args[0]))
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
				return object.Bool(false)
			}
		}
		return object.Bool(true)
	case "superset":
		requireArgs(method, args, 1)
		other, ok := args[0].(*object.Set)
		if !ok {
			fatalf("'superset' argument must be a set")
		}
		for _, e := range other.Elements {
			if !s.Contains(e) {
				return object.Bool(false)
			}
		}
		return object.Bool(true)
	}
	fatalKind(kindName, "unknown set method %q", method)
	return nil
}

// swappedPut: m at put[key, value] (or add) that looks written value first,
// in a list's order: the key's type is none of the map's key types, and the
// value is already a key, or every entry is the other way round (keys of the
// value's type, values of the key's type). A key of the usual type returns
// at once, so a correct put costs one comparison.
func swappedPut(m *object.Map, key, val object.Object) bool {
	es := m.Entries()
	if len(es) == 0 || key.Type() == es[0].Key.Type() {
		return false
	}
	kt, vt := es[0].Key.Type(), es[0].Val.Type()
	reversed := key.Type() == vt && val.Type() == kt
	valCouldBeKey := false
	for _, e := range es {
		if e.Key.Type() == key.Type() {
			return false
		}
		if e.Key.Type() == val.Type() {
			valCouldBeKey = true
		}
		if e.Key.Type() != kt || e.Val.Type() != vt {
			reversed = false
		}
	}
	if reversed {
		return true
	}
	if !valCouldBeKey {
		return false
	}
	_, isKey := m.Get(val)
	return isKey
}

// swapWarned holds the lines already warned about, so a loop warns once.
var swapWarned sync.Map

// warnIfSwapped warns, once per line, about a map put or add whose key and
// value look swapped. The call still runs as written.
func warnIfSwapped(m *object.Map, method string, key, val object.Object) {
	if !swappedPut(m, key, val) {
		return
	}
	if _, seen := swapWarned.LoadOrStore(fmt.Sprintf("%s:%d", currentFile, currentLine), true); seen {
		return
	}
	k, v := object.Shown(key), object.Shown(val)
	does := "adds the key " + k
	if method == "add" {
		does = "adds to the key " + k
	}
	stderrLine(fmt.Sprintf("warning: %s[%s, %s] looks swapped: a map's %s takes the key first, so this %s. For the key %s, write %s[%s, %s] (line %d)",
		method, k, v, method, does, v, method, v, k, currentLine))
}

func mapMethod(m *object.Map, method string, args []object.Object) object.Object {
	switch method {
	case "len", "length":
		requireArgs(method, args, 0)
		return object.Int(int64(m.Len()))
	case "contains":
		// Whether the map has this key.
		requireArgs(method, args, 1)
		_, ok := m.Get(args[0])
		return object.Bool(ok)
	case "isempty":
		requireArgs(method, args, 0)
		return object.Bool(m.Len() == 0)
	case "get":
		requireArgs(method, args, 1)
		v, ok := m.Get(args[0])
		if !ok {
			fatalKind(kindKey, "key %s not found in map", showKey(args[0]))
		}
		return v
	case "getvalues":
		list := &object.List{}
		list.Elements = m.ValueList()
		return list
	case "getkeys":
		list := &object.List{}
		list.Elements = m.KeyList()
		return list
	case "add":
		// add[key, value] is the statement "add value to m at key": a new
		// key gets the value, a number adds up, text joins on.
		requireArgs(method, args, 2)
		warnIfSwapped(m, method, args[0], args[1])
		addAt("map", m, args[0], args[1])
		return m
	case "put":
		// put[key, value] is the statement "put value to m at key": it sets
		// the key, new or not.
		requireArgs(method, args, 2)
		warnIfSwapped(m, method, args[0], args[1])
		m.Put(args[0], args[1])
		return m
	case "delete":
		requireArgs(method, args, 1)
		if !m.Delete(args[0]) {
			fatalKind(kindKey, "key %s not found in map", showKey(args[0]))
		}
		return m
	case "invert":
		// Two keys with one value can't both be keys of the result: the later
		// one wins, and a warning says which entry was dropped.
		result := object.NewMap()
		for _, e := range m.Entries() {
			if old, dup := result.Get(e.Val); dup {
				stderrLine(fmt.Sprintf("warning: %s is the value of both %s and %s, so the inverted map keeps only %s (line %d)",
					object.Shown(e.Val), object.Shown(old), object.Shown(e.Key), object.Shown(e.Key), currentLine))
			}
			result.Put(e.Val, e.Key)
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
		return object.Int(int64(utf8.RuneCountInString(s.Value)))
	case "isempty":
		requireArgs(method, args, 0)
		return object.Bool(s.Value == "")
	case "isnumber":
		requireArgs(method, args, 0)
		_, err := strconv.ParseFloat(strings.TrimSpace(s.Value), 64)
		return object.Bool(err == nil)
	case "upper":
		requireArgs(method, args, 0)
		// ß has no single capital letter, so it becomes SS, as Python does.
		return &object.String{Value: strings.ReplaceAll(strings.ToUpper(s.Value), "ß", "SS")}
	case "lower":
		requireArgs(method, args, 0)
		return &object.String{Value: strings.ToLower(s.Value)}
	case "reverse":
		requireArgs(method, args, 0)
		runes := []rune(s.Value)
		for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
			runes[i], runes[j] = runes[j], runes[i]
		}
		return &object.String{Value: string(runes)}
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
		return object.Bool(strings.Contains(s.Value, asStringArg(method, args[0])))
	case "indexof":
		requireArgs(method, args, 1)
		return object.Int(int64(runeIndexOf(s.Value, asStringArg(method, args[0]))))
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
		fatalf("%q argument must be a string, got %s", method, typeName(obj))
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
// mathOf is a math method of one number with no argument: sin, cos, exp.
func mathOf(env *object.Environment, method string, receiver object.Object, args []object.Object, f func(float64) float64) object.Object {
	requireModule(env, "math", method)
	requireArgs(method, args, 0)
	x, _, _ := numeric(receiver)
	if math.IsInf(x, 0) && isBigInt(receiver) {
		pastFloat(method)
	}
	return &object.Float{Value: f(x)}
}

// pastFloat stops a float function given a whole number too big for a
// float (past about 1.8e308), rather than answer inf or nan for it.
func pastFloat(method string) {
	fatalKind(kindMath, "'%s' works in floats, and this whole number is too big for one (more than about 1.8e308)", method)
}

func (it *Interpreter) numberMethod(receiver object.Object, method string, args []object.Object, env *object.Environment) object.Object {
	switch method {
	case "fixed":
		return fixedText(method, receiver, args)
	case "commas":
		return commasText(method, receiver, args)
	case "is_close":
		requireModule(env, "math", "is_close")
		if len(args) < 1 || len(args) > 2 {
			fatalf("'is_close' takes a number and an optional tolerance: x at is_close[y] or x at is_close[y, 0.001], got %d argument(s)", len(args))
		}
		a, _, _ := numeric(receiver)
		b, _, ok := numeric(args[0])
		if !ok {
			fatalf("'is_close' needs a number to compare with, got %s", typeName(args[0]))
		}
		return object.Bool(closeEnough(a, b, closeTolerance("is_close", args[1:])))
	case "sin":
		return mathOf(env, method, receiver, args, math.Sin)
	case "cos":
		return mathOf(env, method, receiver, args, math.Cos)
	case "tan":
		return mathOf(env, method, receiver, args, math.Tan)
	case "asin", "acos":
		// Only -1 to 1 is the sine or cosine of an angle; past that there's
		// no answer, as sqrt of a negative has none.
		if x, _, _ := numeric(receiver); x < -1 || x > 1 {
			requireModule(env, "math", method)
			fatalKind(kindMath, "'%s' needs a number from -1 to 1, got %v", method, x)
		}
		if method == "asin" {
			return mathOf(env, method, receiver, args, math.Asin)
		}
		return mathOf(env, method, receiver, args, math.Acos)
	case "atan":
		return mathOf(env, method, receiver, args, math.Atan)
	case "atan2":
		// y at atan2[x] is the angle of the point (x, y), as Python's
		// math.atan2(y, x): -pi to pi, right for every quadrant.
		requireModule(env, "math", method)
		requireArgs(method, args, 1)
		y, _, _ := numeric(receiver)
		x, _, ok := numeric(args[0])
		if !ok {
			fatalf("'atan2' needs a number for x, as y at atan2[x], got %s", typeName(args[0]))
		}
		return &object.Float{Value: math.Atan2(y, x)}
	case "exp":
		r := mathOf(env, method, receiver, args, math.Exp)
		if math.IsInf(r.(*object.Float).Value, 0) {
			fatalKind(kindMath, "'exp' is too big for a float (the result is more than about 1.8e308)")
		}
		return r
	case "log":
		requireModule(env, "math", "log")
		if len(args) > 1 {
			fatalf("'log' takes an optional base: x at log or x at log[10], got %d argument(s)", len(args))
		}
		f, _, _ := numeric(receiver)
		if f <= 0 {
			fatalKind(kindMath, "'log' needs a number above 0, got %v", f)
		}
		// A whole number past a float's range: its log from its bits, as
		// x = m * 2^e (m from 0.5 to 1) is log m + e log 2.
		var ln, log2, log10 float64
		huge := math.IsInf(f, 0)
		if huge {
			m := new(big.Float)
			e := new(big.Float).SetInt(receiver.(*object.BigInt).V).MantExp(m)
			mf, _ := m.Float64()
			// Each product rounded on its own (float64(...)), so no processor
			// fuses it into the add and every one gives the same last digit.
			ln, log2, log10 = math.Log(mf)+float64(float64(e)*math.Ln2), math.Log2(mf)+float64(e), math.Log10(mf)+float64(float64(e)*math.Log10(2))
		}
		if len(args) == 0 {
			if huge {
				return &object.Float{Value: ln}
			}
			return &object.Float{Value: math.Log(f)}
		}
		base, _, _ := numeric(args[0])
		if base <= 0 || base == 1 {
			fatalKind(kindMath, "'log' needs a base above 0 and not 1, got %v", base)
		}
		if huge {
			switch base {
			case 10:
				return &object.Float{Value: log10}
			case 2:
				return &object.Float{Value: log2}
			}
			return &object.Float{Value: ln / math.Log(base)}
		}
		// Base 10 and 2 use the exact functions, so 1000 at log[10] is 3.0.
		switch base {
		case 10:
			return &object.Float{Value: math.Log10(f)}
		case 2:
			return &object.Float{Value: math.Log2(f)}
		}
		return &object.Float{Value: math.Log(f) / math.Log(base)}
	case "sqrt":
		requireModule(env, "math", "sqrt")
		requireArgs(method, args, 0)
		f, _, _ := numeric(receiver)
		if f < 0 {
			fatalKind(kindMath, "'sqrt' of a negative number (%v)", f)
		}
		if math.IsInf(f, 0) && isBigInt(receiver) {
			// Too big for a float, its root may not be: 2^2000 has 2^1000.
			x := new(big.Float).SetPrec(64).SetInt(receiver.(*object.BigInt).V)
			r, _ := new(big.Float).Sqrt(x).Float64()
			if math.IsInf(r, 0) {
				pastFloat(method)
			}
			return &object.Float{Value: r}
		}
		return &object.Float{Value: math.Sqrt(f)}
	case "abs":
		requireModule(env, "math", "abs")
		requireArgs(method, args, 0)
		if bb, isBig := receiver.(*object.BigInt); isBig {
			return object.IntOf(new(big.Int).Abs(bb.V))
		}
		f, isInt, _ := numeric(receiver)
		if isInt {
			v := receiver.(*object.Integer).Value
			if v == math.MinInt64 {
				return object.IntOf(new(big.Int).Neg(big.NewInt(v)))
			}
			if v < 0 {
				return object.Int(-v)
			}
			return object.Int(v)
		}
		return &object.Float{Value: math.Abs(f)}
	case "round":
		// round: to a whole number. round[2]: to 2 places, still a
		// number (3.14159 -> 3.14); round[-2]: to hundreds (1234 -> 1200).
		requireModule(env, "math", "round")
		if len(args) > 1 {
			fatalf("'round' takes nothing, or how many places (round[2]), got %d arguments", len(args))
		}
		bb, isBig := receiver.(*object.BigInt)
		if isBig && len(args) == 0 {
			return receiver
		}
		f, _, _ := numeric(receiver)
		if len(args) == 0 {
			return wholeOf("round", math.Round(f))
		}
		p, ok := args[0].(*object.Integer)
		if !ok || p.Value < -15 || p.Value > 15 {
			fatalf("'round' takes a whole number of places from -15 to 15 (round[2]), got %s", args[0].Inspect())
		}
		places := int(p.Value)
		if isBig {
			return roundBig(bb.V, places)
		}
		if places <= 0 {
			scale := math.Pow(10, float64(-places))
			return wholeOf("round", math.Round(f/scale)*scale)
		}
		if _, isInt := receiver.(*object.Integer); isInt {
			return receiver // a whole number already has every place
		}
		// Through the decimal text, so the answer is the number nearest the
		// rounded digits (0.6000000000000001 -> 0.6). A value stored a hair
		// below a half (2.675 is 2.67499...) rounds down, as in Python.
		r, _ := strconv.ParseFloat(strconv.FormatFloat(f, 'f', places, 64), 64)
		return &object.Float{Value: r}
	case "floor":
		requireModule(env, "math", "floor")
		requireArgs(method, args, 0)
		if isBigInt(receiver) {
			return receiver
		}
		f, _, _ := numeric(receiver)
		return wholeOf("floor", math.Floor(f))
	case "ceil":
		requireModule(env, "math", "ceil")
		requireArgs(method, args, 0)
		if isBigInt(receiver) {
			return receiver
		}
		f, _, _ := numeric(receiver)
		return wholeOf("ceil", math.Ceil(f))
	case "pow":
		requireModule(env, "math", "pow")
		requireArgs(method, args, 1)
		base, baseInt, _ := numeric(receiver)
		exp, expInt, ok := numeric(args[0])
		if !ok {
			fatalf("%q argument must be a number, got %s", method, typeName(args[0]))
		}
		// A whole number to a non-negative whole power is a whole number,
		// computed exactly, however big: 2 at pow 10 is 1024, 3 at pow 40 is
		// 12157665459056928801.
		// A whole power too big to be an integer exponent at all (past the
		// limit for any base of 2 or more) stops before it's converted.
		if (baseInt || isBigInt(receiver)) && expInt && exp > maxIntBits && math.Abs(base) >= 2 {
			tooBig()
		}
		if baseInt && expInt && exp >= 0 {
			b := receiver.(*object.Integer).Value
			if r, fits := powInt(b, int64(exp)); fits {
				return object.Int(r)
			}
			return powBig(big.NewInt(b), int64(exp))
		}
		if bb, isBig := receiver.(*object.BigInt); isBig && expInt && exp >= 0 {
			return powBig(bb.V, int64(exp))
		}
		if base == 0 && exp < 0 {
			fatalKind(kindMath, "zero can't be raised to a negative power")
		}
		return floatChecked("pow", math.Pow(base, exp))
	case "random":
		requireModule(env, "math", "random")
		requireArgs(method, args, 0)
		n, ok := receiver.(*object.Integer)
		if isBigInt(receiver) {
			fatalf("%q needs an upper bound below 9223372036854775808, got %s", method, receiver.Inspect())
		}
		if !ok {
			fatalf("%q needs an integer receiver (the exclusive upper bound), got %s", method, typeName(receiver))
		}
		if n.Value <= 0 {
			fatalKind(kindMath, "%q needs a positive upper bound, got %d", method, n.Value)
		}
		return object.Int(rand.Int63n(n.Value))
	}
	if textOnlyMethod(method) {
		fatalKind(kindName, "%q works on text, not on a number: change the number to text first, as x to string, then x at %s[...]", method, method)
	}
	fatalKind(kindName, "unknown number method %q", method)
	return nil
}

// hasUpper is whether name has a capital letter.
func hasUpper(name string) bool {
	for i := 0; i < len(name); i++ {
		if c := name[i]; c >= 'A' && c <= 'Z' {
			return true
		}
	}
	return false
}

// textOnlyMethod is a method of text that a number doesn't have, so a call on
// a number can say to change it to text first.
func textOnlyMethod(method string) bool {
	switch method {
	case "padleft", "padright", "upper", "lower", "trim", "split", "replace", "substring":
		return true
	}
	return false
}

// powInt is b to the power e (e >= 0), by repeated squaring. It reports
// whether the answer fits an int64.
func powInt(b, e int64) (r int64, fits bool) {
	result := int64(1)
	for {
		if e&1 == 1 {
			var ok bool
			if result, ok = mulFits(result, b); !ok {
				return 0, false
			}
		}
		e >>= 1
		if e == 0 {
			return result, true
		}
		var ok bool
		if b, ok = mulFits(b, b); !ok {
			return 0, false
		}
	}
}

// mulFits is a * b, and whether it fits an integer.
func mulFits(a, b int64) (int64, bool) {
	if a == 0 || b == 0 {
		return 0, true
	}
	r := a * b
	if r/b != a || (a == -1 && b == math.MinInt64) || (b == -1 && a == math.MinInt64) {
		return 0, false
	}
	return r, true
}

// intKey and textKey are a sort element's value beside the object it came from.
type intKey struct {
	v int64
	o object.Object
}

type textKey struct {
	s string
	o object.Object
}
