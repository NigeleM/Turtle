# Turtle rewrite — progress log

Status as of this session: **the rewrite is functionally complete and passing
regression against every real historical test script found in this repo.**
All 8 planned tasks (spec extraction, grammar design, scaffold, lexer,
parser/AST, evaluator, file/sys stdlib, testing) are done.

## What exists now

A real lexer -> parser -> AST -> tree-walking evaluator, replacing the old
single-file `Turtle_interpreter.go` (which parsed every line by re-scanning
raw strings with `strings.Contains`/`strings.Index`):

- `token/token.go` — token types and keyword table
- `lexer/lexer.go` — tokenizer (comments, strings w/ escapes, numbers,
  operators; `sys` gets special raw-rest-of-line capture, see below)
- `ast/ast.go` — AST node types
- `parser/parser.go` — recursive-descent parser
- `object/object.go`, `object/environment.go` — runtime values + scoping
- `evaluator/*.go` — tree-walking evaluator (statements, expressions,
  data-structure ops/methods, file I/O, sys, import)
- `cmd/turtle/main.go` — new entry point (`go build -o turtle ./cmd/turtle`)
- `SPEC.md` — the language spec this was built from, including every
  deliberate deviation from legacy behavior and why

The old `Turtle_interpreter.go` and `Files/` package are untouched — nothing
was deleted. `cmd/turtle` is the new way to run scripts.

## How to run it

```
cd /Users/nigy/Desktop/turtle-claude/Turtle
go build -o turtle ./cmd/turtle
./turtle path/to/script.t
```

## How this was verified

Real historical example scripts already in the repo (`test.trt`, `if.txt`,
`define.txt`, `play.txt`, `loop.txt`, `show.txt` — the user added these
mid-session and they were essential for recovering the *real* syntax, since
it differs from what a first pass assumed) were copied into `testdata/*.t`
and run end to end. All pass and produce arithmetically-verified-correct
output. Additional hand-written scripts cover recursion, list/set/map
operations and methods, `break`/`continue`, `sys`, and `[read]`/`[write]`/
`[append]`/`[directory]`.

Two edits were made to the copied `testdata/` scripts vs. the originals:
- `testdata/play.t`: the legacy script defines a function literally named
  `show` — legal under the old ad-hoc scanner, but `show` is a real reserved
  keyword in the new tokenizer. Renamed to `showStuff` in the test copy only
  (the call site was already commented out in the original, so this has no
  behavioral effect). This is a known, documented trade-off of having real
  keywords instead of legacy's incidental substring matching.
- `testdata/loop.t`: the legacy script has two `[loop][...]` headers
  (nested) but only one `[loop][end]` — legacy's flat single-end-closes-
  everything collection made that work by accident, but only when the
  nested loop is the last statement in its parent's body (undocumented,
  fragile). The rewrite requires one `[end]` per loop (see SPEC.md's Loops
  section for the reasoning); added the missing `[end]` in the test copy.

## Bugs found and fixed during implementation (beyond what SPEC.md already covers)

- **Lexer**: the `"` case fell through to a shared trailing `readChar()`
  meant for single-char tokens, silently eating one extra character after
  every string literal whenever no whitespace followed it (e.g.
  `,"x",` lost the second comma). Fixed by returning immediately after
  `readString()`.
- **Parser — call vs. bracket-close ambiguity**: `name[args]` call syntax
  and the reversed-bracket if-header closer (`... [`) are both just
  "identifier or bracket followed by `[`" at the token level. Resolved by
  requiring a genuine call's identifier, `[`, and first argument all sit on
  the *same source line* — a header-closing `[` is always the last token on
  its line in every real example. First version of this check only looked
  at the `[` and what followed it, not whether the *identifier* was even on
  the same line as the `[` — that gap made `k` at the end of one line
  followed by a same-line-internally `[loop][end]` on the next line
  misparse as a call `k[loop]`. Fixed by also requiring identifier and `[`
  share a line.
