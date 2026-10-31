# `search` — searching

[Library index](index.md) · `import search`

Finding items by a rule, and the classic search algorithms.


`import search` finds things in lists, sets, maps and text, and in what's
inside them. Keys work as in the sort library.

| Function | Gives back |
|---|---|
| `find_first[x, test]` | the first item `test` says yes to (a map: its key), or `none` |
| `find_last[x, test]` | the last one, or `none` |
| `find_all[x, test]` | every one, a new list (a map: a new map of those entries) |
| `find_index[x, test]` | the first one's position, from 0, or `-1` |
| `count_where[x, test]` | how many |
| `find_key[map, value]` | the first key holding `value`, or `none` |
| `linear_search[x, value [, key]]` | the position of the first item equal to `value`, or `-1`; works on anything |
| `binary_search`, `jump_search`, `exponential_search`, `interpolation_search`, `ternary_search` `[sorted, value [, key]]` | the same answer, faster, on a collection sorted smallest first by the same key |
| `insert_position[sorted, value [, key]]` | where `value` would go to keep it sorted |

`test` is a function giving `true` or `false`: `b give price of b < 1000`.

```
import sort
import search

assemble Book [title, price]
books = list [Book["Dune", 950], Book["Emma", 700], Book["Kindred", 950]]

cheap = find_all[books, b give price of b < 900]          // [ Book Emma ]
by_price = min_sort[books, "price"]
spot = binary_search[by_price, 950, "price"]                // 1: the first 950
pos = insert_position[by_price, 800, "price"]               // 1
```

`binary_search` takes about 20 steps for a million items where
`linear_search` may take a million, but it needs the collection sorted
first, by the same key.

---

Copyright 2017-2026 Nigele McCoy. Licensed under the
[Apache License 2.0](../../LICENSE); see [NOTICE](../../NOTICE).

<!-- generated: turtle doc `search` (regenerate with the script; do not edit by hand) -->
## Function reference

The entries below are copied from the binary's own documentation (`turtle doc search`), so they match the version you run.

```text
import search
```

### `Finding things. Works on lists, sets, maps and text, and what's inside`

```text
Finding things. Works on lists, sets, maps and text, and what's inside
them, like the sort library. Over a map, a function gets the value
(v give ...) or the key and value ([k, v] give ...).
```

### `find_first[collection, test]`

```text
find_first[collection, test]        (import search)
  The first item for which test says yes.
  test   a function giving true or false: b give price of b < 1000
  Gives back: the item (for a map, its key), or none if nothing matches.
  Example:
    books = list [map ["title": "Emma", "price": 900, "stock": 0], map ["title": "Ida", "price": 1500, "stock": 4]]
    cheap = find_first[books, b give price of b < 1000]
    show cheap .
    out_of_stock = find_all[books, b give stock of b == 0]
    show length of out_of_stock .
```

### `find_last[collection, test]`

```text
find_last[collection, test]        (import search)
  The last item for which test says yes, or none.
```

### `find_all[collection, test]`

```text
find_all[collection, test]        (import search)
  Every item for which test says yes, in order.
  Gives back: a new list (for a map, a new map of the matching entries).
  Example:
    books = list [map ["title": "Emma", "stock": 0], map ["title": "Ida", "stock": 4]]
    out_of_stock = find_all[books, b give stock of b == 0]
    show length of out_of_stock .
```

### `find_index[collection, test]`

```text
find_index[collection, test]        (import search)
  The position (from 0) of the first item for which test says yes, or -1.
```

### `count_where[collection, test]`

```text
count_where[collection, test]        (import search)
  How many items test says yes to.
```

### `find_key[map, value]`

```text
find_key[map, value]        (import search)
  The first key whose value is value, or none.
  Example:
    ages = map ["Ann": 30, "Bo": 25]
    who = find_key[ages, 30]
    show who .
```

### `linear_search[collection, value, key]`

```text
linear_search[collection, value, key]        (import search)
  The position of the first item equal to value (by key, if given),
  checking each in turn, or -1. Works on anything, sorted or not.
  key     optional: what to order by. A function (b give b at get["price"]),
          a map key or field name ("price"), a position in a list of lists
          (1), or a list of those (list ["author", "price"]): ties go to
          the next. For a map, the key picks from each value; a function
          gets the value (v give ...) or the key and value ([k, v] give ...).
          Leave it out (or none) to order the items themselves.
  Example:
    names = list ["Ann", "Bo"]
    pos = linear_search[names, "Ann"]
    show pos .
```

### `binary_search[sorted, value, key]`

```text
binary_search[sorted, value, key]        (import search)
  Like linear_search, on a collection sorted smallest first by the same
  key (min_sort makes one): halves the range each step, about 20 steps
  for a million items. Gives the first match's position, or -1.
  Example:
    books = list [map ["price": 900], map ["price": 1500]]
    import sort
    books = list [map ["price": 900], map ["price": 1500]]
    import sort
    books = list [map ["price": 900], map ["price": 1500]]
    by_price = min_sort[books, "price"]
    spot = binary_search[by_price, 1500, "price"]
    show spot .
```

### `jump_search[sorted, value, key]`

```text
jump_search[sorted, value, key]        (import search)
  binary_search's answer, by jumping ahead in blocks of about the square
  root of the length, then looking through one block.
```

### `exponential_search[sorted, value, key]`

```text
exponential_search[sorted, value, key]        (import search)
  binary_search's answer, by doubling a bound (1, 2, 4, 8, ...) then
  searching inside it. Quick when the value is near the front.
```

### `interpolation_search[sorted, value, key]`

```text
interpolation_search[sorted, value, key]        (import search)
  binary_search's answer, guessing the position from the values at the
  ends, like opening a phone book near the right letter. Very fast on
  evenly spread numbers.
```

### `ternary_search[sorted, value, key]`

```text
ternary_search[sorted, value, key]        (import search)
  binary_search's answer, splitting the range in three each step.
```

### `insert_position[sorted, value, key]`

```text
insert_position[sorted, value, key]        (import search)
  Where value would go to keep a sorted collection in order: the
  position of the first item not smaller than it.
  Example:
    scores = list [60, 70, 90]
    pos = insert_position[scores, 75]
    show pos .
    insert 75 to scores at pos .
    show scores .
```

<!-- end generated -->
