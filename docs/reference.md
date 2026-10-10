# Language reference

The complete, formal syntax and semantics of Turtle, as implemented by the
`token`/`lexer`/`ast`/`parser`/`object`/`evaluator` packages. For a gentler,
example-first introduction see [`tour.md`](tour.md); for the standard
library see [the library docs](library/index.md).

Notation below: `code` is a literal token; `<angle-brackets>` is a
non-terminal; `[x]` is optional; `{x}` is zero-or-more; `|` is alternation.

## Lexical grammar

- **Files**: Turtle programs, libraries and tests end in `.turtle`
  (`report.turtle`, `lib/utils.turtle`, `test_orders.turtle`). The older
  ending `.trt` is accepted too, everywhere `.turtle` is: `import utils`
  finds `utils.turtle` or `utils.trt` (both at once is an error, so an
  import never quietly picks one), and `turtle test` runs `test_*.trt`
  files as well.
- **Comments**: `// ...` runs to end of line. `//* ... *//` is a block
  comment that may span multiple lines. Both are stripped by the lexer.
- **Identifiers**: `<letter|_> {letter|digit|_}`. Turtle's own names are
  all lowercase (`isempty`, `scriptfolder`, `sql_open`).
- **Numbers**: `<digits>` (integer) or `<digits>.<digits>` (float), and
  scientific notation, always a float: `1e-18`, `2.5e6`, `6.02E+23`.
- **Strings**: `"double"` or `'single'` quoted (the same kind of string;
  inside single quotes a `"` needs no escape). Escapes: `\n`, `\t`, `\"`,
  `\'`, `\\`, `\{`, `\}`.
  `{<expr>}` inside a string is interpolation (see §Strings). A string
  may go over several lines (SQL, a message): the line breaks are part
  of the text.
- **Raw strings**: `` `backticks` `` keep every character as typed: no
  escapes, no interpolation, and they may span lines. For patterns
  (`` `\d{3}` ``) and Windows paths (`` `C:\new\table` ``). A raw string
  can't hold a backtick.
