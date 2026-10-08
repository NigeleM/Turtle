//go:build ignore

package main

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"
)

func main() {
	best, total := time.Hour, 0
	for r := 0; r < 3; r++ {
		t := time.Now()
		records := []map[string]any{}
		for i := 0; i < 20000; i++ {
			records = append(records, map[string]any{"id": i, "name": "item" + strconv.Itoa(i), "price": i % 100, "tags": []string{"a", "b"}})
		}
		data, _ := json.Marshal(records)
		var back []map[string]any
		json.Unmarshal(data, &back)
		total = 0
		for _, x := range back {
			total += int(x["price"].(float64))
		}
		best = min(best, time.Since(t))
	}
	fmt.Printf("result=%d ms=%.1f\n", total, float64(best.Microseconds())/1000)
}
