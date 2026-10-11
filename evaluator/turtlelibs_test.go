// Copyright 2017-2026 Nigele McCoy
// Licensed under the Apache License, Version 2.0;
// see LICENSE and NOTICE.

package evaluator

import (
	"maps"
	"slices"
	"strings"
	"testing"
	"testing/fstest"

	"Turtle/lexer"
	"Turtle/object"
	"Turtle/parser"
)

// A standard library written only in Turtle, and Turtle added to a
// library written in Go, as maintainers would add them to evaluator/lib.
// They're for these tests: turtle itself doesn't have them.
var testLibs = fstest.MapFS{
	// A whole library, in Turtle only.
	"lib/geometry.turtle": {Data: []byte(`// Shapes and their measurements, written in Turtle.
import math

// A point on a plane.
assemble Point [x, y]

// distance is how far apart two points are.
def distance[a, b]
    dx = x of b - x of a
    dy = y of b - y of a
    return ~root[dx * dx + dy * dy]
def [end]

//* area is the area of a rectangle,
   width times height. *//
def area[width, height]
    if ] width < 0 || height < 0 [
        fail "area: a side can't be negative"
    if [end]
    return width * height
def [end]

def perimeter[width, height]
    return 2 * width + 2 * height
def [end]

// ~root is a private helper.
def ~root[n]
    return n at sqrt
def [end]
`)},
	// Turtle added to data, a library written in Go: it uses data's own
	// Go functions through import data.
	"lib/data.turtle": {Data: []byte(`import data

// clamp keeps each number between low and high.
//   clamp[list [3, 15, -2], 0, 10]      gives [ 3, 10, 0 ]
def clamp[nums, low, high]
    return nums process n give ~limit[n, low, high]
def [end]

// spread is how far the numbers are from their mean, at most.
def spread[nums]
    m = mean[nums]
    return reduce[nums, 0, [far, n] give ~bigger[far, n - m, m - n]]
def [end]

def ~limit[n, low, high]
    return min of list [max of list [n, low], high]
def [end]

def ~bigger[a, b, c]
    return max of list [a, b, c]
def [end]
`)},
}

// withLibs registers files as evaluator/lib for one test, and puts the
// standard library back as it was afterwards.
func withLibs(t *testing.T, files fstest.MapFS) error {
	t.Helper()
	savedFS, savedDocs, savedLibs := libFiles, maps.Clone(moduleDocs), maps.Clone(turtleLibs)
	savedModules := builtinModules
	copies := map[string]*object.Module{}
	for name, m := range builtinModules {
		c := *m
		c.Funcs = slices.Clone(m.Funcs)
		copies[name] = &c
	}
	builtinModules = copies
	t.Cleanup(func() {
		libFiles, moduleDocs, turtleLibs, builtinModules = savedFS, savedDocs, savedLibs, savedModules
	})
	libFiles = files
	return registerTurtleLibs(files, builtinModules, moduleDocs, turtleLibs)
}

func TestStdlibInTurtle(t *testing.T) {
	if err := withLibs(t, testLibs); err != nil {
		t.Fatal(err)
	}
	cases := []struct{ src, want string }{
		// A library of its own, written in Turtle.
		{"import geometry\nshow distance[Point[0, 0], Point[3, 4]], \" \", area[2, 5], \" \", perimeter[2, 5] .", "5.0 10 14"},
		{"import geometry [area]\nshow area[3, 3] .", "9"},
		{"import geometry\nshow geometry area[1, 2] .", "2"},
		// Its errors point at the caller's line, as a Go function's do.
		{"import geometry\nsafe\n    x = area[-1, 2]\nhandle [] e .\n    show kind of e, \" \", line of e, \" \", message of e .\nsafe [end]", "custom 3 area: a side can't be negative"},
		// data: its Go functions and the Turtle ones, alike.
		{"import data\nshow clamp[list [3, 15, -2], 0, 10], \" \", mean[list [1, 2, 3]], \" \", spread[list [1, 2, 6]] .", "[ 3, 10, 0 ] 2.0 3.0"},
		{"import data [clamp, sum]\nshow sum[clamp[list [50, -5], 0, 10]] .", "10"},
	}
	for _, c := range cases {
		got, err := run(t, c.src, "")
		if err != nil {
			t.Errorf("%s: %v", c.src, err)
		} else if strings.TrimSpace(got) != c.want {
			t.Errorf("%s:\n got  %q\n want %q", c.src, strings.TrimSpace(got), c.want)
		}
	}
	// Private helpers stay private; nothing leaks from one library to the
	// program.
	for src, want := range map[string]string{
		"import geometry\nx = ~root[4]":              "~root is private to geometry",
		"import data\nx = ~limit[1, 2, 3]":           "~limit is private to data",
		"import geometry\nx = clamp[list [1], 0, 1]": `"clamp" needs "import data" first`,
		"x = area[1, 2]":                             `"area" needs "import geometry" first`,
	} {
		_, err := run(t, src, "")
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%s:\n got  %v\n want %q", src, err, want)
		}
	}
}

