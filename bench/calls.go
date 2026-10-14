// Copyright 2017-2026 Nigele McCoy
// Licensed under the Apache License, Version 2.0;
// see LICENSE and NOTICE.

//go:build ignore

package main

import (
	"fmt"
	"time"
)

func mix(a, b int) int {
	return (a*31 + b) % 1000000007
}

func main() {
	best, acc := time.Hour, 0
	for r := 0; r < 3; r++ {
		t := time.Now()
		acc = 0
		for i := 0; i < 3000000; i++ {
			acc = mix(acc, i)
		}
		best = min(best, time.Since(t))
	}
	fmt.Printf("result=%d ms=%.1f\n", acc, float64(best.Microseconds())/1000)
}
