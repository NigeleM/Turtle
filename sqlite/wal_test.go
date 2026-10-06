package sqlite

import (
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

// TestWALMadeBySQLite: sqlite3 leaves changes in the WAL (copied while
// it's open, as after a crash); Turtle reads them, then writes.
func TestWALMadeBySQLite(t *testing.T) {
	bin := unixSQLite3(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "w.db")
	copyPath := filepath.Join(dir, "copy.db")
	script := `PRAGMA journal_mode = WAL;
PRAGMA wal_autocheckpoint = 0;
CREATE TABLE t (id INTEGER PRIMARY KEY, v TEXT);
WITH RECURSIVE n(i) AS (SELECT 1 UNION ALL SELECT i + 1 FROM n WHERE i < 300) INSERT INTO t SELECT i, printf('%.*c', 50, 'w') FROM n;
UPDATE t SET v = 'changed' WHERE id % 10 = 0;
.shell cp ` + path + ` ` + copyPath + ` && cp ` + path + `-wal ` + copyPath + `-wal
`
	cmd := exec.Command(bin, "-cmd", "PRAGMA auto_vacuum = NONE", path)
	cmd.Stdin = strings.NewReader(script)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("sqlite3: %v\n%s", err, out)
	}
	if info, err := os.Stat(copyPath + "-wal"); err != nil || info.Size() == 0 {
		t.Skip("no WAL frames were left to read")
	}
	db, err := Open(copyPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	got := mustQuery(t, db, "SELECT count(*), sum(v = 'changed') FROM t")
	if !reflect.DeepEqual(got[0], []Value{int64(300), int64(30)}) {
		t.Fatalf("reading through the WAL: %v", got)
	}
	if got := mustQuery(t, db, "PRAGMA journal_mode"); got[0][0] != "wal" {
		t.Errorf("journal_mode: %v", got)
	}
	mustExec(t, db, "DELETE FROM t WHERE id > 250")
	mustExec(t, db, "INSERT INTO t (v) VALUES ('turtle')")
	mustCheck(t, db)
	db.Close()
	out, err := exec.Command(bin, copyPath, "SELECT count(*), max(v) FROM t; PRAGMA integrity_check; PRAGMA journal_mode").CombinedOutput()
	if err != nil || strings.TrimSpace(string(out)) != "251|wwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwww\nok\nwal" {
		t.Fatalf("sqlite3 after Turtle's writes: %v\n%s", err, out)
	}
}

func TestWALSwitchAndCrash(t *testing.T) {
	syncFiles = false
	defer func() { syncFiles = true }()
	db, path := createTest(t, 1024)
	mustExec(t, db, "CREATE TABLE t (id INTEGER PRIMARY KEY, v TEXT)")
	if got := mustQuery(t, db, "PRAGMA journal_mode = WAL"); got[0][0] != "wal" {
		t.Fatalf("switching: %v", got)
	}
	mustExec(t, db, "BEGIN")
	for i := 0; i < 200; i++ {
		mustExec(t, db, "INSERT INTO t (v) VALUES (printf('%.*c', 100, 'a'))")
	}
	mustExec(t, db, "COMMIT")
	mustCheck(t, db)
	sqlite3Check(t, path)
	// A crash after the WAL holds the commit, before the file has it.
	crashAt = "wal"
	func() {
		defer func() { recover(); crashAt = "" }()
		db.Exec("UPDATE t SET v = 'after crash' WHERE id <= 50", nil)
		t.Fatal("no crash")
	}()
	forget(db)
	db2, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer db2.Close()
	if got := mustQuery(t, db2, "SELECT count(*) FROM t WHERE v = 'after crash'"); got[0][0] != int64(50) {
		t.Fatalf("the committed change was lost: %v", got)
	}
	mustExec(t, db2, "INSERT INTO t (v) VALUES ('next')")
	mustCheck(t, db2)
	sqlite3Check(t, path)
	if got := mustQuery(t, db2, "PRAGMA journal_mode = DELETE"); got[0][0] != "delete" {
		t.Fatalf("switching back: %v", got)
	}
	mustExec(t, db2, "INSERT INTO t (v) VALUES ('rollback journal again')")
	mustCheck(t, db2)
	sqlite3Check(t, path)
}

// TestWALOpenElsewhere: while sqlite3 has a WAL database open, Turtle
// waits, then says so.
func TestWALOpenElsewhere(t *testing.T) {
	bin := unixSQLite3(t)
	old := busyTimeout
	busyTimeout = 300 * time.Millisecond
	defer func() { busyTimeout = old }()
	db, path := createTest(t, 4096)
	mustExec(t, db, "CREATE TABLE t (v)")
	mustExec(t, db, "PRAGMA journal_mode = WAL")
	db.Close()
	marker := filepath.Join(t.TempDir(), "open")
	cmd := exec.Command(bin, "-cmd", "PRAGMA auto_vacuum = NONE", path)
	cmd.Stdin = strings.NewReader("SELECT count(*) FROM t;\n.shell touch " + marker + " && sleep 2\n")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	for deadline := time.Now().Add(10 * time.Second); ; time.Sleep(10 * time.Millisecond) {
		if _, err := os.Stat(marker); err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("sqlite3 never opened the file")
		}
	}
	db2, err := Open(path)
	if err == nil {
		_, _, err = db2.Query("SELECT count(*) FROM t", nil)
		db2.Close()
	}
	if err == nil || !strings.Contains(err.Error(), "open in another program") {
		t.Errorf("while sqlite3 has it open: %v", err)
	}
	cmd.Wait()
	db3, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer db3.Close()
	mustExec(t, db3, "INSERT INTO t VALUES (1)")
	mustCheck(t, db3)
}
