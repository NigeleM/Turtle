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
| `find` | value | Boolean, whether present |
| `insert` | value, index | the list, with value inserted |
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

`import data` (or `import data [process, keep, copy]`) provides three
functions. `process` and `keep` change the collection **in place** and
also return it. Use `copy` first to keep the original.

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

These are ordinary functions, so you can write your own in a `.t` library
and call them the same sentence style. See
[`reference.md`](reference.md#sentence-style-calls).

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
`turtle ~/tools/count.t notes.txt` reads `./notes.txt`. (`import` is
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
// turtle count.t notes.txt
import system
import strings

a = args[]
if ] length of a == 0 [
    show "usage: count.t <file>" .
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
// turtle tool.t notes.txt sub missing.txt
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

Reads `<name>.t` (relative to the script's own folder, wherever `turtle`
was run from) and runs it
once, in its own scope. Its top-level functions become available to your
program: all of them, or only the ones listed in `[...]`. Its top-level
variables stay private to it. When two imports export the same function
name, call it qualified by module, as in `mylib now[]` or `time now[]`.
An unqualified call to a name that clashes is a fatal error. Full rules
are in [`reference.md`](reference.md#modules).
