# `linear`: matrices and linear algebra

[Library index](index.md) · `import linear`

Matrices, their arithmetic, and solving, inverting and decomposing them.
The work is done in compiled code, so it's fast: a 200 × 200 product takes a few
milliseconds.

```
import linear
a = matrix [2, 1; 1, 3]
x = solve[a, list [3, 5]]
show x .
```

```
[ 0.8, 1.4 ]
```

## Writing a matrix

Rows end at `;` or at the end of a line. Every row needs the same count
of numbers, and any expression can be a number.

```
import linear
a = matrix [1, 2; 3, 4]
b = matrix [
    1, 2, 3
    4, 5, 6
]
show b .
```

```
[ 1  2  3 ]
[ 4  5  6 ]
```

- `matrix` is a word only in a file that imports `linear`.
- A matrix of whole numbers stays whole. Anything that can make a
  fraction (`/`, `inverse`, `solve`, a float in it) gives a float matrix.
- `show` lines each column up on its decimal points. Joined into text
  (`"m = {m}"`), its rows stay lined up under the first.
- `change rows to matrix` makes one from a list of lists (or a table's
  rows); `change m to list` turns it back.
- A matrix is shared, like a list: after `b = a`, a `put` on `b` changes
  `a` too. `copy[a]` (`import data`) makes a separate one.

## Operators

| Expression | Gives |
|---|---|
| `a + b`, `a - b` | element by element; the sizes must match |
| `a + 1`, `10 - a` | the number on every element |
| `a * b` | the matrix product: `a` needs as many columns as `b` has rows |
| `a * v` (`v` a list) | a matrix times a vector: a list |
| `a * 2`, `a / 2` | every number scaled |
| `-a` | every number negated |
| `a == b`, `a != b` | the same size and the same numbers |

```
import linear
a = matrix [1, 2; 3, 4]
b = matrix [5, 6; 7, 8]
show a * b .
show a + 1 .
```

```
[ 19  22 ]
[ 43  50 ]
[ 2  3 ]
[ 4  5 ]
```

## Methods

Called with `at`: `m at rows`.

| Method | Gives |
|---|---|
| `m at rows`, `m at columns` | how many rows or columns |
| `m at shape` | `[rows, columns]` |
| `m at get[r, c]` | the number at row `r`, column `c`, from 0 |
| `m at put[value, r, c]` | puts a number there, and gives the matrix |
| `m at row[r]`, `m at column[c]` | a row or column, as a list |
| `m at flatten` | the numbers in one row |
| `m at reshape[r, c]` | the numbers as an r × c matrix |
| `m at isempty`, `m at tostring` | as for lists |

```
import linear
m = matrix [1, 2, 3; 4, 5, 6]
show m at get[1, 2] .
m at put[0, 0, 0]
show m .
```

```
6
[ 0  2  3 ]
[ 4  5  6 ]
```

## Making a matrix

### `identity[n]`

The n × n identity matrix: 1s on the diagonal, 0s elsewhere.

- `n`: the size, a whole number, 0 or more

**Gives:** a matrix.

```
import linear
show identity[3] .
```

```
[ 1  0  0 ]
[ 0  1  0 ]
[ 0  0  1 ]
```

### `zeros[rows, columns]`

A matrix full of 0s.

- `rows`: how many rows
- `columns`: how many columns; leave it out for a square matrix

**Gives:** a matrix.

```
import linear
show zeros[2, 3] .
show zeros[2] .
```

```
[ 0  0  0 ]
[ 0  0  0 ]
[ 0  0 ]
[ 0  0 ]
```

### `ones[rows, columns]`

A matrix full of 1s.

- `rows`: how many rows
- `columns`: how many columns; leave it out for a square matrix

**Gives:** a matrix.

```
import linear
show ones[2, 3] .
```

```
[ 1  1  1 ]
[ 1  1  1 ]
```

### `diagonal[x]`

