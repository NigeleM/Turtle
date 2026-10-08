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
| `replaceall` | text, pattern, with | the text with every match replaced; in `with`, `$1` is the first group |
| `splitby` | text, pattern | a list of the pieces between matches |
| `groups` | text, pattern | a list of the parts in `( )` of the first match, or `none` if nothing matches |

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

The pattern words: `\d` a digit, `\w` a letter, digit or `_`, `\s` a
space, `.` any character; `[abc]` one of, `[^abc]` none of; `+` one or
more, `*` any number, `?` maybe, `{3}` / `{2,4}` exactly / between; `^`
start, `$` end; `( )` a group; `a|b` either. The syntax is Go's (RE2):
no backreferences, and no pattern can take forever. Patterns are compiled
once and reused. A pattern that isn't valid is an error of kind `pattern`.
