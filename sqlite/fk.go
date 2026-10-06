package sqlite

import (
	"strings"
)

// Foreign keys, enforced when PRAGMA foreign_keys = ON (off by default,
// as in SQLite). The method is SQLite's own: a count of broken references
// goes up when a change leaves a child row pointing at no parent, and
// down when a change mends one. Immediate keys must be back to zero when
// the statement ends, deferred keys (DEFERRABLE INITIALLY DEFERRED, or
// PRAGMA defer_foreign_keys) at COMMIT. So one statement may add a child
// before its parent.
//
// Deleting or changing a parent key also does the key's ON DELETE / ON
// UPDATE action to its children: CASCADE, SET NULL, SET DEFAULT,
// RESTRICT (fails at once), or NO ACTION (counted).

// fkRef is a foreign key of child pointing at some table.
type fkRef struct {
	child *Table
	fk    *ForeignKey
}

// incoming lists the foreign keys that point at table t.
func (f *dbFile) incoming(t *Table) []fkRef {
	var out []fkRef
	for _, n := range f.allTableNames() {
		c := f.tables[n]
		for _, fk := range c.FKs {
			if strings.EqualFold(fk.Parent, t.Name) {
				out = append(out, fkRef{c, fk})
			}
		}
	}
	return out
}

func (f *dbFile) allTableNames() []string {
	var out []string
	for _, r := range f.schemaRows {
		if r.kind == "table" {
			if _, ok := f.tables[strings.ToLower(r.name)]; ok {
				out = append(out, strings.ToLower(r.name))
			}
		}
	}
	return out
}

func (f *dbFile) fkCount(fk *ForeignKey, d int) {
	if fk.Deferred || f.deferFKs {
		f.fkDeferred += d
	} else {
		f.fkStmt += d
	}
}

func mismatch(child, parent string) {
	fail("foreign key mismatch - \"%s\" referencing \"%s\"", child, parent)
}

// parentKey finds the parent table of fk and the columns its key uses,
// which must be its PRIMARY KEY or have a UNIQUE index.
func (f *dbFile) parentKey(child *Table, fk *ForeignKey) (*Table, []int) {
	p := f.tables[strings.ToLower(fk.Parent)]
	if p == nil {
		fail("no such table: main.%s", fk.Parent)
	}
	var cols []int
	if len(fk.ParentCols) == 0 {
		cols = p.PK
		if len(cols) == 0 || len(cols) != len(fk.Cols) {
			mismatch(child.Name, p.Name)
		}
	} else {
		for _, n := range fk.ParentCols {
			ci := colIndex(p.Columns, n)
			if ci < 0 {
				mismatch(child.Name, p.Name)
			}
			cols = append(cols, ci)
		}
	}
	if len(cols) == 1 && cols[0] == p.RowidCol {
		return p, cols
	}
	if f.uniqueIndexOn(p, cols) == nil {
		mismatch(child.Name, p.Name)
	}
	return p, cols
}

// uniqueIndexOn finds a UNIQUE index of t on exactly cols (in any order).
func (f *dbFile) uniqueIndexOn(t *Table, cols []int) *Index {
	for _, ix := range t.Indexes {
		if !ix.Unique || ix.Partial != nil || len(ix.Cols) != len(cols) {
			continue
		}
		all := true
		for _, ic := range ix.Cols {
			found := false
			for _, c := range cols {
				if ic.Col == c {
					found = true
				}
			}
			all = all && found
		}
		if all {
			return ix
		}
	}
	return nil
}

// childKey is the key a child row points at, with the parent columns'
// affinity; ok is false when part of it is NULL (it points at nothing).
func childKey(p *Table, pcols []int, fk *ForeignKey, vals []Value) ([]Value, bool) {
	key := make([]Value, len(fk.Cols))
	for i, ci := range fk.Cols {
		if vals[ci] == nil {
			return nil, false
		}
		key[i] = applyAffinity(vals[ci], p.Columns[pcols[i]].Affinity)
	}
	return key, true
}

// parentExists reports whether a parent row has key on pcols.
func (f *dbFile) parentExists(p *Table, pcols []int, key []Value) bool {
	if len(pcols) == 1 && pcols[0] == p.RowidCol {
		id, ok := exactRowid(key[0])
		if !ok {
			return false
		}
		_, found := f.seekRow(p.Root, id)
		return found
	}
	ix := f.uniqueIndexOn(p, pcols)
	prefix := make([]Value, len(ix.Cols))
	for i, ic := range ix.Cols {
		for k, c := range pcols {
			if c == ic.Col {
				prefix[i] = key[k]
			}
		}
	}
	return len(f.indexEqual(ix, prefix)) > 0
}

