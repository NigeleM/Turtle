package evaluator

import (
	"math"
	"slices"
	"strings"

	"Turtle/object"
)

// The "sort" builtin module. Every function gives back a new list and
// leaves its argument as it was. A key says what to order by:
//
//	min_sort[books]                          the items themselves, smallest first
//	min_sort[books, b give b at get["price"]]  a function of each item
//	min_sort[books, "price"]                 a map key, or an assembled value's field
//	min_sort[pairs, 1]                       a position in a list of lists
//	min_sort[books, list ["author", "price"]]   several keys: ties go to the next
//	max_sort[books, "price", "first"]        just the top item (none if empty)
//	max_sort[books, "price", 3]              the first 3, as a list
//	min_sort[ages]                           a map: by its keys, as a new map
//	max_sort[ages, a give a]                a map: by its values (or [k, v] give ...)
//	max_sort[staff, "salary"]                a map of maps or assembled values: by a field
//
// The classic algorithms (bubble_sort ... radix_sort) take the same keys
// and give the same answer as min_sort; they're there to learn from and
// to compare. Search functions are in the "search" module (searchlib.go).

// ---- ordering any two values ----

// orderRank puts kinds of values in one order, as SQLite does: none,
// then true/false, numbers, text, dates, and lists.
func orderRank(v object.Object) int {
	switch v.(type) {
	case *object.None:
		return 0
	case *object.Boolean:
		return 1
	case *object.Integer, *object.Float:
		return 2
	case *object.String:
		return 3
	case *object.Date:
		return 4
	case *object.List, *object.Set:
		return 5
	case *object.Assembly:
		return 6
	case *object.Map:
		return 7
	}
	return -1
}

func orderKindName(v object.Object) string {
	switch v.(type) {
	case *object.Function:
		return "a function"
	}
	return "a " + strings.ToLower(string(v.Type()))
}

// compareValues orders a and b: -1, 0 or 1. Numbers compare by size
// (1 equals 1.0), text by its characters, dates by time, lists item by
// item (then the shorter first), assembled values by their type's name
// then field by field, maps entry by entry (key, then value). Different
// kinds go by orderRank.
func compareValues(fn string, a, b object.Object) int {
	ra, rb := orderRank(a), orderRank(b)
	if ra < 0 || rb < 0 {
		bad := a
		if ra >= 0 {
			bad = b
		}
		fatalKind(kindType, "%s: can't put %s in order", fn, orderKindName(bad))
	}
	if ra != rb {
		return cmpInt(ra, rb)
	}
	switch x := a.(type) {
	case *object.None:
		return 0
	case *object.Boolean:
		y := b.(*object.Boolean)
		switch {
		case x.Value == y.Value:
			return 0
		case !x.Value:
			return -1
		}
		return 1
	case *object.Integer:
		if y, ok := b.(*object.Integer); ok {
			return cmpInt64(x.Value, y.Value)
		}
	case *object.String:
		return strings.Compare(x.Value, b.(*object.String).Value)
	case *object.Date:
		return x.Time.Compare(b.(*object.Date).Time)
	case *object.List, *object.Set:
		ea, eb := elementsOf(a), elementsOf(b)
		for i := 0; i < len(ea) && i < len(eb); i++ {
			if c := compareValues(fn, ea[i], eb[i]); c != 0 {
				return c
			}
		}
		return cmpInt(len(ea), len(eb))
	case *object.Assembly:
		y := b.(*object.Assembly)
		if c := strings.Compare(x.Shape.Name, y.Shape.Name); c != 0 {
			return c
		}
		for i := 0; i < len(x.Values) && i < len(y.Values); i++ {
			if c := compareValues(fn, x.Values[i], y.Values[i]); c != 0 {
				return c
			}
		}
		return cmpInt(len(x.Values), len(y.Values))
	case *object.Map:
		y := b.(*object.Map)
		for i := 0; i < len(x.Keys) && i < len(y.Keys); i++ {
			if c := compareValues(fn, x.KeyOf(x.Keys[i]), y.KeyOf(y.Keys[i])); c != 0 {
				return c
			}
			if c := compareValues(fn, x.Values[x.Keys[i]], y.Values[y.Keys[i]]); c != 0 {
				return c
			}
		}
		return cmpInt(len(x.Keys), len(y.Keys))
	}
	fa, _, _ := numeric(a)
	fb, _, _ := numeric(b)
	switch {
	case fa < fb:
		return -1
	case fa > fb:
		return 1
	case fa == fb:
		return 0
	}
	// NaN: after every other number, equal to itself.
	return cmpBool(math.IsNaN(fa), math.IsNaN(fb))
}

