# Architecture

Internals documentation for anyone maintaining or extending the
interpreter. If you just want to write Turtle programs, see
[`tour.md`](tour.md) and [`reference.md`](reference.md) instead. For the
concrete list of legacy behaviors this replaced and why, see `SPEC.md` at
the repo root.

## Pipeline

```
source text
   |  lexer.New / (*Lexer).NextToken
   v
token stream
   |  parser.New / (*Parser).ParseProgram
   v
*ast.Program (tree of ast.Statement / ast.Expression)
   |  evaluator.New / (*Interpreter).Run
   v
side effects (show, file I/O, sys) + program exit
```

This replaced `Turtle_interpreter.go`, a single ~6,800-line file that
parsed every statement by re-scanning raw strings with
`strings.Contains`/`strings.Index` and kept all interpreter state in
global maps (`variableDict`, `functionDict`, ...). The current design is a
conventional lexer → recursive-descent parser → AST → tree-walking
evaluator, so each stage can be reasoned about independently.

## Packages

| Package | Responsibility |
|---|---|
| `token` | Token types and the keyword table (`token.LookupIdent`) |
| `lexer` | Turns source text into a `token.Token` stream |
| `ast` | AST node types (one Go struct per statement/expression form) |
| `parser` | Recursive-descent parser: tokens → `*ast.Program` |
| `object` | Runtime value types (`Integer`, `Float`, `String`, `Boolean`, `List`, `Set`, `Map`, `Function`) and `Environment` (scoping) |
| `evaluator` | Tree-walking evaluator: `*ast.Program` → executed program |
| `cmd/turtle` | Entry point: resolves a script path, wires the above together |

## Parser conventions

Two different "where does curToken end up" conventions are used
deliberately, and mixing them up is the single easiest way to introduce a
bug here:

- **Statement parsers** (`parseXStatement`) advance `curToken` to the
  *first token of whatever follows* before returning. `ParseProgram` and
  `parseBlockUntil` never call `nextToken()` after a statement — the
  statement parser already did it.
- **Expression parsers** leave `curToken` on their *own last consumed
  token* (standard Pratt-parser convention), so the caller can inspect
  `peekToken` to decide what comes next (e.g. "is there a `,` — another
  list element?").

`parseBlockUntil(stop)` parses statements until `stop()` reports true,
*without* consuming whatever satisfied it — the caller inspects/consumes
the terminator itself. Both `ParseProgram` and `parseBlockUntil` guard
against a stalled statement parser (one that returns `nil` without
advancing) by force-advancing one token if `curToken` didn't change; this
keeps a malformed program from hanging the parser, at the cost of
potentially skipping a token past a genuine syntax error. Parse errors
accumulate in `p.errors` rather than aborting immediately, so one bad
statement doesn't hide the next one.

### Lookahead beyond one token: `peekN`

The parser normally only tracks `curToken`/`peekToken` (one token of
lookahead), but a few constructs need more: recognizing the 4-to-6 token
`if [end]` / `[loop][end]` markers, and scanning a loop header for a
top-level `;` to decide C-style vs. while-style. `peekN(n)` serves this via
a small FIFO buffer (`p.buf`) that `nextToken()` drains in order — it's
just a lookahead cache, not a re-lexing mechanism, so the underlying token
stream is still only produced once.

## The trickiest parsing problems

### If/else nesting via bracket depth

Real Turtle syntax is `if ] cond [ ... else if ] cond [ ... else ] ... if
[end]` (reversed brackets — confirmed against real historical `.trt`
scripts, not invented). A nested if/else chain is written by prefixing
every keyword with one extra leading `[`, and has **no closing marker of
its own** — it implicitly ends the moment a clause at the same-or-
shallower bracket depth appears.

`parseIfChain(tok, nested bool)` handles both the top-level and nested
cases with the same logic; `nested` only changes what happens at the very
end (consume a real `if [end]`, vs. return immediately without consuming,
leaving the terminating bare clause for the enclosing chain to see). The
recursion is natural: a nested chain is just another statement
(`parseNestedIfStatement`) encountered while parsing a clause's body via
the ordinary statement dispatcher, so it generalizes to depths beyond what
any known real program uses without any depth counter — depth 0 and 1
were the only depths in the original interpreter's hard-coded (and
admittedly buggy, by its own code comment) implementation.

### Function calls vs. an if-header's closing bracket

`name[args]` (calls) and the if-header closer (`... [`) are both just
"something followed by `[`" at the token level — genuinely ambiguous
without more context. The rule `parseIdentifier` uses: a real call's
identifier, its `[`, and its first argument must all sit on the *same
source line* (verified true of every real call example, even ones with a
space like `run [30]`); a header-closing `[` is always the last token on
its line.

