# Changelog

Turtle is pre-release (0.x). Breaking changes are listed first.

## 0.9.226 (2026-10-23)

- Functions and top-level loops are compiled the first time they run: their variables are kept in numbered slots instead of looked up by name, whole numbers stay plain numbers between steps, and a call makes no scope. Over the 17 benchmark programs, 54% less time and 22% less peak memory (loop 71% faster, calls 64%, sieve 57%, search 56%; loop and calls 41% less memory); 100,000 nested calls use 138 MB, not 243 MB. Anything compiling doesn't cover runs as before, and every program's output is the same. `TURTLE_NOVM=1` turns it off.
- Number methods work on whole numbers past 64 bits instead of failing ("needs a ... number receiver, got integer"): `abs`, `pow`, `floor`, `ceil`, `round` (to places, or to tens and hundreds, exactly), `fixed` and `commas` give every digit; `sqrt` and `log` give the float Python's math does (log from the number's bits, so 2 at pow 2000 has a log); a float function given one too big for a float is a `math` error, not inf or nan.
- `abs` no longer makes a big number for every integer it's given: about 10% faster.

## 0.9.225 (2026-10-23)

Breaking changes (limits on input that could take all memory):

- An integer stops at 10,000,000 bits (about 3 million digits): `*` and `pow` that would go past it are a `math` error, `integer too big`, checked before the work is done. `change ... to integer` reads text of at most 1,000,000 digits. Before, one line such as `pow[2, 4000000000]` could take all memory or run for hours.

Other changes:

- `m["apples"] = 3` says how to set a key or an item (`m at put[key, value]`, or `put value to nums at index .`) instead of that a line can't start with `=`.
- Each level of a function call takes about half the stack it did: 100,000 nested calls use 243 MB instead of 483 MB.
- Text with a long run of plain braces is read in linear time: 300,000 braces take 0.02 s, where they took about 9 s.
- Releases and checks run on GitHub's Node 24 actions (checkout v7, setup-go v7, action-gh-release v3), as Node 20 is retired.

## 0.9.224 (2026-10-23)

Breaking changes (limits on input that could hang or crash Turtle):

- SQL: an expression nested deeper than 1000 levels (brackets, NOTs, signs, or a run of operators such as `1 + 1 + ...`) is an error, `Expression tree is too large (maximum depth 1000)`, as in SQLite. Such a query used to take about 1 GB of stack, and a deeper one would crash.
- `config_read`: TOML arrays and inline tables nested more than 10,000 deep, or a key with more than 10,000 parts, is a `config` error. Deeply nested inline tables used to take time growing with the square of the depth (200,000 levels never finished).
- A web response larger than 64 MB is an `http` error. It used to be cut off at 64 MB without a word.
- `passwordcheck` refuses a stored hash asking for more than 10 million rounds or a key over 64 bytes, as `passwordhash` never makes one; a changed database row could otherwise make a single check run for minutes.

Other changes:

- `json_text` and `json_write` write text directly instead of making an encoder for every key and value: about 4x faster, with a quarter less memory, and the same output byte for byte.
- Finished loops leave their scope and snapshot for the next loop, so code that runs many short loops (a game search) is about 7% faster.
- Tables shown with `table` and `.env` files written by `config_write` no longer build a text replacer for every value.

## 0.9.223 (2026-10-23)

- An assignment remembers where it stored its name, and stores there again while that place is still right, instead of searching the scopes each time: programs that assign in loops run 2 to 6% faster (binary search 6%). New variables, and the error for changing an outer variable from a function, work as before.

## 0.9.222 (2026-10-23)

- A call to a library function by name (`abs[d]`, `sqrt[x]`, `mean[nums]`) remembers where it goes, so the next call there skips the search: about 25% faster in a loop of such calls. It looks again after a `def`, an `import`, or a variable taking a function value, so a function of the program's own still wins.

## 0.9.221 (2026-10-23)

Breaking changes:

- The map methods `add` and `put` take the key first, as a map is written: `m at add["apples", 3]` adds 3 at `"apples"` (a new key gets the value, so on an empty map it gives `{ "apples": 3 }`; on `{ "apples": 10 }` it gives 13), and `m at put["apples", 5]` sets `"apples"` to 5. In 0.9.216 to 0.9.220 both took the value first; swap the two arguments of map `add[...]` and `put[...]` calls written for those versions. The statements `add 3 to m at "apples" .` and `put 5 to m at "apples" .`, `apples of m = 5`, and list and matrix `put` are unchanged.

