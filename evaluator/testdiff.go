// Copyright 2017-2026 Nigele McCoy
// Licensed under the Apache License, Version 2.0;
// see LICENSE and NOTICE.

package evaluator

import (
	"fmt"
	"math"
	"strings"
	"unicode/utf8"

	"Turtle/object"
)

// How two values differ, for a failed check a == b: what was got (the
// left side) against what was wanted (the right side), down to the first
// item, key, field or character that differs.

const maxDiffLines = 8

// diffLines describes how got differs from want.
func diffLines(got, want object.Object) []string {
	var out []string
	describeDiff("", got, want, &out)
	if len(out) == 0 {
		out = append(out, fmt.Sprintf("got %s, want %s", object.ShownExact(got), object.ShownExact(want)))
	}
	if len(out) > maxDiffLines {
		out = append(out[:maxDiffLines], "...")
	}
	return out
}

// valueKind is a value's kind in words: "an integer", "a list", "an Order".
func valueKind(v object.Object) string {
	switch x := v.(type) {
	case *object.Integer:
		return "an integer"
	case *object.Float:
		return "a float"
	case *object.String:
		return "a string"
	case *object.Boolean:
		return "a boolean"
	case *object.List:
		return "a list"
	case *object.Set:
		return "a set"
	case *object.Map:
		return "a map"
	case *object.Date:
		return "a date"
	case *object.None:
		return "none"
	case *object.Function:
		return "a function"
	case *object.Assembly:
		name := x.Shape.Name
		if strings.ContainsRune("AEIOUaeiou", rune(name[0])) {
			return "an " + name
		}
		return "a " + name
	}
	return "a " + strings.ToLower(string(v.Type()))
}

func at(path string) string {
	if path == "" {
		return ""
	}
	return path + ": "
}

func describeDiff(path string, got, want object.Object, out *[]string) {
	if len(*out) > maxDiffLines || valuesEqual(got, want) {
		return
	}
	add := func(format string, args ...any) {
		*out = append(*out, at(path)+fmt.Sprintf(format, args...))
	}
	gf, gInt, gNum := numeric(got)
	wf, wInt, wNum := numeric(want)
	if gNum && wNum {
		d := gf - wf
		how := "more"
		if d < 0 {
			how, d = "less", -d
		}
		by := (&object.Float{Value: d}).Inspect()
		if gInt && wInt {
			gi, wi := got.(*object.Integer).Value, want.(*object.Integer).Value
			if gi >= wi {
				by = fmt.Sprint(uint64(gi - wi))
			} else {
				by = fmt.Sprint(uint64(wi - gi))
			}
		}
		add("got %s, want %s (%s %s)", object.Exact(got), object.Exact(want), by, how)
		return
	}
	if valueKind(got) != valueKind(want) {
		add("got %s (%s), want %s (%s)", object.Shown(got), valueKind(got), object.Shown(want), valueKind(want))
		return
	}
	switch g := got.(type) {
	case *object.String:
		w := want.(*object.String)
		add("got %s, want %s", object.ShownExact(g), object.ShownExact(w))
		gr, wr := []rune(g.Value), []rune(w.Value)
		for i := 0; i < len(gr) || i < len(wr); i++ {
			switch {
			case i >= len(gr):
				*out = append(*out, fmt.Sprintf("  got is shorter: it ends where want goes on with %q", string(wr[i:])))
			case i >= len(wr):
				*out = append(*out, fmt.Sprintf("  got is longer: %q is extra", string(gr[i:])))
			case gr[i] != wr[i]:
				*out = append(*out, fmt.Sprintf("  first difference at position %d: %q, want %q", i, string(gr[i]), string(wr[i])))
			default:
				continue
			}
			return
		}
	case *object.List:
		w := want.(*object.List)
		diffSequence(path, g.Elements, w.Elements, out)
	case *object.Set:
		w := want.(*object.Set)
		var missing, extra []string
		for _, e := range w.Elements {
			if !g.Contains(e) {
				missing = append(missing, object.Shown(e))
			}
		}
		for _, e := range g.Elements {
			if !w.Contains(e) {
				extra = append(extra, object.Shown(e))
			}
		}
		if len(missing) > 0 {
			add("missing %s", strings.Join(missing, ", "))
		}
		if len(extra) > 0 {
			add("not wanted: %s", strings.Join(extra, ", "))
		}
	case *object.Map:
		w := want.(*object.Map)
		for _, we := range w.Entries() {
			key := we.Key
			gv, ok := g.Get(key)
			sub := path + "key " + object.Shown(key)
			if path != "" {
				sub = path + ", key " + object.Shown(key)
			}
			if !ok {
				*out = append(*out, fmt.Sprintf("%smissing (want %s)", at(sub), object.Shown(we.Val)))
				continue
			}
			describeDiff(sub, gv, we.Val, out)
		}
		for _, ge := range g.Entries() {
			key := ge.Key
			if _, ok := w.Get(key); !ok {
				sub := "key " + object.Shown(key)
				if path != "" {
					sub = path + ", " + sub
				}
				*out = append(*out, fmt.Sprintf("%snot wanted (got %s)", at(sub), object.Shown(ge.Val)))
			}
		}
	case *object.Assembly:
		w := want.(*object.Assembly)
		for i, f := range g.Shape.Fields {
			sub := "field " + f
			if path != "" {
				sub = path + ", " + sub
			}
			if i < len(w.Values) {
				describeDiff(sub, g.Values[i], w.Values[i], out)
			}
		}
	default:
		add("got %s, want %s", object.ShownExact(got), object.ShownExact(want))
	}
}

