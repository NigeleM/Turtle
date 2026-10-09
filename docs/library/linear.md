# `linear`: matrices and linear algebra

[Library index](index.md) · `import linear`

Matrices, their arithmetic, and solving, inverting and decomposing them.
The work is done in Go, so it's fast: a 200 × 200 product takes a few
milliseconds.

```
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
show norm[list [3, 4]] .
```

```
5.0
```

### `unit[v]`

The vector scaled to length 1, in the same direction.

- `v`: a list of numbers, not all 0

**Gives:** a list.

```
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
