// 11_csv_import.t: change a database from CSV files: create records
// (sql_load), change them (sql_update), remove them (sql_delete), and
// add-or-change (sql_upsert). Each file is all or nothing: if one line
// is refused, no line of that file is kept.
import sql
import data
import system [scriptFolder, erase, exists]
import lib/verify [check, finish]

here = scriptFolder[]
path = "{here}/work11.db"
if ] exists[path] [
    erase[path]
if [end]
db = sql_create[path]
sql_run[db, "CREATE TABLE authors (id INTEGER PRIMARY KEY, name TEXT NOT NULL UNIQUE, country TEXT)"]
sql_run[db, "CREATE TABLE books (sku TEXT PRIMARY KEY, title TEXT NOT NULL, author_id INTEGER, price INTEGER NOT NULL CHECK (price > 0), stock INTEGER NOT NULL DEFAULT 0, published TEXT)"]

def count_of[db, table]
    rows = sql_query[db, "SELECT count(*) AS n FROM {table}"]
    return rows at get[0] at get["n"]
def [end]

def price_of[db, sku]
    rows = sql_query[db, "SELECT price FROM books WHERE sku = ?", list [sku]]
    if ] length of rows == 0 [
        return none
    if [end]
    return rows at get[0] at get["price"]
def [end]

// ---------- create: one record per line; the header names the columns ----------
check["authors added", sql_load[db, "authors", "{here}/data/authors.csv"], 5]
check["books added", sql_load[db, "books", "{here}/data/books.csv"], 6]
show table[sql_query[db, "SELECT * FROM books ORDER BY sku"]] .
// CSV values are text; the INTEGER columns stored them as numbers.
check["price is a number", sql_query[db, "SELECT typeof(price) AS t FROM books WHERE sku = 'B1'"], list [map ["t": "integer"]]]

// ---------- update: the key column picks the record ----------
// price_changes.csv has sku and price; B9 isn't in the table.
check["two prices changed", sql_update[db, "books", "sku", "{here}/data/price_changes.csv"], 2]
check["B1 new price", price_of[db, "B1"], 999]

// Which keys weren't found: read the file, and look each one up.
missing = list []
[loop][row in table_read["{here}/data/price_changes.csv"]]
    sku = row at get["sku"]
    if ] price_of[db, sku] == none [
        add sku to missing .
    if [end]
[loop][end]
check["not found", missing, list ["B9"]]

// ---------- delete: every record whose key is in the file ----------
check["deleted", sql_delete[db, "books", "sku", "{here}/data/discontinued.csv"], 2]
check["books left", count_of[db, "books"], 4]

// ---------- upsert: add new books, change the ones already there ----------
// restock.csv: B1 is there (new price and stock), B7 and B8 are new.
// B8's stock cell is empty: none, which NOT NULL refuses, so the upsert
// gives it the default first.
sql_run[db, "CREATE TABLE restock AS SELECT * FROM books WHERE 0"]
sql_load[db, "restock", "{here}/data/restock.csv"]
sql_run[db, "UPDATE restock SET stock = 0 WHERE stock IS NULL"]
sql_save[db, "SELECT * FROM restock", "{here}/export_restock.csv"]
check["upserted", sql_upsert[db, "books", "sku", "{here}/export_restock.csv"], 3]
final = sql_query[db, "SELECT sku, title, price, stock FROM books ORDER BY sku"]
show table[final] .
check["B1 changed", final at get[0], map ["sku": "B1", "title": "Dune", "price": 1050, "stock": 10]]
check["two new books", count_of[db, "books"], 6]
check["empty cell became 0", final at get[5], map ["sku": "B8", "title": "The Caves of Steel", "price": 700, "stock": 0]]

// ---------- a bad file changes nothing ----------
// B4's new price is 0, which the CHECK refuses: B3's change is undone too.
safe
    sql_update[db, "books", "sku", "{here}/data/bad_prices.csv"]
handle [sql] e .
    show "refused: ", message of e .
safe [end]
check["B3 kept its price", price_of[db, "B3"], 850]
check["B4 kept its price", price_of[db, "B4"], 700]

// ---------- round trip: save, load into a copy, compare ----------
sql_save[db, "SELECT * FROM books", "{here}/export_final.csv"]
sql_run[db, "CREATE TABLE books_copy AS SELECT * FROM books WHERE 0"]
check["copied", sql_load[db, "books_copy", "{here}/export_final.csv"], 6]
diff = sql_query[db, "SELECT count(*) AS n FROM (SELECT * FROM books EXCEPT SELECT * FROM books_copy)"]
check["the copy matches", diff, list [map ["n": 0]]]

// ---------- .json files: values keep their kind ----------
sql_save[db, "SELECT sku, title, price, stock FROM books WHERE sku = 'B1'", "{here}/export_b1.json"]
saved = table_read["{here}/export_b1.json"]
check["json saved a number", saved at get[0] at get["price"], 1050]
table_write["{here}/export_new.json", list [map ["sku": "B9", "title": "Kindred", "author_id": 5, "price": 900, "stock": 2]]]
check["json loaded", sql_load[db, "books", "{here}/export_new.json"], 1]
check["B9 price", price_of[db, "B9"], 900]
sql_run[db, "DELETE FROM books WHERE sku = 'B9'"]
erase["{here}/export_b1.json"]
erase["{here}/export_new.json"]

erase["{here}/export_restock.csv"]
erase["{here}/export_final.csv"]
sql_close[db]
erase[path]
finish[]
