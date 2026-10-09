package evaluator

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"Turtle/lexer"
	"Turtle/object"
	"Turtle/parser"
)

// The linear library: results checked by hand or against numpy's, the
// identities each decomposition promises (q * r == m, ...), how a matrix
// shows, and every misuse an error of the right kind.

func TestLinear(t *testing.T) {
	cases := []struct{ src, want string }{
		{"show matrix [1, 2; 3, 4] .", "[ 1  2 ]\n[ 3  4 ]"},
		{"m = matrix [\n    1, 2, 3\n    4, 5, -6\n]\nshow m .", "[ 1  2   3 ]\n[ 4  5  -6 ]"},
		{"m = matrix [1, 2,\n    3; 4, 5, 6]\nshow m at shape .", "[ 2, 3 ]"}, // a comma carries the row on
		{"show matrix [1.5, 2; 3, 4] .", "[ 1.5  2.0 ]\n[ 3.0  4.0 ]"},
		{"show matrix [] .", "[ ]"},
		{"show list [matrix [1, 2; 3, 4]] .", "[ matrix [1, 2; 3, 4] ]"},
		{"x = 2\nshow matrix [x, x * 2; -x, 0] .", "[  2  4 ]\n[ -2  0 ]"},
		{"show typeof[matrix [1]] .", "matrix"},
		{"a = matrix [1, 2; 3, 4]\nb = matrix [5, 6; 7, 8]\nshow a * b .", "[ 19  22 ]\n[ 43  50 ]"},
		{"a = matrix [1, 2; 3, 4]\nshow a + a, a - a .", "[ 2  4 ]\n[ 6  8 ][ 0  0 ]\n[ 0  0 ]"},
		{"a = matrix [1, 2; 3, 4]\nshow 2 * a == a * 2, -a == a * -1 .", "truetrue"},
		{"a = matrix [1, 2; 3, 4]\nshow a + 1, 10 - a, a - 0.5 .", "[ 2  3 ]\n[ 4  5 ][ 9  8 ]\n[ 7  6 ][ 0.5  1.5 ]\n[ 2.5  3.5 ]"},
		{"show matrix [1, 2; 3, 4] / 2 .", "[ 0.5  1.0 ]\n[ 1.5  2.0 ]"},
		{"show matrix [1, 2; 3, 4] * list [1, 1] .", "[ 3, 7 ]"},
		{"show list [1, 1] * matrix [1, 2; 3, 4] .", "[ 4, 6 ]"},
		{"show matrix [1, 2, 3] * matrix [4; 5; 6] .", "[ 32 ]"},
		{"show matrix [1, 2; 3, 4] == matrix [1.0, 2.0; 3.0, 4.0], matrix [1, 2] == matrix [1; 2] .", "truefalse"},
		{"a = matrix [4, 7; 2, 6]\nshow a * inverse[a] == identity[2], inverse[a] * a .", "true[ 1.0  0.0 ]\n[ 0.0  1.0 ]"},
		{"show \"m: \" + matrix [1] .", "m: [ 1 ]"},
		// Methods.
		{"m = matrix [1, 2, 3; 4, 5, 6]\nshow m at rows, m at columns, m at get[1, 2] .", "236"},
		{"m = matrix [1, 2, 3; 4, 5, 6]\nshow m at row[1], m at column[0] .", "[ 4, 5, 6 ][ 1, 4 ]"},
		{"m = zeros[2]\nm = m at put[7, 0, 1]\nshow m .", "[ 0  7 ]\n[ 0  0 ]"},
		{"m = zeros[1, 2]\nm = m at put[0.5, 0, 0]\nshow m, m at get[0, 1] .", "[ 0.5  0.0 ]0.0"},
		{"show matrix [1] at isempty, matrix [] at isempty .", "falsetrue"},
		// Functions.
		{"show identity[2], zeros[1, 3], ones[2, 1] .", "[ 1  0 ]\n[ 0  1 ][ 0  0  0 ][ 1 ]\n[ 1 ]"},
		{"show diagonal[list [1, 2]], diagonal[matrix [1, 2; 3, 4]] .", "[ 1  0 ]\n[ 0  2 ][ 1, 4 ]"},
		{"m = matrix [1, 2, 3; 4, 5, 6]\nshow shape[m], row[m, 0], column[m, 2] .", "[ 2, 3 ][ 1, 2, 3 ][ 3, 6 ]"},
		{"show transpose[matrix [1, 2, 3; 4, 5, 6]] .", "[ 1  4 ]\n[ 2  5 ]\n[ 3  6 ]"},
		{"m = matrix [1, 2; 3, 4]\nshow m transpose .", "[ 1  3 ]\n[ 2  4 ]"},
		{"show trace[matrix [1, 2; 3, 4]], determinant[matrix [1, 2; 3, 4]] .", "5-2"},
		{"show determinant[matrix [2, 0, 1; 1, 3, 2; 1, 1, 2]], determinant[matrix [2, 0, 1; 1, 3, 2; 1, 1, 1]] .", "60"},
		{"show determinant[matrix [0.5, 1; 1, 4]] .", "1.0"},
		{"show determinant[matrix [1, 2; 2, 4]], determinant[matrix []] .", "01"},
		{"show inverse[matrix [4, 7; 2, 6]] .", "[  0.6  -0.7 ]\n[ -0.2   0.4 ]"},
		{"show rank[matrix [1, 2; 2, 4]], rank[identity[3]], rank[zeros[2]] .", "130"},
		{"show power[matrix [1, 1; 1, 0], 10] .", "[ 89  55 ]\n[ 55  34 ]"},
		{"show power[matrix [2, 0; 0, 4], -1], power[matrix [5], 0] .", "[ 0.5   0.0 ]\n[ 0.0  0.25 ][ 1 ]"},
		{"show multiply_each[matrix [1, 2; 3, 4], matrix [5, 6; 7, 8]] .", "[  5  12 ]\n[ 21  32 ]"},
		{"show solve[matrix [2, 1; 1, 3], list [3, 5]] .", "[ 0.8, 1.4 ]"},
		{"a = matrix [2, 1; 1, 3]\nshow a solve list [3, 5] .", "[ 0.8, 1.4 ]"},
		{"show solve[matrix [2, 0; 0, 4], matrix [2, 4; 8, 4]] .", "[ 1.0  2.0 ]\n[ 2.0  1.0 ]"},
		{"show least_squares[matrix [1, 1; 1, 2; 1, 3], list [1, 2, 2]] .", "[ 0.666666666666667, 0.5 ]"},
		{"show dot[list [1, 2, 3], list [4, 5, 6]], cross[list [1, 0, 0], list [0, 1, 0]] .", "32[ 0, 0, 1 ]"},
		{"show norm[list [3, 4]], norm[matrix [1, 1; 1, 1]], unit[list [3, 4]] .", "5.02.0[ 0.6, 0.8 ]"},
		// Decompositions: each gives back what it promises.
		{"m = matrix [2, 1, 1; 4, -6, 0; -2, 7, 2]\nf = lu[m]\nshow f at get[\"p\"] * m == f at get[\"l\"] * f at get[\"u\"] .", "true"},
		{"m = matrix [12, -51, 4; 6, 167, -68; -4, 24, -41]\nf = qr[m]\nshow f at get[\"q\"] * f at get[\"r\"] == m, f at get[\"r\"] .",
			"true[ 14.0   21.0  -14.0 ]\n[  0.0  175.0  -70.0 ]\n[  0.0    0.0   35.0 ]"},
		{"m = matrix [1, 2; 3, 4; 5, 6]\nf = qr[m]\nq = f at get[\"q\"]\nshow q * f at get[\"r\"] == m, transpose[q] * q == identity[2] .", "truetrue"},
		{"e = eigen[matrix [2, 1; 1, 2]]\nshow e at get[\"values\"] .", "[ 3.0, 1.0 ]"},
		{"m = matrix [4, 1, 2; 1, 3, 0; 2, 0, 5]\ne = eigen[m]\nv = e at get[\"vectors\"]\nshow m * v == v * diagonal[e at get[\"values\"]] .", "true"},
		{"m = matrix [1, 2; 3, 4; 5, 6]\nf = svd[m]\nshow f at get[\"u\"] * diagonal[f at get[\"s\"]] * transpose[f at get[\"v\"]] == m .", "true"},
		{"m = matrix [1, 2, 3; 4, 5, 6]\nf = svd[m]\nshow f at get[\"u\"] * diagonal[f at get[\"s\"]] * transpose[f at get[\"v\"]] == m .", "true"},
		{"show svd[matrix [3, 0; 0, 4]] at get[\"s\"] .", "[ 4.0, 3.0 ]"},
		// change, copy, tables.
		{"m = change list [list [1, 2], list [3, 4]] to matrix\nshow m .", "[ 1  2 ]\n[ 3  4 ]"},
		{"m = change list [map [\"a\": 1, \"b\": 2.5]] to matrix\nshow m .", "[ 1.0  2.5 ]"},
		{"show change matrix [1, 2; 3, 4] to list .", "[ [ 1, 2 ], [ 3, 4 ] ]"},
		{"import data\na = matrix [1, 2]\nb = copy[a]\nx = b at put[9, 0, 0]\nshow a, b .", "[ 1  2 ][ 9  2 ]"},
		{"import data\nshow table[matrix [1, 20; 300, 4]] .", "0   1\n---  --\n  1  20\n300   4"},
		// A matrix can be a qualified call's result, and linear's names
		// can be named with the module.
		{"show linear identity[1] .", "[ 1 ]"},
	}
	for _, c := range cases {
		got, err := run(t, "import linear\n"+c.src, "")
		if err != nil {
			t.Errorf("%s: %v", c.src, err)
		} else if strings.TrimSpace(got) != c.want {
			t.Errorf("%s:\n got  %q\n want %q", c.src, strings.TrimSpace(got), c.want)
		}
	}
}

