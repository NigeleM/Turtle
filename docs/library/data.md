# `data` — collections, tables and statistics

`import data`

Work across collections (process, keep, reduce), text tables and table files, and statistics. Use it to change, filter or add up lists and tables, and to read and write CSV or JSON files.


## Functions

`process`, `unique`, `group`, `table_column`, `table_row`, `keep`, `copy`, `range`, `reduce`, `sum`, `mean`, `median`, `mode`, `variance`, `stdev`, `pvariance`, `pstdev`, `percentile`, `covariance`, `correlation`, `zscores`, `describe`, `table`, `table_write`, `table_read`.

### `process[collection, function]`

Replaces every element (of a list or set) or every value (of a map) with what the function gives for it. collection   a list, set or map function     x give ...  (for a map: x give ..., or [key, value] give ...) As a sentence on its own it changes the collection itself; used as a value (assigned, or in an expression) it gives a new collection and leaves the original alone.

**Example**

```turtle
nums = list [1, 2, 3]
bigger = nums process x give x * 10
show bigger .
```

**Output**

```text
[ 10, 20, 30 ]
```

### `unique[collection]`

A new list with each value once, in the order they first appear (2 and 2.0 are one value).

**Example**

```turtle
show unique[list [3, 1, 3, 2]] .
```

**Output**

```text
[ 3, 1, 2 ]
```

### `group[collection, function]`

A map from each result of the function to the list of the items that gave that result, in their order in the collection.

**Example**

```turtle
show group[list [1, 2, 3, 4], x give x % 2 == 0] .
```

**Output**

```text
{ false: [ 1, 3 ], true: [ 2, 4 ] }
```

### `table_column[rows, name]`

The values of one column of a table, one per row, in row order.

**Example**

```turtle
rows = list [map ["title": "Emma", "copies": 12], map ["title": "Ida", "copies": 4]]
show table_column[rows, "copies"] .
```

**Output**

```text
[ 12, 4 ]
```

### `table_row[rows, n]`

Row n of a table, counted from 0, as the record itself.

**Example**

```turtle
rows = list [map ["title": "Emma", "copies": 12], map ["title": "Ida", "copies": 4]]
show table_row[rows, 1] .
```

**Output**

```text
{ "title": "Ida", "copies": 4 }
```

### `keep[collection, function]`

Keeps only the elements (or map entries) for which the function gives true: a filter.

**Example**

```turtle
nums = list [1, 2, 3, 4, 5]
big = nums keep x give x > 3
show big .
```

**Output**

```text
[ 4, 5 ]
```

### `copy[value, deep]`

A new list, set, map or assembled value with the same contents. value   what to copy deep    optional: true copies what's inside too (lists in a list, maps in a list, an assembled value's lists), so nothing is shared; false or left out shares what's inside

**Example**

```turtle
rows = list [1, 2]
outer = copy[rows]
full = copy[rows, true]
show full .
```

**Output**

```text
[ 1, 2 ]
```

### `range[from, to, step]`

The whole numbers from from up to, not including, to, as a list.

**Example**

```turtle
show range[5] .
show range[1, 5] .
show range[0, 10, 2] .
show range[5, 0, -1] .
```

**Output**

```text
[ 0, 1, 2, 3, 4 ]
[ 1, 2, 3, 4 ]
[ 0, 2, 4, 6, 8 ]
[ 5, 4, 3, 2, 1 ]
```

### `reduce[collection, start, function]`

