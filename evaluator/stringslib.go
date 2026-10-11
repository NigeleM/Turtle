// Copyright 2017-2026 Nigele McCoy
// Licensed under the Apache License, Version 2.0;
// see LICENSE and NOTICE.

package evaluator

import (
	"strings"

	"Turtle/object"
)

// The "strings" builtin module: string functions that read well
// sentence-style, sharing their logic with the string methods
// (find = indexof, substring = slice, isinstring = contains).
//
//	pos = line find "lo"                 // find[line, "lo"]: index, or -1
//	part = line substring 0, 5           // substring[line, 0, 5]
//	if ] "lo" isinstring line [ ...      // isinstring["lo", line]
//	csv = words join ", "                // join[words, ", "]
func callStrings(name string, args []object.Object) object.Object {
	switch name {
	case "find":
		requireFuncArgs(name, args, 2)
		text := asStringArg(name, args[0])
		return object.Int(int64(runeIndexOf(text, asStringArg(name, args[1]))))
	case "substring":
		if len(args) < 2 || len(args) > 3 {
			fatalf("'substring' expects 2 or 3 arguments (text, start [, end]), got %d", len(args))
		}
		text, ok := args[0].(*object.String)
		if !ok {
			fatalf("'substring' needs a string, got %s", typeName(args[0]))
		}
		return stringSlice(text, name, args[1:])
	case "isinstring":
		requireFuncArgs(name, args, 2)
		part := asStringArg(name, args[0])
		return object.Bool(strings.Contains(asStringArg(name, args[1]), part))
	case "join":
		if len(args) < 1 || len(args) > 2 {
			fatalf("'join' expects 1 or 2 arguments (a list or set [, separator]), got %d", len(args))
		}
		var elems []object.Object
		switch c := args[0].(type) {
		case *object.List:
			elems = c.Elements
		case *object.Set:
			elems = c.Elements
		default:
			fatalf("'join' needs a list or set, got %s", typeName(args[0]))
		}
		sep := ""
		if len(args) == 2 {
			sep = asStringArg(name, args[1])
		}
		parts := make([]string, len(elems))
		for i, e := range elems {
			parts[i] = e.Inspect()
		}
		return &object.String{Value: strings.Join(parts, sep)}
	}
	fatalKind(kindName, "no strings function %q", name)
	return nil
}
