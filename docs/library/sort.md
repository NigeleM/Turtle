# `sort` — sorting

`import sort`

Sort lists and maps by value or by a key, with several sorting methods to choose from.


## Functions

`min_sort`, `max_sort`, `is_sorted`, `reverse_list`, `bubble_sort`, `insertion_sort`, `selection_sort`, `merge_sort`, `quick_sort`, `heap_sort`, `shell_sort`, `counting_sort`, `radix_sort`.

### `min_sort[collection, key, how]`

Puts things in order, smallest first. collection   a list, set, map or text key          optional: what to order by.

**Example**

```turtle
books = list [map ["price": 900], map ["price": 1500]]
cheap = min_sort[books, b give price of b]
show cheap .
```

**Output**

```text
[ { "price": 900 }, { "price": 1500 } ]
```

### `max_sort[collection, key, how]`

Like min_sort, largest first: max_sort[x, key, "first"] is the largest.

**Example**

```turtle
scores = list [70, 95, 88, 60]
top3 = max_sort[scores, s give s, 3]
show top3 .
```

**Output**

```text
[ 95, 88, 70 ]
```

### `is_sorted[collection, key]`

Whether it's already in order, smallest first (equal items allowed).

**Example**

```turtle
show is_sorted[list [1, 2, 2, 5]] .
```

**Output**

```text
true
```

### `reverse_list[collection]`

The items in the opposite order (a map: its entries).

**Example**

```turtle
log = list ["first", "second"]
newest_first = reverse_list[log]
show newest_first .
```

**Output**

```text
[ "second", "first" ]
```

### `bubble_sort[collection, key]`

The same answer as min_sort, by bubble sort: swaps neighbours that are out of order until nothing moves.

**Example**

```turtle
show bubble_sort[list [3, 1, 2]] .
```

**Output**

```text
[ 1, 2, 3 ]
```

### `insertion_sort[collection, key]`

min_sort's answer by insertion sort: slides each item back into place.

**Example**

```turtle
show insertion_sort[list [3, 1, 2]] .
```

**Output**

```text
[ 1, 2, 3 ]
```

### `selection_sort[collection, key]`

min_sort's answer by selection sort: picks the smallest of the rest, again and again.

**Example**

```turtle
show selection_sort[list [3, 1, 2]] .
```

**Output**

```text
[ 1, 2, 3 ]
```

### `merge_sort[collection, key]`

min_sort's answer by merge sort: sorts each half, then merges them.

**Example**

```turtle
show merge_sort[list [3, 1, 2]] .
```

**Output**

```text
[ 1, 2, 3 ]
```

### `quick_sort[collection, key]`

min_sort's answer by quick sort: splits around a middle value.

**Example**

```turtle
show quick_sort[list [3, 1, 2]] .
```

**Output**

```text
[ 1, 2, 3 ]
```

### `heap_sort[collection, key]`

min_sort's answer by heap sort: builds a heap, then takes the largest off the top.

**Example**

```turtle
show heap_sort[list [3, 1, 2]] .
```

**Output**

```text
[ 1, 2, 3 ]
```

### `shell_sort[collection, key]`

min_sort's answer by Shell sort: insertion sort over shrinking gaps.

**Example**

```turtle
show shell_sort[list [3, 1, 2]] .
```

**Output**

```text
[ 1, 2, 3 ]
```

### `counting_sort[collection, key]`

min_sort's answer by counting sort, for whole-number keys in a modest range (ages, scores): counts each value, then places them.

**Example**

```turtle
show counting_sort[list [3, 1, 2]] .
```

**Output**

```text
[ 1, 2, 3 ]
```

### `radix_sort[collection, key]`

min_sort's answer by radix sort, for whole-number keys: orders them one byte at a time.

**Example**

```turtle
show radix_sort[list [30, 4, 12]] .
```

**Output**

```text
[ 4, 12, 30 ]
```
