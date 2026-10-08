package evaluator

import (
	"math"
	"sort"
	"strconv"
	"strings"

	"Turtle/object"
)

// data's statistics: mean, median, mode, variance, stdev (and their
// population forms), percentile, covariance, correlation, zscores and
// describe. Each takes a list, a set or a map (its values), or a list of
// rows with a column name: mean[books, "price"]. none is skipped, as a
// database's missing value; any other value that isn't a number is an
// error. variance, stdev and covariance are the sample forms (dividing by
// n - 1), as Python and spreadsheets give; pvariance and pstdev divide by
// n.

var statsFuncs = []string{
	"mean", "median", "mode", "variance", "stdev", "pvariance", "pstdev",
	"percentile", "covariance", "correlation", "zscores", "describe",
}

// statItems is the values a statistic works on: args[0], or a column of
// it when args[1] names one. rest is the arguments after those.
func statItems(fn string, args []object.Object, extra int) (items []object.Object, rest []object.Object) {
	if len(args) < 1 {
		fatalf("function %q needs a list of numbers", fn)
	}
	coll := args[0]
	rest = args[1:]
	if len(rest) > extra {
		col, ok := rest[0].(*object.String)
		if !ok {
			fatalf("function %q: after the rows comes a column name, got %s", fn, object.Shown(rest[0]))
		}
		return columnValues(fn, coll, col.Value), rest[1:]
	}
	if len(rest) != extra {
		fatalf("function %q expects %d argument(s) (and a column name for rows), got %d", fn, 1+extra, len(args))
	}
	switch coll.(type) {
	case *object.List, *object.Set, *object.Map:
	default:
		fatalKind(kindType, "'%s' needs a list, set or map of numbers, got %s", fn, typeName(coll))
	}
	return reduceItems(coll), rest
}

// columnValues is one column of a list of maps or assembled values.
func columnValues(fn string, coll object.Object, col string) []object.Object {
	l, ok := coll.(*object.List)
	if !ok || !allRecords(l.Elements) {
		fatalKind(kindType, "'%s' with a column name needs a list of rows (maps or assembled values), got %s", fn, typeName(coll))
	}
	header, rows := recordRows(l.Elements)
	idx := -1
	for i, h := range header {
		if h == col {
			idx = i
		}
	}
	if idx < 0 {
		fatalKind(kindKey, "'%s': no column %q in these rows", fn, col)
	}
	out := make([]object.Object, 0, len(rows))
	for _, r := range rows {
		if r[idx] != nil {
			out = append(out, r[idx])
		}
	}
	return out
}

// numbers is the numbers among items, none skipped; ints is whether all
// are integers.
func numbers(fn string, items []object.Object) (xs []float64, ints bool) {
	ints = true
	for i, it := range items {
		if _, ok := it.(*object.None); ok {
			continue
		}
		f, isInt, ok := numeric(it)
		if !ok {
			fatalKind(kindType, "'%s' works on numbers; item %d is %s%s", fn, i, object.Shown(it), textNumberHint(it))
		}
		ints = ints && isInt
		xs = append(xs, f)
	}
	return xs, ints
}

// textNumberHint is the fix for a number that's text, as a CSV file's
// cells are: read it with column types.
func textNumberHint(v object.Object) string {
	if s, ok := v.(*object.String); ok {
		if _, err := strconv.ParseFloat(strings.TrimSpace(s.Value), 64); err == nil {
			return ` (a number as text: change it to float, or read its file with column types, table_read[path, map ["price": "float"]])`
		}
	}
	return ""
}

func needCount(fn string, xs []float64, n int) {
	if len(xs) < n {
		what := "any numbers"
		if n > 1 {
			what = "at least " + itoa(n) + " numbers"
		}
		fatalKind(kindIndex, "'%s' needs %s, got %d", fn, what, len(xs))
	}
}

