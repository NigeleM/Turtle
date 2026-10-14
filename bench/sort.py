import time
best = 1e18
for _ in range(3):
    t = time.perf_counter(); nums = []; x = 1
    for i in range(300000):
        x = (x * 75 + 74) % 65537; nums.append(x)
    nums.sort(); total = 0
    for i in range(300000):
        total = (total + nums[i] * (i % 7 + 1)) % 1000000007
    best = min(best, (time.perf_counter() - t) * 1000)
print(f"result={total} ms={best:.1f}")
