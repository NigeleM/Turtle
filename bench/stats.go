// Copyright 2017-2026 Nigele McCoy
// Licensed under the Apache License, Version 2.0;
// see LICENSE and NOTICE.

//go:build ignore

package main

import (
	"fmt"
	"math"
	"sort"
	"time"
)

func main() {
	best, check := time.Hour, int64(0)
	for r := 0; r < 3; r++ {
		t := time.Now()
		nums := make([]float64, 300000)
		for i := range nums {
			nums[i] = float64(i*7919%1000) / 10
		}
		n := float64(len(nums))
		sum := 0.0
		for _, x := range nums {
			sum += x
		}
		mean := sum / n
		ss := 0.0
		for _, x := range nums {
			ss += (x - mean) * (x - mean)
		}
		sd := math.Sqrt(ss / (n - 1))
		s := append([]float64(nil), nums...)
		sort.Float64s(s)
		k := len(s)
		median := (s[k/2-1] + s[k/2]) / 2
		if k%2 == 1 {
			median = s[k/2]
		}
		pos := 0.9 * float64(k-1)
		lo := int(pos)
		p90 := s[lo] + (pos-float64(lo))*(s[lo+1]-s[lo])
		check = int64(math.Round((mean + sd + median + p90) * 1000000))
		best = min(best, time.Since(t))
	}
	fmt.Printf("result=%d ms=%.1f\n", check, float64(best.Microseconds())/1000)
}
