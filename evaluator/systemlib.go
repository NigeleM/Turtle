package evaluator

import (
	"os"

	"Turtle/object"
)

// The "system" builtin module: the program's command-line arguments and
// checks on the filesystem.
//
//	turtle report.t data.txt --verbose
//	a = args[]                       // [ data.txt, --verbose ]
//	if ] exists["data.txt"] [ ... if [end]
//
// Paths resolve the same way [read]/[write] do: relative to the script's
// own directory, unless absolute.
func (it *Interpreter) callSystem(name string, args []object.Object) object.Object {
	if name == "args" {
		if len(args) != 0 {
			fatalf("'args' expects 0 arguments, got %d", len(args))
		}
		list := &object.List{}
		for _, a := range it.Args {
			list.Elements = append(list.Elements, &object.String{Value: a})
		}
		return list
	}
	if len(args) != 1 {
		fatalf("'%s' expects 1 argument (a path), got %d", name, len(args))
	}
	path, ok := args[0].(*object.String)
	if !ok {
		fatalf("'%s' needs a path string, got %s", name, args[0].Type())
	}
	info, err := os.Stat(it.resolvePath(path.Value))
	found := err == nil
	switch name {
	case "isFile":
		found = found && info.Mode().IsRegular()
	case "isFolder":
		found = found && info.IsDir()
	}
	return &object.Boolean{Value: found}
}
