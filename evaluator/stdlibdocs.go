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

### number at round places
  The nearest whole number; a half rounds away from zero (4.5 becomes 5).
  places     optional: round[2] keeps 2 places and gives a float
             (3.14159 -> 3.14, 0.6000000000000001 -> 0.6); round[-2]
             rounds to hundreds (1234 -> 1200). -15 to 15.
  Example:
    r is 4.5 at round .
    c is prices process p give p * 1.08 at round[2] .

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

	"linear": `Matrices and vectors. After import linear, matrix [...] makes a
matrix; its rows end at ; or at the end of a line:
    a = matrix [1, 2; 3, 4]
    b = matrix [
        1, 2, 3
        4, 5, 6
    ]
+ and - work on two matrices of the same size, or a matrix and a number
(on every element: a + 1), * is the matrix product (a matrix times a list
gives a list), and * or / by a number scales every element. Vectors are plain lists of numbers. A matrix of whole numbers
shows and gives back integers; anything that can make a fraction gives
floats. Methods: m at rows, columns, shape, get[r, c], put[value, r, c],
row[r], column[c], flatten, reshape[r, c]. change rows to matrix makes one from a list of lists
(or table_read's rows); change m to list turns it back.

### identity[n]
  The n x n identity matrix: 1s on the diagonal, 0s elsewhere.
  Example:
    i = identity[3]

### zeros[rows, columns]
  A matrix of 0s; zeros[n] is n x n.
  Example:
    z = zeros[2, 3]

### ones[rows, columns]
  A matrix of 1s; ones[n] is n x n.
  Example:
    o = ones[2, 3]

### diagonal[x]
  From a list: the square matrix with those numbers on its diagonal. From
  a matrix: the list of the numbers on its diagonal.
  Example:
    d = diagonal[list [1, 2, 3]]

### shape[m]
  The size: list [rows, columns].
  Example:
    size = shape[m]                 // [ 2, 3 ]

### row[m, r]
  Row r (from 0) as a list.
  Example:
    first = row[m, 0]

### column[m, c]
  Column c (from 0) as a list.
  Example:
    prices = column[m, 2]

### flatten[m]
  The numbers of m in one row, read row by row: a 1 x n matrix. m is
  unchanged. Method: m at flatten. Sentence form: m flatten.
  Example:
    m = matrix [1, 2, 3; 4, 5, 6]
    f = flatten[m]                  // [ 1  2  3  4  5  6 ]

### reshape[m, rows, columns]
  The same numbers, in the same order, as a rows x columns matrix. m is
  unchanged. rows times columns must be how many numbers m has, or it's a
  linear error. Method: m at reshape[rows, columns]. Sentence form:
  m reshape rows, columns.
  Example:
    m = matrix [1, 2, 3; 4, 5, 6]
    r = reshape[m, 3, 2]            // [ 1  2 ]
                                    // [ 3  4 ]
                                    // [ 5  6 ]

### transpose[m]
  The matrix with its rows as columns. Sentence form: m transpose.
  Example:
    t = transpose[m]

### trace[m]
  The sum of the diagonal of a square matrix.
  Example:
    t = trace[m]

### determinant[m]
  The determinant of a square matrix: 0 when it has no inverse. An integer
  for a matrix of whole numbers.
  Example:
    d = determinant[matrix [1, 2; 3, 4]]     // -2

### inverse[m]
  The matrix that multiplies m to the identity. A singular matrix (its
  determinant is 0) is a linear error.
  Example:
    inv = inverse[m]

### rank[m]
  How many rows (or columns) are independent: none of them a mix of the
  others.
  Example:
    r = rank[matrix [1, 2; 2, 4]]            // 1

### power[m, k]
  m multiplied by itself k times; power[m, 0] is the identity, and a
  negative k is a power of the inverse.
  Example:
    later = power[steps, 10]

### multiply_each[a, b]
  Element by element: each number times the one in the same place.
  Example:
    c = multiply_each[a, b]

### solve[a, b]
  The x with a * x == b, for a square a. b is a list (x is a list) or a
  matrix (one answer per column). Sentence form: a solve b.
  Example:
    x = solve[matrix [2, 1; 1, 3], list [3, 5]]

### least_squares[a, b]
  The x that brings a * x closest to b, for more equations (rows) than
  unknowns (columns): the line or curve of best fit.
  Example:
    fit = least_squares[points, ys]

### dot[u, v]
  The dot product of two lists of the same length.
  Example:
    d = dot[list [1, 2, 3], list [4, 5, 6]]  // 32

### cross[u, v]
  The cross product of two 3-D vectors (lists of 3 numbers).
  Example:
    n = cross[list [1, 0, 0], list [0, 1, 0]]

### norm[x]
  A list's length (the square root of its squares added up); for a
  matrix, every number counts (the Frobenius norm).
  Example:
    n = norm[list [3, 4]]                    // 5.0

### unit[v]
  The list scaled to length 1, the same direction.
  Example:
    u = unit[list [3, 4]]                    // [ 0.6, 0.8 ]

### lu[m]
  The LU decomposition of a square matrix: a map with "l" (lower, 1s on
  the diagonal), "u" (upper) and "p" (row order), where p * m == l * u.
  Example:
    parts = lu[m]

### qr[m]
  The QR decomposition: a map with "q" (orthonormal columns) and "r"
  (upper triangular), where q * r == m. Needs at least as many rows as
  columns.
  Example:
    parts = qr[m]

### eigen[m]
  The eigenvalues and eigenvectors of a symmetric matrix: a map with
  "values" (a list, largest first) and "vectors" (a matrix, one vector
  per column, each of length 1).
  Example:
    e = eigen[covariances]

### svd[m]
  The singular value decomposition, any shape: a map with "u", "s" (a
  list, largest first) and "v", where m == u * diagonal[s] * transpose[v].
  Example:
    parts = svd[m]
`,

	"time": `The clock, dates, time zones, date arithmetic, and waiting.
Units are "seconds", "minutes", "hours", "days", "weeks", "months",
"years" (or the singular). A date shows as 2026-10-03 14:05:00 and has
parts read with "of": year, month, day, hour, minute, second, weekday,
zone. A date in another zone than the computer's shows its zone:
2026-12-25 09:00:00 GMT. Zones are names like "America/New_York",
"Europe/London", "Asia/Tokyo", or "UTC" or "local"; dates compare as
moments, whatever their zones.

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

### today[zone]
  The date and time now, in local time, or in a time zone.
  zone     optional: "Asia/Tokyo", "UTC", ...
  Gives back: a date.
  Example:
    show weekday of today[] .
    tokyo = today["Asia/Tokyo"]

### to_zone[date, zone]
  The same moment, on another zone's clock.
  Gives back: a date (== the original: it's the same moment).
  Example:
    ny = to_zone[d, "America/New_York"]
    ny = d to_zone "America/New_York"
    show zone of ny .

### today_utc[]
  The date and time now, in UTC.
  Gives back: a date.

### make_date[year, month, day, hour, minute, second, zone]
  A date from its parts. Hour, minute and second are optional (all three,
  or none); a time zone can go last. A date that doesn't exist, like
  February 30, is a date error.
  Gives back: a date, in local time or in the zone.
  Example:
    d = make_date[2026, 12, 25]
    d = make_date[2026, 12, 25, 9, 0, 0, "Europe/London"]

### to_date[text, zone]
  Reads a date from text: "2026-10-03", "2026-10-03 14:05",
  "2026-10-03 14:05:00", or ISO 8601 ("2026-10-03T14:05:00Z",
  "2026-10-03T14:05:00-04:00", which keep their offset).
  zone     optional: the zone for text that names none (local otherwise)
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
  Weekday (Thursday), Wkd (Thu), Zone (EST), Offset (-05:00). Anything
  else is copied as is.
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
  with what the function gives for it.
  collection   a list, set or map
  function     x give ...  (for a map: x give ..., or [key, value] give ...)
  As a sentence on its own it changes the collection itself; used as a
  value (assigned, or in an expression) it gives a new collection and
  leaves the original alone.
  Example:
    nums process x give x * 10 .
    bigger is nums process x give x * 10 .

### keep[collection, function]
  Keeps only the elements (or map entries) for which the function gives
  true: a filter. As a sentence on its own it changes the collection
  itself; used as a value it gives a new one and leaves the original.
  Example:
    nums keep x give x > 3 .
    big is nums keep x give x > 3 .

### copy[value, deep]
  A new list, set, map or assembled value with the same contents.
  value   what to copy
  deep    optional: true copies what's inside too (lists in a list, maps
          in a list, an assembled value's lists), so nothing is shared;
          false or left out shares what's inside
  Example:
    outer = copy[rows]
    full = copy[rows, true]

### range[from, to, step]
  The whole numbers from from up to, not including, to, as a list (as in
  Python and Go). Sentence form: 1 range 5.
  from       optional: 0 if left out (range[5] is 0 to 4)
  to         where it stops, not included
  step       optional: 2 counts by twos; a negative step counts down
  Example:
    range[5]             // [ 0, 1, 2, 3, 4 ]
    range[1, 5]          // [ 1, 2, 3, 4 ]
    range[0, 10, 2]      // [ 0, 2, 4, 6, 8 ]
    range[5, 0, -1]      // [ 5, 4, 3, 2, 1 ]
    [loop][i in 1 range 4] ... [loop][end]

### reduce[collection, start, function]
  Boils a collection down to one value: a running total that starts at
  start, and for each item (a map's values) becomes what the function
  gives for the total so far and the item. Changes nothing.
  function   [total, x] give ...
  Example:
    total = nums reduce 0, [t, x] give t + x
    word = letters reduce "", [w, c] give w + c

### sum[collection]
  Adds up a list's or set's numbers (a map's values): an integer if they
  all are, 0 when there are none. Sentence form: nums sum.
  Example:
    total = sum[prices]

Statistics: each takes a list, a set or a map (its values), or a list of
rows (maps or assembled values, as sql_query and table_read give) and a
column name. none is skipped, as a missing value; anything else that
isn't a number is an error. Sentence form: prices mean.

### mean[numbers, column]
  The average: the numbers added up, divided by how many there are.
  column   optional: with a list of rows, the column to use
  Gives back: a float.
  Example:
    avg = mean[prices]
    avg = mean[books, "price"]

### median[numbers, column]
  The middle number once sorted; with an even count, the average of the
  two in the middle.
  Gives back: the middle number itself (an integer stays one), or a float.
  Example:
    mid = median[list [3, 1, 2]]          // 2

### mode[values, column]
  The value that appears most often; a tie goes to the one seen first.
  Works on any values, not only numbers.
  Example:
    top = mode[list ["tea", "coffee", "tea"]]   // "tea"

### variance[numbers, column]
  The sample variance: the average squared distance from the mean,
  dividing by n - 1 (as Python and spreadsheets do). Needs 2 numbers.
  Example:
    v = variance[prices]

### stdev[numbers, column]
  The sample standard deviation: the square root of variance. Needs 2
  numbers.
  Example:
    spread = stdev[prices]

### pvariance[numbers, column]
  The population variance: like variance, dividing by n. Use it when the
  numbers are the whole population, not a sample of it.
  Example:
    v = pvariance[scores]

### pstdev[numbers, column]
  The population standard deviation: the square root of pvariance.
  Example:
    spread = pstdev[scores]

### percentile[numbers, column, percent]
  The number below which that percent of the numbers fall, between the
  two nearest when it lands between them (as numpy and a spreadsheet's
  PERCENTILE give). percentile[x, 50] is the median.
  percent   0 to 100
  Example:
    p90 = percentile[times, 90]
    p90 = percentile[rows, "ms", 90]

### covariance[xs, ys]
  How two lists of numbers move together, the sample form (n - 1):
  positive when they rise together. Also covariance[rows, "a", "b"]. A
  pair with none on either side is skipped.
  Example:
    c = covariance[heights, weights]

### correlation[xs, ys]
  Pearson's correlation, from -1 (one falls as the other rises) through 0
  (no straight-line link) to 1 (they rise together). Also
  correlation[rows, "a", "b"].
  Example:
    r = correlation[ads, sales]

### zscores[numbers, column]
  Each number's distance from the mean, counted in (sample) standard
  deviations.
  Gives back: a new list of floats, in the same order.
  Example:
    z = zscores[list [1, 2, 3]]          // [ -1.0, 0.0, 1.0 ]

### describe[numbers, column]
  A summary: count, mean, stdev (none with one number), min, 25%, median,
  75%, max.
  Gives back: a map; show table[describe[x]] . lays it out.
  Example:
    show table[describe[books, "price"]] .

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

### table_write[path, rows, options]
  Saves rows to a file, in the layout table[...] shows. The file's
  ending picks the format: .csv (comma-separated, also any other ending),
  .tsv (tab-separated), .txt (the aligned table, every row), or .json
  (a list of objects, one per row, keeping numbers and true/false).
  path     where to save; an existing file is replaced
  rows     anything table[...] takes
  options  optional: map ["quote": "text"] quotes every text value of a
           .csv or .tsv file ("007", not 007). The default, "needed",
           quotes text only where it must (it holds a comma, a quote or
           a line break).
  Numbers are never quoted, so they read back as numbers.
  Gives back: how many rows were written.
  Example:
    table_write["orders.csv", orders]
    table_write["codes.csv", codes, map ["quote": "text"]]

### table_read[path [, types]]
  Reads a .csv, .tsv or .json file. A .csv or .tsv file's first line
  names the columns; a .json file is a list of objects, one per row.
  path   the file
  types  optional: a map of column name to type, to say what a
         column is: "code": "text", "qty": "integer". Types: string,
         text, integer, float, boolean, date, or a SQL type
         (VARCHAR(10), NUMERIC(10, 2), ...). Columns left out keep what
         the file has. "primary_key": "column" checks every row has a
         different one.
  Gives back: a list of maps, one per row, keyed by the column names
  (the same shape sql_query gives). From .csv and .tsv, what a cell holds
  decides, quoted or not (as in pandas): a plain number (950, "950",
  -2.5, 1e5) is a number; 007 and other text is text; an empty cell is
  none. From .json, values keep their kind.
  A badly formed file is a csv error (json for a .json file); a cell
  that isn't its type is a number, date or type error.
  Example:
    rows = table_read["orders.csv"]
    types = map ["code": "text", "qty": "integer", "primary_key": "code"]
    rows = table_read["orders.csv", types]
`,

	"system": `The command line, environment, files and folders, and the program itself.
Paths start from the folder turtle was run in (scriptfolder[] gives the
script's own folder).

### args[]
  The words typed after the script's name: turtle report.turtle a.txt b.txt
  Gives back: a list of text (empty if none).

### exists[path]
  Is there a file or folder at path?
  Gives back: true or false.

### isfile[path]
  Is path a file?
  Gives back: true or false.

### isfolder[path]
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

### scriptfolder[]
  The full path of the folder the running script is in.
  Example:
    data = "{scriptfolder[]}/data.csv"

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

### copyto[from, to, replace]
  Copies a file, or a folder and everything in it, to the path to
  (missing folders on the way are made).
  replace    optional: true replaces what's already at to; without it,
             something there is an error
  Example:
    copyto["report.csv", "backup/report.csv"]

### moveto[from, to, replace]
  Moves (or renames) a file or folder; replace as for copyto. Works
  across disks.
  Example:
    moveto["old.txt", "archive/old.txt"]

### makefolder[path]
  Makes a folder, and any missing folders above it. Fine if it's there.
  Example:
    makefolder["out/2026/october"]

### walk[folder]
  Every file under the folder, in its subfolders too, sorted; each path
  starts with folder and uses /. "." if left out.
  Gives back: a list of text.
  Example:
    [loop][f in walk["src"]] ... [loop][end]

### pack[from, archive, replace]
  Puts a file, or a folder and everything in it, into an archive. The
  kind comes from the name: .zip, .tar, .tar.gz or .tgz. A folder goes in
  under its own name, so unpacking gives the folder back.
  Example:
    pack["src", "src.zip"]

### unpack[archive, folder, replace]
  Puts an archive's files into folder. A file already there is an error
  unless replace is true; an entry that would land outside the folder is
  refused.
  Example:
    unpack["src.zip", "restored"]       // restored/src/...

### loadenv[file]
  Reads a .env file (".env" if left out): KEY=value lines, # comments,
  "quoted" or 'quoted' values. Each key is also set for env[...], unless
  the environment already has it.
  Gives back: a map of the file's keys and values (all text).
  Example:
    settings = loadenv[]
    key = env["API_KEY"]

### options[name: default, ...]
  Named options from the command line. Each default sets the option's
  kind: false is an on/off switch (-v), a number takes a number (--count
  5), text takes text (--out file.csv or --out=file.csv). -- ends the
  options; --help shows them and ends the program. What isn't an option
  is left for args[].
  Gives back: a map of every option and its value.
  Example:
    opts = options["--out": "result.csv", "-v": false]
    out = opts at get["--out"]
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

	"pattern": `Patterns (regular expressions) in text. Write patterns in backticks,
