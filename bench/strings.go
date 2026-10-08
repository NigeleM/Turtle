//go:build ignore

package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

func main() {
	best, total := time.Hour, 0
	for r := 0; r < 3; r++ {
		t := time.Now()
		parts := []string{}
		for i := 0; i < 200000; i++ {
			parts = append(parts, "item"+strconv.Itoa(i))
		}
		back := strings.Split(strings.Join(parts, ","), ",")
		total = 0
		for _, p := range back {
			total += len(p)
		}
		best = min(best, time.Since(t))
	}
	fmt.Printf("result=%d ms=%.1f\n", total, float64(best.Microseconds())/1000)
}
