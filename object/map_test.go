// Copyright 2017-2026 Nigele McCoy
// Licensed under the Apache License, Version 2.0;
// see LICENSE and NOTICE.

package object

import (
	"fmt"
	"testing"
)

// TestMapAcrossTheIndexSwitch: a map keeps its order, keeps 1 and "1"
// apart, and finds, changes and deletes keys the same whether it's small
// (scanned) or large (indexed), and while it grows and shrinks across the
// line between them.
func TestMapAcrossTheIndexSwitch(t *testing.T) {
	m := NewMap()
	var order []string
	for i := 0; i < 20; i++ {
		m.Put(Int(int64(i)), &String{Value: fmt.Sprint("n", i)})
		m.Put(&String{Value: fmt.Sprint(i)}, Int(int64(i*10)))
		order = append(order, fmt.Sprint(i), fmt.Sprintf("%q", fmt.Sprint(i)))
		check := func() {
			for j := 0; j <= i; j++ {
				if v, ok := m.Get(Int(int64(j))); !ok || v.Inspect() != fmt.Sprint("n", j) {
					t.Fatalf("size %d: key %d gave %v %v", m.Len(), j, v, ok)
				}
				if v, ok := m.Get(&String{Value: fmt.Sprint(j)}); !ok || v.(*Integer).Value != int64(j*10) {
					t.Fatalf("size %d: key %q gave %v %v", m.Len(), fmt.Sprint(j), v, ok)
				}
			}
		}
		check()
	}
	for i, e := range m.Entries() {
		if e.K != order[i] {
			t.Fatalf("entry %d is %s, want %s", i, e.K, order[i])
		}
	}
	m.Put(Int(3), &String{Value: "three"}) // changing keeps the place
	if m.Entries()[6].Val.Inspect() != "three" || m.Len() != 40 {
		t.Fatalf("update moved or added an entry")
	}
	for i := 0; i < 20; i++ { // shrink back across the line, deleting from the front
		if !m.Delete(Int(int64(i))) || m.Delete(Int(int64(i))) {
			t.Fatalf("delete %d", i)
		}
		for j := i + 1; j < 20; j++ {
			if _, ok := m.Get(Int(int64(j))); !ok {
				t.Fatalf("after deleting %d, %d is gone (size %d)", i, j, m.Len())
			}
		}
	}
	if m.Len() != 20 || m.Entries()[0].K != `"0"` || m.KeyOf(`"5"`).Inspect() != "5" {
		t.Fatalf("left: %s", m.Inspect())
	}
	small, big := NewMap(), NewMap()
	for i := 0; i < 12; i++ {
		big.Put(Int(int64(i)), Int(int64(i)))
	}
	for i := 11; i >= 0; i-- {
		small.Put(Int(int64(i)), Int(int64(i)))
	}
	if !Equal(small, big) || Key(small) != Key(big) {
		t.Fatalf("maps with the same entries in another order should be equal")
	}
}
