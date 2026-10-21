// Copyright 2017-2026 Nigele McCoy
// Licensed under the Apache License, Version 2.0;
// see LICENSE and NOTICE.

package object

import (
	"fmt"
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
func Equal(a, b Object) bool {
	if ai, ok := a.(*Integer); ok {
		if bi, ok := b.(*Integer); ok {
			return ai.Value == bi.Value
		}
	}
	if af, ok := number(a); ok {
		bf, ok := number(b)
		return ok && af == bf
	}
	switch av := a.(type) {
	case *List:
		bv, ok := b.(*List)
		if !ok || len(av.Elements) != len(bv.Elements) {
			return false
		}
		for i := range av.Elements {
			if !Equal(av.Elements[i], bv.Elements[i]) {
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
			if !bv.Contains(e) {
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
			if !ok || !Equal(e.Val, v) {
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
			if !Equal(av.Values[i], bv.Values[i]) {
				return false
			}
		}
		return true
	}
	return a.Type() == b.Type() && a.Inspect() == b.Inspect()
}

// entryKey is a map's or set's index key for obj: one key exactly when
// Key says so, without making anything for the usual key, text. Text is
// itself; anything else is a NUL and its Key, which no text starts with
// (text that does start with a NUL gets the same treatment).
func entryKey(obj Object) string {
	if s, ok := obj.(*String); ok && (s.Value == "" || s.Value[0] != 0) {
		return s.Value
	}
	return "\x00" + Key(obj)
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
func Key(obj Object) string {
	switch v := obj.(type) {
	case *Date:
		// The same moment is the same key, whichever clock shows it.
		return "date:" + v.Time.UTC().Format(time.RFC3339)
	case *String:
		return strconv.Quote(v.Value)
	case *Float:
		// The exact value, as Equal compares: 0.1 + 0.2 and 0.3 are two
		// keys. A whole number matches its integer key: 2.0 is the key of 2.
		if v.Value == float64(int64(v.Value)) {
			return strconv.FormatInt(int64(v.Value), 10)
		}
		return floatText(v.Value)
	case *Set:
		parts := make([]string, len(v.Elements))
		for i, e := range v.Elements {
			parts[i] = Key(e)
		}
		sort.Strings(parts)
		return "{" + strings.Join(parts, ",") + "}"
	case *List:
		parts := make([]string, len(v.Elements))
		for i, e := range v.Elements {
			parts[i] = Key(e)
		}
		return "[" + strings.Join(parts, ",") + "]"
	case *Map:
		parts := make([]string, len(v.entries))
		for i, e := range v.entries {
			parts[i] = Key(e.Key) + ":" + Key(e.Val)
		}
		sort.Strings(parts)
		return "map{" + strings.Join(parts, ",") + "}"
	case *Assembly:
		parts := make([]string, len(v.Values))
		for i, e := range v.Values {
			parts[i] = v.Shape.Fields[i] + ":" + Key(e)
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
