# `random` — random values

[Library index](index.md) · `import random`

Random values of any shape, written as a sentence, and picking, shuffling and sampling.


`import random` makes random values of any shape. Describe the value in
words after `random`; each time the line runs, it gives a new one.

```
import random
assemble Order [item, qty, price]

die = random integer from 1 to 6
nums = random list of 5 integers from 0 to 9       // [ 9, 5, 1, 7, 6 ]
price = random float from 0.5 to 99.99 rounded to 2 // 38.61
code = random string of 8                            // "AGhwhRHU"
pin = random digits of 4                           // "5875": a string, so "0427" keeps its 0
id = random string of 6 from "ABCDEF0123456789"      // "6B3838"
day = random date from "2026-01-01" to "2026-12-31"
grid = random list of 3 lists of 3 integers from 1 to 9
groups = random set of 4 lists of 2 strings of 3
ages = random map of 3 string of 2 to integer from 0 to 99
order = random Order [string of 5, integer from 1 to 10, float rounded to 2]
orders = random list of 20 Order [string, integer, float]
```

| Kind | Gives | Options |
|---|---|---|
| `integer` | a whole number, -1000 to 1000 | `from A to B` (both included) |
| `float` | a decimal, 0 up to 1 | `from A to B`, `rounded to N` (places) |
| `string` | 1 to 10 letters, a–z and A–Z | `of N`, `of N to M` (length), `from "chars"` (only these) |
| `digits of N` | a string of N digits (`"0427"`) | `of N to M` |
| `digit`, `letter` | one digit or letter, as a string | |
| `boolean` | `true` or `false` | |
| `date` | a day, 2000-01-01 to 2030-12-31 | `from A to B`: dates or strings like `"2026-01-31"` |
| `time` | a date with a time of day, to the second | `from A to B` |
| `list of <kind>` | a list | a count: `list of 5 integers`, `list of 2 to 8 integers`, `list of n integers` |
| `set of <kind>` | a set; its items are all different | a count, as for lists |
| `map of <kind> to <kind>` | a map; its keys are all different | a count: `map of 3 string to integer` |
| `Order [<kind>, ...]` | an assembled value, one kind per field, in order | |

- **Plurals** read naturally and mean the same: `integers`, `floats`,
  `strings`, `letters`, `booleans`, `dates`, `lists`, `sets`, `maps`
  (`digits` without `of` is single digits: `list of 3 digits`).
- **A count comes before the kind**, and is a whole number or a
  variable's name. Without one, a list, set or map has 0 to 10 items.
  Limits (`from`, `of`, `rounded`) come after the kind.
- **Kinds nest** any way: `list of 3 lists of 3 integers`,
  `set of 4 lists of string`, `map of string to list of Order [string, float]`.
- **Where the sentence ends**: arithmetic belongs to a limit (`to n + 1`),
  but a comparison applies to the random value:
  `random integer from 1 to 6 == 6` is true one time in six. A method
  works on the value right before it (`to 9 at abs` is about the 9), so
  to call one on the random value, name it first: `nums = random list
  of 5 integers`, then `nums at length`.
- A set or map that can't get enough different values
  (`set of 5 integers from 1 to 3`) is a `math` error, as is a range
  whose `from` is more than its `to`.
- `random` is a sentence word only in a file that has `import random`;
  elsewhere it's an ordinary name, and `n at random` (math) works as
  before.

**Choosing from your own values** (`pick` and `chance` are written in
Turtle, in `evaluator/lib/random.turtle`, on the `random` sentence; `shuffle`
and `sample`, which go through every item, are in Go for speed):

| Function | Gives |
|---|---|
| `pick[x]` | one item of a list or set, one key of a map, one character of a string |
| `shuffle[x]` | a new list in random order (a string, for a string) |
| `sample[x, n]` | a list of `n` different items |
| `chance[p]` | `true` with probability `p` (0 to 1) |

```
color = pick[list ["red", "green", "blue"]]
hand = sample[shuffle[deck], 5]
if ] chance[0.3] [
    show "rain" .
if [end]
```

**The same values every run.** `import random` also makes the variable
`seed`, `none`: different values each run. Set it to a whole number and
every random value from there on is the same each time the program runs,
which is how to repeat a run exactly (a test that failed on random
data, say). Setting it again, even to the same number, starts the values
over; `seed = none` goes back to different values each run.

```
seed = 42
a = random list of 3 integers
seed = 42
b = random list of 3 integers      // the same as a
```
