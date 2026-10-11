# The Turtle Programming Language ⓒ 2017

Turtle is copyrighted by Nigele McCoy. My goal was to make a very simple language with a paradigm shift.

## Status

Turtle 0.9 is usable today: the full language, a standard library (data,
linear algebra, SQL, HTTP, servers, JSON, files and more), a REPL, a
formatter, a debugger, editor support, and `turtle build` for single-file
programs on Mac, Windows and Linux.

## Quick start

Install from a [release](https://github.com/NigeleM/Turtle/releases)
(Mac `.pkg`, Windows setup, Linux `.deb`), or build it:

```sh
go build -o turtle ./cmd/turtle
./turtle path/to/script.turtle
```

On Windows (PowerShell):

```powershell
go build -o turtle.exe ./cmd/turtle
.\turtle.exe path\to\script.turtle
```

See [`docs/getting-started.md`](docs/getting-started.md) for more.

## A taste of Turtle

```
import time [now, sleep]
import mylib [binary]          // only what you list; mylib.turtle's variables stay private

def make_adder[n]              // functions are values; a nested def is a closure
    def adder[x]
        return x + n
    def [end]
    return adder
def [end]

add5 = make_adder[5]
show add5[1] .                 // 6

def log[msg]
    show msg .                 // no return, so the call yields none
def [end]

r = log["hi"]
show r == none .               // true

t = time now[]                 // naming the module always works, and is required
                               // when two imports both provide "now"
```

Working with data: `give` makes a quick function, and library functions
read like sentences:

```
import data
import system [args, exists]

assemble Order [item, qty, price]    // your own type with named fields

orders = list [Order["pen", 3, 1.5], Order["pad", 1, 4.0], Order["mug", 5, 8.0]]
orders keep o give qty of o > 2 .   // a filter: keeps pen and mug

total = 0
[loop][o in orders]                  // for-each
    total = total + qty of o * price of o
[loop][end]
show total .                         // 44.5

names = list ["ana", "bo"]
names process n give n at upper .   // [ "ANA", "BO" ]
show names + list ["CY"] .           // [ "ANA", "BO", "CY" ]

[loop][path in args[]]               // turtle report.turtle notes.txt ...
    if ] path exists [
        show path, " found" .
    if [end]
[loop][end]
```

Full rules: [`docs/reference.md`](docs/reference.md) (Functions, Closures,
Anonymous functions, Sentence-style calls, None, Assembled types, Loops,
Modules) and [the library docs](docs/library/index.md) (Data library, System
library).

## Documentation

- [Getting started](docs/getting-started.md): build and run your first program
- [Tour](docs/tour.md): the language, by example
- [Reference](docs/reference.md): the complete syntax, with every [keyword](docs/reference.md#keywords)
- [Library](docs/library/index.md): one page per library (data, linear, sql, server, ...)
- [REPL](docs/repl.md): the interactive prompt (`turtle` with no file)
- [Testing](docs/testing.md): `check`, `verify`, `validate`, `turtle test`
- [Editors](docs/editors.md): VS Code, Neovim, Helix and others
- [Contributing](docs/contributing.md) and [architecture](docs/architecture.md): for working on Turtle itself
- [Changelog](CHANGELOG.md)

## License

Turtle was created by **Nigele McCoy**, who holds its copyright. It is
licensed under the [Apache License 2.0](LICENSE); the license is what
counts, but in short:

- **Use it for anything**, including commercial work.
- **Change it and share it**, keeping the [`LICENSE`](LICENSE) and
  [`NOTICE`](NOTICE) files.
- **Programs you write in Turtle are yours.**
- **No warranty.** Turtle is provided as is; its authors aren't liable for
  any damage or loss from using it (sections 7 and 8 of the license).
- **Contributions** are made under the same license, and what goes into
  official Turtle is the creator's decision.
