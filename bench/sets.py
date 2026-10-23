import time
best = 1e18
for _ in range(3):
    t = time.perf_counter(); a = set(); b = set()
    for i in range(200000): a.add((i * 7) % 300007); b.add(i * 3)
    hits = 0
    for i in range(300000):
        if i in a: hits += 1
    total = hits + len(a & b) + len(a | b)
    best = min(best, (time.perf_counter() - t) * 1000)
print(f"result={total} ms={best:.1f}")
