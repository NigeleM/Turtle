# Changelog

What changed in each release of Turtle, newest first.

**Turtle is pre-release (0.x).** The language and its libraries are still
settling, so a release may change how existing code behaves. Every such
change is listed under **Breaking**, with what to do about it. From 1.0 on,
a breaking change will wait for a major version.

For the full story of each change, see [`PROGRESS.md`](PROGRESS.md).

## 0.9.166 (2026-10-08)

### Breaking

- **CSV and TSV files: numbers come back as numbers.** `table_read`
  without column types used to give every value as text (`"950"`); now a
  plain number gives a number (`950`), quoted or not, as pandas and
  spreadsheets read it. `007`, `1e5`, `+5`, `1,000` and whole numbers too
  big for an integer stay text, so codes and ids keep their zeros. An
  empty cell, or `""`, is `none`.
  *What to do:* code that compared a value with text (`price of row ==
  "950"`) or changed it to a number first still runs, but the comparison
  is now false; compare with the number, or keep the column as text with
  column types: `table_read["codes.csv", map ["code": "text"]]`.
- **Columns a types map leaves out keep what the file has**, rather than
  becoming text. `sql_load` making a new table types such a column from
  its values (integer, float or text) instead of `TEXT`.
- **`sql_load`, `sql_update` and `sql_upsert` into a table that's already
  there convert each cell to the table's column types**, on SQLite,
  PostgreSQL and MySQL. A cell that can't be its column's type (`lots` in
  an `INTEGER` column) stops the file, with the row and column, before
  anything is written; it used to go in as given.

### New

- **`import linear`**: a matrix value, `matrix [1, 2; 3, 4]` or one row
  per line. `+`, `-` and `*` as in mathematics (`*` is the matrix
  product; a number added or multiplied goes on every element), methods
  `rows`, `columns`, `shape`, `get`, `put`, `row`, `column`, and
  functions `identity`, `zeros`, `ones`, `diagonal`, `shape`, `row`,
  `column`, `transpose`, `trace`, `determinant`, `inverse`, `rank`,
  `power`, `multiply_each`, `solve`, `least_squares`, `dot`, `cross`,
  `norm`, `unit`, `lu`, `qr`, `eigen`, `svd`. `show` lines a matrix up and
  cuts it past 20 rows or 10 columns. `matrix` is a word only in a file
  that imports `linear`. Errors are of kind `linear`.
- **Statistics in `data`**: `mean`, `median`, `mode`, `variance`,
  `stdev`, `pvariance`, `pstdev`, `percentile`, `covariance`,
  `correlation`, `zscores`, `describe`, on a list or a column of rows
  (`mean[books, "price"]`). `none` is skipped; sample forms by default.
- `table_write[path, rows, map ["quote": "text"]]` quotes every text
  value (`"007"`); by default text is quoted only where it must be.
- `change rows to matrix` and `change m to list`; `table[m]` and `copy[m]`
  take a matrix.
- The REPL continues an unfinished `matrix [`, `turtle fmt` indents a
  matrix's rows, and the VS Code grammar and `turtle doc` know the new
  words.

### Faster

- The benchmark has `matrix` and `stats`: Turtle 13 ms against Python's
  256 ms for a 200 x 200 product and solve (see `bench/RESULTS.md`).
  `bench/linear.trt` shows how each operation scales.

## Earlier

Releases before 0.9.166 are described in [`PROGRESS.md`](PROGRESS.md),
session by session.
