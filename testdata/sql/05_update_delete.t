// 05_update_delete.t: changing and removing rows: UPDATE and DELETE
// with WHERE, counts of changed rows, upserts, and OR IGNORE / REPLACE.
import sql
import data
import system [scriptFolder]
import lib/verify [check, finish]
import lib/shop [make_shop, drop_shop]

path = "{scriptFolder[]}/work05.db"
db = make_shop[path]

// ---------- UPDATE gives back how many rows changed ----------
n = sql_run[db, "UPDATE books SET price = price + 50 WHERE price < ?", list [700]]
check["two cheap books", n, 2]
n = sql_run[db, "UPDATE books SET stock = stock - 1 WHERE sku = 'nope'"]
check["no match changes nothing", n, 0]

// An UPDATE can use a subquery: stock down by what each book sold.
n = sql_run[db, "UPDATE books SET stock = max(stock - coalesce((SELECT sum(qty) FROM sales s WHERE s.sku = books.sku), 0), 0)"]
check["every book updated", n, 6]
show table[sql_query[db, "SELECT sku, title, price, stock FROM books ORDER BY sku"]] .
check["dune stock", sql_query[db, "SELECT stock FROM books WHERE sku = 'B1'"], list [map ["stock": 0]]]

// ---------- DELETE ----------
n = sql_run[db, "DELETE FROM sales WHERE region = ?", list ["east"]]
check["two east sales gone", n, 2]
n = sql_run[db, "DELETE FROM sales WHERE day < '2026-09-05'"]
check["three early sales gone", n, 3]
check["sales left", sql_query[db, "SELECT count(*) AS n FROM sales"], list [map ["n": 3]]]

// ---------- upsert: add, or change if it's already there ----------
q = "INSERT INTO books (sku, title, author_id, price, stock) VALUES (?, ?, ?, ?, ?) ON CONFLICT (sku) DO UPDATE SET stock = stock + excluded.stock, price = excluded.price"
sql_run[db, q, list ["B1", "Dune", 1, 990, 6]]
sql_run[db, q, list ["B9", "Beloved", 5, 1100, 2]]
rows = sql_query[db, "SELECT sku, title, price, stock FROM books WHERE sku IN ('B1', 'B9') ORDER BY sku"]
show table[rows] .
check["upsert changed B1", rows at get[0], map ["sku": "B1", "title": "Dune", "price": 990, "stock": 6]]
check["upsert added B9", rows at get[1] at get["title"], "Beloved"]

// ---------- OR IGNORE and OR REPLACE ----------
n = sql_run[db, "INSERT OR IGNORE INTO authors (id, name) VALUES (1, 'Someone Else')"]
check["ignored", n, 0]
sql_run[db, "INSERT OR REPLACE INTO authors (id, name, country) VALUES (5, 'Toni Morrison', 'USA')"]
check["replaced", sql_query[db, "SELECT country FROM authors WHERE id = 5"], list [map ["country": "USA"]]]

// ---------- DELETE everything ----------
n = sql_run[db, "DELETE FROM sales"]
check["all sales", n, 3]

drop_shop[db, path]
finish[]
