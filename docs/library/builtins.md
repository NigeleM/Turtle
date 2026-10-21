# Built-in types and methods

[Library index](index.md)

Lists, sets, maps, text and numbers: making them, changing them, and the methods every value has. No import needed.


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
- map: `{ "Alice": 30, "Bob": 25 }` (always in insertion order)

## Statement-form operations

All require a trailing `.`. Target must be a variable name (not an
arbitrary expression).

| Statement | Applies to | Effect |
|---|---|---|
| `add <expr> to <target> .` | list, set | Appends. Sets silently skip the add if the value's already present. |
| `add <expr> to <target> at <key> .` | map | A new key starts at the value: `add 12 to ages at "Cy"` gives `"Cy": 12`. An existing value changes by its type: a number adds up (5 becomes 17), text joins on, and a list or set takes the value (`add "b" to groups at "x"` appends to that list). |
| `add <expr> to <target> at <position> .` | list of lists | Changes the item at that position by the same rules (`add 9 to rows at 1` appends 9 to row 1). Fatal if the position is out of range. |
| `remove <expr> from <target> .` | list, set | Removes the first matching element. No-op if not found. |
| `delete <expr> from <target> .` | map | Deletes the key. **Fatal error if the key doesn't exist.** |
| `sort <target> .` | list, set | Sorts in place, as Python does: numbers with numbers, text with text (capitals first), lists element by element. Mixing kinds is an error. |
| `reverse <target> .` | list, set | Reverses in place. |
| `insert <expr> to <target> at <expr> .` | list | Inserts at the given index. Fatal if index is out of range (`0..length` inclusive). |
| `put <expr> to <target> at <expr> .` | list, map | A list: replaces the item at the given index (`0..length-1`); the length stays the same. A map: sets the key to the value, new or not (`put 0 to ages at "Ann"`). |
| `put <expr> to <target> at <row>, <column> .` | matrix (`import linear`) | Replaces the number at that row and column. |
| `length of <expr> .` | list, set, map, string | Prints the count (map: number of keys; string: character count, in Unicode code points, not bytes). |
| `min of <expr> .` | list, set, map | Prints the smallest element (list/set) or smallest **key** (map; values aren't compared). |
| `max of <expr> .` | list, set, map | Same as `min of`, but largest. |

`min` and `max` compare the items of a list or set, and the items must be
numbers, text or booleans, all of one kind. A list or set inside a list or
set is an error: for a list of lists, a set of sets, or a record, use
`max_sort` or `min_sort` from `import sort`, which take a key, as in
`max_sort[rows, "v", "first"]`.
| `min[<expr>]` | list, set, map | The smallest element, as an expression: `x = min[nums]`. Same as `min of nums`. |
| `min[<a>, <b>, ...]` | any values that compare | The smallest of the values: `lo = min[3, 5]` is 3. Mixing numbers and text is an error, as in `sort`. |
| `max[<expr>]`, `max[<a>, <b>, ...]` | as `min` | The largest, the same way. |

`min[...]` and `max[...]` need no `of`, so they can be used anywhere an
expression can, including right before a method: `min[nums] at pow[2]` is the
smallest number squared, where `min of nums at pow[2]` would apply the method
to `nums` alone. `min of` and `max of` keep working as before.

`length of`/`min of`/`max of` are also ordinary expressions usable anywhere
(`x = 1 + length of nums`) — they only print when they're the entire
statement. They apply to the value right after `of`, so
`length of nums == 0` compares the length with 0. For the length of a
computed value, store it first: `ab = a + b` then `length of ab`.

## Method-call form

Every name is lowercase: `isempty`, `getkeys`, `indexof`. (Before
v0.9.152 a few had capitals: `isEmpty`, `isNumber`, `getKeys`,
`getValues`, `indexOf`, `toString`, and system's `scriptFolder`, `isFile`,
`isFolder`. Those spellings still work.)

```
<result> is <receiver> at <method> [<arg> {, <arg>}] .
```

Anywhere else, `x at method[a, b]`, or with one argument
`x at method a` (just the value after it: `nums at get 0`,
`2 at pow 10`).

A method call can also be a line of its own, for what it does: then it
takes its arguments the way the `is` form does, with or without
brackets:

```
nums at add 4
nums at sort
nums at put 9, 0 .
nums at put[9, 0]
```

### list

| Method | Args | Returns |
|---|---|---|
| `add` | value | the list (mutated) |
| `len` / `length` | — | element count (Integer) |
| `isempty` | — | Boolean: `true` when there are no elements |
| `tostring` | — | display string |
| `clear` | — | the list, now empty |
| `count` | value | number of matching elements (Integer) |
| `index` | value | first matching index, or `-1` |
| `sort` | — | the list, sorted |
| `remove` | value | the list, first match removed (the earliest one when a value repeats). To remove by position, use `pop` (last, `pop[0]` first, `pop[i]` at `i`) |
| `reverse` | — | the list, reversed |
| `pop` | [position] | removes the item and gives it back. With no position, the **last** item. With a position, that item: `pop[0]` is the first, `pop[i]` the one at `i`. Fatal if empty or out of range. To remove without using the item, write `nums at pop .` as a statement. |
| `contains` / `find` | value | Boolean, whether present: `nums at contains[3]` |
| `insert` | value, index | the list, with value inserted |
| `put` | value, index | the list, with the item at index replaced: `nums at put[99, 2]` (the same order as `insert`) |
| `get` | index | the element at that index, from `0`; fatal if out of range (including negative indexes; use `slice` to count from the end) |
| `slice` | start [, end] | a new `list`, the elements `[start, end)`; same negative-index/clamping rules as string's `slice` below |

### set

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

### map

| Method | Args | Returns |
|---|---|---|
| `get` | key | the value (fatal if key not found) |
| `len` / `length` | — | how many entries (Integer), like `length of m` |
| `contains` | key | Boolean, whether the map has that key |
| `getvalues` | — | a list of values, insertion order |
| `getkeys` | — | a list of the keys, insertion order, each with its own type |
| `isempty` | — | Boolean: `true` when the map has no entries |
| `add` | value, key | the map: the statement `add value to m at key` (a new key gets the value; an existing number adds up, text joins on, a list or set takes the value) |
| `put` | value, key | the map, with the key set to the value (new or not) |
| `delete` | key | the map, with the entry removed (fatal if not found) |
| `invert` | — | new map: values become keys, keys become values. When two keys share a value, only one entry survives: the later key in insertion order (`{"Ana": "B", "Eli": "B"}` inverts to `{"B": "Eli"}`), and Turtle prints a warning naming the dropped key |
| `tostring` | — | display string |

### string

All indices are Unicode code points (runes), not bytes — consistent with
`length of` above and with `change ...to ascii/char` (see
[`reference.md`](../reference.md#type-conversion)).

| Method | Args | Returns |
|---|---|---|
| `len` / `length` | — | how many characters (Integer), like `length of s` |
| `upper` / `lower` | — | case-converted string. `upper` turns `ß` into `SS`, as Python does (`"straße" at upper` is `"STRASSE"`) |
| `reverse` | — | the text backwards, character by character: `"héllo" at reverse` is `"olléh"` |
| `isempty` | — | Boolean: `true` for `""` |
| `trim` | — | leading/trailing whitespace stripped |
| `get` | index | the one-character string at that index, from `0`; fatal if out of range (including negative) |
| `slice` | start [, end] | substring `[start, end)`; see below |
| `split` | separator | a `list` of substrings |
| `contains` | substring | Boolean |
| `indexof` | substring | first matching index, or `-1` |
| `replace` | old, new | new string, all occurrences of `old` replaced with `new` |
| `isnumber` | — | Boolean: would `change ... to integer/float` succeed on this string |
| `tostring` | — | itself |
| `padleft` | width [, fill] | the text with `fill` (one character, a space if left out) added on the left until it's `width` characters; longer text is left as it is |
| `padright` | width [, fill] | the same, added on the right |

```
id = "7"
show id at padleft[3, "0"] .         // 007
show "ab" at padright[5, "."], "|" . // ab...|
```

`isnumber` exists so you can validate untrusted input (from `?`) before
converting it, instead of letting a bad `change` crash the program:

```
raw = ? "Enter a number: "
ok is raw at isnumber .
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

---

Copyright 2017-2026 Nigele McCoy. Licensed under the
[Apache License 2.0](../../LICENSE); see [NOTICE](../../NOTICE).
