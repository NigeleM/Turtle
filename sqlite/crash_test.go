// Copyright 2017-2026 Nigele McCoy
// Licensed under the Apache License, Version 2.0;
// see LICENSE and NOTICE.

package sqlite

import (
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"
)

// unixSQLite3 finds the sqlite3 tool for tests that also use its .shell
// command with cp, sleep and touch.
func unixSQLite3(t *testing.T) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("uses Unix shell commands")
	}
	bin, err := exec.LookPath("sqlite3")
	if err != nil {
		if os.Getenv("TURTLE_REQUIRE_SQLITE3") != "" {
			t.Fatal("sqlite3 isn't installed, and TURTLE_REQUIRE_SQLITE3 is set")
		}
		t.Skip("sqlite3 isn't installed")
	}
	return bin
}

// forget drops a database as if its program had died: no rollback, no
// cleanup, locks released by closing the file.
func forget(db *DB) {
	openMu.Lock()
	delete(openFiles, fileKey(db.path))
	openMu.Unlock()
	db.f.Close()
	if db.walSt != nil && db.walSt.shm != nil {
		db.walSt.shm.Close() // the OS closes every file of a program that dies
		db.walSt.shm = nil
	}
	db.closed = true
}

func snapshot(t *testing.T, db *DB) [][]Value {
	t.Helper()
	return mustQuery(t, db, "SELECT id, v FROM c ORDER BY id")
}

func TestCrashDuringCommit(t *testing.T) {
	for _, stage := range []string{"journal", "pages"} {
		t.Run(stage, func(t *testing.T) {
			db, path := createTest(t, 1024)
			mustExec(t, db, "CREATE TABLE c (id INTEGER PRIMARY KEY, v TEXT)")
			mustExec(t, db, "CREATE INDEX c_v ON c (v)")
			mustExec(t, db, "BEGIN")
			for i := 0; i < 300; i++ {
				mustExec(t, db, "INSERT INTO c (v) VALUES (printf('%.*c', 200, char(65 + ? % 26)))", int64(i))
			}
			mustExec(t, db, "COMMIT")
			before := snapshot(t, db)
			// A big change that dies halfway through its commit.
			mustExec(t, db, "BEGIN")
			mustExec(t, db, "UPDATE c SET v = 'changed' WHERE id % 2 = 0")
			mustExec(t, db, "DELETE FROM c WHERE id > 250")
			mustExec(t, db, "INSERT INTO c (v) SELECT v || 'x' FROM c WHERE id < 50")
			crashAt = stage
			func() {
				defer func() {
					if r := recover(); r != nil {
						if _, ok := r.(simulatedCrash); !ok {
							panic(r)
						}
					}
					crashAt = ""
				}()
				db.Exec("COMMIT", nil)
				t.Fatal("commit didn't crash")
			}()
			forget(db)
			if _, err := os.Stat(path + "-journal"); err != nil {
				t.Fatalf("no journal after the crash: %v", err)
			}
			// The next open puts the old pages back.
			db2, err := Open(path)
			if err != nil {
				t.Fatal(err)
			}
			defer db2.Close()
			if got := snapshot(t, db2); !reflect.DeepEqual(got, before) {
				t.Fatalf("after recovery: %d rows, want %d", len(got), len(before))
			}
			if _, err := os.Stat(path + "-journal"); err == nil {
				t.Error("journal still there after recovery")
			}
			mustCheck(t, db2)
			sqlite3Check(t, path)
		})
	}
}

