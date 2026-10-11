// Copyright 2017-2026 Nigele McCoy
// Licensed under the Apache License, Version 2.0;
// see LICENSE and NOTICE.

package evaluator

import (
	"Turtle/object"
)

// The "search" builtin module: finding things in collections. Like the
// sort module, it works through a list's or set's items, a text's
// characters, or a map's entries, and an optional key says what to
// compare (for a map, picked from each value). Over a map, find_first
// and find_last give the matching key, find_all a map of the matches.
//
//	find_first[books, b give b at get["price"] < 1000]   the first match, or none
//	find_all[books, b give b at get["stock"] == 0]       every match, a list
//	find_key[ages, 30]                                    the key holding 30, or none
//	binary_search[sorted, 950, "price"]                   where 950 is, or -1
//
// The fast searches (binary, jump, exponential, interpolation, ternary)
// need the collection sorted smallest first by the same key, e.g. by
// min_sort. Every search gives the position of the first match, so they
// all give the same answer, or -1.

func (it *Interpreter) callSearch(name string, args []object.Object) object.Object {
	switch name {
	case "find_first", "find_last", "find_all", "find_index", "count_where":
		requireFuncArgs(name, args, 2)
		items, isMap := entriesOf(name, args[0])
		test, ok := args[1].(*object.Function)
		if !ok {
			fatalf("'%s' needs a function that says yes or no (x give x > 3), got %s", name, typeName(args[1]))
		}
		match := func(e entry) bool {
			if isMap {
				return isTruthy(it.callFunction(test, name, mapFunctionArgs(name, test, e.item, e.val)))
			}
			return isTruthy(it.callFunction(test, name, []object.Object{e.val}))
		}
		switch name {
		case "find_first":
			for _, e := range items {
				if match(e) {
					return e.item
				}
			}
			return object.NoneValue
		case "find_last":
			for i := len(items) - 1; i >= 0; i-- {
				if match(items[i]) {
					return items[i].item
				}
			}
			return object.NoneValue
		case "find_index":
			for i, e := range items {
				if match(e) {
					return object.Int(int64(i))
				}
			}
			return object.Int(-1)
		case "find_all":
			var ks []keyed
			for _, e := range items {
				if match(e) {
					ks = append(ks, keyed{entry: e})
				}
			}
			return itemsBack(ks, isMap)
		}
		n := 0
		for _, e := range items {
			if match(e) {
				n++
			}
		}
		return object.Int(int64(n))
	case "find_key":
		requireFuncArgs(name, args, 2)
		m, ok := args[0].(*object.Map)
		if !ok {
			fatalf("'find_key' needs a map, got %s", typeName(args[0]))
		}
		for _, me := range m.Entries() {
			if object.Equal(me.Val, args[1]) {
				return me.Key
			}
		}
		return object.NoneValue
	case "linear_search", "binary_search", "jump_search", "exponential_search",
		"interpolation_search", "ternary_search", "insert_position":
		if len(args) != 2 && len(args) != 3 {
			fatalf("'%s' expects a collection, the value to look for, and an optional key, got %d argument(s)", name, len(args))
		}
		var key object.Object
		if len(args) == 3 {
			key = args[2]
		}
		es, _ := entriesOf(name, args[0])
		ks := it.newSortKey(name, key).all(es)
		s := searcher{fn: name, ks: ks, want: args[1]}
		var at int
		switch name {
		case "linear_search":
			at = s.linear()
		case "binary_search":
			at = s.binary()
		case "jump_search":
			at = s.jump()
		case "exponential_search":
			at = s.exponential()
		case "interpolation_search":
			at = s.interpolation()
		case "ternary_search":
			at = s.ternary()
		case "insert_position":
			at = s.lowerBound(0, len(ks))
		}
		return object.Int(int64(at))
	}
	fatalKind(kindName, "no search function %q", name)
	return nil
}

// searcher looks for want among the keys of ks.
type searcher struct {
	fn   string
	ks   []keyed
	want object.Object
}

// cmp orders the key at i against what's wanted.
func (s searcher) cmp(i int) int { return compareValues(s.fn, s.ks[i].key, s.want) }

// found turns "the first key at least as large as want, from i on" into
// the answer: i if it's a match, else -1.
func (s searcher) found(i int) int {
	if i < len(s.ks) && s.cmp(i) == 0 {
		return i
	}
	return -1
}

// lowerBound is the first position in [lo, hi) whose key isn't smaller
// than want (hi if none), by halving.
func (s searcher) lowerBound(lo, hi int) int {
	for lo < hi {
		mid := lo + (hi-lo)/2
		if s.cmp(mid) < 0 {
			lo = mid + 1
		} else {
			hi = mid
		}
	}
	return lo
}

// linear checks each item in turn; the collection needn't be sorted.
func (s searcher) linear() int {
	for i := range s.ks {
		if s.cmp(i) == 0 {
			return i
		}
	}
	return -1
}

// binary halves the range each step: about 20 steps for a million items.
func (s searcher) binary() int { return s.found(s.lowerBound(0, len(s.ks))) }

// jump steps ahead by about the square root of the length until it
// passes want, then looks through the last block.
func (s searcher) jump() int {
	n := len(s.ks)
	if n == 0 {
		return -1
	}
	step := 1
	for step*step < n {
		step++
	}
	prev := 0
	for at := step; at < n && s.cmp(at-1) < 0; at += step {
		prev = at
	}
	end := min(prev+step, n)
	for i := prev; i < end; i++ {
		if c := s.cmp(i); c >= 0 {
			return s.found(i)
		}
	}
	return -1
}

// exponential doubles a bound (1, 2, 4, 8, ...) until it passes want,
// then searches binary inside it: quick when want is near the front.
func (s searcher) exponential() int {
	n := len(s.ks)
	if n == 0 {
		return -1
	}
	bound := 1
	for bound < n && s.cmp(bound) < 0 {
		bound *= 2
	}
	return s.found(s.lowerBound(bound/2, min(bound+1, n)))
}

// interpolation guesses where want is from the values at the ends, as
// one opens a phone book near the right letter: very fast on evenly
// spread numbers. With keys that aren't numbers it searches binary.
func (s searcher) interpolation() int {
	lo, hi := 0, len(s.ks)-1
	target, _, ok := numeric(s.want)
	for ok && lo <= hi {
		a, _, aok := numeric(s.ks[lo].key)
		b, _, bok := numeric(s.ks[hi].key)
		if !aok || !bok {
			break
		}
		if target < a || target > b {
			return -1
		}
		if a == b {
			break
		}
		pos := lo + int(float64(hi-lo)*(target-a)/(b-a))
		switch c := s.cmp(pos); {
		case c < 0:
			lo = pos + 1
		case c > 0:
			hi = pos - 1
		default:
			// A match: the first of equal keys is at or before it.
			return s.found(s.lowerBound(lo, pos+1))
		}
	}
	if lo > hi {
		return -1
	}
	return s.found(s.lowerBound(lo, hi+1))
}

// ternary splits the range in three each step.
func (s searcher) ternary() int {
	lo, hi := 0, len(s.ks)
	for hi-lo > 2 {
		third := (hi - lo) / 3
		m1, m2 := lo+third, hi-third
		switch {
		case s.cmp(m1) >= 0:
			hi = m1 + 1
		case s.cmp(m2) >= 0:
			lo, hi = m1+1, m2+1
		default:
			lo = m2 + 1
		}
	}
	return s.found(s.lowerBound(lo, hi))
}
