// Copyright 2017-2026 Nigele McCoy
// Licensed under the Apache License, Version 2.0;
// see LICENSE and NOTICE.

//go:build ignore

package main

import (
	"fmt"
	"sort"
	"time"
)

func main() {
	best, total := time.Hour, 0
	for r := 0; r < 3; r++ {
		t := time.Now()
		nums := []int{}
		x := 1
		for i := 0; i < 300000; i++ {
			x = (x*75 + 74) % 65537
			nums = append(nums, x)
		}
		sort.Ints(nums)
		total = 0
		for i := 0; i < 300000; i++ {
			total = (total + nums[i]*(i%7+1)) % 1000000007
		}
		best = min(best, time.Since(t))
	}
	fmt.Printf("result=%d ms=%.1f\n", total, float64(best.Microseconds())/1000)
}
