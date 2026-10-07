# Standard library proposal (not yet implemented)

Concrete syntax for `TODO.md`'s open item #1 ("Expand the standard
library"). Nothing in this file is built — it's the "bring concrete
example syntax" step `contributing.md` and `TODO.md` both ask for before
implementation starts. Once a section here is agreed, follow
`contributing.md`'s five-layers checklist to build it, and fold the
agreed tables into `docs/stdlib.md`/`docs/reference.md` — this file goes
away once its contents are real.

Priority order and rationale: strings first (highest value, fits the
existing method-call pattern with zero grammar changes), then list/set
higher-order methods (small addition, real practical win), then time
(small, low-risk, lower value), then HTTP (recommended to skip — see
bottom).

## 1. Strings

Extends the existing `<result> is <receiver> at <method> [<args>] .`
pattern (already used by list/set/map) to string receivers. This is the
same shape as the "worked example: a new data-structure method" in
`contributing.md` — no grammar changes, just new cases in the method
dispatcher keyed on receiver type. `length of <string>` is **already
implemented** (see `docs/stdlib.md`) — not repeated here.

```
clean  is raw at trim .
loud   is name at upper .
quiet  is name at lower .
parts  is line at split ", " .
found  is s at contains "abc" .
starts is s at startsWith "Mr" .
ends   is s at endsWith "!" .
fixed  is s at replace "foo", "bar" .
piece  is s at substring 0, 3 .
where  is s at indexOf "abc" .
```

| Method | Args | Returns | Fatal error |
|---|---|---|---|
| `trim` | — | string with leading/trailing whitespace stripped | — |
| `upper` | — | uppercased string | — |
| `lower` | — | lowercased string | — |
| `split` | separator (string) | `list` of strings | — |
| `contains` | substring | Boolean | — |
| `startsWith` | prefix | Boolean | — |
| `endsWith` | suffix | Boolean | — |
| `replace` | old, new | string, **all** occurrences of `old` replaced with `new` | — |
| `substring` | start, end | substring `[start, end)` | start/end out of range |
| `indexOf` | substring | first index found, or `-1` | — |

Companion **list** method (receiver is a list of strings, not a string —
the inverse of `split`):

```
csv is words at join ", " .
```

| Method | Args | Returns | Fatal error |
|---|---|---|---|
| `join` | separator (string) | string, elements joined with separator | element isn't a string |

**Implementation notes**: evaluator-only change (layer 5 in
`contributing.md`'s terms) — `is ... at ...` already parses any
identifier as a method name against any receiver type; this just adds a
`stringMethod` switch alongside the existing `listMethod`/`setMethod`/
`mapMethod`, plus one `join` case in `listMethod`. No token, lexer, ast,
or parser changes.

## 2. List/set higher-order methods

> **Superseded (2026-09-30).** Built as the `data` library's `process`
> (map) and `keep` (filter), taking real function values (`x give x + 1`
> or a named function) rather than a name passed as a string. See
> `stdlib.md` §Data library. `reduce` isn't built yet.

`map`/`filter`/`reduce` over a list or set, dispatched to an existing
top-level function **by name, passed as a string** — deliberately not a
new "function value" type or lambda syntax (see the "high order
functions" discussion this proposal follows from: without closures, full
first-class functions buy little and cost a real grammar/object-model
change). Passing the function name as a plain string argument means this
needs *zero* grammar changes — same reason as the string methods above.

```
def double[x]
  return x * 2
def [end]

def isEven[x]
  return x % 2 == 0
def [end]

def add[a, b]
  return a + b
def [end]

doubled is nums at map "double" .
evens   is nums at filter "isEven" .
total   is nums at reduce "add", 0 .
```

| Method | Args | Returns | Fatal error |
|---|---|---|---|
| `map` | function name (string) | new list: `fn(x)` applied to every element | name isn't a defined function, or function doesn't take exactly 1 arg |
| `filter` | function name (string) | new list: elements where `fn(x)` is `true` | name isn't a defined function, or `fn` doesn't return Boolean |
| `reduce` | function name (string), initial value | single value: `fn(acc, x)` folded left to right, starting from `initial` | name isn't a defined function, or function doesn't take exactly 2 args |

**Implementation notes**: also evaluator-only. Each case resolves the
string argument against the same top-level function table ordinary
`name[args]` calls already use, then invokes it once per element with a
fresh call frame (already how every function call works, per `SPEC.md`'s
"(fixed)" section on call frames). `%` (modulo, used in `isEven` above)
doesn't exist yet either — see `contributing.md`'s own worked example for
exactly how to add it; worth doing in the same pass since `filter`
examples are much weaker without it.

## 3. Time

Kept deliberately minimal — no strftime-style format-string parsing
(inconsistent with how little format-string complexity exists anywhere
else in the language). Three no-arg builtin functions, callable like any
other function, plus the `sleep` statement `contributing.md` already
uses as its own worked example (proposed for real here, not just as a
teaching example):

```
t = now[]                  // Integer, unix epoch seconds
d = today[]                // String, "2026-09-16"
c = clock[]                // String, "14:03:07"

sleep 500 .                // pause 500ms
```

| Name | Args | Returns |
|---|---|---|
| `now[]` | — | Integer: unix epoch seconds |
| `today[]` | — | String: `YYYY-MM-DD`, local time |
| `clock[]` | — | String: `HH:MM:SS`, local time |
| `sleep <expr> .` | Integer milliseconds | (statement; no value) |

**Implementation notes**: `now`/`today`/`clock` need a small builtin
registry checked before the "is this a user-defined function" lookup in
call evaluation (layer 5) — no token/lexer/ast/parser changes, since
`name[]` call syntax already exists. `sleep` is a new statement and needs
all five layers; `contributing.md`'s worked example is the literal
implementation to use, unchanged.

## 4. HTTP — not recommended yet

Skip this rather than build it now. Reasons, unchanged from the earlier
discussion:

- Biggest surface area of the three gaps named in `TODO.md`, with the
  least precedent in the existing language to design against (strings
  and time both map cleanly onto patterns Turtle already has; HTTP does
  not).
- Real HTTP needs error handling, timeouts, and status/response
  structure — all things the language's "errors are fatal, no exception
  mechanism" model (`contributing.md`) handles awkwardly at best.
- `sys curl ...` already covers the "make a request" case given the
  existing shell escape, for whatever the actual motivating use case
  turns out to be.

If a concrete script shows up that genuinely needs in-language HTTP
(not shellable via `sys`), revisit with that script as the spec input —
per `contributing.md`'s own rule, a real example beats a speculative
design.
