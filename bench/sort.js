let best = Infinity, total;
for (let r = 0; r < 3; r++) { const t = performance.now(); const nums = []; let x = 1; for (let i = 0; i < 300000; i++) { x = (x * 75 + 74) % 65537; nums.push(x); } nums.sort((a, b) => a - b); total = 0; for (let i = 0; i < 300000; i++) total = (total + nums[i] * (i % 7 + 1)) % 1000000007; best = Math.min(best, performance.now() - t); }
console.log(`result=${total} ms=${best.toFixed(1)}`);