// children finds the rows of ref.child that point at the parent key.
func (f *dbFile) children(ref fkRef, p *Table, pcols []int, key []Value) []srcRow {
	c := ref.child
	r := &runner{db: f, core: f.rowPlan(c, "", false), cur: make([]*srcRow, 1)}
	matches := func(vals []Value) bool {
		for i, ci := range ref.fk.Cols {
			if vals[ci] == nil {
				return false
			}
			col := p.Columns[pcols[i]]
			if compareColl(applyAffinity(vals[ci], col.Affinity), key[i], col.Collate) != 0 {
				return false
			}
		}
		return true
	}
	var out []srcRow
	// An index whose first column is the key's first column narrows it.
	for _, ix := range c.Indexes {
		if ix.Partial == nil && len(ix.Cols) > 0 && ix.Cols[0].Col == ref.fk.Cols[0] && !ix.HasExpr {
			for _, e := range f.indexEqual(ix, []Value{key[0]}) {
				id := e[len(e)-1].(int64)
				if vals, ok := f.readRow(c, id, r); ok && matches(vals) {
					out = append(out, srcRow{id, vals})
				}
			}
			return out
		}
	}
	if len(ref.fk.Cols) == 1 && ref.fk.Cols[0] == c.RowidCol {
		if id, ok := exactRowid(key[0]); ok {
			if vals, found := f.readRow(c, id, r); found {
				out = append(out, srcRow{id, vals})
			}
		}
		return out
	}
	f.scanTable(c.Root, func(rowid int64, payload []byte) error {
		vals := f.fullRow(c, rowid, payload, r)
		if matches(vals) {
			out = append(out, srcRow{rowid, vals})
		}
		return nil
	})
	return out
}

// fkChild counts a child row coming (d = 1) or going (d = -1): only a row
// that points at no parent counts.
func (f *dbFile) fkChild(t *Table, vals []Value, d int, only func(*ForeignKey) bool) {
	for _, fk := range t.FKs {
		if only != nil && !only(fk) {
			continue
		}
		p, pcols := f.parentKey(t, fk)
		key, ok := childKey(p, pcols, fk, vals)
		if ok && !f.parentExists(p, pcols, key) {
			f.fkCount(fk, d)
		}
	}
}

// fkBeforeDelete runs before a row of t is deleted: its own references
// stop counting, and its children get their ON DELETE action.
func (f *dbFile) fkBeforeDelete(t *Table, rowid int64, vals []Value) {
	f.fkChild(t, vals, -1, nil)
	for _, ref := range f.incoming(t) {
		p, pcols := f.parentKey(ref.child, ref.fk)
		key, ok := keyOf(vals, pcols)
		if !ok {
			continue
		}
		kids := f.children(ref, p, pcols, key)
		var others []srcRow
		for _, k := range kids {
			if !(ref.child == t && k.rowid == rowid) {
				others = append(others, k)
			}
		}
		if len(kids) == 0 {
			continue
		}
		switch ref.fk.OnDelete {
		case "CASCADE":
			for _, k := range others {
				if f.fkBusy(ref.child, k.rowid) {
					continue
				}
				r := &runner{db: f, core: f.rowPlan(ref.child, "", false), cur: make([]*srcRow, 1)}
				if cur, ok := f.readRow(ref.child, k.rowid, r); ok {
					f.deleteRow(ref.child, k.rowid, cur)
				}
			}
		case "SET NULL", "SET DEFAULT":
			f.fkSetChildren(ref, others, ref.fk.OnDelete)
		case "RESTRICT":
			fail("FOREIGN KEY constraint failed")
		default: // NO ACTION
			f.fkCount(ref.fk, len(kids))
		}
	}
}

// fkAfterInsert runs after a row of t is added: its references count if
// they point at nothing, and children waiting for it stop counting.
func (f *dbFile) fkAfterInsert(t *Table, vals []Value) {
	f.fkChild(t, vals, 1, nil)
	f.fkParentArrived(t, vals, nil)
}

// fkParentArrived: a parent key appeared; children already pointing at
// it were counted as broken, and aren't any more.
func (f *dbFile) fkParentArrived(t *Table, vals []Value, only func(fkRef) bool) {
	if f.fkStmt == 0 && f.fkDeferred == 0 {
		return
	}
	for _, ref := range f.incoming(t) {
		if only != nil && !only(ref) {
			continue
		}
		p, pcols := f.parentKey(ref.child, ref.fk)
		key, ok := keyOf(vals, pcols)
		if !ok {
			continue
		}
		f.fkCount(ref.fk, -len(f.children(ref, p, pcols, key)))
	}
}

