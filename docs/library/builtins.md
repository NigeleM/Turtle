# Built-in types and methods

Lists, sets, maps, text, numbers, matrices and dates: making them, changing them, and the methods every value has. No import is needed.

## Lists

Methods: `add`, `get`, `put`, `insert`, `remove`, `pop`, `slice`, `contains`, `index`, `count`, `sort`, `reverse`, `clear`, `len`, `isempty`, `tostring`.

### `add`

adds a value at the end of the list (changes the list).

**Example**

```turtle
nums = list [4, 9]
nums at add[7]
show nums .
```

**Output**

```text
[ 4, 9, 7 ]
```

### `get`

gives back the item at an index. Positions start at 0.

**Example**

```turtle
nums = list [4, 9, 2]
show nums at get[1] .
```

**Output**

```text
9
```

### `put`

replaces the item at an index (changes the list). The value comes first, then the index.

**Example**

```turtle
nums = list [4, 9, 2]
nums at put[99, 1]
show nums .
```

**Output**

```text
[ 4, 99, 2 ]
```

### `insert`

inserts a value before an index, so the items after it move up (changes the list).

**Example**

```turtle
nums = list [4, 9, 2]
nums at insert[0, 1]
show nums .
```

**Output**

```text
[ 4, 0, 9, 2 ]
```

### `remove`

removes the first item equal to the value (changes the list). A value that is not there is not an error.

**Example**

```turtle
nums = list [4, 9, 2, 9]
nums at remove[9]
show nums .
nums at remove[42]
show nums .
```

**Output**

```text
[ 4, 2, 9 ]
[ 4, 2, 9 ]
```

### `pop`

removes an item and gives it back. With no index it takes the last item; with an index it takes that item.

**Example**

```turtle
nums = list [4, 9, 2]
show nums at pop .
show nums at pop[0] .
show nums .
```

**Output**

```text
2
4
[ 9 ]
```

### `slice`

gives a new list of the items from a start position up to, but not including, an end position. With no end, it goes to the end of the list. The original list is unchanged.

**Example**

```turtle
nums = list [4, 9, 2, 9]
show nums at slice[1, 3] .
show nums at slice[1] .
show nums .
```

**Output**

```text
[ 9, 2 ]
[ 9, 2, 9 ]
[ 4, 9, 2, 9 ]
```

### `contains`

gives `true` when the value is in the list. `find` is another name for `contains`.

**Example**

```turtle
nums = list [4, 9, 2]
show nums at contains[2] .
show nums at find[7] .
```

**Output**

```text
true
false
```

### `index`

gives the position of the first item equal to the value, or `-1` when there is none.

**Example**

```turtle
nums = list [4, 9, 2, 9]
show nums at index[9] .
show nums at index[7] .
```

**Output**

```text
1
-1
```

### `count`

gives how many items are equal to the value.

**Example**

```turtle
nums = list [4, 9, 2, 9]
show nums at count[9] .
```

**Output**

```text
2
```

### `sort`

sorts the list in place (changes the list).

**Example**

```turtle
nums = list [4, 9, 2]
nums at sort
show nums .
```

**Output**

```text
[ 2, 4, 9 ]
```

### `reverse`

reverses the list in place (changes the list).

**Example**

```turtle
nums = list [4, 9, 2]
nums at reverse
show nums .
```

**Output**

```text
[ 2, 9, 4 ]
```

### `clear`

removes every item (changes the list).

**Example**

```turtle
nums = list [4, 9, 2]
nums at clear
show nums .
```

**Output**

```text
[]
```

### `len`

gives the number of items. `length` is another name for `len`.

**Example**

```turtle
nums = list [4, 9, 2]
show nums at len .
show nums at length .
```

**Output**

```text
3
3
```

### `isempty`

gives `true` when the list has no items.

**Example**

```turtle
nums = list []
show nums at isempty .
nums at add[1]
show nums at isempty .
```

**Output**

```text
true
false
```

### `tostring`

gives the list as text.

**Example**

```turtle
nums = list [4, 9]
text = nums at tostring
show "the list is {text}" .
```

**Output**

```text
the list is [ 4, 9 ]
```

## Sets

Methods: `add`, `insert`, `get`, `remove`, `contains`, `count`, `union`, `intersection`, `difference`, `subset`, `superset`, `pop`, `sort`, `clear`, `len`, `isempty`, `tostring`.

### `add`

adds a value. A value already in the set is not added twice (changes the set).

**Example**

```turtle
colors = set ["red", "blue"]
colors at add["red"]
colors at add["green"]
show colors .
```

**Output**

```text
{ "red", "blue", "green" }
```

### `insert`

puts a value at a position in the set's order (changes the set). The value comes first, then the position. Sets have no order that means anything, so the book uses `add` instead.

