# `config` — settings files

[Library index](index.md) · `import config`

Reading and writing settings in TOML, JSON and .env files.


`import config` reads and writes settings files. The file's extension
picks the format: `.toml`, `.json` or `.env`.

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
  A missing file is a `file` error.

---

Copyright 2017-2026 Nigele McCoy. Licensed under the
[Apache License 2.0](../../LICENSE); see [NOTICE](../../NOTICE).
