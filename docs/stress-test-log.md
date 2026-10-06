# Stress test log

A record of stress-testing the interpreter: what was tried, what broke,
and what happened to each finding. Newest run first.

## How the stress test works

Two layers:

1. **`testdata/everything.trt`** (with `everything_lib.trt`): one script that
   uses every feature together and checks its own results with a small
   `check[label, got, want]` function. It prints one line per section and
   exits with code 1 if any check fails. `go test` runs it, so it can't
   silently break. To run it by hand:

   ```sh
   cd testdata
   echo 5 | turtle everything.trt first "second arg"
   ```

2. **Edge-case probes**: small separate programs, each run on its own
   (so one crash can't hide the others) and compared with the output a
   user would expect. They cover lexing, operators, input, conversions,
   conditionals, every loop form, functions and closures, recursion,
   lists/sets/maps, strings, modules, the `data`/`strings`/`system`
   libraries, `assemble`, files, parsing edge cases (tabs, Windows line
   endings, comments), and combinations of features. Most findings were
   turned into `go test` cases when fixed.

## Run 2: 2026-10-01 (after v0.9.135)

**Result: 148 of 151 probes pass; `everything.trt` passes every check.**
The 3 that don't pass are deliberate design limits (below).

New probes in this run covered the features added since run 1 (float
display, quoted text in collections, typed map keys, integer limits,
`isEmpty`, falsy empty collections, sentence calls without parentheses)
and combinations of features: assembled values as map values, `keep`
with `isEmpty`, sets of maps, lists as map keys, closures over fields,
recursion over collections, nested maps of lists, chained method calls
in conditions, deeply nested data.

| # | Finding | Status |
|---|---|---|
| 1 | A reserved word used as a name (`max = 5`, `def show[...]`, a field called `length`, a loop variable `max`) gave 4+ confusing parse errors; `def f[list]` was silently accepted | **Fixed**: one clear error, e.g. `"max" is a reserved word, so it can't be used as a variable name — pick another name (e.g. max_value)` |
| 2 | `!r at isEmpty` was read as `(!r) at isEmpty` | **Fixed**: `!` applies to the whole method call, like Python's `not r.isEmpty()` |
| 3 | A literal list/set/map, or a call's result, couldn't be the left side of a sentence-style call (`list [1, 2] join ","`, `copy[nums] process ...`) | **Fixed** |
| 4 | `"abc" find "c" - 1` is `find["abc", "c" - 1]`: a sentence's argument takes the rest of the expression | **By design**, documented (same rule as Ruby). Store the result first to do arithmetic on it. |

## Run 1: 2026-10-01 (v0.9.13)

**Result: 100 of 111 probes passed.** Fixed in v0.9.135 unless noted.

| # | Finding | Status |
|---|---|---|
| 1 | A function created inside a loop forgot the loop variable once the loop ended (`undefined variable "x"`) | **Fixed**: each loop pass has its own scope for the loop names |
| 2 | Infinite recursion crashed Go itself with pages of stack dump | **Fixed**: stops at 100,000 calls in progress with "recursion too deep … missing its stopping case?" |
| 3 | Integer overflow wrapped silently (`9223372036854775807 + 1` became negative) | **Fixed**: 64-bit limits (industry standard), overflow is an error |
| 4 | Errors from library functions said "method `find`" | **Fixed**: says "function" |
| 5 | A sentence-style call inside `[...]` swallowed the outer commas, so parentheses were needed | **Fixed**: one argument inside brackets; no parentheses needed |
| 6 | Map keys always became text (`map [1: "one"]` gave back `"1"`, so `k + 1` was `"11"`) | **Fixed**: keys keep their type |
| 7 | `get -1` was an error while `slice` accepts negatives | **By design**: negative positions are for slicing only |
| 8 | An empty list, set or map counted as true | **Changed**: empty collections are false (the Python rule); new `isEmpty` |
| 9 | Floats lost their decimal point (`4.0` showed as `4`) | **Fixed**: floats show as floats; text inside collections is quoted (`[ 1, "1" ]`) |
| 10 | Reserved word as a name gave a cascade of errors | **Fixed in run 2** (see above) |
| 11 | A function can't update a global variable | **Design limit**: assignment inside a function is local; change a global list/map instead |
| 12 | A function can't be called before its `def` line | **Design limit** |
| 13 | The C-style loop counter is gone after the loop | **Design limit** |
| 14 | No way to delete a file | **Open** (a `system` function would cover it) |
| 15 | Adding to a set got slower as it grew (100,000 adds ≈ 13 s) | **Fixed**: indexed; now 0.03 s |

Also found while writing `everything.trt`: a method call at the end of an
if-header (`if ] x at isEmpty [`) swallowed the header's closing `[`.
**Fixed.**

## Performance (run 2)

Measured on the author's Mac with the release build.

| Workload | Time |
|---|---|
| 1,000,000 loop additions | 0.12 s |
| 100,000 list adds | 0.02 s |
| 100,000 set adds | 0.03 s |
| 20,000 map adds | 0.01 s |
| for-each over 100,000 elements | 0.03 s |
| `process` over 100,000 elements | 0.03 s |
| `fib[25]` (about 250,000 recursive calls) | 0.05 s |
| Recursion 99,000 calls deep | 0.10 s |
| Infinite recursion | stops cleanly in 0.09 s |

## Still open

- **Error handling:** every runtime error still ends the program; there's
  no way to catch one. The biggest remaining gap (see `TODO.md`).
- Deleting files (finding 14 of run 1).
- The three design limits above, if they're ever revisited.
