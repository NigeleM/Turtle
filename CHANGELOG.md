# Changelog

Turtle is pre-release (0.x). Breaking changes are listed first.

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
