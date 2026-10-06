# Standard library

Everything built into the language beyond core syntax: data structure
methods, file I/O, `sys`, and `import`. See [`reference.md`](reference.md)
for the statement/expression grammar these use.


**From the terminal:** `turtle doc` lists every library function;
`turtle doc sql` shows one library in full, `turtle doc sql_update`
one function: what each argument is, what it gives back, and an example.

## Data structures

Three composite types: `list` (ordered, duplicates allowed), `set`
(insertion-ordered, deduplicated), `map` (insertion-ordered key → value).

```
nums = list [3, 1, 2]
letters = set ["a", "b", "b"]
ages = map ["Alice": 30, "Bob": 25]
```

`show`ing any of them prints:
- list: `[ 3, 1, 2 ]`
- set: `{ "a", "b" }`
- map: `{ "Alice": 30, "Bob": 25 }` (insertion order, always — this is
  deterministic, unlike the legacy interpreter's randomized map iteration)

### Statement-form operations

All require a trailing `.`. Target must be a variable name (not an
arbitrary expression).

| Statement | Applies to | Effect |
|---|---|---|
| `add <expr> to <target> .` | list, set | Appends. Sets silently skip the add if the value's already present. |
| `remove <expr> from <target> .` | list, set | Removes the first matching element. No-op if not found. |
| `delete <expr> from <target> .` | map | Deletes the key. **Fatal error if the key doesn't exist.** |
| `sort <target> .` | list, set | Sorts in place: numerically if all elements are numbers, lexicographically (by display string) otherwise. |
| `reverse <target> .` | list, set | Reverses in place. |
| `insert <expr> to <target> at <expr> .` | list | Inserts at the given index. Fatal if index is out of range (`0..length` inclusive). |
| `put <expr> to <target> at <expr> .` | list | Replaces the item at the given index (`0..length-1`); the length stays the same. |
| `length of <expr> .` | list, set, map, string | Prints the count (map: number of keys; string: character count, in Unicode code points, not bytes). |
| `min of <expr> .` | list, set, map | Prints the smallest element (list/set) or smallest **key** (map — matches legacy behavior; map values aren't compared). |
| `max of <expr> .` | list, set, map | Same as `min of`, but largest. |

`length of`/`min of`/`max of` are also ordinary expressions usable anywhere
(`x = 1 + length of nums`) — they only print when they're the entire
statement. They apply to the value right after `of`, so
`length of nums == 0` compares the length with 0. For the length of a
computed value, store it first: `ab = a + b` then `length of ab`.

### Method-call form

```
<result> is <receiver> at <method> [<arg> {, <arg>}] .
```

#### list

| Method | Args | Returns |
|---|---|---|
| `add` | value | the list (mutated) |
| `len` / `length` | — | element count (Integer) |
| `isEmpty` | — | Boolean: `true` when there are no elements |
| `toString` | — | display string |
| `clear` | — | the list, now empty |
| `count` | value | number of matching elements (Integer) |
| `index` | value | first matching index, or `-1` |
| `sort` | — | the list, sorted |
| `remove` | value | the list, first match removed |
| `reverse` | — | the list, reversed |
| `pop` | — | the **removed last element** (fatal if empty) |
| `contains` / `find` | value | Boolean, whether present: `nums at contains[3]` |
| `insert` | value, index | the list, with value inserted |
| `put` | index, value | the list, with the item at index replaced: `nums at put[2, 99]` |
| `get` | index | the element at that index, from `0`; fatal if out of range (including negative indexes; use `slice` to count from the end) |
| `slice` | start [, end] | a new `list`, the elements `[start, end)`; same negative-index/clamping rules as string's `slice` below |

#### set

All list methods above except `count`/`find`/`copy` behave the same way
(see note below), plus:

| Method | Args | Returns |
|---|---|---|
| `union` | other set | new set: everything in either |
| `intersection` | other set | new set: only what's in both |
| `difference` | other set | new set: in this set but not the other |
| `subset` | other set | Boolean: is this set ⊆ the other |
| `superset` | other set | Boolean: is this set ⊇ the other |

(In practice `count`/`find`/`insert`/`get` etc. all work on sets too — the
implementation shares logic between list and set rather than restricting
methods per type.)

#### map

| Method | Args | Returns |
|---|---|---|
| `get` | key | the value (fatal if key not found) |
| `len` / `length` | — | how many entries (Integer), like `length of m` |
| `contains` | key | Boolean, whether the map has that key |
| `getValues` | — | a list of values, insertion order |
| `getKeys` | — | a list of the keys, insertion order, each with its own type |
| `isEmpty` | — | Boolean: `true` when the map has no entries |
| `add` | key, value | the map, with the entry set |
| `delete` | key | the map, with the entry removed (fatal if not found) |
| `invert` | — | new map: values become keys, keys become values |
| `toString` | — | display string |

#### string

All indices are Unicode code points (runes), not bytes — consistent with
`length of` above and with `change ...to ascii/char` (see
[`reference.md`](reference.md#type-conversion)).

| Method | Args | Returns |
|---|---|---|
| `len` / `length` | — | how many characters (Integer), like `length of s` |
| `upper` / `lower` | — | case-converted string |
| `isEmpty` | — | Boolean: `true` for `""` |
| `trim` | — | leading/trailing whitespace stripped |
| `get` | index | the one-character string at that index, from `0`; fatal if out of range (including negative) |
| `slice` | start [, end] | substring `[start, end)`; see below |
| `split` | separator | a `list` of substrings |
| `contains` | substring | Boolean |
| `indexOf` | substring | first matching index, or `-1` |
| `replace` | old, new | new string, all occurrences of `old` replaced with `new` |
| `isNumber` | — | Boolean: would `change ... to integer/float` succeed on this string |
| `toString` | — | itself |

`isNumber` exists so you can validate untrusted input (from `?`) before
converting it, instead of letting a bad `change` crash the program:

```
raw = ? "Enter a number: "
ok is raw at isNumber .
if ] ok [
    n = change raw to float
else ]
    show "that's not a number" .
if [end]
```

`slice` is Python-style but without a step: a negative bound counts from
the end (`-1` is the last character), an omitted `end` defaults to the
string's length, and out-of-range bounds clamp instead of erroring
(unlike `get`, which is a strict single-index lookup):

```
s = "Hello, World"
first5 is s at slice 0, 5 .    // "Hello"
last5 is s at slice -5 .       // "World"
tail is s at slice 7 .         // "World" (index 7 to the end)
```

#### number

Every method here requires `import math` first (see
[`reference.md`](reference.md#modules)) — calling one before that is a
fatal error naming exactly which import is missing.

| Method | Args | Returns |
|---|---|---|
| `sqrt` | — | square root, as a `float` (fatal on a negative receiver) |
| `abs` | — | absolute value, same type as the receiver |
| `round` | — | nearest integer (half rounds away from zero) |
| `floor` | — | next integer toward negative infinity |
| `ceil` | — | next integer toward positive infinity |
| `pow` | exponent | receiver raised to exponent: an `integer` when both are whole numbers and the exponent isn't negative, otherwise a `float` |
| `random` | — | a random integer in `[0, receiver)`; the receiver is the exclusive upper bound and must be a positive integer |

```
import math

r is 16 at sqrt .        // 4.0 (sqrt always gives a float)
p is 2 at pow 10 .       // 1024
n is 10 at random .      // some integer in [0, 10)
```

`%` (modulo) is a core operator, not a method, so it needs no import — see
[`reference.md`](reference.md#expressions).

### Builtin functions: `time`

`import time` gives the clock, dates, date arithmetic, and waiting:

| Function | Args | Returns |
|---|---|---|
| `now[]` | — | milliseconds since the Unix epoch, as an Integer |
| `sleep[amount [, unit]]` | amount, optional `"seconds"` (default) or `"ms"` | pauses that long; returns `none` |
| `today[]` | — | the date and time now, in local time |
| `today_utc[]` | — | the same moment, shown in UTC |
| `make_date[y, m, d]` or `[y, m, d, h, mi, s]` | whole numbers | that local date and time; one that doesn't exist (Feb 30) is a `date` error |
| `to_date[text]` | `"YYYY-MM-DD"`, optionally with ` hh:mm` or ` hh:mm:ss`, or ISO 8601 (`2026-10-03T14:05:00Z`) | the date; other text is a `date` error |
| `add_time[date, amount, unit]` | date, whole number, unit | a new date; a negative amount goes back |
| `time_between[a, b, unit]` | two dates, unit | how many whole units from `a` to `b` (negative if `b` is earlier) |
| `format_date[date, pattern]` | date, pattern text | the date written with the pattern (below) |
| `wait_until[date]` | date | sleeps until then (at once if it's past); returns `none` |
| `every[amount, unit, job]` | whole number, unit, a function with no parameters | runs `job` now and then on a repeat, until `job` returns `false` |

**Units:** `"seconds"`, `"minutes"`, `"hours"`, `"days"`, `"weeks"`,
`"months"`, `"years"` (or the singular, `"day"`).

**A date** shows as `2026-10-03 14:05:00`, whole seconds, in its own
clock (local, or UTC for `today_utc[]` and `...Z` text). Read its parts
with `of`: `year`, `month`, `day`, `hour`, `minute`, `second` (whole
numbers) and `weekday` (`"Saturday"`). Parts are read-only; make a new date
with `add_time` or `make_date`. Dates compare with `< > <= >= == !=` (the
same moment is equal whichever clock shows it), can be map keys and set
elements, join text with `+` or `{d}`, and are written to JSON as text.

```
import time

d = today[]
show d .                                     // 2026-10-03 14:05:00
show weekday of d .                          // Saturday

due = add_time[d, 30, "days"]                // 30 days from now
last_week = add_time[d, -7, "days"]
left = time_between[d, due, "days"]          // 30
show format_date[due, "Weekday, Month D YYYY"] .   // Monday, November 2 2026

if ] today[] > due [
    show "overdue" .
if [end]
```

**Months and years stay in the month:** January 31 + 1 month is
February 28 (29 in a leap year), not March 3; February 29 + 1 year is
February 28. `time_between` counts months the same way, so January 31 to
February 28 is 1 month.

**Days keep the clock time** across daylight-saving changes: noon + 1 day
is noon the next day, even when that day has 23 or 25 hours. `"hours"`
is exact: noon + 24 hours can be 11:00 or 13:00 on those days.

**`format_date` patterns:** `YYYY` year, `MM`/`M` month (`03`/`3`),
`DD`/`D` day, `hh` hour 00-23, `mm` minute, `ss` second, `Month`
(`March`), `Mon` (`Mar`), `Weekday` (`Thursday`), `Wkd` (`Thu`). Anything
else is copied as is: `format_date[d, "DD/MM/YYYY hh:mm"]` → `05/03/2026 14:07`.

**Automation.** `wait_until` and `every` run inside your script, so the
script has to keep running (in a terminal, or as a service). `every`
schedules each run from when it started, so a slow job doesn't push later
runs back. Return `false` from the job to stop; an error in the job stops
the program unless the job handles it with `safe`.

```
import time
import system

def backup[]
    sys cp data.db backups/
    warn "backed up at ", today[] .
def [end]

wait_until[add_time[today[], 1, "hours"]]    // run once, an hour from now
backup[]

every[7, "days", backup]                     // then every 7 days, forever
```

`sleep` examples:

```
sleep[1]            // 1 second
sleep[0.25]         // a quarter second
sleep[250, "ms"]    // 250 milliseconds
```

If you've already defined your own top-level function with one of these
names (`now`, `today`, ...), it always wins — the builtin only kicks in when no user function
of that name exists.

Plain `<result> is <receiver> .` (no `at`) is just assignment/aliasing —
`<result>` becomes another reference to the same underlying value.

## Data library

`import data` (or `import data [process, keep, copy, table, table_read,
table_write]`) provides six functions. `process` and `keep` change the
collection **in place** and also return it. Use `copy` first to keep the
original. `table`, `table_read` and `table_write` are described
[below](#tables).

| Function | Args | Effect |
|---|---|---|
| `process` | collection, function | replaces each element (list/set) or each value (map) with the function's result; a set is deduplicated afterwards |
| `keep` | collection, function | keeps only the elements (list/set) or entries (map) for which the function gives a truthy result: a filter |
| `copy` | collection | a new list/set/map with the same elements, or a new assembled value with the same fields |

For a map, the function takes the value (`x gives ...`), or the key and the
value (`[k, v] gives ...`).

```
import data

nums = list [5, 3, 8, 1]
nums process x gives x + 1 .        // [ 6, 4, 9, 2 ]
nums keep x gives x > 3 .           // [ 6, 4, 9 ]

words = list ["hey", "do"]
words process x gives x at upper .  // [ "HEY", "DO" ]

nums process double .               // any function value works

ages = map ["Alice": 30, "Bob": 25]
labels = copy[ages]
labels process [name, age] gives name + " is " + age .
ages keep [name, age] gives age > 26 .   // { "Alice": 30 }

nums process [x] gives              // block form for longer logic
    if ] x > 5 [
        return x * 2
    if [end]
    return x
gives [end]
```

These are ordinary functions, so you can write your own in a `.trt` library
and call them the same sentence style. See
[`reference.md`](reference.md#sentence-style-calls).

### Tables

`show` prints a value on one line. `table[x]` returns the same value laid
out as a text table, one row per line, so `show table[x] .` is easy to
read. It works on:

| Value | Rows | Columns |
|---|---|---|
| list of maps (what `sql_query` returns) | one per map | one per key |
| list of assembled values | one per value | one per field |
| list of lists | one per inner list | `0`, `1`, `2`, ... |
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
- Columns of numbers are right-aligned, everything else left-aligned.
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

### Table files

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
show rows at get[0] .                   // { "item": "pen", "qty": "3", "price": "1.5" }
```

- `table_write` returns how many rows it wrote and replaces the file if
  it's there. The columns are the ones `table[x]` shows (fields, keys,
  `#` and `value` for a plain list, ...).
- **`.json` keeps kinds**: numbers read back as numbers, `true`/`false`
  as booleans, `null` as `none`, and a list or object inside a row as a
  list or map. Dates are written as text. A row missing a name has
  `none` there; the columns are every name any row has, in the order
  they first appear.
- From `.csv` and `.tsv`, values read back are text, as in the file
  (`"3"`), unless you give [column types](#column-types). An empty cell
  is `none`, and `none` is written as an empty cell, so files round-trip.
- A value with the separator, a quote or a line break is put in quotes,
  with quotes doubled (`"say ""hi"""`), the standard rule (RFC 4180).
  Blank lines are skipped, and a byte order mark at the start (Excel
  writes one) is ignored.
- A line with fewer values than the header gets `none` for the rest; one
  with more, or broken quotes, is an error of kind `csv`. A missing file
  is kind `file`.

### Column types

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
- **Columns the map leaves out are text.** A column the map names that
  the file doesn't have is an error of kind `name` (usually a typo).
- **`"primary_key": "column"`** names the key. `table_read` checks that
  every row has one and no two are the same (after converting, so `1`
  and `01` are the same integer), an error of kind `key` otherwise.
- An empty cell stays `none`. A cell that isn't its type is an error
  naming the row and column: kind `number` for numbers, `date` for
  dates, `type` for booleans.
- From `.json`, values that already have the right kind stay as they
  are; a number in a `text` column becomes its text, and a list or
  object in a `text` column becomes its JSON text.

For `sql_load`, see [Files: CSV in and out](#files-csv-in-and-out).

## Random library

`import random` makes random values of any shape. Describe the value in
words after `random`; each time the line runs, it gives a new one.

```
import random
assemble Order [item, qty, price]

die = random integer from 1 to 6
nums = random list of 5 integers from 0 to 9       // [ 9, 5, 1, 7, 6 ]
price = random float from 0.5 to 99.99 rounded to 2 // 38.61
code = random string of 8                            // "AGhwhRHU"
pin = random digits of 4                           // "5875": a string, so "0427" keeps its 0
id = random string of 6 from "ABCDEF0123456789"      // "6B3838"
day = random date from "2026-01-01" to "2026-12-31"
grid = random list of 3 lists of 3 integers from 1 to 9
groups = random set of 4 lists of 2 strings of 3
ages = random map of 3 string of 2 to integer from 0 to 99
order = random Order [string of 5, integer from 1 to 10, float rounded to 2]
orders = random list of 20 Order [string, integer, float]
```

| Kind | Gives | Options |
|---|---|---|
| `integer` | a whole number, -1000 to 1000 | `from A to B` (both included) |
| `float` | a decimal, 0 up to 1 | `from A to B`, `rounded to N` (places) |
| `string` | 1 to 10 letters, a–z and A–Z | `of N`, `of N to M` (length), `from "chars"` (only these) |
| `digits of N` | a string of N digits (`"0427"`) | `of N to M` |
| `digit`, `letter` | one digit or letter, as a string | |
| `boolean` | `true` or `false` | |
| `date` | a day, 2000-01-01 to 2030-12-31 | `from A to B`: dates or strings like `"2026-01-31"` |
| `time` | a date with a time of day, to the second | `from A to B` |
| `list of <kind>` | a list | a count: `list of 5 integers`, `list of 2 to 8 integers`, `list of n integers` |
| `set of <kind>` | a set; its items are all different | a count, as for lists |
| `map of <kind> to <kind>` | a map; its keys are all different | a count: `map of 3 string to integer` |
| `Order [<kind>, ...]` | an assembled value, one kind per field, in order | |

- **Plurals** read naturally and mean the same: `integers`, `floats`,
  `strings`, `letters`, `booleans`, `dates`, `lists`, `sets`, `maps`
  (`digits` without `of` is single digits: `list of 3 digits`).
- **A count comes before the kind**, and is a whole number or a
  variable's name. Without one, a list, set or map has 0 to 10 items.
  Limits (`from`, `of`, `rounded`) come after the kind.
- **Kinds nest** any way: `list of 3 lists of 3 integers`,
  `set of 4 lists of string`, `map of string to list of Order [string, float]`.
- **Where the sentence ends**: arithmetic belongs to a limit (`to n + 1`),
  but a comparison applies to the random value:
  `random integer from 1 to 6 == 6` is true one time in six. A method
  works on the value right before it (`to 9 at abs` is about the 9), so
  to call one on the random value, name it first: `nums = random list
  of 5 integers`, then `nums at length`.
- A set or map that can't get enough different values
  (`set of 5 integers from 1 to 3`) is a `math` error, as is a range
  whose `from` is more than its `to`.
- `random` is a sentence word only in a file that has `import random`;
  elsewhere it's an ordinary name, and `n at random` (math) works as
  before.

**Choosing from your own values** (`pick` and `chance` are written in
Turtle, in `evaluator/lib/random.trt`, on the `random` sentence; `shuffle`
and `sample`, which go through every item, are in Go for speed):

| Function | Gives |
|---|---|
| `pick[x]` | one item of a list or set, one key of a map, one character of a string |
| `shuffle[x]` | a new list in random order (a string, for a string) |
| `sample[x, n]` | a list of `n` different items |
| `chance[p]` | `true` with probability `p` (0 to 1) |

```
color = pick[list ["red", "green", "blue"]]
hand = sample[shuffle[deck], 5]
if ] chance[0.3] [
    show "rain" .
if [end]
```

**The same values every run.** `import random` also makes the variable
`seed`, `none`: different values each run. Set it to a whole number and
every random value from there on is the same each time the program runs,
which is how to repeat a run exactly (a test that failed on random
data, say). Setting it again, even to the same number, starts the values
over; `seed = none` goes back to different values each run.

```
seed = 42
a = random list of 3 integers
seed = 42
b = random list of 3 integers      // the same as a
```

## Test library

`import test` adds three statements, `check` (one fact), `verify` (a
rule for every item) and `validate` (a rule on many random inputs), and
the settings `turtle test` reads. Everything is in
[`testing.md`](testing.md).

```
import test

def test_evens[]
    check evens[list [1, 2, 3, 4]] == list [2, 4] .
    verify evens[nums] each x gives x % 2 == 0 .
    validate evens[nums] with nums as list of integer
        that result each x gives x % 2 == 0 .
def [end]
```

```sh
turtle test
```

## Sort library

`import sort` puts things in order. Every function gives back a **new**
list (a new map, for a map) and leaves the original unchanged. The
`sort nums .` statement still sorts a list in place.

| Function | Gives back |
|---|---|
| `min_sort[x [, key [, how]]]` | `x` in order, smallest first; equal items keep their order |
| `max_sort[x [, key [, how]]]` | the same, largest first |
| `is_sorted[x [, key]]` | `true` when `x` is already smallest first |
| `reverse_list[x]` | the items (a map: its entries) in the opposite order |
| `bubble_sort`, `insertion_sort`, `selection_sort`, `merge_sort`, `quick_sort`, `heap_sort`, `shell_sort`, `counting_sort`, `radix_sort` `[x [, key]]` | `min_sort`'s answer, by that algorithm (for learning and comparing; `counting_sort` and `radix_sort` need whole-number keys) |

**The key** says what to order by:

| Key | Orders by | Example |
|---|---|---|
| left out, or `none` | the items themselves (a map: its keys) | `min_sort[nums]` |
| a function | what it gives for each item | `min_sort[books, b gives price of b]` |
| text | that map key, or assembled field | `min_sort[books, "price"]` |
| an integer | that position in a list of lists | `min_sort[pairs, 1]` |
| a list of those | the first, then the next for ties | `min_sort[books, list ["author", "price"]]` |

**How much** (`min_sort` and `max_sort` only): `"first"` gives just the
first item, or `none` when there's nothing; a number `n` gives the first
`n`. So `max_sort[books, "price", "first"]` is the most expensive book,
like Python's `max(books, key=...)`.

**Maps** sort by entry and give back a new map in that order. With no
key, by the map's keys; a text or integer key picks from each value; a
function gets the value (`v gives ...`) or the key and value
(`[k, v] gives ...`). `"first"` gives a key.

```
import sort

ages = map ["Cy": 41, "Ann": 30, "Bo": 25]
show min_sort[ages] .                      // { "Ann": 30, "Bo": 25, "Cy": 41 }
show max_sort[ages, a gives a] .           // { "Cy": 41, "Ann": 30, "Bo": 25 }
show min_sort[ages, a gives a, "first"] .  // Bo
```

**One order for every value**, so mixed and nested collections sort with
or without a key: `none`, then `true`/`false` (false first), numbers
(`1` equals `1.0`), text (character by character), dates, lists and
sets (item by item, shorter first), assembled values (type name, then
field by field), maps (entry by entry). A function value can't be put
in order (an error of kind `type`).

## Search library

`import search` finds things in lists, sets, maps and text, and in what's
inside them. Keys work as in the sort library.

| Function | Gives back |
|---|---|
| `find_first[x, test]` | the first item `test` says yes to (a map: its key), or `none` |
| `find_last[x, test]` | the last one, or `none` |
| `find_all[x, test]` | every one, a new list (a map: a new map of those entries) |
| `find_index[x, test]` | the first one's position, from 0, or `-1` |
| `count_where[x, test]` | how many |
| `find_key[map, value]` | the first key holding `value`, or `none` |
| `linear_search[x, value [, key]]` | the position of the first item equal to `value`, or `-1`; works on anything |
| `binary_search`, `jump_search`, `exponential_search`, `interpolation_search`, `ternary_search` `[sorted, value [, key]]` | the same answer, faster, on a collection sorted smallest first by the same key |
| `insert_position[sorted, value [, key]]` | where `value` would go to keep it sorted |

`test` is a function giving `true` or `false`: `b gives price of b < 1000`.

```
import sort
import search

assemble Book [title, price]
books = list [Book["Dune", 950], Book["Emma", 700], Book["Kindred", 950]]

cheap = find_all[books, b gives price of b < 900]          // [ Book Emma ]
by_price = min_sort[books, "price"]
at = binary_search[by_price, 950, "price"]                  // 1: the first 950
pos = insert_position[by_price, 800, "price"]               // 1
```

`binary_search` takes about 20 steps for a million items where
`linear_search` may take a million, but it needs the collection sorted
first, by the same key.

## JSON library

`import json` (or `import json [load, json_get]`):

| Function | Args | Returns |
|---|---|---|
| `load[text]` | JSON text | the Turtle value |
| `json_text[value]` | any JSON-able value | JSON text on one line |
| `json_read[path]` | file path | the file's JSON as a Turtle value |
| `json_write[path, value]` | file path, value | writes the value as indented JSON; returns `none` |
| `json_get[value, step, ...]` | a value, then keys and indexes | the value at the end of the path, or `none` if any step is missing |

```
import json

cfg = json_read["config.json"]
port = json_get[cfg, "server", "port"]
if ] port == none [
    port = 8080
if [end]

user = load['{"name": "Ann", "tags": ["admin"]}']
show user at get["name"] .               // Ann
tags = user at get["tags"]
add "editor" to tags .                   // the same list, so user changes too
json_write["user.json", user]
show json_text[user] .                   // {"name":"Ann","tags":["admin","editor"]}
```

**JSON to Turtle:** objects become maps (keys stay in the file's order),
arrays become lists, whole numbers become integers (`3`), other numbers
floats (`2.5`, `1e3` → `1000.0`), `true`/`false` booleans, `null` `none`.

**Turtle to JSON:** maps become objects and lists arrays; sets become
arrays; an assembled value becomes an object of its fields
(`Order["pen", 3]` → `{"item": "pen", "qty": 3}`); `none` becomes `null`.
Number map keys become text keys (`1` → `"1"`), since JSON keys are
always text; any other key, a function, or a value that contains itself
is a `type` error.

**`json_get`** takes map keys, assembled-value field names, and list
indexes (from 0), one step each. It never fails on a missing step; it
returns `none`, so you can check instead of using `safe`.

**Errors.** Bad JSON is kind `json` and says where:
`json_read config.json: invalid JSON at line 6, column 1: invalid
character '}' looking for beginning of value`. A missing file is kind
`file`.

```
safe
    cfg = json_read["config.json"]
handle [file, json] e .
    warn "using defaults: ", e .
    cfg = map []
safe [end]
```

JSON typed into Turtle code reads best in single quotes, where `"` needs
no escape and braces before a quote are plain: `'{"a": [1, 2]}'`.

## HTTP library

`import http` makes web requests, using only Go's standard library:

| Function | Args | Returns |
|---|---|---|
| `http_get[url]` | web address | the response body as text |
| `http_post[url, body]` | web address, body | the response body as text |
| `http_request[method, url [, body [, headers]]]` | `"GET"`, `"PUT"`, ...; address; optional body and headers map | a map with `status` (integer), `body` (text), `headers` (map, lowercase names) |

**Bodies:** text is sent as is (`text/plain`); a map, list, set or
assembled value is sent as JSON (`application/json`); `none` sends no body.

**Errors** are kind `http`. `http_get` and `http_post` fail on a network
problem or a 4xx/5xx status: `http_get https://api.example.com/x: 404 Not
Found: no such page`. `http_request` only fails on a network problem, and
hands back any status for you to check. A request with no answer in 30
seconds fails.

```
import http
import json

safe
    users = load[http_get["https://api.example.com/users"]]   // a list of maps
handle [http, json] e .
    warn "couldn't fetch users: ", e .
    users = list []
safe [end]

reply = http_post["https://api.example.com/users", map ["name": "Ann"]]

r = http_request["DELETE", "https://api.example.com/users/7", none, map ["Authorization": "Bearer {token}"]]
if ] r at get["status"] != 204 [
    show "delete failed: ", r at get["body"] .
if [end]
```

## SQL library

`import sql` works with SQLite files and with PostgreSQL and MySQL (or
MariaDB) servers, through the same functions; the address given to
`sql_open` picks which. Every driver is written from scratch for Turtle
(no third-party code).

For SQLite, that's the file format, B-trees, the journal that makes
changes crash-safe, file locking, and the SQL engine. The files are
ordinary SQLite files: the `sqlite3` tool, DB Browser, Python and every
other SQLite program can open them, and Turtle can open theirs. For the
servers, it's their network protocols, logins and TLS; the server runs
the SQL (see [Servers](#servers-postgresql-and-mysql)).

| Function | Args | Returns |
|---|---|---|
| `sql_open[address]` | a SQLite file path (or `"sqlite:path"`), or a server address `postgres://...` / `mysql://...` | a database value |
| `sql_create[path]` | a path where no file is yet | a new, empty SQLite database |
| `sql_query[db, query [, values]]` | database, a `SELECT` (or a change with `RETURNING`), and a list of values for its `?` placeholders | a list of maps, one per row, column name → value |
| `sql_run[db, statement [, values]]` | database, a statement that changes something, and its `?` values | how many rows it inserted, updated or deleted |
| `sql_tables[db]` | database | a list of its table names |
| `sql_save`, `sql_load`, `sql_update`, `sql_delete`, `sql_upsert` | see [Files](#files-csv-in-and-out) | how many records |
| `sql_close[db]` | database | `none`; closing twice is fine; an unfinished transaction is rolled back |

```
import sql
import data

db = sql_create["shop.db"]
sql_run[db, "CREATE TABLE books (id INTEGER PRIMARY KEY, title TEXT NOT NULL, price INTEGER)"]
sql_run[db, "INSERT INTO books (title, price) VALUES (?, ?)", list ["Dune", 950]]
n = sql_run[db, "UPDATE books SET price = price + 50 WHERE price < ?", list [1000]]

rows = sql_query[db, "SELECT title, price FROM books ORDER BY price"]
show table[rows] .
sql_close[db]
```

**Values.** `NULL` is `none`; integers, reals and text come back as
integers, floats and text; a BLOB comes back as text. `?` values can be
integers, floats, text, booleans (1/0), `none` (`NULL`) and dates (as
`2026-10-03 14:05:00` text). Always pass values as `?` placeholders rather
than building the statement with `+` or `{ }`, so a value can never change
the statement. Text going into an `INTEGER` or `REAL` column becomes a
number when it looks like one (`"950"` → `950`), as in SQLite; that's
what makes CSV imports work.

**Several statements at once.** `sql_run` runs statements separated by
`;` in order, when no `?` values are given:
`sql_run[db, "CREATE TABLE a (x); CREATE TABLE b (y)"]`.

**Looking at results.** `show rows .` prints them on one line; with
`import data`, `show table[rows] .` prints one row per line (see
[Tables](#tables)). When a column name appears twice (`SELECT *` over a
join), the second one is named `id:1`, the third `id:2`.

### Changing data

Everything SQLite's own SQL can change:

- `INSERT INTO t (cols) VALUES (...), (...)`, `INSERT ... SELECT ...`,
  `INSERT ... DEFAULT VALUES`, `REPLACE INTO`, and `INSERT OR IGNORE` /
  `OR REPLACE` / `OR FAIL` / `OR ABORT` / `OR ROLLBACK`.
- Upsert: `INSERT ... ON CONFLICT (col) DO UPDATE SET qty = qty + excluded.qty`
  (with an optional `WHERE`), or `ON CONFLICT DO NOTHING`.
- `UPDATE t SET a = ..., b = ... WHERE ...` (`OR IGNORE` / `OR REPLACE` too)
  and `DELETE FROM t WHERE ...`.
- `RETURNING` after any of them gives the changed rows back; run such a
  statement with `sql_query`:
  `rows = sql_query[db, "INSERT INTO t (v) VALUES ('a') RETURNING id"]`.
- `WITH ...` before `INSERT`, `UPDATE` or `DELETE`.
- `UPDATE ... FROM` to change rows using another table:
  `UPDATE stock SET qty = qty + d.n FROM delivery AS d WHERE d.sku = stock.sku`.

**Constraints** are enforced, and a statement that breaks one changes
nothing at all (an error of kind `sql`): `NOT NULL`, `UNIQUE`,
`PRIMARY KEY`, `CHECK`, `DEFAULT` (including `CURRENT_TIMESTAMP`),
`COLLATE NOCASE`, `INTEGER PRIMARY KEY` (the row's id, filled in for you),
`AUTOINCREMENT`, and `STRICT` tables. Foreign keys (`REFERENCES`, with
`ON DELETE CASCADE / SET NULL / SET DEFAULT / RESTRICT`) are enforced
after `PRAGMA foreign_keys = ON`; as in SQLite, they're off until then.

```
safe
    sql_run[db, "INSERT INTO books (title) VALUES (NULL)"]
handle [sql] e .
    show message of e .        // sql_run: NOT NULL constraint failed: books.title
safe [end]
```

### Transactions

Each statement is saved on its own, and is all-or-nothing. To save
several together, or to undo them:

```
sql_run[db, "BEGIN"]
sql_run[db, "UPDATE accounts SET balance = balance - 30 WHERE id = 1"]
sql_run[db, "UPDATE accounts SET balance = balance + 30 WHERE id = 2"]
sql_run[db, "COMMIT"]          // or ROLLBACK to undo both
```

A statement that fails inside `BEGIN ... COMMIT` is undone by itself and
the transaction stays open: the program decides whether to `COMMIT` what
worked or `ROLLBACK` everything. A program that ends (or crashes) before
`COMMIT` leaves the file as it was.

Saving waits until the disk has the data, so many separate statements
are slow (a few milliseconds each); the same statements inside one
`BEGIN ... COMMIT` are written together, thousands per second.

**Crash safety.** Before changing the file, Turtle writes the original
pages to a journal (`shop.db-journal`, SQLite's own format) and makes it
durable. If the program or the computer stops halfway, the next open
(by Turtle or any SQLite program) puts the original pages back. Turtle
also repairs files that another SQLite program left half-written.

**Other programs.** Turtle locks the file the way SQLite does, so it can
be used at the same time as `sqlite3` or another program: readers share,
one writer at a time. A program that has to wait more than 5 seconds gets
an error of kind `sql` ("database is locked").

### Changing the schema

- `CREATE TABLE` (with `IF NOT EXISTS`), `CREATE TABLE ... AS SELECT`.
- `CREATE [UNIQUE] INDEX` (on columns, with `COLLATE` and `DESC`, and
  partial indexes with `WHERE`).
- `CREATE VIEW`.
- `ALTER TABLE t RENAME TO new` (views that use it follow), `ADD COLUMN`,
  `RENAME COLUMN a TO b`, `DROP COLUMN`.
- `DROP TABLE`, `DROP INDEX`, `DROP VIEW` (with `IF EXISTS`).

Indexes are kept up to date by every change, and used to find rows fast:
`WHERE id = ?`, `WHERE sku = ?` on an indexed column, and joins on an id
or an indexed column look rows up instead of reading the whole table.

### Queries

`SELECT` with `DISTINCT`, expressions and `AS` names, `*` and `t.*`;
`FROM` tables, views, subqueries and `WITH` tables; `JOIN`, `LEFT JOIN`,
`RIGHT JOIN`, `FULL JOIN`, `CROSS JOIN`, `NATURAL JOIN`, with `ON` or
`USING`; `WHERE`; `GROUP BY` and `HAVING`; `UNION`, `UNION ALL`,
`INTERSECT`, `EXCEPT`; `ORDER BY` (columns, expressions, result names or
positions, `ASC`/`DESC`, `NULLS FIRST`/`NULLS LAST`, `COLLATE`);
`LIMIT`/`OFFSET`; `WITH` and `WITH RECURSIVE`; `VALUES (...)`.

Subqueries work as values `(SELECT max(price) FROM books)`, in
`IN (SELECT ...)` and `EXISTS (SELECT ...)`, as tables in `FROM`, and may
use columns of the query around them.

Operators: `= == != <> < > <= >= AND OR NOT + - * / % || & | << >> ~`,
`IS [NOT]`, `IS [NOT] DISTINCT FROM`, `IS [NOT] NULL`, `[NOT] IN`,
`[NOT] BETWEEN`, `[NOT] LIKE ... [ESCAPE]`, `[NOT] GLOB`, `CASE`, `CAST`,
and `COLLATE` (`BINARY`, `NOCASE`, `RTRIM`).

Aggregates (with `DISTINCT`, and `FILTER (WHERE ...)`): `count sum total
avg min max group_concat string_agg`.

Functions:

| Kind | Functions |
|---|---|
| text | `length lower upper substr trim ltrim rtrim replace instr printf format quote char unicode hex unhex concat concat_ws octet_length like glob` |
| numbers | `abs round sign min max random ceil floor trunc sqrt pow exp ln log log2 log10 mod pi sin cos tan asin acos atan atan2 sinh cosh tanh degrees radians` |
| dates | `date time datetime julianday unixepoch strftime`, with modifiers such as `'+1 month'`, `'start of month'`, `'weekday 0'`, `'unixepoch'`, `'localtime'`; `CURRENT_DATE`, `CURRENT_TIME`, `CURRENT_TIMESTAMP` |
| values | `coalesce ifnull nullif iif typeof zeroblob randomblob likely unlikely` |
| the database | `last_insert_rowid changes total_changes sqlite_version` |

Answers match real SQLite, including its type rules (a `TEXT` column
holding `'5'` equals `5`) and its date arithmetic (`'2026-01-31'` plus one
month is `2026-03-03`).

### Files: CSV in and out

These move rows straight between a file and the database, in the same
files as the data library's [table files](#table-files): `sql_save`
writes `.csv`, `.tsv`, `.txt` or `.json`; the others read `.csv`, `.tsv`
or `.json`. From `.json`, numbers and booleans go in as they are, and a
list or object inside a row is stored as its JSON text. Each returns how many records it wrote or changed. A file is
**all or nothing**: if one line is refused (a duplicate key, a `CHECK`,
a missing `NOT NULL` value), no line of that file is kept, even inside
`BEGIN ... COMMIT`.

| Function | Does |
|---|---|
| `sql_save[db, query, path [, values]]` | runs the query and writes its rows to the file (the header too, even with no rows) |
| `sql_load[db, table, path [, types]]` | adds a record per line; the header names the columns. If the table isn't there, makes it first (see below) |
| `sql_update[db, table, key, path]` | for each line, changes the record whose `key` column matches, setting the file's other columns |
| `sql_delete[db, table, key, path]` | removes the records whose `key` matches a line (other columns are ignored) |
| `sql_upsert[db, table, key, path]` | adds the lines whose key is new and changes the ones already there (`key` must be `UNIQUE` or the `PRIMARY KEY`) |

```
import sql

db = sql_open["shop.db"]
sql_save[db, "SELECT sku, title, price FROM books ORDER BY sku", "books.csv"]
sql_save[db, "SELECT * FROM books WHERE price < ?", "cheap.tsv", list [1000]]

sql_load[db, "books", "new_books.csv"]
n = sql_update[db, "books", "sku", "price_changes.csv"]    // sku,price
sql_delete[db, "books", "sku", "discontinued.csv"]         // sku
sql_upsert[db, "books", "sku", "restock.csv"]
```

```
sku,title,price
B1,Dune,950
B5,"I, Robot",650
```

- Values from the file are text; columns declared `INTEGER` or `REAL`
  store them as numbers (`"950"` → `950`), as SQLite does. An empty cell
  is `NULL`, so it sets the column to `NULL` in `sql_update`.
- **Column types.** `sql_load` takes the same map of
  [column types](#column-types) as `table_read`, in place or in a
  variable. The file's values are converted first, so a bad one stops
  the load before anything is written.
- **A missing table is made** from the file's header, in the file's
  column order. With a map, each column gets its type and
  `"primary_key"` becomes the table's `PRIMARY KEY`; without one, every
  column is text. If the file is then refused, the new table is dropped
  again. When the table is already there, the map only converts values
  (its own column types and key stay).

```
types = map ["code": "text", "missions": "integer", "joined": "date", "primary_key": "code"]
sql_load[db, "agents", "agents.csv", types]
// CREATE TABLE "agents" ("code" TEXT PRIMARY KEY, "name" TEXT, "missions" INTEGER, ..., "joined" DATE)
```

  A Turtle type word becomes each database's own type; a SQL type is
  used exactly as written:

  | Word | SQLite | PostgreSQL | MySQL |
  |---|---|---|---|
  | `string`, `text` | `TEXT` | `TEXT` | `TEXT` (`VARCHAR(255)` for the key, which MySQL needs) |
  | `integer` | `INTEGER` | `BIGINT` | `BIGINT` |
  | `float` | `REAL` | `DOUBLE PRECISION` | `DOUBLE` |
  | `boolean` | `BOOLEAN` | `BOOLEAN` | `BOOLEAN` (MySQL gives back `1` / `0`) |
  | `date` | `DATE` (stored as text, `1953-04-13`) | `DATE` | `DATE` |
- A line whose key isn't in the table changes nothing in `sql_update`
  and `sql_delete`; compare the count with the file's lines, or look the
  keys up (see `testdata/sql/11_csv_import.trt`).
- Each file runs as one statement, saved in one write, so a file of
  thousands of lines is fast.
- Table and column names go into the SQL quoted, so any name works and
  none can change the statement.

### More of SQLite

- **Window functions**: `row_number rank dense_rank percent_rank
  cume_dist ntile lag lead first_value last_value nth_value`, and every
  aggregate with `OVER (PARTITION BY ... ORDER BY ...)`, frames (`ROWS`,
  `RANGE`, `GROUPS`, `EXCLUDE`) and named `WINDOW`s.
- **JSON**: `json json_extract json_object json_array json_set
  json_insert json_replace json_remove json_patch json_type json_valid
  json_group_array json_group_object` and more, the `->` and `->>`
  operators, and `json_each` / `json_tree` as tables. Also
  `generate_series(1, 10)` as a table.
- **Triggers**: `CREATE TRIGGER` (`BEFORE`, `AFTER`, `INSTEAD OF` on
  views, `FOR EACH ROW`, `WHEN`, `UPDATE OF`), `RAISE(...)`.
- **Savepoints**: `SAVEPOINT name`, `RELEASE name`, `ROLLBACK TO name`.
- **Generated columns** (`AS (price * qty) STORED` or `VIRTUAL`) and
  indexes on expressions (`CREATE INDEX ... ON users (lower(email))`).
- **PRAGMA**: `table_info`, `index_list`, `foreign_key_list`,
  `foreign_keys`, `integrity_check`, `journal_mode` (`DELETE` or `WAL`),
  `user_version` and others; also as tables
  (`SELECT * FROM pragma_table_info('books')`).
- **VACUUM** (and `VACUUM INTO 'copy.db'`), and **ATTACH** another
  database file to read from it (`ATTACH 'old.db' AS old`).

### Not yet

Changing `WITHOUT ROWID` tables (they're read fine), virtual tables
(`fts5` and others), and changing an attached database (it's read-only).
Databases that use auto-vacuum or store text as UTF-16 can be read but
not changed.

### Servers: PostgreSQL and MySQL

Give `sql_open` a server address instead of a file. Every other
function is the same, including the CSV ones:

```
import sql
import data

db = sql_open["postgres://ann:secret@localhost:5432/shop"]
// or: db = sql_open["mysql://ann:secret@localhost:3306/shop"]
rows = sql_query[db, "SELECT title, price FROM books WHERE price < ?", list [1000]]
show table[rows] .
sql_upsert[db, "books", "sku", "restock.csv"]
sql_close[db]
```

- **The server runs the SQL**, so it's the server's own dialect:
  PostgreSQL's or MySQL's functions and types, not SQLite's. `?` marks
  values for all three (Turtle turns them into PostgreSQL's `$1, $2`),
  and values go to the server separately from the statement.
- **Values**: integers, floats, text and `NULL` as for SQLite; a
  server's `BOOLEAN` comes back as `true`/`false` (MySQL stores booleans
  as `TINYINT`, so `1`/`0`); `DATE`, `TIMESTAMP` and `DATETIME` come
  back as dates; `NUMERIC`/`DECIMAL` as integers when whole, floats
  otherwise; `JSON` as text.
- **Transactions**: `BEGIN`, `COMMIT`, `ROLLBACK` work as with SQLite.
  The CSV functions are still all or nothing, inside a transaction or not.
- **Logins**: PostgreSQL's `scram-sha-256`, `md5` and `password`;
  MySQL's `caching_sha2_password`, `sha256_password` and
  `mysql_native_password`. A wrong password is an error of kind `sql`.
- **TLS** (encryption) is used when the server offers it. Options go
  after `?` in the address: PostgreSQL `sslmode=disable`, `prefer` (the
  default), `require` or `verify-full`, and `connect_timeout=10`; MySQL
  `tls=false`, `preferred` (the default), `skip-verify` or `true`, and
  `timeout=10`.
- `sql_create` is for SQLite files only; make a server's database with
  `CREATE DATABASE` (or its own tools), then `sql_open` it.
- `sql_tables` lists the tables of the database (PostgreSQL: of the
  current schema).
- `testdata/sql/13_servers.trt` runs the same program against both
  servers (set `TURTLE_PG_URL` and `TURTLE_MYSQL_URL`).

**Errors** are kind `sql` (`sql_query: no such table: shelves`); a missing
file is kind `file`.

## Strings library

`import strings` (or `import strings [find, join]`) provides string
functions that read well sentence-style. Three share their behaviour with
string methods: `find` is `indexOf`, `substring` is `slice`, `isinstring`
is `contains`. All indices count characters, not bytes.

| Function | Args | Returns |
|---|---|---|
| `find[text, part]` | two strings | index of the first `part` in `text`, or `-1` |
| `substring[text, start [, end]]` | string, integer(s) | the characters `[start, end)`; negative counts from the end, `end` defaults to the end, out-of-range bounds clamp (same rules as `slice`) |
| `isinstring[part, text]` | two strings | Boolean: does `text` contain `part` |
| `join[items [, separator]]` | list or set, optional string | one string: each element's shown form, with `separator` (default `""`) between them |

```
import strings

line = "hello world"
show line find "wor" .           // 6
show line substring 0, 5 .       // hello
show line substring -5 .         // world: -5 counts from the end
show "wor" isinstring line .     // true
words is line at split " " .
show words join ", " .           // hello, world
```

`isinstring` takes the part first, so it reads as a sentence with the
literal on the left: `"wor" isinstring line`.

## System library

`import system` (or `import system [args, exists]`):

| Function | Args | Returns |
|---|---|---|
| `args[]` | — | a `list` of the command-line arguments after the script path, as strings (empty if none) |
| `exists[path]` | path string | Boolean: is there a file or folder at `path` |
| `isFile[path]` | path string | Boolean: is `path` a regular file |
| `isFolder[path]` | path string | Boolean: is `path` a folder |
| `exit[code]` | optional integer (default `0`) | ends the program immediately with that exit code; `0` means success, anything else failure |
| `env[name]` | variable name string | the environment variable's value as a string, or `none` if it isn't set |
| `scriptFolder[]` | — | the full path of the folder the running script is in |
| `erase[path]` | path string | deletes the file, or the folder and **everything in it** (no undo); returns `none`. A missing path is a `file` error. It refuses the folder `turtle` runs in and any folder above it |
| `contents[path]` | optional folder path (default `"."`) | a `list` of the names of the files and folders inside, sorted (names only, not full paths); a missing folder is a fatal error. The same listing as the `[directory]` statement, usable inline |

`warn <expr> {, <expr>} .` is `show` for stderr: same pieces, same
period, but the line goes to standard error, so it isn't mixed into
output that's piped or saved to a file. It needs `import system` (or
`import system [warn]`).

```
import system
warn "can't read ", name, ", skipping" .
erase["build"]
```

**Paths resolve from the folder you ran `turtle` in**, like any
command-line tool, unless absolute. That applies here and to `[read]`,
`[write]`, `[append]` and `[directory]`. So
`turtle ~/tools/count.trt notes.txt` reads `./notes.txt`. (`import` is
different: it always looks next to the script, so a program and its
libraries can be moved together.) To use a file that sits next to the
script, build its path from `scriptFolder[]`:

```
import system
config = scriptFolder[] + "/config.txt"
[read] config to lines [end]
```

A complete tool, with usage message and exit codes:

```
// turtle count.trt notes.txt
import system
import strings

a = args[]
if ] length of a == 0 [
    show "usage: count.trt <file>" .
    exit[2]
if [end]
name is a at get 0 .
if ] !name exists [
    show "missing: " + name .
    exit[1]
if [end]
[read] name to lines [end]
show name, ": ", length of lines, " lines" .
show lines join " | " .
```

Listing a folder:

```
import system
show contents[] .                    // the folder turtle was run in
[loop][name in contents["sub"]]
    if ] isFolder["sub/" + name] [
        show name, "/" .
    else ]
        show name .
    if [end]
[loop][end]
```

Check a file before reading it, since a missing file is a fatal error for
`[read]`:

```
// turtle tool.trt notes.txt sub missing.txt
import system

[loop][name in args[]]
    if ] name exists [
        if ] isFolder[name] [
            show name, " is a folder" .
        else ]
            show name, " is a file" .
        if [end]
    else ]
        show name, " does not exist" .
    if [end]
[loop][end]
```

## Files

Paths are quoted strings or barewords (`file.txt`, `data/in.csv`),
resolved relative to the folder `turtle` was run in, like any command-line
tool (use `system`'s `scriptFolder[]` for files next to the script). A
single bare word with no `.` or `/` uses the variable of that name if one
exists. Any I/O failure (file not found, permission denied, etc.) is a
fatal error. Check first with `system`'s `exists[path]`.

```
[read] <path> to <ident> [end]
```

Reads the whole file, splits on newlines, assigns the result as a `list`
of strings (one per line) to `<ident>`.

```
[write] <path>
<line>
...
[end]

[append] <path>
<line>
...
[end]
```

Each `<line>` is either a quoted string (written verbatim) or a bare
identifier (replaced with that variable's current display value). Lines
are newline-joined and written with a trailing newline. `[write]` creates
or overwrites the file; `[append]` creates it if missing, otherwise adds to
the end.

```
[directory] <path> to <ident> [end]
```

Lists the directory's entries (names only, not full paths) into `<ident>`
as a `list`.

## Shell escape: `sys`

```
sys <rest of line>
```

`sys` must be the very first word of the statement — this is a real
keyword, not a substring match, so it only triggers there (unlike the
legacy interpreter, where any line merely *containing* "sys" anywhere
would misfire into shell execution). Everything after it, verbatim to the
end of the line, is passed to `sh -c`. The child process inherits stdin,
stdout, and stderr; its exit status is not checked or reported back to
the Turtle program.

This is a deliberately dangerous, unsandboxed feature — the Turtle
equivalent of Python's `os.system`. Only use it with trusted script
content.

## Modules: `import`

```
import <name>
```

```
import <name> [<f>, <g>]
```

Reads `<name>.trt` (relative to the script's own folder, wherever `turtle`
was run from) and runs it
once, in its own scope. Its top-level functions become available to your
program: all of them, or only the ones listed in `[...]`. Its top-level
variables stay private to it. When two imports export the same function
name, call it qualified by module, as in `mylib now[]` or `time now[]`.
An unqualified call to a name that clashes is a fatal error. Full rules
are in [`reference.md`](reference.md#modules).
