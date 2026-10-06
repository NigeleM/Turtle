package evaluator

import (
	"Turtle/sqlite"
	"bufio"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"Turtle/lexer"
	"Turtle/parser"
)

// run parses and evaluates src, feeding stdin to any "?" prompts, and
// returns everything written to stdout plus the error from a fatalf, if
// any. Tests in this file must not run in parallel: currentLine
// (evaluator.go) is a package-level variable, safe only because one
// evaluation runs at a time, exactly like the real CLI.
func run(t *testing.T, src, stdin string) (string, error) {
	t.Helper()
	return runIn(t, ".", src, stdin)
}

// runIn is run with imports and file paths resolving relative to dir.
func runIn(t *testing.T, dir, src, stdin string) (string, error) {
	t.Helper()
	return runFull(t, dir, src, stdin, nil)
}

// runScript is runIn for a script named main.t, as "file of" an error
// in it reports.
func runScript(t *testing.T, dir, src string) (string, error) {
	t.Helper()
	scriptName = "main.t"
	defer func() { scriptName = "" }()
	return runFull(t, dir, src, "", nil)
}

// scriptName is the Script runFull gives the interpreter.
var scriptName string

// runFull is runIn plus command-line arguments for system's args[].
func runFull(t *testing.T, dir, src, stdin string, args []string) (string, error) {
	t.Helper()
	p := parser.New(lexer.New(src))
	program := p.ParseProgram()
	if errs := p.Errors(); len(errs) > 0 {
		t.Fatalf("parse error(s) for %q: %v", src, errs)
	}

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	origStdout := os.Stdout
	os.Stdout = w

	outCh := make(chan string, 1)
	go func() {
		var sb strings.Builder
		sc := bufio.NewScanner(r)
		sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
		for sc.Scan() {
			sb.WriteString(sc.Text())
			sb.WriteByte('\n')
		}
		outCh <- sb.String()
	}()

	it := NewWithStdin(dir, strings.NewReader(stdin))
	it.WorkDir = dir
	it.Script = scriptName
	it.Args = args
	runErr := it.Run(program)

	w.Close()
	os.Stdout = origStdout
	out := <-outCh
	return out, runErr
}

func TestArithmeticAndModulo(t *testing.T) {
	cases := []struct{ name, src, want string }{
		{"add", `show 1 + 2 .`, "3\n"},
		{"precedence", `show 1 + 2 * 3 .`, "7\n"},
		{"parens", `show (1 + 2) * 3 .`, "9\n"},
		{"int div truncates", `show 7 / 2 .`, "3\n"},
		{"float div", `show 7.0 / 2 .`, "3.5\n"},
		{"modulo int", `show 10 % 3 .`, "1\n"},
		{"modulo float", `show 10.5 % 3 .`, "1.5\n"},
		{"string concat", `show "a" + "b" .`, "ab\n"},
		{"mixed concat", `show "x=" + 5 .`, "x=5\n"},
		{"unary minus", `show -5 + 8 .`, "3\n"},
		{"comparison", `show 3 < 5 .`, "true\n"},
		{"logical", `show true && false .`, "false\n"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			out, err := run(t, c.src, "")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if out != c.want {
				t.Errorf("got %q, want %q", out, c.want)
			}
		})
	}
}

func TestFatalErrorsHaveLineNumbers(t *testing.T) {
	cases := []struct{ name, src, wantErr string }{
		{"division by zero", "show 1 .\nshow 5 / 0 .", "line 2: division by zero"},
		{"modulo by zero", "show 5 % 0 .", "line 1: modulo by zero"},
		{"undefined variable", "a = 1\nshow b .", `line 2: undefined variable "b"`},
		{"math not imported", `r is 5 at sqrt .`, `line 1: "sqrt" needs "import math" first`},
		{"time not imported", "t = now[]", `line 1: "now" needs "import time" first`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := run(t, c.src, "")
			if err == nil {
				t.Fatalf("expected an error, got none")
			}
			if err.Error() != c.wantErr {
				t.Errorf("got error %q, want %q", err.Error(), c.wantErr)
			}
		})
	}
}

func TestChangeConversions(t *testing.T) {
	cases := []struct{ name, src, want string }{
		{"string to integer statement form mutates in place", `a = "42"
change a to integer .
show a + 1 .`, "43\n"},
		{"expression form does not mutate source", `raw = "7"
n = change raw to integer
show raw .
show n + 1 .`, "7\n8\n"},
		{"string to float", `show change "3.5" to float .`, "3.5\n"},
		{"number to string", `show change 255 to string , "!" .`, "255!\n"},
		{"ascii", `show change "A" to ascii .`, "65\n"},
		{"char", `show change 66 to char .`, "B\n"},
		{"hex from integer", `show change 255 to hex .`, "ff\n"},
		{"hex to integer", `show change "ff" to hex .`, "255\n"},
		{"hex with 0x prefix", `show change "0xff" to hex .`, "255\n"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			out, err := run(t, c.src, "")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if out != c.want {
				t.Errorf("got %q, want %q", out, c.want)
			}
		})
	}
}

func TestChangeInvalidIntegerIsFatal(t *testing.T) {
	// "change <ident> to <type> ." (statement form) requires an
	// identifier, so a bad conversion is exercised via the expression
	// form instead.
	_, err := run(t, `n = change "not a number" to integer`, "")
	if err == nil {
		t.Fatal("expected an error, got none")
	}
	if !strings.Contains(err.Error(), "not a valid integer") {
		t.Errorf("got error %q, want it to mention 'not a valid integer'", err.Error())
	}
}

func TestStringMethods(t *testing.T) {
	// "at" method-call syntax only exists inside "is <receiver> at
	// <method> ." — it's not (yet) a general expression form, so every
	// case here captures each method's result in a variable first, then
	// shows the variable. (This grammar limitation is tracked as a
	// possible future UFCS-style extension.)
	cases := []struct{ name, src, want string }{
		{"upper/lower/trim", `s = "  Hi  "
t is s at trim .
u is t at upper .
l is t at lower .
show u .
show l .`, "HI\nhi\n"},
		{"get", `s = "Hello"
c is s at get 1 .
show c .`, "e\n"},
		{"slice range", `s = "Hello, World"
r is s at slice 0, 5 .
show r .`, "Hello\n"},
		{"slice negative", `s = "Hello, World"
r is s at slice -5 .
show r .`, "World\n"},
		{"split/contains/indexOf/replace", `s = "a,b,c"
p is s at split "," .
hasB is s at contains "b" .
idxC is s at indexOf "c" .
r is s at replace "b", "X" .
show p .
show hasB .
show idxC .
show r .`, "[ \"a\", \"b\", \"c\" ]\ntrue\n4\na,X,c\n"},
		{"isNumber true", `ok is "42" at isNumber .
show ok .`, "true\n"},
		{"isNumber false", `ok is "abc" at isNumber .
show ok .`, "false\n"},
		{"unicode length and indexing are rune-based", `s = "café"
c is s at get 3 .
show length of s .
show c .`, "4\né\n"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			out, err := run(t, c.src, "")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if out != c.want {
				t.Errorf("got %q, want %q", out, c.want)
			}
		})
	}
}

func TestListMethodsAndSlice(t *testing.T) {
	cases := []struct{ name, src, want string }{
		{"add/sort/remove/insert/get", `nums = list [3, 1, 2]
add 4 to nums .
sort nums .
remove 2 from nums .
insert 99 to nums at 0 .
r is nums at get 1 .
show nums .
show r .`, "[ 99, 1, 3, 4 ]\n1\n"},
		{"slice range", `nums = list [10, 20, 30, 40, 50]
r is nums at slice 0, 3 .
show r .`, "[ 10, 20, 30 ]\n"},
		{"slice negative", `nums = list [10, 20, 30, 40, 50]
r is nums at slice -2 .
show r .`, "[ 40, 50 ]\n"},
		{"slice clamps out of range end", `nums = list [1, 2, 3]
r is nums at slice 0, 999 .
show r .`, "[ 1, 2, 3 ]\n"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			out, err := run(t, c.src, "")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if out != c.want {
				t.Errorf("got %q, want %q", out, c.want)
			}
		})
	}
}

