let best = Infinity, total;
for (let r = 0; r < 3; r++) { const t = performance.now(); const lines = []; for (let i = 0; i < 50000; i++) lines.push(`id=${i} name=item${i} qty=${i % 7}`); const text = lines.join("\n"); total = 0; for (const m of text.matchAll(/qty=(\d+)/g)) total += parseInt(m[1]); best = Math.min(best, performance.now() - t); }
console.log(`result=${total} ms=${best.toFixed(1)}`);
