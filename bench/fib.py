import time
def fib(n):
    return n if n < 2 else fib(n - 1) + fib(n - 2)
best = None
for _ in range(3):
    t = time.perf_counter(); result = fib(27); took = (time.perf_counter() - t) * 1000
    best = took if best is None or took < best else best
print(f"result={result} ms={best:.1f}")
