# `sort` — sorting

[Library index](index.md) · `import sort`

Sorting by a function, field or position, and the classic sorting algorithms.


`import sort` puts things in order. Every function gives back a **new**
list (a new map, for a map) and leaves the original unchanged. The
`sort nums .` statement still sorts a list in place.

| Function | Gives back |
|---|---|
| `min_sort[x [, key [, how]]]` | `x` in order, smallest first; equal items keep their order |
| `max_sort[x [, key [, how]]]` | the same, largest first |
| `is_sorted[x [, key]]` | `true` when `x` is already smallest first |
| `reverse_list[x]` | the items (a map: its entries) in the opposite order |
| `bubble_sort`, `insertion_sort`, `selection_sort`, `merge_sort`, `quick_sort`, `heap_sort`, `shell_sort`, `counting_sort`, `radix_sort` `[x [, key]]` | `min_sort`'s answer, by that algorithm (for learning and comparing; `counting_sort` and `radix_sort` need whole-number keys) |

**The key** says what to order by:

| Key | Orders by | Example |
|---|---|---|
| left out, or `none` | the items themselves (a map: its keys) | `min_sort[nums]` |
| a function | what it gives for each item | `min_sort[books, b give price of b]` |
| text | that map key, or assembled field | `min_sort[books, "price"]` |
| an integer | that position in a list of lists | `min_sort[pairs, 1]` |
| a list of those | the first, then the next for ties | `min_sort[books, list ["author", "price"]]` |

**How much** (`min_sort` and `max_sort` only): `"first"` gives just the
first item, or `none` when there's nothing; a number `n` gives the first
`n`. So `max_sort[books, "price", "first"]` is the most expensive book,
like Python's `max(books, key=...)`.

**Maps** sort by entry and give back a new map in that order. With no
key, by the map's keys; a text or integer key picks from each value; a
function gets the value (`v give ...`) or the key and value
(`[k, v] give ...`). `"first"` gives a key.

```
import sort

ages = map ["Cy": 41, "Ann": 30, "Bo": 25]
show min_sort[ages] .                      // { "Ann": 30, "Bo": 25, "Cy": 41 }
show max_sort[ages, a give a] .           // { "Cy": 41, "Ann": 30, "Bo": 25 }
show min_sort[ages, a give a, "first"] .  // Bo
```

**One order for every value**, so mixed and nested collections sort with
or without a key: `none`, then `true`/`false` (false first), numbers
(`1` equals `1.0`), text (character by character), dates, lists and
sets (item by item, shorter first), assembled values (type name, then
field by field), maps (entry by entry). A function value can't be put
in order (an error of kind `type`).

---

Copyright 2017-2026 Nigele McCoy. Licensed under the
[Apache License 2.0](../../LICENSE); see [NOTICE](../../NOTICE).
