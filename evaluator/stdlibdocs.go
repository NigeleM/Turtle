package evaluator

// Documentation for every builtin library function, shown by
// "turtle doc". Each module's text starts with what the library is for;
// then each "### " line is one function, written the way it's called,
// followed by its description: what each argument is, what it gives
// back, what it does, and an example. Keep it in step with the code:
// TestEveryBuiltinIsDocumented fails if a function has no entry.

var moduleDocs = map[string]string{
	"math": `Number methods. Called on a number with "at":  r is 16 at sqrt .

### number at sqrt
  The square root.
  Gives back: a float (16 at sqrt is 4.0). A negative number is a math error.
  Example:
    r is 16 at sqrt .

### number at abs
  The number without its sign: -7 becomes 7.
  Gives back: the same kind of number (integer or float).
  Example:
    a is -7 at abs .

### number at round
  The nearest whole number; a half rounds away from zero (4.5 becomes 5).
  Example:
    r is 4.5 at round .

### number at floor
  The whole number at or below it (4.7 becomes 4, -4.2 becomes -5).
  Example:
    f is 4.7 at floor .

### number at ceil
  The whole number at or above it (4.1 becomes 5).
  Example:
    c is 4.1 at ceil .

### number at pow exponent
  The number raised to a power.
  exponent   the power, a number
  Gives back: an integer when both are whole and the exponent isn't
  negative (2 at pow 10 is 1024), else a float.
  Example:
    p is 2 at pow 10 .

### number at random
  A random whole number from 0 up to (not including) the number.
  The number must be a positive integer.
  Example:
    dice is 6 at random .
`,

	"time": `The clock, dates, date arithmetic, and waiting.
Units are "seconds", "minutes", "hours", "days", "weeks", "months",
"years" (or the singular). A date shows as 2026-10-03 14:05:00 and has
parts read with "of": year, month, day, hour, minute, second, weekday.

### now[]
  Milliseconds since 1 January 1970: for measuring how long something takes.
  Gives back: an integer.
  Example:
    start = now[]
    show "took ", now[] - start, " ms" .

### sleep[amount, unit]
  Pauses the program.
  amount   how long, a number (0.25 is fine)
  unit     optional: "seconds" (the default) or "ms"
  Example:
    sleep[250, "ms"]

### today[]
  The date and time now, in local time.
  Gives back: a date.
  Example:
    show weekday of today[] .

### today_utc[]
  The date and time now, in UTC.
  Gives back: a date.

### make_date[year, month, day, hour, minute, second]
  A date from its parts. Hour, minute and second are optional (all three,
  or none). A date that doesn't exist, like February 30, is a date error.
  Gives back: a date, in local time.
  Example:
    d = make_date[2026, 12, 25]

### to_date[text]
  Reads a date from text: "2026-10-03", "2026-10-03 14:05",
  "2026-10-03 14:05:00", or ISO 8601 ("2026-10-03T14:05:00Z").
  Gives back: a date. Other text is a date error.
  Example:
    d = to_date["2026-10-03"]

### add_time[date, amount, unit]
  A date moved forward (or back, with a negative amount). Months and
  years stay in their month: January 31 plus 1 month is February 28.
  date     the date to start from
  amount   a whole number
  unit     "days", "months", ...
  Gives back: a new date (the original doesn't change).
  Example:
    due = add_time[today[], 30, "days"]

### time_between[a, b, unit]
  How many whole units from date a to date b (negative if b is earlier).
  Gives back: an integer.
  Example:
    left = time_between[today[], due, "days"]

### format_date[date, pattern]
  Writes a date with a pattern: YYYY year, MM or M month, DD or D day,
  hh hour (00-23), mm minute, ss second, Month (March), Mon (Mar),
  Weekday (Thursday), Wkd (Thu). Anything else is copied as is.
  Gives back: text.
  Example:
    show format_date[today[], "DD/MM/YYYY hh:mm"] .

### wait_until[date]
  Pauses until that moment (at once if it's past).
  Example:
    wait_until[add_time[today[], 1, "hours"]]

### every[amount, unit, job]
  Runs job now, and then again every amount units, until job returns false.
  amount   a whole number
  unit     "minutes", "days", ...
  job      a function with no parameters
  Example:
    every[7, "days", backup]
`,

	"data": `Working with collections (lists, sets, maps), and with tables of rows:
showing them, and saving them to and reading them from files.
"import data" also makes the variable tablerows (20): how many rows
table[...] shows. Change it like any variable; none shows every row.

### process[collection, function]
  Replaces every element (of a list or set) or every value (of a map)
  with what the function gives for it. Changes the collection itself.
  collection   a list, set or map
  function     x gives ...  (for a map: x gives ..., or [key, value] gives ...)
  Gives back: the same collection, changed.
  Example:
    nums process x gives x * 10 .

### keep[collection, function]
  Keeps only the elements (or map entries) for which the function gives
  true: a filter. Changes the collection itself.
  Gives back: the same collection, changed.
  Example:
    nums keep x gives x > 3 .

### copy[value]
  A new list, set, map or assembled value with the same contents, so
  changing the copy leaves the original alone.
  Example:
    big = copy[nums] process x gives x * 10

### table[rows, limit]
  Lays rows out as a text table, one row per line, columns lined up.
  rows    a list of maps (what sql_query gives), a list of assembled
          values, a list of lists, a plain list, a set, one map, or one
          assembled value
  limit   optional: at most this many rows for this call (else tablerows)
  Gives back: text, so show it: show table[rows] .
  A row without one of the columns leaves that cell blank; none shows as
  none. Numbers line up on the right. Rows past the limit end with a
  line like "... 12 more rows".
  Example:
    show table[sql_query[db, "SELECT * FROM books"]] .

### table_write[path, rows]
  Saves rows to a file, in the layout table[...] shows. The file's
  ending picks the format: .csv (comma-separated, also any other ending),
  .tsv (tab-separated), .txt (the aligned table, every row), or .json
  (a list of objects, one per row, keeping numbers and true/false).
  path   where to save; an existing file is replaced
  rows   anything table[...] takes
  Gives back: how many rows were written.
  Example:
    table_write["orders.csv", orders]

### table_read[path]
  Reads a .csv, .tsv or .json file. A .csv or .tsv file's first line
  names the columns; a .json file is a list of objects, one per row.
  path   the file
  Gives back: a list of maps, one per row, keyed by the column names
  (the same shape sql_query gives). From .csv and .tsv every value is
  text ("950") and an empty cell is none; use change to turn text into
  numbers. From .json, values keep their kind (950 stays a number).
  A badly formed file is a csv error (json for a .json file).
  Example:
    rows = table_read["orders.csv"]
    show rows at get[0] at get["item"] .
`,

	"system": `The command line, environment, files and folders, and the program itself.
Paths start from the folder turtle was run in (scriptFolder[] gives the
script's own folder).

### args[]
  The words typed after the script's name: turtle report.t a.txt b.txt
  Gives back: a list of text (empty if none).

### exists[path]
  Is there a file or folder at path?
  Gives back: true or false.

### isFile[path]
  Is path a file?
  Gives back: true or false.

### isFolder[path]
  Is path a folder?
  Gives back: true or false.

### exit[code]
  Ends the program now. code is optional: 0 (the default) means it went
  well, anything else that it failed.
  Example:
    exit[1]

### env[name]
  An environment variable's value.
  Gives back: text, or none if it isn't set.
  Example:
    home = env["HOME"]

### scriptFolder[]
  The full path of the folder the running script is in.
  Example:
    data = "{scriptFolder[]}/data.csv"

### contents[path]
  The names of the files and folders in a folder (path is optional, "."
  by default).
  Gives back: a sorted list of names.

### erase[path]
  Deletes a file, or a folder and everything in it. There is no undo.
  A missing path is a file error.

### warn ... .
  Like show, but to standard error, so it isn't mixed with output that's
  piped or saved.
  Example:
    warn "can't read ", name .
`,

	"strings": `Text functions that read well as sentences. Positions count
characters from 0.

### find[text, part]
  Where part first appears in text.
  Gives back: its position, or -1 if it isn't there.
  Example:
    show line find "wor" .

### substring[text, start, end]
  The characters from start up to (not including) end. end is optional
  (to the end); a negative position counts from the end.
  Example:
    show line substring 0, 5 .

### isinstring[part, text]
  Does text contain part?
  Gives back: true or false.
  Example:
    if ] "wor" isinstring line [

### join[items, separator]
  Joins a list (or set) into one text, with separator between the items
  (optional: nothing by default).
  Example:
    show words join ", " .
`,

	"json": `Reading and writing JSON. Objects become maps, arrays lists, null none.

### load[text]
  Reads JSON text.
  Gives back: the Turtle value (a map, list, number, text, ...).
  Example:
    user = load['{"name": "Ann"}']

### json_text[value]
  Writes a value as JSON, on one line.
  Gives back: text.

### json_read[path]
  Reads a JSON file.
  Gives back: the Turtle value. Bad JSON is a json error.

### json_write[path, value]
  Saves a value to a file as indented JSON.

### json_get[value, step, step, ...]
  Follows a path of map keys and list positions into a value.
  Gives back: what's at the end, or none if any step is missing.
  Example:
    port = json_get[config, "server", "port"]
`,

	"http": `Web requests. A map, list or assembled value sent as a body goes as JSON.

### http_get[url]
  Fetches a web address.
  Gives back: the response's body as text. A network problem or an
  error status (4xx, 5xx) is an http error.

### http_post[url, body]
  Sends body to a web address.
  Gives back: the response's body as text.

### http_request[method, url, body, headers]
  Any request: "GET", "PUT", "DELETE", ... body and headers (a map) are
  optional.
  Gives back: a map with "status" (a number), "body" (text) and
  "headers" (a map). Only a network problem is an error; check the
  status yourself.
`,

	"sql": `Databases: SQLite files, and PostgreSQL and MySQL servers, all through
the same functions; sql_open's address picks which. Read and change
them, and move CSV files in and out. SQLite files work with every other
SQLite program. Values for ? in a statement are given as a list. A bad
statement, or a change the database refuses, is an sql error.

### sql_open[address]
  Opens a database.
  address   a SQLite file ("shop.db", or "sqlite:shop.db"), or a server:
            "postgres://user:password@host:5432/database"
            "mysql://user:password@host:3306/database"
  Gives back: the database, to pass to the other sql_ functions.
  A missing file is a file error (sql_create makes a new one). A server
  that can't be reached, or a wrong password, is an sql error.
  Server options go after a ?: for PostgreSQL sslmode=disable, require
  or prefer (the default) and connect_timeout=10; for MySQL tls=false,
  true, skip-verify or preferred (the default) and timeout=10.
  Example:
    db = sql_open["shop.db"]
    db = sql_open["postgres://ann:secret@localhost:5432/shop"]

### sql_create[path]
  Makes a new, empty SQLite database file and opens it. Fails if the
  file is already there. (A server's databases are made on the server,
  with CREATE DATABASE.)
  Gives back: the database.
  Example:
    db = sql_create["new.db"]

### sql_query[db, query, values]
  Asks the database for rows: runs a SELECT.
  db       the database
  query    a SELECT statement, with ? where values go
  values   optional: a list, one value for each ?
  Gives back: a list of maps, one per row: column name to value.
  NULL is none; a server's booleans and dates come back as booleans
  and dates. An INSERT, UPDATE or DELETE ending in RETURNING also
  works here, giving back the rows it changed.
  Example:
    rows = sql_query[db, "SELECT title FROM books WHERE price < ?", list [1000]]

### sql_run[db, statement, values]
  Changes the database: INSERT, UPDATE, DELETE, CREATE TABLE, ALTER,
  DROP, BEGIN, COMMIT, ROLLBACK, ... Each statement is saved at once
  (or with COMMIT, inside BEGIN). Several statements separated by ; can
  run together when no values are given.
  db          the database
  statement   the SQL, with ? where values go
  values      optional: a list, one value for each ?
  Gives back: how many rows it added, changed or removed (0 for others).
  Example:
    n = sql_run[db, "UPDATE books SET price = ? WHERE sku = ?", list [900, "B1"]]

### sql_tables[db]
  The names of the database's tables.
  Gives back: a list of text.

### sql_save[db, query, path, values]
  Runs a query and saves its rows to a file: the first line names the
  columns, then one line per row. The file's ending picks the format:
  .csv (comma-separated), .tsv (tab-separated), .txt (an aligned table)
  or .json (a list of objects, one per row).
  db       the database
  query    a SELECT, with ? where values go
  path     where to save; an existing file is replaced
  values   optional: a list, one value for each ?
  Gives back: how many rows were saved.
  Example:
    sql_save[db, "SELECT * FROM books", "books.csv"]

### sql_load[db, table, path]
  Adds a record to a table for each line of a .csv, .tsv or .json file.
  db      the database
  table   the table's name
  path    the file; its first line names the table's columns
  Gives back: how many records were added.
  All or nothing: if one line is refused (say, a duplicate key), no line
  is added. Text in a number column becomes a number; an empty cell is NULL.
  Example:
    sql_load[db, "books", "new_books.csv"]

### sql_update[db, table, key, path]
  Changes records from a .csv, .tsv or .json file: for each line, finds the
  record whose key column has the line's key, and sets the line's other
  columns.
  db      the database
  table   the table's name
  key     the column that says which record (like "sku" or "id")
  path    a .csv, .tsv or .json file; it needs the key column and at least one other
  Gives back: how many records changed. A key not in the table changes
  nothing. All or nothing, as with sql_load.
  Example (prices.csv has sku,price):
    sql_update[db, "books", "sku", "prices.csv"]

### sql_delete[db, table, key, path]
  Removes the records whose key column matches a line of a .csv, .tsv or .json file. The file's other columns are ignored.
  Gives back: how many records were removed. All or nothing.
  Example (gone.csv has sku):
    sql_delete[db, "books", "sku", "gone.csv"]

### sql_upsert[db, table, key, path]
  Adds or changes, from a .csv, .tsv or .json file: a line whose key isn't in
  the table becomes a new record; a line whose key is there changes that
  record. The key column must be the table's PRIMARY KEY or UNIQUE.
  Gives back: how many records were added or changed. All or nothing.
  Example:
    sql_upsert[db, "books", "sku", "restock.csv"]

### sql_close[db]
  Closes the database. A transaction not yet committed is undone.
  Closing twice is fine.
`,
}
