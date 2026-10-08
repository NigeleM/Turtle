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
			x = (x*1103515245 + 12345) % 2147483648
			nums = append(nums, x)
		}
		sort.Ints(nums)
		total = 0
		for i := 0; i < 300000; i += 1000 {
			total += nums[i]
		}
		best = min(best, time.Since(t))
	}
	fmt.Printf("result=%d ms=%.1f\n", total, float64(best.Microseconds())/1000)
}
