// Copyright 2017-2026 Nigele McCoy
// Licensed under the Apache License, Version 2.0;
// see LICENSE and NOTICE.

package sqlite

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// writeScripts are run statement by statement by both sqlite3 (when
// rebuilding testdata) and this package, from an empty database. After
// each script, every table's rows, their storage types, and the schema
// must match, and the same statements must have failed. Each statement
// is one line.
var writeScripts = map[string][]string{
	"update from": {
		"CREATE TABLE stock (sku TEXT PRIMARY KEY, qty INTEGER, price INTEGER, note TEXT)",
		"CREATE TABLE delivery (sku TEXT, n INTEGER, cost INTEGER)",
		"INSERT INTO stock VALUES ('a', 1, 10, NULL), ('b', 2, 20, NULL), ('c', 3, 30, NULL)",
		"INSERT INTO delivery VALUES ('a', 5, 11), ('c', 1, 33), ('x', 9, 99)",
		"UPDATE stock SET qty = qty + d.n, price = d.cost FROM delivery AS d WHERE d.sku = stock.sku",
		"UPDATE stock AS s SET note = 'total ' || t.n FROM (SELECT sku, sum(n) AS n FROM delivery GROUP BY sku) AS t WHERE t.sku = s.sku AND t.n > 1",
		"CREATE TABLE prices (sku TEXT, region TEXT, p INTEGER)",
		"INSERT INTO prices VALUES ('b', 'eu', 21), ('b', 'us', 22)",
		"UPDATE stock SET price = prices.p FROM prices JOIN delivery ON 1 WHERE prices.sku = stock.sku AND prices.region = 'us' AND delivery.sku = 'x'",
		"UPDATE stock SET qty = 0 FROM delivery WHERE delivery.sku = 'none'",
		"UPDATE stock SET qty = nope FROM delivery",
	},
	"generated and expression indexes": {
		"CREATE TABLE line (id INTEGER PRIMARY KEY, price REAL, qty INTEGER, total REAL AS (price * qty) STORED, label TEXT GENERATED ALWAYS AS (upper(name) || ' x' || qty) VIRTUAL, name TEXT)",
		"INSERT INTO line (price, qty, name) VALUES (1.5, 4, 'pen'), (8, 2, 'mug')",
		"INSERT INTO line (price, qty, total) VALUES (1, 1, 99)",
		"UPDATE line SET qty = qty + 1 WHERE name = 'pen'",
		"UPDATE line SET total = 0",
		"CREATE INDEX line_label ON line (label)",
		"CREATE TABLE users (id INTEGER PRIMARY KEY, email TEXT)",
		"CREATE UNIQUE INDEX users_email ON users (lower(email))",
		"CREATE INDEX users_len ON users (length(email) DESC, email COLLATE NOCASE)",
		"INSERT INTO users (email) VALUES ('Ann@x.com'), ('bo@y.org')",
		"INSERT INTO users (email) VALUES ('ANN@X.COM')",
		"UPDATE users SET email = 'BO@Y.ORG' WHERE id = 2",
		"INSERT INTO users (email) VALUES ('cy@z.net')",
		"DELETE FROM users WHERE id = 1",
		"INSERT INTO users (email) VALUES ('ann@x.com')",
		"ALTER TABLE line ADD COLUMN note TEXT DEFAULT 'none'",
		"INSERT INTO line (price, qty, name) VALUES (3, 3, 'cup')",
	},
	"savepoints": {
		"CREATE TABLE s (id INTEGER PRIMARY KEY, v TEXT)",
		"BEGIN",
		"INSERT INTO s (v) VALUES ('a')",
		"SAVEPOINT one",
		"INSERT INTO s (v) VALUES ('b')",
		"SAVEPOINT two",
		"INSERT INTO s (v) VALUES ('c')",
		"ROLLBACK TO two",
		"INSERT INTO s (v) VALUES ('d')",
		"RELEASE one",
		"ROLLBACK TO one",
		"COMMIT",
		"SAVEPOINT outer_sp",
		"INSERT INTO s (v) VALUES ('e')",
		"SAVEPOINT inner_sp",
		"CREATE TABLE gone (x)",
		"INSERT INTO s (v) VALUES ('f')",
		"ROLLBACK TO SAVEPOINT inner_sp",
		"RELEASE outer_sp",
		"RELEASE outer_sp",
		"SAVEPOINT x",
		"INSERT INTO s (v) VALUES ('g')",
		"ROLLBACK",
	},
	"triggers": {
		"CREATE TABLE books (sku TEXT PRIMARY KEY, title TEXT, stock INTEGER, updated TEXT)",
		"CREATE TABLE log (id INTEGER PRIMARY KEY, what TEXT, sku TEXT, n INTEGER)",
		"CREATE TRIGGER b_ins AFTER INSERT ON books BEGIN INSERT INTO log (what, sku, n) VALUES ('add', NEW.sku, NEW.stock); END",
		"CREATE TRIGGER b_upd AFTER UPDATE OF stock ON books WHEN NEW.stock < OLD.stock BEGIN INSERT INTO log (what, sku, n) VALUES ('sold', NEW.sku, OLD.stock - NEW.stock); END",
		"CREATE TRIGGER b_del BEFORE DELETE ON books BEGIN INSERT INTO log (what, sku) VALUES ('gone', OLD.sku); END",
		"CREATE TRIGGER b_chk BEFORE INSERT ON books WHEN NEW.stock < 0 BEGIN SELECT RAISE(ABORT, 'stock cannot be negative'); END",
		"CREATE TRIGGER b_skip BEFORE INSERT ON books WHEN NEW.sku LIKE 'X%' BEGIN SELECT RAISE(IGNORE); END",
		"INSERT INTO books VALUES ('B1', 'Dune', 5, NULL), ('B2', 'Emma', 2, NULL)",
		"INSERT INTO books VALUES ('B3', 'Bad', -1, NULL)",
		"INSERT INTO books VALUES ('X1', 'Skipped', 1, NULL), ('B4', 'Kept', 1, NULL)",
		"UPDATE books SET stock = stock - 1",
		"UPDATE books SET title = upper(title)",
		"UPDATE books SET stock = stock + 10 WHERE sku = 'B1'",
		"DELETE FROM books WHERE sku = 'B2'",
		"CREATE VIEW stock_view AS SELECT sku, title, stock FROM books",
		"CREATE TRIGGER v_ins INSTEAD OF INSERT ON stock_view BEGIN INSERT INTO books (sku, title, stock) VALUES (NEW.sku, NEW.title, coalesce(NEW.stock, 0)); END",
		"CREATE TRIGGER v_upd INSTEAD OF UPDATE ON stock_view BEGIN UPDATE books SET stock = NEW.stock WHERE sku = OLD.sku; END",
		"CREATE TRIGGER v_del INSTEAD OF DELETE ON stock_view BEGIN DELETE FROM books WHERE sku = OLD.sku; END",
		"INSERT INTO stock_view (sku, title) VALUES ('B5', 'Via view')",
		"UPDATE stock_view SET stock = 99 WHERE sku = 'B5'",
		"DELETE FROM stock_view WHERE sku = 'B4'",
		"CREATE TRIGGER touch AFTER UPDATE ON books BEGIN UPDATE books SET updated = 'yes' WHERE sku = NEW.sku; END",
		"UPDATE books SET stock = 3 WHERE sku = 'B1'",
		"CREATE TRIGGER stopper BEFORE UPDATE ON books WHEN NEW.title = 'STOP' BEGIN SELECT RAISE(FAIL, 'stopped'); END",
		"UPDATE books SET title = 'STOP' WHERE sku = 'B1'",
		"ALTER TABLE books RENAME TO items",
		"INSERT INTO items VALUES ('B7', 'After rename', 4, NULL)",
		"DROP TRIGGER b_del",
		"DELETE FROM items WHERE sku = 'B7'",
		"CREATE TRIGGER bad INSTEAD OF INSERT ON items BEGIN SELECT 1; END",
		"CREATE TRIGGER nowhere AFTER INSERT ON missing BEGIN SELECT 1; END",
	},
	"foreign keys": {
		"PRAGMA foreign_keys = ON",
		"CREATE TABLE author (id INTEGER PRIMARY KEY, name TEXT UNIQUE)",
		"CREATE TABLE book (id INTEGER PRIMARY KEY, author_id INTEGER REFERENCES author (id) ON DELETE CASCADE ON UPDATE CASCADE, title TEXT)",
		"CREATE TABLE review (id INTEGER PRIMARY KEY, book_id INTEGER REFERENCES book (id) ON DELETE SET NULL, stars INTEGER)",
		"CREATE TABLE loan (id INTEGER PRIMARY KEY, book_id INTEGER REFERENCES book (id) ON DELETE RESTRICT)",
		"CREATE TABLE tag (book_id INTEGER REFERENCES book (id), tag TEXT)",
		"INSERT INTO author VALUES (1, 'Ann'), (2, 'Bo')",
		"INSERT INTO book VALUES (10, 1, 'A1'), (11, 1, 'A2'), (20, 2, 'B1')",
		"INSERT INTO book VALUES (30, 99, 'orphan')",
		"INSERT INTO review VALUES (1, 10, 5), (2, 11, 4), (3, 20, 3)",
		"INSERT INTO loan VALUES (1, 20)",
		"INSERT INTO tag VALUES (11, 'x')",
		"DELETE FROM author WHERE id = 2",
		"DELETE FROM loan",
		"DELETE FROM author WHERE id = 2",
		"DELETE FROM book WHERE id = 11",
		"UPDATE author SET id = 5 WHERE id = 1",
		"DELETE FROM tag",
		"DELETE FROM book WHERE id = 11",
		"INSERT INTO tag VALUES (NULL, 'free')",
		"CREATE TABLE emp (id INTEGER PRIMARY KEY, boss INTEGER REFERENCES emp (id))",
		"INSERT INTO emp VALUES (2, 1), (1, NULL)",
		"UPDATE emp SET boss = 7 WHERE id = 2",
		"DELETE FROM emp WHERE id = 1",
		"CREATE TABLE d (x REFERENCES author (id) DEFERRABLE INITIALLY DEFERRED)",
		"BEGIN",
		"INSERT INTO d VALUES (77)",
		"INSERT INTO author VALUES (77, 'late')",
		"COMMIT",
		"BEGIN",
		"INSERT INTO d VALUES (88)",
		"COMMIT",
		"ROLLBACK",
		"INSERT INTO d VALUES (99)",
		"CREATE TABLE code (c TEXT PRIMARY KEY)",
		"CREATE TABLE uses (c TEXT REFERENCES code ON UPDATE SET NULL)",
		"INSERT INTO code VALUES ('a'), ('b')",
		"INSERT INTO uses VALUES ('a'), ('b'), ('a')",
		"UPDATE code SET c = 'z' WHERE c = 'a'",
		"DROP TABLE loan",
		"CREATE TABLE badref (x REFERENCES author (nope))",
		"INSERT INTO badref VALUES (1)",
		"PRAGMA user_version = 42",
	},
	"basics": {
		"CREATE TABLE t (id INTEGER PRIMARY KEY, name TEXT NOT NULL, qty INTEGER DEFAULT 1, price REAL, note)",
		"INSERT INTO t (name, price) VALUES ('pen', 1.5), ('pad', 4), ('mug', '8.25')",
		"INSERT INTO t VALUES (10, 'cup', '3', 2, 'x')",
		"INSERT INTO t (name) VALUES (NULL)",
		"INSERT INTO t (id, name) VALUES (10, 'dup')",
		"INSERT INTO t (name, qty) VALUES ('box', 'many')",
		"INSERT INTO t (name, qty, price) VALUES ('big', 1e20, 7)",
		"INSERT INTO t (name, qty) VALUES ('three', 3.0)",
		"UPDATE t SET qty = qty + 1 WHERE price > 2",
		"UPDATE t SET id = 20 WHERE name = 'cup'",
		"DELETE FROM t WHERE name = 'pad'",
		"INSERT INTO t (name) VALUES ('after')",
		"INSERT INTO t DEFAULT VALUES",
	},
	"constraints": {
		"CREATE TABLE u (a TEXT UNIQUE, b INTEGER CHECK (b > 0), c TEXT COLLATE NOCASE, UNIQUE (b, c))",
		"INSERT INTO u VALUES ('x', 1, 'A')",
		"INSERT INTO u VALUES ('x', 2, 'B')",
		"INSERT INTO u VALUES ('y', 0, 'B')",
		"INSERT INTO u VALUES ('y', 1, 'a')",
		"INSERT INTO u VALUES (NULL, 2, 'z'), (NULL, 3, 'z')",
		"INSERT OR IGNORE INTO u VALUES ('x', 5, 'q')",
		"INSERT OR REPLACE INTO u VALUES ('x', 6, 'r')",
		"REPLACE INTO u VALUES ('w', 2, 'Z')",
		"UPDATE u SET a = 'w' WHERE b = 6",
		"UPDATE OR REPLACE u SET a = 'w' WHERE b = 6",
		"UPDATE OR IGNORE u SET b = -1",
		"INSERT INTO u VALUES ('m', 9, 'm'), ('m', 10, 'n')",
	},
	"upsert": {
		"CREATE TABLE stock (sku TEXT PRIMARY KEY, qty INTEGER NOT NULL DEFAULT 0, seen INTEGER DEFAULT 0)",
		"INSERT INTO stock (sku, qty) VALUES ('B1', 5), ('B2', 1)",
		"INSERT INTO stock (sku, qty) VALUES ('B1', 3) ON CONFLICT (sku) DO UPDATE SET qty = qty + excluded.qty, seen = seen + 1",
		"INSERT INTO stock (sku, qty) VALUES ('B3', 7) ON CONFLICT (sku) DO UPDATE SET qty = qty + excluded.qty",
		"INSERT INTO stock (sku, qty) VALUES ('B2', 9) ON CONFLICT DO NOTHING",
		"INSERT INTO stock (sku, qty) VALUES ('B2', 4) ON CONFLICT (sku) DO UPDATE SET qty = excluded.qty WHERE excluded.qty > qty",
		"INSERT INTO stock (sku, qty) VALUES ('B2', 2) ON CONFLICT (sku) DO UPDATE SET qty = excluded.qty WHERE excluded.qty > qty",
		"INSERT INTO stock (sku, qty) VALUES ('B4', 1) ON CONFLICT (qty) DO NOTHING",
		"INSERT INTO stock (sku, qty) SELECT sku || 'x', qty FROM stock WHERE true ON CONFLICT DO NOTHING",
	},
	"autoincrement": {
		"CREATE TABLE a (id INTEGER PRIMARY KEY AUTOINCREMENT, v TEXT)",
		"INSERT INTO a (v) VALUES ('one'), ('two'), ('three')",
		"DELETE FROM a WHERE id = 3",
		"INSERT INTO a (v) VALUES ('four')",
		"INSERT INTO a (id, v) VALUES (100, 'jump')",
		"DELETE FROM a",
		"INSERT INTO a (v) VALUES ('after clear')",
		"CREATE TABLE plain (id INTEGER PRIMARY KEY, v TEXT)",
		"INSERT INTO plain (v) VALUES ('one'), ('two')",
		"DELETE FROM plain WHERE id = 2",
		"INSERT INTO plain (v) VALUES ('reused')",
	},
	"alter": {
		"CREATE TABLE b (id INTEGER PRIMARY KEY, title TEXT, price INTEGER)",
		"CREATE INDEX b_price ON b (price)",
		"INSERT INTO b (title, price) VALUES ('Dune', 950), ('Emma', 700)",
		"ALTER TABLE b ADD COLUMN rating REAL DEFAULT 2.5",
		"ALTER TABLE b ADD COLUMN genre TEXT NOT NULL DEFAULT 'none'",
		"ALTER TABLE b ADD COLUMN bad TEXT NOT NULL",
		"INSERT INTO b (title, price, rating) VALUES ('Ulysses', 1200, 4)",
		"UPDATE b SET genre = 'sf' WHERE title = 'Dune'",
		"ALTER TABLE b RENAME COLUMN price TO cost",
		"CREATE VIEW cheap AS SELECT b.title, x.n FROM b JOIN (SELECT count(*) AS n FROM b) AS x WHERE b.cost < 900",
		"ALTER TABLE b RENAME TO books",
		"ALTER TABLE books DROP COLUMN rating",
		"ALTER TABLE books DROP COLUMN cost",
		"INSERT INTO books (title, cost, genre) VALUES ('Beloved', 800, 'lit')",
	},
	"schema": {
		"CREATE TABLE p (id INTEGER PRIMARY KEY, name TEXT, team TEXT)",
		"INSERT INTO p (name, team) VALUES ('ann', 'red'), ('bo', 'blue'), ('cy', 'red'), ('di', NULL)",
		"CREATE UNIQUE INDEX p_name ON p (name)",
		"CREATE UNIQUE INDEX p_team ON p (team)",
		"CREATE INDEX p_team2 ON p (team DESC) WHERE team IS NOT NULL",
		"CREATE INDEX IF NOT EXISTS p_team2 ON p (team)",
		"CREATE VIEW reds AS SELECT name FROM p WHERE team = 'red'",
		"CREATE VIEW early AS SELECT * FROM not_yet",
		"ALTER TABLE p RENAME TO people",
		"CREATE TABLE not_yet (x)",
		"CREATE TABLE IF NOT EXISTS p (x)",
		"CREATE TABLE p (x)",
		"CREATE TABLE teams AS SELECT team, count(*) AS n, max(id) AS last FROM p GROUP BY team",
		"DROP INDEX p_team2",
		"DROP VIEW reds",
		"DROP TABLE IF EXISTS missing",
		"DROP TABLE missing",
		"CREATE TABLE sqlite_mine (x)",
		"INSERT INTO p (name, team) VALUES ('ann', 'x')",
	},
	"transactions": {
		"CREATE TABLE acct (id INTEGER PRIMARY KEY, balance INTEGER CHECK (balance >= 0))",
		"INSERT INTO acct VALUES (1, 100), (2, 50)",
		"BEGIN",
		"UPDATE acct SET balance = balance - 70 WHERE id = 1",
		"UPDATE acct SET balance = balance + 70 WHERE id = 2",
		"ROLLBACK",
		"BEGIN",
		"UPDATE acct SET balance = balance - 30 WHERE id = 1",
		"UPDATE acct SET balance = balance - 80 WHERE id = 2",
		"UPDATE acct SET balance = balance + 30 WHERE id = 2",
		"COMMIT",
		"COMMIT",
		"INSERT INTO acct VALUES (3, 5), (4, -1)",
	},
	"bigvalues": {
		"CREATE TABLE big (id INTEGER PRIMARY KEY, body TEXT, blob BLOB)",
		"CREATE INDEX big_body ON big (body)",
		"INSERT INTO big (body) VALUES (printf('%.*c', 5000, 'a')), (printf('%.*c', 70000, 'b')), ('small')",
		"INSERT INTO big (body, blob) SELECT body || 'x', zeroblob(9000) FROM big",
		"UPDATE big SET body = printf('%.*c', 300, 'c') WHERE id = 2",
		"DELETE FROM big WHERE id IN (1, 4)",
		"INSERT INTO big (body) VALUES (printf('%.*c', 2000, 'd'))",
	},
	"strict": {
		"CREATE TABLE s (i INTEGER, r REAL, t TEXT, b BLOB, a ANY) STRICT",
		"INSERT INTO s VALUES (1, 2, 'x', x'00', 'any')",
		"INSERT INTO s VALUES ('5', '2.5', 3, NULL, 4)",
		"INSERT INTO s VALUES ('five', 1, 'y', NULL, NULL)",
		"INSERT INTO s VALUES (1, 'nope', 'y', NULL, NULL)",
		"INSERT INTO s VALUES (1, 1, 'y', 'text', NULL)",
	},
	"many rows": {
		"CREATE TABLE n (k INTEGER PRIMARY KEY, sq INTEGER, word TEXT)",
		"CREATE INDEX n_word ON n (word)",
		"WITH RECURSIVE c(x) AS (SELECT 1 UNION ALL SELECT x + 1 FROM c WHERE x < 3000) INSERT INTO n SELECT x, x * x, 'w' || (x % 97) FROM c",
		"INSERT INTO n SELECT k + 3000, sq, word FROM n WHERE k % 2 = 0",
		"DELETE FROM n WHERE k % 3 = 0",
		"UPDATE n SET word = 'odd' WHERE k % 5 = 1",
		"DELETE FROM n WHERE word = 'w7'",
		"INSERT INTO n (sq, word) SELECT count(*), 'count' FROM n",
	},
}

