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
		if !ok || len(av.Keys) != len(bv.Keys) {
			return false
		}
		for _, k := range av.Keys {
			v, ok := bv.Values[k]
			if !ok || !Equal(av.Values[k], v) {
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
		if v.Value == float64(int64(v.Value)) {
			return strconv.FormatInt(int64(v.Value), 10)
		}
		return v.Inspect()
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
		parts := make([]string, len(v.Keys))
		for i, k := range v.Keys {
			parts[i] = k + ":" + Key(v.Values[k])
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
	}
	return obj.Inspect()
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
