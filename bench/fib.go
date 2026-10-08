//go:build ignore

package main

import (
	"fmt"
	"time"
)

func fib(n int) int {
	if n < 2 {
		return n
	}
	return fib(n-1) + fib(n-2)
}

func main() {
	best, result := time.Hour, 0
	for r := 0; r < 3; r++ {
		t := time.Now()
		result = fib(27)
		best = min(best, time.Since(t))
	}
	fmt.Printf("result=%d ms=%.1f\n", result, float64(best.Microseconds())/1000)
}
