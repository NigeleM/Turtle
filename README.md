# The Turtle Programming Language ⓒ 2017

Turtle was my side project during community college and undergrad.
It is copyrighted by Nigele McCoy. My goal was to make a very simple language with a paradigm shift.
Philosophy of Turtle coming soon. 


## Status

The interpreter has been rewritten in Go as a proper lexer → parser → AST
→ tree-walking evaluator (`token/`, `lexer/`, `ast/`, `parser/`,
`object/`, `evaluator/`, `cmd/turtle/`), replacing the original
`Turtle_interpreter.go`, which parsed source by re-scanning raw strings.
Surface syntax is unchanged and verified against real historical example
scripts (see `SPEC.md` and `docs/architecture.md` for exactly what changed
under the hood and why). The original interpreter, its `Files/` package
and the old `test.trt` example live in [`legacy/`](legacy/README.md), kept
for reference; they still build and run.

| Feature | Status |
|---|---|
| Variables | Done |
| Conditional statements | Done |
| Loops | Done |
| Functions (can read globals; local assignment never writes through) | Done |
| First-class functions + closures (lexical scope, read-only capture) | Done |
| `none` value/type | Done |
| Anonymous functions (`x give x + 1`) + sentence-style calls (`nums process f`) | Done |
| For-each loops (`[loop][x in nums]`) | Done |
| `+`/`-` on lists, sets, maps | Done |
| Standard library: data (`process`, `keep`, `copy`, `range`, `reduce`, `sum`, `table`, `table_read`, `table_write`; `.csv`, `.tsv`, `.txt`, `.json` table files; statistics: `mean`, `median`, `stdev`, `percentile`, `correlation`, `describe`, ...) | Done |
| Standard library: linear (`matrix [1, 2; 3, 4]`, `*` as the matrix product, `solve`, `inverse`, `determinant`, `least_squares`, `lu`, `qr`, `eigen`, `svd`) | Done |
| Standard library: system (`args`, `options`, `exit`, `env`, `loadenv`, `scriptfolder`, `contents`, `walk`, `exists`/`isfile`/`isfolder`, `copyto`, `moveto`, `makefolder`, `erase`, `pack`/`unpack` for .zip/.tar/.tar.gz) | Done |
| `turtle trace`: each line as it runs, with the values it sets | Done |
| `turtle debug`: step, breakpoints, look at and change values | Done |
| `turtle fmt` (and Format Document in editors) | Done |
| `turtle build`: one program file that runs without Turtle (Mac, Windows, Linux) | Done |
| Benchmark against Python, Node, Ruby and Go (`bench/`) | Done — see `bench/RESULTS.md` |
| Parse errors: the first one, with a `^` under the spot and how to fix it | Done |
| Standard library: strings (`find`, `substring`, `isinstring`, `join`) | Done |
| Assembled types (`assemble Order [item, qty]`, `qty of o`) | Done |
| Scrolls (`x is scroll 3 into add1, double .`), `here`, `diagnose` | Done |
| Data structures (list/set/map) | Done |
| Type conversion (`change`) | Done |
| Standard library: math (`import math`) | Done |
| Standard library: time (`import time`) | Done |
| Standard library: sort (`min_sort`, `max_sort`, the classic sorts) and search (`find_first`, `find_all`, binary search, ...) | Done |
| Standard library: random (`random list of 5 integers from 0 to 9`, any shape, `pick`, `shuffle`, `sample`, `chance`, `seed`) | Done |
| Standard library: log (`log warn "disk at ", pct, "%" .`, levels, files, rotation, JSON lines, a copy of the console) | Done |
| Standard library: crypt (`hash`, `hmac`, `encode`/`decode`, `uuid`, `token`, `passwordhash`/`passwordcheck`, `encrypt`/`decrypt`) | Done |
| Standard library: server (`serve[app, 8080]`: routes in a map, handlers as functions, `reply`, `redirect`, static files) | Done |
| Standard library: config (`config_read`/`config_write`: .toml, .json, .env) and time zones (`today["Asia/Tokyo"]`, `to_zone`) | Done |
| Standard library: schedule (`fetchall`, `runall`, `queryall`: many at once, `schedulelimit` at a time) | Done |
| Standard library: pattern (`matches`, `findall`, `replaceall`, `splitby`, `groups`; backtick strings) | Done |
| Number and text formatting (`fixed`, `commas`, `padleft`, `padright`), `typeof[x]`, `x type integer`, scientific notation (`1e-9`) | Done |
| Standard library: JSON (`import json`) | Done |
| Standard library: HTTP (`import http`) | Done |
| Standard library: SQL — SQLite read and write, joins, groups, window functions, JSON, triggers, transactions, CSV in and out; PostgreSQL and MySQL through the same functions (`import sql`) | Done |
| Imports (per-module scope, `import m [a, b]`, `m name[...]` on clash) | Done |
| File management (read/write/append/directory) | Done |
| Testing in Turtle: `check`, `verify`, `validate` (random inputs, shrinking), `turtle test` with suites and benchmarks | Done — see `docs/testing.md` |
| REPL: `turtle` with no file, colored as you type, history, multi-line blocks | Done — see `docs/repl.md` |
| Editors: `turtle lsp` language server (errors, colors, completion, hover, go to definition) and a VS Code extension | Done — see `docs/editors.md` |
| Automated tests (`go test ./...`) | Done |
| CI (build/vet/test on every push) + tag-triggered releases | Done |
| Documentation | Done — see `docs/` |
| `.turtle` file extension for programs, libraries, imports and tests (`.trt`, the earlier one, still works) | Done |

