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
