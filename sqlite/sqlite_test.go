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
	for _, q := range append(append(queries, pageQueries...), dateQueries...) {
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
	if err := os.WriteFile(filepath.Join("testdata", "answers.json"), append(data, '\n'), 0o644); err != nil {
		return err
	}
	return rebuildScripts(bin)
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
CREATE TABLE authors (id INTEGER PRIMARY KEY, name TEXT COLLATE NOCASE UNIQUE, country TEXT);
INSERT INTO authors VALUES (1, 'Frank Herbert', 'US'), (2, 'Gillian Flynn', 'US'), (3, 'Isaac Asimov', 'RU'), (4, 'Nobody', 'FR');
CREATE TABLE wrote (book_id INTEGER, author_id INTEGER);
INSERT INTO wrote VALUES (1, 1), (3, 2), (4, 3), (99, 3), (2, NULL);
CREATE INDEX wrote_author ON wrote (author_id);
CREATE TABLE sales (sku TEXT, region TEXT, qty INTEGER, day TEXT);
INSERT INTO sales VALUES ('B1', 'north', 3, '2026-01-02'), ('B1', 'south', 1, '2026-01-03'), ('B2', 'north', 5, '2026-01-03'),
  ('B3', 'north', NULL, '2026-01-04'), ('B3', 'south', 2, '2026-01-05'), ('B9', 'east', 7, '2026-01-05'), ('b1', 'north', 1, '2026-01-06');
CREATE VIEW cheap AS SELECT title, price FROM books WHERE price < 1000;
CREATE TABLE wr (x TEXT, a TEXT, b INTEGER UNIQUE, c, PRIMARY KEY (a DESC, x)) WITHOUT ROWID;
CREATE INDEX wr_c ON wr (c);
INSERT INTO wr VALUES ('x1', 'a1', 1, 'c1'), ('x2', 'a2', 2, 'c2'), ('x0', 'a1', 3, 'c3'), ('x9', 'a0', NULL, 4.5);
WITH RECURSIVE n(i) AS (SELECT 1 UNION ALL SELECT i + 1 FROM n WHERE i < 400) INSERT INTO wr SELECT 'k' || i, 'big' || (i % 7), NULL, printf('%.*c', i, 'z') FROM n;
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
	// joins
	"SELECT b.title, a.name FROM books b JOIN wrote w ON w.book_id = b.id JOIN authors a ON a.id = w.author_id ORDER BY b.title",
	"SELECT b.title, w.author_id FROM books b LEFT JOIN wrote w ON w.book_id = b.id ORDER BY b.id, w.author_id",
	"SELECT a.name, w.book_id FROM wrote w RIGHT JOIN authors a ON a.id = w.author_id ORDER BY a.id, w.book_id",
	"SELECT b.id AS bid, w.book_id AS wid FROM books b FULL JOIN wrote w ON w.book_id = b.id ORDER BY b.id, w.book_id",
	"SELECT count(*) AS c FROM books CROSS JOIN authors",
	"SELECT title, name FROM books, authors WHERE books.id = authors.id ORDER BY 1",
	"SELECT * FROM sales JOIN books USING (sku) ORDER BY sku, region",
	"SELECT sku, title, qty FROM sales NATURAL JOIN books ORDER BY sku, region",
	"SELECT a.*, w.book_id FROM authors a JOIN wrote w ON a.id = w.author_id ORDER BY w.book_id",
	"SELECT b.sku, w.author_id FROM books b JOIN wrote w ON w.book_id = b.id WHERE w.author_id = 3",
	"SELECT a.name, b.title FROM authors a LEFT JOIN wrote w ON w.author_id = a.id LEFT JOIN books b ON b.id = w.book_id ORDER BY a.id, b.id",
	// groups
	"SELECT sku, sum(qty) AS total, count(*) AS n, count(qty) AS nq FROM sales GROUP BY sku ORDER BY sku",
	"SELECT region, count(*) AS n FROM sales GROUP BY region HAVING count(*) > 1 ORDER BY n DESC, region",
	"SELECT region, max(qty) AS m, sku FROM sales GROUP BY region ORDER BY region",
	"SELECT region, min(day) AS first, sku FROM sales GROUP BY region ORDER BY region",
	"SELECT sku FROM sales GROUP BY 1 HAVING sum(qty) >= 4 ORDER BY 1",
	"SELECT length(region) AS l, count(*) AS c FROM sales GROUP BY l ORDER BY l",
	"SELECT upper(sku) AS s, count(*) AS c FROM sales GROUP BY sku COLLATE NOCASE ORDER BY 1",
	"SELECT a.country, count(DISTINCT w.book_id) AS books FROM authors a LEFT JOIN wrote w ON w.author_id = a.id GROUP BY a.country ORDER BY a.country",
	"SELECT group_concat(sku, '|') AS g FROM (SELECT sku FROM books ORDER BY sku)",
	"SELECT sum(qty) FILTER (WHERE region = 'north') AS north, count(*) FILTER (WHERE qty > 2) AS big FROM sales",
	"SELECT sku, sum(qty) AS t FROM sales GROUP BY sku HAVING t > 2 ORDER BY t DESC",
	"SELECT count(*) AS n FROM books HAVING count(*) > 100",
	"SELECT max(price) AS m, title FROM books",
	"SELECT min(price) AS m, title FROM books",
	"SELECT region, avg(qty) AS a, total(qty) AS t, group_concat(DISTINCT sku) AS skus FROM sales GROUP BY region ORDER BY region",
	"SELECT count(*) AS c FROM sales WHERE qty > 100 GROUP BY region",
	// subqueries
	"SELECT title FROM books WHERE price > (SELECT avg(price) FROM books) ORDER BY title",
	"SELECT title FROM books WHERE id IN (SELECT book_id FROM wrote) ORDER BY id",
	"SELECT title FROM books WHERE id NOT IN (SELECT book_id FROM wrote WHERE book_id IS NOT NULL) ORDER BY id",
	"SELECT name FROM authors a WHERE EXISTS (SELECT 1 FROM wrote w WHERE w.author_id = a.id) ORDER BY name",
	"SELECT name FROM authors a WHERE NOT EXISTS (SELECT 1 FROM wrote w WHERE w.author_id = a.id) ORDER BY name",
	"SELECT name, (SELECT count(*) FROM wrote w WHERE w.author_id = a.id) AS n FROM authors a ORDER BY id",
	"SELECT x.region, x.total FROM (SELECT region, sum(qty) AS total FROM sales GROUP BY region) AS x WHERE x.total > 3 ORDER BY x.region",
	"SELECT title FROM books b WHERE price = (SELECT max(price) FROM books WHERE substr(sku, 1, 1) = substr(b.sku, 1, 1))",
	"SELECT (SELECT title FROM books WHERE price > 99999) AS none_found, (SELECT count(*) FROM books) AS n",
	// compound
	"SELECT sku FROM books UNION SELECT sku FROM sales ORDER BY 1",
	"SELECT sku FROM sales UNION ALL SELECT sku FROM books ORDER BY sku",
	"SELECT sku FROM books INTERSECT SELECT sku FROM sales ORDER BY 1",
	"SELECT sku FROM books EXCEPT SELECT sku FROM sales ORDER BY 1",
	"SELECT 1 AS a UNION SELECT 2 ORDER BY a DESC LIMIT 1",
	"SELECT region FROM sales UNION SELECT country FROM authors",
	// WITH and VALUES
	"WITH t AS (SELECT sku, price FROM books WHERE price > 900) SELECT sku FROM t ORDER BY price",
	"WITH RECURSIVE c(x) AS (SELECT 1 UNION ALL SELECT x + 1 FROM c WHERE x < 10) SELECT sum(x) AS s, count(*) AS n FROM c",
	"WITH RECURSIVE fib(a, b) AS (SELECT 0, 1 UNION ALL SELECT b, a + b FROM fib LIMIT 15) SELECT a FROM fib",
	"WITH RECURSIVE c(x) AS (SELECT 1 UNION SELECT x % 3 + 1 FROM c) SELECT x FROM c ORDER BY x",
	"VALUES (1, 'a'), (2, 'b')",
	"SELECT * FROM (VALUES (1), (2)) AS v",
	// views, collations, GLOB, NULLS
	"SELECT * FROM cheap ORDER BY price",
	"SELECT name FROM authors WHERE name = 'frank herbert'",
	"SELECT name FROM authors ORDER BY name DESC",
	"SELECT title FROM books WHERE title GLOB 'G*' ORDER BY 1",
	"SELECT sku FROM books WHERE sku GLOB '[AB][1-3]' ORDER BY 1",
	"SELECT title FROM books ORDER BY rating DESC NULLS LAST, title",
	"SELECT title FROM books ORDER BY price NULLS LAST",
	"SELECT title FROM books ORDER BY title COLLATE NOCASE",
	"SELECT DISTINCT region FROM sales ORDER BY region",
	"SELECT sku FROM sales WHERE sku = 'b1' COLLATE NOCASE ORDER BY region, sku",
	"SELECT title FROM books WHERE price BETWEEN 900 AND 1300 ORDER BY 1",
	"SELECT 'a' IN (SELECT 'A') AS x, 1 IN () AS y",
	// window functions
	"SELECT sku, region, qty, row_number() OVER (ORDER BY day, sku, region) AS n FROM sales ORDER BY n",
	"SELECT sku, region, qty, rank() OVER (PARTITION BY region ORDER BY qty DESC) AS r, dense_rank() OVER (PARTITION BY region ORDER BY qty DESC) AS d FROM sales ORDER BY region, r, sku",
	"SELECT sku, qty, sum(qty) OVER (ORDER BY day, sku, region ROWS UNBOUNDED PRECEDING) AS running FROM sales ORDER BY day, sku, region",
	"SELECT day, qty, sum(qty) OVER (ORDER BY day) AS by_day, count(*) OVER () AS total FROM sales ORDER BY day, sku, region",
	"SELECT sku, day, lag(qty) OVER w AS prev, lead(qty, 1, -1) OVER w AS next FROM sales WINDOW w AS (PARTITION BY sku ORDER BY day) ORDER BY sku, day",
	"SELECT title, price, ntile(3) OVER (ORDER BY price) AS bucket, percent_rank() OVER (ORDER BY price) AS pr, cume_dist() OVER (ORDER BY price) AS cd FROM books ORDER BY price, title",
	"SELECT n, avg(n) OVER (ORDER BY n ROWS BETWEEN 2 PRECEDING AND 2 FOLLOWING) AS moving FROM nums WHERE n <= 10 ORDER BY n",
	"SELECT n, sum(n) OVER (ORDER BY n RANGE BETWEEN 3 PRECEDING AND 1 FOLLOWING) AS s FROM nums WHERE n <= 10 ORDER BY n",
	"SELECT n, sum(n) OVER (ORDER BY n DESC RANGE BETWEEN 2 PRECEDING AND CURRENT ROW) AS s FROM nums WHERE n <= 8 ORDER BY n",
	"SELECT qty, group_concat(sku) OVER (ORDER BY qty GROUPS BETWEEN 1 PRECEDING AND 1 FOLLOWING) AS g FROM sales WHERE qty IS NOT NULL ORDER BY qty, sku",
	"SELECT sku, qty, first_value(sku) OVER w AS f, last_value(sku) OVER w AS l, nth_value(sku, 2) OVER w AS second FROM sales WINDOW w AS (ORDER BY qty, sku ROWS BETWEEN UNBOUNDED PRECEDING AND UNBOUNDED FOLLOWING) ORDER BY qty, sku",
	"SELECT sku, qty, sum(qty) OVER (ORDER BY qty, sku ROWS BETWEEN 1 PRECEDING AND 1 FOLLOWING EXCLUDE CURRENT ROW) AS around, count(*) OVER (ORDER BY qty RANGE CURRENT ROW EXCLUDE TIES) AS t FROM sales ORDER BY qty, sku",
	"SELECT region, sum(qty) AS q, rank() OVER (ORDER BY sum(qty) DESC) AS r, sum(sum(qty)) OVER () AS all_q FROM sales GROUP BY region ORDER BY r",
	"SELECT sku, sum(qty) FILTER (WHERE region = 'north') OVER (PARTITION BY sku) AS north, max(qty) OVER (PARTITION BY sku) AS m FROM sales ORDER BY sku, region",
	"SELECT * FROM (SELECT sku, day, row_number() OVER (PARTITION BY sku ORDER BY day DESC) AS rn FROM sales) WHERE rn = 1 ORDER BY sku",
	"SELECT name, country, count(*) OVER (PARTITION BY country) AS same FROM authors ORDER BY name",
	"SELECT sku, qty, row_number() OVER (ORDER BY qty DESC NULLS LAST, sku) AS n FROM sales ORDER BY n",
	// WITHOUT ROWID
	"SELECT x, a, b, c FROM wr WHERE a NOT LIKE 'big%'",
	"SELECT count(*) AS n, sum(length(c)) AS s, min(x) AS lo FROM wr",
	"SELECT a, count(*) AS n FROM wr GROUP BY a ORDER BY a",
	"SELECT w.x, b.title FROM wr w JOIN books b ON b.id = w.b ORDER BY w.x",
	// PRAGMA and table-valued functions
	"PRAGMA table_info(authors)",
	"SELECT * FROM pragma_table_info('books')",
	"PRAGMA table_xinfo('aff')",
	"PRAGMA index_list(authors)",
	"SELECT name, \"unique\", origin, partial FROM pragma_index_list('books')",
	"PRAGMA index_info(wrote_author)",
	"SELECT * FROM pragma_index_xinfo('sqlite_autoindex_authors_1')",
	"PRAGMA foreign_key_list(books)",
	"PRAGMA user_version",
	"PRAGMA page_size",
	"PRAGMA encoding",
	"PRAGMA foreign_keys",
	"PRAGMA journal_mode",
	"PRAGMA integrity_check",
	"PRAGMA synchronous",
	"PRAGMA freelist_count",
	"SELECT name, type, ncol FROM pragma_table_list WHERE schema = 'main' ORDER BY name",
	"SELECT value FROM generate_series(1, 10, 3)",
	"SELECT value FROM generate_series(5, 1, -2)",
	"SELECT s.name, p.name AS col, p.type FROM sqlite_schema s, pragma_table_info(s.name) p WHERE s.type = 'table' AND s.name IN ('later', 'wrote') ORDER BY s.name, p.cid",
	"SELECT g.value, b.title FROM generate_series(1, 3) g LEFT JOIN books b ON b.id = g.value ORDER BY g.value",
	// JSON
	"SELECT json(' { \"a\" : [1, 2.50, \"x\"], \"b\": null } ') AS j, json_valid('{\"a\":1}') AS v, json_valid('{a:1}') AS bad",
	"SELECT json_extract('{\"a\": {\"b\": [10, 20, 30]}}', '$.a.b[1]') AS one, json_extract('{\"a\": [1, 2]}', '$.a') AS arr, json_extract('{\"a\": 1, \"b\": \"t\"}', '$.a', '$.b', '$.c') AS many",
	"SELECT '{\"a\": {\"b\": \"hi\"}}' -> '$.a' AS j, '{\"a\": {\"b\": \"hi\"}}' ->> '$.a.b' AS t, '[5, 6, 7]' ->> 2 AS i, '{\"x\": true}' ->> 'x' AS b, '[1, 2, 3]' -> '$[#-1]' AS last",
	"SELECT json_type('{\"a\": [1, 2.5, \"s\", null, true]}', '$.a') AS t1, json_type('[1, 2.5, \"s\", null, true]', '$[1]') AS t2, json_type('[1]', '$[9]') AS t3, json_array_length('[1, 2, 3]') AS n, json_array_length('{\"a\": [1]}', '$.a') AS m",
	"SELECT json_object('name', title, 'price', price, 'tags', json_array('a', 1, NULL)) AS o FROM books WHERE id <= 2 ORDER BY id",
	"SELECT json_array(1, 2.5, 'x', NULL, json('{\"k\":1}'), '{\"k\":1}') AS a, json_quote('say \"hi\"') AS q, json_quote(3.0) AS r",
	"SELECT json_set('{\"a\": 1}', '$.b', 2, '$.a', 10) AS s, json_insert('{\"a\": 1}', '$.a', 9, '$.c.d', 4) AS i, json_replace('{\"a\": 1}', '$.a', 'x', '$.z', 1) AS r",
	"SELECT json_set('[1, 2]', '$[#]', 3) AS app, json_remove('[1, 2, 3, 4]', '$[1]', '$[#-1]') AS rem, json_remove('{\"a\": 1, \"b\": 2}', '$.a') AS remo",
	"SELECT json_patch('{\"a\": 1, \"b\": {\"c\": 2, \"d\": 3}}', '{\"a\": null, \"b\": {\"c\": 9}, \"e\": [1]}') AS p",
	"SELECT region, json_group_array(sku) AS skus, json_group_object(sku, qty) AS q FROM sales GROUP BY region ORDER BY region",
	"SELECT json_group_object(qty, sku) AS by_qty, json_group_object(qty * 1.5, region) AS by_real FROM sales",
	"SELECT key, value, type, atom, fullkey, path FROM json_each('{\"a\": 1, \"b\": [2, 3], \"c d\": {\"e\": null}}')",
	"SELECT key, value, type, fullkey, path FROM json_each('[10, \"x\", true]')",
	"SELECT key, value, type, atom, fullkey, path FROM json_tree('{\"a\": 1, \"b\": [2, {\"c\": 3}]}')",
	"SELECT key, value FROM json_each('{\"a\": {\"x\": 1, \"y\": 2}}', '$.a')",
	"SELECT b.title, j.value AS tag FROM books b, json_each(json_array(b.sku, b.id)) j WHERE b.id <= 2 ORDER BY b.id, j.key",
	"SELECT json_group_array(json_object('sku', sku)) OVER (ORDER BY id ROWS 1 PRECEDING) AS w FROM books ORDER BY id",
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

func TestFunctionsMatchSQLite(t *testing.T) {
	db := openFixture(t, 4096)
	for _, q := range dateQueries {
		want := sqliteAnswer(t, q)
		got, blobs, err := ourJSON(t, db, q)
		if err != nil {
			t.Errorf("%s\n  error: %v", q, err)
			continue
		}
		dropColumns(want, blobs)
		if !reflect.DeepEqual(got, want) {
			t.Errorf("%s\n  got:  %v\n  want: %v", q, got, want)
		}
	}
}

func TestTablesAndTypes(t *testing.T) {
	db := openFixture(t, 4096)
	if got := strings.Join(db.Tables(), ","); got != "books,odd name,nums,big,later,aff,authors,wrote,sales,wr" {
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
		"SELECT * FROM nope":                             "no such table: nope",
		"SELECT nope FROM books":                         "no such column: nope",
		"SELECT title FROM books WHERE":                  "the statement ends too soon",
		"SELECT title FROM books WHERE price ?":          "unexpected",
		"INSERT INTO books VALUES (1)":                   "run it with sql_run",
		"SELECT title FROM books GROUP BY count(*)":      "can't be used in GROUP BY",
		"SELECT id FROM books, authors":                  "ambiguous column name: id",
		"SELECT 1 UNION SELECT 1, 2":                     "do not have the same number of result columns",
		"SELECT row_number() FROM books":                 "misuse of window function row_number()",
		"SELECT sku FROM books WHERE rank() OVER () > 1": "can't be used in WHERE",
		"SELECT sum(price) OVER nope FROM books":         "no such window: nope",
		"SELECT ntile(0) OVER () FROM books":             "argument of ntile must be a positive integer",
		"SELECT x FROM (SELECT 1 AS y)":                  "no such column: x",
		"SELECT (SELECT 1, 2)":                           "sub-select returns 2 columns",
		"SELECT * FROM cheap JOIN nope":                  "no such table: nope",
		"SELECT * FROM books JOIN authors USING (nope)":  "cannot join using column nope",
		"SELECT 'unclosed":                               "isn't closed",
		"SELECT title FROM books WHERE price = ?":        "1 ? placeholder(s) but 0 value(s)",
		"SELECT count(*) FROM books WHERE count(*) > 1":  "can't be used in WHERE",
		"SELECT foo(1)":                                  "no such function: foo",
		"SELECT title FROM books ORDER BY 9":             "ORDER BY 9 is outside",
		"":                                               "the statement is empty",
		"SELECT books.title FROM books b":                "no such column: books.title",
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

// dateQueries are compared with sqlite3 like queries; they avoid 'now'.
var dateQueries = []string{
	"SELECT date('2026-01-31', '+1 month') AS a, date('2026-01-31', '+1 day') AS b, datetime('2026-10-05 14:05:09') AS c",
	"SELECT datetime('2026-10-05T14:05:09.123Z') AS a, time('2026-10-05 14:05', '+90 minutes') AS b, date('2026-03-15', 'start of month') AS c",
	"SELECT date('2026-10-05', 'start of year') AS a, date('2026-10-05', 'weekday 0') AS b, date('2026-10-05', '-1 year') AS c",
	"SELECT julianday('2026-10-05') AS a, unixepoch('2026-10-05 00:00:00') AS b, datetime(1791209100, 'unixepoch') AS c",
	"SELECT strftime('%Y/%m/%d %H:%M:%S %j %w %u', '2026-10-05 14:05:09') AS a, strftime('%s', '2026-01-01') AS b",
	"SELECT date('2026-10-05 23:30:00+02:00') AS a, datetime('2026-10-05 12:00', '+1.5 days') AS b, date('nonsense') AS c",
	"SELECT datetime(2461318.5) AS a, date(1791209100, 'auto') AS b, time('12:34:56.789', 'subsec') AS c",
	"SELECT strftime('%W %U %V %G', '2026-01-01') AS a, strftime('%W %U %V', '2026-12-31') AS b, strftime('%f', '2026-10-05 01:02:03.456') AS c",
	"SELECT date('2024-02-29', '+1 year') AS a, date('2026-01-31', '+13 months') AS b, date('2026-05-31', '-3 months') AS c",
	"SELECT printf('%d|%5d|%-5d|%05d|%,d', 42, 42, 42, 42, 1234567) AS a, printf('%.2f|%8.3f|%e', 3.14159, 2.5, 12345.678) AS b",
	"SELECT printf('%s|%10s|%-10s|%.3s', 'abc', 'abc', 'abc', 'abcdef') AS a, printf('%q|%Q|%Q|%x|%X|%o', 'it''s', 'x', NULL, 255, 255, 8) AS b",
	"SELECT format('%d%%', 50) AS a, quote('it''s') AS b, quote(NULL) AS c, quote(12) AS d, quote(1.5) AS e, quote(x'00ff') AS f",
	"SELECT char(72, 105) AS a, unicode('é') AS b, sign(-3) AS c, sign(0.0) AS d, concat('a', NULL, 1) AS e, concat_ws('-', 'a', NULL, 'b') AS f",
	"SELECT octet_length('é') AS a, hex(unhex('CAFE')) AS b, hex(zeroblob(2)) AS c, likely(5) AS d, glob('a*', 'abc') AS e, like('A%', 'abc') AS f",
	"SELECT floor(2.7) AS a, ceil(2.1) AS b, trunc(-2.7) AS c, round(sqrt(16)) AS d, pow(2, 10) AS e, log(100) AS f, log(2, 8) AS g, mod(7, 3) AS h",
	"SELECT floor(5) AS a, typeof(floor(5)) AS b, ln(0) AS c, round(pi(), 4) AS d, round(degrees(pi())) AS e, exp(0) AS f",
}