func TestMathModuleGating(t *testing.T) {
	src := `import math

r is 16 at sqrt .
show r .

a is -7 at abs .
show a .

r2 is 4.5 at round .
show r2 .
r3 is 4.5 at floor .
show r3 .
r4 is 4.1 at ceil .
show r4 .

p is 2 at pow 10 .
show p .

n is 10 at random .
show n >= 0 && n < 10 .`
	want := "4.0\n7\n5\n4\n5\n1024\ntrue\n"
	out, err := run(t, src, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != want {
		t.Errorf("got %q, want %q", out, want)
	}
}

func TestTimeModuleGating(t *testing.T) {
	src := `import time

t1 = now[]
sleep[20, "ms"]
t2 = now[]
show (t2 - t1) >= 20 .`
	out, err := run(t, src, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != "true\n" {
		t.Errorf("got %q, want %q", out, "true\n")
	}
}

func TestSleepDefaultUnitIsSeconds(t *testing.T) {
	src := `import time

t1 = now[]
sleep[0.02]
t2 = now[]
show (t2 - t1) >= 20 .`
	out, err := run(t, src, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != "true\n" {
		t.Errorf("got %q, want %q", out, "true\n")
	}
}

func TestGlobalReadLocalWriteShadow(t *testing.T) {
	src := `PI = 3.14159
count = 0
nums = list [1, 2, 3]

def circleArea[r]
    return PI * r * r
def [end]

def bump[]
    count = count + 1
    return count
def [end]

def addToNums[v]
    add v to nums .
def [end]

show circleArea[2] .
show bump[] .
show bump[] .
show count .
addToNums[99]
show nums .`
	want := "12.56636\n1\n1\n0\n[ 1, 2, 3, 99 ]\n"
	out, err := run(t, src, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != want {
		t.Errorf("got %q, want %q", out, want)
	}
}

func TestRecursion(t *testing.T) {
	src := `def fact[n]
    if ] n <= 1 [
        return 1
    else ]
        return n * fact[n - 1]
    if [end]
def [end]

show fact[5] .`
	out, err := run(t, src, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != "120\n" {
		t.Errorf("got %q, want %q", out, "120\n")
	}
}

func TestLoopBreakContinue(t *testing.T) {
	src := `total = 0
[loop][k = 0; k < 10; k++]
    if ] k == 5 [
        break
    if [end]
    total = total + k
[loop][end]
show total .

c = 0
n = 0
[loop][c < 5]
    c = c + 1
    if ] c == 3 [
        continue
    if [end]
    n = n + 1
[loop][end]
show n .`
	out, err := run(t, src, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != "10\n4\n" {
		t.Errorf("got %q, want %q", out, "10\n4\n")
	}
}

func TestIfElseNesting(t *testing.T) {
	src := `if ] 189 > 99 [
    show "outer-true" .
    [if ] 187 > 89 [
        show "inner-true" .
    [else ]
        show "inner-false" .
else ]
    show "outer-false" .
if [end]`
	out, err := run(t, src, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != "outer-true\ninner-true\n" {
		t.Errorf("got %q, want %q", out, "outer-true\ninner-true\n")
	}
}

func TestInputStatementAndEOF(t *testing.T) {
	t.Run("reads a line", func(t *testing.T) {
		// The prompt itself is printed to stdout (no trailing newline),
		// same stream "show" writes to, so it's part of the captured
		// output too.
		out, err := run(t, `name = ? "prompt: "
show "hi ", name .`, "world\n")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := "prompt: hi world\n"
		if out != want {
			t.Errorf("got %q, want %q", out, want)
		}
	})

	t.Run("EOF is a fatal error, not an infinite loop", func(t *testing.T) {
		_, err := run(t, `x = ? "prompt: "`, "")
		if err == nil {
			t.Fatal("expected an error at EOF, got none")
		}
		if !strings.Contains(err.Error(), "unexpected end of input") {
			t.Errorf("got error %q, want it to mention EOF", err.Error())
		}
	})
}

func TestUserFunctionNamedNowIsNotShadowedByBuiltin(t *testing.T) {
	src := `def now[]
    return 42
def [end]

show now[] .`
	out, err := run(t, src, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != "42\n" {
		t.Errorf("got %q, want %q (user's own now[] must win, time isn't even imported)", out, "42\n")
	}
}

func TestClosuresAndFunctionValues(t *testing.T) {
	cases := []struct{ name, src, want string }{
		{"closure captures parameter", `def make_adder[n]
    def adder[x]
        return x + n
    def [end]
    return adder
def [end]

add5 = make_adder[5]
add10 = make_adder[10]
show add5[1] .
show add10[1] .`, "6\n11\n"},
		{"closure mutates captured list", `def make_counter[]
    counts = list [0]
    def inc[]
        c is counts at pop .
        c = c + 1
        add c to counts .
        return c
    def [end]
    return inc
def [end]

a = make_counter[]
b = make_counter[]
show a[] .
show a[] .
show b[] .`, "1\n2\n1\n"},
		{"assignment to captured name shadows", `def outer[]
    n = 1
    def inner[]
        n = 99
        return n
    def [end]
    show inner[] .
    return n
def [end]

show outer[] .`, "99\n1\n"},
		{"closure reads globals", `greeting = "hi"
def outer[]
    def inner[]
        return greeting
    def [end]
    return inner[]
def [end]

show outer[] .`, "hi\n"},
		{"nested def is not global", `def outer[]
    def inner[]
        return 1
    def [end]
    return inner[]
def [end]

show outer[] .
show inner[] .`, "1\n"},
		{"top-level function as argument", `def double[x]
    return x * 2
def [end]

def apply[f, v]
    return f[v]
def [end]

show apply[double, 21] .
g = double
show g[4] .`, "42\n8\n"},
		{"function equality is identity", `def f[]
    return 1
def [end]
def g[]
    return 1
def [end]

show f == f .
show f == g .`, "true\nfalse\n"},
		{"recursive nested def", `def run[]
    def fact[n]
        if ] n <= 1 [
            return 1
        if [end]
        return n * fact[n - 1]
    def [end]
    return fact[5]
def [end]

show run[] .`, "120\n"},
		{"lexical not dynamic scope", `def reader[]
    return secret
def [end]

def caller[]
    secret = "caller's local"
    return reader[]
def [end]

secret = "global"
show caller[] .`, "global\n"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			out, err := run(t, c.src, "")
			if c.name == "nested def is not global" {
				if err == nil || !strings.Contains(err.Error(), `undefined function "inner"`) {
					t.Fatalf("want undefined function error, got %v", err)
				}
			} else if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if out != c.want {
				t.Errorf("got %q, want %q", out, c.want)
			}
		})
	}
}

func TestNone(t *testing.T) {
	cases := []struct{ name, src, want string }{
		{"literal", `x = none
show x .`, "none\n"},
		{"no return yields none", `def f[]
    y = 1
def [end]

show f[] .`, "none\n"},
		{"bare return yields none", `def f[x]
    if ] x > 0 [
        return
    if [end]
    return x
def [end]

show f[1] .
show f[-1] .`, "none\n-1\n"},
		{"equality", `show none == none .
show none != none .
show 0 == none .
show "none" == none .`, "true\nfalse\nfalse\nfalse\n"},
		{"falsy", `if ] none [
    show "yes" .
else ]
    show "no" .
if [end]
show !none .`, "no\ntrue\n"},
		{"none + none is none", `show none + none .`, "none\n"},
		{"shows inside text by interpolation", `x = none
show "x={x}" .`, "x=none\n"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			out, err := run(t, c.src, "")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if out != c.want {
				t.Errorf("got %q, want %q", out, c.want)
			}
		})
	}
}

func TestNoneArithmeticIsFatal(t *testing.T) {
	_, err := run(t, `show none - 1 .`, "")
	if err == nil || !strings.Contains(err.Error(), "NONE") {
		t.Fatalf("want a type error naming NONE, got %v", err)
	}
}

// moduleDir writes each name -> source pair to <tmp>/<name>.t and returns
// the directory, for import tests.
func moduleDir(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for name, src := range files {
		if err := os.WriteFile(filepath.Join(dir, name+".t"), []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

const mylibSrc = `SECRET = "mylib's"
def helper[x]
    return x * 10
def [end]
def binary[x]
    return helper[x] + 1
def [end]
def now[]
    return "mylib now"
def [end]
def reveal[]
    return SECRET
def [end]
show "mylib loaded" .`

func TestImports(t *testing.T) {
	dir := moduleDir(t, map[string]string{
		"mylib": mylibSrc,
		"other": `import mylib [binary]
def twice[x]
    return binary[x] * 2
def [end]`,
		"a": `import b`,
		"b": `import a`,
	})
	cases := []struct{ name, src, want, wantErr string }{
		{name: "full import, plain names", src: `import mylib
show binary[2] .
show helper[1] .`, want: "mylib loaded\n21\n10\n"},
		{name: "module runs once", src: `import mylib
import mylib
import other
show twice[2] .`, want: "mylib loaded\n42\n"},
		{name: "module variables are private but its functions see them", src: `import mylib
SECRET = "main's"
show reveal[] .
show SECRET .`, want: "mylib loaded\nmylib's\nmain's\n"},
		{name: "module variables don't leak", src: `import mylib
show SECRET .`, wantErr: `undefined variable "SECRET"`},
		{name: "partial import limits plain names", src: `import mylib [binary]
show binary[1] .
show reveal[] .`, wantErr: `undefined function "reveal"`},
		{name: "partial import still allows qualifying listed names", src: `import mylib [binary]
show mylib binary[1] .`, want: "mylib loaded\n11\n"},
		{name: "partial import blocks qualified unlisted names", src: `import mylib [binary]
show mylib reveal[] .`, wantErr: `"reveal" isn't imported`},
		{name: "unknown name in list", src: `import mylib [binery]`, wantErr: `module "mylib" has no "binery"`},
		{name: "clash requires qualifying", src: `import time
import mylib
show now[] .`, wantErr: `"now" is provided by more than one import (time, mylib)`},
		{name: "qualified call resolves clash", src: `import time
import mylib
show mylib now[] .
show time now[] > 0 .
show binary[1] .`, want: "mylib loaded\nmylib now\ntrue\n11\n"},
		{name: "clash only for that name", src: `import time [now]
import mylib [now]
show time now[] > 0 .`, want: "mylib loaded\ntrue\n"},
		{name: "own def beats imports", src: `import time
import mylib
def now[]
    return "mine"
def [end]
show now[] .
show mylib now[] .`, want: "mylib loaded\nmine\nmylib now\n"},
		{name: "qualified function value", src: `import mylib
f = mylib binary
show f[3] .`, want: "mylib loaded\n31\n"},
		{name: "unknown module", src: `show nope now[] .`, wantErr: `unknown module or variable "nope"`},
		{name: "circular import", src: `import a`, wantErr: "circular import"},
		{name: "builtin partial import", src: `import math [sqrt]
r is 16 at sqrt .
show r .
r is 2 at pow 3 .`, want: "4.0\n", wantErr: `"pow" isn't imported — add it to "import math [...]"`},
		{name: "builtin function partial import", src: `import time [sleep]
sleep[0.001]
show now[] .`, wantErr: `"now" isn't imported — add it to "import time [...]"`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			out, err := runIn(t, dir, c.src, "")
			if c.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), c.wantErr) {
					t.Fatalf("want error containing %q, got %v", c.wantErr, err)
				}
			} else if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if c.want != "" && out != c.want {
				t.Errorf("got %q, want %q", out, c.want)
			}
		})
	}
}

func TestStructuralEquality(t *testing.T) {
	cases := []struct{ name, src, want string }{
		{"sets ignore order", `show set [1, 2] == set [2, 1] .`, "true\n"},
		{"set of equal sets dedups", `a = set [1, 2]
b = set [2, 1]
c = set [a, b]
show length of c .`, "1\n"},
		{"number and string are distinct", `s = set [1, "1"]
show length of s .
show "1" == 1 .`, "2\nfalse\n"},
		{"int and float are equal", `s = set [1, 1.0]
show length of s .
show 1 == 1.0 .`, "1\ntrue\n"},
		{"lists keep order", `show list [1, 2] == list [2, 1] .
show list [set [1, 2]] == list [set [2, 1]] .`, "false\ntrue\n"},
		{"maps ignore order", `show map ["x": 1, "y": 2] == map ["y": 2, "x": 1] .`, "true\n"},
		{"equal sets are the same map key", `k = map [set [1, 2]: "pair"]
v is k at get set [2, 1] .
show v .`, "pair\n"},
		{"searches use equality", `nums = list [1, "1", 2]
n is nums at count 1 .
show n .
i is nums at index "1" .
show i .
remove "1" from nums .
show nums .`, "1\n1\n[ 1, 2 ]\n"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			out, err := run(t, c.src, "")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if out != c.want {
				t.Errorf("got %q, want %q", out, c.want)
			}
		})
	}
}

func TestGivesSentenceCallsAndDataLib(t *testing.T) {
	cases := []struct{ name, src, want, wantErr string }{
		{name: "gives one and many params", src: `double = x gives x * 2
plus = [a, b] gives a + b
show double[4] .
show plus[2, 3] .`, want: "8\n5\n"},
		{name: "gives with no params", src: `def twice[f]
    f[]
    f[]
def [end]
twice[[] gives
    show "hi" .
gives [end]]`, want: "hi\nhi\n"},
		{name: "gives captures scope", src: `n = 10
f = x gives x + n
show f[1] .`, want: "11\n"},
		{name: "block form", src: `f = [x] gives
    if ] x > 3 [
        return x * 100
    if [end]
    return x
gives [end]
show f[1] .
show f[5] .`, want: "1\n500\n"},
		{name: "process list in place", src: `import data
nums = list [5, 3, 8, 1]
nums process x gives x + 1 .
show nums .`, want: "[ 6, 4, 9, 2 ]\n"},
		{name: "process with method in body", src: `import data
words = list ["hey", "do"]
words process x gives x at upper .
show words .`, want: "[ \"HEY\", \"DO\" ]\n"},
		{name: "process with named function", src: `import data
def double[x]
    return x * 2
def [end]
nums = list [1, 2]
nums process double .
show nums .`, want: "[ 2, 4 ]\n"},
		{name: "process block", src: `import data
vals = list [1, 5]
vals process [x] gives
    if ] x > 3 [
        return 0
    if [end]
    return x
gives [end]
show vals .`, want: "[ 1, 0 ]\n"},
		{name: "process map values and key-value", src: `import data
ages = map ["Alice": 30, "Bob": 25]
ages process x gives x + 1 .
show ages .
ages process [k, v] gives k + "=" + v .
show ages .`, want: "{ \"Alice\": 31, \"Bob\": 26 }\n{ \"Alice\": \"Alice=31\", \"Bob\": \"Bob=26\" }\n"},
		{name: "process set dedups", src: `import data
s = set [1, 2, 3, 4]
s process x gives x % 2 .
show s .`, want: "{ 1, 0 }\n"},
		{name: "keep filters list, set, map", src: `import data
nums = list [5, 3, 8, 1]
nums keep x gives x > 3 .
show nums .
s = set [1, 2, 3]
s keep x gives x != 2 .
show s .
m = map ["a": 1, "b": 5]
m keep [k, v] gives v > 2 .
show m .`, want: "[ 5, 8 ]\n{ 1, 3 }\n{ \"b\": 5 }\n"},
		{name: "copy leaves original", src: `import data
orig = list [1, 2]
big = copy[orig]
big process x gives x * 10 .
show orig .
show big .`, want: "[ 1, 2 ]\n[ 10, 20 ]\n"},
		{name: "sentence call with user function and args", src: `def scale[xs, f, times]
    out = list []
    [loop][x in xs]
        add f[x] * times to out .
    [loop][end]
    return out
def [end]
nums = list [1, 2]
r = nums scale x gives x + 1, 10
show r .`, want: "[ 20, 30 ]\n"},
		{name: "sentence call with no args", src: `def total[xs]
    t = 0
    [loop][x in xs]
        t = t + x
    [loop][end]
    return t
def [end]
nums = list [1, 2, 3]
show nums total .`, want: "6\n"},
		{name: "process needs import", src: `nums = list [1]
nums process x gives x .`, wantErr: `"process" needs "import data" first`},
		{name: "process needs a function", src: `import data
nums = list [1]
nums process 5 .`, wantErr: "'process' needs a function"},
		{name: "variable named like a module is refused", src: `import data
data = list [1]
data process x gives x .`, wantErr: `"data" is both a variable and an imported module`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			out, err := run(t, c.src, "")
			if c.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), c.wantErr) {
					t.Fatalf("want error containing %q, got %v", c.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if out != c.want {
				t.Errorf("got %q, want %q", out, c.want)
			}
		})
	}
}

func TestForEachLoop(t *testing.T) {
	cases := []struct{ name, src, want string }{
		{"list with break and continue", `[loop][x in list [1, 2, 3, 4, 5]]
    if ] x == 2 [
        continue
    if [end]
    if ] x == 5 [
        break
    if [end]
    show x .
[loop][end]`, "1\n3\n4\n"},
		{"index and element", `[loop][i, w in list ["a", "b"]]
    show i, w .
[loop][end]`, "0a\n1b\n"},
		{"map keys, then key and value", `m = map ["x": 1, "y": 2]
[loop][k in m]
    show k .
[loop][end]
[loop][k, v in m]
    show k, "=", v .
[loop][end]`, "x\ny\nx=1\ny=2\n"},
		{"string characters", `[loop][c in "hi"]
    show c .
[loop][end]`, "h\ni\n"},
		{"loop name restored", `x = "before"
[loop][x in list [1, 2]]
[loop][end]
show x .`, "before\n"},
		{"snapshot: adding inside doesn't loop forever", `nums = list [1, 2]
[loop][x in nums]
    add x to nums .
[loop][end]
show nums .`, "[ 1, 2, 1, 2 ]\n"},
		{"return from inside", `def first_big[xs]
    [loop][x in xs]
        if ] x > 2 [
            return x
        if [end]
    [loop][end]
    return none
def [end]
show first_big[list [1, 5, 9]] .`, "5\n"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			out, err := run(t, c.src, "")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if out != c.want {
				t.Errorf("got %q, want %q", out, c.want)
			}
		})
	}
}

func TestCollectionOperators(t *testing.T) {
	cases := []struct{ name, src, want string }{
		{"list +", `show list [5, 3] + list [6, 7] .`, "[ 5, 3, 6, 7 ]\n"},
		{"list -", `show list [1, 2, 3, 2] - list [2] .`, "[ 1, 3 ]\n"},
		{"set +", `show set [1, 2] + set [2, 3] .`, "{ 1, 2, 3 }\n"},
		{"set -", `show set [1, 2, 3] - set [2] .`, "{ 1, 3 }\n"},
		{"map + keeps first on shared key", `show map ["a": 1, "b": 2] + map ["b": 99, "c": 3] .`, "{ \"a\": 1, \"b\": 2, \"c\": 3 }\n"},
		{"map - by key", `show map ["a": 1, "b": 2] - map ["b": 0] .`, "{ \"a\": 1 }\n"},
		{"operands unchanged", `a = list [1]
b = a + list [2]
show a .`, "[ 1 ]\n"},
		{"values taken out of lists add normally", `a = list [1, 2]
b = list [10, 20]
x = a at get[0]
y = b at get[1]
show x + y .`, "21\n"},
		{"method call in expression", `show "hi" at upper .
t = "Hello, World" at slice[0, 5]
show t .`, "HI\nHello\n"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			out, err := run(t, c.src, "")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if out != c.want {
				t.Errorf("got %q, want %q", out, c.want)
			}
		})
	}
}

func TestSystemLibrary(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("hi"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(dir, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name, src string
		args      []string
		want      string
		wantErr   string
	}{
		{name: "args", src: `import system
a = args[]
show a .
show length of a .`, args: []string{"one", "two words"}, want: "[ \"one\", \"two words\" ]\n2\n"},
		{name: "no args is an empty list", src: `import system
show length of args[] .`, want: "0\n"},
		{name: "exists, isFile, isFolder", src: `import system
show exists["notes.txt"], isFile["notes.txt"], isFolder["notes.txt"] .
show exists["sub"], isFile["sub"], isFolder["sub"] .
show exists["missing.txt"], isFile["missing.txt"], isFolder["missing.txt"] .`,
			want: "truetruefalse\ntruefalsetrue\nfalsefalsefalse\n"},
		{name: "sentence style", src: `import system
p = "notes.txt"
if ] p exists [
    show "yes" .
if [end]`, want: "yes\n"},
		{name: "needs import", src: `show exists["notes.txt"] .`, wantErr: `"exists" needs "import system" first`},
		{name: "path must be a string", src: `import system
show exists[5] .`, wantErr: "'exists' needs a path string"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			out, err := runFull(t, dir, c.src, "", c.args)
			if c.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), c.wantErr) {
					t.Fatalf("want error containing %q, got %v", c.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if out != c.want {
				t.Errorf("got %q, want %q", out, c.want)
			}
		})
	}
}

func TestAssemble(t *testing.T) {
	dir := moduleDir(t, map[string]string{
		"shapes": `assemble Point [x, y]`,
	})
	cases := []struct{ name, src, want, wantErr string }{
		{name: "construct, show, read fields", src: `assemble Order [item, qty, price]
o = Order["pen", 3, 1.5]
show o .
show qty of o * price of o .`, want: "Order { item: \"pen\", qty: 3, price: 1.5 }\n4.5\n"},
		{name: "change a field", src: `assemble Order [item, qty]
o = Order["pen", 3]
qty of o = 10
show o .`, want: "Order { item: \"pen\", qty: 10 }\n"},
		{name: "shared reference, copy separates", src: `import data
assemble P [v]
a = P[1]
b = a
v of b = 2
c = copy[a]
v of c = 3
show v of a, v of b, v of c .`, want: "223\n"},
		{name: "equality is structural within one type", src: `assemble P [v]
assemble Q [v]
show P[1] == P[1] .
show P[1] == P[2] .
show P[1] == Q[1] .
s = set [P[1], P[1]]
show length of s .`, want: "true\nfalse\nfalse\n1\n"},
		{name: "nested fields", src: `assemble P [x, y]
assemble L [a, b]
l = L[P[0, 0], P[3, 4]]
show x of b of l .
x of b of l = 7
show l .`, want: "3\nL { a: P { x: 0, y: 0 }, b: P { x: 7, y: 4 } }\n"},
		{name: "imported from a module", src: `import shapes [Point]
p = Point[1, 2]
show y of p .`, want: "2\n"},
		{name: "local to a function", src: `def make[]
    assemble Pair [a, b]
    return Pair[1, 2]
def [end]
show make[] .`, want: "Pair { a: 1, b: 2 }\n"},
		{name: "constructor is a value", src: `assemble P [v]
mk = P
show mk[5] .
show mk .`, want: "P { v: 5 }\nassemble P\n"},
		{name: "wrong value count", src: `assemble A [x, y]
a = A[1]`, wantErr: "A needs 2 value(s), one per field (x, y), got 1"},
		{name: "unknown field", src: `assemble A [x]
a = A[1]
show z of a .`, wantErr: `A has no field "z" (its fields: x)`},
		{name: "unknown field on change", src: `assemble A [x]
a = A[1]
z of a = 2`, wantErr: `A has no field "z"`},
		{name: "field of a non-assembled value", src: `n = 5
show x of n .`, wantErr: "'x of' needs an assembled value or an error, got INTEGER"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			out, err := runIn(t, dir, c.src, "")
			if c.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), c.wantErr) {
					t.Fatalf("want error containing %q, got %v", c.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if out != c.want {
				t.Errorf("got %q, want %q", out, c.want)
			}
		})
	}
}

