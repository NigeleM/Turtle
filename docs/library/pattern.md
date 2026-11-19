# `pattern` — regular expressions

`import pattern`

Find, check and replace text with regular expressions: patterns that describe what text looks like.


## Functions

`matches`, `findall`, `replaceall`, `splitby`, `groups`.

### `matches[text, pattern]`

Whether the pattern is found anywhere in text (^ and $ for all of it).

**Example**

```turtle
code = "ABC-1234"
if ] matches[code, `^[A-Z]{3}-\d{4}$`] [
    show "valid code" .
if [end]
```

**Output**

```text
valid code
```

### `findall[text, pattern]`

Every match, in order.

**Example**

```turtle
show findall["a1 b22", `\d+`] .
```

**Output**

```text
[ "1", "22" ]
```

### `replaceall[text, pattern, with]`

Replaces every match.

**Example**

```turtle
show replaceall["2026-10-06", `(\d+)-(\d+)-(\d+)`, `$3/$2/$1`] .
```

**Output**

```text
06/10/2026
```

### `splitby[text, pattern]`

Splits text wherever the pattern matches.

**Example**

```turtle
show splitby["a, b;c", `[,;]\s*`] .
```

**Output**

```text
[ "a", "b", "c" ]
```

### `groups[text, pattern]`

The parts in ( ) of the first match; a group that took no part in it is "".

**Example**

```turtle
show groups["2026-10-06", `(\d+)-(\d+)`] .
```

**Output**

```text
[ "2026", "10" ]
```
