import time
best = 1e18
for _ in range(3):
    t = time.perf_counter(); nums = [i * 2 for i in range(300000)]
    found = 0; x = 1
    for q in range(300000):
        x = (x * 75 + 74) % 600001
        lo, hi = 0, 299999
        while lo <= hi:
            mid = (lo + hi) // 2
            v = nums[mid]
            if v == x:
                found += 1; lo = hi + 1
            elif v < x:
                lo = mid + 1
            else:
                hi = mid - 1
    best = min(best, (time.perf_counter() - t) * 1000)
print(f"result={found} ms={best:.1f}")
