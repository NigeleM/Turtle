//go:build ignore

package main

import (
	"fmt"
	"math"
	"time"
)

func main() {
	const n = 200
	best, check := time.Hour, int64(0)
	for r := 0; r < 3; r++ {
		t := time.Now()
		a, b, rhs := make([][]float64, n), make([][]float64, n), make([]float64, n)
		for i := range n {
			a[i], b[i] = make([]float64, n), make([]float64, n)
			for j := range n {
				v := i*7 + j*13
				a[i][j], b[i][j] = float64(v%10), float64(v%7)
			}
			a[i][i] = float64(n*10 + i%10)
			rhs[i] = float64(i%7 + 1)
		}
		total := 0.0
		for i := range n {
			for k := range n {
				f := a[i][k]
				for j := range n {
					total += f * b[k][j]
				}
			}
		}
		m := make([][]float64, n)
		for i := range n {
			m[i] = append(append([]float64{}, a[i]...), rhs[i])
		}
		for k := range n {
			p := k
			for r2 := k + 1; r2 < n; r2++ {
				if math.Abs(m[r2][k]) > math.Abs(m[p][k]) {
					p = r2
				}
			}
			m[k], m[p] = m[p], m[k]
			for r2 := k + 1; r2 < n; r2++ {
				f := m[r2][k] / m[k][k]
				for j := k; j <= n; j++ {
					m[r2][j] -= f * m[k][j]
				}
			}
		}
		x := make([]float64, n)
		sum := 0.0
		for i := n - 1; i >= 0; i-- {
			s := m[i][n]
			for j := i + 1; j < n; j++ {
				s -= m[i][j] * x[j]
			}
			x[i] = s / m[i][i]
			sum += x[i]
		}
		check = int64(total) + int64(math.Round(sum*1000000))
		best = min(best, time.Since(t))
	}
	fmt.Printf("result=%d ms=%.1f\n", check, float64(best.Microseconds())/1000)
}