// TestSQLiteRecoversOurJournal: a crash in Turtle's commit, repaired by
// the real SQLite.
func TestSQLiteRecoversOurJournal(t *testing.T) {
	bin, err := exec.LookPath("sqlite3")
	if err != nil {
		if os.Getenv("TURTLE_REQUIRE_SQLITE3") != "" {
			t.Fatal("sqlite3 isn't installed, and TURTLE_REQUIRE_SQLITE3 is set")
		}
		t.Skip("sqlite3 isn't installed")
	}
	db, path := createTest(t, 1024)
	mustExec(t, db, "CREATE TABLE c (id INTEGER PRIMARY KEY, v TEXT)")
	mustExec(t, db, "INSERT INTO c (v) VALUES ('a'), ('b'), ('c')")
	mustExec(t, db, "BEGIN")
	mustExec(t, db, "INSERT INTO c (v) SELECT printf('%.*c', 900, 'z') FROM c")
	mustExec(t, db, "UPDATE c SET v = 'gone' WHERE id = 1")
	crashAt = "pages"
	func() {
		defer func() { recover(); crashAt = "" }()
		db.Exec("COMMIT", nil)
	}()
	forget(db)
	out, err := exec.Command(bin, path, "SELECT group_concat(v) FROM c; PRAGMA integrity_check").CombinedOutput()
	// sqlite3 on Windows ends its lines with \r\n.
	if err != nil || strings.ReplaceAll(strings.TrimSpace(string(out)), "\r\n", "\n") != "a,b,c\nok" {
		t.Fatalf("sqlite3 after our crash: %v\n%s", err, out)
	}
}

// TestWeRecoverSQLiteJournal: sqlite3 dies in the middle of a
// transaction that already changed the file; Turtle repairs it.
func TestWeRecoverSQLiteJournal(t *testing.T) {
	bin := unixSQLite3(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "s.db")
	crash := filepath.Join(dir, "crash.db")
	script := `PRAGMA page_size = 1024;
CREATE TABLE c (id INTEGER PRIMARY KEY, v TEXT);
WITH RECURSIVE n(x) AS (SELECT 1 UNION ALL SELECT x + 1 FROM n WHERE x < 2000) INSERT INTO c SELECT x, printf('%.*c', 100, 'q') FROM n;
PRAGMA cache_size = 10;
PRAGMA cache_spill = 10;
BEGIN;
UPDATE c SET v = printf('%.*c', 150, 'w');
DELETE FROM c WHERE id > 1500;
.shell cp ` + path + ` ` + crash + ` && cp ` + path + `-journal ` + crash + `-journal
ROLLBACK;
`
	cmd := exec.Command(bin, "-cmd", "PRAGMA auto_vacuum = NONE", path)
	cmd.Stdin = strings.NewReader(script)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("sqlite3: %v\n%s", err, out)
	}
	if _, err := os.Stat(crash + "-journal"); err != nil {
		t.Skip("sqlite3 didn't leave a journal to copy")
	}
	// Without its journal the file is half changed: damaged.
	broken := filepath.Join(dir, "broken.db")
	data, _ := os.ReadFile(crash)
	os.WriteFile(broken, data, 0o644)
	if bdb, err := Open(broken); err == nil {
		if bdb.Check() == nil {
			bdb.Close()
			t.Skip("this sqlite3 didn't write to the file before COMMIT, so there's nothing to repair")
		}
		bdb.Close()
	}
	db, err := Open(crash)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	rows := mustQuery(t, db, "SELECT count(*), min(v) = max(v), length(min(v)) FROM c")
	if !reflect.DeepEqual(rows[0], []Value{int64(2000), int64(1), int64(100)}) {
		t.Fatalf("after recovery: %v", rows)
	}
	mustCheck(t, db)
	sqlite3Check(t, crash)
}

