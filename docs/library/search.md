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
