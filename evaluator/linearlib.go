package evaluator

import (
	"strings"
	"unicode/utf8"

	"Turtle/ast"
	"Turtle/object"
)

// The "linear" builtin module: the matrix value (matrix [1, 2; 3, 4],
// made only in a file that imports linear), its operators (+, -, * as
// in mathematics: * of two matrices is the matrix product), its methods
// (m at get[0, 1]) and the library's functions (linalg.go does the
// arithmetic). Vectors are plain lists of numbers.

var linearFuncs = []string{
	"identity", "zeros", "ones", "diagonal", "shape", "row", "column",
	"flatten", "reshape", "transpose", "trace", "determinant", "inverse", "rank", "power",
	"multiply_each", "solve", "least_squares", "dot", "cross", "norm",
	"unit", "lu", "qr", "eigen", "svd",
}

// joinShown is text with v shown after it. A matrix shows over several
// lines: its rows after the first line up under the first, wherever on
// the line it starts ("m is [ 1  2 ]" then "      [ 3  4 ]").
func joinShown(text string, v object.Object) string {
	shown := v.Inspect()
	if _, ok := v.(*object.Matrix); ok && strings.Contains(shown, "\n") {
		col := utf8.RuneCountInString(text[strings.LastIndex(text, "\n")+1:])
		if col > 0 {
			shown = strings.ReplaceAll(shown, "\n", "\n"+strings.Repeat(" ", col))
		}
	}
	return text + shown
}

// evalMatrixLiteral works out each number of matrix [...].
func (it *Interpreter) evalMatrixLiteral(ml *ast.MatrixLiteral, env *object.Environment) object.Object {
	if len(ml.Rows) == 0 {
		return &object.Matrix{Ints: true}
	}
	m := object.NewMatrix(len(ml.Rows), len(ml.Rows[0]))
	for r, row := range ml.Rows {
		for c, e := range row {
			v := it.evalExpression(e, env)
			f, isInt, ok := numeric(v)
			if !ok {
				fatalKind(kindType, "a matrix holds numbers; row %d, column %d is %s", r, c, object.Shown(v))
			}
			if !isInt {
				m.Ints = false
			}
			m.Set(r, c, f)
		}
	}
	m.CheckInts()
	return m
}

// toMatrix is change ... to matrix: from a list of lists (each a row),
// or a list of maps or assembled values (what table_read and sql_query
// give: one row each, the columns in order).
func toMatrix(v object.Object) *object.Matrix {
	switch x := v.(type) {
	case *object.Matrix:
		return x.Copy()
	case *object.List:
		if len(x.Elements) == 0 {
			return &object.Matrix{Ints: true}
		}
		var rows [][]object.Object
		var header []string
		switch {
		case allLists(x.Elements):
			for _, e := range x.Elements {
				rows = append(rows, e.(*object.List).Elements)
			}
		case allRecords(x.Elements):
			header, rows = recordRows(x.Elements)
		default:
			fatalKind(kindType, "change ... to matrix needs a list of rows (lists, or maps as table_read gives); for one row, write list [nums]")
		}
		m := object.NewMatrix(len(rows), len(rows[0]))
		for r, row := range rows {
			if len(row) != m.Cols {
				fatalKind(kindLinear, "change ... to matrix: every row needs the same count of numbers; row 0 has %d, row %d has %d", m.Cols, r, len(row))
			}
			for c, cell := range row {
				f, isInt, ok := numeric(cell)
				if !ok {
					where := itoa(c)
					if header != nil {
						where = "\"" + header[c] + "\""
					}
					if cell == nil {
						cell = object.NoneValue
					}
					fatalKind(kindType, "change ... to matrix: row %d, column %s is %s, not a number%s", r, where, object.Shown(cell), textNumberHint(cell))
				}
				if !isInt {
					m.Ints = false
				}
				m.Set(r, c, f)
			}
		}
		m.CheckInts()
		return m
	}
	fatalKind(kindType, "change ... to matrix needs a list of rows, got %s", typeName(v))
	return nil
}

// matrixRows is change m to list: a list of rows, each a list.
func matrixRows(m *object.Matrix) *object.List {
	out := &object.List{Elements: make([]object.Object, m.Rows)}
	for r := range m.Rows {
		out.Elements[r] = matrixRowList(m, r)
	}
	return out
}

func matrixRowList(m *object.Matrix, r int) *object.List {
	row := &object.List{Elements: make([]object.Object, m.Cols)}
	for c := range m.Cols {
		row.Elements[c] = m.Value(r, c)
	}
	return row
}

func matrixColumnList(m *object.Matrix, c int) *object.List {
	col := &object.List{Elements: make([]object.Object, m.Rows)}
	for r := range m.Rows {
		col.Elements[r] = m.Value(r, c)
	}
	return col
}

// numbersList is a list of floats, or of integers when ints is true.
func numbersList(xs []float64, ints bool) *object.List {
	out := &object.List{Elements: make([]object.Object, len(xs))}
	for i, x := range xs {
		if ints {
			out.Elements[i] = object.Int(int64(x))
		} else {
			out.Elements[i] = &object.Float{Value: x}
		}
	}
	return out
}

