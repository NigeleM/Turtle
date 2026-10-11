// Copyright 2017-2026 Nigele McCoy
// Licensed under the Apache License, Version 2.0;
// see LICENSE and NOTICE.

package sqlite

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestReturningAndCounters(t *testing.T) {
	db, _ := createTest(t, 4096)
	mustExec(t, db, "CREATE TABLE r (id INTEGER PRIMARY KEY, v TEXT, n INTEGER DEFAULT 7)")
	cols, rows, err := db.Query("INSERT INTO r (v) VALUES ('a'), ('b') RETURNING id, v || '!' AS shout, n", nil)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(cols, []string{"id", "shout", "n"}) || !reflect.DeepEqual(rows, [][]Value{{int64(1), "a!", int64(7)}, {int64(2), "b!", int64(7)}}) {
		t.Errorf("insert returning: %v %v", cols, rows)
	}
	_, rows, err = db.Query("UPDATE r SET n = n * 2 WHERE id = 2 RETURNING *", nil)
	if err != nil || !reflect.DeepEqual(rows, [][]Value{{int64(2), "b", int64(14)}}) {
		t.Errorf("update returning: %v %v", rows, err)
	}
	_, rows, err = db.Query("DELETE FROM r WHERE id = 1 RETURNING v", nil)
	if err != nil || !reflect.DeepEqual(rows, [][]Value{{"a"}}) {
		t.Errorf("delete returning: %v %v", rows, err)
	}
	res := mustExec(t, db, "INSERT INTO r (v) VALUES ('c')")
	got := mustQuery(t, db, "SELECT last_insert_rowid(), changes(), total_changes()")
	if !reflect.DeepEqual(got[0], []Value{res.LastRowid, int64(1), int64(5)}) {
		t.Errorf("counters: %v (last rowid %d)", got, res.LastRowid)
	}
}

func TestMisuse(t *testing.T) {
	db, _ := createTest(t, 4096)
	mustExec(t, db, "CREATE TABLE m (v)")
	cases := []struct {
		exec      bool
		sql, want string
	}{
		{true, "SELECT 1", "run it with Query"},
		{false, "INSERT INTO m VALUES (1)", "run it with sql_run"},
		{true, "INSERT INTO m VALUES (?); INSERT INTO m VALUES (?)", "single statement"},
		{true, "COMMIT", "no transaction is active"},
		{true, "ROLLBACK", "no transaction is active"},
		{true, "INSERT INTO nope VALUES (1)", "no such table: nope"},
		{true, "INSERT INTO m (x) VALUES (1)", "has no column named x"},
		{true, "INSERT INTO m VALUES (1, 2)", "1 columns but 2 values"},
		{true, "UPDATE m SET x = 1", "no such column: x"},
		{true, "DELETE FROM m WHERE nope = 1", "no such column: nope"},
		{true, "CREATE TABLE m (x)", "table m already exists"},
		{true, "CREATE TEMP TABLE x (a)", "TEMP"},
		{true, "DROP TABLE sqlite_schema", "may not be"},
		{true, "INSERT INTO sqlite_schema VALUES ('table', 'x', 'x', 0, '')", "may not be modified"},
		{true, "UPDATE m SET v = 1 FROM nope", "no such table: nope"},
	}
	for _, c := range cases {
		var err error
		if c.exec {
			_, err = db.Exec(c.sql, []Value{int64(1), int64(2)}[:strings.Count(c.sql, "?")])
		} else {
			_, _, err = db.Query(c.sql, nil)
		}
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%q: want error containing %q, got %v", c.sql, c.want, err)
		}
	}
	mustCheck(t, db)
}

