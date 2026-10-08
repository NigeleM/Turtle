package evaluator

import (
	"math"
	"math/rand"
	"strings"
	"time"

	"Turtle/ast"
	"Turtle/object"
)

// The random library (import random): random values of any shape, and
// pick / shuffle / sample / chance.
//
//	die = random integer from 1 to 6
//	nums = random list of 5 integers from 0 to 9
//	orders = random list of 3 Order [string, integer from 1 to 10, float]
//
// "import random" also makes the variable seed (none). none gives
// different values each run; a whole number gives the same values every
// run from the point it's set, so a run can be repeated exactly.

const (
	seedName       = "seed"
	defaultLetters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	digitChars     = "0123456789"
	maxRandomCount = 10_000_000 // a count above this is almost surely a mistake
)

// Without from/to: integers from -1000 to 1000, floats from 0 up to 1,
// dates and times from 2000 through 2030, strings and collections 0 or 1 to
// 10 long.
var (
	defaultDateFrom = time.Date(2000, 1, 1, 0, 0, 0, 0, time.Local)
	defaultDateTo   = time.Date(2030, 12, 31, 0, 0, 0, 0, time.Local)
	defaultTimeTo   = time.Date(2030, 12, 31, 23, 59, 59, 0, time.Local)
)

// defineSeed gives the importing file its seed setting, unless the file
// already has a variable of that name.
func defineSeed(env *object.Environment) {
	if _, ok := env.Get(seedName); !ok {
		env.Set(seedName, object.NoneValue)
	}
}

// randomSource is the random number source, started again each time
// the program sets seed: from the seed when it's a number, from the clock
// when it's none. Every "seed = 7" makes a new value, so setting the
// same number again starts the same values over.
//
// Code with no seed variable (the random library's own Turtle code) uses
// the source as it is.
func (it *Interpreter) randomSource(env *object.Environment) *rand.Rand {
	var v object.Object = object.NoneValue
	got, ok := env.Get(seedName)
	if ok {
		v = got
	}
	if it.rng != nil && (v == it.rngSeed || !ok) {
		return it.rng
	}
	var seed int64
	switch x := v.(type) {
	case *object.None:
		seed = time.Now().UnixNano()
	case *object.Integer:
		seed = x.Value
	default:
		fatalf("seed must be a whole number (the same values every run) or none (different values each run), got %s", v.Inspect())
	}
	it.rng = rand.New(rand.NewSource(seed))
	it.rngSeed = v
	return it.rng
}

// randomUpTo is a whole number from 0 to n-1 (n > 0), every one equally
// likely, for any n up to 2^64.
func randomUpTo(rng *rand.Rand, n uint64) uint64 {
	if n&(n-1) == 0 {
		return rng.Uint64() & (n - 1)
	}
	limit := math.MaxUint64 - math.MaxUint64%n
	for {
		if v := rng.Uint64(); v < limit {
			return v % n
		}
	}
}

// randomBetween is a whole number from lo to hi, both included.
func randomBetween(rng *rand.Rand, lo, hi int64) int64 {
	span := uint64(hi-lo) + 1
	if span == 0 { // every int64
		return int64(rng.Uint64())
	}
	return lo + int64(randomUpTo(rng, span))
}

func (it *Interpreter) evalRandom(s *ast.Shape, env *object.Environment) object.Object {
	return it.randomValue(s, env, it.randomSource(env))
}

