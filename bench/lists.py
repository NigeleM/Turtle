import time
best = 1e18
for _ in range(3):
    t = time.perf_counter(); nums = []; x = 1
    for i in range(300000):
        x = (x * 1103515245 + 12345) % 2147483648; nums.append(x)
    nums.sort(); total = sum(nums[i] for i in range(0, 300000, 1000))
    best = min(best, (time.perf_counter() - t) * 1000)
print(f"result={total} ms={best:.1f}")
