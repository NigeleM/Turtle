package sqlite

import (
	"encoding/binary"
	"math"
)

// Changing B-trees. A page is read into a node (its cells as byte
// strings), changed, and written back laid out from scratch, so pages
// never fragment. When a node no longer fits its page it splits in two
// (or more), giving its parent new dividers; when a non-root node runs
// out of cells it merges with a sibling. Every leaf stays at the same
// depth, as SQLite requires.

type node struct {
	pgno  uint32
	kind  byte
	cells []cell
	right uint32 // interior pages: the rightmost child
}

// cell is one entry of a node. Table interior cells are child + key;
// table leaf cells keep the whole cell in body (key is its rowid); index
// cells keep their payload part in body (child is added on interior pages).
type cell struct {
	child uint32
	key   int64
	body  []byte
}

func (n *node) leaf() bool  { return n.kind == leafTable || n.kind == leafIndex }
func (n *node) table() bool { return n.kind == leafTable || n.kind == interiorTable }

func hdrOffset(pgno uint32) int {
	if pgno == 1 {
		return 100
	}
	return 0
}

func (n *node) headerSize() int {
	if n.leaf() {
		return 8
	}
	return 12
}

// cellSize is the bytes a cell takes on a page of n's kind.
func (n *node) cellSize(c cell) int {
	var s int
	switch n.kind {
	case interiorTable:
		s = 4 + varintLen(uint64(c.key))
	case interiorIndex:
		s = 4 + len(c.body)
	default:
		s = len(c.body)
	}
	if s < 4 {
		s = 4 // SQLite never gives a cell less than 4 bytes
	}
	return s
}

func (f *dbFile) fits(n *node) bool {
	used := hdrOffset(n.pgno) + n.headerSize()
	for _, c := range n.cells {
		used += 2 + n.cellSize(c)
	}
	return used <= f.usable
}

func (f *dbFile) readNode(pgno uint32) *node {
	p := f.mustPage(pgno)
	h := hdrOffset(pgno)
	n := &node{pgno: pgno, kind: p[h]}
	count := int(binary.BigEndian.Uint16(p[h+3:]))
	ptrs := h + n.headerSize()
	switch n.kind {
	case interiorTable, interiorIndex:
		n.right = binary.BigEndian.Uint32(p[h+8:])
	case leafTable, leafIndex:
	default:
		panic(errorf("damaged database: page %d has unknown type %d", pgno, n.kind))
	}
	n.cells = make([]cell, count)
	for i := 0; i < count; i++ {
		off := int(binary.BigEndian.Uint16(p[ptrs+2*i:]))
		if off < ptrs || off >= len(p) {
			panic(errorf("damaged database: bad cell on page %d", pgno))
		}
		var c cell
		switch n.kind {
		case interiorTable:
			c.child = binary.BigEndian.Uint32(p[off:])
			k, _ := varint(p, off+4)
			c.key = int64(k)
		case leafTable:
			size, k1 := varint(p, off)
			rowid, k2 := varint(p, off+k1)
			c.key = int64(rowid)
			end := off + k1 + k2 + int(f.localSize(int64(size), true))
			if f.localSize(int64(size), true) < int64(size) {
				end += 4
			}
			c.body = p[off:end:end] // points into the page; see detach
		case interiorIndex:
			c.child = binary.BigEndian.Uint32(p[off:])
			c.body = f.indexBody(p, off+4)
		case leafIndex:
			c.body = f.indexBody(p, off)
		}
		n.cells[i] = c
	}
	return n
}

func (f *dbFile) indexBody(p []byte, off int) []byte {
	size, k := varint(p, off)
	local := f.localSize(int64(size), false)
	end := off + k + int(local)
	if local < int64(size) {
		end += 4
	}
	if end > len(p) {
		panic(errorf("damaged database: cell runs past its page"))
	}
	return p[off:end:end]
}

// detach gives cells their own copies of their bytes. A node read from a
// page points into that page; before cells move to another page (a split
// or merge, which writes pages in turn), they must stop pointing into
// pages that are about to be rewritten.
func detach(cells []cell) {
	for i := range cells {
		if cells[i].body != nil {
			cells[i].body = append([]byte(nil), cells[i].body...)
		}
	}
}

