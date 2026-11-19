# Book examples, with their output

Every Turtle example in the book, in chapter order, run on the current build (v1.0.0-dev). Each shows the program and what it printed. The book is the reference; this page lets you see every example run.

Some examples depend on a file, a database, input from the keyboard or an earlier part of their chapter, so they report an error when run alone. Those show the error they give. Examples that serve pages run until they are stopped, and are marked that way. Examples that call a web address need a server at that address.

## Part 1: the basics

From the book, `book/src/01-part1.md`.

**Example 1**

```turtle
show "Hello, world" .
```

```text
Hello, world
```


**Example 2**

```turtle
// a line comment, which runs to the end of the line
//* a block comment
   that spans several lines *//
show "Comments are ignored" .
```

```text
Comments are ignored
```


**Example 3**

```turtle
// FizzBuzz from 1 to 30.
[loop][n = 1; n <= 30; n++]
    if ] n % 15 == 0 [
        show "FizzBuzz" .
    else if ] n % 3 == 0 [
        show "Fizz" .
    else if ] n % 5 == 0 [
        show "Buzz" .
    else ]
        show n .
    if [end]
[loop][end]
```

```text
1
2
Fizz
4
Buzz
Fizz
7
8
Fizz
Buzz
11
Fizz
13
14
FizzBuzz
16
17
Fizz
19
Buzz
Fizz
22
23
Fizz
Buzz
26
Fizz
28
29
FizzBuzz
```


**Example 4**

```turtle
show typeof[5] .
show typeof[2.5] .
show typeof["hi"] .
show typeof[list [1]] .
```

```text
integer
float
string
list
```


**Example 5**

```turtle
x = 5
x = x + 1
show x .
```

```text
6
```


**Example 6**

```turtle
import math
show 2 at pow 100 .
```

```text
1267650600228229401496703205376
```


**Example 7**

```turtle
show 7 / 2 .
show 7 div 2 .
show -7 div 2 .
show -7 % 2 .
```

```text
3.5
3
-3
-1
```


**Example 8**

```turtle
show 1e-9 .
show 6.02e23 .
show 1e308 * 10 .
show 1e308 * 10 - 1e308 * 10 .
```

```text
1e-09
6.02e+23
inf
nan
```


**Example 9**

```turtle
show change "1_000" to integer .
```

```text
1000
```


**Example 10**

```turtle
name = "Nigele"
greeting = "Hello, " + name + "!"
show greeting .
```

```text
Hello, Nigele!
```


**Example 11**

```turtle
show "5" + 0 .
```

```text
turtle: line 1: '+' can't join string and integer ("5" + 0): to add, change the text to a number first, as (change t to integer) + n; to join, write "{t}{n}"
  1 | show "5" + 0 .
```

This example stops with an error (exit code 1). It depends on something outside the example, or it shows an error on purpose.


**Example 12**

```turtle
show "tab\there" .
show 'She said "hi"' .
show "set \{1, 2}" .
show `raw \n stays` .
```

```text
tab	here
She said "hi"
set {1, 2}
raw \n stays
```


**Example 13**

```turtle
raw = "42"
change raw to integer .
show raw + 1 .

n = change "3.5" to float
show n * 2 .
```

```text
43
7.0
```


**Example 14**

```turtle
show change "A" to ascii .
show change 66 to char .
show change 255 to hex .
show change "ff" to hex .
```

```text
65
B
ff
255
```


**Example 15**

```turtle
x = none
show x == none .
show x == 0 .
show none + 1 .
```

```text
true
false
turtle: line 4: '+' can't add none and integer — none only adds to none; check for it first, e.g. if ] x != none [
  4 | show none + 1 .
```

This example stops with an error (exit code 1). It depends on something outside the example, or it shows an error on purpose.


**Example 16**

```turtle
// Convert a list of Celsius temperatures to Fahrenheit and Kelvin.
celsius = list [-40, 0, 37, 100]
[loop][c in celsius]
    f = c * 9 / 5 + 32
    k = c + 273.15
    show c, " C is ", f, " F and ", k, " K" .
[loop][end]
```

```text
-40 C is -40.0 F and 233.14999999999998 K
0 C is 32.0 F and 273.15 K
37 C is 98.6 F and 310.15 K
100 C is 212.0 F and 373.15 K
```


**Example 17**

```turtle
show 2 + 3 * 4 .
show (2 + 3) * 4 .
show 10 - 4 - 3 .
show 7 div 2 * 2 .
```

```text
14
20
3
6
```


**Example 18**

```turtle
show 1 + 2 * 3 == 7 && 2 > 1 .
show 1 > 2 || 3 > 2 && false .
```

```text
true
false
```


**Example 19**

```turtle
show "a" + "b" at upper .
assemble O [qty, price]
o = O[3, 1.5]
show qty of o * price of o .
```

```text
aB
4.5
```


**Example 20**

```turtle
show -2 * 3 .
show 1 - -2 .
```

```text
-6
3
```


**Example 21**

```turtle
x = 4
y = 9
show "x = ", x, ", y = ", y .
```

```text
x = 4, y = 9
```


**Example 22**

```turtle
x = 4
y = 9
show "x = {x}, y = {y}, total = {x + y}" .
```

```text
x = 4, y = 9, total = 13
```


**Example 23**

```turtle
def fix[a, b]
    return a + b
def [end]

x = 1
show x fix 2 ; "The answer" .
show x fix 2 ; x fix 5 .
show x fix 2 ; 7 ; 8 .
```

```text
3 The answer
3 6
3 7 8
```


**Example 24**

```turtle
x = 1 . y = 2 . show x + y .
```

```text
3
```


**Example 25**

```turtle
name = ? "What's your name? "
show "Hi, {name}!" .
```

```text
What's your name? turtle: line 1: unexpected end of input reading "name"
  1 | name = ? "What's your name? "
```

This example stops with an error (exit code 1). It depends on something outside the example, or it shows an error on purpose.


**Example 26**

```turtle
raw = ? "Age? "
age = change raw to integer
show age + 1 .
```

```text
Age? turtle: line 1: unexpected end of input reading "raw"
  1 | raw = ? "Age? "
```

This example stops with an error (exit code 1). It depends on something outside the example, or it shows an error on purpose.


**Example 27**

```turtle
import system
warn "disk is nearly full" .
```

```text
disk is nearly full
```


**Example 28**

```turtle
show typeof[2.5] .
show typeof["hi"] .
show typeof[none] .
x = none
show x == none .
show x == 0 .
```

```text
float
string
none
true
false
```


**Example 29**

```turtle
import data
help["sum"]
```

```text
sum[collection]        (import data)
  Adds up a list's or set's numbers (a map's values): an integer if they
  all are, 0 when there are none. Sentence form: nums sum.
  Example:
    prices = list [4, 9, 6]
    total = sum[prices]
    avg = mean[prices]
    spread = stdev[prices]
    show total, " ", avg, " ", spread .

Statistics: each takes a list, a set or a map (its values), or a list of
rows (maps or assembled values, as sql_query and table_read give) and a
column name. none is skipped, as a missing value; anything else that
isn't a number is an error. Sentence form: prices mean.
```


**Example 30**

```turtle
help[]
```

```text
Turtle 1.0.0-dev

help["linear"]     a library: what it's for and its functions
help["reshape"]    one function or method, with an example
help["if"]         a keyword; help["keywords"] lists them all
help[x]            what x is and what you can do with it
stdlib[]           every library and its functions, as a map
version            the version of Turtle

Libraries (import one to use it):
  config    Settings files, the format chosen by the file's extension: .toml, .json or .env.
  crypt     Hashes, signatures, encodings, random ids and tokens, passwords and encryption.
  data      Working with collections (lists, sets, maps), and with tables of rows
  http      Web requests.
  json      Reading and writing JSON.
  linear    Matrices and vectors.
  log       Log lines: a level, a time and where they came from, to the console (stderr) and, if you like, a file.
  math      Number methods.
  pattern   Patterns (regular expressions) in text.
  random    Random values of any shape, written as a sentence after "random"
  schedule  Many web requests, shell commands or database queries at once.
  search    Finding things.
  server    A web server.
  sort      Putting things in order.
  sql       Databases: SQLite files, and PostgreSQL and MySQL servers, all through the same functions; sql_open's address picks which.
  strings   Text functions that read well as sentences.
  system    The command line, environment, files and folders, and the program itself.
  test      Tests.
  time      The clock, dates, time zones, date arithmetic, and waiting.
```


**Example 31**

```turtle
nums = list [1, 2]
help[nums]
```

```text
This is a list of 2 items.
A list's methods, as x at add[value]:
  add[value], get[index], put[value, index], insert[value, index],
  remove[value], pop, slice[start, end], contains[value], index[value],
  count[value], sort, reverse, clear, len, isempty, tostring
  (put and insert take the value first, then the position: put[99, 1];
   the sentence names each: put 99 to nums at 1 .)
Libraries for it: help["data"], help["sort"], help["search"], help["random"]
```


**Example 32**

```turtle
show version .
```

```text
1.0.0-dev
```


**Example 33**

```turtle
x = 12
if ] x > 10 [
    show "big" .
else if ] x > 0 [
    show "small" .
else ]
    show "non-positive" .
if [end]
```

```text
big
```


**Example 34**

```turtle
age = 21
has_ticket = true
if ] age >= 18 && has_ticket [
    show "come in" .
if [end]
```

```text
come in
```


**Example 35**

```turtle
matches = list []
if ] matches [
    show "found some" .
else ]
    show "none found" .
if [end]
```

```text
none found
```


**Example 36**

```turtle
if ] a > 0 [
    show "a is positive" .
    [if ] b > 0 [
        show "both positive" .
    [else ]
        show "a positive, b not" .
else ]
    show "a not positive" .
if [end]
```

```text
turtle: line 1: undefined variable "a"
  1 | if ] a > 0 [
```

This example stops with an error (exit code 1). It depends on something outside the example, or it shows an error on purpose.


**Example 37**

```turtle
if ] "" [
    show "empty text is true" .
else ]
    show "empty text is false" .
if [end]
show !"" .
show 0 && 1 .
```

```text
empty text is false
true
false
```


**Example 38**

```turtle
x = none
if ] x != none && x > 5 [
    show "big" .
else ]
    show "not big, or none" .
if [end]
```

```text
not big, or none
```


**Example 39**

```turtle
n = 5
if ] n type integer [
    show "a whole number" .
if [end]
```

```text
a whole number
```


**Example 40**

```turtle
// Classify a number as negative or positive, and as even or odd, with nested ifs.
[loop][n in list [-4, -3, 0, 7, 10]]
    if ] n < 0 [
        if ] n % 2 == 0 [
            show n, " is negative and even" .
        else ]
            show n, " is negative and odd" .
        if [end]
    else ]
        if ] n == 0 [
            show n, " is zero" .
        else ]
            show n, " is positive" .
        if [end]
    if [end]
[loop][end]
```

```text
-4 is negative and even
-3 is negative and odd
0 is zero
7 is positive
10 is positive
```


**Example 41**

```turtle
[loop][i = 0; i < 5; i++]
    show i .
[loop][end]
```

```text
0
1
2
3
4
```


**Example 42**

```turtle
count = 0
[loop][count < 3]
    show count .
    count = count + 1
[loop][end]
```

```text
0
1
2
```


**Example 43**

```turtle
nums = list [4, 8, 15]
total = 0
[loop][x in nums]
    total = total + x
[loop][end]
show total .
```

```text
27
```


**Example 44**