func elementsOf(v object.Object) []object.Object {
	switch c := v.(type) {
	case *object.List:
		return c.Elements
	case *object.Set:
		return c.Elements
	}
	return nil
}

func cmpInt(a, b int) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	}
	return 0
}

func cmpInt64(a, b int64) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	}
	return 0
}

func cmpBool(a, b bool) int {
	switch {
	case a == b:
		return 0
	case a:
		return 1
	}
	return -1
}

// ---- items and keys ----

// entry is one thing sort and search work through: a list's or set's
// item, a text's character, or a map's entry (item is its key, val its
// value).
type entry struct {
	item, val object.Object
	isMap     bool
}

// entriesOf gives x's entries, and whether x is a map.
func entriesOf(fn string, x object.Object) ([]entry, bool) {
	var items []object.Object
	switch c := x.(type) {
	case *object.List:
		items = c.Elements
	case *object.Set:
		items = c.Elements
	case *object.Map:
		out := make([]entry, len(c.Keys))
		for i, k := range c.Keys {
			out[i] = entry{item: c.KeyOf(k), val: c.Values[k], isMap: true}
		}
		return out, true
	case *object.String:
		for _, r := range c.Value {
			items = append(items, &object.String{Value: string(r)})
		}
	case *object.Assembly:
		fatalKind(kindType, "%s: a %s is one assembled value; it works on several in a list, set or map (list [a, b, c])", fn, c.Shape.Name)
	default:
		fatalKind(kindType, "%s needs a list, set, map or text, got %s", fn, x.Type())
	}
	out := make([]entry, len(items))
	for i, e := range items {
		out[i] = entry{item: e, val: e}
	}
	return out, false
}

// sortKey is how to get what an item is ordered or searched by.
type sortKey struct {
	fn   string
	key  object.Object // nil: the item itself
	it   *Interpreter
	many []sortKey // a list of keys
}

func (it *Interpreter) newSortKey(fn string, key object.Object) sortKey {
	k := sortKey{fn: fn, it: it}
	switch x := key.(type) {
	case nil, *object.None:
	case *object.Function, *object.String, *object.Integer:
		k.key = x
	case *object.List:
		if len(x.Elements) == 0 {
			fatalKind(kindType, "%s: a list of keys needs at least one key", fn)
		}
		for _, e := range x.Elements {
			if _, ok := e.(*object.List); ok {
				fatalKind(kindType, "%s: a list of keys can't hold another list", fn)
			}
			k.many = append(k.many, it.newSortKey(fn, e))
		}
	default:
		fatalKind(kindType, "%s: the key must be a function (x give ...), a field or map key (\"price\"), a position (1), or a list of those, got %s", fn, key.Type())
	}
	return k
}

// of gives the value an entry is ordered by. With no key that's the
// item itself, or a map entry's key. A key picks from the item, or a map
// entry's value; a function gets the item, or a map entry's value (one
// parameter) or key and value (two).
func (k sortKey) of(e entry) object.Object {
	if k.many != nil {
		out := &object.List{Elements: make([]object.Object, len(k.many))}
		for i, sub := range k.many {
			out.Elements[i] = sub.of(e)
		}
		return out
	}
	item := e.val
	switch key := k.key.(type) {
	case nil:
		return e.item
	case *object.Function:
		if e.isMap {
			return k.it.callFunction(key, k.fn, mapFunctionArgs(k.fn, key, e.item, e.val))
		}
		return k.it.callFunction(key, k.fn, []object.Object{item})
	case *object.String:
		switch c := item.(type) {
		case *object.Map:
			v, ok := c.Get(key)
			if !ok {
				fatalKind(kindKey, "%s: an item has no %q key: %s", k.fn, key.Value, item.Inspect())
			}
			return v
		case *object.Assembly:
			for i, f := range c.Shape.Fields {
				if f == key.Value {
					return c.Values[i]
				}
			}
			fatalKind(kindName, "%s: a %s has no field %q (its fields: %s)", k.fn, c.Shape.Name, key.Value, strings.Join(c.Shape.Fields, ", "))
		}
		fatalKind(kindType, "%s: the key %q needs maps or assembled values, but an item is %s", k.fn, key.Value, item.Inspect())
	case *object.Integer:
		l, ok := item.(*object.List)
		if !ok {
			fatalKind(kindType, "%s: the key %d is a position, so it needs lists, but an item is %s", k.fn, key.Value, item.Inspect())
		}
		if key.Value < 0 || key.Value >= int64(len(l.Elements)) {
			fatalKind(kindIndex, "%s: position %d is past the end of %s", k.fn, key.Value, item.Inspect())
		}
		return l.Elements[key.Value]
	}
	return item
}

