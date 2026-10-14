let best = Infinity, found;
for (let r = 0; r < 3; r++) { const t = performance.now(); const nums = []; for (let i = 0; i < 300000; i++) nums.push(i * 2); found = 0; let x = 1; for (let q = 0; q < 300000; q++) { x = (x * 75 + 74) % 600001; let lo = 0, hi = 299999; while (lo <= hi) { const mid = Math.floor((lo + hi) / 2); const v = nums[mid]; if (v === x) { found++; lo = hi + 1; } else if (v < x) { lo = mid + 1; } else { hi = mid - 1; } } } best = Math.min(best, performance.now() - t); }
console.log(`result=${found} ms=${best.toFixed(1)}`);