Goes both ways: a list becomes a square matrix with those numbers on its diagonal, and a matrix gives back the list of its diagonal.

- `x`: a list of numbers, or a matrix

**Gives:** a matrix (from a list) or a list (from a matrix).

```
import linear
show diagonal[list [1, 2, 3]] .
show diagonal[matrix [1, 2; 3, 4]] .
```

```
[ 1  0  0 ]
[ 0  2  0 ]
[ 0  0  3 ]
[ 1, 4 ]
```

## Shape and parts

### `shape[m]`

The size of a matrix.

- `m`: a matrix

**Gives:** a list: [rows, columns].

```
import linear
m = matrix [1, 2, 3; 4, 5, 6]
show shape[m] .
rows = m at rows
cols = m at columns
show rows, " rows, ", cols, " columns" .
```

```
[ 2, 3 ]
2 rows, 3 columns
```

Also as methods: `m at shape`, `m at rows`, `m at columns`. Use these to pull the sizes out for your own loops.

### `row[m, r]`

One row of a matrix.

- `m`: a matrix
- `r`: the row number, from 0

**Gives:** a list.

```
import linear
m = matrix [1, 2, 3; 4, 5, 6]
show row[m, 1] .
show m at row[0] .
```

```
[ 4, 5, 6 ]
[ 1, 2, 3 ]
```

**Errors:** An `index` error if `r` is outside the matrix.

### `column[m, c]`

One column of a matrix.

- `m`: a matrix
- `c`: the column number, from 0

**Gives:** a list.

```
import linear
m = matrix [1, 2, 3; 4, 5, 6]
show column[m, 2] .
```

```
[ 3, 6 ]
```

**Errors:** An `index` error if `c` is outside the matrix.

### `flatten[m]`

All the numbers of a matrix in one row, read row by row. The matrix you pass in is unchanged.

- `m`: a matrix

**Gives:** a new 1 × n matrix.

```
import linear
m = matrix [1, 2, 3; 4, 5, 6]
show flatten[m] .
show m flatten .
```

```
[ 1  2  3  4  5  6 ]
[ 1  2  3  4  5  6 ]
```

Also as a method, `m at flatten`, and a sentence, `m flatten`.

### `reshape[m, rows, columns]`

The same numbers in the same order, laid out as a different shape. The matrix you pass in is unchanged.

- `m`: a matrix
- `rows`: the new number of rows
- `columns`: the new number of columns

**Gives:** a new rows × columns matrix.

```
import linear
m = matrix [1, 2, 3; 4, 5, 6]
show reshape[m, 3, 2] .
show m reshape 1, 6 .
```

```
[ 1  2 ]
[ 3  4 ]
[ 5  6 ]
[ 1  2  3  4  5  6 ]
```

Also as a method, `m at reshape[3, 2]`, and a sentence, `m reshape 3, 2`.

**Errors:** A `linear` error if `rows * columns` isn't how many numbers `m` has: "can't reshape a 2 x 3 matrix (6 numbers) into 4 x 2 (8 numbers)".

### `transpose[m]`

Rows become columns, and columns rows.

- `m`: a matrix

**Gives:** a new matrix.

```
import linear
m = matrix [1, 2, 3; 4, 5, 6]
show transpose[m] .
```

```
[ 1  4 ]
[ 2  5 ]
[ 3  6 ]
```

Also a sentence: `m transpose`.

## Arithmetic

### `multiply_each[a, b]`

Multiplies element by element: each number times the one in the same place. (`a * b` is the matrix product.)

- `a, b`: two matrices of the same size

**Gives:** a new matrix.

```
import linear
a = matrix [1, 2; 3, 4]
b = matrix [5, 6; 7, 8]
show multiply_each[a, b] .
```

```
[  5  12 ]
[ 21  32 ]
```

**Errors:** A `linear` error if the sizes differ.

### `power[m, k]`

Multiplies a square matrix by itself k times.

