package evaluator

import (
	"archive/tar"
	"archive/zip"
	"bufio"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"Turtle/object"
)

// More of the "system" builtin module: files and folders, archives, .env
// files and named command-line options.
//
//	copyto["a.csv", "backup/a.csv"]       // a file or a whole folder
//	moveto["old.txt", "archive/old.txt"]  // also renames
//	makefolder["out/2026/october"]        // with any missing parents
//	files = walk["src"]                   // every file under src
//	pack["src", "src.zip"]                // .zip, .tar, .tar.gz by extension
//	unpack["src.zip", "restored"]
//	settings = loadenv[".env"]
//	opts = options["--out": "result.csv", "-v": false]
//
// copyto, moveto, pack and unpack won't replace what's already there
// unless their last argument is true.

// callFiles runs one of the functions above; ok is false for a name that
// isn't one of them.
func (it *Interpreter) callFiles(name string, args []object.Object) (object.Object, bool) {
	switch name {
	case "copyto", "moveto":
		from, to, replace := it.fromTo(name, args)
		if name == "copyto" {
			it.copyPath(name, from, to, replace)
		} else {
			it.movePath(from, to, replace)
		}
		return object.NoneValue, true
	case "makefolder":
		requireFuncArgs(name, args, 1)
		path := asStringArg(name, args[0])
		if err := os.MkdirAll(it.resolvePath(path), 0o755); err != nil {
			fatalKind(kindFile, "makefolder %s: %s", path, fileProblem(err))
		}
		return object.NoneValue, true
	case "walk":
		return it.walk(args), true
	case "pack", "unpack":
		from, to, replace := it.fromTo(name, args)
		if name == "pack" {
			it.pack(from, to, replace)
		} else {
			it.unpack(from, to, replace)
		}
		return object.NoneValue, true
	case "loadenv":
		return it.loadenv(args), true
	case "options":
		return it.options(args), true
	}
	return nil, false
}

// fromTo reads [from, to] or [from, to, replace].
func (it *Interpreter) fromTo(name string, args []object.Object) (from, to string, replace bool) {
	if len(args) != 2 && len(args) != 3 {
		fatalKind(kindType, "%s expects 2 or 3 arguments (from, to, and true to replace), got %d", name, len(args))
	}
	from, to = asStringArg(name, args[0]), asStringArg(name, args[1])
	if len(args) == 3 {
		b, ok := args[2].(*object.Boolean)
		if !ok {
			fatalKind(kindType, "%s: the third argument is true or false (replace what's there), got %s", name, typeName(args[2]))
		}
		replace = b.Value
	}
	return from, to, replace
}

// clearTarget makes room at to: an error if something's there and
// replace is false; else it's removed (never the folder turtle runs in, or
// one above it).
func (it *Interpreter) clearTarget(name, to string, replace bool) {
	path := it.resolvePath(to)
	if _, err := os.Lstat(path); err != nil {
		return
	}
	if !replace {
		fatalKind(kindFile, "%s: %s is already there; add true as the last argument to replace it", name, to)
	}
	it.refuseWorkFolder(name, to)
	if err := os.RemoveAll(path); err != nil {
		fatalKind(kindFile, "%s %s: %s", name, to, fileProblem(err))
	}
}

// refuseWorkFolder stops a change to the folder turtle runs in, or one
// above it, as erase does.
func (it *Interpreter) refuseWorkFolder(name, target string) {
	abs, err1 := realPath(it.resolvePath(target))
	wd, err2 := filepath.Abs(it.WorkDir)
	if err2 == nil {
		wd, err2 = filepath.EvalSymlinks(wd)
	}
	if err1 != nil || err2 != nil || abs == wd || abs == filepath.Dir(abs) ||
		strings.HasPrefix(wd, abs+string(filepath.Separator)) {
		fatalKind(kindFile, "%s %s: won't change the folder turtle is running in, or a folder above it", name, target)
	}
}

// inside reports whether path is dir or somewhere in it.
func inside(path, dir string) bool {
	a, err1 := realPath(path)
	b, err2 := realPath(dir)
	return err1 == nil && err2 == nil && (a == b || strings.HasPrefix(a, b+string(filepath.Separator)))
}

