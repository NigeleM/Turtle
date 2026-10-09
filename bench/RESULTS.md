# Turtle benchmark results

Made by `turtle run.turtle` in this folder on 2026-10-08.
Times are milliseconds: each program times its own work, best of 3 runs,
so starting the language isn't counted (startup is measured on its own,
from start to finish). Every language computes the same answer, checked.

| benchmark | Turtle ms | Python ms | Node ms | Ruby ms | Go ms |
|---|---|---|---|---|---|
| fib | 34.0 | 8.9 | 0.8 | 9.4 | 0.4 |
| loop | 208.0 | 136.2 | 20.5 | 76.1 | 1.2 |
| strings | 32.0 | 20.4 | 7.9 | 31.1 | 9.4 |
| lists | 85.0 | 62.4 | 42.1 | 29.1 | 15.0 |
| maps | 66.0 | 35.1 | 23.4 | 44.4 | 19.0 |
| sieve | 220.0 | 68.4 | 3.2 | 63.3 | 1.7 |
| objects | 29.0 | 21.4 | 10.5 | 18.1 | 3.4 |
| jsondata | 24.0 | 14.6 | 4.7 | 29.8 | 23.3 |
| patterns | 17.0 | 8.2 | 3.9 | 20.7 | 12.6 |
| functional | 21.0 | 14.6 | 3.6 | 13.1 | 0.7 |
| matrix | 11.0 | 260.1 | 16.0 | 384.9 | 7.9 |
| stats | 41.0 | 87.3 | 11.0 | 45.7 | 8.9 |
| startup (hello) | 6 | 15 | 22 | 40 |  |

How many times slower Turtle is than each (1.0x is the same speed; under
1.0x, Turtle is faster):

| benchmark | Turtle ÷ Python | Turtle ÷ Node | Turtle ÷ Ruby | Turtle ÷ Go |
|---|---|---|---|---|
| fib | 3.8x | 42.5x | 3.6x | 85.0x |
| loop | 1.5x | 10.1x | 2.7x | 173.3x |
| strings | 1.6x | 4.1x | 1.0x | 3.4x |
| lists | 1.4x | 2.0x | 2.9x | 5.7x |
| maps | 1.9x | 2.8x | 1.5x | 3.5x |
| sieve | 3.2x | 68.8x | 3.5x | 129.4x |
| objects | 1.4x | 2.8x | 1.6x | 8.5x |
| jsondata | 1.6x | 5.1x | 0.81x | 1.0x |
| patterns | 2.1x | 4.4x | 0.82x | 1.3x |
| functional | 1.4x | 5.8x | 1.6x | 30.0x |
| matrix | 0.04x | 0.69x | 0.03x | 1.4x |
| stats | 0.47x | 3.7x | 0.9x | 4.6x |
| startup (hello) | 0.4x | 0.27x | 0.15x |  |

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

