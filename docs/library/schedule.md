# `schedule` — many jobs at once

`import schedule`

Run many web requests, shell commands or database queries together, each over its own connection.


Run many web requests, shell commands or database queries together. Each runs over its own connection.

## Functions

`fetchall`, `runall`, `queryall`.

### `fetchall[urls, settings]`

Fetches every address in `urls` and gives back a list of the pages' text, in the same order. `settings` is optional; `"skip_errors": true` leaves out pages that fail instead of stopping.

**Example**

```turtle
import schedule
urls = list ["http://127.0.0.1:8765/data", "http://127.0.0.1:8765/health"]
settings = map ["skip_errors": true]
pages = fetchall[urls, settings]
show pages .
```

**Output**

```text
[ "some text", "ok" ]
```

### `runall[commands, settings]`

Runs every shell command and gives back, for each one, its output, its errors and its exit code. A nonzero code is not an error; check it.

**Example**

```turtle
import schedule
outs = runall[list ["echo hi", "exit 3"]]
show outs at get[1] at get["code"] .
show outs at get[0] at get["output"] .
```

**Output**

```text
3
hi
```

### `queryall[db, queries, settings]`

Runs every query on a PostgreSQL or MySQL server, each over its own connection. A query with `?` placeholders is written as a list of the query and its values. Not for SQLite; use `sql_query` there. The example needs a server database, opened with `sql_open` (see the `sql` library).

**Example**

```turtle
import sql
import schedule
db = sql_open["postgres://user:pass@localhost/shop"]
r = queryall[db, list ["SELECT count(*) AS n FROM books", list ["SELECT * FROM books WHERE price < ?", list [1000]]]]
show r .
sql_close[db]
```

**Output**

```text
[ [ { "n": 2 } ], [ { "title": "Dune", "price": 900 } ] ]
```
