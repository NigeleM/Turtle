package main

import (
	"archive/zip"
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"Turtle/ast"
	"Turtle/evaluator"
	"Turtle/lexer"
	"Turtle/parser"
)

// turtle build: one program file that runs without Turtle installed.
//
//	turtle build report.trt            -> report (report.exe on Windows)
//	turtle build report.trt -o tool    -> tool
//
// The result is a copy of this turtle with the script and the .trt files
// it imports packed onto its end (a zip, then its size and a marker).
// When such a program starts it finds them and runs the script, with
// every argument passed to it. Data files (CSV, settings) aren't packed:
// they're read as with turtle, from the folder the program is run in
// (scriptfolder[] is the program's own folder). It's built for the system
// turtle runs on: turtle on Windows makes a Windows program.

// bundleMarker ends a packed program, after the zip's size.
const bundleMarker = "TURTLE-BUNDLE-1\n"

// bundleMain is the zip entry naming the program's main script.
const bundleMain = ".turtle-main"

func buildCommand(args []string, stdout, stderr io.Writer) int {
	var script, out string
	for i := 0; i < len(args); i++ {
		switch a := args[i]; {
		case a == "-o" || a == "--out":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "turtle build: -o needs a name for the program")
				return 2
			}
			out = args[i+1]
			i++
		case strings.HasPrefix(a, "-"):
			fmt.Fprintf(stderr, "turtle build: unknown option %s (use -o name)\n", a)
			return 2
		case script == "":
			script = a
		default:
			fmt.Fprintf(stderr, "turtle build: one script at a time, got %s and %s\n", script, a)
			return 2
		}
	}
	if script == "" {
		fmt.Fprintln(stderr, "usage: turtle build script.trt [-o name]")
		return 2
	}
	files, err := bundleFiles(script)
	if err != nil {
		fmt.Fprintln(stderr, "turtle build:", err)
		return 1
	}
	if out == "" {
		out = strings.TrimSuffix(filepath.Base(script), filepath.Ext(script))
	}
	if runtime.GOOS == "windows" && !strings.EqualFold(filepath.Ext(out), ".exe") {
		out += ".exe"
	}
	if err := writeBundle(out, files); err != nil {
		fmt.Fprintln(stderr, "turtle build:", err)
		return 1
	}
	names := make([]string, 0, len(files))
	for _, f := range files {
		if f.name != bundleMain {
			names = append(names, f.name)
		}
	}
	fmt.Fprintf(stdout, "built %s (%s, %s)\n", out, strings.Join(names, ", "), runtime.GOOS+"/"+runtime.GOARCH)
	return 0
}

type bundleFile struct {
	name string // the path the program imports it by, from the script's folder
	data []byte
}

// bundleFiles is the script and every .trt file it imports, however
// deep, by their paths from the script's folder.
func bundleFiles(script string) ([]bundleFile, error) {
	dir := filepath.Dir(script)
	main := filepath.Base(script)
	var files []bundleFile
	seen := map[string]bool{}
	var add func(rel string) error
	add = func(rel string) error {
		if seen[rel] {
			return nil
		}
		seen[rel] = true
		data, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(rel)))
		if err != nil {
			return err
		}
		p := parser.New(lexer.New(string(data)))
		program := p.ParseProgram()
		if errs := p.ErrorList(); len(errs) > 0 {
			return fmt.Errorf("%s, %s", rel, parser.Format(string(data), errs[0]))
		}
		files = append(files, bundleFile{rel, data})
		for _, st := range program.Statements {
			im, ok := st.(*ast.ImportStatement)
			if !ok || evaluator.IsLibrary(im.Path) {
				continue
			}
			if err := add(im.Path + ".trt"); err != nil {
				return fmt.Errorf("%s imports %s: %w", rel, im.Path, err)
			}
		}
		return nil
	}
	if err := add(main); err != nil {
		return nil, err
	}
	files = append(files, bundleFile{bundleMain, []byte(main)})
	return files, nil
}

