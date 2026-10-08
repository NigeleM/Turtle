package evaluator

import "Turtle/object"

// The "data" builtin module: ordinary functions (no special syntax) that
// apply a function across a collection. With the sentence-style call form
// they read as
//
//	nums process x give x + 1 .     // process[nums, x give x + 1]
//	nums keep x give x > 2 .        // keep[nums, x give x > 2]
//
// As a sentence on its own, process and keep change the collection in
// place (nums process x give x + 1 .). Used as a value (assigned, or
// inside an expression) they give a new collection and leave the
// original alone: new is nums process x give x + 1 . copy makes a copy
// on purpose: copy[x] (the outer collection), copy[x, true] (everything
// inside it too).

// dataProcess replaces every element (list/set) or every value (map) with
// f's result. A set is deduplicated afterwards. For a map, f takes the
// value, or the key and the value if it has two parameters.
func (it *Interpreter) dataProcess(args []object.Object) object.Object {
	inPlace := it.takeInPlace()
	coll, fn := collectionAndFunction("process", args)
	if !inPlace {
		coll = shallowCopy(coll)
	}
	switch c := coll.(type) {
	case *object.List:
		for i, e := range c.Elements {
			c.Elements[i] = it.callFunction(fn, "process", []object.Object{e})
		}
	case *object.Set:
		out := &object.Set{}
		for _, e := range c.Elements {
			out.Add(it.callFunction(fn, "process", []object.Object{e}))
		}
		c.Elements = out.Elements
		c.Changed()
	case *object.Map:
		for _, k := range c.Keys {
			c.Values[k] = it.callFunction(fn, "process", mapFunctionArgs("process", fn, c.KeyOf(k), c.Values[k]))
		}
	}
	return coll
}

// dataKeep keeps only the elements (list/set) or entries (map) for which
// f gives a truthy result — a filter.
func (it *Interpreter) dataKeep(args []object.Object) object.Object {
	inPlace := it.takeInPlace()
	coll, fn := collectionAndFunction("keep", args)
	if !inPlace {
		coll = shallowCopy(coll)
	}
	switch c := coll.(type) {
	case *object.List:
		c.Elements = it.keepElements(fn, c.Elements)
	case *object.Set:
		c.Elements = it.keepElements(fn, c.Elements)
		c.Changed()
	case *object.Map:
		for _, k := range append([]string{}, c.Keys...) {
			if !isTruthy(it.callFunction(fn, "keep", mapFunctionArgs("keep", fn, c.KeyOf(k), c.Values[k]))) {
				c.DeleteKey(k)
			}
		}
	}
	return coll
}

func (it *Interpreter) keepElements(fn *object.Function, elems []object.Object) []object.Object {
	var out []object.Object
	for _, e := range elems {
		if isTruthy(it.callFunction(fn, "keep", []object.Object{e})) {
			out = append(out, e)
		}
	}
	return out
}

// dataCopy is copy[x [, deep]]: a new list/set/map/assembled value. With
// deep false (the default) it holds the same items, so lists inside are
// shared; with deep true everything inside is copied too, so nothing is.
func dataCopy(args []object.Object) object.Object {
	if len(args) != 1 && len(args) != 2 {
		fatalf("'copy' expects 1 or 2 arguments (a list, set, map, or assembled value, and true to copy what's inside too), got %d", len(args))
	}
	switch args[0].(type) {
	case *object.List, *object.Set, *object.Map, *object.Assembly:
	default:
		fatalf("'copy' needs a list, set, map, or assembled value, got %s", typeName(args[0]))
	}
	if len(args) == 2 {
		deep, ok := args[1].(*object.Boolean)
		if !ok {
			fatalf("'copy' takes true (copy what's inside too) or false as its second argument, got %s", object.Shown(args[1]))
		}
		if deep.Value {
			return deepCopy(args[0])
		}
	}
	return shallowCopy(args[0])
}

// shallowCopy is a new collection holding the same items.
func shallowCopy(x object.Object) object.Object {
	switch c := x.(type) {
	case *object.List:
		return &object.List{Elements: append([]object.Object{}, c.Elements...)}
	case *object.Set:
		return &object.Set{Elements: append([]object.Object{}, c.Elements...)}
	case *object.Map:
		m := object.NewMap()
		for _, k := range c.Keys {
			m.Put(c.KeyOf(k), c.Values[k])
		}
		return m
	case *object.Assembly:
		return &object.Assembly{Shape: c.Shape, Values: append([]object.Object{}, c.Values...)}
	}
	return x
}

func collectionAndFunction(name string, args []object.Object) (object.Object, *object.Function) {
	if len(args) != 2 {
		fatalf("'%s' expects 2 arguments (a list, set, or map, and a function), got %d", name, len(args))
	}
	switch args[0].(type) {
	case *object.List, *object.Set, *object.Map:
	default:
		fatalf("'%s' needs a list, set, or map, got %s", name, typeName(args[0]))
	}
	fn, ok := args[1].(*object.Function)
	if !ok {
		fatalf("'%s' needs a function (e.g. x give x + 1), got %s", name, typeName(args[1]))
	}
	return args[0], fn
}

func mapFunctionArgs(name string, fn *object.Function, key, val object.Object) []object.Object {
	switch len(fn.Parameters) {
	case 1:
		return []object.Object{val}
	case 2:
		return []object.Object{key, val}
	}
	fatalf("'%s' over a map needs a function of the value (x give ...) or of the key and value ([k, v] give ...), got %d parameters", name, len(fn.Parameters))
	return nil
}

