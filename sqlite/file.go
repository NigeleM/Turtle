// Package sqlite reads and writes SQLite database files, written from
// scratch on the standard library (no cgo, no third-party code). It
// follows the file format documented at https://www.sqlite.org/fileformat2.html:
// a header, fixed-size pages, table and index B-trees of records, overflow
// pages, a free-page list, and a rollback journal that makes every change
// all-or-nothing, even if the program or the computer stops halfway.
//
// Query runs SELECT statements (see select.go); Exec runs everything that
// changes the database (see exec.go).
package sqlite

import (
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"sync"
	"time"
	"unicode/utf16"
)

// Value is one SQLite value: nil (NULL), int64, float64, string (TEXT), or
// []byte (BLOB).
type Value = any

// DB is an open database file.
type DB struct {
	*dbFile
	closed bool
}

// dbFile is one database file, shared by every DB that opened it in this
// program, so their locks and changes never get in each other's way.
type dbFile struct {
	path       string
	f          *os.File
	readonly   bool
	refs       int
	lockLevel  int
	pageSize   int
	usable     int    // page size minus the reserved bytes at each page's end
	pageCount  uint32 // pages in the file
	encoding   uint32 // 1 UTF-8, 2 UTF-16le, 3 UTF-16be
	format     byte   // schema format number (header byte 44)
	counter    uint32 // file change counter when last read or written
	tables     map[string]*Table
	views      map[string]*View
	triggers   map[string][]*Trigger // by table or view name, lowercase
	trig       *trigCtx              // the trigger running now, if any
	attached   []*attachedDB         // ATTACH ... AS name
	names      []string              // table names, in schema order
	schemaRows []schemaRow
	cantWrite  string // why this file can't be changed, or ""
	wal        bool   // in WAL mode (see wal.go)
	autoVacuum bool   // has pointer-map pages (auto_vacuum or incremental)
	walSt      *walState

	// cache holds pages as they are in the file (or WAL), so a page is
	// read from the disk once. It's emptied whenever the file may have
	// changed under it: another program's commit (the change counter
	// moves), a WAL that changed, a repaired crash.
	cache map[uint32][]byte

	scratch []byte // where writeNode lays a page out

	lastRowid, changes, totalChanges int64 // for last_insert_rowid(), changes(), total_changes()

	// Settings (PRAGMA).
	foreignKeys bool          // enforce REFERENCES
	deferFKs    bool          // check foreign keys at COMMIT
	busy        time.Duration // how long to wait for a lock
	busySet     bool
	sync        int // PRAGMA synchronous + 1; 0: the default (FULL)

	// Foreign keys: broken references counted in this statement, and
	// deferred ones in this transaction; rows being deleted by a cascade.
	fkStmt, fkDeferred int
	fkDeleting         map[string]bool

	// The write transaction, while one is open.
	inWrite       bool
	explicit      bool              // BEGIN ... COMMIT, rather than one statement
	dirty         map[uint32][]byte // changed pages
	orig          map[uint32][]byte // their contents before the transaction, for the journal
	origCount     uint32            // pages before the transaction
	stmt          *stmtUndo         // the running statement's undo record
	savepoints    []*savepoint      // SAVEPOINT ..., innermost last
	schemaChanged bool
}

// stmtUndo holds what a statement changed, so a failed statement is
// undone without undoing the rest of its transaction.
type stmtUndo struct {
	pages      map[uint32][]byte // page contents before the statement (nil: wasn't changed yet)
	pageCount  uint32
	schema     bool
	fkDeferred int
}

// Error is a database problem: a bad query, an unknown table, a damaged
// file. Its message is meant for the person running the program.
type Error struct{ Msg string }

func (e *Error) Error() string { return e.Msg }

func errorf(format string, args ...any) error {
	return &Error{Msg: fmt.Sprintf(format, args...)}
}

const headerMagic = "SQLite format 3\x00"

// crashAt, in tests, stops a commit partway, as if the program died:
// "journal" after the journal is written, "pages" after half the pages.
var crashAt string

type simulatedCrash struct{}

// syncFiles makes every commit wait until the disk has the data. Tests
// that make thousands of commits turn it off.
var syncFiles = true

func syncFile(f *os.File) error {
	if !syncFiles {
		return nil
	}
	return f.Sync()
}

// syncsOff is PRAGMA synchronous = OFF: don't wait for the disk.
func (f *dbFile) syncsOff() bool { return f.sync == 1 }

// sqliteVersion is written in the header as the version that last changed
// the file; Turtle writes files the way SQLite 3.46 does.
const sqliteVersion = 3046001

var (
	openMu    sync.Mutex
	openFiles = map[string]*dbFile{}
)