**Example**

```turtle
colors = set ["red", "blue"]
colors at insert["green", 0]
show colors .
```

**Output**

```text
{ "green", "red", "blue" }
```

### `get`

gives the item at a position in the set's order. Positions start at 0. Sets have no order that means anything, so use `get` only to look at the items one by one.

**Example**

```turtle
colors = set ["red", "blue", "green"]
show colors at get[0] .
show colors at get[2] .
```

**Output**

```text
red
green
```

### `remove`

removes a value (changes the set). A value that is not there is not an error.

**Example**

```turtle
colors = set ["red", "blue"]
colors at remove["red"]
colors at remove["pink"]
show colors .
```

**Output**

```text
{ "blue" }
```

### `contains`

gives `true` when the value is in the set. `find` is another name for `contains`.

**Example**

```turtle
colors = set ["red", "blue"]
show colors at contains["red"] .
show colors at find["pink"] .
```

**Output**

```text
true
false
```

### `count`

gives `1` when the value is in the set, and `0` when it is not.

**Example**

```turtle
colors = set ["red", "blue"]
show colors at count["blue"] .
show colors at count["pink"] .
```

**Output**

```text
1
0
```

### `union`

gives a new set with the items of both sets. The original sets are unchanged.

**Example**

```turtle
a = set [1, 2]
b = set [2, 3]
show a at union[b] .
show a .
```

**Output**

```text
{ 1, 2, 3 }
{ 1, 2 }
```

### `intersection`

gives a new set with the items that are in both sets.

**Example**

```turtle
a = set [1, 2]
b = set [2, 3]
show a at intersection[b] .
```

**Output**

```text
{ 2 }
```

### `difference`

gives a new set with the items of this set that are not in the other set.

**Example**

```turtle
a = set [1, 2]
b = set [2, 3]
show a at difference[b] .
```

**Output**

```text
{ 1 }
```

### `subset`

gives `true` when every item of this set is in the other set.

**Example**

```turtle
a = set [1, 2]
b = set [1, 2, 3]
show a at subset[b] .
show b at subset[a] .
```

**Output**

```text
true
false
```

### `superset`

gives `true` when every item of the other set is in this set.

**Example**

```turtle
a = set [1, 2]
b = set [1, 2, 3]
show b at superset[a] .
show a at superset[b] .
```

**Output**

```text
true
false
```

### `pop`

removes an item and gives it back. It takes the item added most recently (changes the set).

**Example**

```turtle
colors = set ["red", "blue"]
show colors at pop .
show colors .
```

**Output**

```text
blue
{ "red" }
```

### `sort`

sorts the items (changes the set). The result is the same set, shown in sorted order.

**Example**

```turtle
nums = set [4, 8, 1]
nums at sort
show nums .
```

**Output**

```text
{ 1, 4, 8 }
```

### `clear`

removes every item (changes the set).

**Example**

```turtle
colors = set ["red", "blue"]
colors at clear
show colors .
```

**Output**

```text
{}
```

### `len`

gives the number of items. `length` is another name for `len`.

**Example**

```turtle
colors = set ["red", "blue"]
show colors at len .
```

**Output**

```text
2
```

### `isempty`

gives `true` when the set has no items.

**Example**

```turtle
colors = set []
show colors at isempty .
```

**Output**

```text
true
```

### `tostring`

gives the set as text.

**Example**

```turtle
colors = set ["red"]
text = colors at tostring
show "the set is {text}" .
```

**Output**

```text
the set is { "red" }
```

## Maps

Methods: `get`, `add`, `put`, `delete`, `contains`, `getkeys`, `getvalues`, `invert`, `len`, `isempty`, `tostring`.

### `get`

gives the value for a key. A key that is not in the map is an error, so check with `contains` first when it may be missing.

**Example**

```turtle
ages = map ["Alice": 30, "Bob": 25]
show ages at get["Alice"] .
```

**Output**

```text
30
```

### `add`

adds to the value at a key (changes the map). For a number, the amount is added to the value, and a new key starts from the amount.

**Example**

```turtle
ages = map ["Alice": 30]
ages at add["Alice", 1]
ages at add["Cy", 5]
show ages .
```

**Output**

```text
{ "Alice": 31, "Cy": 5 }
```

### `put`

sets the value at a key, whether or not the key is there (changes the map).

**Example**

```turtle
ages = map ["Alice": 30]
ages at put["Alice", 31]
ages at put["Dee", 19]
show ages .
```

**Output**

```text
{ "Alice": 31, "Dee": 19 }
```

### `delete`

removes a key and its value (changes the map). A key that is not there is an error.

**Example**

