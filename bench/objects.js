let best = Infinity, total;
for (let r = 0; r < 3; r++) { const t = performance.now(); const points = []; for (let i = 0; i < 200000; i++) points.push({ x: i, y: i + 1 }); total = 0; for (const p of points) total += (p.x * p.y) % 1000; best = Math.min(best, performance.now() - t); }
console.log(`result=${total} ms=${best.toFixed(1)}`);
