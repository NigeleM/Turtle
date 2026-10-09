# `sql` — databases

[Library index](index.md) · `import sql`

SQLite files, PostgreSQL and MySQL servers: queries, changes, and moving files in and out.


`import sql` works with SQLite files and with PostgreSQL and MySQL (or
MariaDB) servers, through the same functions; the address given to
`sql_open` picks which. Every driver is written from scratch for Turtle
(no third-party code).

> **About your data.** Turtle reads and writes SQLite files with its own
> engine, written for Turtle. The files are ordinary SQLite files: the
> official `sqlite3` tool and any SQLite program open them, and Turtle's
> tests check the files it writes with `sqlite3`'s `PRAGMA
> integrity_check`, including after simulated crashes. Still, as with any
> young software, keep backups of data that matters (copying the `.db`
> file while no program has it open is a full backup), and if a file
> ever seems wrong, `sqlite3 shop.db "PRAGMA integrity_check"` tells you
> whether it's sound. PostgreSQL and MySQL store data in their own
> servers; Turtle only talks to them.

For SQLite, that's the file format, B-trees, the journal that makes
changes crash-safe, file locking, and the SQL engine. The files are
ordinary SQLite files: the `sqlite3` tool, DB Browser, Python and every
other SQLite program can open them, and Turtle can open theirs. For the
servers, it's their network protocols, logins and TLS; the server runs
the SQL (see [Servers](#servers-postgresql-and-mysql)).

| Function | Args | Returns |
|---|---|---|
| `sql_open[address]` | a SQLite file path (or `"sqlite:path"`), or a server address `postgres://...` / `mysql://...` | a database value |
| `sql_create[path]` | a path where no file is yet | a new, empty SQLite database |
| `sql_query[db, query [, values]]` | database, a `SELECT` (or a change with `RETURNING`), and a list of values for its `?` placeholders | a list of maps, one per row, column name → value |
| `sql_run[db, statement [, values]]` | database, a statement that changes something, and its `?` values | how many rows it inserted, updated or deleted |
| `sql_tables[db]` | database | a list of its table names |
| `sql_save`, `sql_load`, `sql_update`, `sql_delete`, `sql_upsert` | see [Files](#files-csv-in-and-out) | how many records |
| `sql_close[db]` | database | `none`; closing twice is fine; an unfinished transaction is rolled back |

```
import sql
import data

db = sql_create["shop.db"]
sql_run[db, "CREATE TABLE books (id INTEGER PRIMARY KEY, title TEXT NOT NULL, price INTEGER)"]
sql_run[db, "INSERT INTO books (title, price) VALUES (?, ?)", list ["Dune", 950]]
n = sql_run[db, "UPDATE books SET price = price + 50 WHERE price < ?", list [1000]]

rows = sql_query[db, "SELECT title, price FROM books ORDER BY price"]
show table[rows] .
sql_close[db]
```

**Values.** `NULL` is `none`; integers, reals and text come back as
integers, floats and text; a BLOB comes back as text. `?` values can be
integers, floats, text, booleans (1/0), `none` (`NULL`) and dates (as
`2026-10-03 14:05:00` text). Always pass values as `?` placeholders rather
than building the statement with `+` or `{ }`, so a value can never change
the statement. Text going into an `INTEGER` or `REAL` column becomes a
number when it looks like one (`"950"` → `950`), as in SQLite; that's
what makes CSV imports work.

**Several statements at once.** `sql_run` runs statements separated by
`;` in order, when no `?` values are given:
`sql_run[db, "CREATE TABLE a (x); CREATE TABLE b (y)"]`.

**Looking at results.** `show rows .` prints them on one line; with
`import data`, `show table[rows] .` prints one row per line (see
[Tables](data.md#tables)). When a column name appears twice (`SELECT *` over a
join), the second one is named `id:1`, the third `id:2`.

## Changing data

Everything SQLite's own SQL can change:

- `INSERT INTO t (cols) VALUES (...), (...)`, `INSERT ... SELECT ...`,
  `INSERT ... DEFAULT VALUES`, `REPLACE INTO`, and `INSERT OR IGNORE` /
  `OR REPLACE` / `OR FAIL` / `OR ABORT` / `OR ROLLBACK`.
- Upsert: `INSERT ... ON CONFLICT (col) DO UPDATE SET qty = qty + excluded.qty`
  (with an optional `WHERE`), or `ON CONFLICT DO NOTHING`.
- `UPDATE t SET a = ..., b = ... WHERE ...` (`OR IGNORE` / `OR REPLACE` too)
  and `DELETE FROM t WHERE ...`.
- `RETURNING` after any of them gives the changed rows back; run such a
  statement with `sql_query`:
  `rows = sql_query[db, "INSERT INTO t (v) VALUES ('a') RETURNING id"]`.
- `WITH ...` before `INSERT`, `UPDATE` or `DELETE`.
- `UPDATE ... FROM` to change rows using another table:
  `UPDATE stock SET qty = qty + d.n FROM delivery AS d WHERE d.sku = stock.sku`.

**Constraints** are enforced, and a statement that breaks one changes
nothing at all (an error of kind `sql`): `NOT NULL`, `UNIQUE`,
`PRIMARY KEY`, `CHECK`, `DEFAULT` (including `CURRENT_TIMESTAMP`),
`COLLATE NOCASE`, `INTEGER PRIMARY KEY` (the row's id, filled in for you),
`AUTOINCREMENT`, and `STRICT` tables. Foreign keys (`REFERENCES`, with
`ON DELETE CASCADE / SET NULL / SET DEFAULT / RESTRICT`) are enforced
after `PRAGMA foreign_keys = ON`; as in SQLite, they're off until then.

```
safe
    sql_run[db, "INSERT INTO books (title) VALUES (NULL)"]
handle [sql] e .
    show message of e .        // sql_run: NOT NULL constraint failed: books.title
safe [end]
```

## Transactions

Each statement is saved on its own, and is all-or-nothing. To save
several together, or to undo them:

```
sql_run[db, "BEGIN"]
sql_run[db, "UPDATE accounts SET balance = balance - 30 WHERE id = 1"]
sql_run[db, "UPDATE accounts SET balance = balance + 30 WHERE id = 2"]
sql_run[db, "COMMIT"]          // or ROLLBACK to undo both
```

A statement that fails inside `BEGIN ... COMMIT` is undone by itself and
the transaction stays open: the program decides whether to `COMMIT` what
worked or `ROLLBACK` everything. A program that ends (or crashes) before
`COMMIT` leaves the file as it was.

Saving waits until the disk has the data, so many separate statements
are slow (a few milliseconds each); the same statements inside one
`BEGIN ... COMMIT` are written together, thousands per second.

**Crash safety.** Before changing the file, Turtle writes the original
pages to a journal (`shop.db-journal`, SQLite's own format) and makes it
durable. If the program or the computer stops halfway, the next open
(by Turtle or any SQLite program) puts the original pages back. Turtle
also repairs files that another SQLite program left half-written.

**Other programs.** Turtle locks the file the way SQLite does, so it can
be used at the same time as `sqlite3` or another program: readers share,
one writer at a time. A program that has to wait more than 5 seconds gets
an error of kind `sql` ("database is locked").

## Changing the schema

- `CREATE TABLE` (with `IF NOT EXISTS`), `CREATE TABLE ... AS SELECT`.
- `CREATE [UNIQUE] INDEX` (on columns, with `COLLATE` and `DESC`, and
  partial indexes with `WHERE`).
- `CREATE VIEW`.
- `ALTER TABLE t RENAME TO new` (views that use it follow), `ADD COLUMN`,
  `RENAME COLUMN a TO b`, `DROP COLUMN`.
- `DROP TABLE`, `DROP INDEX`, `DROP VIEW` (with `IF EXISTS`).

Indexes are kept up to date by every change, and used to find rows fast:
`WHERE id = ?`, `WHERE sku = ?` on an indexed column, and joins on an id
or an indexed column look rows up instead of reading the whole table.

## Queries

`SELECT` with `DISTINCT`, expressions and `AS` names, `*` and `t.*`;
`FROM` tables, views, subqueries and `WITH` tables; `JOIN`, `LEFT JOIN`,
`RIGHT JOIN`, `FULL JOIN`, `CROSS JOIN`, `NATURAL JOIN`, with `ON` or
`USING`; `WHERE`; `GROUP BY` and `HAVING`; `UNION`, `UNION ALL`,
`INTERSECT`, `EXCEPT`; `ORDER BY` (columns, expressions, result names or
positions, `ASC`/`DESC`, `NULLS FIRST`/`NULLS LAST`, `COLLATE`);
`LIMIT`/`OFFSET`; `WITH` and `WITH RECURSIVE`; `VALUES (...)`.

Subqueries work as values `(SELECT max(price) FROM books)`, in
`IN (SELECT ...)` and `EXISTS (SELECT ...)`, as tables in `FROM`, and may
use columns of the query around them.

Operators: `= == != <> < > <= >= AND OR NOT + - * / % || & | << >> ~`,
`IS [NOT]`, `IS [NOT] DISTINCT FROM`, `IS [NOT] NULL`, `[NOT] IN`,
`[NOT] BETWEEN`, `[NOT] LIKE ... [ESCAPE]`, `[NOT] GLOB`, `CASE`, `CAST`,
and `COLLATE` (`BINARY`, `NOCASE`, `RTRIM`).

Aggregates (with `DISTINCT`, and `FILTER (WHERE ...)`): `count sum total
avg min max group_concat string_agg`.

Functions:

| Kind | Functions |
|---|---|
| text | `length lower upper substr trim ltrim rtrim replace instr printf format quote char unicode hex unhex concat concat_ws octet_length like glob` |
| numbers | `abs round sign min max random ceil floor trunc sqrt pow exp ln log log2 log10 mod pi sin cos tan asin acos atan atan2 sinh cosh tanh degrees radians` |
| dates | `date time datetime julianday unixepoch strftime`, with modifiers such as `'+1 month'`, `'start of month'`, `'weekday 0'`, `'unixepoch'`, `'localtime'`; `CURRENT_DATE`, `CURRENT_TIME`, `CURRENT_TIMESTAMP` |
| values | `coalesce ifnull nullif iif typeof zeroblob randomblob likely unlikely` |
| the database | `last_insert_rowid changes total_changes sqlite_version` |

Answers match real SQLite, including its type rules (a `TEXT` column
holding `'5'` equals `5`) and its date arithmetic (`'2026-01-31'` plus one
month is `2026-03-03`).

## Files: CSV in and out

These move rows straight between a file and the database, in the same
files as the data library's [table files](data.md#table-files): `sql_save`
writes `.csv`, `.tsv`, `.txt` or `.json`; the others read `.csv`, `.tsv`
or `.json`. From `.json`, numbers and booleans go in as they are, and a
list or object inside a row is stored as its JSON text. Each returns how many records it wrote or changed. A file is
**all or nothing**: if one line is refused (a duplicate key, a `CHECK`,
a missing `NOT NULL` value), no line of that file is kept, even inside
`BEGIN ... COMMIT`.

| Function | Does |
|---|---|
| `sql_save[db, query, path [, values]]` | runs the query and writes its rows to the file (the header too, even with no rows) |
| `sql_load[db, table, path [, types]]` | adds a record per line; the header names the columns. If the table isn't there, makes it first (see below) |
| `sql_update[db, table, key, path]` | for each line, changes the record whose `key` column matches, setting the file's other columns |
| `sql_delete[db, table, key, path]` | removes the records whose `key` matches a line (other columns are ignored) |
| `sql_upsert[db, table, key, path]` | adds the lines whose key is new and changes the ones already there (`key` must be `UNIQUE` or the `PRIMARY KEY`) |

```
import sql

db = sql_open["shop.db"]
sql_save[db, "SELECT sku, title, price FROM books ORDER BY sku", "books.csv"]
sql_save[db, "SELECT * FROM books WHERE price < ?", "cheap.tsv", list [1000]]

sql_load[db, "books", "new_books.csv"]
n = sql_update[db, "books", "sku", "price_changes.csv"]    // sku,price
sql_delete[db, "books", "sku", "discontinued.csv"]         // sku
sql_upsert[db, "books", "sku", "restock.csv"]
```

```
sku,title,price
B1,Dune,950
B5,"I, Robot",650
```

- **A table that's already there decides the types.** `sql_load`,
  `sql_update` and `sql_upsert` convert each cell to its column's
  declared type (`INTEGER`, `TEXT`, `REAL`, `DATE`, ... as in the
  [column types](data.md#column-types) table), so `950` goes into an `INTEGER`
  column as `950`, a `7` into a `TEXT` one as `"7"`, and `yes` into a
  `BOOLEAN` one as `true`. A cell that can't be its column's type stops the file before
  anything is written, with the row and column (kind `number`, `date` or
  `type`). A types map, if given, wins over the table's types; a column
  declared without a type (SQLite allows it) takes what the file has.
  This works the same on SQLite, PostgreSQL and MySQL.
- An empty cell is `NULL`, so it sets the column to `NULL` in
  `sql_update`.
- **Column types.** `sql_load` takes the same map of
  [column types](data.md#column-types) as `table_read`, in place or in a
  variable. The file's values are converted first, so a bad one stops
  the load before anything is written.
- **A missing table is made** from the file's header, in the file's
  column order. With a map, each column gets its type and
  `"primary_key"` becomes the table's `PRIMARY KEY`. A column the map
  doesn't name gets its type from the file: integer if every value is
  a whole number, a float if every one is a number, else text. If the
  file is then refused, the new table is dropped again. When the table
  is already there, its own column types and key stay (see above).

```
types = map ["code": "text", "missions": "integer", "joined": "date", "primary_key": "code"]
sql_load[db, "agents", "agents.csv", types]
// CREATE TABLE "agents" ("code" TEXT PRIMARY KEY, "name" TEXT, "missions" INTEGER, ..., "joined" DATE)
```

  A Turtle type word becomes each database's own type; a SQL type is
  used exactly as written:

  | Word | SQLite | PostgreSQL | MySQL |
  |---|---|---|---|
  | `string`, `text` | `TEXT` | `TEXT` | `TEXT` (`VARCHAR(255)` for the key, which MySQL needs) |
  | `integer` | `INTEGER` | `BIGINT` | `BIGINT` |
  | `float` | `REAL` | `DOUBLE PRECISION` | `DOUBLE` |
  | `boolean` | `BOOLEAN` | `BOOLEAN` | `BOOLEAN` (MySQL gives back `1` / `0`) |
  | `date` | `DATE` (stored as text, `1953-04-13`) | `DATE` | `DATE` |
- A line whose key isn't in the table changes nothing in `sql_update`
  and `sql_delete`; compare the count with the file's lines, or look the
  keys up (see `testdata/sql/11_csv_import.trt`).
- Each file runs as one statement, saved in one write, so a file of
  thousands of lines is fast.
- Table and column names go into the SQL quoted, so any name works and
  none can change the statement.

## More of SQLite

- **Window functions**: `row_number rank dense_rank percent_rank
  cume_dist ntile lag lead first_value last_value nth_value`, and every
  aggregate with `OVER (PARTITION BY ... ORDER BY ...)`, frames (`ROWS`,
  `RANGE`, `GROUPS`, `EXCLUDE`) and named `WINDOW`s.
- **JSON**: `json json_extract json_object json_array json_set
  json_insert json_replace json_remove json_patch json_type json_valid
  json_group_array json_group_object` and more, the `->` and `->>`
  operators, and `json_each` / `json_tree` as tables. Also
  `generate_series(1, 10)` as a table.
- **Triggers**: `CREATE TRIGGER` (`BEFORE`, `AFTER`, `INSTEAD OF` on
  views, `FOR EACH ROW`, `WHEN`, `UPDATE OF`), `RAISE(...)`.
- **Savepoints**: `SAVEPOINT name`, `RELEASE name`, `ROLLBACK TO name`.
- **Generated columns** (`AS (price * qty) STORED` or `VIRTUAL`) and
  indexes on expressions (`CREATE INDEX ... ON users (lower(email))`).
- **PRAGMA**: `table_info`, `index_list`, `foreign_key_list`,
  `foreign_keys`, `integrity_check`, `journal_mode` (`DELETE` or `WAL`),
  `user_version` and others; also as tables
  (`SELECT * FROM pragma_table_info('books')`).
- **VACUUM** (and `VACUUM INTO 'copy.db'`), and **ATTACH** another
  database file to read from it (`ATTACH 'old.db' AS old`).

## Not yet

Changing `WITHOUT ROWID` tables (they're read fine), virtual tables
(`fts5` and others), and changing an attached database (it's read-only).
Databases that use auto-vacuum or store text as UTF-16 can be read but
not changed.

## Servers: PostgreSQL and MySQL

Give `sql_open` a server address instead of a file. Every other
function is the same, including the CSV ones:

```
import sql
import data

db = sql_open["postgres://ann:secret@localhost:5432/shop"]
// or: db = sql_open["mysql://ann:secret@localhost:3306/shop"]
rows = sql_query[db, "SELECT title, price FROM books WHERE price < ?", list [1000]]
show table[rows] .
sql_upsert[db, "books", "sku", "restock.csv"]
sql_close[db]
```

- **The server runs the SQL**, so it's the server's own dialect:
  PostgreSQL's or MySQL's functions and types, not SQLite's. `?` marks
  values for all three (Turtle turns them into PostgreSQL's `$1, $2`),
  and values go to the server separately from the statement.
- **Values**: integers, floats, text and `NULL` as for SQLite; a
  server's `BOOLEAN` comes back as `true`/`false` (MySQL stores booleans
  as `TINYINT`, so `1`/`0`); `DATE`, `TIMESTAMP` and `DATETIME` come
  back as dates; `NUMERIC`/`DECIMAL` as integers when whole, floats
  otherwise; `JSON` as text.
- **Transactions**: `BEGIN`, `COMMIT`, `ROLLBACK` work as with SQLite.
  The CSV functions are still all or nothing, inside a transaction or not.
- **Logins**: PostgreSQL's `scram-sha-256`, `md5` and `password`;
  MySQL's `caching_sha2_password`, `sha256_password` and
  `mysql_native_password`. A wrong password is an error of kind `sql`.
- **TLS** (encryption) is used when the server offers it. Options go
  after `?` in the address: PostgreSQL `sslmode=disable`, `prefer` (the
  default), `require` or `verify-full`, and `connect_timeout=10`; MySQL
  `tls=false`, `preferred` (the default), `skip-verify` or `true`, and
  `timeout=10`.
- `sql_create` is for SQLite files only; make a server's database with
  `CREATE DATABASE` (or its own tools), then `sql_open` it.
- `sql_tables` lists the tables of the database (PostgreSQL: of the
  current schema).
- `testdata/sql/13_servers.trt` runs the same program against both
  servers (set `TURTLE_PG_URL` and `TURTLE_MYSQL_URL`).

**Errors** are kind `sql` (`sql_query: no such table: shelves`); a missing
file is kind `file`.
