


  // b = 100
  //delta = 0
  //a = list [3,2,1]
  //b = map ["Nigele McCoy" : 2, "Khris":3, "Mom":4]
  //show b .
  //z is max of a 
  //show "max of a is " , z .
  //z is min of a 
  //show "min of a is ", z .
  //c = "Nigele McCoy"
  //delete c from b 
  //show b . 
  //show a .
  //insert 4 to a at delta
  //show a .


def back[]
  show "In back function" .
  bd = map ["Nigele" : 2, "Khris":3, "Mom":4]
  show bd .
  a = "Khris"
   delete "Khris" from bd 
  show bd . 
def [end]

 def run[]
 delta = 0
 show "run" .
  a = list [1,2,3,"Hello World",1]
  show a . 
  b = "good day" 
  insert b to a at delta
 // c = set [1,2,3,4,4,5]
  show a .
 // show " Set C Below " . 
 // show c .
 // b = map ["Nigele" : 2, "Khris":3, "Mom":4]
 // show b , " my map " , c , a , " ", 1+1 .
 // d = []
 // show d .
 //  show a . 
  def [end]

//run[]

//back[]

Bd = map ["Nigele" : 2, "Khris":3, "Mom":4]
C = Bd
// show "C is " , C .
A is Bd at len

// show "A is ",  A . 
// show "Bd is ",  Bd . 
a = list [3,2,1]
c = 10
b is a at add c .
b is b at add c .
// show b .
// show a .
d is b at index c .
// show d+1 .

// C = "Dad"
// Bd is Bd at add "Dad baddy" , 1 .
// show Bd. 
// abb is Bd at get C .
// show abb . 

b = set [1,2,3,4,5]
c = set [1]

// a is b at superset c .
// show a .
// a is b at subset c .
// show a .

Bd = map ["Nigele" : "e", "Khris":"c", "Mom":"d"]
// dvd is Bd at invert .
// show dvd .

show "Get keys " .
d is Bd at getKeys .
show d .
show "Get Values " .
d is Bd at getValues .
show d .

import data 
hello[]

def day[]
  show "outer function" .
  hello[]
  show "works inside of function" .  
def [end]

day[]

letter = list ["Nigele"]

show letter .
show "help" .

 