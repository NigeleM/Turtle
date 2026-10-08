import time, json
best = 1e18
for _ in range(3):
    t = time.perf_counter()
    records = [{"id": i, "name": f"item{i}", "price": i % 100, "tags": ["a", "b"]} for i in range(20000)]
    back = json.loads(json.dumps(records)); total = sum(r["price"] for r in back)
    best = min(best, (time.perf_counter() - t) * 1000)
print(f"result={total} ms={best:.1f}")
