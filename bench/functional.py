import time
from functools import reduce
best = 1e18; nums = list(range(1, 300001))
for _ in range(3):
    t = time.perf_counter()
    doubled = list(map(lambda n: n * 2, nums)); thirds = list(filter(lambda n: n % 3 == 0, doubled)); total = reduce(lambda a, b: a + b, thirds, 0)
    best = min(best, (time.perf_counter() - t) * 1000)
print(f"result={total} ms={best:.1f}")
