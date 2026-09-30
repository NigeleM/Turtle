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
- **Reserved words** (cannot be used as identifiers): `true false show if
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
`==`/`!=` compare numbers by value, everything else by type + string form.

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
break
continue
```

`return` ends the current function call; the value is usable directly at
the call site. `break`/`continue` only affect the nearest enclosing loop;
outside a loop they're a no-op.

## Functions

```
def <name>[<param>, ...]
    <statement>
    ...
def [end]
```

Call: `<name>[<expr>, ...]`. Each call gets a fresh scope seeded with only
its parameters and no closure over the *caller's* locals — but variable
lookup falls through to the global scope when a name isn't a parameter or
local, so a function body can read top-level variables (a parameter/local
of the same name shadows the global). Assignment always writes to the
call's own local scope, never the global, so `x = ...` inside a function
can't clobber a global `x` — it just shadows it for the rest of that call.
Mutating a global `list`/`set`/`map` via a data-structure statement or
method call *does* affect the global, since that mutates the same
underlying value `Get` returned rather than rebinding a name. The body can
call any other top-level function, including itself, recursively.
Argument count must match parameter count.

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
import <ident>
```

Parses and evaluates `<ident>.t` in the current global scope before
continuing — **except** for two reserved builtin module names, `math` and
`time`, which don't read a file at all: `import math`/`import time` just
enable a native capability for the rest of the program. Because of this,
avoid naming your own module file `math.t` or `time.t` — those names
always resolve to the builtin, never to a file.

- `import math` unlocks number methods: `sqrt`, `abs`, `round`, `floor`,
  `ceil`, `pow`, `random` (see [`stdlib.md`](stdlib.md#number)). Calling
  any of these before `import math` is a fatal error naming exactly which
  import is missing.
- `import time` unlocks two builtin functions: `now[]` (milliseconds since
  the Unix epoch, as an Integer) and `sleep[<amount> [, <unit>]]` (pauses;
  `<unit>` is `"seconds"`, the default, or `"ms"`). Same gating behavior —
  used before `import time`, both fail with a clear "needs import time"
  error. If you've already defined your own function named `now` or
  `sleep`, yours always wins; the builtin
  only applies when no user function of that name exists.

## Shell escape

```
sys <rest of line>
```

`sys` must be the first word of the statement. Everything after it,
verbatim to end of line, runs through a shell with inherited stdin/stdout/
stderr.
