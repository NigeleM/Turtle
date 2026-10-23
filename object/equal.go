// Copyright 2017-2026 Nigele McCoy
// Licensed under the Apache License, Version 2.0;
// see LICENSE and NOTICE.

package object

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Equal is Turtle's one notion of value equality, used by ==/!=, set
// deduplication and membership, and list/set searches (count, index,
// find, remove). Numbers compare by value across Integer/Float (1 ==
// 1.0); functions by identity; lists element by element in order; sets
// and maps regardless of order; assemblies when they're the same assembled
// type with equal fields. Anything else needs the same type and
// the same value — so 1 and "1" are different, even though both show
// as 1.
func Equal(a, b Object) bool { return equal(a, b, 0) }

// equal is Equal at depth levels inside other values (see MaxNesting).
func equal(a, b Object, depth int) bool {
	if ai, ok := a.(*Integer); ok {
		if bi, ok := b.(*Integer); ok {
			return ai.Value == bi.Value
		}
	}
	if ab, ok := a.(*BigInt); ok {
		if bb, ok := b.(*BigInt); ok {
			return ab.V.Cmp(bb.V) == 0
		}
		if bf, ok := b.(*Float); ok {
			return equalBigFloat(ab.V, bf.Value)
		}
		return false
	}
	if bb, ok := b.(*BigInt); ok {
		if af, ok := a.(*Float); ok {
			return equalBigFloat(bb.V, af.Value)
		}
		return false
	}
	if af, ok := number(a); ok {
		bf, ok := number(b)
		return ok && af == bf
	}
	switch a.(type) {
	case *List, *Set, *Map, *Assembly:
		if a == b {
			// The same value equals itself without a walk, as in Python:
			// even one that holds itself, or holds nan.
			return true
		}
		if depth >= MaxNesting {
			TooDeep()
		}
		depth++
	}
	switch av := a.(type) {
	case *List:
		bv, ok := b.(*List)
		if !ok || len(av.Elements) != len(bv.Elements) {
			return false
		}
		for i := range av.Elements {
			if !equal(av.Elements[i], bv.Elements[i], depth) {
				return false
			}
		}
		return true
	case *Set:
		bv, ok := b.(*Set)
		if !ok || len(av.Elements) != len(bv.Elements) {
			return false
		}
		for _, e := range av.Elements {
			if !bv.contains(e, depth) {
				return false
			}
		}
		return true
	case *Map:
		bv, ok := b.(*Map)
		if !ok || av.Len() != bv.Len() {
			return false
		}
		for _, e := range av.entries {
			v, ok := bv.Get(e.Key)
			if !ok || !equal(e.Val, v, depth) {
				return false
			}
		}
		return true
	case *Function:
		return a == b
	case *Date:
		bv, ok := b.(*Date)
		return ok && av.Time.Equal(bv.Time)
	case *Database:
		return a == b
	case *Matrix:
		bv, ok := b.(*Matrix)
		return ok && EqualMatrix(av, bv)
	case *Assembly:
		bv, ok := b.(*Assembly)
		if !ok || av.Shape != bv.Shape {
			return false
		}
		for i := range av.Values {
			if !equal(av.Values[i], bv.Values[i], depth) {
				return false
			}
		}
		return true
	}
	return a.Type() == b.Type() && a.Inspect() == b.Inspect()
}

// hashKey is a map's or set's index key: one key exactly when Key says so.
// The usual keys, text and whole numbers, are kept as themselves, so
// finding one makes nothing.
type hashKey struct {
	whole bool   // n is the key: a whole number that fits 64 bits
	n     int64  //
	s     string // otherwise: text as itself, anything else a NUL and its Key
}

// entryKey is obj's index key. A whole number that fits 64 bits is its
// value, whether an integer, a whole float (2.0 is 2, as Key says) or a
// big integer that fits. Text is itself; anything else is a NUL and its
// Key, which no text starts with (text that does start with a NUL gets
// the same treatment).
func entryKey(obj Object) hashKey {
	switch v := obj.(type) {
	case *String:
		if v.Value == "" || v.Value[0] != 0 {
			return hashKey{s: v.Value}
		}
	case *Integer:
		return hashKey{whole: true, n: v.Value}
	case *Float:
		if v.Value == math.Trunc(v.Value) && v.Value >= -(1<<63) && v.Value < 1<<63 {
			return hashKey{whole: true, n: int64(v.Value)}
		}
	case *BigInt:
		if v.V.IsInt64() {
			return hashKey{whole: true, n: v.V.Int64()}
		}
	}
	return hashKey{s: "\x00" + Key(obj)}
}