func TestStringsLibrary(t *testing.T) {
	cases := []struct{ name, src, want, wantErr string }{
		{name: "find", src: `import strings
s = "hello world"
show s find "wor" .
show s find "xyz" .
show find["héllo", "llo"] .`, want: "6\n-1\n2\n"},
		{name: "substring", src: `import strings
s = "Hello, World"
show s substring 0, 5 .
show s substring (-5) .
show substring[s, 7] .`, want: "Hello\nWorld\nWorld\n"},
		{name: "isinstring, literal on the left", src: `import strings
line = "hello world"
show "wor" isinstring line .
show "xyz" isinstring line .`, want: "true\nfalse\n"},
		{name: "join", src: `import strings
words = list ["a", "b", "c"]
show words join ", " .
show join[words] .
show join[list [1, 2.5, true], "-"] .
show join[list [], ","] == "" .`, want: "a, b, c\nabc\n1-2.5-true\ntrue\n"},
		{name: "join needs a list", src: `import strings
show join["abc", ","] .`, wantErr: "'join' needs a list or set, got STRING"},
		{name: "partial import", src: `import strings [join]
show find["ab", "b"] .`, wantErr: `"find" isn't imported`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			out, err := run(t, c.src, "")
			if c.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), c.wantErr) {
					t.Fatalf("want error containing %q, got %v", c.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if out != c.want {
				t.Errorf("got %q, want %q", out, c.want)
			}
		})
	}
}

