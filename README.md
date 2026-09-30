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
| Data structures (list/set/map) | Done |
| Type conversion (`change`) | Done |
| Standard library: strings | Done |
| Standard library: math (`import math`) | Done |
| Standard library: time (`import time`) | Done |
| Standard library: HTTP | Not started |
| Imports | Done |
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

## Documentation

- [`docs/getting-started.md`](docs/getting-started.md) — build and run your first script
- [`docs/tour.md`](docs/tour.md) — a guided, example-driven walkthrough of the language
- [`docs/reference.md`](docs/reference.md) — the complete formal syntax reference
- [`docs/stdlib.md`](docs/stdlib.md) — data structure methods, file I/O, `sys`, `import`
- [`docs/architecture.md`](docs/architecture.md) — how the interpreter itself is built, for contributors
- [`docs/contributing.md`](docs/contributing.md) — how to add new syntax or standard-library functions
- [`SPEC.md`](SPEC.md) — the language specification this rewrite was built from, including every deliberate deviation from the original interpreter's behavior and why
- [`PROGRESS.md`](PROGRESS.md) — session-by-session log of what's been done and what's left
- [`TODO.md`](TODO.md) — forward-looking punch list of what's next
