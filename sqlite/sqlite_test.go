package sqlite

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// The sqlite3 tool is only used here, to build test files and to check
// answers against the real SQLite. The package itself never needs it.
func needSQLite3(t *testing.T) string {
	t.Helper()
	path, err := exec.LookPath("sqlite3")
	if err != nil {
		t.Skip("sqlite3 tool not installed")
	}
	// -json output (used to compare answers) needs sqlite3 3.33 or later.
	if out, err := exec.Command(path, "-json", ":memory:", "SELECT 1 AS x").Output(); err != nil || !strings.Contains(string(out), `"x"`) {
		t.Skip("sqlite3 tool is too old for -json")
	}
	return path
}

func buildDB(t *testing.T, script string) string {
	t.Helper()
	bin := needSQLite3(t)
	path := filepath.Join(t.TempDir(), "test.db")
	cmd := exec.Command(bin, path)
	cmd.Stdin = strings.NewReader(script)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("sqlite3: %v\n%s", err, out)
	}
	return path
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

// sqliteJSON runs a query through sqlite3 -json and returns the rows as
// generic JSON values.
func sqliteJSON(t *testing.T, path, query string) []map[string]any {
	t.Helper()
	cmd := exec.Command(needSQLite3(t), "-json", path, query)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("sqlite3 %q: %v\n%s", query, err, out)
	}
	var rows []map[string]any
	if strings.TrimSpace(string(out)) == "" {
		return []map[string]any{}
	}
	if err := json.Unmarshal(out, &rows); err != nil {
		t.Fatalf("sqlite3 output for %q isn't JSON: %v\n%s", query, err, out)
	}
	return rows
}

// ourJSON runs a query through this package and shapes the rows like
// sqlite3 -json does. Blob columns are left out, and named in the
// returned set so the caller drops them from sqlite3's rows too: how the
// tool writes blob bytes in JSON changed between versions (3.45 and
// older write byte 0xCA as "\uffffffca"), so blobs are checked through
// hex() in queries and exact bytes in TestTablesAndTypes instead.
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
	path := buildDB(t, fixture)
	db, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	for _, q := range queries {
		want := sqliteJSON(t, path, q)
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
	db, err := Open(buildDB(t, fixture))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
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
	db, err := Open(buildDB(t, fixture))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
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
	db, err := Open(buildDB(t, fixture))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
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

func TestPageSizes(t *testing.T) {
	for _, size := range []int{512, 1024, 4096, 65536} {
		path := buildDB(t, "PRAGMA page_size="+strconv.Itoa(size)+";\n"+fixture)
		db, err := Open(path)
		if err != nil {
			t.Fatalf("page size %d: %v", size, err)
		}
		for _, q := range []string{"SELECT count(*) AS c, sum(n) AS s FROM nums", "SELECT k, length(body) AS len FROM big", "SELECT * FROM books"} {
			want := sqliteJSON(t, path, q)
			got, blobs, err := ourJSON(t, db, q)
			dropColumns(want, blobs)
			if err != nil || !reflect.DeepEqual(got, want) {
				t.Errorf("page size %d, %s: got %v (%v), want %v", size, q, got, err, want)
			}
		}
		db.Close()
	}
}
