package syntax

import (
	"fmt"
	"strings"
)

// Extensions are the endings of Turtle files: .turtle, the official one,
// then .trt, also accepted (Turtle's first ending, kept so every program
// written with it still works).
var Extensions = []string{".turtle", ".trt"}

// IsTurtleFile reports whether name ends with one of Turtle's endings.
func IsTurtleFile(name string) bool {
	for _, ext := range Extensions {
		if strings.HasSuffix(name, ext) && len(name) > len(ext) {
			return true
		}
	}
	return false
}

// TrimExtension is name without its Turtle ending (name itself if it has
// none).
func TrimExtension(name string) string {
	for _, ext := range Extensions {
		if strings.HasSuffix(name, ext) {
			return strings.TrimSuffix(name, ext)
		}
	}
	return name
}

// FindModule picks the file for an import of base (a path without an
// ending): exists reports whether base plus an ending is there. It gives
// back the ending to use, or "" when there's none. Both at once is an
// error, so an import never quietly picks one of two files.
func FindModule(base string, exists func(ext string) bool) (string, error) {
	var found []string
	for _, ext := range Extensions {
		if exists(ext) {
			found = append(found, ext)
		}
	}
	switch len(found) {
	case 0:
		return "", nil
	case 1:
		return found[0], nil
	}
	return "", fmt.Errorf("both %s%s and %s%s are there; keep one (Turtle files end in .turtle, or .trt)", base, found[0], base, found[1])
}