func (it *Interpreter) copyPath(name, from, to string, replace bool) {
	src, dst := it.resolvePath(from), it.resolvePath(to)
	info, err := os.Stat(src)
	if err != nil {
		fatalKind(kindFile, "%s %s: %s", name, from, fileProblem(err))
	}
	if info.IsDir() && inside(dst, src) {
		fatalKind(kindFile, "%s: can't copy the folder %s into itself (%s)", name, from, to)
	}
	it.clearTarget(name, to, replace)
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		fatalKind(kindFile, "%s %s: %s", name, to, fileProblem(err))
	}
	if !info.IsDir() {
		if err := copyFile(src, dst, info.Mode()); err != nil {
			fatalKind(kindFile, "%s %s to %s: %s", name, from, to, fileProblem(err))
		}
		return
	}
	err = filepath.WalkDir(src, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, p)
		target := filepath.Join(dst, rel)
		info, err := d.Info()
		if err != nil {
			return err
		}
		switch {
		case d.IsDir():
			return os.MkdirAll(target, info.Mode().Perm()|0o700)
		case info.Mode().IsRegular():
			return copyFile(p, target, info.Mode())
		case info.Mode()&fs.ModeSymlink != 0:
			link, err := os.Readlink(p)
			if err != nil {
				return err
			}
			return os.Symlink(link, target)
		}
		return nil // devices, sockets: skipped
	})
	if err != nil {
		fatalKind(kindFile, "%s %s to %s: %s", name, from, to, fileProblem(err))
	}
}

func copyFile(src, dst string, mode fs.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode.Perm())
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}

func (it *Interpreter) movePath(from, to string, replace bool) {
	src, dst := it.resolvePath(from), it.resolvePath(to)
	info, err := os.Lstat(src)
	if err != nil {
		fatalKind(kindFile, "moveto %s: %s", from, fileProblem(err))
	}
	it.refuseWorkFolder("moveto", from)
	if info.IsDir() && inside(dst, src) {
		fatalKind(kindFile, "moveto: can't move the folder %s into itself (%s)", from, to)
	}
	it.clearTarget("moveto", to, replace)
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		fatalKind(kindFile, "moveto %s: %s", to, fileProblem(err))
	}
	if os.Rename(src, dst) == nil {
		return
	}
	// Another disk: copy, then remove the original.
	it.copyPath("moveto", from, to, false)
	if err := os.RemoveAll(src); err != nil {
		fatalKind(kindFile, "moveto %s: copied, but couldn't remove the original: %s", from, fileProblem(err))
	}
}

func (it *Interpreter) walk(args []object.Object) object.Object {
	if len(args) > 1 {
		fatalKind(kindType, "walk expects 0 or 1 arguments (a folder), got %d", len(args))
	}
	folder := "."
	if len(args) == 1 {
		folder = asStringArg("walk", args[0])
	}
	root := it.resolvePath(folder)
	var found []string
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			rel, _ := filepath.Rel(root, p)
			found = append(found, filepath.ToSlash(filepath.Join(folder, rel)))
		}
		return nil
	})
	if err != nil {
		fatalKind(kindFile, "walk %s: %s", folder, fileProblem(err))
	}
	sort.Strings(found)
	list := &object.List{}
	for _, f := range found {
		list.Elements = append(list.Elements, &object.String{Value: f})
	}
	return list
}

// archiveKind is "zip", "tar" or "tar.gz" from a file name.
func archiveKind(name, path string) string {
	lower := strings.ToLower(path)
	switch {
	case strings.HasSuffix(lower, ".zip"):
		return "zip"
	case strings.HasSuffix(lower, ".tar.gz"), strings.HasSuffix(lower, ".tgz"):
		return "tar.gz"
	case strings.HasSuffix(lower, ".tar"):
		return "tar"
	}
	fatalKind(kindFile, "%s %s: an archive name ends in .zip, .tar, .tar.gz or .tgz", name, path)
	return ""
}

