import time
class Point:
    __slots__ = ("x", "y")
    def __init__(self, x, y): self.x = x; self.y = y
best = 1e18
for _ in range(3):
    t = time.perf_counter(); points = [Point(i, i + 1) for i in range(200000)]; total = 0
    for p in points: total += p.x * p.y % 1000
    best = min(best, (time.perf_counter() - t) * 1000)
print(f"result={total} ms={best:.1f}")
