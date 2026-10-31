# `data` — collections, tables and statistics

[Library index](index.md) · `import data`

Working across collections (process, keep, reduce), text tables and table files, and statistics.


`import data` (or `import data [process, keep, copy, range, reduce, sum,
table, table_read, table_write]`) provides nine functions, and the
[statistics](#statistics) below. `table`, `table_read` and `table_write`
are described [below](#tables).

**`process` and `keep`: alone or as a value.** Written as a sentence on
its own, they change the collection **in place**. Used as a value
(assigned with `=` or `is`, or inside an expression), they give a **new**
collection and leave the original alone:

```
prices = list [100, 250]
prices process p give p * 2 .                  // prices is now [ 200, 500 ]

doubled is prices process p give p * 2 .        // doubled is [ 400, 1000 ];
doubled = prices process p give p * 2           //   prices stays [ 200, 500 ]
doubled = process[prices, p give p * 2]         // the call form, the same
```

Inside a function, a sentence changes the list the caller passed in
(the function gets that list, not a copy); give back a new one instead
when the caller's should stay as it was.

| Function | Args | Effect |
|---|---|---|
| `process` | collection, function | replaces each element (list/set) or each value (map) with the function's result; a set is deduplicated afterwards |
| `keep` | collection, function | keeps only the elements (list/set) or entries (map) for which the function gives a truthy result: a filter |
| `unique` | list or set | a new list with each value once, in the order they first appear (2 and 2.0 are one value); a set comes back as a copy. A map is an error: use `unique[map values]`-style code by passing its values as a list |
| `group` | collection, function | a map from each result of the function to the list of the items that gave it, in their order |
| `table_column` | rows, name | one column of a table as a list: the field (of an assembled row) or key (of a map row) named, one value per row |
| `table_row` | rows, n | row `n` of a table, counted from 0, as the record itself |
| `copy` | collection [, deep] | a new list/set/map/assembled value. `copy[x]`: the same items, so lists, maps or assembled values *inside* are shared; `copy[x, true]`: everything inside is copied too, so nothing is shared |

| `range` | [from,] to [, step] | a list of the whole numbers from `from` (0 if left out) up to `to`, **not including** `to`: `range[5]` is 0 to 4. `step` counts by that much; a negative step counts down (`range[5, 0, -1]` is 5 to 1). Empty when the start is already past the end |
| `reduce` | collection, start, function | one value: a running total that begins as `start` and, for each item (a map's values), becomes what `[total, x] give ...` gives. Changes nothing |
| `sum` | collection | the numbers of a list or set (a map's values) added up: an `integer` if they all are, else a `float`; `0` when empty. Anything that isn't a number is an error |

For a map, the function takes the value (`x give ...`), or the key and the
value (`[k, v] give ...`).

Each has a call form and a sentence form:

```
import data

r = range[5]                         // [ 0, 1, 2, 3, 4 ]: up to 5, not including it
r = range[1, 5]                      // [ 1, 2, 3, 4 ]
r = 1 range 5                        //   the same
evens = range[0, 10, 2]              // [ 0, 2, 4, 6, 8 ]
down = range[5, 0, -1]               // [ 5, 4, 3, 2, 1 ]
[loop][i in 1 range 4]               // 1, 2, 3
    show i .
[loop][end]

nums = list [1, 2, 3]
total = sum[nums]                    // 6
total = nums sum                     //   the same
total = nums reduce 0, [t, x] give t + x       // 6, spelled out
word = list ["t", "u"] reduce "", [w, c] give w + c   // "tu"
```

`reduce` works through the items in order: with `nums` above, the total
goes `0` → `0 + 1` → `1 + 2` → `3 + 3`, and the last total is the answer.
Where `process` gives one new item for each item, `reduce` gives one value
for the whole collection.

```
def double[x]
    return x * 2
def [end]
import data

nums = list [5, 3, 8, 1]
nums process x give x + 1 .        // [ 6, 4, 9, 2 ]
nums keep x give x > 3 .           // [ 6, 4, 9 ]

words = list ["hey", "do"]
words process x give x at upper .  // [ "HEY", "DO" ]

nums process double .               // any function value works

ages = map ["Alice": 30, "Bob": 25]
labels is ages process [name, age] give "{name} is {age}" .
ages keep [name, age] give age > 26 .   // { "Alice": 30 }

rows = list [list [1, 2], list [3, 4]]
outer = copy[rows]          // a new list holding the same inner lists
full = copy[rows, true]     // the inner lists copied too

nums process [x] give              // block form for longer logic
    if ] x > 5 [
        return x * 2
    if [end]
    return x
give [end]
```

These are ordinary functions, so you can write your own in a `.turtle` library
and call them the same sentence style. See
[`reference.md`](../reference.md#sentence-style-calls).

## Statistics

`import data` also has the everyday statistics. Each takes a list, a set
or a map (its values), or a list of rows (maps or assembled values, what
`sql_query` and `table_read` give) and a column name.

| Function | Gives |
|---|---|
| `mean[x]` | the average, a float |
| `median[x]` | the middle number once sorted (the number itself, so an integer stays one); with an even count, the average of the two middle ones |
| `mode[x]` | the value that appears most often, on a tie the one seen first; works on any values, not only numbers |
| `variance[x]`, `stdev[x]` | the **sample** variance and standard deviation, dividing by n − 1 (the usual choice for a sample); need 2 numbers |
| `pvariance[x]`, `pstdev[x]` | the **population** forms, dividing by n: for when the numbers are the whole population |
| `percentile[x, p]` | `p` from 0 to 100: the number that much of the data is below, between the two nearest when it falls between (numpy's default, a spreadsheet's `PERCENTILE`); `percentile[x, 50]` is the median |
| `covariance[xs, ys]` | how two lists move together, the sample form |
| `correlation[xs, ys]` | Pearson's correlation, from −1 through 0 to 1 |
| `zscores[x]` | a new list: each number's distance from the mean, in (sample) standard deviations |
| `describe[x]` | a map: `count`, `mean`, `stdev`, `min`, `25%`, `median`, `75%`, `max` |

> *Illustrative.* This example needs a database that already exists or a server, so it is shown for reading and is not run as a test.

```
import data

prices = list [950, 3000, 1225, 800]
show mean[prices] .               // 1493.75
show prices median .              // 1087.5: sentence form works too
show percentile[prices, 90] .     // 2467.5
show table[describe[prices]] .

books = sql_query[shelf, "SELECT * FROM books"]
show mean[books, "price"] .                  // one column of the rows
show correlation[books, "price", "rating"] .  // two columns
```

- **`none` is skipped**, as a missing value: a book without a rating
  doesn't stop its column's average. With nothing left, it's an error
  (kind `index`), as is `stdev` of a single number.
- Anything else that isn't a number is an error (kind `type`). Numbers
  in a CSV file come back as numbers (see [table files](#table-files));
  a column you read as text, or text from elsewhere, needs `change ...
  to float` first.
- Floats are added the careful way (compensated summation), so the mean
  of a thousand `0.1`s is `0.1`.

## Tables

`show` prints a value on one line. `table[x]` returns the same value laid
out as a text table, one row per line, so `show table[x] .` is easy to
read. It works on:

| Value | Rows | Columns |
|---|---|---|
| list of maps (what `sql_query` returns) | one per map | one per key |
| list of assembled values | one per value | one per field |
| list of lists | one per inner list | `0`, `1`, `2`, ... |
| map of records (assembled values or maps) | one per entry | `key`, then one per field or key |
| map of plain values | one per entry | `key`, `value` |
| a matrix (`import linear`) | one per row | `0`, `1`, `2`, ... |
| any other list or set | one per element | `#` (counting from 0) and `value` |
| one map | one per entry | `key` and `value` |
| one assembled value | one per field | `field` and `value` |

> *Illustrative.* This example needs a database that already exists or a server, so it is shown for reading and is not run as a test.

```
import sql
import data

shelf = sql_open["books.db"]
show table[sql_query[shelf, "SELECT * FROM books"]] .
```

```text
id  sku  title        price  rating  added
--  ---  -----------  -----  ------  ----------
 1  B1   Dune           950     4.5  2026-01-15
 2  B2   Long Book     3000    none  2026-02-01
 3  B3   Gone Girl     1225     3.9  2026-02-20
```

- Columns appear in the order their keys or fields are first seen, so
  rows with different keys still line up. A row without a column's key
  leaves the cell blank; a value of `none` shows as `none`.
- Columns of numbers are right-aligned, everything else left-aligned. Numbers
  already written as text (`"$1,250.50"`, `"12%"`) count as numbers.
  Text is shown without quotes; a line break inside a value shows as `\n`
  so the row stays on one line. Wide characters (emoji, CJK) are counted
  as two columns so the table still lines up.
- An empty list or set gives `no rows`.
- The result is a string, so it can also be written to a file:
  `report = table[rows]`.

**How many rows.** `import data` also creates a variable, `tablerows`,
set to `20`. A table shows at most that many rows, then a line such as
`... 480 more rows`. Set it in the import line, or change it like any
variable afterwards:

```
rows = list [1, 2, 3, 4, 5, 6, 7, 8]
import data [tablerows = 50]          // from the start
import data [tablerows = 50, process] // the same, and only process imported
tablerows = 50        // up to 50 rows from now on
tablerows = none      // every row
show table[rows, 5] . // at most 5 rows, for this table only
```

`tablerows` belongs to the file that imported `data`, and setting it
inside a function only changes it for that call, like any other
variable. If the file already has a variable called `tablerows` before
`import data`, it's kept.

## Table files

`table_write[path, x]` saves anything `table[x]` can show, and
`table_read[path]` reads it back as a list of maps, one per line, keyed
by the header line's names: the shape `sql_query` gives. The file name's
extension picks the format:

| Extension | Format | `table_read` |
|---|---|---|
| `.csv` (or any other) | comma-separated values | yes |
| `.tsv` | tab-separated values | yes |
| `.txt` | the aligned table `show table[x] .` prints, every row | no (it's for people) |
| `.json` | a list of objects, one per row: `[ {"item": "pen", "qty": 3}, ... ]` | yes, **keeping each value's kind** |

```
import data

assemble Order [item, qty, price]
orders = list [Order["pen", 3, 1.5], Order["mug", 2, 8.0]]
table_write["orders.csv", orders]       // item,qty,price / pen,3,1.5 / mug,2,8.0
rows = table_read["orders.csv"]
show rows at get[0] .                   // { "item": "pen", "qty": 3, "price": 1.5 }
```

- `table_write` returns how many rows it wrote and replaces the file if
  it's there. The columns are the ones `table[x]` shows (fields, keys,
  `#` and `value` for a plain list, ...).
- **`.json` keeps kinds**: numbers read back as numbers, `true`/`false`
  as booleans, `null` as `none`, and a list or object inside a row as a
  list or map. Dates are written as text. A row missing a name has
  `none` there; the columns are every name any row has, in the order
  they first appear.
- **From `.csv` and `.tsv`, what a cell holds decides its kind**, as in
  pandas and spreadsheets. Quotes don't change it; they only let a value
  hold a comma, a quote or a line break:

  | Cell | Reads as |
  |---|---|
  | `7`, `"7"`, `-3`, `2.5`, `1e5` | a number (`7` an integer; `2.5` and `1e5` floats) |
  | `007`, `"007"`, `+5`, `.5`, `1,000`, `$5` | text: as likely a code or an id as a quantity, so `007` keeps its zeros (pandas would make it `7`) |
  | a whole number too big for an integer (`123456789012345678901234`) | text: an id, not a quantity |
  | `Dune`, `true` | text |
  | an empty cell, or `""` | `none` |

  To keep a number-like column as text (a code column of `7`, `8`, ...),
  or read `true`/`yes` as booleans, give [column types](#column-types):
  `table_read["codes.csv", map ["code": "text"]]`.
- **Writing**, numbers are never quoted, and text only where it must be:
  when it holds the separator, a quote or a line break, or starts or
  ends with a space. To quote every text value instead (`"007"` rather
  than `007`), as some programs want, pass `map ["quote": "text"]`:

  ```
  table_write["codes.csv", rows]                             // 007,7
  table_write["codes.csv", rows, map ["quote": "text"]]      // "007",7
  ```
- Quotes inside a value are doubled (`"say ""hi"""`), the standard rule
  (RFC 4180). Blank lines are skipped, and a byte order mark at the
  start (Excel writes one) is ignored. `none` is written as an empty
  cell, so it reads back as `none`.
- A line with fewer values than the header gets `none` for the rest; one
  with more, or broken quotes (a `"` inside an unquoted value, text after
  a closing quote, a quote never closed), is an error of kind `csv`. A missing file
  is kind `file`.

## Column types

`table_read[path, types]` and `sql_load[db, table, path, types]` take a
map of column name to type after the file, so each column is read as the
right kind of value: `"007"` stays text, `"24"` becomes a number, and no
`change` is needed later. The map can be written in place or kept in a
variable.

> *Illustrative.* This example needs files that the example does not create, so it is shown for reading and is not run as a test.

```
import data

types = map ["code": "text", "missions": "integer", "rating": "float",
             "active": "boolean", "joined": "date", "primary_key": "code"]
rows = table_read["agents.csv", types]
show rows at get[0] .
// { "code": "007", "name": "James Bond", "missions": 24, "rating": 9.5, "active": true, "joined": 1953-04-13 00:00:00 }
```

| Type | Read as | Cells it takes |
|---|---|---|
| `string`, `text` | text, exactly as in the file | anything |
| `integer`, `int` | an integer | `42`, `3.0` (a whole number) |
| `float` | a float | `1.5`, `2` |
| `boolean`, `bool` | `true` / `false` | `true` `false` `yes` `no` `1` `0` `t` `f` `y` `n`, any case |
| `date` | a date | the forms `to_date` reads: `2026-10-06`, `2026-10-06 09:30`, ... |
| a SQL type: `VARCHAR(10)`, `BIGINT`, `DOUBLE PRECISION`, `NUMERIC(10, 2)`, ... | sorted by its words, as SQLite does: `INT` an integer; `CHAR`, `TEXT`, `CLOB`, `BLOB` text; `REAL`, `FLOA`, `DOUB` a float; `BOOL` a boolean; `DATE`, `TIME` a date; `NUMERIC`, `DECIMAL`, `NUMBER` an integer when whole, else a float | |

- Type words are in any case (`"INTEGER"`, `"Text"`).
- **Columns the map leaves out keep what the file has**: an unquoted
  number a number, anything else text (see [table files](#table-files)).
  A column the map names that the file doesn't have is an error of kind
  `name` (usually a typo).
- **`"primary_key": "column"`** names the key. `table_read` checks that
  every row has one and no two are the same (after converting, so `1`
  and `01` are the same integer), an error of kind `key` otherwise.
- An empty cell stays `none`. A cell that isn't its type is an error
  naming the row and column: kind `number` for numbers, `date` for
  dates, `type` for booleans.
- From `.json`, values that already have the right kind stay as they
  are; a number in a `text` column becomes its text, and a list or
  object in a `text` column becomes its JSON text.

For `sql_load`, see [Files: CSV in and out](sql.md#files-csv-in-and-out).

---

Copyright 2017-2026 Nigele McCoy. Licensed under the
[Apache License 2.0](../../LICENSE); see [NOTICE](../../NOTICE).

<!-- generated: turtle doc `data` (regenerate with the script; do not edit by hand) -->
## Function reference

The entries below are copied from the binary's own documentation (`turtle doc data`), so they match the version you run.

```text
import data
```

### `Working with collections (lists, sets, maps), and with tables of rows:`

```text
Working with collections (lists, sets, maps), and with tables of rows:
showing them, and saving them to and reading them from files.
"import data" also makes the variable tablerows (20): how many rows
table[...] shows. Change it like any variable; none shows every row.
```

### `process[collection, function]`

```text
process[collection, function]        (import data)
  Replaces every element (of a list or set) or every value (of a map)
  with what the function gives for it.
  collection   a list, set or map
  function     x give ...  (for a map: x give ..., or [key, value] give ...)
  As a sentence on its own it changes the collection itself; used as a
  value (assigned, or in an expression) it gives a new collection and
  leaves the original alone.
  Example:
    nums = list [1, 2, 3]
    bigger = nums process x give x * 10
    show bigger .
```

### `unique[collection]`

```text
unique[collection]        (import data)
  A new list with each value once, in the order they first appear (2 and
  2.0 are one value). A set is already unique: its copy comes back.
  Example:
    unique[list [3, 1, 3, 2]] .        // [ 3, 1, 2 ]
```

### `group[collection, function]`

```text
group[collection, function]        (import data)
  A map from each result of the function to the list of the items that
  gave that result, in their order in the collection. Example:
    group[list ["ant", "bee", "ape"], x give x at get[0]] .
    // { "a": [ "ant", "ape" ], "b": [ "bee" ] }
```

### `table_column[rows, name]`

```text
table_column[rows, name]        (import data)
  The values of one column of a table, one per row, in row order. A row
  is an assembled value or a map; the name is the field or key. Example:
    people = list [map ["first": "Ann", "last": "Lee"], map ["first": "Bob", "last": "Park"]]
    show table_column[people, "last"] .
    show table_row[people, 1] .
```

### `table_row[rows, n]`

```text
table_row[rows, n]        (import data)
  Row n of a table, counted from 0, as the record itself. Example:
    people = list [map ["first": "Ann", "last": "Lee"], map ["first": "Bob", "last": "Park"]]
    show table_row[people, 1] .
```

### `keep[collection, function]`

```text
keep[collection, function]        (import data)
  Keeps only the elements (or map entries) for which the function gives
  true: a filter. As a sentence on its own it changes the collection
  itself; used as a value it gives a new one and leaves the original.
  Example:
    nums = list [1, 2, 3, 4, 5]
    big = nums keep x give x > 3
    show big .
```

### `copy[value, deep]`

```text
copy[value, deep]        (import data)
  A new list, set, map or assembled value with the same contents.
  value   what to copy
  deep    optional: true copies what's inside too (lists in a list, maps
          in a list, an assembled value's lists), so nothing is shared;
          false or left out shares what's inside
  Example:
    rows = list [1, 2]
    outer = copy[rows]
    full = copy[rows, true]
    show full .
```

### `range[from, to, step]`

```text
range[from, to, step]        (import data)
  The whole numbers from from up to, not including, to, as a list.
  Sentence form: 1 range 5.
  from       optional: 0 if left out (range[5] is 0 to 4)
  to         where it stops, not included
  step       optional: 2 counts by twos; a negative step counts down
  Example:
    show range[5] .
    show range[1, 5] .
    show range[0, 10, 2] .
    show range[5, 0, -1] .
```

### `reduce[collection, start, function]`

```text
reduce[collection, start, function]        (import data)
  Boils a collection down to one value: a running total that starts at
  start, and for each item (a map's values) becomes what the function
  gives for the total so far and the item. Changes nothing.
  function   [total, x] give ...
  Example:
    nums = list [1, 2, 3]
    total = nums reduce 0, [t, x] give t + x
    show total .
```

### `sum[collection]`

```text
sum[collection]        (import data)
  Adds up a list's or set's numbers (a map's values): an integer if they
  all are, 0 when there are none. Sentence form: nums sum.
  Example:
    prices = list [4, 9, 6]
    total = sum[prices]
    avg = mean[prices]
    spread = stdev[prices]
    show total, " ", avg, " ", spread .
```

### `Statistics: each takes a list, a set or a map (its values), or a list of`

```text
Statistics: each takes a list, a set or a map (its values), or a list of
rows (maps or assembled values, as sql_query and table_read give) and a
column name. none is skipped, as a missing value; anything else that
isn't a number is an error. Sentence form: prices mean.
```

### `mean[numbers, column]`

```text
mean[numbers, column]        (import data)
  The average: the numbers added up, divided by how many there are.
  column   optional: with a list of rows, the column to use
  Gives back: a float.
  Example:
    prices = list [4, 9, 6]
    avg = mean[prices]
    show avg .
```

### `median[numbers, column]`

```text
median[numbers, column]        (import data)
  The middle number once sorted; with an even count, the average of the
  two in the middle.
  Gives back: the middle number itself (an integer stays one), or a float.
  Example:
    mid = median[list [3, 1, 2]]          // 2
```

### `mode[values, column]`

```text
mode[values, column]        (import data)
  The value that appears most often; a tie goes to the one seen first.
  Works on any values, not only numbers.
  Example:
    top = mode[list ["tea", "coffee", "tea"]]   // "tea"
```

### `variance[numbers, column]`

```text
variance[numbers, column]        (import data)
  The sample variance: the average squared distance from the mean,
  dividing by n - 1, the usual choice for a sample. Needs 2 numbers.
  Example:
    prices = list [4, 9, 6]
    prices = list [4, 9, 6]
    v = variance[prices]
    show v .
```

### `stdev[numbers, column]`

```text
stdev[numbers, column]        (import data)
  The sample standard deviation: the square root of variance. Needs 2
  numbers.
  Example:
    prices = list [4, 9, 6]
    spread = stdev[prices]
    show spread .
```

### `pvariance[numbers, column]`

```text
pvariance[numbers, column]        (import data)
  The population variance: like variance, dividing by n. Use it when the
  numbers are the whole population, not a sample of it.
  Example:
    scores = list [2, 4, 4, 4, 5, 5, 7, 9]
    v = pvariance[scores]
    spread = pstdev[scores]
    show v, " ", spread .
```

### `pstdev[numbers, column]`

```text
pstdev[numbers, column]        (import data)
  The population standard deviation: the square root of pvariance.
  Example:
    scores = list [2, 4, 4, 4, 5, 5, 7, 9]
    spread = pstdev[scores]
    show spread .
```

### `percentile[numbers, column, percent]`

```text
percentile[numbers, column, percent]        (import data)
  The number below which that percent of the numbers fall, between the
  two nearest when it lands between them (as numpy and a spreadsheet's
  PERCENTILE give). percentile[x, 50] is the median.
  percent   0 to 100
  Example:
    times = list [1, 2, 3, 4, 5, 6, 7, 8, 9, 10]
    p90 = percentile[times, 90]
    show p90 .
```

### `covariance[xs, ys]`

```text
covariance[xs, ys]        (import data)
  How two lists of numbers move together, the sample form (n - 1):
  positive when they rise together. Also covariance[rows, "a", "b"]. A
  pair with none on either side is skipped.
  Example:
    heights = list [150, 160, 170, 180]
    weights = list [50, 58, 70, 81]
    c = covariance[heights, weights]
    show c .
```

### `correlation[xs, ys]`

```text
correlation[xs, ys]        (import data)
  Pearson's correlation, from -1 (one falls as the other rises) through 0
  (no straight-line link) to 1 (they rise together). Also
  correlation[rows, "a", "b"].
  Example:
    ads = list [1, 2, 3, 4]
    sales = list [2, 4, 5, 8]
    r = correlation[ads, sales]
    show r .
```

### `zscores[numbers, column]`

```text
zscores[numbers, column]        (import data)
  Each number's distance from the mean, counted in (sample) standard
  deviations.
  Gives back: a new list of floats, in the same order.
  Example:
    z = zscores[list [1, 2, 3]]          // [ -1.0, 0.0, 1.0 ]
```

### `describe[numbers, column]`

```text
describe[numbers, column]        (import data)
  A summary: count, mean, stdev (none with one number), min, 25%, median,
  75%, max.
  Gives back: a map; show table[describe[x]] . lays it out.
  Example:
    books = list [map ["price": 900], map ["price": 1500]]
    show table[describe[books, "price"]] .
```

### `table[rows, limit]`

```text
table[rows, limit]        (import data)
  Lays rows out as a text table, one row per line, columns lined up.
  rows    a list of maps (what sql_query gives), a list of assembled
          values, a list of lists, a plain list, a set, one map, or one
          assembled value
  limit   optional: at most this many rows for this call (else tablerows)
  Gives back: text, so show it: show table[rows] .
  A row without one of the columns leaves that cell blank; none shows as
  none. Numbers line up on the right, and so does text written as a number
  ("$1,250.50", "12%"). Rows past the limit end with a
  line like "... 12 more rows".
  Example:
    import sql
    db = sql_create["shop.db"]
    sql_run[db, "CREATE TABLE books (title TEXT, price INTEGER)", list []]
    show table[sql_query[db, "SELECT * FROM books"]] .
    sql_close[db]
```

### `table_write[path, rows, options]`

```text
table_write[path, rows, options]        (import data)
  Saves rows to a file, in the layout table[...] shows. The file's
  ending picks the format: .csv (comma-separated, also any other ending),
  .tsv (tab-separated), .txt (the aligned table, every row), or .json
  (a list of objects, one per row, keeping numbers and true/false).
  path     where to save; an existing file is replaced
  rows     anything table[...] takes
  options  optional: map ["quote": "text"] quotes every text value of a
           .csv or .tsv file ("007", not 007). The default, "needed",
           quotes text only where it must (it holds a comma, a quote or
           a line break).
  Numbers are never quoted, so they read back as numbers.
  Gives back: how many rows were written.
  Example:
    orders = list [map ["item": "pen", "qty": 3]]
    table_write["orders.csv", orders]
    codes = list [map ["code": "007"]]
    table_write["codes.csv", codes, map ["quote": "text"]]
```

### `table_read[path [, types]]`

```text
table_read[path [, types]]        (import data)
  Reads a .csv, .tsv or .json file. A .csv or .tsv file's first line
  names the columns; a .json file is a list of objects, one per row.
  path   the file
  types  optional: a map of column name to type, to say what a
         column is: "code": "text", "qty": "integer". Types: string,
         text, integer, float, boolean, date, or a SQL type
         (VARCHAR(10), NUMERIC(10, 2), ...). Columns left out keep what
         the file has. "primary_key": "column" checks every row has a
         different one.
  Gives back: a list of maps, one per row, keyed by the column names
  (the same shape sql_query gives). From .csv and .tsv, what a cell holds
  decides, quoted or not (as in pandas): a plain number (950, "950",
  -2.5, 1e5) is a number; 007 and other text is text; an empty cell is
  none. From .json, values keep their kind.
  A badly formed file is a csv error (json for a .json file); a cell
  that isn't its type is a number, date or type error.
  Example:
    [write] orders.csv
    "item,qty"
    "pen,3"
    [end]
    [write] orders.csv
    "item,qty"
    "pen,3"
    [end]
    rows = table_read["orders.csv"]
    show table[rows] .
```

<!-- end generated -->
