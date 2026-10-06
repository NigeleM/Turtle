package evaluator

import (
	"fmt"
	"io"
	"io/fs"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"Turtle/ast"
	"Turtle/lexer"
	"Turtle/object"
	"Turtle/parser"
)

// turtle test: runs the test_ functions in test_*.trt files.
//
//	turtle test                  every test_*.trt here and in folders below
//	turtle test test_lists.trt     one file
//	turtle test tests/           one folder
//
// A file's top-level code runs once, first; then each test_ function, in
// the order written, each in its own scope. Settings, from import test,
// can be set at the top of the file (every test) or inside a test (that
// test only):
//
//	suite = false | "stop" | "all"    (true means "stop")
//	benchmark = false | true
//	runs = none (Go-style: as many as fit in benchtime) | a number
//	benchtime = 1                     seconds, when runs is none
//
// It prints PASS, FAIL or SKIP for each test with its time, then the
// totals, and returns 1 if anything failed.

// TestCommand runs turtle test with the given paths (none: ".") from
// folder cwd, writing the report to out. It returns the exit code.
func TestCommand(paths []string, cwd string, out io.Writer) int {
	if len(paths) == 0 {
		paths = []string{"."}
	}
	var files []string
	for _, p := range paths {
		full := p
		if !filepath.IsAbs(full) {
			full = filepath.Join(cwd, p)
		}
		info, err := os.Stat(full)
		if err != nil {
			fmt.Fprintf(out, "turtle test: %s: %s\n", p, fileProblem(err))
			return 1
		}
		if !info.IsDir() {
			if !isTestFile(filepath.Base(full)) {
				fmt.Fprintf(out, "turtle test: %s: a test file's name starts with test_ and ends with .trt, like test_lists.trt\n", p)
				return 1
			}
			files = append(files, full)
			continue
		}
		filepath.WalkDir(full, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if d.IsDir() && path != full && strings.HasPrefix(d.Name(), ".") {
				return filepath.SkipDir
			}
			if !d.IsDir() && isTestFile(d.Name()) {
				files = append(files, path)
			}
			return nil
		})
	}
	sort.Strings(files)
	if len(files) == 0 {
		fmt.Fprintf(out, "turtle test: no test files (test_*.trt) in %s\n", strings.Join(paths, ", "))
		return 1
	}
	start := time.Now()
	var total testCounts
	for i, f := range files {
		if i > 0 {
			fmt.Fprintln(out)
		}
		name := f
		if rel, err := filepath.Rel(cwd, f); err == nil {
			name = filepath.ToSlash(rel)
		}
		total.add(runTestFile(f, name, out))
	}
	fmt.Fprintln(out)
	summary := fmt.Sprintf("%d passed, %d failed", total.passed, total.failed)
	if total.skipped > 0 {
		summary += fmt.Sprintf(", %d skipped", total.skipped)
	}
	if total.broken > 0 {
		summary += fmt.Sprintf(", %d %s couldn't run", total.broken, plural(total.broken, "file"))
	}
	summary += fmt.Sprintf(" (%d %s, %s)", len(files), plural(len(files), "file"), fmtDuration(time.Since(start)))
	if total.failed > 0 || total.broken > 0 {
		fmt.Fprintln(out, "FAILED: "+summary)
		return 1
	}
	fmt.Fprintln(out, "ok: "+summary)
	return 0
}

func isTestFile(name string) bool {
	return strings.HasPrefix(name, "test_") && strings.HasSuffix(name, ".trt")
}

type testCounts struct{ passed, failed, skipped, broken int }

func (c *testCounts) add(o testCounts) {
	c.passed += o.passed
	c.failed += o.failed
	c.skipped += o.skipped
	c.broken += o.broken
}

// testOutcome is one test's result.
type testOutcome struct {
	name    string
	err     *fatalError // nil: passed
	exited  bool        // it called system's exit[]
	time    time.Duration
	runs    []time.Duration // benchmark runs (the first run not counted)
	failRun int             // the benchmark run that failed (0: the first)
}

