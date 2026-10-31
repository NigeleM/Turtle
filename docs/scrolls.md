# Scrolls

A **scroll** is a value that holds a sequence of steps. Applied to a starting value,
it passes the value through each step in order, and the result of one step becomes
the input of the next.

A chain of calls can be written two ways. The **functional** form nests the calls
inside out. The **sentence** form, a scroll, lists the steps in the order they run.
Both give the same result:

```turtle
def add1[x]
    return x + 1
def [end]
def double[x]
    return x * 2
def [end]
def half[x]
    return x / 2
def [end]

show half[double[add1[3]]] .          // functional: 4.0
show scroll 3 into add1, double, half . // sentence: 4.0
```

Read the sentence form from left to right: start with 3, add 1, double, then halve.
The functional form must be read from the middle outwards. Use whichever is clearer
for the chain at hand. A scroll is most useful when a chain has several steps.

## Writing a scroll

A scroll starts with `scroll`, then a starting value, then `into`, then the steps
separated by commas. It ends with a period.

```turtle
def add1[x]
    return x + 1
def [end]
def double[x]
    return x * 2
def [end]
def half[x]
    return x / 2
def [end]

x is scroll 3 into add1, double, half .   // stores 4.0 in x
x = scroll 3 into add1, double, half .    // the same, with =
```

A scroll without a starting value is saved, and is called with brackets:

```turtle
def add1[x]
    return x + 1
def [end]
def double[x]
    return x * 2
def [end]
s = scroll add1, double .      // a saved scroll: no starting value
show s[3] .                    // functional call of the saved scroll: 8
show scroll 3 into s .         // sentence form: 8
```

A saved scroll can be a step of another scroll. Its steps run in place:

```turtle
def add1[x]
    return x + 1
def [end]
def double[x]
    return x * 2
def [end]
first = scroll add1 .
second = scroll double .
show scroll 2 into first, second .     // 6
show scroll 2 into add1, double .      // 6
```

## Steps

The table shows how each kind of step is written. The value goes first in a call,
unless `here` says where it goes.

| The step is | Written as | Means |
|---|---|---|
| a function that needs only the value | `double` | `double[value]` |
| a method | `at upper` | `value at upper` |
| a function that needs other values | `max[here, 5]` | that call, with the value at `here` |
| a `give` function | `n give n * 5` | the function, applied to the value |
| a saved scroll | `second` | its steps, run in place |

```turtle
show scroll 3 into max[here, 5] .       // 5
show scroll 3 into here * 10 .          // 30
show scroll "  hi  " into at trim, at upper .   // HI
show scroll 4 into n give n * 5 .       // 20
```

`here` is the value reaching the step. Where it is not written, the value goes
first. Where it is written, the value goes at `here`.

## Lists

When the value is a list, each step receives the whole list. Functions such as `sum`
and `keep` work on the whole list. `process` applies a rule to each item:

```turtle
import data
rows = list [map ["item": "pen", "amount": 1200], map ["item": "ink", "amount": 300], map ["item": "desk", "amount": 2100]]

// functional
show process[keep[rows, r give r at get["amount"] > 1000], r give r at get["item"]] .
// sentence
show scroll rows into keep r give r at get["amount"] > 1000, process r give r at get["item"] .
```

Both lines print `[ "pen", "desk" ]`. A `give` function in a step ends at the
comma that starts the next step.

## Errors

An error in one step ends the scroll at that step, because the steps after it need
its result. The message names the step, and the error keeps its kind, so `handle`
can still catch it:

> *Illustrative.* This example shows an error: the program stops at the line that divides by zero, so it is shown for reading and is not run as a test.

```turtle
show scroll 0 into 10 / here .
```

```
turtle: line 1: scroll step 1 of 1 (10 / here): division by zero
```

A step that gives `none` is not an error. The value `none` is passed to the next
step.

## Where scrolls fit

Use a scroll when a value passes through the same steps in order, and the reader
should see those steps in order. Use nested calls for a single function, and a
plain sequence of assignments when each step has a different job that needs a name.