// Their documentation comes from the comments above each def, for turtle
// doc, help and the editor; a library of its own is described by the
// comment at the top of its file.
func TestStdlibInTurtleDocs(t *testing.T) {
	if err := withLibs(t, testLibs); err != nil {
		t.Fatal(err)
	}
	for topic, want := range map[string]string{
		"geometry":  "import geometry\n\nShapes and their measurements, written in Turtle.",
		"distance":  "distance[a, b]        (import geometry)\n  distance is how far apart two points are.",
		"area":      "area[width, height]        (import geometry)\n  area is the area of a rectangle,\n  width times height.",
		"perimeter": "(no description yet: write // lines just above it)",
		"Point":     "Point[x, y]        (import geometry)\n  A point on a plane.",
		"clamp":     "clamp[nums, low, high]        (import data)\n  clamp keeps each number between low and high.\n    clamp[list [3, 15, -2], 0, 10]      gives [ 3, 10, 0 ]",
	} {
		got, err := Doc(topic, ".")
		if err != nil || !strings.Contains(got, want) {
			t.Errorf("turtle doc %s: %v\n%s\nwant %q", topic, err, got, want)
		}
	}
	if _, err := Doc("~root", "."); err == nil {
		t.Error("turtle doc ~root: a private helper has no documentation")
	}
	if !slices.Contains(Libraries(), "geometry") || LibraryOf("clamp") != "data" {
		t.Errorf("libraries %v, clamp in %q", Libraries(), LibraryOf("clamp"))
	}
	got, err := run(t, `help["geometry"]`, "")
	if err != nil || !strings.Contains(got, "distance[a, b]\n      distance is how far apart two points are.") {
		t.Errorf("help: %v\n%s", err, got)
	}
}

// A function in Turtle can't have the name of one its library has in Go,
// and a library that doesn't parse stops the build's tests.
func TestStdlibInTurtleMistakes(t *testing.T) {
	for file, want := range map[string]string{
		"def mean[nums]\n    return 0\ndef [end]\n": "lib/data.turtle: mean is already a function of data written in Go",
		"def broken[\n": "builtin library lib/data.turtle:",
	} {
		err := withLibs(t, fstest.MapFS{"lib/data.turtle": {Data: []byte(file)}})
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%q: got %v, want %q", file, err, want)
		}
	}
}

// The libraries turtle ships with in Turtle (lib/*.turtle) parse, and
// each public function has a description.
func TestShippedStdlibInTurtle(t *testing.T) {
	err := registerTurtleLibs(turtleLibFiles, map[string]*object.Module{"random": {Name: "random", Funcs: []string{"shuffle", "sample"}}}, map[string]string{}, map[string]bool{})
	if err != nil {
		t.Fatal(err)
	}
	// Their theories are proven: every proof case and theorem holds.
	it := New(".")
	for name := range turtleLibs {
		mod := it.loadTurtleLib(builtinModules[name])
		for _, fn := range mod.Env.Functions() {
			if fn.Theory == nil {
				continue
			}
			if p := it.proveTheory(fn, 1); p.failed() {
				t.Errorf("%s's theory %s isn't proven:\n%s", name, fn.Name, p.report())
			}
		}
	}
	for name := range turtleLibs {
		for _, f := range builtinModules[name].Funcs {
			if strings.Contains(moduleDocs[name], "### "+f+"[") {
				continue
			}
			t.Errorf("%s's %s has no documentation", name, f)
		}
	}
}