// writeNode lays n out on its page.
func (f *dbFile) writeNode(n *node) {
	if !f.fits(n) {
		panic(errorf("internal error: node doesn't fit page %d", n.pgno))
	}
	page := f.writable(n.pgno)
	// Laid out in a scratch page first: n's cells may point into the
	// page being written.
	if len(f.scratch) != f.pageSize {
		f.scratch = make([]byte, f.pageSize)
	}
	p := f.scratch
	h := hdrOffset(n.pgno)
	defer func() { copy(page[h:f.usable], p[h:f.usable]) }()
	for i := h; i < f.usable; i++ {
		p[i] = 0
	}
	p[h] = n.kind
	binary.BigEndian.PutUint16(p[h+3:], uint16(len(n.cells)))
	if !n.leaf() {
		binary.BigEndian.PutUint32(p[h+8:], n.right)
	}
	ptrs := h + n.headerSize()
	top := f.usable
	for i, c := range n.cells {
		size := n.cellSize(c)
		top -= size
		binary.BigEndian.PutUint16(p[ptrs+2*i:], uint16(top))
		switch n.kind {
		case interiorTable:
			binary.BigEndian.PutUint32(p[top:], c.child)
			putVarint(p[top+4:], uint64(c.key))
		case interiorIndex:
			binary.BigEndian.PutUint32(p[top:], c.child)
			copy(p[top+4:], c.body)
		default:
			copy(p[top:], c.body)
		}
	}
	start := top
	if start == 65536 {
		start = 0
	}
	binary.BigEndian.PutUint16(p[h+5:], uint16(start))
}

// ---- pages: allocate and free ----

// allocPage takes a page from the free list, or adds one to the file.
func (f *dbFile) allocPage() uint32 {
	p1 := f.writable(1)
	free := binary.BigEndian.Uint32(p1[36:])
	trunk := binary.BigEndian.Uint32(p1[32:])
	if free > 0 && trunk != 0 {
		t := f.writable(trunk)
		n := binary.BigEndian.Uint32(t[4:])
		binary.BigEndian.PutUint32(p1[36:], free-1)
		if n > 0 {
			leaf := binary.BigEndian.Uint32(t[8+4*(n-1):])
			binary.BigEndian.PutUint32(t[4:], n-1)
			binary.BigEndian.PutUint32(t[8+4*(n-1):], 0)
			if leaf == 0 || leaf > f.pageCount {
				panic(errorf("damaged database: bad page %d on the free list", leaf))
			}
			clear(f.writable(leaf))
			return leaf
		}
		binary.BigEndian.PutUint32(p1[32:], binary.BigEndian.Uint32(t[0:]))
		clear(t)
		return trunk
	}
	f.pageCount++
	if f.pageCount == f.pendingPage() {
		f.pageCount++
	}
	clear(f.writable(f.pageCount))
	return f.pageCount
}

// freePage puts a page on the free list.
func (f *dbFile) freePage(pg uint32) {
	if pg <= 1 || pg > f.pageCount {
		panic(errorf("internal error: freeing page %d", pg))
	}
	p1 := f.writable(1)
	free := binary.BigEndian.Uint32(p1[36:])
	trunk := binary.BigEndian.Uint32(p1[32:])
	binary.BigEndian.PutUint32(p1[36:], free+1)
	maxLeaves := uint32(f.usable/4 - 8)
	if trunk != 0 {
		t := f.writable(trunk)
		n := binary.BigEndian.Uint32(t[4:])
		if n < maxLeaves {
			binary.BigEndian.PutUint32(t[8+4*n:], pg)
			binary.BigEndian.PutUint32(t[4:], n+1)
			return
		}
	}
	t := f.writable(pg)
	clear(t)
	binary.BigEndian.PutUint32(t[0:], trunk)
	binary.BigEndian.PutUint32(p1[32:], pg)
}

// ---- payloads and overflow pages ----

// storePayload splits a payload into the part kept in the cell and an
// overflow chain for the rest, and returns the cell's payload bytes
// (local part plus the first overflow page number, if any).
func (f *dbFile) storePayload(payload []byte, table bool) []byte {
	size := int64(len(payload))
	local := f.localSize(size, table)
	out := append([]byte{}, payload[:local]...)
	if local == size {
		return out
	}
	rest := payload[local:]
	chunk := f.usable - 4
	first := f.allocPage()
	pg := first
	for {
		n := len(rest)
		if n > chunk {
			n = chunk
		}
		var next uint32
		if len(rest) > n {
			next = f.allocPage()
		}
		p := f.writable(pg)
		binary.BigEndian.PutUint32(p[0:], next)
		copy(p[4:], rest[:n])
		rest = rest[n:]
		if next == 0 {
			break
		}
		pg = next
	}
	var ptr [4]byte
	binary.BigEndian.PutUint32(ptr[:], first)
	return append(out, ptr[:]...)
}

