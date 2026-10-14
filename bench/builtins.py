import time
best = 1e18
for _ in range(3):
    t = time.perf_counter(); total = 0
    for i in range(3000000):
        total = (total + abs(i - 1500000)) % 1000000007
    best = min(best, (time.perf_counter() - t) * 1000)
print(f"result={total} ms={best:.1f}")
