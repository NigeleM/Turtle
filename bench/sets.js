let best = Infinity, total;
for (let r = 0; r < 3; r++) { const t = performance.now(); const a = new Set(), b = new Set(); for (let i = 0; i < 200000; i++) { a.add((i * 7) % 300007); b.add(i * 3); } let hits = 0; for (let i = 0; i < 300000; i++) if (a.has(i)) hits++; let both = 0; for (const x of a) if (b.has(x)) both++; const either = new Set(a); for (const x of b) either.add(x); total = hits + both + either.size; best = Math.min(best, performance.now() - t); }
console.log(`result=${total} ms=${best.toFixed(1)}`);
