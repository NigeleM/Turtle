package sqlite

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// The tests compare this package with real SQLite without running it:
// testdata/ holds the fixture database at each page size, built by the
// sqlite3 tool, and answers.json holds sqlite3's answer to every query.
// So the tests behave the same on every machine, whatever sqlite3 it
// has, or none. Builds differ: GitHub's macOS sqlite3 uppercases é with
// ICU, where standard SQLite (and this package) changes ASCII only, and
// 3.45 and older write blobs wrongly in -json. After changing the fixture
// or the queries, rebuild with a standard sqlite3 (upper('é') stays 'é')
// and commit the result:
//
//	go test ./sqlite -update
var update = flag.Bool("update", false, "rebuild testdata/ with the sqlite3 tool")

var pageSizes = []int{512, 1024, 4096, 65536}

func fixturePath(size int) string {
	return filepath.Join("testdata", "fixture-"+strconv.Itoa(size)+".db")
}

func TestMain(m *testing.M) {
	flag.Parse()
	if *update {
		if err := rebuildTestdata(); err != nil {
			fmt.Fprintln(os.Stderr, "rebuilding testdata:", err)
			os.Exit(1)
		}
	}
	os.Exit(m.Run())
}

// rebuildTestdata is the only use of the sqlite3 tool; the package itself
// never needs it.
func rebuildTestdata() error {
	bin, err := exec.LookPath("sqlite3")
	if err != nil {
		return fmt.Errorf("the sqlite3 tool isn't installed")
	}
	if err := os.MkdirAll("testdata", 0o755); err != nil {
		return err
	}
	for _, size := range pageSizes {
		path := fixturePath(size)
		os.Remove(path)
		cmd := exec.Command(bin, path)
		cmd.Stdin = strings.NewReader("PRAGMA page_size=" + strconv.Itoa(size) + ";\n" + fixture)
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("sqlite3: %v\n%s", err, out)
		}
	}
	answers := map[string]json.RawMessage{}
	for _, q := range append(queries, pageQueries...) {
		out, err := exec.Command(bin, "-json", fixturePath(4096), q).CombinedOutput()
		if err != nil {
			return fmt.Errorf("sqlite3 %q: %v\n%s", q, err, out)
		}
		text := strings.TrimSpace(string(out))
		if text == "" {
			text = "[]"
		}
		if !json.Valid([]byte(text)) {
			return fmt.Errorf("sqlite3 output for %q isn't JSON:\n%s", q, out)
		}
		answers[q] = json.RawMessage(text)
	}
	data, err := json.MarshalIndent(answers, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join("testdata", "answers.json"), append(data, '\n'), 0o644)
}