```turtle
ages = map ["Alice": 30, "Bob": 25]
ages at delete["Alice"]
show ages .
```

**Output**

```text
{ "Bob": 25 }
```

### `contains`

gives `true` when the key is in the map.

**Example**

```turtle
ages = map ["Alice": 30]
show ages at contains["Alice"] .
show ages at contains["Zed"] .
```

**Output**

```text
true
false
```

### `getkeys`

gives a list of the keys, in the order they were added.

**Example**

```turtle
ages = map ["Bob": 25, "Alice": 30]
show ages at getkeys .
```

**Output**

```text
[ "Bob", "Alice" ]
```

### `getvalues`

gives a list of the values, in the same order as the keys.

**Example**

```turtle
ages = map ["Bob": 25, "Alice": 30]
show ages at getvalues .
```

**Output**

```text
[ 25, 30 ]
```

### `invert`

gives a new map with each key and value exchanged. The values become the keys. When two keys share a value, Turtle prints a warning and keeps only the last key.

**Example**

```turtle
ages = map ["Bob": 25, "Alice": 30]
show ages at invert .
show ages .
```

**Output**

```text
{ 25: "Bob", 30: "Alice" }
{ "Bob": 25, "Alice": 30 }
```

### `len`

gives the number of entries. `length` is another name for `len`.

**Example**

```turtle
ages = map ["Bob": 25, "Alice": 30]
show ages at len .
```

**Output**

```text
2
```

### `isempty`

gives `true` when the map has no entries.

**Example**

```turtle
ages = map []
show ages at isempty .
```

**Output**

```text
true
```

### `tostring`

gives the map as text.

**Example**

```turtle
ages = map ["Bob": 25]
text = ages at tostring
show "the map is {text}" .
```

**Output**

```text
the map is { "Bob": 25 }
```

## Strings

Methods: `upper`, `lower`, `reverse`, `trim`, `get`, `slice`, `split`, `contains`, `indexof`, `replace`, `padleft`, `padright`, `isnumber`, `len`, `isempty`.

### `upper`

gives back the text in capital letters.

**Example**

```turtle
show "turtle" at upper .
```

**Output**

```text
TURTLE
```

### `lower`

gives back the text in small letters.

**Example**

```turtle
show "TURTLE" at lower .
```

**Output**

```text
turtle
```

### `reverse`

gives back the text with its characters in reverse order.

**Example**

```turtle
show "turtle" at reverse .
```

**Output**

```text
eltrut
```

### `trim`

gives back the text with spaces removed from both ends.

**Example**

```turtle
show "  hello  " at trim .
```

**Output**

```text
hello
```

### `get`

gives back the character at an index. Positions start at 0; an index past the end is an error.

**Example**

```turtle
show "turtle" at get[1] .
```

**Output**

```text
u
```

### `slice`

gives back the part of the text from `start` up to, but not including, `end`.

**Example**

```turtle
show "turtle" at slice[0, 3] .
```

**Output**

```text
tur
```

### `split`

gives back a list of the pieces between each `separator`.

**Example**

```turtle
show "a,b,c" at split[","] .
```

**Output**

```text
[ "a", "b", "c" ]
```

### `contains`

gives back `true` if the text contains `text`.

**Example**

```turtle
show "turtle" at contains["urt"] .
```

**Output**

```text
true
```

### `indexof`

gives back the position of the first `text` in the string, or `-1` if it is not there.

**Example**

```turtle
show "turtle" at indexof["rt"] .
show "turtle" at indexof["q"] .
```

**Output**

```text
2
-1
```

### `replace`

gives back the text with every `old` changed to `new`.

**Example**

```turtle
show "a-b-c" at replace["-", "+"] .
```

**Output**

```text
a+b+c
```

### `padleft`

gives back the text padded with spaces on the left to at least `width` characters.

**Example**

```turtle
show "7" at padleft[3] .
```

**Output**

```text
  7
```

### `padright`

gives back the text padded with spaces on the right to at least `width` characters.

**Example**

```turtle
show "7" at padright[3] .
```

**Output**

```text
7  
```

### `isnumber`

gives back `true` if the text is a number, such as `"42"` or `"3.5"`.

**Example**

```turtle
show "42" at isnumber .
show "x3" at isnumber .
```

**Output**

```text
true
false
```

### `len`

gives back how many characters the text has.

**Example**

```turtle
show "turtle" at len .
```

**Output**

```text
6
```

### `isempty`

gives back `true` if the text has no characters.

**Example**

```turtle
show "" at isempty .
```

**Output**

```text
true
```

## Numbers

Methods: `fixed`, `commas`, `commas[places]`, `sqrt`, `abs`, `round`, `floor`, `ceil`, `pow`, `random`.

### `fixed`

