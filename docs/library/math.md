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
| `sin`, `cos`, `tan` | — | the sine, cosine or tangent of an angle in radians (a full turn is `2 * pi`), as a float |
| `asin`, `acos`, `atan` | — | the angle in radians whose sine, cosine or tangent is the number, as a float (`1 at asin` is `pi / 2`). `asin` and `acos` need a number from -1 to 1; anything else is a math error |
| `atan2` | x | `y at atan2[x]` is the angle in radians of the point (x, y), from `-pi` to `pi` |
| `exp` | — | `e` raised to the number, as a float. A result too big for a float is a math error |
| `log` | [base] | the natural logarithm (base `e`), or the logarithm in `base` (`100 at log[10]` is `2.0`). Zero or a negative number is a math error |
| `is_close` | other [, tolerance] | `true` or `false`: whether the two numbers are close, by the rule of `is close to` in a check. The optional tolerance is the most they may differ (0 or more): `(0.1 + 0.2) at is_close[0.3]` is `true` |

```
import math

r is 16 at sqrt .        // 4.0 (sqrt always give a float)
p is 2 at pow 10 .       // 1024
n is 10 at random .      // some integer in [0, 10)
```

The constants `pi` and `e` come with `import math` too, as floats:
`2 * pi * r` and `e at pow[x]`. Write `math pi` if you want the qualified
name. A variable of your own named `e` or `pi` takes precedence.

`%` (modulo) is a core operator, not a method, so it needs no import — see
[`reference.md`](../reference.md#expressions).

---

Copyright 2017-2026 Nigele McCoy. Licensed under the
[Apache License 2.0](../../LICENSE); see [NOTICE](../../NOTICE).

<!-- generated: turtle doc `math` (regenerate with the script; do not edit by hand) -->
## Function reference

The entries below are copied from the binary's own documentation (`turtle doc math`), so they match the version you run.

```text
import math
```

### `Number methods. Called on a number with "at":  r is 16 at sqrt .`

```text
Number methods. Called on a number with "at":  r is 16 at sqrt .
```

### `number at sqrt`

```text
number at sqrt        (import math)
  The square root.
  Gives back: a float (16 at sqrt is 4.0). A negative number is a math error.
  Example:
    r is 16 at sqrt .
```

### `number at sin`

```text
number at sin        (import math)
  The sine of an angle in radians (a full turn is 2 * pi). Gives back: a float.
  Example:
    s is (math pi / 2) at sin .            // 1.0
```

### `number at cos`

```text
number at cos        (import math)
  The cosine of an angle in radians. Gives back: a float.
  Example:
    c is 0 at cos .                        // 1.0
```

### `number at tan`

```text
number at tan        (import math)
  The tangent of an angle in radians. Gives back: a float.
  Example:
    t is 0 at tan .                        // 0.0
```

### `number at asin`

```text
number at asin        (import math)
  The angle, in radians, whose sine is the number: from -pi/2 to pi/2. The
  number must be from -1 to 1; anything else is a math error.
  Example:
    a is 1 at asin .                       // 1.5707963267948966 (pi / 2)
```

### `number at acos`

```text
number at acos        (import math)
  The angle, in radians, whose cosine is the number: from 0 to pi. The
  number must be from -1 to 1; anything else is a math error.
  Example:
    a is 0 at acos .                       // 1.5707963267948966 (pi / 2)
```

### `number at atan`

```text
number at atan        (import math)
  The angle, in radians, whose tangent is the number: from -pi/2 to pi/2.
  Example:
    a is 1 at atan .                       // 0.7853981633974483 (pi / 4)
```

### `number at atan2 x`

```text
number at atan2 x        (import math)
  The angle, in radians, of the point (x, y), where y is the number it's
  called on: from -pi to pi, right in every quadrant.
  x          the point's x.
  Example:
    a is 1 at atan2[-1] .                  // 2.356194490192345 (3 pi / 4)
```

### `number at exp`

```text
number at exp        (import math)
  e raised to the number: 0 at exp is 1.0. A result too big for a float is
  a math error.
  Example:
    g is 2 at exp .                        // 7.38905609893065
```

### `number at log`

```text
number at log        (import math)
  The natural logarithm (base e) of a number above 0. With a base, the
  logarithm in that base: 1000 at log[10] is 3.0. Zero or a negative number
  is a math error.
  base       optional: a base above 0 and not 1.
  Example:
    n is 1 at exp at log .                 // 1.0
    d is 1000 at log[10] .                 // 3.0
```

### `number at is_close`

```text
number at is_close        (import math)
  Whether two numbers are close, by the same rule as "check ... is close to":
  within 1e-9 of the larger one, or 1e-12 of each other, whichever is more.
  tolerance  optional: the most they may differ (0 or more), as "within".
  Gives back: true or false.
  Example:
    ok is (0.1 + 0.2) at is_close[0.3] .       // true
    ok is (10 / 3) at is_close[3.33, 0.01] .   // true
```

### `number at abs`

```text
number at abs        (import math)
  The number without its sign: -7 becomes 7.
  Gives back: the same kind of number (integer or float).
  Example:
    a is -7 at abs .
```

### `number at round places`

```text
number at round places        (import math)
  The nearest whole number; a half rounds away from zero (4.5 becomes 5).
  places     optional: round[2] keeps 2 places and gives a float
             (3.14159 -> 3.14, 0.6000000000000001 -> 0.6); round[-2]
             rounds to hundreds (1234 -> 1200). -15 to 15.
  Example:
    r is 4.5 at round .
    prices = list [10, 20]
    import data
    prices = list [10, 20]
    import data
    prices = list [10, 20]
    c is prices process p give p * 1.08 at round[2] .
    show c .
```

### `number at floor`

```text
number at floor        (import math)
  The whole number at or below it (4.7 becomes 4, -4.2 becomes -5).
  Example:
    f is 4.7 at floor .
```

### `number at ceil`

```text
number at ceil        (import math)
  The whole number at or above it (4.1 becomes 5).
  Example:
    c is 4.1 at ceil .
```

### `number at pow exponent`

```text
number at pow exponent        (import math)
  The number raised to a power.
  exponent   the power, a number
  Gives back: an integer when both are whole and the exponent isn't
  negative (2 at pow 10 is 1024), else a float.
  Example:
    p is 2 at pow 10 .
```

### `number at random`

```text
number at random        (import math)
  A random whole number from 0 up to (not including) the number.
  The number must be a positive integer.
  Example:
    dice is 6 at random .
```

<!-- end generated -->
