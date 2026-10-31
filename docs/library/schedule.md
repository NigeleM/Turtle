# `schedule` — many jobs at once

[Library index](index.md) · `import schedule`

Fetching, running commands and querying databases many at a time.


`import schedule` runs many web requests, shell commands or database
queries at once. Only that work runs side by side, inside Turtle; your
own code still runs one line at a time.

> *Illustrative.* This example needs a network address or a running server, so it is shown for reading and is not run as a test.

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

> *Illustrative.* This example needs a network address or a running server, so it is shown for reading and is not run as a test.

```
pages = fetchall[urls, map ["limit": 10, "skip_errors": true]]
```

| Function | Takes | Gives back |
|---|---|---|
| `fetchall[urls [, settings]]` | a list of web addresses | a list of the pages' text |
| `runall[commands [, settings]]` | a list of shell command lines | a list of maps: `output`, `errors` (what it printed to each), `code` (exit code) |
| `queryall[db, queries [, settings]]` | a PostgreSQL or MySQL database, a list of queries (a query with `?` placeholders as `list [query, list of values]`) | a list of results, each a list of maps as `sql_query` gives |

- `runall` runs each command with `sh -c` (`cmd /c` on Windows, with any
  quotes in the command reaching `cmd` as written: `"echo \"two words\""`)
  in the folder turtle was run in, where file paths resolve too. A nonzero exit code isn't an error: check `code`.
  Trailing newlines are dropped from `output` and `errors`.
- `queryall` opens one extra connection per item running at once and
  closes them when it's done, so the queries don't see a transaction
  open on `db`. SQLite allows one writer at a time, so `queryall` is
  for servers only: on a SQLite file it's an error; use `sql_query`.
- A bad setting (`schedulelimit = 0`, an unknown key in the map), a
  command that can't start, or `queryall` on SQLite is an error of kind
  `schedule`.

---

Copyright 2017-2026 Nigele McCoy. Licensed under the
[Apache License 2.0](../../LICENSE); see [NOTICE](../../NOTICE).

<!-- generated: turtle doc `schedule` (regenerate with the script; do not edit by hand) -->
## Function reference

The entries below are copied from the binary's own documentation (`turtle doc schedule`), so they match the version you run.

```text
import schedule
```

### `Many web requests, shell commands or database queries at once. The`

```text
Many web requests, shell commands or database queries at once. The
work runs side by side; your Turtle code still runs one line at a time.
```

### `At most schedulelimit items run at once, like a semaphore: each one that`

```text
At most schedulelimit items run at once, like a semaphore: each one that
finishes starts the next straight away. Results come back in the list's
order, whatever order they finish in. The first failure stops the call
with that item's error (the running ones finish, the waiting ones never
start); with skipschedule_error = true a failed item becomes none instead.
```

### `Settings ("import schedule" makes them; change them like any variable):`

```text
Settings ("import schedule" makes them; change them like any variable):
    schedulelimit = 5             how many at once; none: all of them
    skipschedule_error = false    true: a failed item becomes none
```

### `A call's own settings, for that call only, go in a map at the end:`

```text
A call's own settings, for that call only, go in a map at the end:
    map ["limit": 10, "skip_errors": true]
```

### `A bad setting, a command that can't start, or queryall on SQLite is an`

```text
A bad setting, a command that can't start, or queryall on SQLite is an
error of kind schedule.
```

### `fetchall[urls, settings]`

```text
fetchall[urls, settings]        (import schedule)
  http_get every address.
  urls       a list of web addresses
  settings   optional: map ["limit": n, "skip_errors": true]
  Gives back: a list of the pages' text, in the same order. A failed
  request (no answer, a 4xx/5xx status) is an http error naming the item.
  Example:
    skipschedule_error = true
    urls = list ["http://127.0.0.1:1/a"]
    pages = fetchall[urls]
    show pages .
```

### `runall[commands, settings]`

```text
runall[commands, settings]        (import schedule)
  Runs every shell command (sh -c; cmd /C on Windows) in the folder
  turtle was run in, capturing what each prints.
  commands   a list of command lines
  settings   optional, as for fetchall
  Gives back: a list of maps, one per command: output (what it printed),
  errors (what it printed as errors), code (its exit code, 0 for
  success). A nonzero code isn't an error; check code. Trailing newlines
  are dropped.
  Example:
    outs = runall[list ["git pull", "make test"]]
    show outs at get[1] at get["code"] .
```

### `queryall[db, queries, settings]`

```text
queryall[db, queries, settings]        (import schedule)
  Runs every query on a PostgreSQL or MySQL database, over extra
  connections that close when it's done (they don't see an open
  transaction on db). Not for SQLite: use sql_query there.
  db         a server database from sql_open
  queries    a list of queries; a query with ? placeholders goes as
             list [query, list of values]
  settings   optional, as for fetchall
  Gives back: a list of results, each a list of maps as sql_query gives.
  A bad query is an sql error naming the item.
  Example:
    r = queryall[db, list ["SELECT count(*) AS n FROM books", list ["SELECT * FROM books WHERE price < ?", list [1000]]]]
```

<!-- end generated -->
