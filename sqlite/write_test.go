package sqlite

import (
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
)

// createTest makes a new database with the given page size for a test.
func createTest(t *testing.T, pageSize int) (*DB, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "t.db")
	db, err := Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if pageSize != 4096 {
		db.Close()
		setPageSize(t, path, pageSize)
		db, err = Open(path)
		if err != nil {
			t.Fatal(err)
		}
	}
	t.Cleanup(func() { db.Close() })
	return db, path
}

// setPageSize rewrites an empty new database with another page size.
func setPageSize(t *testing.T, path string, size int) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	p := make([]byte, size)
	copy(p, data[:100])
	s := size
	if s == 65536 {
		s = 1
	}
	p[16], p[17] = byte(s>>8), byte(s)
	p[100] = leafTable
	cs := size
	if cs == 65536 {
		cs = 0
	}
	p[105], p[106] = byte(cs>>8), byte(cs)
	if err := os.WriteFile(path, p, 0o644); err != nil {
		t.Fatal(err)
	}
}

func mustExec(t *testing.T, db *DB, sql string, params ...Value) Result {
	t.Helper()
	res, err := db.Exec(sql, params)
	if err != nil {
		t.Fatalf("%s: %v", sql, err)
	}
	return res
}

func mustQuery(t *testing.T, db *DB, sql string, params ...Value) [][]Value {
	t.Helper()
	_, rows, err := db.Query(sql, params)
	if err != nil {
		t.Fatalf("%s: %v", sql, err)
	}
	return rows
}

func mustCheck(t *testing.T, db *DB) {
	t.Helper()
	if err := db.Check(); err != nil {
		t.Fatal(err)
	}
}

// sqlite3Check runs the real SQLite's integrity check when the sqlite3
// tool is installed.
func sqlite3Check(t *testing.T, path string) {
	t.Helper()
	bin, err := exec.LookPath("sqlite3")
	if err != nil {
		return
	}
	out, err := exec.Command(bin, path, "PRAGMA integrity_check").CombinedOutput()
	if err != nil || strings.TrimSpace(string(out)) != "ok" {
		t.Fatalf("sqlite3 integrity_check: %v\n%s", err, out)
	}
}

func TestFixturesPassCheck(t *testing.T) {
	for _, size := range pageSizes {
		db := openFixture(t, size)
		if err := db.Check(); err != nil {
			t.Errorf("page size %d: %v", size, err)
		}
	}
}

func TestCreateInsertSelect(t *testing.T) {
	db, path := createTest(t, 4096)
	mustExec(t, db, "CREATE TABLE books (id INTEGER PRIMARY KEY, title TEXT NOT NULL, price INTEGER, rating REAL)")
	res := mustExec(t, db, "INSERT INTO books (title, price, rating) VALUES (?, ?, ?), ('Gone Girl', 1225, 3.9)", "Dune", int64(950), 4.5)
	if res.Changes != 2 || res.LastRowid != 2 {
		t.Errorf("insert result: %+v", res)
	}
	rows := mustQuery(t, db, "SELECT id, title, price, rating FROM books ORDER BY id")
	want := [][]Value{{int64(1), "Dune", int64(950), 4.5}, {int64(2), "Gone Girl", int64(1225), 3.9}}
	if !reflect.DeepEqual(rows, want) {
		t.Errorf("got %v", rows)
	}
	mustCheck(t, db)
	sqlite3Check(t, path)
	// The file reopens with the rows in it.
	db.Close()
	db2, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer db2.Close()
	if got := mustQuery(t, db2, "SELECT count(*) FROM books"); got[0][0] != int64(2) {
		t.Errorf("after reopen: %v", got)
	}
}

// model is what a table should hold, row by row.
type modelRow struct {
	id   int64
	name string
	n    int64
	body string
}

