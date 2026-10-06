// test_verify.t: every form of verify. A verify tries a rule on every
// item and says which items break it.
//
//   verify xs each R .        every item follows R        (count == length)
//   verify xs any R .         at least one does           (count >= 1)
//   verify xs not R .         none does                   (count == 0)
//   verify xs at least N R .  N or more do                (count >= N)
//   verify xs at most N R .   N or fewer do               (count <= N)
//   verify xs exactly N R .   exactly N do                (count == N)
//
// where count is how many items follow the rule R.
import test
import data

rolls = list [6, 2, 6, 3, 5]      // two sixes

def is_even[n]
    return n % 2 == 0
def [end]

def test_each_any_not[]
    verify rolls each r gives r >= 1 && r <= 6 .
    verify rolls any r gives r == 6 .
    verify rolls not r gives r == 1 .
    verify list [2, 4, 8] each is_even .  // a function's name works too
def [end]

// at least / at most / exactly count how many items follow the rule.
// rolls has two sixes, so:
def test_counts[]
    verify rolls at least 1 r gives r == 6 .     // 2 >= 1
    verify rolls at least 2 r gives r == 6 .     // 2 >= 2
    verify rolls at most 2 r gives r == 6 .      // 2 <= 2
    verify rolls at most 5 r gives r == 6 .      // 2 <= 5
    verify rolls exactly 2 r gives r == 6 .      // 2 == 2
    verify rolls exactly 0 r gives r == 4 .      // no fours: 0 == 0
    verify rolls at least 0 r gives r == 4 .     // always true: 0 >= 0
    verify rolls at most 0 r gives r == 4 .      // the same as: not r gives r == 4
def [end]

// Inside a safe block, a failed verify can be caught to look at it.
def test_what_a_failure_says[]
    msg = ""
    safe
        verify rolls at least 3 r gives r == 6 .
    handle [test] e .
        msg = message of e
    safe [end]
    check msg at contains["2 of 5 items follow the rule; at least 3 should:"] .
    check msg at contains["at 0: 6"] .
def [end]

// each pair: neighbors two at a time, for order rules.
def test_pairs[]
    verify list [1, 2, 2, 7] each pair [a, b] gives a <= b .   // never goes down
    verify list [1, 2, 3] each pair [a, b] gives b == a + 1 .  // counts up by 1
    verify "abc" each pair [a, b] gives a < b .                // works on strings too
def [end]

// Maps: one name gets each value; two names get the key and the value.
def test_maps[]
    ages = map ["ann": 30, "bo": 25]
    verify ages each a gives a >= 18 .
    verify ages each [name, age] gives name at length <= 3 && age < 100 .
def [end]

// Sets and strings: their items and characters.
def test_sets_and_strings[]
    verify set [2, 4] each is_even .
    verify "hello" not c gives c == "z" .
    verify "hello" exactly 2 c gives c == "l" .
def [end]

// Nested data: the rule can look inside.
def test_nested[]
    grid = list [list [1, 2, 3], list [4, 5, 6]]
    verify grid each row gives length of row == 3 .
    verify grid each row gives row at contains[2] || row at contains[5] .
def [end]
