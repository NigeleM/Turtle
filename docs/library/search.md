# `search` — searching

`import search`

Find items in lists, sets and maps, including fast searches on sorted lists such as binary search.


## Functions

`find_first`, `find_last`, `find_all`, `find_index`, `count_where`, `find_key`, `linear_search`, `binary_search`, `jump_search`, `exponential_search`, `interpolation_search`, `ternary_search`, `insert_position`.

### `find_first[collection, test]`

The first item for which test says yes. test   a function giving true or false: b give price of b < 1000 Gives back: the item (for a map, its key), or none if nothing matches.

**Example**

```turtle
books = list [map ["title": "Emma", "price": 900, "stock": 0], map ["title": "Ida", "price": 1500, "stock": 4]]
cheap = find_first[books, b give price of b < 1000]
show cheap .
out_of_stock = find_all[books, b give stock of b == 0]
show length of out_of_stock .
```

**Output**

```text
{ "title": "Emma", "price": 900, "stock": 0 }
1
```

### `find_last[collection, test]`

The last item for which test says yes, or none.

**Example**

```turtle
show find_last[list [1, 4, 6, 7], x give x % 2 == 0] .
```

**Output**

```text
6
```

### `find_all[collection, test]`

Every item for which test says yes, in order.

**Example**

```turtle
books = list [map ["title": "Emma", "stock": 0], map ["title": "Ida", "stock": 4]]
out_of_stock = find_all[books, b give stock of b == 0]
show length of out_of_stock .
```

**Output**

```text
1
```

### `find_index[collection, test]`

The position (from 0) of the first item for which test says yes, or -1.

**Example**

```turtle
show find_index[list [1, 4, 6, 7], x give x % 2 == 0] .
```

**Output**

```text
1
```

### `count_where[collection, test]`

How many items test says yes to.

**Example**

```turtle
show count_where[list [1, 4, 6, 7], x give x % 2 == 0] .
```

**Output**

```text
2
```

### `find_key[map, value]`

The first key whose value is value, or none.

**Example**

```turtle
ages = map ["Ann": 30, "Bo": 25]
who = find_key[ages, 30]
show who .
```

**Output**

```text
Ann
```

### `linear_search[collection, value, key]`

The position of the first item equal to value (by key, if given), checking each in turn, or -1.

**Example**

```turtle
names = list ["Ann", "Bo"]
pos = linear_search[names, "Ann"]
show pos .
```

**Output**

```text
0
```

### `binary_search[sorted, value, key]`

Like linear_search, on a collection sorted smallest first by the same key (min_sort makes one): halves the range each step, about 20 steps for a million items.

**Example**

```turtle
books = list [map ["price": 900], map ["price": 1500]]
import sort
books = list [map ["price": 900], map ["price": 1500]]
import sort
books = list [map ["price": 900], map ["price": 1500]]
by_price = min_sort[books, "price"]
spot = binary_search[by_price, 1500, "price"]
show spot .
```

**Output**

```text
1
```

### `jump_search[sorted, value, key]`

binary_search's answer, by jumping ahead in blocks of about the square root of the length, then looking through one block.

**Example**

```turtle
show jump_search[list [1, 3, 5, 7, 9], 7] .
```

**Output**

```text
3
```

### `exponential_search[sorted, value, key]`

binary_search's answer, by doubling a bound (1, 2, 4, 8, ...) then searching inside it.

**Example**

```turtle
show exponential_search[list [1, 3, 5, 7, 9], 7] .
```

**Output**

```text
3
```

### `interpolation_search[sorted, value, key]`

binary_search's answer, guessing the position from the values at the ends, like opening a phone book near the right letter.

**Example**

```turtle
show interpolation_search[list [1, 3, 5, 7, 9], 7] .
```

**Output**

```text
3
```

### `ternary_search[sorted, value, key]`

binary_search's answer, splitting the range in three each step.

**Example**

```turtle
show ternary_search[list [1, 3, 5, 7, 9], 7] .
```

**Output**

```text
3
```

### `insert_position[sorted, value, key]`

Where value would go to keep a sorted collection in order: the position of the first item not smaller than it.

**Example**

```turtle
scores = list [60, 70, 90]
pos = insert_position[scores, 75]
show pos .
insert 75 to scores at pos .
show scores .
```

**Output**

```text
2
[ 60, 70, 75, 90 ]
```
