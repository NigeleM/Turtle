//go:build ignore

package main

import (
	"fmt"
	"time"
)

func main() {
	best, count := time.Hour, 0
	for r := 0; r < 3; r++ {
		t := time.Now()
		n := 1000000
		flags := make([]bool, n)
		for i := range flags {
			flags[i] = true
		}
		count = 0
		for i := 2; i < n; i++ {
			if flags[i] {
				count++
				for j := i * i; j < n; j += i {
					flags[j] = false
				}
			}
		}
		best = min(best, time.Since(t))
	}
	fmt.Printf("result=%d ms=%.1f\n", count, float64(best.Microseconds())/1000)
}
