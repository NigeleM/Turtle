# TODO

Forward-looking punch list. For what's already done, see `PROGRESS.md`.

## 1. Expand the standard library

Done so far: string methods, `strings` (`find`, `substring`, `isinstring`,
`join`), `math`, `time` (`now`, `sleep`, dates: `today`, `make_date`,
`to_date`, `add_time`, `time_between`, `format_date`, `wait_until`,
`every`), `data` (`process`, `keep`, `copy`), `system` (`args`, `exit`,
`env`, `scriptfolder`, `contents`, `exists`, `isfile`, `isfolder`,
`erase`, `warn`), `json` (`load`, `json_text`, `json_read`, `json_write`,
`json_get`), `http` (`http_get`, `http_post`, `http_request`), `sql`
(`sql_open`, `sql_create`, `sql_query`, `sql_run`, `sql_tables`,
`sql_save`, `sql_load`, `sql_update`, `sql_delete`, `sql_upsert`,
`sql_close`), `data`'s `table`, `table_read`, `table_write` (`.csv`,
`.tsv`, `.txt`, `.json`), `sort` (`min_sort`, `max_sort`, the classic
sorts) and `search` (`find_first`, `find_all`, binary search, ...).
See `docs/stdlib.md`.

Next, roughly in priority order:

- [x] **Error handling** — `safe` / `handle [kinds] e .` / `fail`
      (2026-10-03). See `docs/reference.md` §Errors.
- [x] JSON: `import json` (2026-10-03)
- [x] Dates in `time`: arithmetic, formatting, `wait_until`, `every`
      (2026-10-03)
- [x] HTTP: `http_get`, `http_post`, `http_request` (2026-10-03)
- [x] `sql` library, step 1: reading real SQLite files (`sql_open`,
      `sql_query`, `sql_tables`, `sql_close`), written from scratch
      (2026-10-03)
- [x] `sql` step 2: writing SQLite files (INSERT/UPDATE/DELETE/CREATE
      TABLE, B-tree splits, rollback journal, new files) (2026-10-05)
- [x] `sql` step 3: indexes kept up to date, GROUP BY, joins, subqueries,
      WITH, UNION, views, ALTER TABLE, transactions, file locks, date and
      printf functions (2026-10-05)
- [x] `sql` step 3b: triggers, window functions, indexes on expressions,
      JSON functions, savepoints, foreign keys, PRAGMAs, VACUUM, ATTACH,
      WAL, UPDATE ... FROM (2026-10-06)
- [x] `sql` step 4: PostgreSQL and MySQL drivers behind the same
      functions (wire protocols, SCRAM / caching_sha2 logins, TLS) (2026-10-06)
- [x] Column types for table files: `table_read` / `sql_load` take a
      map of column to type and `"primary_key"`; `sql_load` makes a
      missing table (2026-10-06)
- [x] `import random`: random values of any shape, pick / shuffle /
      sample / chance, seed (2026-10-06)
- [x] Test library: `check` / `verify` / `validate`, `turtle test`,
      suites, benchmarks (2026-10-06)
- [x] `/` is exact, `div` keeps the whole part; methods work on the
      value right before them (2026-10-06)
- [x] `put` (set the item at a position); a general speed test
      (testdata/speed) (2026-10-06)
- [x] process / keep give a new collection when used as a value, and
      copy[x, true] copies deeply (2026-10-06)
- [ ] Benchmark suite with a baseline (held off by the user, 2026-10-06)
- [x] Log library (2026-10-06)
- [x] REPL with colors (2026-10-06); editor: see the next option below
- [x] `turtle lsp` and a VS Code extension (2026-10-06)
- [ ] `sql` leftovers: writing WITHOUT ROWID tables, UTF-16 and
      auto-vacuum files; virtual tables (fts5)
- [x] Sort by a function, and search: `import sort`, `import search`
      (2026-10-06)
