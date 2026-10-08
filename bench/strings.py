import time
best = 1e18
for _ in range(3):
    t = time.perf_counter()
    parts = []
    for i in range(200000): parts.append(f"item{i}")
    text = ",".join(parts); back = text.split(","); total = 0
    for p in back: total += len(p)
    best = min(best, (time.perf_counter() - t) * 1000)
print(f"result={total} ms={best:.1f}")
