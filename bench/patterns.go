//go:build ignore

package main

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

func main() {
	best, total := time.Hour, 0
	re := regexp.MustCompile(`qty=(\d+)`)
	for r := 0; r < 3; r++ {
		t := time.Now()
		lines := []string{}
		for i := 0; i < 50000; i++ {
			lines = append(lines, fmt.Sprintf("id=%d name=item%d qty=%d", i, i, i%7))
		}
		total = 0
		for _, m := range re.FindAllStringSubmatch(strings.Join(lines, "\n"), -1) {
			n, _ := strconv.Atoi(m[1])
			total += n
		}
		best = min(best, time.Since(t))
	}
	fmt.Printf("result=%d ms=%.1f\n", total, float64(best.Microseconds())/1000)
}