func TestCommandLineTool(t *testing.T) {
	work := t.TempDir()
	if err := os.WriteFile(filepath.Join(work, "notes.txt"), []byte("one\ntwo\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("TURTLE_TEST_VAR", "on")
	if err := os.MkdirAll(filepath.Join(work, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(work, "sub", "a.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	t.Run("exit code ends the program", func(t *testing.T) {
		out, err := runFull(t, work, `import system
show "before" .
exit[3]
show "after" .`, "", nil)
		var ex ExitRequest
		if !errors.As(err, &ex) || ex.Code != 3 {
			t.Fatalf("want ExitRequest{3}, got %v", err)
		}
		if out != "before\n" {
			t.Errorf("got %q", out)
		}
	})
	t.Run("exit with no code is 0", func(t *testing.T) {
		_, err := runFull(t, work, "import system\nexit[]\n", "", nil)
		var ex ExitRequest
		if !errors.As(err, &ex) || ex.Code != 0 {
			t.Fatalf("want ExitRequest{0}, got %v", err)
		}
	})
	cases := []struct {
		name, src string
		args      []string
		want      string
		wantErr   string
	}{
		{name: "env", src: `import system
show env["TURTLE_TEST_VAR"] .
show env["TURTLE_SURELY_UNSET_123"] .`, want: "on\nnone\n"},
		{name: "read a file named by an argument", src: `import system
name is args[] at get 0 .
[read] name to lines [end]
show lines .`, args: []string{"notes.txt"}, want: "[ \"one\", \"two\" ]\n"},
		{name: "bare word with no such variable is a filename", src: `[write] plain
"x"
[end]
[read] plain to l [end]
show l .`, want: "[ \"x\" ]\n"},
		{name: "contents", src: `import system
show contents["sub"] .
c = contents[]
show c at find["notes.txt"] .
f = "sub"
n = f contents
show length of n .`, want: "[ \"a.txt\" ]\ntrue\n1\n"},
		{name: "contents of a missing folder", src: `import system
show contents["nope"] .`, wantErr: "'contents' nope:"},
		{name: "exit code must be an integer", src: `import system
exit["no"]`, wantErr: "'exit' code must be an integer"},
		{name: "length of binds tightly", src: `a = list []
if ] length of a == 0 [
    show "empty" .
if [end]
show length of list [1, 2] + 1 .`, want: "empty\n3\n"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			out, err := runFull(t, work, c.src, "", c.args)
			if c.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), c.wantErr) {
					t.Fatalf("want error containing %q, got %v", c.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if out != c.want {
				t.Errorf("got %q, want %q", out, c.want)
			}
		})
	}
}

// File paths resolve from where turtle runs (WorkDir), imports from the
// script's folder (Dir) — the two differ when a tool is run from
// elsewhere.
func TestPathsVersusImports(t *testing.T) {
	scriptDir, work := t.TempDir(), t.TempDir()
	if err := os.WriteFile(filepath.Join(scriptDir, "helpers.t"), []byte("def hi[]\n    return \"hi\"\ndef [end]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(work, "data.txt"), []byte("work\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	p := parser.New(lexer.New(`import helpers
import system
[read] data.txt to l [end]
show hi[], " ", l .
show exists["helpers.t"] .
show scriptFolder[] == "` + scriptDir + `" .`))
	program := p.ParseProgram()
	if errs := p.Errors(); len(errs) > 0 {
		t.Fatal(errs)
	}
	r, w, _ := os.Pipe()
	orig := os.Stdout
	os.Stdout = w
	it := New(scriptDir)
	it.WorkDir = work
	runErr := it.Run(program)
	w.Close()
	os.Stdout = orig
	buf := new(strings.Builder)
	sc := bufio.NewScanner(r)
	for sc.Scan() {
		buf.WriteString(sc.Text() + "\n")
	}
	if runErr != nil {
		t.Fatal(runErr)
	}
	if want := "hi [ \"work\" ]\nfalse\ntrue\n"; buf.String() != want {
		t.Errorf("got %q, want %q", buf.String(), want)
	}
}

func TestStressFixes(t *testing.T) {
	cases := []struct{ name, src, want, wantErr string }{
		{name: "closures made in for-each keep their pass's value", src: `def make[]
    fs = list []
    [loop][x in list [1, 2, 3]]
        add [] gives x to fs .
    [loop][end]
    return fs
def [end]
fs = make[]
a is fs at get 0 .
c is fs at get 2 .
show a[], c[] .`, want: "13\n"},
		{name: "closure made in c-style loop survives the loop", src: `fs = list []
[loop][i = 0 ; i < 3 ; i++]
    add [] gives i to fs .
[loop][end]
f is fs at get 0 .
show f[] .`, want: "3\n"},
		{name: "loop body assignments still reach outside", src: `t = 0
[loop][x in list [1, 2, 3]]
    t = t + x
    last = x
[loop][end]
show t, last .`, want: "63\n"},
		{name: "loop names don't touch outside names", src: `x = "outer"
i = "outer"
[loop][x in list [1]]
[loop][end]
[loop][i = 0 ; i < 2 ; i++]
[loop][end]
show x, i .`, want: "outerouter\n"},
		{name: "def inside a top-level loop is still top-level", src: `[loop][x in list [1]]
    def f[]
        return 5
    def [end]
[loop][end]
show f[] .`, want: "5\n"},
		{name: "runaway recursion is a clean error", src: `def f[n]
    return f[n + 1]
def [end]
f[0]`, wantErr: "recursion too deep"},
		{name: "get rejects negative indexes", src: `l = list [1, 2, 3]
a is l at get -1 .`, wantErr: "index -1 out of range for list (length 3)"},
		{name: "slice still counts from the end", src: `l = list [1, 2, 3, 4]
a is l at slice -2 .
s is "Hello, World" at slice -5 .
show a, s .`, want: "[ 3, 4 ]World\n"},
		{name: "floats show as floats", src: `import math
show 4.0, " ", 2.5 * 2, " ", 7 / 2, " ", 1.5 .
r is 16 at sqrt .
p is 2 at pow 10 .
q is 2 at pow -1 .
show r, " ", p, " ", q .`, want: "4.0 5.0 3 1.5\n4.0 1024 0.5\n"},
		{name: "function arg count says function", src: `import strings
show find["a"] .`, wantErr: `function "find" expects 2 argument(s)`},
		{name: "map keys keep their type", src: `m = map [1: "int", "1": "str", 2.5: "f", true: "b"]
show length of m .
a is m at get 1 .
b is m at get "1" .
c is m at get 1.0 .
show a, b, c .
[loop][k in map [1: "a", 2: "b"]]
    show k + 1 .
[loop][end]
show min of map [3: "c", 10: "x", 2: "b"] .
i is map ["a": 1] at invert .
v is i at get 1 .
show v .`, want: "4\nintstrint\n2\n3\n2\na\n"},
		{name: "missing map key error quotes strings", src: `m = map [1: "a"]
v is m at get "1" .`, wantErr: `key "1" not found in map`},
		{name: "big set stays correct", src: `s = set []
[loop][i = 0 ; i < 3000 ; i++]
    add i to s .
    add i to s .
[loop][end]
remove 5 from s .
add 5 to s .
add 5 to s .
show length of s .
f is s at find 2999 .
g is s at find 3000 .
show f, g .`, want: "3000\ntruefalse\n"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			out, err := run(t, c.src, "")
			if c.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), c.wantErr) {
					t.Fatalf("want error containing %q, got %v", c.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if out != c.want {
				t.Errorf("got %q, want %q", out, c.want)
			}
		})
	}
}

func TestSentenceCallsWithoutParentheses(t *testing.T) {
	cases := []struct{ name, src, want string }{
		{"one argument inside brackets", `import strings
def pair[a, b]
    return a + "|" + b
def [end]
t = "Hello"
show pair[t find "l", 7] .`, "2|7\n"},
		{"outside brackets takes all arguments", `import strings
t = "Hello, World"
show t substring 0, 5 .`, "Hello\n"},
		{"inside list and map literals", `import strings
t = "Hello"
show list [t find "e", t find "o"] .
show map ["at": t find "l"] .`, "[ 1, 4 ]\n{ \"at\": 2 }\n"},
		{"negative argument", `import strings
t = "Hello, World"
show t substring -5 .
show list [t substring -5, 1] .`, "World\n[ \"World\", 1 ]\n"},
		{"spaced or unspaced minus is subtraction", `def total[xs]
    t = 0
    [loop][x in xs]
        t = t + x
    [loop][end]
    return t
def [end]
n = list [1, 2]
show n total - 1 .
show n total-1 .`, "2\n2\n"},
		{"not before a sentence", `import system
name = "surely-missing.file"
if ] !name exists [
    show "missing" .
if [end]`, "missing\n"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			out, err := run(t, c.src, "")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if out != c.want {
				t.Errorf("got %q, want %q", out, c.want)
			}
		})
	}
}

// testdata/everything.t checks its own results and exits 1 on any failure.
func TestEverythingScript(t *testing.T) {
	dir, err := filepath.Abs("../testdata")
	if err != nil {
		t.Fatal(err)
	}
	src, err := os.ReadFile(filepath.Join(dir, "everything.t"))
	if err != nil {
		t.Fatal(err)
	}
	work := t.TempDir()
	if err := os.WriteFile(filepath.Join(work, "everything.t"), src, 0o644); err != nil {
		t.Fatal(err)
	}
	p := parser.New(lexer.New(string(src)))
	program := p.ParseProgram()
	if errs := p.Errors(); len(errs) > 0 {
		t.Fatal(errs)
	}
	r, w, _ := os.Pipe()
	orig := os.Stdout
	os.Stdout = w
	it := NewWithStdin(dir, strings.NewReader("5\n"))
	it.WorkDir = work
	it.Args = []string{"first", "second arg"}
	runErr := it.Run(program)
	w.Close()
	os.Stdout = orig
	var out strings.Builder
	sc := bufio.NewScanner(r)
	for sc.Scan() {
		out.WriteString(sc.Text() + "\n")
	}
	if runErr != nil || !strings.Contains(out.String(), "failures: 0") {
		t.Fatalf("everything.t failed (err %v):\n%s", runErr, out.String())
	}
}

func TestTypedDisplayAndIntegerLimits(t *testing.T) {
	cases := []struct{ name, src, want, wantErr string }{
		{name: "strings quoted inside collections only", src: `show map [1: "a", "1": "b", true: list ["x", 3, none]] .
show "plain" .
assemble P [name]
show P["Ana"] .`, want: "{ 1: \"a\", \"1\": \"b\", true: [ \"x\", 3, none ] }\nplain\nP { name: \"Ana\" }\n"},
		{name: "limits themselves are fine", src: `hi = 9223372036854775807
lo = -9223372036854775807 - 1
show hi, " ", lo .`, want: "9223372036854775807 -9223372036854775808\n"},
		{name: "add overflow", src: `show 9223372036854775807 + 1 .`, wantErr: "integer overflow: 9223372036854775807 + 1 is past the integer limits"},
		{name: "subtract overflow", src: `lo = -9223372036854775807 - 1
show lo - 1 .`, wantErr: "integer overflow"},
		{name: "multiply overflow", src: `show 4611686018427387904 * 2 .`, wantErr: "integer overflow"},
		{name: "negate overflow", src: `lo = -9223372036854775807 - 1
show -lo .`, wantErr: "integer overflow"},
		{name: "floats go further", src: `show 9223372036854775807 * 10.0 > 9223372036854775807 .`, want: "true\n"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			out, err := run(t, c.src, "")
			if c.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), c.wantErr) {
					t.Fatalf("want error containing %q, got %v", c.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if out != c.want {
				t.Errorf("got %q, want %q", out, c.want)
			}
		})
	}
}

func TestIsEmpty(t *testing.T) {
	src := `show list [] at isEmpty, list [1] at isEmpty .
show set [] at isEmpty, set [1] at isEmpty .
show map [] at isEmpty, map ["a": 1] at isEmpty .
show "" at isEmpty, "x" at isEmpty .
found = list []
if ] found at isEmpty [
    show "nothing found" .
if [end]
e is found at isEmpty .
show e .`
	out, err := run(t, src, "")
	if err != nil {
		t.Fatal(err)
	}
	want := "truefalse\ntruefalse\ntruefalse\ntruefalse\nnothing found\ntrue\n"
	if out != want {
		t.Errorf("got %q, want %q", out, want)
	}
}

func TestEmptyCollectionsAreFalsy(t *testing.T) {
	src := `[loop][c in list [list [], set [], map [], "", 0, none, list [0], set [""], map ["a": none], "x"]]
    if ] c [
        show "T" .
    else ]
        show "F" .
    if [end]
[loop][end]
show !list [] .
show list [] || "default" .`
	out, err := run(t, src, "")
	if err != nil {
		t.Fatal(err)
	}
	want := "F\nF\nF\nF\nF\nF\nT\nT\nT\nT\ntrue\ntrue\n"
	if out != want {
		t.Errorf("got %q, want %q", out, want)
	}
}

func TestNotWithMethodsAndValueSubjects(t *testing.T) {
	src := `import data
import strings
rows = list [list [], list [1]]
rows keep r gives !r at isEmpty .
show rows .
s = "abc"
b is !s at contains "z" .
show b .
show !s at isEmpty .
show list [1, "a", 2.0] join "|" .
orig = list [1, 2]
big = copy[orig] process x gives x * 10
show orig, big .`
	out, err := run(t, src, "")
	if err != nil {
		t.Fatal(err)
	}
	want := "[ [ 1 ] ]\ntrue\ntrue\n1|a|2.0\n[ 1, 2 ][ 10, 20 ]\n"
	if out != want {
		t.Errorf("got %q, want %q", out, want)
	}
}

func TestSafeHandle(t *testing.T) {
	dir := moduleDir(t, map[string]string{
		"broken": "x = 1 / 0\n",
	})
	cases := []struct{ name, src, want, wantErr string }{
		{name: "no error skips the handle code", src: `safe
    x = 1
handle [] e .
    show "not reached" .
safe [end]
show e .`, want: "none\n"},
		{name: "error stops the block and runs the handle code", src: `safe
    x = 1 / 0
    show "not reached" .
handle [math] e .
    show e .
safe [end]
show "after" .`, want: "line 2: division by zero\nafter\n"},
		{name: "kind, line and message", src: `safe
    [read] no_such_file.txt to lines [end]
handle [file] e .
    show kind of e .
    show line of e .
    show message of e .
safe [end]`, want: "file\n2\n[read] no_such_file.txt: no such file or folder\n"},
		{name: "empty handle code", src: `safe
    x = 1 / 0
handle [] e .
safe [end]
show kind of e .`, want: "math\n"},
		{name: "empty list handles any kind", src: `safe
    y = nothing_here
handle [] e .
    show kind of e .
safe [end]`, want: "name\n"},
		{name: "several kinds", src: `safe
    n = change "abc" to integer
handle [math, number] e .
    show kind of e .
safe [end]`, want: "number\n"},
		{name: "index and key", src: `l = list [1]
safe
    v is l at get 5 .
handle [index] e .
    show kind of e .
safe [end]
m = map ["a":1]
safe
    v is m at get "b" .
handle [key] e .
    show kind of e .
safe [end]`, want: "index\nkey\n"},
		{name: "type", src: `safe
    x = "a" * 2
handle [type] e .
    show kind of e .
safe [end]`, want: "type\n"},
		{name: "unlisted kind still stops the program", src: `safe
    x = 1 / 0
handle [file] e .
    show "not reached" .
safe [end]
show "not reached" .`, wantErr: "line 2: division by zero"},
		{name: "error in the handle code is not handled by its own safe", src: `safe
    x = 1 / 0
handle [] e .
    y = missing
safe [end]`, wantErr: `line 4: undefined variable "missing"`},
		{name: "fail raises custom", src: `def withdraw[amount]
    if ] amount > 100 [
        fail "not enough money"
    if [end]
    return 100 - amount
def [end]
safe
    left = withdraw[500]
handle [custom] e .
    show e .
    show kind of e .
safe [end]`, want: "line 3: not enough money\ncustom\n"},
		{name: "unhandled fail stops the program", src: `fail "bad input " + 5`, wantErr: "line 1: bad input 5"},
		{name: "fail in the handle code passes it on", src: `safe
    safe
        x = 1 / 0
    handle [math] inner .
        fail "gave up: " + message of inner
    safe [end]
handle [custom] outer .
    show outer .
safe [end]`, want: "line 5: gave up: division by zero\n"},
		{name: "nested: inner passes it on", src: `safe
    safe
        x = 1 / 0
    handle [file] inner .
        show "not reached" .
    safe [end]
    show "not reached" .
handle [math] outer .
    show outer .
safe [end]`, want: "line 3: division by zero\n"},
		{name: "return inside safe and handle", src: `def f[n]
    safe
        return 10 / n
    handle [math] e .
        return -1
    safe [end]
def [end]
show f[2] .
show f[0] .`, want: "5\n-1\n"},
		{name: "break and continue inside safe", src: `[loop][i = 0; i < 6; i++]
    safe
        if ] i == 1 [
            continue
        if [end]
        if ] i == 4 [
            break
        if [end]
        k = i - 2
        x = 10 / k
        show i .
    handle [] e .
        show "skip " + i .
        continue
    safe [end]
[loop][end]`, want: "0\nskip 2\n3\n"},
		{name: "handled inside a deep call", src: `def down[n]
    if ] n == 0 [
        fail "bottom"
    if [end]
    return down[n - 1]
def [end]
safe
    down[50]
handle [] e .
    show message of e .
safe [end]`, want: "bottom\n"},
		{name: "error variable is local in a function", src: `def f[]
    safe
        x = 1 / 0
    handle [] problem .
        return kind of problem
    safe [end]
def [end]
show f[] .`, want: "math\n"},
		{name: "unknown error field", src: `safe
    x = 1 / 0
handle [] e .
    show code of e .
safe [end]`, wantErr: `an error has no field "code" (its fields: kind, file, line, message)`},
		{name: "failed import can be handled and tried again", src: `safe
    import broken
handle [math] e .
    show e .
safe [end]
safe
    import broken
handle [math] e .
    show e .
safe [end]`, want: "broken.t line 1: division by zero\nbroken.t line 1: division by zero\n"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			out, err := runIn(t, dir, c.src, "")
			if c.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), c.wantErr) {
					t.Fatalf("want error containing %q, got %v (output %q)", c.wantErr, err, out)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if out != c.want {
				t.Errorf("got %q, want %q", out, c.want)
			}
		})
	}
}

func TestSafeDoesNotHandleExit(t *testing.T) {
	_, err := run(t, `import system
safe
    exit[4]
handle [] e .
    show "not reached" .
safe [end]`, "")
	var ex ExitRequest
	if !errors.As(err, &ex) || ex.Code != 4 {
		t.Fatalf("want ExitRequest{4}, got %v", err)
	}
}

func TestStringInterpolation(t *testing.T) {
	cases := []struct{ name, src, want string }{
		{"names and expressions", `name = "Ann"
qty = 3
show "Hi {name}, {qty * 2} items" .`, "Hi Ann, 6 items\n"},
		{"escaped braces", `show "\{x\} {1 + 1}" .`, "{x} 2\n"},
		{"field, method and sentence call", `import strings
assemble P [x]
p = P[4]
l = list [1, 2]
w = "turtle"
show "{x of p} {l at len} {w substring 3}" .`, "4 2 tle\n"},
		{"values show like show does", `l = list [1, "a"]
show "{l} {none} {2.0}" .`, "[ 1, \"a\" ] none 2.0\n"},
		{"in a function result", `def greet[who]
    return "Hi {who}!"
def [end]
show greet["Bo"] .`, "Hi Bo!\n"},
		{"plain string unchanged", `show "no braces here }" .`, "no braces here }\n"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			out, err := run(t, c.src, "")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if out != c.want {
				t.Errorf("got %q, want %q", out, c.want)
			}
		})
	}
}

