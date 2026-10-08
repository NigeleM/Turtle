function fib(n) { return n < 2 ? n : fib(n - 1) + fib(n - 2); }
let best = Infinity, result;
for (let r = 0; r < 3; r++) { const t = performance.now(); result = fib(27); best = Math.min(best, performance.now() - t); }
console.log(`result=${result} ms=${best.toFixed(1)}`);
