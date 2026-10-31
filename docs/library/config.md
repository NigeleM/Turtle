# `config` — settings files

[Library index](index.md) · `import config`

Reading and writing settings in TOML, JSON and .env files.


`import config` reads and writes settings files. The file's extension
picks the format: `.toml`, `.json` or `.env`.

> *Illustrative.* This example needs files that the example does not create, so it is shown for reading and is not run as a test.

```
import config

settings = config_read["app.toml"]
port = port of server of settings              // settings at get["server"] at get["port"]
srv = server of settings
port of srv = 9090
config_write["app.toml", settings]
```

with `app.toml`:

```toml
# My app
name = "Shop"
debug = false

[server]
port = 8080
hosts = ["localhost", "0.0.0.0"]

[[users]]
name = "Ann"
```

| Function | Takes | Gives back |
|---|---|---|
| `config_read[path]` | a `.toml`, `.json` or `.env` file | a map |
| `config_write[path, settings]` | a file and a map | writes it; `none` |

- **TOML** (TOML 1.0, read and written by Turtle's own code): keys in the
  file's order; `[section]` and `{ inline }` tables become maps,
  `[[name]]` a list of maps; integers, floats, booleans, text and lists
  keep their kinds; dates become dates (one with an offset keeps it), and
  a time of day alone (`07:32:00`) becomes text. Writing puts the plain
  keys first, then each map as a `[section]`, and a list of maps as
  `[[name]]`s. TOML has no `none`: leave such keys out.
- **JSON** works as `json_read` does; the file must hold an object `{ ... }`.
- **.env** files read as `loadenv` does (`KEY=value` lines, `#` comments,
  quotes), but only into the map: nothing is set for `env[...]`. Every
  value is text. Writing needs single values (quoted when they have
  spaces).
- A file that isn't well written, an unknown extension, or a value the
  format can't hold is an error of kind `config`, naming the line:
  `config_read app.toml line 3: "Shop" isn't a value: text needs quotes`.
  So is TOML whose arrays and inline tables nest more than 10,000 deep,
  or a key with more than 10,000 parts. A missing file is a `file` error.

---

Copyright 2017-2026 Nigele McCoy. Licensed under the
[Apache License 2.0](../../LICENSE); see [NOTICE](../../NOTICE).

<!-- generated: turtle doc `config` (regenerate with the script; do not edit by hand) -->
## Function reference

The entries below are copied from the binary's own documentation (`turtle doc config`), so they match the version you run.

```text
import config
```

### `Settings files, the format chosen by the file's extension: .toml,`

```text
Settings files, the format chosen by the file's extension: .toml,
.json or .env. TOML's [sections] become maps inside the map; numbers,
booleans, dates and lists keep their kinds. A file that isn't well
written is an error of kind config naming its line.
```

### `config_read[path]`

```text
config_read[path]        (import config)
  Reads a settings file.
  path      a .toml, .json or .env file
  Gives back: a map ([sections] and objects as maps inside it; a .env
  file's values are text).
  Example:
    settings = map ["server": map ["port": 8080]]
    config_write["app.toml", settings]
    settings = config_read["app.toml"]
    show settings at get "server" .
```

### `config_write[path, settings]`

```text
config_write[path, settings]        (import config)
  Writes a map as a settings file, by the file's extension. TOML has no
  none, and a .env file holds only single values.
  Example:
    settings = map ["server": map ["port": 8080]]
    config_write["app.toml", settings]
    show "saved" .
```

<!-- end generated -->
