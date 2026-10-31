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

**Case-insensitive text** is a function that gives the lower-case text,
for example `min_sort[names, s give s at lower]`
puts `"Alice"` among the `a`s. Plain `sort names .` compares the
characters themselves, so capitals come first.

**How much** (`min_sort` and `max_sort` only): `"first"` gives just the
first item, or `none` when there's nothing; a number `n` gives the first
`n`. So `max_sort[books, "price", "first"]` is the most expensive book.

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

<!-- generated: turtle doc `sort` (regenerate with the script; do not edit by hand) -->
## Function reference

The entries below are copied from the binary's own documentation (`turtle doc sort`), so they match the version you run.

```text
import sort
```

### `Putting things in order. Works on lists, sets, maps (their entries),`

```text
Putting things in order. Works on lists, sets, maps (their entries),
text (its characters), and anything inside them: lists of assembled
values, maps of maps, lists of lists... Gives back a new list (a new
map, for a map) and leaves the original as it was.
Values order the same way everywhere: none, then true/false, numbers,
text, dates, lists (item by item), assembled values (field by field),
maps (entry by entry).
```

### `min_sort[collection, key, how]`

```text
min_sort[collection, key, how]        (import sort)
  Puts things in order, smallest first.
  collection   a list, set, map or text
  key          optional: what to order by. A function (b give b at get["price"]),
               a map key or field name ("price"), a position in a list of lists
               (1), or a list of those (list ["author", "price"]): ties go to
               the next. For a map, the key picks from each value; a function
               gets the value (v give ...) or the key and value ([k, v] give ...).
               Leave it out (or none) to order the items themselves.
  how          optional: "first" gives just the first item (none if
               there's nothing); a number gives that many, as a list.
               For a map, "first" gives the key; a number, a smaller map.
  Gives back: a new list (or map), equal items in their first order.
  Example:
    books = list [map ["price": 900], map ["price": 1500]]
    cheap = min_sort[books, b give price of b]
    show cheap .
```

### `max_sort[collection, key, how]`

```text
max_sort[collection, key, how]        (import sort)
  Like min_sort, largest first: max_sort[x, key, "first"] is the largest.
  Example:
    scores = list [70, 95, 88, 60]
    top3 = max_sort[scores, s give s, 3]
    show top3 .
```

### `is_sorted[collection, key]`

```text
is_sorted[collection, key]        (import sort)
  Whether it's already in order, smallest first (equal items allowed).
  Gives back: true or false.
```

### `reverse_list[collection]`

```text
reverse_list[collection]        (import sort)
  The items in the opposite order (a map: its entries). The reverse
  statement (reverse nums .) changes a list in place instead.
  Example:
    log = list ["first", "second"]
    newest_first = reverse_list[log]
    show newest_first .
```

### `bubble_sort[collection, key]`

```text
bubble_sort[collection, key]        (import sort)
  The same answer as min_sort, by bubble sort: swaps neighbours that are
  out of order until nothing moves. About n*n steps: for learning and
  comparing, not for large lists.
```

### `insertion_sort[collection, key]`

```text
insertion_sort[collection, key]        (import sort)
  min_sort's answer by insertion sort: slides each item back into place.
  Quick on lists that are almost in order.
```

### `selection_sort[collection, key]`

```text
selection_sort[collection, key]        (import sort)
  min_sort's answer by selection sort: picks the smallest of the rest,
  again and again. Equal items may change order.
```

### `merge_sort[collection, key]`

```text
merge_sort[collection, key]        (import sort)
  min_sort's answer by merge sort: sorts each half, then merges them.
  About n*log(n) steps, always.
```

### `quick_sort[collection, key]`

```text
quick_sort[collection, key]        (import sort)
  min_sort's answer by quick sort: splits around a middle value. Fast on
  average; equal items may change order.
```

### `heap_sort[collection, key]`

```text
heap_sort[collection, key]        (import sort)
  min_sort's answer by heap sort: builds a heap, then takes the largest
  off the top. Equal items may change order.
```

### `shell_sort[collection, key]`

```text
shell_sort[collection, key]        (import sort)
  min_sort's answer by Shell sort: insertion sort over shrinking gaps.
  Equal items may change order.
```

### `counting_sort[collection, key]`

```text
counting_sort[collection, key]        (import sort)
  min_sort's answer by counting sort, for whole-number keys in a modest
  range (ages, scores): counts each value, then places them.
```

### `radix_sort[collection, key]`

```text
radix_sort[collection, key]        (import sort)
  min_sort's answer by radix sort, for whole-number keys: orders them one
  byte at a time.
```

<!-- end generated -->