- `m`: a square matrix
- `k`: a whole number; 0 gives the identity, a negative k a power of the inverse

**Gives:** a matrix.

```
import linear
fib = matrix [1, 1; 1, 0]
show power[fib, 10] .
```

```
[ 89  55 ]
[ 55  34 ]
```

### `trace[m]`

The sum of the numbers on the diagonal.

- `m`: a square matrix

**Gives:** a number.

```
import linear
show trace[matrix [1, 2; 3, 4]] .
```

```
5
```

### `determinant[m]`

The determinant of a square matrix: 0 when it has no inverse.

- `m`: a square matrix

**Gives:** an integer for a matrix of whole numbers, otherwise a float.

```
import linear
show determinant[matrix [1, 2; 3, 4]] .
show determinant[matrix [1, 2; 2, 4]] .
```

```
-2
0
```

### `inverse[m]`

The matrix that multiplies m to the identity.

- `m`: a square matrix

**Gives:** a float matrix.

```
import linear
m = matrix [4, 7; 2, 6]
show inverse[m] .
show m * inverse[m] .
```

```
[  0.6  -0.7 ]
[ -0.2   0.4 ]
[ 1.0  0.0 ]
[ 0.0  1.0 ]
```

**Errors:** A `linear` error if `m` is singular (its determinant is 0).

### `rank[m]`

How many rows are independent: not a mix of the others.

- `m`: a matrix

**Gives:** an integer.

```
import linear
show rank[matrix [1, 2; 2, 4]] .
show rank[identity[3]] .
```

```
1
3
```

## Solving equations

### `solve[a, b]`

Finds x where a * x == b.

- `a`: a square matrix
- `b`: a list (one answer) or a matrix (one answer per column)

**Gives:** a list, or a matrix when b is one.

```
import linear
// 2x + y = 3 and x + 3y = 5
a = matrix [2, 1; 1, 3]
x = solve[a, list [3, 5]]
show x .
show a * x .
```

```
[ 0.8, 1.4 ]
[ 3.0, 5.0 ]
```

Also a sentence: `a solve list [3, 5]`.

**Errors:** A `linear` error if `a` is singular or the sizes don't fit.

### `least_squares[a, b]`

The x that brings a * x closest to b, when there are more equations than unknowns: a line or curve of best fit.

- `a`: a matrix with at least as many rows as columns
- `b`: a list, one number per row of a

**Gives:** a list.

```
import linear
// A line y = b0 + b1 * x through (1, 1), (2, 2), (3, 2)
points = matrix [1, 1; 1, 2; 1, 3]
fit = least_squares[points, list [1, 2, 2]]
show fit .
```

```
[ 0.666666666666667, 0.5 ]
```

**Errors:** A `linear` error if the columns depend on each other.

## Vectors (lists of numbers)

### `dot[u, v]`

The dot product: each pair multiplied, then added up.

- `u, v`: two lists of numbers, the same length

**Gives:** a number.

```
import linear
show dot[list [1, 2, 3], list [4, 5, 6]] .
```

```
32
```

### `cross[u, v]`

The cross product of two 3-D vectors.

- `u, v`: two lists of 3 numbers

**Gives:** a list of 3 numbers.

```
import linear
show cross[list [1, 0, 0], list [0, 1, 0]] .
```

```
[ 0, 0, 1 ]
```

### `norm[x]`

The length of a vector. For a matrix, every number counts (the Frobenius norm).

- `x`: a list of numbers, or a matrix

**Gives:** a float.

```
import linear
show norm[list [3, 4]] .
```

```
5.0
```

### `all_close[a, b]`

Whether two matrices of the same size, or two lists of the same length, have
every pair of numbers close. The rule is the one `is close to` uses in a check
(within 1e-9 of the larger number, or 1e-12 of each other). Use it for results
of float arithmetic, where `==` is exact and a leftover like `1e-16` breaks it.
Different sizes or lengths give `false`.