// TestLocksWithSQLite: while sqlite3 holds a write lock Turtle waits and
// then gives up, and the other way round.
func TestLocksWithSQLite(t *testing.T) {
	bin := unixSQLite3(t)
	old := busyTimeout
	busyTimeout = 300 * time.Millisecond
	defer func() { busyTimeout = old }()
	db, path := createTest(t, 4096)
	mustExec(t, db, "CREATE TABLE l (v)")
	mustExec(t, db, "INSERT INTO l VALUES (1)")
	// sqlite3 takes an exclusive lock, says so with a marker file, and
	// holds the lock for two seconds.
	marker := filepath.Join(t.TempDir(), "locked")
	cmd := exec.Command(bin, "-cmd", "PRAGMA auto_vacuum = NONE", path)
	cmd.Stdin = strings.NewReader("BEGIN EXCLUSIVE;\nINSERT INTO l VALUES (2);\n.shell touch " + marker + " && sleep 2\nCOMMIT;\n")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	for deadline := time.Now().Add(10 * time.Second); ; time.Sleep(10 * time.Millisecond) {
		if _, err := os.Stat(marker); err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("sqlite3 never took its lock")
		}
	}
	if _, err := db.Exec("INSERT INTO l VALUES (3)", nil); err == nil || !strings.Contains(err.Error(), "locked") {
		t.Errorf("write during sqlite3's lock: %v", err)
	}
	if _, _, err := db.Query("SELECT count(*) FROM l", nil); err == nil || !strings.Contains(err.Error(), "locked") {
		t.Errorf("read during sqlite3's exclusive lock: %v", err)
	}
	cmd.Wait()
	mustExec(t, db, "INSERT INTO l VALUES (3)")
	if got := mustQuery(t, db, "SELECT group_concat(v) FROM l"); got[0][0] != "1,2,3" {
		t.Errorf("after sqlite3 finished: %v", got)
	}
	// Turtle holds a write transaction; sqlite3 can read but not write.
	mustExec(t, db, "BEGIN")
	mustExec(t, db, "INSERT INTO l VALUES (4)")
	out, _ := exec.Command(bin, "-cmd", ".timeout 100", path, "SELECT count(*) FROM l").CombinedOutput()
	if strings.TrimSpace(string(out)) != "3" {
		t.Errorf("sqlite3 reading during our transaction: %s", out)
	}
	out, _ = exec.Command(bin, "-cmd", ".timeout 100", path, "INSERT INTO l VALUES (5)").CombinedOutput()
	if !strings.Contains(string(out), "locked") {
		t.Errorf("sqlite3 writing during our transaction: %s", out)
	}
	mustExec(t, db, "COMMIT")
	out, _ = exec.Command(bin, path, "SELECT group_concat(v) FROM l").CombinedOutput()
	if strings.TrimSpace(string(out)) != "1,2,3,4" {
		t.Errorf("sqlite3 after our commit: %s", out)
	}
}

// TestSeesOtherProgramsChanges: a change made by sqlite3 shows up in the
// next query.
func TestSeesOtherProgramsChanges(t *testing.T) {
	bin, err := exec.LookPath("sqlite3")
	if err != nil {
		if os.Getenv("TURTLE_REQUIRE_SQLITE3") != "" {
			t.Fatal("sqlite3 isn't installed, and TURTLE_REQUIRE_SQLITE3 is set")
		}
		t.Skip("sqlite3 isn't installed")
	}
	db, path := createTest(t, 4096)
	mustExec(t, db, "CREATE TABLE s (v)")
	if out, err := exec.Command(bin, path, "INSERT INTO s VALUES ('from sqlite3'); CREATE TABLE extra (x)").CombinedOutput(); err != nil {
		t.Fatalf("%v %s", err, out)
	}
	if got := mustQuery(t, db, "SELECT v FROM s"); len(got) != 1 || got[0][0] != "from sqlite3" {
		t.Errorf("got %v", got)
	}
	if got := db.Tables(); !reflect.DeepEqual(got, []string{"s", "extra"}) {
		t.Errorf("tables: %v", got)
	}
	mustExec(t, db, "INSERT INTO extra VALUES (1)")
	mustCheck(t, db)
	sqlite3Check(t, path)
}

