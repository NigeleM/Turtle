import time, re
best = 1e18
for _ in range(3):
    t = time.perf_counter()
    text = "\n".join(f"id={i} name=item{i} qty={i % 7}" for i in range(50000))
    total = sum(int(q) for q in re.findall(r"qty=(\d+)", text))
    best = min(best, (time.perf_counter() - t) * 1000)
print(f"result={total} ms={best:.1f}")
