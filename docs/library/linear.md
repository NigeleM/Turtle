# `linear` — matrices and linear algebra

[Library index](index.md) · `import linear`

A matrix value with its operators, and solving, inverting and decomposing matrices.


`import linear` adds a value for matrices and the arithmetic of linear
algebra. The work is done in Go, on one block of numbers per matrix, so
it's fast: a 200 × 200 product takes a few milliseconds (see
[the benchmark](../../bench/RESULTS.md)).

## Writing a matrix

```
import linear

a = matrix [1, 2; 3, 4]           // rows end at ;
b = matrix [
    1, 2, 3
    4, 5, 6
]                                  // ... or at the end of a line
c = matrix [
    1, 2, 3, 4,
    5, 6, 7, 8
]                                  // a comma at the end carries a long row on
show b .
```
```
[ 1  2  3 ]
[ 4  5  6 ]
```

- `matrix` is a word only in a file that imports `linear`. Elsewhere
  it's an ordinary name; in such a file it can't name a variable or a
  function.
- Every row needs the same count of numbers. Any expression can be a
  number: `matrix [x * 2, f[x]; 0, 1]`. A row can start with a minus sign:
  on a new line, `-3, 4` is a row, not a subtraction.
- A matrix of whole numbers shows and gives back integers. Anything that
  can make a fraction (`/`, `inverse`, `solve`, a float in it) gives a
  float matrix, which shows `2.0`, as a float does.
- `show` lines the columns up. A big matrix shows its first 20 rows and
  10 columns, `...` for the rest, and its size: `(25 x 14 matrix)`.
- Inside a list or map a matrix shows on one line, the way it's written:
  `[ matrix [1, 2; 3, 4] ]`.
- A number smaller than a ten-trillionth of the matrix's largest is float
  arithmetic's leftover (`inverse[a] * a` gives `1e-16` where a `0`
  belongs): it shows as `0.0` and compares equal to `0`, as `0.1 + 0.2`
  shows `0.3`. `get` still gives it exactly.
- Like a list, a matrix is shared, not copied: after `b = a`, a `put` on
  `b` changes `a` too. `copy[a]` (`import data`) makes a separate one.
  Operators always make a new matrix.

From other values, and back, with `change`:

```
rows = list [list [1, 2], list [3, 4]]
m = change rows to matrix                    // a list of lists: each a row
m = change table_read["points.csv", types] to matrix   // rows of a table file
back = change m to list                      // [ [ 1, 2 ], [ 3, 4 ] ]
```

## Operators

As in mathematics:

| Expression | Gives |
|---|---|
| `a + b`, `a - b` | element by element; the sizes must match |
| `a + 1`, `a - 1`, `10 - a` | the number on every element |
| `a * b` | the matrix product: `a` needs as many columns as `b` has rows |
| `a * v`, `v * a` (`v` a list) | a matrix times a vector (or a vector times a matrix): a list |
| `a * 2`, `2 * a`, `a / 2` | every number scaled |
| `-a` | every number negated |
| `a == b`, `a != b` | the same size and the same numbers |

A number added to or taken from a matrix goes on every element, as in
numpy and MATLAB: `a + 1`, `10 - a`. (That isn't `a + identity[n]`,
which adds to the diagonal only.) For element-by-element multiplication
of two matrices, use `multiply_each[a, b]`.

## Methods

| Method | Gives |
|---|---|
| `m at rows`, `m at columns` | the counts |
| `m at shape` | `list [rows, columns]` |
| `m at get[r, c]` | the number at row `r`, column `c`, from 0 |
| `m at put[value, r, c]` | puts a number there (value first, as a list's `put`); gives the matrix. On its own line: `m at put[9, 1, 2]`, `m at put 9, 1, 2 .`, or the sentence `put 9 to m at 1, 2 .` |
| `m at row[r]`, `m at column[c]` | that row or column, as a list |
| `m at isempty`, `m at tostring` | as for lists |

## Functions

| Function | Gives |
|---|---|
| `identity[n]` | the n × n identity |
| `zeros[r, c]`, `ones[r, c]` | a matrix of 0s or 1s; with one size, square |
| `diagonal[list]` / `diagonal[m]` | the square matrix with those numbers on its diagonal / the list of a matrix's diagonal |
| `shape[m]`, `row[m, r]`, `column[m, c]` | as the methods |
| `transpose[m]` | rows as columns |
| `trace[m]` | the sum of the diagonal |
| `determinant[m]` | an integer for a matrix of whole numbers |
| `inverse[m]` | the inverse; a singular matrix is an error |
| `rank[m]` | how many rows are independent |
| `power[m, k]` | `m` times itself `k` times; `0` gives the identity, a negative `k` the inverse's power |
| `multiply_each[a, b]` | element by element |
| `solve[a, b]` | the `x` with `a * x == b`, for a square `a`: `b` a list (a list back) or a matrix (one answer per column) |
| `least_squares[a, b]` | the `x` that brings `a * x` closest to `b`, for more equations than unknowns: a line of best fit |
| `dot[u, v]`, `cross[u, v]` | of two lists (cross: of 3 numbers each) |
| `norm[v]` | a list's length; a matrix's Frobenius norm |
| `unit[v]` | the list scaled to length 1 |
| `lu[m]` | a map: `"l"`, `"u"`, `"p"`, where `p * m == l * u` |
| `qr[m]` | a map: `"q"`, `"r"`, where `q * r == m` (at least as many rows as columns) |
| `eigen[m]` | for a symmetric matrix: a map, `"values"` (largest first) and `"vectors"` (one per column) |
| `svd[m]` | any shape: a map, `"u"`, `"s"` (a list, largest first), `"v"`, where `m == u * diagonal[s] * transpose[v]` |

```
import linear

a = matrix [2, 1; 1, 3]
x = solve[a, list [3, 5]]          // [ 0.8, 1.4 ]
x = a solve list [3, 5]            //   the same, as a sentence
show a * x .                       // [ 3.0, 5.0 ]

// A line through three points: y = b0 + b1 * x.
points = matrix [1, 1; 1, 2; 1, 3]
fit = least_squares[points, list [1, 2, 2]]   // [ 0.666666666666667, 0.5 ]
```

- Errors are of kind `linear`: sizes that don't fit (the message names
  both: "can't multiply a 2 x 3 matrix by a 2 x 3 matrix"), a singular
  matrix for `inverse` or `solve`, a non-symmetric one for `eigen`.
- A matrix counts as singular when it is, next to the size of its
  numbers: `matrix [1e-18, 0; 0, 1e-18]` has an inverse.
- `solve` and `least_squares` refine their answer once, which wins back
  the last digits rounding loses.
- Big products use every core of the computer.
- A worked example that uses most of it: `testdata/linear/housing.turtle`.
