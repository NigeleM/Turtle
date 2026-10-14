import time, statistics, bisect
def make(n):
    out = []; x = 1
    for _ in range(n):
        x = (x * 75 + 74) % 65537; out.append(x)
    return out
nums = make(200000); small = make(3000); sorted_nums = sorted(nums)
def best(f, runs=3):
    b = 1e18
    for _ in range(runs):
        t = time.perf_counter(); f(); b = min(b, (time.perf_counter() - t) * 1000)
    return round(b)
def ins(xs):
    xs = list(xs)
    for i in range(1, len(xs)):
        v = xs[i]; j = i - 1
        while j >= 0 and xs[j] > v:
            xs[j + 1] = xs[j]; j -= 1
        xs[j + 1] = v
    return xs
def heap(xs):
    import heapq
    h = list(xs); heapq.heapify(h); return [heapq.heappop(h) for _ in range(len(h))]
def shell(xs):
    xs = list(xs); gap = len(xs) // 2
    while gap > 0:
        for i in range(gap, len(xs)):
            v = xs[i]; j = i
            while j >= gap and xs[j - gap] > v:
                xs[j] = xs[j - gap]; j -= gap
            xs[j] = v
        gap //= 2
    return xs
def msort(xs):
    return sorted(xs)
tests = [
 ("quick_sort", lambda: msort(nums)), ("merge_sort", lambda: msort(nums)),
 ("heap_sort", lambda: heap(nums)), ("shell_sort", lambda: shell(nums)),
 ("insertion_sort 3k", lambda: ins(small)),
 ("sort built-in", lambda: sorted(nums)),
 ("mean", lambda: statistics.fmean(nums)), ("median", lambda: statistics.median(nums)),
 ("stdev", lambda: statistics.stdev(nums)),
 ("percentile 90", lambda: statistics.quantiles(nums, n=100)[89]),
 ("describe", lambda: (min(nums), max(nums), statistics.fmean(nums), statistics.stdev(nums), statistics.median(nums))),
 ("binary_search", lambda: bisect.bisect_left(sorted_nums, 4000)),
 ("linear_search", lambda: nums.index(4000) if 4000 in nums else -1),
]
print("stdlib (ms, best of 3) — python")
for label, f in tests:
    print(label, best(f))