func TestLinearShowsBigMatrices(t *testing.T) {
	got, err := run(t, "import linear\nshow ones[25, 14] .", "")
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(got), "\n")
	if len(lines) != 22 || lines[0] != "[ 1  1  1  1  1  1  1  1  1  1  ... ]" || lines[20] != "  ..." || lines[21] != "(25 x 14 matrix)" {
		t.Errorf("got %d lines:\n%s", len(lines), got)
	}
	// Small enough: no cut, no size line.
	got, _ = run(t, "import linear\nshow ones[20, 10] .", "")
	if strings.Contains(got, "...") || strings.Contains(got, "matrix)") {
		t.Errorf("a 20 x 10 matrix shows whole:\n%s", got)
	}
}

func TestLinearMisuse(t *testing.T) {
	cases := []struct{ src, kind, want string }{
		{`matrix [1, 2] + matrix [1, 2; 3, 4]`, "linear", "can't add a 1 x 2 matrix and a 2 x 2 matrix"},
		{`matrix [1, 2] * matrix [1, 2]`, "linear", "can't multiply a 1 x 2 matrix by a 1 x 2 matrix"},
		{`matrix [1, 2] - list [1]`, "linear", "not a matrix and list"},
		{`matrix [1, 2] * list [1]`, "linear", "the list needs 2"},
		{`1 / matrix [1]`, "linear", "can only be divided by a number"},
		{`matrix [1] / 0`, "math", "division by zero"},
		{`matrix [1] < matrix [2]`, "linear", "doesn't work on a matrix"},
		{`matrix [1, "a"]`, "type", "row 0, column 1 is \"a\""},
		{`inverse[matrix [1, 2; 2, 4]]`, "linear", "singular"},
		{`solve[matrix [1, 2; 2, 4], list [1, 2]]`, "linear", "singular"},
		{`solve[matrix [1, 2; 3, 4], list [1]]`, "linear", "needs 2 numbers"},
		{`inverse[matrix [1, 2]]`, "linear", "needs a square matrix, got a 1 x 2 matrix"},
		{`determinant[list [1]]`, "type", "needs a matrix, got list"},
		{`eigen[matrix [1, 2; 3, 4]]`, "linear", "symmetric"},
		{`least_squares[matrix [1, 2], list [1]]`, "linear", "at least as many rows"},
		{`least_squares[matrix [1, 2; 2, 4; 3, 6], list [1, 2, 3]]`, "linear", "depend on each other"},
		{`qr[matrix [1, 2]]`, "linear", "at least as many rows"},
		{`cross[list [1, 2], list [3, 4]]`, "linear", "3 numbers"},
		{`dot[list [1], list [1, 2]]`, "linear", "same length"},
		{`unit[list [0, 0]]`, "linear", "no direction"},
		{`zeros[-1]`, "linear", "0 or more"},
		{`matrix [1, 2] at get[0, 2]`, "index", "column 2 is out of range for a 1 x 2 matrix"},
		{`matrix [1, 2] at row[1]`, "index", "row 1 is out of range"},
		{`matrix [1, 2] at upper`, "name", "a matrix has no method \"upper\""},
		{`change list [1, 2] to matrix`, "type", "for one row, write list [nums]"},
		{`change list [list [1], list [1, 2]] to matrix`, "linear", "same count"},
		{`change list [map ["a": "x"]] to matrix`, "type", "column \"a\" is \"x\""},
		{`multiply_each[matrix [1], matrix [1, 2]]`, "linear", "same size"},
		{`power[matrix [1, 2], 2]`, "linear", "square"},
	}
	for _, c := range cases {
		src := "import linear\nsafe\n    x = " + c.src + "\nhandle [] e .\n    show kind of e, \"|\", message of e .\nsafe [end]"
		got, err := run(t, src, "")
		if err != nil {
			t.Errorf("%s: %v", c.src, err)
			continue
		}
		if !strings.HasPrefix(got, c.kind+"|") || !strings.Contains(got, c.want) {
			t.Errorf("%s: got %q, want kind %s and %q", c.src, got, c.kind, c.want)
		}
	}
}

