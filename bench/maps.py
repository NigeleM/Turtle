import time
best = 1e18
for _ in range(3):
    t = time.perf_counter(); m = {}
    for i in range(200000): m[f"k{i}"] = i
    total = 0
    for i in range(200000): total += m[f"k{i}"]
    best = min(best, (time.perf_counter() - t) * 1000)
print(f"result={total} ms={best:.1f}")
