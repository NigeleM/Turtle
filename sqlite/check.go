package sqlite

import (
	"encoding/binary"
	"sort"
	"strings"
)

// Check reads the whole file and reports the first problem it finds, the
// way SQLite's PRAGMA integrity_check does: every page belongs to exactly
// one B-tree, overflow chain or the free list; every B-tree is in order
// with all its leaves at the same depth; every index holds exactly its
// table's rows.
func (db *DB) Check() (err error) {
	defer catch(&err)
	done, err := db.beginRead()
	if err != nil {
		return err
	}
	defer done()
	return db.checkLocked()
}

// checkLocked is Check, with the file already readable.
func (db *DB) checkLocked() (err error) {
	defer catch(&err)
	c := &checker{f: db.dbFile, used: make([]bool, db.pageCount+1)}
	c.mark(1, "the header page")
	if c.f.pendingPage() <= c.f.pageCount {
		c.used[c.f.pendingPage()] = true
	}
	c.tree(1, nil, "sqlite_schema")
	for _, r := range c.f.schemaRows {
		if r.root == 0 {
			continue // a view, a trigger, or a virtual table
		}
		switch r.kind {
		case "table":
			if t := c.f.tables[strings.ToLower(r.name)]; t != nil && t.WithoutRowid {
				c.tree(r.root, t.pkIdx, r.name)
				continue
			}
			c.tree(r.root, nil, r.name)
		case "index":
			var ix *Index
			var tab *Table
			if t := c.f.tables[strings.ToLower(r.tbl)]; t != nil {
				for _, x := range t.Indexes {
					if strings.EqualFold(x.Name, r.name) {
						ix, tab = x, t
					}
				}
			}
			if ix == nil {
				// An index Turtle can't read the definition of: check its
				// pages but not its order.
				c.tree(r.root, &Index{Name: r.name}, r.name)
				continue
			}
			c.tree(r.root, ix, r.name)
			if !tab.WithoutRowid {
				c.indexMatches(tab, ix)
			}
		}
	}
	c.freeList()
	for p := uint32(1); p <= c.f.pageCount; p++ {
		if !c.used[p] {
			fail("damaged database: page %d is never used", p)
		}
	}
	return nil
}

type checker struct {
	f    *dbFile
	used []bool
}

func (c *checker) mark(p uint32, what string) {
	if p < 1 || p > c.f.pageCount {
		fail("damaged database: %s points to page %d, outside the file", what, p)
	}
	if c.used[p] {
		fail("damaged database: page %d is used twice (again by %s)", p, what)
	}
	c.used[p] = true
}

// tree checks one B-tree and returns nothing; it fails on a problem.
func (c *checker) tree(root uint32, ix *Index, name string) {
	table := ix == nil
	leafDepth := -1
	var lastKey int64
	haveKey := false
	var lastIdx []Value
	var walkNode func(pg uint32, depth int, lo, hi *int64)
	walkNode = func(pg uint32, depth int, lo, hi *int64) {
		if pg != 1 || root != 1 {
			c.mark(pg, name)
		}
		if depth > 40 {
			fail("damaged database: %s is deeper than 40 levels", name)
		}
		c.cellLayout(pg, name)
		n := c.f.readNode(pg)
		if n.table() != table {
			fail("damaged database: page %d of %s has the wrong type", pg, name)
		}
		if pg != root && len(n.cells) == 0 && n.leaf() {
			fail("damaged database: page %d of %s is empty", pg, name)
		}
		if n.leaf() {
			if leafDepth == -1 {
				leafDepth = depth
			} else if leafDepth != depth {
				fail("damaged database: %s has leaves at different depths", name)
			}
		}
		for i, cl := range n.cells {
			if table {
				if lo != nil && cl.key <= *lo || hi != nil && cl.key > *hi {
					fail("damaged database: rowid %d out of order on page %d of %s", cl.key, pg, name)
				}
				if n.leaf() {
					if haveKey && cl.key <= lastKey {
						fail("damaged database: rowid %d out of order on page %d of %s", cl.key, pg, name)
					}
					lastKey, haveKey = cl.key, true
					c.overflow(n, cl, name)
				} else {
					var clo *int64
					if i > 0 {
						clo = &n.cells[i-1].key
					} else {
						clo = lo
					}
					k := cl.key
					walkNode(cl.child, depth+1, clo, &k)
				}
				continue
			}
			if !n.leaf() {
				walkNode(cl.child, depth+1, nil, nil)
			}
			key := c.f.cellKey(n, cl)
			if lastIdx != nil && ix.Cols != nil && indexKeyCompare(lastIdx, key, ix) >= 0 {
				fail("damaged database: index entries out of order on page %d of %s", pg, name)
			}
			lastIdx = key
			c.overflow(n, cl, name)
		}
		if !n.leaf() {
			var rlo *int64
			if len(n.cells) > 0 && table {
				rlo = &n.cells[len(n.cells)-1].key
			} else {
				rlo = lo
			}
			walkNode(n.right, depth+1, rlo, hi)
		}
	}
	walkNode(root, 0, nil, nil)
}

