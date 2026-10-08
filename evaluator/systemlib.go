package evaluator

import (
	"os"
	"path/filepath"
	"strings"

	"Turtle/object"
)

// The "system" builtin module: what a command-line tool needs from its
// environment.
//
//	turtle report.trt data.txt --verbose
//	a = args[]                       // [ data.txt, --verbose ]
//	if ] exists["data.txt"] [ ... if [end]
//	home = env["HOME"]               // a string, or none if unset
//	exit[2]                          // end now, with exit code 2
//	here = scriptfolder[]            // the script's own folder
//	names = contents["sub"]          // what's in a folder ("." if omitted)
//	erase["old.txt"]                 // delete a file, or a folder and all in it
//	warn "can't find ", name .       // like show, but to stderr (evaluator.go)
//
// Paths resolve the same way [read]/[write] do: relative to the folder
// turtle was run from, unless absolute.
// realPath is p made absolute with the symlinks in its folders resolved
// (/tmp/x is /private/tmp/x on macOS), so two spellings of one folder
// compare equal. The last part isn't resolved: erasing a symlink removes
// the link, not what it points to.
func realPath(p string) (string, error) {
	abs, err := filepath.Abs(p)
	if err != nil || filepath.Dir(abs) == abs { // an error, or the root
		return abs, err
	}
	dir, err := filepath.EvalSymlinks(filepath.Dir(abs))
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, filepath.Base(abs)), nil
}

func (it *Interpreter) callSystem(name string, args []object.Object) object.Object {
	if v, ok := it.callFiles(name, args); ok {
		return v
	}
	switch name {
	case "exit":
		code := int64(0)
		if len(args) > 1 {
			fatalf("'exit' expects 0 or 1 arguments (an exit code), got %d", len(args))
		}
		if len(args) == 1 {
			n, ok := args[0].(*object.Integer)
			if !ok {
				fatalf("'exit' code must be an integer, got %s", typeName(args[0]))
			}
			code = n.Value
		}
		panic(ExitRequest{Code: int(code)})
	case "env":
		requireFuncArgs(name, args, 1)
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
			fatalKind(kindFile, "'contents' %s: %s", path, fileProblem(err))
		}
		list := &object.List{}
		for _, e := range entries {
			list.Elements = append(list.Elements, &object.String{Value: e.Name()})
		}
		return list
	case "erase":
		// Deletes a file, or a folder and everything in it. There's no
		// undo, so it refuses the folder turtle runs in and any folder
		// above it (erase["."] would otherwise wipe the project).
		requireFuncArgs(name, args, 1)
		target := asStringArg(name, args[0])
		path := it.resolvePath(target)
		if _, err := os.Lstat(path); err != nil {
			fatalKind(kindFile, "erase %s: %s", target, fileProblem(err))
		}
		abs, err1 := realPath(path)
		wd, err2 := filepath.Abs(it.WorkDir)
		if err2 == nil {
			wd, err2 = filepath.EvalSymlinks(wd)
		}
		if err1 != nil || err2 != nil || abs == wd || abs == filepath.Dir(abs) ||
			strings.HasPrefix(wd, abs+string(filepath.Separator)) {
			fatalKind(kindFile, "erase %s: won't erase the folder turtle is running in, or a folder above it", target)
		}
		if err := os.RemoveAll(path); err != nil {
			fatalKind(kindFile, "erase %s: %s", target, fileProblem(err))
		}
		return object.NoneValue
	case "scriptfolder":
		requireFuncArgs(name, args, 0)
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
		fatalf("'%s' needs a path string, got %s", name, typeName(args[0]))
	}
	info, err := os.Stat(it.resolvePath(path.Value))
	found := err == nil
	switch name {
	case "isfile":
		found = found && info.Mode().IsRegular()
	case "isfolder":
		found = found && info.IsDir()
	}
	return &object.Boolean{Value: found}
}