```turtle
s = set [3, 1, 2]
total = 0
[loop][x in s]
    total = total + x
[loop][end]
show total .
```

```text
6
```


**Example 45**

```turtle
ages = map ["Alice": 30, "Bob": 25]
[loop][name, age in ages]
    show name, " is ", age .
[loop][end]
```

```text
Alice is 30
Bob is 25
```


**Example 46**

```turtle
ages = map ["Alice": 30, "Bob": 25]
[loop][name in ages]
    show name .
[loop][end]
```

```text
Alice
Bob
```


**Example 47**

```turtle
ages = map ["Alice": 30, "Bob": 25]
[loop][age in change ages to values]
    show age .
[loop][end]
```

```text
30
25
```


**Example 48**

```turtle
ages = map ["Alice": 30, "Bob": 25]
[loop][name in ages at getkeys]
    show name, " is ", ages at get name .
[loop][end]
```

```text
Alice is 30
Bob is 25
```


**Example 49**

```turtle
ages = map ["Alice": 30, "Bob": 25]
keys = ages at getkeys
[loop][i = 0; i < keys at len; i++]
    name = keys at get i
    show name, " is ", ages at get name .
[loop][end]
```

```text
Alice is 30
Bob is 25
```


**Example 50**

```turtle
[loop][i = 0; i < 10; i++]
    if ] i == 5 [
        break
    if [end]
    show i .
[loop][end]
```

```text
0
1
2
3
4
```


**Example 51**

```turtle
[loop][i = 0; i < 3; i++]
    [loop][j = 0; j < 3; j++]
        show i , ",", j .
    [loop][end]
[loop][end]
```

```text
0, 0
0, 1
0, 2
1, 0
1, 1
1, 2
2, 0
2, 1
2, 2
```


**Example 52**

```turtle
// Stop scanning at the first negative value.
readings = list [4, 8, 15, -2, 16, -9]
position = -1
[loop][i = 0; i < readings at len; i++]
    if ] readings at get[i] < 0 [
        position = i
        break
    if [end]
[loop][end]
show "first negative at index ", position .
```

```text
first negative at index 3
```


**Example 53**

```turtle
// Print 1 to 20, skipping multiples of four.
[loop][n = 1; n <= 20; n++]
    if ] n % 4 == 0 [
        continue
    if [end]
    show n .
[loop][end]
```

```text
1
2
3
5
6
7
9
10
11
13
14
15
17
18
19
```


**Example 54**

```turtle
def plus[a, b]
    result = a + b
    return result
def [end]

sum = plus[1, 2]
show sum .
```

```text
3
```


**Example 55**

```turtle
def fix[a, b]
    return a + b
def [end]

show fix[1, 2] .
x = 1
show x fix 2 .
show fix 1, 2 .
```

```text
3
3
3
```


**Example 56**

```turtle
f = x give x * 3
show -1 f .
show f[-1] .
```

```text
-3
-3
```


**Example 57**

```turtle
def fix[a, b]
    return a + b
def [end]
show fix 1, 2; " is the sum" .
```

```text
3 is the sum
```


**Example 58**

```turtle
def fix[a, b]
    return a + b
def [end]
show list [fix 1, 2;, fix 3, 4;, 5] .
show list[fix 1, 2; fix 3, 4; 5] .
```

```text
[ 3, 7, 5 ]
[ 3, 7, 5 ]
```


**Example 59**

```turtle
def greet[]
    show "hello" .
def [end]

greet[]
```

```text
hello
```


**Example 60**

```turtle
orders = list [1, 2]
report[orders]

def report[list_of_orders]
    show length of list_of_orders .
def [end]
```

```text
2
```


**Example 61**

```turtle
count = 0

def bump[n]
    return n + 1
def [end]

count = bump[count]
show count .
```

```text
1
```


**Example 62**

```turtle
nums = list [1, 2, 3]

def add_value[v]
    add v to nums .
def [end]

add_value[99]
show nums .
```

```text
[ 1, 2, 3, 99 ]
```


**Example 63**

```turtle
double = x give x * 2
show double[4] .        // 8: the function doubles 4

plus = [a, b] give a + b
show plus[2, 3] .       // 5: the function adds its two arguments

def apply[f, v]
    return f[v]
def [end]

show apply[x give x + 1, 3] .   // 4: apply runs x + 1 on 3
```

```text
8
5
4
```


**Example 64**

```turtle
def make_adder[n]
    def adder[x]
        return x + n
    def [end]
    return adder
def [end]

add5 = make_adder[5]
show add5[1] .
```

```text
6
```


**Example 65**

```turtle
show trim[" hi "] .
show " hi " at trim .
```

```text
hi
hi
```


**Example 66**

```turtle
big = [x] give
    if ] x > 3 [
        return true
    if [end]
    return false
give [end]
show big[5] .
show big[1] .
```

```text
true
false
```


**Example 67**

```turtle
def add1[x]
    return x + 1
def [end]

def double[x]
    return x * 2
def [end]

x is scroll 3 into add1, double .
show x .
```

```text
8
```


**Example 68**

```turtle
def factorial[n]
    if ] n <= 1 [
        return 1
    else ]
        return n * factorial[n - 1]
    if [end]
def [end]

show factorial[5] .
```

```text
120
```


**Example 69**

```turtle
def double[x]
    return x * 2
def [end]

def apply[f, v]
    return f[v]
def [end]

show apply[double, 21] .
g = double
show g[4] .
```

```text
42
8
```


**Example 70**

```turtle
// Sum the numbers from 1 to n by recursion.
def total_to[n]
    if ] n <= 0 [
        return 0
    if [end]
    return n + total_to[n - 1]
def [end]

show "sum 1..10: ", total_to[10] .
show "sum 1..100: ", total_to[100] .
```

```text
sum 1..10: 55
sum 1..100: 5050
```


**Example 71**

```turtle
def ~helper[x]
    return x + 1
def [end]
show ~helper[1] .
```

```text
2
```


**Example 72**

```turtle
// Format a price with two decimals and a dollar sign.
def ~money[cents]
    return "${cents}.00"
def [end]

def price_line[item, cents]
    return item + ": " + ~money[cents]
def [end]

show price_line["Tea", 3] .
show price_line["Cake", 4] .
```

```text
Tea: $3.00
Cake: $4.00
```


**Example 73**

```turtle
// Two public functions share one check that callers never need.
def ~in_range[n, low, high]
    return n >= low && n <= high
def [end]

def check_age[age]
    if ] ~in_range[age, 0, 130] [
        return "ok"
    else ]
        return "age must be from 0 to 130"
    if [end]
def [end]

def check_score[score]
    if ] ~in_range[score, 0, 100] [
        return "ok"
    else ]
        return "score must be from 0 to 100"
    if [end]
def [end]

show check_age[30] .
show check_age[-1] .
show check_score[101] .
```

```text
ok
age must be from 0 to 130
score must be from 0 to 100
```


**Example 74**

```turtle
// Pad a value on the left so every row of a table lines up.
def ~pad[value, width]
    result = "{value}"
    [loop][length of result < width]
        result = " " + result
    [loop][end]
    return result
def [end]

def show_row[name, count]
    show ~pad[name, 8] + ~pad[count, 4] .
def [end]

show_row["apples", 12] .
show_row["figs", 3] .
```

```text
  apples  12
    figs   3
```


**Example 75**

```turtle
def greet[name, greeting]
    return "{greeting}, {name}!"
def [end]
show greet["Ann"] .
```

```text
turtle: line 4: function "greet" expects 2 argument(s), got 1
  4 | show greet["Ann"] .
```

This example stops with an error (exit code 1). It depends on something outside the example, or it shows an error on purpose.


**Example 76**

```turtle
def greet[name, greeting]
    if ] greeting == none [
        greeting = "Hello"
    if [end]
    return "{greeting}, {name}!"
def [end]

show greet["Ann", none] .
show greet["Ann", "Hi"] .
```

```text
Hello, Ann!
Hi, Ann!
```


**Example 77**

```turtle
nums = list [3, 1, 2]
show nums at get 0 .
show nums at length .
```

```text
3
3
```


**Example 78**

```turtle
nums = list [3, 1, 2]
add 4 to nums .
insert 9 to nums at 0 .
put 7 to nums at 1 .
show nums .
```

```text
[ 9, 7, 1, 2, 4 ]
```


**Example 79**

```turtle
nums = list [1]
put 5 to nums at 3 .
```

```text
turtle: line 2: index 3 out of range for list nums (length 1)
  2 | put 5 to nums at 3 .
```

This example stops with an error (exit code 1). It depends on something outside the example, or it shows an error on purpose.


**Example 80**

```turtle
nums = list [3, 1, 2]
sort nums .
reverse nums .
show nums .
```

```text
[ 3, 2, 1 ]
```


**Example 81**

```turtle
s = set [1, 2, 2]
show s .
```

```text
{ 1, 2 }
```


**Example 82**

```turtle
s = set [1, 2]
add 3 to s .
add 3 to s .
remove 2 from s .
remove 9 from s .
show s .
```

```text
{ 1, 3 }
```


**Example 83**

```turtle
s = set [1, 2]
show s contains 2 .
show s contains 5 .
show length of s .
```

```text
true
false
2
```


**Example 84**

```turtle
ages = map ["Alice": 30, "Bob": 25]
show ages at get "Alice" .
show ages contains "Carol" .
```

```text
30
false
```


**Example 85**

```turtle
ages = map ["Alice": 30]
put 31 to ages at "Alice" .
add 5 to ages at "Alice" .
put 25 to ages at "Bob" .
delete "Alice" from ages .
show ages .
```

```text
{ "Bob": 25 }
```


**Example 86**

```turtle
import data
nums = list [1, 2, 3]
new = nums process x give x * 10
show new .
show nums .
```

```text
[ 10, 20, 30 ]
[ 1, 2, 3 ]
```


**Example 87**

```turtle
import data
nums = list [1, 2, 3]
show nums keep x give x > 1 .
```

```text
[ 2, 3 ]
```


**Example 88**

```turtle
nums = list [3, 1, 2]
add 4 to nums .
remove 1 from nums .
insert 99 to nums at 0 .
put 7 to nums at 1 .
sort nums .
reverse nums .
```

```text
(prints nothing)
```


**Example 89**

```turtle
ages = map ["Alice": 30, "Bob": 25]
put 31 to ages at "Alice" .
delete "Bob" from ages .
```

```text
(prints nothing)
```


**Example 90**

```turtle
nums = list [3, 1, 2]
show length of nums .
show min of nums .
show max of nums .
```

```text
3
1
3
```


**Example 91**

```turtle
show nums at get 0 .
show nums at get[0] + nums at get[1] .
```

```text
turtle: line 1: undefined variable "nums"
  1 | show nums at get 0 .
```

This example stops with an error (exit code 1). It depends on something outside the example, or it shows an error on purpose.


**Example 92**

```turtle
nums = list [4, 9, 2]
show nums .
show nums at get[length of nums - 1] .
last = nums at length - 1
show nums at get last .
```

```text
[ 4, 9, 2 ]
2
2
```


**Example 93**

```turtle
show "a" + "b" at upper .
show ("a" + "b") at upper .
```

```text
aB
AB
```


**Example 94**

```turtle
a = list [1, 2]
b = list [2, 3]
show a + b .
show a - b .
```

```text
[ 1, 2, 2, 3 ]
[ 1 ]
```


**Example 95**

```turtle
nums = list [3, 1, 3, 2, 1]
show change nums to set .
show change "abc" to list .
```

