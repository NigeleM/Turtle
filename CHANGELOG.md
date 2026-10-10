# Changelog

Turtle is pre-release (0.x). Breaking changes are listed first.

## Unreleased

- Private theories: `theory ~name` is for its own file (and its tests).
- The standard library can have theories, giving any library new phrases.
- `of` and `is` can be a notation's words: `notation average of xs to places .`
- Theories: imports are as fast as before theories; a theory's random
  numbers reach past its proof cases' largest; diagnose previews big
  values without copying them.

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
