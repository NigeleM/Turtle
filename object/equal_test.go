// Copyright 2017-2026 Nigele McCoy
// Licensed under the Apache License, Version 2.0;
// see LICENSE and NOTICE.

package object

import (
	"math"
	"math/rand"
	"testing"
)

// Floats compare as they show, to 15 significant digits. These check
// that the rule holds together on many random numbers: equal exactly
// when they show the same, ordering consistent with equality, keys
// matching equality, and numbers that are clearly apart never equal.
func TestFloatComparisonIsConsistent(t *testing.T) {
	r := rand.New(rand.NewSource(1))
	random := func() float64 {
		switch r.Intn(5) {
		case 0:
			return float64(r.Intn(2000)-1000) / 10 // tenths: where 0.1 + 0.2 lives
		case 1:
			return r.Float64() * math.Pow(10, float64(r.Intn(40)-20))
		case 2:
			return -r.Float64() * 1000
		case 3:
			return float64(r.Intn(100))
		}
		return r.NormFloat64()
	}
	for i := 0; i < 20000; i++ {
		a := random()
		// b: a, nudged by one binary step or a few, or a sum that should equal it.
		var b float64
		switch r.Intn(4) {
		case 0:
			b = math.Nextafter(a, math.Inf(1))
		case 1:
			b = a + a*1e-16*float64(r.Intn(5))
		case 2:
			b = random()
		default:
			x := float64(r.Intn(100)) / 10
			a, b = x+0.1+0.2, x+0.3
		}
		fa, fb := &Float{Value: a}, &Float{Value: b}
		eq := Equal(fa, fb)
		if eq != (fa.Inspect() == fb.Inspect()) {
			t.Fatalf("%v and %v: equal %v, but they show %s and %s", a, b, eq, fa.Inspect(), fb.Inspect())
		}
		if eq != (Key(fa) == Key(fb)) {
			t.Fatalf("%v and %v: equal %v, keys %s and %s", a, b, eq, Key(fa), Key(fb))
		}
		c, back := CompareFloats(a, b), CompareFloats(b, a)
		if (c == 0) != eq || c != -back {
			t.Fatalf("%v and %v: compare %d / %d, equal %v", a, b, c, back, eq)
		}
		if math.Abs(a-b) > 1e-9*math.Max(math.Abs(a), math.Abs(b)) && eq {
			t.Fatalf("%v and %v are clearly apart but compare equal", a, b)
		}
	}
}

func TestFloatComparisonExamples(t *testing.T) {
	f := func(v float64) *Float { return &Float{Value: v} }
	i := func(v int64) *Integer { return &Integer{Value: v} }
	cases := []struct {
		a, b Object
		eq   bool
	}{
		{f(0.1 + 0.2), f(0.3), true},
		{f(0.6000000000000001), f(0.6), true},
		{f(1.1 * 3), f(3.3), true},
		{f(2.0000000000000004), i(2), true},
		{f(0.3), f(0.30000000000001), false}, // differs in the 14th digit
		{f(1e-20), f(0), false},
		{f(0), f(math.Copysign(0, -1)), true},
		{i(9007199254740993), i(9007199254740992), false}, // integers exact
		{f(math.Inf(1)), f(math.Inf(1)), true},
		{f(math.NaN()), f(math.NaN()), false},
	}
	for _, c := range cases {
		if got := Equal(c.a, c.b); got != c.eq {
			t.Errorf("%s == %s: got %v, want %v", c.a.Inspect(), c.b.Inspect(), got, c.eq)
		}
	}
	if CompareFloats(0.30000000000000004, 0.3) != 0 || CompareFloats(0.3, 0.31) != -1 {
		t.Error("ordering")
	}
}
