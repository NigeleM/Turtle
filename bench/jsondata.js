let best = Infinity, total;
for (let r = 0; r < 3; r++) { const t = performance.now(); const records = []; for (let i = 0; i < 20000; i++) records.push({ id: i, name: `item${i}`, price: i % 100, tags: ["a", "b"] }); const back = JSON.parse(JSON.stringify(records)); total = 0; for (const x of back) total += x.price; best = Math.min(best, performance.now() - t); }
console.log(`result=${total} ms=${best.toFixed(1)}`);
