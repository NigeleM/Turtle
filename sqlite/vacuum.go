package sqlite

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// VACUUM rebuilds the database compactly: every table copied row by row
// into fresh, full pages, every index rebuilt in order, and the free
// pages gone, so the file shrinks. VACUUM INTO 'file' writes that copy to
// a new file instead (a backup) and leaves the database as it is.

type vacuumStmt struct {
	into expr
}

func (db *DB) runVacuum(s *vacuumStmt, params []Value) error {
	f := db.dbFile
	if f.inWrite {
		return errorf("cannot VACUUM from within a transaction")
	}
	if s.into != nil {
		r := &runner{db: f, params: params, core: &corePlan{q: &queryPlan{db: f}, sc: &scope{}}}
		var target string
		err := func() (err error) {
			defer catch(&err)
			target = textValue(r.eval(s.into))
			return nil
		}()
		if err != nil {
			return err
		}
		if _, err := os.Stat(target); err == nil {
			return errorf("output file already exists: %s", target)
		}
		done, err := f.beginRead()
		if err != nil {
			return err
		}
		defer done()
		return f.copyInto(target)
	}
	// In place: build the copy beside the file, then replace every page
	// of the file with it, in one journaled transaction.
	tmp, err := os.CreateTemp(filepath.Dir(f.path), filepath.Base(f.path)+"-vacuum-*")
	if err != nil {
		return errorf("VACUUM can't make its working file: %v", err)
	}
	tmpPath := tmp.Name()
	tmp.Close()
	os.Remove(tmpPath)
	defer os.Remove(tmpPath)
	defer os.Remove(tmpPath + "-journal")
	if err := f.beginWrite(); err != nil {
		return err
	}
	if err := f.copyInto(tmpPath); err != nil {
		f.rollbackTx()
		return err
	}
	data, err := os.ReadFile(tmpPath)
	if err != nil {
		f.rollbackTx()
		return err
	}
	err = func() (err error) {
		defer catch(&err)
		newCount := uint32(len(data) / f.pageSize)
		// Pages the file loses go in the journal too, so a crash can
		// put them back.
		for n := newCount + 1; n <= f.pageCount; n++ {
			f.writable(n)
			delete(f.dirty, n)
		}
		if newCount > f.pageCount {
			f.pageCount = newCount
		}
		for n := uint32(1); n <= newCount; n++ {
			copy(f.writable(n), data[int(n-1)*f.pageSize:int(n)*f.pageSize])
		}
		f.pageCount = newCount
		f.schemaChanged = true
		return nil
	}()
	if err != nil {
		f.rollbackTx()
		return err
	}
	if err := f.commit(); err != nil {
		f.rollbackTx()
		return err
	}
	f.tables = nil
	return nil
}

// copyInto writes a compact copy of the database to a new file.
func (f *dbFile) copyInto(path string) error {
	out, err := createFile(path, f.pageSize)
	if err != nil {
		return errorf("VACUUM INTO %s: %v", path, err)
	}
	defer out.Close()
	d := out.dbFile
	if err := d.beginWrite(); err != nil {
		return err
	}
	err = func() (err error) {
		defer catch(&err)
		h := f.mustPage(1)
		p1 := d.writable(1)
		for _, off := range []int{44, 48, 56, 60, 64, 68} {
			copy(p1[off:off+4], h[off:off+4])
		}
		roots := map[string]uint32{}
		// Tables first, rows copied as they're stored, in rowid order.
		for _, r := range f.schemaRows {
			if r.kind != "table" || r.root == 0 {
				continue
			}
			if t := f.tables[strings.ToLower(r.name)]; t != nil && t.WithoutRowid {
				// Stored as an index: copy its entries in order.
				roots[strings.ToLower(r.name)] = f.copyIndex(d, schemaRow{root: r.root, name: "", tbl: ""})
				continue
			}
			root := d.newTree(true)
			roots[strings.ToLower(r.name)] = root
			ferr := f.scanTable(r.root, func(rowid int64, payload []byte) error {
				d.tablePut(root, rowid, payload)
				return nil
			})
			if ferr != nil {
				panic(ferr)
			}
		}
		// The schema, with the new roots; indexes rebuilt in order.
		for _, r := range f.schemaRows {
			root := uint32(0)
			switch {
			case r.kind == "table" && r.root != 0:
				root = roots[strings.ToLower(r.name)]
			case r.kind == "index" && r.root != 0:
				root = f.copyIndex(d, r)
			}
			var sql Value
			if r.sql != "" {
				sql = r.sql
			}
			d.addSchemaRow(r.kind, r.name, r.tbl, root, sql)
		}
		return nil
	}()
	if err != nil {
		d.rollbackTx()
		return err
	}
	return d.commit()
}

// copyIndex rebuilds an index in the copy, entries added in order.
func (f *dbFile) copyIndex(d *dbFile, r schemaRow) uint32 {
	t := f.tables[strings.ToLower(r.tbl)]
	var ix *Index
	if t != nil {
		for _, x := range t.Indexes {
			if strings.EqualFold(x.Name, r.name) {
				ix = x
			}
		}
	}
	root := d.newTree(false)
	if ix == nil {
		// An index Turtle can't read the definition of: copy its
		// entries as they are, in their order.
		var walk func(pg uint32)
		walk = func(pg uint32) {
			n := f.readNode(pg)
			for _, c := range n.cells {
				if !n.leaf() {
					walk(c.child)
				}
				d.indexAppend(root, f.cellPayload(n, c))
			}
			if !n.leaf() {
				walk(n.right)
			}
		}
		walk(r.root)
		return root
	}
	var keys [][]Value
	rr := &runner{db: f, core: f.rowPlan(t, "", false), cur: make([]*srcRow, 1)}
	f.scanTable(t.Root, func(rowid int64, payload []byte) error {
		vals := f.fullRow(t, rowid, payload, rr)
		if f.inIndex(t, ix, rowid, vals, nil) {
			keys = append(keys, f.indexKey(t, ix, rowid, vals))
		}
		return nil
	})
	sort.SliceStable(keys, func(i, j int) bool { return indexKeyCompare(keys[i], keys[j], ix) < 0 })
	for _, k := range keys {
		d.indexAppend(root, f.encodeRecord(k))
	}
	return root
}

// indexAppend adds an entry known to sort after every entry already in
// the index: at the end of its rightmost leaf.
func (f *dbFile) indexAppend(root uint32, payload []byte) {
	var path []pathStep
	n := f.readNode(root)
	for !n.leaf() {
		path = append(path, pathStep{n: n, pgno: n.pgno, idx: len(n.cells)})
		n = f.readNode(n.right)
	}
	n.cells = append(n.cells, f.indexCell(payload))
	f.balance(path, n, true)
}

// createFile makes a new, empty database file with the given page size.
func createFile(path string, pageSize int) (*DB, error) {
	db, err := Create(path)
	if err != nil {
		return nil, err
	}
	if pageSize == 4096 {
		return db, nil
	}
	db.Close()
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	p := make([]byte, pageSize)
	copy(p, data[:100])
	s := pageSize
	if s == 65536 {
		s = 1
	}
	binary.BigEndian.PutUint16(p[16:], uint16(s))
	p[100] = leafTable
	cs := pageSize
	if cs == 65536 {
		cs = 0
	}
	binary.BigEndian.PutUint16(p[105:], uint16(cs))
	if err := os.WriteFile(path, p, 0o644); err != nil {
		return nil, err
	}
	return Open(path)
}