## 0.9.220 (2026-10-23)

Breaking changes:

- `change "1_000" to integer` is `1000`: single underscores between digits group them, as in Python and as `change ... to float` already allowed. `"1__000"`, `"_1000"` and `"1000_"` are still not numbers.
- A list, set, map or assembled value equals itself without its contents being compared, as in Python: `x == x` is `true`, even after `add x to x .`, and a list holding `nan` equals itself (two separate lists holding `nan` still don't).
- `json_text` and `json_write` refuse a value nested more than 10,000 levels deep, the depth `load` can read back.

Other changes:

- A value that holds itself (`add x to x .`), or one nested more than 100,000 levels deep, no longer crashes Turtle when it's shown, compared, sorted, deep-copied, put in text or used as a set item or map key. It's an error that `safe` can handle.
- `math` has `asin`, `acos`, `atan` (`1 at asin` is `pi / 2`) and `atan2` (`y at atan2[x]`, as Python's `math.atan2(y, x)`). `asin` and `acos` of a number outside -1 to 1 is a math error.
- `servertimeout` (default 30) is how many seconds a server handler may run. One that runs longer is stopped at its next line and answers 503, and the server goes on; `0` is no limit. A loop with nothing in its body can now be stopped too, by this and by Ctrl+C in the REPL.
- Sets and maps find whole-number keys without turning them into text: sets are about 50% faster and use about 40% less memory.
- `1 << 3` and `8 >> 1` say Turtle has no bitwise operators, as `&`, `|` and `^` already did.
- The note "at round works on v only" no longer appears when brackets already say what the method works on: `2 * (v at round[0])`.
- The shared integers of 0.9.219 are taken back: programs use about 15% less memory overall, and a few (list indexing in long loops, binary search) run 8–10% slower.

## 0.9.219 (2026-10-21)

- Whole numbers up to a million are shared rather than made one at a time: faster arithmetic in some programs, at about 15 MB more memory. (Taken back in 0.9.220.)

## 0.9.218 (2026-10-21)

Breaking changes:

- Integers have no limit, as in Python: arithmetic is exact past 64 bits (`9223372036854775807 + 1` is `9223372036854775808`), a literal can be as long as you like, and `pow` of whole numbers is exact. What used to stop with "integer overflow" now gives the exact answer. Converting a float keeps its whole part exactly: `change 1e19 to integer` is `10000000000000000000`.
- Floats past their range give `inf` or `-inf`, and `inf - inf` gives `nan`, as in Python. A power or math function past the range is an error instead: `2.0 at pow 1100`, `0.0 at pow -1.0`, and a power that gives `nan`.
- `shuffle[x]` as a statement on its own shuffles the list in place (it used to give back a shuffled copy and leave `x` unchanged). `x = shuffle[x]` works as before.

Other changes:

- `shuffle` on a set changes the set; on text or a map, the message says how to write it.
- In `show`, a semicolon starts the next item, so a sentence call's commas stay its arguments: `show 1 add3 2, 3; " " .`.
- Messages name the fix: a set operator or method after a bracket call, a text method used on a number, `change ... to float` on text that isn't a number, and a builtin function used without brackets. A number alone in braces in text (`"{2}"`) gets a note saying how to keep the braces.
- Faster: an assignment looks its name up once, whole-number addition stays on the direct path, and integers and text sort through typed keys.
- Go 1.27.

## 0.9.217 (2026-10-21)

- Tests and CI only; no change to the language.

## 0.9.216 (2026-10-21)

Breaking changes:

- The map method `add` takes the value first, then the key, as the sentence `add value to m at key` does: `m at add[3, "apples"]` adds 3 to `"apples"`. `m at put[value, key]` sets a key. (In 0.9.215 the method was `add[key, value]` and set the key.) A call written in the old order is not refused: it stores the key under the value.

Other changes:

- `min` and `max` take values in brackets: `min[a, b]`, `max[nums]`.
- `pop` takes a position: `pop[0]` removes the first item, `pop[i]` the one at `i`.
- `math` has `sin`, `cos`, `tan`, `exp` and `log` (with an optional base: `1000 at log[10]` is `3.0`).
- `invert` prints a warning when two keys share a value, naming the entry it drops.
- A function that assigns a name also used at top level gets a note that the top-level one is not changed.
- `"straße" at upper` is `"STRASSE"`, as in Python.

## 0.9.215 (2026-10-21)

Breaking changes:

- `add value to map at key` adds to the value there: a new key gets the value (`add 12 to ages at "Cy"` gives `"Cy": 12`), a number adds up (5 becomes 17), text joins on, and a list or set takes the value. Setting a key outright is `put value to map at key`, which now works on maps as it does on lists.
- Floats compare exactly, as in Python, JavaScript and Go: `0.1 + 0.2 == 0.3` is `false`. Floats show the shortest text that reads back as the same float (`0.30000000000000004`). Matrices compare and show exactly too.
- `sort` compares like with like, as Python does: numbers with numbers, text with text, lists element by element. Mixing kinds is an error, not a comparison of their text.

Other changes:

- `pi` and `e` come with `import math` (as `pi` or `math pi`); a variable of the same name wins.
- `reverse` works on text: `"héllo" at reverse` is `"olléh"`.
- `is_close` (math) and `all_close` (linear) check closeness by the rule of `is close to`, with an optional tolerance.
- `add ... to rows at i` changes the item at position `i` of a list of lists.
- Empty collections show as `[]`, `{}`.

## 0.9.214 (2026-10-14)

- `table` shows a map of records with one column per field, after its key.

## 0.9.213 (2026-10-14)

- `table_column[rows, name]` and `table_row[rows, n]` pick one column or one row of a table (records can be assembled values or maps).

## 0.9.212 (2026-10-14)

- A theory phrase can take a function value, so a generator can be a theory word (see the Theories section).

## 0.9.211 (2026-10-14)

- A second top-level `def` with the same name gives a warning naming both lines, in the terminal and in the VS Code editor.

## 0.9.210 (2026-10-14)

- `unique[collection]` gives each value once, and `group[collection, function]` gives a map from each result to its items. Both need `import data`.

## 0.9.208 (2026-10-14)

- Variable reads remember where the variable was found, so a name read
  again from the same place is faster: searching with many variables is
  about 10% quicker.

## 0.9.207 (2026-10-14)

- Scopes with up to 32 names are searched in a list, not a map: programs
  with many top-level variables run about 15% faster on lookups.

## 0.9.206 (2026-10-14)

- A function that assigns a name also used at top level no longer checks
  the assignment on every call: calls to small functions are about 2.3x
  faster.

## 0.9.205 (2026-10-14)

- Library functions called in function form (`abs[x]`, `sqrt[x]`) look up
  their module once, not on every call: those calls are about 2.2x faster.

## 0.9.204 (2026-10-14)

- Assembled values (`assemble Point [x, y]`) are made in one allocation:
  creating records is about 10% faster.

## 0.9.203 (2026-10-14)

- Sorting plain numbers and text is about 30% faster.
- Text with `{...}` values is built once, not piece by piece.

## 0.9.202 (2026-10-14)

- Peak memory is lower: the garbage collector runs a little more often
  when a program keeps a lot of data, cutting peaks by 15–30% on most
  programs, at a small cost in time.

## 0.9.201 (2026-10-14)

- Printed output is buffered: printing many lines is about six times
  faster. Output still appears in order with input prompts, error
  messages, and `sys` commands.

## 0.9.200 (2026-10-14)

- `show` builds its text once: printing many values is about 20% faster.

## 0.9.198 (2026-10-14)

- Source files saved with Windows line endings (CRLF) run the same as
  files with plain line endings; line breaks inside text are one `\n`.

## 0.9.197 (2026-10-14)

- `show` puts one space between its values: `show 3, 2 .` gives `3 2`.
  No space goes next to text that already has whitespace, or before
  punctuation: `show "total: ", n .` gives `total: 3`, and `show name,
  "!" .` gives `Ann!`. Programs that joined values with `show` and
  relied on no space need `+` or a space inside the text.
- A method chain takes one step after another without brackets:
  `grid at get 0 at put 10, 0` and `grid at get[0] at put[10, 0]` both
  work, as statements and in expressions.
- `every` takes `"ms"` and `"ns"` steps, so a job can run every few
  milliseconds: `every[10, "ms", job]`.
- Imports can rename: `import data as d` (the module is called `d`, and
  its names are only reached as `d sum[...]`), and
  `import data [mean as average]` (`average` is `mean` in that file).
- The programs in `testdata/programs` check each area of the language
  (collections, assembled types, files, SQL, matrices, dates, theories
  and more) and run in the test suite.

## 0.9.196 (2026-10-11)

- Arithmetic inside arithmetic (`total + i * i % m`) is worked out
  without making a value for each step in between: loops doing sums are
  about 10% faster, and lists built from them use less memory.

## 0.9.195 (2026-10-11)

- Smaller maps: a 12-field row (a CSV, JSON or database row) takes about
  a quarter of the memory it did, and maps and JSON are 15-20% faster.
- Function calls about 20% faster.
- Names may use any language's letters: `café`, `größe`, `名前`.
- A runtime error shows the line it happened on; clearer messages for an
  extra `]`, a stray `[end]`, and a character Turtle doesn't know.

## 0.9.194 (2026-10-11)

- Anything in brackets can go over several lines: a call's or a method's
  arguments (`f[` with the arguments on the lines after), assembled
  values, as lists, sets, maps and imports already could.
- A comma before the closing `]` is allowed: `list [1, 2,]`, and one item
  per line, each with its comma.

## 0.9.193 (2026-10-11)

- Less memory: programs that keep a lot of data use 15-40% less at
  their peak, at the same speed.

## 0.9.192 (2026-10-10)

- REPL: `colors` opens a picker with the VS Code extension's twelve
  schemes, color for color: Up/Down the scheme (a short program recolors
  as you go), Left/Right the shade (auto, dark, light), Enter keeps it,
  Esc leaves it. `colors okabe-ito` picks one by name. Kept in
  `~/.config/turtle/repl`; `auto` asks the terminal whether its background
  is light or dark. Exact colors on 24-bit terminals, the nearest of 256
  elsewhere.
- Copyright headers in the source files, and a copyright line at the end
  of each documentation page.

## 0.9.191 (2026-10-10)

- `import system [warn]`: `warn` can be listed with system's other names.
- `d at tostring` writes a date as text, as `show` does.
- `turtle lsp`: the standard library's own Turtle files (evaluator/lib/)
  read as Turtle reads them, so their own sentences aren't marked as errors.

## 0.9.190 (2026-10-10)

- Colors grouped by what a word does: keywords, imports, functions, data
  structures, `show`, text, numbers and theories (a theory's word wherever
  it's used); where a function or type is made, bold. `turtle lsp` sends
  the groups to editors that take them, and the REPL uses them.
- VS Code extension 0.1.7: twelve color schemes (Turtle Color Scheme in
  the settings), five of them for color vision, and `turtle.colors` for
  your own color for a group.

## 0.9.189 (2026-10-10)

- REPL: a theory typed a line at a time waits for `theory [end]` (it ran
  `theory name` on its own line and gave an error).

## 0.9.188 (2026-10-10)

- Theories: each random input runs the theory once, with every theorem
  checked on that result (it ran once more per theorem: three times as
  often for two theorems).
- `reportfile` set inside a function counts for `diagnose[...]` there.
- `pattern`: patterns made as a program goes no longer pile up in memory
  (at most 1,000 are kept compiled).
- `replaceall`: a `$` that names no group stays as written (`"$10"`), and
  a group number ends where its digits do (`$1x` is group 1, then `x`).
  Both gave empty text before; `$1`, `${1}`, `$name` and `$$` work as before.
- Docs: `\w` is English letters only (`\p{L}` for any language); write
  replacements in backticks; `groups` gives `""` for a group that took no
  part.

## 0.9.187 (2026-10-10)

- Theories: random inputs try edges of text, lists, sets and maps too
  (empty ones, and ones holding a single edge), not only numbers'.
- `turtle test` says when a theory's theorems could only be checked on its
  proof cases.

## 0.9.186 (2026-10-10)

- A function's own values are freed when it returns. A finished call used
  to keep them until the next call, so a big list made inside a function
  stayed in memory.
- `system`: `memory[]`, `sizeof[x]` and `freememory[]`.
- `diagnose`, `hypothesis` and `turtle test` show the memory each step or
  test allocated.
- `import test`: `memorylimit = 5000000` fails a test that allocates more.
- Loops over `range[...]` count without making the list first, and a big
  list grows by doubling: building one by `at add` allocates about half
  as much, and runs faster.

## 0.9.185 (2026-10-10)

- `!w type string` is "w isn't text": `!` takes the whole type check (it
  was `(!w) type string`, always false). `s at upper type string` works.
- `hypothesis[...]` isn't written in a theory; it checks theories from
  outside them.

## 0.9.184 (2026-10-10)

- Theories on sets, and on maps with number keys, are proved again (0.9.183
  could stop with "wanted 5 different values").

## 0.9.183 (2026-10-10)

- `hypothesis[...]` tries a claim about data, a function, or a theory and
  gives true or false; on its own line it shows where the claim breaks.
- `reportfile = "checks.txt"` also adds diagnose and hypothesis reports to
  a file.
- Theories: half the random inputs stay within the proof cases' range,
  starting at its edges and 0, so a theory that refuses most numbers is
  still tried on many; reports say how many inputs it refused.

## 0.9.182 (2026-10-10)

- Private theories: `theory ~name` is for its own file (and its tests).
- The standard library can have theories, giving any library new phrases.
- `of` and `is` can be a notation's words: `notation average of xs to places .`
- Theories: imports are as fast as before theories; a theory's random
  numbers reach past its proof cases' largest and include negatives;
  diagnose previews big values without copying them.
- An error inside a library's theory names it: `celsius (units library): ...`
- A built program can use theories from its own imported files.

## 0.9.181 (2026-10-10)

- Theories: new words and phrases written in Turtle, with an abstract,
  notations, a definition, theorems and a proof. `turtle test` proves
  them; `diagnose` and `help` look inside them.

## 0.9.180 (2026-10-10)

- Private functions: a function named with `~` (`def ~limit[...]`) can
  be used only in its own file, and by test files, which can test both
  public and private functions.
- The standard library can be written in Turtle: a file in
  `evaluator/lib/` adds to a library or makes a new one, documented by its
  comments.
- `turtle doc` no longer takes a block comment above a file's first
  function for the file's description.

## 0.9.179 (2026-10-09)

- `[read]`, `[write]`, `[append]` and `[directory]` take any expression
  for the path: `[read] folder + "/" + name + ".txt" to lines [end]`.
- `make_date` also takes year, month, day, hour and minute.
- `diagnose` shows each step's value as it was then, with what's inside.
- A matrix joined into text keeps its rows lined up: `show "m is " + m .`
- A matrix's columns line up on their decimal points.
- `commas[2]`: commas and exactly 2 decimals, `1250.5 at commas[2]` is `"1,250.50"`.
- `table` right-aligns numbers written as text, like `"$1,250.50"` and `"12%"`.
- A failed step in a saved scroll says which step it sits in.
- `turtle fmt` keeps maps and lists written over several lines indented,
  and a one-line scroll inside brackets no longer indents what follows.

## 0.9.178 (2026-10-09)

- Linux: downloads for ARM too, and a `.tar.gz` for any Linux. The `.deb`
  installs `turtle` in `/usr/bin` (it was `/usr/local/bin`).

## 0.9.177 (2026-10-09)

- VS Code: a ▶ button (and Ctrl-F5) runs the open file in a terminal.

## 0.9.176 (2026-10-09)

- Input without a prompt, `a = ?`, and a closing period: `a = ? "n: " .`

## 0.9.175 (2026-10-09)

- macOS: `turtle.pkg` runs on Intel Macs too, gives `.turtle` files the
  Turtle icon, and runs them on a double-click.

## 0.9.174 (2026-10-09)

### Breaking

- `count = count + 1` in a function, where `count` is a global, is an
  error. It only ever changed a local copy; the message says what to do.

## 0.9.173 (2026-10-09)

- Windows: a double-clicked `.turtle` file's window closes when it ends.

## 0.9.172 (2026-10-09)

- The `linear` docs show every function with an example and its output.
- Releases keep the newest 5 downloads.

## 0.9.171 (2026-10-09)

### New

- `help`: `help[m]` says what a value is and what it can do;
  `help["linear"]` a library; `help["reshape"]` a function; `help["if"]`
  a keyword; `help["keywords"]` all of them. In the REPL: `help m`,
  `help linear`, `help if`.
- `stdlib[]`: every library and its functions, as a map.
- `version`: the version of Turtle running.

## 0.9.170 (2026-10-09)

### Breaking

- On Windows, `sys` runs through `cmd`, as `runall` does.
- Bad JSON is described the same way on every Go version; some error
  columns moved.

### Fixed

- Windows: double quotes in a `sys` or `runall` command reach `cmd` as
  written.
- Windows: `turtle build` over a program that's still running says so,
  and leaves no half-made file behind.

### New

- `flatten` and `reshape` in `linear`.
- A Windows installer with each release, `turtle-windows-amd64-setup.exe`:
  Turtle in Program Files and on the PATH, `.turtle` files that run on a
  double-click, and a Start menu entry.