- [x] `data`: `reduce`, `range`, `sum` (2026-10-06)
- [ ] `data`: the discrete-math tier (powerset, combinations, ...)
      proposed 2026-09-30
- [x] Patterns (`import pattern`), backtick strings, number and text
      formatting (`fixed`, `commas`, `padleft`, `padright`), `typeof` and
      `x type order` (2026-10-06)
- [x] Calling a function before its def (top-level functions only;
      2026-10-06)
- [ ] Assembled types: looping over fields,
      default field values
- [x] Quick wins: files and folders, archives, .env, named options,
      `turtle trace` (2026-10-06)
- [ ] Later from the 2026-10-06 list: time zones, a web server (`serve`),
      TOML (or .env + JSON only?), step mode, a VS Code debugger
- [ ] A cookbook doc (asked for 2026-10-06): runnable code examples
      for every stdlib library and function (math, time, data, strings,
      system, json, http, sql, sort, search, random, pattern, log, test),
      from simple to real-world, each checked by a test so it can't go
      stale; plus a guide to writing the best and fastest Turtle code:
      the idiomatic ways (sentences, give, process/keep vs. loops), what's
      fast and slow (Go builtins vs. Turtle loops, in-place vs. copies,
      sql for big data), with measured numbers from the speed tests
- [x] Delete files: `system`'s `erase[path]` (2026-10-03)
- [x] stderr (`warn ... .`), string interpolation, subfolder imports,
      module file names in errors (2026-10-03)
- [x] `get[...]` / `slice[...]` bind to the value before them, so
      `a at get[0] + b at get[1]` works without temporaries (2026-10-06)

## 2. Fix the CI/release pipeline — DONE (2026-09-29)

Split into two workflows:

- **`.github/workflows/ci.yml`** (new): `go build ./...`, `go vet ./...`,
  `go test ./...` on every push to `main` and every pull request. This is
  the "catch a broken commit immediately" gate that didn't exist before.
