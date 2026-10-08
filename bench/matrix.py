import time
best = 1e18
n = 200
for _ in range(3):
    t = time.perf_counter()
    a = [[0.0] * n for _ in range(n)]; b = [[0] * n for _ in range(n)]; rhs = []
    for i in range(n):
        for j in range(n):
            v = i * 7 + j * 13
            a[i][j] = v % 10; b[i][j] = v % 7
        a[i][i] = n * 10 + i % 10
        rhs.append(i % 7 + 1)
    bt = list(zip(*b))
    c = [[sum(x * y for x, y in zip(row, col)) for col in bt] for row in a]
    total = sum(sum(row) for row in c)
    m = [row[:] + [rhs[i]] for i, row in enumerate(a)]
    for k in range(n):
        p = max(range(k, n), key=lambda r: abs(m[r][k]))
        m[k], m[p] = m[p], m[k]
        for r in range(k + 1, n):
            f = m[r][k] / m[k][k]
            if f:
                rk, rr = m[k], m[r]
                for j in range(k, n + 1):
                    rr[j] -= f * rk[j]
    x = [0.0] * n
    for i in range(n - 1, -1, -1):
        s = m[i][n] - sum(m[i][j] * x[j] for j in range(i + 1, n))
        x[i] = s / m[i][i]
    check = int(total + round(sum(x) * 1000000))
    best = min(best, (time.perf_counter() - t) * 1000)
print(f"result={check} ms={best:.1f}")
