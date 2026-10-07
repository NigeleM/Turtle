# Testing

Turtle's test library checks that your code does what you meant. It has
three words, from simple to thorough, and a command that runs your tests:

| Word | Tests | Example |
|---|---|---|
| `check` | one fact | `check total[order] == 45 .` |
| `verify` | a rule, against every item of a collection | `verify evens[nums] each x give x % 2 == 0 .` |
| `validate` | a rule, against many random inputs | `validate evens[nums] with nums as list of integer that result each x give x % 2 == 0 .` |
| `turtle test` | runs every `test_` function in every `test_*.trt` file | `turtle test` |

The three words are independent: a test can use any one, two, or all
three. They mean something only in a file that has `import test`;
anywhere else `check`, `verify` and `validate` are ordinary names.

Working examples of everything on this page: `testdata/testlib/`
(`turtle test testdata/testlib`).

## A first test

`test_orders.trt`:

```
import test

assemble Order [item, qty, price]

def total[order]
    return qty of order * price of order
def [end]

def test_total[]
    check total[Order["pen", 3, 15]] == 45 .
def [end]

def test_total_of_nothing[]
    check total[Order["pen", 0, 15]] == 0 .
def [end]
```

```
$ turtle test
test_orders.trt
  PASS  test_total              35.0µs
  PASS  test_total_of_nothing   11.9µs

ok: 2 passed, 0 failed (1 file, 2.1ms)
```

If `total` were wrong:

```
test_orders.trt
  FAIL  test_total   12.3µs
        test_orders.trt line 9: failed: check total[Order["pen", 3, 15]] == 45 .
            got 40, want 45 (5 less)
```

## How tests are found and run

- **Test files** are named `test_` + anything + `.trt`: `test_orders.trt`,
  `test_lists.trt`. A test file must `import test`.
- **Test functions** are the top-level functions named `test_` + anything,
  with no arguments: `def test_total[]`. Other functions in the file are
  helpers and don't run on their own.
- **Both** are needed: a `test_` function in an ordinary file is just a
  function, and nothing runs it.

| Command | Runs |
|---|---|
| `turtle test` | every `test_*.trt` in this folder and the folders below it (not hidden ones, like `.git`) |
| `turtle test test_orders.trt` | one file |
| `turtle test tests/ more/` | the files in those folders |

For each file, in order:

1. The file's own top-level code runs **once**: imports, `assemble`s,
   shared values, settings. If it fails, the file's tests don't run.
2. Each `test_` function runs, **in the order written**, each in its
   own scope (its variables start fresh).
3. A test **passes** if it ends without an error. It **fails** at its
   first failed `check`, `verify` or `validate`, or at any other error
   (a division by zero, a missing key ...), which is reported with its
   kind:
   ```
   FAIL  test_crash   1.5µs
         test_orders.trt line 28: stopped with a math error: division by zero
   ```

`turtle test` ends with the totals and exits with 0 when everything
passed, 1 when anything failed, so it works in CI:

```
FAILED: 4 passed, 3 failed, 1 skipped (3 files, 294.0ms)
```

Running a test file the ordinary way (`turtle test_orders.trt`) runs its
top-level code only; the tests run under `turtle test`.

A `show` in a test prints as the test runs, between the report lines.

## Testing your own code

Keep the code in an ordinary `.trt` file and the tests in a `test_` file
that imports it, as any program would. Paths are from the test file's
folder; a library doesn't need `import test`.

```
my_project/
    lib/geometry.trt        // the code
    test_geometry.trt       // its tests
```

`lib/geometry.trt`:

```
assemble Rect [width, height]

def area[r]
    return width of r * height of r
def [end]
```

`test_geometry.trt`:

```
import test
import lib/geometry                // or: import lib/geometry [area, Rect]

def test_area[]
    check area[Rect[3, 4]] == 12 .
    check geometry area[Rect[2, 5]] == 10 .   // naming the library works too
    check Rect[1, 1] is Rect .
def [end]

def test_area_is_never_negative[]
    validate area[r] with r as Rect [integer from 0 to 50, integer from 0 to 50]
        that result >= 0 .
def [end]
```

An error inside the library names the library's file and line:
`lib/geometry.trt line 6: stopped with a math error: division by zero`.

