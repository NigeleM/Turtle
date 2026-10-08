package evaluator

import (
	"math"

	"Turtle/object"
)

// matrixMethod is m at name[args]: its size, and getting or putting its
// numbers. row and column give lists.
func matrixMethod(m *object.Matrix, name string, args []object.Object) object.Object {
	switch name {
	case "rows":
		requireArgs(name, args, 0)
		return object.Int(int64(m.Rows))
	case "columns":
		requireArgs(name, args, 0)
		return object.Int(int64(m.Cols))
	case "shape":
		requireArgs(name, args, 0)
		return matrixShape(m)
	case "get":
		requireArgs(name, args, 2)
		r, c := matrixIndex(m, args[0], args[1])
		return m.Value(r, c)
	case "put":
		requireArgs(name, args, 3)
		f, isInt, ok := numeric(args[0])
		if !ok {
			fatalKind(kindType, "a matrix holds numbers, not %s", object.Shown(args[0]))
		}
		r, c := matrixIndex(m, args[1], args[2])
		if !isInt || math.Abs(f) > object.MaxExactInt {
			m.Ints = false // only the new number can change that
		}
		m.Set(r, c, f)
		return m
	case "row":
		requireArgs(name, args, 1)
		return matrixRowList(m, rowIndex(m, args[0]))
	case "column":
		requireArgs(name, args, 1)
		return matrixColumnList(m, columnIndex(m, args[0]))
	case "isempty":
		requireArgs(name, args, 0)
		return object.Bool(m.Rows == 0 || m.Cols == 0)
	case "tostring":
		requireArgs(name, args, 0)
		return &object.String{Value: m.Inspect()}
	}
	fatalKind(kindName, "a matrix has no method %q (it has rows, columns, shape, get, put, row, column, isempty, tostring)", name)
	return nil
}

func matrixShape(m *object.Matrix) *object.List {
	return &object.List{Elements: []object.Object{object.Int(int64(m.Rows)), object.Int(int64(m.Cols))}}
}

func wholeNumber(what string, v object.Object) int {
	n, ok := v.(*object.Integer)
	if !ok {
		fatalKind(kindType, "%s needs a whole number, got %s", what, object.Shown(v))
	}
	return int(n.Value)
}

func rowIndex(m *object.Matrix, v object.Object) int {
	r := wholeNumber("a row number", v)
	if r < 0 || r >= m.Rows {
		fatalKind(kindIndex, "row %d is out of range for a %s (rows 0 to %d)", r, m.SizeText(), m.Rows-1)
	}
	return r
}

func columnIndex(m *object.Matrix, v object.Object) int {
	c := wholeNumber("a column number", v)
	if c < 0 || c >= m.Cols {
		fatalKind(kindIndex, "column %d is out of range for a %s (columns 0 to %d)", c, m.SizeText(), m.Cols-1)
	}
	return c
}

func matrixIndex(m *object.Matrix, r, c object.Object) (int, int) {
	return rowIndex(m, r), columnIndex(m, c)
}

func matrixArg(fn string, args []object.Object, i int) *object.Matrix {
	m, ok := args[i].(*object.Matrix)
	if !ok {
		fatalKind(kindType, "'%s' needs a matrix, got %s", fn, typeName(args[i]))
	}
	return m
}

func squareArg(fn string, args []object.Object, i int) *object.Matrix {
	m := matrixArg(fn, args, i)
	if m.Rows != m.Cols {
		fatalKind(kindLinear, "'%s' needs a square matrix, got a %s", fn, m.SizeText())
	}
	return m
}

func sizeArg(fn string, v object.Object) int {
	n := wholeNumber("'"+fn+"'", v)
	if n < 0 {
		fatalKind(kindLinear, "'%s' needs a size of 0 or more, got %d", fn, n)
	}
	return n
}

// singularError is the one message for a matrix with no inverse.
func singularError(fn string) {
	fatalKind(kindLinear, "'%s': the matrix is singular (its determinant is 0: some row is a mix of the others), so it has no inverse and no single answer", fn)
}