// keyIndex finds a V by hashKey: text keys in one Go map, whole numbers in
// another, so each kind hashes as itself. Each map is made on first use.
type keyIndex[V any] struct {
	text  map[string]V
	whole map[int64]V
	size  int // how many entries to make room for at first
}

func newKeyIndex[V any](size int) *keyIndex[V] { return &keyIndex[V]{size: size} }

func (x *keyIndex[V]) get(k hashKey) (V, bool) {
	if k.whole {
		v, ok := x.whole[k.n]
		return v, ok
	}
	v, ok := x.text[k.s]
	return v, ok
}

func (x *keyIndex[V]) put(k hashKey, v V) {
	if k.whole {
		if x.whole == nil {
			x.whole = make(map[int64]V, x.size)
		}
		x.whole[k.n] = v
		return
	}
	if x.text == nil {
		x.text = make(map[string]V, x.size)
	}
	x.text[k.s] = v
}

// sameKey is whether a and b are one key, as Key(a) == Key(b) says (2
// and 2.0 are; "1" and 1 aren't), with nothing made for text or whole
// numbers.
func sameKey(a, b Object) bool {
	switch x := a.(type) {
	case *String:
		y, ok := b.(*String)
		return ok && x.Value == y.Value
	case *Integer:
		if y, ok := b.(*Integer); ok {
			return x.Value == y.Value
		}
	}
	return Key(a) == Key(b)
}

// Key is the internal string a value is stored under as a map key (and
// grouped by in a set's index). Equal values always have the same Key:
// a set's elements are put in a fixed order first, and a whole-number
// float matches its integer (2.0 == 2). Strings are quoted, so the
// string "1" and the integer 1 are different keys.
func Key(obj Object) string { return key(obj, 0) }

// key is Key at depth levels inside other values (see MaxNesting).
func key(obj Object, depth int) string {
	switch obj.(type) {
	case *Set, *List, *Map, *Assembly:
		if depth >= MaxNesting {
			TooDeep()
		}
		depth++
	}
	switch v := obj.(type) {
	case *Date:
		// The same moment is the same key, whichever clock shows it.
		return "date:" + v.Time.UTC().Format(time.RFC3339)
	case *String:
		return strconv.Quote(v.Value)
	case *Float:
		// The exact value, as Equal compares: 0.1 + 0.2 and 0.3 are two
		// keys. A whole number matches its integer key: 2.0 is the key of 2.
		if v.Value == math.Trunc(v.Value) && !math.IsInf(v.Value, 0) {
			if whole, ok := FloatToInt(v.Value); ok {
				return whole.Inspect() // the same key as the integer, even past 64 bits
			}
		}
		return floatText(v.Value)
	case *Set:
		parts := make([]string, len(v.Elements))
		for i, e := range v.Elements {
			parts[i] = key(e, depth)
		}
		sort.Strings(parts)
		return "{" + strings.Join(parts, ",") + "}"
	case *List:
		parts := make([]string, len(v.Elements))
		for i, e := range v.Elements {
			parts[i] = key(e, depth)
		}
		return "[" + strings.Join(parts, ",") + "]"
	case *Map:
		parts := make([]string, len(v.entries))
		for i, e := range v.entries {
			parts[i] = key(e.Key, depth) + ":" + key(e.Val, depth)
		}
		sort.Strings(parts)
		return "map{" + strings.Join(parts, ",") + "}"
	case *Assembly:
		parts := make([]string, len(v.Values))
		for i, e := range v.Values {
			parts[i] = v.Shape.Fields[i] + ":" + key(e, depth)
		}
		return v.Shape.Name + "{" + strings.Join(parts, ",") + "}"
	case *Function:
		return fmt.Sprintf("function %p", v)
	case *Matrix:
		parts := make([]string, len(v.Data))
		for i, x := range v.Data {
			parts[i] = Key(&Float{Value: x})
		}
		return "matrix" + strconv.Itoa(v.Cols) + "[" + strings.Join(parts, ",") + "]"
	}
	return obj.Inspect()
}

// CompareFloats orders two numbers exactly, as IEEE 754 does (-1, 0 or
// 1). A NaN is not equal to anything, and sorts after the numbers.
func CompareFloats(a, b float64) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	case a == b:
		return 0
	}
	return 1 // a NaN
}

func number(obj Object) (float64, bool) {
	switch v := obj.(type) {
	case *Integer:
		return float64(v.Value), true
	case *Float:
		return v.Value, true
	}
	return 0, false
}