func fileKey(path string) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		return path
	}
	if real, err := filepath.EvalSymlinks(abs); err == nil {
		return real
	}
	return abs
}

// Open opens an existing SQLite file for reading and writing (or only
// reading, if the file is read-only).
func Open(path string) (*DB, error) {
	openMu.Lock()
	defer openMu.Unlock()
	key := fileKey(path)
	if f, ok := openFiles[key]; ok {
		f.refs++
		return &DB{dbFile: f}, nil
	}
	f, err := os.OpenFile(path, os.O_RDWR, 0)
	readonly := false
	if err != nil {
		if !errors.Is(err, os.ErrPermission) {
			return nil, err
		}
		f, err = os.Open(path)
		if err != nil {
			return nil, err
		}
		readonly = true
	}
	df := &dbFile{path: path, f: f, readonly: readonly, refs: 1}
	if err := df.openRead(); err != nil {
		df.unlock(lockNone)
		f.Close()
		return nil, err
	}
	df.unlock(lockNone)
	openFiles[key] = df
	return &DB{dbFile: df}, nil
}

// Create makes a new, empty database file (4096-byte pages, UTF-8) and
// opens it. It fails if the file already exists.
func Create(path string) (*DB, error) {
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return nil, err
	}
	const size = 4096
	p := make([]byte, size)
	copy(p, headerMagic)
	binary.BigEndian.PutUint16(p[16:], size)
	p[18], p[19] = 1, 1 // rollback journal, not WAL
	p[21], p[22], p[23] = 64, 32, 32
	binary.BigEndian.PutUint32(p[24:], 1) // change counter
	binary.BigEndian.PutUint32(p[28:], 1) // pages
	binary.BigEndian.PutUint32(p[44:], 4) // schema format
	binary.BigEndian.PutUint32(p[56:], 1) // UTF-8
	binary.BigEndian.PutUint32(p[92:], 1)
	binary.BigEndian.PutUint32(p[96:], sqliteVersion)
	p[100] = leafTable // the empty sqlite_schema table
	binary.BigEndian.PutUint16(p[105:], size)
	_, err = f.Write(p)
	if err == nil {
		err = f.Sync()
	}
	f.Close()
	if err != nil {
		os.Remove(path)
		return nil, err
	}
	syncDir(filepath.Dir(path))
	return Open(path)
}

// Close closes the database. An unfinished transaction is rolled back.
func (db *DB) Close() error {
	if db.closed {
		return nil
	}
	db.closed = true
	openMu.Lock()
	defer openMu.Unlock()
	db.refs--
	if db.refs > 0 {
		return nil
	}
	if db.inWrite {
		db.rollbackTx()
	}
	for _, a := range db.attached {
		a.db.closeLocked()
	}
	db.attached = nil
	db.unlock(lockNone)
	delete(openFiles, fileKey(db.path))
	return db.f.Close()
}

// openRead takes a SHARED lock, recovers from a crash if a journal says
// one happened, and reads the header and schema if they changed.
func (f *dbFile) openRead() error {
	if f.lockLevel >= lockShared {
		return nil
	}
	if err := f.lock(lockShared); err != nil {
		return err
	}
	if err := f.recoverJournal(); err != nil {
		return err
	}
	h := make([]byte, 100)
	if _, err := f.f.ReadAt(h, 0); err == nil && (h[18] == 2 || h[19] == 2) {
		if err := f.walEnter(); err != nil {
			return err
		}
	}
	return f.refresh()
}

// beginRead makes the file readable for one statement; done gives the
// lock back unless a transaction is open.
func (f *dbFile) beginRead() (done func(), err error) {
	if f.lockLevel >= lockShared {
		if !f.inWrite {
			if err := f.refresh(); err != nil {
				return nil, err
			}
		}
		return func() {}, nil
	}
	if err := f.openRead(); err != nil {
		f.unlock(lockNone)
		return nil, err
	}
	return func() {
		if !f.inWrite {
			f.unlock(lockNone)
		}
	}, nil
}

// refresh rereads the header, and the schema when another program has
// changed the file since.
func (f *dbFile) refresh() error {
	if f.inWrite {
		return nil
	}
	h := make([]byte, 100)
	if _, err := f.f.ReadAt(h, 0); err != nil && err != io.EOF {
		return err
	}
	walChanged := false
	if f.walSt != nil && f.walSt.shm != nil {
		before := f.walSt.walSize
		if err := f.walRead(); err != nil {
			return err
		}
		walChanged = f.walSt.walSize != before
		if f.walSt.pages != nil {
			ps := int(binary.BigEndian.Uint16(h[16:]))
			if ps == 1 {
				ps = 65536
			}
			p1 := make([]byte, ps)
			if f.pageSize == 0 {
				f.pageSize = ps
			}
			if f.walPage(1, p1) {
				copy(h, p1[:100])
			}
		}
	}
	if f.tables != nil && !walChanged && binary.BigEndian.Uint32(h[24:]) == f.counter {
		return nil
	}
	f.clearCache()
	if err := f.readHeader(h); err != nil {
		return err
	}
	return f.loadSchema()
}

