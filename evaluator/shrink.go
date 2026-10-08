package evaluator

import (
	"math"
	"time"

	"Turtle/ast"
	"Turtle/object"
)

// Shrinking: once validate finds inputs that fail, it tries simpler ones
// (smaller numbers, shorter strings, fewer items) that still fit their
// shape, keeping any that still fail, so the report shows the smallest
// failing case it can find: nums = [ -3 ] rather than fourteen numbers.

const shrinkBudget = 2000 // most tries before settling for what it has

func (it *Interpreter) shrink(inputs []*ast.ValidateInput, vals []object.Object, env *object.Environment, fails func([]object.Object) bool) []object.Object {
	budget := shrinkBudget
	for improved := true; improved && budget > 0; {
		improved = false
		for i := range vals {
			for _, cand := range it.simpler(inputs[i].Shape, vals[i], env) {
				if budget--; budget <= 0 {
					return vals
				}
				try := append([]object.Object{}, vals...)
				try[i] = cand
				if fails(try) {
					vals, improved = try, true
					break
				}
			}
			if improved {
				break
			}
		}
	}
	return vals
}

// simpler gives values simpler than v that still fit shape s, simplest
// first.
func (it *Interpreter) simpler(s *ast.Shape, v object.Object, env *object.Environment) []object.Object {
	var out []object.Object
	add := func(c object.Object) {
		if !valuesEqual(c, v) {
			for _, o := range out {
				if valuesEqual(o, c) {
					return
				}
			}
			out = append(out, c)
		}
	}
	switch s.Kind {
	case "integer":
		n := v.(*object.Integer).Value
		lo, hi := int64(math.MinInt64), int64(math.MaxInt64)
		if s.From != nil {
			lo = it.shapeInteger(s, "from", s.From, env)
			hi = it.shapeInteger(s, "to", s.To, env)
		}
		// Toward 0 (or the nearest limit): the target, then ever closer to
		// n, as QuickCheck does: -225 tries 0, -113, -169, ..., -224.
		target := clampInt(0, lo, hi)
		for d := n - target; d != 0; d /= 2 {
			add(object.Int(n - d))
		}
		// and halving: -15 tries -7, -3, -1
		for h := n / 2; h != 0 && h >= lo && h <= hi; h /= 2 {
			add(object.Int(h))
		}
	case "float":
		f := v.(*object.Float).Value
		lo, hi := math.Inf(-1), math.Inf(1)
		if s.From != nil {
			lo = it.shapeNumber(s, "from", s.From, env)
			hi = it.shapeNumber(s, "to", s.To, env)
		}
		for _, c := range []float64{math.Max(lo, math.Min(hi, 0)), math.Trunc(f), f / 2} {
			if c >= lo && c <= hi {
				add(&object.Float{Value: c})
			}
		}
	case "string", "digits":
		str := v.(*object.String).Value
		runes := []rune(str)
		min := 1
		if s.Length != nil {
			min = int(it.shapeInteger(s, "length", s.Length, env))
		}
		first := "a"
		switch {
		case s.Kind == "digits":
			first = "0"
		case s.Chars != nil:
			if c, ok := it.evalExpression(s.Chars, env).(*object.String); ok && c.Value != "" {
				first = string([]rune(c.Value)[0])
			}
		}
		for _, n := range []int{min, len(runes) / 2, len(runes) - 1} {
			if n >= min && n < len(runes) {
				add(&object.String{Value: string(runes[:n])})
			}
		}
		if len(runes) > min {
			add(&object.String{Value: string(runes[1:])})
		}
		for i, r := range runes {
			if string(r) != first {
				c := append([]rune{}, runes...)
				c[i] = []rune(first)[0]
				add(&object.String{Value: string(c)})
				break
			}
		}
	case "digit":
		add(&object.String{Value: "0"})
	case "letter":
		add(&object.String{Value: "a"})
	case "boolean":
		add(object.Bool(false))
	case "date", "time":
		from := defaultDateFrom
		if s.From != nil {
			from = it.shapeDate(s, "from", s.From, env)
			if s.Kind == "date" {
				from = dayOf(from)
			}
		}
		add(&object.Date{Time: from.Truncate(time.Second)})
	case "list", "set":
		var elems []object.Object
		if l, ok := v.(*object.List); ok {
			elems = l.Elements
		} else {
			elems = v.(*object.Set).Elements
		}
		min := 0
		if s.Count != nil {
			min = int(it.shapeInteger(s, "count", s.Count, env))
		}
		build := func(items []object.Object) object.Object {
			if s.Kind == "list" {
				return &object.List{Elements: items}
			}
			set := &object.Set{}
			for _, e := range items {
				set.Add(e)
			}
			return set
		}
		size := func(o object.Object) int {
			if l, ok := o.(*object.List); ok {
				return len(l.Elements)
			}
			return len(o.(*object.Set).Elements)
		}
		try := func(items []object.Object) {
			c := build(items)
			if size(c) >= min && size(c) == len(items) {
				add(c)
			}
		}
		n := len(elems)
		if n > min {
			try(append([]object.Object{}, elems[:max(min, n/2)]...))
			try(append([]object.Object{}, elems[n-max(min, n/2):]...))
			for i := 0; i < n && i < 30; i++ {
				try(append(append([]object.Object{}, elems[:i]...), elems[i+1:]...))
			}
		}
		for i := 0; i < n && i < 30; i++ {
			for j, c := range it.simpler(s.Item, elems[i], env) {
				if j == 16 {
					break
				}
				items := append([]object.Object{}, elems...)
				items[i] = c
				try(items)
			}
		}
	case "map":
		m := v.(*object.Map)
		min := 0
		if s.Count != nil {
			min = int(it.shapeInteger(s, "count", s.Count, env))
		}
		keys := m.Keys
		build := func(skip int, at int, val object.Object) *object.Map {
			out := object.NewMap()
			for i, k := range keys {
				if i == skip {
					continue
				}
				x := m.Values[k]
				if i == at {
					x = val
				}
				out.Put(m.KeyOf(k), x)
			}
			return out
		}
		if len(keys) > min {
			for i := 0; i < len(keys) && i < 30; i++ {
				add(build(i, -1, nil))
			}
		}
		for i := 0; i < len(keys) && i < 30; i++ {
			for j, c := range it.simpler(s.Item, m.Values[keys[i]], env) {
				if j == 16 {
					break
				}
				add(build(-1, i, c))
			}
		}
	case "assembled":
		a := v.(*object.Assembly)
		for i, f := range s.Fields {
			for j, c := range it.simpler(f, a.Values[i], env) {
				if j == 16 {
					break
				}
				vals := append([]object.Object{}, a.Values...)
				vals[i] = c
				add(&object.Assembly{Shape: a.Shape, Values: vals})
			}
		}
	}
	return out
}

func clampInt(n, lo, hi int64) int64 {
	if n < lo {
		return lo
	}
	if n > hi {
		return hi
	}
	return n
}