// TestMatrixNeedsImport: matrix is a word only in a file that imports
// linear; elsewhere it stays an ordinary name, as it always was.
func TestMatrixNeedsImport(t *testing.T) {
	got, err := run(t, "matrix = 5\ndef grid[matrix]\n    return matrix + 1\ndef [end]\nshow grid[matrix] .", "")
	if err != nil || strings.TrimSpace(got) != "6" {
		t.Errorf("matrix as a name without import linear: %q, %v", got, err)
	}
	if _, err := run(t, "m = change list [list [1]] to matrix", ""); err == nil || !strings.Contains(err.Error(), "import linear") {
		t.Errorf("change ... to matrix without the import: %v", err)
	}
	if _, err := run(t, "import data\nx = row[list [1], 0]", ""); err == nil || !strings.Contains(err.Error(), "import linear") {
		t.Errorf("a linear function without the import: %v", err)
	}
	for _, src := range []string{
		"import linear\nmatrix = 5",
		"import linear\ndef matrix[x]\n    return x\ndef [end]",
		"import linear\nm = matrix [1, 2; 3]",
		"import linear\nm = matrix [1, ; 3]",
	} {
		p := parser.New(lexer.New(src))
		p.ParseProgram()
		if len(p.Errors()) == 0 {
			t.Errorf("%q: no parse error", src)
		}
	}
}