// sumOf adds xs up with Neumaier's compensated summation: the rounding
// lost at each step is kept and added back, so a thousand 0.1s average to
// 0.1, not 0.09999999999999859.
func sumOf(n int, x func(int) float64) float64 {
	sum, lost := 0.0, 0.0
	for i := range n {
		v := x(i)
		t := sum + v
		if math.Abs(sum) >= math.Abs(v) {
			lost += (sum - t) + v
		} else {
			lost += (v - t) + sum
		}
		sum = t
	}
	return sum + lost
}

func meanOf(xs []float64) float64 {
	return sumOf(len(xs), func(i int) float64 { return xs[i] }) / float64(len(xs))
}

// sumSquares is the sum of each number's squared distance from the mean.
func sumSquares(xs []float64) float64 {
	m := meanOf(xs)
	return sumOf(len(xs), func(i int) float64 { return (xs[i] - m) * (xs[i] - m) })
}

// percentileOf is the p-th percentile of sorted xs, between the two
// nearest numbers (as numpy, and a spreadsheet's PERCENTILE, give).
func percentileOf(sorted []float64, p float64) float64 {
	pos := p / 100 * float64(len(sorted)-1)
	lo := int(math.Floor(pos))
	if lo >= len(sorted)-1 {
		return sorted[len(sorted)-1]
	}
	return sorted[lo] + (pos-float64(lo))*(sorted[lo+1]-sorted[lo])
}

func sortedCopy(xs []float64) []float64 {
	s := append([]float64(nil), xs...)
	sort.Float64s(s)
	return s
}

// pairs is covariance's and correlation's two series: two lists, or two
// columns of rows. A pair with none on either side is skipped.
func pairs(fn string, args []object.Object) ([]float64, []float64) {
	var a, b []object.Object
	switch len(args) {
	case 2:
		for _, a := range args {
			switch a.(type) {
			case *object.List, *object.Set, *object.Map:
			default:
				fatalKind(kindType, "'%s' needs two lists of numbers, got %s", fn, typeName(a))
			}
		}
		a, b = reduceItems(args[0]), reduceItems(args[1])
	case 3:
		c1, ok1 := args[1].(*object.String)
		c2, ok2 := args[2].(*object.String)
		if !ok1 || !ok2 {
			fatalf("function %q takes two lists, or rows and two column names", fn)
		}
		l, ok := args[0].(*object.List)
		if !ok || !allRecords(l.Elements) {
			fatalKind(kindType, "'%s' with column names needs a list of rows (maps or assembled values), got %s", fn, typeName(args[0]))
		}
		header, rows := recordRows(l.Elements)
		i1, i2 := -1, -1
		for i, h := range header {
			if h == c1.Value {
				i1 = i
			}
			if h == c2.Value {
				i2 = i
			}
		}
		for _, c := range []struct {
			i    int
			name string
		}{{i1, c1.Value}, {i2, c2.Value}} {
			if c.i < 0 {
				fatalKind(kindKey, "'%s': no column %q in these rows", fn, c.name)
			}
		}
		for _, r := range rows {
			a, b = append(a, orNone(r[i1])), append(b, orNone(r[i2]))
		}
	default:
		fatalf("function %q takes two lists, or rows and two column names, got %d argument(s)", fn, len(args))
	}
	if len(a) != len(b) {
		fatalKind(kindIndex, "'%s' needs two lists of the same length, got %d and %d", fn, len(a), len(b))
	}
	var xs, ys []float64
	for i := range a {
		_, an := a[i].(*object.None)
		_, bn := b[i].(*object.None)
		if an || bn {
			continue
		}
		x, _ := numbers(fn, a[i:i+1])
		y, _ := numbers(fn, b[i:i+1])
		xs, ys = append(xs, x[0]), append(ys, y[0])
	}
	needCount(fn, xs, 2)
	return xs, ys
}

func orNone(o object.Object) object.Object {
	if o == nil {
		return object.NoneValue
	}
	return o
}

func covarianceOf(xs, ys []float64) float64 {
	mx, my := meanOf(xs), meanOf(ys)
	s := sumOf(len(xs), func(i int) float64 { return (xs[i] - mx) * (ys[i] - my) })
	return s / float64(len(xs)-1)
}

