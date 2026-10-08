# `schedule` — many jobs at once

[Library index](index.md) · `import schedule`

Fetching, running commands and querying databases many at a time.


`import schedule` runs many web requests, shell commands or database
queries at once. Only that work runs side by side, inside Turtle; your
own code still runs one line at a time.

```
import schedule

pages = fetchall[urls]                    // http_get each address
pages = urls fetchall                     // the sentence form
outs = runall[list ["git pull", "make"]]  // shell commands
r = queryall[db, list ["SELECT ...", "SELECT ..."]]   // PostgreSQL or MySQL
```

**How many at once.** It works like a semaphore in an async pool: at most
`schedulelimit` items run at once, and each one that finishes starts the
next straight away (it doesn't wait for a whole batch). Results always
come back in the list's order, whatever order they finish in.

**When one fails.** The first failure stops the call with that item's
error, of its usual kind (`http`, `sql`), and says which item it was:
`fetchall item 3: http_get https://...: 404 Not Found`. The items already
running finish; the ones waiting never start. Handle it with `safe` as
usual. With `skipschedule_error = true`, a failed item becomes `none` and
the rest carry on.

**Settings.** `import schedule` makes two variables in the importing
file, changed like any variable (`tablerows` and the log settings work
the same way):

```
schedulelimit = 5             // the default; none = all at once
skipschedule_error = false    // true: a failed item becomes none
```

A setting changed inside a function lasts for that call. If the file
already has a variable of that name before `import schedule`, it's kept.
One call can override both with a map at the end, for that call only:

```
pages = fetchall[urls, map ["limit": 10, "skip_errors": true]]
```

| Function | Takes | Gives back |
|---|---|---|
| `fetchall[urls [, settings]]` | a list of web addresses | a list of the pages' text |
| `runall[commands [, settings]]` | a list of shell command lines | a list of maps: `output`, `errors` (what it printed to each), `code` (exit code) |
| `queryall[db, queries [, settings]]` | a PostgreSQL or MySQL database, a list of queries (a query with `?` placeholders as `list [query, list of values]`) | a list of results, each a list of maps as `sql_query` gives |

- `runall` runs each command with `sh -c` (`cmd /C` on Windows) in the
  folder turtle was run in, where file paths resolve too. A nonzero exit code isn't an error: check `code`.
  Trailing newlines are dropped from `output` and `errors`.
- `queryall` opens one extra connection per item running at once and
  closes them when it's done, so the queries don't see a transaction
  open on `db`. SQLite allows one writer at a time, so `queryall` is
  for servers only: on a SQLite file it's an error; use `sql_query`.
- A bad setting (`schedulelimit = 0`, an unknown key in the map), a
  command that can't start, or `queryall` on SQLite is an error of kind
  `schedule`.
