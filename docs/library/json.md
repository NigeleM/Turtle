# `json` — JSON

[Library index](index.md) · `import json`

JSON text and files to Turtle values and back.


`import json` (or `import json [load, json_get]`):

| Function | Args | Returns |
|---|---|---|
| `load[text]` | JSON text | the Turtle value |
| `json_text[value]` | any JSON-able value | JSON text on one line |
| `json_read[path]` | file path | the file's JSON as a Turtle value |
| `json_write[path, value]` | file path, value | writes the value as indented JSON; returns `none` |
| `json_get[value, step, ...]` | a value, then keys and indexes | the value at the end of the path, or `none` if any step is missing |

```
import json

cfg = json_read["config.json"]
port = json_get[cfg, "server", "port"]
if ] port == none [
    port = 8080
if [end]

user = load['{"name": "Ann", "tags": ["admin"]}']
show user at get["name"] .               // Ann
tags = user at get["tags"]
add "editor" to tags .                   // the same list, so user changes too
json_write["user.json", user]
show json_text[user] .                   // {"name":"Ann","tags":["admin","editor"]}
```

**JSON to Turtle:** objects become maps (keys stay in the file's order),
arrays become lists, whole numbers become integers (`3`), other numbers
floats (`2.5`, `1e3` → `1000.0`), `true`/`false` booleans, `null` `none`.

**Turtle to JSON:** maps become objects and lists arrays; sets become
arrays; an assembled value becomes an object of its fields
(`Order["pen", 3]` → `{"item": "pen", "qty": 3}`); `none` becomes `null`.
Number map keys become text keys (`1` → `"1"`), since JSON keys are
always text; any other key, a function, or a value that contains itself
is a `type` error.

**`json_get`** takes map keys, assembled-value field names, and list
indexes (from 0), one step each. It never fails on a missing step; it
returns `none`, so you can check instead of using `safe`.

**Errors.** Bad JSON is kind `json` and says where:
`json_read config.json: invalid JSON at line 6, column 1: invalid
character '}' looking for beginning of value`. A missing file is kind
`file`.

```
safe
    cfg = json_read["config.json"]
handle [file, json] e .
    warn "using defaults: ", e .
    cfg = map []
safe [end]
```

JSON typed into Turtle code reads best in single quotes, where `"` needs
no escape and braces before a quote are plain: `'{"a": [1, 2]}'`.
