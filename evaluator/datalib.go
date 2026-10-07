package evaluator

import "Turtle/object"

// The "data" builtin module: ordinary functions (no special syntax) that
// apply a function across a collection. With the sentence-style call form
// they read as
//
//	nums process x give x + 1 .     // process[nums, x give x + 1]
//	nums keep x give x > 2 .        // keep[nums, x give x > 2]
//
// process and keep change the collection in place and also return it;
// copy makes a new one first when the original should stay as it was.

// dataProcess replaces every element (list/set) or every value (map) with
// f's result. A set is deduplicated afterwards. For a map, f takes the
// value, or the key and the value if it has two parameters.
func (it *Interpreter) dataProcess(args []object.Object) object.Object {
	coll, fn := collectionAndFunction("process", args)
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
	coll, fn := collectionAndFunction("keep", args)
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

// dataCopy returns a new list/set/map holding the same elements, so
// "big = copy[nums]" then "big process x give x * 10 ." leaves nums alone.
func dataCopy(args []object.Object) object.Object {
	if len(args) != 1 {
		fatalf("'copy' expects 1 argument (a list, set, map, or assembled value), got %d", len(args))
	}
	switch c := args[0].(type) {
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
	fatalf("'copy' needs a list, set, map, or assembled value, got %s", args[0].Type())
	return nil
}

func collectionAndFunction(name string, args []object.Object) (object.Object, *object.Function) {
	if len(args) != 2 {
		fatalf("'%s' expects 2 arguments (a list, set, or map, and a function), got %d", name, len(args))
	}
	switch args[0].(type) {
	case *object.List, *object.Set, *object.Map:
	default:
		fatalf("'%s' needs a list, set, or map, got %s", name, args[0].Type())
	}
	fn, ok := args[1].(*object.Function)
	if !ok {
		fatalf("'%s' needs a function (e.g. x give x + 1), got %s", name, args[1].Type())
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
