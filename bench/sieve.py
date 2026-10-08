import time
best = 1e18
for _ in range(3):
    t = time.perf_counter(); n = 1000000; flags = [True] * n; count = 0
    for i in range(2, n):
        if flags[i]:
            count += 1
            for j in range(i * i, n, i): flags[j] = False
    best = min(best, (time.perf_counter() - t) * 1000)
print(f"result={count} ms={best:.1f}")