func (f *dbFile) readHeader(h []byte) error {
	if string(h[:16]) != headerMagic {
		return errorf("%s isn't a SQLite database", f.path)
	}
	size := int(binary.BigEndian.Uint16(h[16:]))
	if size == 1 {
		size = 65536
	}
	if size < 512 || size > 65536 || size&(size-1) != 0 {
		return errorf("%s: bad page size %d", f.path, size)
	}
	f.pageSize = size
	f.usable = size - int(h[20])
	if f.usable < 480 {
		return errorf("%s: too little usable space per page", f.path)
	}
	f.counter = binary.BigEndian.Uint32(h[24:])
	// The page count in the header is only trusted when it was written by
	// a SQLite that also bumped the version-valid-for number.
	f.pageCount = binary.BigEndian.Uint32(h[28:])
	if f.walSt != nil && f.walSt.pages != nil && f.walSt.dbSize > 0 {
		f.pageCount = f.walSt.dbSize
	} else if binary.BigEndian.Uint32(h[92:]) != f.counter || f.pageCount == 0 {
		info, err := f.f.Stat()
		if err != nil {
			return err
		}
		f.pageCount = uint32(info.Size() / int64(size))
	}
	f.encoding = binary.BigEndian.Uint32(h[56:])
	if f.encoding == 0 {
		f.encoding = 1
	}
	if f.encoding > 3 {
		return errorf("%s: unknown text encoding %d", f.path, f.encoding)
	}
	f.format = h[47]
	f.cantWrite = ""
	f.wal = h[18] == 2 || h[19] == 2
	f.autoVacuum = binary.BigEndian.Uint32(h[52:]) != 0
	switch {
	case h[19] > 2:
		f.cantWrite = "it was made by a newer SQLite"
	case binary.BigEndian.Uint32(h[52:]) != 0:
		f.cantWrite = "it uses auto_vacuum, which isn't supported for writing yet"
	case f.encoding != 1:
		f.cantWrite = "it stores text as UTF-16, which isn't supported for writing yet"
	case f.format > 4:
		f.cantWrite = "its schema format is newer than SQLite 3"
	}
	return nil
}

func (f *dbFile) syncJournal(jf *os.File) error {
	if f.syncsOff() {
		return nil
	}
	return syncFile(jf)
}

// pendingPage is the page holding the lock bytes, which never stores data.
func (f *dbFile) pendingPage() uint32 { return uint32(pendingByte/f.pageSize) + 1 }

// page returns page n (numbered from 1): the changed copy during a write
// transaction, else from the file. Callers must not change it; see
// writable.
func (f *dbFile) page(n uint32) ([]byte, error) {
	if n < 1 || n > f.pageCount {
		return nil, errorf("damaged database: page %d is outside the file (%d pages)", n, f.pageCount)
	}
	if p, ok := f.dirty[n]; ok {
		return p, nil
	}
	if p, ok := f.cache[n]; ok {
		return p, nil
	}
	buf := make([]byte, f.pageSize)
	if f.walPage(n, buf) {
		f.cachePut(n, buf)
		return buf, nil
	}
	if _, err := f.f.ReadAt(buf, int64(n-1)*int64(f.pageSize)); err != nil {
		if err == io.EOF && f.inWrite && n > f.origCount {
			return buf, nil
		}
		return nil, errorf("reading page %d: %v", n, err)
	}
	f.cachePut(n, buf)
	return buf, nil
}

const maxCachedPages = 8192

func (f *dbFile) clearCache() { f.cache = nil }

func (f *dbFile) cachePut(n uint32, p []byte) {
	if f.cache == nil || len(f.cache) >= maxCachedPages {
		f.cache = map[uint32][]byte{}
	}
	f.cache[n] = p
}

// mustPage is page for code that can't return an error.
func (f *dbFile) mustPage(n uint32) []byte {
	p, err := f.page(n)
	if err != nil {
		panic(err)
	}
	return p
}