func TestTransactionsKeepGoingAfterAnError(t *testing.T) {
	db, _ := createTest(t, 4096)
	mustExec(t, db, "CREATE TABLE k (id INTEGER PRIMARY KEY, v UNIQUE)")
	mustExec(t, db, "BEGIN")
	mustExec(t, db, "INSERT INTO k (v) VALUES ('a')")
	if _, err := db.Exec("INSERT INTO k (v) VALUES ('b'), ('a')", nil); err == nil {
		t.Fatal("duplicate accepted")
	}
	// The failed statement left nothing; the transaction goes on.
	mustExec(t, db, "INSERT INTO k (v) VALUES ('c')")
	mustExec(t, db, "COMMIT")
	if got := mustQuery(t, db, "SELECT group_concat(v) FROM k"); got[0][0] != "a,c" {
		t.Errorf("got %v", got)
	}
	// ON CONFLICT FAIL keeps the rows before the failing one.
	if _, err := db.Exec("INSERT OR FAIL INTO k (v) VALUES ('d'), ('a'), ('e')", nil); err == nil {
		t.Fatal("duplicate accepted")
	}
	if got := mustQuery(t, db, "SELECT group_concat(v) FROM k"); got[0][0] != "a,c,d" {
		t.Errorf("after FAIL: %v", got)
	}
	// ON CONFLICT ROLLBACK ends the whole transaction.
	mustExec(t, db, "BEGIN")
	mustExec(t, db, "INSERT INTO k (v) VALUES ('f')")
	if _, err := db.Exec("INSERT OR ROLLBACK INTO k (v) VALUES ('a')", nil); err == nil {
		t.Fatal("duplicate accepted")
	}
	if _, err := db.Exec("COMMIT", nil); err == nil {
		t.Error("the transaction is still open after ROLLBACK")
	}
	if got := mustQuery(t, db, "SELECT group_concat(v) FROM k"); got[0][0] != "a,c,d" {
		t.Errorf("after ROLLBACK: %v", got)
	}
	mustCheck(t, db)
}

func TestCloseRollsBackAndHandlesShareAFile(t *testing.T) {
	db, path := createTest(t, 4096)
	mustExec(t, db, "CREATE TABLE h (v)")
	other, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	mustExec(t, db, "BEGIN")
	mustExec(t, db, "INSERT INTO h VALUES (1)")
	// Same program, same file: the other handle sees the transaction.
	if got := mustQuery(t, other, "SELECT count(*) FROM h"); got[0][0] != int64(1) {
		t.Errorf("other handle: %v", got)
	}
	other.Close()
	db.Close()
	db2, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer db2.Close()
	if got := mustQuery(t, db2, "SELECT count(*) FROM h"); got[0][0] != int64(0) {
		t.Errorf("an unfinished transaction survived Close: %v", got)
	}
}

func TestFilesThatCantBeWritten(t *testing.T) {
	db, path := createTest(t, 4096)
	mustExec(t, db, "CREATE TABLE f (v)")
	db.Close()
	// Read-only.
	os.Chmod(path, 0o444)
	defer os.Chmod(path, 0o644)
	ro, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := ro.Query("SELECT count(*) FROM f", nil); err != nil {
		t.Errorf("reading a read-only file: %v", err)
	}
	// root may write any file, read-only or not (as in a container).
	if _, err := ro.Exec("INSERT INTO f VALUES (1)", nil); os.Geteuid() != 0 && (err == nil || !strings.Contains(err.Error(), "read-only")) {
		t.Errorf("writing a read-only file: %v", err)
	}
	ro.Close()
	os.Chmod(path, 0o644)
	// UTF-16 text can be read but not written.
	data, _ := os.ReadFile(path)
	data[59] = 2
	u16 := filepath.Join(t.TempDir(), "u16.db")
	os.WriteFile(u16, data, 0o644)
	w, err := Open(u16)
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	if _, err := w.Exec("INSERT INTO f VALUES (1)", nil); err == nil || !strings.Contains(err.Error(), "UTF-16") {
		t.Errorf("writing a UTF-16 file: %v", err)
	}
}

func TestCreateRefusesExistingFile(t *testing.T) {
	_, path := createTest(t, 4096)
	if _, err := Create(path); err == nil {
		t.Error("Create replaced an existing file")
	}
}

