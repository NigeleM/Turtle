# `sql` — databases

`import sql`

Work with SQLite files and PostgreSQL or MySQL servers: run queries, change tables, and move rows in and out of files.


Work with a SQLite file, or a PostgreSQL or MySQL server, with the same functions. Use `?` in a query where a value goes, and pass the values in a list.

## Functions

`sql_create`, `sql_open`, `sql_query`, `sql_run`, `sql_tables`, `sql_save`, `sql_load`, `sql_update`, `sql_delete`, `sql_upsert`, `sql_close`.

### `sql_create[path]`

Makes a new, empty SQLite database file. It is an error if the file already exists.

**Example**

```turtle
import sql
db = sql_create["new.db"]
sql_close[db]
```

### `sql_open[address]`

Opens a database that already exists: a SQLite file path, or a server address such as `postgres://user:pass@host/name`.

**Example**

```turtle
import sql
db = sql_create["shop.db"]
sql_close[db]
db = sql_open["shop.db"]
show sql_tables[db] .
sql_close[db]
```

**Output**

```text
[]
```

### `sql_query[db, query, values]`

Runs a query that reads rows and gives back a list of maps, one per row.

**Example**

```turtle
import sql
db = sql_create["shop.db"]
sql_run[db, "CREATE TABLE books (title TEXT, price INTEGER)", list []]
sql_run[db, "INSERT INTO books VALUES (?, ?)", list ["Emma", 900]]
rows = sql_query[db, "SELECT title FROM books WHERE price < ?", list [1000]]
show rows .
sql_close[db]
```

**Output**

```text
[ { "title": "Emma" } ]
```

### `sql_run[db, statement, values]`

Runs a statement that changes the database, such as `CREATE`, `INSERT`, `UPDATE` or `DELETE`. Gives back how many rows it changed.

**Example**

```turtle
import sql
db = sql_create["shop.db"]
sql_run[db, "CREATE TABLE books (sku TEXT PRIMARY KEY, price INTEGER)", list []]
sql_run[db, "INSERT INTO books VALUES (?, ?)", list ["B1", 800]]
n = sql_run[db, "UPDATE books SET price = ? WHERE sku = ?", list [900, "B1"]]
show n .
sql_close[db]
```

**Output**

```text
1
```

### `sql_tables[db]`

Gives back the names of the tables in the database.

**Example**

```turtle
import sql
db = sql_create["shop.db"]
sql_run[db, "CREATE TABLE books (title TEXT)", list []]
show sql_tables[db] .
sql_close[db]
```

**Output**

```text
[ "books" ]
```

### `sql_save[db, query, path, values]`

Saves the rows of a query to a file. The file's ending picks the format, as in `table_write`.

**Example**

```turtle
import sql
import data
db = sql_create["shop.db"]
sql_run[db, "CREATE TABLE books (sku TEXT PRIMARY KEY, price INTEGER)", list []]
sql_run[db, "INSERT INTO books VALUES (?, ?)", list ["B1", 800]]
sql_save[db, "SELECT * FROM books", "books.csv", list []]
show table[table_read["books.csv"]] .
sql_close[db]
```

**Output**

```text
sku  price
---  -----
B1     800
```

### `sql_load[db, table, path [, types]]`

Loads the rows of a file into a table.

**Example**

```turtle
import sql
db = sql_create["shop.db"]
sql_run[db, "CREATE TABLE books (sku TEXT PRIMARY KEY, price INTEGER)", list []]
[write] new_books.csv
"sku,price"
"B2,700"
[end]
sql_load[db, "books", "new_books.csv"]
show sql_query[db, "SELECT * FROM books", list []] .
sql_close[db]
```

**Output**

```text
[ { "sku": "B2", "price": 700 } ]
```

### `sql_update[db, table, key, path]`

Updates rows from a file, matching each row by its key column.

**Example**

```turtle
import sql
db = sql_create["shop.db"]
sql_run[db, "CREATE TABLE books (title TEXT, price INTEGER)", list []]
sql_run[db, "INSERT INTO books VALUES (?, ?)", list ["Emma", 900]]
sql_run[db, "UPDATE books SET price = ? WHERE title = ?", list [950, "Emma"]]
show sql_query[db, "SELECT price FROM books", list []] .
sql_close[db]
```

**Output**

```text
[ { "price": 950 } ]
```

### `sql_delete[db, table, key, path]`

Deletes the rows listed in a file, matching them by key.

**Example**

```turtle
import sql
db = sql_create["shop.db"]
sql_run[db, "CREATE TABLE books (title TEXT)", list []]
sql_run[db, "INSERT INTO books VALUES (?)", list ["Emma"]]
sql_run[db, "DELETE FROM books WHERE title = ?", list ["Emma"]]
show sql_query[db, "SELECT title FROM books", list []] .
sql_close[db]
```

**Output**

```text
[]
```

### `sql_upsert[db, table, key, path]`

Adds each row of a file to the table, or replaces the row that has the same key.

**Example**

```turtle
import sql
db = sql_create["shop.db"]
sql_run[db, "CREATE TABLE books (sku TEXT PRIMARY KEY, price INTEGER)", list []]
sql_run[db, "INSERT INTO books VALUES (?, ?)", list ["B1", 800]]
[write] restock.csv
"sku,price"
"B1,950"
[end]
sql_upsert[db, "books", "sku", "restock.csv"]
show sql_query[db, "SELECT * FROM books", list []] .
sql_close[db]
```

**Output**

```text
[ { "sku": "B1", "price": 950 } ]
```

### `sql_close[db]`

Closes the database. Do this when you are done with it.

**Example**

```turtle
import sql
db = sql_create["shop.db"]
sql_close[db]
show "closed" .
```

**Output**

```text
closed
```

For a PostgreSQL or MySQL server, use `sql_open` with the address, and the same functions. Running queries over several connections at once is in the `schedule` library.
