package evaluator

import (
	"bufio"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

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
		{"concat", `show "x=" + none .`, "x=none\n"},
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
		{"string + list still concatenates", `show "x" + list [1] .`, "x[ 1 ]\n"},
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
show x of n .`, wantErr: "'x of' needs an assembled value, got INTEGER"},
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
