// Copyright 2017-2026 Nigele McCoy
// Licensed under the Apache License, Version 2.0;
// see LICENSE and NOTICE.

package mysql

import (
	"os"
	"reflect"
	"strings"
	"testing"
	"time"
)

// These tests need a server: TURTLE_MYSQL_URL=mysql://user:pw@host:port/db
// (CI starts one; locally, docker run mysql). Without it they skip.
func connect(t *testing.T) *DB {
	t.Helper()
	dsn := os.Getenv("TURTLE_MYSQL_URL")
	if dsn == "" {
		t.Skip("TURTLE_MYSQL_URL isn't set")
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

func TestRoundTrip(t *testing.T) {
	db := connect(t)
	mustExec(t, db, "DROP TABLE IF EXISTS turtle_books")
	mustExec(t, db, "CREATE TABLE turtle_books (id INT AUTO_INCREMENT PRIMARY KEY, sku VARCHAR(20) UNIQUE NOT NULL, title TEXT, price DECIMAL(10,2), qty INT, rating DOUBLE, ok BOOLEAN, added DATE, at DATETIME(3), data JSON, raw BLOB, big BIGINT UNSIGNED)")
	n := mustExec(t, db, "INSERT INTO turtle_books (sku, title, price, qty, rating, ok, added, at, data, raw, big) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?), (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
		"B1", "Dune ? 🐢", 9.5, int64(4), 4.25, true, "2026-10-05", "2026-10-05 14:30:00.250", `{"a": [1, 2]}`, []byte{0xca, 0xfe}, int64(7),
		"B2", "Emma", "12.00", nil, nil, false, nil, nil, nil, nil, nil)
	if n != 2 {
		t.Errorf("inserted %d", n)
	}
	query := "SELECT sku, title, price, qty, rating, ok, added, at, data, raw, big FROM turtle_books ORDER BY sku"
	check := func(how string, cols []string, rows [][]any) {
		if strings.Join(cols, ",") != "sku,title,price,qty,rating,ok,added,at,data,raw,big" {
			t.Errorf("%s columns: %v", how, cols)
		}
		r := rows[0]
		if r[0] != "B1" || r[1] != "Dune ? 🐢" || r[2] != 9.5 || r[3] != int64(4) || r[4] != 4.25 || r[5] != int64(1) || r[10] != int64(7) {
			t.Errorf("%s row 1: %#v", how, r)
		}
		if d, ok := r[6].(time.Time); !ok || d.Format("2006-01-02") != "2026-10-05" {
			t.Errorf("%s date: %#v", how, r[6])
		}
		if at, ok := r[7].(time.Time); !ok || !at.Equal(time.Date(2026, 10, 5, 14, 30, 0, 250e6, time.Local)) {
			t.Errorf("%s datetime: %#v", how, r[7])
		}
		if r[8] != `{"a": [1, 2]}` || !reflect.DeepEqual(r[9], []byte{0xca, 0xfe}) {
			t.Errorf("%s json, blob: %#v %#v", how, r[8], r[9])
		}
		if r := rows[1]; r[2] != 12.0 || r[3] != nil || r[5] != int64(0) || r[6] != nil {
			t.Errorf("%s row 2: %#v", how, r)
		}
	}
	// Text protocol (no values) and binary protocol (prepared, with a value).
	cols, rows, err := db.Query(query, nil)
	if err != nil {
		t.Fatal(err)
	}
	check("text", cols, rows)
	cols, rows, err = db.Query(strings.Replace(query, "ORDER BY", "WHERE id > ? ORDER BY", 1), []any{int64(0)})
	if err != nil {
		t.Fatal(err)
	}
	check("binary", cols, rows)

	if n := mustExec(t, db, "UPDATE turtle_books SET qty = coalesce(qty, 0) + ?", int64(1)); n != 2 {
		t.Errorf("updated %d", n)
	}
	_, err = db.Exec("INSERT INTO turtle_books (sku) VALUES (?)", []any{"B1"})
	if e, ok := err.(*Error); !ok || e.Code != 1062 || !strings.Contains(e.Msg, "Duplicate entry") {
		t.Errorf("duplicate: %v", err)
	}
	if _, err := db.Exec("UPDATE turtle_books SET qty = 0; UPDATE turtle_books SET qty = qty + 1 WHERE sku = 'B1'", nil); err != nil {
		t.Fatal(err)
	}
	if got := mustQuery(t, db, "SELECT sum(qty) FROM turtle_books"); got[0][0] != int64(1) {
		t.Errorf("after two statements: %#v", got)
	}
	tables, err := db.Tables()
	if err != nil || !strings.Contains(strings.Join(tables, ","), "turtle_books") {
		t.Errorf("tables: %v %v", tables, err)
	}
	if _, _, err := db.Query("SELEC 1", nil); err == nil || !strings.Contains(err.Error(), "SQL syntax") {
		t.Errorf("syntax error: %v", err)
	}
	if got := mustQuery(t, db, "SELECT 1 + ?", int64(1)); got[0][0] != int64(2) {
		t.Errorf("after an error: %#v", got)
	}
	mustExec(t, db, "DROP TABLE turtle_books")
}

func TestTransactionsAndRows(t *testing.T) {
	db := connect(t)
	mustExec(t, db, "DROP TABLE IF EXISTS turtle_t")
	mustExec(t, db, "CREATE TABLE turtle_t (k VARCHAR(10) PRIMARY KEY, v INT CHECK (v > 0))")
	n, err := db.ExecRows("INSERT INTO turtle_t VALUES (?, ?)", [][]any{{"a", int64(1)}, {"b", int64(2)}})
	if err != nil || n != 2 {
		t.Fatalf("rows: %d %v", n, err)
	}
	if _, err := db.ExecRows("INSERT INTO turtle_t VALUES (?, ?)", [][]any{{"c", int64(3)}, {"d", int64(-1)}}); err == nil {
		t.Fatal("bad row accepted")
	}
	if got := mustQuery(t, db, "SELECT count(*) FROM turtle_t"); got[0][0] != int64(2) {
		t.Errorf("after the bad rows: %#v", got)
	}
	mustExec(t, db, "BEGIN")
	if !db.InTransaction() {
		t.Error("not in a transaction after BEGIN")
	}
	mustExec(t, db, "INSERT INTO turtle_t VALUES ('e', 5)")
	if _, err := db.ExecRows("INSERT INTO turtle_t VALUES (?, ?)", [][]any{{"f", int64(6)}, {"a", int64(1)}}); err == nil {
		t.Fatal("duplicate accepted")
	}
	if !db.InTransaction() {
		t.Error("the transaction ended after a failed savepoint")
	}
	mustExec(t, db, "COMMIT")
	if got := mustQuery(t, db, "SELECT group_concat(k ORDER BY k) FROM turtle_t"); got[0][0] != "a,b,e" {
		t.Errorf("after the transaction: %#v", got)
	}
	mustExec(t, db, "BEGIN")
	mustExec(t, db, "DELETE FROM turtle_t")
	mustExec(t, db, "ROLLBACK")
	if got := mustQuery(t, db, "SELECT count(*) FROM turtle_t"); got[0][0] != int64(3) {
		t.Errorf("after ROLLBACK: %#v", got)
	}
	mustExec(t, db, "DROP TABLE turtle_t")
}

func TestLoginErrors(t *testing.T) {
	dsn := os.Getenv("TURTLE_MYSQL_URL")
	if dsn == "" {
		t.Skip("TURTLE_MYSQL_URL isn't set")
	}
	i := strings.Index(dsn, "@")
	bad := dsn[:strings.Index(dsn, "://")+3] + "nobody:wrong" + dsn[i:]
	if _, err := Open(bad); err == nil || !strings.Contains(err.Error(), "login failed") {
		t.Errorf("wrong password: %v", err)
	}
	if _, err := Open("mysql://x@127.0.0.1:1/db?timeout=2"); err == nil || !strings.Contains(err.Error(), "is the server running") {
		t.Errorf("no server: %v", err)
	}
}
