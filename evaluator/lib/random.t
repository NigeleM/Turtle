// random.t: the random library's choosing functions, written in Turtle on
// the parts written in Go (the "random" sentence, and seed). It's built
// into turtle; "import random" loads it. pick, shuffle, sample and chance
// are exported; items_of and is_string are helpers, private to it.
import strings [join]

// items_of is what pick, shuffle and sample choose from: a list's or
// set's items, a map's keys, a string's characters.
def items_of[x, fn]
    safe
        return change x to list
    handle [] e .
        fail "{fn} needs a list, set, map or string, got {x}"
    safe [end]
def [end]

// is_string: only a string is the same as itself changed to a string.
def is_string[x]
    return x == change x to string
def [end]

def pick[x]
    items = items_of[x, "pick"]
    if ] length of items == 0 [
        fail "pick: {x} is empty, so there's nothing to pick"
    if [end]
    return items at get[random integer from 0 to length of items - 1]
def [end]

// shuffle takes the items out one at a time, from a random place each
// time. Removing the first item equal to the one taken is fine: equal
// items can't be told apart.
def shuffle[x]
    rest = items_of[x, "shuffle"]
    out = list []
    [loop][length of rest > 0]
        item = rest at get[random integer from 0 to length of rest - 1]
        add item to out .
        remove item from rest .
    [loop][end]
    if ] is_string[x] [
        return join[out, ""]
    if [end]
    return out
def [end]

// sample: n items from different places, in random order.
def sample[x, n]
    items = items_of[x, "sample"]
    if ] n < 0 || n > length of items [
        fail "sample: can't take {n} different items from {length of items}"
    if [end]
    return shuffle[items] at slice[0, n]
def [end]

def chance[p]
    if ] p < 0 || p > 1 [
        fail "chance: needs a number from 0 to 1 (0.3 is 30% of the time), got {p}"
    if [end]
    return random float < p
def [end]
