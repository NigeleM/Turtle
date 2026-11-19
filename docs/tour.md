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

Numbers are integers or floats. `+` adds two numbers and joins two pieces
of text. Text and a number is an error: `"5" + 0` stops the
program instead of giving `"50"`. To put a value in text, write it inside
the text with `{...}`: `"total: {n}"`. (Lists, sets and maps only add to
their own kind; `list [1] + 1` is an error.)

```
a = 1 + 2 * (3 + 4)     // 15 — normal precedence, parens work
greeting = "Hello, " + name + "!"
```

Standard comparison and logical operators work in conditions:
`> < >= <= == != && ||`.

## Printing: `show`

```
x = 4
y = 9
show "Hello, world" .
show 1 + 2 .
show "x is ", x, ", y is ", y .
```

`show` always ends with a period. Comma-separated pieces are written with
one space between them, except that `, . : ; )`
sit against the piece before them and no space is added where a piece
already has one (see the reference):

```
x = 4
y = 9
show "x = ", x, ", y = ", y .
```

Put a value straight into a string with `{...}`:

```
x = 4
y = 9
show "x = {x}, y = {y}, total = {x + y}" .
```

Strings can also use single quotes, handy when the text has double
quotes in it: `show 'She said "hi"' .`

## Reading input

> *Illustrative.* This example waits for keyboard input, so it is shown for reading and is not run as a test.

```
name = ? "What's your name? "
show "Hi, {name}!" .
```

Prints the prompt, reads one line from stdin, stores it as a string. No
trailing period.

## Converting types: `change`

Input from `?` is always a string, so `change` converts it (or any value)
to another type. As a statement it mutates a variable in place; as an
expression it produces a new value without touching the source:

> *Illustrative.* This example waits for keyboard input, so it is shown for reading and is not run as a test.

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
x = 12
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

```turtle
a = 1
b = 2
if ] a > 0 [
    show "a is positive" .
    if ] b > 0 [
        show "both positive" .
    else ]
        show "a positive, b not" .
    if [end]
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

A function can't change a global by assigning to it, though. Trying
it is an error that says what to do:

```
count = 0

def bump[]
    count = count + 1    // error: count is a global, and a function can't change it ...
def [end]
```

Give the function the value, and assign what it returns:

```
count = 0

def bump[n]
    return n + 1
def [end]

count = bump[count]
show count .     // 1
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

### Three ways to call a function

A call can be written in three forms that mean the same thing, so a program can use the
one that reads best:

```turtle
def apple[a, b]
    return a + b
def [end]
show apple[1, 2] .          // bracket form: 3
show 1 apple 2 .            // value first: 3
show apple 1, 2 .           // name first: 3
```

In `show`, a `;` ends the arguments, so the next item is printed separately:
`show apple 1, 2; " is the sum" .` prints `3 is the sum`. Inside a list, `;` ends a
call's arguments and also separates items: `list[apple 1,2; apple 3,4;]` is `[ 3, 7 ]`.

## Data structures

```
nums = list [3, 1, 2]
letters = set ["a", "b", "b", "c"]     // duplicates dropped
ages = map ["Alice": 30, "Bob": 25]
```

Statement-form operations (all take a trailing period):

```turtle
nums = list [3, 1, 2]
ages = map ["Alice": 30, "Bob": 25]
add 4 to nums .
remove 1 from nums .
sort nums .
reverse nums .
insert 99 to nums at 0 .
put 7 to nums at 1 .   // replaces the item at 1
put 31 to ages at "Alice" .   // sets a key
delete "Bob" from ages .
length of nums .    // prints
min of nums .        // prints
max of nums .         // prints
```

Method-call form, for everything else (`get`, `union`, `invert`, ...):

```turtle
nums = list [3, 1, 2]
otherSet = set [3, 4]
first is nums at get 0 .
combined is set [1, 2] at union otherSet .
show first .
show combined .
```

Inside an expression, `get[...]` and `slice[...]` take brackets and bind
to the value right before them, so they work in arithmetic:

```
a = list [1, 2]
b = list [10, 20]
show a at get[0] + b at get[1] .     // 21
```

`put VALUE to THING at POSITION .` is the same for lists and maps. As
methods the order differs: a list takes the value first (`nums at put[7, 1]`),
a map the key first, as it's written (`ages at put["Alice", 31]`). A map put
that looks swapped gets a warning.

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

> *Illustrative.* This example depends on values set elsewhere on the page, so it is shown for reading and is not run as a test.

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

```turtle
import json
json_write["user.json", map ["name": "Ann"]]
import json
user = json_read["user.json"]
show json_get[user, "address", "city"] .   // none if it isn't there
json_write["copy.json", user]
data = load['{"name": "Ann", "tags": ["a", "b"]}']
```

## The web

`import http` fetches pages and talks to APIs; JSON answers go straight
into `load`:

> *Illustrative.* This example needs a network address, so it is shown for reading and is not run as a test.

```
import http
import json
users = load[http_get["https://api.example.com/users"]]
reply = http_post["https://api.example.com/users", map ["name": "Ann"]]
```

## Databases

`import sql` reads SQLite databases; rows come back as a list of maps:

> *Illustrative.* This example needs an import that the example does not include, so it is shown for reading and is not run as a test.

```
db = sql_create["shop.db"]
sql_close[db]
import sql
db = sql_open["shop.db"]
cheap = sql_query[db, "SELECT title FROM books WHERE price < ?", list [1000]]
```

## Files

```turtle
[write] notes.txt
"first line"
[end]
[read] notes.txt to lines [end]
show lines .
someVariable = "second line"
[write] out.txt
"first line"
someVariable
[end]
[append] out.txt
"more text"
[end]
[directory] . to entries [end]
show length of entries > 0 .
```

## Shell escape

```
sys echo hello from turtle
```

Runs the rest of the line as a shell command, inheriting stdin/stdout/
stderr. Deliberately dangerous, so use
it only when you mean to.

## Tables, CSV files and statistics

`import data` works on whole collections, and reads and writes tables:

```
[write] sales.csv
"copies,price"
"12,9.5"
"4,7"
[end]
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

`import` brings in a library that comes with Turtle, or a `.turtle` file of
your own:

> *Illustrative.* This example needs a module file (shop.turtle) that is not in the repository, so it is shown for reading and is not run as a test.

```
import data                    // a library that comes with Turtle
import time [now, sleep]       // only some of its names
import shop                    // shop.turtle, next to your program
import lib/money               // lib/money.turtle; its name is "money"
```

A file you import runs once, in its own scope. Its top-level functions
become yours to call; its variables stay private to it:

```
// shop.turtle
taxrate = 0.08
def withtax[amount]
    return amount + amount * taxrate
