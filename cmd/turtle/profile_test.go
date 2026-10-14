// Copyright 2017-2026 Nigele McCoy
// Licensed under the Apache License, Version 2.0;
// see LICENSE and NOTICE.

package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStartProfileWritesFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cpu.prof")
	t.Setenv("TURTLE_CPUPROFILE", path)
	stop := startProfile()
	stop()
	info, err := os.Stat(path)
	if err != nil || info.Size() == 0 {
		t.Fatalf("profile not written: %v", err)
	}
}

func TestStartProfileOffByDefault(t *testing.T) {
	t.Setenv("TURTLE_CPUPROFILE", "")
	startProfile()()
}
