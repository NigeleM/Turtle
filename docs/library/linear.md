# `linear`: matrices and linear algebra

`import linear`

Matrices and linear algebra: build, multiply and invert matrices, solve systems of equations, and find eigenvalues and decompositions.


## Functions

`identity`, `zeros`, `ones`, `diagonal`, `shape`, `row`, `column`, `flatten`, `reshape`, `transpose`, `trace`, `determinant`, `inverse`, `rank`, `power`, `multiply_each`, `solve`, `least_squares`, `all_close`, `dot`, `cross`, `norm`, `unit`, `lu`, `qr`, `eigen`, `svd`.

Matrices and vectors. After `import linear`, `matrix [...]` makes a matrix: each row is separated by a `;` or a new line, and the numbers in a row by `,`. For example `a = matrix [1, 2; 3, 4]`. `+` and `-` work on two matrices of the same size, or a matrix and a number (on every element: `a + 1`). `*` is the matrix product (a matrix times a list gives a list), and `*` or `/` by a number scales every element.

### `identity[n]`

The n x n identity matrix: 1s on the diagonal, 0s elsewhere.

**Example**

```turtle
import linear
i = identity[3]
show i .
```

**Output**

```text
[ 1  0  0 ]
[ 0  1  0 ]
[ 0  0  1 ]
```

### `zeros[rows, columns]`

A matrix of 0s; zeros[n] is n x n.

**Example**

```turtle
import linear
z = zeros[2, 3]
show z .
```

**Output**

```text
[ 0  0  0 ]
[ 0  0  0 ]
```

### `ones[rows, columns]`

A matrix of 1s; ones[n] is n x n.

**Example**

```turtle
import linear
o = ones[2, 3]
show o .
```

**Output**

```text
[ 1  1  1 ]
[ 1  1  1 ]
```

### `diagonal[x]`

From a list: the square matrix with those numbers on its diagonal.

**Example**

```turtle
import linear
d = diagonal[list [1, 2, 3]]
show d .
```

**Output**

```text
[ 1  0  0 ]
[ 0  2  0 ]
[ 0  0  3 ]
```

### `shape[m]`

The size: list [rows, columns].

**Example**

```turtle
m = matrix [1, 2, 3; 4, 5, 6]
show shape[m] .
first = row[m, 0]
show first .
prices = column[m, 2]
show prices .
t = transpose[m]
show t .
```

**Output**

```text
[ 2, 3 ]
[ 1, 2, 3 ]
[ 3, 6 ]
[ 1  4 ]
[ 2  5 ]
[ 3  6 ]
```

### `row[m, r]`

Row r (from 0) as a list.

**Example**

```turtle
m = matrix [1, 2, 3; 4, 5, 6]
first = row[m, 0]
show first .
```

**Output**

```text
[ 1, 2, 3 ]
```

### `column[m, c]`

Column c (from 0) as a list.

**Example**

```turtle
m = matrix [1, 2, 3; 4, 5, 6]
prices = column[m, 2]
show prices .
```

**Output**

```text
[ 3, 6 ]
```

### `flatten[m]`

The numbers of m in one row, read row by row: a 1 x n matrix. m is unchanged.

**Example**

```turtle
import linear
m = matrix [1, 2, 3; 4, 5, 6]
f = flatten[m]
show f .
```

**Output**

```text
[ 1  2  3  4  5  6 ]
```

### `reshape[m, rows, columns]`

The same numbers, in the same order, as a rows x columns matrix. m is unchanged. rows times columns must be how many numbers m has, or it's a linear error.

**Example**

```turtle
import linear
m = matrix [1, 2, 3; 4, 5, 6]
r = reshape[m, 3, 2]
show r .
```

**Output**

```text
[ 1  2 ]
[ 3  4 ]
[ 5  6 ]
```

### `transpose[m]`

The matrix with its rows as columns.

**Example**

```turtle
m = matrix [1, 2, 3; 4, 5, 6]
t = transpose[m]
show t .
```

**Output**

```text
[ 1  4 ]
[ 2  5 ]
[ 3  6 ]
```

### `trace[m]`

The sum of the diagonal of a square matrix.

**Example**

```turtle
square = matrix [2, 1; 1, 3]
show trace[square] .
inv = inverse[square]
show all_close[inv * square, identity[2]] .
```

**Output**

```text
5
true
```

### `determinant[m]`

The determinant of a square matrix: 0 when it has no inverse.

**Example**

```turtle
import linear
d = determinant[matrix [1, 2; 3, 4]]
show d .
```