// writable returns page n to change, keeping its old contents for the
// journal and for undoing the statement.
func (f *dbFile) writable(n uint32) []byte {
	if !f.inWrite {
		panic(errorf("internal error: writing outside a transaction"))
	}
	if n < 1 || n > f.pageCount {
		panic(errorf("internal error: page %d is outside the file", n))
	}
	buf, ok := f.dirty[n]
	save := func(u *stmtUndo) {
		if _, saved := u.pages[n]; !saved {
			if ok {
				u.pages[n] = append([]byte{}, buf...)
			} else {
				u.pages[n] = nil
			}
		}
	}
	if f.stmt != nil {
		save(f.stmt)
	}
	for _, sp := range f.savepoints {
		save(sp.undo)
	}
	if ok {
		return buf
	}
	buf = make([]byte, f.pageSize)
	if n <= f.origCount {
		if cached, ok := f.cache[n]; ok {
			copy(buf, cached)
		} else if _, err := f.f.ReadAt(buf, int64(n-1)*int64(f.pageSize)); err != nil && err != io.EOF {
			panic(errorf("reading page %d: %v", n, err))
		}
		// Some SQLite builds (Apple's) reserve bytes at the end of each
		// page and leave them zero; Turtle keeps them as they are. Bytes
		// that are in use belong to an extension (checksums,
		// encryption) that would see Turtle's changes as damage.
		for _, b := range buf[f.usable:] {
			if b != 0 {
				panic(errorf("%s can't be changed: its pages carry data for an extension (encryption or checksums) that Turtle doesn't support", f.path))
			}
		}
		f.orig[n] = append([]byte{}, buf...)
	}
	f.dirty[n] = buf
	return buf
}

// beginWrite starts a write transaction: RESERVED lock, fresh header.
func (f *dbFile) beginWrite() error {
	if f.inWrite {
		return nil
	}
	if f.readonly {
		return errorf("%s is read-only, so it can't be changed", f.path)
	}
	if err := f.openRead(); err != nil {
		f.unlock(lockNone)
		return err
	}
	if err := f.lock(lockReserved); err != nil {
		f.unlock(lockNone)
		return err
	}
	// Another program may have written between our SHARED and RESERVED.
	if err := f.refresh(); err != nil {
		f.unlock(lockNone)
		return err
	}
	if f.cantWrite != "" {
		f.unlock(lockNone)
		return errorf("%s can't be changed: %s", f.path, f.cantWrite)
	}
	if f.wal && f.walSt != nil && f.walSt.pages != nil {
		// Changes waiting in the WAL go into the file first.
		if err := f.lock(lockExclusive); err != nil {
			f.unlock(lockNone)
			return err
		}
		if err := f.walCheckpoint(); err != nil {
			f.unlock(lockNone)
			return err
		}
		f.unlock(lockShared)
		if err := f.lock(lockReserved); err != nil {
			f.unlock(lockNone)
			return err
		}
	}
	f.inWrite = true
	f.dirty = map[uint32][]byte{}
	f.orig = map[uint32][]byte{}
	f.origCount = f.pageCount
	f.schemaChanged = false
	return nil
}

// commit makes the transaction's changes permanent: journal first, then
// the database file, then the journal is deleted. If the journal can't be
// written, or other programs keep reading for longer than busyTimeout,
// nothing is written and the transaction stays open: COMMIT can be tried
// again, or ROLLBACK.
func (f *dbFile) commit() error {
	if !f.inWrite {
		return nil
	}
	if len(f.dirty) == 0 {
		f.endWrite()
		return nil
	}
	p1 := f.writable(1)
	counter := binary.BigEndian.Uint32(p1[24:]) + 1
	binary.BigEndian.PutUint32(p1[24:], counter)
	binary.BigEndian.PutUint32(p1[28:], f.pageCount)
	binary.BigEndian.PutUint32(p1[92:], counter)
	binary.BigEndian.PutUint32(p1[96:], sqliteVersion)
	if f.schemaChanged {
		binary.BigEndian.PutUint32(p1[40:], binary.BigEndian.Uint32(p1[40:])+1)
		if binary.BigEndian.Uint32(p1[44:]) == 0 {
			binary.BigEndian.PutUint32(p1[44:], 4)
		}
	}
	if f.wal {
		if err := f.lock(lockExclusive); err != nil {
			return err
		}
		pages := make([]uint32, 0, len(f.dirty))
		for n := range f.dirty {
			if n <= f.pageCount {
				pages = append(pages, n)
			}
		}
		sort.Slice(pages, func(i, j int) bool { return pages[i] < pages[j] })
		if err := f.walCommit(pages); err != nil {
			return err
		}
		f.endWrite()
		f.counter = counter
		return nil
	}
	if err := f.writeJournal(); err != nil {
		return err
	}
	if crashAt == "journal" {
		panic(simulatedCrash{})
	}
	if err := f.lock(lockExclusive); err != nil {
		os.Remove(f.journalPath())
		return err
	}
	pages := make([]uint32, 0, len(f.dirty))
	for n := range f.dirty {
		pages = append(pages, n)
	}
	sort.Slice(pages, func(i, j int) bool { return pages[i] < pages[j] })
	for i, n := range pages {
		if crashAt == "pages" && i == len(pages)/2 {
			panic(simulatedCrash{})
		}
		if _, err := f.f.WriteAt(f.dirty[n], int64(n-1)*int64(f.pageSize)); err != nil {
			return f.writeFailed(err)
		}
	}
	if err := f.f.Truncate(int64(f.pageCount) * int64(f.pageSize)); err != nil {
		return f.writeFailed(err)
	}
	if !f.syncsOff() {
		if err := syncFile(f.f); err != nil {
			return f.writeFailed(err)
		}
	}
	f.keepWritten()
	f.endWrite()
	if err := os.Remove(f.journalPath()); err != nil {
		f.tables = nil
		return err
	}
	if syncFiles {
		syncDir(filepath.Dir(f.path))
	}
	f.counter = counter
	return nil
}

