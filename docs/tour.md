# A tour of Turtle

This walks through the language by example. For the complete formal syntax
see [`reference.md`](reference.md); for the standard library (data
structure methods, files, `sys`) see [the library docs](library/index.md).

## Comments

```
// a line comment, runs to end of line

//* a block comment
   can span multiple lines *//
```

## Variables and expressions

Assignment never takes a trailing period:

```
x = 5
name = "Nigele"
y = x + 1
pi = 3.14159
```

Numbers are integers or floats. `+` does double duty: numeric addition when
both sides are numbers, string concatenation otherwise. (Lists, sets and
maps only add to their own kind; `list [1] + 1` is an error.)

```
a = 1 + 2 * (3 + 4)     // 15 — normal precedence, parens work
greeting = "Hello, " + name + "!"
```

Standard comparison and logical operators work in conditions:
`> < >= <= == != && ||`.

## Printing: `show`

```
show "Hello, world" .
show 1 + 2 .
show "x is ", x, ", y is ", y .
```

`show` always ends with a period. Multiple comma-separated pieces are
concatenated with **no separator** — any spacing you want has to be inside
the string literals themselves:

```
show "x = ", x, ", y = ", y .
```

Put a value straight into a string with `{...}`:

```
show "x = {x}, y = {y}, total = {x + y}" .
```

Strings can also use single quotes, handy when the text has double
quotes in it: `show 'She said "hi"' .`

## Reading input

```
name = ? "What's your name? "
show "Hi, ", name, "!" .
```

Prints the prompt, reads one line from stdin, stores it as a string. No
trailing period.

## Converting types: `change`

Input from `?` is always a string, so `change` converts it (or any value)
to another type. As a statement it mutates a variable in place; as an
expression it produces a new value without touching the source:

```
raw = ? "Enter a number: "
change raw to integer .        // raw itself is now the integer
show raw + 1 .

n = change "3.5" to float      // expression form, assigned to a new name
show n * 2 .
```

The type names — `integer`, `float`, `string`, `ascii`, `char`, `hex` — go
after `to`. `ascii`/`char` convert between a one-character string and its
code point, and `hex` converts between an integer and its hex digits,
either direction inferred from what you hand it:

```
show change "A" to ascii .     // 65
show change 66 to char .       // B
show change 255 to hex .       // ff
show change "ff" to hex .      // 255
```

`list` and `set` convert between the two. A quick way to drop duplicates:

```
nums = list [3, 1, 3, 2, 1]
show change nums to set .      // { 3, 1, 2 }
```

## Conditionals

Turtle's if/else uses **reversed brackets** — the condition is wrapped
`] ... [`, not `[ ... ]`:

```
if ] x > 10 [
    show "big" .
else if ] x > 0 [
    show "small" .
else ]
    show "non-positive" .
if [end]
```

Every if-statement closes with `if [end]`. `else if`/`else` are optional and
work like you'd expect.

### Nesting

A nested if goes inside another clause's body, and every keyword of the
nested chain gets one extra leading `[`:

```
if ] a > 0 [
    show "a is positive" .
    [if ] b > 0 [
        show "both positive" .
    [else ]
        show "a positive, b not" .
else ]
    show "a not positive" .
if [end]
```

The nested chain has **no `[end]` of its own** — it implicitly closes the
moment a clause at the same or shallower depth (a bare `else`/`else if`/
`if [end]`, with no leading `[`) shows up. This can nest arbitrarily deep by
adding another leading `[` per level, though every real-world example only
goes one level deep.

## Loops

Two header forms, both inside `[loop][...]` ... `[loop][end]`:

```
// C-style: init; condition; post
[loop][i = 0; i < 5; i++]
    show i .
[loop][end]

// while-style: condition only, re-checked every pass
count = 0
[loop][count < 3]
    show count .
    count = count + 1
[loop][end]
```

The post clause accepts `i++`, `i--`, or a full `i = <expr>`.

`break` exits the loop immediately; `continue` skips to the next iteration's
condition check:

```
[loop][i = 0; i < 10; i++]
    if ] i == 5 [
        break
    if [end]
    show i .
[loop][end]
```

### Nesting

Unlike if/else, **every loop needs its own `[end]`**, nested or not:

```
[loop][i = 0; i < 3; i++]
    [loop][j = 0; j < 3; j++]
        show i , ",", j .
    [loop][end]
[loop][end]
```

(See [`architecture.md`](architecture.md) if you're curious why this is
stricter than if/else nesting — short version: it removes a real footgun
that existed in the original implementation.)

## Functions

```
def plus[a, b]
    result = a + b
    return result
def [end]

sum = plus[1, 2]
show sum .
```

A function has no closure over its *caller's* variables — it can't see
another function's locals — but it can read top-level (global) variables,
and it can call any other top-level function, including itself:

```
PI = 3.14159

def circleArea[r]
    return PI * r * r
def [end]

show circleArea[2] .    // 12.56636
```

Assigning inside a function never writes back to the global, though — it
always creates or updates a local of that name for the rest of the call,
leaving the global untouched:

```
count = 0

def bump[]
    count = count + 1    // shadows the global; doesn't change it
    return count
def [end]

show bump[] .    // 1
show bump[] .    // 1 again — global count is still 0
show count .     // 0
```

If you want a function to actually mutate shared state, use a data
structure instead — `add`/`remove`/method calls on a global `list`/`set`/
`map` mutate that same value in place, visible after the call returns,
unlike a plain scalar assignment:

```
nums = list [1, 2, 3]

def addToNums[v]
    add v to nums .
def [end]

addToNums[99]
show nums .    // [ 1, 2, 3, 99 ] — the global list really did change
```

Recursion works the same way it always did:

```
def factorial[n]
    if ] n <= 1 [
        return 1
    else ]
        return n * factorial[n - 1]
    if [end]
def [end]

show factorial[5] .    // 120
```

`return` can be used directly inside a larger expression at the call site
(`x = plus[1, 2] + 3`), and a call with no arguments still needs the
brackets: `def greet[] ... def [end]`, called as `greet[]`.

## Data structures

```
nums = list [3, 1, 2]
letters = set ["a", "b", "b", "c"]     // duplicates dropped
ages = map ["Alice": 30, "Bob": 25]
```

Statement-form operations (all take a trailing period):

```
add 4 to nums .
remove 1 from nums .
sort nums .
reverse nums .
insert 99 to nums at 0 .
put 7 to nums at 1 .   // replaces the item at 1
delete "Bob" from ages .
length of nums .    // prints
min of nums .        // prints
max of nums .         // prints
```

Method-call form, for everything else (`get`, `union`, `invert`, ...):

```
first is nums at get 0 .
combined is letters at union otherSet .
show first .
```

Inside an expression, `get[...]` and `slice[...]` take brackets and bind
to the value right before them, so they work in arithmetic:

```
a = list [1, 2]
b = list [10, 20]
show a at get[0] + b at get[1] .     // 21
```

See [the library docs](library/index.md) for the full method list per type.

## Your own types: `assemble`

When the same shape of data keeps appearing (an order, a point, a user),
name it once with `assemble`:

```
assemble Order [item, qty, price]

o = Order["pen", 3, 1.5]     // one value per field, in order
show o .                     // Order { item: "pen", qty: 3, price: 1.5 }
show qty of o .              // 3
qty of o = 10                // fields can be changed

total = qty of o * price of o
```

A misspelled field (`prise of o`) or a missing value (`Order["pen", 3]`)
is an error that names the fields, rather than a silently wrong map.
Two `Order`s with the same fields are `==`.

## When things go wrong: `safe` and `handle`

An error, like a missing file or dividing by zero, normally stops the
program. To deal with it yourself, put the code in a `safe` block. Under
`handle`, list the kinds of error you expect, name the error, and write
what to do if one happens. `safe [end]` closes it:

```
safe
    [read] scores.txt to lines [end]
handle [file] error .
    show "no scores yet: ", error .
    lines = list []
safe [end]
```