```text
{ 3, 1, 2 }
[ "a", "b", "c" ]
```


**Example 96**

```turtle
ages = map ["Alice": 30, "Bob": 25]
[loop][name in change ages to keys]
    show "{name} is {ages at get name}" .
[loop][end]
```

```text
Alice is 30
Bob is 25
```


**Example 97**

```turtle
nums = list [4, 9, 2]
add 7 to nums .
remove 9 from nums .
show nums at index[7] .
show nums at slice[0, 2] .
show min[nums] .
show nums at pop .
show nums .
```

```text
2
[ 4, 2 ]
2
7
[ 4, 2 ]
```


**Example 98**

```turtle
nums = list [4, 9]
nums at add[7]
show nums .
```

```text
[ 4, 9, 7 ]
```


**Example 99**

```turtle
nums = list [4, 9, 2]
show nums at get[1] .
```

```text
9
```


**Example 100**

```turtle
nums = list [4, 9, 2]
nums at put[99, 1]
show nums .
```

```text
[ 4, 99, 2 ]
```


**Example 101**

```turtle
nums = list [4, 9, 2]
nums at insert[0, 1]
show nums .
```

```text
[ 4, 0, 9, 2 ]
```


**Example 102**

```turtle
nums = list [4, 9, 2, 9]
nums at remove[9]
show nums .
nums at remove[42]
show nums .
```

```text
[ 4, 2, 9 ]
[ 4, 2, 9 ]
```


**Example 103**

```turtle
nums = list [4, 9, 2]
show nums at pop .
show nums at pop[0] .
show nums .
```

```text
2
4
[ 9 ]
```


**Example 104**

```turtle
nums = list [4, 9, 2, 9]
show nums at slice[1, 3] .
show nums at slice[1] .
show nums .
```

```text
[ 9, 2 ]
[ 9, 2, 9 ]
[ 4, 9, 2, 9 ]
```


**Example 105**

```turtle
nums = list [4, 9, 2]
show nums at contains[2] .
show nums at find[7] .
```

```text
true
false
```


**Example 106**

```turtle
nums = list [4, 9, 2, 9]
show nums at index[9] .
show nums at index[7] .
```

```text
1
-1
```


**Example 107**

```turtle
nums = list [4, 9, 2, 9]
show nums at count[9] .
```

```text
2
```


**Example 108**

```turtle
nums = list [4, 9, 2]
nums at sort
show nums .
```

```text
[ 2, 4, 9 ]
```


**Example 109**

```turtle
nums = list [4, 9, 2]
nums at reverse
show nums .
```

```text
[ 2, 9, 4 ]
```


**Example 110**

```turtle
nums = list [4, 9, 2]
nums at clear
show nums .
```

```text
[]
```


**Example 111**

```turtle
nums = list [4, 9, 2]
show nums at len .
show nums at length .
```

```text
3
3
```


**Example 112**

```turtle
nums = list []
show nums at isempty .
nums at add[1]
show nums at isempty .
```

```text
true
false
```


**Example 113**

```turtle
nums = list [4, 9]
text = nums at tostring
show "the list is {text}" .
```

```text
the list is [ 4, 9 ]
```


**Example 114**

```turtle
colors = set ["red", "blue"]
colors at add["red"]
colors at add["green"]
show colors .
```

```text
{ "red", "blue", "green" }
```


**Example 115**

```turtle
colors = set ["red", "blue"]
colors at insert["green", 0]
show colors .
```

```text
{ "green", "red", "blue" }
```


**Example 116**

```turtle
colors = set ["red", "blue", "green"]
show colors at get[0] .
show colors at get[2] .
```

```text
red
green
```


**Example 117**

```turtle
colors = set ["red", "blue"]
colors at remove["red"]
colors at remove["pink"]
show colors .
```

```text
{ "blue" }
```


**Example 118**

```turtle
colors = set ["red", "blue"]
show colors at contains["red"] .
show colors at find["pink"] .
```

```text
true
false
```


**Example 119**

```turtle
colors = set ["red", "blue"]
show colors at count["blue"] .
show colors at count["pink"] .
```

```text
1
0
```


**Example 120**

```turtle
a = set [1, 2]
b = set [2, 3]
show a at union[b] .
show a .
```

```text
{ 1, 2, 3 }
{ 1, 2 }
```


**Example 121**

```turtle
a = set [1, 2]
b = set [2, 3]
show a at intersection[b] .
```

```text
{ 2 }
```


**Example 122**

```turtle
a = set [1, 2]
b = set [2, 3]
show a at difference[b] .
```

```text
{ 1 }
```


**Example 123**

```turtle
a = set [1, 2]
b = set [1, 2, 3]
show a at subset[b] .
show b at subset[a] .
```

```text
true
false
```


**Example 124**

```turtle
a = set [1, 2]
b = set [1, 2, 3]
show b at superset[a] .
show a at superset[b] .
```

```text
true
false
```


**Example 125**

```turtle
colors = set ["red", "blue"]
show colors at pop .
show colors .
```

```text
blue
{ "red" }
```


**Example 126**

```turtle
nums = set [4, 8, 1]
nums at sort
show nums .
```

```text
{ 1, 4, 8 }
```


**Example 127**

```turtle
colors = set ["red", "blue"]
colors at clear
show colors .
```

```text
{}
```


**Example 128**

```turtle
colors = set ["red", "blue"]
show colors at len .
```

```text
2
```


**Example 129**

```turtle
colors = set []
show colors at isempty .
```

```text
true
```


**Example 130**

```turtle
colors = set ["red"]
text = colors at tostring
show "the set is {text}" .
```

```text
the set is { "red" }
```


**Example 131**

```turtle
ages = map ["Alice": 30, "Bob": 25]
show ages at get["Alice"] .
```

```text
30
```


**Example 132**

```turtle
ages = map ["Alice": 30]
ages at add["Alice", 1]
ages at add["Cy", 5]
show ages .
```

```text
{ "Alice": 31, "Cy": 5 }
```


**Example 133**

```turtle
ages = map ["Alice": 30]
ages at put["Alice", 31]
ages at put["Dee", 19]
show ages .
```

```text
{ "Alice": 31, "Dee": 19 }
```


**Example 134**

```turtle
ages = map ["Alice": 30, "Bob": 25]
ages at delete["Alice"]
show ages .
```

```text
{ "Bob": 25 }
```


**Example 135**

```turtle
ages = map ["Alice": 30]
show ages at contains["Alice"] .
show ages at contains["Zed"] .
```

```text
true
false
```


**Example 136**

```turtle
ages = map ["Bob": 25, "Alice": 30]
show ages at getkeys .
```

```text
[ "Bob", "Alice" ]
```


**Example 137**

```turtle
ages = map ["Bob": 25, "Alice": 30]
show ages at getvalues .
```

```text
[ 25, 30 ]
```


**Example 138**

```turtle
ages = map ["Bob": 25, "Alice": 30]
show ages at invert .
show ages .
```

```text
{ 25: "Bob", 30: "Alice" }
{ "Bob": 25, "Alice": 30 }
```


**Example 139**

```turtle
ages = map ["Bob": 25, "Alice": 30]
show ages at len .
```

```text
2
```


**Example 140**

```turtle
ages = map []
show ages at isempty .
```

```text
true
```


**Example 141**

```turtle
ages = map ["Bob": 25]
text = ages at tostring
show "the map is {text}" .
```

```text
the map is { "Bob": 25 }
```


**Example 142**

```turtle
nums = list [4]
show nums at get[5] .
```

```text
turtle: line 2: index 5 out of range for list (length 1)
  2 | show nums at get[5] .
```

This example stops with an error (exit code 1). It depends on something outside the example, or it shows an error on purpose.


**Example 143**

```turtle
nums = list []
show nums at pop .
```

```text
turtle: line 2: 'pop' called on an empty list
  2 | show nums at pop .
```

This example stops with an error (exit code 1). It depends on something outside the example, or it shows an error on purpose.


**Example 144**

```turtle
ages = map ["Bob": 25]
show ages at get["Zed"] .
```

```text
turtle: line 2: key "Zed" not found in map
  2 | show ages at get["Zed"] .
```

This example stops with an error (exit code 1). It depends on something outside the example, or it shows an error on purpose.


**Example 145**

```turtle
colors = set ["red"]
colors at put["blue", 0]
```

```text
turtle: line 2: unknown set method "put"
  2 | colors at put["blue", 0]
```

This example stops with an error (exit code 1). It depends on something outside the example, or it shows an error on purpose.


**Example 146**

```turtle
ages = map ["Bob": 25]
ages at clear
```

```text
turtle: line 2: unknown map method "clear"
  2 | ages at clear
```

This example stops with an error (exit code 1). It depends on something outside the example, or it shows an error on purpose.


**Example 147**

```turtle
colors = set ["red", "blue"]
show colors at get[5] .
```

```text
turtle: line 2: index 5 out of range for set (length 2)
  2 | show colors at get[5] .
```

This example stops with an error (exit code 1). It depends on something outside the example, or it shows an error on purpose.


**Example 148**

```turtle
ages = map ["Bob": 25, "Ann": 25]
show ages at invert .
```

```text
{ 25: "Ann" }
warning: 25 is the value of both "Bob" and "Ann", so the inverted map keeps only "Ann" (line 2)
```


**Example 149**

```turtle
// A tiny phone book: look up, add and change entries.
book = map ["Ana": "555-0101", "Ben": "555-0102"]
book at put["Cy", "555-0103"]
book at put["Ben", "555-0199"]
show "Ana: ", book at get["Ana"] .
show "Ben: ", book at get["Ben"] .
show "entries: ", book at len .
show "Dee in book? ", book at contains["Dee"] .
```

```text
Ana: 555-0101
Ben: 555-0199
entries: 3
Dee in book? false
```


**Example 150**

```turtle
assemble Order [item, qty, price]

o = Order["pen", 3, 1.5]
show o .
show qty of o .
```

```text
Order { item: "pen", qty: 3, price: 1.5 }
3
```


**Example 151**

```turtle
assemble Order [item, qty, price]
o = Order["pen", 3, 1.5]
qty of o = 10
total = qty of o * price of o
show total .
```

```text
15.0
```


**Example 152**

```turtle
assemble Order [item, qty, price]
o = Order["pen", 3, 1.5]
qty of o = 10
show o .
price of o = price of o * 2
show o .
```

```text
Order { item: "pen", qty: 10, price: 1.5 }
Order { item: "pen", qty: 10, price: 3.0 }
```


**Example 153**

```turtle
assemble Order [item, qty, price]
orders = list [Order["pen", 3, 1.5], Order["ink", 1, 4.0]]
[loop][o in orders]
    qty of o = qty of o * 2
[loop][end]
show orders .
```

```text
[ Order { item: "pen", qty: 6, price: 1.5 }, Order { item: "ink", qty: 2, price: 4.0 } ]
```


**Example 154**

```turtle
assemble Order [item, qty, price]

def discount[o, pct]
    price of o = price of o * (100 - pct) / 100
def [end]

orders = list [Order["pen", 3, 1.5], Order["ink", 1, 4.0]]
discount[orders at get 0, 10]
show orders at get 0 .
```

```text
Order { item: "pen", qty: 3, price: 1.35 }
```


**Example 155**

```turtle
assemble Order [item, qty, price]
m = map ["a": Order["pen", 3, 1.5]]
qty of (m at get "a") = 7
show m at get "a" .
```

```text
Order { item: "pen", qty: 7, price: 1.5 }
```


