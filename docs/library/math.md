# `math` — number methods

`import math`

Number methods: square roots, trigonometry, logarithms, rounding and powers. Each is called on a number with `at`.


## Methods

`number at sqrt`, `number at sin`, `number at cos`, `number at tan`, `number at asin`, `number at acos`, `number at atan`, `number at atan2 x`, `number at exp`, `number at log`, `number at is_close`, `number at abs`, `number at round places`, `number at floor`, `number at ceil`, `number at pow exponent`, `number at random`.

### `number at sqrt`

The square root.

**Example**

```turtle
r is 16 at sqrt .
```

**Output**

```text
(prints nothing)
```

### `number at sin`

The sine of an angle in radians (a full turn is 2 * pi).

**Example**

```turtle
s is (math pi / 2) at sin .
```

**Output**

```text
(prints nothing)
```

### `number at cos`

The cosine of an angle in radians.

**Example**

```turtle
c is 0 at cos .
```

**Output**

```text
(prints nothing)
```

### `number at tan`

The tangent of an angle in radians.

**Example**

```turtle
t is 0 at tan .
```

**Output**

```text
(prints nothing)
```

### `number at asin`

The angle, in radians, whose sine is the number: from -pi/2 to pi/2.

**Example**

```turtle
a is 1 at asin .
```

**Output**

```text
(prints nothing)
```

### `number at acos`

The angle, in radians, whose cosine is the number: from 0 to pi.

**Example**

```turtle
a is 0 at acos .
```

**Output**

```text
(prints nothing)
```

### `number at atan`

The angle, in radians, whose tangent is the number: from -pi/2 to pi/2.

**Example**

```turtle
a is 1 at atan .
```

**Output**

```text
(prints nothing)
```

### `number at atan2 x`

The angle, in radians, of the point (x, y), where y is the number it's called on: from -pi to pi, right in every quadrant. x          the point's x.

**Example**

```turtle
a is 1 at atan2[-1] .
```

**Output**

```text
(prints nothing)
```

### `number at exp`

e raised to the number: 0 at exp is 1.0.

**Example**

```turtle
g is 2 at exp .
```

**Output**

```text
(prints nothing)
```

### `number at log`

The natural logarithm (base e) of a number above 0.

**Example**

```turtle
n is 1 at exp at log .
d is 1000 at log[10] .
```

**Output**

```text
(prints nothing)
```

### `number at is_close`

Whether two numbers are close, by the same rule as "check ... is close to": within 1e-9 of the larger one, or 1e-12 of each other, whichever is more. tolerance  optional: the most they may differ (0 or more), as "within".

**Example**

```turtle
ok is (0.1 + 0.2) at is_close[0.3] .
ok is (10 / 3) at is_close[3.33, 0.01] .
```

**Output**

```text
(prints nothing)
```

### `number at abs`

The number without its sign: -7 becomes 7.

**Example**

```turtle
a is -7 at abs .
```

**Output**

```text
(prints nothing)
```

### `number at round places`

The nearest whole number; a half rounds away from zero (4.5 becomes 5). places     optional: round[2] keeps 2 places and gives a float (3.14159 -> 3.14, 0.6000000000000001 -> 0.6); round[-2] rounds to hundreds (1234 -> 1200). -15 to 15.

**Example**

```turtle
r is 4.5 at round .
prices = list [10, 20]
import data
prices = list [10, 20]
import data
prices = list [10, 20]
c is prices process p give p * 1.08 at round[2] .
show c .
```

**Output**

```text
[ 10.8, 21.6 ]
```

### `number at floor`

The whole number at or below it (4.7 becomes 4, -4.2 becomes -5).

**Example**

```turtle
f is 4.7 at floor .
```

**Output**

```text
(prints nothing)
```

### `number at ceil`

The whole number at or above it (4.1 becomes 5).

**Example**

```turtle
c is 4.1 at ceil .
```

**Output**

```text
(prints nothing)
```

### `number at pow exponent`

The number raised to a power. exponent   the power, a number Gives back: an integer when both are whole and the exponent isn't negative (2 at pow 10 is 1024), else a float.

**Example**

```turtle
p is 2 at pow 10 .
```

**Output**

```text
(prints nothing)
```

### `number at random`

A random whole number from 0 up to (not including) the number.

**Example**

```turtle
dice is 6 at random .
```

**Output**

```text
(prints nothing)
```