// TestCommitWaitsForReaders: while sqlite3 reads, COMMIT can't finish;
// the transaction stays open, and COMMIT works once the reader is done.
func TestCommitWaitsForReaders(t *testing.T) {
	bin := unixSQLite3(t)
	old := busyTimeout
	busyTimeout = 200 * time.Millisecond
	defer func() { busyTimeout = old }()
	db, path := createTest(t, 4096)
	mustExec(t, db, "CREATE TABLE w (v)")
	marker := filepath.Join(t.TempDir(), "reading")
	cmd := exec.Command(bin, "-cmd", "PRAGMA auto_vacuum = NONE", path)
	cmd.Stdin = strings.NewReader("BEGIN;\nSELECT count(*) FROM w;\n.shell touch " + marker + " && sleep 2\nCOMMIT;\n")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	for deadline := time.Now().Add(10 * time.Second); ; time.Sleep(10 * time.Millisecond) {
		if _, err := os.Stat(marker); err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("sqlite3 never started reading")
		}
	}
	mustExec(t, db, "BEGIN")
	mustExec(t, db, "INSERT INTO w VALUES (1)")
	if _, err := db.Exec("COMMIT", nil); err == nil || !strings.Contains(err.Error(), "locked") {
		t.Fatalf("commit during a read: %v", err)
	}
	cmd.Wait()
	mustExec(t, db, "COMMIT")
	if got := mustQuery(t, db, "SELECT count(*) FROM w"); got[0][0] != int64(1) {
		t.Errorf("after retrying COMMIT: %v", got)
	}
	mustCheck(t, db)
}

// TestCrashDuringVacuum: VACUUM shrinks the file; a crash halfway must
// still bring every original page back.
func TestCrashDuringVacuum(t *testing.T) {
	syncFiles = false
	defer func() { syncFiles = true }()
	db, path := createTest(t, 1024)
	mustExec(t, db, "CREATE TABLE c (id INTEGER PRIMARY KEY, v TEXT)")
	mustExec(t, db, "BEGIN")
	for i := 0; i < 600; i++ {
		mustExec(t, db, "INSERT INTO c (v) VALUES (printf('%.*c', 300, 'q'))")
	}
	mustExec(t, db, "COMMIT")
	mustExec(t, db, "DELETE FROM c WHERE id % 4 != 0")
	before := snapshot(t, db)
	crashAt = "pages"
	func() {
		defer func() { recover(); crashAt = "" }()
		db.Exec("VACUUM", nil)
		t.Fatal("VACUUM didn't crash")
	}()
	forget(db)
	db2, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer db2.Close()
	if got := snapshot(t, db2); !reflect.DeepEqual(got, before) {
		t.Fatalf("after recovery: %d rows, want %d", len(got), len(before))
	}
	mustCheck(t, db2)
	sqlite3Check(t, path)
}

// TestCheckAutoVacuum: files sqlite3 makes with auto_vacuum (some
// builds, such as GitHub's macOS one, turn it on by default) have
// pointer-map pages; Check counts them, and Turtle reads the file but
// refuses to change it.
func TestCheckAutoVacuum(t *testing.T) {
	bin := unixSQLite3(t)
	for _, mode := range []string{"FULL", "INCREMENTAL"} {
		path := filepath.Join(t.TempDir(), "av.db")
		cmd := exec.Command(bin, "-cmd", "PRAGMA page_size = 1024; PRAGMA auto_vacuum = "+mode, path)
		cmd.Stdin = strings.NewReader("CREATE TABLE t (id INTEGER PRIMARY KEY, v TEXT); CREATE INDEX t_v ON t (v);" +
			"WITH RECURSIVE n(i) AS (SELECT 1 UNION ALL SELECT i + 1 FROM n WHERE i < 3000) INSERT INTO t (v) SELECT printf('%.*c', i % 300, 'x') FROM n;" +
			"DELETE FROM t WHERE id % 3 = 0;")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("sqlite3: %v %s", err, out)
		}
		db, err := Open(path)
		if err != nil {
			t.Fatal(err)
		}
		if err := db.Check(); err != nil {
			t.Errorf("%s: %v", mode, err)
		}
		if got := mustQuery(t, db, "SELECT count(*) FROM t"); got[0][0] != int64(2000) {
			t.Errorf("%s: %v", mode, got)
		}
		if _, err := db.Exec("DELETE FROM t", nil); err == nil || !strings.Contains(err.Error(), "auto_vacuum") {
			t.Errorf("%s: writing: %v", mode, err)
		}
		db.Close()
	}
}