**Example 156**

```turtle
assemble Order [item, qty, price]
o = Order["pen", 3, 1.5]
prise of o = 2
```

```text
turtle: line 3: Order has no field "prise" (its fields: item, qty, price)
  3 | prise of o = 2
```

This example stops with an error (exit code 1). It depends on something outside the example, or it shows an error on purpose.


**Example 157**

```turtle
assemble Point [x, y]

def midpoint[a, b]
    return Point[(x of a + x of b) / 2, (y of a + y of b) / 2]
def [end]

m = midpoint[Point[0, 0], Point[4, 6]]
show m .
```

```text
Point { x: 2.0, y: 3.0 }
```


**Example 158**

```turtle
// An assembled type gives a name and fields to a kind of record.
assemble Point [x, y]
p = Point[3, 4]
q = Point[-1, 2]
show p .
show x of p, " ", y of p .
x of p = 10
show p .
show p == Point[10, 4] .
```

```text
Point { x: 3, y: 4 }
3 4
Point { x: 10, y: 4 }
true
```


## Nested structures

From the book, `book/src/02-nested.md`.

**Example 159**

```turtle
school = map ["ann": map ["grades": list [90, 85]], "bo": map ["grades": list [70]]]
show school at get["ann"] at get["grades"] .
show school at get["ann"] at get["grades"] at get[1] .
```

```text
[ 90, 85 ]
85
```


**Example 160**

```turtle
import data
school = map ["ann": map ["grades": list [90, 85]], "bo": map ["grades": list [70]]]
g = school at get["bo"] at get["grades"]
add 80 to g .
show school at get["bo"] .
```

```text
{ "grades": [ 70, 80 ] }
```


**Example 161**

```turtle
school = map ["ann": map ["grades": list [90, 85]], "bo": map ["grades": list [70]]]
put list [1, 2] to school at "cy" .
delete "bo" from school .
show getkeys[school] .
```

```text
[ "ann", "cy" ]
```


**Example 162**

```turtle
import data
school = map ["ann": map ["grades": list [90, 85]]]
g = school at get["ann"] at get["grades"]
add 95 to g .
c = copy[school, true]
add 60 to g .
show c .
show school .
```

```text
{ "ann": { "grades": [ 90, 85, 95 ] } }
{ "ann": { "grades": [ 90, 85, 95, 60 ] } }
```


**Example 163**

```turtle
rows = list [map ["n": "a", "tags": list ["x"]], map ["n": "b", "tags": list []]]
[loop][r in rows]
    show r at get["n"], " has ", length of (r at get["tags"]), " tags" .
[loop][end]
```

```text
a has 1 tags
b has 0 tags
```


**Example 164**

```turtle
assemble Point [x, y]
s = set [Point[1, 2], Point[1, 2], Point[3, 4]]
show length of s .
m = map [Point[0, 0]: "origin"]
show m at get[Point[0, 0]] .
```

```text
2
origin
```


**Example 165**

```turtle
assemble Student [name, grades]
s = Student["Ann", list [90, 85]]
g = grades of s
add 70 to g .
show s .
```

```text
Student { name: "Ann", grades: [ 90, 85, 70 ] }
```


## Scrolls

From the book, `book/src/02-scrolls.md`.

**Example 166**

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

show half[double[add1[3]]] .
```

```text
4.0
```


**Example 167**

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

x is scroll 3 into add1, double, half .
show x .
```

```text
4.0
```


**Example 168**

```turtle
def add1[x]
    return x + 1
def [end]

show scroll 3 into add1, here * 10 .
show scroll 5 into list [here, here * 2] .
```

```text
40
[ 5, 10 ]
```


**Example 169**

```turtle
show scroll "  hi  " into at trim, at upper .
show scroll 4 into n give n * 5 .
```

```text
HI
20
```


**Example 170**

```turtle
def add1[x]
    return x + 1
def [end]

def double[x]
    return x * 2
def [end]

s = scroll add1, double .
show s[3] .
show typeof[s] .
```

```text
8
scroll
```


**Example 171**

```turtle
def add1[x]
    return x + 1
def [end]

def double[x]
    return x * 2
def [end]

first = scroll add1 .
second = scroll double .
show scroll 2 into first, second .
show scroll 2 into add1, double .
```

```text
6
6
```


**Example 172**

```turtle
import data
rows = list [map ["item": "pen", "amount": 1200], map ["item": "ink", "amount": 300], map ["item": "desk", "amount": 2100]]
big = scroll rows into keep r give r at get["amount"] > 1000 .
show length of big .
show scroll rows into process r give r at get["item"] .
show scroll rows into keep r give r at get["amount"] > 1000, process r give r at get["item"] .
```

```text
2
[ "pen", "ink", "desk" ]
[ "pen", "desk" ]
```


**Example 173**

```turtle
import data
clean = scroll process s give s at trim .
show clean[list [" a ", " b "]] .
```

```text
[ "a", "b" ]
```


**Example 174**

```turtle
show scroll 0 into 10 / here .
```

```text
turtle: line 1: scroll step 1 of 1 (10 / here): division by zero
  1 | show scroll 0 into 10 / here .
```

This example stops with an error (exit code 1). It depends on something outside the example, or it shows an error on purpose.


**Example 175**

```turtle
import data

orders = list [map ["id": 1, "amount": 1200], map ["id": 2, "amount": 300], map ["id": 3, "amount": 2100]]

total = scroll orders into keep o give o at get["amount"] >= 1000, process o give o at get["amount"], sum[here] .
show "total over 1000: {total}" .
```

```text
total over 1000: 3300
```


## Theories

From the book, `book/src/02-theories.md`.

**Example 176**

```turtle
theory tally
    abstract
        tally says how many times b appears in the list a.
    notation tally b in a .
    definition
        n = 0
        [loop][x in a]
            if ] x == b [
                n = n + 1
            if [end]
        [loop][end]
        return n
    theorem result >= 0
    theorem result <= length of a
    proof
        tally 1 in list [1, 1, 0, 3] . is 2
        tally 9 in list [] . is 0
theory [end]

nums = list [1, 1, 0, 3]
show "ones: ", tally 1 in nums .
```

```text
ones: 2
```


**Example 177**

```turtle
theory grab
    abstract
        grab takes the first item of a list.
    notation grab n in g .
    definition
        return g at get 0
theory [end]
show grab 1 in list [4] .
```

```text
turtle: main.turtle, line 8: this isn't how grab is written: grab n in g .
  8 | show grab 1 in list [4] .
    |      ^
```

This example stops with an error (exit code 1). It depends on something outside the example, or it shows an error on purpose.


**Example 178**

```turtle
theory tally
    abstract
        tally says how many times b appears in the list a.
    notation tally b in a .
    notation tally b among a .
    definition
        n = 0
        [loop][x in a]
            if ] x == b [
                n = n + 1
            if [end]
        [loop][end]
        return n
    theorem result >= 0
    theorem result <= length of a
    proof
        tally 1 in list [1, 1, 0, 3] . is 2
        tally 9 in list [] . is 0
        tally 1 among list [1, 1] . is 2
theory [end]

nums = list [1, 1, 0, 3]
show "ones: ", tally 1 in nums .
show "ones among: ", tally 1 among nums .
```

```text
ones: 2
ones among: 2
```


**Example 179**

```turtle
import test
theory tally
    abstract
        tally says how many times b appears in the list a.
    notation tally b in a .
    notation tally b among a .
    definition
        n = 0
        [loop][x in a]
            if ] x == b [
                n = n + 1
            if [end]
        [loop][end]
        return n
    theorem result >= 0
    theorem result <= length of a
    proof
        tally 1 in list [1, 1, 0, 3] . is 2
        tally 9 in list [] . is 0
        tally 1 among list [1, 1] . is 2
theory [end]

nums = list [1, 1, 0, 3]
show "ones: ", tally 1 in nums .
show "ones among: ", tally 1 among nums .
```

```text
ones: 2
ones among: 2
```


**Example 180**

```turtle
theory tally
    abstract
        tally says how many times b appears in the list a.
    notation tally b in a .
    definition
        n = 0
        [loop][x in a]
            if ] x == b [
                n = n + 1
            if [end]
        [loop][end]
        return n
    theorem result >= 0
    theorem result <= length of a
    proof
        tally 1 in list [1, 1, 0, 3] . is 2
        tally 9 in list [] . is 0
theory [end]
nums = list [1, 1, 0, 3]
show tally 1 in nums == 2 .
```

```text
true
```


**Example 181**

```turtle
theory tally
    abstract
        tally says how many times b appears in the list a.
    notation tally b in a .
    definition
        n = 0
        [loop][x in a]
            if ] x == b [
                n = n + 1
            if [end]
        [loop][end]
        return n
    theorem result >= 0
    theorem result <= length of a
    proof
        tally 1 in list [1, 1, 0, 3] . is 2
        tally 9 in list [] . is 0
theory [end]
nums = list [1, 1, 0, 3]
show tally 1 in nums + 1 .
```

```text
turtle: line 20: '+' can't add list and integer — a list, set or map only adds to another of its own kind, e.g. list [1] + list [2]
  20 | show tally 1 in nums + 1 .
```

This example stops with an error (exit code 1). It depends on something outside the example, or it shows an error on purpose.


**Example 182**

```turtle
def make_counter[]
    state = list [0]
    def next[]
        state at put[state at get[0] + 1, 0] .
        return state at get[0]
    def [end]
    return next
def [end]

theory yield
    abstract
        yield says the next value of gen.
    notation yield gen .
    definition
        value = gen[]
        return value
theory [end]

c = make_counter[]
show yield c .
show yield c .
show yield c .
```

```text
1
2
3
```


**Example 183**

```turtle
// A theory phrase that tests whether a value lies between two bounds, used as a sentence.
theory within
    abstract
        within x, lo, hi says whether x lies from lo to hi inclusive.
    notation within x, lo, hi .
    definition
        return x >= lo && x <= hi
    theorem result == (x >= lo && x <= hi)
    proof
        within 5, 1, 10 . is true
        within 11, 1, 10 . is false
theory [end]

[loop][t in list [0, 4, 10, 12]]
    ok is within t, 1, 10 .
    show t, " in range: ", ok .
[loop][end]
```

```text
0 in range: false
4 in range: true
10 in range: true
12 in range: false
```


**Example 184**

```turtle
theory within
    abstract
        within x, lo, hi says whether x lies from lo to hi inclusive.
    notation within x, lo, hi .
    definition
        return x >= lo && x <= hi
    theorem result == (x >= lo && x <= hi)
    proof
        within 5, 1, 10 . is true
theory [end]

theory squared
    abstract
        squared x says the square of x.
    notation squared x .
    definition
        return x * x
    theorem result >= 0
    proof
        squared 3 . is 9
theory [end]

[loop][n in list [2, 11]]
    show n, " -> ", within squared n, 0, 100 .
[loop][end]
```

```text
2 -> true
11 -> false
```


**Example 185**

```turtle
def square[x]
    return x * x
def [end]

theory squared
    abstract
        squared x says the square of x, computed by a function outside this theory.
    notation squared x .
    definition
        return square[x]
    theorem result >= 0
    proof
        squared 3 . is 9
theory [end]

show squared 4 .
```

```text
16
```


**Example 186**