// runTestFile runs one file's tests and reports them.
func runTestFile(path, name string, out io.Writer) testCounts {
	broken := testCounts{broken: 1}
	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(out, "%s: %s\n", name, fileProblem(err))
		return broken
	}
	p := parser.New(lexer.New(string(data)))
	program := p.ParseProgram()
	if errs := p.Errors(); len(errs) > 0 {
		for _, e := range errs {
			fmt.Fprintf(out, "%s: parse error: %s\n", name, e)
		}
		return broken
	}
	if !importsTest(program) {
		fmt.Fprintf(out, "%s: add import test at the top (it turns on check, verify and validate)\n", name)
		return broken
	}
	it := New(filepath.Dir(path))
	it.Script = filepath.Base(path)
	defer it.closeDatabases()
	if fe, exited := it.runTop(program); fe != nil || exited {
		msg := "it called exit[]"
		if fe != nil {
			msg = fe.msg
		}
		fmt.Fprintf(out, "%s: the file's own code stopped before the tests: %s\n", name, msg)
		return broken
	}

	mode := "off"
	if v, ok := it.Global.Get(suiteName); ok {
		switch x := v.(type) {
		case *object.Boolean:
			if x.Value {
				mode = "stop"
			}
		case *object.None:
		case *object.String:
			if x.Value != "stop" && x.Value != "all" {
				fmt.Fprintf(out, "%s: suite must be false, \"stop\" or \"all\", got %s\n", name, object.Shown(v))
				return broken
			}
			mode = x.Value
		default:
			fmt.Fprintf(out, "%s: suite must be false, \"stop\" or \"all\", got %s\n", name, object.Shown(v))
			return broken
		}
	}

	var tests []*object.Function
	for _, s := range program.Statements {
		if d, ok := s.(*ast.FunctionDefStatement); ok && strings.HasPrefix(d.Name, "test_") {
			if fn, ok := it.Global.GetFunction(d.Name); ok {
				tests = append(tests, fn)
			}
		}
	}
	header := name
	if mode != "off" {
		header += fmt.Sprintf(" (suite: %s)", mode)
	}
	fmt.Fprintln(out, header)
	if len(tests) == 0 {
		fmt.Fprintln(out, "  no test_ functions")
		return testCounts{}
	}
	width := 0
	for _, t := range tests {
		width = max(width, len(t.Name))
	}

	var counts testCounts
	var failures []testOutcome
	stoppedBy := ""
	for _, fn := range tests {
		pad := strings.Repeat(" ", width-len(fn.Name))
		if stoppedBy != "" {
			fmt.Fprintf(out, "  SKIP  %s%s   (stopped: %s failed)\n", fn.Name, pad, stoppedBy)
			counts.skipped++
			continue
		}
		o := it.runTest(fn)
		if o.err == nil && !o.exited {
			counts.passed++
			fmt.Fprintf(out, "  PASS  %s%s   %s\n", fn.Name, pad, o.timing())
			continue
		}
		counts.failed++
		fmt.Fprintf(out, "  FAIL  %s%s   %s\n", fn.Name, pad, fmtDuration(o.time))
		if mode == "all" {
			failures = append(failures, o)
		} else {
			writeFailure(out, name, o, "        ")
		}
		if mode == "stop" {
			stoppedBy = fn.Name
		}
	}
	if mode != "off" {
		verdict := "passed"
		if counts.failed > 0 {
			verdict = "FAILED"
		}
		line := fmt.Sprintf("suite %s: %d passed, %d failed", verdict, counts.passed, counts.failed)
		if counts.skipped > 0 {
			line += fmt.Sprintf(", %d skipped", counts.skipped)
		}
		fmt.Fprintln(out, line)
	}
	if len(failures) > 0 {
		fmt.Fprintln(out, "\nfailures:")
		for _, o := range failures {
			fmt.Fprintf(out, "  %s\n", o.name)
			writeFailure(out, name, o, "      ")
		}
	}
	return counts
}

func writeFailure(out io.Writer, file string, o testOutcome, indent string) {
	if o.exited {
		fmt.Fprintf(out, "%sthe test called exit[]\n", indent)
		return
	}
	e := o.err
	where := file
	if e.file != "" {
		where = e.file
	}
	if e.line > 0 {
		where += fmt.Sprintf(" line %d", e.line)
	}
	lines := strings.Split(e.text, "\n")
	if o.failRun > 0 {
		where += fmt.Sprintf(", benchmark run %d", o.failRun)
	}
	if e.kind != kindTest {
		lines[0] = fmt.Sprintf("stopped with a %s error: %s", e.kind, lines[0])
	}
	fmt.Fprintf(out, "%s%s: %s\n", indent, where, lines[0])
	for _, l := range lines[1:] {
		fmt.Fprintf(out, "%s%s\n", indent, l)
	}
}

func (o testOutcome) timing() string {
	if len(o.runs) == 0 {
		return fmtDuration(o.time)
	}
	var sum time.Duration
	fast, slow := o.runs[0], o.runs[0]
	for _, d := range o.runs {
		sum += d
		fast, slow = min(fast, d), max(slow, d)
	}
	avg := sum / time.Duration(len(o.runs))
	return fmt.Sprintf("%d %s   avg %s   fastest %s   slowest %s", len(o.runs), plural(len(o.runs), "run"), fmtDuration(avg), fmtDuration(fast), fmtDuration(slow))
}

