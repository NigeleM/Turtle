# SQL examples

One Turtle program per topic. Each makes its own database next to it,
prints its tables, checks its own results, removes the files it made,
and ends with `failures: 0` (exit code 1 if a check failed). Run one with

    turtle testdata/sql/02_joins.turtle

or all of them with `go test ./evaluator -run TestSQLExamples`.

| Program | Shows |
|---|---|
| `01_create_insert.turtle` | `sql_create`, `CREATE TABLE` with constraints and defaults, `INSERT` with `?` values, `RETURNING`, `last_insert_rowid()` |
| `02_joins.turtle` | `JOIN`, `LEFT JOIN`, `RIGHT JOIN`, `USING`, three tables at once, "never sold" with `IS NULL` |
| `03_group_having.turtle` | `GROUP BY`, `HAVING`, `count sum avg min max group_concat`, `DISTINCT` and `FILTER` in aggregates |
| `04_subqueries.turtle` | subqueries as values, `IN`, `NOT IN`, `EXISTS`, per-row subqueries, subqueries as tables, `UNION INTERSECT EXCEPT`, `WITH`, `WITH RECURSIVE`, `VALUES` |
| `05_update_delete.turtle` | `UPDATE` and `DELETE` with counts, `UPDATE` with a subquery, upsert (`ON CONFLICT DO UPDATE`), `OR IGNORE`, `OR REPLACE` |
| `06_constraints.turtle` | `NOT NULL`, `UNIQUE`, `PRIMARY KEY`, `CHECK`, `STRICT` refusing bad rows, caught with `safe / handle [sql]` |
| `07_transactions.turtle` | `BEGIN` / `COMMIT` / `ROLLBACK`, all-or-nothing changes, many rows in one transaction |
| `08_schema.turtle` | `CREATE INDEX`, unique `NOCASE` index, `CREATE VIEW`, `ALTER TABLE` (add, rename, drop a column; rename a table), `CREATE TABLE ... AS SELECT`, `DROP` |
| `09_functions.turtle` | text, number and date functions, `printf`, `CASE`, `CAST`, `COALESCE`, `IIF` |
| `10_csv_export.turtle` | query results to `.csv`, `.tsv` and `.txt` files with `sql_save`, read back with `table_read` |
| `11_csv_import.turtle` | `sql_load`, `sql_update`, `sql_delete`, `sql_upsert`: create, change and remove records from CSV files, each file all or nothing; round trip |
| `12_table_files.turtle` | the data library without a database: `table`, `tablerows`, `table_write` and `table_read` on assembled values, maps, lists |
| `13_servers.turtle` | the same functions on PostgreSQL and MySQL servers: CSV in and out, joins, dates, transactions, errors. Runs when `TURTLE_PG_URL` / `TURTLE_MYSQL_URL` are set; otherwise says so |
| `14_typed_load.turtle` | column types: `table_read` and `sql_load` with a map of column to type ("007" stays text), `sql_load` making the table and its primary key, SQL type words, the errors |

`lib/` has the helpers they share: `verify.turtle` (the checks) and `shop.turtle`
(the bookshop database, loaded from `data/` with `sql_load`). `data/` has
the CSV files (`agents.csv` is the one with `007`).
