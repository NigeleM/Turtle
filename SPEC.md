# Turtle Language Specification (for the rewritten interpreter)

This documents the Turtle language as implemented by the rewrite: a real
lexer -> parser -> AST -> tree-walking evaluator, under `token/`, `lexer/`,
`ast/`, `parser/`, `object/`, `evaluator/`, replacing the single-file
`Turtle_interpreter.go`, which parsed every statement by re-scanning raw
strings with `strings.Contains`/`strings.Index` and kept all state in
global maps.

Surface syntax is unchanged from the legacy interpreter and is verified
against real historical example scripts recovered for this project
(`test.trt`, `if.txt`, `define.txt`, `play.txt`, `loop.txt`, `show.txt`).
Where the legacy implementation was a non-functional stub (`[write]`,
`[append]`, `[directory]` only ever printed debug text), had a documented
open limitation (if/else nesting capped at one level — see the old
`ifelse` comment "working on a better algorithm"), or was an outright bug
(shared mutable function locals breaking recursion; loop induction
variables that never actually updated; `break` that didn't break), this
rewrite implements the evidently-intended behavior rather than
reproducing the defect. Each such deviation is called out explicitly
below as **(fixed)**.

## Lexical elements

- **Comments**: `// ...` to end of line. `//* ... *//` block comment,
  may span multiple lines.
- **Identifiers**: letter or `_`, then letters/digits/`_`.
- **Numbers**: integers (`42`) and floats (`3.14`).
- **Strings**: double-quoted; `\n` is a real escape (used constantly in
  legacy scripts, e.g. `"he is a grown man \n , \n great day"`).
- **Booleans**: `true` / `false` (as eval'd condition results).
- One statement per source line, except explicit multi-line blocks with
  their own begin/end markers (function/if/loop bodies, `[write]`,
  `[append]`, `[read]`, `[directory]` blocks).

## Statement terminators

Confirmed from every real example: plain assignment, `return`, and input
prompts never take a trailing `.`; `show` and data-structure "sentence"
operations always do.

```
x = 5                 // assignment: no period
return x              // return: no period
name = ? "prompt"     // input: no period
show x .              // show: period required
add 5 to myList .     // data op: period required
```

## Expressions

Precedence, lowest to highest: `||`, `&&`, `== !=`, `< <= > >=`, `+ -`,
`* /`, unary `-`, primary.

`+` is overloaded: numeric addition when both operands are numbers,
string concatenation otherwise (matches the legacy dual behavior visible
throughout `show.txt`).

Function calls are written `name[arg1, arg2, ...]` — Turtle uses `[...]`
for call/definition argument lists, not `(...)` — and may nest inside
expressions, e.g. `hello[a] + 1` or `show hello[a] ," ", hello[80] .`.

## Variables

```
x = 5
name = "Nigele"
y = x + 1
```

## Show

```
show <expr> [, <expr>]* .
```

Pieces are concatenated with **no separator** (confirmed: `show 29 + 1 ,
" is Nigele's age ", ... .` relies on the literal spaces already inside
the string pieces for spacing).

## Input prompts

```
name = ? "What is your name?"
```

Prints the prompt, reads one line from stdin, assigns it as a string.

## Functions

```
def add[a, b]
  result = a + b
  return result
def [end]

sum = add[1, 2]
```

- A function body sees its declared parameters as local variables and
  can **read global (top-level) variables**; it never sees its caller's
  locals. Assignment inside a function always creates/updates a local,
  never a global. It can call any other top-level function (confirmed:
  `hello[]` calls `multiply[10,50]` and `definition[12,9]`).
- **(new)** Functions are values and nested `def`s are closures with
  lexical scoping — see `docs/reference.md` §Functions. Captured variables
  are read-only by assignment, by the same rule as globals.
- **(new)** `gives` makes anonymous functions (`x gives x + 1`, block form
  closed by `gives [end]`), and `a f args` calls `f[a, args]`
  sentence-style. Library verbs like `data`'s `process`/`keep` are
  ordinary functions, not syntax. See `docs/reference.md` §Functions.
- **(new)** `assemble Name [fields]` declares a named type; `Name[...]`
  builds one, `field of v` reads a field, `field of v = x` changes it.
  See `docs/reference.md` §Assembled types.
- **(new)** For-each loops `[loop][x in c]` / `[loop][k, v in c]`, and
  `+`/`-` on two lists, sets, or maps. See `docs/reference.md`.
- **(new)** A function without `return`, or with a bare `return`, yields
  `none` (legacy/earlier rewrite: integer `0`). `none` is a new literal and
  reserved word; see `docs/reference.md` §None.
- `return <expr>` ends the call and the value is usable directly in an
  expression (`a = hello[C]`, `a = a + 1.00`, `show hello[a] ,...`).
- **(fixed)** Each call gets its own fresh local environment/frame.
  Legacy stored one shared `funcVariableDict` map per function
  *definition*, reused mutably by every call — breaking recursion and
  any re-entrant/nested call to the same function. The rewrite gives
  every call a fresh frame, so recursion works.
- **(fixed)** `return` yields its value directly to the call expression.
  Legacy located the caller's variable by linearly scanning all
  variables for one whose current string value happened to contain the
  callee's function name as a substring — fragile and wrong whenever two
  calls to the same function were outstanding, or another variable's
  value happened to contain the name.

## Conditionals

Real syntax uses **reversed brackets** — confirmed byte-for-byte against
`if.txt`, `define.txt`, `play.txt`:

```
if ] a > b [
    show "hello nigele" .
else if ] 89 > 99 [
    show "..." .
else ]
    show "..." .
if [end]
```

Nesting: prefix every clause keyword of the inner if-statement with one
extra literal `[`:

```
if ] 189 > 99 [
    show "hello nigele" .
    [if ] 187 > 89 [
        show "cold weather" .
    [else if ] 69 > 89 [
        show "99 > 89" .
    [else ]
        show "cold life" .
else if ] 89 > 99 [
    ...
else ]
    show "life." .
if [end]
```

Note the nested block has no `[end]` of its own — it implicitly closes
when a clause at the *same or shallower* bracket-depth is reached (a
bare `else if`/`else`/`if [end]` at depth 0, in the example above).

- **(fixed)** Arbitrary nesting depth. The legacy `ifelse` state machine
  used plain booleans (not a stack/counter) and only ever tracked one
  extra level, by its own admission ("working on a better algorithm").
  The rewrite generalizes the same bracket-depth convention (count of
  literal `[` immediately preceding the clause keyword, with nothing
  else in between) to arbitrary depth via real recursive descent.
  Depth-0 and depth-1 programs parse identically to before; depth 2+
  (never used in any real example, and previously guaranteed to
  misparse) now works correctly instead of silently producing garbage.
- Condition operators: `> < >= <= == != && ||`, with normal grouping via
  `(...)` inside the condition.

## Loops

Two header forms inside `[loop][...]` ... `[loop][end]`, both confirmed
by real usage (`loop.txt`):

```
[loop][i = 0; i <= 5; i++]   // C-style: init; condition; post
[loop][end]

[loop][i < 5]                // while-style: condition only, re-evaluated
[loop][end]                  // each pass (from a commented-out example)
```

Post-clause accepts `i++`, `i--`, or a full `i = <expr>` assignment.

- **(fixed)** The induction variable is a real Turtle variable, visible
  and updated in the environment on every pass. Legacy's native Go
  counter drove iteration internally but never wrote its value back into
  `variableDict`, so `show i` inside a C-style loop printed the same
  (initial) value every single iteration.
- **(fixed)** `break` exits the loop; `continue` skips to the next
  iteration's condition check. Legacy's `break` only escaped the
  per-line token scan for the current pass (never the outer native `for`
  loop), so it never actually stopped a Turtle loop early; `continue`
  was a complete no-op.
- **Nesting**: each `[loop][...]` is matched with its own `[loop][end]`
  (ordinary balanced nesting, arbitrary depth). This is a deliberate
  deviation from one specific legacy artifact: legacy's flat
  single-`[end]`-closes-everything collection meant a single trailing
  `[loop][end]` could close multiple nested loops, but *only* if the
  inner loop was the last statement in every enclosing loop's body —
  anything the outer loop needed to run after a nested loop was silently
  dropped. That was never a documented feature (no comment describes it,
  unlike the if/else nesting cap), so it is treated as a bug, not a
  syntax contract: nested loops now need their own `[end]`, and, unlike
  before, may appear anywhere in the body, with statements after them
  running normally.

## Data structures

```
a = list [1, 2, "three"]
s = set [1, 2, 2, 3]
m = map ["k1":1, "k2":2]
```

### Statement-form operations (all require a trailing period)

```
add 5 to a .
remove 5 from a .        // list/set
delete "k1" from m .     // map only
sort a .                 // list/set
reverse a .              // list/set
insert 5 to a at 0 .     // list only
length of a .            // prints
min of a .               // prints (list/set element, or map key)
max of a .                // prints
```

### Method-call form

```
result is a at get 0 .
result is a at count 5 .
result is s1 at union s2 .
result is m at getKeys .
result is a .              // plain alias/assign, no "at"
```

Supported methods — list: `add len toString clear count index sort
remove reverse pop find insert length get`; set: all list methods except
`count`/`copy` plus `union intersection difference subset superset`;
map: `get getValues getKeys add delete invert toString`.

- **(fixed)** `at invert` on a map used to panic (legacy allocated the
  result struct without initializing its underlying Go map before
  writing into it).
- **(fixed)** `at copy` and `at delete` (the method-chain spelling of
  map deletion) used to be silently-parsed no-ops; both now actually do
  the operation their name promises.
- **(fixed)** Map iteration order (`toString`, `getKeys`, `getValues`)
  is deterministic (insertion order), not Go's randomized map order.
- **(fixed)** List/set/map operations inside a function body now
  correctly read/write that function's local scope. (Legacy's method
  chain form re-used the global-scope code path even when called from
  inside a function.)

## Files

```
[read] file.txt to lines [end]        // lines = list, one element per line

[write] file.txt
"first line, written verbatim"
somevar
[end]

[append] file.txt
"more text"
[end]

[directory] . to entries [end]        // entries = list of directory entries
```

`[write]`/`[append]` body items are one per line: a quoted string is
written as-is; a bare identifier is replaced with that variable's
current value. Items are newline-joined.

- **(fixed)** `[write]`/`[append]` actually write to disk. In legacy
  these were 100% non-functional stubs (`Files.Filewrite`/`Fileappend`
  only ever printed debug text; the rich syntax was documented in a
  comment but never implemented) — there was no working behavior to
  preserve, so this rewrite implements the documented intent.
- **(fixed)** `[directory]` is implemented (was an empty no-op branch in
  legacy with no code anywhere to reference).

## Modules

```
import util
```

- **(changed)** Legacy, and the rewrite until now, ran `util.t` in the
  importer's global environment (flatten-into-globals), so names from
  different files silently overwrote each other. Now each `.t` module runs
  once in its own global scope and exports only its top-level functions.
- `import m [a, b]` limits what's imported. A function can be qualified
  by module (`time now[]`, two identifiers side by side), and an
  unqualified call to a name that two imports provide is a fatal error
  asking for the qualified form. Full rules are in `docs/reference.md`
  §Modules.

## Shell escape

```
sys echo hello
```

Runs the rest of the line through a shell, inheriting stdin/stdout/
stderr. Deliberately dangerous, on par with Python's `os.system` — kept
because the user asked to preserve it as a documented feature.

- **(fixed)** Implemented with `os/exec` instead of cgo's `system()`, so
  the interpreter no longer needs a C toolchain to build.
- **(fixed)** Legacy triggered on a bare *substring* match for "sys"
  anywhere in a line (so `show "system" .` would misfire into shell
  exec) and stripped every occurrence of the substring "sys" from the
  line before running it (mangling the command). The rewrite makes `sys`
  a real leading keyword token recognized only at the start of a
  statement, with the remainder of the line taken verbatim as the
  command.