def [end]
```

> *Illustrative.* This example needs a module file (shop.turtle) that is not in the repository, so it is shown for reading and is not run as a test.

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

> *Illustrative.* This example needs an import that the example does not include, so it is shown for reading and is not run as a test.

```
nums = list [1, 2]
x = sum[nums]                  // "sum" needs "import data" first
```

Every library, grouped by what it's for: [the library docs](library/index.md).
Every keyword in one table: [Keywords](reference.md#keywords).

## Theories

A theory teaches Turtle a new phrase. It has a plain-language **abstract**, a **notation**
that shows how the phrase is written, a **definition** with the code, **theorem** lines that
every result must keep, and **proof** examples that must give the right answers. Here the
phrase is `tally b in a`, which counts how many times `b` appears in the list `a`:

```turtle
theory tally
    abstract
        tally says how many times b appears in the list a.
    notation tally b in a .
    definition
        n = 0
        [loop][x in a]
            if ] x == b [
                n = n + 1
            if [end]
        [loop][end]
        return n
    theorem result >= 0
    proof
        tally 1 in list [1, 1, 0, 3] . is 2
        tally 9 in list [] . is 0
theory [end]

nums = list [1, 1, 0, 3]
show "ones: ", tally 1 in nums .
```

```
ones: 2
```

Use `turtle test` to check every theory's proof and theorems. The reference's
[Theories](reference.md#theories) section has the full rules.

## Scrolls

A scroll passes a value through a list of steps, in order. Each step receives the result of
the one before it. The `into` word starts the steps:

```turtle
def add1[x]
    return x + 1
def [end]
def double[x]
    return x * 2
def [end]
show scroll 3 into add1, double .
```

```
8
```

The [scrolls page](scrolls.md) covers `here`, `none` in a scroll, and errors inside a scroll.

## Sentence form

Some calls can be written as words, so a line reads like a sentence. This works with the
functions of an import:

```turtle
import data
nums = list [1, 2, 3, 4]
show nums keep x give x > 2 .
show nums sum .
```

```
[ 3, 4 ]
10
```

The [reference](reference.md#calls-with-the-arguments-after-the-name) explains the forms and the
rules for negative values and `;`.

## Help and version

`help` explains what you give it. `help[]` lists the topics, `help["keywords"]` lists every
keyword, `help["if"]` explains one keyword, and `help[nums]` explains a value and its methods.
`stdlib[]` gives every library and its functions, and `show version .` prints the version:

```turtle
help["if"]
show version .
```

```
if        (a keyword)
  Runs code when a condition holds.
  Example: if ] x > 1 [ ... if [end]
  More: Conditionals (docs/reference.md#conditionals)
```

## Tracing a program

`turtle trace` runs a program and prints each line on standard error as it runs, with the
values it produced. Loops show each pass:

```
$ turtle trace tr.turtle
line 1   import data
line 2   nums = list [5, 7]               nums = [ 5, 7 ]
line 3   total = 0                        total = 0
line 4   [loop][x in nums]
                                          pass 1: x = 5
line 5       total = total + x            total = 5
                                          pass 2: x = 7
line 5       total = total + x            total = 12
```

To keep the trace apart from the program's output, send the trace to a file:
`turtle trace report.turtle 2> trace.txt`.

## Debugging a program

`turtle debug` runs a program one line at a time. At the `(debug)` prompt, Enter or `s` steps,
`n` steps over a call, `o` runs to the end of a function, `c` continues to the next breakpoint,
`b 5` sets a breakpoint, `d 5` removes one, `p total` shows a value, `v` lists the variables, `w`
shows where you are, `l` lists the lines around you, `h` lists the commands and `q` quits:

```
$ turtle debug dbg2.turtle
turtle debug: stopped before the first line. Enter steps; h shows the commands.
→ line 1  import data
(debug) s
→ line 2  nums = list [5, 7]
(debug) s
→ line 3  total = 0
(debug) s
→ line 4  [loop][x in nums]
(debug) s
→ line 5      total = total + x
(debug) p total
0
(debug) b 5
will stop at 5
(debug) c
→ line 5      total = total + x
(debug) p total
5
(debug) q
```

A breakpoint stops the program only on a line that runs. The debugger warns you about blank
lines, comments, and declarations such as `def` and `notation`.

## Formatting

`turtle fmt` lays out `.turtle` files in one standard way. `--check` only lists the files that
need it:

```
$ turtle fmt --check messy.turtle
messy.turtle needs formatting
$ turtle fmt messy.turtle
formatted messy.turtle
```

## Building a program

`turtle build` makes a single program file that runs without Turtle installed:

```
$ turtle build hello.turtle -o hello
$ ./hello
Hello, world
```

The file runs on the kind of computer it was built on.

## Editors

Turtle has a language server, `turtle lsp`, which gives an editor help while you type:
completions, errors as you type, hover help and go to definition. The [editors page](editors.md)
lists the editors and how to set them up, and the VS Code extension includes syntax colours.

---

Copyright 2017-2026 Nigele McCoy. Licensed under the
[Apache License 2.0](../LICENSE); see [NOTICE](../NOTICE).