The code under `handle` runs only if something went wrong.
`handle [] error .` handles every kind. `kind of error`,
`line of error` and `message of error` give the parts. Raise your own
with `fail "message"`. See the reference for the list of kinds.

## Dates and time

`import time` gives dates you can add to and compare:

```
import time
now_day = today[]
due = add_time[now_day, 30, "days"]
show "due {due}, a {weekday of due}" .
show time_between[now_day, due, "days"] .   // 30
every[1, "hours", check_inbox]              // run a function every hour
```

## JSON

`import json` reads and writes JSON, as maps and lists:

```
import json
user = json_read["user.json"]
show json_get[user, "address", "city"] .   // none if it isn't there
json_write["copy.json", user]
data = load['{"name": "Ann", "tags": ["a", "b"]}']
```

## The web

`import http` fetches pages and talks to APIs; JSON answers go straight
into `load`:

```
import http
import json
users = load[http_get["https://api.example.com/users"]]
reply = http_post["https://api.example.com/users", map ["name": "Ann"]]
```

## Databases

`import sql` reads SQLite databases; rows come back as a list of maps:

```
import sql
db = sql_open["shop.db"]
cheap = sql_query[db, "SELECT title FROM books WHERE price < ?", list [1000]]
```

## Files

```
[read] notes.txt to lines [end]
show lines .                 // a list, one element per line

[write] out.txt
"first line"
someVariable
[end]

[append] out.txt
"more text"
[end]

[directory] . to entries [end]
show entries .
```

## Shell escape

```
sys echo hello from turtle
```

Runs the rest of the line as a shell command, inheriting stdin/stdout/
stderr. Deliberately dangerous — same idea as Python's `os.system` — so use
it only when you mean to.

## Tables, CSV files and statistics

`import data` works on whole collections, and reads and writes tables:

```
import data

sales = table_read["sales.csv"]       // a list of maps; plain numbers come back as numbers
show table[sales] .                   // lined up in columns
show mean[sales, "copies"], " ", median[sales, "copies"] .
show table[describe[sales, "price"]] .
big = sales keep s give copies of s > 10 .
table_write["big.csv", big]
```

A cell like `007` stays text, so codes and ids keep their zeros; give
column types to say otherwise (`table_read["codes.csv", map ["code":
"text"]]`).

## Matrices

`import linear` adds matrices. Rows end at `;` or at the end of a line,
and `*` is the matrix product:

```
import linear

a = matrix [
    2, 1
    1, 3
]
x = solve[a, list [3, 5]]        // [ 0.8, 1.4 ]
show a * x .                     // [ 3.0, 5.0 ]
show inverse[a] * a == identity[2] .
```

Numbers can be written the scientific way too: `1e-9`, `6.02e23`.

## Modules

`import` brings in a library that comes with Turtle, or a `.trt` file of
your own:

```
import data                    // a library that comes with Turtle
import time [now, sleep]       // only some of its names
import shop                    // shop.trt, next to your program
import lib/money               // lib/money.trt; its name is "money"
```

A file you import runs once, in its own scope. Its top-level functions
become yours to call; its variables stay private to it:

```
// shop.trt
taxrate = 0.08
def withtax[amount]
    return amount + amount * taxrate
def [end]
```

```
import shop
show withtax[100] .            // 108.0
show shop withtax[50] .        // 54.0: the module named in front
f = shop withtax               // the function itself, as a value
```

If two imports have a function with the same name, name the module in
front (`time now[]`, `shop now[]`); calling the plain name then is an
error that says exactly this. Your own `def` always wins over an imported
function of the same name.

Some imports turn on words of the language, in that file only:
`import random` (`random list of 5 integers from 0 to 9`), `import test`
(`check`, `verify`, `validate`), `import log` (`log warn "..." .`) and
`import linear` (`matrix`).

Using a library's function without its import is an error that names the
import to add:

```
x = sum[nums]                  // "sum" needs "import data" first
```

Every library, grouped by what it's for: [the library docs](library/index.md).
Every keyword in one table: [Keywords](reference.md#keywords).
