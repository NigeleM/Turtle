# The Turtle Programming Language ⓒ 2017

Turtle was my side project during community college and undergrad.
It is copyrighted by me. My goal was to make a very simple language.

## Status

The interpreter has been rewritten in Go as a proper lexer → parser → AST
→ tree-walking evaluator (`token/`, `lexer/`, `ast/`, `parser/`,
`object/`, `evaluator/`, `cmd/turtle/`), replacing the original
`Turtle_interpreter.go`, which parsed source by re-scanning raw strings.
Surface syntax is unchanged and verified against real historical example
scripts (see `SPEC.md` and `docs/architecture.md` for exactly what changed
under the hood and why). `Turtle_interpreter.go` and `Files/` are kept
around unmodified for reference.

| Feature | Status |
|---|---|
| Variables | Done |
| Conditional statements | Done |
| Loops | Done |
| Functions (can read globals; local assignment never writes through) | Done |
| First-class functions + closures (lexical scope, read-only capture) | Done |
| `none` value/type | Done |
| Anonymous functions (`x gives x + 1`) + sentence-style calls (`nums process f`) | Done |
| For-each loops (`[loop][x in nums]`) | Done |
| `+`/`-` on lists, sets, maps | Done |
| Standard library: data (`process`, `keep`, `copy`) | Done |
| Standard library: system (`args`, `exit`, `env`, `scriptFolder`, `contents`, `exists`/`isFile`/`isFolder`) | Done |
| Standard library: strings (`find`, `substring`, `isinstring`, `join`) | Done |
| Assembled types (`assemble Order [item, qty]`, `qty of o`) | Done |
| Data structures (list/set/map) | Done |
| Type conversion (`change`) | Done |
| Standard library: strings | Done |
| Standard library: math (`import math`) | Done |
| Standard library: time (`import time`) | Done |
| Standard library: HTTP | Not started |
| Imports (per-module scope, `import m [a, b]`, `m name[...]` on clash) | Done |
| File management (read/write/append/directory) | Done |
| Automated tests (`go test ./...`) | Done |
| CI (build/vet/test on every push) + tag-triggered releases | Done |
| Documentation | Done — see `docs/` |
| `.trt`-only file extension enforcement | Not started |

## Quick start

```sh
go build -o turtle ./cmd/turtle
./turtle path/to/script.t
```

See [`docs/getting-started.md`](docs/getting-started.md) for more.

## A taste of Turtle

```
import time [now, sleep]
import mylib [binary]          // only what you list; mylib.t's variables stay private

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

Working with data: `gives` makes a quick function, and library functions
read like sentences:

```
import data
import system [args, exists]

assemble Order [item, qty, price]    // your own type with named fields

orders = list [Order["pen", 3, 1.5], Order["pad", 1, 4.0], Order["mug", 5, 8.0]]
orders keep o gives qty of o > 2 .   // a filter: keeps pen and mug

total = 0
[loop][o in orders]                  // for-each
    total = total + qty of o * price of o
[loop][end]
show total .                         // 44.5

names = list ["ana", "bo"]
names process n gives n at upper .   // [ "ANA", "BO" ]
show names + list ["CY"] .           // [ "ANA", "BO", "CY" ]

[loop][path in args[]]               // turtle report.t notes.txt ...
    if ] path exists [
        show path, " found" .
    if [end]
[loop][end]
```

Full rules: [`docs/reference.md`](docs/reference.md) (Functions, Closures,
Anonymous functions, Sentence-style calls, None, Assembled types, Loops,
Modules) and [`docs/stdlib.md`](docs/stdlib.md) (Data library, System
library).

## Documentation

- [`docs/getting-started.md`](docs/getting-started.md) — build and run your first script
- [`docs/tour.md`](docs/tour.md) — a guided, example-driven walkthrough of the language
- [`docs/reference.md`](docs/reference.md) — the complete formal syntax reference
- [`docs/stdlib.md`](docs/stdlib.md) — data structure methods, file I/O, `sys`, `import`
- [`docs/architecture.md`](docs/architecture.md) — how the interpreter itself is built, for contributors
- [`docs/contributing.md`](docs/contributing.md) — how to add new syntax or standard-library functions
- [`docs/stress-test-log.md`](docs/stress-test-log.md) — what the stress tests tried, what broke, and how each finding was handled
- [`SPEC.md`](SPEC.md) — the language specification this rewrite was built from, including every deliberate deviation from the original interpreter's behavior and why
- [`PROGRESS.md`](PROGRESS.md) — session-by-session log of what's been done and what's left
- [`TODO.md`](TODO.md) — forward-looking punch list of what's next
