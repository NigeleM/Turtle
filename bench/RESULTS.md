# Turtle benchmark results

Made by `turtle run.trt` in this folder on 2026-10-08.
Times are milliseconds: each program times its own work, best of 3 runs,
so starting the language isn't counted (startup is measured on its own,
from start to finish). Every language computes the same answer, checked.

| benchmark | Turtle ms | Python ms | Node ms | Ruby ms | Go ms |
|---|---|---|---|---|---|
| fib | 33.0 | 8.6 | 0.8 | 9.4 | 0.4 |
| loop | 207.0 | 131.9 | 20.7 | 75.6 | 1.2 |
| strings | 32.0 | 21.2 | 8.1 | 31.1 | 9.6 |
| lists | 85.0 | 62.3 | 41.8 | 29.6 | 15.1 |
| maps | 76.0 | 34.8 | 22.5 | 44.5 | 19.3 |
| sieve | 225.0 | 63.4 | 3.0 | 62.9 | 1.6 |
| objects | 29.0 | 21.8 | 10.4 | 18.4 | 3.5 |
| jsondata | 30.0 | 14.5 | 5.3 | 29.8 | 23.2 |
| patterns | 18.0 | 8.3 | 3.8 | 19.9 | 12.4 |
| functional | 20.0 | 14.5 | 3.6 | 13.0 | 0.7 |
| matrix | 11.0 | 258.6 | 15.8 | 386.9 | 7.9 |
| stats | 43.0 | 86.5 | 11.1 | 45.8 | 9.0 |
| startup (hello) | 6 | 14 | 23 | 40 |  |

How many times slower Turtle is than each (1.0x is the same speed; under
1.0x, Turtle is faster):

| benchmark | Turtle ÷ Python | Turtle ÷ Node | Turtle ÷ Ruby | Turtle ÷ Go |
|---|---|---|---|---|
| fib | 3.8x | 41.2x | 3.5x | 82.5x |
| loop | 1.6x | 10.0x | 2.7x | 172.5x |
| strings | 1.5x | 4.0x | 1.0x | 3.3x |
| lists | 1.4x | 2.0x | 2.9x | 5.6x |
| maps | 2.2x | 3.4x | 1.7x | 3.9x |
| sieve | 3.5x | 75.0x | 3.6x | 140.6x |
| objects | 1.3x | 2.8x | 1.6x | 8.3x |
| jsondata | 2.1x | 5.7x | 1.0x | 1.3x |
| patterns | 2.2x | 4.7x | 0.9x | 1.5x |
| functional | 1.4x | 5.6x | 1.5x | 28.6x |
| matrix | 0.04x | 0.7x | 0.03x | 1.4x |
| stats | 0.5x | 3.9x | 0.94x | 4.8x |
| startup (hello) | 0.43x | 0.26x | 0.15x |  |

What each one does:

- **fib**: recursive function calls: fib[27]
- **loop**: a counting loop with arithmetic, 3,000,000 passes
- **strings**: 200,000 pieces of text built, joined, split
- **lists**: 300,000 numbers made, sorted, read
- **maps**: 200,000 keys put in a map, then looked up
- **sieve**: primes below 1,000,000: list indexing in nested loops
- **objects**: 200,000 records (assembled values) made and read
- **jsondata**: 20,000 records to JSON text and back
- **patterns**: a pattern (regex) over 50,000 lines
- **functional**: process, keep, reduce over 300,000 numbers
- **matrix**: a 200 x 200 matrix built, multiplied, and a 200 x 200 system solved (Turtle: import linear)
- **stats**: mean, stdev, median, 90th percentile of 300,000 numbers (Turtle: import data; Python: statistics)

Versions: turtle dev; Python 3.14.5; v24.21.0; ruby 2.6.10p210 (2022-04-12 revision 67958) [universal.arm64e-darwin26]; go version go1.26.3 darwin/arm64; Darwin arm64