// freeOverflow frees the overflow chain of a payload of size bytes whose
// local part ends at body[end:].
func (f *dbFile) freeOverflow(body []byte, start int, size int64, table bool) {
	local := f.localSize(size, table)
	if local == size {
		return
	}
	pg := binary.BigEndian.Uint32(body[start+int(local):])
	for count := 0; pg != 0; count++ {
		if count > int(f.pageCount) {
			panic(errorf("damaged database: overflow chain loops"))
		}
		next := binary.BigEndian.Uint32(f.mustPage(pg))
		f.freePage(pg)
		pg = next
	}
}

func (f *dbFile) tableCell(rowid int64, payload []byte) cell {
	body := appendVarint(nil, uint64(len(payload)))
	body = appendVarint(body, uint64(rowid))
	body = append(body, f.storePayload(payload, true)...)
	return cell{key: rowid, body: body}
}

func (f *dbFile) indexCell(payload []byte) cell {
	body := appendVarint(nil, uint64(len(payload)))
	return cell{body: append(body, f.storePayload(payload, false)...)}
}

// freeCell frees a leaf cell's overflow pages.
func (f *dbFile) freeCell(n *node, c cell) {
	if n.kind == interiorTable {
		return
	}
	size, k := varint(c.body, 0)
	if n.table() {
		_, k2 := varint(c.body, k)
		k += k2
	}
	f.freeOverflow(c.body, k, int64(size), n.table())
}

// cellPayload is a cell's whole payload, following overflow pages.
func (f *dbFile) cellPayload(n *node, c cell) []byte {
	size, k := varint(c.body, 0)
	if n.table() {
		_, k2 := varint(c.body, k)
		k += k2
	}
	p, err := f.payload(c.body, k, int64(size), n.table())
	if err != nil {
		panic(err)
	}
	return p
}

// ---- searching ----

type pathStep struct {
	n    *node // read when needed (a split or merge reaches it)
	pgno uint32
	idx  int // which child was taken: 0..len(cells), len(cells) is right
}

// node gives the step's page as a node, reading it the first time.
func (f *dbFile) stepNode(s *pathStep) *node {
	if s.n == nil {
		s.n = f.readNode(s.pgno)
	}
	return s.n
}

// Searching works on the page bytes: only a page about to change is
// decoded into a node.

type rawPage struct {
	p     []byte
	h     int // header offset (100 on page 1)
	kind  byte
	count int
	ptrs  int
}

func (f *dbFile) raw(pgno uint32) rawPage {
	p := f.mustPage(pgno)
	h := hdrOffset(pgno)
	r := rawPage{p: p, h: h, kind: p[h], count: int(binary.BigEndian.Uint16(p[h+3:]))}
	r.ptrs = h + 8
	if r.kind == interiorTable || r.kind == interiorIndex {
		r.ptrs = h + 12
	}
	if r.ptrs+2*r.count > len(p) {
		panic(errorf("damaged database: bad page %d", pgno))
	}
	return r
}

func (r rawPage) cellOff(i int) int { return int(binary.BigEndian.Uint16(r.p[r.ptrs+2*i:])) }

func (r rawPage) right() uint32 { return binary.BigEndian.Uint32(r.p[r.h+8:]) }

func (r rawPage) child(i int) uint32 {
	if i == r.count {
		return r.right()
	}
	return binary.BigEndian.Uint32(r.p[r.cellOff(i):])
}

// tableKey is the rowid (leaf) or key (interior) of cell i.
func (r rawPage) tableKey(i int) int64 {
	off := r.cellOff(i)
	if r.kind == interiorTable {
		k, _ := varint(r.p, off+4)
		return int64(k)
	}
	_, n := varint(r.p, off)
	k, _ := varint(r.p, off+n)
	return int64(k)
}

// searchTable finds the first cell whose key is at least k.
func (r rawPage) searchTable(k int64) int {
	lo, hi := 0, r.count
	for lo < hi {
		m := (lo + hi) / 2
		if r.tableKey(m) < k {
			lo = m + 1
		} else {
			hi = m
		}
	}
	return lo
}

