# Changelog

Turtle is pre-release (0.x). Breaking changes are listed first.

## Unreleased

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
