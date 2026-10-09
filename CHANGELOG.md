# Changelog

Turtle is pre-release (0.x). Breaking changes are listed first.

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