// indexKey decodes the record of index cell i.
func (f *dbFile) rawIndexKey(r rawPage, i int) []Value {
	off := r.cellOff(i)
	if r.kind == interiorIndex {
		off += 4
	}
	size, n := varint(r.p, off)
	payload, err := f.payload(r.p, off+n, int64(size), false)
	if err != nil {
		panic(err)
	}
	vals, err := f.decodeRecord(payload)
	if err != nil {
		panic(err)
	}
	return vals
}

func childAt(n *node, i int) uint32 {
	if i == len(n.cells) {
		return n.right
	}
	return n.cells[i].child
}

func setChildAt(n *node, i int, pg uint32) {
	if i == len(n.cells) {
		n.right = pg
	} else {
		n.cells[i].child = pg
	}
}

// findTable descends a table B-tree to the leaf where rowid is or would
// be, and returns the path above it, the leaf, and the cell position.
func (f *dbFile) findTable(root uint32, rowid int64) ([]pathStep, *node, int, bool) {
	var path []pathStep
	pg := root
	for depth := 0; ; depth++ {
		r := f.raw(pg)
		if depth > 40 || (r.kind != interiorTable && r.kind != leafTable) {
			panic(errorf("damaged database: bad table B-tree at page %d", pg))
		}
		if r.kind == leafTable {
			n := f.readNode(pg)
			i := searchKeys(n, rowid)
			return path, n, i, i < len(n.cells) && n.cells[i].key == rowid
		}
		i := r.searchTable(rowid)
		path = append(path, pathStep{pgno: pg, idx: i})
		pg = r.child(i)
	}
}

// searchKeys finds the first cell whose key is at least k.
func searchKeys(n *node, k int64) int {
	lo, hi := 0, len(n.cells)
	for lo < hi {
		m := (lo + hi) / 2
		if n.cells[m].key < k {
			lo = m + 1
		} else {
			hi = m
		}
	}
	return lo
}

// seekRow returns the payload of the row with this rowid.
func (f *dbFile) seekRow(root uint32, rowid int64) ([]byte, bool) {
	pg := root
	for depth := 0; ; depth++ {
		r := f.raw(pg)
		switch {
		case depth > 40:
			panic(errorf("damaged database: B-tree too deep"))
		case r.kind == interiorTable:
			pg = r.child(r.searchTable(rowid))
		case r.kind == leafTable:
			i := r.searchTable(rowid)
			if i >= r.count || r.tableKey(i) != rowid {
				return nil, false
			}
			off := r.cellOff(i)
			size, n1 := varint(r.p, off)
			_, n2 := varint(r.p, off+n1)
			p, err := f.payload(r.p, off+n1+n2, int64(size), true)
			if err != nil {
				panic(err)
			}
			return p, true
		default:
			panic(errorf("damaged database: bad table B-tree at page %d", pg))
		}
	}
}

// maxRowid is the largest rowid in a table, or 0 if it's empty.
func (f *dbFile) maxRowid(root uint32) int64 {
	pg := root
	for depth := 0; ; depth++ {
		r := f.raw(pg)
		if depth > 40 {
			panic(errorf("damaged database: B-tree too deep"))
		}
		if r.kind == leafTable {
			if r.count == 0 {
				return 0
			}
			return r.tableKey(r.count - 1)
		}
		pg = r.right()
	}
}

// ---- table B-trees: insert and delete ----

// tablePut writes a row, replacing the row with the same rowid if any.
func (f *dbFile) tablePut(root uint32, rowid int64, payload []byte) {
	path, leaf, i, found := f.findTable(root, rowid)
	c := f.tableCell(rowid, payload)
	if found {
		f.freeCell(leaf, leaf.cells[i])
		leaf.cells[i] = c
	} else {
		leaf.cells = append(leaf.cells, cell{})
		copy(leaf.cells[i+1:], leaf.cells[i:])
		leaf.cells[i] = c
	}
	f.balance(path, leaf, !found && i == len(leaf.cells)-1)
}

// tableDelete removes the row with this rowid; it reports whether there
// was one.
func (f *dbFile) tableDelete(root uint32, rowid int64) bool {
	path, leaf, i, found := f.findTable(root, rowid)
	if !found {
		return false
	}
	f.freeCell(leaf, leaf.cells[i])
	leaf.cells = append(leaf.cells[:i], leaf.cells[i+1:]...)
	f.balance(path, leaf, false)
	return true
}

