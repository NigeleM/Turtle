let best = Infinity, total;
for (let r = 0; r < 3; r++) { const t = performance.now(); let s = 0; for (let i = 0; i < 3000000; i++) s = s + (i * i) % 1000000007; total = s; best = Math.min(best, performance.now() - t); }
console.log(`result=${total} ms=${best.toFixed(1)}`);