- **Booleans**: `true`, `false`.
- **None**: `none` — the single "no value" value (type `NONE`).
- **Reserved words** (cannot be used as identifiers; each is explained in
  [Keywords](#keywords)): `true false none show if
  else def end loop return list set map import sys to from at of is add
  change remove delete sort reverse insert min max length read write append
  directory break continue give in assemble safe handle fail warn div scroll`. Type names used after `change ... to` —
  `integer`, `float`, `string`, `ascii`, `char`, `hex` — are **not** reserved;
  like method names (`get`, `union`, ...) they're plain identifiers whose
  meaning is only special right after `to`.

One statement per source line, except explicit multi-line blocks with their
own begin/end markers (function/if/loop bodies, `[write]`/`[append]`/
`[read]`/`[directory]` blocks), and text, which may go over several lines.
Otherwise expressions never span multiple lines — an
operator at the start of a new line is never treated as a continuation of
the expression on the previous line.

## Keywords

Every word Turtle reserves, what it does, and where it's explained. A
reserved word can't name a variable or a function (`length = 3` is an
error that says so).

### Reserved words

| Word | What it does | Example | See |
|---|---|---|---|
| `add` | adds an item to a list or set (or a key to a map) | `add 4 to nums .` | [Data structures](#data-structures) |
| `append` | adds lines to the end of a file | `[append] log.txt` ... `[end]` | [Files](#files) |
| `assemble` | names a kind of value with fields | `assemble Order [item, qty]` | [Assembled types](#assembled-types) |
| `at` | calls a method on the value before it | `nums at get[0]`, `name at upper` | [Expressions](#expressions) |
| `break` | leaves a loop | `break` | [Return / break / continue](#return--break--continue) |
| `change` | converts a value to another type | `n = change "42" to integer` | [Type conversion](#type-conversion) |
| `continue` | goes on to a loop's next pass | `continue` | [Return / break / continue](#return--break--continue) |
| `def` | defines a function | `def total[a, b]` ... `def [end]` | [Functions](#functions) |
| `delete` | removes a key from a map | `delete "Bo" from ages .` | [Data structures](#data-structures) |
| `directory` | lists a folder's entries | `[directory] data to names [end]` | [Files](#files) |
| `div` | whole-number division | `7 div 2` is `3` | [Expressions](#expressions) |
| `else` | the other branches of an `if` | `else if ] x > 2 [`, `else ]` | [Conditionals](#conditionals) |
| `end` | closes a block | `def [end]`, `if [end]`, `[loop][end]` | [Conditionals](#conditionals) |
| `fail` | raises your own error | `fail "no stock"` | [Errors](#errors-safe--handle--fail) |
| `false` | the boolean false | `done = false` | [Lexical grammar](#lexical-grammar) |
| `from` | in sentences: where from | `remove 1 from nums .` | [Data structures](#data-structures) |
| `give` | makes a function without a name | `x give x + 1`, `[a, b] give a + b` | [Anonymous functions](#anonymous-functions-give) |
| `handle` | the part of a `safe` block that runs on an error | `handle [file] e .` | [Errors](#errors-safe--handle--fail) |
| `if` | runs code when a condition holds | `if ] x > 1 [` ... `if [end]` | [Conditionals](#conditionals) |
| `import` | brings in a library or your own file | `import sql`, `import lib/utils`, `import time [now]` | [Modules](#modules) |
| `in` | the items a loop goes through | `[loop][x in nums]` | [Loops](#loops) |
| `insert` | puts an item into a list at a position | `insert 9 to nums at 0 .` | [Data structures](#data-structures) |
| `is` | assigns, in sentence form | `r is nums at get 0 .` | [Expressions](#expressions) |
| `length` | how many items or characters | `length of nums` | [Data structures](#data-structures) |
| `list` | makes a list | `list [1, 2, 3]` | [Data structures](#data-structures) |
| `loop` | repeats code | `[loop][x in nums]`, `[loop][i = 0; i < 3; i++]` | [Loops](#loops) |
| `map` | makes a map of keys to values | `map ["a": 1]` | [Data structures](#data-structures) |
| `max` | the largest item | `max of nums` | [Data structures](#data-structures) |
| `min` | the smallest item | `min of nums` | [Data structures](#data-structures) |
| `none` | no value | `x = none` | [None](#none) |
| `of` | a field or key of a value | `qty of order`, `length of nums` | [Assembled types](#assembled-types) |
| `read` | reads a file's lines | `[read] notes.txt to lines [end]` | [Files](#files) |
| `remove` | removes an item from a list or set | `remove 4 from nums .` | [Data structures](#data-structures) |
| `return` | gives a function's answer back | `return total` | [Return / break / continue](#return--break--continue) |
| `reverse` | reverses a list in place | `reverse nums .` | [Data structures](#data-structures) |
| `safe` | runs code that may fail, with `handle` for the errors | `safe` ... `handle [] e .` ... `safe [end]` | [Errors](#errors-safe--handle--fail) |
| `scroll` | steps a value goes through, in order | `x is scroll 3 into add1, double .` | [Scrolls](#scrolls) |
| `set` | makes a set: no repeats | `set [1, 2]` | [Data structures](#data-structures) |
| `show` | prints values | `show "total: ", n .` | [Show](#show) |
| `sort` | sorts a list in place (`import sort` for more) | `sort nums .` | [Data structures](#data-structures) |
| `sys` | runs a shell command | `sys ls -la` | [Shell escape](#shell-escape) |
| `to` | in sentences: where to, or what into | `add 4 to nums .`, `change x to integer` | [Type conversion](#type-conversion) |
| `true` | the boolean true | `done = true` | [Lexical grammar](#lexical-grammar) |
| `warn` | prints to the error stream (`import system`) | `warn "careful" .` | [Show](#show) |
| `write` | writes a file's lines | `[write] notes.txt` ... `[end]` | [Files](#files) |

### Words an import turns on

These mean something only in a file with the import; anywhere else
they're ordinary names. In such a file they can't name a function.

| Word | Import | What it does | Example | See |
|---|---|---|---|---|
| `random` | `import random` | a random value of any shape | `random list of 5 integers from 0 to 9` | [the `random` library](library/random.md) |
| `check` | `import test` | one fact a test checks | `check total[o] == 45 .` | [Tests](#tests-check-verify-validate-turtle-test) |
| `verify` | `import test` | a rule over every item | `verify evens[nums] each x give x % 2 == 0 .` | [Tests](#tests-check-verify-validate-turtle-test) |
| `validate` | `import test` | a rule over random inputs | `validate evens[nums] with nums as list of integer that ...` | [Tests](#tests-check-verify-validate-turtle-test) |
| `log` | `import log` | a log line, with a level | `log warn "disk at ", pct, "%" .` | [Logging](#logging-log) |
| `matrix` | `import linear` | makes a matrix | `m = matrix [1, 2; 3, 4]` | [the `linear` library](library/linear.md) |

### Words special only in one place

Ordinary names everywhere else, so a variable can still be called `put`
or `here`.

| Word | Where | Example | See |
|---|---|---|---|
| `put` | first on a line: replace an item | `put 9 to nums at 2 .`, `put 9 to m at 1, 2 .` | [Data structures](#data-structures) |
| `into` | in a scroll: the steps follow | `x is scroll 3 into add1, double .` | [Scrolls](#scrolls) |
| `here` | in a scroll step: the value so far | `here at get[0]` | [Scrolls](#scrolls) |
| `diagnose` | a line of its own, or `diagnose[...]`: look inside | `diagnose` ... `diagnose [end]` | [diagnose](#looking-inside-diagnose) |
| `type` | after a value: whether it's that kind, true or false | `if ] n type integer [` | [The kind of a value](#the-kind-of-a-value-typeof-type) |
| `integer`, `float`, `string`, `ascii`, `char`, `hex`, `keys`, `values` | after `change ... to` | `change x to float` | [Type conversion](#type-conversion) |
| `each`, `any`, `not`, `exactly`, `least`, `most`, `pair`, `that`, `with`, `as`, `matches`, `fails`, `close`, `within` | in `check`, `verify` and `validate` sentences | `verify xs at least 2 x give x > 0 .` | [Tests](#tests-check-verify-validate-turtle-test) |
| `debug`, `info`, `error` | right after `log` | `log debug "x is ", x .` | [Logging](#logging-log) |
| `theory`, `abstract`, `notation`, `definition`, `theorem`, `proof` | a theory, and its sections | `theory tally` ... `theory [end]` | [Theories](#theories) |

## Statement terminators

| Statement kind | Trailing `.` |
|---|---|
| Assignment (`x = expr`) | No |
| Input (`x = ?`, `x = ? "prompt"`) | Optional |
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
| 6 | `*` `/` `div` `%` |
| 7 | unary `-` `!` |
| 8 | a method: `x at m` (works on the value right before it) |
| 9 (highest) | primary: literals, identifiers, calls, `(...)` grouping |

Operators on the same level go left to right: `10 - 4 - 3` is `3`.

`+` is overloaded: numeric addition when both operands are numbers; on two
lists, two sets, or two maps it combines them (below); otherwise string
concatenation (either side coerced via its natural string form). A list,
set or map added to anything else, including text or a different kind of
collection, is a `type` error: `list [1] + 1` and `"n=" + nums` stop the
program. `none` only adds to `none` (see §None). Values taken out of a list add like any other value.
`-` and `*` require two numbers.

**Division** has two operators, one for each kind:

| Expression | Result | |
|---|---|---|
| `7 / 2` | `3.5` | `/` always gives the exact answer, a float |
| `6 / 3` | `2.0` | even when it comes out whole |
| `7 div 2` | `3` | `div` gives the whole part, an integer |
| `-7 div 2` | `-3` | toward zero (Python's `//` gives `-4`) |
| `7.9 div 2` | `3` | works on floats too; still an integer |
| `7 % 2` | `1` | the remainder |
| `-7 % 2` | `-1` | the remainder that goes with `div`: `(a div b) * b + a % b` is `a` |

Division or `div` by zero is a `math` error. `%` of two integers is an
integer; with a float on either side it's a float (`10.5 % 3` is `1.5`);
`%` by zero is a `math` error too. (`//` starts a comment, so Turtle's
whole-number division is the word `div`.)
`<`/`>`/`<=`/`>=` work on two numbers or two strings (lexicographic).
`==`/`!=` compare numbers by value (`1 == 1.0`), functions by identity
(the same definition), lists element by element in order, and sets and
maps regardless of order (`set [1, 2] == set [2, 1]`). Everything else
needs the same type and value, so `1 != "1"`, `none == none`, and `none`
never equals `0`, `""`, or `"none"`. The same equality is used for set
deduplication and membership and for `count`/`index`/`find`/`remove`.
Map keys can be any value and keep their type: in `map [1: "a", "1":
"b"]` the integer `1` and the string `"1"` are two different keys, and
looping over a map or `getkeys` gives back integers as integers. Equal
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
matches". `matches at isempty` asks the same thing explicitly.

`&&` and `||` stop as soon as the answer is known, as in other languages:
in `x != none && x > 5` the `x > 5` only runs when `x` isn't `none`, and
in `found || search[]` the search only runs when nothing was found.
Either way the result is `true` or `false`.

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
`show name at upper .`, `w = x at slice[0, 3]`. One argument can go
without brackets: it's just the value right after the method, so
`3 at pow 2 == 9` compares the power, `s at contains "a" && ok` asks
both, and `list [x at get 0, 1]` is two items. More than one argument
goes in brackets (`x at slice[0, 3]`), except in the statement form,
`r is x at slice 0, 3 .`, which takes them unbracketed.

**A method works on the value right before it**, with or without
brackets, the way Python's `a.invert()` does. Use `( )` to apply it to a
whole calculation:

| Turtle | Means | Python |
|---|---|---|
| `a at invert == b at invert` | both maps inverted, then compared | `a.invert() == b.invert()` |
| `"a" + "b" at upper` | `"aB"` | `"a" + "b".upper()` |
| `("a" + "b") at upper` | `"AB"` | `("a" + "b").upper()` |
| `2 + 16 at sqrt * 2` | `10.0` | `2 + sqrt(16) * 2` |
| `nums at get[0] + nums at get[2]` | adds two items | `nums[0] + nums[2]` |
| `-7 at abs` | `7`: `-7` is one number | `abs(-7)` |
| `-x at abs` | `-(x at abs)` | `-abs(x)` |
| `!r at isempty` | "r is not empty" | `not r.isempty()` |
| `title of b at upper` | the title, upper-cased (`title of b` is one value) | `b.title.upper()` |

The same rule holds in the statement form: `r is "a" + "b" at upper .`
is `"aB"`, like `r = "a" + "b" at upper`. Methods that give a new value
(`invert`, `upper`, `abs`, ...) leave the variable as it was; list, set
and map methods that change the collection (`add`, `sort`, `remove`,
...) change it, as before.

**Order of operations**, highest first: `( )`; a method (`at`); `-` and
`!` in front of a value; `*` `/` `div` `%`; `+` `-`; `<` `>` `<=` `>=`;
`==` `!=`; `&&`; `||`. Operators of the same level go left to
right: `10 - 4 - 3` is `3`, `100 / 10 / 5` is `2`. Expressions can be
as long and nested as you like:
`1 + 2 - 3 + 4 - (5 + 6) + 7 * 8` is `49`,
`2 * (3 + 4) * (5 - (6 - 7))` is `84`.

Function calls: `<name>[<expr>, ...]` — Turtle uses `[...]` for call and
definition argument lists, not `(...)`. See also sentence-style calls
under Functions.

## Variables

```
<ident> = <expr>
<ident> = <expr> .
```

The closing period is optional, as after a function call; it reads
naturally after a sentence: `big = nums keep n give n > 10 .`

## Input

```
<ident> = ?
<ident> = ? <string>
```

Reads one line from stdin and assigns it as a string, like Python's
`input()`. With a string, prints it first as the prompt. A closing period
is optional:

```
a = ?
age = ? "Age for {name}? " .
n = change age to integer
```

## Type conversion

```
change <ident> to <type> .            // mutates <ident> in place
<ident> = change <expr> to <type>     // expression form, no mutation
```

`<type>` is one of `integer`, `float`, `string`, `ascii`, `char`, `hex`,
`list`, `set`, `keys`, `values`, and, in a file that imports `linear`,
`matrix`. All but `list` and `set` are ordinary identifiers, not reserved
words — they only mean anything right after `change ... to`.

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
| `set` | list | a new set: duplicates dropped, each kept where it first appeared |
| `list` | set | a new list, in the set's order |
| `list` / `set` | the same kind | a copy, so changing it doesn't change the original |
| `keys` | map | a new list of its keys, in order |
| `values` | map | a new list of its values, in order |
| `list` / `set` | map | its keys (like `keys`) |
| `list` / `set` | string | its characters: `change "abc" to list` is `[ "a", "b", "c" ]` |
| `matrix` | a list of lists (rows), or of maps (what `table_read` gives) | a new matrix (`import linear`) |
| `list` | matrix | a list of its rows, each a list |

Any other combination is a fatal error naming the source and target types.

```
nums = list [3, 1, 3, 2, 1]
unique = change nums to set     // { 3, 1, 2 }
back = change unique to list    // [ 3, 1, 2 ]
change nums to set .            // nums itself becomes { 3, 1, 2 }

ages = map ["ann": 30, "bo": 25]
names = change ages to keys     // [ "ann", "bo" ]
years = change ages to values   // [ 30, 25 ]
```

## Show

```
show <expr> {, <expr>} .
```

Each piece is evaluated and its display form concatenated, in order, with
no separator inserted.

**Floats show to 15 significant digits**, as Excel and SQLite show them,
so the leftovers of binary arithmetic don't: `0.1 + 0.2` shows `0.3`, and
`list [1, 2, 3] process x give x * 0.2` shows `[ 0.2, 0.4, 0.6 ]`. They
compare the same way: when either side is a float, `==`, `!=`, `<`,
`<=`, `>` and `>=` use the 15-digit values, so `0.1 + 0.2 == 0.3` is
`true` and `0.30000000000000004 > 0.3` is `false`: what shows the same
is equal. Sets, map keys, `contains` and `count` agree. Integers compare
exactly. Very big and very small floats show in scientific notation, where
Python switches too: `1e+16` and up, and below `0.0001` (`1e-05`). The value itself keeps every digit, and writing to a file
(JSON, CSV) or a database keeps them all. For a looser match, round
first (`x at round[2] == y at round[2]`), or use `is close to` in a test.
For a fixed number of places on screen, use `x at fixed[2]` (`"0.30"`).

## Strings

`{<expr>}` inside a string puts the value of `<expr>` there, shown the
way `show` shows it:

```
name = "Ann"
show "Hi {name}, you have {qty * 2} items" .   // Hi Ann, you have 6 items
show "{x of p} {nums at len} {w substring 3}" .
show "a plain \{brace\}" .                       // a plain {brace}
```

- Any expression works inside the braces except one that contains a
  quoted string: the quote would end the outer string. Put the text in a
  variable first. Spaces inside are fine: `{ name }`.
- It works in every string, single- or double-quoted: `show`,
  assignments, `return`, `?` prompts, and the lines of a
  `[write]`/`[append]` block.
- **Plain braces.** A `{` is just a brace when the next non-space
  character is a quote or `}`, or nothing follows it. So JSON never needs
  escaping, and `"{}"` is `{}`:

  ```
  data = load['{"name": "Ann", "tags": ["a", "b"]}']
  show "{}" .                                // {}
  ```

  `\{` is always a plain brace; use it for other text with braces,
  like `"set \{1, 2}"`. A `{` with no closing `}` or an unfinished
  expression is a parse error. A `}` on its own is plain text.

**Backticks** make a raw string: what's between them is the text,
exactly. `{` and `\` are ordinary characters, and the string may run over
several lines:

```
n = 5
show `{n} stays` .                // {n} stays
show `C:\new\table` .             // C:\new\table
digits = `\d{3}-\d{4}`            // a pattern (import pattern)
```

**Single quotes** make text full of double quotes readable:
`'She said "hi"'`, `'{"a": 1}'`. They're the same strings as
double-quoted ones, with the same escapes and interpolation; only the
quote that ends them differs.

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

Call: `<name>[<expr>, ...]`. Written as `key: value` pairs, the arguments
are one map: `options["--out": "a.csv", "-v": false]` is
`options[map ["--out": "a.csv", "-v": false]]`. Each call gets a fresh scope seeded with its
parameters. At most 100,000 calls can be in progress at once; past that
is a "recursion too deep" error (usually a recursive function missing its
stopping case). Scoping is **lexical**: a name that isn't a parameter or local
is looked up in the scope the function was *defined* in — for a top-level
function that's the global scope, so every function body can read
top-level variables; it never sees the *caller's* locals. A parameter/local
of the same name shadows the outer one. Assignment always writes to the
call's own local scope, never an outer one, so `x = ...` inside a function
can't clobber a global `x` — it just shadows it for the rest of that call.
Reading the outer one to assign it, as in `count = count + 1` with no local
`count` yet, is an error that says what to do instead: give the function
the value and return the new one, or keep it in a list or map.
Mutating an outer `list`/`set`/`map` via a data-structure statement or
method call *does* affect it, since that mutates the same underlying value
rather than rebinding a name. The body can call any other top-level
function, including itself, recursively. Argument count must match
parameter count.

**A function can be called before its `def`.** A file's top-level
functions are all defined before its first line runs, so the main code
can come first and the helpers below it, and functions can call each
other in any order:

```
orders = load_orders["orders.csv"]
report[orders]

def load_orders[path]
    ...
def [end]
```

Only functions work this way. Everything else still runs top to bottom:
a variable is set when its line runs (a function called early can't read
a global set further down), and so is an assembled type. A def inside a
function is made when its line runs. A layout that reads well: imports,
then settings, then the main code, then the functions.

### Give back, don't change outer variables

A function can read a top-level variable, but `x = ...` inside it makes
a new local `x`; the outer one stays as it was. To change a value, give
the new one back and assign it where you call:

```
total = 0

def addtax[amount]
    return amount * 1.2
def [end]

total = addtax[100]        // the caller decides what changes
```

For several results, give back a list or an assembled value:

```
def minmax[nums]
    return list [min of nums, max of nums]
def [end]

r = minmax[scores]
low = r at get[0]
```

Lists, sets and maps are shared, not copied, when passed in, so a
statement that changes one (`add 4 to nums .`, `nums process x give x * 2 .`)
inside a function changes the caller's too. That's handy for "fill this
list", but giving back a new collection (`return nums process x give x * 2`)
keeps the function's effect visible at the call.

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
changing the captured one (and `n = n + 1` is an error). To keep mutable state in a closure, capture a
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

### Anonymous functions: `give`

```
x give <expr>             // one parameter
[a, b] give <expr>        // several, bracketed like a def
[] give <expr>            // none
[x] give                  // block form: a whole body, closed by give [end]
    <statement> ...
give [end]
```

`give` makes a function without a name. It's a value like any other
function, and it can read the variables around it (it's a closure).

```
double = x give x * 2
show double[4] .                     // 8
plus = [a, b] give a + b
show apply[x give x + 1, 3] .       // 4

big = [x] give
    if ] x > 3 [
        return true
    if [end]
    return false
give [end]
```

The expression form's body runs to the end of the expression, so inside a
call's `[...]` it stops at the next `,` or `]`.

### Methods as functions

A method can also be called like a function, the value it works on going
first, and so also as a sentence: `trim[s]` and `s trim` are `s at trim`;
`round[x, 2]` and `x round 2` are `x at round[2]`; `get[nums, 0]` is
`nums at get[0]`. A function of the program's own, or a library's, with
the same name comes first. Methods that need an import (`round` needs
`import math`) need it in every form.

### Sentence-style calls

```
<variable> <function> [<arg>, ...]
```

calls `<function>[<variable>, <arg>, ...]`: the variable on the left
becomes the first argument. It works with any function: your own, an
imported one, or a builtin like `data`'s `process`.

```
nums process x give x + 1 .         // process[nums, x give x + 1]
r = nums scale x give x + 1, 10     // scale[nums, x give x + 1, 10]
show nums total .                    // total[nums]
```

- The left side is a variable name, a literal (`"lo" isinstring line`,
  `list [1, 2] join ","`), or a call's result
  (`big = copy[nums] process x give x * 10` processes a copy).
- An argument takes in arithmetic, so `"abc" find "c" - 1` is
  `find["abc", "c" - 1]`, and `nums get i + 1` is `get[nums, i + 1]`. To
  do arithmetic on a sentence's result, store it first
  (`i = "abc" find "c"`, then `i - 1`) or use brackets
  (`find["abc", "c"] - 1`).
- **A comparison, `&&` or `||` ends the argument** and works on the
  call's result, as in most languages:

  ```
  if ] s has "urt" && ok [         // (s has "urt") && ok
  done = nums total > 100 || late  // (nums total > 100) || late
  x = a solve b == y               // (a solve b) == y
  ```
- A `give` function takes the rest of its line, as a lambda does in other
  languages: `nums keep n give n > 2 && n < 9` keeps the numbers between
  them. When more follows a call that has one, use brackets:
  `count_where[nums, n give n > 2] >= 5`.
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
- **`of` with `get` and `slice`.** The `get` picks the item first:
  `title of books at get[0]` is the title of the first book, and
  `title of books at get[0] at upper` upper-cases it. Other methods work
  on the field: `name of p at upper`. To index inside a field, name it
  first: `tags = tags of book`, then `tags at get[0]`.
- **`of` reads map keys too**: `title of row` is `row at get["title"]`
  (a missing key is an error of kind `key`), and `title of row = "Emma"`
  sets it. Handy for SQL rows: `title of rows at get[0]`.
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

## The kind of a value: `typeof`, `type`

`typeof[x]` gives the kind of any value as text, and needs no import:
`"integer"`, `"float"`, `"string"`, `"boolean"`, `"list"`, `"set"`,
`"map"`, `"date"`, `"none"`, `"function"`, `"error"`, an assembled
value's type name (`"order"`), or `"assembled type"` for a type itself.

`<value> type <kind>` is `true` or `false`, and reads as a condition:

```
assemble order [item, qty]
o = order ["tea", 2]

show typeof[o] .                 // order
if ] o type order [ ... if [end]
if ] n type integer && s type string [ ... if [end]
```

The kinds are the ones `check ... is` takes (see
[`testing.md`](testing.md)): `integer float number string boolean list set
map date none function empty` and assembled type names. A kind that
doesn't exist (`n type intger`) is an error. `type` is only special
between a value and a kind, so it still works as a variable name.

## Finding things out: `help`, `stdlib`, `version`

Three names that need no import. `help` knows every library, function,
method and keyword (its keyword text is the [Keywords](#keywords) tables
here). A variable or function of your own with
the same name comes first.

```
import linear
m = matrix [1, 2, 3; 4, 5, 6]

help[m]                 // This is a matrix, 2 x 3.
                        // A matrix's methods, as x at rows:
                        //   rows, columns, shape, get[r, c], ...
                        // Libraries for it: help["linear"]
help["linear"]          // the library, and each function in a line
help["reshape"]         // one function in full, with an example
help["list"]            // a kind of value's methods
help["if"]              // a keyword, from the table above
help["keywords"]        // every keyword in a line
help["typeof"]          // a function every program has
help[]                  // every library, and how to ask

libs = stdlib[]         // map: library name -> its function names
show libs at get["linear"] at contains["flatten"] .   // true

show version .          // 0.9.171
```

`help` shows its text and gives `none`. Text names what to look up, so
`help["resahpe"]` says `did you mean reshape?`. In the REPL the same
works without brackets: `help m`, `help linear`.

## Scrolls

A scroll is a value going through steps in order, each step's result
feeding the next. It reads top to bottom, instead of inside-out like
`half[double[add1[3]]]`, and needs no names for the values in between.
It always ends with a period, like a sentence.

```
x is scroll 3 into add1, double, half .     // 3 -> 4 -> 8 -> 4.0
x = scroll 3 into add1, double, half .      // the period ends it here too
s = scroll add1, double, half .             // no starting value: a saved scroll
y is scroll 10 into s .                     // 11.0
```

**Steps.** Outside `[ ]`, a comma starts the next step; inside, commas
separate a call's arguments. Count only the values you type yourself:

| Besides the value, the function needs | Write | Means |
|---|---|---|
| nothing | `double` | `double[value]` |
| one value | `splitby ","` | `splitby[value, ","]` |
| two or more | `add_time[here, 30, "days"]` | that call, with the value at `here` |

- **`here`** is the value reaching the step. Without it, the value goes
  first (`join[" - "]` is `join[value, " - "]`); with it, it goes where
  `here` is: first, last, in the middle, or inside a list
  (`sql_query[db, "... ?", list [here]]`). A step can also be any
  expression with `here`: `here * 2`, `here at get[0]`.
- **Methods** start with `at`: `at trim`, `at upper`, `at round[2]`.
- **A function** works too: `n give n * 5`.
- **A saved scroll** used as a step runs its steps right there, in order:
  with `first = scroll a, b, c .` and `second = scroll d, e .`,
  `scroll 1 into first, second, f .` is `scroll 1 into a, b, c, d, e, f .`
- **Lists:** each step gets the whole list (as `sum`, `join`, `keep` and
  `table_write` want). For each item, use `process`:
  `process s give s at trim`. Saving such a step makes it reusable:
  `trimall = scroll process s give s at trim .`

```
written is scroll "https://api.example.com/users" into
    http_get,
    load,
    keep u give age of u >= 18,
    table_write["adults.csv", here] .
```

- A scroll over several lines ends at its period; inside `[ ]` the period
  goes before the `]`: `typeof[scroll 1 into add1 .]`.
- A line can be just a scroll, run for what it does:
  `scroll rows into table_write["out.csv", here] .`
- A saved scroll is a value of type `scroll`: call it on one value
  (`s[3]`), give it to `process` or `keep`, keep scrolls in lists and maps.
- `here` means the value of the scroll it's written in. Outside any
  scroll it's an ordinary name, so `here = scriptfolder[]` still works.

**none and errors.** `none` is a value like any other: a step that gives
back `none` passes it to the next step, since some functions answer
`none` on purpose (`find_first` when nothing matches). An error ends the
scroll at its step, because the steps after it depend on it. It keeps
its own kind (`handle [http]` still catches a failed `http_get`), and
its message says which step: `scroll step 1 of 4 (http_get): ...: 404 Not Found`.

### Looking inside: `diagnose`

`diagnose` traces a scroll or a function: each step's actual value,
every `none`, and the error where one happened. It shows that on the
screen, then gives back what it ended with, and it doesn't stop the
program, so it can be added to a line to find a problem and taken off
again. It needs no import.

| Given | It shows | It gives back |
|---|---|---|
| a saved scroll and a value: `diagnose[s, 3]` or `s diagnose 3` | each step and what it returned | the result, or the error |
| a scroll written in it: `diagnose[scroll 3 into add1, double .]` | the same | the result, or the error |
| a function and its values: `diagnose[double, 21]`, `diagnose[load, text]` | what it was given, what it returned, how long it took | the result, or the error |
| an error from `handle`: `diagnose[e]` | its kind and message, and for a scroll each step | the error |

```
s = scroll add1, double .
x is diagnose[s, 3] .
```

```
diagnose s (line 2)
  start      3
  1 add1   → returned 4
  2 double → returned 8
  result     8