```turtle
theory squared
    abstract
        squared x says the square of x.
    notation squared x .
    definition
        return x * x
    theorem result >= 0
    proof
        squared 3 . is 9
theory [end]

theory within
    abstract
        within x, lo, hi says whether x lies from lo to hi inclusive.
    notation within x, lo, hi .
    definition
        return x >= lo && x <= hi
    theorem result == (x >= lo && x <= hi)
    proof
        within 5, 1, 10 . is true
theory [end]

theory small_square
    abstract
        small_square x says whether the square of x lies within 0 and 100.
    notation small_square x .
    definition
        return within squared x, 0, 100
    theorem result == (squared x >= 0 && squared x <= 100)
    proof
        small_square 5 . is false
        small_square 3 . is true
theory [end]

show small_square 3 .
show small_square 5 .
```

```text
true
true
```


**Example 187**

```turtle
theory outer
    abstract
        outer x says x plus one, using another theory.
    notation outer x .
    definition
        return inner x
    theorem result == x + 1
    proof
        outer 2 . is 3
theory [end]
theory inner
    abstract
        inner x says x plus one.
    notation inner x .
    definition
        return x + 1
theory [end]
show outer 4 .
```

```text
turtle: main.turtle, line 6: inner is defined by a theory on line 11, below this line: a theory comes before its word is used
  6 |         return inner x
    |                ^
```

This example stops with an error (exit code 1). It depends on something outside the example, or it shows an error on purpose.


**Example 188**

```turtle
theory outer
    abstract
        outer x says x plus one, built from an inner theory.
    notation outer x .
    definition
        theory inner
            abstract
                inner x says x plus one.
            notation inner x .
            definition
                return x + 1
        theory [end]
        return inner x
    theorem result == x + 1
theory [end]
show outer 4 .
```

```text
turtle: line 6: theory inner: a theory is written at the top level of a file, not inside other code
  6 |         theory inner
```

This example stops with an error (exit code 1). It depends on something outside the example, or it shows an error on purpose.


**Example 189**

```turtle
x = 1 div 0
show "carrying on" .
```

```text
turtle: line 1: division by zero
  1 | x = 1 div 0
```

This example stops with an error (exit code 1). It depends on something outside the example, or it shows an error on purpose.


**Example 190**

```turtle
safe
    [read] scores.txt to lines [end]
handle [file] error .
    show "no scores yet: ", error .
    lines = list []
safe [end]
show length of lines .
```

```text
no scores yet: line 2: [read] scores.txt: no such file or folder
0
```


**Example 191**

```turtle
safe
    x = 10 div 0
handle [math] error .
    show "kind ", kind of error, " line ", line of error .
safe [end]
show "carrying on" .
```

```text
kind math line 2
carrying on
```


**Example 192**

```turtle
values = list [10, 0]
m = map ["a": 1]
safe
    show 10 / values at get 1 .
    show m at get "zz" .
handle [math, key] problem .
    show "handled: ", problem .
safe [end]
show "carrying on" .
```

```text
handled: line 4: division by zero
carrying on
```


**Example 193**

```turtle
safe
    show 1 / 0 .
handle [math, data, sys] e .
    show e .
safe [end]
```

```text
turtle: main.turtle, line 3: "data" isn't a kind of error; handle can list file, number, math, index, key, name, type, json, date, http, sql, csv, test, pattern, crypt, schedule, config, server, scroll, linear, custom, or [] for any error
  3 | handle [math, data, sys] e .
    |               ^
```

This example stops with an error (exit code 1). It depends on something outside the example, or it shows an error on purpose.


**Example 194**

```turtle
def withdraw[balance, amount]
    if ] amount > balance [
        fail "not enough money"
    if [end]
    return balance - amount
def [end]
```

```text
(prints nothing)
```


**Example 195**

```turtle
def withdraw[balance, amount]
    if ] amount > balance [
        fail "not enough money"
    if [end]
    return balance - amount
def [end]
safe
    x = withdraw[10, 50]
handle [] error .
    show "kind ", kind of error, ": ", message of error .
safe [end]
```

```text
kind custom: not enough money
```


**Example 196**

```turtle
// Division by zero is an error. Catch it with safe and handle, and print a fallback value instead.
pairs = list [list [10, 2], list [7, 0], list [9, 3]]
[loop][p in pairs]
    result = "n/a"
    safe
        result = p at get[0] / p at get[1]
    handle [] error .
        result = "error"
    safe [end]
    show p, " -> ", result .
[loop][end]
```

```text
[ 10, 2 ] -> 5.0
[ 7, 0 ] -> error
[ 9, 3 ] -> 3.0
```


**Example 197**

```turtle
def f[x]
    return x
def [end]
def f[x]
    return x + 1
def [end]
show f[1] .
```

```text
2
warning: f is defined more than once: the def on line 4 replaces the one on line 1 (calls use the newer one once its line has run)
```


**Example 198**

```turtle
count = 0
def bump[n]
    count = n
def [end]
bump[2]
show count .
```

```text
0
turtle: note: main.turtle, line 3: count is also a top-level variable, so this makes a new local count in bump and the top-level count is not changed. To change it, return the new value and assign that, or keep it in a list or map
```


**Example 199**

```turtle
  // lib/setup.turtle
  show "module setup running" .
  x = 1
  
```

```text
module setup running
```


**Example 200**

```turtle
  show "before import" .
  import lib/setup
  show "after import" .
  
```

```text
before import
turtle: line 2: import lib/setup: lib/setup.turtle: no such file or folder
  2 |   import lib/setup
```

This example stops with an error (exit code 1). It depends on something outside the example, or it shows an error on purpose.


**Example 201**

```turtle
import data                    // a library that comes with Turtle
import time [now, sleep]       // only some of its names
import shop                    // shop.turtle, next to this program
import lib/money               // lib/money.turtle; its name is "money"
```

```text
turtle: line 3: import shop: shop.turtle: no such file or folder
  3 | import shop                    // shop.turtle, next to this program
```

This example stops with an error (exit code 1). It depends on something outside the example, or it shows an error on purpose.


**Example 202**

```turtle
x = sum[list [1, 2]]
```

```text
turtle: line 1: "sum" needs "import data" first
  1 | x = sum[list [1, 2]]
```

This example stops with an error (exit code 1). It depends on something outside the example, or it shows an error on purpose.


**Example 203**

```turtle
// shop.turtle
taxrate = 0.08

def withtax[amount]
    return amount + amount * taxrate
def [end]
```

```text
(prints nothing)
```


**Example 204**

```turtle
import shop
show withtax[100] .
```

```text
turtle: line 1: import shop: shop.turtle: no such file or folder
  1 | import shop
```

This example stops with an error (exit code 1). It depends on something outside the example, or it shows an error on purpose.


**Example 205**

```turtle
import shop
show shop withtax[50] .
```

```text
turtle: line 1: import shop: shop.turtle: no such file or folder
  1 | import shop
```

This example stops with an error (exit code 1). It depends on something outside the example, or it shows an error on purpose.


**Example 206**

```turtle
import time [now]
show "imported names only" .
show now[] > 0 .
```

```text
imported names only
true
```


**Example 207**

```turtle
import time [now]
show sleep[0] .
```

```text
turtle: line 2: "sleep" isn't imported — add it to "import time [...]"
  2 | show sleep[0] .
```

This example stops with an error (exit code 1). It depends on something outside the example, or it shows an error on purpose.


**Example 208**

```turtle
// shop.turtle
taxrate = 0.08

def withtax[amount]
    return amount + amount * taxrate
def [end]
```

```text
(prints nothing)
```


**Example 209**

```turtle
// store.turtle
taxrate = 0.20

def withtax[amount]
    return amount + amount * taxrate
def [end]
```

```text
(prints nothing)
```


**Example 210**

```turtle
import shop
import store
show shop withtax[10] .
show store withtax[10] .
```

```text
turtle: line 1: import shop: shop.turtle: no such file or folder
  1 | import shop
```

This example stops with an error (exit code 1). It depends on something outside the example, or it shows an error on purpose.


**Example 211**

```turtle
import shop
f = shop withtax
show f[10] .
```

```text
turtle: line 1: import shop: shop.turtle: no such file or folder
  1 | import shop
```

This example stops with an error (exit code 1). It depends on something outside the example, or it shows an error on purpose.


**Example 212**

```turtle
import shop
import store
show withtax[10] .
```

```text
turtle: line 1: import shop: shop.turtle: no such file or folder
  1 | import shop
```

This example stops with an error (exit code 1). It depends on something outside the example, or it shows an error on purpose.


**Example 213**

```turtle
import data [tablerows = 50]
import log [loglevel = "debug"]
import random [seed = 7]
import data [tablerows = 50, process]
```

```text
(prints nothing)
```


**Example 214**

```turtle
import shop [taxrate]
```

```text
turtle: line 1: import shop: shop.turtle: no such file or folder
  1 | import shop [taxrate]
```

This example stops with an error (exit code 1). It depends on something outside the example, or it shows an error on purpose.


**Example 215**

```turtle
// shop.turtle
taxrate = 0.08

def withtax[amount, rate]
    if ] rate == none [
        rate = taxrate
    if [end]
    return amount + amount * rate
def [end]
```

```text
(prints nothing)
```


**Example 216**

```turtle
import shop
show withtax[100, none] .
show withtax[100, 0.1] .
```

```text
turtle: line 1: import shop: shop.turtle: no such file or folder
  1 | import shop
```

This example stops with an error (exit code 1). It depends on something outside the example, or it shows an error on purpose.


**Example 217**

```turtle
// shop.turtle
taxrate = 0.08

def withtax[amount, settings]
    rate = taxrate
    if ] settings contains "rate" [
        rate = settings at get "rate"
    if [end]
    return amount + amount * rate
def [end]
```

```text
(prints nothing)
```


**Example 218**

```turtle
import shop
show withtax[100, map []] .
show withtax[100, map ["rate": 0.2]] .
```

```text
turtle: line 1: import shop: shop.turtle: no such file or folder
  1 | import shop
```

This example stops with an error (exit code 1). It depends on something outside the example, or it shows an error on purpose.


**Example 219**

```turtle
import system
import shop

opts = options["--rate": 0.08]
show withtax[100, map ["rate": opts at get "--rate"]] .
```

```text
turtle: line 2: import shop: shop.turtle: no such file or folder
  2 | import shop
```

This example stops with an error (exit code 1). It depends on something outside the example, or it shows an error on purpose.


**Example 220**

```turtle
// shop.turtle
settings = map ["rate": 0.08]

// configure copies each new setting into the module's map.
def configure[changes]
    [loop][key in changes getkeys]
        value = changes at get key
        put value to settings at key .
    [loop][end]
def [end]

def withtax[amount]
    return amount + amount * settings at get "rate"
def [end]
```

```text
(prints nothing)
```


**Example 221**

```turtle
import shop
show withtax[100] .
configure[map ["rate": 0.25]]
show withtax[100] .
```

```text
turtle: line 1: import shop: shop.turtle: no such file or folder
  1 | import shop
```

This example stops with an error (exit code 1). It depends on something outside the example, or it shows an error on purpose.


**Example 222**

```turtle
import shop
import data

rate = 0.08
orders = list [100, 250, 40]

total = 0
[loop][o in orders]
    total = total + withtax[o]
[loop][end]
show "total: {total}" .

def helper[x]
    return x
def [end]
```

```text
turtle: line 1: import shop: shop.turtle: no such file or folder
  1 | import shop
```

This example stops with an error (exit code 1). It depends on something outside the example, or it shows an error on purpose.


**Example 223**