type scriptAnswer struct {
	Errors  map[string]string `json:"errors"`
	Queries []string          `json:"queries"`
	Rows    []json.RawMessage `json:"rows"`
}

func scriptsPath() string { return filepath.Join("testdata", "writes.json") }

var errLine = regexp.MustCompile(`(?i)error near line (\d+): (.*)`)

// rebuildScripts runs every script through sqlite3 and records the
// results.
func rebuildScripts(bin string) error {
	out := map[string]scriptAnswer{}
	for name, script := range writeScripts {
		dir, err := os.MkdirTemp("", "scripts")
		if err != nil {
			return err
		}
		path := filepath.Join(dir, "s.db")
		// Apple's sqlite3 turns on legacy_alter_table, which leaves views
		// naming a renamed table broken; standard SQLite rewrites them.
		cmd := exec.Command(bin, "-cmd", "PRAGMA legacy_alter_table=OFF", path)
		cmd.Stdin = strings.NewReader(strings.Join(script, ";\n") + ";\n")
		text, _ := cmd.CombinedOutput()
		ans := scriptAnswer{Errors: map[string]string{}}
		for _, line := range strings.Split(string(text), "\n") {
			if m := errLine.FindStringSubmatch(line); m != nil {
				n, _ := strconv.Atoi(m[1])
				msg := regexp.MustCompile(` \(\d+\)$`).ReplaceAllString(m[2], "")
				ans.Errors[strconv.Itoa(n-1)] = msg
			}
		}
		// Every table's rows and types, and the schema.
		tables, err := exec.Command(bin, path, "SELECT name FROM sqlite_schema WHERE type = 'table' ORDER BY name").Output()
		if err != nil {
			return err
		}
		ans.Queries = append(ans.Queries, "SELECT type, name, tbl_name, sql FROM sqlite_schema ORDER BY name")
		for _, t := range strings.Fields(strings.ReplaceAll(strings.TrimSpace(string(tables)), " ", "\x00")) {
			t = strings.ReplaceAll(t, "\x00", " ")
			cols, err := exec.Command(bin, path, fmt.Sprintf("SELECT name FROM pragma_table_info('%s')", t)).Output()
			if err != nil {
				return err
			}
			q := `"` + t + `"`
			var types []string
			for _, c := range strings.Split(strings.TrimSpace(string(cols)), "\n") {
				types = append(types, fmt.Sprintf(`typeof("%s") AS "%s"`, c, c))
			}
			ans.Queries = append(ans.Queries,
				"SELECT rowid AS _r, * FROM "+q+" ORDER BY rowid",
				"SELECT rowid AS _r, "+strings.Join(types, ", ")+" FROM "+q+" ORDER BY rowid")
		}
		for _, q := range ans.Queries {
			res, err := exec.Command(bin, "-json", path, q).CombinedOutput()
			if err != nil {
				return fmt.Errorf("%s: %v\n%s", q, err, res)
			}
			body := strings.TrimSpace(string(res))
			if body == "" {
				body = "[]"
			}
			ans.Rows = append(ans.Rows, json.RawMessage(body))
		}
		os.RemoveAll(dir)
		out[name] = ans
	}
	data, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(scriptsPath(), append(data, '\n'), 0o644)
}