// ---- index B-trees ----

// indexKeyCompare compares two index records: the indexed columns with
// their collations and sort orders, then the rowid. With prefix, only
// a's length of values is compared.
func indexKeyCompare(a, b []Value, ix *Index) int {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	for i := 0; i < n; i++ {
		coll, desc := "BINARY", false
		if ix != nil && i < len(ix.Cols) {
			coll, desc = ix.Cols[i].Coll, ix.Cols[i].Desc
		}
		d := compareColl(a[i], b[i], coll)
		if d != 0 {
			if desc {
				return -d
			}
			return d
		}
	}
	return 0
}

func (f *dbFile) cellKey(n *node, c cell) []Value {
	vals, err := f.decodeRecord(f.cellPayload(n, c))
	if err != nil {
		panic(err)
	}
	return vals
}

// findIndex descends an index B-tree looking for exactly key. It returns
// the path, the node where the search ended, the position, and whether
// key is at that position.
func (f *dbFile) findIndex(ix *Index, key []Value) ([]pathStep, *node, int, bool) {
	var path []pathStep
	pg := ix.Root
	for depth := 0; ; depth++ {
		r := f.raw(pg)
		if depth > 40 || (r.kind != interiorIndex && r.kind != leafIndex) {
			panic(errorf("damaged database: bad index B-tree at page %d", pg))
		}
		lo, hi := 0, r.count
		for lo < hi {
			m := (lo + hi) / 2
			if indexKeyCompare(f.rawIndexKey(r, m), key, ix) < 0 {
				lo = m + 1
			} else {
				hi = m
			}
		}
		if lo < r.count && indexKeyCompare(f.rawIndexKey(r, lo), key, ix) == 0 {
			return path, f.readNode(pg), lo, true
		}
		if r.kind == leafIndex {
			return path, f.readNode(pg), lo, false
		}
		path = append(path, pathStep{pgno: pg, idx: lo})
		pg = r.child(lo)
	}
}

// indexInsert adds key (the indexed values, then the rowid).
func (f *dbFile) indexInsert(ix *Index, key []Value) {
	path, n, i, found := f.findIndex(ix, key)
	if found {
		panic(errorf("damaged database: index %s already has this entry", ix.Name))
	}
	c := f.indexCell(f.encodeRecord(key))
	n.cells = append(n.cells, cell{})
	copy(n.cells[i+1:], n.cells[i:])
	n.cells[i] = c
	f.balance(path, n, i == len(n.cells)-1)
}

// indexDelete removes key. An entry on an interior page is replaced by
// the entry just before it, taken from a leaf.
func (f *dbFile) indexDelete(ix *Index, key []Value) {
	path, n, i, found := f.findIndex(ix, key)
	if !found {
		panic(errorf("damaged database: index %s is missing an entry (run REINDEX in sqlite3)", ix.Name))
	}
	if n.leaf() {
		f.freeCell(n, n.cells[i])
		n.cells = append(n.cells[:i], n.cells[i+1:]...)
		f.balance(path, n, false)
		return
	}
	// The predecessor: the last entry of the rightmost leaf of child i.
	path = append(path, pathStep{n: n, pgno: n.pgno, idx: i})
	m := f.readNode(n.cells[i].child)
	for !m.leaf() {
		path = append(path, pathStep{n: m, pgno: m.pgno, idx: len(m.cells)})
		m = f.readNode(m.right)
	}
	if len(m.cells) == 0 {
		panic(errorf("damaged database: empty index page %d", m.pgno))
	}
	pred := m.cells[len(m.cells)-1]
	pred = cell{body: append([]byte(nil), pred.body...)}
	m.cells = m.cells[:len(m.cells)-1]
	f.balance(path, m, false)
	// Find the entry again (balancing may have moved it) and put the
	// predecessor in its place.
	path, n, i, found = f.findIndex(ix, key)
	if !found {
		panic(errorf("internal error: index entry lost while deleting"))
	}
	f.freeCell(n, n.cells[i])
	if n.leaf() {
		n.cells[i] = cell{body: pred.body}
	} else {
		n.cells[i] = cell{child: n.cells[i].child, body: pred.body}
	}
	f.balance(path, n, false)
}

