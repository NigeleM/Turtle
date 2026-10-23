// Copyright 2017-2026 Nigele McCoy
// Licensed under the Apache License, Version 2.0;
// see LICENSE and NOTICE.

package evaluator

import (
	"math"
	"math/big"

	"Turtle/object"
)

// Whole numbers are unlimited, as in Python. A result that fits an int64
// stays a plain number (the fast path); one that doesn't is worked out
// exactly with math/big. The overflow error is gone: the answer is simply
// bigger.

// intArithFits is a whole-number operation on two int64s, when its answer
// fits an int64. Otherwise ok is false and the caller uses bigArith.
func intArithFits(op string, a, b int64) (num, bool) {
	switch op {
	case "+":
		r := a + b
		if (b > 0 && r < a) || (b < 0 && r > a) {
			return num{}, false
		}
		return num{isInt: true, i: r}, true
	case "-":
		r := a - b
		if (b < 0 && r < a) || (b > 0 && r > a) {
			return num{}, false
		}
		return num{isInt: true, i: r}, true
	case "*":
		if _, ok := mulFits(a, b); !ok {
			return num{}, false
		}
		return num{isInt: true, i: a * b}, true
	case "div":
		if b == 0 {
			fatalKind(kindMath, "division by zero")
		}
		if b == -1 && a == math.MinInt64 {
			return num{}, false
		}
		return num{isInt: true, i: a / b}, true
	case "%":
		if b == 0 {
			fatalKind(kindMath, "modulo by zero")
		}
		return num{isInt: true, i: a % b}, true
	}
	return num{}, false
}

// bigOf is a whole number as a big.Int.
func bigOf(n num) *big.Int {
	if n.big != nil {
		return n.big
	}
	return big.NewInt(n.i)
}

// numOfBig is a big.Int as a number, plain when it fits an int64.
func numOfBig(z *big.Int) num {
	if z.IsInt64() {
		return num{isInt: true, i: z.Int64()}
	}
	return num{isInt: true, big: z}
}

// maxIntBits is how big a whole number may get: about three million
// digits. Integers are otherwise unlimited, as in Python, but arithmetic
// that would pass this stops with an error before doing the work, so one
// line (2 at pow 10000000000) can't take all the memory or minutes of time.
const maxIntBits = 10_000_000

// maxIntDigits is how long text may be to read as a whole number. Reading
// takes time growing with the square of the length (a million digits takes
// about half a second), so a long enough text from outside could stall a
// program; Python limits it for the same reason.
const maxIntDigits = 1_000_000

// tooBig stops a whole number past maxIntBits.
func tooBig() {
	fatalKind(kindMath, "integer too big: past %d bits (about 3 million digits), where Turtle stops so one number can't take all the memory", maxIntBits)
}

// bigArith is + - * div % on whole numbers, exactly.
func bigArith(op string, a, b num) num {
	x, y := bigOf(a), bigOf(b)
	z := new(big.Int)
	switch op {
	case "+":
		z.Add(x, y)
	case "-":
		z.Sub(x, y)
	case "*":
		// A product has at least one bit fewer than its two factors
		// together: past the limit by then, it isn't worked out.
		if x.Sign() != 0 && y.Sign() != 0 && x.BitLen()+y.BitLen()-1 > maxIntBits {
			tooBig()
		}
		z.Mul(x, y)
	case "div":
		if y.Sign() == 0 {
			fatalKind(kindMath, "division by zero")
		}
		z.Quo(x, y)
	case "%":
		if y.Sign() == 0 {
			fatalKind(kindMath, "modulo by zero")
		}
		z.Rem(x, y)
	}
	if z.BitLen() > maxIntBits {
		tooBig()
	}
	return numOfBig(z)
}

// bigInfix is an operator with a big whole number on one side (and a whole
// number on the other): arithmetic, division, and comparisons, all exact.
// ok is false when neither side is big, or a float is involved (the float
// rules then apply).
func bigInfix(op string, left, right object.Object) (object.Object, bool) {
	_, lBig := left.(*object.BigInt)
	_, rBig := right.(*object.BigInt)
	if !lBig && !rBig {
		return nil, false
	}
	x, lOK := object.BigOf(left)
	y, rOK := object.BigOf(right)
	if !lOK || !rOK {
		return nil, false
	}
	switch op {
	case "<", ">", "<=", ">=", "==", "!=":
		c := x.Cmp(y)
		switch op {
		case "<":
			return object.Bool(c < 0), true
		case ">":
			return object.Bool(c > 0), true
		case "<=":
			return object.Bool(c <= 0), true
		case ">=":
			return object.Bool(c >= 0), true
		case "==":
			return object.Bool(c == 0), true
		}
		return object.Bool(c != 0), true
	case "+", "-", "*", "div", "%":
		ln, _ := asNum(left)
		rn, _ := asNum(right)
		return numArith(op, ln, rn).box(), true
	case "/":
		if y.Sign() == 0 {
			fatalKind(kindMath, "division by zero")
		}
		// The exact quotient, as a float: 7 / 2 is 3.5, as for plain numbers.
		return &object.Float{Value: ratFloat(x, y)}, true
	}
	return nil, false
}

// wholeOf is a float's whole part as an integer object (exact, past 64 bits),
// or an error for infinity and NaN, which have no whole part.
func wholeOf(what string, f float64) object.Object {
	o, ok := object.FloatToInt(f)
	if !ok {
		fatalKind(kindMath, "%s: %s is not a number, so it has no whole-number value", what, (&object.Float{Value: f}).Inspect())
	}
	return o
}

// powBig is b to the power e, exactly (e >= 0). With |b| of at least 2
// the result has at least (bits of b - 1) * e + 1 bits: past maxIntBits by
// then, it isn't worked out.
func powBig(b *big.Int, e int64) object.Object {
	if bits := int64(b.BitLen()); bits > 1 && e > 0 && bits-1 > maxIntBits/e {
		tooBig()
	}
	z := new(big.Int).Exp(b, big.NewInt(e), nil)
	if z.BitLen() > maxIntBits {
		tooBig()
	}
	return object.IntOf(z)
}

// ratFloat is x / y, exact and rounded once to a float.
func ratFloat(x, y *big.Int) float64 {
	f, _ := new(big.Rat).SetFrac(x, y).Float64()
	return f
}

// exactAsFloat is whether a whole number converts to a float without
// rounding (up to 2^53).
func exactAsFloat(i int64) bool {
	return i >= -(1<<53) && i <= 1<<53
}

// isBigInt is whether o is a whole number past 64 bits.
func isBigInt(o object.Object) bool {
	_, ok := o.(*object.BigInt)
	return ok
}
