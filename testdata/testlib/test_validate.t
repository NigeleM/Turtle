// test_validate.t: validate runs a function on many random inputs and
// checks a rule on every answer. When one fails, it shrinks the input to
// the smallest that still fails, and prints the seed to repeat the run.
import test
import data
import sort [min_sort, quick_sort, merge_sort]

assemble Order [item, qty, price]

def evens[nums]
    out = list []
    [loop][n in nums]
        if ] n % 2 == 0 [
            add n to out .
        if [end]
    [loop][end]
    return out
def [end]

def plus[a, b]
    return a + b
def [end]

def total[order]
    return qty of order * price of order
def [end]

// that <rule about result>: the answer is called result.
def test_rule_on_the_answer[]
    validate evens[nums] with nums as list of integer
        that result each x gives x % 2 == 0 .
def [end]

// to name: call the answer something else; the rule can use the inputs.
def test_rule_on_inputs_and_answer[]
    validate evens[nums] to found with nums as list of integer from -50 to 50
        that length of found <= length of nums .
def [end]

// Several inputs.
def test_two_inputs[]
    validate plus[a, b] with a as integer, b as integer that result == plus[b, a] .
    validate plus[a, b] with a as string of 2, b as string of 3 that result at length == 5 .
def [end]

// matches: a fast new version against a slow trusted one.
def test_matches[]
    validate quick_sort[nums] with nums as list of integer matches min_sort[nums] .
    validate merge_sort[words] with words as list of strings matches min_sort[words] .
def [end]

// Assembled values: a kind for each field, in order.
def test_assembled[]
    validate total[o] with o as Order [string, integer from 1 to 10, integer from 0 to 100]
        that result >= 0 && result <= 1000 .
def [end]

// Leave out "with": validate makes inputs like the ones in this test's
// checks on the same function (here, lists of integers).
def test_learned_inputs[]
    check evens[list [1, 2, 3, 4]] == list [2, 4] .
    validate evens[nums] that result each x gives x % 2 == 0 .
def [end]

// A failing validate, caught to show what it reports. This version keeps
// every number that isn't 1 more than an even: -3 % 2 is -1, so negative
// odd numbers slip through.
def bad_evens[nums]
    return nums keep n gives n % 2 != 1
def [end]

def test_what_a_failure_says[]
    msg = ""
    safe
        validate bad_evens[nums] with nums as list of integer
            that result each x gives x % 2 == 0 .
    handle [test] e .
        msg = message of e
    safe [end]
    check msg at contains["nums = [ -1 ]"] .        // shrunk to the smallest
    check msg at contains["to repeat this run: seed = "] .
def [end]
