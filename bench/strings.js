let best = Infinity, total;
for (let r = 0; r < 3; r++) { const t = performance.now(); const parts = []; for (let i = 0; i < 200000; i++) parts.push(`item${i}`); const back = parts.join(",").split(","); total = 0; for (const p of back) total += p.length; best = Math.min(best, performance.now() - t); }
console.log(`result=${total} ms=${best.toFixed(1)}`);
