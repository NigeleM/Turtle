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
	best, found := time.Hour, 0
	for r := 0; r < 3; r++ {
		t := time.Now()
		nums := make([]int, 300000)
		for i := range nums {
			nums[i] = i * 2
		}
		found = 0
		x := 1
		for q := 0; q < 300000; q++ {
			x = (x*75 + 74) % 600001
			lo, hi := 0, 299999
			for lo <= hi {
				mid := (lo + hi) / 2
				v := nums[mid]
				if v == x {
					found++
					lo = hi + 1
				} else if v < x {
					lo = mid + 1
				} else {
					hi = mid - 1
				}
			}
		}
		best = min(best, time.Since(t))
	}
	fmt.Printf("result=%d ms=%.1f\n", found, float64(best.Microseconds())/1000)
}
