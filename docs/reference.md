# Language reference

The complete, formal syntax and semantics of Turtle, as implemented by the
`token`/`lexer`/`ast`/`parser`/`object`/`evaluator` packages. For a gentler,
example-first introduction see [`tour.md`](tour.md); for the standard
library see [`stdlib.md`](stdlib.md).

Notation below: `code` is a literal token; `<angle-brackets>` is a
non-terminal; `[x]` is optional; `{x}` is zero-or-more; `|` is alternation.

## Lexical grammar

- **Comments**: `// ...` runs to end of line. `//* ... *//` is a block
  comment that may span multiple lines. Both are stripped by the lexer.
- **Identifiers**: `<letter|_> {letter|digit|_}`.
- **Numbers**: `<digits>` (integer) or `<digits>.<digits>` (float).
- **Strings**: double-quoted. Escapes: `\n`, `\t`, `\"`, `\\`.
- **Booleans**: `true`, `false`.
- **None**: `none` — the single "no value" value (type `NONE`).
- **Reserved words** (cannot be used as identifiers): `true false none show if
  else def end loop return list set map import sys to from at of is add
  change remove delete sort reverse insert min max length read write append
  directory break continue gives in assemble`. Type names used after `change ... to` —
  `integer`, `float`, `string`, `ascii`, `char`, `hex` — are **not** reserved;
  like method names (`get`, `union`, ...) they're plain identifiers whose
  meaning is only special right after `to`.

One statement per source line, except explicit multi-line blocks with their
own begin/end markers (function/if/loop bodies, `[write]`/`[append]`/
`[read]`/`[directory]` blocks). Expressions never span multiple lines — an
operator at the start of a new line is never treated as a continuation of
the expression on the previous line.

## Statement terminators

| Statement kind | Trailing `.` |
|---|---|
| Assignment (`x = expr`) | No |
| Input (`x = ? "prompt"`) | No |
| `return` | No |
| Bare function call statement | No |
| `show` | **Yes** |
| Data-structure statement ops (`add`/`remove`/`delete`/`sort`/`reverse`/`insert`) | **Yes** |
| `min of`/`max of`/`length of` as a statement | **Yes** |
| `is ... [at ...]` | **Yes** |
| `import`, `sys`, `break`, `continue`, `def`/`if`/`loop` headers and `[end]` markers | No |

## Expressions

Precedence, low to high:

| Level | Operators |
|---|---|
| 1 (lowest) | `\|\|` |
| 2 | `&&` |
| 3 | `==` `!=` |
| 4 | `<` `<=` `>` `>=` |
| 5 | `+` `-` |
| 6 | `*` `/` `%` |
| 7 | unary `-` `!` |
| 8 (highest) | primary: literals, identifiers, calls, `(...)` grouping |