```

**Each line** is a step's number (`2`, or `2.1` for a step inside a
saved scroll used as step 2), the step as written, and what it did:
- `→ returned` and the value as it was when that step returned, even if a
  later step changes the same list or map: its size and what's in it,
  cut short past 60 characters (`map of 2: { "total": 100, "fee": 50 }`,
  `"text"... (82 characters)`);
- `→ returned nothing (none)   is none expected!?`: a `none` went on to the next step.
  Was it meant? If not, that step is where to look (a forgotten
  `return`, nothing found);
- `✗ failed:` and the error: the scroll ended here. A step inside a saved
  scroll says where it sits: `scroll step 2 of 2 (here / 0), in step 1
  of 1 (s)`. When the step was
  given `none`, a line under it says where that came from:
  `(it was given none, from step 2)`, or `from the start`;
- `not reached`: the steps after a failure.

```
x is diagnose[scroll 3 into add1, shownumber, double .] .
show kind of x .
```

```
number 4
diagnose scroll (line 1)
  start          3
  1 add1       → returned 4
  2 shownumber → returned nothing (none)   is none expected!?
  3 double     ✗ failed: operator "*" needs two numbers, got none and integer
                 (it was given none, from step 2)
type
```

Here `diagnose` captures both the `none` (step 2, "is none expected!?") and the error it
led to (step 3, `✗`, pointing back at step 2), and `x` holds the error
(a value like the one `handle` gives: `kind of x`, `message of x`), so
the program goes on. Without `diagnose`, the same error stops the
program, or goes to a `safe` block, as usual.

- A scroll that ends with `none` gives back `none`, marked "is none expected!?".
- In a `handle` block, `diagnose[e]` shows the error and, for a scroll,
  what each step did, and gives back `e`.
- Only Turtle's errors are traced: `exit[]` and Ctrl-C still stop the program.
- A program's own function called `diagnose` wins over this one.
- `turtle trace` also shows each scroll step's result as the program runs.

### Looking inside any code: the `diagnose` block

`diagnose` on a line of its own starts a block, ended by `diagnose [end]`.
The code in it runs as usual; `diagnose` shows each line as it runs, the
value each assignment gave, and each loop pass with its names. An error
in it is shown and the program goes on after `diagnose [end]`.

```
nums = list [4, 7]
diagnose
    total = 0
    [loop][x in nums]
        total = total + x
    [loop][end]
    avg = total / 0