This needed two separate checks, not one — an earlier version only
verified the `[` and its own following token were on the same line, which
still misparsed `... k` at the end of one line followed by `[loop][end]`
starting the next, because `[` and `loop` happen to share a line with each
other even though `k` doesn't share a line with `[`. Both checks are
necessary:

```go
if p.peekTokenIs(token.LBRACKET) &&
    p.peekToken.Line == tok.Line &&      // '[' is on the same line as the identifier
    p.peekN(2).Line == p.peekToken.Line { // '['s own next token is on the same line as '['
```

### Loop headers and loop closing

`headerHasSemicolon` scans forward (via `peekN`, tracking bracket depth so
a nested `[...]` inside the header doesn't confuse it) for a top-level `;`
to decide C-style vs. while-style — cheaper and simpler than trying to
parse both forms speculatively.

Loop nesting is **not** modeled after the legacy behavior. The legacy
interpreter collected a loop's body as one flat token blob up to the first
`[loop][end]` line, then re-split that blob recursively at execution
time — which happened to let one `[end]` close multiple nested loops, but
only when the nested loop was the last statement in every enclosing loop's
body (anything after it was silently dropped, and it wasn't documented
anywhere as intentional, unlike the if/else bracket convention). The
rewrite instead requires one `[loop][end]` per `[loop][...]`, matched by
ordinary recursive descent (`isLoopEnd` + the closing token sequence in
`parseLoopStatement`) — nesting works at any position in the body, not
just last.

## Evaluator

### Values and mutation

`object.Object` is the runtime value interface. Scalars (`Integer`,
`Float`, `String`, `Boolean`) are immutable value types; `List`, `Set`,
`Map` are always used as pointers (`*object.List` etc.) specifically so
that in-place mutation (`add`, `sort`, ...) is visible through every
reference to the same variable without needing to re-`Set()` it back into
the environment after every operation.

### Scoping

`object.Environment` intentionally has **no parent-chained variable
lookup** — a function body only sees its own parameters, never the
caller's or global variables (matching the legacy language's actual
behavior, confirmed from real example scripts). Function *definitions*
are the one thing that do chain: `Environment.GetFunction` delegates up to
the global environment, so any scope can call any top-level function.

Each call gets `env.NewCallEnvironment()` — a fresh, empty variable map.
This is a deliberate fix: the legacy interpreter stored one mutable
variable map *per function definition*, shared by every call to that
function, which broke recursion (a recursive call would stomp the outer
call's locals mid-execution). Fresh-per-call environments make recursion
work correctly (see `testdata/recursion.t`).

Loops are a partial exception to "no shared scope": a C-style loop's
induction variable lives in the *same* environment as everything else
around it (loops don't get their own scope), which means two loops nested
with the same induction-variable name would clobber each other's
iteration state. `evalLoop` guards against this by snapshotting the
variable's pre-loop value (or noting it didn't exist) and restoring it via
`defer` when the loop exits — so nested loops reusing a name like `i` work
correctly (verified: `testdata/loop.t`, an outer/inner loop pair that both
use `i`).

### Control flow: `Signal` / `ExecResult`

Statements return `ExecResult{Signal, Value}`. `Signal` is one of `SigNone`
/ `SigReturn` / `SigBreak` / `SigContinue`. `evalStatements` stops and
propagates the first non-`SigNone` result it sees; `evalBlock` is just a
thin wrapper over it. This is what lets `return` inside a deeply nested
`if` inside a `loop` inside a function body correctly unwind all the way
out: each layer (`evalIf`, `evalLoop`, the function-call machinery in
`evalCall`) just checks the signal it got back from evaluating its body
and either handles it (a loop catches `SigBreak`) or passes it further up
unchanged (a loop passing `SigReturn` on through, since a loop doesn't
know how to "return" — only the enclosing function call does, in
`evalCall`).

### Errors

There's no recoverable error type — `fatalf` (evaluator package) prints to
stderr and calls `os.Exit(1)`. This matches the legacy interpreter's own
philosophy of crashing hard on most real errors, deliberately chosen over
building a full recoverable-exception system for a scripting language
this size. The one exception: legacy would `os.Exit(1)` on things like a
missing map key from *deep inside* a rendering/display helper with no
context; the rewrite raises these from the specific operation that failed,
with a message naming the operation and the value involved.

## Testing

`testdata/*.t` plus `test.trt` at the repo root are the closest thing to a
regression suite right now — real scripts (several copied from the
project's own historical `.txt` example files, which is how the real
if/else and loop syntax got confirmed in the first place) plus hand-written
coverage for recursion, data structures, and loop control flow. There is
**no automated pass/fail harness** — verification so far has been running
each script and manually checking the output against hand-computed
expected values. Building an actual test runner (e.g. `expected-output.txt`
per script diffed on each run) is a known gap; see `PROGRESS.md`.