// pack puts a file, or a folder and everything in it, into an archive.
// A folder's files go in under its name (src/main.turtle), so unpacking
// gives the folder back.
func (it *Interpreter) pack(from, to string, replace bool) {
	kind := archiveKind("pack", to)
	src, dst := it.resolvePath(from), it.resolvePath(to)
	info, err := os.Stat(src)
	if err != nil {
		fatalKind(kindFile, "pack %s: %s", from, fileProblem(err))
	}
	if info.IsDir() && inside(dst, src) {
		fatalKind(kindFile, "pack: the archive %s can't go inside the folder it packs (%s)", to, from)
	}
	it.clearTarget("pack", to, replace)
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		fatalKind(kindFile, "pack %s: %s", to, fileProblem(err))
	}
	if err := writeArchive(kind, src, dst); err != nil {
		os.Remove(dst)
		fatalKind(kindFile, "pack %s to %s: %s", from, to, fileProblem(err))
	}
}

func writeArchive(kind, src, dst string) (err error) {
	f, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func() {
		if cerr := f.Close(); err == nil {
			err = cerr
		}
	}()
	base := filepath.Dir(src)
	var zw *zip.Writer
	var tw *tar.Writer
	var gz *gzip.Writer
	switch kind {
	case "zip":
		zw = zip.NewWriter(f)
	case "tar.gz":
		gz = gzip.NewWriter(f)
		tw = tar.NewWriter(gz)
	default:
		tw = tar.NewWriter(f)
	}
	err = filepath.WalkDir(src, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		if !d.IsDir() && !info.Mode().IsRegular() {
			return nil // links, devices: left out
		}
		rel, _ := filepath.Rel(base, p)
		name := filepath.ToSlash(rel)
		if d.IsDir() {
			name += "/"
		}
		if zw != nil {
			h, err := zip.FileInfoHeader(info)
			if err != nil {
				return err
			}
			h.Name = name
			if !d.IsDir() {
				h.Method = zip.Deflate
			}
			w, err := zw.CreateHeader(h)
			if err != nil || d.IsDir() {
				return err
			}
			return copyInto(w, p)
		}
		h, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		h.Name = name
		if err := tw.WriteHeader(h); err != nil || d.IsDir() {
			return err
		}
		return copyInto(tw, p)
	})
	if zw != nil {
		if cerr := zw.Close(); err == nil {
			err = cerr
		}
		return err
	}
	if cerr := tw.Close(); err == nil {
		err = cerr
	}
	if gz != nil {
		if cerr := gz.Close(); err == nil {
			err = cerr
		}
	}
	return err
}

func copyInto(w io.Writer, path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(w, f)
	return err
}

// unpack puts an archive's files into the folder to. A file there already
// is an error unless replace; an entry that would land outside to
// (../x, /etc/x) is refused.
func (it *Interpreter) unpack(from, to string, replace bool) {
	kind := archiveKind("unpack", from)
	src, dst := it.resolvePath(from), it.resolvePath(to)
	if _, err := os.Stat(src); err != nil {
		fatalKind(kindFile, "unpack %s: %s", from, fileProblem(err))
	}
	if err := os.MkdirAll(dst, 0o755); err != nil {
		fatalKind(kindFile, "unpack %s: %s", to, fileProblem(err))
	}
	place := func(entry string) string {
		clean := filepath.FromSlash(strings.TrimLeft(entry, "/\\"))
		target := filepath.Join(dst, clean)
		if filepath.IsAbs(entry) || filepath.VolumeName(entry) != "" ||
			(target != dst && !strings.HasPrefix(target, dst+string(filepath.Separator))) {
			fatalKind(kindFile, "unpack %s: the entry %q would land outside %s; not unpacking it", from, entry, to)
		}
		return target
	}
	write := func(entry string, mode fs.FileMode, r io.Reader) {
		target := place(entry)
		if _, err := os.Lstat(target); err == nil && !replace {
			fatalKind(kindFile, "unpack: %s is already there; add true as the last argument to replace it", filepath.ToSlash(filepath.Join(to, filepath.FromSlash(entry))))
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			fatalKind(kindFile, "unpack %s: %s", from, fileProblem(err))
		}
		out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode.Perm()|0o600)
		if err == nil {
			_, err = io.Copy(out, r)
			if cerr := out.Close(); err == nil {
				err = cerr
			}
		}
		if err != nil {
			fatalKind(kindFile, "unpack %s: %s: %s", from, entry, fileProblem(err))
		}
	}
	folder := func(entry string) {
		if err := os.MkdirAll(place(entry), 0o755); err != nil {
			fatalKind(kindFile, "unpack %s: %s", from, fileProblem(err))
		}
	}
	bad := func(err error) {
		fatalKind(kindFile, "unpack %s: not a readable %s archive: %v", from, kind, err)
	}

	if kind == "zip" {
		zr, err := zip.OpenReader(src)
		if err != nil {
			bad(err)
		}
		defer zr.Close()
		for _, f := range zr.File {
			if f.FileInfo().IsDir() {
				folder(f.Name)
				continue
			}
			if !f.Mode().IsRegular() {
				continue
			}
			r, err := f.Open()
			if err != nil {
				bad(err)
			}
			write(f.Name, f.Mode(), r)
			r.Close()
		}
		return
	}
	file, err := os.Open(src)
	if err != nil {
		fatalKind(kindFile, "unpack %s: %s", from, fileProblem(err))
	}
	defer file.Close()
	var r io.Reader = file
	if kind == "tar.gz" {
		gz, err := gzip.NewReader(file)
		if err != nil {
			bad(err)
		}
		defer gz.Close()
		r = gz
	}
	tr := tar.NewReader(r)
	for {
		h, err := tr.Next()
		if errors.Is(err, io.EOF) {
			return
		}
		if err != nil {
			bad(err)
		}
		switch h.Typeflag {
		case tar.TypeDir:
			folder(h.Name)
		case tar.TypeReg:
			write(h.Name, fs.FileMode(h.Mode), tr)
		}
	}
}

