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
go run ./cmd/turtle path/to/script.trt
```

## Run a script

```sh
./turtle path/to/script.trt
```

Anything after the script path is passed to the script, which reads it
with `import system` and `args[]` (see
[the `system` docs](library/system.md)):

```sh
./turtle report.trt data.txt --verbose
```

File paths in a script (`[read] data.txt ...`) are relative to the folder
you run `turtle` from, like any command-line tool. `import` always looks
next to the script.

Run `turtle` with no file for the interactive prompt (the REPL), with
the code colored as you type ([`repl.md`](repl.md)):

```sh
./turtle
```

```
>>> 1 + 2
3
```

## Your first script

Create `hello.trt`:

```
name = "World"
show "Hello, ", name, "!" .
```

Run it:

```sh
./turtle hello.trt
```

```
Hello, World!
```

## Test it

Put tests in a file whose name starts with `test_`, with `import test`
and functions whose names start with `test_`. `test_hello.trt`:

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
test_hello.trt
  PASS  test_greeting   12.0µs

ok: 1 passed, 0 failed (1 file, 1.3ms)
```

See [`testing.md`](testing.md) for `check`, `verify`, `validate`, suites
and benchmarks.

To see a program run line by line, with each variable's new value, use
`turtle trace` (the trace goes to standard error):

```sh
./turtle trace hello.trt
```

To stop at each line and look around (values, variables, where you are),
use `turtle debug hello.trt`: Enter steps, `p name` shows a value, `h`
lists the commands. `turtle fmt` lays your files out the standard way.

## Share it as a program

`turtle build` makes one file that runs without Turtle installed, for the
system you're on (a `.exe` on Windows):

```sh
./turtle build hello.trt
./hello
```

Your own imported `.trt` files are packed in; data files stay beside it.
See [Sharing a program](reference.md#sharing-a-program-turtle-build).

## Where to go next

- [`tour.md`](tour.md) — a guided walkthrough of the language, by example
- [`reference.md`](reference.md) — the complete, formal syntax reference; every keyword is in its [Keywords](reference.md#keywords) table
- [`library/`](library/index.md) — the library, one page per library, plus built-in types, files and modules
- [`testing.md`](testing.md) — tests: `check`, `verify`, `validate`, `turtle test`
- [`repl.md`](repl.md) — the interactive prompt: colors, keys, commands
- [`editors.md`](editors.md) — Turtle in VS Code and other editors
- [`architecture.md`](architecture.md) — how the interpreter itself works,
  for anyone modifying it
- [`contributing.md`](contributing.md) — how to add new syntax or stdlib
  functions

## Try the existing examples

The `testdata/` directory has working `.trt` scripts exercising most of the
language — functions, recursion, nested if/else, nested loops, data
structures, file I/O:

```sh
./turtle testdata/play.trt
./turtle testdata/datastruct.trt
```

Two bigger programs use nearly everything together and check their own
answers; run them from their folders:

```sh
cd testdata/bookshop && ../../turtle bookshop.trt   # every library, ending with a web API
cd testdata/linear && ../../turtle housing.trt      # statistics and matrices on housing data
```