- `a`, `b`: two matrices, or two lists of numbers
- `tolerance`: optional, a third argument: the most each pair may differ (0 or more)

**Gives:** `true` or `false`.

```
import linear
m = matrix [4, 7; 2, 6]
show all_close[m * inverse[m], identity[2]] .
show all_close[list [1, 2], list [1.01, 2], 0.1] .
```

```
true
true
```

### `unit[v]`

The vector scaled to length 1, in the same direction.

- `v`: a list of numbers, not all 0

**Gives:** a list.

```
import linear
show unit[list [3, 4]] .
```

```
[ 0.6, 0.8 ]
```

## Decompositions

### `lu[m]`

Splits a square matrix into lower and upper triangles.

- `m`: a square matrix

**Gives:** a map: "l" (lower, 1s on the diagonal), "u" (upper) and "p" (row order), where p * m == l * u.

```
import linear
m = matrix [2, 1; 4, 3]
parts = lu[m]
show parts at get["u"] .
show parts at get["p"] * m == parts at get["l"] * parts at get["u"] .
```

```
[ 4.0   3.0 ]
[ 0.0  -0.5 ]
true
```

### `qr[m]`

Splits a matrix into orthonormal columns and an upper triangle.

- `m`: a matrix with at least as many rows as columns

**Gives:** a map: "q" and "r", where q * r == m.

```
import linear
m = matrix [12, -51, 4; 6, 167, -68; -4, 24, -41]
parts = qr[m]
show parts at get["r"] .
```

```
[ 14.0   21.0  -14.0 ]
[  0.0  175.0  -70.0 ]
[  0.0    0.0   35.0 ]
```

### `eigen[m]`

The eigenvalues and eigenvectors of a symmetric matrix.

- `m`: a symmetric square matrix

**Gives:** a map: "values" (a list, largest first) and "vectors" (a matrix, one vector per column).

```
import linear
e = eigen[matrix [2, 1; 1, 2]]
show e at get["values"] .
```

```
[ 3.0, 1.0 ]
```

**Errors:** A `linear` error if `m` isn't symmetric.

### `svd[m]`

The singular value decomposition, for any shape.

- `m`: a matrix

**Gives:** a map: "u", "s" (a list, largest first) and "v", where m == u * diagonal[s] * transpose[v].

```
import linear
parts = svd[matrix [3, 0; 0, 4]]
show parts at get["s"] .
```

```
[ 4.0, 3.0 ]
```

## Errors

Mistakes with matrices are errors of kind `linear`, and the message names
the sizes: "can't multiply a 2 x 3 matrix by a 2 x 3 matrix". Catch them
with `safe ... handle [linear] e .`.

A worked example that uses most of the library:
[`testdata/linear/housing.turtle`](../../testdata/linear/housing.turtle).

---

Copyright 2017-2026 Nigele McCoy. Licensed under the
[Apache License 2.0](../../LICENSE); see [NOTICE](../../NOTICE).

<!-- generated: turtle doc `linear` (regenerate with the script; do not edit by hand) -->
## Function reference

The entries below are copied from the binary's own documentation (`turtle doc linear`), so they match the version you run.

```text
import linear
```

### `Matrices and vectors. After import linear, matrix [...] makes a`

```text
Matrices and vectors. After import linear, matrix [...] makes a
matrix; its rows end at ; or at the end of a line:
    a = matrix [1, 2; 3, 4]
    b = matrix [
        1, 2, 3
        4, 5, 6
    ]
+ and - work on two matrices of the same size, or a matrix and a number
(on every element: a + 1), * is the matrix product (a matrix times a list
gives a list), and * or / by a number scales every element. Vectors are plain lists of numbers. A matrix of whole numbers
shows and gives back integers; anything that can make a fraction gives
floats. Methods: m at rows, columns, shape, get[r, c], put[value, r, c],
row[r], column[c], flatten, reshape[r, c]. change rows to matrix makes one from a list of lists
(or table_read's rows); change m to list turns it back.
```

