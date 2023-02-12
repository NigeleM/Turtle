
def hello [b]
    c = b/10.0
    // show c .
     return c 
def [end]

def run[C]
    show "running run function" .
    a = hello[C]
    show "call here" .
    // a = hello[C]
    show "expression here" .
    // show 1.0 + a . 
    a = a + 1.00
    show a .
    show hello[a] ," ", hello[80] .
    show "completed" .
def [end]


def show []
    a = 100
    b = 999
    show a + b , " HELLO WORLD " .
    show a , " A HELLO WORLD " .
    show b , " B HELLO WORLD " .
    if ] a > b [
        show "hello nigele" .
        [if ] 187 > 89 [
            show "cold weather" .
        [else if ] 69 > 89 [
            show "99 > 89" . 
        [else ]
            show "cold life" .
    else if ] 89 > 99 [
        [if ] 80 > 89 [
            show "cold weather on the easy" .
        [else if ] 69 > 89 [
            show "99 > 89 on the water" . 
        [else ]
            show "cold life" .
    else ]
        show a + b .
    if [end]
def [end]

// good[]

// show " hello [] " .
// show hello [212] .
a = 30
b = 50
// show 4*12 .
show b + hello [106] +  hello [318] + 10 + hello [212] + a .
// show b , " ", hello [b] + 10, " ",  hello [318] ," ", 10 ,  " ",hello [212] , " ", a ," " ,hello [318] + a .
// show "good day " .
 run [30]
// show b  + hello [106] , .

ass = 1890000
// show "in parser test isMain" .
 if ] hello[ass] > 89 [
    show "cold weather isMain" .
else if ] 69 > 89 [
    show "99 > 89" . 
else ]
    show "cold life" .
if [end]

def parserTest[]
 a = 100000
 // show "in parser test function" .
 if ] hello[1000000] > 89 [
    show "cold weather not Main" .
else if ] 69 > 89 [
    show "99 > 89" . 
else ]
    show "cold life" .
if [end]

def [end]
// show []
// parserTest[]