// indexEqual returns the entries whose first len(prefix) values equal
// prefix (with the index's collations), in index order.
func (f *dbFile) indexEqual(ix *Index, prefix []Value) [][]Value {
	var out [][]Value
	var visit func(pg uint32, depth int)
	visit = func(pg uint32, depth int) {
		if depth > 40 {
			panic(errorf("damaged database: B-tree too deep"))
		}
		r := f.raw(pg)
		leaf := r.kind == leafIndex
		// Skip, by binary search, the cells before the prefix.
		lo, hi := 0, r.count
		for lo < hi {
			m := (lo + hi) / 2
			if indexKeyCompare(prefix, f.rawIndexKey(r, m), ix) > 0 {
				lo = m + 1
			} else {
				hi = m
			}
		}
		for i := lo; i < r.count; i++ {
			k := f.rawIndexKey(r, i)
			d := indexKeyCompare(prefix, k, ix)
			if !leaf {
				visit(r.child(i), depth+1)
			}
			if d < 0 {
				return
			}
			out = append(out, k)
		}
		if !leaf {
			visit(r.right(), depth+1)
		}
	}
	visit(ix.Root, 0)
	return out
}

// ---- creating and dropping B-trees ----

// newTree makes an empty table or index B-tree and returns its root.
func (f *dbFile) newTree(table bool) uint32 {
	pg := f.allocPage()
	kind := byte(leafIndex)
	if table {
		kind = leafTable
	}
	f.writeNode(&node{pgno: pg, kind: kind})
	return pg
}

// clearTree frees every page of a B-tree but its root, which is left
// empty (or frees the root too, with dropRoot).
func (f *dbFile) clearTree(root uint32, dropRoot bool) {
	var free func(pg uint32, depth int)
	free = func(pg uint32, depth int) {
		if depth > 40 {
			panic(errorf("damaged database: B-tree too deep"))
		}
		n := f.readNode(pg)
		for _, c := range n.cells {
			if n.leaf() || n.kind == interiorIndex {
				f.freeCell(n, c)
			}
			if !n.leaf() {
				free(c.child, depth+1)
			}
		}
		if !n.leaf() {
			free(n.right, depth+1)
		}
		if pg != root {
			f.freePage(pg)
		}
	}
	n := f.readNode(root)
	free(root, 0)
	if dropRoot {
		f.freePage(root)
		return
	}
	kind := byte(leafIndex)
	if n.table() {
		kind = leafTable
	}
	f.writeNode(&node{pgno: root, kind: kind})
}

// ---- keeping the tree balanced ----

// balance writes n back after a change and repairs the tree above it: a
// node too full for its page splits, a non-root node left empty merges
// with a sibling, and a root left with a single child absorbs it. path
// is the nodes above n, root first. appending says the change was at
// n's end, so a split leaves the left page full (rows added in order).
func (f *dbFile) balance(path []pathStep, n *node, appending bool) {
	for {
		root := len(path) == 0
		switch {
		case f.fits(n) && (root || len(n.cells) > 0):
			if root && !n.leaf() && len(n.cells) == 0 {
				f.collapseRoot(n)
				return
			}
			f.writeNode(n)
			return
		case !f.fits(n) && root:
			f.splitRoot(n, appending)
			return
		case !f.fits(n):
			step := &path[len(path)-1]
			parent := f.stepNode(step)
			f.splitChild(parent, step.idx, n, appending)
			n, path = parent, path[:len(path)-1]
		default: // an empty non-root node
			step := &path[len(path)-1]
			parent := f.stepNode(step)
			f.mergeChild(parent, step.idx, n)
			n, path = parent, path[:len(path)-1]
			appending = false
		}
	}
}

// collapseRoot moves a root's only child into the root page, making the
// tree one level shorter. Page 1 has less room than other pages, so a
// full child stays where it is (a root on page 1 may have no cells).
func (f *dbFile) collapseRoot(root *node) {
	child := f.readNode(root.right)
	detach(child.cells)
	moved := &node{pgno: root.pgno, kind: child.kind, cells: child.cells, right: child.right}
	if !f.fits(moved) {
		f.writeNode(root)
		return
	}
	f.writeNode(moved)
	f.freePage(child.pgno)
}

// piece is one page's worth of cells after a split.
type piece struct {
	cells []cell
	right uint32
	pgno  uint32
}

