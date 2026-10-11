# Contributing / extending the language

How to add new syntax or standard-library functionality, following the
existing patterns. Read [`architecture.md`](architecture.md) first if you
haven't — this assumes you know the pipeline and the parser conventions.

The fastest way to spec a new feature is a concrete example of the syntax
and behavior you want (real Turtle-code-shaped snippets, even informal
ones); it removes guesswork about edge cases.

## Contributor terms

- You keep the copyright in what you contribute, and license it to the
  project permanently under Turtle's license.
- What goes into official Turtle is Nigele McCoy's decision.
- You confirm the contribution is yours to give.

Outside code contributions open once a contributor agreement is ready;
issues with ideas and bug reports are welcome now.

## The five layers you might touch

Not every change needs all five:

1. **`token`** — only if you're introducing a new reserved word or
   operator symbol. Add a constant and, for keywords, an entry in the
   `keywords` map.
2. **`lexer`** — only if the new syntax needs special tokenizing (a new
   operator character, a new escape sequence). Most new *statements*
   don't need lexer changes at all, since they're built from tokens that
   already exist (`IDENT`, `LBRACKET`, etc.) plus one new keyword token.
3. **`ast`** — a new Go struct for the new statement or expression, plus
   its `statementNode()`/`expressionNode()` and `TokenLiteral()` methods.
4. **`parser`** — a `parseXStatement`/`parseXExpression` function, wired
   into `parseStatement`'s switch (for a statement) or
   `prefixParseFns`/`infixParseFns` (for an expression).
5. **`evaluator`** — a `case` in `evalStatement` or `evalExpression`
   implementing the runtime behavior.

## Worked example: a new data-structure method

Easiest case — no grammar changes at all, since `is <receiver> at <method>
<args>` already parses any identifier as the method name. Say you want
`at contains` on a set (alias for `find`, but returning early / no extra
allocation):

```go
// evaluator/data.go, inside setMethod's switch:
case "contains":
	requireArgs(method, args, 1)
	return &object.Boolean{Value: s.Contains(args[0])}
```

That's it. Update the method table in `docs/library/builtins.md`, add a line to a
`testdata/*.turtle` script exercising it, done.

## Worked example: a new builtin library

Libraries like `json` and `time` are Go code behind `import <name>`; no
parser change is needed, since calls are ordinary `name[args]`.

1. Add the module and its function names to `builtinModules` in
   `evaluator/evaluator.go`. The names gate the import: `import json
   [load]` and the "needs import json first" message both come from here.
2. Write `evaluator/<name>lib.go` with a `call<Name>(name, args)` that
   switches on the function name. Use the helpers: `requireFuncArgs`,
   `asStringArg`, `asIntArg`; fail with `fatalf` (kind `type`) or
   `fatalKind(kind..., ...)`.
3. Dispatch to it from `callBuiltin` in `evaluator/expr.go`.
4. A new kind of error (like `json`, `date`) needs a `kind...` constant in
   `evaluator.go` and its name in `errorKinds` in `parser/parser.go`, so
   `handle [<kind>] e .` accepts it.
5. Document it on its library's page, `docs/library/<name>.md` (table, rules, a runnable example),
   add tests, and a section in `testdata/everything.turtle`.

## Worked example: the standard library in Turtle

A library can be written in Turtle, wholly or in part, with no Go: put a
file in `evaluator/lib/` and rebuild.

- `evaluator/lib/data.turtle`, named after a library written in Go,
  **adds to it**: `import data` gives its Go functions and these alike.
- `evaluator/lib/geometry.turtle`, any other name, is **a new library**:
  `import geometry`.

```
// evaluator/lib/data.turtle
import data                        // here: data's Go half (mean, process, ...)

// clamp keeps each number between low and high.
//   clamp[list [3, 15, -2], 0, 10]      gives [ 3, 10, 0 ]
def clamp[nums, low, high]
    return nums process n give ~limit[n, low, high]
def [end]

// ~limit is a private helper: not exported, not documented.
def ~limit[n, low, high]
    return min of list [max of list [n, low], high]
def [end]
```

- **What's exported:** the file's top-level `def`s, `assemble` types
  and theories. A `~` function or theory is a private helper.
- **Theories:** a theory in the file gives the library a new phrase,
  whether the library is written in Go, in Turtle or both; its abstract,
  notations and theorems are its documentation.
  `TestShippedStdlibInTurtle` proves each one: its proof cases and
  theorems must hold.
- **Documentation:** each function's `turtle doc`, `help` and editor
  hover is the comment above its `def` (`//` lines or a `//* *//`
  block); a new library is described by the comment at the top of its
  file. A function documented by hand in `evaluator/stdlibdocs.go`
  keeps that text.
- **Its own library's Go functions:** `import data` inside
  `data.turtle` means the Go half, so `mean` and `process` are there. It
  can import other libraries as any file can, and its own library's
  sentences work without an import.