// keyed pairs each entry with its key, worked out once.
type keyed struct {
	entry
	key object.Object
}

func (k sortKey) all(es []entry) []keyed {
	out := make([]keyed, len(es))
	for i, e := range es {
		out[i] = keyed{e, k.of(e)}
	}
	return out
}

// itemsBack gives entries back as a list, or for a map as a new map in
// their order.
func itemsBack(ks []keyed, isMap bool) object.Object {
	if isMap {
		m := object.NewMap()
		for _, k := range ks {
			m.Put(k.item, k.val)
		}
		return m
	}
	out := &object.List{Elements: make([]object.Object, len(ks))}
	for i, k := range ks {
		out.Elements[i] = k.item
	}
	return out
}

// ---- the module ----

// sortArgs reads [collection [, key]] (and for min_sort/max_sort an
// optional third, how much to give back).
func (it *Interpreter) sortArgs(fn string, args []object.Object, most int) ([]keyed, bool, func(a, b keyed) int) {
	if len(args) < 1 || len(args) > most {
		want := "a collection and an optional key"
		if most == 3 {
			want = `a collection, an optional key, and optionally "first" or how many`
		}
		fatalf("'%s' expects %s, got %d argument(s)", fn, want, len(args))
	}
	var key object.Object
	if len(args) > 1 {
		key = args[1]
	}
	k := it.newSortKey(fn, key)
	es, isMap := entriesOf(fn, args[0])
	return k.all(es), isMap, func(a, b keyed) int { return compareValues(fn, a.key, b.key) }
}

func (it *Interpreter) callSort(name string, args []object.Object) object.Object {
	switch name {
	case "min_sort", "max_sort":
		ks, isMap, cmp := it.sortArgs(name, args, 3)
		if name == "max_sort" {
			slices.SortStableFunc(ks, func(a, b keyed) int { return cmp(b, a) })
		} else {
			slices.SortStableFunc(ks, cmp)
		}
		if len(args) < 3 {
			return itemsBack(ks, isMap)
		}
		switch how := args[2].(type) {
		case *object.String:
			if how.Value != "first" {
				fatalf(`'%s': the third argument is "first" or how many items to give back, got %q`, name, how.Value)
			}
			if len(ks) == 0 {
				return object.NoneValue
			}
			return ks[0].item
		case *object.Integer:
			if how.Value < 0 {
				fatalf("'%s': can't give back %d items", name, how.Value)
			}
			if how.Value < int64(len(ks)) {
				ks = ks[:how.Value]
			}
			return itemsBack(ks, isMap)
		}
		fatalf(`'%s': the third argument is "first" or how many items to give back, got %s`, name, args[2].Type())
	case "is_sorted":
		ks, _, cmp := it.sortArgs(name, args, 2)
		for i := 1; i < len(ks); i++ {
			if cmp(ks[i-1], ks[i]) > 0 {
				return &object.Boolean{Value: false}
			}
		}
		return &object.Boolean{Value: true}
	case "reverse_list":
		requireFuncArgs(name, args, 1)
		es, isMap := entriesOf(name, args[0])
		ks := make([]keyed, len(es))
		for i, e := range es {
			ks[len(es)-1-i] = keyed{entry: e}
		}
		return itemsBack(ks, isMap)
	case "bubble_sort", "insertion_sort", "selection_sort", "merge_sort", "quick_sort",
		"heap_sort", "shell_sort", "counting_sort", "radix_sort":
		ks, isMap, cmp := it.sortArgs(name, args, 2)
		classicSorts[name](name, ks, cmp)
		return itemsBack(ks, isMap)
	}
	fatalKind(kindName, "no sort function %q", name)
	return nil
}

