# Getting started

## Install on Windows

Run `turtle-windows-amd64-setup.exe` from a
[release](https://github.com/NigeleM/Turtle/releases). It puts `turtle` on
the PATH; double-clicking a `.turtle` file runs it in its own folder. It
isn't signed yet: if Windows says it "protected your PC", choose More
info, then Run anyway.

## Install on macOS

Open `turtle.pkg` from a
[release](https://github.com/NigeleM/Turtle/releases). It works on Apple
silicon and Intel Macs, puts `turtle` in `/usr/local/bin`, and adds
Turtle to Applications: double-clicking a `.turtle` file runs it in
Terminal, in its own folder, and opening Turtle itself starts the prompt.
It isn't signed yet: if macOS won't open it, right-click it, choose Open,
then Open again. To remove it: `sudo /usr/local/share/turtle/uninstall.sh`.

## Install on Linux

From a [release](https://github.com/NigeleM/Turtle/releases), for Intel or
AMD (`amd64`) or ARM (`arm64`, such as a Raspberry Pi):

- Debian or Ubuntu: `sudo apt install ./turtle-linux-amd64.deb`
- any Linux: unpack `turtle-linux-amd64.tar.gz` and put `turtle` on your
  PATH, for example in `~/.local/bin`

`uname -m` says which you have: `x86_64` is amd64, `aarch64` is arm64.

## Requirements

Go 1.24 or later. No C toolchain needed: Turtle is pure Go.

## Build

From the repository root:

```sh
go build -o turtle ./cmd/turtle
```

This produces a `turtle` binary in the current directory. On Windows,
name it `turtle.exe` (Go doesn't add the `.exe` for you), and run it as
`.\turtle.exe` wherever this page says `./turtle`:

```powershell
go build -o turtle.exe ./cmd/turtle
.\turtle.exe path\to\script.turtle
```

You can also skip the build step for one-off runs:

```sh
go run ./cmd/turtle path/to/script.turtle
```

## Run a script

```sh
./turtle path/to/script.turtle
```

Anything after the script path is passed to the script, which reads it
with `import system` and `args[]` (see
[the `system` docs](library/system.md)):

```sh
./turtle report.turtle data.txt --verbose
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

Create `hello.turtle`:

```
name = "World"
show "Hello, ", name, "!" .
```

Run it:

```sh
./turtle hello.turtle
```

```
Hello, World!
```

## Test it

Put tests in a file whose name starts with `test_`, with `import test`
and functions whose names start with `test_`. `test_hello.turtle`:

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
test_hello.turtle
  PASS  test_greeting   12.0µs

ok: 1 passed, 0 failed (1 file, 1.3ms)
```

See [`testing.md`](testing.md) for `check`, `verify`, `validate`, suites
and benchmarks.

To see a program run line by line, with each variable's new value, use
`turtle trace` (the trace goes to standard error):

```sh
./turtle trace hello.turtle
```

To stop at each line and look around (values, variables, where you are),
use `turtle debug hello.turtle`: Enter steps, `p name` shows a value, `h`
lists the commands. `turtle fmt` lays your files out the standard way.

## Share it as a program

`turtle build` makes one file that runs without Turtle installed, for the
system you're on (a `.exe` on Windows):

```sh
./turtle build hello.turtle
./hello
```

Your own imported `.turtle` files are packed in; data files stay beside it.
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

The `testdata/` directory has working `.turtle` scripts exercising most of the
language — functions, recursion, nested if/else, nested loops, data
structures, file I/O:

```sh
./turtle testdata/play.turtle
./turtle testdata/datastruct.turtle
```

Two bigger programs use nearly everything together and check their own
answers; run them from their folders:

```sh
cd testdata/bookshop && ../../turtle bookshop.turtle   # every library, ending with a web API
cd testdata/linear && ../../turtle housing.turtle      # statistics and matrices on housing data
```

In Windows PowerShell:

```powershell
cd testdata\bookshop; ..\..\turtle.exe bookshop.turtle
cd testdata\linear; ..\..\turtle.exe housing.turtle
```

(The bookshop program ends by calling its own web API with a Unix shell
command, which `cmd` on Windows doesn't run, so that last part is for
macOS and Linux.)