gives back the number as text with `places` digits after the point.

**Example**

```turtle
show 3.14159 at fixed[2] .
```

**Output**

```text
3.14
```

### `commas`

gives back a whole number as text with commas every three digits.

**Example**

```turtle
show 1234567 at commas .
```

**Output**

```text
1,234,567
```

### `commas[places]`

gives back a number as text with commas in the whole part and `places` digits after the point.

**Example**

```turtle
show 1234567.891 at commas[2] .
```

**Output**

```text
1,234,567.89
```

### `sqrt`

gives back the square root as a float. It needs `import math`. A negative number is a math error.

**Example**

```turtle
import math
show 16 at sqrt .
```

**Output**

```text
4.0
```

### `abs`

gives back the number without its sign. It needs `import math`.

**Example**

```turtle
import math
show -7 at abs .
```

**Output**

```text
7
```

### `round`

gives back the number rounded to the nearest whole number. It needs `import math`.

**Example**

```turtle
import math
show 2.6 at round .
```

**Output**

```text
3
```

### `floor`

gives back the largest whole number that is not more than the number. It needs `import math`.

**Example**

```turtle
import math
show 2.6 at floor .
```

**Output**

```text
2
```

### `ceil`

gives back the smallest whole number that is not less than the number. It needs `import math`.

**Example**

```turtle
import math
show 2.1 at ceil .
```

**Output**

```text
3
```

### `pow`

gives back the number raised to `power`. It needs `import math`.

**Example**

```turtle
import math
show 2 at pow[10] .
```

**Output**

```text
1024
```

### `random`

gives back a random whole number from 0 up to, but not including, the number. It needs `import math`. The example checks the range rather than printing the value, because the value changes each run.

**Example**

```turtle
import math
r = 10 at random
show r >= 0 && r < 10 .
```

**Output**

```text
true
```

## Matrices

Methods: `rows`, `columns`, `shape`, `get`, `put`, `row`, `column`, `flatten`, `reshape`, `isempty`, `tostring`.

### `rows`

gives back how many rows the matrix has. It needs `import linear`.

**Example**

```turtle
import linear
m = matrix [1, 2; 3, 4]
show m at rows .
```

**Output**

```text
2
```

### `columns`

gives back how many columns the matrix has.

**Example**

```turtle
import linear
m = matrix [1, 2; 3, 4]
show m at columns .
```

**Output**

```text
2
```

### `shape`

gives back the rows and columns together, as a list `[rows, columns]`.

**Example**

```turtle
import linear
m = matrix [1, 2; 3, 4]
show m at shape .
```

**Output**

```text
[ 2, 2 ]
```

### `get`

gives back the item at row `r` and column `c`. Both start at 0.

**Example**

```turtle
import linear
m = matrix [1, 2; 3, 4]
show m at get[0, 1] .
```

**Output**

```text
2
```

### `put`

puts `value` at row `r` and column `c`. It changes the matrix.

**Example**

```turtle
import linear
m = matrix [1, 2; 3, 4]
m at put[9, 0, 0]
show m .
```

**Output**

```text
[ 9  2 ]
[ 3  4 ]
```

### `row`

gives back row `r` as a matrix.

**Example**

```turtle
import linear
m = matrix [1, 2; 3, 4]
show m at row[1] .
```

**Output**

```text
[ 3, 4 ]
```

### `column`

gives back column `c` as a matrix.

**Example**

```turtle
import linear
m = matrix [1, 2; 3, 4]
show m at column[0] .
```

**Output**

```text
[ 1, 3 ]
```

### `flatten`

gives back all the items in one row.

**Example**

```turtle
import linear
m = matrix [1, 2; 3, 4]
show m at flatten .
```

**Output**

```text
[ 1  2  3  4 ]
```

### `reshape`

gives back the same items arranged in `rows` rows and `columns` columns. The item count must match.

**Example**

```turtle
import linear
m = matrix [1, 2; 3, 4]
show m at reshape[1, 4] .
```

**Output**

```text
[ 1  2  3  4 ]
```

### `isempty`

gives back `true` if the matrix has no items.

**Example**

```turtle
import linear
m = matrix [1, 2; 3, 4]
show m at isempty .
```

**Output**

```text
false
```

### `tostring`

gives back the matrix as text.

**Example**

```turtle
import linear
m = matrix [1, 2; 3, 4]
show m at tostring .
```

**Output**

```text
[ 1  2 ]
[ 3  4 ]
```

## Dates

Methods: `tostring`.

### `tostring`

gives back the date as text, as `show` writes it. No import is needed.

**Example**

```turtle
import time
d = make_date[2026, 10, 10]
show d at tostring .
```

**Output**

```text
2026-10-10 00:00:00
```
