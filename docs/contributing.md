# Contributing / extending the language

How to add new syntax or standard-library functionality, following the
existing patterns. Read [`architecture.md`](architecture.md) first if you
haven't — this assumes you know the pipeline and the parser conventions.

The fastest way to spec a new feature is a concrete example of the syntax
and behavior you want (real Turtle-code-shaped snippets, even informal
ones) — that's exactly how the real if/else and loop syntax got nailed
down for this rewrite (from the project's historical `.txt` example
scripts), and it removes guesswork about edge cases.

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

That's it. Update the method table in `docs/stdlib.md`, add a line to a
`testdata/*.t` script exercising it, done.

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
and `docs/tour.md` if it's a headline feature, and add a `testdata/*.t`
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
  `docs/stdlib.md` (if it's a data-structure/file/sys feature),
  `docs/tour.md` (if it's a headline feature worth teaching), and
  `SPEC.md` at the repo root if it changes or clarifies intended semantics
  versus the legacy interpreter.

## Testing

There's no automated test runner yet (see `architecture.md`'s Testing
section) — add a `.t` script under `testdata/` exercising the new feature,
run it with `go run ./cmd/turtle testdata/yourscript.t`, and check the
output by hand against what you expect. Building a real
expected-output-diffing harness is on the list; volunteering to build one
is very welcome.
