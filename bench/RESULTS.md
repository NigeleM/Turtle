# Turtle benchmark results

Made by `turtle run.trt` in this folder on 2026-10-08.
Times are milliseconds: each program times its own work, best of 3 runs,
so starting the language isn't counted (startup is measured on its own,
from start to finish). Every language computes the same answer, checked.

| benchmark | Turtle ms | Python ms | Node ms | Ruby ms | Go ms |
|---|---|---|---|---|---|
| fib | 34.0 | 8.8 | 0.8 | 9.9 | 0.4 |
| loop | 205.0 | 132.5 | 20.6 | 76.0 | 1.3 |
| strings | 31.0 | 20.8 | 8.0 | 31.4 | 9.9 |
| lists | 86.0 | 63.7 | 41.5 | 29.1 | 15.0 |
| maps | 68.0 | 33.2 | 22.5 | 44.5 | 19.9 |
| sieve | 220.0 | 59.7 | 3.4 | 62.8 | 1.6 |
| objects | 30.0 | 22.5 | 10.2 | 18.1 | 3.4 |
| jsondata | 24.0 | 14.2 | 5.3 | 29.6 | 23.1 |
| patterns | 17.0 | 8.2 | 3.8 | 19.8 | 12.5 |
| functional | 20.0 | 14.5 | 3.7 | 13.1 | 0.7 |
| matrix | 11.0 | 261.4 | 17.1 | 385.8 | 7.9 |
| stats | 42.0 | 85.8 | 11.0 | 44.7 | 9.2 |
| startup (hello) | 6 | 14 | 23 | 39 |  |

How many times slower Turtle is than each (1.0x is the same speed; under
1.0x, Turtle is faster):

| benchmark | Turtle ÷ Python | Turtle ÷ Node | Turtle ÷ Ruby | Turtle ÷ Go |
|---|---|---|---|---|
| fib | 3.9x | 42.5x | 3.4x | 85.0x |
| loop | 1.5x | 10.0x | 2.7x | 157.7x |
| strings | 1.5x | 3.9x | 0.99x | 3.1x |
| lists | 1.4x | 2.1x | 3.0x | 5.7x |
| maps | 2.0x | 3.0x | 1.5x | 3.4x |
| sieve | 3.7x | 64.7x | 3.5x | 137.5x |
| objects | 1.3x | 2.9x | 1.7x | 8.8x |
| jsondata | 1.7x | 4.5x | 0.81x | 1.0x |
| patterns | 2.1x | 4.5x | 0.86x | 1.4x |
| functional | 1.4x | 5.4x | 1.5x | 28.6x |
| matrix | 0.04x | 0.64x | 0.03x | 1.4x |
| stats | 0.49x | 3.8x | 0.94x | 4.6x |
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