// callLinear runs one of the linear library's functions.
func callLinear(name string, args []object.Object) object.Object {
	switch name {
	case "identity":
		requireFuncArgs(name, args, 1)
		n := sizeArg(name, args[0])
		m := object.NewMatrix(n, n)
		for i := range n {
			m.Set(i, i, 1)
		}
		return m
	case "zeros", "ones":
		if len(args) != 1 && len(args) != 2 {
			fatalf("function %q expects 1 or 2 arguments (rows, and columns if they differ), got %d", name, len(args))
		}
		r := sizeArg(name, args[0])
		c := r
		if len(args) == 2 {
			c = sizeArg(name, args[1])
		}
		m := object.NewMatrix(r, c)
		if name == "ones" {
			for i := range m.Data {
				m.Data[i] = 1
			}
		}
		return m
	case "diagonal":
		requireFuncArgs(name, args, 1)
		if m, ok := args[0].(*object.Matrix); ok { // the numbers on its diagonal
			d := make([]float64, min(m.Rows, m.Cols))
			for i := range d {
				d[i] = m.At(i, i)
			}
			return numbersList(d, m.Ints)
		}
		v, ints := vector(name, args[0])
		m := object.NewMatrix(len(v), len(v))
		m.Ints = ints
		for i, x := range v {
			m.Set(i, i, x)
		}
		return m
	case "shape":
		requireFuncArgs(name, args, 1)
		return matrixShape(matrixArg(name, args, 0))
	case "row", "column":
		requireFuncArgs(name, args, 2)
		return matrixMethod(matrixArg(name, args, 0), name, args[1:])
	case "transpose":
		requireFuncArgs(name, args, 1)
		return transposed(matrixArg(name, args, 0))
	case "trace":
		requireFuncArgs(name, args, 1)
		m := squareArg(name, args, 0)
		t := 0.0
		for i := range m.Rows {
			t += m.At(i, i)
		}
		return numberOf(t, m.Ints)
	case "determinant":
		requireFuncArgs(name, args, 1)
		m := squareArg(name, args, 0)
		d := determinant(m)
		if m.Ints && math.Abs(d) < object.MaxExactInt {
			return object.Int(int64(math.Round(d))) // whole numbers in, a whole number out
		}
		return &object.Float{Value: d}
	case "inverse":
		requireFuncArgs(name, args, 1)
		m := squareArg(name, args, 0)
		lu := luDecompose(m)
		if lu.singular {
			singularError(name)
		}
		n := m.Rows
		id := make([]float64, n*n)
		for i := range n {
			id[i*n+i] = 1
		}
		return floatMatrix(n, n, lu.solve(id, n))
	case "rank":
		requireFuncArgs(name, args, 1)
		m := matrixArg(name, args, 0)
		if len(m.Data) == 0 {
			return object.Int(0)
		}
		_, s, _ := svd(m)
		tol := s[0] * float64(max(m.Rows, m.Cols)) * 2.220446049250313e-16
		rank := 0
		for _, x := range s {
			if x > tol {
				rank++
			}
		}
		return object.Int(int64(rank))
	case "power":
		requireFuncArgs(name, args, 2)
		m := squareArg(name, args, 0)
		k := wholeNumber("'power'", args[1])
		if k < 0 { // a negative power is the inverse's
			m = callLinear("inverse", args[:1]).(*object.Matrix)
			k = -k
		}
		out := callLinear("identity", []object.Object{object.Int(int64(m.Rows))}).(*object.Matrix)
		for base := m; k > 0; k >>= 1 { // squaring: about log2(k) products
			if k&1 == 1 {
				out = matMul(out, base)
			}
			if k > 1 {
				base = matMul(base, base)
			}
		}
		return out
	case "multiply_each":
		requireFuncArgs(name, args, 2)
		a, b := matrixArg(name, args, 0), matrixArg(name, args, 1)
		if a.Rows != b.Rows || a.Cols != b.Cols {
			fatalKind(kindLinear, "'multiply_each' needs two matrices of the same size, got a %s and a %s", a.SizeText(), b.SizeText())
		}
		out := &object.Matrix{Rows: a.Rows, Cols: a.Cols, Data: make([]float64, len(a.Data)), Ints: a.Ints && b.Ints}
		for i := range a.Data {
			out.Data[i] = a.Data[i] * b.Data[i]
		}
		out.CheckInts()
		return out
	case "solve":
		requireFuncArgs(name, args, 2)
		a := squareArg(name, args, 0)
		b, list := rightSide(name, a, args[1])
		lu := luDecompose(a)
		if lu.singular {
			singularError(name)
		}
		x := floatMatrix(a.Rows, b.Cols, lu.solve(b.Data, b.Cols))
		ax := matMul(a, x) // one round of refinement, as least_squares
		resid := make([]float64, len(b.Data))
		for i := range resid {
			resid[i] = b.Data[i] - ax.Data[i]
		}
		for i, f := range lu.solve(resid, b.Cols) {
			x.Data[i] += f
		}
		return answer(x, list)
	case "least_squares":
		requireFuncArgs(name, args, 2)
		a := matrixArg(name, args, 0)
		if a.Rows < a.Cols {
			fatalKind(kindLinear, "'least_squares' needs at least as many rows (equations) as columns (unknowns), got a %s", a.SizeText())
		}
		b, list := rightSide(name, a, args[1])
		q, r := householderQR(a)
		tol := tolerance(r)
		for i := range r.Rows {
			if math.Abs(r.At(i, i)) <= tol {
				fatalKind(kindLinear, "'least_squares': the columns depend on each other (column %d is a mix of the ones before it), so there's no single best answer", i)
			}
		}
		x := qrSolve(q, r, b)
		// One round of refinement: solve again for what's left over, and
		// add it, which wins back the last digits lost to rounding.
		resid := floatMatrix(b.Rows, b.Cols, make([]float64, len(b.Data)))
		ax := matMul(a, x)
		for i := range resid.Data {
			resid.Data[i] = b.Data[i] - ax.Data[i]
		}
		fix := qrSolve(q, r, resid)
		for i := range x.Data {
			x.Data[i] += fix.Data[i]
		}
		return answer(x, list)
	case "dot":
		requireFuncArgs(name, args, 2)
		u, ui := vector(name, args[0])
		v, vi := vector(name, args[1])
		if len(u) != len(v) {
			fatalKind(kindLinear, "'dot' needs two lists of the same length, got %d and %d", len(u), len(v))
		}
		s := 0.0
		for i := range u {
			s += u[i] * v[i]
		}
		return numberOf(s, ui && vi)
	case "cross":
		requireFuncArgs(name, args, 2)
		u, ui := vector(name, args[0])
		v, vi := vector(name, args[1])
		if len(u) != 3 || len(v) != 3 {
			fatalKind(kindLinear, "'cross' works on two lists of 3 numbers (3-D vectors), got %d and %d", len(u), len(v))
		}
		return numbersList([]float64{u[1]*v[2] - u[2]*v[1], u[2]*v[0] - u[0]*v[2], u[0]*v[1] - u[1]*v[0]}, ui && vi)
	case "norm":
		requireFuncArgs(name, args, 1)
		var xs []float64
		if m, ok := args[0].(*object.Matrix); ok {
			xs = m.Data // Frobenius: every number, squared, added, rooted
		} else {
			xs, _ = vector(name, args[0])
		}
		return &object.Float{Value: euclid(xs)}
	case "unit":
		requireFuncArgs(name, args, 1)
		v, _ := vector(name, args[0])
		n := euclid(v)
		if n == 0 {
			fatalKind(kindLinear, "'unit' of a vector of zeros: it has no direction")
		}
		out := make([]float64, len(v))
		for i, x := range v {
			out[i] = x / n
		}
		return numbersList(out, false)
	case "lu":
		requireFuncArgs(name, args, 1)
		m := squareArg(name, args, 0)
		res := luDecompose(m)
		if res.singular {
			singularError(name)
		}
		n := m.Rows
		l, u, p := object.NewMatrix(n, n), object.NewMatrix(n, n), object.NewMatrix(n, n)
		l.Ints, u.Ints = false, false
		for i := range n {
			p.Set(i, res.piv[i], 1)
			for j := range n {
				switch {
				case j < i:
					l.Set(i, j, res.lu[i*n+j])
				case j == i:
					l.Set(i, j, 1)
					u.Set(i, j, res.lu[i*n+j])
				default:
					u.Set(i, j, res.lu[i*n+j])
				}
			}
		}
		return resultMap("l", l, "u", u, "p", p)
	case "qr":
		requireFuncArgs(name, args, 1)
		m := matrixArg(name, args, 0)
		if m.Rows < m.Cols {
			fatalKind(kindLinear, "'qr' needs at least as many rows as columns, got a %s (try qr[transpose[m]])", m.SizeText())
		}
		q, r := householderQR(m)
		return resultMap("q", q, "r", r)
	case "eigen":
		requireFuncArgs(name, args, 1)
		m := squareArg(name, args, 0)
		tol := tolerance(m) * 1e3
		for i := range m.Rows {
			for j := i + 1; j < m.Cols; j++ {
				if math.Abs(m.At(i, j)-m.At(j, i)) > tol {
					fatalKind(kindLinear, "'eigen' works on symmetric matrices (the same across the diagonal), and this one isn't: row %d, column %d differs from row %d, column %d", i, j, j, i)
				}
			}
		}
		values, vectors, ok := symmetricEigen(m)
		if !ok {
			fatalKind(kindLinear, "'eigen' didn't settle on an answer for this matrix")
		}
		return resultMap("values", numbersList(values, false), "vectors", vectors)
	case "svd":
		requireFuncArgs(name, args, 1)
		u, s, v := svd(matrixArg(name, args, 0))
		return resultMap("u", u, "s", numbersList(s, false), "v", v)
	}
	fatalKind(kindName, "no builtin function %q in \"linear\"", name)
	return nil
}

