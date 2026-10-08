//go:build ignore

package main

import (
	"fmt"
	"time"
)

type Point struct{ X, Y int }

func main() {
	best, total := time.Hour, 0
	for r := 0; r < 3; r++ {
		t := time.Now()
		points := []*Point{}
		for i := 0; i < 200000; i++ {
			points = append(points, &Point{i, i + 1})
		}
		total = 0
		for _, p := range points {
			total += p.X * p.Y % 1000
		}
		best = min(best, time.Since(t))
	}
	fmt.Printf("result=%d ms=%.1f\n", total, float64(best.Microseconds())/1000)
}