// writeFailed handles a failed database write: the journal stays, so the
// next open puts the old pages back.
func (f *dbFile) writeFailed(err error) error {
	f.clearCache()
	f.endWrite()
	f.tables = nil // reread everything next time
	return errorf("writing %s failed (%v); the change will be undone when the file is next opened", f.path, err)
}

// keepWritten puts the pages a commit just wrote into the cache.
func (f *dbFile) keepWritten() {
	for n, p := range f.dirty {
		if n <= f.pageCount {
			f.cachePut(n, p)
		}
	}
	for n := range f.cache {
		if n > f.pageCount {
			delete(f.cache, n)
		}
	}
}

func (f *dbFile) endWrite() {
	f.fkDeferred, f.fkStmt = 0, 0
	f.inWrite, f.explicit = false, false
	f.dirty, f.orig, f.stmt, f.savepoints = nil, nil, nil, nil
	f.unlock(lockNone)
}

// rollbackTx throws away the transaction's changes.
func (f *dbFile) rollbackTx() {
	if !f.inWrite {
		return
	}
	schema := f.schemaChanged
	f.inWrite, f.explicit = false, false
	f.dirty, f.orig, f.stmt, f.savepoints = nil, nil, nil, nil
	_ = schema
	f.fkDeferred, f.fkStmt = 0, 0
	h := make([]byte, 100)
	f.f.ReadAt(h, 0)
	f.readHeader(h)
	f.loadSchema()
	f.unlock(lockNone)
}

// ---- statements inside a transaction ----

// savepoint is SAVEPOINT name: a point inside a transaction to go back to.
type savepoint struct {
	name      string
	undo      *stmtUndo
	startedTx bool // it began the transaction, so releasing it commits
}

func (f *dbFile) newUndo() *stmtUndo {
	return &stmtUndo{pages: map[uint32][]byte{}, pageCount: f.pageCount, schema: f.schemaChanged, fkDeferred: f.fkDeferred}
}

func (f *dbFile) startStatement() {
	f.stmt = &stmtUndo{pages: map[uint32][]byte{}, pageCount: f.pageCount, schema: f.schemaChanged, fkDeferred: f.fkDeferred}
	f.fkStmt = 0
}

// undoStatement puts back every page the statement changed.
func (f *dbFile) undoStatement() {
	if f.stmt == nil {
		return
	}
	f.undo(f.stmt)
	f.stmt = nil
}

// undo puts back the pages and state an undo record kept.
func (f *dbFile) undo(u *stmtUndo) {
	f.stmt = u
	for n, old := range f.stmt.pages {
		if old == nil {
			delete(f.dirty, n)
		} else {
			f.dirty[n] = old
		}
	}
	f.pageCount = f.stmt.pageCount
	f.fkDeferred, f.fkStmt = f.stmt.fkDeferred, 0
	changed := f.schemaChanged
	f.schemaChanged = f.stmt.schema
	f.stmt = nil
	if changed {
		f.loadSchema()
	}
}

// ---- the rollback journal ----

var journalMagic = []byte{0xd9, 0xd5, 0x05, 0xf9, 0x20, 0xa1, 0x63, 0xd7}

const journalSector = 512

func (f *dbFile) journalPath() string { return f.path + "-journal" }

func journalChecksum(nonce uint32, data []byte) uint32 {
	sum := nonce
	for i := len(data) - 200; i > 0; i -= 200 {
		sum += uint32(data[i])
	}
	return sum
}

