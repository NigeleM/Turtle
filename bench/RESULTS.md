# Turtle benchmark results

Made by `turtle run.trt` in this folder on 2026-10-08.
Times are milliseconds: each program times its own work, best of 3 runs,
so starting the language isn't counted (startup is measured on its own,
from start to finish). Every language computes the same answer, checked.

| benchmark | Turtle ms | Python ms | Node ms | Ruby ms | Go ms |
|---|---|---|---|---|---|
| fib | 44.0 | 8.6 | 0.9 | 9.3 | 0.4 |
| loop | 209.0 | 135.3 | 20.4 | 75.6 | 1.2 |
| strings | 39.0 | 22.1 | 7.9 | 31.4 | 9.7 |
| lists | 85.0 | 64.9 | 42.5 | 28.8 | 15.1 |
| maps | 75.0 | 33.6 | 22.4 | 44.0 | 19.8 |
| sieve | 222.0 | 58.7 | 3.3 | 63.0 | 1.6 |
| objects | 37.0 | 22.3 | 10.7 | 18.1 | 3.3 |
| jsondata | 58.0 | 14.7 | 4.6 | 29.4 | 23.1 |
| patterns | 19.0 | 8.3 | 3.9 | 20.2 | 12.5 |
| functional | 34.0 | 15.7 | 3.9 | 13.3 | 0.7 |
| matrix | 11.0 | 265.2 | 14.9 | 385.0 | 7.9 |
| stats | 42.0 | 87.0 | 11.0 | 45.0 | 8.8 |
| startup (hello) | 6 | 13 | 22 | 39 |  |

How many times slower Turtle is than each (1.0x is the same speed; under
1.0x, Turtle is faster):

| benchmark | Turtle ÷ Python | Turtle ÷ Node | Turtle ÷ Ruby | Turtle ÷ Go |
|---|---|---|---|---|
| fib | 5.1x | 48.9x | 4.7x | 110.0x |
| loop | 1.5x | 10.2x | 2.8x | 174.2x |
| strings | 1.8x | 4.9x | 1.2x | 4.0x |
| lists | 1.3x | 2.0x | 3.0x | 5.6x |
| maps | 2.2x | 3.3x | 1.7x | 3.8x |
| sieve | 3.8x | 67.3x | 3.5x | 138.8x |
| objects | 1.7x | 3.5x | 2.0x | 11.2x |
| jsondata | 3.9x | 12.6x | 2.0x | 2.5x |
| patterns | 2.3x | 4.9x | 0.94x | 1.5x |
| functional | 2.2x | 8.7x | 2.6x | 48.6x |
| matrix | 0.04x | 0.74x | 0.03x | 1.4x |
| stats | 0.48x | 3.8x | 0.93x | 4.8x |
| startup (hello) | 0.46x | 0.27x | 0.15x |  |

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

