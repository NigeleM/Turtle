let best = 1e18, check = 0;
for (let r = 0; r < 3; r++) {
  const t = performance.now();
  const nums = [];
  for (let i = 0; i < 300000; i++) nums.push(i * 7919 % 1000 / 10);
  const n = nums.length;
  const mean = nums.reduce((a, b) => a + b, 0) / n;
  const sd = Math.sqrt(nums.reduce((a, x) => a + (x - mean) ** 2, 0) / (n - 1));
  const s = Float64Array.from(nums).sort();
  const median = n % 2 ? s[(n - 1) / 2] : (s[n / 2 - 1] + s[n / 2]) / 2;
  const pos = 0.9 * (n - 1), lo = Math.floor(pos);
  const p90 = s[lo] + (pos - lo) * (s[lo + 1] - s[lo]);
  check = Math.round((mean + sd + median + p90) * 1000000);
  best = Math.min(best, performance.now() - t);
}
console.log(`result=${check} ms=${best.toFixed(1)}`);
