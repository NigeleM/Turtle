// 12_table_files.t: the data library's tables, without a database:
// table[x] shows rows, table_write saves them (.csv, .tsv, .txt), and
// table_read reads .csv and .tsv files. They take any collection: a list
// of maps, of assembled values, of lists, a plain list, a map.
import data
import system [scriptFolder, erase]
import lib/verify [check, finish]

here = scriptFolder[]

// ---------- a list of assembled values ----------
assemble Order [item, qty, price]
orders = list [Order["pen", 3, 1.5], Order["mug", 2, 8.0], Order["pad", 10, 0.75]]
show table[orders] .
file = "{here}/export_orders.csv"
check["written", table_write[file, orders], 3]
back = table_read[file]
check["read back as maps of text", back at get[1], map ["item": "mug", "qty": "2", "price": "8.0"]]

// The values are text: change them to numbers to compute.
total = 0.0
[loop][row in back]
    q = change row at get["qty"] to integer
    p = change row at get["price"] to float
    total = total + q * p
[loop][end]
check["total from the file", total, 28.0]

// ---------- the same rows as tab-separated and as a text table ----------
tsv = "{here}/export_orders.tsv"
table_write[tsv, orders]
check["tsv round trip", table_read[tsv], back]
txt = "{here}/export_orders.txt"
table_write[txt, orders]
[read] txt to lines [end]
check["txt is what table shows", lines at get[0], "item  qty  price"]

// ---------- .json keeps each value's kind ----------
json = "{here}/export_orders.json"
check["json written", table_write[json, orders], 3]
typed = table_read[json]
check["json reads numbers as numbers", typed at get[1], map ["item": "mug", "qty": 2, "price": 8.0]]
total = 0.0
[loop][row in typed]
    q = row at get["qty"]
    p = row at get["price"]
    total = total + q * p
[loop][end]
check["no change needed", total, 28.0]
// Rows may differ: a missing name is none, nested values stay lists.
mixed = list [map ["sku": "A", "tags": list ["new", "sale"]], map ["sku": "B", "ok": true]]
table_write[json, mixed]
back = table_read[json]
check["nested list", back at get[0] at get["tags"], list ["new", "sale"]]
check["missing name is none", back at get[0] at get["ok"], none]
check["boolean", back at get[1] at get["ok"], true]
erase[json]

// ---------- contains: lists, sets, maps (keys), text ----------
check["list contains", list [1, 2, 3] at contains[2], true]
check["list doesn't contain", list [1, 2, 3] at contains[9], false]
check["set contains", set ["a", "b"] at contains["b"], true]
check["map contains a key", map ["Ann": 30] at contains["Ann"], true]
check["not a value", map ["Ann": 30] at contains[30], false]
check["text contains", "turtle" at contains["tle"], true]

// ---------- other shapes ----------
table_write[file, map ["Ann": 30, "Bo": 25]]
check["a map: key and value", table_read[file], list [map ["key": "Ann", "value": "30"], map ["key": "Bo", "value": "25"]]]
table_write[file, list ["x", "y"]]
check["a plain list: # and value", table_read[file] at get[1], map ["#": "1", "value": "y"]]
table_write[file, list [list [1, "a"], list [2, "b"]]]
check["lists: columns 0, 1", table_read[file] at get[0], map ["0": "1", "1": "a"]]

// ---------- how many rows table shows ----------
many = list []
[loop][i = 0; i < 30; i++]
    add i to many .
[loop][end]
shown is table[many] at split "\n" .
check["20 rows by default, then a line saying how many more", length of shown, 23]
tablerows = 5
shown is table[many] at split "\n" .
check["tablerows changes it", length of shown, 8]
shown is table[many, 2] at split "\n" .
check["or one call", length of shown, 5]
check["files get every row", table_write[file, many], 30]

erase[file]
erase[tsv]
erase[txt]
finish[]
