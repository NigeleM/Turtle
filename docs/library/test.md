# `test` — testing

`import test`

Check that code works with `check`, `verify` and `validate`, and run them with the `turtle test` command.


## Statements

`check`, `verify` and `validate`. `turtle test` runs the `test_` functions in files named `test_*.turtle`.

## Example

### `check expression .`

Passes when the expression is true. If it is false, the program stops with a test error.

**Example**

```turtle
import test
def sum_two[a, b]
    return a + b
def [end]
check sum_two[2, 3] == 5 .
verify list [1, 2, 3] each x give x > 0 .
show "checks ran" .
```

**Output**

```text
checks ran
```

Everything else about testing is in [the testing guide](../testing.md).
