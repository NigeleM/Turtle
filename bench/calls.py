import time
def mix(a, b):
    return (a * 31 + b) % 1000000007
best = 1e18
for _ in range(3):
    t = time.perf_counter(); acc = 0
    for i in range(3000000):
        acc = mix(acc, i)
    best = min(best, (time.perf_counter() - t) * 1000)
print(f"result={acc} ms={best:.1f}")