func (it *Interpreter) randomValue(s *ast.Shape, env *object.Environment, rng *rand.Rand) object.Object {
	switch s.Kind {
	case "integer":
		lo, hi := int64(-1000), int64(1000)
		if s.From != nil {
			lo = it.shapeInteger(s, "from", s.From, env)
			hi = it.shapeInteger(s, "to", s.To, env)
			if lo > hi {
				fatalKind(kindMath, "random integer from %d to %d: from is more than to", lo, hi)
			}
		}
		return object.Int(randomBetween(rng, lo, hi))
	case "float":
		lo, hi := 0.0, 1.0
		if s.From != nil {
			lo = it.shapeNumber(s, "from", s.From, env)
			hi = it.shapeNumber(s, "to", s.To, env)
			if lo > hi {
				fatalKind(kindMath, "random float from %s to %s: from is more than to", (&object.Float{Value: lo}).Inspect(), (&object.Float{Value: hi}).Inspect())
			}
		}
		v := lo + rng.Float64()*(hi-lo)
		if s.Places != nil {
			places := it.shapeInteger(s, "rounded to", s.Places, env)
			if places < 0 || places > 15 {
				fatalKind(kindMath, "random float ... rounded to %d: places must be from 0 to 15", places)
			}
			m := math.Pow(10, float64(places))
			r := math.Round(v*m) / m
			if r > hi {
				r = math.Floor(v*m) / m
			}
			if r < lo {
				r = math.Ceil(v*m) / m
			}
			v = r
		}
		return &object.Float{Value: v}
	case "string", "digits":
		chars := defaultLetters
		if s.Kind == "digits" {
			chars = digitChars
		}
		if s.Chars != nil {
			c, ok := it.evalExpression(s.Chars, env).(*object.String)
			if !ok || c.Value == "" {
				fatalf("random string from ...: give the characters to use as a string, e.g. from \"ABC123\"")
			}
			chars = c.Value
		}
		n := it.shapeLength(s, env, rng)
		runes := []rune(chars)
		var b strings.Builder
		for range n {
			b.WriteRune(runes[randomUpTo(rng, uint64(len(runes)))])
		}
		return &object.String{Value: b.String()}
	case "digit":
		return &object.String{Value: string(digitChars[rng.Intn(10)])}
	case "letter":
		return &object.String{Value: string(defaultLetters[rng.Intn(len(defaultLetters))])}
	case "boolean":
		return object.Bool(rng.Intn(2) == 1)
	case "date":
		from, to := defaultDateFrom, defaultDateTo
		if s.From != nil {
			from = dayOf(it.shapeDate(s, "from", s.From, env))
			to = dayOf(it.shapeDate(s, "to", s.To, env))
		}
		days := daysBetween(from, to)
		if days < 0 {
			fatalKind(kindDate, "random date from %s to %s: from is after to", from.Format(time.DateOnly), to.Format(time.DateOnly))
		}
		n := randomBetween(rng, 0, days)
		return &object.Date{Time: time.Date(from.Year(), from.Month(), from.Day()+int(n), 0, 0, 0, 0, time.Local)}
	case "time":
		from, to := defaultDateFrom, defaultTimeTo
		if s.From != nil {
			from = it.shapeDate(s, "from", s.From, env)
			to = it.shapeDate(s, "to", s.To, env)
		}
		if from.After(to) {
			fatalKind(kindDate, "random time from %s to %s: from is after to", from.Format(time.DateTime), to.Format(time.DateTime))
		}
		sec := randomBetween(rng, from.Unix(), to.Unix())
		return &object.Date{Time: time.Unix(sec, 0).In(time.Local)}
	case "list":
		n := it.shapeCount(s, env, rng)
		out := &object.List{Elements: make([]object.Object, 0, n)}
		for range n {
			out.Elements = append(out.Elements, it.randomValue(s.Item, env, rng))
		}
		return out
	case "set":
		n := it.shapeCount(s, env, rng)
		out := &object.Set{}
		it.distinct(s, n, func() bool { return out.Add(it.randomValue(s.Item, env, rng)) }, func() int { return len(out.Elements) })
		return out
	case "map":
		n := it.shapeCount(s, env, rng)
		out := object.NewMap()
		it.distinct(s, n, func() bool {
			k := it.randomValue(s.Key, env, rng)
			if _, there := out.Get(k); there {
				return false
			}
			out.Put(k, it.randomValue(s.Item, env, rng))
			return true
		}, func() int { return out.Len() })
		return out
	case "assembled":
		shape := assembledShape(env, s.TypeName)
		if len(s.Fields) != len(shape.Fields) {
			fatalf("random %s: %s has %d fields (%s), so give %d kinds, got %d", s.Describe(), shape.Name, len(shape.Fields), strings.Join(shape.Fields, ", "), len(shape.Fields), len(s.Fields))
		}
		vals := make([]object.Object, len(s.Fields))
		for i, f := range s.Fields {
			vals[i] = it.randomValue(f, env, rng)
		}
		return &object.Assembly{Shape: shape, Values: vals}
	}
	fatalf("random: unknown kind %q", s.Kind)
	return nil
}

// distinct adds values until there are n different ones, giving up when
// the shape can't make that many (a set of 5 integers from 1 to 3).
func (it *Interpreter) distinct(s *ast.Shape, n int, add func() bool, size func() int) {
	tries := 0
	for size() < n {
		if !add() {
			tries++
			if tries > 1000+100*n {
				fatalKind(kindMath, "random %s: wanted %d different values, but found only %d (is the range too small?)", s.Describe(), n, size())
			}
		}
	}
}

// assembledShape finds the assembled type called name, as a call to it
// would.
func assembledShape(env *object.Environment, name string) *object.Shape {
	var fn *object.Function
	if v, ok := env.Get(name); ok {
		fn, _ = v.(*object.Function)
	}
	if fn == nil {
		fn, _ = env.GetFunction(name)
	}
	if fn == nil {
		if im := resolveImported(env, name); im != nil {
			fn, _ = im.Module.Function(name)
		}
	}
	if fn == nil {
		fatalKind(kindName, "random %s [...]: no assembled type called %q (make one with assemble %s [field, ...])", name, name, name)
	}
	if fn.Shape == nil {
		fatalf("random %s [...]: %q is a function, not an assembled type", name, name)
	}
	return fn.Shape
}