// Theories in the standard library: a new library made of them (units), and
// one added to data, a library written in Go, using data's own process.
var theoryLibs = fstest.MapFS{
	"lib/units.turtle": {Data: []byte(`// Units of measure, as phrases.
import math

theory celsius
    abstract
        celsius is a temperature f in Fahrenheit, in Celsius.
    notation celsius f from fahrenheit .
    definition
        above = f - 32
        return ~tenths[above * 5 / 9]
    theorem result <= f || f < -40
    proof
        celsius 212 from fahrenheit . is 100.0
theory [end]

def ~tenths[x]
    return x at round[1]
def [end]

theory ~secret
    abstract
        ~secret is private to units.
    notation ~secret n .
    definition
        return n
theory [end]
`)},
	"lib/data.turtle": {Data: []byte(`import data

theory clampall
    abstract
        clampall keeps every number in nums between low and high.
    notation clampall nums from low to high .
    definition
        return nums process n give min of list [max of list [n, low], high]
    proof
        clampall list [3, 15, -2] from 0 to 10 . is list [3, 10, 0]
theory [end]
`)},
}

func TestStdlibTheories(t *testing.T) {
	if err := withLibs(t, theoryLibs); err != nil {
		t.Fatal(err)
	}
	cases := []struct{ src, want string }{
		{"import units\nshow celsius 212 from fahrenheit .", "100.0"},
		{"import units\nc is celsius 50 from fahrenheit .\nshow c .", "10.0"},
		{"import data\nshow clampall list [3, 15, -2] from 0 to 10, \" \", mean[list [2, 4]] .", "[ 3, 10, 0 ] 3.0"},
		{"import data [clampall]\nshow clampall list [50] from 0 to 9 .", "[ 9 ]"},
		{"import units\nshow typeof[celsius] .", "theory"},
	}
	for _, c := range cases {
		got, err := run(t, c.src, "")
		if err != nil {
			t.Errorf("%s: %v", c.src, err)
		} else if strings.TrimSpace(got) != c.want {
			t.Errorf("%s:\n got  %q\n want %q", c.src, strings.TrimSpace(got), c.want)
		}
	}
	// An error inside a library theory points at the user's line and
	// names the theory it came from.
	_, err := run(t, "import units\nx = 1\ny = celsius \"hot\" from fahrenheit", "")
	if err == nil || !strings.Contains(err.Error(), "line 3: celsius (units library): ") {
		t.Errorf("an error in a library theory: %v", err)
	}
	// A private theory stays in its library.
	p := parser.New(lexer.New("import units\nx = ~secret 1"))
	p.ParseProgram()
	if errs := strings.Join(p.Errors(), "; "); !strings.Contains(errs, "~secret is private to units") {
		t.Errorf("private theory: %s", errs)
	}
	// Docs from the abstract, for turtle doc, help and the editor.
	for topic, want := range map[string]string{
		"units":    "import units\n\nUnits of measure, as phrases.",
		"celsius":  "celsius f from fahrenheit        (import units)\n  celsius is a temperature f in Fahrenheit, in Celsius.\n  Written: celsius f from fahrenheit .\n  Theorem: result <= f || f < -40",
		"clampall": "clampall nums from low to high        (import data)",
	} {
		got, err := Doc(topic, ".")
		if err != nil || !strings.Contains(got, want) {
			t.Errorf("turtle doc %s: %v\n%s\nwant %q", topic, err, got, want)
		}
	}
	if _, err := Doc("~secret", "."); err == nil {
		t.Error("a private theory has no documentation")
	}
	// Proven like any theory: diagnose shows its proof.
	got, err := run(t, "import units\nseed = 1\nx is diagnose[celsius] .", "")
	if err != nil || !strings.Contains(got, "celsius 212 from fahrenheit . is 100.0") || !strings.Contains(got, "diagnose theory celsius (the units library, line 4)") || !strings.Contains(got, "holds") {
		t.Errorf("diagnose: %v\n%s", err, got)
	}
}
