# Standard library

Everything built into the language beyond core syntax: data structure
methods, file I/O, `sys`, and `import`. See [`reference.md`](reference.md)
for the statement/expression grammar these use.

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
- set: `{ a, b }`
- map: `{ Alice: 30, Bob: 25 }` (insertion order, always — this is
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
| `length of <expr> .` | list, set, map, string | Prints the count (map: number of keys; string: character count, in Unicode code points, not bytes). |
| `min of <expr> .` | list, set, map | Prints the smallest element (list/set) or smallest **key** (map — matches legacy behavior; map values aren't compared). |
| `max of <expr> .` | list, set, map | Same as `min of`, but largest. |

`length of`/`min of`/`max of` are also ordinary expressions usable anywhere
(`x = 1 + length of nums`) — they only print when they're the entire
statement.

### Method-call form

```
<result> is <receiver> at <method> [<arg> {, <arg>}] .
```

#### list

| Method | Args | Returns |
|---|---|---|
| `add` | value | the list (mutated) |
| `len` / `length` | — | element count (Integer) |
| `toString` | — | display string |
| `clear` | — | the list, now empty |
| `count` | value | number of matching elements (Integer) |
| `index` | value | first matching index, or `-1` |
| `sort` | — | the list, sorted |
| `remove` | value | the list, first match removed |
| `reverse` | — | the list, reversed |
| `pop` | — | the **removed last element** (fatal if empty) |
| `find` | value | Boolean, whether present |
| `insert` | value, index | the list, with value inserted |
| `get` | index | the element at that index (fatal if out of range) |
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
| `getValues` | — | a list of values, insertion order |
| `getKeys` | — | a list of keys (as strings), insertion order |
| `add` | key, value | the map, with the entry set |
| `delete` | key | the map, with the entry removed (fatal if not found) |
| `invert` | — | new map: values become keys, keys become (string) values |
| `toString` | — | display string |

#### string

All indices are Unicode code points (runes), not bytes — consistent with
`length of` above and with `change ...to ascii/char` (see
[`reference.md`](reference.md#type-conversion)).

| Method | Args | Returns |
|---|---|---|
| `upper` / `lower` | — | case-converted string |
| `trim` | — | leading/trailing whitespace stripped |
| `get` | index | the one-character string at that index (fatal if out of range) |
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
| `pow` | exponent | receiver raised to exponent, as a `float` |
| `random` | — | a random integer in `[0, receiver)`; the receiver is the exclusive upper bound and must be a positive integer |

```
import math

r is 16 at sqrt .        // 4
p is 2 at pow 10 .       // 1024
n is 10 at random .      // some integer in [0, 10)
```

`%` (modulo) is a core operator, not a method, so it needs no import — see
[`reference.md`](reference.md#expressions).

### Builtin functions: `time`

`import time` unlocks two ordinary function calls (not methods — there's
no natural receiver for "the current time"):

| Function | Args | Returns |
|---|---|---|
| `now[]` | — | milliseconds since the Unix epoch, as an Integer |
| `sleep[amount [, unit]]` | amount, optional unit | pauses execution for that long; returns `none` |

`sleep`'s `unit` is the string `"seconds"` (the default, when omitted) or
`"ms"`:

```
import time

sleep[1]            // 1 second
sleep[0.25]          // a quarter second
sleep[250, "ms"]     // 250 milliseconds, explicitly

t1 = now[]
sleep[0.5]
t2 = now[]
show t2 - t1 .    // at least 500 (now[] is always in milliseconds)
```

If you've already defined your own top-level function named `now` or
`sleep`, it always wins — the builtin only kicks in when no user function
of that name exists.

Plain `<result> is <receiver> .` (no `at`) is just assignment/aliasing —
`<result>` becomes another reference to the same underlying value.

## Files

Paths are quoted strings or barewords (`file.txt`, `data/in.csv`),
resolved relative to the directory the running script lives in (not the
process's current working directory). Any I/O failure (file not found,
permission denied, etc.) is a fatal error.

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

Reads `<name>.t` (relative to the current script's directory) and runs it
once, in its own scope. Its top-level functions become available to your
program: all of them, or only the ones listed in `[...]`. Its top-level
variables stay private to it. When two imports export the same function
name, call it qualified by module, as in `mylib now[]` or `time now[]`.
An unqualified call to a name that clashes is a fatal error. Full rules
are in [`reference.md`](reference.md#modules).
