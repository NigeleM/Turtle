// lib/pricing.t: the shop's types and money rules. Imported by shop.t
// as "import lib/pricing", so its qualified name is "pricing".
import math

assemble Item [sku, title, price, stock, tags]

// make_discount returns a closure that takes pct percent off an amount
// in cents. Each coupon keeps its own pct.
def make_discount[pct]
    return amount gives amount - amount * pct / 100
def [end]

// add_tax adds rate percent, rounded to the nearest cent.
def add_tax[cents, rate]
    t = cents * rate / 100.0
    r is t at round .
    return cents + r
def [end]

// average of a list of numbers: integer division, so whole cents.
// An empty list divides by zero: shop.t checks that the error names
// this file.
def average[nums]
    total = 0
    [loop][n in nums]
        total = total + n
    [loop][end]
    return total / length of nums
def [end]
