package sqlite

import (
	"errors"
	"io/fs"
	"strings"
)

// ATTACH 'other.db' AS other opens a second database file beside the
// main one: its tables are other.table (or just table, when the main
// database has none of that name), so a query can read both:
//
//	INSERT INTO books SELECT * FROM other.books
//
// Attached databases are read here, not changed: SQLite makes a change
// to several files all-or-nothing with an extra super-journal, which
// Turtle doesn't write, so it changes only the main file. A file that
// doesn't exist is made, as in SQLite.

type attachedDB struct {
	name string
	db   *DB
}

type attachStmt struct {
	path expr
	name string
}

type detachStmt struct{ name string }

func (p *sqlParser) parseAttach() *attachStmt {
	p.acceptWord("DATABASE")
	s := &attachStmt{path: p.parseExpr()}
	p.expectWord("AS")
	s.name = p.expectName()
	return s
}

func (db *DB) runAttach(s *attachStmt, params []Value) (err error) {
	defer catch(&err)
	f := db.dbFile
	switch strings.ToLower(s.name) {
	case "main", "temp":
		fail("database %s is already in use", s.name)
	}
	for _, a := range f.attached {
		if strings.EqualFold(a.name, s.name) {
			fail("database %s is already in use", s.name)
		}
	}
	if f.inWrite {
		fail("cannot ATTACH database within transaction")
	}
	r := &runner{db: f, params: params, core: &corePlan{q: &queryPlan{db: f}, sc: &scope{}}}
	path := textValue(r.eval(s.path))
	if path == "" || path == ":memory:" {
		fail("in-memory databases aren't supported")
	}
	other, oerr := Open(path)
	if errors.Is(oerr, fs.ErrNotExist) {
		other, oerr = Create(path)
	}
	if oerr != nil {
		return oerr
	}
	if other.dbFile == f {
		other.Close()
		fail("database %s is already the main database", path)
	}
	f.attached = append(f.attached, &attachedDB{name: s.name, db: other})
	return nil
}

func (db *DB) runDetach(s *detachStmt) error {
	f := db.dbFile
	for i, a := range f.attached {
		if strings.EqualFold(a.name, s.name) {
			if f.inWrite {
				return errorf("cannot DETACH database within transaction")
			}
			a.db.Close()
			f.attached = append(f.attached[:i], f.attached[i+1:]...)
			return nil
		}
	}
	return errorf("no such database: %s", s.name)
}

// readAttached makes the attached files readable for a statement.
func (f *dbFile) readAttached() (func(), error) {
	var dones []func()
	release := func() {
		for _, d := range dones {
			d()
		}
	}
	for _, a := range f.attached {
		done, err := a.db.beginRead()
		if err != nil {
			release()
			return nil, err
		}
		dones = append(dones, done)
	}
	return release, nil
}

// closeLocked closes an attached database (the open-files lock is held).
func (db *DB) closeLocked() {
	if db.closed {
		return
	}
	db.closed = true
	db.refs--
	if db.refs > 0 {
		return
	}
	db.unlock(lockNone)
	delete(openFiles, fileKey(db.path))
	db.f.Close()
}