diagnose [end]
show "after: ", total .
```

```
diagnose (lines 3-7)
  line 3       total = 0                    total = 0
  line 4       [loop][x in nums]
                                            pass 1: x = 4
  line 5           total = total + x        total = 4
                                            pass 2: x = 7
  line 5           total = total + x        total = 11
  line 7       avg = total / 0
  ✗ failed: math error: line 7: division by zero
after: 11
```

- It shows the first 60 lines, then how many more there were (a long loop).
- Lines of functions called from the block show too, as in `turtle trace`.
- It ends with `finished`, or `✗ failed:` and the error.
- `turtle fmt` indents the block, and the REPL waits for `diagnose [end]`.


## Theories

A theory brings a new word into Turtle: how it's written, what it means,
and what it promises. Once a file has a theory, its phrases read like
the rest of the language.

```
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
    theorem result <= length of a
    proof
        tally 1 in list [1, 1, 0, 3] . is 2
        tally 9 in list [] . is 0
theory [end]

tally 1 in nums .                  // on its own line
c is tally 1 in nums .             // its result, named
show "ones: ", tally 1 in nums .   // inside other sentences
```

| Section | | What it does |
|---|---|---|
| `abstract` | required | what the word means, in plain words; `help`, `turtle doc` and the editor show it. It should name the word. |
| `notation` | required, one or more | how the word is written. The names the definition uses (`b`, `a`) are its **values**; every other word (`in`) is **fixed**. Two values need a word or a comma between them. |
| `definition` | required | the code, with each value by its name; `return` gives the result |
| `theorem` | optional, any number | something every result satisfies, written with `result` and the values |
| `proof` | optional | worked cases: a use, `is`, what it gives |

**Rules.**
- The word follows `theory` and starts every notation. It must be new:
  not a keyword, a function, a method, a variable or another theory.
- A theory comes before its word is used, reading top to bottom. It's
  written at the top level of a file.
- `import shapes` brings in that file's theories with its functions;
  `import shapes [tally]` names the ones to bring. Two imports with the
  same word is an error.
- `theory ~name` is a private theory, for its own file, like a `~`
  function: a file that imports it can't use its word, except a test
  file, which can test it.
- A library of the standard library can have theories too (written in
  Turtle, in `evaluator/lib/`): `import data` brings them in.
- A value reaches as a sentence call's does: through arithmetic, not past
  a comparison, `&&` or `||` (`tally 1 in nums == 2` compares the
  result). A fixed word ends the value before it.
- The section words are special only in a theory, and `theory` only at
  the start of a line before a name: elsewhere they're ordinary names.
- The word alone is the theory itself, a value: `typeof[tally]` is
  `"theory"`, `tally type theory` is true, `help[tally]` describes it.

**Checking a theory.** `turtle test` proves the theories of each test
file and of the files it imports: every proof case must give what it
says, and every theorem must hold for the proof cases and for 100 random
inputs made like their values. A definition that refuses an input with
`fail` doesn't count that input against a theorem. A theorem that breaks
shows the smallest input that breaks it. A theory with no proof, or an
abstract that doesn't name its word, gets a warning. In an ordinary run
none of this is checked. See [`testing.md`](testing.md#theories).

**Looking inside.** `diagnose[tally 1 in nums]` shows one use: its values,
its result, each theorem for it, and how long it took. `diagnose[tally]`
runs the theory's proof and its theorems, as `turtle test` does, and
shows where they fail. Neither stops the program.

## None

`none` is Turtle's "no value". It's what a function returns when it has no
`return` or uses a bare `return`, and what the time module's `sleep[...]`
returns. It shows as `none`, is falsy, and equals only itself.
`none + none` is `none`; `none` added to anything else (`"x=" + none`,
`5 + none`) is a `type` error, as are other arithmetic and ordering
comparisons on `none`. To put it in text, use interpolation or `show`:
`"x={x}"`, `show "x=", x .`.

## Mistakes in the code

When a file has a mistake Turtle can't read past, it shows the first one,
with the line and a `^` under the spot, and says how to fix the usual
ones:

```
turtle: report.turtle, line 2: this line needs a '.' at the end
  2 | show x
    |       ^
