package object

import (
	"math"
	"strconv"
	"strings"
)

const MATRIX Type = "MATRIX"

// Matrix is linear's matrix [1, 2; 3, 4]: rows × columns numbers, kept
// row by row in one block of floats so the arithmetic runs fast. Ints is
// true while every number is whole and came from integers, so the matrix
// shows (and gives back) integers; anything that can make a fraction
// (/, inverse, solve) gives a float matrix, which shows 2.0, as a float
// does.
type Matrix struct {
	Rows, Cols int
	Data       []float64
	Ints       bool
}

// MaxExactInt is the largest whole number a float holds exactly: past it
// an integer matrix turns into a float one rather than lose digits quietly.
const MaxExactInt = 1 << 53

// NewMatrix is an all-zero integer matrix.
func NewMatrix(rows, cols int) *Matrix {
	return &Matrix{Rows: rows, Cols: cols, Data: make([]float64, rows*cols), Ints: true}
}

func (m *Matrix) Type() Type { return MATRIX }

// At is the number at row r, column c, from 0.
func (m *Matrix) At(r, c int) float64 { return m.Data[r*m.Cols+c] }

// Set puts v at row r, column c.
func (m *Matrix) Set(r, c int, v float64) { m.Data[r*m.Cols+c] = v }

// Value is the number at row r, column c as a Turtle value: an integer
// in an integer matrix, else a float.
func (m *Matrix) Value(r, c int) Object {
	return m.number(m.At(r, c))
}

func (m *Matrix) number(v float64) Object {
	if m.Ints {
		return Int(int64(v))
	}
	return &Float{Value: v}
}

// CheckInts keeps Ints true only while every number is whole and exact.
func (m *Matrix) CheckInts() {
	if !m.Ints {
		return
	}
	for _, v := range m.Data {
		if v != math.Trunc(v) || math.Abs(v) > MaxExactInt {
			m.Ints = false
			return
		}
	}
}

// Copy is a new matrix with the same numbers.
func (m *Matrix) Copy() *Matrix {
	return &Matrix{Rows: m.Rows, Cols: m.Cols, Data: append([]float64(nil), m.Data...), Ints: m.Ints}
}

// Leftover is how small, next to a matrix's largest number, a number is
// counted as float arithmetic's leftover rather than a value: inverse[a]
// times a gives 1s and 0s, give or take 1e-16. Such a number shows as 0.0
// and compares equal to 0, as 0.1 + 0.2 shows 0.3; get still gives it
// exactly.
const Leftover = 1e-13

// scale is the largest number's size, for Leftover.
func (m *Matrix) scale() float64 {
	big := 0.0
	for _, v := range m.Data {
		big = math.Max(big, math.Abs(v))
	}
	return big
}

// cell is how one number shows.
func (m *Matrix) cell(v, scale float64) string {
	if m.Ints {
		return strconv.FormatInt(int64(v), 10)
	}
	if math.Abs(v) < Leftover*scale {
		v = 0
	}
	return (&Float{Value: v}).Inspect()
}

// Most rows and columns show prints; the rest become "...".
const (
	MatrixShowRows = 20
	MatrixShowCols = 10
)

// Inspect is how show prints a matrix: one line per row, columns lined
// up, numbers on the right. A big matrix shows its first 20 rows and 10
// columns, "..." where the rest would be, and its size at the end.
func (m *Matrix) Inspect() string {
	if m.Rows == 0 || m.Cols == 0 {
		return "[ ]"
	}
	rows, cols := min(m.Rows, MatrixShowRows), min(m.Cols, MatrixShowCols)
	cutCols, cutRows := cols < m.Cols, rows < m.Rows
	// Each column lines up on its decimal points: the whole parts to the
	// right, the fractions to the left (980.0 over 7.125 is " 980.0" over
	// "   7.125").
	wholes := make([][]string, rows)
	fracs := make([][]string, rows)
	wholeW := make([]int, cols)
	fracW := make([]int, cols)
	scale := m.scale()
	for r := range rows {
		wholes[r] = make([]string, cols)
		fracs[r] = make([]string, cols)
		for c := range cols {
			s := m.cell(m.At(r, c), scale)
			whole, frac := s, ""
			if i := strings.IndexByte(s, '.'); i >= 0 {
				whole, frac = s[:i], s[i:]
			}
			wholes[r][c], fracs[r][c] = whole, frac
			wholeW[c] = max(wholeW[c], len(whole))
			fracW[c] = max(fracW[c], len(frac))
		}
	}
	var b strings.Builder
	for r := range rows {
		b.WriteString("[")
		for c := range cols {
			b.WriteString(" ")
			b.WriteString(strings.Repeat(" ", wholeW[c]-len(wholes[r][c])))
			b.WriteString(wholes[r][c])
			b.WriteString(fracs[r][c])
			b.WriteString(strings.Repeat(" ", fracW[c]-len(fracs[r][c])))
			if c < cols-1 {
				b.WriteString(" ")
			}
		}
		if cutCols {
			b.WriteString("  ...")
		}
		b.WriteString(" ]\n")
	}
	if cutRows {
		b.WriteString("  ...\n")
	}
	if cutRows || cutCols {
		b.WriteString("(" + m.SizeText() + ")\n")
	}
	return strings.TrimSuffix(b.String(), "\n")
}

// SizeText is "3 x 4 matrix", for show and for errors.
func (m *Matrix) SizeText() string {
	return strconv.Itoa(m.Rows) + " x " + strconv.Itoa(m.Cols) + " matrix"
}

// Line is the matrix on one line, as it shows inside a list or map:
// matrix [1, 2; 3, 4], which is also the code that makes it.
func (m *Matrix) Line() string {
	rows := make([]string, m.Rows)
	scale := m.scale()
	for r := range m.Rows {
		cells := make([]string, m.Cols)
		for c := range m.Cols {
			cells[c] = m.cell(m.At(r, c), scale)
		}
		rows[r] = strings.Join(cells, ", ")
	}
	return "matrix [" + strings.Join(rows, "; ") + "]"
}

// EqualMatrix is == for matrices: the same size, and every number equal
// as numbers compare (to 15 significant digits), or apart by no more than
// a leftover (see Leftover) of the larger matrix.
func EqualMatrix(a, b *Matrix) bool {
	if a.Rows != b.Rows || a.Cols != b.Cols {
		return false
	}
	tol := Leftover * math.Max(a.scale(), b.scale())
	for i, v := range a.Data {
		if CompareFloats(v, b.Data[i]) != 0 && math.Abs(v-b.Data[i]) > tol {
			return false
		}
	}
	return true
}
