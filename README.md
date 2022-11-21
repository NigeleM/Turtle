# Turtle Is Under Update and rewriting
Turtle Programming Language
# Where I am Currently
1. Re-writing interpreter in go
2. Output establsihed
3. Variable and non variable expressions can be evaluated
4. Display different data to the screen.



# The Turtle Programming Language
Turtle was my side project during community college and university. 
It is copyrighted by me. My goal was to make a very simple language
that could be made a universal language and shift some of the coding paradigms
used in the industry. Being that i had limited knowledge on how to write an interpreter 
and compiler it has taken a while to complete. I was able to actually write an interpreter in python 
but it wasn't independent of python. I ended up re-writing it in C, but i may have to start writing it in C#.
I had attempted to re-write in Java but it would be dependent on Java. I also wanted it to be very fast so C was a better option.
My knowledge of C is limited as well, when it comes to memory allocation. My language has some of the basics but needs some help in
memory allocation to help with developing objects and recursive functions.

# Features
1. variables assignments
2. variable arithametic 
3. string concatenation
4. built in functions
5. output function
6. decision structures (if else )
7. develop personal functions 
8. Has interactive mode (REPL) like PYTHON

# Last update
I was developing a package for it.
I had developed a successful package that could run when you click a *.t file.
the timestamp file was able to determine was file had been clicked and then tell the interpreter to run that file.
Package was for MacOs, had not tested a windows version of it. 

# Interpreter 
1. The interpreter is the file Interpreter.C 
2. Compile interpreter using clang
3. pass in turtle file to run program/script example in commandline ./turtle file.t
4. pass in no file to execute interactive mode ./turtle
