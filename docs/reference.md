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
  directory break continue`. Type names used after `change ... to` —
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

`+` is overloaded: numeric addition when both operands are numbers, string
concatenation otherwise (either side coerced via its natural string form).
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
Map keys are stored by their shown form, except that equal sets always
make the same key.

Truthiness (conditions, `&&`, `||`, `!`): `false`, `0`, `0.0`, `""`, and
`none` are falsy; everything else is truthy.

Function calls: `<name>[<expr>, ...]` — Turtle uses `[...]` for call and
definition argument lists, not `(...)`.

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
parameters. Scoping is **lexical**: a name that isn't a parameter or local
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
```

where `<post>` is `<ident>++`, `<ident>--`, or `<ident> = <expr>`.

Every `[loop][...]` — nested or not — requires its own matching
`[loop][end]`; there is no shared-closer shortcut. The induction variable
(C-style form) is a real, live variable, visible and updated in `show`/
expressions on every pass; it's automatically saved and restored around the
loop, so a nested loop reusing the same variable name as its parent doesn't
clobber the parent's iteration.

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