// split divides n's cells into pieces that each fit a page, and returns
// them with the dividers that go between them in the parent.
func (f *dbFile) split(n *node, appending bool) ([]piece, []cell) {
	detach(n.cells)
	capacity := f.usable - n.headerSize() // a non-root page, or a new page
	sizes := make([]int, len(n.cells))
	total := 0
	for i, c := range n.cells {
		sizes[i] = 2 + n.cellSize(c)
		total += sizes[i]
	}
	consume := n.kind != leafTable // the divider leaves the child
	// How many pages are needed when each is filled as far as it goes.
	pages := 1
	used := 0
	for i := 0; i < len(sizes); i++ {
		if used+sizes[i] > capacity {
			pages++
			used = 0
			if consume {
				continue // this cell becomes the divider
			}
		}
		used += sizes[i]
	}
	if pages < 2 {
		pages = 2
	}
	target := total / pages
	if appending {
		target = capacity
	}
	var pieces []piece
	var dividers []cell
	cur := piece{}
	used = 0
	for i := 0; i < len(n.cells); i++ {
		c := n.cells[i]
		full := used+sizes[i] > capacity || (used >= target && len(pieces) < pages-1)
		if full && len(cur.cells) > 0 {
			switch {
			case !consume:
				// Table leaves: the divider is a copy of the left page's
				// last rowid.
				dividers = append(dividers, cell{key: cur.cells[len(cur.cells)-1].key})
				pieces = append(pieces, cur)
				cur, used = piece{}, 0
			case i < len(n.cells)-1:
				// This cell moves up to the parent as the divider.
				dividers = append(dividers, c)
				cur.right = c.child
				pieces = append(pieces, cur)
				cur, used = piece{}, 0
				continue
			case len(cur.cells) >= 2:
				// The last cell can't be the divider (the last page would
				// be empty): the one before it moves up instead.
				d := cur.cells[len(cur.cells)-1]
				cur.cells = cur.cells[:len(cur.cells)-1]
				dividers = append(dividers, d)
				cur.right = d.child
				pieces = append(pieces, cur)
				cur, used = piece{}, 0
			}
		}
		cur.cells = append(cur.cells, c)
		used += sizes[i]
	}
	cur.right = n.right
	pieces = append(pieces, cur)
	return pieces, dividers
}

func (f *dbFile) writePieces(kind byte, pieces []piece) {
	for _, p := range pieces {
		f.writeNode(&node{pgno: p.pgno, kind: kind, cells: p.cells, right: p.right})
	}
}

// parentCells turns pieces and dividers into the cells a parent holds:
// each divider with the piece to its left as its child.
func parentCells(n *node, pieces []piece, dividers []cell) []cell {
	out := make([]cell, len(dividers))
	for i, d := range dividers {
		if n.table() {
			out[i] = cell{child: pieces[i].pgno, key: d.key}
		} else {
			out[i] = cell{child: pieces[i].pgno, body: d.body}
		}
	}
	return out
}

func interiorKind(n *node) byte {
	if n.table() {
		return interiorTable
	}
	return interiorIndex
}

// splitRoot keeps the root's page number: its cells move to new pages
// and the root becomes an interior page over them.
func (f *dbFile) splitRoot(n *node, appending bool) {
	pieces, dividers := f.split(n, appending)
	for i := range pieces {
		pieces[i].pgno = f.allocPage()
	}
	f.writePieces(n.kind, pieces)
	root := &node{pgno: n.pgno, kind: interiorKind(n), cells: parentCells(n, pieces, dividers), right: pieces[len(pieces)-1].pgno}
	if !f.fits(root) {
		// Possible only with huge index keys on a tiny page.
		panic(errorf("internal error: root page %d can't hold its dividers", n.pgno))
	}
	f.writeNode(root)
}

// splitChild splits parent's child at idx (node n) and puts the new
// dividers in parent (which the caller balances next).
func (f *dbFile) splitChild(parent *node, idx int, n *node, appending bool) {
	pieces, dividers := f.split(n, appending)
	pieces[0].pgno = n.pgno
	for i := 1; i < len(pieces); i++ {
		pieces[i].pgno = f.allocPage()
	}
	f.writePieces(n.kind, pieces)
	setChildAt(parent, idx, pieces[len(pieces)-1].pgno)
	add := parentCells(n, pieces, dividers)
	cells := make([]cell, 0, len(parent.cells)+len(add))
	cells = append(cells, parent.cells[:idx]...)
	cells = append(cells, add...)
	cells = append(cells, parent.cells[idx:]...)
	parent.cells = cells
}

