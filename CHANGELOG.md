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

### New

- `flatten` and `reshape` in `linear`.
