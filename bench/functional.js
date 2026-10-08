let best = Infinity, total; const nums = Array.from({ length: 300000 }, (_, i) => i + 1);
for (let r = 0; r < 3; r++) { const t = performance.now(); total = nums.map(n => n * 2).filter(n => n % 3 === 0).reduce((a, b) => a + b, 0); best = Math.min(best, performance.now() - t); }
console.log(`result=${total} ms=${best.toFixed(1)}`);
