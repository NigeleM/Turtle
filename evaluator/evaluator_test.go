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
		{name: "unknown module", src: `show nope now[] .`, wantErr: `unknown module "nope"`},
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
