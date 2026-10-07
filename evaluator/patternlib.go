package evaluator

import (
	"regexp"
	"sync"

	"Turtle/object"
)

// The pattern library (import pattern): regular expressions, through Go's
// regexp package (RE2 syntax: no backtracking, so no pattern can take
// forever). Write patterns in backticks, which keep { } and \ as typed:
//
//	matches["order 42", `\d+`]              true
//	findall["a1 b22", `\d+`]                [ "1", "22" ]
//	replaceall["a1 b2", `\d`, "#"]          "a# b#"
//	splitby["a, b;c", `[,;]\s*`]            [ "a", "b", "c" ]
//	groups["2026-10-06", `(\d+)-(\d+)`]     [ "2026", "10" ]

var patterns sync.Map // pattern text -> *regexp.Regexp, compiled once

func compilePattern(fn string, v object.Object) *regexp.Regexp {
	text := asStringArg(fn, v)
	if re, ok := patterns.Load(text); ok {
		return re.(*regexp.Regexp)
	}
	re, err := regexp.Compile(text)
	if err != nil {
		fatalKind(kindPattern, "%s: %q isn't a valid pattern: %v", fn, text, err)
	}
	patterns.Store(text, re)
	return re
}

func strings2list(items []string) *object.List {
	out := &object.List{Elements: make([]object.Object, len(items))}
	for i, s := range items {
		out.Elements[i] = &object.String{Value: s}
	}
	return out
}

func (it *Interpreter) callPattern(name string, args []object.Object) object.Object {
	switch name {
	case "matches":
		requireFuncArgs(name, args, 2)
		text := asStringArg(name, args[0])
		return &object.Boolean{Value: compilePattern(name, args[1]).MatchString(text)}
	case "findall":
		requireFuncArgs(name, args, 2)
		text := asStringArg(name, args[0])
		return strings2list(compilePattern(name, args[1]).FindAllString(text, -1))
	case "replaceall":
		requireFuncArgs(name, args, 3)
		text := asStringArg(name, args[0])
		with := asStringArg(name, args[2])
		return &object.String{Value: compilePattern(name, args[1]).ReplaceAllString(text, with)}
	case "splitby":
		requireFuncArgs(name, args, 2)
		text := asStringArg(name, args[0])
		return strings2list(compilePattern(name, args[1]).Split(text, -1))
	case "groups":
		requireFuncArgs(name, args, 2)
		text := asStringArg(name, args[0])
		m := compilePattern(name, args[1]).FindStringSubmatch(text)
		if m == nil {
			return object.NoneValue
		}
		return strings2list(m[1:])
	}
	fatalKind(kindName, "no pattern function %q", name)
	return nil
}