```turtle
// shop.turtle
def clamp[n]
    return ~limit[n, 0, 100]
def [end]

// Private: only this file can call it.
def ~limit[n, low, high]
    if ] n < low [
        return low
    if [end]
    if ] n > high [
        return high
    if [end]
    return n
def [end]
```

```text
(prints nothing)
```


**Example 224**

```turtle
import shop
show clamp[150] .
show clamp[-5] .
```

```text
turtle: line 1: import shop: shop.turtle: no such file or folder
  1 | import shop
```

This example stops with an error (exit code 1). It depends on something outside the example, or it shows an error on purpose.


**Example 225**

```turtle
import shop
show shop ~limit[1, 0, 10] .
```

```text
turtle: line 1: import shop: shop.turtle: no such file or folder
  1 | import shop
```

This example stops with an error (exit code 1). It depends on something outside the example, or it shows an error on purpose.


**Example 226**

```turtle
import shop [~limit]
show ~limit[1, 0, 10] .
```

```text
turtle: line 1: import shop: shop.turtle: no such file or folder
  1 | import shop [~limit]
```

This example stops with an error (exit code 1). It depends on something outside the example, or it shows an error on purpose.


**Example 227**

```turtle
import data [tablerows = 50]

rows = list [1, 2, 3, 4, 5, 6, 7, 8]

def show_top[items, limit]
    tablerows = limit
    show table[items] .
def [end]

show "file setting: {tablerows}" .
show_top[rows, 2] .
show_top[rows, 4] .
show "after the calls: {tablerows}" .
```

```text
file setting: 50
#  value
-  -----
0      1
1      2
... 6 more rows
#  value
-  -----
0      1
1      2
2      3
3      4
... 4 more rows
after the calls: 50
```


## Sentence form

From the book, `book/src/02z-sentence-form.md`.

**Example 228**

```turtle
nums = list [4, 9, 2]
add 7 to nums .
show nums .
```

```text
[ 4, 9, 2, 7 ]
```


**Example 229**

```turtle
import data
import strings
nums = list [4, 9, 2]
show nums sum .
show nums at length .
show "turtle reads code" upper .
```

```text
15
3
TURTLE READS CODE
```


**Example 230**

```turtle
s = set [1, 2]
add 3 to s .
remove 1 from s .
show s .
show s at contains 3 .
```

```text
{ 2, 3 }
true
```


**Example 231**

```turtle
def fix[a, b]
    return a + b
def [end]
show fix 1, 2 .
show fix 1, 2; " is the sum" .
```

```text
3
3 is the sum
```


**Example 232**

```turtle
import data
prices = list [100, 250, 40]
big = prices keep p give p > 50
show big .
sale = prices process p give p - p * 10 div 100
show sale .
show big sum .
```

```text
[ 100, 250 ]
[ 90, 225, 36 ]
350
```


**Example 233**

```turtle
theory double
    abstract
        double says twice n.
    notation double n .
    definition
        return n * 2
theory [end]
show double 4 .
x = 5
show double x .
```

```text
8
10
```


**Example 234**

```turtle
  import data
  prices = list [100, 250, 40]
  big = prices keep p give p > 50
  total = big sum
  show "{total} from {big at length} items" .
  
```

```text
350 from 2 items
```


**Example 235**

```turtle
  words = list ["ann", "ben"]
  if ] words at contains "ann" [
      show "ann is here" .
  if [end]
  
```

```text
ann is here
```


**Example 236**

```turtle
import data
def discount[price, pct]
    return price - price * pct div 100
def [end]

prices = list [100, 250, 40]
big = prices keep p give p > 50
sale = big process p give discount p, 10
total = sale sum
show "sale total: {total}" .
if ] total > 300 [
    show "above the threshold" .
if [end]
```

```text
sale total: 315
above the threshold
```


## Collections and data

From the book, `book/src/03a-collections.md`.

**Example 237**

```turtle
import data

orders = list [100, 250, 40]
with_tax = orders process x give x * 1.08
show with_tax .
```

```text
[ 108.0, 270.0, 43.2 ]
```


**Example 238**

```turtle
import data

nums = list [1, 2, 3, 4]
show nums keep x give x > 2 .
show nums reduce 0, [total, x] give total + x .
show nums sum .
```

```text
[ 3, 4 ]
10
10
```


**Example 239**

```turtle
import sort

rows = list [map ["n": "b", "v": 2], map ["n": "a", "v": 1]]
show min_sort[rows, "v", "first"] .
```

```text
{ "n": "a", "v": 1 }
```


**Example 240**

```turtle
import sort

nums = list [5, 2, 9, 1]
show quick_sort[nums, x give x] .
```

```text
[ 1, 2, 5, 9 ]
```


**Example 241**

```turtle
import search

nums = list [4, 8, 15, 16, 23, 42]
show find_first[nums, x give x > 10] .
show count_where[nums, x give x % 2 == 0] .
```

```text
15
4
```


**Example 242**

```turtle
import search

nums = list [4, 8, 15, 16, 23, 42]
show binary_search[nums, 16, x give x] .
```

```text
3
```


**Example 243**

```turtle
import data

words = list ["ant", "bee", "ax"]
show group[words, w give w at slice[0, 1]] .
show unique[list [1, 1, 2]] .
```

```text
{ "a": [ "ant", "ax" ], "b": [ "bee" ] }
[ 1, 2 ]
```


**Example 244**

```turtle
// Unique values and grouping of records by a function.
import data
nums = list [3, 1, 3, 2, 1, 5]
show "unique: ", unique[nums] .
def parity[n]
    return n % 2
def [end]

groups = group[nums, parity]
show groups .
```

```text
unique: [ 3, 1, 2, 5 ]
{ 1: [ 3, 1, 3, 1, 5 ], 0: [ 2 ] }
```


**Example 245**

```turtle
// Summary of monthly sales figures with the data library: total, average and highest month.
import data
sales = list [map ["month": "Jan", "amount": 1200], map ["month": "Feb", "amount": 1850], map ["month": "Mar", "amount": 990], map ["month": "Apr", "amount": 2100]]
amounts = list []
[loop][s in sales]
    add s at get["amount"] to amounts .
[loop][end]
show "total: ", sum[amounts] .
show "average: ", mean[amounts] .
show "maximum: ", max of amounts .
```

```text
total: 6140
average: 1535.0
maximum: 2100
```


**Example 246**

```turtle
import data
rows = list [map ["title": "Emma", "copies": 12], map ["title": "Ida", "copies": 4]]
show table[rows] .
show table[rows, 1] .
```

```text
title  copies
-----  ------
Emma       12
Ida         4
title  copies
-----  ------
Emma       12
... 1 more row
```


**Example 247**

```turtle
import data

sales = table_read["sales.csv"]
show table[sales, 3] .
```

```text
turtle: line 3: table_read sales.csv: no such file or folder
  3 | sales = table_read["sales.csv"]
```

This example stops with an error (exit code 1). It depends on something outside the example, or it shows an error on purpose.


**Example 248**

```turtle
import data
sales = table_read["sales.csv"]
show sales mean "copies" .
show sales median "copies" .
show describe[sales, "copies"] .
```

```text
turtle: line 2: table_read sales.csv: no such file or folder
  2 | sales = table_read["sales.csv"]
```

This example stops with an error (exit code 1). It depends on something outside the example, or it shows an error on purpose.


**Example 249**

```turtle
big = sales keep s give copies of s > 10
table_write["big.csv", big]
```

```text
turtle: line 1: unknown module or variable "sales" in front of a function name
  1 | big = sales keep s give copies of s > 10
```

This example stops with an error (exit code 1). It depends on something outside the example, or it shows an error on purpose.


**Example 250**

```turtle
import data
codes = table_read["codes.csv", map ["code": "text"]]
```

```text
turtle: line 2: table_read codes.csv: no such file or folder
  2 | codes = table_read["codes.csv", map ["code": "text"]]
```

This example stops with an error (exit code 1). It depends on something outside the example, or it shows an error on purpose.


**Example 251**

```turtle
import data
heights = list [150, 160, 170, 180]
weights = list [50, 58, 70, 81]
show covariance[heights, weights] .
show mode[list ["tea", "coffee", "tea"]] .
show pstdev[list [2, 4, 4, 4, 5, 5, 7, 9]] .
show pvariance[list [2, 4, 4, 4, 5, 5, 7, 9]] .
```

```text
175.0
tea
2.0
4.0
```


**Example 252**

```turtle
import sort
show is_sorted[list [1, 2, 2, 5], x give x] .
show reverse_list[list [1, 2, 3]] .
show max_sort[list [4, 9, 1, 7], s give s, 2] .
show bubble_sort[list [3, 1, 2], x give x] .
show counting_sort[list [3, 1, 2], x give x] .
show radix_sort[list [30, 4, 12], x give x] .
```

```text
true
[ 3, 2, 1 ]
[ 9, 7 ]
[ 1, 2, 3 ]
[ 1, 2, 3 ]
[ 4, 12, 30 ]
```


**Example 253**

```turtle
import search
names = list ["Ann", "Bo", "Cy"]
ages = map ["Ann": 30, "Bo": 25]
show linear_search[names, "Bo"] .
show find_key[ages, 25] .
show find_index[names, x give x == "Cy"] .
show find_last[list [1, 4, 6, 8], x give x % 2 == 0] .
show jump_search[list [1, 3, 5, 7], 5, x give x] .
show exponential_search[list [1, 3, 5, 7], 7, x give x] .
show ternary_search[list [1, 3, 5, 7], 3, x give x] .
```

```text
1
Bo
2
8
2
3
1
```


**Example 254**

```turtle
import data
people = list [map ["first": "Ann", "last": "Lee"], map ["first": "Bob", "last": "Park"]]
show table_column[people, "last"] .
show table_row[people, 1] .
```

```text
[ "Lee", "Park" ]
{ "first": "Bob", "last": "Park" }
```


**Example 255**

```turtle
import sort
show selection_sort[list [3, 1, 2], x give x] .
show shell_sort[list [5, 2, 9], x give x] .
```

```text
[ 1, 2, 3 ]
[ 2, 5, 9 ]
```


## Numbers, chance and text

From the book, `book/src/03b-numbers-text.md`.

**Example 256**

```turtle
import strings
line = "order 42 shipped on 2026-10-30"
show line find "shipped" .
show line substring 0, 5 .
show list ["a", "b", "c"] join "-" .
```

```text
9
order
a-b-c
```


**Example 257**

```turtle
import pattern

line = "order 42 shipped on 2026-10-30"
show findall[line, `\d+`] .
show replaceall[line, `\d{4}-\d{2}-\d{2}`, "DATE"] .
```

```text
[ "42", "2026", "10", "30" ]
order 42 shipped on DATE
```


**Example 258**

```turtle
import pattern

show splitby["a, b;c", `[,;]\s*`] .
show groups["2026-10-30", `(\d+)-(\d+)-(\d+)`] .
```

```text
[ "a", "b", "c" ]
[ "2026", "10", "30" ]
```


**Example 259**

```turtle
// Find email-like words and digit groups in a line of text with the pattern library.
import pattern
text = "contact ana at 555-0199 or ben at 555-0042 today"
show "numbers: ", findall[text, "[0-9]+-[0-9]+"] .
```

```text
numbers: [ "555-0199", "555-0042" ]
```


**Example 260**

```turtle
import strings
// Capitalise the first letter of every word in a sentence.
text = "the quick brown fox jumps over the lazy dog"
words = text at split " "
out = list []
[loop][w in words]
    cap = w at slice[0, 1] at upper + w at slice[1, w at length]
    add cap to out .
[loop][end]
show out at len, " words" .
show join[out, " "] .
```