func TestRandomChanges(t *testing.T) {
	syncFiles = false
	defer func() { syncFiles = true }()
	for _, size := range []int{512, 1024, 4096} {
		t.Run(fmt.Sprint(size), func(t *testing.T) {
			db, path := createTest(t, size)
			rng := rand.New(rand.NewSource(int64(size)))
			mustExec(t, db, "CREATE TABLE r (id INTEGER PRIMARY KEY, name TEXT COLLATE NOCASE, n INTEGER, body TEXT)")
			mustExec(t, db, "CREATE INDEX r_n ON r (n DESC, name)")
			mustExec(t, db, "CREATE UNIQUE INDEX r_name ON r (name)")
			mustExec(t, db, "CREATE INDEX r_body ON r (body) WHERE n > 50")
			model := map[int64]modelRow{}
			names := map[string]int64{}
			text := func() string {
				n := rng.Intn(40)
				if rng.Intn(10) == 0 {
					n = rng.Intn(3 * size) // overflow pages
				}
				b := make([]byte, n)
				for i := range b {
					b[i] = byte('a' + rng.Intn(26))
				}
				return string(b)
			}
			for step := 0; step < 1500; step++ {
				switch op := rng.Intn(10); {
				case op < 5: // insert
					id := int64(rng.Intn(3000)) + 1
					name := fmt.Sprintf("N%d", rng.Intn(4000))
					if _, ok := model[id]; ok {
						continue
					}
					if _, ok := names[strings.ToLower(name)]; ok {
						_, err := db.Exec("INSERT INTO r VALUES (?, ?, ?, ?)", []Value{id, strings.ToLower(name), int64(1), "x"})
						if err == nil || !strings.Contains(err.Error(), "UNIQUE constraint failed: r.name") {
							t.Fatalf("duplicate name: %v", err)
						}
						continue
					}
					row := modelRow{id, name, int64(rng.Intn(100)), text()}
					mustExec(t, db, "INSERT INTO r VALUES (?, ?, ?, ?)", row.id, row.name, row.n, row.body)
					model[id] = row
					names[strings.ToLower(name)] = id
				case op < 7: // delete some
					lo := int64(rng.Intn(3000))
					hi := lo + int64(rng.Intn(60))
					res := mustExec(t, db, "DELETE FROM r WHERE id BETWEEN ? AND ?", lo, hi)
					var n int64
					for id, row := range model {
						if id >= lo && id <= hi {
							delete(model, id)
							delete(names, strings.ToLower(row.name))
							n++
						}
					}
					if res.Changes != n {
						t.Fatalf("delete changed %d rows, want %d", res.Changes, n)
					}
				case op < 9: // update
					lo := int64(rng.Intn(3000))
					hi := lo + int64(rng.Intn(100))
					body := text()
					res := mustExec(t, db, "UPDATE r SET n = n + 7, body = ? WHERE id >= ? AND id <= ?", body, lo, hi)
					var n int64
					for id, row := range model {
						if id >= lo && id <= hi {
							row.n += 7
							row.body = body
							model[id] = row
							n++
						}
					}
					if res.Changes != n {
						t.Fatalf("update changed %d rows, want %d", res.Changes, n)
					}
				default: // move a row to a new rowid
					for id, row := range model {
						nid := id + 5000
						if _, taken := model[nid]; taken {
							break
						}
						mustExec(t, db, "UPDATE r SET id = ? WHERE id = ?", nid, id)
						delete(model, id)
						row.id = nid
						model[nid] = row
						names[strings.ToLower(row.name)] = nid
						break
					}
				}
				if step%250 == 249 {
					compareModel(t, db, model)
					mustCheck(t, db)
				}
			}
			compareModel(t, db, model)
			mustCheck(t, db)
			sqlite3Check(t, path)
			// Lookups through each index agree with the model.
			for _, row := range model {
				got := mustQuery(t, db, "SELECT id FROM r WHERE name = ?", strings.ToUpper(row.name))
				if len(got) != 1 || got[0][0] != row.id {
					t.Fatalf("name lookup %q: %v", row.name, got)
				}
				break
			}
			// Delete everything, row by row: the file is all free pages.
			mustExec(t, db, "DELETE FROM r WHERE id > 0")
			compareModel(t, db, map[int64]modelRow{})
			mustCheck(t, db)
			sqlite3Check(t, path)
		})
	}
}

