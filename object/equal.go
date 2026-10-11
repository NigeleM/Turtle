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
func Equal(a, b Object) bool {
	if ai, ok := a.(*Integer); ok {
		if bi, ok := b.(*Integer); ok {
			return ai.Value == bi.Value
		}
	}
	if af, ok := number(a); ok {
		bf, ok := number(b)
		return ok && CompareFloats(af, bf) == 0
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
			v, ok := bv.GetK(e.K)
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
		// The 15-digit value, as Equal compares: 0.1 + 0.2 and 0.3 are one
		// key, and 2.0000000000000004 is the key of 2.
		r := Round15(v.Value)
		if r == float64(int64(r)) {
			return strconv.FormatInt(int64(r), 10)
		}
		return floatText(r)
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
			parts[i] = e.K + ":" + Key(e.Val)
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

// Round15 is v to 15 significant digits, the way floats show (Inspect).
func Round15(v float64) float64 {
	if v == 0 || math.IsInf(v, 0) || math.IsNaN(v) {
		return v
	}
	r, _ := strconv.ParseFloat(strconv.FormatFloat(v, 'g', 15, 64), 64)
	return r
}

// CompareFloats orders two numbers as they show, to 15 significant
// digits: 0.1 + 0.2 equals 0.3, and 0.30000000000000004 isn't more than
// 0.3. Numbers that are clearly apart are compared directly, without
// rounding, so loops stay fast. -1, 0 or 1.
func CompareFloats(a, b float64) int {
	if a == b {
		return 0
	}
	if math.Abs(a-b) > 1e-12*math.Max(math.Abs(a), math.Abs(b)) || math.IsNaN(a) || math.IsNaN(b) {
		if a < b {
			return -1
		}
		return 1
	}
	ra, rb := Round15(a), Round15(b)
	switch {
	case ra < rb:
		return -1
	case ra > rb:
		return 1
	}
	return 0
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
