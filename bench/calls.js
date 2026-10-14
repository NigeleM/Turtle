function mix(a, b) { return (a * 31 + b) % 1000000007; }
let best = Infinity, acc;
for (let r = 0; r < 3; r++) { const t = performance.now(); acc = 0; for (let i = 0; i < 3000000; i++) acc = mix(acc, i); best = Math.min(best, performance.now() - t); }
console.log(`result=${acc} ms=${best.toFixed(1)}`);