func TestInterpolationInPromptAndWrite(t *testing.T) {
	dir := t.TempDir()
	out, err := runIn(t, dir, `who = "Ann"
age = ? "Age for {who}? "
[write] out.txt
"{who} is {age}"
[end]
[read] out.txt to lines [end]
show lines .`, "7\n")
	if err != nil {
		t.Fatal(err)
	}
	if out != "Age for Ann? [ \"Ann is 7\" ]\n" {
		t.Errorf("got %q", out)
	}
}

func TestErrorsNameTheModuleFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "lib"), 0o755); err != nil {
		t.Fatal(err)
	}
	files := map[string]string{
		"lib/utils.t": "def half[n]\n    return 10 / n\ndef [end]\n",
		"lib/bad.t":   "x = 1\nshow x\n",
	}
	for name, src := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	cases := []struct{ name, src, want, wantErr string }{
		{name: "subfolder import, qualified by its last name", src: `import lib/utils
show half[4] .
show utils half[5] .`, want: "2\n2\n"},
		{name: "unhandled error in a module names it", src: `import lib/utils
x = half[0]`, wantErr: "lib/utils.t line 2: division by zero"},
		{name: "handled error in a module", src: `import lib/utils
safe
    x = half[0]
handle [math] e .
    show e .
    show file of e .
    show line of e .
safe [end]`, want: "lib/utils.t line 2: division by zero\nlib/utils.t\n2\n"},
		{name: "error in the main script", src: `safe
    x = nope
handle [] e .
    show e .
    show file of e .
safe [end]`, want: "line 2: undefined variable \"nope\"\nmain.t\n"},
		{name: "after a call, errors name the caller's line", src: `import lib/utils
x = 1
show half[2] / 0 .`, wantErr: "line 3: division by zero"},
		{name: "parse error in a module", src: `import lib/bad`, wantErr: "lib/bad.t line 2: expected next token to be ."},
		{name: "missing module", src: `import lib/nothing`, wantErr: "line 1: import lib/nothing: lib/nothing.t: no such file or folder"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			out, err := runScript(t, dir, c.src)
			if c.wantErr != "" {
				if err == nil || err.Error() != c.wantErr && !strings.HasPrefix(err.Error(), c.wantErr) {
					t.Fatalf("want error starting %q, got %v", c.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if out != c.want {
				t.Errorf("got %q, want %q", out, c.want)
			}
		})
	}
}

func TestEraseAndWarn(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "build", "deep"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, f := range []string{"old.txt", "build/a.txt", "build/deep/b.txt"} {
		if err := os.WriteFile(filepath.Join(dir, f), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	out, err := runIn(t, dir, `import system
erase["old.txt"]
erase["build"]
show exists["old.txt"], " ", exists["build"] .
warn "to stderr, not stdout" .
safe
    erase["old.txt"]
handle [file] e .
    show e .
safe [end]
safe
    erase["."]
handle [file] e .
    show e .
safe [end]`, "")
	if err != nil {
		t.Fatal(err)
	}
	want := "false false\n" +
		"line 7: erase old.txt: no such file or folder\n" +
		"line 12: erase .: won't erase the folder turtle is running in, or a folder above it\n"
	if out != want {
		t.Errorf("got %q, want %q", out, want)
	}
	for _, c := range []struct{ src, want string }{
		{`warn "x" .`, `"warn" needs "import system" first`},
		{"import system [args]\nerase[\"x\"]", `"erase" isn't imported`},
	} {
		if _, err := runIn(t, dir, c.src, ""); err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%q: want %q, got %v", c.src, c.want, err)
		}
	}
}

func TestReviewFixes(t *testing.T) {
	dir := t.TempDir()
	for _, sub := range []string{"a", "b", "lib"} {
		if err := os.MkdirAll(filepath.Join(dir, sub), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	files := map[string]string{
		"a/utils.t":  "def f[]\n    return 1\ndef [end]\n",
		"b/utils.t":  "def g[]\n    return 2\ndef [end]\n",
		"lib/math.t": "def h[]\n    return 3\ndef [end]\n",
		"lib/bad.t":  "x = 1\nshow x\n",
	}
	for name, src := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	cases := []struct{ name, src, want, wantErr string }{
		{name: "two imports with the same last name", src: "import a/utils\nimport b/utils",
			wantErr: `import b/utils: this file already imports another module called "utils"`},
		{name: "same module imported twice is fine", src: "import a/utils\nimport a/utils [f]\nshow f[] .", want: "1\n"},
		{name: "module named like a builtin", src: "import lib/math",
			wantErr: `import lib/math: "math" is the name of a builtin module`},
		{name: "parse error in a module is never handled", src: `safe
    import lib/bad
handle [] e .
    show "not reached" .
safe [end]`, wantErr: "lib/bad.t line 2: expected next token to be ."},
		{name: "an error's parts are read-only", src: `safe
    x = 1 / 0
handle [] e .
    kind of e = "file"
safe [end]`, wantErr: "an error's parts can't be changed (kind of an error is read-only)"},
		{name: "field change evaluates its target once", src: `assemble P [x]
calls = list []
p = P[1]
def get_p[]
    add 1 to calls .
    return p
def [end]
x of get_p[] = 5
show x of p, " ", length of calls .`, want: "5 1\n"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			out, err := runIn(t, dir, c.src, "")
			if c.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), c.wantErr) {
					t.Fatalf("want error containing %q, got %v (output %q)", c.wantErr, err, out)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if out != c.want {
				t.Errorf("got %q, want %q", out, c.want)
			}
		})
	}
}

func TestEraseGuardThroughSymlink(t *testing.T) {
	real := t.TempDir()
	work := filepath.Join(real, "work")
	if err := os.Mkdir(work, 0o755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(t.TempDir(), "link")
	if err := os.Symlink(real, link); err != nil {
		t.Skip("no symlinks:", err)
	}
	// work is <real>/work; the script names it through the link.
	_, err := runIn(t, work, "import system\nerase[\""+filepath.Join(link, "work")+"\"]", "")
	if err == nil || !strings.Contains(err.Error(), "won't erase the folder turtle is running in") {
		t.Fatalf("want the guard, got %v", err)
	}
	if _, err := os.Stat(work); err != nil {
		t.Fatalf("work folder is gone: %v", err)
	}
}

func TestPlusMixingCollectionsIsATypeError(t *testing.T) {
	for _, src := range []string{
		`x = "x" + list [1]`,
		`x = list [1] + "x"`,
		`x = list [1, 2] + 1`,
		`x = 1.5 + set [1]`,
		`x = map ["a": 1] + true`,
		`x = list [1] + set [1]`,
		`x = set [1] + map ["a": 1]`,
		`x = list [1] + none`,
	} {
		_, err := run(t, src, "")
		if err == nil || !strings.Contains(err.Error(), "a list, set or map only adds to another of its own kind") {
			t.Errorf("%s: want a type error, got %v", src, err)
		}
	}
	out, err := run(t, `safe
    x = list [1] + 1
handle [type] e .
    show kind of e .
safe [end]`, "")
	if err != nil || out != "type\n" {
		t.Fatalf("got %q, %v", out, err)
	}
}

func TestPlusWithNoneIsATypeError(t *testing.T) {
	for _, src := range []string{
		`x = "x=" + none`,
		`x = none + "x"`,
		`x = 5 + none`,
		`x = none + 2.5`,
		`x = true + none`,
	} {
		_, err := run(t, src, "")
		if err == nil || !strings.Contains(err.Error(), "none only adds to none; check for it first, e.g. if ] x != none [") {
			t.Errorf("%s: want a type error, got %v", src, err)
		}
	}
	_, err := run(t, `x = list [1] + 1`, "")
	if err == nil || !strings.Contains(err.Error(), "e.g. list [1] + list [2]") {
		t.Errorf("collection error should show an example, got %v", err)
	}
}

func TestChangeListAndSet(t *testing.T) {
	cases := []struct{ name, src, want, wantErr string }{
		{name: "list to set drops duplicates, keeps first order", src: `nums = list [3, 1, 3, 2, 1]
show change nums to set .
show nums .`, want: "{ 3, 1, 2 }\n[ 3, 1, 3, 2, 1 ]\n"},
		{name: "set to list", src: `s = set [5, 4, 5]
show change s to list .`, want: "[ 5, 4 ]\n"},
		{name: "statement form changes the variable", src: `nums = list [1, 1, 2]
change nums to set .
show nums .
change nums to list .
show nums .`, want: "{ 1, 2 }\n[ 1, 2 ]\n"},
		{name: "same kind is a copy", src: `a = list [1]
b = change a to list
add 2 to b .
s = set [1]
t = change s to set
add 2 to t .
show a, " ", b, " ", s, " ", t .`, want: "[ 1 ] [ 1, 2 ] { 1 } { 1, 2 }\n"},
		{name: "equal values count once", src: `show change list [1, 1.0, "1"] to set .`, want: "{ 1, \"1\" }\n"},
		{name: "empty", src: `show change list [] to set .`, want: "{  }\n"},
		{name: "number to list", src: `x = change 5 to list`, wantErr: "change: can't convert INTEGER to list"},
		{name: "map to set", src: `x = change map ["a": 1] to set`, wantErr: "change: can't convert MAP to set"},
		{name: "unknown target lists the types", src: `x = change 5 to tuple`, wantErr: "want integer, float, string, ascii, char, hex, list, or set"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			out, err := run(t, c.src, "")
			if c.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), c.wantErr) {
					t.Fatalf("want error containing %q, got %v", c.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if out != c.want {
				t.Errorf("got %q, want %q", out, c.want)
			}
		})
	}
}

func TestJSONLibrary(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "in.json"), []byte("{\n  \"users\": [\n    {\"name\": \"Bo\", \"age\": 7}\n  ],\n  \"bad\": \n}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "good.json"), []byte(`{"users": [{"name": "Bo", "age": 7}], "n": null}`), 0o644); err != nil {
		t.Fatal(err)
	}
	cases := []struct{ name, src, want, wantErr string }{
		{name: "load types", src: `import json
d = load['{"s": "x", "i": 3, "f": 2.5, "w": 2.0, "b": true, "n": null, "l": [1, "a"], "m": {}}']
show d .`, want: `{ "s": "x", "i": 3, "f": 2.5, "w": 2.0, "b": true, "n": none, "l": [ 1, "a" ], "m": {  } }` + "\n"},
		{name: "key order kept", src: `import json
show load['{"z": 1, "a": 2, "m": 3}'] at getKeys .`, want: `[ "z", "a", "m" ]` + "\n"},
		{name: "numbers", src: `import json
show load["1e3"], " ", load["-4"], " ", load["99999999999999999999"] .`, want: "1000.0 -4 100000000000000000000.0\n"},
		{name: "json_text one line", src: `import json
assemble P [x, y]
show json_text[map ["a": list [1, none], "s": set [2], "p": P[1, "<&>"], 3: true]] .`,
			want: `{"a":[1,null],"s":[2],"p":{"x":1,"y":"<&>"},"3":true}` + "\n"},
		{name: "escapes round trip", src: `import json
s = "line\n\"q\" \\ tab\t"
show load[json_text[s]] == s .`, want: "true\n"},
		{name: "json_get", src: `import json
d = json_read["good.json"]
show json_get[d, "users", 0, "name"] .
show json_get[d, "users", 3, "name"] .
show json_get[d, "users", -1] .
show json_get[d, "nope", "x"] .
show json_get[d, "n"] .
show json_get[d, "users", "0"] .`, want: "Bo\nnone\nnone\nnone\nnone\nnone\n"},
		{name: "json_get into an assembled value", src: `import json
assemble P [x]
show json_get[map ["p": P[5]], "p", "x"] .`, want: "5\n"},
		{name: "write then read", src: `import json
json_write["out.json", map ["a": list [1, 2], "b": map []]]
[read] out.json to lines [end]
show length of lines .
show json_read["out.json"] .`, want: "7\n" + `{ "a": [ 1, 2 ], "b": {  } }` + "\n"},
		{name: "bad JSON is kind json", src: `import json
safe
    d = load['{"a": 1,}']
handle [json] e .
    show kind of e .
    show message of e .
safe [end]`, want: "json\nload: invalid JSON at line 1, column 9: invalid character '}' looking for beginning of object key string\n"},
		{name: "bad JSON file names the file and line", src: `import json
d = json_read["in.json"]`, wantErr: "json_read in.json: invalid JSON at line 6, column 1: invalid character '}' looking for beginning of value"},
		{name: "empty text", src: `import json
d = load["  "]`, wantErr: "there's no JSON, the text is empty"},
		{name: "cut short", src: `import json
d = load['[1, 2']`, wantErr: "the text ended before the JSON did"},
		{name: "extra text", src: `import json
d = load["[1] 2"]`, wantErr: "extra text after the JSON value"},
		{name: "missing file is kind file", src: `import json
safe
    d = json_read["nope.json"]
handle [file] e .
    show e .
safe [end]`, want: "line 3: json_read nope.json: no such file or folder\n"},
		{name: "function can't be JSON", src: `import json
t = json_text[x gives x]`, wantErr: "json_text: a FUNCTION can't be written as JSON"},
		{name: "boolean key can't be JSON", src: `import json
t = json_text[map [true: 1]]`, wantErr: "a map key that's a BOOLEAN can't be a JSON key"},
		{name: "self-containing list", src: `import json
l = list [1]
add l to l .
t = json_text[l]`, wantErr: "the value contains itself"},
		{name: "same list twice is fine", src: `import json
l = list [1]
show json_text[list [l, l]] .`, want: "[[1],[1]]\n"},
		{name: "needs import", src: `d = load["1"]`, wantErr: `"load" needs "import json" first`},
		{name: "load needs text", src: `import json
d = load[5]`, wantErr: `"load" argument must be a string, got INTEGER`},
		{name: "json_get needs a key", src: `import json
d = json_get[map []]`, wantErr: "'json_get' expects a value and at least one key or index"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			out, err := runIn(t, dir, c.src, "")
			if c.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), c.wantErr) {
					t.Fatalf("want error containing %q, got %v (output %q)", c.wantErr, err, out)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if out != c.want {
				t.Errorf("got %q, want %q", out, c.want)
			}
		})
	}
}