// mergeChild joins parent's empty child at idx (node n) with a sibling,
// through the divider between them, and splits the result again if it
// doesn't fit one page.
func (f *dbFile) mergeChild(parent *node, idx int, n *node) {
	if len(parent.cells) == 0 {
		panic(errorf("internal error: merging under a parent with one child"))
	}
	li := idx // left of the pair, as a child position in parent
	var left, right *node
	if idx > 0 {
		li = idx - 1
		left, right = f.readNode(childAt(parent, li)), n
	} else {
		left, right = n, f.readNode(childAt(parent, 1))
	}
	detach(left.cells)
	detach(right.cells)
	detach(parent.cells)
	div := parent.cells[li]
	m := &node{pgno: left.pgno, kind: n.kind, right: right.right}
	m.cells = append(m.cells, left.cells...)
	switch n.kind {
	case interiorTable:
		m.cells = append(m.cells, cell{child: left.right, key: div.key})
	case leafIndex:
		m.cells = append(m.cells, cell{body: div.body})
	case interiorIndex:
		m.cells = append(m.cells, cell{child: left.right, body: div.body})
	}
	m.cells = append(m.cells, right.cells...)
	// The divider leaves the parent; the pair becomes one child, m.
	parent.cells = append(parent.cells[:li], parent.cells[li+1:]...)
	setChildAt(parent, li, m.pgno)
	f.freePage(right.pgno)
	if f.fits(m) {
		f.writeNode(m)
		return
	}
	f.splitChild(parent, li, m, false)
}

// ---- records ----

// encodeRecord turns values into a record: a header of serial types,
// then the values.
func (f *dbFile) encodeRecord(vals []Value) []byte {
	types := make([]uint64, len(vals))
	hlen := 0
	for i, v := range vals {
		types[i] = f.serialType(v)
		hlen += varintLen(types[i])
	}
	// The header's length counts itself.
	hsize := hlen + 1
	for hlen+varintLen(uint64(hsize)) != hsize {
		hsize = hlen + varintLen(uint64(hsize))
	}
	out := appendVarint(nil, uint64(hsize))
	for _, t := range types {
		out = appendVarint(out, t)
	}
	for i, v := range vals {
		switch x := v.(type) {
		case int64:
			n := serialSize(types[i])
			for k := n - 1; k >= 0; k-- {
				out = append(out, byte(x>>(8*uint(k))))
			}
		case float64:
			var b [8]byte
			binary.BigEndian.PutUint64(b[:], math.Float64bits(x))
			out = append(out, b[:]...)
		case string:
			out = append(out, x...)
		case []byte:
			out = append(out, x...)
		}
	}
	return out
}

func (f *dbFile) serialType(v Value) uint64 {
	switch x := v.(type) {
	case nil:
		return 0
	case int64:
		switch {
		case x == 0 && f.format >= 4:
			return 8
		case x == 1 && f.format >= 4:
			return 9
		case x >= -128 && x <= 127:
			return 1
		case x >= -32768 && x <= 32767:
			return 2
		case x >= -8388608 && x <= 8388607:
			return 3
		case x >= -2147483648 && x <= 2147483647:
			return 4
		case x >= -140737488355328 && x <= 140737488355327:
			return 5
		}
		return 6
	case float64:
		return 7
	case string:
		return uint64(13 + 2*len(x))
	case []byte:
		return uint64(12 + 2*len(x))
	}
	panic(errorf("internal error: can't store a %T", v))
}

func varintLen(v uint64) int {
	if v > 0x00ffffffffffffff {
		return 9
	}
	n := 1
	for v >= 0x80 {
		v >>= 7
		n++
	}
	return n
}

func appendVarint(b []byte, v uint64) []byte {
	var buf [9]byte
	n := putVarint(buf[:], v)
	return append(b, buf[:n]...)
}

// putVarint writes v as a SQLite varint and returns its length.
func putVarint(p []byte, v uint64) int {
	if v > 0x00ffffffffffffff {
		p[8] = byte(v)
		v >>= 8
		for i := 7; i >= 0; i-- {
			p[i] = byte(v&0x7f) | 0x80
			v >>= 7
		}
		return 9
	}
	var tmp [9]byte
	n := 0
	for {
		tmp[n] = byte(v & 0x7f)
		n++
		v >>= 7
		if v == 0 {
			break
		}
	}
	for i := 0; i < n; i++ {
		b := tmp[n-1-i]
		if i < n-1 {
			b |= 0x80
		}
		p[i] = b
	}
	return n
}
