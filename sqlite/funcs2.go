// Copyright 2017-2026 Nigele McCoy
// Licensed under the Apache License, Version 2.0;
// see LICENSE and NOTICE.

package sqlite

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"math"
	"strconv"
	"strings"
	"unicode/utf8"
)

// More scalar functions: dates (datetime.go), printf/format, math, and
// the smaller text and value functions SQLite has built in.
func (r *runner) callMore(name string, args []Value) (Value, bool) {
	need := func(lo, hi int) {
		if len(args) < lo || len(args) > hi {
			if lo == hi {
				fail("SQL: %s() takes %d value(s), got %d", name, lo, len(args))
			}
			fail("SQL: %s() takes %d to %d values, got %d", name, lo, hi, len(args))
		}
	}
	switch name {
	case "date", "time", "datetime", "julianday", "unixepoch":
		return dateFunc(name, args), true
	case "strftime":
		if len(args) < 1 {
			fail("SQL: strftime() needs a format")
		}
		return dateFunc(name, args), true
	case "printf", "format":
		if len(args) == 0 {
			fail("SQL: %s() needs a format", name)
		}
		if args[0] == nil {
			return nil, true
		}
		return sqlPrintf(textValue(args[0]), args[1:]), true
	case "quote":
		need(1, 1)
		return quoteValue(args[0]), true
	case "char":
		var sb strings.Builder
		for _, a := range args {
			c := integerValue(a)
			if c < 0 || c > 0x10ffff {
				c = 0xfffd
			}
			sb.WriteRune(rune(c))
		}
		return sb.String(), true
	case "unicode":
		need(1, 1)
		if args[0] == nil {
			return nil, true
		}
		s := textValue(args[0])
		if s == "" {
			return nil, true
		}
		c, _ := utf8.DecodeRuneInString(s)
		return int64(c), true
	case "random":
		need(0, 0)
		var b [8]byte
		rand.Read(b[:])
		return int64(binary.BigEndian.Uint64(b[:])), true
	case "randomblob":
		need(1, 1)
		n := integerValue(args[0])
		if n < 1 {
			n = 1
		}
		if n > 1<<24 {
			fail("SQL: string or blob too big")
		}
		b := make([]byte, n)
		rand.Read(b)
		return b, true
	case "zeroblob":
		need(1, 1)
		n := integerValue(args[0])
		if n < 0 {
			n = 0
		}
		if n > 1<<26 {
			fail("SQL: string or blob too big")
		}
		return make([]byte, n), true
	case "sign":
		need(1, 1)
		switch x := numericOrNil(args[0]).(type) {
		case int64:
			switch {
			case x > 0:
				return int64(1), true
			case x < 0:
				return int64(-1), true
			}
			return int64(0), true
		case float64:
			switch {
			case x > 0:
				return int64(1), true
			case x < 0:
				return int64(-1), true
			}
			return int64(0), true
		}
		return nil, true
	case "concat":
		var sb strings.Builder
		for _, a := range args {
			sb.WriteString(textValue(a))
		}
		return sb.String(), true
	case "concat_ws":
		if len(args) < 1 {
			fail("SQL: concat_ws() needs a separator")
		}
		if args[0] == nil {
			return nil, true
		}
		var parts []string
		for _, a := range args[1:] {
			if a != nil {
				parts = append(parts, textValue(a))
			}
		}
		return strings.Join(parts, textValue(args[0])), true
	case "octet_length":
		need(1, 1)
		switch x := args[0].(type) {
		case nil:
			return nil, true
		case []byte:
			return int64(len(x)), true
		}
		return int64(len(textValue(args[0]))), true
	case "unhex":
		need(1, 2)
		if args[0] == nil {
			return nil, true
		}
		s := textValue(args[0])
		if len(args) == 2 && args[1] != nil {
			ignore := textValue(args[1])
			s = strings.Map(func(c rune) rune {
				if strings.ContainsRune(ignore, c) {
					return -1
				}
				return c
			}, s)
		}
		b, ok := decodeHex(s)
		if !ok {
			return nil, true
		}
		return b, true
	case "likely", "unlikely":
		need(1, 1)
		return args[0], true
	case "likelihood":
		need(2, 2)
		return args[0], true
	case "glob":
		need(2, 2)
		if args[0] == nil || args[1] == nil {
			return nil, true
		}
		return boolValue(glob([]rune(textValue(args[0])), []rune(textValue(args[1])))), true
	case "like":
		need(2, 3)
		if args[0] == nil || args[1] == nil {
			return nil, true
		}
		esc := rune(0)
		if len(args) == 3 {
			e := []rune(textValue(args[2]))
			if len(e) != 1 {
				fail("SQL: ESCAPE needs a single character")
			}
			esc = e[0]
		}
		return boolValue(like([]rune(textValue(args[0])), []rune(textValue(args[1])), esc)), true
	case "last_insert_rowid":
		need(0, 0)
		return r.db.lastRowid, true
	case "changes":
		need(0, 0)
		return r.db.changes, true
	case "total_changes":
		need(0, 0)
		return r.db.totalChanges, true
	case "sqlite_version":
		need(0, 0)
		return "3.46.1", true
	case "pi":
		need(0, 0)
		return math.Pi, true
	case "ceil", "ceiling", "floor", "trunc":
		need(1, 1)
		switch x := numericOrNil(args[0]).(type) {
		case int64:
			return x, true
		case float64:
			switch name {
			case "floor":
				return math.Floor(x), true
			case "trunc":
				return math.Trunc(x), true
			}
			return math.Ceil(x), true
		}
		return nil, true
	case "acos", "asin", "atan", "cos", "sin", "tan", "acosh", "asinh", "atanh", "cosh", "sinh", "tanh",
		"exp", "ln", "log10", "log2", "sqrt", "degrees", "radians":
		need(1, 1)
		x, ok := floatOrNil(args[0])
		if !ok {
			return nil, true
		}
		if (name == "ln" || name == "log10" || name == "log2") && x <= 0 {
			return nil, true
		}
		var v float64
		switch name {
		case "acos":
			v = math.Acos(x)
		case "asin":
			v = math.Asin(x)
		case "atan":
			v = math.Atan(x)
		case "cos":
			v = math.Cos(x)
		case "sin":
			v = math.Sin(x)
		case "tan":
			v = math.Tan(x)
		case "acosh":
			v = math.Acosh(x)
		case "asinh":
			v = math.Asinh(x)
		case "atanh":
			v = math.Atanh(x)
		case "cosh":
			v = math.Cosh(x)
		case "sinh":
			v = math.Sinh(x)
		case "tanh":
			v = math.Tanh(x)
		case "exp":
			v = math.Exp(x)
		case "ln":
			v = math.Log(x)
		case "log10":
			v = math.Log10(x)
		case "log2":
			v = math.Log2(x)
		case "sqrt":
			v = math.Sqrt(x)
		case "degrees":
			v = x * 180 / math.Pi
		case "radians":
			v = x * math.Pi / 180
		}
		if math.IsNaN(v) {
			return nil, true
		}
		return v, true
	case "log":
		need(1, 2)
		if len(args) == 1 {
			x, ok := floatOrNil(args[0])
			if !ok || x <= 0 {
				return nil, true
			}
			return math.Log10(x), true
		}
		b, ok1 := floatOrNil(args[0])
		x, ok2 := floatOrNil(args[1])
		if !ok1 || !ok2 || b <= 0 || b == 1 || x <= 0 {
			return nil, true
		}
		return math.Log(x) / math.Log(b), true
	case "pow", "power", "atan2", "mod":
		need(2, 2)
		a, ok1 := floatOrNil(args[0])
		b, ok2 := floatOrNil(args[1])
		if !ok1 || !ok2 {
			return nil, true
		}
		var v float64
		switch name {
		case "atan2":
			v = math.Atan2(a, b)
		case "mod":
			if b == 0 {
				return nil, true
			}
			v = math.Mod(a, b)
		default:
			v = math.Pow(a, b)
		}
		if math.IsNaN(v) {
			return nil, true
		}
		return v, true
	}
	return nil, false
}