// vector reads a list of numbers: their values, and whether all are
// integers.
func vector(fn string, v object.Object) ([]float64, bool) {
	l, ok := v.(*object.List)
	if !ok {
		fatalKind(kindType, "'%s' needs a list of numbers (a vector), got %s", fn, typeName(v))
	}
	out := make([]float64, len(l.Elements))
	ints := true
	for i, e := range l.Elements {
		f, isInt, ok := numeric(e)
		if !ok {
			fatalKind(kindType, "'%s' needs a list of numbers; item %d is %s", fn, i, object.Shown(e))
		}
		ints = ints && isInt
		out[i] = f
	}
	return out, ints
}

func floatMatrix(rows, cols int, data []float64) *object.Matrix {
	return &object.Matrix{Rows: rows, Cols: cols, Data: data}
}

// matrixInfix is an operator with a matrix on at least one side; ok is
// false when it isn't one (== and != compare as any values do).
func matrixInfix(op string, left, right object.Object) (object.Object, bool) {
	lm, lok := left.(*object.Matrix)
	rm, rok := right.(*object.Matrix)
	if !lok && !rok {
		return nil, false
	}
	switch op {
	case "==", "!=", "&&", "||":
		return nil, false
	case "+":
		if _, ok := left.(*object.String); ok {
			return nil, false // text + matrix joins, as text + anything does
		}
		if _, ok := right.(*object.String); ok {
			return nil, false
		}
		fallthrough
	case "-":
		if !lok || !rok {
			// A matrix and a number: the number goes on every element, as in
			// numpy and MATLAB (a + identity[n] is the diagonal only).
			m, other := lm, right
			if !lok {
				m, other = rm, left
			}
			f, isInt, ok := numeric(other)
			if !ok {
				fatalKind(kindLinear, "'%s' works on two matrices of the same size, or a matrix and a number, not a matrix and %s", op, typeName(other))
			}
			out := &object.Matrix{Rows: m.Rows, Cols: m.Cols, Data: make([]float64, len(m.Data)), Ints: m.Ints && isInt}
			for i, x := range m.Data {
				switch {
				case op == "+":
					out.Data[i] = x + f
				case lok: // m - n
					out.Data[i] = x - f
				default: // n - m
					out.Data[i] = f - x
				}
			}
			out.CheckInts()
			return out, true
		}
		if lm.Rows != rm.Rows || lm.Cols != rm.Cols {
			fatalKind(kindLinear, "can't %s a %s and a %s: they need the same size", map[string]string{"+": "add", "-": "subtract"}[op], lm.SizeText(), rm.SizeText())
		}
		out := &object.Matrix{Rows: lm.Rows, Cols: lm.Cols, Data: make([]float64, len(lm.Data)), Ints: lm.Ints && rm.Ints}
		for i, a := range lm.Data {
			if op == "+" {
				out.Data[i] = a + rm.Data[i]
			} else {
				out.Data[i] = a - rm.Data[i]
			}
		}
		out.CheckInts()
		return out, true
	case "*":
		switch {
		case lok && rok:
			if lm.Cols != rm.Rows {
				fatalKind(kindLinear, "can't multiply a %s by a %s: the first needs as many columns as the second has rows (for element by element, use multiply_each)", lm.SizeText(), rm.SizeText())
			}
			return matMul(lm, rm), true
		case lok:
			if l, ok := right.(*object.List); ok { // matrix × vector
				v, ints := vector("*", l)
				if len(v) != lm.Cols {
					fatalKind(kindLinear, "can't multiply a %s by a list of %d numbers: the list needs %d, one per column", lm.SizeText(), len(v), lm.Cols)
				}
				out := matMul(lm, &object.Matrix{Rows: len(v), Cols: 1, Data: v, Ints: ints})
				return numbersList(out.Data, out.Ints), true
			}
			return scaleMatrix(lm, right, "*"), true
		default:
			if l, ok := left.(*object.List); ok { // vector × matrix
				v, ints := vector("*", l)
				if len(v) != rm.Rows {
					fatalKind(kindLinear, "can't multiply a list of %d numbers by a %s: the list needs %d, one per row", len(v), rm.SizeText(), rm.Rows)
				}
				out := matMul(&object.Matrix{Rows: 1, Cols: len(v), Data: v, Ints: ints}, rm)
				return numbersList(out.Data, out.Ints), true
			}
			return scaleMatrix(rm, left, "*"), true
		}
	case "/":
		if lok && !rok {
			return scaleMatrix(lm, right, "/"), true
		}
		fatalKind(kindLinear, "a matrix can only be divided by a number; to undo a matrix product, multiply by its inverse (inverse[m]) or use solve[a, b]")
	}
	fatalKind(kindLinear, "operator %q doesn't work on a matrix (it has +, -, *, / by a number, == and !=)", op)
	return nil, false
}

// scaleMatrix multiplies (or divides) every number by n.
func scaleMatrix(m *object.Matrix, n object.Object, op string) *object.Matrix {
	f, isInt, ok := numeric(n)
	if !ok {
		fatalKind(kindLinear, "a matrix can be multiplied by a number, a list or another matrix, not %s", typeName(n))
	}
	if op == "/" && f == 0 {
		fatalKind(kindMath, "division by zero")
	}
	out := &object.Matrix{Rows: m.Rows, Cols: m.Cols, Data: make([]float64, len(m.Data)), Ints: m.Ints && isInt && op == "*"}
	for i, x := range m.Data {
		if op == "*" {
			out.Data[i] = x * f
		} else {
			out.Data[i] = x / f
		}
	}
	out.CheckInts()
	return out
}
