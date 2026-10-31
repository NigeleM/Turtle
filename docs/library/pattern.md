# `pattern` — regular expressions

[Library index](index.md) · `import pattern`

Matching, finding, replacing and splitting text by pattern.


`import pattern` finds and changes text by pattern (regular expressions).
Write patterns in **backticks**: a backtick string keeps every character
as typed, so `{3}` isn't interpolation and `\d` isn't an escape (see
[`reference.md`](../reference.md#literals)).

| Function | Args | Returns |
|---|---|---|
| `matches` | text, pattern | `true` if the pattern is found anywhere in the text (`^` and `$` to match all of it) |
| `findall` | text, pattern | a list of every match, in order (empty if none) |
| `replaceall` | text, pattern, with | the text with every match replaced; in `with`, `$1` is the first group (see below) |
| `splitby` | text, pattern | a list of the pieces between matches |
| `groups` | text, pattern | a list of the parts in `( )` of the first match (`""` for a group that took no part), or `none` if nothing matches |

```
import pattern

code = "ab-1234"
if ] matches[code, `^[a-z]{2}-\d{4}$`] [
    show "valid" .
if [end]

show findall["a1 b22 c333", `\d+`] .                      // [ "1", "22", "333" ]
show replaceall["2026-10-06", `(\d+)-(\d+)-(\d+)`, `$3/$2/$1`] .   // 06/10/2026
show splitby["a, b;c", `[,;]\s*`] .                        // [ "a", "b", "c" ]
parts = groups["2026-10-06", `(\d+)-(\d+)`]               // [ "2026", "10" ]
```

The pattern words: `\d` a digit, `\w` an English letter, digit or `_`, `\s` a
space, `.` any character; `[abc]` one of, `[^abc]` none of; `+` one or
more, `*` any number, `?` maybe, `{3}` / `{2,4}` exactly / between; `^`
start, `$` end; `( )` a group; `a|b` either. Patterns have no backreferences, and no pattern can take forever. Patterns are compiled
once and reused. A pattern that isn't valid is an error of kind `pattern`.

**Letters in any language.** `\w` and `\b` (a word's edge) know only
English letters: ``findall["héllo", `\w+`]`` is `[ "h", "llo" ]`. For
letters in any language use `\p{L}`, and `[\p{L}\p{N}_]` for a word
character:

```
import pattern
show findall["héllo wörld", `\p{L}+`] .        // [ "héllo", "wörld" ]
```

**In the replacement**, `$1` or `${1}` is the first group, `$name` or
`${name}` a named group (`(?P<name>...)`), and `$$` a `$`. A number ends
where the digits do (`$1x` is group 1, then `x`), and a `$` that names no
group of the pattern stays as written:

```
import pattern
show replaceall["a-b", `(\w)-(\w)`, `$1x$2`] .           // axb
show replaceall["price: 5", `\d+`, "$10"] .              // price: $10
show replaceall["ann lee", `(?P<first>\w+) (?P<last>\w+)`, `$last, $first`] .   // lee, ann
```

Write the replacement in backticks too: inside `"..."`, `{1}` is
interpolation, so `"${1}"` is `"$1"`.

---

Copyright 2017-2026 Nigele McCoy. Licensed under the
[Apache License 2.0](../../LICENSE); see [NOTICE](../../NOTICE).

<!-- generated: turtle doc `pattern` (regenerate with the script; do not edit by hand) -->
## Function reference

The entries below are copied from the binary's own documentation (`turtle doc pattern`), so they match the version you run.

```text
import pattern
```

### `Patterns (regular expressions) in text. Write patterns in backticks,`

```text
Patterns (regular expressions) in text. Write patterns in backticks,
which keep { } and \ exactly as typed: \d is a digit, \s a space, \w an
English letter, digit or _ (\p{L} is a letter in any language: é, ß, я);
{3} three of the one before, + one or more, * any number, ? maybe; [abc]
one of; ^ start, $ end; ( ) a group. Patterns
cannot take forever, so every match finishes. A pattern that isn't valid is
an error of kind pattern.
```

### `matches[text, pattern]`

```text
matches[text, pattern]        (import pattern)
  Whether the pattern is found anywhere in text (^ and $ for all of it).
  Gives back: true or false.
  Example:
    code = "ABC-1234"
    if ] matches[code, `^[A-Z]{3}-\d{4}$`] [
        show "valid code" .
    if [end]
```

### `findall[text, pattern]`

```text
findall[text, pattern]        (import pattern)
  Every match, in order.
  Gives back: a list of text (empty if none).
  Example:
    findall["a1 b22", `\d+`]          // [ "1", "22" ]
```

### `replaceall[text, pattern, with]`

```text
replaceall[text, pattern, with]        (import pattern)
  Replaces every match. In with, $1 (or ${1}) is the first group, $name
  a named one, $$ a $; a $ that names no group stays as written ("$10").
  Write with in backticks too: in "...", {1} is interpolation.
  Example:
    replaceall["2026-10-06", `(\d+)-(\d+)-(\d+)`, `$3/$2/$1`]   // "06/10/2026"
```

### `splitby[text, pattern]`

```text
splitby[text, pattern]        (import pattern)
  Splits text wherever the pattern matches.
  Gives back: a list of text.
  Example:
    splitby["a, b;c", `[,;]\s*`]      // [ "a", "b", "c" ]
```

### `groups[text, pattern]`

```text
groups[text, pattern]        (import pattern)
  The parts in ( ) of the first match; a group that took no part in it
  is "".
  Gives back: a list of text, or none if nothing matches.
  Example:
    groups["2026-10-06", `(\d+)-(\d+)`]    // [ "2026", "10" ]
```

<!-- end generated -->