// testdata/shop is a whole program (main script, two modules in lib/,
// JSON data in data/) that checks its own results; it runs in a copy so
// the report and log it writes and erases never touch testdata.
func TestShopProgram(t *testing.T) {
	src, err := filepath.Abs("../testdata/shop")
	if err != nil {
		t.Fatal(err)
	}
	work := t.TempDir()
	err = filepath.WalkDir(src, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, path)
		if d.IsDir() {
			return os.MkdirAll(filepath.Join(work, rel), 0o755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(work, rel), data, 0o644)
	})
	if err != nil {
		t.Fatal(err)
	}
	script, err := os.ReadFile(filepath.Join(work, "shop.t"))
	if err != nil {
		t.Fatal(err)
	}
	out, err := runFull(t, work, string(script), "Sam\n", []string{"--verbose"})
	if err != nil || !strings.Contains(out, "failures: 0") {
		t.Fatalf("shop.t failed (err %v):\n%s", err, out)
	}
	for _, f := range []string{"shop_report.json", "shop_log.txt"} {
		if _, err := os.Stat(filepath.Join(work, f)); err == nil {
			t.Errorf("%s wasn't erased", f)
		}
	}
}

func TestTimeLibrary(t *testing.T) {
	cases := []struct{ name, src, want, wantErr string }{
		{name: "make_date shows and has parts", src: `import time
d = make_date[2026, 1, 31, 9, 5, 7]
show d .
show year of d, " ", month of d, " ", day of d, " ", hour of d, " ", minute of d, " ", second of d, " ", weekday of d .`,
			want: "2026-01-31 09:05:07\n2026 1 31 9 5 7 Saturday\n"},
		{name: "add_time units", src: `import time
d = make_date[2026, 1, 31]
show add_time[d, 30, "days"] .
show add_time[d, -7, "days"] .
show add_time[d, 2, "weeks"] .
show add_time[d, 90, "minutes"] .
show add_time[d, 1, "hour"] .
show add_time[d, 45, "seconds"] .`,
			want: "2026-03-02 00:00:00\n2026-01-24 00:00:00\n2026-02-14 00:00:00\n2026-01-31 01:30:00\n2026-01-31 01:00:00\n2026-01-31 00:00:45\n"},
		{name: "months stay in the month", src: `import time
show add_time[make_date[2026, 1, 31], 1, "months"] .
show add_time[make_date[2028, 1, 31], 1, "months"] .
show add_time[make_date[2026, 3, 31], -1, "months"] .
show add_time[make_date[2024, 2, 29], 1, "years"] .
show add_time[make_date[2026, 11, 15], 3, "months"] .`,
			want: "2026-02-28 00:00:00\n2028-02-29 00:00:00\n2026-02-28 00:00:00\n2025-02-28 00:00:00\n2027-02-15 00:00:00\n"},
		{name: "time_between", src: `import time
a = make_date[2026, 1, 31]
b = add_time[a, 30, "days"]
show time_between[a, b, "days"], " ", time_between[b, a, "days"], " ", time_between[a, b, "weeks"], " ", time_between[a, b, "hours"] .
show time_between[a, make_date[2026, 2, 28], "months"], " ", time_between[make_date[2026, 1, 15], make_date[2026, 2, 14], "months"] .
show time_between[make_date[2000, 6, 15], make_date[2026, 6, 14], "years"], " ", time_between[make_date[2000, 6, 15], make_date[2026, 6, 15], "years"] .
show time_between[make_date[2026, 5, 1], make_date[2026, 1, 1], "months"] .`,
			want: "30 -30 4 720\n1 0\n25 26\n-4\n"},
		{name: "format_date", src: `import time
d = make_date[2026, 3, 5, 14, 7, 0]
show format_date[d, "Weekday, Month D YYYY at hh:mm"] .
show format_date[d, "Wkd DD/MM/YYYY ss"] .
show format_date[d, "Mon M"] .`,
			want: "Thursday, March 5 2026 at 14:07\nThu 05/03/2026 00\nMar 3\n"},
		{name: "to_date forms", src: `import time
show to_date["2026-10-03"] .
show to_date[" 2026-10-03 14:05 "] .
show to_date["2026-10-03 14:05:09"] .
show to_date["2026-10-03T14:05:09"] .
show to_date["2026-10-03T14:05:09Z"] .`,
			want: "2026-10-03 00:00:00\n2026-10-03 14:05:00\n2026-10-03 14:05:09\n2026-10-03 14:05:09\n2026-10-03 14:05:09\n"},
		{name: "compare, equal, map key, set", src: `import time
a = make_date[2026, 1, 1]
b = make_date[2026, 1, 2]
show a < b, " ", b >= a, " ", a == make_date[2026, 1, 1], " ", a != b .
m = map [a: "new year"]
show m at get[make_date[2026, 1, 1]] .
show length of set [a, b, make_date[2026, 1, 2]] .`,
			want: "true true true true\nnew year\n2\n"},
		{name: "today and today_utc are the same moment", src: `import time
t = today[]
u = today_utc[]
show t == u, " ", time_between[u, t, "seconds"] <= 1, " ", now[] > 0 .`, want: "true true true\n"},
		{name: "dates in text and JSON", src: `import time
import json
d = make_date[2026, 3, 2]
show "Due " + d .
show "Due {d}" .
show json_text[map ["due": d]] .
show change d to string .`,
			want: "Due 2026-03-02 00:00:00\nDue 2026-03-02 00:00:00\n" + `{"due":"2026-03-02 00:00:00"}` + "\n2026-03-02 00:00:00\n"},
		{name: "every until false", src: `import time
runs = list []
def job[]
    add 1 to runs .
    return length of runs < 3
def [end]
every[1, "seconds", job]
show length of runs .`, want: "3\n"},
		{name: "wait_until a past date returns at once", src: `import time
before = now[]
wait_until[add_time[today[], -1, "days"]]
show now[] - before < 500 .`, want: "true\n"},
		{name: "impossible date", src: `import time
d = make_date[2026, 2, 30]`, wantErr: "make_date: 2026-02-30 00:00:00 isn't a real date and time"},
		{name: "bad date text is kind date", src: `import time
safe
    d = to_date["next tuesday"]
handle [date] e .
    show kind of e .
safe [end]`, want: "date\n"},
		{name: "bad unit", src: `import time
d = add_time[today[], 1, "fortnights"]`, wantErr: "'add_time' unit must be one of seconds, minutes, hours, days, weeks, months, years, got \"fortnights\""},
		{name: "amount must be whole", src: `import time
d = add_time[today[], 1.5, "days"]`, wantErr: "'add_time' amount must be a whole number, got FLOAT"},
		{name: "needs a date", src: `import time
d = add_time["2026-01-01", 1, "days"]`, wantErr: "'add_time' needs a date (from today[], make_date or to_date), got STRING"},
		{name: "unknown part", src: `import time
show week of today[] .`, wantErr: `a date has no part "week" (its parts: year, month, day, hour, minute, second, weekday)`},
		{name: "parts are read-only", src: `import time
d = today[]
day of d = 3`, wantErr: "a date's parts can't be changed"},
		{name: "every needs a no-parameter function", src: `import time
every[1, "days", x gives x]`, wantErr: "'every' runs a function with no parameters, but this one takes 1"},
		{name: "every needs a positive amount", src: `import time
every[0, "days", [] gives false]`, wantErr: "'every' amount must be at least 1"},
		{name: "compare a date with text", src: `import time
x = today[] < "2026"`, wantErr: "needs two numbers, two strings or two dates, got DATE and STRING"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			out, err := run(t, c.src, "")
			if c.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), c.wantErr) {
					t.Fatalf("want error containing %q, got %v (output %q)", c.wantErr, err, out)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if out != c.want {
				t.Errorf("got %q, want %q", out, c.want)
			}
		})
	}
}

// Days keep the clock time across a daylight-saving change (a 23-hour
// day still counts as one day); hours are exact.
func TestTimeAcrossDaylightSaving(t *testing.T) {
	ny, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Skip("no time zone data:", err)
	}
	orig := time.Local
	time.Local = ny
	defer func() { time.Local = orig }()
	out, err := run(t, `import time
d = make_date[2026, 3, 7, 12, 0, 0]
next = add_time[d, 1, "days"]
show next .
show time_between[d, next, "days"], " ", time_between[d, next, "hours"] .
show add_time[d, 24, "hours"] .`, "")
	if err != nil {
		t.Fatal(err)
	}
	want := "2026-03-08 12:00:00\n1 23\n2026-03-08 13:00:00\n"
	if out != want {
		t.Errorf("got %q, want %q", out, want)
	}
}

func TestHTTPLibrary(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/hello":
			fmt.Fprint(w, "hello turtle")
		case "/users":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `[{"name": "Ann", "age": 30}, {"name": "Bo", "age": 7}]`)
		case "/echo":
			body, _ := io.ReadAll(r.Body)
			w.Header().Set("X-Seen-Method", r.Method)
			w.WriteHeader(201)
			fmt.Fprintf(w, "%s|%s|%s|%s", r.Method, r.Header.Get("Content-Type"), r.Header.Get("Authorization"), body)
		case "/missing":
			http.Error(w, "no such page", 404)
		case "/boom":
			w.WriteHeader(500)
		}
	}))
	defer srv.Close()
	base := "base = \"" + srv.URL + "\"\n"
	// A port that was just free and is now closed refuses connections on
	// every system; a fixed port like 1 could be filtered and time out.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	closed := "http://" + ln.Addr().String() + "/"
	ln.Close()
	cases := []struct{ name, src, want, wantErr string }{
		{name: "get text", src: `import http
show http_get[base + "/hello"] .`, want: "hello turtle\n"},
		{name: "get JSON into a list of maps", src: `import http
import json
users = load[http_get[base + "/users"]]
show json_get[users, 1, "name"], " ", length of users .`, want: "Bo 2\n"},
		{name: "post text", src: `import http
show http_post[base + "/echo", "hi there"] .`, want: "POST|text/plain; charset=utf-8||hi there\n"},
		{name: "post a map as JSON", src: `import http
show http_post[base + "/echo", map ["name": "Ann", "tags": list ["a"]]] .`, want: `POST|application/json||{"name":"Ann","tags":["a"]}` + "\n"},
		{name: "request with headers, status and response headers", src: `import http
r = http_request["put", base + "/echo", "x", map ["Authorization": "Bearer t0k"]]
show r at get["status"] .
show r at get["body"] .
h = r at get["headers"]
show h at get["x-seen-method"] .`, want: "201\nPUT|text/plain; charset=utf-8|Bearer t0k|x\nPUT\n"},
		{name: "request hands back a 404", src: `import http
r = http_request["GET", base + "/missing"]
show r at get["status"] .`, want: "404\n"},
		{name: "get fails on 404 with kind http", src: `import http
safe
    x = http_get[base + "/missing"]
handle [http] e .
    show kind of e .
    show message of e .
safe [end]`, want: "http\nhttp_get " + srv.URL + "/missing: 404 Not Found: no such page\n"},
		{name: "500 without a body", src: `import http
x = http_post[base + "/boom", "x"]`, wantErr: "http_post " + srv.URL + "/boom: 500 Internal Server Error"},
		{name: "connection refused", src: `import http
x = http_get["` + closed + `"]`, wantErr: "http_get " + closed + ": connection refused (is the server running?)"},
		{name: "not a web address", src: `import http
x = http_get["ftp://example.com"]`, wantErr: `http_get: "ftp://example.com" isn't a web address`},
		{name: "headers must be a map", src: `import http
x = http_request["GET", base + "/hello", none, list []]`, wantErr: "'http_request' headers must be a map"},
		{name: "body can't be a number", src: `import http
x = http_post[base + "/echo", 5]`, wantErr: "'http_post' body must be text, or a map or list to send as JSON, got INTEGER"},
		{name: "needs import", src: `x = http_get[base]`, wantErr: `"http_get" needs "import http" first`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			out, err := run(t, base+c.src, "")
			if c.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), c.wantErr) {
					t.Fatalf("want error containing %q, got %v (output %q)", c.wantErr, err, out)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if out != c.want {
				t.Errorf("got %q, want %q", out, c.want)
			}
		})
	}
}