Don't start a library's name with `test_`: `turtle test` would take it
for a test file (and ask for `import test`). See
`testdata/testlib/test_import.trt` for a full example.

## check: one fact

`check` takes anything that is true or false. When it's false, it says why.

### Comparisons

```
check total[o] == 45 .
check total[o] != 0 .
check total[o] > 40 .
check total[o] >= 45 && total[o] < 100 .
check "apple" < "banana" .
```

Write what you worked out on the **left** and what you expect on the
**right**: the message calls them *got* and *want*.

### What a failed `==` says

It finds the first difference, down to the item, key, field or letter:

| Check | Says |
|---|---|
| `check 40 == 45 .` | `got 40, want 45 (5 less)` |
| `check list [1, 2, 3] == list [1, 5, 3, 4] .` | `got 3 items, want 4` / `at 1: got 2, want 5 (3 less)` / `at 3: missing (want 4)` |
| `check list [3, 1, 2] == list [1, 2, 3] .` | `the same items, in a different order` |
| `check map ["a": 1, "b": 2] == map ["a": 1, "c": 3] .` | `key "c": missing (want 3)` / `key "b": not wanted (got 2)` |
| `check map ["a": list [1, 2]] == map ["a": list [1, 3]] .` | `key "a", at 1: got 2, want 3 (1 less)` |
| `check set [1, 2] == set [2, 3] .` | `missing 3` / `not wanted: 1` |
| `check "abd" == "abc" .` | `got "abd", want "abc"` / `first difference at position 2: "d", want "c"` |
| `check Order["pen", 1] == Order["pen", 2] .` | `field qty: got 1, want 2 (1 less)` |
| `check 5 == "5" .` | `got 5 (an integer), want "5" (a string)` |

Other failed checks name their parts:

| Check | Says |
|---|---|
| `check length of nums > 5 .` | `length of nums is 2, which isn't > 5` |
| `check nums at contains[7] .` | `nums is [ 1, 2 ]` |
| `check length of nums > 5 && nums at contains[1] .` | `length of nums > 5 is false` (only the side that was false) |
| `check 4 != 4 .` | `both are 4` |

### Methods and library functions

Anything that gives true or false can be checked, so every method and
library function works:

```
check nums at contains[3] .
check nums at find[3] .
check nums at index[3] == 2 .
check !nums at isEmpty .
check small at subset[big] .
check big at superset[small] .
check "Turtle" at contains["urt"] .
check "42" at isNumber .
check is_sorted[nums] .                         // import sort
check find_first[books, b give price of b > 900] != none .   // import search
```

A method works on the value right before it, the way Python's
`a.invert()` does, so both sides of a comparison can have one:

```
check a at invert == b at invert .              // Python: a.invert() == b.invert()
check "a" + "b" at upper == "aB" .
check ("a" + "b") at upper == "AB" .            // ( ) groups
```

`invert`, `upper` and the like give a new value; `a` and `b` are left as
they were.

### The kind of a value: `is`

```
check x is integer .
check x is float .
check x is number .        // an integer or a float
check x is string .
check x is boolean .
check x is list .
check x is set .
check x is map .
check x is date .
check x is none .
check x is function .
check o is Order .         // an assembled type
check nums is empty .      // a list, set, map or string with nothing in it
check x is not none .      // "is not" for any of them
```

`"42" at isNumber` is **true** (the text looks like a number), but
`check "42" is number .` **fails**: the value is a string.
`check "x" is integer .` says `"x" is a string, not an integer`.

### Decimals: `is close to`

Decimals are stored in binary, so most aren't exact: `0.1 + 0.2` is
`0.30000000000000004`, and `check 0.1 + 0.2 == 0.3 .` fails. Compare
them with `is close to`:

```
check 0.1 + 0.2 is close to 0.3 .              // allows a tiny rounding difference
check 10 / 3 is close to 3.33 within 0.01 .    // allows up to 0.01 either way
check 3.5 is not close to 3 within 0.1 .
```

On failure: `got 0.5, want 0.3 (off by 0.2; allowed: a tiny rounding difference)`.

### Errors: `fails`

When the right answer is an error:

```
check 1 div 0 fails .                       // any error
check 1 div 0 fails [math] .                // an error of this kind
check list [1] at get[5] fails [index] .
check parse_age["abc"] fails [number, custom] .   // one of these kinds
```

The kinds are the ones `safe ... handle [...]` uses: `file number math
index key name type json date http sql csv test custom`. On failure:
`expected an error, but it gave 2`, or `expected a math error, got a type
error: ...`.

## verify: a rule for every item

`verify` goes through a list, set, map or string, tries a rule on each
item, and counts how many follow it:

```
verify <collection> <how many> <rule> .
```

The rule is a function that gives true or false: `x give x > 0`, or a
function's name.

### How many: each, any, not

```
verify evens[nums] each x give x % 2 == 0 .   // every item follows the rule
verify orders any o give qty of o > 100 .     // at least one does
verify ages not a give a < 0 .                // none does
verify list [2, 4, 8] each is_even .           // a function's name
```

### How many: at least, at most, exactly

These count how many items follow the rule and compare that count with
a number. Each one is a plain comparison on the count:

| verify | means | the same as |
|---|---|---|
| `verify xs each R .` | every item follows R | count `==` length of xs |
| `verify xs any R .` | at least one does | count `>=` 1 |
| `verify xs not R .` | none does | count `==` 0 |
| `verify xs at least N R .` | N or more do | count `>=` N |
| `verify xs at most N R .` | N or fewer do | count `<=` N |
| `verify xs exactly N R .` | exactly N do | count `==` N |

Worked through on one list, `rolls = list [6, 2, 6, 3, 5]`, which has
two sixes (the count of `r give r == 6` is 2):

| verify | count vs N | result |
|---|---|---|
| `verify rolls at least 1 r give r == 6 .` | 2 >= 1 | passes |
| `verify rolls at least 2 r give r == 6 .` | 2 >= 2 | passes |
| `verify rolls at least 3 r give r == 6 .` | 2 >= 3 | **fails** |
| `verify rolls at most 2 r give r == 6 .` | 2 <= 2 | passes |
| `verify rolls at most 5 r give r == 6 .` | 2 <= 5 | passes |
| `verify rolls at most 1 r give r == 6 .` | 2 <= 1 | **fails** |
| `verify rolls exactly 2 r give r == 6 .` | 2 == 2 | passes |
| `verify rolls exactly 1 r give r == 6 .` | 2 == 1 | **fails** |
| `verify rolls exactly 3 r give r == 6 .` | 2 == 3 | **fails** |

The edges:

| verify | always | because |
|---|---|---|
| `at least 0 ...` | passes | any count is 0 or more |
| `at most 0 ...` | = `not ...` | no item may follow the rule |
| `exactly 0 ...` | = `not ...` | the same |
| `at least 1 ...` | = `any ...` | one or more |
| `exactly (length of xs) ...` | = `each ...` | all of them |
| `each ...` on an empty list | passes | no item breaks the rule |
| `any ...` on an empty list | fails | no item follows it |

More examples:

```
verify scores at least 3 s give s >= 90 .          // three or more A grades
verify errors at most 2 e give e == "timeout" .    // timeouts are rare
verify cards exactly 4 c give c at contains["A"] . // four aces in the deck
verify week exactly 2 d give d == "Sat" || d == "Sun" .
verify password at least 1 c give c at isNumber .  // a string: its characters
```

### What a failed verify says

```
verify list [1, -2, 3, -4] each x give x > 0 .
    2 of 4 items break the rule:
      at 1: -2
      at 3: -4

verify rolls any r give r == 1 .
    none of the 5 items follows the rule

verify rolls not r give r == 6 .
    2 of 5 items follow the rule, and none should:
      at 0: 6
      at 2: 6

verify rolls at least 3 r give r == 6 .
    2 of 5 items follow the rule; at least 3 should:
      at 0: 6
      at 2: 6

verify rolls exactly 1 r give r == 6 .
    2 of 5 items follow the rule; exactly 1 should:
      at 0: 6
      at 2: 6
```

Up to 10 items are listed, then `... and N more`.

### Neighbors: each pair

`each pair [a, b]` tries the rule on neighbors two at a time: the first
and second item, the second and third, and so on. It's how to say
"sorted" or "always increasing":