### `identity[n]`

```text
identity[n]        (import linear)
  The n x n identity matrix: 1s on the diagonal, 0s elsewhere.
  Example:
    i = identity[3]
```

### `zeros[rows, columns]`

```text
zeros[rows, columns]        (import linear)
  A matrix of 0s; zeros[n] is n x n.
  Example:
    z = zeros[2, 3]
```

### `ones[rows, columns]`

```text
ones[rows, columns]        (import linear)
  A matrix of 1s; ones[n] is n x n.
  Example:
    o = ones[2, 3]
```

### `diagonal[x]`

```text
diagonal[x]        (import linear)
  From a list: the square matrix with those numbers on its diagonal. From
  a matrix: the list of the numbers on its diagonal.
  Example:
    d = diagonal[list [1, 2, 3]]
```

### `shape[m]`

```text
shape[m]        (import linear)
  The size: list [rows, columns].
  Example:
    m = matrix [1, 2, 3; 4, 5, 6]
    show shape[m] .
    first = row[m, 0]
    show first .
    prices = column[m, 2]
    show prices .
    t = transpose[m]
    show t .
```

### `row[m, r]`

```text
row[m, r]        (import linear)
  Row r (from 0) as a list.
  Example:
    m = matrix [1, 2, 3; 4, 5, 6]
    first = row[m, 0]
    show first .
```

### `column[m, c]`

```text
column[m, c]        (import linear)
  Column c (from 0) as a list.
  Example:
    m = matrix [1, 2, 3; 4, 5, 6]
    prices = column[m, 2]
    show prices .
```

### `flatten[m]`

```text
flatten[m]        (import linear)
  The numbers of m in one row, read row by row: a 1 x n matrix. m is
  unchanged. Method: m at flatten. Sentence form: m flatten.
  Example:
    m = matrix [1, 2, 3; 4, 5, 6]
    f = flatten[m]                  // [ 1  2  3  4  5  6 ]
```

### `reshape[m, rows, columns]`

```text
reshape[m, rows, columns]        (import linear)
  The same numbers, in the same order, as a rows x columns matrix. m is
  unchanged. rows times columns must be how many numbers m has, or it's a
  linear error. Method: m at reshape[rows, columns]. Sentence form:
  m reshape rows, columns.
  Example:
    m = matrix [1, 2, 3; 4, 5, 6]
    r = reshape[m, 3, 2]            // [ 1  2 ]
                                    // [ 3  4 ]
                                    // [ 5  6 ]
```

### `transpose[m]`

```text
transpose[m]        (import linear)
  The matrix with its rows as columns. Sentence form: m transpose.
  Example:
    m = matrix [1, 2, 3; 4, 5, 6]
    t = transpose[m]
    show t .
```

### `trace[m]`

```text
trace[m]        (import linear)
  The sum of the diagonal of a square matrix.
  Example:
    square = matrix [2, 1; 1, 3]
    show trace[square] .
    inv = inverse[square]
    show all_close[inv * square, identity[2]] .
```

### `determinant[m]`

```text
determinant[m]        (import linear)
  The determinant of a square matrix: 0 when it has no inverse. An integer
  for a matrix of whole numbers.
  Example:
    d = determinant[matrix [1, 2; 3, 4]]     // -2
```

### `inverse[m]`

```text
inverse[m]        (import linear)
  The matrix that multiplies m to the identity. A singular matrix (its
  determinant is 0) is a linear error.
  Example:
    m = matrix [2, 1; 1, 3]
    m = matrix [2, 1; 1, 3]
    inv = inverse[m]
    show inv .
```

### `rank[m]`

```text
rank[m]        (import linear)
  How many rows (or columns) are independent: none of them a mix of the
  others.
  Example:
    r = rank[matrix [1, 2; 2, 4]]            // 1
```

### `power[m, k]`

