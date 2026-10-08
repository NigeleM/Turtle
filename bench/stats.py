import time, statistics
best = 1e18
for _ in range(3):
    t = time.perf_counter()
    nums = [i * 7919 % 1000 / 10 for i in range(300000)]
    q = statistics.quantiles(nums, n=10, method="inclusive")[8]
    total = statistics.fmean(nums) + statistics.stdev(nums) + statistics.median(nums) + q
    check = round(total * 1000000)
    best = min(best, (time.perf_counter() - t) * 1000)
print(f"result={check} ms={best:.1f}")
