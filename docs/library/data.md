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
| `copy` | collection [, deep] | a new list/set/map/assembled value. `copy[x]`: the same items, so lists, maps or assembled values *inside* are shared; `copy[x, true]`: everything inside is copied too, so nothing is shared |

| `range` | [from,] to [, step] | a list of the whole numbers from `from` (0 if left out) up to `to`, **not including** `to`, as in Python and Go: `range[5]` is 0 to 4. `step` counts by that much; a negative step counts down (`range[5, 0, -1]` is 5 to 1). Empty when the start is already past the end |
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
import data

nums = list [5, 3, 8, 1]
nums process x give x + 1 .        // [ 6, 4, 9, 2 ]
nums keep x give x > 3 .           // [ 6, 4, 9 ]

words = list ["hey", "do"]
words process x give x at upper .  // [ "HEY", "DO" ]

nums process double .               // any function value works

ages = map ["Alice": 30, "Bob": 25]
labels is ages process [name, age] give name + " is " + age .
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
| `variance[x]`, `stdev[x]` | the **sample** variance and standard deviation, dividing by n − 1, as Python's `statistics` and spreadsheets do; need 2 numbers |
| `pvariance[x]`, `pstdev[x]` | the **population** forms, dividing by n: for when the numbers are the whole population |
| `percentile[x, p]` | `p` from 0 to 100: the number that much of the data is below, between the two nearest when it falls between (numpy's default, a spreadsheet's `PERCENTILE`); `percentile[x, 50]` is the median |
| `covariance[xs, ys]` | how two lists move together, the sample form |
| `correlation[xs, ys]` | Pearson's correlation, from −1 through 0 to 1 |
| `zscores[x]` | a new list: each number's distance from the mean, in (sample) standard deviations |
| `describe[x]` | a map: `count`, `mean`, `stdev`, `min`, `25%`, `median`, `75%`, `max` |

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
| a matrix (`import linear`) | one per row | `0`, `1`, `2`, ... |
| any other list or set | one per element | `#` (counting from 0) and `value` |
| one map | one per entry | `key` and `value` |
| one assembled value | one per field | `field` and `value` |

```
import sql
import data

shelf = sql_open["books.db"]
show table[sql_query[shelf, "SELECT * FROM books"]] .
```

```
id  sku  title        price  rating  added
--  ---  -----------  -----  ------  ----------
 1  B1   Dune           950     4.5  2026-01-15
 2  B2   The Go Book   3000    none  2026-02-01
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
`... 480 more rows`. Change it like any variable:

```
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
