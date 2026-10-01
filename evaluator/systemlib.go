package evaluator

import (
	"os"
	"path/filepath"

	"Turtle/object"
)

// The "system" builtin module: what a command-line tool needs from its
// environment.
//
//	turtle report.t data.txt --verbose
//	a = args[]                       // [ data.txt, --verbose ]
//	if ] exists["data.txt"] [ ... if [end]
//	home = env["HOME"]               // a string, or none if unset
//	exit[2]                          // end now, with exit code 2
//	here = scriptFolder[]            // the script's own folder
//	names = contents["sub"]          // what's in a folder ("." if omitted)
//
// Paths resolve the same way [read]/[write] do: relative to the folder
// turtle was run from, unless absolute.
func (it *Interpreter) callSystem(name string, args []object.Object) object.Object {
	switch name {
	case "exit":
		code := int64(0)
		if len(args) > 1 {
			fatalf("'exit' expects 0 or 1 arguments (an exit code), got %d", len(args))
		}
		if len(args) == 1 {
			n, ok := args[0].(*object.Integer)
			if !ok {
				fatalf("'exit' code must be an integer, got %s", args[0].Type())
			}
			code = n.Value
		}
		panic(ExitRequest{Code: int(code)})
	case "env":
		requireArgs(name, args, 1)
		v, ok := os.LookupEnv(asStringArg(name, args[0]))
		if !ok {
			return object.NoneValue
		}
		return &object.String{Value: v}
	case "contents":
		// The same listing as the [directory] statement, as a value:
		// names only, sorted, files and folders together.
		if len(args) > 1 {
			fatalf("'contents' expects 0 or 1 arguments (a folder path), got %d", len(args))
		}
		path := "."
		if len(args) == 1 {
			path = asStringArg(name, args[0])
		}
		entries, err := os.ReadDir(it.resolvePath(path))
		if err != nil {
			fatalf("'contents' %s: %v", path, err)
		}
		list := &object.List{}
		for _, e := range entries {
			list.Elements = append(list.Elements, &object.String{Value: e.Name()})
		}
		return list
	case "scriptFolder":
		requireArgs(name, args, 0)
		abs, err := filepath.Abs(it.Dir)
		if err != nil {
			abs = it.Dir
		}
		return &object.String{Value: abs}
	}
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