func TestStatistics(t *testing.T) {
	// Expected values from Python's statistics module.
	cases := []struct{ src, want string }{
		{"p = list [950, 3000, 1225, 800]\nshow mean[p], \" \", median[p], \" \", variance[p] .", "1493.75 1087.5 1039322.91666667"},
		{"p = list [950, 3000, 1225, 800]\nshow stdev[p], \" \", pstdev[p], \" \", pvariance[p] .", "1019.47188125356 882.888547609493 779492.1875"},
		{"p = list [950, 3000, 1225, 800]\nshow percentile[p, 90], \" \", percentile[p, 0], \" \", percentile[p, 100] .", "2467.5 800.0 3000.0"},
		{"show median[list [3, 1, 2]], \" \", median[list [1.5]] .", "2 1.5"},
		{"show mode[list [\"a\", \"b\", \"b\", \"a\", \"c\"]], mode[list [1, 2, 2]] .", "a2"},
		{"show zscores[list [1, 2, 3]] .", "[ -1.0, 0.0, 1.0 ]"},
		{"show correlation[list [1, 2, 3, 4], list [2, 4, 6, 8]], \" \", correlation[list [1, 2, 3], list [3, 2, 1]] .", "1.0 -1.0"},
		{"show covariance[list [1, 2, 3], list [1, 2, 4]] .", "1.5"},
		{"show mean[set [1, 2]], \" \", mean[map [\"a\": 4, \"b\": 6]], \" \", mean[list [1, none, 3]] .", "1.5 5.0 2.0"},
		{"p = list [1, 2]\nshow p mean .", "1.5"},
		{"r = list [map [\"p\": 10, \"r\": 4.5], map [\"p\": 20, \"r\": none], map [\"p\": 30, \"r\": 3.5]]\nshow mean[r, \"r\"], \" \", percentile[r, \"p\", 50], \" \", covariance[r, \"p\", \"r\"] .", "4.0 20.0 -10.0"},
		{"assemble book[title, price]\nb = list [book[\"a\", 2], book[\"b\", 4]]\nshow mean[b, \"price\"] .", "3.0"},
		{"d = describe[list [4, 1, 3, 2]]\nshow d .", `{ "count": 4, "mean": 2.5, "stdev": 1.29099444873581, "min": 1, "25%": 1.75, "median": 2.5, "75%": 3.25, "max": 4 }`},
		{"show describe[list [7]] at get[\"stdev\"] .", "none"},
	}
	for _, c := range cases {
		got, err := run(t, "import data\n"+c.src, "")
		if err != nil {
			t.Errorf("%s: %v", c.src, err)
		} else if strings.TrimSpace(got) != c.want {
			t.Errorf("%s:\n got  %q\n want %q", c.src, strings.TrimSpace(got), c.want)
		}
	}
}

func TestStatisticsMisuse(t *testing.T) {
	cases := []struct{ src, kind, want string }{
		{`mean[list []]`, "index", "needs any numbers, got 0"},
		{`stdev[list [1]]`, "index", "needs at least 2 numbers, got 1"},
		{`mean[list [1, "a"]]`, "type", `item 1 is "a"`},
		{`mean[5]`, "type", "needs a list, set or map of numbers, got integer"},
		{`percentile[list [1], 101]`, "math", "0 to 100"},
		{`correlation[list [1, 1], list [1, 2]]`, "math", "every number the same"},
		{`correlation[list [1, 2], list [1]]`, "index", "same length"},
		{`mean[list [map ["a": 1]], "b"]`, "key", `no column "b"`},
		{`mean[list [1], "b"]`, "type", "needs a list of rows"},
		{`zscores[list [2, 2]]`, "math", "every number is the same"},
	}
	for _, c := range cases {
		src := "import data\nsafe\n    x = " + c.src + "\nhandle [] e .\n    show kind of e, \"|\", message of e .\nsafe [end]"
		got, err := run(t, src, "")
		if err != nil {
			t.Errorf("%s: %v", c.src, err)
			continue
		}
		if !strings.HasPrefix(got, c.kind+"|") || !strings.Contains(got, c.want) {
			t.Errorf("%s: got %q, want kind %s and %q", c.src, got, c.kind, c.want)
		}
	}
}