func TestPageSizesWrite(t *testing.T) {
	syncFiles = false
	defer func() { syncFiles = true }()
	for _, size := range []int{512, 65536} {
		db, path := createTest(t, size)
		mustExec(t, db, "CREATE TABLE w (id INTEGER PRIMARY KEY, body TEXT)")
		mustExec(t, db, "CREATE INDEX w_body ON w (body)")
		mustExec(t, db, "BEGIN")
		for i := 0; i < 400; i++ {
			mustExec(t, db, "INSERT INTO w (body) VALUES (printf('%.*c', ?, char(97 + ? % 26)))", int64(i*37%2000), int64(i))
		}
		mustExec(t, db, "COMMIT")
		mustExec(t, db, "DELETE FROM w WHERE id % 3 = 0")
		mustExec(t, db, "UPDATE w SET body = body || 'x' WHERE id % 5 = 0")
		if got := mustQuery(t, db, "SELECT count(*), sum(length(body)) FROM w"); got[0][0] != int64(267) {
			t.Errorf("page size %d: %v", size, got)
		}
		mustCheck(t, db)
		sqlite3Check(t, path)
	}
}

func TestExecRows(t *testing.T) {
	db, _ := createTest(t, 4096)
	mustExec(t, db, "CREATE TABLE e (id INTEGER PRIMARY KEY, v TEXT UNIQUE)")
	res, err := db.ExecRows("INSERT INTO e (v) VALUES (?)", [][]Value{{"a"}, {"b"}, {"c"}})
	if err != nil || res.Changes != 3 {
		t.Fatalf("%v %v", res, err)
	}
	// One bad row undoes them all, inside a transaction too.
	mustExec(t, db, "BEGIN")
	mustExec(t, db, "INSERT INTO e (v) VALUES ('kept')")
	if _, err := db.ExecRows("INSERT INTO e (v) VALUES (?)", [][]Value{{"d"}, {"a"}}); err == nil {
		t.Fatal("duplicate accepted")
	}
	mustExec(t, db, "COMMIT")
	if got := mustQuery(t, db, "SELECT group_concat(v) FROM e"); got[0][0] != "a,b,c,kept" {
		t.Errorf("got %v", got)
	}
	if _, err := db.ExecRows("SELECT ?", [][]Value{{1}}); err == nil {
		t.Error("SELECT accepted")
	}
	if _, err := db.ExecRows("INSERT INTO e (v) VALUES (?)", [][]Value{{"x", "y"}}); err == nil {
		t.Error("wrong value count accepted")
	}
	mustCheck(t, db)
}

func TestVacuum(t *testing.T) {
	syncFiles = false
	defer func() { syncFiles = true }()
	db, path := createTest(t, 1024)
	mustExec(t, db, "CREATE TABLE v (id INTEGER PRIMARY KEY, body TEXT, n INTEGER)")
	mustExec(t, db, "CREATE INDEX v_n ON v (n DESC)")
	mustExec(t, db, "CREATE UNIQUE INDEX v_body ON v (lower(body))")
	mustExec(t, db, "CREATE VIEW vv AS SELECT count(*) AS c FROM v")
	mustExec(t, db, "PRAGMA user_version = 7")
	mustExec(t, db, "BEGIN")
	for i := 0; i < 2000; i++ {
		mustExec(t, db, "INSERT INTO v (body, n) VALUES (printf('row %d %.*c', ?, ?, 'x'), ?)", int64(i), int64(i%500), int64(i%37))
	}
	mustExec(t, db, "COMMIT")
	mustExec(t, db, "DELETE FROM v WHERE id % 3 != 0")
	before := mustQuery(t, db, "SELECT id, body, n FROM v ORDER BY id")
	pagesBefore := mustQuery(t, db, "PRAGMA page_count")[0][0].(int64)
	// VACUUM INTO: a compact copy.
	backup := filepath.Join(t.TempDir(), "backup.db")
	mustExec(t, db, "VACUUM INTO ?", backup)
	b, err := Open(backup)
	if err != nil {
		t.Fatal(err)
	}
	if got := mustQuery(t, b, "SELECT id, body, n FROM v ORDER BY id"); !reflect.DeepEqual(got, before) {
		t.Fatal("the copy's rows differ")
	}
	if got := mustQuery(t, b, "SELECT c FROM vv"); got[0][0] != int64(len(before)) {
		t.Errorf("view in the copy: %v", got)
	}
	if got := mustQuery(t, b, "PRAGMA user_version"); got[0][0] != int64(7) {
		t.Errorf("user_version in the copy: %v", got)
	}
	mustCheck(t, b)
	sqlite3Check(t, backup)
	b.Close()
	if _, err := db.Exec("VACUUM INTO ?", []Value{backup}); err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Errorf("VACUUM INTO an existing file: %v", err)
	}
	// VACUUM in place: same rows, fewer pages.
	mustExec(t, db, "VACUUM")
	if got := mustQuery(t, db, "SELECT id, body, n FROM v ORDER BY id"); !reflect.DeepEqual(got, before) {
		t.Fatal("rows changed by VACUUM")
	}
	pagesAfter := mustQuery(t, db, "PRAGMA page_count")[0][0].(int64)
	if pagesAfter >= pagesBefore {
		t.Errorf("VACUUM didn't shrink the file: %d -> %d pages", pagesBefore, pagesAfter)
	}
	if got := mustQuery(t, db, "PRAGMA freelist_count"); got[0][0] != int64(0) {
		t.Errorf("free pages after VACUUM: %v", got)
	}
	mustExec(t, db, "INSERT INTO v (body, n) VALUES ('after', 1)")
	mustCheck(t, db)
	sqlite3Check(t, path)
	mustExec(t, db, "BEGIN")
	if _, err := db.Exec("VACUUM", nil); err == nil {
		t.Error("VACUUM inside a transaction")
	}
	mustExec(t, db, "ROLLBACK")
}

