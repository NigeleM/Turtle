// Copyright 2017-2026 Nigele McCoy
// Licensed under the Apache License, Version 2.0;
// see LICENSE and NOTICE.

package main

import (
	"os"
	"runtime/pprof"
)

// startProfile writes a CPU profile of the run to the file named by
// TURTLE_CPUPROFILE, for finding where a program spends its time (see
// go tool pprof). Without the variable it does nothing. The returned
// function ends the profile and must run before the program exits.
func startProfile() func() {
	path := os.Getenv("TURTLE_CPUPROFILE")
	if path == "" {
		return func() {}
	}
	f, err := os.Create(path)
	if err != nil {
		return func() {}
	}
	if err := pprof.StartCPUProfile(f); err != nil {
		f.Close()
		return func() {}
	}
	return func() {
		pprof.StopCPUProfile()
		f.Close()
	}
}