func (it *Interpreter) shapeInteger(s *ast.Shape, what string, e ast.Expression, env *object.Environment) int64 {
	v := it.evalExpression(e, env)
	n, ok := v.(*object.Integer)
	if !ok {
		fatalf("random %s %s: needs a whole number, got %s", s.Describe(), what, v.Inspect())
	}
	return n.Value
}

func (it *Interpreter) shapeNumber(s *ast.Shape, what string, e ast.Expression, env *object.Environment) float64 {
	v := it.evalExpression(e, env)
	f, _, ok := numeric(v)
	if !ok {
		fatalf("random %s %s: needs a number, got %s", s.Describe(), what, v.Inspect())
	}
	return f
}

// shapeDate reads a date limit: a date, or text such as "2026-01-31".
func (it *Interpreter) shapeDate(s *ast.Shape, what string, e ast.Expression, env *object.Environment) time.Time {
	v := it.evalExpression(e, env)
	switch d := v.(type) {
	case *object.Date:
		return d.Time
	case *object.String:
		if t, ok := tryParseDate(d.Value); ok {
			return t.(*object.Date).Time
		}
	}
	fatalKind(kindDate, "random %s %s: needs a date, or text such as \"2026-01-31\", got %s", s.Describe(), what, v.Inspect())
	return time.Time{}
}

func dayOf(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.Local)
}

// daysBetween counts calendar days, whatever daylight saving does.
func daysBetween(from, to time.Time) int64 {
	a := time.Date(from.Year(), from.Month(), from.Day(), 0, 0, 0, 0, time.UTC)
	b := time.Date(to.Year(), to.Month(), to.Day(), 0, 0, 0, 0, time.UTC)
	return int64(b.Sub(a).Hours() / 24)
}

// shapeCount is how many items a list, set or map gets: N, a number
// from N to M, or 0 to 10 when not given.
func (it *Interpreter) shapeCount(s *ast.Shape, env *object.Environment, rng *rand.Rand) int {
	return it.shapeRange(s, "count", s.Count, s.CountTo, 0, env, rng)
}

// shapeLength is how long a string is: "of N", "of N to M", or 1 to 10.
func (it *Interpreter) shapeLength(s *ast.Shape, env *object.Environment, rng *rand.Rand) int {
	return it.shapeRange(s, "length", s.Length, s.LengthTo, 1, env, rng)
}

func (it *Interpreter) shapeRange(s *ast.Shape, what string, from, to ast.Expression, min int64, env *object.Environment, rng *rand.Rand) int {
	lo, hi := min, int64(10)
	if from != nil {
		lo = it.shapeInteger(s, what, from, env)
		hi = lo
		if to != nil {
			hi = it.shapeInteger(s, what, to, env)
		}
	}
	if lo < 0 || hi < lo || hi > maxRandomCount {
		fatalKind(kindMath, "random %s: the %s must be from 0 to %d, and the first no more than the second (got %d to %d)", s.Describe(), what, maxRandomCount, lo, hi)
	}
	return int(randomBetween(rng, lo, hi))
}

// ---- shuffle and sample (pick and chance are in lib/random.trt) ----

func (it *Interpreter) callRandom(name string, args []object.Object, env *object.Environment) object.Object {
	rng := it.randomSource(env)
	switch name {
	case "shuffle":
		requireFuncArgs(name, args, 1)
		items := append([]object.Object{}, randomItems(name, args[0])...)
		rng.Shuffle(len(items), func(i, j int) { items[i], items[j] = items[j], items[i] })
		if _, isText := args[0].(*object.String); isText {
			var b strings.Builder
			for _, c := range items {
				b.WriteString(c.(*object.String).Value)
			}
			return &object.String{Value: b.String()}
		}
		return &object.List{Elements: items}
	case "sample":
		requireFuncArgs(name, args, 2)
		items := randomItems(name, args[0])
		n, ok := args[1].(*object.Integer)
		if !ok {
			fatalf("sample: how many must be a whole number, got %s", object.Shown(args[1]))
		}
		if n.Value < 0 || n.Value > int64(len(items)) {
			fatalKind(kindIndex, "sample: can't take %d different items from %d", n.Value, len(items))
		}
		picked := make([]object.Object, n.Value)
		for i, p := range rng.Perm(len(items))[:n.Value] {
			picked[i] = items[p]
		}
		return &object.List{Elements: picked}
	}
	fatalKind(kindName, "no random function %q", name)
	return nil
}

// randomItems are what shuffle and sample choose from: a list's or set's
// items, a map's keys, a string's characters.
func randomItems(fn string, x object.Object) []object.Object {
	if items, ok := changeItems(x); ok {
		return items
	}
	fatalf("%s needs a list, set, map or string, got %s", fn, object.Shown(x))
	return nil
}