## Quick start

```sh
go build -o turtle ./cmd/turtle
./turtle path/to/script.turtle
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

- [`docs/getting-started.md`](docs/getting-started.md) — build and run your first script
- [`docs/tour.md`](docs/tour.md) — a guided, example-driven walkthrough of the language
- [`docs/reference.md`](docs/reference.md) — the complete formal syntax reference, with every keyword in one [table](docs/reference.md#keywords)
- [`docs/library/`](docs/library/index.md) — the library, one page per library (data, linear, sql, server, ...), plus built-in types, files and modules
- [`docs/editors.md`](docs/editors.md) — VS Code, Neovim, Helix and others, through `turtle lsp`
- [`docs/repl.md`](docs/repl.md) — the interactive prompt (`turtle` with no file)
- [`docs/testing.md`](docs/testing.md) — testing Turtle code: `check`, `verify`, `validate`, `turtle test`
- [`docs/architecture.md`](docs/architecture.md) — how the interpreter itself is built, for contributors
- [`docs/contributing.md`](docs/contributing.md) — how to add new syntax or standard-library functions
- [`docs/stress-test-log.md`](docs/stress-test-log.md) — what the stress tests tried, what broke, and how each finding was handled
- [`legacy/README.md`](legacy/README.md) — the original 2017 interpreter, kept for reference
- [`SPEC.md`](SPEC.md) — the language specification this rewrite was built from, including every deliberate deviation from the original interpreter's behavior and why
- [`CHANGELOG.md`](CHANGELOG.md) — what changed in each release, breaking changes first (Turtle is pre-release, 0.x)
- [`PROGRESS.md`](PROGRESS.md) — session-by-session log of what's been done and what's left
- [`TODO.md`](TODO.md) — forward-looking punch list of what's next

## License

Turtle is created by **Nigele McCoy** and licensed under the
[Apache License 2.0](LICENSE). In plain words (the license is what
counts):

- **Use Turtle for anything**, including at work and in commercial
  projects, and include it in your own builds, devices and products.
- **Your programs are yours.** What you write in Turtle, including
  programs made with `turtle build`, you own, and may sell under any
  terms.
- **Change Turtle and share it**, keeping the [`LICENSE`](LICENSE) and
  [`NOTICE`](NOTICE) files, which credit Turtle's creator.
- **The name and logo identify the official Turtle.** The license
  doesn't grant them: a changed version needs its own name, and products
  named after Turtle need permission; see [`BRAND.md`](BRAND.md).
- **Contributions to Turtle itself** (interpreter, standard library,
  tools, docs) stay their authors' but are licensed to the project for
  good, and what stays in Turtle is the creator's call; see
  [`docs/contributing.md`](docs/contributing.md#contributor-terms-who-owns-a-contribution).

Versions before 0.9.168 carried an earlier license file.

### Permissions and licensing

For a use of the Turtle name or logo that needs permission, or anything
else the license doesn't cover, open an issue at
https://github.com/NigeleM/Turtle and say what you'd like to do.

## Credits

Written by Nigele McCoy, Claude-assisted.