// qrSolve is the least-squares x for q r x = b: R x = Qᵀ b, from the
// bottom row up.
func qrSolve(q, r, b *object.Matrix) *object.Matrix {
	qtb := matMul(transposed(q), b)
	n := r.Rows
	x := floatMatrix(n, b.Cols, make([]float64, n*b.Cols))
	for c := range b.Cols {
		for i := n - 1; i >= 0; i-- {
			s := qtb.At(i, c)
			for j := i + 1; j < n; j++ {
				s -= r.At(i, j) * x.At(j, c)
			}
			x.Set(i, c, s/r.At(i, i))
		}
	}
	return x
}

// svd works for any shape: a wide matrix is done as its transpose.
func svd(m *object.Matrix) (u *object.Matrix, s []float64, v *object.Matrix) {
	if m.Rows >= m.Cols {
		return svdJacobi(m)
	}
	vt, s, ut := svdJacobi(transposed(m))
	return ut, s, vt
}

// rightSide reads solve's b: a list (one answer, a list) or a matrix
// (one answer per column, a matrix), with as many numbers as a has rows.
func rightSide(fn string, a *object.Matrix, b object.Object) (*object.Matrix, bool) {
	if bm, ok := b.(*object.Matrix); ok {
		if bm.Rows != a.Rows {
			fatalKind(kindLinear, "'%s': the right side needs %d rows, one per equation, got a %s", fn, a.Rows, bm.SizeText())
		}
		return bm, false
	}
	v, _ := vector(fn, b)
	if len(v) != a.Rows {
		fatalKind(kindLinear, "'%s': the right side needs %d numbers, one per row of the matrix, got %d", fn, a.Rows, len(v))
	}
	return floatMatrix(len(v), 1, v), true
}

func answer(x *object.Matrix, list bool) object.Object {
	if list {
		return numbersList(x.Data, false)
	}
	return x
}

func numberOf(f float64, ints bool) object.Object {
	if ints && math.Abs(f) < object.MaxExactInt {
		return object.Int(int64(f))
	}
	return &object.Float{Value: f}
}

func euclid(xs []float64) float64 {
	n := 0.0
	for _, x := range xs {
		n = math.Hypot(n, x)
	}
	return n
}

// resultMap is a decomposition's parts, by name.
func resultMap(kv ...any) *object.Map {
	m := object.NewMap()
	for i := 0; i < len(kv); i += 2 {
		m.Put(&object.String{Value: kv[i].(string)}, kv[i+1].(object.Object))
	}
	return m
}