func importsTest(program *ast.Program) bool {
	for _, s := range program.Statements {
		if im, ok := s.(*ast.ImportStatement); ok && im.Path == "test" {
			return true
		}
	}
	return false
}

// runTop runs a program's top-level code, without closing its databases
// (the tests still use them).
func (it *Interpreter) runTop(program *ast.Program) (fe *fatalError, exited bool) {
	defer func() {
		if r := recover(); r != nil {
			switch e := r.(type) {
			case fatalError:
				fe = &e
			case ExitRequest:
				exited = true
			default:
				panic(r)
			}
		}
	}()
	it.evalStatements(program.Statements, it.Global)
	return nil, false
}

// runTest runs one test function: once, then, if its benchmark setting is
// true (the test's own, else the file's), again and again for timing.
func (it *Interpreter) runTest(fn *object.Function) testOutcome {
	o := testOutcome{name: fn.Name}
	if len(fn.Parameters) > 0 {
		e := fatalError{text: fmt.Sprintf("a test function takes no arguments: write def %s[]", fn.Name), kind: kindType}
		o.err = &e
		return o
	}
	env, d, fe, exited := it.runTestOnce(fn)
	o.time, o.err, o.exited = d, fe, exited
	if fe != nil || exited {
		return o
	}
	setting := func(name string) object.Object {
		v, _ := env.Get(name)
		return v
	}
	if b, ok := setting(benchmarkName).(*object.Boolean); !ok || !b.Value {
		if _, isBool := setting(benchmarkName).(*object.Boolean); !isBool && setting(benchmarkName) != nil {
			e := fatalError{text: "benchmark must be true or false, got " + object.Shown(setting(benchmarkName)), kind: kindType}
			o.err = &e
		}
		return o
	}
	runs := -1
	switch r := setting(runsName).(type) {
	case nil, *object.None:
	case *object.Integer:
		if r.Value < 1 {
			e := fatalError{text: "runs must be a whole number of 1 or more, or none, got " + r.Inspect(), kind: kindType}
			o.err = &e
			return o
		}
		runs = int(r.Value)
	default:
		e := fatalError{text: "runs must be a whole number of 1 or more, or none, got " + object.Shown(r), kind: kindType}
		o.err = &e
		return o
	}
	benchtime := time.Second
	if bt := setting(benchtimeName); bt != nil {
		secs, _, ok := numeric(bt)
		if !ok || secs <= 0 || math.IsInf(secs, 0) {
			e := fatalError{text: "benchtime must be a number of seconds above 0, got " + object.Shown(bt), kind: kindType}
			o.err = &e
			return o
		}
		benchtime = time.Duration(secs * float64(time.Second))
	}
	var spent time.Duration
	for n := 1; runs < 0 && (spent < benchtime && n <= 100_000_000) || runs >= 0 && n <= runs; n++ {
		_, d, fe, exited := it.runTestOnce(fn)
		if fe != nil || exited {
			o.err, o.exited, o.failRun = fe, exited, n
			return o
		}
		o.runs = append(o.runs, d)
		spent += d
	}
	return o
}

// runTestOnce calls a test function in a fresh scope and gives back that
// scope (for its settings), how long it took, and how it failed, if it
// did.
func (it *Interpreter) runTestOnce(fn *object.Function) (env *object.Environment, d time.Duration, fe *fatalError, exited bool) {
	defEnv := fn.Env
	if defEnv == nil {
		defEnv = it.Global
	}
	env = object.NewEnclosedEnvironment(defEnv)
	prevFile, prevLine := currentFile, currentLine
	currentFile = defEnv.File()
	start := time.Now()
	defer func() {
		d = time.Since(start)
		currentFile, currentLine = prevFile, prevLine
		it.depth = 0
		if r := recover(); r != nil {
			switch e := r.(type) {
			case fatalError:
				fe = &e
			case ExitRequest:
				exited = true
			default:
				panic(r)
			}
		}
	}()
	it.depth = 1
	it.evalBlock(fn.Body, env)
	return env, 0, nil, false
}

// fmtDuration writes a time briefly: 350ns, 81.2µs, 1.2ms, 2.35s.
func fmtDuration(d time.Duration) string {
	switch {
	case d < time.Microsecond:
		return fmt.Sprintf("%dns", d.Nanoseconds())
	case d < time.Millisecond:
		return fmt.Sprintf("%.1fµs", float64(d)/float64(time.Microsecond))
	case d < time.Second:
		return fmt.Sprintf("%.1fms", float64(d)/float64(time.Millisecond))
	}
	return fmt.Sprintf("%.2fs", d.Seconds())
}