// loadenv reads a .env file: KEY=value lines; # comments and blank lines
// skipped; an optional "export " in front; a value in "double quotes"
// (with \n, \t, \" and \\) or 'single quotes' (as typed). Each key is
// also set for env[...], unless the environment already has it: what's
// set outside the file wins, as with every .env tool.
func (it *Interpreter) loadenv(args []object.Object) object.Object {
	if len(args) > 1 {
		fatalKind(kindType, "loadenv expects 0 or 1 arguments (a file, .env if left out), got %d", len(args))
	}
	path := ".env"
	if len(args) == 1 {
		path = asStringArg("loadenv", args[0])
	}
	f, err := os.Open(it.resolvePath(path))
	if err != nil {
		fatalKind(kindFile, "loadenv %s: %s", path, fileProblem(err))
	}
	defer f.Close()
	m := object.NewMap()
	sc := bufio.NewScanner(f)
	for n := 1; sc.Scan(); n++ {
		line := strings.TrimSpace(sc.Text())
		if n == 1 {
			line = strings.TrimPrefix(line, "\ufeff")
		}
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimPrefix(line, "export ")
		key, value, ok := strings.Cut(line, "=")
		key = strings.TrimSpace(key)
		if !ok || key == "" || strings.ContainsAny(key, " \t") {
			fatalKind(kindFile, "loadenv %s: line %d isn't KEY=value: %s", path, n, line)
		}
		value, ok = envValue(strings.TrimSpace(value))
		if !ok {
			fatalKind(kindFile, "loadenv %s: line %d: a quote isn't closed", path, n)
		}
		m.Put(&object.String{Value: key}, &object.String{Value: value})
		if _, set := os.LookupEnv(key); !set {
			os.Setenv(key, value)
		}
	}
	if err := sc.Err(); err != nil {
		fatalKind(kindFile, "loadenv %s: %s", path, fileProblem(err))
	}
	return m
}

// envValue is a .env value without its quotes, or an unquoted one
// without a trailing " # comment".
func envValue(v string) (string, bool) {
	switch {
	case strings.HasPrefix(v, `"`):
		var b strings.Builder
		for i := 1; i < len(v); i++ {
			c := v[i]
			if c == '"' {
				return b.String(), true
			}
			if c == '\\' && i+1 < len(v) {
				i++
				switch v[i] {
				case 'n':
					b.WriteByte('\n')
				case 't':
					b.WriteByte('\t')
				case 'r':
					b.WriteByte('\r')
				default:
					b.WriteByte(v[i])
				}
				continue
			}
			b.WriteByte(c)
		}
		return "", false
	case strings.HasPrefix(v, "'"):
		end := strings.IndexByte(v[1:], '\'')
		if end < 0 {
			return "", false
		}
		return v[1 : end+1], true
	}
	if i := strings.Index(v, " #"); i >= 0 {
		v = strings.TrimSpace(v[:i])
	}
	return v, true
}

