# Turtle benchmark results

Made by `turtle run.trt` in this folder on 2026-10-07.
Times are milliseconds: each program times its own work, best of 3 runs,
so starting the language isn't counted (startup is measured on its own,
from start to finish). Every language computes the same answer, checked.

| benchmark | Turtle ms | Python ms | Node ms | Ruby ms | Go ms |
|---|---|---|---|---|---|
| fib | 104.0 | 8.5 | 1.0 | 9.8 | 0.6 |
| loop | 445.0 | 133.5 | 20.7 | 75.9 | 1.7 |
| strings | 75.0 | 21.7 | 8.4 | 31.4 | 10.4 |
| lists | 273.0 | 62.4 | 42.1 | 28.8 | 16.5 |
| maps | 119.0 | 33.1 | 21.9 | 43.5 | 19.7 |
| sieve | 587.0 | 60.4 | 3.1 | 62.8 | 2.6 |
| objects | 82.0 | 23.8 | 10.5 | 18.1 | 4.8 |
| jsondata | 64.0 | 14.7 | 4.8 | 28.8 | 22.9 |
| patterns | 31.0 | 8.4 | 3.9 | 20.3 | 13.9 |
| functional | 84.0 | 15.7 | 3.9 | 13.6 | 0.9 |
| startup (hello) | 6 | 14 | 22 | 39 |  |

How many times slower Turtle is than each (1.0x is the same speed):

| benchmark | Turtle ÷ Python | Turtle ÷ Node | Turtle ÷ Ruby | Turtle ÷ Go |
|---|---|---|---|---|
| fib | 12.2x | 104.0x | 10.6x | 173.3x |
| loop | 3.3x | 21.5x | 5.9x | 261.8x |
| strings | 3.5x | 8.9x | 2.4x | 7.2x |
| lists | 4.4x | 6.5x | 9.5x | 16.5x |
| maps | 3.6x | 5.4x | 2.7x | 6.0x |
| sieve | 9.7x | 189.4x | 9.3x | 225.8x |
| objects | 3.4x | 7.8x | 4.5x | 17.1x |
| jsondata | 4.4x | 13.3x | 2.2x | 2.8x |
| patterns | 3.7x | 7.9x | 1.5x | 2.2x |
| functional | 5.4x | 21.5x | 6.2x | 93.3x |
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