Boils a collection down to one value: a running total that starts at start, and for each item (a map's values) becomes what the function gives for the total so far and the item.

**Example**

```turtle
nums = list [1, 2, 3]
total = nums reduce 0, [t, x] give t + x
show total .
```

**Output**

```text
6
```

### `sum[collection]`

Adds up a list's or set's numbers (a map's values): an integer if they all are, 0 when there are none.

**Example**

```turtle
prices = list [4, 9, 6]
total = sum[prices]
avg = mean[prices]
spread = stdev[prices]
show total, " ", avg, " ", spread .
```

**Output**

```text
19 6.333333333333333 2.516611478423583
```

### `mean[numbers, column]`

The average: the numbers added up, divided by how many there are. column   optional: with a list of rows, the column to use Gives back: a float.

**Example**

```turtle
prices = list [4, 9, 6]
avg = mean[prices]
show avg .
```

**Output**

```text
6.333333333333333
```

### `median[numbers, column]`

The middle number once sorted; with an even count, the average of the two in the middle.

**Example**

```turtle
mid = median[list [3, 1, 2]]
```

**Output**

```text
(prints nothing)
```

### `mode[values, column]`

The value that appears most often; a tie goes to the one seen first.

**Example**

```turtle
top = mode[list ["tea", "coffee", "tea"]]
```

**Output**

```text
(prints nothing)
```

### `variance[numbers, column]`

The sample variance: the average squared distance from the mean, dividing by n - 1, the usual choice for a sample.

**Example**

```turtle
prices = list [4, 9, 6]
prices = list [4, 9, 6]
v = variance[prices]
show v .
```

**Output**

```text
6.333333333333333
```

### `stdev[numbers, column]`

The sample standard deviation: the square root of variance.

**Example**

```turtle
prices = list [4, 9, 6]
spread = stdev[prices]
show spread .
```

**Output**

```text
2.516611478423583
```

### `pvariance[numbers, column]`

The population variance: like variance, dividing by n.

**Example**

```turtle
scores = list [2, 4, 4, 4, 5, 5, 7, 9]
v = pvariance[scores]
spread = pstdev[scores]
show v, " ", spread .
```

**Output**

```text
4.0 2.0
```

### `pstdev[numbers, column]`

The population standard deviation: the square root of pvariance.

**Example**

```turtle
scores = list [2, 4, 4, 4, 5, 5, 7, 9]
spread = pstdev[scores]
show spread .
```

**Output**

```text
2.0
```

### `percentile[numbers, column, percent]`

The number below which that percent of the numbers fall, between the two nearest when it lands between them (as numpy and a spreadsheet's PERCENTILE give). percentile[x, 50] is the median. percent   0 to 100

**Example**

```turtle
times = list [1, 2, 3, 4, 5, 6, 7, 8, 9, 10]
p90 = percentile[times, 90]
show p90 .
```

**Output**

```text
9.1
```

### `covariance[xs, ys]`

How two lists of numbers move together, the sample form (n - 1): positive when they rise together.

**Example**

```turtle
heights = list [150, 160, 170, 180]
weights = list [50, 58, 70, 81]
c = covariance[heights, weights]
show c .
```

**Output**

```text
175.0
```

### `correlation[xs, ys]`

Pearson's correlation, from -1 (one falls as the other rises) through 0 (no straight-line link) to 1 (they rise together).

**Example**

```turtle
ads = list [1, 2, 3, 4]
sales = list [2, 4, 5, 8]
r = correlation[ads, sales]
show r .
```

**Output**

```text
0.9811557810392121
```

### `zscores[numbers, column]`

Each number's distance from the mean, counted in (sample) standard deviations.

**Example**

```turtle
z = zscores[list [1, 2, 3]]
```

**Output**

```text
(prints nothing)
```

### `describe[numbers, column]`

A summary: count, mean, stdev (none with one number), min, 25%, median, 75%, max.

**Example**

```turtle
books = list [map ["price": 900], map ["price": 1500]]
show table[describe[books, "price"]] .
```

**Output**

```text
key                  value
------  ------------------
count                    2
mean                1200.0
stdev   424.26406871192853
min                    900
25%                 1050.0
median              1200.0
75%                 1350.0
max                   1500
```

### `table[rows, limit]`

Lays rows out as a text table, one row per line, columns lined up. rows    a list of maps (what sql_query gives), a list of assembled values, a list of lists, a plain list, a set, one map, or one assembled value limit   optional: at most this many rows for this call (else tablerows) Gives back: text, so show it: show table[rows] .

**Example**

```turtle
import sql
db = sql_create["shop.db"]
sql_run[db, "CREATE TABLE books (title TEXT, price INTEGER)", list []]
show table[sql_query[db, "SELECT * FROM books"]] .
sql_close[db]
```

**Output**

```text
no rows
```

### `table_write[path, rows, options]`

Saves rows to a file, in the layout table[...] shows.

**Example**

```turtle
orders = list [map ["item": "pen", "qty": 3]]
show table_write["orders.csv", orders] .
codes = list [map ["code": "007"]]
show table_write["codes.csv", codes, map ["quote": "text"]] .
```

**Output**

```text
1
1
```

### `table_read[path [, types]]`

Reads a .csv, .tsv or .json file.

**Example**

```turtle
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

**Output**

```text
item  qty
----  ---
pen     3
```
