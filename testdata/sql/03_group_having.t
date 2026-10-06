// 03_group_having.t: totals per group: GROUP BY, HAVING, and the
// aggregates count, sum, avg, min, max, group_concat, with DISTINCT
// and FILTER.
import sql
import data
import system [scriptFolder]
import lib/verify [check, finish]
import lib/shop [make_shop, drop_shop]

path = "{scriptFolder[]}/work03.db"
db = make_shop[path]

// ---------- one row per region ----------
rows = sql_query[db, "SELECT region, count(*) AS sales, sum(qty) AS copies FROM sales GROUP BY region ORDER BY copies DESC"]
show table[rows] .
check["regions", length of rows, 3]
check["north sold most", rows at get[0], map ["region": "north", "sales": 4, "copies": 10]]

// ---------- HAVING keeps only some groups ----------
rows = sql_query[db, "SELECT sku, sum(qty) AS copies FROM sales GROUP BY sku HAVING sum(qty) >= 5 ORDER BY sku"]
show table[rows] .
check["having", rows, list [map ["sku": "B1", "copies": 8], map ["sku": "B3", "copies": 6]]]

// ---------- every aggregate at once, per author ----------
q = "SELECT a.name, count(b.sku) AS books, min(b.price) AS cheapest, max(b.price) AS dearest, avg(b.price) AS average, group_concat(b.title, ' / ') AS titles FROM authors a LEFT JOIN books b ON b.author_id = a.id GROUP BY a.id ORDER BY a.id"
rows = sql_query[db, q]
show table[rows] .
check["austen", rows at get[3] at get["books"], 2]
check["austen average", rows at get[3] at get["average"], 650.0]
check["no books: count is 0, max is none", list [rows at get[4] at get["books"], rows at get[4] at get["dearest"]], list [0, none]]

// ---------- DISTINCT and FILTER inside an aggregate ----------
rows = sql_query[db, "SELECT count(DISTINCT sku) AS titles, sum(qty) FILTER (WHERE region = 'north') AS north, sum(qty) FILTER (WHERE region != 'north') AS elsewhere FROM sales"]
check["distinct and filter", rows at get[0], map ["titles": 5, "north": 10, "elsewhere": 9]]

// ---------- money: revenue per book, best first ----------
rows = sql_query[db, "SELECT b.title, sum(s.qty * b.price) AS revenue FROM sales s JOIN books b ON b.sku = s.sku GROUP BY b.sku ORDER BY revenue DESC LIMIT 3"]
show table[rows] .
check["top seller", rows at get[0], map ["title": "Dune", "revenue": 7600]]

// ---------- a group per month ----------
rows = sql_query[db, "SELECT substr(day, 1, 7) AS month, sum(qty) AS copies FROM sales GROUP BY month ORDER BY month"]
check["per month", rows, list [map ["month": "2026-09", "copies": 18], map ["month": "2026-10", "copies": 1]]]

drop_shop[db, path]
finish[]
