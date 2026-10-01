package evaluator

import (
	"bufio"
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
show r .`, "[ a, b, c ]\ntrue\n4\na,X,c\n"},
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
	want := "4\n7\n5\n4\n5\n1024\ntrue\n"
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
r is 2 at pow 3 .`, want: "4\n", wantErr: `"pow" isn't imported — add it to "import math [...]"`},
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
show words .`, want: "[ HEY, DO ]\n"},
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
show ages .`, want: "{ Alice: 31, Bob: 26 }\n{ Alice: Alice=31, Bob: Bob=26 }\n"},
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
show m .`, want: "[ 5, 8 ]\n{ 1, 3 }\n{ b: 5 }\n"},
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
		{"map + keeps first on shared key", `show map ["a": 1, "b": 2] + map ["b": 99, "c": 3] .`, "{ a: 1, b: 2, c: 3 }\n"},
		{"map - by key", `show map ["a": 1, "b": 2] - map ["b": 0] .`, "{ a: 1 }\n"},
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
show length of a .`, args: []string{"one", "two words"}, want: "[ one, two words ]\n2\n"},
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
