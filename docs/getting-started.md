# Getting started

## Requirements

Go 1.24 or later. No C toolchain needed — unlike the legacy
`legacy/Turtle_interpreter.go`, the current interpreter is pure Go (no cgo).

## Build

From the repository root:

```sh
go build -o turtle ./cmd/turtle
```

This produces a `turtle` binary in the current directory. You can also skip
the build step for one-off runs:

```sh
go run ./cmd/turtle path/to/script.t
```

## Run a script

```sh
./turtle path/to/script.t
```

Anything after the script path is passed to the script, which reads it
with `import system` and `args[]` (see
[`stdlib.md`](stdlib.md#system-library)):

```sh
./turtle report.t data.txt --verbose
```

File paths in a script (`[read] data.txt ...`) are relative to the folder
you run `turtle` from, like any command-line tool. `import` always looks
next to the script.

If you omit the path, `turtle` looks for the most recently modified `.t` or
`.T` file in the current directory and runs that instead:

```sh
./turtle
```

## Your first script

Create `hello.t`:

```
name = "World"
show "Hello, ", name, "!" .
```

Run it:

```sh
./turtle hello.t
```

```
Hello, World!
```

## Test it

Put tests in a file whose name starts with `test_`, with `import test`
and functions whose names start with `test_`. `test_hello.t`:

```
import test

def test_greeting[]
    name = "World"
    check "Hello, " + name == "Hello, World" .
def [end]
```

```sh
./turtle test
```

```
test_hello.t
  PASS  test_greeting   12.0µs

ok: 1 passed, 0 failed (1 file, 1.3ms)
```

See [`testing.md`](testing.md) for `check`, `verify`, `validate`, suites
and benchmarks.

## Where to go next

- [`tour.md`](tour.md) — a guided walkthrough of the language, by example
- [`reference.md`](reference.md) — the complete, formal syntax reference
- [`stdlib.md`](stdlib.md) — data structure methods, file I/O, `sys`, `import`
- [`testing.md`](testing.md) — tests: `check`, `verify`, `validate`, `turtle test`
- [`architecture.md`](architecture.md) — how the interpreter itself works,
  for anyone modifying it
- [`contributing.md`](contributing.md) — how to add new syntax or stdlib
  functions

## Try the existing examples

The `testdata/` directory has working `.t` scripts exercising most of the
language — functions, recursion, nested if/else, nested loops, data
structures, file I/O:

```sh
./turtle testdata/play.t
./turtle testdata/datastruct.t
```
