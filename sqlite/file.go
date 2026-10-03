// Package sqlite reads SQLite database files, written from scratch on the
// standard library (no cgo, no third-party code). It follows the file
// format documented at https://www.sqlite.org/fileformat2.html: a header,
// fixed-size pages, table B-trees of records, and overflow pages.
//
// This first version reads: it opens a .db file, lists its tables, and
// runs SELECT queries (see sql.go). Writing comes next.
package sqlite

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"unicode/utf16"
)

// Value is one SQLite value: nil (NULL), int64, float64, string (TEXT), or
// []byte (BLOB).
type Value = any

// DB is an open database file.
type DB struct {
	f         *os.File
	pageSize  int
	usable    int    // page size minus the reserved bytes at each page's end
	pageCount uint32 // pages in the file
	encoding  uint32 // 1 UTF-8, 2 UTF-16le, 3 UTF-16be
	tables    map[string]*Table
	names     []string // table names, in schema order
}

// Error is a database problem: a bad query, an unknown table, a damaged
// file. Its message is meant for the person running the program.
type Error struct{ Msg string }

func (e *Error) Error() string { return e.Msg }

func errorf(format string, args ...any) error {
	return &Error{Msg: fmt.Sprintf(format, args...)}
}

const headerMagic = "SQLite format 3\x00"

// Open opens an existing SQLite file for reading.
func Open(path string) (*DB, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	db := &DB{f: f}
	if err := db.readHeader(path); err != nil {
		f.Close()
		return nil, err
	}
	if err := db.loadSchema(); err != nil {
		f.Close()
		return nil, err
	}
	return db, nil
}

// Close closes the file.
func (db *DB) Close() error { return db.f.Close() }

func (db *DB) readHeader(path string) error {
	h := make([]byte, 100)
	if _, err := io.ReadFull(db.f, h); err != nil || string(h[:16]) != headerMagic {
		return errorf("%s isn't a SQLite database", path)
	}
	size := int(binary.BigEndian.Uint16(h[16:]))
	if size == 1 {
		size = 65536
	}
	if size < 512 || size > 65536 || size&(size-1) != 0 {
		return errorf("%s: bad page size %d", path, size)
	}
	db.pageSize = size
	db.usable = size - int(h[20])
	if db.usable < 480 {
		return errorf("%s: too little usable space per page", path)
	}
	// The page count in the header is only trusted when it was written by
	// a SQLite that also bumped the version-valid-for number.
	db.pageCount = binary.BigEndian.Uint32(h[28:])
	if binary.BigEndian.Uint32(h[92:]) != binary.BigEndian.Uint32(h[24:]) || db.pageCount == 0 {
		info, err := db.f.Stat()
		if err != nil {
			return err
		}
		db.pageCount = uint32(info.Size() / int64(size))
	}
	db.encoding = binary.BigEndian.Uint32(h[56:])
	if db.encoding == 0 {
		db.encoding = 1
	}
	if db.encoding > 3 {
		return errorf("%s: unknown text encoding %d", path, db.encoding)
	}
	// In WAL mode recent changes may still be in the -wal file, which this
	// reader doesn't apply: refuse rather than return stale rows.
	if h[18] == 2 || h[19] == 2 {
		if info, err := os.Stat(path + "-wal"); err == nil && info.Size() > 0 {
			return errorf("%s is in WAL mode with unsaved changes in %s-wal; close the program using it, or run PRAGMA wal_checkpoint, first", path, path)
		}
	}
	return nil
}

// page reads page n (numbered from 1).
func (db *DB) page(n uint32) ([]byte, error) {
	if n < 1 || n > db.pageCount {
		return nil, errorf("damaged database: page %d is outside the file (%d pages)", n, db.pageCount)
	}
	buf := make([]byte, db.pageSize)
	if _, err := db.f.ReadAt(buf, int64(n-1)*int64(db.pageSize)); err != nil {
		return nil, errorf("reading page %d: %v", n, err)
	}
	return buf, nil
}

// B-tree page types.
const (
	interiorIndex = 0x02
	interiorTable = 0x05
	leafIndex     = 0x0a
	leafTable     = 0x0d
)

// scanTable calls fn with every row of the table B-tree rooted at root,
// in rowid order.
func (db *DB) scanTable(root uint32, fn func(rowid int64, payload []byte) error) error {
	return db.scanPage(root, 0, fn)
}

func (db *DB) scanPage(n uint32, depth int, fn func(int64, []byte) error) error {
	if depth > 40 {
		return errorf("damaged database: B-tree deeper than 40 levels")
	}
	p, err := db.page(n)
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
			if err := db.scanPage(binary.BigEndian.Uint32(p[off:]), depth+1, fn); err != nil {
				return err
			}
		}
		return db.scanPage(binary.BigEndian.Uint32(p[h+8:]), depth+1, fn)
	case leafTable:
		ptrs := h + 8
		for i := 0; i < cells; i++ {
			off := int(binary.BigEndian.Uint16(p[ptrs+2*i:]))
			size, k := varint(p, off)
			rowid, k2 := varint(p, off+k)
			payload, err := db.payload(p, off+k+k2, int64(size), true)
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

// payload returns a cell's whole payload: the part stored on the page, plus
// any overflow pages it continues on.
func (db *DB) payload(p []byte, off int, size int64, table bool) ([]byte, error) {
	u := int64(db.usable)
	maxLocal := u - 35
	if !table {
		maxLocal = (u-12)*64/255 - 23
	}
	minLocal := (u-12)*32/255 - 23
	local := size
	if size > maxLocal {
		local = minLocal + (size-minLocal)%(u-4)
		if local > maxLocal {
			local = minLocal
		}
	}
	if off+int(local) > len(p) {
		return nil, errorf("damaged database: cell runs past its page")
	}
	out := make([]byte, 0, size)
	out = append(out, p[off:off+int(local)]...)
	if local == size {
		return out, nil
	}
	next := binary.BigEndian.Uint32(p[off+int(local):])
	for pages := 0; int64(len(out)) < size; pages++ {
		if next == 0 || pages > int(db.pageCount) {
			return nil, errorf("damaged database: overflow chain ends early")
		}
		op, err := db.page(next)
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
func (db *DB) decodeRecord(rec []byte) ([]Value, error) {
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
			vals[i] = db.text(b)
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

func (db *DB) text(b []byte) string {
	if db.encoding == 1 {
		return string(b)
	}
	u := make([]uint16, len(b)/2)
	for i := range u {
		if db.encoding == 2 {
			u[i] = binary.LittleEndian.Uint16(b[2*i:])
		} else {
			u[i] = binary.BigEndian.Uint16(b[2*i:])
		}
	}
	return string(utf16.Decode(u))
}

// errStop ends a scan early without an error.
var errStop = errors.New("stop")
