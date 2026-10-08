let best = Infinity, count;
for (let r = 0; r < 3; r++) { const t = performance.now(); const n = 1000000; const flags = new Array(n).fill(true); count = 0; for (let i = 2; i < n; i++) { if (flags[i]) { count++; for (let j = i * i; j < n; j += i) flags[j] = false; } } best = Math.min(best, performance.now() - t); }
console.log(`result=${count} ms=${best.toFixed(1)}`);