func compareModel(t *testing.T, db *DB, model map[int64]modelRow) {
	t.Helper()
	rows := mustQuery(t, db, "SELECT id, name, n, body FROM r ORDER BY id")
	var ids []int64
	for id := range model {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	if len(rows) != len(ids) {
		t.Fatalf("table has %d rows, want %d", len(rows), len(ids))
	}
	for i, id := range ids {
		m := model[id]
		want := []Value{m.id, m.name, m.n, m.body}
		if !reflect.DeepEqual(rows[i], want) {
			t.Fatalf("row %d: got %v, want %v", id, rows[i], want)
		}
	}
}

// setReserved gives a new, empty database reserved bytes at the end of
// each page, as Apple's SQLite does.
func setReserved(t *testing.T, path string, n int) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	size := int(data[16])<<8 | int(data[17])
	data[20] = byte(n)
	data[105], data[106] = byte((size-n)>>8), byte(size-n)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestReservedBytes(t *testing.T) {
	syncFiles = false
	defer func() { syncFiles = true }()
	path := filepath.Join(t.TempDir(), "r.db")
	db, err := Create(path)
	if err != nil {
		t.Fatal(err)
	}
	db.Close()
	setReserved(t, path, 12)
	db, err = Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	mustExec(t, db, "CREATE TABLE r (id INTEGER PRIMARY KEY, v TEXT)")
	mustExec(t, db, "CREATE INDEX r_v ON r (v)")
	mustExec(t, db, "BEGIN")
	for i := 0; i < 500; i++ {
		mustExec(t, db, "INSERT INTO r (v) VALUES (printf('%.*c', ?, 'x'))", int64(i*13%3000))
	}
	mustExec(t, db, "COMMIT")
	mustExec(t, db, "DELETE FROM r WHERE id % 4 = 0")
	mustCheck(t, db)
	sqlite3Check(t, path)
	// Reserved bytes that hold something (a checksum extension's) stop
	// writing.
	db.Close()
	data, _ := os.ReadFile(path)
	data[4096-1] = 0x5a
	os.WriteFile(path, data, 0o644)
	db, err = Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("UPDATE r SET v = 'y' WHERE id = 1", nil); err == nil || !strings.Contains(err.Error(), "extension") {
		t.Errorf("writing a page with used reserved bytes: %v", err)
	}
}

// TestWriteSQLiteMadeFile changes a file the sqlite3 tool made.
func TestWriteSQLiteMadeFile(t *testing.T) {
	syncFiles = false
	defer func() { syncFiles = true }()
	data, err := os.ReadFile(fixturePath(1024))
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "f.db")
	os.WriteFile(path, data, 0o644)
	db, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	mustExec(t, db, "INSERT INTO books (sku, title, price) VALUES ('B7', 'New', 10)")
	mustExec(t, db, "UPDATE books SET price = price + 1 WHERE price IS NOT NULL")
	mustExec(t, db, "DELETE FROM nums WHERE n % 7 = 0")
	mustExec(t, db, "UPDATE big SET body = 'short now' WHERE k = 2")
	mustExec(t, db, "INSERT INTO authors (name, country) VALUES ('Ursula Le Guin', 'x')")
	mustExec(t, db, "ALTER TABLE later ADD COLUMN c TEXT DEFAULT 'c'")
	mustExec(t, db, "CREATE INDEX nums_t ON nums (t)")
	if _, err := db.Exec("INSERT INTO authors (name) VALUES ('ISAAC ASIMOV')", nil); err == nil {
		t.Error("NOCASE UNIQUE let a duplicate in")
	}
	got := mustQuery(t, db, "SELECT count(*), sum(price) FROM books")
	if !reflect.DeepEqual(got[0], []Value{int64(7), int64(5992)}) {
		t.Errorf("books: %v", got)
	}
	mustCheck(t, db)
	sqlite3Check(t, path)
}
