# Turtle benchmark results

Made by `turtle run.trt` in this folder on 2026-10-07.
Times are milliseconds: each program times its own work, best of 3 runs,
so starting the language isn't counted (startup is measured on its own,
from start to finish). Every language computes the same answer, checked.

| benchmark | Turtle ms | Python ms | Node ms | Ruby ms | Go ms |
|---|---|---|---|---|---|
| fib | 48.0 | 8.7 | 0.8 | 9.8 | 0.4 |
| loop | 240.0 | 132.8 | 20.7 | 75.8 | 1.2 |
| strings | 43.0 | 20.8 | 7.9 | 31.0 | 9.5 |
| lists | 96.0 | 63.2 | 42.0 | 29.0 | 15.1 |
| maps | 85.0 | 33.3 | 22.7 | 43.2 | 19.8 |
| sieve | 251.0 | 59.4 | 3.1 | 63.1 | 1.6 |
| objects | 40.0 | 23.2 | 10.6 | 18.5 | 3.5 |
| jsondata | 58.0 | 14.7 | 5.3 | 29.1 | 23.8 |
| patterns | 21.0 | 8.3 | 3.8 | 20.0 | 12.3 |
| functional | 41.0 | 15.9 | 3.9 | 13.3 | 0.7 |
| startup (hello) | 6 | 14 | 22 | 39 |  |

How many times slower Turtle is than each (1.0x is the same speed):

| benchmark | Turtle ÷ Python | Turtle ÷ Node | Turtle ÷ Ruby | Turtle ÷ Go |
|---|---|---|---|---|
| fib | 5.5x | 60.0x | 4.9x | 120.0x |
| loop | 1.8x | 11.6x | 3.2x | 200.0x |
| strings | 2.1x | 5.4x | 1.4x | 4.5x |
| lists | 1.5x | 2.3x | 3.3x | 6.4x |
| maps | 2.6x | 3.7x | 2.0x | 4.3x |
| sieve | 4.2x | 81.0x | 4.0x | 156.9x |
| objects | 1.7x | 3.8x | 2.2x | 11.4x |
| jsondata | 3.9x | 10.9x | 2.0x | 2.4x |
| patterns | 2.5x | 5.5x | 1.1x | 1.7x |
| functional | 2.6x | 10.5x | 3.1x | 58.6x |
| startup (hello) | 0.4x | 0.3x | 0.2x |  |

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

Versions: turtle dev; Python 3.14.5; v24.21.0; ruby 2.6.10p210 (2022-04-12 revision 67958) [universal.arm64e-darwin26]; go version go1.26.3 darwin/arm64; Darwin arm64