// writeJournal writes the original contents of every changed page that
// existed before the transaction, in SQLite's journal format, and makes
// it durable before the database file is touched.
func (f *dbFile) writeJournal() error {
	jf, err := os.OpenFile(f.journalPath(), os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return errorf("can't write the journal %s: %v", f.journalPath(), err)
	}
	// On failure: close, then delete (Windows can't delete an open file).
	abandon := func(err error) error {
		jf.Close()
		os.Remove(f.journalPath())
		return err
	}
	var nb [4]byte
	rand.Read(nb[:])
	nonce := binary.BigEndian.Uint32(nb[:])
	hdr := make([]byte, journalSector)
	copy(hdr, journalMagic)
	binary.BigEndian.PutUint32(hdr[12:], nonce)
	binary.BigEndian.PutUint32(hdr[16:], f.origCount)
	binary.BigEndian.PutUint32(hdr[20:], journalSector)
	binary.BigEndian.PutUint32(hdr[24:], uint32(f.pageSize))
	pages := make([]uint32, 0, len(f.orig))
	for n := range f.orig {
		pages = append(pages, n)
	}
	sort.Slice(pages, func(i, j int) bool { return pages[i] < pages[j] })
	buf := append([]byte{}, hdr...)
	for _, n := range pages {
		data := f.orig[n]
		var rec [4]byte
		binary.BigEndian.PutUint32(rec[:], n)
		buf = append(buf, rec[:]...)
		buf = append(buf, data...)
		binary.BigEndian.PutUint32(rec[:], journalChecksum(nonce, data))
		buf = append(buf, rec[:]...)
	}
	if _, err := jf.Write(buf); err != nil {
		return abandon(errorf("can't write the journal %s: %v", f.journalPath(), err))
	}
	if err := f.syncJournal(jf); err != nil {
		return abandon(err)
	}
	if syncFiles {
		syncDir(filepath.Dir(f.path))
	}
	// Only now say how many pages it holds: a journal cut short by a
	// crash before this point has no records, so nothing is played back.
	binary.BigEndian.PutUint32(nb[:], uint32(len(pages)))
	if _, err := jf.WriteAt(nb[:], 8); err != nil {
		return abandon(err)
	}
	if err := f.syncJournal(jf); err != nil {
		return abandon(err)
	}
	return jf.Close()
}

// recoverJournal checks for a journal left by a program (Turtle or any
// SQLite) that stopped in the middle of writing, and puts the original
// pages back. Called with a SHARED lock.
func (f *dbFile) recoverJournal() error {
	info, err := os.Stat(f.journalPath())
	if err != nil || info.Size() == 0 {
		return nil
	}
	if osCheckReserved(f.f) {
		return nil // its writer is still running
	}
	jf, err := os.Open(f.journalPath())
	if err != nil {
		return nil
	}
	// Closed before the journal is deleted: Windows can't delete an open
	// file.
	closed := false
	closeJournal := func() {
		if !closed {
			jf.Close()
			closed = true
		}
	}
	defer closeJournal()
	first := make([]byte, 1)
	if _, err := jf.ReadAt(first, 0); err != nil || first[0] == 0 {
		return nil
	}
	if f.readonly {
		return errorf("%s has an unfinished change from a program that stopped while writing, and can't be repaired because it's read-only", f.path)
	}
	if err := f.lock(lockExclusive); err != nil {
		return err
	}
	defer f.unlock(lockShared)
	// Someone else may have recovered it while we waited.
	if _, err := os.Stat(f.journalPath()); err != nil {
		return nil
	}
	f.clearCache()
	if err := f.playback(jf, info.Size()); err != nil {
		return err
	}
	closeJournal()
	if err := os.Remove(f.journalPath()); err != nil {
		return err
	}
	syncDir(filepath.Dir(f.path))
	f.tables = nil
	return nil
}

