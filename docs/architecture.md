# Architecture

Internals documentation for anyone maintaining or extending the
interpreter. If you just want to write Turtle programs, see
[`tour.md`](tour.md) and [`reference.md`](reference.md) instead.

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

A conventional lexer → recursive-descent parser → AST → tree-walking
evaluator, so each stage can be reasoned about independently.

## Packages

| Package | Responsibility |
|---|---|
| `token` | Token types and the keyword table (`token.LookupIdent`) |
| `lexer` | Turns source text into a `token.Token` stream |
| `ast` | AST node types (one Go struct per statement/expression form) |
| `parser` | Recursive-descent parser: tokens → `*ast.Program`; its errors carry the spot they point at (`errors.go`: plain words, fixes for the usual mistakes, `Format` with a `^`) |
| `object` | Runtime value types (`Integer`, `Float`, `String`, `Boolean`, `List`, `Set`, `Map`, `Function`, `Assembly`, `None`, `Error`, `Date`, `Database`, `Matrix`), shared small integers and booleans (`Int`, `Bool`), and `Environment` (scoping) |
| `evaluator` | Tree-walking evaluator: `*ast.Program` → executed program; the builtin libraries live here (`jsonlib.go`, `timelib.go`, `httplib.go`, `sqllib.go`, `statslib.go`, `linearlib.go` with its arithmetic in `linalg.go`, `csvfile.go` for .csv/.tsv, ...) |
| `sqlite` | SQLite written from scratch (no dependencies): the file format, table and index B-trees (search, insert, delete, split, merge), overflow pages, the free-page list, the rollback journal and crash recovery, SQLite-compatible file locks (`lock_*.go`), a SQL parser, a query planner and runner (joins, groups, subqueries, `WITH`), the changing statements (`exec.go`), and a file checker (`check.go`). Knows nothing about Turtle values; `evaluator/sqllib.go` adapts it |
| `postgres` | A PostgreSQL client from scratch: the v3 wire protocol, SCRAM-SHA-256 / MD5 / password logins, TLS, `?` → `$n`, values decoded by type. `evaluator/sqllib.go` puts it behind the same `sqlConn` interface as SQLite |
| `mysql` | A MySQL / MariaDB client from scratch: the client/server protocol, `caching_sha2_password` (fast, RSA and TLS paths), `sha256_password` and `mysql_native_password` logins, TLS, prepared statements with binary rows |
| `toml` | TOML 1.0, read and written from scratch, for the `config` library |
| `syntax` | Colors from the real tokens (shared by the REPL and the language server), the words libraries turn on, and function descriptions from comments (`FunctionDoc`) |
| `format` | `turtle fmt` and Format Document: indentation, line ends and blank lines from the tokens, checked to leave the code itself unchanged |
| `lsp` | `turtle lsp`, the language server: errors, completion, hover, definitions, the outline, colors, formatting |
| `repl` | The interactive prompt: line editing, history, colors, raw terminal mode per OS |
| `cmd/turtle` | Entry point: runs a script, or the commands (`test`, `doc`, `fmt`, `trace`, `debug`, `build`, `lsp`, the REPL); a program made by `turtle build` (`bundle.go`) runs its packed script |
| `bench` | The benchmark: the same programs in Turtle, Python, Node, Ruby and Go, run and compared by `bench/run.turtle` (not part of the build) |

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
[end]` (reversed brackets). A nested if/else chain is written by prefixing
every keyword with one extra leading `[`, and has **no closing marker of
its own** — it implicitly ends the moment a clause at the same-or-
shallower bracket depth appears.

`parseIfChain(tok, nested bool)` handles both the top-level and nested
cases with the same logic; `nested` only changes what happens at the very
end (consume a real `if [end]`, vs. return immediately without consuming,
leaving the terminating bare clause for the enclosing chain to see). The
recursion is natural: a nested chain is just another statement
(`parseNestedIfStatement`) encountered while parsing a clause's body via
the ordinary statement dispatcher, so it works at any depth without a
depth counter.

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

Each `[loop][...]` needs its own `[loop][end]`, matched by ordinary
recursive descent (`isLoopEnd` + the closing token sequence in
`parseLoopStatement`), so nesting works at any position in the body.

## Evaluator

### Values and mutation

`object.Object` is the runtime value interface. Scalars (`Integer`,
`Float`, `String`, `Boolean`) are immutable value types, never changed
once made, which is what lets `object.Int` and `object.Bool` share the
common ones (-128 to 1023, true, false) instead of making new ones; `List`, `Set`,
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
captured variable; it shadows it. An assignment that would shadow one
while reading it (`count = count + 1`) is an error (`outerassign.go`).

Top-level function definitions live in a separate table on the global
environment (`GetFunction`/`DefineFunction`), callable by name from any
scope. A nested `def` is instead stored as an ordinary local variable
holding the `*object.Function`. Identifier evaluation falls back to the
function table, which is how `f = add` turns a top-level function into a
value.

The main program and every imported `.turtle` module each have their own
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

Each call gets `object.NewEnclosedEnvironment(fn.Env)`: a fresh, empty
scope. A scope keeps its names in a short list (most hold a few), and
builds a map once it has more than eight.
A recursive call can't overwrite the outer call's locals (see
`testdata/recursion.turtle`).

Loops get scopes of their own that hold only the loop's names
(`object.NewLoopEnvironment`): each for-each pass has one with its names,
and a C-style loop one with its counter. Every other assignment in the
body passes through to the enclosing scope, as if the loop had none. So
two nested loops can both use `i`, an outside variable called `i` is
untouched, and a closure made in a pass keeps that pass's values
(`testdata/loop.turtle`).

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
`math`, `index`, `key`, `name`, `type`, `json`, `date`, `http`, `sql`, `csv`, `custom`), and the line and file
it happened in. Two package-level variables track where code is running:
`currentLine` (set by every statement) and `currentFile` ("" for the main
script, "lib/utils.turtle" for a module; switched by `callFunction` and
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
- `testdata/everything.turtle` uses every feature and checks its own results;
  `testdata/shop/` is a whole program (modules in `lib/`, JSON data) with
  its own checks. Both run in `go test`.
- `sqlite/sqlite_test.go` checks that this package's answers match real
  SQLite, query by query and at page sizes 512 to 65536, without running
  SQLite: `sqlite/testdata/` holds the fixture database at each page size
  and `answers.json`, sqlite3's answer to every query. So the tests give
  the same result on every machine, Windows included. After changing the
  fixture or the queries, run `go test ./sqlite -update` (needs a
  standard `sqlite3` tool) and commit `sqlite/testdata/`. Not just any
  sqlite3 will do: GitHub's macOS one uppercases é (ICU), and 3.45 and
  older write blobs wrongly in `-json`, which is also why blob columns are
  checked through `hex()` and exact bytes rather than `-json`. The package
  itself never uses the tool.
- Writing is checked the same way: `sqlite/scripts_test.go` runs scripts
  of changing statements (constraints, upserts, `ALTER TABLE`,
  `AUTOINCREMENT`, transactions, big values, `STRICT`) and compares every
  table's rows, types, the schema text and which statements failed with
  `sqlite/testdata/writes.json`, made by sqlite3 with `-update` (with
  `legacy_alter_table` off, since Apple's sqlite3 turns it on).
  `write_test.go` makes thousands of random changes at several page sizes
  and compares them with a model in memory. Every test that writes ends
  with `db.Check()` (`check.go`: every page used once, trees in order,
  leaves at one depth, indexes matching their tables).
- When a `sqlite3` tool is installed, the tests also use it, but only for
  things that don't differ between its versions: `PRAGMA
  integrity_check` on the files Turtle wrote, repairing a journal Turtle
  left by a simulated crash (and Turtle repairing one left by sqlite3),
  and taking turns with Turtle through the file locks
  (`crash_test.go`). Without the tool those parts are skipped, and the
  ones that use Unix shell commands skip on Windows.
- `testdata/books.db` is a small committed SQLite file the Turtle-level
  tests read. `testdata/sql/` has one Turtle program per SQL topic, run
  by `TestSQLExamples`.
- The `postgres` and `mysql` packages, and `testdata/sql/13_servers.turtle`,
  need real servers: they run when `TURTLE_PG_URL` / `TURTLE_MYSQL_URL`
  are set (CI starts both as service containers) and skip otherwise.
- The scripts in `testdata/` should keep producing the same output.