**Output**

```text
-2
```

### `inverse[m]`

The matrix that multiplies m to the identity.

**Example**

```turtle
m = matrix [2, 1; 1, 3]
m = matrix [2, 1; 1, 3]
inv = inverse[m]
show inv .
```

**Output**

```text
[  0.6  -0.2 ]
[ -0.2   0.4 ]
```

### `rank[m]`

How many rows (or columns) are independent: none of them a mix of the others.

**Example**

```turtle
import linear
r = rank[matrix [1, 2; 2, 4]]
show r .
```

**Output**

```text
1
```

### `power[m, k]`

m multiplied by itself k times; power[m, 0] is the identity, and a negative k is a power of the inverse.

**Example**

```turtle
steps = matrix [1, 1; 0, 1]
later = power[steps, 10]
show later .
```

**Output**

```text
[ 1  10 ]
[ 0   1 ]
```

### `multiply_each[a, b]`

Element by element: each number times the one in the same place.

**Example**

```turtle
a = matrix [1, 2; 3, 4]
b = matrix [5, 6; 7, 8]
c = multiply_each[a, b]
show c .
```

**Output**

```text
[  5  12 ]
[ 21  32 ]
```

### `solve[a, b]`

The x with a * x == b, for a square a. b is a list (x is a list) or a matrix (one answer per column).

**Example**

```turtle
import linear
x = solve[matrix [2, 1; 1, 3], list [3, 5]]
show x .
```

**Output**

```text
[ 0.8, 1.4 ]
```

### `least_squares[a, b]`

The x that brings a * x closest to b, for more equations (rows) than unknowns (columns): the line or curve of best fit.

**Example**

```turtle
points = matrix [1, 1; 2, 1; 3, 1]
ys = list [2, 4, 6]
fit = least_squares[points, ys]
show fit .
```

**Output**

```text
[ 1.9999999999999998, 1.7536473844680002e-16 ]
```

### `all_close[a, b]`

Whether two matrices of the same size, or two lists of the same length, have every pair of numbers close (by the rule of is_close).

**Example**

```turtle
square = matrix [2, 1; 1, 3]
square = matrix [2, 1; 1, 3]
same is all_close[inverse[square] * square, identity[2]] .
show same .
```

**Output**

```text
true
```

### `dot[u, v]`

The dot product of two lists of the same length.

**Example**

```turtle
import linear
d = dot[list [1, 2, 3], list [4, 5, 6]]
show d .
```

**Output**

```text
32
```

### `cross[u, v]`

The cross product of two 3-D vectors (lists of 3 numbers).

**Example**

```turtle
import linear
n = cross[list [1, 0, 0], list [0, 1, 0]]
show n .
```

**Output**

```text
[ 0, 0, 1 ]
```

### `norm[x]`

A list's length (the square root of its squares added up); for a matrix, every number counts (the Frobenius norm).

**Example**

```turtle
import linear
n = norm[list [3, 4]]
show n .
```

**Output**

```text
5.0
```

### `unit[v]`

The list scaled to length 1, the same direction.

**Example**

```turtle
import linear
u = unit[list [3, 4]]
show u .
```

**Output**

```text
[ 0.6, 0.8 ]
```

### `lu[m]`

The LU decomposition of a square matrix: a map with "l" (lower, 1s on the diagonal), "u" (upper) and "p" (row order), where p * m == l * u.

**Example**

```turtle
square = matrix [2, 1; 1, 3]
parts = lu[square]
show length of parts .
```

**Output**

```text
3
```

### `qr[m]`

The QR decomposition: a map with "q" (orthonormal columns) and "r" (upper triangular), where q * r == m.

**Example**

```turtle
m = matrix [1, 2; 3, 4; 5, 6]
parts = qr[m]
show length of parts .
```

**Output**

```text
2
```

### `eigen[m]`

The eigenvalues and eigenvectors of a symmetric matrix: a map with "values" (a list, largest first) and "vectors" (a matrix, one vector per column, each of length 1).

**Example**

```turtle
covariances = matrix [2, 0; 0, 3]
e = eigen[covariances]
show length of e .
```

**Output**

```text
2
```

### `svd[m]`

The singular value decomposition, any shape: a map with "u", "s" (a list, largest first) and "v", where m == u * diagonal[s] * transpose[v].

**Example**

```turtle
m = matrix [1, 2; 3, 4]
parts = svd[m]
show length of parts .
```

**Output**

```text
3
```
