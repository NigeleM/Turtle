package sqlite

import (
	"math"
	"strings"
	"unicode/utf8"
)

// callFunc runs a scalar SQL function: abs, coalesce, ifnull, nullif,
// iif, length, lower, upper, substr, trim/ltrim/rtrim, replace, instr,
// round, typeof, hex, and min/max with two or more values.
func (r *runner) callFunc(c *callExpr) Value {
	if c.star {
		fail("SQL: %s(*) isn't a function", c.name)
	}
	args := make([]Value, len(c.args))
	for i, a := range c.args {
		args[i] = r.eval(a)
	}
	need := func(lo, hi int) {
		if len(args) < lo || len(args) > hi {
			if lo == hi {
				fail("SQL: %s() takes %d value(s), got %d", c.name, lo, len(args))
			}
			fail("SQL: %s() takes %d to %d values, got %d", c.name, lo, hi, len(args))
		}
	}
	switch c.name {
	case "abs":
		need(1, 1)
		switch x := numericValue(args[0]).(type) {
		case int64:
			if x < 0 {
				if x == math.MinInt64 {
					fail("SQL: integer overflow in abs()")
				}
				return -x
			}
			return x
		case float64:
			return math.Abs(x)
		}
		return nil
	case "coalesce", "ifnull":
		if c.name == "ifnull" {
			need(2, 2)
		} else if len(args) < 2 {
			fail("SQL: coalesce() takes at least 2 values")
		}
		for _, a := range args {
			if a != nil {
				return a
			}
		}
		return nil
	case "nullif":
		need(2, 2)
		if args[0] != nil && args[1] != nil && compare(args[0], args[1]) == 0 {
			return nil
		}
		return args[0]
	case "iif":
		need(3, 3)
		if truthy(args[0]) {
			return args[1]
		}
		return args[2]
	case "length":
		need(1, 1)
		switch x := args[0].(type) {
		case nil:
			return nil
		case []byte:
			return int64(len(x))
		}
		return int64(utf8.RuneCountInString(textValue(args[0])))
	case "lower", "upper":
		need(1, 1)
		if args[0] == nil {
			return nil
		}
		// Like SQLite, only A-Z and a-z change.
		b := []byte(textValue(args[0]))
		for i, ch := range b {
			if c.name == "lower" && ch >= 'A' && ch <= 'Z' {
				b[i] = ch + 32
			} else if c.name == "upper" && ch >= 'a' && ch <= 'z' {
				b[i] = ch - 32
			}
		}
		return string(b)
	case "substr", "substring":
		need(2, 3)
		if args[0] == nil || args[1] == nil {
			return nil
		}
		s := []rune(textValue(args[0]))
		start := integerValue(args[1])
		n := int64(len(s)) + 1
		if len(args) == 3 {
			if args[2] == nil {
				return nil
			}
			n = integerValue(args[2])
		}
		// 1-based; a negative start counts from the end; 0 is before the
		// first character.
		if start < 0 {
			start += int64(len(s)) + 1
		}
		if n < 0 {
			start, n = start+n, -n
		}
		from, to := start-1, start-1+n
		if from < 0 {
			from = 0
		}
		if to > int64(len(s)) {
			to = int64(len(s))
		}
		if from >= to {
			return ""
		}
		return string(s[from:to])
	case "trim", "ltrim", "rtrim":
		need(1, 2)
		if args[0] == nil {
			return nil
		}
		cut := " "
		if len(args) == 2 {
			if args[1] == nil {
				return nil
			}
			cut = textValue(args[1])
		}
		s := textValue(args[0])
		switch c.name {
		case "ltrim":
			return strings.TrimLeft(s, cut)
		case "rtrim":
			return strings.TrimRight(s, cut)
		}
		return strings.Trim(s, cut)
	case "replace":
		need(3, 3)
		if args[0] == nil || args[1] == nil || args[2] == nil {
			return nil
		}
		old := textValue(args[1])
		if old == "" {
			return textValue(args[0])
		}
		return strings.ReplaceAll(textValue(args[0]), old, textValue(args[2]))
	case "instr":
		need(2, 2)
		if args[0] == nil || args[1] == nil {
			return nil
		}
		s, sub := textValue(args[0]), textValue(args[1])
		i := strings.Index(s, sub)
		if i < 0 {
			return int64(0)
		}
		return int64(utf8.RuneCountInString(s[:i]) + 1)
	case "round":
		need(1, 2)
		if args[0] == nil {
			return nil
		}
		digits := int64(0)
		if len(args) == 2 {
			digits = integerValue(args[1])
		}
		f := toFloat(args[0])
		p := math.Pow(10, float64(digits))
		// Half away from zero, like SQLite.
		if f < 0 {
			return -math.Floor(-f*p+0.5) / p
		}
		return math.Floor(f*p+0.5) / p
	case "typeof":
		need(1, 1)
		switch args[0].(type) {
		case nil:
			return "null"
		case int64:
			return "integer"
		case float64:
			return "real"
		case string:
			return "text"
		}
		return "blob"
	case "hex":
		need(1, 1)
		var b []byte
		if x, ok := args[0].([]byte); ok {
			b = x
		} else {
			b = []byte(textValue(args[0]))
		}
		const digits = "0123456789ABCDEF"
		out := make([]byte, 2*len(b))
		for i, c := range b {
			out[2*i], out[2*i+1] = digits[c>>4], digits[c&15]
		}
		return string(out)
	case "min", "max":
		// Scalar form (two or more values); one value is the aggregate.
		var best Value
		for i, a := range args {
			if a == nil {
				return nil
			}
			if i == 0 || (c.name == "min" && compare(a, best) < 0) || (c.name == "max" && compare(a, best) > 0) {
				best = a
			}
		}
		return best
	}
	if v, ok := r.callMore(c.name, args); ok {
		return v
	}
	if v, ok := r.callJSON(c, args); ok {
		return v
	}
	fail("SQL: no such function: %s", c.name)
	return nil
}