- **Evaluator — loop variable scoping**: making the loop induction variable
  a real, live Turtle variable (to fix the legacy bug where `show i` inside
  a loop never changed) introduced a new problem: nested loops reusing the
  same variable name (a real pattern in `loop.txt`, both use `i`) would
  clobber each other's iteration control. Fixed by having `evalLoop`
  save/restore the induction variable's pre-loop value (or delete it if it
  didn't exist) around each loop's execution, so nesting with a shared name
  now works correctly (verified: outer loop runs its full 7 iterations,
  inner loop's range shrinks correctly each pass).

## Known limitations / things not attempted

- `[directory]` and working `[write]`/`[append]` are new functionality —
  the legacy versions were non-functional stubs, so "parity" here means
  implementing the documented-but-never-built intent, not preserving old
  behavior (there was none to preserve).
- Reserved keywords (`show`, `if`, `def`, `loop`, `add`, `change`, `is`,
  `at`, `to`, `from`, `of`, `min`, `max`, `length`, etc.) can't be used as
  variable or function names. Legacy had no real keyword concept, so this
  is a genuine, unavoidable trade-off of building an actual tokenizer.
- No attempt was made to support arbitrary-depth if/else nesting beyond
  what the bracket-counting generalization naturally provides (untested
  beyond depth 2, since no real example goes deeper than depth 1).
- Did not delete/replace `Turtle_interpreter.go` or `Files/`. Worth doing
  once you're satisfied, but left alone deliberately so nothing is lost if
  something in the rewrite still needs fixing.

## If picking this back up

- Read `SPEC.md` first — it's the authoritative reference for intended
  syntax/semantics and documents every intentional deviation from legacy.
- `testdata/*.t` plus `test.trt` are the regression suite; there's no test
  runner script, just run `turtle` against each and eyeball the output
  (all outputs in this log are known-correct baselines to diff against).
- One earlier false start: two research subagents launched mid-session
  overstepped their (research-only) brief and wrote a first draft of this
  same rewrite using invented, incorrect if/else syntax (plain `if[cond]`
  instead of the real reversed-bracket form). That draft was fully
  discarded before any of it was reused — everything under `token/`,
  `lexer/`, `ast/`, `parser/`, `object/`, `evaluator/`, `cmd/` was written
  fresh afterward against the syntax confirmed from the real historical
  scripts. Mentioned here only so it isn't a mystery from git history if
  you go looking.

## Follow-up session: calculator smoke test + a new `change` keyword

Wrote `testdata/calculator.t` as a real end-to-end functionality test
(functions, nested-ish conditionals, C-style loops over parallel lists,
`is ... at get`, interactive `?` input). It surfaced two real gaps:

1. `add` is reserved for the `add <expr> to <target> .` data-structure
   statement, which meant a function literally named `add` (the obvious
   name for calculator addition) couldn't be defined. This is a real,
   deliberate keyword — not changed. `testdata/calculator.t`'s addition
   function is named `addition[a, b]` instead.
2. `?` input only ever produces a string, and there was no way to turn
   `"7"` into `7` for arithmetic. **Fixed** by adding a `change` keyword
   (`token.CHANGE`, `ast.ChangeExpression`, `evalChange` in
   `evaluator/expr.go`): `change <ident> to <type> .` mutates that variable
   in place; used as an expression (`x = change a to integer`) it converts
   without mutating the source. Target types: `integer`, `float`, `string`,
   `ascii`, `char`, `hex` — plain identifiers after `to`, not reserved
   words, same treatment as method names. `ascii`/`char` cover
   string-char <-> code-point (Python `ord`/`chr` equivalent) and `hex`
   covers integer <-> hex-digit string, direction inferred from the
   source's actual type in both cases. See `docs/reference.md`'s "Type
   conversion" section for the full conversion matrix.

`testdata/change.t` covers both statement and expression forms and all six
target types. `testdata/calculator.t`'s interactive mode now does real
`change`-based numeric arithmetic end to end (verified with piped stdin,
including the division-by-zero guard path).

## Follow-up: string methods (get/slice/split/contains/indexOf/replace/upper/lower/trim)

Extended the existing `<result> is <receiver> at <method> [args] .`
mechanism (already used by list/set/map) to strings — no new tokens or
parser changes needed, since `evalMethodCall` (`evaluator/data.go`) just
type-switches on the receiver; added `case *object.String` dispatching to
a new `stringMethod`.

Two design decisions worth remembering:
- **All indexing is rune-based, not byte-based** (`get`, `slice`,
  `indexOf`'s returned position). This required fixing `length of` on a
  string (`lengthOf` in `evaluator/data.go`) from `len(v.Value)` (byte
  count) to `len([]rune(v.Value))` — a real behavior change for any
  non-ASCII string, verified with `"café"` (byte length 5, rune length 4;
  `get 3` correctly returns `é`).
- **`slice` is Python-style but without a step**: `slice <start> [, <end>]`
  — negative bounds count from the end (`-1` = last char), an omitted
  `end` defaults to the string's length, and out-of-range bounds clamp
  rather than erroring. This is deliberately looser than `get`, which is a
  strict single-index lookup that's fatal out of range — matches Python's
  own get-vs-slice distinction. No `[::step]` support; reversal, if
  wanted later, should extend the existing `reverse` data-op instead of
  overloading `slice`.

`testdata/strings.t` covers all nine methods plus the Unicode length/index
case. See `docs/stdlib.md`'s new `#### string` section for the full method
table.

## Follow-up: runtime error line numbers + functions can read globals

Two gaps flagged during a language-quality review: runtime errors carried
no location at all (worse: `evaluator/expr.go`'s undefined-variable error
claimed `line %d` and hardcoded the number to `0`), and functions couldn't
read even a global constant, only their own parameters.

**Line numbers**: added `Line() int` to the `ast.Statement` interface,
implemented for all 18 statement types (`ast/ast.go`) as a one-liner
returning `Token.Line` — every statement node already carried its token,
this just exposes it. `evalStatement` (`evaluator/evaluator.go`, the single
dispatch point every statement passes through, top-level or nested) sets a
package-level `currentLine` before dispatching; `fatalf` now prefixes every
message with it. No call-site changes needed anywhere else in the
evaluator — every existing `fatalf(...)` call automatically picked up a
real line number for free. Verified: an error deep in a loop body reports
the loop-body line, and an error inside a function call reports the
function body's own line, not the call site.

**Global reads**: `object.Environment.Get` (`object/environment.go`)
previously only checked its own `vars` map. Now it falls back to
`e.global.Get(name)` when the name isn't local, so a function body can
read a top-level variable without it being passed as a parameter — a
parameter/local of the same name still shadows the global. Deliberately
did **not** make `Set` write through to global: a plain `x = ...` inside a
function always creates/updates a *local* `x`, even if a global `x`
exists, so a function can't silently clobber shared state through a name
collision. Mutating a global `list`/`set`/`map` via a data-op or method
call still works and is visible after the call returns, since that
mutates the same `*object.List`/`Set`/`Map` value `Get` returned rather
than rebinding a name through `Set` — verified with `add v to nums .` on
a global list from inside a function. See `docs/tour.md`'s expanded
Functions section and `docs/reference.md`'s Functions section for the
full read/write/mutate distinction, with runnable examples.

## Follow-up: modulo, `import math`/`import time`, list slicing, safe input validation

User-requested batch, building on the earlier stdlib/quality work:

1. **Modulo (`%`)**: new `token.PERCENT`, lexer case, `PRODUCT`-level
   precedence (same as `*`/`/`), and an `evalInfix` case — int `%` int
   stays an integer, otherwise falls back to `math.Mod`; modulo by zero is
   fatal like division by zero. Mechanical, exactly per
   `docs/contributing.md`'s own tutorial for adding an operator.
2. **`import math` / `import time` as builtin modules**: `evalImport`
   (`evaluator/io.go`) now special-cases these two names (the
   `builtinModules` map in `evaluator/evaluator.go`) to flip an internal
   `Interpreter.modules["math"/"time"]` flag instead of reading a
   `<name>.t` file. A new `requireModule` helper fatals with a message
   naming exactly which import is missing (`"sqrt" needs "import math"
   first`) — this is deliberately a better error than "unknown method",
   since the whole point of gating is to tell you what to add. Consequence
   documented in `reference.md`: a real file named `math.t`/`time.t` would
   never be reachable via `import`, since those two names always resolve
   to the builtin.
3. **Math methods on numbers** (`evaluator/data.go`, gated behind `import
   math`): `sqrt`, `abs`, `round`, `floor`, `ceil`, `pow`, `random`, added
   by extending `evalMethodCall`'s receiver dispatch to also accept
   `*object.Integer`/`*object.Float` (previously list/set/map/string only).
   `random` reads naturally with the existing `is <receiver> at <method>`
   grammar: `n is 10 at random .` = "n is [derived from] 10, at random."
4. **`now[]`/`sleep[amount [, unit]]`** (`evaluator/expr.go`'s `evalCall`,
   gated behind `import time`): the first builtin *functions* in the
   language (everything before this was either a keyword-driven statement
   or a user-defined function). Implemented as a fallback in `evalCall`,
   checked only when no user-defined function of that name exists — so a
   user's own `now`/`sleep` function always wins and is never shadowed.
   `sleep`'s `unit` defaults to `"seconds"` when omitted, or `"ms"` when
   given explicitly. First pass split this into two functions
   (`sleep`/`sleepSeconds`); reworked into one function with an optional
   unit argument on request — "keep names simple," and it already matched
   the variadic-arg-count pattern `slice` established.
5. **List `slice`**: extended `listMethod` with the same
   negative-index/clamping `slice` semantics already built for strings
   (reusing the existing `normalizeSliceIndex` helper) — `nums at slice
   0, 2 .`, `nums at slice -1 .`.
6. **`isNumber` string method**: `raw at isNumber .` reports whether
   `change ... to integer/float` would succeed, without attempting the
   conversion — lets a program validate untrusted `?` input and reprompt
   instead of crashing.

**A real bug this surfaced**: wiring `isNumber` into `testdata/calculator.t`
(reprompt-on-bad-input via `continue` in the while-style interactive loop)
hung forever in testing. Root cause: `evalStatement`'s `*ast.InputStatement`
case (`evaluator/evaluator.go`) called `it.stdin.Scan()` without checking
its return value — at real EOF, `Scan()` returns `false` and `Text()`
keeps returning `""`, so a retry loop reading `?` past EOF span an infinite
loop of empty-string reprompts instead of ever terminating. This was
latent before (a bad `change` conversion used to crash immediately, which
accidentally hid it) and is now fixed: EOF on `?` is a fatal
"unexpected end of input" error. Verified both the fixed retry flow
(bad input → reprompt → success) and that a genuine EOF now exits
immediately instead of hanging.

New tests: `testdata/math.t`, `testdata/time.t`, list-slice lines added to
`testdata/datastruct.t`. Docs updated: `docs/reference.md` (precedence
table, Modules section), `docs/stdlib.md` (new `#### number` and
`### Builtin functions: time` sections, `isNumber`, list `slice`),
`docs/tour.md` (Modules section example).

**Also revisited on request**: `sleep` was first split into two functions
(`sleep[ms]` / `sleepSeconds[s]`); reworked into one `sleep[amount [,
unit]]` (`unit` is `"seconds"`, the default, or `"ms"`) on request — "keep
names simple," matching the variadic-arg-count pattern `slice` already
established, rather than minting a name per unit.

## Follow-up: a real `go test` suite, and fixing the CI/release pipeline

Two things flagged as urgent in a language-quality review, tackled
together since the second needed the first to actually run anything.

**Testability groundwork**: every runtime error went through `fatalf`
(`evaluator/evaluator.go`), which called `os.Exit(1)` directly — meaning
no test could exercise an error path (division by zero, a bad `change`
conversion, a missing `import math`) without killing the whole test
binary. Refactored `fatalf` to `panic(fatalError{...})` and gave
`Interpreter.Run` a `recover` that catches exactly that type and returns
it as a normal Go `error`; anything else panicking is re-panicked, so a
real interpreter bug is never silently swallowed. `cmd/turtle/main.go`
now does `if err := it.Run(program); err != nil { ... os.Exit(1) }` —
verified byte-for-byte identical CLI output/exit codes before and after
(same `"turtle: line N: message"` format). Also added
`evaluator.NewWithStdin(dir, io.Reader)` alongside the existing `New`, so
tests can feed `?` input without touching real stdin.

**The test suite itself**: `go test ./...` went from "no test files" in
every package to real coverage in four:
- `lexer/lexer_test.go` — token sequences for core syntax (including the
  new `%` and `change`/`addition` keywords), line tracking, `sys`'s
  raw-rest-of-line capture, reserved-word lookup.
- `parser/parser_test.go` — AST shape for assignment, the reversed-bracket
  if/else, both loop header forms, function def/call, `is ... at ...`
  method-call lowering, `change`'s self-assignment lowering, and that
  parse errors carry real line numbers.
- `evaluator/evaluator_test.go` — the highest-value layer: table-driven
  end-to-end programs asserting captured stdout (via an `os.Pipe` swap
  around `os.Stdout`) and/or the returned error, covering arithmetic +
  modulo, all six `change` targets (statement vs. expression form), every
  string/list method added this session including negative-index `slice`,
  `import math`/`import time` gating (both the success and the
  "not imported yet" error path), the global-read/local-write-shadow
  semantics, recursion, loop `break`/`continue`, if/else nesting, `?`
  input, and the EOF-doesn't-hang fix.
- `object/environment_test.go` — the global/local scoping rules directly,
  without going through the whole pipeline.

Writing these surfaced no new interpreter bugs — every initial test
failure was a wrong expectation in the test itself (e.g. `at` method-call
syntax only working inside `is ... at ...`, not as a bare expression
inside `show`; the input prompt text sharing the same captured stdout
stream), which is itself a useful confirmation that last session's manual
testing had already been accurate.

**CI/release pipeline** (`TODO.md` item 2, now marked done there with
full detail): split into `.github/workflows/ci.yml` (build+vet+test on
every push/PR — the safety net that didn't exist before) and a rewritten
`.github/workflows/release.yml` (renamed from `go.yml`) that now triggers
only on a pushed version tag, matching the `v0.1.4`/`v0.1.5`/`v0.1.6` tags
already in this repo, and derives the packaged version from the tag
instead of a hardcoded string. Each release job also now builds
`./cmd/turtle` explicitly — previously a bare `go build` silently built
the legacy root-package `Turtle_interpreter.go`, meaning **every
previously-released binary was the old interpreter**, none of the
rewrite's features included. Not verified against a real GitHub Actions
run (no network access to trigger one from here); YAML syntax was
validated and every underlying command was run locally.

**First real CI run** (pushed by the user, 2026-09-29) came back showing
"6 errors, 2 warnings, 1 notice" in GitHub's Annotations panel — alarming
at a glance, but the job's actual `conclusion` was `success` (checked via
the Actions API directly: every step — build, vet, test — passed). The
"6 errors" were the same harmless cgo compiler warning (`ignoring return
value of 'system'`) from the legacy `Turtle_interpreter.go`, appearing
twice each across the build/vet/test steps (3 steps × 2 lines each);
GitHub's annotation extractor tags cgo/gcc warnings as `failure`-level
even though they don't fail anything. Real fix, not just cosmetic: `./...`
was pulling the legacy root package (cgo) and `Files/` (no cgo, harmless)
into scope. Confirmed via `grep -rl 'import "C"'` that `Turtle_interpreter.go`
is the *only* cgo file in the repo, then scoped every `go build`/`vet`/
`test` invocation in both `ci.yml` and `release.yml` to the eight rewrite
packages explicitly (`./Files/... ./ast/... ./cmd/... ./evaluator/...
./lexer/... ./object/... ./parser/... ./token/...`), never touching the
root package. This isn't just about quieting the annotation noise: the
Windows release job has no C compiler by default, so `go vet ./...`/`go
test ./...` there would have hit the cgo file and could have genuinely
failed (not just warned) the next time that job actually ran — this fix
heads that off before it happens. Verified all three commands locally
(build/vet/test with the explicit package list, matching exactly what CI
now runs).

## First real release (v0.9.0) — and a real macOS packaging bug it surfaced

Pushed `v0.9.0` (deleting two stray, wrongly-shaped tags — `v0.9` and
`0.9` — first; the trigger is `v*.*.*`, three dot-separated components,
so neither matched and neither fired anything). `release.yml` ran clean
on all three OS jobs and published a real GitHub Release with all three
installers attached — the first release ever built from the rewritten
interpreter rather than the legacy `Turtle_interpreter.go`.

Installing the `.pkg` on the actual Mac this was developed on surfaced a
real, pre-existing bug: the installer reported success, but `turtle` was
nowhere on `PATH` afterward. Root cause, confirmed directly via `pkgutil
--pkg-info`/`--files` and `find`: the binary landed at
`/usr/local/bin/usr/local/bin/turtle`, not `/usr/local/bin/turtle`. The
`pkgbuild` invocation had `--install-location /usr/local/bin` *and* a
payload root (`pkgroot/usr/local/bin/turtle`) that already contained that
same path — the two concatenate, so the destination path got applied
twice. This exact `pkgbuild` command predates this session (inherited
unchanged from the original `go.yml`), so **every prior `.pkg` release
(`v0.1.4`–`v0.1.6`) almost certainly had the same bug** — worth asking
before assuming this is new.

One wrinkle worth recording: the user recalled that the "obviously
correct" flat-payload form (`pkgroot/turtle` directly, no nested path) had
been tried before and silently failed to write anything — which is why
the nested form existed in the first place. Before touching it again,
confirmed with them that the old failure looked like *this exact
symptom* (installer reports success, binary missing from `PATH`), not a
Gatekeeper/code-signing block (which would refuse to run the installer at
all, not misplace its output) — so the earlier nested-path change was
very likely an attempted fix for the same underlying bug that didn't
actually solve it, just moved where the misplaced file ended up.

Fixed by flattening the payload (`pkgroot/turtle`, not `pkgroot/usr/local/bin/turtle`)
and verified two ways before touching `release.yml`, since `pkgbuild` is
available on this Mac:
1. `pkgutil --payload-files` on a locally-built test package showed
   `./turtle` (not `./usr/local/bin/turtle`).
2. `pkgutil --expand` + `lsbom` on the Bill of Materials — the actual
   manifest macOS's Installer reads — confirmed `.` → `/usr/local/bin`
   and `./turtle` → `/usr/local/bin/turtle`, no nesting.

Also manually fixed the already-broken install on this machine (`cp`'d
the misplaced binary to the correct path, removed the bogus nested
`/usr/local/bin/usr/` directory) so `turtle` works from the terminal here
right now, independent of whether/when a new tagged release goes out with
the packaging fix.

## 2026-09-30 – 10-01: functions as values, modules, and the data/system libraries

Language features, each with tests and docs (`docs/reference.md` is the
full rulebook):

- **Functions as values and closures** (lexical scope; captured variables
  are read-only by assignment, like globals), **`none`**, bare `return`.
- **Module namespacing**: each `.t` module runs once in its own scope and
  exports only its top-level functions; `import m [a, b]`; `m name[...]`
  to qualify; an unqualified call to a name two imports share is an error.
- **Structural equality** (`set [1, 2] == set [2, 1]`; `1 != "1"`).
- **`gives`** anonymous functions and **sentence-style calls**
  (`nums process f` = `process[nums, f]`). Library verbs are ordinary
  functions rather than keywords, so `.t` libraries can define their own.
- **For-each loops**, **`+`/`-` on lists/sets/maps**, `x at m` in any
  expression.
- **`data`** (`process`, `keep`, `copy`) and **`system`** (`args`,
  `exists`, `isFile`, `isFolder`) libraries.
- **`assemble`** named types with fields (`qty of o`, `qty of o = 5`).

New reserved words: `none`, `gives`, `in`, `assemble`. Importing a file
no longer exposes its variables.

Regression check: all `testdata/` scripts print exactly what the
pre-session build (`11c1977`) printed. Note: `timeout` isn't installed on
this Mac. An early version of that check silently compared two "command
not found" errors; it now uses `perl -e 'alarm N; exec @ARGV'`.

## 2026-10-01: stress test and the fixes it drove

`testdata/everything.t` (with `everything_lib.t`) uses every feature and
checks its own results, exiting 1 on any failure; `go test` runs it.
111 separate edge-case probes found these, now fixed:

- Closures made inside a loop lost the loop variable once the loop ended
  (loops now give their names their own scope per pass).
- Runaway recursion crashed Go; now "recursion too deep" at 100,000 calls.
- Sets were O(n) per membership check; now indexed.
- Floats show as floats (`4.0`); text inside collections is quoted
  (`[ 1, "1" ]`); map keys keep their type; 64-bit integer overflow is an
  error, not a wrap-around; empty list/set/map are falsy; `isEmpty`.
- Sentence-style calls need no parentheses: one argument inside `[...]`,
  and `s substring -5` passes a negative argument.
- A method call ending an if-header (`if ] x at isEmpty [`) misread the
  header's closing `[`.

A second stress run (151 probes, 148 passing; the rest are design
limits) and its fixes are logged in `docs/stress-test-log.md`.

## 2026-10-03: error handling with `safe` / `handle` / `fail`

The user chose the words and the shape:

```
safe
    [read] data.txt to lines [end]
handle [file] error .
    show error .
safe [end]
```

- The code under `handle` runs only when the safe code hit an error of a
  listed kind; `safe [end]` closes the whole thing. (A first version had
  `handle` close the block with the code after it always running; the
  user switched to the handle block the same day.)
- Every runtime error now has a kind: file, number, math, index, key,
  name, type, or custom (from `fail "message"`). `handle [] e .` takes
  any kind; an unlisted kind still stops the program, as does an error
  in the handle code itself.
- `show e .` prints the message; `kind of e`, `line of e`,
  `message of e` read the parts. With no error, `e` is `none`.
- `return`/`break`/`continue` inside `safe` or `handle` still work;
  `exit[code]` and parse errors are never handled. A failed import can
  be handled.
- File errors are shorter: `[read] nope.txt: no such file or folder`
  instead of Go's message with the full path twice.

## 2026-10-03: file names in errors, interpolation, subfolder imports, erase, warn

- Errors inside an imported module name its file:
  `lib/utils.t line 2: division by zero`, both when the program stops and
  in a handled error (`file of e`). Parse errors in a module too. After a
  function call returns, errors name the caller's line again (they used
  to keep the callee's last line).
- `"Hi {name}, {qty * 2} items"`: any expression without quotes inside
  the braces; `\{` for a plain brace. Works in prompts and `[write]`
  lines too.
- `import lib/utils` reads `lib/utils.t`; the qualified name is `utils`.
- `system`: `erase[path]` deletes a file or a whole folder (refusing the
  folder turtle runs in and those above it), and `warn "..." .` is show
  for stderr. Names chosen by the user.
- `+` with a list, set or map on one side and anything else is now a
  `type` error (`list [1] + 1` used to give "[ 1 ]1"). Same-kind
  collections still combine.
- `none + none` is `none`; `none` plus anything else is a `type` error
  (`"x=" + none` used to give "x=none"). Both errors suggest the fix.
- `change x to set` / `change x to list` convert between lists and sets
  (duplicates dropped, first-seen order kept; same kind gives a copy).
  Put in `change` rather than `data`, at the user's call.

## 2026-10-03: JSON library, single-quoted strings

- `import json`: `load[text]`, `json_text[value]`, `json_read[path]`,
  `json_write[path, value]` (indented), `json_get[value, keys...]`
  (returns none on a missing step). Names chosen by the user. Objects
  keep key order; whole numbers load as integers; null is none. Bad JSON
  is a new error kind, `json`, with line and column.
- Strings can be `'single-quoted'`, so JSON in code needs no backslashes.
- A `{` followed (after spaces) by a quote or `}`, or by nothing, is a
  plain brace, not interpolation: `'{"a": 1}'` and `"{}"` just work.
- HTTP is the next wave; `load` already takes any text.

## 2026-10-03: dates in the time library

- New date value: shows as `2026-10-03 14:05:00`, parts via `of`
  (`year`, `month`, `day`, `hour`, `minute`, `second`, `weekday`),
  compares with `< >`, works as a map key, written to JSON as text.
- `today[]` (local), `today_utc[]`, `make_date[...]`, `to_date[text]`,
  `add_time[d, n, unit]`, `time_between[a, b, unit]`,
  `format_date[d, pattern]`, `wait_until[d]`, `every[n, unit, job]`.
  Shape chosen by the user. Months clamp to the month's end; days keep
  the clock time across daylight saving.
- New error kind `date` for bad date text or impossible dates.

## 2026-10-03: HTTP library

- `import http`: `http_get[url]`, `http_post[url, body]` (text, or a
  map/list sent as JSON) return the body; `http_request[method, url,
  body, headers]` returns status/body/headers and doesn't fail on 4xx/5xx.
  Standard library only. New error kind `http`.
- The user ruled out OS scheduling and third-party dependencies: the
  database layer will be written from scratch (SQLite file format first,
  PostgreSQL wire protocol later).

## 2026-10-03: SQL library, reading SQLite from scratch

- New `sqlite/` package, no dependencies (the user's rule): reads the
  SQLite file format (header, pages, table B-trees, overflow pages,
  records, UTF-8/16 text), the schema (CREATE TABLE columns, INTEGER
  PRIMARY KEY as rowid, ALTER TABLE-added columns), and runs SELECT with a
  from-scratch SQL lexer, parser and executor following SQLite's rules
  (affinity in comparisons, NULL, integer overflow to real, x/0 NULL).
- Tested against the real sqlite3: 61 queries match exactly, at page
  sizes 512 to 65536. The comparison found and fixed: whole REALs stored
  as integers on disk, ASCII-only upper/lower, rowid column naming, alias
  scoping.
- `import sql`: `sql_open`, `sql_query` (list of maps, `?` placeholders),
  `sql_tables`, `sql_close`. Error kind `sql`. Reading only; writing is
  next, then GROUP BY/joins, then PostgreSQL.


## 2026-10-04: CI fix for the sqlite tests (v0.9.140)

- v0.9.139 failed `go test` in CI and the Linux/macOS release jobs, so
  no packages were built for those (Windows passed only because its
  runner has no `sqlite3`, so the tests skip). The cause was the test, not
  the driver: GitHub's sqlite3 (3.45.1 on Ubuntu) writes blob bytes
  0x80 and up as broken `\uffffffca` escapes in `-json` output, and the
  test compared against that. Reproduced locally with a self-built
  3.45.1.
- The `-json` comparison now leaves out blob columns. Blobs are still
  checked against sqlite3 through `hex(cover)`, and by exact bytes in
  `TestTablesAndTypes`.

## 2026-10-04: sqlite tests no longer run sqlite3 (v0.9.141)

- v0.9.140's macOS release still failed: the runner's sqlite3 is Apple's
  ICU build, so `upper('café')` gave `CAFÉ` where standard SQLite (and
  this package) gives `CAFé`. Testing against whichever sqlite3 a machine
  has was the real problem.
- The fixture databases (page sizes 512 to 65536) and sqlite3's answer to
  every query are now committed in `sqlite/testdata/`; the tests read
  them and never run the tool, so they also run on Windows now.
  `go test ./sqlite -update` rebuilds them with a standard sqlite3.
- The http "connection refused" test uses a port that was just closed
  instead of port 1, which a firewall could filter into a timeout.

## 2026-10-05: SQL writing, joins and groups; tables and table files

- `data`: `table[x]` lays out rows (lists of maps or assembled values,
  plain lists, one map) as a text table; `import data` defines
  `tablerows = 20` (the user picked a default variable); `table[x, n]`
  for one call. `table_write[path, x]` / `table_read[path]` save and read
  the same layout as `.csv`, `.tsv` or `.txt` (write only); new error
  kind `csv`.
- `sql`: `sql_run` (changes, returns the row count), `sql_create` (new
  database file); `sql_query` also runs changes with `RETURNING`; a column
  name used twice gets `:1`. Files in and out (the user's names):
  `sql_save` (query to file), `sql_load`, `sql_update`, `sql_delete`,
  `sql_upsert` (by a key column), each file all or nothing (one
  statement through `sqlite.ExecRows`).
- `sqlite/` now writes: table and index B-trees (insert, delete, split,
  merge, overflow pages, free list), the rollback journal in SQLite's own
  format (crash recovery both ways with the real SQLite), SQLite's file
  locks, constraints (NOT NULL, UNIQUE, PRIMARY KEY, CHECK, STRICT,
  AUTOINCREMENT, ON CONFLICT, upsert), CREATE/DROP/ALTER, views,
  transactions. Queries gained joins (inner, left, right, full, cross,
  natural, USING), GROUP BY/HAVING, subqueries (correlated too), UNION/
  INTERSECT/EXCEPT, WITH (recursive), VALUES, collations, GLOB, lookups by
  rowid and index, and the date, printf and math functions.
- Checked against real SQLite: 122 queries and 16 function queries,
  10 write scripts (rows, types, schema text, errors), PRAGMA
  integrity_check on every file Turtle writes, random changes against a
  model, files with Apple's 12 reserved bytes per page.
- `testdata/sql/`: twelve Turtle programs, one per topic, with CSV data;
  run by `TestSQLExamples`.

## 2026-10-06: the rest of SQL; PostgreSQL and MySQL (v0.9.142)

- `sqlite/` gained window functions (frames, EXCLUDE, named windows),
  JSON1 (`->`, `->>`, `json_each`, `json_tree`), table-valued functions
  (`generate_series`, `pragma_*`), triggers (INSTEAD OF on views, RAISE),
  savepoints, foreign keys (PRAGMA foreign_keys), PRAGMAs, generated
  columns, expression indexes, VACUUM / VACUUM INTO, read-only ATTACH,
  reading WITHOUT ROWID tables, WAL mode, `UPDATE ... FROM`. Faster: a
  byte-level B-tree search, a page cache and hash joins.
- `postgres/` and `mysql/`: clients written from scratch (wire
  protocols, SCRAM / MD5 / caching_sha2 / sha256 / native logins, TLS,
  prepared statements). `sql_open` picks the driver by address
  (`postgres://`, `mysql://`); every `sql_` function works on all three
  through `evaluator/sqllib.go`'s `sqlConn` interface.
- `turtle doc [topic]`: the standard library's documentation, and the
  `//` comments above a `.t` file's functions.
- Tests: more sqlite3 comparisons (queries, write scripts, crash and
  lock tests), server tests gated by `TURTLE_PG_URL` /
  `TURTLE_MYSQL_URL`, which CI now sets with service containers;
  `testdata/sql/13_servers.t`.

## 2026-10-06: table files, sort and search, `get` binding (v0.9.143)

- `.json` table files: `table_write`, `table_read` and the `sql_` file
  functions read and write a list of objects, one per row, keeping
  numbers, booleans, null and nested values.
- `contains` on lists and sets (a value) and maps (a key).
- `import sort`: `min_sort` / `max_sort` with keys (a function, a field
  or map key, a position, or a list of them) and `"first"` or a count,
  `is_sorted`, `reverse_list`, and the classic algorithms.
- `import search`: `find_first`, `find_last`, `find_all`, `find_index`,
  `count_where`, `find_key`, linear / binary / jump / exponential /
  interpolation / ternary search, `insert_position`.
- `get[...]` and `slice[...]` bind to the value before them, so
  `a at get[0] + b at get[1]` adds two items and `title of books at
  get[0]` reads the first book's title. `of` reads map keys.
- v0.9.142's release failed on macOS and Windows (fixed in d648bbb);
  CI now runs on all three systems.

## 2026-10-06: repository clean-up

- The legacy interpreter (`Turtle_interpreter.go`, `Files/`,
  `test.trt`) moved to `legacy/`; it still builds and runs. The old
  compiled `turtle` binary is no longer tracked (`/turtle` is ignored).
- The release workflow now vets and tests `postgres/` and `mysql/` too.

## 2026-10-06: column types for table files (not yet released)

- `table_read[path, types]` and `sql_load[db, table, path, types]` take
  a map of column name to type (a literal or a variable): Turtle words
  (`string`, `text`, `integer`, `float`, `boolean`, `date`) or SQL types
  (`VARCHAR(10)`, `NUMERIC(10, 2)`, ...), in any case. "007" stays text;
  columns left out are text; `"primary_key": "column"` names the key.
- `sql_load` makes a missing table from the file's header (all text
  without a map), with the map's types and primary key, spelled for each
  database; if the file is refused, the new table is dropped again.
- `table_read` with a key checks every row has a different one.
- Errors name the row and column. `testdata/sql/14_typed_load.t`, a
  typed load in `13_servers.t` (PostgreSQL and MySQL), unit tests.

## 2026-10-06: random library (not yet released)

- `import random`: random values of any shape as a sentence:
  `random integer from 1 to 6`, `random list of 5 integers from 0 to 9`,
  `random float from 0.5 to 99.99 rounded to 2`, `random string of 8`,
  `random digits of 4`, `random string of 6 from "ABC123"`, `random date
  from "2026-01-01" to "2026-12-31"`, `random time`, `letter`, `digit`,
  `boolean`, sets and maps (all-different keys), nesting to any depth, and
  assembled values with a kind per field: `random Order [string, integer]`.
  Plurals, counts (`list of 2 to 8 integers`, `list of n integers`).
- `pick`, `shuffle`, `sample`, `chance`; the `seed` variable (none:
  different each run; a number: the same values every run; setting it
  restarts them).
- `random` is a sentence word only in a file with `import random`
  (parser/random.go); the shape grammar (ast.Shape) is meant to be shared
  with the test library's `validate`.

## 2026-10-06: methods work on the value right before them (not yet released)

- `a at invert == b at invert` compares the two inverted maps, like
  Python's `a.invert() == b.invert()`; `"a" + "b" at upper` is `"aB"`,
  and `( )` groups: `("a" + "b") at upper` is `"AB"`. Before, `at`
  bound loosest (only `get[...]` / `slice[...]` bound tightest).
- `-7 at abs` is 7 (`-7` is one number); `-x at abs` is `-(x at abs)`.
- `r is ... at m args .` follows the same rule; `title of b at upper`
  still upper-cases the title.
- Tests for the rule and for long expressions (`1 + 2 - 3 + 4 - (5 + 6)
  + 7 * 8`), and reference.md's order of operations.
- The random library's `text` is now `string` (`random string of 8`).
- Strings gained `at length` / `at len`, as lists have. Test of long
  expressions mixing strings, lists, maps and sets with chained methods.

## 2026-10-06: `/` is exact, `div` keeps the whole part (not yet released)

- `7 / 2` is 3.5 and `6 / 3` is 2.0: `/` always gives a float. The new
  operator `div` (a reserved word) gives the whole part, toward zero:
  `7 div 2` is 3, `-7 div 2` is -3, so `(a div b) * b + a % b` is `a`.
  Before, two integers divided as whole numbers. `//` was taken by
  comments.
- Example programs that meant whole numbers now use `div` (the shop's
  money and averages); tests for both operators.

## 2026-10-06: the test library (not yet released)

- `import test` makes `check`, `verify` and `validate` statements (only
  in that file; elsewhere they're ordinary names):
  - `check <true/false> .`: any comparison, method or library function;
    a failed `==` shows the first difference (items, keys, fields,
    characters), other failures show the parts' values. Also
    `is [not] <kind>` (integer, float, number, string, boolean, list,
    set, map, date, none, function, empty, an assembled type),
    `is [not] close to x [within d]`, and `fails [kinds]`.
  - `verify <values> each | any | not | at least N | at most N |
    exactly N [pair] <rule> .`: lists which items break the rule.
  - `validate <call> [to name] with x as <kind>, ... that <rule> |
    matches <call> .`: 100 random inputs (`cases`), shrunk to the
    smallest failing one, with the seed to repeat it. Without `with`,
    it learns the kinds from the test's checks.
- `turtle test [file | folder ...]`: `test_*.t` files, `test_`
  functions; PASS / FAIL / SKIP with times; `suite = false | "stop" |
  "all"`; `benchmark`, `runs` (none: Go-style, as many as fit in
  `benchtime`), `benchtime`; settings at file level or inside a test.
  Exit code 1 on failure. Failures name the file and line (the
  library's, for an error inside an imported library).
- Error kind `test`. Maps got `at length` too.
- docs/testing.md (the full guide, with at least / at most / exactly
  worked through), reference, stdlib, getting started, `turtle doc
  test`; testdata/testlib/ examples (run by TestTestExamples), and tests
  of every message and runner case.

## 2026-10-06: the first hybrid library (not yet released)

- Builtin libraries can be written partly in Turtle: `evaluator/lib/
  <name>.t` is built in (Go's embed) and runs once per program on first
  import; only the functions listed for the builtin are exported, the
  rest are private helpers; errors point at the caller's line.
- `random`'s `pick`, `shuffle`, `sample` and `chance` are now Turtle
  (lib/random.t) on the Go sentence and seed; the caller's `seed` still
  decides their values. Their errors are now kind `custom`.
- `change x to list` / `set` take a map (its keys) and a string (its
  characters); `change m to keys` / `values` pick a map's keys or values.
- docs/contributing.md: how to write library functions in Turtle.

## 2026-10-06: put, Go shuffle and sample, a speed test (not yet released)

- Timing the Turtle-written random functions against Go: `chance` was
  2x slower, `pick` 16x, `shuffle` / `sample` 14x to 180x and growing
  with size (Turtle had no way to set an item, so shuffle removed items
  one by one). `shuffle` and `sample` are Go again; `pick` and `chance`
  stay Turtle.
- `put`: set the item at a position. `put 99 to nums at 2 .`,
  `nums at put[2, 99]`, `r is nums at put 2, 99 .`. A Turtle shuffle
  with put (Fisher-Yates) takes 289µs for 1,000 items (was 700µs).
- `testdata/speed/test_speed.t`: 32 timed tests across the language
  (`turtle test testdata/speed`); the Go tests run it once unbenchmarked.

## 2026-10-06: Windows test fix (v0.9.146)

- v0.9.145's CI and release failed on Windows only: TestSpeedFile looked
  for "\nbenchmark = true\n", but Git on Windows checks files out with
  \r\n line endings. The test now normalizes them; the interpreter
  itself already handled both (checked with a \r\n copy).