func TestAttach(t *testing.T) {
	db, path := createTest(t, 4096)
	otherPath := filepath.Join(t.TempDir(), "other.db")
	other, err := Create(otherPath)
	if err != nil {
		t.Fatal(err)
	}
	mustExec(t, other, "CREATE TABLE old_books (sku TEXT PRIMARY KEY, title TEXT)")
	mustExec(t, other, "CREATE INDEX old_title ON old_books (title)")
	mustExec(t, other, "INSERT INTO old_books VALUES ('B1', 'Dune'), ('B2', 'Emma')")
	other.Close()
	mustExec(t, db, "CREATE TABLE books (sku TEXT PRIMARY KEY, title TEXT)")
	mustExec(t, db, "INSERT INTO books VALUES ('B2', 'Emma'), ('B3', 'Beloved')")
	mustExec(t, db, "ATTACH ? AS other", otherPath)
	got := mustQuery(t, db, "SELECT b.sku, o.title FROM books b LEFT JOIN other.old_books o ON o.sku = b.sku ORDER BY b.sku")
	if !reflect.DeepEqual(got, [][]Value{{"B2", "Emma"}, {"B3", nil}}) {
		t.Errorf("join across files: %v", got)
	}
	if got := mustQuery(t, db, "SELECT sku FROM old_books WHERE title = 'Dune'"); !reflect.DeepEqual(got, [][]Value{{"B1"}}) {
		t.Errorf("unqualified name found in the attached file: %v", got)
	}
	res := mustExec(t, db, "INSERT INTO books SELECT * FROM other.old_books WHERE sku NOT IN (SELECT sku FROM main.books)")
	if res.Changes != 1 {
		t.Errorf("copied %d rows", res.Changes)
	}
	if _, err := db.Exec("DELETE FROM other.old_books", nil); err == nil || !strings.Contains(err.Error(), "read-only") {
		t.Errorf("writing to an attached database: %v", err)
	}
	if got := mustQuery(t, db, "SELECT name FROM pragma_database_list ORDER BY seq"); !reflect.DeepEqual(got, [][]Value{{"main"}, {"other"}}) {
		t.Errorf("database_list: %v", got)
	}
	if _, err := db.Exec("ATTACH ? AS other", []Value{otherPath}); err == nil {
		t.Error("attached twice under one name")
	}
	mustExec(t, db, "DETACH other")
	if _, _, err := db.Query("SELECT * FROM other.old_books", nil); err == nil || !strings.Contains(err.Error(), "unknown database other") {
		t.Errorf("after DETACH: %v", err)
	}
	mustCheck(t, db)
	sqlite3Check(t, path)
}
