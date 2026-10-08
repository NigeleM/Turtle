let best = 1e18, check = 0;
const n = 200;
for (let r = 0; r < 3; r++) {
  const t = performance.now();
  const a = [], b = [], rhs = [];
  for (let i = 0; i < n; i++) {
    a.push(new Float64Array(n)); b.push(new Float64Array(n));
    for (let j = 0; j < n; j++) { const v = i * 7 + j * 13; a[i][j] = v % 10; b[i][j] = v % 7; }
    a[i][i] = n * 10 + i % 10; rhs.push(i % 7 + 1);
  }
  let total = 0;
  for (let i = 0; i < n; i++) for (let j = 0; j < n; j++) { let s = 0; for (let k = 0; k < n; k++) s += a[i][k] * b[k][j]; total += s; }
  const m = a.map((row, i) => { const x = Array.from(row); x.push(rhs[i]); return x; });
  for (let k = 0; k < n; k++) {
    let p = k; for (let r2 = k + 1; r2 < n; r2++) if (Math.abs(m[r2][k]) > Math.abs(m[p][k])) p = r2;
    [m[k], m[p]] = [m[p], m[k]];
    for (let r2 = k + 1; r2 < n; r2++) { const f = m[r2][k] / m[k][k]; if (f) for (let j = k; j <= n; j++) m[r2][j] -= f * m[k][j]; }
  }
  const x = new Array(n).fill(0);
  for (let i = n - 1; i >= 0; i--) { let s = m[i][n]; for (let j = i + 1; j < n; j++) s -= m[i][j] * x[j]; x[i] = s / m[i][i]; }
  check = total + Math.round(x.reduce((p, q) => p + q, 0) * 1000000);
  best = Math.min(best, performance.now() - t);
}
console.log(`result=${check} ms=${best.toFixed(1)}`);
