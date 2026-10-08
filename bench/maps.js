let best = Infinity, total;
for (let r = 0; r < 3; r++) { const t = performance.now(); const m = new Map(); for (let i = 0; i < 200000; i++) m.set(`k${i}`, i); total = 0; for (let i = 0; i < 200000; i++) total += m.get(`k${i}`); best = Math.min(best, performance.now() - t); }
console.log(`result=${total} ms=${best.toFixed(1)}`);