// cellLayout checks that a page's cells sit inside it without overlap.
func (c *checker) cellLayout(pg uint32, name string) {
	p := c.f.mustPage(pg)
	h := hdrOffset(pg)
	kind := p[h]
	hsize := 8
	if kind == interiorTable || kind == interiorIndex {
		hsize = 12
	}
	count := int(binary.BigEndian.Uint16(p[h+3:]))
	ptrEnd := h + hsize + 2*count
	if ptrEnd > c.f.usable {
		fail("damaged database: page %d of %s has too many cells", pg, name)
	}
	type span struct{ start, end int }
	var spans []span
	n := c.f.readNode(pg)
	for i, cl := range n.cells {
		off := int(binary.BigEndian.Uint16(p[h+hsize+2*i:]))
		size := n.cellSize(cl)
		if off < ptrEnd || off+size > c.f.usable {
			fail("damaged database: a cell on page %d of %s is outside its page", pg, name)
		}
		spans = append(spans, span{off, off + size})
	}
	sort.Slice(spans, func(i, j int) bool { return spans[i].start < spans[j].start })
	for i := 1; i < len(spans); i++ {
		if spans[i].start < spans[i-1].end {
			fail("damaged database: cells overlap on page %d of %s", pg, name)
		}
	}
}

// overflow marks a cell's overflow pages.
func (c *checker) overflow(n *node, cl cell, name string) {
	size, k := varint(cl.body, 0)
	if n.table() {
		_, k2 := varint(cl.body, k)
		k += k2
	}
	local := c.f.localSize(int64(size), n.table())
	if local == int64(size) {
		return
	}
	need := (int64(size) - local + int64(c.f.usable) - 5) / int64(c.f.usable-4)
	pg := binary.BigEndian.Uint32(cl.body[k+int(local):])
	for i := int64(0); i < need; i++ {
		c.mark(pg, name+" (overflow)")
		pg = binary.BigEndian.Uint32(c.f.mustPage(pg))
	}
	if pg != 0 {
		fail("damaged database: an overflow chain of %s is too long", name)
	}
}

func (c *checker) freeList() {
	h := c.f.mustPage(1)
	trunk := binary.BigEndian.Uint32(h[32:])
	want := binary.BigEndian.Uint32(h[36:])
	var got uint32
	for trunk != 0 {
		c.mark(trunk, "the free list")
		got++
		t := c.f.mustPage(trunk)
		n := binary.BigEndian.Uint32(t[4:])
		if n > uint32(c.f.usable/4-2) {
			fail("damaged database: free-list page %d says it holds %d pages", trunk, n)
		}
		for i := uint32(0); i < n; i++ {
			c.mark(binary.BigEndian.Uint32(t[8+4*i:]), "the free list")
			got++
		}
		trunk = binary.BigEndian.Uint32(t[0:])
	}
	if got != want {
		fail("damaged database: the free list has %d pages, but the header says %d", got, want)
	}
}

// indexMatches checks that an index holds exactly one entry per row of
// its table (that its partial WHERE accepts), with the right values.
func (c *checker) indexMatches(t *Table, ix *Index) {
	var want []string
	r := &runner{db: c.f, core: c.f.rowPlan(t, "", false), cur: make([]*srcRow, 1)}
	err := c.f.scanTable(t.Root, func(rowid int64, payload []byte) error {
		vals := c.f.fullRow(t, rowid, payload, r)
		if c.f.inIndex(t, ix, rowid, vals, nil) {
			want = append(want, rowKey(c.f.indexKey(t, ix, rowid, vals)))
		}
		return nil
	})
	if err != nil {
		panic(err)
	}
	var got []string
	var visit func(pg uint32)
	visit = func(pg uint32) {
		n := c.f.readNode(pg)
		for _, cl := range n.cells {
			if !n.leaf() {
				visit(cl.child)
			}
			got = append(got, rowKey(c.f.cellKey(n, cl)))
		}
		if !n.leaf() {
			visit(n.right)
		}
	}
	visit(ix.Root)
	sort.Strings(want)
	sort.Strings(got)
	if strings.Join(want, "\n") != strings.Join(got, "\n") {
		fail("damaged database: index %s doesn't match table %s (%d entries for %d rows)", ix.Name, t.Name, len(got), len(want))
	}
}
