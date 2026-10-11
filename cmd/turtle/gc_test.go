// Copyright 2017-2026 Nigele McCoy
// Licensed under the Apache License, Version 2.0;
// see LICENSE and NOTICE.

package main

import (
	"runtime"
	"runtime/debug"
	"testing"
	"time"
)

func TestGCPercentFor(t *testing.T) {
	cases := map[uint64]int{
		0:         gcMaxPercent,
		1 << 20:   gcMaxPercent, // 1 MB kept: collect rarely
		8 << 20:   400,
		16 << 20:  200,
		32 << 20:  100,
		512 << 20: gcMinPercent, // never under Go's default
	}
	for live, want := range cases {
		if got := gcPercentFor(live); got != want {
			t.Errorf("gcPercentFor(%d MB) = %d, want %d", live>>20, got, want)
		}
	}
}

// A program that keeps a lot collects more often than one that keeps
// little.
func TestAdaptGCLowersThePercent(t *testing.T) {
	t.Setenv("GOGC", "")
	defer debug.SetGCPercent(debug.SetGCPercent(100))
	adaptGC()
	if p := percent(); p != gcMaxPercent {
		t.Fatalf("at the start: %d, want %d", p, gcMaxPercent)
	}
	kept := make([][]byte, 0, 200)
	for i := 0; i < 200; i++ {
		kept = append(kept, make([]byte, 1<<20)) // 200 MB kept
	}
	deadline := time.Now().Add(5 * time.Second)
	for percent() != gcMinPercent && time.Now().Before(deadline) {
		runtime.GC()
		time.Sleep(10 * time.Millisecond)
	}
	if p := percent(); p != gcMinPercent {
		t.Errorf("with 200 MB kept: %d, want %d", p, gcMinPercent)
	}
	runtime.KeepAlive(kept)
}

func percent() int {
	p := debug.SetGCPercent(100)
	debug.SetGCPercent(p)
	return p
}
