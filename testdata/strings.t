s = "  Hello, World  "

t is s at trim .
show t .

u is t at upper .
show u .

l is t at lower .
show l .

length of t .

c is t at get 0 .
show c .

last is t at get 4 .
show last .

first5 is t at slice 0, 5 .
show first5 .

last5 is t at slice -5 .
show last5 .

tail is t at slice 7 .
show tail .

whole is t at slice 0, 999 .
show whole .

parts is t at split ", " .
show parts .

hasWorld is t at contains "World" .
show hasWorld .
hasZ is t at contains "Z" .
show hasZ .

idx is t at indexOf "World" .
show idx .
idxMissing is t at indexOf "nope" .
show idxMissing .

r is t at replace "World", "Turtle" .
show r .

// unicode sanity check: rune-based, not byte-based
uni = "café"
length of uni .
lastChar is uni at get 3 .
show lastChar .