func openFixture(t *testing.T, size int) *DB {
	t.Helper()
	db, err := Open(fixturePath(size))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

// sqliteAnswer is sqlite3's answer to a query, from testdata/answers.json.
func sqliteAnswer(t *testing.T, query string) []map[string]any {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", "answers.json"))
	if err != nil {
		t.Fatal(err)
	}
	var answers map[string][]map[string]any
	if err := json.Unmarshal(data, &answers); err != nil {
		t.Fatal(err)
	}
	rows, ok := answers[query]
	if !ok {
		t.Fatalf("no answer for %q in testdata/answers.json; run go test ./sqlite -update", query)
	}
	if rows == nil {
		rows = []map[string]any{}
	}
	return rows
}

const fixture = `
CREATE TABLE books (id INTEGER PRIMARY KEY, sku TEXT, title TEXT, price INTEGER, rating REAL, cover BLOB);
INSERT INTO books (sku, title, price, rating, cover) VALUES
  ('B1', 'Dune', 950, 4.5, X'CAFE'),
  ('B2', 'The Go Book', 3000, NULL, NULL),
  ('B3', 'Gone Girl', 1225, 3.9, NULL),
  ('B4', 'Foundation', 800, 4.25, X''),
  ('B5', 'Café Ünïcode 🐢', 1, 5.0, NULL),
  ('B6', 'zebra', NULL, 2, NULL);
CREATE TABLE "odd name" ("first col" TEXT, [second] INTEGER, ` + "`third`" + ` REAL);
INSERT INTO "odd name" VALUES ('a', 1, 1.5), ('b', 2, NULL), (NULL, NULL, NULL);
CREATE TABLE nums (n INTEGER, t TEXT, v);
WITH RECURSIVE c(x) AS (SELECT 1 UNION ALL SELECT x + 1 FROM c WHERE x < 5000)
  INSERT INTO nums SELECT x, 'row ' || x, CASE WHEN x % 3 = 0 THEN x * 1.5 WHEN x % 3 = 1 THEN 'text' || x ELSE NULL END FROM c;
CREATE TABLE big (k INTEGER PRIMARY KEY, body TEXT);
INSERT INTO big VALUES (1, 'short'), (2, printf('%.*c', 70000, 'x')), (3, printf('%.*c', 5000, 'y') || 'end');
CREATE TABLE later (a TEXT);
INSERT INTO later VALUES ('old');
ALTER TABLE later ADD COLUMN b INTEGER;
INSERT INTO later VALUES ('new', 7);
CREATE TABLE aff (t TEXT, i INTEGER, r REAL, n NUMERIC, none_col);
INSERT INTO aff VALUES ('5', 5, 5, '5', '5'), ('10', 10, 10.5, 10, 10), ('abc', 0, 0, 'abc', 'abc');
CREATE INDEX books_price ON books (price);
`

// queries are run by both this package and sqlite3, and must agree.
var queries = []string{
	"SELECT * FROM books",
	"SELECT id, title FROM books WHERE price > 900 ORDER BY price DESC",
	"SELECT title FROM books WHERE rating IS NULL",
	"SELECT title FROM books WHERE rating IS NOT NULL AND price BETWEEN 800 AND 1300 ORDER BY title",
	"SELECT title, price FROM books ORDER BY price",
	"SELECT title FROM books WHERE title LIKE 'g%' ORDER BY 1",
	"SELECT title FROM books WHERE title LIKE '%o_k%'",
	"SELECT sku FROM books WHERE sku IN ('B1', 'B3', 'B9') ORDER BY sku DESC",
	"SELECT sku FROM books WHERE price NOT IN (950, 800)",
	"SELECT count(*) AS n, sum(price) AS total, avg(price) AS mean, min(title) AS first, max(rating) AS best FROM books",
	"SELECT count(rating) AS rated, total(price) AS t, group_concat(sku) AS skus FROM books",
	"SELECT count(*) AS c FROM books WHERE price > 100000",
	"SELECT sum(price) AS s, avg(price) AS a FROM books WHERE price > 100000",
	"SELECT DISTINCT price > 1000 AS pricey FROM books ORDER BY pricey",
	"SELECT title FROM books ORDER BY title LIMIT 2",
	"SELECT title FROM books ORDER BY title LIMIT 2 OFFSET 3",
	"SELECT title FROM books ORDER BY title LIMIT 1, 2",
	"SELECT upper(title) AS u, lower(sku) AS l, length(title) AS len FROM books ORDER BY id",
	"SELECT substr(title, 2, 3) AS a, substr(title, -3) AS b, replace(title, 'o', '0') AS c, instr(title, 'o') AS d FROM books",
	"SELECT price / 3 AS d, price % 7 AS m, price * 2 + 1 AS e, -price AS neg, price / 0 AS z FROM books",
	"SELECT rating * 2 AS r2, round(rating) AS rr, round(rating, 1) AS r1, abs(-rating) AS ab FROM books",
	"SELECT title || ' (' || sku || ')' AS label FROM books ORDER BY label DESC",
	"SELECT coalesce(rating, -1) AS c, ifnull(price, 0) AS p, nullif(price, 950) AS np FROM books",
	"SELECT typeof(price) AS a, typeof(rating) AS b, typeof(cover) AS c, typeof(title) AS d FROM books",
	"SELECT CASE WHEN price > 1000 THEN 'high' WHEN price > 900 THEN 'mid' ELSE 'low' END AS band FROM books",
	"SELECT CASE sku WHEN 'B1' THEN 'first' ELSE 'other' END AS x FROM books",
	"SELECT CAST(price AS TEXT) AS t, CAST(rating AS INTEGER) AS i, CAST('12abc' AS INTEGER) AS j FROM books",
	"SELECT hex(cover) AS h FROM books",
	`SELECT "first col", second, third FROM "odd name" ORDER BY "first col"`,
	`SELECT * FROM "odd name" WHERE third IS NULL`,
	"SELECT count(*) AS c, sum(n) AS s, max(t) AS m FROM nums",
	"SELECT n, t, v FROM nums WHERE n > 4990",
	"SELECT n FROM nums WHERE v = 'text4999' OR n = 2500",
	"SELECT count(*) AS c FROM nums WHERE v IS NULL",
	"SELECT count(*) AS c FROM nums WHERE t LIKE 'ROW 1%'",
	"SELECT n FROM nums ORDER BY n DESC LIMIT 3",
	"SELECT k, length(body) AS len, substr(body, -3) AS tail FROM big",
	"SELECT * FROM later",
	"SELECT a FROM later WHERE b IS NULL",
	"SELECT t FROM aff WHERE t = 5",
	"SELECT i FROM aff WHERE i = '10'",
	"SELECT r FROM aff WHERE r > '6'",
	"SELECT n FROM aff WHERE n = 5",
	"SELECT none_col FROM aff WHERE none_col = 5",
	"SELECT none_col FROM aff WHERE none_col = '5'",
	"SELECT t FROM aff ORDER BY t",
	"SELECT 1 + 1 AS two, 'a' || 'b' AS ab, 7 / 2 AS half, 7.0 / 2 AS realhalf, NULL IS NULL AS isn",
	"SELECT 10 > 9 AS a, '10' > '9' AS b, 1 = 1.0 AS c, 'abc' < 'abd' AS d, NULL = NULL AS e",
	"SELECT name, type FROM sqlite_master WHERE type = 'table' ORDER BY name",
	"SELECT min(1, 2, 3) AS a, max('a', 'b') AS b, iif(1 > 2, 'y', 'n') AS c, trim('  hi  ') AS d",
	"SELECT title FROM books WHERE NOT (price > 1000) ORDER BY title",
	"SELECT title FROM books WHERE price > 1000 OR rating > 4.4 ORDER BY id",
	"SELECT count(DISTINCT rating > 4) AS d FROM books",
	"SELECT 9223372036854775807 + 1 AS big, -9223372036854775808 - 1 AS small",
	"SELECT 5 & 3 AS a, 5 | 3 AS b, 1 << 4 AS c, ~0 AS d",
	"SELECT b.title FROM books b WHERE b.price = 950",
	"SELECT books.title FROM books WHERE books.price = 950",
	"SELECT rowid FROM books WHERE rowid = 3",
	"SELECT rowid, n FROM nums WHERE rowid = 3",
	"SELECT upper('café') AS u, lower('ÉCOLE') AS l",
}

func sortRows(rows []map[string]any) {
	sort.Slice(rows, func(i, j int) bool {
		a, _ := json.Marshal(rows[i])
		b, _ := json.Marshal(rows[j])
		return string(a) < string(b)
	})
}

// ourJSON runs a query through this package and shapes the rows like
// sqlite3 -json does. Blob columns are left out, and named in the
// returned set so the caller drops them from sqlite3's rows too: how the
// tool writes blob bytes in JSON changed between versions (3.45 and
// older write byte 0xCA as "\uffffffca"), so blobs are checked through
// hex() in queries and exact bytes in TestTablesAndTypes instead, and
// answers.json stays right whichever sqlite3 rebuilt it.
func ourJSON(t *testing.T, db *DB, query string) ([]map[string]any, map[string]bool, error) {
	t.Helper()
	cols, rows, err := db.Query(query, nil)
	if err != nil {
		return nil, nil, err
	}
	blobs := map[string]bool{}
	for _, r := range rows {
		for i, c := range cols {
			if _, ok := r[i].([]byte); ok {
				blobs[c] = true
			}
		}
	}
	out := []map[string]any{}
	for _, r := range rows {
		m := map[string]any{}
		for i, c := range cols {
			if !blobs[c] {
				m[c] = r[i]
			}
		}
		out = append(out, m)
	}
	data, _ := json.Marshal(out)
	var back []map[string]any
	if err := json.Unmarshal(data, &back); err != nil {
		t.Fatal(err)
	}
	return back, blobs, nil
}

// dropColumns removes the named columns from rows.
func dropColumns(rows []map[string]any, cols map[string]bool) {
	for _, r := range rows {
		for c := range cols {
			delete(r, c)
		}
	}
}

func TestQueriesMatchSQLite(t *testing.T) {
	db := openFixture(t, 4096)
	for _, q := range queries {
		want := sqliteAnswer(t, q)
		got, blobs, err := ourJSON(t, db, q)
		if err != nil {
			t.Errorf("%s\n  error: %v", q, err)
			continue
		}
		dropColumns(want, blobs)
		// Without ORDER BY, SQLite may return rows in any order (it can
		// read through an index), so compare those as sets.
		if !strings.Contains(strings.ToUpper(q), "ORDER BY") {
			sortRows(got)
			sortRows(want)
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("%s\n  got:  %v\n  want: %v", q, got, want)
		}
	}
}

func TestTablesAndTypes(t *testing.T) {
	db := openFixture(t, 4096)
	if got := strings.Join(db.Tables(), ","); got != "books,odd name,nums,big,later,aff" {
		t.Errorf("tables: %s", got)
	}
	// Exact Go types, which JSON comparison can't see.
	_, rows, err := db.Query("SELECT id, price, rating, cover, title, NULL, 7 / 2, 7.0 / 2 FROM books WHERE id = 1", nil)
	if err != nil {
		t.Fatal(err)
	}
	want := []Value{int64(1), int64(950), 4.5, []byte{0xca, 0xfe}, "Dune", nil, int64(3), 3.5}
	if !reflect.DeepEqual(rows[0], want) {
		t.Errorf("got %#v, want %#v", rows[0], want)
	}
}

func TestParameters(t *testing.T) {
	db := openFixture(t, 4096)
	_, rows, err := db.Query("SELECT title FROM books WHERE price < ? AND sku != ? ORDER BY title", []Value{int64(1000), "B5"})
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 || rows[0][0] != "Dune" || rows[1][0] != "Foundation" {
		t.Errorf("got %v", rows)
	}
	_, rows, err = db.Query("SELECT ?2 || ?1 AS x", []Value{"b", "a"})
	if err != nil || rows[0][0] != "ab" {
		t.Errorf("numbered: %v %v", rows, err)
	}
	// A text parameter compared with an INTEGER column converts, like SQLite.
	_, rows, err = db.Query("SELECT sku FROM books WHERE price = ?", []Value{"950"})
	if err != nil || len(rows) != 1 {
		t.Errorf("affinity on a parameter: %v %v", rows, err)
	}
}

func TestErrors(t *testing.T) {
	db := openFixture(t, 4096)
	cases := map[string]string{
		"SELECT * FROM nope":                            "no such table: nope",
		"SELECT nope FROM books":                        "no such column: nope",
		"SELECT title FROM books WHERE":                 "the statement ends too soon",
		"SELECT title FROM books WHERE price ?":         "unexpected",
		"INSERT INTO books VALUES (1)":                  "only SELECT is supported so far, not INSERT",
		"SELECT title FROM books GROUP BY sku":          "GROUP BY isn't supported yet",
		"SELECT * FROM books, nums":                     "joins aren't supported yet",
		"SELECT 'unclosed":                              "isn't closed",
		"SELECT title FROM books WHERE price = ?":       "1 ? placeholder(s) but 0 value(s)",
		"SELECT count(*) FROM books WHERE count(*) > 1": "can't be used in WHERE",
		"SELECT foo(1)":                                 "no such function: foo",
		"SELECT title FROM books ORDER BY 9":            "ORDER BY 9 is outside",
		"":                                              "the statement is empty",
		"SELECT books.title FROM books b":               "no such column: books.title",
	}
	for q, want := range cases {
		_, _, err := db.Query(q, nil)
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%q: want error containing %q, got %v", q, want, err)
		}
	}
}

func TestNotADatabase(t *testing.T) {
	path := filepath.Join(t.TempDir(), "x.db")
	os.WriteFile(path, []byte("hello, I am not a database file at all......................................................................................."), 0o644)
	if _, err := Open(path); err == nil || !strings.Contains(err.Error(), "isn't a SQLite database") {
		t.Errorf("got %v", err)
	}
	if _, err := Open(filepath.Join(t.TempDir(), "missing.db")); err == nil {
		t.Error("missing file opened")
	}
}

// pageQueries are checked at every page size.
var pageQueries = []string{"SELECT count(*) AS c, sum(n) AS s FROM nums", "SELECT k, length(body) AS len FROM big", "SELECT * FROM books"}

func TestPageSizes(t *testing.T) {
	for _, size := range pageSizes {
		db := openFixture(t, size)
		for _, q := range pageQueries {
			want := sqliteAnswer(t, q)
			got, blobs, err := ourJSON(t, db, q)
			dropColumns(want, blobs)
			sortRows(got)
			sortRows(want)
			if err != nil || !reflect.DeepEqual(got, want) {
				t.Errorf("page size %d, %s: got %v (%v), want %v", size, q, got, err, want)
			}
		}
	}
}
