let best = Infinity, total;
for (let r = 0; r < 3; r++) { const t = performance.now(); total = 0; for (let i = 0; i < 3000000; i++) total = (total + Math.abs(i - 1500000)) % 1000000007; best = Math.min(best, performance.now() - t); }
console.log(`result=${total} ms=${best.toFixed(1)}`);
