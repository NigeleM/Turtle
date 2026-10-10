package main

import (
	"Turtle/syntax"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"Turtle/format"
)

// fmtCommand is turtle fmt [--check] [file.turtle | folder ...]: lays out
// the .turtle files named, or every one in the folders (the current one if
// none), in place, and names those it changed. With --check it changes
// nothing, names those that need it, and exits 1 if any do. A file that
// doesn't parse is reported and left alone.
func fmtCommand(args []string, out, errOut io.Writer) int {
	check := false
	var targets []string
	for _, a := range args {
		if a == "--check" {
			check = true
			continue
		}
		targets = append(targets, a)
	}
	if len(targets) == 0 {
		targets = []string{"."}
	}
	var files []string
	for _, t := range targets {
		info, err := os.Stat(t)
		if err != nil {
			fmt.Fprintf(errOut, "turtle fmt: %s: no such file or folder\n", t)
			return 2
		}
		if !info.IsDir() {
			files = append(files, t)
			continue
		}
		filepath.WalkDir(t, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if d.IsDir() && path != t && strings.HasPrefix(d.Name(), ".") {
				return filepath.SkipDir
			}
			if !d.IsDir() && syntax.IsTurtleFile(d.Name()) {
				files = append(files, path)
			}
			return nil
		})
	}
	changed, broken := 0, 0
	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			fmt.Fprintf(errOut, "turtle fmt: %s: %v\n", f, err)
			broken++
			continue
		}
		formatted, err := format.FormatIn(string(data), filepath.Dir(f))
		if err != nil {
			fmt.Fprintf(errOut, "turtle fmt: %s: %v\n", f, err)
			broken++
			continue
		}
		if formatted == string(data) {
			continue
		}
		changed++
		if check {
			fmt.Fprintf(out, "%s needs formatting\n", f)
			continue
		}
		info, _ := os.Stat(f)
		if err := os.WriteFile(f, []byte(formatted), info.Mode().Perm()); err != nil {
			fmt.Fprintf(errOut, "turtle fmt: %s: %v\n", f, err)
			broken++
			continue
		}
		fmt.Fprintf(out, "formatted %s\n", f)
	}
	if broken > 0 || check && changed > 0 {
		return 1
	}
	return 0
}
