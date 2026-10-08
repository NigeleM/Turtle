# Turtle benchmark results

Made by `turtle run.trt` in this folder on 2026-10-08.
Times are milliseconds: each program times its own work, best of 3 runs,
so starting the language isn't counted (startup is measured on its own,
from start to finish). Every language computes the same answer, checked.

| benchmark | Turtle ms | Python ms | Node ms | Ruby ms | Go ms |
|---|---|---|---|---|---|
| fib | 47.0 | 8.8 | 0.8 | 9.8 | 0.4 |
| loop | 240.0 | 144.8 | 20.3 | 75.8 | 1.2 |
| strings | 43.0 | 23.3 | 8.0 | 31.2 | 9.5 |
| lists | 97.0 | 63.8 | 42.1 | 29.6 | 14.8 |
| maps | 86.0 | 33.6 | 23.3 | 44.2 | 19.8 |
| sieve | 258.0 | 61.3 | 3.0 | 63.3 | 1.6 |
| objects | 40.0 | 24.0 | 10.4 | 18.4 | 3.3 |
| jsondata | 58.0 | 14.6 | 5.4 | 29.6 | 23.2 |
| patterns | 21.0 | 8.4 | 3.8 | 20.0 | 12.5 |
| functional | 41.0 | 15.6 | 3.9 | 13.3 | 0.7 |
| matrix | 13.0 | 256.4 | 16.8 | 386.3 | 8.0 |
| stats | 45.0 | 87.9 | 11.0 | 45.3 | 9.0 |
| startup (hello) | 6 | 14 | 23 | 41 |  |

How many times slower Turtle is than each (1.0x is the same speed; under
1.0x, Turtle is faster):

| benchmark | Turtle ÷ Python | Turtle ÷ Node | Turtle ÷ Ruby | Turtle ÷ Go |
|---|---|---|---|---|
| fib | 5.3x | 58.8x | 4.8x | 117.5x |
| loop | 1.7x | 11.8x | 3.2x | 200.0x |
| strings | 1.8x | 5.4x | 1.4x | 4.5x |
| lists | 1.5x | 2.3x | 3.3x | 6.6x |
| maps | 2.6x | 3.7x | 1.9x | 4.3x |
| sieve | 4.2x | 86.0x | 4.1x | 161.2x |
| objects | 1.7x | 3.8x | 2.2x | 12.1x |
| jsondata | 4.0x | 10.7x | 2.0x | 2.5x |
| patterns | 2.5x | 5.5x | 1.1x | 1.7x |
| functional | 2.6x | 10.5x | 3.1x | 58.6x |
| matrix | 0.05x | 0.77x | 0.03x | 1.6x |
| stats | 0.51x | 4.1x | 0.99x | 5.0x |
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