```
verify prices each pair [a, b] give a <= b .     // never goes down
verify days each pair [a, b] give b == a + 1 .   // counts up by 1
verify "abc" each pair [a, b] give a < b .       // strings too
```

```
verify list [3, 1, 2] each pair [a, b] give a <= b .
    1 of 2 pairs breaks the rule:
      at 0 and 1: 3, 1
```

`any pair`, `not pair`, `at least N pair` ... work the same way.

### What counts as an item

| Collection | Items | Shown as |
|---|---|---|
| list | its items | `at 3: -3` |
| set | its items | `-3` |
| string | its characters | `at 1: "b"` |
| map, rule of one name | its values | `at key "bo": 25` |
| map, rule of two names | key and value: `[name, age] give ...` | `at key "bo": "bo", 25` |

```
verify ages each a give a >= 18 .
verify ages each [name, age] give name at length <= 3 && age < 100 .
verify grid each row give length of row == 3 .     // nested: look inside
```

## validate: many random inputs

`check` and `verify` test the examples you thought of. `validate` makes
up inputs, 100 by default, runs your function on each, and checks a rule
on every answer. It finds the cases you didn't think of.

```
validate <call> [to <name>] with <input> as <kind>, ... <rule> .
```

```
validate evens[nums] with nums as list of integer
    that result each x give x % 2 == 0 .
```

Read it as: call `evens[nums]` with `nums` as a random list of integers;
the answer (`result`) must have only even numbers. A validate sentence
can go over several lines; it ends with its period.

### The inputs: with ... as