- **`.github/workflows/release.yml`** (renamed from `go.yml`): now
  triggers only on a pushed version tag (`v*.*.*`) instead of every push
  to `main` — matching the versioning approach this repo already uses in
  practice (`v0.1.4`, `v0.1.5`, `v0.1.6` tags already exist). Each of the
  three OS jobs now:
  - builds `./cmd/turtle` explicitly (previously a bare `go build`, which
    silently built the legacy root-package `Turtle_interpreter.go` — every
    released binary before this fix was the old interpreter, none of the
    rewrite's features included),
  - sets `CGO_ENABLED=0` (the rewrite's `sys` uses `os/exec`, not cgo, so
    the C-toolchain dependency was already dead weight),
  - runs `go vet`/`go test` as a gate before packaging,
  - derives the packaged version from `${{ github.ref_name }}` (the tag
    that triggered the run) instead of a hardcoded string — tagging
    `v0.1.7` and pushing the tag is now the entire release process, no
    file to edit first,
  - uses Go 1.24 (matching `go.mod`'s `go 1.24.0`; was pinned to 1.23,
    a real version mismatch) and current `actions/checkout@v4`/
    `actions/setup-go@v5`.

Also fixed in passing while touching the Linux job: the old `fpm ... -C
release .` packaged the entire `release/` directory verbatim, which would
have installed as `/usr/local/bin/turtle-linux-amd64`, not
`/usr/local/bin/turtle`. Now maps it explicitly:
`turtle-linux-amd64=turtle`.

**Not verified against a real GitHub Actions run** (no network access to
trigger one from here) — YAML syntax was validated, and every `go build`/
`vet`/`test` step was run locally against the exact same commands. Worth
watching the first real tag push closely.

## 2 (was). Original notes, superseded by the above

<details><summary>kept for context</summary>

Checked the current file — it has real problems beyond just being stale:

- **Builds the wrong package.** All three jobs run a bare
      `go build -o release/...` from the repo root, which builds
      `Turtle_interpreter.go` (the legacy file, still `package main` at
      root) — not `./cmd/turtle`, the new interpreter. Needs
      `go build -o release/... ./cmd/turtle`.
- **Still implicitly depends on cgo.** Building the root package
      requires a working C toolchain per target OS (the legacy `sys`
      implementation uses cgo's `system()`). The whole point of the
      rewrite's `os/exec`-based `sys` (see `SPEC.md`) was to drop that
      requirement — once the build target is `./cmd/turtle`, add
      `CGO_ENABLED=0` and the C-toolchain steps become unnecessary
      (this may also fix whatever the windows-latest job was doing about
      a C compiler, which isn't installed there by default).
- **Version is hardcoded to `0.1.6` everywhere** — `--version 0.1.6`,
      `tag_name: v0.1.6`, `turtle_0.1.6_amd64.deb`, in all three jobs. A
      second push to `main` would try to reuse the same tag and release,
      which will fail or silently stop updating the release. Needs a real
      versioning source (a version file, git tag, or commit SHA) before
      "full deployment of updates" can work at all.
- No build/vet gate before packaging — consider a `go vet ./...` (and
      `go build ./...` for every package, not just `cmd/turtle`) step so a
      broken commit doesn't get packaged and released.
- Only triggers on push to `main` — no validation on pull requests.
      Worth a separate lightweight "build + vet" job on PRs if this repo
      starts taking contributions before merge.

</details>

## 3. File extension — DONE (2026-10-06)

`.trt` for everything: programs, imported libraries (`import utils` finds
`utils.trt`), test files (`test_*.trt`), the built-in Turtle libraries,
and `turtle` with no file (the newest `.trt`). `.t` is no longer read.
A file named on the command line (`turtle report.trt`) runs whatever its
extension. The legacy interpreter in `legacy/` is unchanged.

## 4. Extensibility: let `import`ed libraries define their own `at` methods

Stated goal (2026-09-29): make parts of the language extendable, so people
can write their own `.trt` libraries and have them feel first-class, not
bolted on.

**What already works today**, confirmed the same day: `import mylib`
makes `mylib.trt`'s top-level functions available, and you can
call one as `binary[a]`, capture it with plain assignment (`r =
binary[a]`) or the sentence-style `is` form (`r is binary[a] .`), or run
it as a bare statement (`binary[a]`). All of that is ordinary
function-call syntax and needs no new work.

**What's still missing**: the *method-call* sentence form, `r is a at
binary .`, only ever dispatches to the hardcoded per-type Go functions in
`evaluator/data.go` (`listMethod`/`setMethod`/`mapMethod`/`stringMethod`/
`numberMethod`) — there's no fallback to a user-defined function. Today,
`a at binary` on a receiver where no built-in type defines `binary` is
just a fatal "unknown method" error, even if you've imported a library
that defines a top-level `binary` function.

**The fix, when picked up**: in `evalMethodCall`, if the built-in dispatch
for the receiver's type doesn't recognize `mc.Method`, fall back to
`env.GetFunction(mc.Method)` and call it with the receiver prepended as
the first argument (`a at binary x, y` → `binary[a, x, y]`) — uniform
function call syntax (UFCS), the same idea Nim/D use to make free
functions read like methods. Small, contained change: one new fallback
branch, no grammar/parser changes needed (the parser already builds a
`MethodCallExpression` generically).

**Module namespacing: DONE (2026-09-30).** Each `.trt` module now runs
once in its own scope and exports only its top-level functions. `import m
[a, b]` limits what's imported, `m name[...]` qualifies a call, and an
unqualified call to a name two imports share is a fatal error asking for
the qualified form (see `docs/reference.md` §Modules). For the UFCS
fallback, that means `evalMethodCall` should resolve `mc.Method` the same
way `evalCall` does (own defs, then unambiguous imports) rather than only
`env.GetFunction`, and qualified methods (`a at mylib binary`) become an
open syntax question.

**Next step**: confirm whether the UFCS fallback alone covers the goal.