// TestLinearExamples runs testdata/linear: the edge cases with turtle
// test, and housing.trt, a program that checks its own answers.
func TestLinearExamples(t *testing.T) {
	dir, err := filepath.Abs("../testdata/linear")
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if code := TestCommand(nil, dir, &out); code != 0 || !strings.Contains(out.String(), "ok: ") {
		t.Fatalf("turtle test: exit %d:\n%s", code, out.String())
	}
	work := t.TempDir()
	src, err := os.ReadFile(filepath.Join(dir, "housing.trt"))
	if err != nil {
		t.Fatal(err)
	}
	got, err := runIn(t, work, string(src), "")
	var exit ExitRequest
	if err != nil && (!errors.As(err, &exit) || exit.Code != 0) {
		t.Fatalf("housing.trt: %v\n%s", err, got)
	}
	if !strings.Contains(got, "housing: all checks passed") {
		t.Errorf("housing.trt:\n%s", got)
	}
}

// TestLoadTypesOnServers: sql_load into a PostgreSQL or MySQL table
// that's already there takes the table's column types, as SQLite's does.
// It runs when TURTLE_PG_URL or TURTLE_MYSQL_URL is set.
func TestLoadTypesOnServers(t *testing.T) {
	ran := false
	for _, env := range []string{"TURTLE_PG_URL", "TURTLE_MYSQL_URL"} {
		dsn := os.Getenv(env)
		if dsn == "" {
			continue
		}
		ran = true
		dir := t.TempDir()
		os.WriteFile(filepath.Join(dir, "codes.csv"), []byte("code,n,f\n7,\"42\",2\n"), 0o644)
		os.WriteFile(filepath.Join(dir, "bad.csv"), []byte("code,n,f\n7,lots,2\n"), 0o644)
		src := `import sql
db = sql_open["` + dsn + `"]
sql_run[db, "DROP TABLE IF EXISTS load_types"]
sql_run[db, "CREATE TABLE load_types (code VARCHAR(10), n INTEGER, f DOUBLE PRECISION)"]
show sql_load[db, "load_types", "codes.csv"] .
show sql_query[db, "SELECT code, n, f FROM load_types"] .
safe
    sql_load[db, "load_types", "bad.csv"]
handle [number] e .
    show "refused" .
safe [end]
sql_run[db, "DROP TABLE load_types"]`
		got, err := runIn(t, dir, src, "")
		want := "1\n[ { \"code\": \"7\", \"n\": 42, \"f\": 2.0 } ]\nrefused\n"
		if err != nil {
			t.Errorf("%s: %v", env, err)
		} else if got != want {
			t.Errorf("%s: got %q, want %q", env, got, want)
		}
	}
	if !ran {
		t.Skip("set TURTLE_PG_URL or TURTLE_MYSQL_URL to run against a server")
	}
}

func TestIsPlainNumber(t *testing.T) {
	for s, want := range map[string]bool{
		"0": true, "7": true, "-3": true, "2.5": true, "-0.25": true, "10": true, "0.5": true,
		"": false, "-": false, "007": false, "01.5": false, "+5": false, ".5": false, "5.": false,
		"1e5": true, "2.5e-3": true, "6E+2": true, "1e": false, "1e+": false, "e5": false, "1.e5": false, "1e5.5": false, "1,000": false, "$5": false, " 5": false, "5 ": false, "--5": false, "1.2.3": false,
	} {
		if got := isPlainNumber(s); got != want {
			t.Errorf("isPlainNumber(%q) = %v, want %v", s, got, want)
		}
	}
}

// TestScientificNumbers: written and shown as Python does, past 1e16 and
// below 0.0001.
func TestScientificNumbers(t *testing.T) {
	cases := []struct{ src, want string }{
		{"show 1e-18, \" \", 2.5e6, \" \", 6.02e23, \" \", -1e3 .", "1e-18 2500000.0 6.02e+23 -1000.0"},
		{"show 0.0001, \" \", 0.00009, \" \", 1e15, \" \", 1e16 .", "0.0001 9e-05 1000000000000000.0 1e+16"},
		{"show 1e-9 == 0.000000001, \" \", typeof[1e3] .", "true float"},
		{"x = 2\nshow x*1e3 .", "2000.0"},
	}
	for _, c := range cases {
		got, err := run(t, c.src, "")
		if err != nil {
			t.Errorf("%s: %v", c.src, err)
		} else if strings.TrimSpace(got) != c.want {
			t.Errorf("%s:\n got  %q\n want %q", c.src, strings.TrimSpace(got), c.want)
		}
	}
}

