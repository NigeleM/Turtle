// 01_create_insert.t: make a new database, create tables with
// constraints, and add rows with ? values.
import sql
import data
import system [scriptFolder, erase, exists]
import lib/verify [check, finish]

here = scriptFolder[]
path = "{here}/work01.db"
if ] exists[path] [
    erase[path]
if [end]

// ---------- a new database ----------
db = sql_create[path]
check["new database has no tables", sql_tables[db], list []]

sql_run[db, "CREATE TABLE authors (id INTEGER PRIMARY KEY, name TEXT NOT NULL UNIQUE, country TEXT DEFAULT 'unknown')"]
sql_run[db, "CREATE TABLE books (sku TEXT PRIMARY KEY, title TEXT NOT NULL, author_id INTEGER, price INTEGER CHECK (price > 0), added TEXT DEFAULT CURRENT_DATE)"]
check["two tables", sql_tables[db], list ["authors", "books"]]

// ---------- insert with ? values ----------
// sql_run gives back how many rows the statement added.
n = sql_run[db, "INSERT INTO authors (name, country) VALUES (?, ?)", list ["Frank Herbert", "US"]]
check["one row added", n, 1]
n = sql_run[db, "INSERT INTO authors (name, country) VALUES (?, ?), (?, ?), (?, ?)", list ["Gillian Flynn", "US", "Isaac Asimov", "RU", "Jane Austen", "UK"]]
check["three rows added", n, 3]
// A missing column gets its DEFAULT.
sql_run[db, "INSERT INTO authors (name) VALUES ('Nobody Yet')"]

// Text that looks like a number becomes a number in an INTEGER column.
sql_run[db, "INSERT INTO books (sku, title, author_id, price) VALUES (?, ?, ?, ?)", list ["B1", "Dune", 1, "950"]]
sql_run[db, "INSERT INTO books (sku, title, author_id, price) VALUES ('B2', 'Gone Girl', 2, 1225), ('B3', 'Foundation', 3, 800), ('B4', 'Emma', 4, 700)"]

// RETURNING gives the new rows back from sql_query.
added = sql_query[db, "INSERT INTO books (sku, title, author_id, price) VALUES ('B5', 'I, Robot', 3, 650) RETURNING sku, title"]
check["returning", added, list [map ["sku": "B5", "title": "I, Robot"]]]

// The id of the last row added.
last = sql_query[db, "SELECT last_insert_rowid() AS id"]
check["last id", last at get[0] at get["id"], 5]

// ---------- look at them ----------
authors = sql_query[db, "SELECT * FROM authors ORDER BY id"]
show table[authors] .
check["authors", length of authors, 5]
check["default country", authors at get[4] at get["country"], "unknown"]

books = sql_query[db, "SELECT sku, title, price, typeof(price) AS type FROM books ORDER BY sku"]
show table[books] .
check["price stored as a number", books at get[0] at get["type"], "integer"]
check["CURRENT_DATE default", sql_query[db, "SELECT count(*) AS n FROM books WHERE added = date('now')"] at get[0] at get["n"], 5]

sql_close[db]
erase[path]
finish[]
