package evaluator

import (
	"math"
	"strconv"
	"strings"
	"unicode/utf8"

	"Turtle/object"
)

// Formatting numbers and padding text, as methods (no import):
//
//	3.5 at fixed[2]              "3.50"
//	1234567 at commas            "1,234,567"
//	"7" at padleft[3, "0"]       "007"
//	"ab" at padright[5, "."]     "ab..."

// fixedText writes f with exactly places decimals.
func fixedText(method string, receiver object.Object, args []object.Object) object.Object {
	requireArgs(method, args, 1)
	f, _, _ := numeric(receiver)
	n, ok := args[0].(*object.Integer)
	if !ok || n.Value < 0 || n.Value > 20 {
		fatalf("'fixed' takes how many decimal places, from 0 to 20, got %s", object.Shown(args[0]))
	}
	if math.IsInf(f, 0) || math.IsNaN(f) {
		return &object.String{Value: strconv.FormatFloat(f, 'f', -1, 64)}
	}
	return &object.String{Value: strconv.FormatFloat(f, 'f', int(n.Value), 64)}
}

// commasText groups a number's whole part in threes: 1,234,567.5.
func commasText(method string, receiver object.Object, args []object.Object) object.Object {
	requireArgs(method, args, 0)
	var text string
	switch n := receiver.(type) {
	case *object.Integer:
		text = strconv.FormatInt(n.Value, 10)
	case *object.Float:
		text = strconv.FormatFloat(n.Value, 'f', -1, 64)
	}
	sign := ""
	if strings.HasPrefix(text, "-") {
		sign, text = "-", text[1:]
	}
	whole, frac, hasFrac := strings.Cut(text, ".")
	var b strings.Builder
	for i, d := range whole {
		if i > 0 && (len(whole)-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(d)
	}
	if hasFrac {
		b.WriteString("." + frac)
	}
	return &object.String{Value: sign + b.String()}
}

// padText adds fill (a space unless given) to one side of s until it's
// width characters long.
func padText(method string, s *object.String, args []object.Object, left bool) object.Object {
	if len(args) != 1 && len(args) != 2 {
		fatalf("%q takes a width, and optionally the character to fill with: %s[8] or %s[3, \"0\"]", method, method, method)
	}
	width, ok := args[0].(*object.Integer)
	if !ok || width.Value < 0 {
		fatalf("%q: the width must be a whole number of 0 or more, got %s", method, object.Shown(args[0]))
	}
	fill := " "
	if len(args) == 2 {
		f, ok := args[1].(*object.String)
		if !ok || utf8.RuneCountInString(f.Value) != 1 {
			fatalf("%q: fill with one character, like \"0\" or \" \", got %s", method, object.Shown(args[1]))
		}
		fill = f.Value
	}
	n := int(width.Value) - utf8.RuneCountInString(s.Value)
	if n <= 0 {
		return s
	}
	pad := strings.Repeat(fill, n)
	if left {
		return &object.String{Value: pad + s.Value}
	}
	return &object.String{Value: s.Value + pad}
}