```text
power[m, k]        (import linear)
  m multiplied by itself k times; power[m, 0] is the identity, and a
  negative k is a power of the inverse.
  Example:
    steps = matrix [1, 1; 0, 1]
    later = power[steps, 10]
    show later .
```

### `multiply_each[a, b]`

```text
multiply_each[a, b]        (import linear)
  Element by element: each number times the one in the same place.
  Example:
    a = matrix [1, 2; 3, 4]
    b = matrix [5, 6; 7, 8]
    c = multiply_each[a, b]
    show c .
```

### `solve[a, b]`

```text
solve[a, b]        (import linear)
  The x with a * x == b, for a square a. b is a list (x is a list) or a
  matrix (one answer per column). Sentence form: a solve b.
  Example:
    x = solve[matrix [2, 1; 1, 3], list [3, 5]]
```

### `least_squares[a, b]`

```text
least_squares[a, b]        (import linear)
  The x that brings a * x closest to b, for more equations (rows) than
  unknowns (columns): the line or curve of best fit.
  Example:
    points = matrix [1, 1; 2, 1; 3, 1]
    ys = list [2, 4, 6]
    fit = least_squares[points, ys]
    show fit .
```

### `all_close[a, b]`

```text
all_close[a, b]        (import linear)
  Whether two matrices of the same size, or two lists of the same length,
  have every pair of numbers close (by the rule of is_close). Different
  sizes or lengths give false.
  tolerance  optional third argument: the most each pair may differ.
  Example:
    square = matrix [2, 1; 1, 3]
    square = matrix [2, 1; 1, 3]
    same is all_close[inverse[square] * square, identity[2]] .
    show same .
```

### `dot[u, v]`

```text
dot[u, v]        (import linear)
  The dot product of two lists of the same length.
  Example:
    d = dot[list [1, 2, 3], list [4, 5, 6]]  // 32
```

### `cross[u, v]`

```text
cross[u, v]        (import linear)
  The cross product of two 3-D vectors (lists of 3 numbers).
  Example:
    n = cross[list [1, 0, 0], list [0, 1, 0]]
```

### `norm[x]`

```text
norm[x]        (import linear)
  A list's length (the square root of its squares added up); for a
  matrix, every number counts (the Frobenius norm).
  Example:
    n = norm[list [3, 4]]                    // 5.0
```

### `unit[v]`

```text
unit[v]        (import linear)
  The list scaled to length 1, the same direction.
  Example:
    u = unit[list [3, 4]]                    // [ 0.6, 0.8 ]
```

### `lu[m]`

```text
lu[m]        (import linear)
  The LU decomposition of a square matrix: a map with "l" (lower, 1s on
  the diagonal), "u" (upper) and "p" (row order), where p * m == l * u.
  Example:
    square = matrix [2, 1; 1, 3]
    parts = lu[square]
    show length of parts .
```

### `qr[m]`

```text
qr[m]        (import linear)
  The QR decomposition: a map with "q" (orthonormal columns) and "r"
  (upper triangular), where q * r == m. Needs at least as many rows as
  columns.
  Example:
    m = matrix [1, 2; 3, 4; 5, 6]
    parts = qr[m]
    show length of parts .
```

### `eigen[m]`

```text
eigen[m]        (import linear)
  The eigenvalues and eigenvectors of a symmetric matrix: a map with
  "values" (a list, largest first) and "vectors" (a matrix, one vector
  per column, each of length 1).
  Example:
    covariances = matrix [2, 0; 0, 3]
    e = eigen[covariances]
    show length of e .
```

### `svd[m]`

```text
svd[m]        (import linear)
  The singular value decomposition, any shape: a map with "u", "s" (a
  list, largest first) and "v", where m == u * diagonal[s] * transpose[v].
  Example:
    m = matrix [1, 2; 3, 4]
    parts = svd[m]
    show length of parts .
```

<!-- end generated -->