func (f *dbFile) playback(jf *os.File, size int64) error {
	h := make([]byte, 100)
	if _, err := f.f.ReadAt(h, 0); err != nil {
		return errorf("%s: can't read the header to repair it: %v", f.path, err)
	}
	pageSize := int64(binary.BigEndian.Uint16(h[16:]))
	if pageSize == 1 {
		pageSize = 65536
	}
	var off int64
	dbSize := int64(-1)
	for off+28 <= size {
		hdr := make([]byte, 28)
		if _, err := jf.ReadAt(hdr, off); err != nil {
			break
		}
		if string(hdr[:8]) != string(journalMagic) {
			break
		}
		nRec := int64(binary.BigEndian.Uint32(hdr[8:]))
		nonce := binary.BigEndian.Uint32(hdr[12:])
		segSize := int64(binary.BigEndian.Uint32(hdr[16:]))
		sector := int64(binary.BigEndian.Uint32(hdr[20:]))
		psize := int64(binary.BigEndian.Uint32(hdr[24:]))
		if sector < 32 || sector > 65536 || psize != pageSize {
			break
		}
		if dbSize < 0 {
			dbSize = segSize
		}
		rec := off + sector
		computed := nRec == 0xFFFFFFFF
		if computed {
			nRec = (size - rec) / (psize + 8)
		}
		data := make([]byte, psize)
		var w [4]byte
		stop := false
		for i := int64(0); i < nRec; i++ {
			if rec+8+psize > size {
				stop = true
				break
			}
			jf.ReadAt(w[:], rec)
			pgno := binary.BigEndian.Uint32(w[:])
			jf.ReadAt(data, rec+4)
			jf.ReadAt(w[:], rec+4+psize)
			if binary.BigEndian.Uint32(w[:]) != journalChecksum(nonce, data) {
				stop = true
				break
			}
			if pgno > 0 && int64(pgno) <= segSize {
				if _, err := f.f.WriteAt(data, int64(pgno-1)*psize); err != nil {
					return errorf("%s: repairing failed: %v", f.path, err)
				}
			}
			rec += 8 + psize
		}
		if stop || computed {
			break
		}
		off = (rec + sector - 1) / sector * sector
	}
	if dbSize >= 0 {
		if err := f.f.Truncate(dbSize * pageSize); err != nil {
			return err
		}
	}
	return f.f.Sync()
}

func (f *dbFile) isWindows() bool { return runtime.GOOS == "windows" }

// ---- reading B-trees ----

// B-tree page types.
const (
	interiorIndex = 0x02
	interiorTable = 0x05
	leafIndex     = 0x0a
	leafTable     = 0x0d
)

// scanTable calls fn with every row of the table B-tree rooted at root,
// in rowid order.
func (f *dbFile) scanTable(root uint32, fn func(rowid int64, payload []byte) error) error {
	return f.scanPage(root, 0, fn)
}

func (f *dbFile) scanPage(n uint32, depth int, fn func(int64, []byte) error) error {
	if depth > 40 {
		return errorf("damaged database: B-tree deeper than 40 levels")
	}
	p, err := f.page(n)
	if err != nil {
		return err
	}
	h := 0
	if n == 1 {
		h = 100
	}
	kind := p[h]
	cells := int(binary.BigEndian.Uint16(p[h+3:]))
	switch kind {
	case interiorTable:
		ptrs := h + 12
		for i := 0; i < cells; i++ {
			off := int(binary.BigEndian.Uint16(p[ptrs+2*i:]))
			if off+4 > len(p) {
				return errorf("damaged database: bad cell on page %d", n)
			}
			if err := f.scanPage(binary.BigEndian.Uint32(p[off:]), depth+1, fn); err != nil {
				return err
			}
		}
		return f.scanPage(binary.BigEndian.Uint32(p[h+8:]), depth+1, fn)
	case leafTable:
		ptrs := h + 8
		for i := 0; i < cells; i++ {
			off := int(binary.BigEndian.Uint16(p[ptrs+2*i:]))
			size, k := varint(p, off)
			rowid, k2 := varint(p, off+k)
			payload, err := f.payload(p, off+k+k2, int64(size), true)
			if err != nil {
				return err
			}
			if err := fn(int64(rowid), payload); err != nil {
				return err
			}
		}
		return nil
	case interiorIndex, leafIndex:
		return errorf("this table is stored as an index (WITHOUT ROWID), which isn't supported yet")
	}
	return errorf("damaged database: page %d has unknown type %d", n, kind)
}

// localSize is how much of a payload of size bytes is stored in the cell
// itself; the rest continues on overflow pages.
func (f *dbFile) localSize(size int64, table bool) int64 {
	u := int64(f.usable)
	maxLocal := u - 35
	if !table {
		maxLocal = (u-12)*64/255 - 23
	}
	minLocal := (u-12)*32/255 - 23
	if size <= maxLocal {
		return size
	}
	local := minLocal + (size-minLocal)%(u-4)
	if local > maxLocal {
		local = minLocal
	}
	return local
}

// payload returns a cell's whole payload: the part stored on the page, plus
// any overflow pages it continues on.
func (f *dbFile) payload(p []byte, off int, size int64, table bool) ([]byte, error) {
	u := int64(f.usable)
	local := f.localSize(size, table)
	if off+int(local) > len(p) {
		return nil, errorf("damaged database: cell runs past its page")
	}
	out := make([]byte, 0, size)
	out = append(out, p[off:off+int(local)]...)
	if local == size {
		return out, nil
	}
	if off+int(local)+4 > len(p) {
		return nil, errorf("damaged database: cell runs past its page")
	}
	next := binary.BigEndian.Uint32(p[off+int(local):])
	for pages := 0; int64(len(out)) < size; pages++ {
		if next == 0 || pages > int(f.pageCount) {
			return nil, errorf("damaged database: overflow chain ends early")
		}
		op, err := f.page(next)
		if err != nil {
			return nil, err
		}
		want := size - int64(len(out))
		if want > u-4 {
			want = u - 4
		}
		out = append(out, op[4:4+want]...)
		next = binary.BigEndian.Uint32(op)
	}
	return out, nil
}