// TestMethodArgumentsWithoutBrackets: x at pow 2 works wherever a value
// goes, its argument running to the end of the value, as a sentence
// call's does; inside brackets a comma ends it.
func TestMethodArgumentsWithoutBrackets(t *testing.T) {
	cases := []struct{ src, want string }{
		{"x = 2 at pow 10\nshow x .", "1024"},
		{"show 2 at pow 10 .", "1024"},
		{"x = list [2 at pow 10, 3]\nshow x .", "[ 1024, 3 ]"},
		{"nums = list [5, 6]\nshow list [nums at get 0, nums at get 1] .", "[ 5, 6 ]"},
		{"s = \"turtle\"\nif ] s at contains \"urt\" [\n    show \"yes\" .\nif [end]", "yes"},
		{"x = 2 at pow[10] + 1\nshow x .", "1025"},
		{"x = 2 at pow 10 + 1\nshow x .", "1025"},
		{"show 3 at pow 2 == 9 .", "true"},
		{"s = \"turtle\"\nok = true\nshow s at contains \"urt\" && ok .", "true"},
		{"x = 2\nshow x at pow 3, \" \", x .", "8 2"},
		{"nums = list [3, 4]\nshow nums at get 1 at pow 2 .", "16"},
		{"show 5 at pow -1 .", "0.2"},
		{"r is 2 at pow 3 .\nshow r .", "8"},
		{"s = \"a,b\"\nshow s at split \",\" .", "[ \"a\", \"b\" ]"},
	}
	for _, c := range cases {
		got, err := run(t, "import math\n"+c.src, "")
		if err != nil {
			t.Errorf("%s: %v", c.src, err)
		} else if strings.TrimSpace(got) != c.want {
			t.Errorf("%s:\n got  %q\n want %q", c.src, strings.TrimSpace(got), c.want)
		}
	}
}

// TestMethodStatements: a method call on its own line is a statement, for
// every kind of value, and put has a matrix sentence: put v to m at r, c .
func TestMethodStatements(t *testing.T) {
	cases := []struct{ src, want string }{
		{"nums = list [3, 1]\nnums at add 2\nnums at sort\nshow nums .", "[ 1, 2, 3 ]"},
		{"nums = list [1, 2]\nnums at put[9, 0] .\nshow nums .", "[ 9, 2 ]"},
		{"nums = list [1, 2]\nnums at put 9, 1 .\nshow nums .", "[ 1, 9 ]"},
		{"import linear\nm = zeros[2]\nm at put 3, 1, 1 .\nshow m at get[1, 1] .", "3"},
		{"ages = map [\"a\": 1]\nages at add[\"b\", 2]\nshow ages .", "{ \"a\": 1, \"b\": 2 }"},
		{"import linear\nm = zeros[2]\nm at put[7, 0, 1]\nput 5 to m at 1, 0 .\nshow m .", "[ 0  7 ]\n[ 5  0 ]"},
		{"import linear\nm = zeros[1]\nput 0.5 to m at 0, 0 .\nshow m at get[0, 0] .", "0.5"},
	}
	for _, c := range cases {
		got, err := run(t, c.src, "")
		if err != nil {
			t.Errorf("%s: %v", c.src, err)
		} else if strings.TrimSpace(got) != c.want {
			t.Errorf("%s:\n got  %q\n want %q", c.src, strings.TrimSpace(got), c.want)
		}
	}
	p := parser.New(lexer.New("nums = list [1]\nnums at get[0] + 1"))
	p.ParseProgram()
	if errs := p.Errors(); len(errs) == 0 || !strings.Contains(errs[0], "this line is a value, not a statement") {
		t.Errorf("a value as a statement: %v", errs)
	}
	for _, c := range []struct{ src, want string }{
		{"import linear\nm = zeros[2]\nx = m at get 0", "without brackets a method takes one argument; for more, use brackets: x at get[a, b]"},
		{"import linear\nm = zeros[2]\nput 1 to m at 0 .", "needs a row and a column for a matrix"},
		{"nums = list [1]\nput 1 to nums at 0, 0 .", "has two positions"},
	} {
		if _, err := run(t, c.src, ""); err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: got %v, want %q", c.src, err, c.want)
		}
	}
}

// TestFieldOfBareGet: after "of", get with one bare argument reaches into
// the value, as get[...] does.
func TestFieldOfBareGet(t *testing.T) {
	got, err := run(t, "rows = list [map [\"n\": 1], map [\"n\": 2]]\nshow n of rows at get 1, \" \", n of rows at get[0] .", "")
	if err != nil || strings.TrimSpace(got) != "2 1" {
		t.Errorf("got %q, %v", got, err)
	}
}

