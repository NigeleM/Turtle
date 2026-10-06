package postgres

import (
	"math"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"
)

// These tests need a server: TURTLE_PG_URL=postgres://user:pw@host:port/db
// (CI starts one; locally, docker run postgres). Without it they skip.
func connect(t *testing.T) *DB {
	t.Helper()
	dsn := os.Getenv("TURTLE_PG_URL")
	if dsn == "" {
		t.Skip("TURTLE_PG_URL isn't set")
	}
	db, err := Open(dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func mustExec(t *testing.T, db *DB, sql string, params ...any) int64 {
	t.Helper()
	n, err := db.Exec(sql, params)
	if err != nil {
		t.Fatalf("%s: %v", sql, err)
	}
	return n
}

func mustQuery(t *testing.T, db *DB, sql string, params ...any) [][]any {
	t.Helper()
	_, rows, err := db.Query(sql, params)
	if err != nil {
		t.Fatalf("%s: %v", sql, err)
	}
	return rows
}

func TestPlaceholders(t *testing.T) {
	cases := map[string]string{
		"SELECT ?, ?":                    "SELECT $1, $2",
		"SELECT '?', \"a?\", ? -- ?\n":   "SELECT '?', \"a?\", $1 -- ?\n",
		"SELECT 'it''s ?', E'\\'?', ?":   "SELECT 'it''s ?', E'\\'?', $1",
		"SELECT $$ ? $$, $tag$?$tag$, ?": "SELECT $$ ? $$, $tag$?$tag$, $1",
		"SELECT /* ? /* ? */ */ ?":       "SELECT /* ? /* ? */ */ $1",
		"SELECT $1":                      "SELECT $1",
	}
	for in, want := range cases {
		got, _, err := convertPlaceholders(in)
		if err != nil || got != want {
			t.Errorf("%q: got %q (%v), want %q", in, got, err, want)
		}
	}
}

func TestRoundTrip(t *testing.T) {
	db := connect(t)
	mustExec(t, db, "DROP TABLE IF EXISTS turtle_books")
	mustExec(t, db, "CREATE TABLE turtle_books (id SERIAL PRIMARY KEY, sku TEXT UNIQUE NOT NULL, title TEXT, price NUMERIC(10,2), qty INTEGER, rating DOUBLE PRECISION, ok BOOLEAN, added DATE, at TIMESTAMPTZ, data JSONB, raw BYTEA)")
	n := mustExec(t, db, "INSERT INTO turtle_books (sku, title, price, qty, rating, ok, added, at, data, raw) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?), (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
		"B1", "Dune ? 🐢", 9.5, int64(4), 4.25, true, "2026-10-05", "2026-10-05 14:30:00+00", `{"a": [1, 2]}`, []byte{0xca, 0xfe},
		"B2", "Emma", "12.00", nil, nil, false, nil, nil, nil, nil)
	if n != 2 {
		t.Errorf("inserted %d", n)
	}
	cols, rows, err := db.Query("SELECT sku, title, price, qty, rating, ok, added, at, data, raw FROM turtle_books ORDER BY sku", nil)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(cols, ",") != "sku,title,price,qty,rating,ok,added,at,data,raw" {
		t.Errorf("columns: %v", cols)
	}
	r := rows[0]
	if r[0] != "B1" || r[1] != "Dune ? 🐢" || r[2] != 9.5 || r[3] != int64(4) || r[4] != 4.25 || r[5] != true {
		t.Errorf("row 1: %#v", r)
	}
	if d, ok := r[6].(time.Time); !ok || d.Format("2006-01-02") != "2026-10-05" {
		t.Errorf("date: %#v", r[6])
	}
	if at, ok := r[7].(time.Time); !ok || !at.Equal(time.Date(2026, 10, 5, 14, 30, 0, 0, time.UTC)) {
		t.Errorf("timestamptz: %#v", r[7])
	}
	if r[8] != `{"a": [1, 2]}` || !reflect.DeepEqual(r[9], []byte{0xca, 0xfe}) {
		t.Errorf("json, bytea: %#v %#v", r[8], r[9])
	}
	if r := rows[1]; r[2] != 12.0 || r[3] != nil || r[5] != false {
		t.Errorf("row 2: %#v", r)
	}
	if n := mustExec(t, db, "UPDATE turtle_books SET qty = coalesce(qty, 0) + ?", int64(1)); n != 2 {
		t.Errorf("updated %d", n)
	}
	// A unique violation: the server's message, with its code.
	_, err = db.Exec("INSERT INTO turtle_books (sku) VALUES (?)", []any{"B1"})
	if e, ok := err.(*Error); !ok || e.Code != "23505" || !strings.Contains(e.Msg, "duplicate key") {
		t.Errorf("duplicate: %v", err)
	}
	// Several statements at once, without values.
	if _, err := db.Exec("UPDATE turtle_books SET qty = 0; UPDATE turtle_books SET qty = qty + 1 WHERE sku = 'B1'", nil); err != nil {
		t.Fatal(err)
	}
	if got := mustQuery(t, db, "SELECT sum(qty) FROM turtle_books"); got[0][0] != int64(1) {
		t.Errorf("after two statements: %v", got)
	}
	tables, err := db.Tables()
	if err != nil || !strings.Contains(strings.Join(tables, ","), "turtle_books") {
		t.Errorf("tables: %v %v", tables, err)
	}
	if _, _, err := db.Query("SELEC 1", nil); err == nil || !strings.Contains(err.Error(), "syntax error") {
		t.Errorf("syntax error: %v", err)
	}
	// The connection still works after an error.
	if got := mustQuery(t, db, "SELECT 1 + ?", int64(1)); got[0][0] != int64(2) {
		t.Errorf("after an error: %v", got)
	}
	if got := mustQuery(t, db, "SELECT 'NaN'::float8, 'Infinity'::float8"); !math.IsNaN(got[0][0].(float64)) || !math.IsInf(got[0][1].(float64), 1) {
		t.Errorf("special floats: %v", got)
	}
	mustExec(t, db, "DROP TABLE turtle_books")
}

func TestTransactionsAndRows(t *testing.T) {
	db := connect(t)
	mustExec(t, db, "DROP TABLE IF EXISTS turtle_t")
	mustExec(t, db, "CREATE TABLE turtle_t (k TEXT PRIMARY KEY, v INTEGER CHECK (v > 0))")
	n, err := db.ExecRows("INSERT INTO turtle_t VALUES (?, ?)", [][]any{{"a", int64(1)}, {"b", int64(2)}})
	if err != nil || n != 2 {
		t.Fatalf("rows: %d %v", n, err)
	}
	// A bad row undoes all of them.
	if _, err := db.ExecRows("INSERT INTO turtle_t VALUES (?, ?)", [][]any{{"c", int64(3)}, {"d", int64(-1)}}); err == nil {
		t.Fatal("bad row accepted")
	}
	if got := mustQuery(t, db, "SELECT count(*) FROM turtle_t"); got[0][0] != int64(2) {
		t.Errorf("after the bad rows: %v", got)
	}
	// Inside BEGIN, a savepoint: the transaction goes on.
	mustExec(t, db, "BEGIN")
	if !db.InTransaction() {
		t.Error("not in a transaction after BEGIN")
	}
	mustExec(t, db, "INSERT INTO turtle_t VALUES ('e', 5)")
	if _, err := db.ExecRows("INSERT INTO turtle_t VALUES (?, ?)", [][]any{{"f", int64(6)}, {"a", int64(1)}}); err == nil {
		t.Fatal("duplicate accepted")
	}
	mustExec(t, db, "COMMIT")
	if got := mustQuery(t, db, "SELECT string_agg(k, ',' ORDER BY k) FROM turtle_t"); got[0][0] != "a,b,e" {
		t.Errorf("after the transaction: %v", got)
	}
	mustExec(t, db, "BEGIN")
	mustExec(t, db, "DELETE FROM turtle_t")
	mustExec(t, db, "ROLLBACK")
	if got := mustQuery(t, db, "SELECT count(*) FROM turtle_t"); got[0][0] != int64(3) {
		t.Errorf("after ROLLBACK: %v", got)
	}
	mustExec(t, db, "DROP TABLE turtle_t")
}

func TestLoginErrors(t *testing.T) {
	dsn := os.Getenv("TURTLE_PG_URL")
	if dsn == "" {
		t.Skip("TURTLE_PG_URL isn't set")
	}
	bad := strings.Replace(dsn, "://", "://nobody:wrong@", 1)
	if i := strings.Index(dsn, "@"); i >= 0 {
		bad = dsn[:strings.Index(dsn, "://")+3] + "nobody:wrong" + dsn[i:]
	}
	if _, err := Open(bad); err == nil || !strings.Contains(err.Error(), "login failed") {
		t.Errorf("wrong password: %v", err)
	}
	if _, err := Open("postgres://x@127.0.0.1:1/db?connect_timeout=2"); err == nil || !strings.Contains(err.Error(), "is the server running") {
		t.Errorf("no server: %v", err)
	}
}