func TestScriptsMatchSQLite(t *testing.T) {
	syncFiles = false
	defer func() { syncFiles = true }()
	data, err := os.ReadFile(scriptsPath())
	if err != nil {
		t.Fatal(err)
	}
	var answers map[string]scriptAnswer
	if err := json.Unmarshal(data, &answers); err != nil {
		t.Fatal(err)
	}
	for name, script := range writeScripts {
		t.Run(name, func(t *testing.T) {
			ans, ok := answers[name]
			if !ok {
				t.Fatalf("no answer for %q; run go test ./sqlite -update", name)
			}
			db, path := createTest(t, 4096)
			for i, stmt := range script {
				_, err := db.Exec(stmt, nil)
				want, shouldFail := ans.Errors[strconv.Itoa(i)]
				switch {
				case shouldFail && err == nil:
					t.Errorf("statement %d %q: want error %q, got none", i, stmt, want)
				case !shouldFail && err != nil:
					t.Errorf("statement %d %q: %v", i, stmt, err)
				case shouldFail && strings.Contains(want, "constraint failed") && !strings.Contains(want, err.Error()):
					t.Errorf("statement %d %q: error %q, sqlite3 says %q", i, stmt, err, want)
				}
			}
			for i, q := range ans.Queries {
				var want []map[string]any
				if err := json.Unmarshal(ans.Rows[i], &want); err != nil {
					t.Fatal(err)
				}
				if want == nil {
					want = []map[string]any{}
				}
				got, blobs, err := ourJSON(t, db, q)
				if err != nil {
					t.Errorf("%s: %v", q, err)
					continue
				}
				dropColumns(want, blobs)
				if !reflect.DeepEqual(got, want) {
					t.Errorf("%s\n  got:  %v\n  want: %v", q, clip(got), clip(want))
				}
			}
			mustCheck(t, db)
			sqlite3Check(t, path)
		})
	}
}

// clip shortens long rows in failure messages.
func clip(rows []map[string]any) string {
	s := fmt.Sprint(rows)
	if len(s) > 600 {
		return s[:600] + "..."
	}
	return s
}