func numericOrNil(v Value) Value {
	if v == nil {
		return nil
	}
	return numericValue(v)
}

func floatOrNil(v Value) (float64, bool) {
	if v == nil {
		return 0, false
	}
	return toFloat(v), true
}

// quoteValue writes v as an SQL literal.
func quoteValue(v Value) string {
	switch x := v.(type) {
	case nil:
		return "NULL"
	case int64:
		return strconv.FormatInt(x, 10)
	case float64:
		s := strconv.FormatFloat(x, 'g', 17, 64)
		if f, err := strconv.ParseFloat(formatReal(x), 64); err == nil && f == x {
			s = formatReal(x)
		}
		if !strings.ContainsAny(s, ".eEIN") {
			s += ".0"
		}
		return s
	case []byte:
		return "X'" + textValue(callHex(x)) + "'"
	}
	return "'" + strings.ReplaceAll(textValue(v), "'", "''") + "'"
}

func callHex(b []byte) string {
	const digits = "0123456789ABCDEF"
	out := make([]byte, 2*len(b))
	for i, c := range b {
		out[2*i], out[2*i+1] = digits[c>>4], digits[c&15]
	}
	return string(out)
}

// sqlPrintf is SQLite's printf: %d %i %u %f %e %E %g %G %x %X %o %s %z
// %c %q %Q %w %%, with flags - + space 0 # , and width and precision
// (* takes them from the arguments).
func sqlPrintf(format string, args []Value) string {
	var sb strings.Builder
	next := func() Value {
		if len(args) == 0 {
			return nil
		}
		a := args[0]
		args = args[1:]
		return a
	}
	for i := 0; i < len(format); i++ {
		c := format[i]
		if c != '%' {
			sb.WriteByte(c)
			continue
		}
		i++
		if i >= len(format) {
			break
		}
		var flags strings.Builder
		comma := false
		for i < len(format) && strings.IndexByte("-+ 0#!,", format[i]) >= 0 {
			switch format[i] {
			case ',':
				comma = true
			case '!':
			default:
				flags.WriteByte(format[i])
			}
			i++
		}
		width := ""
		if i < len(format) && format[i] == '*' {
			w := integerValue(next())
			if w < 0 {
				flags.WriteByte('-')
				w = -w
			}
			width = strconv.FormatInt(w, 10)
			i++
		} else {
			for i < len(format) && format[i] >= '0' && format[i] <= '9' {
				width += string(format[i])
				i++
			}
		}
		prec := ""
		hasPrec := false
		if i < len(format) && format[i] == '.' {
			hasPrec = true
			i++
			if i < len(format) && format[i] == '*' {
				prec = strconv.FormatInt(integerValue(next()), 10)
				i++
			} else {
				for i < len(format) && format[i] >= '0' && format[i] <= '9' {
					prec += string(format[i])
					i++
				}
			}
			if prec == "" {
				prec = "0"
			}
		}
		for i < len(format) && (format[i] == 'l' || format[i] == 'h') {
			i++
		}
		if i >= len(format) {
			break
		}
		spec := "%" + flags.String() + width
		if hasPrec {
			spec += "." + prec
		}
		switch t := format[i]; t {
		case '%':
			sb.WriteByte('%')
		case 'd', 'i', 'u':
			n := integerValue(next())
			s := fmt.Sprintf(strings.Replace(spec, "#", "", 1)+"d", n)
			if comma {
				s = withCommas(s)
			}
			sb.WriteString(s)
		case 'x', 'X', 'o':
			n := integerValue(next())
			sb.WriteString(fmt.Sprintf(spec+string(t), uint64(n)))
		case 'f', 'e', 'E', 'g', 'G':
			v := next()
			f := 0.0
			if v != nil {
				f = toFloat(v)
			}
			if !hasPrec && (t == 'f' || t == 'e' || t == 'E') {
				spec += ".6"
			}
			s := fmt.Sprintf(spec+string(t), f)
			if comma && t == 'f' {
				s = withCommas(s)
			}
			sb.WriteString(s)
		case 's', 'z':
			v := next()
			s := textValue(v)
			if hasPrec {
				p, _ := strconv.Atoi(prec)
				rs := []rune(s)
				if p < len(rs) {
					s = string(rs[:p])
				}
			}
			sb.WriteString(fmt.Sprintf("%"+flags.String()+width+"s", s))
		case 'c':
			s := textValue(next())
			r, _ := utf8.DecodeRuneInString(s)
			if s == "" {
				break
			}
			// A precision repeats the character: %.3c of 'x' is xxx.
			n := 1
			if hasPrec {
				n, _ = strconv.Atoi(prec)
				if n > 1<<24 {
					fail("SQL: string or blob too big")
				}
			}
			sb.WriteString(fmt.Sprintf("%"+flags.String()+width+"s", strings.Repeat(string(r), n)))
		case 'q':
			sb.WriteString(fmt.Sprintf("%"+flags.String()+width+"s", strings.ReplaceAll(textValue(next()), "'", "''")))
		case 'Q':
			v := next()
			if v == nil {
				sb.WriteString("NULL")
			} else {
				sb.WriteString("'" + strings.ReplaceAll(textValue(v), "'", "''") + "'")
			}
		case 'w':
			sb.WriteString(strings.ReplaceAll(textValue(next()), `"`, `""`))
		default:
			sb.WriteByte('%')
			sb.WriteByte(t)
		}
	}
	return sb.String()
}

// withCommas puts thousands separators in the digits of a formatted
// number.
func withCommas(s string) string {
	start := strings.IndexAny(s, "0123456789")
	if start < 0 {
		return s
	}
	end := start
	for end < len(s) && s[end] >= '0' && s[end] <= '9' {
		end++
	}
	digits := s[start:end]
	var out []byte
	for i := range digits {
		if i > 0 && (len(digits)-i)%3 == 0 {
			out = append(out, ',')
		}
		out = append(out, digits[i])
	}
	return s[:start] + string(out) + s[end:]
}
