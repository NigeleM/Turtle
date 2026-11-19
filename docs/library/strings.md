# `strings` — text functions

`import strings`

Text functions: find a part of a text, cut out a piece of it, and join a list of text into one.


## Functions

`find`, `substring`, `isinstring`, `join`.

### `find[text, part]`

Where part first appears in text.

**Example**

```turtle
line = "a word here"
show line find "wor" .
show line substring 0, 5 .
if ] "wor" isinstring line [
    show "found" .
if [end]
words = list ["red", "green"]
show words join ", " .
```

**Output**

```text
2
a wor
found
red, green
```

### `substring[text, start, end]`

The characters from start up to (not including) end. end is optional (to the end); a negative position counts from the end.

**Example**

```turtle
line = "a word here"
show line substring 0, 5 .
```

**Output**

```text
a wor
```

### `isinstring[part, text]`

Does text contain part?

**Example**

```turtle
line = "a word here"
if ] "wor" isinstring line [
    show "found" .
if [end]
```

**Output**

```text
found
```

### `join[items, separator]`

Joins a list (or set) into one text, with separator between the items (optional: nothing by default).

**Example**

```turtle
words = list ["red", "green"]
show words join ", " .
```

**Output**

```text
red, green
```