// collectionOp implements + and - on two lists, sets, or maps. ok is
// false for any other operand types, which keep their existing meaning.
//
//	list + list   joined, in order          list - list  left without anything in right
//	set + set     union                     set - set    difference
//	map + map     all entries; on a shared  map - map    left without right's keys
//	              key the left map wins
//
// The result is always new; neither operand changes.
func collectionOp(op string, left, right object.Object) (object.Object, bool) {
	switch l := left.(type) {
	case *object.List:
		r, ok := right.(*object.List)
		if !ok {
			return nil, false
		}
		if op == "+" {
			return &object.List{Elements: append(append([]object.Object{}, l.Elements...), r.Elements...)}, true
		}
		out := &object.List{}
		drop := &object.Set{Elements: r.Elements} // for fast membership checks
		for _, e := range l.Elements {
			if !drop.Contains(e) {
				out.Elements = append(out.Elements, e)
			}
		}
		return out, true
	case *object.Set:
		r, ok := right.(*object.Set)
		if !ok {
			return nil, false
		}
		out := &object.Set{}
		for _, e := range l.Elements {
			if op == "+" || !r.Contains(e) {
				out.Add(e)
			}
		}
		if op == "+" {
			for _, e := range r.Elements {
				out.Add(e)
			}
		}
		return out, true
	case *object.Map:
		r, ok := right.(*object.Map)
		if !ok {
			return nil, false
		}
		out := object.NewMap()
		for _, k := range l.Keys {
			if _, inRight := r.Values[k]; op == "+" || !inRight {
				out.Put(l.KeyOf(k), l.Values[k])
			}
		}
		if op == "+" {
			for _, k := range r.Keys {
				if _, inLeft := l.Values[k]; !inLeft {
					out.Put(r.KeyOf(k), r.Values[k])
				}
			}
		}
		return out, true
	}
	return nil, false
}

// maxRange is the most items range makes: past it is almost surely a
// mistake (range[1, 10000000000]).
const maxRange = 10_000_000

// dataRange is range[end], range[from, to] or range[from, to, step]: the
// whole numbers from from (0 if left out) up to, not including, to, as in
// Python; a negative step counts down.
func dataRange(args []object.Object) object.Object {
	if len(args) < 1 || len(args) > 3 {
		fatalf("'range' takes an end, or a start and an end, and optionally a step: range[5], range[1, 10], range[10, 0, -2]; got %d arguments", len(args))
	}
	num := func(i int, what string) int64 {
		n, ok := args[i].(*object.Integer)
		if !ok {
			fatalf("'range' needs whole numbers; the %s is %s", what, object.Shown(args[i]))
		}
		return n.Value
	}
	// As in Python: the end is left out. range[5] is 0 to 4.
	from, to, step := int64(0), int64(0), int64(1)
	if len(args) == 1 {
		to = num(0, "end")
	} else {
		from, to = num(0, "start"), num(1, "end")
	}
	if len(args) == 3 {
		step = num(2, "step")
		if step == 0 {
			fatalKind(kindMath, "'range' step can't be 0 (use a negative step to count down: range[10, 0, -1])")
		}
	}
	count := int64(0)
	switch {
	case step > 0 && to > from:
		count = (to - from + step - 1) / step
	case step < 0 && to < from:
		count = (from - to - step - 1) / -step
	}
	if count > maxRange {
		fatalKind(kindMath, "range[%d, %d] would be %d items; the most is %d", from, to, count, maxRange)
	}
	out := &object.List{Elements: make([]object.Object, 0, count)}
	for i, v := int64(0), from; i < count; i, v = i+1, v+step {
		out.Elements = append(out.Elements, &object.Integer{Value: v})
	}
	return out
}

// dataReduce is reduce[collection, start, [total, x] give ...]: the
// running total, from start, through every item (a map's values).
func (it *Interpreter) dataReduce(args []object.Object) object.Object {
	if len(args) != 3 {
		fatalf("'reduce' takes a collection, a starting value and a function of two names: reduce[nums, 0, [total, x] give total + x], got %d arguments", len(args))
	}
	fn, ok := args[2].(*object.Function)
	if !ok || fn.Shape != nil || len(fn.Parameters) != 2 {
		fatalf("'reduce' needs a function of two names, the total so far and the next item: [total, x] give total + x")
	}
	total := args[1]
	for _, item := range reduceItems(args[0]) {
		total = it.callFunction(fn, "reduce", []object.Object{total, item})
	}
	return total
}

func reduceItems(x object.Object) []object.Object {
	switch c := x.(type) {
	case *object.List:
		return c.Elements
	case *object.Set:
		return c.Elements
	case *object.Map:
		vals := make([]object.Object, len(c.Keys))
		for i, k := range c.Keys {
			vals[i] = c.Values[k]
		}
		return vals
	}
	fatalf("'reduce' and 'sum' need a list, set or map, got %s", typeName(x))
	return nil
}

// dataSum adds up a collection's numbers: an integer if they all are.
func dataSum(args []object.Object) object.Object {
	requireFuncArgs("sum", args, 1)
	var total object.Object = &object.Integer{Value: 0}
	for i, item := range reduceItems(args[0]) {
		if _, _, ok := numeric(item); !ok {
			fatalf("'sum' adds numbers; item %d is %s", i, object.Shown(item))
		}
		total = evalInfix("+", total, item)
	}
	return total
}