func TestSQLLibrary(t *testing.T) {
	dir, err := filepath.Abs("../testdata")
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct{ name, src, want, wantErr string }{
		{name: "open, query, rows are maps", src: `import sql
db = sql_open["books.db"]
rows = sql_query[db, "SELECT title, price FROM books WHERE price < 1000 ORDER BY price"]
show rows .
show db .`, want: `[ { "title": "Café 🐢", "price": 1 }, { "title": "Foundation", "price": 800 }, { "title": "Dune", "price": 950 } ]` + "\ndatabase books.db\n"},
		{name: "placeholders", src: `import sql
db = sql_open["sqlite:books.db"]
rows = sql_query[db, "SELECT sku FROM books WHERE price > ? AND title LIKE ?", list [900, "%o%"]]
show rows .`, want: `[ { "sku": "B2" }, { "sku": "B3" } ]` + "\n"},
		{name: "types: null, real, integer primary key", src: `import sql
import json
db = sql_open["books.db"]
r = sql_query[db, "SELECT id, rating, added FROM books WHERE sku = ?", list ["B2"]]
show json_get[r, 0, "id"], " ", json_get[r, 0, "rating"], " ", json_get[r, 0, "added"] .
r = sql_query[db, "SELECT rating FROM books WHERE sku = 'B5'"]
show json_get[r, 0, "rating"] .`, want: "2 none 2026-02-01\n5.0\n"},
		{name: "aggregates", src: `import sql
db = sql_open["books.db"]
show sql_query[db, "SELECT count(*) AS n, sum(qty) AS total FROM orders WHERE customer = ?", list ["bo"]] .`,
			want: `[ { "n": 3, "total": 6 } ]` + "\n"},
		{name: "tables", src: `import sql
db = sql_open["books.db"]
show sql_tables[db] .`, want: `[ "books", "orders" ]` + "\n"},
		{name: "parameter kinds", src: `import sql
import time
db = sql_open["books.db"]
show sql_query[db, "SELECT ? AS a, ? AS b, ? AS c, ? AS d, ? AS e", list [true, none, 2.5, "x", make_date[2026, 1, 2]]] .`,
			want: `[ { "a": 1, "b": none, "c": 2.5, "d": "x", "e": "2026-01-02 00:00:00" } ]` + "\n"},
		{name: "loop over rows", src: `import sql
db = sql_open["books.db"]
[loop][row in sql_query[db, "SELECT title FROM books WHERE rating > 4 ORDER BY rating DESC"]]
    show row at get["title"] .
[loop][end]
sql_close[db]
sql_close[db]`, want: "Café 🐢\nDune\nFoundation\n"},
		{name: "bad query is kind sql", src: `import sql
db = sql_open["books.db"]
safe
    r = sql_query[db, "SELECT nope FROM books"]
handle [sql] e .
    show kind of e .
    show message of e .
safe [end]`, want: "sql\nsql_query: SQL: no such column: nope\n"},
		{name: "no such table", src: `import sql
db = sql_open["books.db"]
r = sql_query[db, "SELECT * FROM shelves"]`, wantErr: "sql_query: no such table: shelves"},
		{name: "wrong number of values", src: `import sql
db = sql_open["books.db"]
r = sql_query[db, "SELECT * FROM books WHERE price < ?"]`, wantErr: "1 ? placeholder(s) but 0 value(s)"},
		{name: "changes go through sql_run", src: `import sql
db = sql_open["books.db"]
r = sql_query[db, "DELETE FROM books"]`, wantErr: "DELETE changes the database; run it with sql_run, or add RETURNING to get rows back"},
		{name: "missing file is kind file", src: `import sql
safe
    db = sql_open["nope.db"]
handle [file] e .
    show e .
safe [end]`, want: "line 3: sql_open nope.db: no such file (sql_create makes a new database)\n"},
		{name: "not a database", src: `import sql
db = sql_open["everything.t"]`, wantErr: "sql_open everything.t: everything.t isn't a SQLite database"},
		{name: "no postgres server", src: `import sql
db = sql_open["postgres://ann:secret@127.0.0.1:1/shop?connect_timeout=2"]`, wantErr: "is the server running"},
		{name: "no mysql server", src: `import sql
db = sql_open["mysql://ann:secret@127.0.0.1:1/shop?timeout=2"]`, wantErr: "is the server running"},
		{name: "sql_create on a server", src: `import sql
db = sql_create["mysql://ann@localhost/shop"]`, wantErr: "made on the server"},
		{name: "closed database", src: `import sql
db = sql_open["books.db"]
sql_close[db]
r = sql_query[db, "SELECT 1"]`, wantErr: "sql_query: database books.db is closed"},
		{name: "values must be a list", src: `import sql
db = sql_open["books.db"]
r = sql_query[db, "SELECT ?", 5]`, wantErr: "values for the ? placeholders must be a list"},
		{name: "unsupported value", src: `import sql
db = sql_open["books.db"]
r = sql_query[db, "SELECT ?", list [list [1]]]`, wantErr: "value 1 for a ? placeholder can't be a LIST"},
		{name: "needs a database", src: `import sql
r = sql_query["books.db", "SELECT 1"]`, wantErr: "'sql_query' needs a database (from sql_open), got STRING"},
		{name: "needs import", src: `db = sql_open["books.db"]`, wantErr: `"sql_open" needs "import sql" first`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			out, err := runIn(t, dir, c.src, "")
			if c.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), c.wantErr) {
					t.Fatalf("want error containing %q, got %v (output %q)", c.wantErr, err, out)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if out != c.want {
				t.Errorf("got %q, want %q", out, c.want)
			}
		})
	}
}

func TestDataTable(t *testing.T) {
	cases := []struct{ name, src, want, wantErr string }{
		{name: "list of maps", src: `import data
rows = list [map ["name": "Ann", "age": 30], map ["name": "Bo", "age": 7]]
show table[rows] .`, want: "name  age\n----  ---\nAnn    30\nBo      7\n"},
		{name: "list of assembled values", src: `import data
assemble Order [item, qty, price]
show table[list [Order["pen", 3, 1.5], Order["mug", 12, 8.0]]] .`,
			want: "item  qty  price\n----  ---  -----\npen     3    1.5\nmug    12    8.0\n"},
		{name: "missing key is blank, none is shown", src: `import data
show table[list [map ["a": 1], map ["b": none, "a": 22]]] .`,
			want: " a  b\n--  ----\n 1\n22  none\n"},
		{name: "plain list is numbered from 0", src: `import data
show table[list ["Ann", "Bo"]] .`, want: "#  value\n-  -----\n0  Ann\n1  Bo\n"},
		{name: "list of lists", src: `import data
show table[list [list [1, "x"], list [2, "y", true]]] .`,
			want: "0  1  2\n-  -  ----\n1  x\n2  y  true\n"},
		{name: "one map", src: `import data
show table[map ["Ann": 30]] .`, want: "key  value\n---  -----\nAnn     30\n"},
		{name: "one assembled value", src: `import data
assemble P [x, y]
show table[P[1, "b"]] .`, want: "field  value\n-----  -----\nx      1\ny      b\n"},
		{name: "empty", src: `import data
show table[list []] .`, want: "no rows\n"},
		{name: "line breaks stay in one cell", src: `import data
show table[list ["a\nb"]] .`, want: "#  value\n-  -----\n0  a\\nb\n"},
		{name: "wide characters line up", src: `import data
show table[list [map ["t": "🐢", "n": 1], map ["t": "abc", "n": 2]]] .`,
			want: "t    n\n---  -\n🐢   1\nabc  2\n"},
		{name: "tablerows defaults to 20", src: `import data
show tablerows .
nums = list []
[loop][i = 0; i < 25; i++]
    add i to nums .
[loop][end]
t = table[nums]
lines is t at split "\n" .
show length of lines .
show lines at get[22] .`, want: "20\n23\n... 5 more rows\n"},
		{name: "tablerows can be changed", src: `import data
tablerows = 1
show table[list ["a", "b"]] .`, want: "#  value\n-  -----\n0  a\n... 1 more row\n"},
		{name: "tablerows none shows every row", src: `import data
tablerows = none
show table[list ["a", "b"]] .`, want: "#  value\n-  -----\n0  a\n1  b\n"},
		{name: "second argument overrides tablerows", src: `import data
show table[list ["a", "b", "c"], 2] .`, want: "#  value\n-  -----\n0  a\n1  b\n... 1 more row\n"},
		{name: "a function's local tablerows", src: `import data
def peek[xs]
    tablerows = 0
    return table[xs]
def [end]
show peek[list [1, 2]] .
show tablerows .`, want: "#  value\n-  -----\n... 2 more rows\n20\n"},
		{name: "a variable already named tablerows is kept", src: `tablerows = 1
import data
show tablerows .`, want: "1\n"},
		{name: "bad tablerows", src: `import data
tablerows = "ten"
show table[list [1]] .`, wantErr: "tablerows must be a whole number of 0 or more"},
		{name: "bad argument", src: `import data
show table[5] .`, wantErr: "'table' needs a list, set, map, or assembled value, got INTEGER"},
		{name: "needs import", src: `show table[list [1]] .`, wantErr: `"table" needs "import data" first`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			out, err := run(t, c.src, "")
			if c.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), c.wantErr) {
					t.Fatalf("want error containing %q, got %v", c.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if out != c.want {
				t.Errorf("got %q, want %q", out, c.want)
			}
		})
	}
}

func TestSQLWriting(t *testing.T) {
	cases := []struct{ name, src, want, wantErr string }{
		{name: "create, run, query", src: `import sql
import data
db = sql_create["shop.db"]
sql_run[db, "CREATE TABLE books (id INTEGER PRIMARY KEY, title TEXT NOT NULL, price INTEGER)"]
n = sql_run[db, "INSERT INTO books (title, price) VALUES (?, ?), (?, ?)", list ["Dune", 950, "Emma", 700]]
show n .
show sql_run[db, "UPDATE books SET price = price + 50 WHERE price < ?", list [900]] .
show table[sql_query[db, "SELECT * FROM books ORDER BY id"]] .
show sql_tables[db] .`, want: "2\n1\nid  title  price\n--  -----  -----\n 1  Dune     950\n 2  Emma     750\n[ \"books\" ]\n"},
		{name: "returning", src: `import sql
db = sql_create["r.db"]
sql_run[db, "CREATE TABLE t (id INTEGER PRIMARY KEY, v TEXT)"]
rows = sql_query[db, "INSERT INTO t (v) VALUES ('a'), ('b') RETURNING id, v"]
show rows .`, want: "[ { \"id\": 1, \"v\": \"a\" }, { \"id\": 2, \"v\": \"b\" } ]\n"},
		{name: "constraint errors are kind sql", src: `import sql
db = sql_create["c.db"]
sql_run[db, "CREATE TABLE t (v TEXT UNIQUE NOT NULL)"]
sql_run[db, "INSERT INTO t VALUES ('a')"]
safe
    sql_run[db, "INSERT INTO t VALUES ('a')"]
handle [sql] e .
    show message of e .
safe [end]
safe
    sql_run[db, "INSERT INTO t VALUES (?)", list [none]]
handle [sql] e .
    show message of e .
safe [end]`, want: "sql_run: UNIQUE constraint failed: t.v\nsql_run: NOT NULL constraint failed: t.v\n"},
		{name: "transactions", src: `import sql
db = sql_create["tx.db"]
sql_run[db, "CREATE TABLE t (v)"]
sql_run[db, "BEGIN"]
sql_run[db, "INSERT INTO t VALUES (1)"]
sql_run[db, "ROLLBACK"]
sql_run[db, "BEGIN"]
sql_run[db, "INSERT INTO t VALUES (2)"]
sql_run[db, "COMMIT"]
show sql_query[db, "SELECT v FROM t"] .`, want: "[ { \"v\": 2 } ]\n"},
		{name: "same column twice", src: `import sql
db = sql_create["j.db"]
sql_run[db, "CREATE TABLE a (id INTEGER PRIMARY KEY, n TEXT); CREATE TABLE b (id INTEGER PRIMARY KEY, a_id INTEGER)"]
sql_run[db, "INSERT INTO a VALUES (1, 'x'); INSERT INTO b VALUES (7, 1)"]
show sql_query[db, "SELECT * FROM a JOIN b ON b.a_id = a.id"] .`, want: "[ { \"id\": 1, \"n\": \"x\", \"id:1\": 7, \"a_id\": 1 } ]\n"},
		{name: "create needs a new file", src: `import sql
db = sql_create["books.db"]`, wantErr: "sql_create books.db: the file already exists (sql_open opens it)"},
		{name: "sql_run needs a list", src: `import sql
db = sql_create["l.db"]
sql_run[db, "CREATE TABLE t (v)", 5]`, wantErr: "values for the ? placeholders must be a list"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			dir := t.TempDir()
			src, _ := os.ReadFile("../testdata/books.db")
			os.WriteFile(filepath.Join(dir, "books.db"), src, 0o644)
			out, err := runIn(t, dir, c.src, "")
			if c.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), c.wantErr) {
					t.Fatalf("want error containing %q, got %v (output %q)", c.wantErr, err, out)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v (output %q)", err, out)
			}
			if out != c.want {
				t.Errorf("got %q, want %q", out, c.want)
			}
		})
	}
}

func TestSQLExamples(t *testing.T) {
	src, err := filepath.Abs("../testdata/sql")
	if err != nil {
		t.Fatal(err)
	}
	work := t.TempDir()
	if err := os.CopyFS(work, os.DirFS(src)); err != nil {
		t.Fatal(err)
	}
	programs, _ := filepath.Glob(filepath.Join(work, "*.t"))
	if len(programs) < 11 {
		t.Fatalf("found %d programs in testdata/sql", len(programs))
	}
	before, _ := os.ReadDir(work)
	for _, prog := range programs {
		t.Run(filepath.Base(prog), func(t *testing.T) {
			code, err := os.ReadFile(prog)
			if err != nil {
				t.Fatal(err)
			}
			scriptName = filepath.Base(prog)
			defer func() { scriptName = "" }()
			out, runErr := runFull(t, work, string(code), "", nil)
			if runErr != nil || !strings.Contains(out, "failures: 0") {
				t.Fatalf("failed (err %v):\n%s", runErr, out)
			}
			after, _ := os.ReadDir(work)
			if len(after) != len(before) {
				var names []string
				for _, e := range after {
					names = append(names, e.Name())
				}
				t.Fatalf("left files behind: %v", names)
			}
		})
	}
}

