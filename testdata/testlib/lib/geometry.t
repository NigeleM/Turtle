// lib/geometry.t: shapes, the library test_import.t tests. It's an
// ordinary Turtle file: no import test, nothing test-specific in it.
import math

assemble Rect [width, height]

def area[r]
    return width of r * height of r
def [end]

def perimeter[r]
    return 2 * width of r + 2 * height of r
def [end]

// circle_area is pi r squared, a float.
def circle_area[radius]
    return 3.14159 * radius * radius
def [end]

// grow makes every side n bigger: a new Rect, the old one unchanged.
def grow[r, n]
    return Rect[width of r + n, height of r + n]
def [end]

def scale_all[rects, n]
    out = list []
    [loop][r in rects]
        add Rect[width of r * n, height of r * n] to out .
    [loop][end]
    return out
def [end]