```text
9 words
The Quick Brown Fox Jumps Over The Lazy Dog
```


**Example 261**

```turtle
import linear

a = matrix [
    2, 1
    1, 3
]
x = a solve list [3, 5]
show x .
show a * x .
```

```text
[ 0.8, 1.4 ]
[ 3.0, 5.0 ]
```


**Example 262**

```turtle
import linear
a = matrix [
    2, 1
    1, 3
]
show all_close[inverse[a] * a, identity[2]] .
```

```text
true
```


**Example 263**

```turtle
// Multiply two 2 by 2 matrices with the linear library and show the determinant of the product.
import linear
A = matrix [
    1, 2
    3, 4
]
B = matrix [
    0, 1
    1, 0
]
show A * B .
show determinant[A * B] .
```

```text
[ 2  1 ]
[ 4  3 ]
2
```


**Example 264**

```turtle
import random

show random list of 5 integers from 0 to 9 .
show list ["red", "green", "blue"] pick .
```

```text
[ 3, 8, 6, 3, 6 ]
red
```


**Example 265**

```turtle
import random

show chance[0.5] .
```

```text
false
```


**Example 266**

```turtle
import math
show 16 at sqrt .
show 3.14159 at round 2 .
show 2 at pow 10 .
```

```text
4.0
3.14
1024
```


**Example 267**

```turtle
// The random library writes values as sentences: integers, lists, strings and dates.
import random
seed = 384
show random integer from 1 to 6 .
show random list of 5 integers from 0 to 9 .
show random string of 6 .
show random digits of 4 .
show random list of 3 integers .
```

```text
3
[ 5, 9, 2, 8, 1 ]
BONeDy
4415
[ -67, -678, -727 ]
```


**Example 268**

```turtle
import math
show 1 at atan2[-1] .
show 1 at asin .
show 1 at cos .
show 2 at exp .
```

```text
2.356194490192345
1.5707963267948966
0.5403023058681398
7.38905609893065
```


**Example 269**

```turtle
import math
show 4.7 at floor .
show -4.2 at floor .
show 1 at ceil .
show 2.5 at round[0] .
show 1 at abs .
```

```text
4
-5
1
3
1
```


**Example 270**

```turtle
import math
show (0.1 + 0.2) at is_close[0.3] .
show (10 / 3) at is_close[3.33, 0.01] .
```

```text
true
true
```


**Example 271**

```turtle
import linear
a = matrix [
    1, 2
    3, 4
]
b = matrix [
    5, 6
    7, 8
]
show dot[list [1, 2], list [3, 4]] .
show norm[list [3, 4]] .
show unit[list [3, 4]] .
show diagonal[a] .
show flatten[a] .
show multiply_each[a, b] .
```

```text
11
5.0
[ 0.6, 0.8 ]
[ 1, 4 ]
[ 1  2  3  4 ]
[  5  12 ]
[ 21  32 ]
```


**Example 272**

```turtle
show "7" at padleft[3] .
show "ab" at padright[5] + "|" .
show "hello" at indexof["l"] .
show "Hello" at lower .
show "  x  " at trim .
show "hello" at slice[1, 3] .
show "hello" at reverse .
show "a-b-c" at split["-"] .
```

```text
  7
ab   |
2
hello
x
el
olleh
[ "a", "b", "c" ]
```


**Example 273**

```turtle
import math
show pi .
show 0 at sin .
show 1 at tan at round[4] .
show 0 at acos .
show 1 at atan at round[4] .
```

```text
3.141592653589793
0.0
1.5574
1.5707963267948966
0.7854
```


**Example 274**

```turtle
import random
seed = 3
show random list of 4 letters .
show random list of 3 booleans .
show random string of 5 .
```

```text
[ "T", "u", "o", "P" ]
[ false, true, false ]
pVwnn
```


**Example 275**

```turtle
import linear
show cross[list [1, 0, 0], list [0, 1, 0]] .
```

```text
[ 0, 0, 1 ]
```


**Example 276**

```turtle
show "12" at isnumber .
show "x" at isnumber .
```

```text
true
false
```


## Files, system and settings

From the book, `book/src/03c-system.md`.

**Example 277**

```turtle
[read] notes.txt to lines [end]
show lines .
```

```text
turtle: line 1: [read] notes.txt: no such file or folder
  1 | [read] notes.txt to lines [end]
```

This example stops with an error (exit code 1). It depends on something outside the example, or it shows an error on purpose.


**Example 278**

```turtle
[write] notes.txt
"first line"
"second line"
[end]

[append] notes.txt
"third line"
[end]
```

```text
(prints nothing)
```


**Example 279**

```turtle
[write] report.txt
"total is {total}"
[end]
```

```text
turtle: line 1: undefined variable "total"
  1 | [write] report.txt
```

This example stops with an error (exit code 1). It depends on something outside the example, or it shows an error on purpose.


**Example 280**

```turtle
import system
show exists["notes.txt"] .
show isfile["notes.txt"] .
makefolder["out"]
copyto["notes.txt", "out/notes.txt", true]
show contents["out"] .
```

```text
false
false
turtle: line 5: copyto notes.txt: no such file or folder
  5 | copyto["notes.txt", "out/notes.txt", true]
```

This example stops with an error (exit code 1). It depends on something outside the example, or it shows an error on purpose.


**Example 281**

```turtle
// Read command-line arguments; with none given the program uses a default.
import system
count = args[] at len
if ] count == 0 [
    show "no arguments, using default" .
else ]
    show "first argument: ", args[] at get[0] .
if [end]
```

```text
no arguments, using default
```


**Example 282**

```turtle
import system
opts = options["--out": "a.csv", "-v": false, "--count": 10]
show opts .
show args[] .
```

```text
{ "--out": "a.csv", "-v": false, "--count": 10 }
[]
```


**Example 283**

```turtle
import config

settings = map ["port": 8080, "debug": false, "name": "shop"]
config_write["app.toml", settings]
back = config_read["app.toml"]
show back at get "port" .
```

```text
8080
```


**Example 284**

```turtle
import system
key = env["SHOP_KEY"]
```

```text
(prints nothing)
```


**Example 285**

```turtle
sys echo hello from turtle
```

```text
hello from turtle
```


**Example 286**

```turtle
import system
makefolder["out/2026"]
[write] out/2026/report.txt
"first line"
[end]
show isfolder["out"] .
show isfolder["out/2026/report.txt"] .
show walk["out"] .
copyto["out/2026/report.txt", "backup/report.txt"]
show contents["backup"] .
moveto["backup/report.txt", "archive/old.txt"]
show walk["archive"] .
pack["out", "out.zip"]
unpack["out.zip", "restored"]
show walk["restored"] .
show env["NO_SUCH_VARIABLE_XYZ"] .
show length of args[] .
erase["archive"]
show exists["archive"] .
```

```text
true
false
[ "out/2026/report.txt" ]
[ "report.txt" ]
[ "archive/old.txt" ]
[ "restored/out/2026/report.txt" ]
none
0
false
```


**Example 287**

```turtle
import system
show sizeof["abc"] > 0 .
show freememory[] >= 0 .
```

```text
true
true
```


## The internet and databases

From the book, `book/src/03d-internet.md`.

**Example 288**

```turtle
import http
import json

body = http_get["http://127.0.0.1:8765/data"]
users = load[body]
show users json_get "name" .
```

```text
turtle: line 4: http_get http://127.0.0.1:8765/data: connection refused (is the server running?)
  4 | body = http_get["http://127.0.0.1:8765/data"]
```

This example stops with an error (exit code 1). It depends on something outside the example, or it shows an error on purpose.


**Example 289**

```turtle
import http

reply = http_post["https://example.com/api/users", map ["name": "Ann"]]
```

```text
turtle: line 3: http_post https://example.com/api/users: 405 Method Not Allowed: <!doctype html><html lang="en"><head><title>Example Domain</title><link rel="icon" href="data:,"><meta name="viewport" content="width=device-width, initial-scale=1"><style>body{background:#eee;width:6...
  3 | reply = http_post["https://example.com/api/users", map ["name": "Ann"]]
```

This example stops with an error (exit code 1). It depends on something outside the example, or it shows an error on purpose.


**Example 290**

```turtle
import server

def hello[]
    return reply[200, "{\"name\": \"Ann\"}", map ["Content-Type": "application/json"]]
def [end]

routes = map ["GET /data": hello]
serve[routes, 8765]
```

It runs until it is stopped, so it is not run here.


**Example 291**

```turtle
import json

user = json_read["user.json"]
json_write["copy.json", user]
```

```text
turtle: line 3: json_read user.json: no such file or folder
  3 | user = json_read["user.json"]
```

This example stops with an error (exit code 1). It depends on something outside the example, or it shows an error on purpose.


**Example 292**

```turtle
// Turn a nested map and list into JSON text.
import json
doc = map ["id": 7, "tags": list ["a", "b"], "meta": map ["ok": true]]
show json_text[doc] .
```

```text
{"id":7,"tags":["a","b"],"meta":{"ok":true}}
```


**Example 293**

```turtle
import sql

db = sql_create["shop.db"]
sql_run[db, "CREATE TABLE IF NOT EXISTS books (title TEXT, price INTEGER)", list []]
sql_run[db, "INSERT INTO books VALUES (?, ?)", list ["Emma", 900]]
cheap = sql_query[db, "SELECT title FROM books WHERE price < ?", list [1000]]
show cheap .
sql_close[db]
```

```text
[ { "title": "Emma" } ]
```


**Example 294**

```turtle
import sql

db = sql_open["shop.db"]
sql_upsert[db, "books", "title", "books_in.csv"]
sql_close[db]
```

```text
turtle: line 3: sql_open shop.db: no such file (sql_create makes a new database)
  3 | db = sql_open["shop.db"]
```

This example stops with an error (exit code 1). It depends on something outside the example, or it shows an error on purpose.


**Example 295**

```turtle
import json
show json_text[map ["x": none]] .
j = load['{"x": null}']
show j at get "x" .
```

```text
{"x":null}
none
```


**Example 296**

```turtle
import server

// A handler is a function that receives one request, called req.
// It gives back the answer for that request.

// For GET /: return a page of HTML text. The status is 200 by default.
def home[req]
    return "<h1>Hello!</h1>"
def [end]

// For GET /users/:id: the id is part of the path, such as 7 in /users/7.
// params of req is the set of path parts, and id of params gets the one named id.
def showuser[req]
    id = id of params of req
    return map ["id": id, "name": "Ann"]
def [end]

// For POST /users: json of req reads the JSON that the client sent.
// reply gives the status 201, which means "created", and the answer text.
def adduser[req]
    data = json of req
    return reply[201, map ["made": name of data]]
def [end]

// The map of routes. Each key is a method and a path, and each value answers it.
// A function is called for each request. A fixed value is sent as it is.
app = map [
    "GET /": home,
    "GET /users/:id": showuser,
    "GET /users/new": reply[200, "the new-user form"],   // a fixed answer
    "POST /users": adduser,
    "GET /old": redirect["/"],                            // sends the visitor to "/"
    "GET /health": "ok"                                   // a plain text answer
]

// Start the server on port 8093. The program keeps running until you press Ctrl+C.
serve[app, 8093]
```

It runs until it is stopped, so it is not run here.


**Example 297**

