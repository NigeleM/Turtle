// Copyright 2017-2026 Nigele McCoy
// Licensed under the Apache License, Version 2.0;
// see LICENSE and NOTICE.

package object

import (
	"math"
	"math/big"
)

// BigInt is a whole number past the 64-bit range. Turtle's integers are
// unlimited, as Python's are: a result that fits an int64 is always an
// *Integer (so the fast path stays), and one that doesn't is a *BigInt. A
// BigInt therefore never equals an *Integer.
type BigInt struct{ V *big.Int }

func (b *BigInt) Type() Type      { return INTEGER }
func (b *BigInt) Inspect() string { return b.V.String() }

// IntOf is a whole number as an integer object: an *Integer when it fits an
// int64, otherwise a *BigInt.
func IntOf(b *big.Int) Object {
	if b.IsInt64() {
		return &Integer{Value: b.Int64()}
	}
	return &BigInt{V: b}
}

// BigOf is an integer object's value as a big.Int, and whether o is one.
func BigOf(o Object) (*big.Int, bool) {
	switch v := o.(type) {
	case *Integer:
		return big.NewInt(v.Value), true
	case *BigInt:
		return v.V, true
	}
	return nil, false
}

// FloatToInt is a finite float as the exact whole number it holds, after
// dropping any fraction (toward zero, as change ... to integer does). It is
// false for infinity and NaN, which aren't numbers of any size.
func FloatToInt(f float64) (Object, bool) {
	if math.IsNaN(f) || math.IsInf(f, 0) {
		return nil, false
	}
	bi, _ := new(big.Float).SetFloat64(math.Trunc(f)).Int(nil)
	return IntOf(bi), true
}

// BigFloat is a BigInt as a float, rounded to the nearest float, as Python's
// int-to-float conversion does.
func BigFloat(b *big.Int) float64 {
	f, _ := new(big.Float).SetInt(b).Float64()
	return f
}

// equalBigFloat is whether an exact whole number and a float are the same
// number, compared exactly (10**30 is not 1e30, as in Python).
func equalBigFloat(b *big.Int, f float64) bool {
	if math.IsNaN(f) || math.IsInf(f, 0) || f != math.Trunc(f) {
		return false
	}
	bi, _ := new(big.Float).SetFloat64(f).Int(nil)
	return bi.Cmp(b) == 0
}
