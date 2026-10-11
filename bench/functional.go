// Copyright 2017-2026 Nigele McCoy
// Licensed under the Apache License, Version 2.0;
// see LICENSE and NOTICE.

//go:build ignore

package main

import (
	"fmt"
	"time"
)

func mapf(xs []int, f func(int) int) []int {
	out := make([]int, 0, len(xs))
	for _, x := range xs {
		out = append(out, f(x))
	}
	return out
}
func filter(xs []int, f func(int) bool) []int {
	out := []int{}
	for _, x := range xs {
		if f(x) {
			out = append(out, x)
		}
	}
	return out
}
func reduce(xs []int, start int, f func(int, int) int) int {
	for _, x := range xs {
		start = f(start, x)
	}
	return start
}

func main() {
	best, total := time.Hour, 0
	nums := make([]int, 300000)
	for i := range nums {
		nums[i] = i + 1
	}
	for r := 0; r < 3; r++ {
		t := time.Now()
		total = reduce(filter(mapf(nums, func(n int) int { return n * 2 }), func(n int) bool { return n%3 == 0 }), 0, func(a, b int) int { return a + b })
		best = min(best, time.Since(t))
	}
	fmt.Printf("result=%d ms=%.1f\n", total, float64(best.Microseconds())/1000)
}
