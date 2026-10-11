# The Turtle library

Everything that comes with Turtle beyond its core syntax, one page per
library. A library is turned on with `import` at the top of a file
(`import data`); the built-in types and file statements need none. For
the language itself, see [the reference](../reference.md).

**From the terminal:** `turtle doc` lists every library function;
`turtle doc sql` shows one library, `turtle doc sql_update` one function:
what each argument is, what it gives back, and an example.

## Built in

| Page | What's in it |
|---|---|
| [Built-in types and methods](builtins.md) | lists, sets, maps, text and numbers, and their methods |
| [Files, shell commands and modules](files.md) | `[read]`, `[write]`, `[append]`, `[directory]`, `sys`, `import` |

## Data

| Library | What it's for |
|---|---|
| [`data`](data.md) | process, keep and reduce collections; text tables and table files; statistics |
| [`linear`](linear.md) | matrices: operators, solving, inverses, decompositions |
| [`sort`](sort.md) | sorting by a function, field or position; the classic algorithms |
| [`search`](search.md) | finding items by a rule; the classic search algorithms |

## Numbers and chance

| Library | What it's for |
|---|---|
| [`math`](math.md) | square roots, rounding, powers |
| [`random`](random.md) | random values of any shape; pick, shuffle, sample |

## Text

| Library | What it's for |
|---|---|
| [`strings`](strings.md) | finding, cutting and joining text |
| [`pattern`](pattern.md) | regular expressions: match, find, replace, split |

## Files, system and settings

| Library | What it's for |
|---|---|
| [`system`](system.md) | arguments, environment, files and folders, the program itself |
| [`config`](config.md) | settings in TOML, JSON and .env files |

## Internet

| Library | What it's for |
|---|---|
| [`http`](http.md) | web requests |
| [`json`](json.md) | JSON text and files |
| [`server`](server.md) | a web server from a map of routes |

## Databases

| Library | What it's for |
|---|---|
| [`sql`](sql.md) | SQLite files, PostgreSQL and MySQL servers |

## Time

| Library | What it's for |
|---|---|
| [`time`](time.md) | dates, time zones, date arithmetic, waiting |
| [`schedule`](schedule.md) | many requests, commands or queries at once |

## Security

| Library | What it's for |
|---|---|
| [`crypt`](crypt.md) | hashes, encodings, ids, passwords, encryption |

## Development

| Library | What it's for |
|---|---|
| [`test`](test.md) | `check`, `verify`, `validate`, `turtle test` (guide: [testing](../testing.md)) |
| [`log`](log.md) | log lines with levels, to the screen or a file |

---

Copyright 2017-2026 Nigele McCoy. Licensed under the
[Apache License 2.0](../../LICENSE); see [NOTICE](../../NOTICE).