`+` is overloaded: numeric addition when both operands are numbers; on two
lists, sets, or maps it combines them (below); otherwise string
concatenation (either side coerced via its natural string form).
`-` and `*` require two numbers. `/` does integer division when both sides
are integers, float division otherwise; division by zero is a fatal error.
`%` is modulo: integer `%` integer stays an integer, anything else falls
back to floating-point modulo (Go's `math.Mod`); modulo by zero is a fatal
error, same as division by zero.
`<`/`>`/`<=`/`>=` work on two numbers or two strings (lexicographic).
`==`/`!=` compare numbers by value (`1 == 1.0`), functions by identity
(the same definition), lists element by element in order, and sets and
maps regardless of order (`set [1, 2] == set [2, 1]`). Everything else
needs the same type and value, so `1 != "1"`, `none == none`, and `none`
never equals `0`, `""`, or `"none"`. The same equality is used for set
deduplication and membership and for `count`/`index`/`find`/`remove`.
Map keys can be any value and keep their type: in `map [1: "a", "1":
"b"]` the integer `1` and the string `"1"` are two different keys, and
looping over a map or `getKeys` gives back integers as integers. Equal
values are the same key (`1` and `1.0`; `set [1, 2]` and `set [2, 1]`).

Values show as their type. An integer shows as `4`, a float always with a
decimal point, `4.0`, so `2.5 * 2` shows `5.0`. Inside a list, set, map
or assembled value, text is quoted so it can't be mistaken for a number:
`show list [1, "1"] .` prints `[ 1, "1" ]`. Text on its own isn't quoted:
`show "hi" .` prints `hi`.

Integers are 64-bit, the industry standard (Java's and C#'s `long`, Go's
`int64`, Rust's `i64`): from `-9223372036854775808` to
`9223372036854775807`. A result past either limit is an "integer
overflow" error, never a silent wrap-around; use a float (`1.0`) for
bigger numbers. An integer literal past the limit is a parse error.

Truthiness (conditions, `&&`, `||`, `!`), the Python rule: `false`,
`none`, `0`, `0.0`, `""`, and an empty list, set or map are falsy;
everything else is truthy. So `if ] matches [` means "if there are any
matches". `matches at isEmpty` asks the same thing explicitly.

`+` and `-` on two collections of the same kind always make a new one;
neither side changes:

| Expression | Result |
|---|---|
| `list + list` | joined, in order |
| `list - list` | the left list without any element that's in the right one |
| `set + set` | union |
| `set - set` | difference |
| `map + map` | all entries; on a key both have, the **left** map's value wins |
| `map - map` | the left map without the right map's **keys** (values ignored) |

`<expr> at <method>` calls a method inside any expression:
`show name at upper .`, `w = x at slice[0, 3]`. In this form, method
arguments go in brackets. The statement form `r is x at slice 0, 3 .` still
takes them unbracketed. `at` binds more loosely than every operator, so
`a + b at upper` is `(a + b) at upper`.

Function calls: `<name>[<expr>, ...]` — Turtle uses `[...]` for call and
definition argument lists, not `(...)`. See also sentence-style calls
under Functions.

## Variables

```
<ident> = <expr>
```

## Input

```
<ident> = ? <string>
```

Prints the prompt, reads one line from stdin, assigns it as a string.

## Type conversion

```
change <ident> to <type> .            // mutates <ident> in place
<ident> = change <expr> to <type>     // expression form, no mutation
```

`<type>` is one of `integer`, `float`, `string`, `ascii`, `char`, `hex`.
These are ordinary identifiers, not reserved words — they only mean
anything right after `change ... to`.

The statement form requires the source to be a plain identifier (like the
target of `add ... to <ident> .`) and rewrites that variable's value
in place. The expression form works anywhere an expression can — assigned
to a different variable, nested inside a larger expression, passed as a
call argument — and never mutates its source:

```
raw = ? "Enter a number: "
n = change raw to integer
show n + 1 .
```

Which conversion runs is inferred from the value's current type, so one
target name covers both directions of a pair:

| Target | From | Result |
|---|---|---|
| `integer` | string | parsed as base-10 (fatal if invalid) |
| `integer` | float | truncated toward zero |
| `float` | string | parsed as a float |
| `float` | integer | widened |
| `string` | integer / float | display form |
| `ascii` | one-character string | its code point, as an integer |
| `char` | integer | the one-character string for that code point |
| `hex` | integer | lowercase hex digits, no `0x` prefix |
| `hex` | string | parsed as hex (`0x` prefix optional) back to an integer |

Any other combination is a fatal error naming the source and target types.

## Show

```
show <expr> {, <expr>} .
```

Each piece is evaluated and its display form concatenated, in order, with
no separator inserted.

## Return / break / continue

```
return <expr>
return
break
continue
```

`return` ends the current function call; the value is usable directly at
the call site. A bare `return` (nothing else on the line), or reaching the
end of the body without a `return`, yields `none`. `break`/`continue` only affect the nearest enclosing loop;
outside a loop they're a no-op.

## Functions

```
def <name>[<param>, ...]
    <statement>
    ...
def [end]
```

Call: `<name>[<expr>, ...]`. Each call gets a fresh scope seeded with its
parameters. At most 100,000 calls can be in progress at once; past that
is a "recursion too deep" error (usually a recursive function missing its
stopping case). Scoping is **lexical**: a name that isn't a parameter or local
is looked up in the scope the function was *defined* in — for a top-level
function that's the global scope, so every function body can read
top-level variables; it never sees the *caller's* locals. A parameter/local
of the same name shadows the outer one. Assignment always writes to the
call's own local scope, never an outer one, so `x = ...` inside a function
can't clobber a global `x` — it just shadows it for the rest of that call.
Mutating an outer `list`/`set`/`map` via a data-structure statement or
method call *does* affect it, since that mutates the same underlying value
rather than rebinding a name. The body can call any other top-level
function, including itself, recursively. Argument count must match
parameter count.

### Functions are values

A function's name, used without `[...]`, is the function itself: it can be
assigned, passed as an argument, returned, and compared with `==`. Calling
`<name>[...]` resolves `name` first as a variable holding a function, then
as a top-level function, then as a builtin (`now`, `sleep`).

```
def double[x]
    return x * 2
def [end]

def apply[f, v]
    return f[v]
def [end]

show apply[double, 21] .    // 42
g = double
show g[4] .                 // 8
```

A result must be stored before it's called — `g = make[]` then `g[]`;
`make[][]` isn't valid syntax.

### Closures

A `def` inside a function body defines a **closure**: a local variable
holding a function that remembers the scope it was defined in, so it can
read the enclosing call's parameters and locals even after that call has
returned. Nested defs are local to their enclosing call, not global.

```
def make_adder[n]
    def adder[x]
        return x + n
    def [end]
    return adder
def [end]

add5 = make_adder[5]
show add5[1] .              // 6
```

Captured variables are **read-only** by assignment, following the same rule
as globals: `n = ...` inside `adder` creates a local `n` instead of
changing the captured one. To keep mutable state in a closure, capture a
list/set/map and mutate it in place:

```
def make_counter[]
    counts = list [0]
    def inc[]
        c is counts at pop .
        c = c + 1
        add c to counts .
        return c
    def [end]
    return inc
def [end]

tick = make_counter[]
show tick[] .               // 1
show tick[] .               // 2
```

### Anonymous functions: `gives`

```
x gives <expr>             // one parameter
[a, b] gives <expr>        // several, bracketed like a def
[] gives <expr>            // none
[x] gives                  // block form: a whole body, closed by gives [end]
    <statement> ...
gives [end]
```

`gives` makes a function without a name. It's a value like any other
function, and it can read the variables around it (it's a closure).

```
double = x gives x * 2
show double[4] .                     // 8
plus = [a, b] gives a + b
show apply[x gives x + 1, 3] .       // 4

big = [x] gives
    if ] x > 3 [
        return true
    if [end]
    return false
gives [end]
```

The expression form's body runs to the end of the expression, so inside a
call's `[...]` it stops at the next `,` or `]`.

### Sentence-style calls

```
<variable> <function> [<arg>, ...]
```

calls `<function>[<variable>, <arg>, ...]`: the variable on the left
becomes the first argument. It works with any function: your own, an
imported one, or a builtin like `data`'s `process`.

```
nums process x gives x + 1 .         // process[nums, x gives x + 1]
r = nums scale x gives x + 1, 10     // scale[nums, x gives x + 1, 10]
show nums total .                    // total[nums]
```

- The left side is a variable name, or a string or number literal
  (`"lo" isinstring line`). It can't be a call: store the result first.
- A negative first argument works when the `-` is attached to it:
  `s substring -5`. A spaced `-` (`a b - 1`) or one attached to the name
  (`a b-1`) means subtraction.
- A trailing `.` is optional when the call is a whole statement.
- `a b` is ambiguous with a module-qualified name (`time now`). It's
  settled when the code runs: if `a` is an imported module, it's that
  module's function; if `a` is a variable, it's a sentence-style call. A
  name that's both is a fatal error.
- Outside brackets, a sentence's arguments run to the end of the
  expression, `,`-separated: `show t substring 0, 5 .`. Inside `[...]`
  (call arguments, list/set/map literals) it takes **one** argument, so
  the commas stay the outer list's: `check["x", t find "W", 7]` is
  `check["x", find[t, "W"], 7]`. For two or more arguments there, use the
  bracket form: `substring[t, 0, 5]`.
- To call an imported function qualified in this style, use the bracket
  form instead: `data process[nums, f]`.

## Assembled types

```
assemble <Name> [<field>, ...]

<Name>[<expr>, ...]          // make one: exactly one value per field, in order
<field> of <expr>            // read a field
<field> of <expr> = <expr>   // change a field
```

`assemble` declares a named type with fixed fields, and `<Name>` becomes
its constructor: a function that takes one value per field.

```
assemble Order [item, qty, price]

o = Order["pen", 3, 1.5]
show o .                     // Order { item: "pen", qty: 3, price: 1.5 }
show qty of o * price of o . // 4.5
qty of o = 10
```

- **Fields hold any value**, including other assembled values. Field
  access chains right to left: `x of finish of line` is
  `x of (finish of line)`, and `x of finish of line = 7` changes it.
  `of` binds tighter than any operator, so `qty of o * price of o`
  multiplies two fields. `length of`/`min of`/`max of` bind the same way.
- **Errors name the fields.** A wrong number of values gives "Order needs
  3 value(s), one per field (item, qty, price), got 2". An unknown field
  gives "Order has no field "prise" (its fields: item, qty, price)".
  Using `of` on a value that isn't assembled is also an error.
- **Assembled values are references**, like lists and maps: after `b = a`,
  changing a field through `b` changes `a` too. `data`'s `copy[a]` makes
  an independent copy.
- **Equality**: two values are `==` when they come from the same
  `assemble` declaration and their fields are equal. Same-named types
  from different declarations are never equal. Equal values are one
  element in a set and the same map key.
- **The constructor is a function**, so it follows the function rules: a
  top-level `assemble` is exported by its module and imported like any
  function (`import shapes [Point]`, `shapes Point[1, 2]` on a clash); one
  inside a function body is local to it; `mk = Order` stores it as a
  value; and `show Order .` prints `assemble Order`.
- Field names must be distinct, and can't be reserved words.

## None

`none` is Turtle's "no value". It's what a function returns when it has no
`return` or uses a bare `return`, and what the time module's `sleep[...]`
returns. It shows as `none`, is falsy, equals only itself, and concatenates
as `"none"` (`"x=" + none` is `"x=none"`). Arithmetic and ordering
comparisons on `none` are fatal type errors.

## Conditionals

```
if ] <expr> [
    <statement> ...
{else if ] <expr> [
    <statement> ...}
[else ]
    <statement> ...]
if [end]
```

Note the reversed brackets around the condition: `]` then the condition
then `[`.

**Nesting**: write a nested if/else-if/else chain as a statement inside an
enclosing clause's body, with one extra leading `[` on every keyword of the
nested chain:

```
[if ] <expr> [
    ...
{[else if ] <expr> [
    ...}
[[else ]
    ...]
```

A nested chain has no `[end]` of its own — it implicitly ends the moment a
clause at the same-or-shallower bracket depth appears. This generalizes to
arbitrary depth by adding another leading `[` per level (only depth 0 and 1
appear in any known real program).

## Loops

```
[loop][<header>]
    <statement> ...
[loop][end]
```

`<header>` is one of:

```
<ident> = <expr> ; <expr> ; <post>      // C-style: init; condition; post
<expr>                                   // while-style: condition only
<ident> in <expr>                        // for-each
<ident>, <ident> in <expr>               // for-each with index or key
```

For-each walks a list or set (each element), a string (each character),
or a map (each key). With two names it gives `index, element`, or
`key, value` for a map:

```
[loop][x in nums]
    show x .
[loop][end]

[loop][name, age in ages]
    show name, " is ", age .
[loop][end]
```

It walks a snapshot of the collection, so adding to or removing from it
inside the loop can't make it skip or repeat elements. `break`,
`continue`, and `return` work as in the other loops.

Loop names (`x`, or `k, v`) belong to the loop: a variable of the same
name outside isn't touched, and they're gone once the loop ends. Each pass
has its own copy, so a function made inside the loop
(`add [] gives x to fs .`) keeps the value it saw. Every other assignment
in the body works as if the loop weren't there: `t = t + x` updates `t`
outside.

where `<post>` is `<ident>++`, `<ident>--`, or `<ident> = <expr>`.

Every `[loop][...]` — nested or not — requires its own matching
`[loop][end]`; there is no shared-closer shortcut. The induction variable
(C-style form) is a real, live variable, visible and updated in `show`/
expressions on every pass. Like for-each names, it belongs to the loop: a
nested loop reusing the same name doesn't clobber its parent's counter, an
outside variable of that name is untouched, and it's gone after the loop.

## Data structures

See [`stdlib.md`](stdlib.md) for full method tables. Declaration:

```
<ident> = list [<expr>, ...]
<ident> = set [<expr>, ...]
<ident> = map [<expr>:<expr>, ...]
```

Statement-form operations (all require a trailing `.`):

```
add <expr> to <ident> .
remove <expr> from <ident> .
delete <expr> from <ident> .      // map only
sort <ident> .
reverse <ident> .
insert <expr> to <ident> at <expr> .
length of <expr> .                 // prints
min of <expr> .                     // prints
max of <expr> .                      // prints
```

Method-call form:

```
<ident> is <expr> [at <method> [<expr> {, <expr>}]] .
```

Plain `<ident> is <expr> .` (no `at`) is assignment/aliasing.

## Files

```
[read] <path> to <ident> [end]

[write] <path>
<string-or-ident>
...
[end]

[append] <path>
<string-or-ident>
...
[end]

[directory] <path> to <ident> [end]
```

`<path>` is a quoted string or a bareword like `file.txt` / `data/in.csv`.
A bareword that's a single name with no `.` or `/` (`[read] name to
lines`) uses the variable of that name if one exists, so a path from
`args[]` or built at runtime works; otherwise it's that literal filename.
Relative paths resolve from the folder `turtle` was run in (see
[`stdlib.md`](stdlib.md#system-library)).
`[write]`/`[append]` body lines are one item each: a quoted string is
written verbatim, a bare identifier is replaced with that variable's
current value; items are newline-joined.

## Modules

```
import <name>                  // everything the module exports
import <name> [<f>, <g>, ...]  // only the listed names
```

`<name>` is a builtin module (`math`, `time`) or a file `<name>.t`,
resolved relative to the current script's directory. Because builtin names
win, don't name your own module file `math.t` or `time.t`.

**What a module exports.** A `.t` module exports its top-level functions,
and only those. It runs once, in its own global scope, the first time any
file imports it; later imports reuse it. Its top-level variables stay
private to it, although its own functions can read them. A function you
import keeps calling the module's other functions, even ones you didn't
import, and those never leak into your program. A module's own imports
aren't passed on to the files that import it.

**Plain and qualified names.** Imported functions are called by their
plain name: `t = now[]`. You can always name the module explicitly by
writing it in front, separated by a space: `t = time now[]`. Without
brackets, `mylib binary` is the function itself as a value, the same as a
plain function name.

**Clashes.** Importing two modules that export the same function name is
fine. Calling that name *unqualified* is a fatal error that asks you to
choose: `"now" is provided by more than one import (time, mylib) — say
which one, e.g. time now[...]`. Every other name from those modules keeps
working unqualified. Your own top-level `def` always wins over an imported
function with the same name, and the module's version stays reachable
qualified (`time now[]`).

**Import lists.** `import time [sleep, now]` makes only the listed names
available, plain or qualified. Anything else from that module is a fatal
error naming the list to add it to. A listed name the module doesn't
export is a fatal error at the `import` line. `import m []` is a parse
error. Importing the same module again merges: lists accumulate, and a
full import makes everything available.

Imports apply per file: the main program and each module only see what
they themselves imported. A circular import (`a` imports `b` imports `a`)
is a fatal error that shows the chain.

**Builtin modules.**

- `import math` unlocks number methods: `sqrt`, `abs`, `round`, `floor`,
  `ceil`, `pow`, `random` (see [`stdlib.md`](stdlib.md#number)). Methods
  are called on a value (`r is 16 at sqrt .`), so they never clash and
  are never qualified; an import list still limits which ones you can use
  (`import math [sqrt]`).
- `import time` provides two builtin functions: `now[]` (milliseconds
  since the Unix epoch, as an Integer) and `sleep[<amount> [, <unit>]]`
  (pauses; `<unit>` is `"seconds"`, the default, or `"ms"`; returns
  `none`). Builtin functions can be called but not used as values.
- `import system` provides `args[]` (the command-line arguments after the
  script path), `exists`/`isFile`/`isFolder`, `contents[path]`,
  `exit[code]`, `env[name]` and `scriptFolder[]` (see
  [`stdlib.md`](stdlib.md#system-library)).
- `import strings` provides `find`, `substring`, `isinstring` and `join`
  (see [`stdlib.md`](stdlib.md#strings-library)).
- `import data` provides `process`, `keep`, and `copy`, which apply a
  function across a list, set, or map (see
  [`stdlib.md`](stdlib.md#data-library)). They're ordinary functions,
  usually called sentence-style: `nums process x gives x + 1 .`

Using a math method or `now`/`sleep` without the matching import is a fatal
error naming exactly which import is missing.

```
import time
import mylib [binary, now]

r = binary[5]          // only mylib has binary
t = time now[]         // both have now: name the module
m = mylib now[]
sleep[0.5]             // only time has sleep
```

## Shell escape

```
sys <rest of line>
```

`sys` must be the first word of the statement. Everything after it,
verbatim to end of line, runs through a shell with inherited stdin/stdout/
stderr.
