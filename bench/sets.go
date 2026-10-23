// Copyright 2017-2026 Nigele McCoy
// Licensed under the Apache License, Version 2.0;
// see LICENSE and NOTICE.

//go:build ignore

package main

import (
	"fmt"
	"time"
)

func main() {
	best, total := time.Hour, 0
	for r := 0; r < 3; r++ {
		t := time.Now()
		a, b := map[int]struct{}{}, map[int]struct{}{}
		for i := 0; i < 200000; i++ {
			a[(i*7)%300007] = struct{}{}
			b[i*3] = struct{}{}
		}
		hits := 0
		for i := 0; i < 300000; i++ {
			if _, ok := a[i]; ok {
				hits++
			}
		}
		both := 0
		either := make(map[int]struct{}, len(a)+len(b))
		for x := range a {
			if _, ok := b[x]; ok {
				both++
			}
			either[x] = struct{}{}
		}
		for x := range b {
			either[x] = struct{}{}
		}
		total = hits + both + len(either)
		best = min(best, time.Since(t))
	}
	fmt.Printf("result=%d ms=%.1f\n", total, float64(best.Microseconds())/1000)
}