// callStats runs one of data's statistics.
func callStats(name string, args []object.Object) object.Object {
	switch name {
	case "covariance":
		xs, ys := pairs(name, args)
		return &object.Float{Value: covarianceOf(xs, ys)}
	case "correlation":
		xs, ys := pairs(name, args)
		sx, sy := math.Sqrt(sumSquares(xs)), math.Sqrt(sumSquares(ys))
		if sx == 0 || sy == 0 {
			fatalKind(kindMath, "'correlation': one of the lists has every number the same, so it has no correlation")
		}
		return &object.Float{Value: covarianceOf(xs, ys) * float64(len(xs)-1) / (sx * sy)}
	case "mode":
		items, _ := statItems(name, args, 0)
		var best object.Object
		counts := map[string]int{}
		var order []object.Object
		for _, it := range items {
			if _, ok := it.(*object.None); ok {
				continue
			}
			k := object.Key(it)
			if counts[k] == 0 {
				order = append(order, it)
			}
			counts[k]++
		}
		top := 0
		for _, it := range order { // a tie goes to the one seen first
			if c := counts[object.Key(it)]; c > top {
				top, best = c, it
			}
		}
		if best == nil {
			fatalKind(kindIndex, "'mode' needs at least one value")
		}
		return best
	}

	extra := 0
	if name == "percentile" {
		extra = 1
	}
	items, rest := statItems(name, args, extra)
	xs, ints := numbers(name, items)
	switch name {
	case "mean":
		needCount(name, xs, 1)
		return &object.Float{Value: meanOf(xs)}
	case "median":
		needCount(name, xs, 1)
		s := sortedCopy(xs)
		n := len(s)
		if n%2 == 1 {
			return numberOf(s[n/2], ints) // the middle number itself
		}
		return &object.Float{Value: (s[n/2-1] + s[n/2]) / 2}
	case "variance", "stdev":
		needCount(name, xs, 2)
		v := sumSquares(xs) / float64(len(xs)-1)
		if name == "stdev" {
			v = math.Sqrt(v)
		}
		return &object.Float{Value: v}
	case "pvariance", "pstdev":
		needCount(name, xs, 1)
		v := sumSquares(xs) / float64(len(xs))
		if name == "pstdev" {
			v = math.Sqrt(v)
		}
		return &object.Float{Value: v}
	case "percentile":
		needCount(name, xs, 1)
		p, _, ok := numeric(rest[0])
		if !ok || p < 0 || p > 100 {
			fatalKind(kindMath, "'percentile' needs a percent from 0 to 100, got %s", object.Shown(rest[0]))
		}
		return &object.Float{Value: percentileOf(sortedCopy(xs), p)}
	case "zscores":
		needCount(name, xs, 2)
		m, sd := meanOf(xs), math.Sqrt(sumSquares(xs)/float64(len(xs)-1))
		if sd == 0 {
			fatalKind(kindMath, "'zscores': every number is the same, so none is any distance from the mean")
		}
		out := make([]float64, len(xs))
		for i, x := range xs {
			out[i] = (x - m) / sd
		}
		return numbersList(out, false)
	case "describe":
		needCount(name, xs, 1)
		s := sortedCopy(xs)
		var sd object.Object = object.NoneValue
		if len(xs) > 1 {
			sd = &object.Float{Value: math.Sqrt(sumSquares(xs) / float64(len(xs)-1))}
		}
		return resultMap(
			"count", object.Int(int64(len(xs))),
			"mean", &object.Float{Value: meanOf(xs)},
			"stdev", sd,
			"min", numberOf(s[0], ints),
			"25%", &object.Float{Value: percentileOf(s, 25)},
			"median", callStats("median", []object.Object{numbersList(xs, ints)}),
			"75%", &object.Float{Value: percentileOf(s, 75)},
			"max", numberOf(s[len(s)-1], ints),
		)
	}
	fatalKind(kindName, "no builtin function %q in \"data\"", name)
	return nil
}