func TestJSONTableFileErrors(t *testing.T) {
	dir := t.TempDir()
	cases := map[string]string{
		`{"a": 1}`:        "holds a list of objects, [ {...}, {...} ], not an object",
		`[{"a": 1}, 5]`:   "row 2 is a number, not an object",
		`[{"a": 1,}]`:     "invalid JSON at line 1",
		`[{"a": [1, 2]}]`: "",
	}
	for text, want := range cases {
		os.WriteFile(filepath.Join(dir, "t.json"), []byte(text), 0o644)
		_, err := runFull(t, dir, "import data\nrows = table_read[\"t.json\"]", "", nil)
		if want == "" && err != nil || want != "" && (err == nil || !strings.Contains(err.Error(), want)) {
			t.Errorf("%s: want %q, got %v", text, want, err)
		}
	}
}

func TestTableFiles(t *testing.T) {
	dir := t.TempDir()
	write := func(name, text string) {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("books.csv", "\ufeffsku,title,price\r\nB1,Dune,950\r\nB2,\"Gone, Girl\",\r\nB3,\"Say \"\"hi\"\"\nthere\",5\r\n\r\n")
	write("books.tsv", "sku\ttitle\nB1\tDune, again\n")
	write("short.csv", "a,b,c\n1,2\n")
	write("long.csv", "a,b\n1,2,3\n")
	write("bad.csv", "a,b\n\"open,2\n")
	write("twice.csv", "a,a\n1,2\n")
	write("empty.csv", "")
	cases := []struct{ name, src, want, wantErr string }{
		{name: "read rows as maps of text", src: `import data
rows = table_read["books.csv"]
show length of rows .
show rows at get[0] .
show rows at get[1] .
show rows at get[2] at get["title"] .`, want: "3\n{ \"sku\": \"B1\", \"title\": \"Dune\", \"price\": \"950\" }\n{ \"sku\": \"B2\", \"title\": \"Gone, Girl\", \"price\": none }\nSay \"hi\"\nthere\n"},
		{name: "tsv", src: `import data
show table_read["books.tsv"] .`, want: "[ { \"sku\": \"B1\", \"title\": \"Dune, again\" } ]\n"},
		{name: "short rows get none", src: `import data
show table_read["short.csv"] .`, want: "[ { \"a\": \"1\", \"b\": \"2\", \"c\": none } ]\n"},
		{name: "empty file", src: `import data
show table_read["empty.csv"] .`, want: "[  ]\n"},
		{name: "round trip csv and tsv", src: `import data
rows = table_read["books.csv"]
show table_write["copy.csv", rows] .
show table_read["copy.csv"] == rows .
table_write["copy.tsv", rows]
show table_read["copy.tsv"] == rows .`, want: "3\ntrue\ntrue\n"},
		{name: "write any shape", src: `import data
assemble P [name, age]
table_write["p.csv", list [P["Ann", 30], P["Bo", none]]]
[read] p.csv to lines [end]
show lines .
table_write["l.csv", list ["x", "y"]]
[read] l.csv to lines [end]
show lines .
table_write["m.csv", map ["a": 1]]
[read] m.csv to lines [end]
show lines .`, want: "[ \"name,age\", \"Ann,30\", \"Bo,\" ]\n[ \"#,value\", \"0,x\", \"1,y\" ]\n[ \"key,value\", \"a,1\" ]\n"},
		{name: "txt is the shown table, every row", src: `import data
tablerows = 1
table_write["t.txt", list [map ["n": 1], map ["n": 22]]]
[read] t.txt to lines [end]
show lines .`, want: "[ \" n\", \"--\", \" 1\", \"22\" ]\n"},
		{name: "txt can't be read", src: `import data
r = table_read["t.txt"]`, wantErr: "reads .csv, .tsv and .json files"},
		{name: "too many values", src: `import data
r = table_read["long.csv"]`, wantErr: "table_read long.csv: line 2 has 3 values but the header has 2 names"},
		{name: "bad quotes are kind csv", src: `import data
safe
    r = table_read["bad.csv"]
handle [csv] e .
    show kind of e .
safe [end]`, want: "csv\n"},
		{name: "header twice", src: `import data
r = table_read["twice.csv"]`, wantErr: "the header names \"a\" twice"},
		{name: "missing file", src: `import data
r = table_read["nope.csv"]`, wantErr: "table_read nope.csv"},
		{name: "write needs a collection", src: `import data
table_write["x.csv", 5]`, wantErr: "'table_write' needs a list, set, map, or assembled value, got INTEGER"},
		{name: "write to a missing folder", src: `import data
table_write["no/such/x.csv", list [1]]`, wantErr: "table_write no/such/x.csv"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			out, err := runIn(t, dir, c.src, "")
			if c.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), c.wantErr) {
					t.Fatalf("want error containing %q, got %v (output %q)", c.wantErr, err, out)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v (output %q)", err, out)
			}
			if out != c.want {
				t.Errorf("got %q, want %q", out, c.want)
			}
		})
	}
}

func TestSQLFiles(t *testing.T) {
	setup := `import sql
import data
db = sql_create["s.db"]
sql_run[db, "CREATE TABLE books (sku TEXT PRIMARY KEY, title TEXT NOT NULL, price INTEGER CHECK (price > 0), stock INTEGER DEFAULT 0)"]
sql_load[db, "books", "books.csv"]
`
	files := map[string]string{
		"books.csv":   "sku,title,price\nB1,Dune,950\nB2,\"Gone, Girl\",1225\nB3,Emma,700\n",
		"prices.csv":  "sku,price\nB1,999\nB9,5\n",
		"gone.csv":    "sku,why\nB2,old\n",
		"restock.csv": "sku,title,stock\nB1,Dune,4\nB4,Beloved,2\n",
		"bad.csv":     "sku,price\nB1,10\nB3,0\n",
		"nokey.csv":   "title\nX\n",
		"dup.csv":     "sku,title,price\nB7,New,5\nB1,Dup,5\n",
		"more.tsv":    "sku\ttitle\tprice\nB5\tTabbed, yes\t10\n",
	}
	cases := []struct{ name, src, want, wantErr string }{
		{name: "load and save", src: setup + `show sql_save[db, "SELECT sku, title, price, typeof(price) AS t FROM books ORDER BY sku", "out.csv"] .
[read] out.csv to lines [end]
show lines .`, want: "3\n[ \"sku,title,price,t\", \"B1,Dune,950,integer\", \"B2,\\\"Gone, Girl\\\",1225,integer\", \"B3,Emma,700,integer\" ]\n"},
		{name: "save with values, and no rows still has a header", src: setup + `show sql_save[db, "SELECT sku FROM books WHERE price > ?", "none.csv", list [99999]] .
[read] none.csv to lines [end]
show lines .`, want: "0\n[ \"sku\" ]\n"},
		{name: "save as txt", src: setup + `sql_save[db, "SELECT sku, price FROM books ORDER BY sku", "out.txt"]
[read] out.txt to lines [end]
show lines at get[2] .`, want: "B1     950\n"},
		{name: "load tsv", src: setup + `show sql_load[db, "books", "more.tsv"] .
show sql_query[db, "SELECT title FROM books WHERE sku = 'B5'"] .`, want: "1\n[ { \"title\": \"Tabbed, yes\" } ]\n"},
		{name: "update by key", src: setup + `show sql_update[db, "books", "sku", "prices.csv"] .
show sql_query[db, "SELECT price FROM books WHERE sku = 'B1'"] .`, want: "1\n[ { \"price\": 999 } ]\n"},
		{name: "delete by key, other columns ignored", src: setup + `show sql_delete[db, "books", "sku", "gone.csv"] .
show sql_query[db, "SELECT count(*) AS n FROM books"] .`, want: "1\n[ { \"n\": 2 } ]\n"},
		{name: "upsert", src: setup + `show sql_upsert[db, "books", "sku", "restock.csv"] .
show sql_query[db, "SELECT sku, title, price, stock FROM books WHERE sku IN ('B1', 'B4') ORDER BY sku"] .`, want: "2\n[ { \"sku\": \"B1\", \"title\": \"Dune\", \"price\": 950, \"stock\": 4 }, { \"sku\": \"B4\", \"title\": \"Beloved\", \"price\": none, \"stock\": 2 } ]\n"},
		{name: "a bad line undoes the whole file", src: setup + `safe
    sql_update[db, "books", "sku", "bad.csv"]
handle [sql] e .
    show message of e .
safe [end]
show sql_query[db, "SELECT price FROM books WHERE sku = 'B1'"] .`, want: "sql_update: CHECK constraint failed: price > 0\n[ { \"price\": 950 } ]\n"},
		{name: "a duplicate undoes the whole load", src: setup + `safe
    sql_load[db, "books", "dup.csv"]
handle [sql] e .
    show message of e .
safe [end]
show sql_query[db, "SELECT count(*) AS n FROM books"] .`, want: "sql_load: UNIQUE constraint failed: books.sku\n[ { \"n\": 3 } ]\n"},
		{name: "delete from a tsv file", src: setup + `show sql_delete[db, "books", "sku", "more.tsv"] .`, want: "0\n"},
		{name: "key column must be in the file", src: setup + `sql_update[db, "books", "sku", "nokey.csv"]`, wantErr: "sql_update nokey.csv: the file has no \"sku\" column (its columns: title)"},
		{name: "unknown table", src: setup + `sql_load[db, "nope", "books.csv"]`, wantErr: "sql_load: no such table: nope"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			dir := t.TempDir()
			for name, text := range files {
				os.WriteFile(filepath.Join(dir, name), []byte(text), 0o644)
			}
			out, err := runIn(t, dir, c.src, "")
			if c.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), c.wantErr) {
					t.Fatalf("want error containing %q, got %v (output %q)", c.wantErr, err, out)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v (output %q)", err, out)
			}
			if out != c.want {
				t.Errorf("got %q, want %q", out, c.want)
			}
		})
	}
}

// Every builtin function and method has an entry in stdlibdocs.go, and
// every entry names a real one.
func TestEveryBuiltinIsDocumented(t *testing.T) {
	documented := map[string]string{}
	for _, e := range allEntries() {
		documented[e.module+"."+e.name()] = e.call
	}
	real := map[string]bool{}
	for name, mod := range builtinModules {
		for _, f := range append(append([]string{}, mod.Funcs...), mod.Methods...) {
			real[name+"."+f] = true
			if _, ok := documented[name+"."+f]; !ok {
				t.Errorf("%s isn't documented in stdlibdocs.go", name+"."+f)
			}
		}
	}
	for k := range documented {
		if !real[k] {
			t.Errorf("stdlibdocs.go documents %s, which doesn't exist", k)
		}
	}
}

func TestDoc(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "tools.t"), []byte("// tools.t: helpers.\n\n// add_tax adds rate percent.\n// cents is an integer.\ndef add_tax[cents, rate]\n    return cents\ndef [end]\n\ndef bare[x]\ndef [end]\n\n// Item is one thing on a shelf.\nassemble Item [sku, title]\n"), 0o644)
	cases := map[string]string{
		"":         "sql: Databases: SQLite files",
		"sql":      "sql_upsert[db, table, key, path]",
		"sql_load": "sql_load[db, table, path]        (import sql)\n  Adds a record to a table",
		"sqrt":     "number at sqrt        (import math)",
		"warn":     "warn ... .",
		"tools.t":  "tools.t: helpers.\n\nadd_tax[cents, rate]\n  add_tax adds rate percent.\n  cents is an integer.\n\nbare[x]\n  (no description",
		"tools":    "assemble Item [sku, title]\n  Item is one thing on a shelf.",
	}
	for topic, want := range cases {
		got, err := Doc(topic, dir)
		if err != nil || !strings.Contains(got, want) {
			t.Errorf("doc %q: want %q in:\n%s (err %v)", topic, want, got, err)
		}
	}
	if _, err := Doc("sql_lod", dir); err == nil || !strings.Contains(err.Error(), "did you mean sql_load") {
		t.Errorf("unknown topic: %v", err)
	}
}

// A program that ends without sql_close: its databases are closed for
// it, an unfinished transaction rolled back, and the files let go.
func TestDatabasesClosedAtEnd(t *testing.T) {
	dir := t.TempDir()
	src := `import sql
db = sql_create["end.db"]
sql_run[db, "CREATE TABLE t (v INTEGER)"]
sql_run[db, "BEGIN"]
sql_run[db, "INSERT INTO t VALUES (1)"]`
	if _, err := runFull(t, dir, src, "", nil); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "end.db-journal")); !os.IsNotExist(err) {
		t.Errorf("the journal is still there: %v", err)
	}
	db, err := sqlite.Open(filepath.Join(dir, "end.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, rows, _ := db.Query("SELECT count(*) FROM t", nil); rows[0][0] != int64(0) {
		t.Errorf("the unfinished insert was kept: %v", rows)
	}
}