// writeBundle writes out: this turtle, then files as a zip, its size,
// and the marker.
func writeBundle(out string, files []bundleFile) error {
	self, err := os.Executable()
	if err != nil {
		return err
	}
	if _, _, packed := bundleOf(self); packed {
		return errors.New("this turtle is itself a built program; use the turtle you installed")
	}
	exe, err := os.ReadFile(self)
	if err != nil {
		return err
	}
	var z bytes.Buffer
	zw := zip.NewWriter(&z)
	sort.SliceStable(files, func(i, j int) bool { return files[i].name < files[j].name })
	for _, f := range files {
		w, err := zw.Create(f.name)
		if err != nil {
			return err
		}
		if _, err := w.Write(f.data); err != nil {
			return err
		}
	}
	if err := zw.Close(); err != nil {
		return err
	}
	var size [8]byte
	binary.LittleEndian.PutUint64(size[:], uint64(z.Len()))
	tmp := out + ".building"
	f, err := os.OpenFile(tmp, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o755)
	if err != nil {
		return err
	}
	for _, part := range [][]byte{exe, z.Bytes(), size[:], []byte(bundleMarker)} {
		if _, err := f.Write(part); err != nil {
			f.Close()
			os.Remove(tmp)
			return err
		}
	}
	if err := f.Close(); err != nil {
		os.Remove(tmp)
		return err
	}
	// On a Mac the program keeps turtle's signature, which covers turtle's
	// part of the file: it runs, though Apple's strict check (codesign -v)
	// doesn't accept data after it. Copied to another Mac by download,
	// macOS may call it damaged: xattr -d com.apple.quarantine program.
	os.Remove(out) // Windows won't rename over a file
	return os.Rename(tmp, out)
}

// bundleOf reports whether the program at path has packed files, and
// where they start and how long they are.
func bundleOf(path string) (start, size int64, ok bool) {
	f, err := os.Open(path)
	if err != nil {
		return 0, 0, false
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return 0, 0, false
	}
	tail := int64(8 + len(bundleMarker))
	if info.Size() < tail {
		return 0, 0, false
	}
	buf := make([]byte, tail)
	if _, err := f.ReadAt(buf, info.Size()-tail); err != nil {
		return 0, 0, false
	}
	if string(buf[8:]) != bundleMarker {
		return 0, 0, false
	}
	size = int64(binary.LittleEndian.Uint64(buf[:8]))
	start = info.Size() - tail - size
	if size <= 0 || start < 0 {
		return 0, 0, false
	}
	return start, size, true
}

// runBundled runs this program's packed script, if it has one.
func runBundled() (code int, ok bool) {
	self, err := os.Executable()
	if err != nil {
		return 0, false
	}
	start, size, packed := bundleOf(self)
	if !packed {
		return 0, false
	}
	f, err := os.Open(self)
	if err != nil {
		fmt.Fprintln(os.Stderr, "turtle:", err)
		return 1, true
	}
	defer f.Close()
	z, err := zip.NewReader(io.NewSectionReader(f, start, size), size)
	if err != nil {
		fmt.Fprintln(os.Stderr, "this program's packed files are damaged:", err)
		return 1, true
	}
	readFile := func(name string) ([]byte, error) {
		r, err := z.Open(name)
		if err != nil {
			return nil, err
		}
		defer r.Close()
		return io.ReadAll(r)
	}
	main, err := readFile(bundleMain)
	if err == nil {
		var src []byte
		if src, err = readFile(string(main)); err == nil {
			// The program's own folder, as a script's (scriptfolder[]).
			dir := filepath.Dir(self)
			if real, err := filepath.EvalSymlinks(self); err == nil {
				dir = filepath.Dir(real)
			}
			bundleFS = z
			return runProgram(string(src), dir, string(main), os.Args[1:]), true
		}
	}
	fmt.Fprintln(os.Stderr, "this program's packed files are damaged:", err)
	return 1, true
}