which keep { } and \ exactly as typed: \d is a digit, \s a space, \w a
letter, digit or _; {3} three of the one before, + one or more, * any
number, ? maybe; [abc] one of; ^ start, $ end; ( ) a group. The syntax is
Go's (RE2), so no pattern can take forever. A pattern that isn't valid is
an error of kind pattern.

### matches[text, pattern]
  Whether the pattern is found anywhere in text (^ and $ for all of it).
  Gives back: true or false.
  Example:
    if ] matches[code, ` + "`" + `^[A-Z]{3}-\d{4}$` + "`" + `] [ ... if [end]

### findall[text, pattern]
  Every match, in order.
  Gives back: a list of text (empty if none).
  Example:
    findall["a1 b22", ` + "`" + `\d+` + "`" + `]          // [ "1", "22" ]

### replaceall[text, pattern, with]
  Replaces every match. In with, $1 is the first group, $2 the second
  (write with in backticks too, so {} and $ stay as typed).
  Example:
    replaceall["2026-10-06", ` + "`" + `(\d+)-(\d+)-(\d+)` + "`" + `, ` + "`" + `$3/$2/$1` + "`" + `]   // "06/10/2026"

### splitby[text, pattern]
  Splits text wherever the pattern matches.
  Gives back: a list of text.
  Example:
    splitby["a, b;c", ` + "`" + `[,;]\s*` + "`" + `]      // [ "a", "b", "c" ]

### groups[text, pattern]
  The parts in ( ) of the first match.
  Gives back: a list of text, or none if nothing matches.
  Example:
    groups["2026-10-06", ` + "`" + `(\d+)-(\d+)` + "`" + `]    // [ "2026", "10" ]
`,

	"crypt": `Hashes, signatures, encodings, random ids and tokens, passwords and
encryption. Bad input (text that isn't base64, a wrong passphrase, an
unknown algorithm) is an error of kind crypt.

### hash[text, algorithm]
  A fingerprint of the text: the same text always gives the same hash.
  algorithm  optional: "sha256" (the default), "sha512", "sha1" or "md5"
             (sha1 and md5 only to match old systems)
  Gives back: hex text.
  Example:
    h = hash["hello"]
    h = "hello" hash "sha512"

### filehash[path, algorithm]
  hash of a file's contents (a checksum), read a piece at a time.
  Example:
    sum = filehash["release.zip"]

### hmac[text, key, algorithm]
  A signature: the hash of text with a secret key, as web APIs and
  webhooks use. algorithm as for hash.
  Gives back: hex text.
  Example:
    sig = body hmac secret

### encode[text, how]
  how: "base64" (the default), "base64url" (for links, no padding) or
  "hex".
  Example:
    b = encode["hi"]              // "aGk="

### decode[text, how]
  The other way; how as for encode.
  Example:
    s = decode["aGk="]            // "hi"

### uuid[]
  A random id (a version 4 UUID): 36 characters, different every time.

### token[length]
  Random letters and digits from the system's secure source, for keys,
  session ids and reset links. length: 32 if left out (1 to 4096).
  Example:
    key = token[]

### passwordhash[password]
  What to store instead of a password: salted PBKDF2-SHA256 (600,000
  rounds), different each time for the same password.
  Example:
    stored = passwordhash[pw]

### passwordcheck[password, stored]
  Whether password is the one stored was made from.
  Gives back: true or false.
  Example:
    if ] passwordcheck[typed, stored] [ ... if [end]

### encrypt[text, passphrase]
  Locks text with a passphrase (AES-256-GCM, the key made from the
  passphrase with PBKDF2). Different each time; only decrypt with the
  same passphrase opens it.
  Gives back: text starting "turtle1:", safe to save or send.
  Example:
    box = encrypt[notes, phrase]

### decrypt[box, passphrase]
  Opens what encrypt made. A wrong passphrase, or a box that was
  changed, is an error.
  Example:
    notes = decrypt[box, phrase]
`,

	"log": `Log lines: a level, a time and where they came from, to the console
(stderr) and, if you like, a file. In a file with "import log":

    log "server started on port ", port .         info, the default level
    log debug "row ", row .                       also: info, warn, error
    log info "login", map ["user": name] .        a map adds fields: user=ann

Settings (variables; set at the top of a file, or inside a function for it):
    loglevel = "info"        the lowest level shown; "debug" ... "error", "off"
    logconsole = true        print log lines to stderr
    logfile = none           a file to add lines to (kept across runs)
    logtime = "YYYY-MM-DD hh:mm:ss"   format_date's patterns; none for no time
    logparts = list ["time", "level", "message"]   also file, line, where
    logformat = "text"       or "json": one JSON object per line
    logmaxsize = none        rotate the file past this many bytes (app.log.1 ...)
    logkeep = 3              rotated files to keep
    outputfile = none        a file that also gets what show and warn print

An error that stops the program is written to logfile too.
`,

	"test": `Tests. In a file with "import test", three statements check your code;
"turtle test" runs every test_ function in every test_*.turtle file.

    check total[order] == 45 .                     one fact; == shows how they differ
    check x is integer .                           also: float number string boolean
                                                   list set map date none function empty,
                                                   an assembled type, "is not"
    check 0.1 + 0.2 is close to 0.3 .              decimals ("within 0.01" to choose)
    check 1 div 0 fails [math] .                   the right answer is an error
    verify nums each x give x > 0 .               a rule for every item; also any, not,
                                                   at least N, at most N, exactly N,
                                                   and "each pair [a, b] give a <= b"
    validate evens[nums] with nums as list of integer
        that result each x give x % 2 == 0 .      the rule on 100 random inputs,
                                                   shrunk to the smallest that fails;
                                                   or: matches other_function[nums]

Settings (set at the top of the file, or inside one test for that test):
    suite = false        false: each test alone; "stop": a failure skips the
                         rest; "all": run all, failures listed at the end
    benchmark = false    true: time many runs of each test
    runs = none          none: as many runs as fit in benchtime; or a number
    benchtime = 1        seconds
    cases = 100          random inputs each validate tries
    seed = none          a number repeats the same random inputs

A failure is an error of kind test. Full guide: docs/testing.md.
`,

	"server": `A web server. Routes are a map from "METHOD /path" to what answers
it: a function (a handler), a fixed value, or, on a /* route, a folder
whose files it serves. A handler gets the request as a map and gives back
text (a page), a map or list (JSON), reply[...] or redirect[...]. An
error in a handler answers 500 and is shown; the server goes on.
Requests take turns. Ctrl+C stops it.

The request: method, path, params (from :name parts, and * for the rest
of a /* route), query, headers (lowercase names), body (text), json (the
body read as JSON, or none), form (a posted form), ip.

Settings ("import server" makes them):
    serverlog = true            show a line per request: GET /users/7 200 3ms
    serverhost = "localhost"    "0.0.0.0" to answer other computers too

### serve[routes, port]
  Starts the server and answers requests until the program is stopped.
  routes   a map: "GET /": home, "GET /users/:id": showuser,
           "POST /users": adduser, "GET /health": "ok",
           "GET /static/*": "public"
  port     optional: 8080 by default
  Example:
    serve[app, 8080]
    app serve 8080

### reply[status, body, headers]
  An answer with its own status, and optionally headers.
  status   200 ok, 201 made, 404 not found, ...
  body     optional: text, or a map or list (JSON)
  headers  optional: a map, like map ["Cache-Control": "no-store"]
  Gives back: a Reply { status, body, headers }.
  Example:
    return reply[404, "no such user"]

### redirect[address, status]
  Sends the visitor to another address (302, or 301, 303, 307, 308).
  Example:
    return redirect["/login"]
`,

	"config": `Settings files, the format chosen by the file's extension: .toml,
.json or .env. TOML's [sections] become maps inside the map; numbers,
booleans, dates and lists keep their kinds. A file that isn't well
written is an error of kind config naming its line.

### config_read[path]
  Reads a settings file.
  path      a .toml, .json or .env file
  Gives back: a map ([sections] and objects as maps inside it; a .env
  file's values are text).
  Example:
    settings = config_read["app.toml"]
    port = port of server of settings
    settings = "app.toml" config_read

### config_write[path, settings]
  Writes a map as a settings file, by the file's extension. TOML has no
  none, and a .env file holds only single values.
  Example:
    config_write["app.toml", settings]
`,

	"schedule": `Many web requests, shell commands or database queries at once. The
work runs side by side; your Turtle code still runs one line at a time.

At most schedulelimit items run at once, like a semaphore: each one that
finishes starts the next straight away. Results come back in the list's
order, whatever order they finish in. The first failure stops the call
with that item's error (the running ones finish, the waiting ones never
start); with skipschedule_error = true a failed item becomes none instead.

Settings ("import schedule" makes them; change them like any variable):
    schedulelimit = 5             how many at once; none: all of them
    skipschedule_error = false    true: a failed item becomes none

A call's own settings, for that call only, go in a map at the end:
    map ["limit": 10, "skip_errors": true]

A bad setting, a command that can't start, or queryall on SQLite is an
error of kind schedule.

### fetchall[urls, settings]
  http_get every address.
  urls       a list of web addresses
  settings   optional: map ["limit": n, "skip_errors": true]
  Gives back: a list of the pages' text, in the same order. A failed
  request (no answer, a 4xx/5xx status) is an http error naming the item.
  Example:
    pages = fetchall[urls]
    pages = urls fetchall

### runall[commands, settings]
  Runs every shell command (sh -c; cmd /C on Windows) in the folder
  turtle was run in, capturing what each prints.
  commands   a list of command lines
  settings   optional, as for fetchall
  Gives back: a list of maps, one per command: output (what it printed),
  errors (what it printed as errors), code (its exit code, 0 for
  success). A nonzero code isn't an error; check code. Trailing newlines
  are dropped.
  Example:
    outs = runall[list ["git pull", "make test"]]
    show outs at get[1] at get["code"] .

### queryall[db, queries, settings]
  Runs every query on a PostgreSQL or MySQL database, over extra
  connections that close when it's done (they don't see an open
  transaction on db). Not for SQLite: use sql_query there.
  db         a server database from sql_open
  queries    a list of queries; a query with ? placeholders goes as
             list [query, list of values]
  settings   optional, as for fetchall
  Gives back: a list of results, each a list of maps as sql_query gives.
  A bad query is an sql error naming the item.
  Example:
    r = queryall[db, list ["SELECT count(*) AS n FROM books", list ["SELECT * FROM books WHERE price < ?", list [1000]]]]
`,

	"random": `Random values of any shape, written as a sentence after "random":

    die = random integer from 1 to 6
    nums = random list of 5 integers from 0 to 9
    price = random float from 0.5 to 99.99 rounded to 2
    code = random string of 8                    pin = random digits of 4
    id = random string of 6 from "ABCDEF0123456789"
    day = random date from "2026-01-01" to "2026-12-31"
    grid = random list of 3 lists of 3 integers
    ages = random map of string to integer from 0 to 99
    order = random Order [string, integer from 1 to 10, float]

Kinds: integer (-1000 to 1000 unless from A to B), float (0 up to 1),
string (1 to 10 letters; of N, of N to M, from "chars"), digits of N,
digit, letter, boolean, date and time (2000 through 2030), list of,
set of (all different), map of K to V, and an assembled type with a kind
for each field. Plurals work too (integers, lists). A count comes before
the item's kind: list of 5 integers, list of 2 to 8 integers, list of n
integers; without one, 0 to 10 items.
"import random" also makes the variable seed, none: different values
each run. seed = 42 gives the same values every run from there on, to
repeat a run exactly; setting it again starts them over.
random is a sentence word only in a file with "import random".
pick and chance are written in Turtle, on the Go-written sentence (the
first hybrid library; see evaluator/lib/random.turtle); shuffle and sample are Go.

### pick[x]
  One item, chosen at random.
  x   a list, set, map (one of its keys) or string (one character)
  An empty one is an error.
  Example:
    color = pick[list ["red", "green", "blue"]]

### shuffle[x]
  The items in a random order.
  x   a list, set, map (its keys) or string
  Gives back: a new list (a string, for a string); x is left as it was.
  Example:
    deck = shuffle[cards]

### sample[x, n]
  n different items (different positions), chosen at random.
  x   a list, set, map (its keys) or string
  n   how many; more than x has is an index error
  Gives back: a list.
  Example:
    hand = sample[deck, 5]

### chance[p]
  true with probability p.
  p   a number from 0 to 1: 0.3 is true 30% of the time
  Example:
    if ] chance[0.3] [
        show "rain" .
    if [end]
`,

	"sort": `Putting things in order. Works on lists, sets, maps (their entries),
text (its characters), and anything inside them: lists of assembled
values, maps of maps, lists of lists... Gives back a new list (a new
map, for a map) and leaves the original as it was.
Values order the same way everywhere: none, then true/false, numbers,
text, dates, lists (item by item), assembled values (field by field),
maps (entry by entry).

### min_sort[collection, key, how]
  Puts things in order, smallest first.
  collection   a list, set, map or text
  key          optional: what to order by. A function (b give b at get["price"]),
               a map key or field name ("price"), a position in a list of lists
               (1), or a list of those (list ["author", "price"]): ties go to
               the next. For a map, the key picks from each value; a function
               gets the value (v give ...) or the key and value ([k, v] give ...).
               Leave it out (or none) to order the items themselves.
  how          optional: "first" gives just the first item (none if
               there's nothing); a number gives that many, as a list.
               For a map, "first" gives the key; a number, a smaller map.
  Gives back: a new list (or map), equal items in their first order.
  Example:
    cheap = min_sort[books, b give price of b]
    cheapest = min_sort[books, "price", "first"]
    youngest = min_sort[ages, a give a, "first"]

### max_sort[collection, key, how]
  Like min_sort, largest first: max_sort[x, key, "first"] is the largest.
  Example:
    top3 = max_sort[scores, s give s, 3]

### is_sorted[collection, key]
  Whether it's already in order, smallest first (equal items allowed).
  Gives back: true or false.

### reverse_list[collection]
  The items in the opposite order (a map: its entries). The reverse
  statement (reverse nums .) changes a list in place instead.
  Example:
    newest_first = reverse_list[log]

### bubble_sort[collection, key]
  The same answer as min_sort, by bubble sort: swaps neighbours that are
  out of order until nothing moves. About n*n steps: for learning and
  comparing, not for large lists.

### insertion_sort[collection, key]
  min_sort's answer by insertion sort: slides each item back into place.
  Quick on lists that are almost in order.

### selection_sort[collection, key]
  min_sort's answer by selection sort: picks the smallest of the rest,
  again and again. Equal items may change order.

### merge_sort[collection, key]
  min_sort's answer by merge sort: sorts each half, then merges them.
  About n*log(n) steps, always.

### quick_sort[collection, key]
  min_sort's answer by quick sort: splits around a middle value. Fast on
  average; equal items may change order.

### heap_sort[collection, key]
  min_sort's answer by heap sort: builds a heap, then takes the largest
  off the top. Equal items may change order.

### shell_sort[collection, key]
  min_sort's answer by Shell sort: insertion sort over shrinking gaps.
  Equal items may change order.

### counting_sort[collection, key]
  min_sort's answer by counting sort, for whole-number keys in a modest
  range (ages, scores): counts each value, then places them.

### radix_sort[collection, key]
  min_sort's answer by radix sort, for whole-number keys: orders them one
  byte at a time.
`,
	"search": `Finding things. Works on lists, sets, maps and text, and what's inside
them, like the sort library. Over a map, a function gets the value
(v give ...) or the key and value ([k, v] give ...).

### find_first[collection, test]
  The first item for which test says yes.
  test   a function giving true or false: b give price of b < 1000
  Gives back: the item (for a map, its key), or none if nothing matches.
  Example:
    cheap = find_first[books, b give price of b < 1000]

### find_last[collection, test]
  The last item for which test says yes, or none.

### find_all[collection, test]
  Every item for which test says yes, in order.
  Gives back: a new list (for a map, a new map of the matching entries).
  Example:
    out_of_stock = find_all[books, b give stock of b == 0]

### find_index[collection, test]
  The position (from 0) of the first item for which test says yes, or -1.

### count_where[collection, test]
  How many items test says yes to.

### find_key[map, value]
  The first key whose value is value, or none.
  Example:
    who = find_key[ages, 30]

### linear_search[collection, value, key]
  The position of the first item equal to value (by key, if given),
  checking each in turn, or -1. Works on anything, sorted or not.
  key     optional: what to order by. A function (b give b at get["price"]),
          a map key or field name ("price"), a position in a list of lists
          (1), or a list of those (list ["author", "price"]): ties go to
          the next. For a map, the key picks from each value; a function
          gets the value (v give ...) or the key and value ([k, v] give ...).
          Leave it out (or none) to order the items themselves.
  Example:
    at = linear_search[names, "Ann"]

### binary_search[sorted, value, key]
  Like linear_search, on a collection sorted smallest first by the same
  key (min_sort makes one): halves the range each step, about 20 steps
  for a million items. Gives the first match's position, or -1.
  Example:
    by_price = min_sort[books, "price"]
    spot = binary_search[by_price, 950, "price"]

### jump_search[sorted, value, key]
  binary_search's answer, by jumping ahead in blocks of about the square
  root of the length, then looking through one block.

### exponential_search[sorted, value, key]
  binary_search's answer, by doubling a bound (1, 2, 4, 8, ...) then
  searching inside it. Quick when the value is near the front.

### interpolation_search[sorted, value, key]
  binary_search's answer, guessing the position from the values at the
  ends, like opening a phone book near the right letter. Very fast on
  evenly spread numbers.

### ternary_search[sorted, value, key]
  binary_search's answer, splitting the range in three each step.

### insert_position[sorted, value, key]
  Where value would go to keep a sorted collection in order: the
  position of the first item not smaller than it.
  Example:
    pos = insert_position[scores, 75]
    insert 75 to scores at pos .
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

### sql_load[db, table, path [, types]]
  Adds a record to a table for each line of a .csv, .tsv or .json file.
  If the table isn't there, makes it from the file's columns first.
  db      the database
  table   the table's name
  path    the file; its first line names the table's columns
  types   optional: a map of column name to type, as table_read takes
          (string, text, integer, float, boolean, date, or a SQL type).
          A new table gets these types (a column left out: integer,
          float or text, from its values), and "primary_key": "column"
          becomes its PRIMARY KEY.
  Gives back: how many records were added.
  A table that's already there decides the types: each cell becomes its
  column's type (950 in a TEXT column is "950"), or the load stops
  at the row and column that can't (a number, date or type error).
  All or nothing: if one line is refused (say, a duplicate key), no line
  is added, and a table it made is dropped again. An empty cell is NULL.
  Example:
    sql_load[db, "books", "new_books.csv"]
    sql_load[db, "agents", "agents.csv", map ["code": "text", "missions": "integer", "primary_key": "code"]]

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
