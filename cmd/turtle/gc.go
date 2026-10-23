// Copyright 2017-2026 Nigele McCoy
// Licensed under the Apache License, Version 2.0;
// see LICENSE and NOTICE.

package main

import (
	"os"
	"runtime"
	"runtime/debug"
	"runtime/metrics"
)

// How memory may grow between garbage collections. A Turtle program makes
// many short-lived values and keeps few, so collecting rarely is faster;
// but how far memory grows is a share of what the program keeps, so a
// program that keeps a lot would grow to several times that. After each
// collection the share is set again from what's kept: gcMaxPercent while
// that's small, less as it grows, so the garbage allowed stays near
// gcAllowance, and never under gcMinPercent (half Go's own default: a
// program that keeps a lot peaks near one and a half times what it keeps,
// not twice, for a little more collecting, measured at no loss of speed).
const (
	gcMaxPercent = 150
	gcMinPercent = 50
	gcAllowance  = 16 << 20 // bytes
)

// gcMark is the value that marks a collection: big enough not to share
// memory with other small values (whose finalizers may never run).
type gcMark struct{ _ [32]byte }

// adaptGC starts the adapting; GOGC set by the user wins.
func adaptGC() {
	if os.Getenv("GOGC") != "" {
		return
	}
	debug.SetGCPercent(gcMaxPercent)
	current := gcMaxPercent
	sample := []metrics.Sample{{Name: "/gc/heap/live:bytes"}}
	var watch func()
	watch = func() {
		// Runs after each collection: a value with a finalizer, made
		// again each time.
		runtime.SetFinalizer(new(gcMark), func(*gcMark) { watch() })
		metrics.Read(sample)
		if sample[0].Value.Kind() != metrics.KindUint64 {
			return
		}
		p := gcPercentFor(sample[0].Value.Uint64())
		// Only a real change: setting it stops the program for a moment.
		// The two ends are always taken, or a program that keeps a lot would
		// stop short of the floor: 110 is inside the band that a change must
		// pass to count.
		if p == gcMinPercent || p == gcMaxPercent || p > current+current/4 || p < current-current/4 {
			debug.SetGCPercent(p)
			current = p
		}
	}
	watch()
}

// gcPercentFor is the share for a program keeping live bytes.
func gcPercentFor(live uint64) int {
	if live == 0 {
		return gcMaxPercent
	}
	return max(gcMinPercent, min(gcMaxPercent, int(gcAllowance*100/live)))
}
