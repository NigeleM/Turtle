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
| `object` | Runtime value types (`Integer`, `Float`, `String`, `Boolean`, `List`, `Set`, `Map`, `Function`, `Assembly`, `None`, `Error`, `Date`, `Database`) and `Environment` (scoping) |
| `evaluator` | Tree-walking evaluator: `*ast.Program` → executed program; the builtin libraries live here (`jsonlib.go`, `timelib.go`, `httplib.go`, `sqllib.go`, ...) |
| `sqlite` | A SQLite file reader written from scratch (no dependencies): file format, B-trees, records, schema, and a SQL `SELECT` engine. Knows nothing about Turtle values; `evaluator/sqllib.go` adapts it |
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

`object.Environment` is a chain of scopes with **lexical** lookup:
`Get` walks from the current scope out through each enclosing one to the
global scope. A call's scope encloses the called function's *defining*
scope (`object.Function.Env`), never the caller's — so a top-level
function reads globals, and a nested `def` (a closure) also reads the
locals of the call it was defined in. `Set` only ever writes to the
current scope, so assignment never writes through to a global or a
captured variable; it shadows it.

Top-level function definitions live in a separate table on the global
environment (`GetFunction`/`DefineFunction`), callable by name from any
scope. A nested `def` is instead stored as an ordinary local variable
holding the `*object.Function`. Identifier evaluation falls back to the
function table, which is how `f = add` turns a top-level function into a
value.

The main program and every imported `.t` module each have their own
root environment (`object.NewGlobalEnvironment`), and each root records its
own imports (`AddImport`/`Imports`/`FindImport`, holding `object.Import`
and `object.Module`). `Interpreter.loadModule` caches modules by resolved
path, so each runs once, and it tracks the in-progress chain to report
circular imports. An unqualified call resolves in `evalCall` in this
order: a variable holding a function, the file's own top-level def, then
`resolveImported`, which fails on a clash rather than guessing. A
qualified `m name` (two identifiers on one line, recorded in
`Identifier.Module`/`CallExpression.Module`) goes straight to
`qualifiedImport`. The `math`/`time` gates in `requireModule` check the
*current file's* imports.

`assemble Name [fields]` creates an `object.Shape` and stores its
constructor as an ordinary `*object.Function` with `Shape` set, in the
function table or as a local exactly like a `def`. That's why import,
export, clash and value rules need no special cases. `callFunction` builds
an `object.Assembly` instead of running a body when `Shape` is set.

Each call gets `object.NewEnclosedEnvironment(fn.Env)` — a fresh, empty
variable map.
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

A runtime error is a `fatalError` panic: `fatalf`/`fatalKind`
(evaluator.go) build it with the message, its kind (`file`, `number`,
`math`, `index`, `key`, `name`, `type`, `json`, `date`, `http`, `sql`, `custom`), and the line and file
it happened in. Two package-level variables track where code is running:
`currentLine` (set by every statement) and `currentFile` ("" for the main
script, "lib/utils.t" for a module; switched by `callFunction` and
`loadModule`, which restore both when they return or unwind).

`Interpreter.Run` recovers a `fatalError` and returns it; the CLI prints
it and exits 1. A `safe` block (`evalSafe`/`evalProtected`) recovers one
first if its kind is listed, stores an `object.Error` in the handle
variable, and runs the handle code outside the recover, so an error there
isn't handled by its own block. Three things are never handled: a panic
that isn't a `fatalError` (an interpreter bug), `ExitRequest` (system's
`exit[code]`), and a parse error in an imported module (`fatalError.parse`).
Errors come from the specific operation that failed, with a message
naming the operation and the value involved.

## Testing

`go test ./...` runs everything. `go vet ./...` should be clean too.

- `parser/`, `lexer/`, `object/` tests cover syntax and values;
  `evaluator/evaluator_test.go` runs Turtle source and checks its output
  or error, for every feature and library.
- `testdata/everything.t` uses every feature and checks its own results;
  `testdata/shop/` is a whole program (modules in `lib/`, JSON data) with
  its own checks. Both run in `go test`.
- `sqlite/sqlite_test.go` builds databases with the `sqlite3` tool and
  checks that this package's answers match real SQLite, query by query
  and at several page sizes. It skips if `sqlite3` (3.33+) isn't
  installed; the package itself never uses the tool.
- `testdata/books.db` is a small committed SQLite file the Turtle-level
  tests read.
- The historical scripts (`*.txt`, `test.trt`, `testdata/*.t`) should keep
  producing the same output.
