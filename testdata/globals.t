// Functions can read globals, but plain assignment inside a function
// never writes back to one — it shadows locally instead. Mutating a
// global list/set/map through a data-op or method call, though, is a
// real mutation visible after the call returns.

PI = 3.14159
count = 0
nums = list [1, 2, 3]

def circleArea[r]
    return PI * r * r
def [end]

def bump[]
    count = count + 1
    return count
def [end]

def addToNums[v]
    add v to nums .
def [end]

show circleArea[2] .

show bump[] .
show bump[] .
show count .

addToNums[99]
show nums .
