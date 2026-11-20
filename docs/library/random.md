# `random` — random values

`import random`

Random values: pick an item, shuffle a list, take a sample, or decide something by chance.


## Functions

`pick`, `shuffle`, `sample`, `chance`.

### `pick[x]`

One item, chosen at random. x   a list, set, map (one of its keys) or string (one character) An empty one is an error.

**Example**

```turtle
import random
color = pick[list ["red", "green", "blue"]]
show color .
```

**Output**

```text
blue
```

The colour is chosen at random, so the output above is one of the three possible results; it can differ on each run.

### `shuffle[x]`

The items in a random order. x   a list, set, map (its keys) or string Gives back: a new list (a string, for a string); x is left as it was.

**Example**

```turtle
cards = list ["A", "K", "Q", "J"]
deck = shuffle[cards]
hand = sample[deck, 2]
show length of hand .
```

**Output**

```text
2
```

### `sample[x, n]`

n different items (different positions), chosen at random. x   a list, set, map (its keys) or string n   how many; more than x has is an index error Gives back: a list.

**Example**

```turtle
cards = list ["A", "K", "Q", "J"]
deck = shuffle[cards]
hand = sample[deck, 2]
show length of hand .
```

**Output**

```text
2
```

### `chance[p]`

true with probability p. p   a number from 0 to 1: 0.3 is true 30% of the time

**Example**

```turtle
if ] chance[0.3] [
    show "rain" .
if [end]
```

**Output**

```text
rain
```