// ---- the classic algorithms ----
//
// Each sorts ks in place, smallest key first. All but selection, quick,
// heap and shell sort keep items with equal keys in their first order
// (stable); those four may not, as in any textbook.

var classicSorts = map[string]func(fn string, ks []keyed, cmp func(a, b keyed) int){
	"bubble_sort":    bubbleSort,
	"insertion_sort": insertionSort,
	"selection_sort": selectionSort,
	"merge_sort":     mergeSort,
	"quick_sort":     quickSort,
	"heap_sort":      heapSort,
	"shell_sort":     shellSort,
	"counting_sort":  countingSort,
	"radix_sort":     radixSort,
}

// bubbleSort swaps neighbours that are out of order until a pass swaps
// nothing. About n*n steps.
func bubbleSort(_ string, ks []keyed, cmp func(a, b keyed) int) {
	for end := len(ks); end > 1; end-- {
		swapped := false
		for i := 1; i < end; i++ {
			if cmp(ks[i-1], ks[i]) > 0 {
				ks[i-1], ks[i] = ks[i], ks[i-1]
				swapped = true
			}
		}
		if !swapped {
			return
		}
	}
}

// insertionSort grows a sorted front, sliding each next item back into
// place. Fast on lists that are almost sorted.
func insertionSort(_ string, ks []keyed, cmp func(a, b keyed) int) {
	for i := 1; i < len(ks); i++ {
		cur := ks[i]
		j := i
		for j > 0 && cmp(ks[j-1], cur) > 0 {
			ks[j] = ks[j-1]
			j--
		}
		ks[j] = cur
	}
}

// selectionSort finds the smallest of the rest and puts it next.
func selectionSort(_ string, ks []keyed, cmp func(a, b keyed) int) {
	for i := range ks {
		min := i
		for j := i + 1; j < len(ks); j++ {
			if cmp(ks[j], ks[min]) < 0 {
				min = j
			}
		}
		ks[i], ks[min] = ks[min], ks[i]
	}
}

// mergeSort sorts each half, then merges them. About n*log(n) steps,
// always.
func mergeSort(_ string, ks []keyed, cmp func(a, b keyed) int) {
	if len(ks) < 2 {
		return
	}
	tmp := make([]keyed, len(ks))
	var sortRange func(lo, hi int)
	sortRange = func(lo, hi int) {
		if hi-lo < 2 {
			return
		}
		mid := (lo + hi) / 2
		sortRange(lo, mid)
		sortRange(mid, hi)
		i, j, k := lo, mid, lo
		for i < mid && j < hi {
			if cmp(ks[j], ks[i]) < 0 {
				tmp[k] = ks[j]
				j++
			} else {
				tmp[k] = ks[i]
				i++
			}
			k++
		}
		k += copy(tmp[k:], ks[i:mid])
		copy(tmp[k:], ks[j:hi])
		copy(ks[lo:hi], tmp[lo:hi])
	}
	sortRange(0, len(ks))
}

// quickSort splits around a middle item (the median of three), smaller
// ones before it, larger after, and sorts each side.
func quickSort(_ string, ks []keyed, cmp func(a, b keyed) int) {
	var sortRange func(lo, hi int) // inclusive
	sortRange = func(lo, hi int) {
		for lo < hi {
			mid := lo + (hi-lo)/2
			if cmp(ks[mid], ks[lo]) < 0 {
				ks[mid], ks[lo] = ks[lo], ks[mid]
			}
			if cmp(ks[hi], ks[lo]) < 0 {
				ks[hi], ks[lo] = ks[lo], ks[hi]
			}
			if cmp(ks[hi], ks[mid]) < 0 {
				ks[hi], ks[mid] = ks[mid], ks[hi]
			}
			pivot := ks[mid]
			i, j := lo, hi
			for i <= j {
				for cmp(ks[i], pivot) < 0 {
					i++
				}
				for cmp(ks[j], pivot) > 0 {
					j--
				}
				if i <= j {
					ks[i], ks[j] = ks[j], ks[i]
					i++
					j--
				}
			}
			// Recurse into the smaller side, loop on the larger.
			if j-lo < hi-i {
				sortRange(lo, j)
				lo = i
			} else {
				sortRange(i, hi)
				hi = j
			}
		}
	}
	sortRange(0, len(ks)-1)
}