// TestAssignmentMayEndInAPeriod: "x = ... ." is fine, as a call line is.
func TestAssignmentMayEndInAPeriod(t *testing.T) {
	got, err := run(t, "import data\nnums = list [1, 2, 3]\nbig = nums keep n give n > 1 .\nx = 5 .\nshow big, x .", "")
	if err != nil || strings.TrimSpace(got) != "[ 2, 3 ]5" {
		t.Errorf("got %q, %v", got, err)
	}
}

// TestReusedScopesKeepClosures: a loop pass's or a call's scope is reused
// only when no function kept it, so closures still see their own values.
func TestReusedScopesKeepClosures(t *testing.T) {
	src := `import data
fns = list []
[loop][x in list [1, 2, 3, 4]]
    if ] x % 2 == 0 [
        add [] give x * 10 to fns .
    if [end]
[loop][end]
show process[fns, f give f[]] .
def adder[n]
    [loop][i in list [1]]
        f = x give x + n + i
    [loop][end]
    return f
def [end]
a = adder[10]
b = adder[20]
show a[1], " ", b[1] .
def count[n]
    if ] n == 0 [
        return list []
    if [end]
    rest = count[n - 1]
    add [] give n to rest .
    return rest
def [end]
show process[count[3], f give f[]] .
def plain[n]
    m = n * 2
    return m
def [end]
total = 0
[loop][i in range[1, 1000]]
    total = total + plain[i]
[loop][end]
show total, " ", a[0] .`
	got, err := run(t, src, "")
	want := "[ 20, 40 ]\n12 22\n[ 1, 2, 3 ]\n999000 11\n"
	if err != nil || got != want {
		t.Errorf("got %q, %v; want %q", got, err, want)
	}
}

// TestFastJSONMatchesTheDecoder: for every text the fast reader takes,
// it gives exactly what the token-stream reader gives, keys in the same
// order; and texts it declines still load (through the decoder).
func TestFastJSONMatchesTheDecoder(t *testing.T) {
	slow := func(text string) (object.Object, bool) {
		dec := json.NewDecoder(strings.NewReader(text))
		dec.UseNumber()
		v, err := decodeJSON(dec)
		if err != nil {
			return nil, false
		}
		if _, extra := dec.Token(); extra != io.EOF {
			return nil, false
		}
		return v, true
	}
	cases := []string{
		`{}`, `[]`, `0`, `-0`, `-12`, `3.25`, `1e5`, `1E+2`, `-2.5e-3`, `12345678901234567890`,
		`true`, `false`, `null`, `""`, `"plain"`, ` { "b" : 1 , "a" : [ 1 , 2 ] } `,
		`{"a": 1, "a": 2, "b": 3}`, `[[[]]]`, `{"x": {"y": {"z": null}}}`,
		`"tab\tin"`, `"esc\n"`, `"é"`, `"é"`, `"bad \u0001"`, `01`, `1.`, `.5`, `+1`, `-`,
		`[1,]`, `{"a":1,}`, `{"a" 1}`, `[1 2]`, `tru`, `nul`, `"open`, `1 2`, `{"a":1}x`, `1e400`,
	}
	r := rand.New(rand.NewSource(7))
	var gen func(d int) any
	gen = func(d int) any {
		switch r.Intn(7) {
		case 0:
			return r.Intn(1000) - 500
		case 1:
			return r.NormFloat64() * 1e6
		case 2:
			return fmt.Sprintf("s%d", r.Intn(100))
		case 3:
			return r.Intn(2) == 0
		case 4:
			return nil
		case 5:
			if d < 4 {
				n := r.Intn(4)
				l := make([]any, n)
				for i := range l {
					l[i] = gen(d + 1)
				}
				return l
			}
		case 6:
			if d < 4 {
				m := map[string]any{}
				for i := r.Intn(4); i > 0; i-- {
					m[fmt.Sprintf("k%d", r.Intn(10))] = gen(d + 1)
				}
				return m
			}
		}
		return 1
	}
	for i := 0; i < 3000; i++ {
		b, _ := json.MarshalIndent(gen(0), "", strings.Repeat(" ", i%3))
		cases = append(cases, string(b))
	}
	for _, c := range cases {
		want, wok := slow(c)
		got, gok := fastJSON(c)
		if gok && !wok {
			t.Errorf("fast reader took %q, which the decoder refuses", c)
		}
		if gok && (!object.Equal(got, want) || got.Inspect() != want.Inspect()) {
			t.Errorf("%q: fast %s, decoder %s", c, got.Inspect(), want.Inspect())
		}
	}
}

