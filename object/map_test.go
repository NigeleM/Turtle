// Copyright 2017-2026 Nigele McCoy
// Licensed under the Apache License, Version 2.0;
// see LICENSE and NOTICE.

package object

import (
	"fmt"
	"math/rand"
	"testing"
	"time"
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
		if Key(e.Key) != order[i] {
			t.Fatalf("entry %d is %s, want %s", i, Key(e.Key), order[i])
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
	if m.Len() != 20 || Key(m.Entries()[0].Key) != `"0"` || m.Entries()[5].Key.Inspect() != "5" {
		t.Fatalf("left: %s", m.Inspect())
	}
	small, big := NewMap(), NewMap()
	for i := 0; i < 40; i++ {
		big.Put(Int(int64(i)), Int(int64(i)))
	}
	for i := 39; i >= 0; i-- {
		small.Put(Int(int64(i)), Int(int64(i)))
	}
	if !Equal(small, big) || Key(small) != Key(big) {
		t.Fatalf("maps with the same entries in another order should be equal")
	}
}

// The map keeps exactly Key's rules for which keys are one key, whatever
// its size: random puts, gets and deletes of mixed keys, against a plain
// Go map keyed by Key.
func TestMapKeysAsKeySays(t *testing.T) {
	keys := []Object{
		&String{Value: "1"}, Int(1), &Float{Value: 1.0}, &Float{Value: 1.5}, &String{Value: "1.5"},
		&String{Value: ""}, &String{Value: "\x00"}, &String{Value: "\x001"}, &String{Value: "\x00\"1\""},
		&String{Value: "true"}, Bool(true), Bool(false), NoneValue, &String{Value: "none"},
		&Float{Value: 0.1 + 0.2}, &Float{Value: 0.3}, Int(2), &Float{Value: 2.0000000000000004}, Int(-7), Int(1 << 40),
		&String{Value: "héllo"}, &String{Value: "a,b"},
		&List{Elements: []Object{&String{Value: "a"}, &String{Value: "b"}}},
		&List{Elements: []Object{&String{Value: "a,b"}}},
		&Set{Elements: []Object{Int(1), Int(2)}}, &Set{Elements: []Object{Int(2), Int(1)}},
		&Date{Time: time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)},
		&Date{Time: time.Date(2026, 1, 2, 4, 4, 5, 0, time.FixedZone("x", 3600))},
	}
	for i := 0; i < 60; i++ { // enough to cross the index line
		keys = append(keys, &String{Value: fmt.Sprint("k", i)}, Int(int64(1000+i)))
	}
	rng := rand.New(rand.NewSource(7))
	for round := 0; round < 50; round++ {
		m := NewMap()
		want := map[string]Object{} // Key -> value
		var order []string
		for step := 0; step < 400; step++ {
			k := keys[rng.Intn(len(keys))]
			ks := Key(k)
			switch rng.Intn(4) {
			case 0, 1:
				v := Int(int64(step))
				m.Put(k, v)
				if _, ok := want[ks]; !ok {
					order = append(order, ks)
				}
				want[ks] = v
			case 2:
				got, ok := m.Get(k)
				w, wok := want[ks]
				if ok != wok || ok && got != w {
					t.Fatalf("round %d step %d: get %s = %v %v, want %v %v", round, step, ks, got, ok, w, wok)
				}
			case 3:
				_, wok := want[ks]
				if m.Delete(k) != wok {
					t.Fatalf("round %d step %d: delete %s", round, step, ks)
				}
				delete(want, ks)
				for i, o := range order {
					if o == ks {
						order = append(order[:i], order[i+1:]...)
						break
					}
				}
			}
			if m.Len() != len(want) {
				t.Fatalf("round %d step %d: %d entries, want %d", round, step, m.Len(), len(want))
			}
		}
		for i, e := range m.Entries() {
			if Key(e.Key) != order[i] || e.Val != want[order[i]] {
				t.Fatalf("round %d: entry %d is %s, want %s", round, i, Key(e.Key), order[i])
			}
		}
	}
}

// A set keeps Key's rules too.
func TestSetKeysAsKeySays(t *testing.T) {
	s := &Set{}
	for _, v := range []Object{&String{Value: "1"}, Int(1), &Float{Value: 1.0}, &String{Value: "\x001"}, &String{Value: "\x00"}, &String{Value: ""}, &Float{Value: 0.1 + 0.2}, &Float{Value: 0.3}} {
		s.Add(v)
	}
	if len(s.Elements) != 6 { // 1 and 1.0 are one; 0.1+0.2 and 0.3 are one
		t.Fatalf("set has %d elements: %s", len(s.Elements), s.Inspect())
	}
}