// diffSequence compares two lists item by item.
func diffSequence(path string, got, want []object.Object, out *[]string) {
	if len(got) != len(want) {
		*out = append(*out, fmt.Sprintf("%sgot %d %s, want %d", at(path), len(got), plural(len(got), "item"), len(want)))
	}
	if len(got) == len(want) && sameItems(got, want) {
		*out = append(*out, fmt.Sprintf("%sthe same items, in a different order", at(path)))
		*out = append(*out, fmt.Sprintf("%sgot  %s", at(path), object.Shown(&object.List{Elements: got})))
		*out = append(*out, fmt.Sprintf("%swant %s", at(path), object.Shown(&object.List{Elements: want})))
		return
	}
	shown := 0
	for i := 0; i < len(got) || i < len(want); i++ {
		if shown >= 5 || len(*out) > maxDiffLines {
			*out = append(*out, at(path)+"...")
			return
		}
		sub := fmt.Sprintf("at %d", i)
		if path != "" {
			sub = path + ", " + sub
		}
		switch {
		case i >= len(got):
			*out = append(*out, fmt.Sprintf("%smissing (want %s)", at(sub), object.Shown(want[i])))
		case i >= len(want):
			*out = append(*out, fmt.Sprintf("%snot wanted (got %s)", at(sub), object.Shown(got[i])))
		case valuesEqual(got[i], want[i]):
			continue
		default:
			describeDiff(sub, got[i], want[i], out)
		}
		shown++
	}
}

// sameItems: the two hold the same values, each as many times.
func sameItems(a, b []object.Object) bool {
	used := make([]bool, len(b))
	for _, x := range a {
		found := false
		for j, y := range b {
			if !used[j] && valuesEqual(x, y) {
				used[j], found = true, true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func plural(n int, word string) string {
	if n == 1 {
		return word
	}
	return word + "s"
}

// closeEnough: the two numbers differ by no more than tol, or (tol < 0)
// by a tiny amount relative to their size, for decimals that can't be
// exact (0.1 + 0.2 is 0.30000000000000004).
func closeEnough(a, b, tol float64) bool {
	d := math.Abs(a - b)
	if tol >= 0 {
		return d <= tol
	}
	return d <= 1e-12 || d <= 1e-9*math.Max(math.Abs(a), math.Abs(b))
}

// lengthOf counts a string's characters or a collection's items, for
// "is empty".
func lengthOf(v object.Object) (int, bool) {
	switch x := v.(type) {
	case *object.String:
		return utf8.RuneCountInString(x.Value), true
	case *object.List:
		return len(x.Elements), true
	case *object.Set:
		return len(x.Elements), true
	case *object.Map:
		return x.Len(), true
	}
	return 0, false
}