// heapSort arranges the items as a heap (each parent at least as large
// as its children), then takes the largest off the top, again and again.
func heapSort(_ string, ks []keyed, cmp func(a, b keyed) int) {
	n := len(ks)
	down := func(i, n int) {
		for {
			big := i
			l, r := 2*i+1, 2*i+2
			if l < n && cmp(ks[l], ks[big]) > 0 {
				big = l
			}
			if r < n && cmp(ks[r], ks[big]) > 0 {
				big = r
			}
			if big == i {
				return
			}
			ks[i], ks[big] = ks[big], ks[i]
			i = big
		}
	}
	for i := n/2 - 1; i >= 0; i-- {
		down(i, n)
	}
	for end := n - 1; end > 0; end-- {
		ks[0], ks[end] = ks[end], ks[0]
		down(0, end)
	}
}

// shellSort is insertion sort over shrinking gaps, so items far from
// their place move there in a few long jumps.
func shellSort(_ string, ks []keyed, cmp func(a, b keyed) int) {
	gap := 1
	for gap < len(ks)/3 {
		gap = gap*3 + 1
	}
	for ; gap > 0; gap /= 3 {
		for i := gap; i < len(ks); i++ {
			cur := ks[i]
			j := i
			for j >= gap && cmp(ks[j-gap], cur) > 0 {
				ks[j] = ks[j-gap]
				j -= gap
			}
			ks[j] = cur
		}
	}
}

// intKeys gives every key as an integer, for counting and radix sort.
func intKeys(fn string, ks []keyed) []int64 {
	out := make([]int64, len(ks))
	for i, k := range ks {
		n, ok := k.key.(*object.Integer)
		if !ok {
			fatalKind(kindType, "%s sorts whole numbers only, but a key is %s (min_sort sorts anything)", fn, k.key.Inspect())
		}
		out[i] = n.Value
	}
	return out
}

// countingSort counts how many items have each key, then places them.
// Whole-number keys only; fast when they're in a small range.
func countingSort(fn string, ks []keyed, _ func(a, b keyed) int) {
	if len(ks) < 2 {
		return
	}
	keys := intKeys(fn, ks)
	lo, hi := slices.Min(keys), slices.Max(keys)
	if hi-lo > 10_000_000 || hi-lo < 0 {
		fatalKind(kindType, "%s: the keys run from %d to %d, too wide a range to count (radix_sort or min_sort handle it)", fn, lo, hi)
	}
	counts := make([]int, hi-lo+2)
	for _, k := range keys {
		counts[k-lo+1]++
	}
	for i := 1; i < len(counts); i++ {
		counts[i] += counts[i-1]
	}
	out := make([]keyed, len(ks))
	for i, k := range keys {
		out[counts[k-lo]] = ks[i]
		counts[k-lo]++
	}
	copy(ks, out)
}

// radixSort orders whole-number keys one byte at a time, from the
// lowest byte up, keeping each pass stable.
func radixSort(fn string, ks []keyed, _ func(a, b keyed) int) {
	if len(ks) < 2 {
		return
	}
	keys := intKeys(fn, ks)
	// Flip the sign bit so negative numbers order before positive ones.
	u := make([]uint64, len(keys))
	for i, k := range keys {
		u[i] = uint64(k) ^ (1 << 63)
	}
	idx := make([]int, len(ks))
	for i := range idx {
		idx[i] = i
	}
	tmp := make([]int, len(ks))
	for shift := 0; shift < 64; shift += 8 {
		var counts [257]int
		for _, i := range idx {
			counts[(u[i]>>shift)&0xff+1]++
		}
		if counts[(u[idx[0]]>>shift)&0xff+1] == len(ks) {
			continue // every key has the same byte here
		}
		for b := 1; b < 257; b++ {
			counts[b] += counts[b-1]
		}
		for _, i := range idx {
			b := (u[i] >> shift) & 0xff
			tmp[counts[b]] = i
			counts[b]++
		}
		idx, tmp = tmp, idx
	}
	out := make([]keyed, len(ks))
	for p, i := range idx {
		out[p] = ks[i]
	}
	copy(ks, out)
}