Name each input and its kind, with the words of the random library
(see [`stdlib.md`](stdlib.md#random-library)):

```
with nums as list of integer
with nums as list of 5 to 20 integers from -100 to 100
with name as string of 3 to 8
with a as integer, b as integer
with grid as list of 3 lists of 3 integers
with ages as map of string to integer from 0 to 120
with o as Order [string, integer from 1 to 10, float]   // a kind per field
```

### The rule

| Rule | Passes when | Example |
|---|---|---|
| `that <true/false>` | it's true | `that result >= 0` |
| `that <value> each/any/not/at least/... R` | the verify rule holds | `that result each x give x > 0` |
| `matches <other call>` | both give the same answer | `matches min_sort[nums]` |

The answer is called `result`; `to <name>` calls it something else. The
rule can use the inputs too:

```
validate evens[nums] to found with nums as list of integer
    that length of found <= length of nums .

validate plus[a, b] with a as integer, b as integer
    that result == plus[b, a] .                     // order doesn't matter

validate quick_sort[nums] with nums as list of integer
    matches min_sort[nums] .                        // a new sort against a trusted one
```

The function gets its own copy of each input, so one that changes its
list in place doesn't change what the rule sees.

### What a failed validate says

```
FAIL  test_evens
      test_lists.trt line 12: failed: validate bad_evens[nums] with nums as list of integer that result each x give x % 2 == 0 .
          failed on case 2 of 100 (seed 713144290)
          nums = [ -1 ]   (smallest found; first failed on [ 617, 577, -451, 999 ])
          result = [ -1 ]
          1 of 1 item breaks the rule:
            at 0: -1
          to repeat this run: seed = 713144290
```

- **Shrinking**: the first failing input had four numbers; validate
  tried simpler ones (fewer items, numbers closer to 0, shorter
  strings) that still fail, and shows the smallest: `[ -1 ]`. Here it
  shows the bug at once: `-1 % 2` is `-1`, not `1`.
- **The seed**: put `seed = 713144290` at the top of the test (or the
  file) and the same inputs come again, so you can fix the bug and see
  it pass.
- An error inside the function counts as a failure too:
  `it stopped with an error: division by zero`.

### Learning inputs from checks

Leave out `with`, and validate makes inputs like the ones in the same
test's checks on that function:

```
def test_evens[]
    check evens[list [1, 2, 3, 4]] == list [2, 4] .
    validate evens[nums] that result each x give x % 2 == 0 .   // nums: lists of integers
def [end]
```

It learns the kind (a list of integers), not the range: numbers are
-1000 to 1000 and lists 0 to 10 long. Write `with` to choose. If the
test has no check on the function, it says so and asks for `with`.

### How many inputs: cases

`cases = 100` (made by `import test`) is how many inputs each validate
tries. Set it at the top of the file, or inside one test.

## Settings

`import test` makes these variables. Set them at the top of the file for
every test, or inside a test for that test only; a test's own setting
wins.

| Setting | Default | Means |
|---|---|---|
| `suite` | `false` | how the file's tests depend on each other (below) |
| `benchmark` | `false` | `true`: time many runs of each test |
| `runs` | `none` | with benchmark: `none` runs as many as fit in `benchtime` (as Go does); a number runs exactly that many |
| `benchtime` | `1` | seconds, when `runs` is `none` |
| `cases` | `100` | random inputs each `validate` tries |
| `seed` | `none` | `none`: different random inputs each run; a number: the same ones (shared with `import random`) |

### Suites

| `suite` | What happens |
|---|---|
| `false` | every test runs, each on its own |
| `"stop"` (or `true`) | the first failure ends the file: the tests after it are **skipped** |
| `"all"` | every test runs; one verdict for the file, and the failures listed together at the end |

Use `"stop"` when each test depends on the one before (create an order,
then pay for it, then ship it); `"all"` to see every failure of a group
in one run.

```
test_orders.trt (suite: stop)
  PASS  test_create_order   2.0ms
  FAIL  test_pay_order      1.0ms
        test_orders.trt line 18: failed: check total[order] == 45 .
            got 40, want 45 (5 less)
  SKIP  test_ship_order     (stopped: test_pay_order failed)
suite FAILED: 1 passed, 1 failed, 1 skipped
```

```
test_orders.trt (suite: all)
  PASS  test_create_order   2.0ms
  FAIL  test_pay_order      1.0ms
  PASS  test_ship_order     3.0ms
  FAIL  test_refund_order   1.0ms
suite FAILED: 2 passed, 2 failed

failures:
  test_pay_order
      test_orders.trt line 18: failed: check total[order] == 45 .
          got 40, want 45 (5 less)
  test_refund_order
      test_orders.trt line 31: failed: check items of order == list ["pen", "mug"] .
          at 1: got "pad", want "mug"
```

### Timing and benchmarks

Every test's time is shown. With `benchmark = true`, a test runs once,
then again and again; the first run isn't counted (it's usually slower),
and the report shows how many runs were timed, their average, the
fastest and the slowest:

```
def test_sort_list[]
    benchmark = true
    runs = 1000            // exactly 1000 timed runs
    check min_sort[list [3, 1, 2]] == list [1, 2, 3] .
def [end]
```

```
PASS  test_sort_list   1000 runs   avg 1.5µs   fastest 1.2µs   slowest 4.6µs
```

With `runs = none` (the default), it runs as many times as fit in
`benchtime` seconds, so fast and slow tests both get a fair measure:

```
benchmark = true           // at the top: every test in the file
benchtime = 0.5

def test_parse[]
    benchmark = false      // ...except this one
    ...
def [end]
```

A benchmarked test repeats everything in it, including what it changes:
start each test by setting up what it needs (a list, a database table)
rather than relying on the run before. A run that fails is reported with
its number: `test_lists.trt line 8, benchmark run 37: ...`.

### Speed of the language itself

`testdata/speed/test_speed.trt` times common work across Turtle (calls,
loops, strings, lists, maps, sorting, JSON, dates, random values,
SQLite) with `benchmark = true`. Every test passes; the timings are for
comparing before and after a change to the interpreter:

```sh
turtle test testdata/speed
```

## Outside of turtle test

`check`, `verify` and `validate` work in any file with `import test`. A
failure is an error of kind `test`: it stops the program, like any
error, unless a `safe` block handles it:

```
import test

safe
    check config at contains["port"] .
handle [test] e .
    warn "config has no port; using 8080" .
safe [end]
```

## Names

- In a file with `import test`, `check`, `verify` and `validate` begin
  statements, so a function there can't be called one of them
  (`def check[...]` is an error). A function called `check` in another
  file can still be used as `mylib check[...]`.
- `test_` functions take no arguments.
- Without `import test`, the three are ordinary names.