// TestLogicLikePopularLanguages: && and || after a sentence call work on
// the call's result, comparisons bind tighter than &&, && tighter than ||,
// ! applies to the value after it, and && / || stop early, as in Python,
// JavaScript and Go. Arithmetic still belongs to a sentence's argument.
func TestLogicLikePopularLanguages(t *testing.T) {
	src := `import strings
import data
def has[text, part]
    return isinstring[part, text]
def [end]
def boom[]
    fail "evaluated"
def [end]
def twice[n]
    return n * 2
def [end]
s = "turtle"
ok = false
yes = true
nums = list [10, 20, 30]
i = 0
show s has "urt" && ok .
show s has "urt" || ok .
show s has "x" || yes .
show s has "urt" && s at contains "tle" .
show s at contains "urt" && ok .
show nums get i + 1 .
show nums get i == 10 .
show nums get i + 1 == 20 .
show 5 twice + 1 .
show 5 twice == 10 && yes .
show 5 twice > 9 || ok .
show yes || ok && ok, " ", !ok && yes, " ", !yes || yes, " ", 1 < 2 == true .
show false && boom[], " ", true || boom[] .
show nums keep n give n > 15 && n < 30 .
if ] s has "urt" && !ok [
    show "if works" .
if [end]
count = 0
[loop][count < 3 && yes]
    count = count + 1
[loop][end]
show count .`
	got, err := run(t, src, "")
	want := "false\ntrue\ntrue\ntrue\nfalse\n20\ntrue\ntrue\n11\ntrue\ntrue\ntrue true true true\nfalse true\n[ 20 ]\nif works\n3\n"
	if err != nil || got != want {
		t.Errorf("got %q, %v\nwant %q", got, err, want)
	}
}

// TestTurtleDatabasesPassSQLite3: a database written by a Turtle program,
// through every kind of change, is one the official sqlite3 tool accepts
// (PRAGMA integrity_check) and reads the same data from. CI sets
// TURTLE_REQUIRE_SQLITE3 so a missing tool fails instead of skipping.
func TestTurtleDatabasesPassSQLite3(t *testing.T) {
	bin, err := exec.LookPath("sqlite3")
	if err != nil {
		if os.Getenv("TURTLE_REQUIRE_SQLITE3") != "" {
			t.Fatal("sqlite3 isn't installed, and TURTLE_REQUIRE_SQLITE3 is set")
		}
		t.Skip("sqlite3 isn't installed")
	}
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "more.csv"), []byte("id,name,qty\n90001,loaded,5\n90002,\"comma, name\",7\n"), 0o644)
	src := `import sql
db = sql_create["shop.db"]
sql_run[db, "CREATE TABLE items (id INTEGER PRIMARY KEY, name TEXT, qty INTEGER)"]
sql_run[db, "CREATE INDEX by_name ON items (name)"]
sql_run[db, "CREATE TABLE log (what TEXT)"]
sql_run[db, "CREATE TRIGGER logged AFTER DELETE ON items BEGIN INSERT INTO log VALUES (old.name); END"]
sql_run[db, "BEGIN"]
[loop][i = 1; i <= 5000; i++]
    sql_run[db, "INSERT INTO items VALUES (?, ?, ?)", list [i, "item " + i, i % 97]]
[loop][end]
sql_run[db, "COMMIT"]
sql_run[db, "UPDATE items SET qty = qty + 1 WHERE id % 3 = 0"]
sql_run[db, "DELETE FROM items WHERE id % 5 = 0"]
sql_run[db, "ALTER TABLE items ADD COLUMN note TEXT"]
sql_run[db, "UPDATE items SET note = 'long note ' || id || ' ' || hex(randomblob(40)) WHERE id < 200"]
sql_run[db, "CREATE VIEW low AS SELECT id FROM items WHERE qty < 5"]
sql_load[db, "items", "more.csv"]
r = sql_query[db, "SELECT count(*) AS n, sum(qty) AS q, (SELECT count(*) FROM log) AS l FROM items"]
show n of r at get 0, " ", q of r at get 0, " ", l of r at get 0 .
sql_close[db]`
	got, err := runIn(t, dir, src, "")
	if err != nil {
		t.Fatal(err)
	}
	out, err := exec.Command(bin, filepath.Join(dir, "shop.db"), "PRAGMA integrity_check").CombinedOutput()
	if err != nil || strings.TrimSpace(string(out)) != "ok" {
		t.Fatalf("sqlite3 integrity_check: %v\n%s", err, out)
	}
	out, err = exec.Command(bin, filepath.Join(dir, "shop.db"), "SELECT count(*) || ' ' || sum(qty) || ' ' || (SELECT count(*) FROM log) FROM items").CombinedOutput()
	if err != nil || strings.TrimSpace(string(out)) != strings.TrimSpace(got) {
		t.Fatalf("sqlite3 reads %q, Turtle %q (%v)", strings.TrimSpace(string(out)), strings.TrimSpace(got), err)
	}
}
