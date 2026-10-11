# `strings` — text functions

[Library index](index.md) · `import strings`

Finding, cutting and joining text.


`import strings` (or `import strings [find, join]`) provides string
functions that read well sentence-style. Three share their behaviour with
string methods: `find` is `indexof`, `substring` is `slice`, `isinstring`
is `contains`. All indices count characters, not bytes.

| Function | Args | Returns |
|---|---|---|
| `find[text, part]` | two strings | index of the first `part` in `text`, or `-1` |
| `substring[text, start [, end]]` | string, integer(s) | the characters `[start, end)`; negative counts from the end, `end` defaults to the end, out-of-range bounds clamp (same rules as `slice`) |
| `isinstring[part, text]` | two strings | Boolean: does `text` contain `part` |
| `join[items [, separator]]` | list or set, optional string | one string: each element's shown form, with `separator` (default `""`) between them |

```
import strings

line = "hello world"
show line find "wor" .           // 6
show line substring 0, 5 .       // hello
show line substring -5 .         // world: -5 counts from the end
show "wor" isinstring line .     // true
words is line at split " " .
show words join ", " .           // hello, world
```

`isinstring` takes the part first, so it reads as a sentence with the
literal on the left: `"wor" isinstring line`.

---

Copyright 2017-2026 Nigele McCoy. Licensed under the
[Apache License 2.0](../../LICENSE); see [NOTICE](../../NOTICE).
