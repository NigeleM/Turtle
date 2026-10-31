# `test` — testing

[Library index](index.md) · `import test`

`check`, `verify` and `validate`, and `turtle test`.


`import test` adds three statements, `check` (one fact), `verify` (a
rule for every item) and `validate` (a rule on many random inputs), and
the settings `turtle test` reads. Everything is in
[`testing.md`](../testing.md).

```
import test

def test_evens[]
    check evens[list [1, 2, 3, 4]] == list [2, 4] .
    verify evens[nums] each x give x % 2 == 0 .
    validate evens[nums] with nums as list of integer
        that result each x give x % 2 == 0 .
def [end]
```

```sh
turtle test
```

---

Copyright 2017-2026 Nigele McCoy. Licensed under the
[Apache License 2.0](../../LICENSE); see [NOTICE](../../NOTICE).

<!-- generated: turtle doc `test` (regenerate with the script; do not edit by hand) -->
## Function reference

The entries below are copied from the binary's own documentation (`turtle doc test`), so they match the version you run.

```text
import test
```

### `Tests. In a file with "import test", three statements check your code;`

```text
Tests. In a file with "import test", three statements check your code;
"turtle test" runs every test_ function in every test_*.turtle file.
```

```text
    check total[order] == 45 .                     one fact; == shows how they differ
    check x is integer .                           also: float number string boolean
                                                   list set map date none function empty,
                                                   an assembled type, "is not"
    check 0.1 + 0.2 is close to 0.3 .              decimals ("within 0.01" to choose)
    check 1 div 0 fails [math] .                   the right answer is an error
    verify nums each x give x > 0 .               a rule for every item; also any, not,
                                                   at least N, at most N, exactly N,
                                                   and "each pair [a, b] give a <= b"
    validate evens[nums] with nums as list of integer
        that result each x give x % 2 == 0 .      the rule on 100 random inputs,
                                                   shrunk to the smallest that fails;
                                                   or: matches other_function[nums]
```

### `Settings (set at the top of the file, or inside one test for that test):`

```text
Settings (set at the top of the file, or inside one test for that test):
    suite = false        false: each test alone; "stop": a failure skips the
                         rest; "all": run all, failures listed at the end
    benchmark = false    true: time many runs of each test
    runs = none          none: as many runs as fit in benchtime; or a number
    benchtime = 1        seconds
    cases = 100          random inputs each validate tries
    seed = none          a number repeats the same random inputs
```

### `A failure is an error of kind test. Full guide: docs/testing.md.`

```text
A failure is an error of kind test. Full guide: docs/testing.md.
```

<!-- end generated -->