- **Names:** a function in Turtle can't have the name of one its library
  has in Go. That, and a file that doesn't parse, stops the build's
  tests (`TestShippedStdlibInTurtle`) with the file and the name.
- **Errors:** `fail "..."` (kind `custom`); they point at the caller's
  line, as a Go function's do.
- **Speed:** code that does a little work around other calls costs little
  more in Turtle (`chance` is about 2× its Go version); code that goes
  through every item of a collection is 15× to 150× slower, so it belongs
  in Go (`shuffle` and `sample` stay there; see
  `testdata/speed/test_speed.turtle`).

`random` is built this way: `shuffle` and `sample` in Go, `pick` and
`chance` in `evaluator/lib/random.turtle`. `evaluator/turtlelibs_test.go`
adds a whole library in Turtle and extends `data`, for its tests. For
colors in VS Code, add a new function's name to the list in
`editors/vscode/syntaxes/turtle.tmLanguage.json`
(`TestVSCodeGrammarIsCurrent` says which).

## Worked example: a new statement keyword

Say you want a `sleep <expr> .` statement (pause execution for N
milliseconds).

**1. Token** (`token/token.go`):

```go
SLEEP Type = "SLEEP"
// ...
"sleep": SLEEP,
```

**2. AST node** (`ast/ast.go`):

```go
type SleepStatement struct {
	Token token.Token
	Value Expression
}

func (s *SleepStatement) statementNode()       {}
func (s *SleepStatement) TokenLiteral() string { return s.Token.Literal }
```

**3. Parser** (`parser/parser.go`) — follow the `parseShowStatement`
pattern (single expression, trailing period, "advance past self"
convention):

```go
func (p *Parser) parseSleepStatement() ast.Statement {
	tok := p.curToken
	p.nextToken()
	val := p.parseExpression(LOWEST)
	if !p.requirePeriod() {
		return nil
	}
	return &ast.SleepStatement{Token: tok, Value: val}
}
```

Wire it into `parseStatement`'s switch:

```go
case token.SLEEP:
	return p.parseSleepStatement()
```

**4. Evaluator** (`evaluator/evaluator.go`), a case in `evalStatement`:

```go
case *ast.SleepStatement:
	ms := it.evalExpression(s.Value, env)
	n, ok := ms.(*object.Integer)
	if !ok {
		fatalf("sleep needs an integer number of milliseconds")
	}
	time.Sleep(time.Duration(n.Value) * time.Millisecond)
	return noneResult
```

**5. Document and test**: add it to `docs/reference.md`'s statement table
and `docs/tour.md` if it's a headline feature, and add a `testdata/*.turtle`
script that exercises it.

## Worked example: a new binary operator

Say you want `%` (modulo). Almost everything already exists — the
precedence table, `evalInfix`'s number-handling — you're mostly filling in
one more case.

1. **Token**: add `MODULO Type = "%"`, and a `case '%':` in the lexer's
   `NextToken` switch.
2. **Parser**: add `token.MODULO: PRODUCT` to the `precedences` map (same
   precedence as `*`/`/`), and `token.MODULO: p.parseInfixExpression` to
   `infixParseFns` in `New`. No new AST node needed — `%` reuses
   `ast.InfixExpression` with `Operator: "%"`.
3. **Evaluator**: add a `case "%":` alongside `-`/`*`/`/` in `evalInfix`
   (`evaluator/expr.go`), requiring two integers (or decide what `%`
   means for floats, if anything).

## Style notes

- **Statement parsers advance past themselves**; expression parsers leave
  `curToken` on their own last token. Getting these backwards is the most
  common way to break something subtly — see the "trickiest parsing
  problems" section of `architecture.md` for two real bugs this caused.
- **Errors are fatal** (`fatalf` in the evaluator package: print to
  stderr, `os.Exit(1)`) — there's no recoverable exception mechanism.
  Match this rather than inventing a new error-handling style for one
  feature.
- **Prefer extending an existing pattern over inventing a new one.** If a
  feature can be expressed as another `is ... at <method>` call, or
  another data-op statement shaped like `add`/`remove`/`sort`, do that
  instead of new bracket syntax — it's less parser surface area and one
  fewer thing for users to learn.
- **Update docs in the same change**: `docs/reference.md` (grammar),
  `docs/library/` (builtins.md or files.md for a data-structure/file/sys feature),
  and `docs/tour.md` (if it's a headline feature worth teaching).

## Testing

There's no automated test runner yet (see `architecture.md`'s Testing
section) — add a `.turtle` script under `testdata/` exercising the new feature,
run it with `go run ./cmd/turtle testdata/yourscript.turtle`, and check the
output by hand against what you expect. Building a real
expected-output-diffing harness is on the list; volunteering to build one
is very welcome.

---

Copyright 2017-2026 Nigele McCoy. Licensed under the
[Apache License 2.0](../LICENSE); see [NOTICE](../NOTICE).