// options reads named options from the command line. Its map gives each
// option and its default; the default's kind is the option's: false for
// an on/off switch (-v), an integer or float for a number (--count 5),
// text for anything else (--out file.csv, or --out=file.csv). What's
// typed comes back in a map with every option; what isn't an option is
// left for args[]. -- ends the options. --help (or -h), unless the map
// has it, shows the options and ends the program.
func (it *Interpreter) options(args []object.Object) object.Object {
	requireFuncArgs("options", args, 1)
	spec, ok := args[0].(*object.Map)
	if !ok {
		fatalKind(kindType, `options takes the options and their defaults, as in options["--out": "result.csv", "-v": false]; got %s`, typeName(args[0]))
	}
	out := object.NewMap()
	defaults := map[string]object.Object{} // option name: default
	var names []string
	for _, me := range spec.Entries() {
		key, isText := me.Key.(*object.String)
		if !isText || !strings.HasPrefix(key.Value, "-") || key.Value == "-" || key.Value == "--" {
			fatalKind(kindType, "options: %s isn't an option name; they're text starting with - or --, as in \"-v\" or \"--out\"", me.Key.Inspect())
		}
		defaults[key.Value] = me.Val
		names = append(names, key.Value)
		out.Put(key, me.Val)
	}
	_, ownHelp := defaults["--help"]
	_, ownH := defaults["-h"]
	var rest []string
	for i := 0; i < len(it.Args); i++ {
		a := it.Args[i]
		if a == "--" {
			rest = append(rest, it.Args[i+1:]...)
			break
		}
		if !strings.HasPrefix(a, "-") || a == "-" || isNumberText(a) {
			rest = append(rest, a)
			continue
		}
		name, value, hasValue := strings.Cut(a, "=")
		if (name == "--help" && !ownHelp) || (name == "-h" && !ownH) {
			fmt.Print(optionsHelp(it.Script, names, defaults))
			panic(ExitRequest{Code: 0})
		}
		def, known := defaults[name]
		if !known {
			fatalKind(kindName, "unknown option %s; the options are %s (--help shows them)", name, strings.Join(names, ", "))
		}
		if _, isSwitch := def.(*object.Boolean); isSwitch {
			v := true
			if hasValue {
				switch strings.ToLower(value) {
				case "true", "yes", "1":
				case "false", "no", "0":
					v = false
				default:
					fatalKind(kindType, "option %s is on or off; %q isn't true or false", name, value)
				}
			}
			out.Put(&object.String{Value: name}, object.Bool(v))
			continue
		}
		if !hasValue {
			if i+1 >= len(it.Args) {
				fatalKind(kindType, "option %s needs a value after it, as in %s %s", name, name, def.Inspect())
			}
			i++
			value = it.Args[i]
		}
		out.Put(&object.String{Value: name}, optionValue(name, value, def))
	}
	it.Args = rest
	return out
}

func isNumberText(s string) bool {
	_, err := strconv.ParseFloat(s, 64)
	return err == nil
}

// optionValue is value read as the default's kind.
func optionValue(name, value string, def object.Object) object.Object {
	switch def.(type) {
	case *object.Integer:
		n, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			fatalKind(kindNumber, "option %s takes a whole number; %q isn't one", name, value)
		}
		return object.Int(n)
	case *object.Float:
		f, err := strconv.ParseFloat(value, 64)
		if err != nil {
			fatalKind(kindNumber, "option %s takes a number; %q isn't one", name, value)
		}
		return &object.Float{Value: f}
	}
	return &object.String{Value: value}
}

// optionsHelp is what --help shows.
func optionsHelp(script string, names []string, defaults map[string]object.Object) string {
	if script == "" {
		script = "script.turtle"
	}
	var b strings.Builder
	fmt.Fprintf(&b, "usage: turtle %s [options]\n\noptions:\n", script)
	width := 0
	for _, k := range names {
		width = max(width, len(k))
	}
	for _, k := range names {
		switch d := defaults[k].(type) {
		case *object.Boolean:
			fmt.Fprintf(&b, "  %-*s  on or off (off unless given)\n", width, k)
		case *object.None:
			fmt.Fprintf(&b, "  %-*s  a value\n", width, k)
		default:
			fmt.Fprintf(&b, "  %-*s  default %s\n", width, k, d.Inspect())
		}
	}
	return b.String()
}
