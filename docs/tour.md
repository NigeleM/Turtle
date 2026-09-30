# A tour of Turtle

This walks through the language by example. For the complete formal syntax
see [`reference.md`](reference.md); for the standard library (data
structure methods, files, `sys`) see [`stdlib.md`](stdlib.md).

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
both sides are numbers, string concatenation otherwise:

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
def add[a, b]
    result = a + b
    return result
def [end]

sum = add[1, 2]
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
(`x = add[1, 2] + 3`), and a call with no arguments still needs the
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

See [`stdlib.md`](stdlib.md) for the full method list per type.

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

## Modules

```
import util
```

Parses and runs `util.t`, adding its top-level functions and variables to
your program's global scope. There's no module namespacing — everything
lands in one shared global scope, same as the rest of the top-level
program.

Two names are special: `import math` and `import time` don't read a file
at all — they unlock built-in capability instead:

```
import math

r is 16 at sqrt .
show r .              // 4

n is 10 at random .
show n .               // some integer in [0, 10)

import time

t1 = now[]
sleep[0.5]              // seconds by default
show now[] - t1 .      // at least 500 (now[] is always milliseconds)

sleep[250, "ms"]        // or be explicit about milliseconds
```

Calling a math method or `now`/`sleep` before the matching `import` is a
fatal error that names exactly which import is missing. See
[`stdlib.md`](stdlib.md#number) for the full method/function list.
