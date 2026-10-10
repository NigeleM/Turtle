# `math` — number methods

[Library index](index.md) · `import math`

Square roots, rounding, powers and random whole numbers, called on a number with `at`.


Every method here except `fixed` and `commas` requires `import math`
first (see [`reference.md`](../reference.md#modules)) — calling one before
that is a fatal error naming exactly which import is missing. `fixed` and
`commas` write a number as text and need no import:

| Method | Args | Returns |
|---|---|---|
| `fixed` | places | text with exactly that many digits after the point (0 to 20), rounded: `3.5 at fixed[2]` is `"3.50"` |
| `commas` | optional places | text with commas between thousands: `1234567 at commas` is `"1,234,567"`; a fraction is kept (`-1234.5` gives `"-1,234.5"`). With places, exactly that many decimals, as `fixed`: `1250.5 at commas[2]` is `"1,250.50"` |

```
price = 1234.5
show "$", price at fixed[2] .    // $1234.50
show 1234567 at commas .         // 1,234,567
show 1250.5 at commas[2] .       // 1,250.50
```

The math methods:

| Method | Args | Returns |
|---|---|---|
| `sqrt` | — | square root, as a `float` (fatal on a negative receiver) |
| `abs` | — | absolute value, same type as the receiver |
| `round` | [places] | nearest integer (half rounds away from zero). `round[2]` keeps 2 places and gives a `float` (`3.14159` → `3.14`, `0.6000000000000001` → `0.6`); `round[-2]` rounds to hundreds (`1234` → `1200`). For a fixed look with trailing zeros (`"0.60"`), use `fixed`, which gives text |
| `floor` | — | next integer toward negative infinity |
| `ceil` | — | next integer toward positive infinity |
| `pow` | exponent | receiver raised to exponent: an `integer` when both are whole numbers and the exponent isn't negative, otherwise a `float` |
| `random` | — | a random integer in `[0, receiver)`; the receiver is the exclusive upper bound and must be a positive integer |

```
import math

r is 16 at sqrt .        // 4.0 (sqrt always give a float)
p is 2 at pow 10 .       // 1024
n is 10 at random .      // some integer in [0, 10)
```

`%` (modulo) is a core operator, not a method, so it needs no import — see
[`reference.md`](../reference.md#expressions).