// fkUpdate runs around an UPDATE of a row of t: before is called with the
// old values before the write, after with the new ones after it.
func (f *dbFile) fkBeforeUpdate(t *Table, rowid int64, old, vals []Value) {
	changedFK := func(fk *ForeignKey) bool {
		for _, ci := range fk.Cols {
			if !sameKey([]Value{old[ci]}, []Value{vals[ci]}) {
				return true
			}
		}
		return false
	}
	f.fkChild(t, old, -1, changedFK)
	for _, ref := range f.incoming(t) {
		p, pcols := f.parentKey(ref.child, ref.fk)
		oldKey, ok := keyOf(old, pcols)
		newKey, _ := keyOf(vals, pcols)
		if !ok || sameKey(oldKey, newKey) {
			continue
		}
		kids := f.children(ref, p, pcols, oldKey)
		var others []srcRow
		for _, k := range kids {
			if !(ref.child == t && k.rowid == rowid) {
				others = append(others, k)
			}
		}
		if len(kids) == 0 {
			continue
		}
		switch ref.fk.OnUpdate {
		case "CASCADE":
			for _, k := range others {
				nv := append([]Value{}, k.vals...)
				for i, ci := range ref.fk.Cols {
					nv[ci] = newKey[i]
				}
				f.fkRewriteChild(ref.child, k.rowid, k.vals, nv)
			}
		case "SET NULL", "SET DEFAULT":
			f.fkSetChildren(ref, others, ref.fk.OnUpdate)
		case "RESTRICT":
			fail("FOREIGN KEY constraint failed")
		default:
			f.fkCount(ref.fk, len(kids))
		}
	}
}

func (f *dbFile) fkAfterUpdate(t *Table, old, vals []Value) {
	changedFK := func(fk *ForeignKey) bool {
		for _, ci := range fk.Cols {
			if !sameKey([]Value{old[ci]}, []Value{vals[ci]}) {
				return true
			}
		}
		return false
	}
	f.fkChild(t, vals, 1, changedFK)
	f.fkParentArrived(t, vals, func(ref fkRef) bool {
		_, pcols := f.parentKey(ref.child, ref.fk)
		a, ok1 := keyOf(old, pcols)
		b, ok2 := keyOf(vals, pcols)
		return !(ok1 && ok2 && sameKey(a, b))
	})
}

// fkSetChildren sets the children's key columns to NULL or their
// defaults.
func (f *dbFile) fkSetChildren(ref fkRef, kids []srcRow, action string) {
	r := &runner{db: f, core: f.rowPlan(ref.child, "", false), cur: make([]*srcRow, 1)}
	for _, k := range kids {
		nv := append([]Value{}, k.vals...)
		for _, ci := range ref.fk.Cols {
			nv[ci] = nil
			if action == "SET DEFAULT" && ref.child.Columns[ci].Default != nil {
				nv[ci] = prepareValue(ref.child, ci, r.eval(ref.child.Columns[ci].Default))
			}
		}
		f.fkRewriteChild(ref.child, k.rowid, k.vals, nv)
	}
}

// fkRewriteChild writes a child row changed by an action, checking that
// its new reference points somewhere.
func (f *dbFile) fkRewriteChild(t *Table, rowid int64, old, vals []Value) {
	if t.RowidCol >= 0 {
		vals[t.RowidCol] = rowid
	}
	f.writeInPlace(t, rowid, old, vals)
	for _, fk := range t.FKs {
		p, pcols := f.parentKey(t, fk)
		key, ok := childKey(p, pcols, fk, vals)
		if ok && !f.parentExists(p, pcols, key) {
			f.fkCount(fk, 1)
		}
	}
}

// fkBusy guards a cascade against coming back to a row it's deleting.
func (f *dbFile) fkBusy(t *Table, rowid int64) bool {
	key := t.Name + "\x00" + itoa64(rowid)
	if f.fkDeleting[key] {
		return true
	}
	return false
}

func keyOf(vals []Value, cols []int) ([]Value, bool) {
	key := make([]Value, len(cols))
	for i, c := range cols {
		if vals[c] == nil {
			return nil, false
		}
		key[i] = vals[c]
	}
	return key, true
}

// fkViolations lists the rows of t whose foreign keys point at no row:
// (table, rowid, parent, fkid), as PRAGMA foreign_key_check gives them.
func (f *dbFile) fkViolations(t *Table) [][]Value {
	var out [][]Value
	r := &runner{db: f, core: f.rowPlan(t, "", false), cur: make([]*srcRow, 1)}
	for i, fk := range t.FKs {
		id := int64(len(t.FKs) - 1 - i)
		p, pcols := f.parentKey(t, fk)
		f.scanTable(t.Root, func(rowid int64, payload []byte) error {
			vals := f.fullRow(t, rowid, payload, r)
			key, ok := childKey(p, pcols, fk, vals)
			if ok && !f.parentExists(p, pcols, key) {
				out = append(out, []Value{t.Name, rowid, fk.Parent, id})
			}
			return nil
		})
	}
	return out
}
