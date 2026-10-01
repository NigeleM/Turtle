package evaluator

import (
	"strings"

	"Turtle/object"
)

// The "strings" builtin module: string functions that read well
// sentence-style, sharing their logic with the string methods
// (find = indexOf, substring = slice, isinstring = contains).
//
//	pos = line find "lo"                 // find[line, "lo"]: index, or -1
//	part = line substring 0, 5           // substring[line, 0, 5]
//	if ] "lo" isinstring line [ ...      // isinstring["lo", line]
//	csv = words join ", "                // join[words, ", "]
func callStrings(name string, args []object.Object) object.Object {
	switch name {
	case "find":
		requireArgs(name, args, 2)
		text := asStringArg(name, args[0])
		return &object.Integer{Value: int64(runeIndexOf(text, asStringArg(name, args[1])))}
	case "substring":
		if len(args) < 2 || len(args) > 3 {
			fatalf("'substring' expects 2 or 3 arguments (text, start [, end]), got %d", len(args))
		}
		text, ok := args[0].(*object.String)
		if !ok {
			fatalf("'substring' needs a string, got %s", args[0].Type())
		}
		return stringSlice(text, name, args[1:])
	case "isinstring":
		requireArgs(name, args, 2)
		part := asStringArg(name, args[0])
		return &object.Boolean{Value: strings.Contains(asStringArg(name, args[1]), part)}
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
			fatalf("'join' needs a list or set, got %s", args[0].Type())
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
	fatalf("no strings function %q", name)
	return nil
}
