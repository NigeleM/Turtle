let best = Infinity, total;
for (let r = 0; r < 3; r++) { const t = performance.now(); const nums = []; let x = 1n; for (let i = 0; i < 300000; i++) { x = (x * 1103515245n + 12345n) % 2147483648n; nums.push(Number(x)); } nums.sort((a, b) => a - b); total = 0; for (let i = 0; i < 300000; i += 1000) total += nums[i]; best = Math.min(best, performance.now() - t); }
console.log(`result=${total} ms=${best.toFixed(1)}`);