```turtle
import sql
import data
[write] incoming.csv
"title,price"
"Emma,950"
"New,300"
[end]
[write] gone.csv
"title"
"Ida"
[end]
db = sql_create["shop.db"]
sql_run[db, "CREATE TABLE books (title TEXT PRIMARY KEY, price INTEGER)", list []]
sql_run[db, "INSERT INTO books VALUES (?, ?)", list ["Emma", 900]]
sql_run[db, "INSERT INTO books VALUES (?, ?)", list ["Ida", 1500]]
show sql_tables[db] .
show sql_query[db, "SELECT title FROM books WHERE price < ?", list [1000]] .
sql_save[db, "SELECT title, price FROM books", "books.csv", list []]
show table[table_read["books.csv"]] .
sql_upsert[db, "books", "title", "incoming.csv"]
sql_delete[db, "books", "title", "gone.csv"]
show sql_query[db, "SELECT title, price FROM books ORDER BY title", list []] .
sql_close[db]
```

```text
[ "books" ]
[ { "title": "Emma" } ]
title  price
-----  -----
Emma     900
Ida     1500
[ { "title": "Emma", "price": 950 }, { "title": "New", "price": 300 } ]
```


## Time, security and development

From the book, `book/src/03e-time-dev.md`.

**Example 298**

```turtle
import time

now_day = today[]
due = now_day add_time 30, "days"
show now_day time_between due, "days" .
show format_date[due, "YYYY-MM-DD"] .
```

```text
30
2026-12-19
```


**Example 299**

```turtle
import time
every[1, "hours", check_inbox]
```

```text
turtle: line 2: undefined variable "check_inbox"
  2 | every[1, "hours", check_inbox]
```

This example stops with an error (exit code 1). It depends on something outside the example, or it shows an error on purpose.


**Example 300**

```turtle
import schedule

urls = list ["http://127.0.0.1:8765/data", "http://127.0.0.1:8765/data"]
results = fetchall[urls, map []]
show length of results .
```

```text
turtle: line 4: fetchall item 0: http_get http://127.0.0.1:8765/data: connection refused (is the server running?)
  4 | results = fetchall[urls, map []]
```

This example stops with an error (exit code 1). It depends on something outside the example, or it shows an error on purpose.


**Example 301**

```turtle
import crypt

stored = passwordhash["secret"]
show passwordcheck["secret", stored] .
show passwordcheck["wrong", stored] .
```

```text
true
false
```


**Example 302**

```turtle
import crypt

box = encrypt["secret", "pw"]
show decrypt[box, "pw"] .
```

```text
secret
```


**Example 303**

```turtle
import log

log info "started" .
log warn "disk at ", 91, "%" .
log error "bad thing" .
```

```text
2026-11-19 01:12:54 INFO  started
2026-11-19 01:12:54 WARN  disk at 91%
2026-11-19 01:12:54 ERROR bad thing
```


**Example 304**

```turtle
import log
loglevel = "debug"
log debug "details ", 3 .
```

```text
2026-11-19 01:12:54 DEBUG details 3
```


**Example 305**

```turtle
import log
name = "ann"
log info "login", map ["user": name] .
```

```text
2026-11-19 01:12:54 INFO  login user=ann
```


**Example 306**

```turtle
// Log messages at different levels.
import log
log info "starting job" .
log warn "disk is 90 percent full" .
log error "could not open file" .
show "done" .
```

```text
done
2026-11-19 01:12:54 INFO  starting job
2026-11-19 01:12:54 WARN  disk is 90 percent full
2026-11-19 01:12:54 ERROR could not open file
```


**Example 307**

```turtle
import time
d = make_date[2026, 12, 25, 9, 30]
show format_date[d, "DD/MM/YYYY hh:mm"] .
show year of d .
show month of d .
show day of d .
show hour of d .
show minute of d .
show weekday of d .
show zone of d .
```

```text
25/12/2026 09:30
2026
12
25
9
30
Friday
local
```


**Example 308**

```turtle
import time
e = to_date["2026-10-03 14:05"]
show format_date[e, "Weekday, Month D"] .
show to_zone[make_date[2026, 12, 25, 9, 30], "UTC"] .
show time_between[make_date[2026, 1, 1], make_date[2026, 1, 8], "days"] .
```

```text
Saturday, October 3
2026-12-25 14:30:00 UTC
7
```


**Example 309**

```turtle
import crypt
show hash["abc", "sha256"] .
show hash["abc", "sha1"] .
show hash["abc", "md5"] .
show encode["hi", "base64"] .
show decode[encode["hi", "base64"], "base64"] .
show encode["hi", "hex"] .
show decode["6869", "hex"] .
show length of token[4] .
show length of uuid[] .
```

```text
ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad
a9993e364706816aba3e25717850c26c9cd0d89d
900150983cd24fb0d6963f7d28e17f72
aGk=
hi
6869
hi
4
36
```


**Example 310**

```turtle
import log
logtime = none
loglevel = "warn"
log info "hidden" .
log error "shown" .
```

```text
ERROR shown
```


**Example 311**

```turtle
import log
logconsole = false
logfile = "app.log"
logformat = "json"
log info "started", map ["port": 8080] .
log warn "disk 90 percent full" .
[read] app.log to lines [end]
show length of lines .
```

```text
2
```


**Example 312**

```turtle
import schedule
schedulelimit = 2
skipschedule_error = true
pages = fetchall[list ["http://127.0.0.1:1/a", "http://127.0.0.1:1/b"]]
show pages .
```

```text
[ none, none ]
```


**Example 313**

```turtle
import log
logconsole = false
logfile = "run.log"
logparts = list ["level", "message"]
log warn "low disk" .
[read] run.log to lines [end]
show lines .
```

```text
[ "WARN  low disk" ]
```


**Example 314**

```turtle
import time
show year of today_utc[] > 2000 .
```

```text
true
```


## Tests and checks

From the book, `book/src/04-part3.md`.

**Example 315**

```turtle
import test
import data

def evens[nums]
    return keep[nums, x give x % 2 == 0]
def [end]

def test_evens_are_even[]
    check length of evens[list [1, 2, 4]] == 2 .
    verify evens[list [1, 2, 4]] each x give x % 2 == 0 .
def [end]

def test_sum_small[]
    check sum[list [1, 2]] == 3 .
def [end]
```

```text
(prints nothing)
```


**Example 316**

```turtle
import test
import data
nums = list [4, 6]
total = 45
check total == 45 .
check sum[nums] == 10 .
```

```text
(prints nothing)
```


**Example 317**

```turtle
import test
import data
nums = list [1, 2, 3]
verify nums each x give x < 3 .
```

```text
turtle: line 4: failed: verify nums each x give x < 3 .
    1 of 3 items breaks the rule:
      at 2: 3
  4 | verify nums each x give x < 3 .
```

This example stops with an error (exit code 1). It depends on something outside the example, or it shows an error on purpose.


**Example 318**

```turtle
import test
import data

def halve[n]
    return n div 2
def [end]

def test_halve_is_smaller[]
    validate halve[n] with n as integer that n >= 0 .
def [end]
```

```text
(prints nothing)
```


**Example 319**

```turtle
import test
import data
benchmark = true

def test_sum[]
    check sum[list [1, 2, 3]] == 6 .
def [end]
```

```text
(prints nothing)
```


**Example 320**

```turtle
theory tally
    abstract
        tally says how many times b appears in the list a.
    notation tally b in a .
    definition
        n = 0
        [loop][x in a]
            if ] x == b [
                n = n + 1
            if [end]
        [loop][end]
        return n
    theorem result >= 0
    theorem result <= length of a
    proof
        tally 1 in list [1, 1, 0, 3] . is 2
        tally 9 in list [] . is 0
theory [end]
nums = list [1, 1, 0, 3]
diagnose[tally 1 in nums]
```

```text
diagnose theory tally (line 20)
  notation  tally b in a .
  b         1
  a         list of 4: [ 1, 1, 0, 3 ]
  returned  2
  theorem   result >= 0            holds
  theorem   result <= length of a  holds
  took      38.5µs
  memory    100.5 KB allocated
```


**Example 321**

```turtle
nums = list [4, 8]
diagnose
    first = nums at get 0
    second = nums at get 5
diagnose [end]
show "after" .
```

```text
diagnose (lines 3-4)
  line 3      first = nums at get 0         first = 4
  line 4      second = nums at get 5
  ✗ failed: index error: line 4: index 5 out of range for list (length 2)
  memory    2.2 KB allocated
after
```


**Example 322**

```turtle
prices = list [5, -2, 9]
hypothesis[prices each p give p > 0]
show "still running" .
```

```text
hypothesis prices each p give p > 0 (line 2)
  holds     no: 1 of 3 items (33.3%) breaks it
  prices    list of 3: [ 5, -2, 9 ]
  where     at 1  -2   (p is -2, which isn't > 0)
  took      62.5µs
  memory    103.1 KB allocated
still running
```


**Example 323**

```turtle
nums = list [1, 2, 3]
hypothesis[nums each x give x > 0]
```

```text
hypothesis nums each x give x > 0 (line 2)
  holds     yes: all 3 items follow it
  nums      list of 3: [ 1, 2, 3 ]
  took      59.5µs
  memory    102.4 KB allocated
```


**Example 324**

```turtle
reportfile = "checks.txt"
nums = list [1, 2, 3]
hypothesis[nums each x give x > 0]
```

```text
hypothesis nums each x give x > 0 (line 3)
  holds     yes: all 3 items follow it
  nums      list of 3: [ 1, 2, 3 ]
  took      54.4µs
  memory    102.4 KB allocated
```


## Running, tracing and debugging

From the book, `book/src/05-part4.md`.

**Example 325**

```turtle
import system
show "args: ", args[] .
```

```text
args: []
```


**Example 326**

```turtle
import data
nums = list [5, 7]
total = 0
[loop][x in nums]
    total = total + x
[loop][end]
d = double[total]
show d .

def double[n]
    return n * 2
def [end]
```

```text
24
```


**Example 327**

```turtle
scores = list [4, 8, 6]
total = 0
count = 0
[loop][s in scores]
    total = total + s
[loop][end]
average = total / count
show average .
```

```text
turtle: line 7: division by zero
  7 | average = total / count
```

This example stops with an error (exit code 1). It depends on something outside the example, or it shows an error on purpose.


**Example 328**

```turtle
import data
nums = list [5, 7]
total = 0
[loop][x in nums]
    total = total + x
[loop][end]
d = double[total]
show d .

def double[n]
    return n * 2
def [end]
```

```text
24
```


**Example 329**

```turtle
def add_score[total, s]
    return total + s
def [end]
scores = list [4, 8, 6]
total = 0
count = 0
[loop][s in scores]
    total = add_score[total, s]
    count = count + 1
[loop][end]
average = total / count
show average .
```

```text
6.0
```


**Example 330**

```turtle
x   =    5
  def f[a]
return a
  def [end]
```

```text
(prints nothing)
```


**Example 331**

```turtle
x   =    5
def f[a]
    return a
def [end]
```

```text
(prints nothing)
```


**Example 332**

```turtle
// shop.turtle: helpers for the bookshop.

// add_tax adds rate percent to an amount, rounded to the cent.
// Gives back the new amount in cents.
def add_tax[cents, rate]
    return cents + cents * rate / 100
def [end]
```

```text
(prints nothing)
```


## 03-part2.md

From the book, `book/src/03-part2.md`.

**Example 333**

```turtle
import data
import sort
import search
import linear
import math
import random
import strings
import pattern
import system
import config
import http
import json
import server
import sql
import time
import schedule
import crypt
import test
import log
```

```text
(prints nothing)
```

