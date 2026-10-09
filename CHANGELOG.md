# Changelog

What changed in each release of Turtle, newest first.

**Turtle is pre-release (0.x).** The language and its libraries are still
settling, so a release may change how existing code behaves. Every such
change is listed under **Breaking**, with what to do about it. From 1.0 on,
a breaking change will wait for a major version.

For the full story of each change, see [`PROGRESS.md`](PROGRESS.md).

## 0.9.169 (2026-10-08)

### Breaking

- **Turtle files end in `.turtle`.** `.trt` still works everywhere
  (imports, `turtle test`, `fmt`, `doc`, `build`, the editors), so
  nothing breaks; but an import that finds both `name.turtle` and
  `name.trt` stops with an error naming both.
  *What to do:* nothing; rename files to `.turtle` when convenient.
- **`&&`, `||` and comparisons after a sentence call work on its
  result**, as in other languages: `s has "a" && ok` is
  `(s has "a") && ok` (it used to pass `"a" && ok` as the argument).
  Arithmetic still belongs to the argument (`nums get i + 1` is
  `get[nums, i + 1]`), and a `give` function still takes the rest of its
  line.

### Fixed

- **The web server:** one slow client could freeze every other request;
  now only the Turtle handler takes a turn, with time limits on reading a
  request. A static folder no longer serves hidden files (`.env`, `.git`)
  or links leading out of it.

### New

- Databases written by Turtle programs are checked by the official
  `sqlite3` in the tests, and CI requires the tool rather than skipping;
  CI also runs the editor, REPL and formatter tests. The `sql` and
  `server` docs have notes on keeping your data and server safe.

## 0.9.168 (2026-10-08)

### Breaking

- **A new license: the Apache License 2.0.** Earlier versions carried a
  file titled "MIT License" that also forbade selling Turtle, which isn't
  MIT. Under Apache 2.0 anyone may use, change, share and build on Turtle,
  including in their own builds, devices and commercial products, and
  programs written in Turtle belong to their authors.
  *What to do:* keep the `LICENSE` and `NOTICE` files with any copy of
  Turtle you share; a changed version needs its own name (see `BRAND.md`).

### New

- `NOTICE` credits Nigele McCoy as Turtle's creator; every redistribution
  must keep it.
- A name and logo policy (`BRAND.md`): what you may do with the name
  without asking, what needs permission, and how to ask. (In 0.9.168 it
  was `TRADEMARKS.md` and called the name a trademark; no trademark is
  registered, so it was reworded after the release.)
- Contributor terms: contributors keep their copyright and give the
  project a permanent license; what stays in Turtle is the creator's
  decision.
- The VS Code extension's publisher is `turtle-lang`, and it carries the
  same `LICENSE` and `NOTICE`.

## 0.9.167 (2026-10-08)

### Breaking

- **Very big and very small floats show in scientific notation**, as
  Python shows them: `1e+20`, `1e-05` (from 1e16 up and below 0.0001).
  They used to show every digit (`100000000000000000000.0`).
  *What to do:* nothing, unless your program compares a float's shown
  text; files and databases still get every digit.
- **CSV and TSV cells written in scientific notation (`1e5`) read as
  numbers**, as pandas reads them; they were text in 0.9.166.
  *What to do:* to keep such a column as text, give column types:
  `table_read[path, map ["code": "text"]]`.

### New

- **Scientific notation** in code: `1e-18`, `2.5e6`, `6.02E+23`.
- **A method's argument without brackets, anywhere**: `2 at pow 10`,
  `nums at get 0`, `if ] s at contains "a" [`. It's just the value right
  after the method, so `3 at pow 2 == 9` compares the power; two or more
  arguments go in brackets (`m at get[0, 1]`). After `of`,
  `n of rows at get 0` reaches into the list as `get[0]` does.
- **A method call can be a line of its own**: `nums at add 4`,
  `m at put[9, 1, 2]`, `m at put 9, 1, 2 .`; and `put 9 to m at 1, 2 .`
  puts into a matrix.
- **An assignment may end in a period**, as a call line may:
  `big = rows keep r give price of r > 10 .`
- **Library docs, one page per library** (`docs/library/`), grouped by
  purpose like Python's; a Keywords table in the reference with every
  word, what it does and an example; import shown by example.
- Releases of 0.x versions are marked pre-release on GitHub.
- VS Code extension 0.1.3: colors the new functions, `matrix` and
  scientific numbers.

### Faster

- Scopes are reused when no function kept them, a fast reader for
  well-formed JSON, maps stored as one list of entries, names shared and
  matched by address. Every result is the same. Against 0.9.166: fib
  44 -> 34 ms, functional 34 -> 20, objects 37 -> 30, strings 39 -> 31,
  jsondata 58 -> 24, maps 75 -> 68; 200,000 small records use 100 MB
  instead of 216 MB.

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
