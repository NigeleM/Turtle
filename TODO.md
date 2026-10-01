# TODO

Forward-looking punch list. For what's already done, see `PROGRESS.md`.

## 1. Expand the standard library

Done so far: string methods, `strings` (`find`, `substring`, `isinstring`,
`join`), `math`, `time`, `data` (`process`, `keep`, `copy`), `system`
(`args`, `exit`, `env`, `scriptFolder`, `contents`, `exists`, `isFile`,
`isFolder`).
See `docs/stdlib.md`.

Next, roughly in priority order:

- [ ] **Error handling** — every runtime error still ends the program, and
      runaway recursion crashes Go with a stack dump instead of a Turtle
      error. Highest-value gap; the rest of this list leans on it.
- [ ] JSON (read/write), then SQLite (pure-Go `modernc.org/sqlite` to keep
      `CGO_ENABLED=0` releases) — rows could come back as assembled values
- [ ] `data`: `reduce`, sort by a function, range generator, the
      discrete-math tier (powerset, combinations, ...) proposed 2026-09-30
- [ ] Assembled types: "is this an Order?" check, looping over fields,
      default field values
- [ ] Clear error for a reserved word used as a `def`/`assemble` name
      (today: a cascade of parse errors)
- [ ] HTTP (lowest priority — biggest surface area)

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

## 3. Settle file extensions, then enforce them

Started, currently on hold (see `PROGRESS.md`). Two decisions, then one
enforcement change:

- [ ] Program file extension — `.trt` was your stated preference.
- [ ] Test file extension — currently `testdata/*.t` (arbitrary, chosen
      during this session, never confirmed with you). Decide whether test
      scripts use the same `.trt` extension as real programs, or a
      distinct one.
- [ ] Once both are settled, restrict `cmd/turtle` to only reading files
      with the agreed extension(s). Still open from before: should an
      explicit `turtle foo.t` argument be rejected outright, or does the
      restriction only apply to the no-argument auto-discovery fallback?
      Should `import` also resolve to the same extension instead of
      `.t`?

**Next step**: your call on the two extension decisions above; I have the
implementation ready to go once they're settled.

## 4. Extensibility: let `import`ed libraries define their own `at` methods

Stated goal (2026-09-29): make parts of the language extendable, so people
can write their own `.t` libraries and have them feel first-class, not
bolted on.

**What already works today**, confirmed the same day: `import mylib`
makes `mylib.t`'s top-level functions available, and you can
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

**Module namespacing: DONE (2026-09-30).** Each `.t` module now runs
once in its own scope and exports only its top-level functions. `import m
[a, b]` limits what's imported, `m name[...]` qualifies a call, and an
unqualified call to a name two imports share is a fatal error asking for
the qualified form (see `docs/reference.md` §Modules). For the UFCS
fallback, that means `evalMethodCall` should resolve `mc.Method` the same
way `evalCall` does (own defs, then unambiguous imports) rather than only
`env.GetFunction`, and qualified methods (`a at mylib binary`) become an
open syntax question.

**Next step**: confirm whether the UFCS fallback alone covers the goal.