```

Only the first is shown: the ones after it are usually the same mistake
seen again.

**Notes.** Some lines are valid but probably don't do what they seem to.
Turtle runs them as written and shows a note first (editors underline
them):

```
turtle: note: report.turtle, line 4: at round works on b only, not on the whole / expression; to round the whole value, store it first: v = ... / b, then v at round
```

A method works on the value right before it, so `a / b at round[1]`
rounds `b`. And `params of req at get["id"]` gets `"id"` from `req`
(the `get` goes first, as in `title of books at get[0]`); when that fails,
the error says so and how to write it: `v = params of req`, then
`v at get["id"]`. Habits from other languages get a pointer to Turtle's way:
`if x > 0 [` (`if ] x > 0 [`), `if ] x = 1 [` (`==`), `x.upper()`
(`x at upper`), `f(1)` (`f[1]`), `for x in` (`[loop][x in nums]`), a
block or a quote never closed. Editors show the same message, under the
same spot.

## Errors: `safe` / `handle` / `fail`

```
safe
    <statement> ...
handle [<kind>, ...] <name> .
    <statement> ...
safe [end]

fail <expr>
```

`safe` runs the code under it. If a statement there hits an error whose
kind is listed, the rest of that code is skipped, the error is stored in
`<name>` (any name), and the code under `handle` runs. Without an error,
the code under `handle` is skipped and `<name>` is `none`. Either way,
the program carries on after `safe [end]`, which takes no period, like
`def [end]` and `if [end]`.

```
safe
    [read] settings.txt to lines [end]
    count = change lines at get[0] to integer
handle [file, number] problem .
    show "using defaults: ", problem .   // line 2: [read] settings.txt: no such file or folder
    count = 10
safe [end]
```

**Kinds** (`handle [] e .` handles every kind):

| kind     | for example                                              |
|----------|----------------------------------------------------------|
| `file`   | a missing file, a file that can't be written, end of input |
| `number` | `change "abc" to integer`                                |
| `math`   | division or modulo by zero, integer overflow, `sqrt` of a negative |
| `index`  | list index out of range, `pop` or `min of` on an empty collection |
| `key`    | map key not found                                        |
| `name`   | undefined variable, function, method, module or field    |
| `type`   | the wrong kind of value (`"a" * 2`) or number of arguments |
| `json`   | text that isn't valid JSON (`json` library)              |
| `date`   | text that isn't a date, or a date that doesn't exist (`time` library) |
| `http`   | a web request that failed, or got a 4xx/5xx status (`http` library) |
| `sql`    | a bad query or a database problem (`sql` library)          |
| `csv`    | a `.csv` or `.tsv` file that isn't well formed (`table_read`, `sql_load`, ...) |
| `test`   | a failed `check`, `verify` or `validate` (`import test`; see [`testing.md`](testing.md)) |
| `pattern` | a pattern that isn't valid (`import pattern`) |
| `crypt`  | text that isn't base64 or hex, a wrong passphrase, an unknown algorithm (`import crypt`) |
| `schedule` | a bad `schedulelimit` or setting, a command that can't start, `queryall` on SQLite (`import schedule`) |
| `server` | a port in use, a bad route, status or port (`import server`) |
| `config` | a settings file that isn't well written, or a value it can't hold (`import config`) |
| `scroll` | a scroll step that isn't a function or scroll, or a scroll inside itself |
| `linear` | matrices of the wrong size for each other, a singular matrix, a non-symmetric matrix for `eigen` (`import linear`) |
| `custom` | your own, from `fail`                                    |

An error of a kind that isn't listed isn't handled: it goes on to an
enclosing `safe`, or stops the program as usual. So does an error in
the code under `handle` itself. A kind that doesn't
exist (`handle [maths] e .`) is a parse error.

**The error value** shows as its full message (`show e .` prints
`line 2: division by zero`), and has three parts: `kind of e` (`"math"`),
`line of e` (`2`), `file of e` (`"report.turtle"`, or `"lib/utils.turtle"` for an
error inside an imported module) and `message of e` (`"division by
zero"`). For an error inside a scroll, `diagnose[e]` shows what each step
did (see [Scrolls](#scrolls)). It's truthy, and still set after `safe [end]`.

**Where.** An error inside an imported module names its file:
`lib/utils.turtle line 2: division by zero`. One in the main script just says
`line 2: ...`. This holds whether the program stops on it or a `handle`
shows it, and for parse errors in a module too.

**`fail <expr>`** raises an error of kind `custom` whose message is the
value of `<expr>`. Like `return`, it takes no period. Unhandled, it stops
the program with that message.

```
def withdraw[amount]
    if ] amount > balance [
        fail "not enough money"
    if [end]
    return balance - amount
def [end]
```

- Errors raised inside function calls, however deep, reach the `safe`
  around the call. Variables set before the error keep their values.
- `return`, `break` and `continue` under `safe` or `handle` work as usual.
- The error variable follows assignment rules: inside a function it's
  local.
- Parse errors are never handled; neither is `system`'s `exit[code]`.

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
(`add [] give x to fs .`) keeps the value it saw. Every other assignment
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

See [the library docs](library/index.md) for full method tables. Declaration:

```
<ident> = list [<expr>, ...]
<ident> = set [<expr>, ...]
<ident> = map [<expr>:<expr>, ...]
```

Statement-form operations (all require a trailing `.`):

```
add <expr> to <ident> .
add <expr> to <ident> at <key> .  // map only: ages["Cy"] = 12 is add 12 to ages at "Cy" .
remove <expr> from <ident> .
delete <expr> from <ident> .      // map only
sort <ident> .
reverse <ident> .
insert <expr> to <ident> at <expr> .
put <expr> to <ident> at <expr> .  // list only: replaces the item there
length of <expr> .                 // prints
min of <expr> .                     // prints
max of <expr> .                      // prints
```

Method-call form:

```
<ident> is <expr> [at <method> [<expr> {, <expr>}]] .
```

Plain `<ident> is <expr> .` (no `at`) is assignment/aliasing. An `is`
line is a sentence, so it ends with a period like `show` and `add`;
`name = value` is the same assignment without one.

`insert` and `put` both take a position from 0: `insert 99 to nums at 1 .`
pushes the items from 1 along (the list gets longer); `put 99 to nums at
1 .` replaces the item at 1 (the length stays). `put` is a statement word
only at the start of a line followed by a value; otherwise it's an
ordinary name. As methods, both take the value first, then the position,
as the sentences do: `nums at insert[99, 1]`, `nums at put[99, 1]`; both
give back the list.

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

`<path>` is any of:

- an expression that builds it: `folder + "/" + name + ".txt"`,
  `names at get[0]`, `path of doc`, `file_for[id]` (a word or string
  followed by `+`, `[`, `at` or `of`)
- a quoted string, with values inside it: `"{folder}/{name}.txt"`
- a variable: `[read] name to lines` uses the variable `name` if there is
  one, else the file called `name`
- a bareword like `file.txt` or `data/in-1.csv`, the file of that name

```
[read] folder + "/" + day + ".csv" to rows [end]
[write] receipts at get[0]
"paid"
[end]
```

Relative paths resolve from the folder `turtle` was run in (see
[the `system` docs](library/system.md)).
`[write]`/`[append]` body lines are one item each: a quoted string is
written verbatim, a bare identifier is replaced with that variable's
current value; items are newline-joined.

## Modules

```
import <name>                  // everything the module exports
import <name> [<f>, <g>, ...]  // only the listed names
```

`<name>` is a builtin module (`math`, `time`, `data`, `strings`, `system`,
`json`, `http`, `sql`, `sort`, `search`, `random`, `pattern`, `crypt`,
`schedule`, `config`, `server`, `log`, `test`, `linear`; see [the library docs](library/index.md)) or a file `<name>.turtle`,
resolved relative to the current script's directory. A module in a
subfolder is written with `/`: `import lib/utils` reads `lib/utils.turtle`,
and its qualified name is the last part, `utils half[4]`. Because builtin names
win, don't name your own module file after one (`math.turtle`, `json.turtle`, ...);
`import lib/json` is an error for the same reason.

**What a module exports.** A `.turtle` module exports its top-level functions,
and only those. It runs once, in its own global scope, the first time any
file imports it; later imports reuse it. Its top-level variables stay
private to it, although its own functions can read them. A function you
import keeps calling the module's other functions, even ones you didn't
import, and those never leak into your program. A module's own imports
aren't passed on to the files that import it.

**Private functions: `~`.** A function whose name starts with `~` is
private to its file: its own code uses it like any other,
and files that import the module can't, by any route (`~limit[...]`,
`shop ~limit[...]` and `import shop [~limit]` all say it's private).
`~limit` and `limit` are two different names, so a public function can
check its values and hand the work to a private one:

```
// limit keeps n between low and high.
def limit[n, low, high]
    if ] low > high [
        fail "limit: low is above high"
    if [end]
    return ~limit[n, low, high]
def [end]

// ~limit does the work, for values already checked.
def ~limit[n, low, high]
    return min of list [max of list [n, low], high]
def [end]
```

Only a function's name can start with `~`. Variables are private to their
file already, and an assembled type stays public, since the values a
module gives back are its users' to work with. A test file run by
`turtle test` can use the private functions of the modules it imports, to
test them (see [`testing.md`](testing.md#public-and-private-functions)). `turtle doc` on the file documents its public
functions and lists the private ones at the end.

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

**Examples**, with what each shows:

```
// shop.turtle, next to the program
taxrate = 0.08                  // private to shop
def withtax[amount]
    return amount + amount * taxrate
def [end]
def now[]
    return "shop's now"
def [end]
```

```
import math                     // a built-in library
import time [now]               // only now
import data
import shop                     // shop.turtle
import lib/money                // lib/money.turtle, named money

show withtax[100] .             // 108.0: the plain name
show shop withtax[50] .         // 54.0: named with its module
show money half[9] .            // 4.5
f = shop withtax                // the function itself, as a value
show f[10] .                    // 10.8
show nums sum .                 // an imported function as a sentence: sum[nums]
show time now[], shop now[] .   // both export now: name the module
x = now[]                       // error: "now" is provided by more than one import (time, shop)
sleep[0]                        // error: "sleep" isn't imported — add it to "import time [...]"
show taxrate .                  // error: undefined variable "taxrate"
```

Without `import data`, `sum[nums]` is an error that names the import:
`"sum" needs "import data" first`. Some imports also turn on words of the
language for that file (`random`, `check`, `log`, `matrix`): see
[Keywords](#words-an-import-turns-on).

**Builtin modules.**

- `import math` unlocks number methods: `sqrt`, `abs`, `round`, `floor`,
  `ceil`, `pow`, `random` (see [the `math` docs](library/math.md)). Methods
  are called on a value (`r is 16 at sqrt .`), so they never clash and
  are never qualified; an import list still limits which ones you can use
  (`import math [sqrt]`).
- `import time` provides two builtin functions: `now[]` (milliseconds
  since the Unix epoch, as an Integer) and `sleep[<amount> [, <unit>]]`
  (pauses; `<unit>` is `"seconds"`, the default, or `"ms"`; returns
  `none`). Builtin functions can be called but not used as values.
- `import system` provides `args[]` (the command-line arguments after the
  script path), `exists`/`isfile`/`isfolder`, `contents[path]`,
  `exit[code]`, `env[name]` and `scriptfolder[]` (see
  [the `system` docs](library/system.md)).
- `import strings` provides `find`, `substring`, `isinstring` and `join`
  (see [the `strings` docs](library/strings.md)).
- `import sql` provides `sql_open`, `sql_create`, `sql_query`, `sql_run`,
  `sql_tables`, `sql_close`, and moves files in and out of a database
  with `sql_save`, `sql_load`, `sql_update`, `sql_delete`, `sql_upsert`
  (see [the `sql` docs](library/sql.md)).
- `import data` provides `process`, `keep`, and `copy`, which apply a
  function across a list, set, or map, `table`, which lays rows out as a
  text table, and `table_write` / `table_read`, which save and read
  `.csv`, `.tsv`, `.txt` and `.json` table files (see
  [the `data` docs](library/data.md)). They're
  ordinary functions, usually called sentence-style:
  `nums process x give x + 1 .`, `show table[rows] .` It also has the
  statistics: `mean`, `median`, `mode`, `variance`, `stdev`, `pvariance`,
  `pstdev`, `percentile`, `covariance`, `correlation`, `zscores`,
  `describe` (see [the `data` docs](library/data.md#statistics)).
- `import linear` adds the `matrix` value, written
  `matrix [1, 2; 3, 4]` (rows end at `;` or at the end of a line), its
  operators (`*` of two matrices is the matrix product), and functions
  such as `solve`, `inverse`, `determinant`, `eigen` and `svd` (see
  [the `linear` docs](library/linear.md)). `matrix` is a word of the
  language only in a file that imports `linear`; elsewhere it's an
  ordinary name, and in such a file it can't name a variable or function.
- `import random` makes random values of any shape, written as a
  sentence: `random list of 5 integers from 0 to 9`, `random Order [string,
  integer]`; plus `pick`, `shuffle`, `sample`, `chance` and the `seed`
  variable (see [the `random` docs](library/random.md)). `random` is a
  sentence word only in a file that imports it.
- `import sort` provides `min_sort` / `max_sort` (order by a function,
  field, position or several, and take the `"first"` or a count),
  `is_sorted`, `reverse_list`, and the classic sorting algorithms; and
  `import search` provides `find_first`, `find_all`, `find_key`, ... and
  linear, binary and other searches (see
  [the `sort` docs](library/sort.md)). `import sort` names the
  library; `sort nums .` on its own is still the in-place sort statement.

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
verbatim to end of line, runs through a shell (`sh`, or `cmd` on Windows)
with inherited stdin/stdout/stderr, in the folder turtle was run in (where
file paths resolve too).

## Logging: `log`

In a file with `import log` (settings and examples: [the `log` docs](library/log.md)):

```
log [debug | info | warn | error] <expr> {, <expr>} .
```

The values are joined as `show` joins them; a map among them adds
`key=value` fields. Without a level word it's `info`. `log` begins a
statement only when a value or a level word follows it, so `log = 5`
still assigns.

## Tests: `check`, `verify`, `validate`, `turtle test`

In a file with `import test`, three more statements (full guide:
[`testing.md`](testing.md)):

```
check <expr> .                                  // true or false; == explains a difference
check <expr> is [not] <kind> .                  // integer float number string boolean list set map date none function empty, or an assembled type
check <expr> is [not] close to <expr> [within <expr>] .
check <expr> fails [<kind>, ...] .              // the brackets are optional: any error
verify <expr> <how many> <rule> .
validate <call> [to <name>] [with <name> as <kind>, ...] that <rule> .
validate <call> [to <name>] [with <name> as <kind>, ...] matches <call> .
```

- `<how many>` is `each`, `any`, `not`, `at least N`, `at most N` or
  `exactly N`, optionally followed by `pair` (neighbors two at a time).
  `<rule>` is a function giving true or false (`x give x > 0`, or a
  function's name).
- `validate`'s `<kind>` uses the random library's words
  (`list of integer`, `Order [string, integer]`); its rule is `that`
  followed by a true/false expression or a verify-style rule
  (`that result each x give ...`), or `matches` another call. Without
  `with`, it learns the kinds from the same test's checks on that
  function. The sentence can go over several lines; it ends at its
  period.
- A failure is an error of kind `test`.
- `check`, `verify` and `validate` are statement words only after
  `import test`; elsewhere they're ordinary names. In such a file a
  function can't be named one of them.
- `import test` also makes the variables `suite`, `benchmark`, `runs`,
  `benchtime`, `cases` and `seed`.

`turtle test [file.turtle | folder ...]` runs every top-level `test_` function
(no arguments) in every `test_*.turtle` file, after the file's own top-level
code, and exits with 1 if any failed.

## Watching it run: `turtle trace`

`turtle trace script.turtle [args]` runs the program as usual and, on
standard error, shows each line as it runs. A line that sets a variable
shows the value it got:

```
line 1   nums = list [5, 7]               nums = [ 5, 7 ]
line 2   total = 0                        total = 0
line 3   [loop][x in nums]
line 4       total = total + x            total = 5
line 4       total = total + x            total = 12
line 6   d = double[total]
line 9       return n * 2
line 6   ...                              d = 24
```

When a line calls a function, the function's lines come next, and the
value follows on a `...` line. Lines of an imported file are named with
the file (`utils.turtle:4`). `def` lines are left out: functions are defined
before the file runs. The program's own output stays on standard output,
so `turtle trace report.turtle 2> trace.txt` keeps the two apart.

## Stepping through it: `turtle debug`

`turtle debug script.turtle [args]` runs the program a line at a time. It
stops before the first line, shows the line it's on, and waits:

```
→ line 9      r = n * 2
(debug) p n
12
(debug) v
here:
  n = 12
globals:
  nums = [ 5, 7 ]
  total = 12
```

| Command | Does |
|---|---|
| Enter or `s` | step: run this line, stop at the next (going into functions) |
| `n` | next: run this line and any functions it calls |
| `o` | out: run to the end of this function |
| `c` | continue to the next breakpoint, or the end |
| `b 12` | stop at line 12 (`b utils.turtle:4` in an imported file); `b` alone lists them |
| `d 12` | remove that breakpoint |
| `p <value>` | show a value: `p total`, `p nums at len` |
| `v` | the variables here, then the globals |
| `w` | where: the functions being run, outermost first |
| `l` | the lines around this one |
| `q` | quit the program |
| `h` | the commands |

Anything else is run as Turtle, right there: `total` shows its value,
`n = 100` changes `n` before the line runs, `show nums .` shows. An error
in what you type is shown and the program carries on. The debugger
writes to standard error and reads standard input, like the program's
`?` prompts. If the input ends, the program runs to the end.

## Layout: `turtle fmt`

`turtle fmt` lays `.turtle` files out the standard way: each block's lines
four spaces in from the line that opens it (`def`, `if ] ... [`,
`[loop][...]`, `safe`, `give`, `[write]`), `else` and `handle` lined up
with their block, `that` one step in under its `validate`, no spaces at
line ends, at most two blank lines in a row, and one newline at the end.

```sh
turtle fmt                    # every .turtle file here and in the folders below
turtle fmt report.turtle lib     # these files and folders
turtle fmt --check            # change nothing; list what needs it (exit 1 if any)
```

It only changes the space at the start and end of lines, never what's on
them (in Turtle a space can matter: `nums get -1` isn't `a - 1`). Lines
inside a `` `raw string` `` or a `//* *//` comment are left as they are. A
file that doesn't parse is reported and left alone. In VS Code, Format
Document (Shift-Alt-F) does the same.

## Sharing a program: `turtle build`

```
turtle build report.turtle            // makes report (report.exe on Windows)
turtle build report.turtle -o tool    // makes tool
```

`turtle build` makes one program file that runs without Turtle installed:
a copy of turtle with the script and every `.turtle` file it imports (however
deep) packed into it. Run it like any program; every argument goes to the
script (`args[]`):

```
./report 2026 "north region"
```

- **For the system it's built on.** turtle on a Mac makes a Mac program,
  on Windows a `.exe`, on Linux a Linux program. To share a tool with
  Windows users, run `turtle build` on Windows.
- **Data files aren't packed.** CSV files, settings and the like are read
  as with turtle: file paths from the folder the program is run in.
  `scriptfolder[]` is the program's own folder.
- **Libraries are built in.** `import json`, `import sql` and the rest
  are part of every program; only your own `.turtle` files are packed.
- **Size.** A program is about the size of turtle (10 to 15 MB): turtle
  is inside it.
- **Mistakes are found first.** A script that doesn't parse, or imports a
  file that isn't there, isn't built.
- **macOS.** A program made on your Mac runs on it. Copied to another Mac
  by download, macOS may say it's damaged (it isn't signed by Apple);
  `xattr -d com.apple.quarantine report` lets it run.
- **Windows.** A program can't be rebuilt while it's running: close it
  first. Downloaded on another PC, Windows may say it "protected your PC"
  (it isn't signed); More info, then Run anyway, lets it run.

## Documentation: `turtle doc`

`turtle doc` shows what every library function takes and gives back:

```sh
turtle doc                 # every library and its functions
turtle doc sql             # one library, every function in full
turtle doc sql_update   # one function
turtle doc lib/shop.turtle      # the functions of your own file
```

Document your own functions with comments **directly above** the `def`
(or `assemble`), or **first in the function's body** (as Python's
docstrings are), either kind: `//` lines or a `//* ... *//` block. The `//`
lines at the very top of a file describe the file. Say what each
parameter is and what the function gives back:

```
// shop.turtle: helpers for the bookshop database.

// add_tax adds rate percent to an amount, rounded to the cent.
// cents is a whole number of cents; rate is a percent, like 8.
// Gives back the new amount in cents.
def add_tax[cents, rate]
    ...
def [end]
```

`turtle doc shop.turtle` then lists `add_tax[cents, rate]` with those lines.
A function without them is listed as having no description. The same
comment can go inside, as the first thing in the body:

```
def add_tax[cents, rate]
    //* adds rate percent to an amount, rounded to the cent.
        Gives back the new amount in cents. *//
    ...
def [end]
```

A comment after the first line of code is an ordinary comment, not the
description. Comments in both places are shown together, the one above
first. Editors show the description on hover (see
[`editors.md`](editors.md)).