// varint reads a SQLite variable-length integer at p[off:]: 1 to 9 bytes,
// big-endian, 7 bits per byte except the ninth, which gives all 8.
func varint(p []byte, off int) (uint64, int) {
	var v uint64
	for i := 0; i < 8; i++ {
		if off+i >= len(p) {
			return v, i
		}
		b := p[off+i]
		v = v<<7 | uint64(b&0x7f)
		if b < 0x80 {
			return v, i + 1
		}
	}
	if off+8 >= len(p) {
		return v, 8
	}
	return v<<8 | uint64(p[off+8]), 9
}

// decodeRecord turns a record (header of serial types, then the values)
// into values.
func (f *dbFile) decodeRecord(rec []byte) ([]Value, error) {
	hsize, n := varint(rec, 0)
	if hsize > uint64(len(rec)) || n == 0 {
		return nil, errorf("damaged database: bad record header")
	}
	var types []uint64
	for pos := n; pos < int(hsize); {
		t, k := varint(rec, pos)
		if k == 0 {
			return nil, errorf("damaged database: bad record header")
		}
		types = append(types, t)
		pos += k
	}
	vals := make([]Value, len(types))
	body := int(hsize)
	for i, t := range types {
		size := serialSize(t)
		if body+size > len(rec) {
			return nil, errorf("damaged database: record shorter than its header says")
		}
		b := rec[body : body+size]
		switch {
		case t == 0:
			vals[i] = nil
		case t >= 1 && t <= 6:
			vals[i] = bigEndianInt(b)
		case t == 7:
			vals[i] = math.Float64frombits(binary.BigEndian.Uint64(b))
		case t == 8:
			vals[i] = int64(0)
		case t == 9:
			vals[i] = int64(1)
		case t >= 12 && t%2 == 0:
			vals[i] = append([]byte{}, b...)
		case t >= 13:
			vals[i] = f.text(b)
		default:
			return nil, errorf("damaged database: unknown value type %d", t)
		}
		body += size
	}
	return vals, nil
}

func serialSize(t uint64) int {
	switch {
	case t <= 4:
		return []int{0, 1, 2, 3, 4}[t]
	case t == 5:
		return 6
	case t == 6 || t == 7:
		return 8
	case t < 12:
		return 0
	}
	return int((t - 12) / 2)
}

// bigEndianInt reads a 1-8 byte two's complement big-endian integer.
func bigEndianInt(b []byte) int64 {
	var v int64
	if len(b) > 0 && b[0]&0x80 != 0 {
		v = -1
	}
	for _, c := range b {
		v = v<<8 | int64(c)
	}
	return v
}

func (f *dbFile) text(b []byte) string {
	if f.encoding == 1 {
		return string(b)
	}
	u := make([]uint16, len(b)/2)
	for i := range u {
		if f.encoding == 2 {
			u[i] = binary.LittleEndian.Uint16(b[2*i:])
		} else {
			u[i] = binary.BigEndian.Uint16(b[2*i:])
		}
	}
	return string(utf16.Decode(u))
}

// errStop ends a scan early without an error.
var errStop = errors.New("stop")

// journalMode is what PRAGMA journal_mode reports.
func (f *dbFile) journalMode() string {
	if f.wal {
		return "wal"
	}
	return "delete"
}

// setJournalMode changes the journal mode: DELETE (a rollback journal)
// or WAL. The mode is in the header, so other programs see it too.
func (db *DB) setJournalMode(mode string) (string, error) {
	f := db.dbFile
	var v byte
	switch mode {
	case "delete":
		v = 1
	case "wal":
		v = 2
	case "truncate", "persist", "memory", "off":
		return "", errorf("journal_mode %s isn't supported; Turtle uses delete or wal", mode)
	default:
		return f.journalMode(), nil
	}
	if f.inWrite {
		return "", errorf("cannot change into %s mode from within a transaction", mode)
	}
	if _, err := db.execOneRows(&byteWrite{off: 18, v: []byte{v, v}}, [][]Value{nil}); err != nil {
		return "", err
	}
	f.wal = v == 2
	return mode, nil
}

// byteWrite is a statement that sets header bytes.
type byteWrite struct {
	off int
	v   []byte
}
