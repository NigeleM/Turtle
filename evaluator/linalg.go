// Copyright 2017-2026 Nigele McCoy
// Licensed under the Apache License, Version 2.0;
// see LICENSE and NOTICE.

package evaluator

import (
	"math"
	"runtime"
	"sort"
	"sync"

	"Turtle/object"
)

// The arithmetic behind the linear library, on plain floats: elimination
// with partial pivoting (determinant, inverse, solve, lu), Householder
// reflections (qr, least squares), and Jacobi rotations (eigen, svd),
// chosen because each is short, well understood and accurate for the
// small and medium matrices a script works with.

// tolerance is how small a pivot (or singular value) may be before a
// matrix counts as singular: relative to its largest number, so scaling
// a matrix doesn't change the answer.
func tolerance(m *object.Matrix) float64 {
	big := 0.0
	for _, v := range m.Data {
		big = math.Max(big, math.Abs(v))
	}
	return 1e-12 * big * float64(max(m.Rows, m.Cols))
}

// luResult is PA = LU: piv[i] is the row of A that ended up in row i,
// lu holds L below the diagonal (its own diagonal is 1s) and U on and
// above it, and swaps counts the row swaps (for the determinant's sign).
type luResult struct {
	n        int
	lu       []float64
	piv      []int
	swaps    int
	singular bool
}

func luDecompose(m *object.Matrix) luResult {
	n := m.Rows
	a := append([]float64(nil), m.Data...)
	piv := make([]int, n)
	for i := range piv {
		piv[i] = i
	}
	tol := tolerance(m)
	res := luResult{n: n, lu: a, piv: piv}
	for k := range n {
		p := k
		for i := k + 1; i < n; i++ {
			if math.Abs(a[i*n+k]) > math.Abs(a[p*n+k]) {
				p = i
			}
		}
		if math.Abs(a[p*n+k]) <= tol {
			res.singular = true
			continue
		}
		if p != k {
			for j := range n {
				a[k*n+j], a[p*n+j] = a[p*n+j], a[k*n+j]
			}
			piv[k], piv[p] = piv[p], piv[k]
			res.swaps++
		}
		for i := k + 1; i < n; i++ {
			f := a[i*n+k] / a[k*n+k]
			a[i*n+k] = f
			for j := k + 1; j < n; j++ {
				a[i*n+j] -= f * a[k*n+j]
			}
		}
	}
	return res
}

// determinant is the product of U's diagonal, its sign flipped once per
// row swap.
func determinant(m *object.Matrix) float64 {
	if m.Rows == 0 {
		return 1
	}
	r := luDecompose(m)
	if r.singular {
		return 0
	}
	d := 1.0
	if r.swaps%2 == 1 {
		d = -1
	}
	for i := range r.n {
		d *= r.lu[i*r.n+i]
	}
	return d
}

// luSolve solves A x = b for each column of b (n × k, row by row).
func (r luResult) solve(b []float64, k int) []float64 {
	n := r.n
	x := make([]float64, n*k)
	for i := range n {
		copy(x[i*k:(i+1)*k], b[r.piv[i]*k:(r.piv[i]+1)*k])
	}
	// Whole rows at a time, so every column of b is done in one pass
	// that reads memory straight through.
	for i := range n { // L y = P b
		xi := x[i*k : (i+1)*k]
		for j := range i {
			if f := r.lu[i*n+j]; f != 0 {
				for c, v := range x[j*k : (j+1)*k] {
					xi[c] -= f * v
				}
			}
		}
	}
	for i := n - 1; i >= 0; i-- { // U x = y
		xi := x[i*k : (i+1)*k]
		for j := i + 1; j < n; j++ {
			if f := r.lu[i*n+j]; f != 0 {
				for c, v := range x[j*k : (j+1)*k] {
					xi[c] -= f * v
				}
			}
		}
		d := r.lu[i*n+i]
		for c := range xi {
			xi[c] /= d
		}
	}
	return x
}

// householderQR factors an m × n matrix (m >= n) as Q R, Q m × n with
// orthonormal columns and R n × n upper triangular.
func householderQR(a *object.Matrix) (q, r *object.Matrix) {
	m, n := a.Rows, a.Cols
	w := append([]float64(nil), a.Data...)
	vs := make([][]float64, n) // the reflections, to build Q afterwards
	for k := range n {
		norm := 0.0
		for i := k; i < m; i++ {
			norm = math.Hypot(norm, w[i*n+k])
		}
		if norm == 0 {
			continue
		}
		alpha := -norm
		if w[k*n+k] < 0 {
			alpha = norm
		}
		v := make([]float64, m)
		for i := k; i < m; i++ {
			v[i] = w[i*n+k]
		}
		v[k] -= alpha
		vn := 0.0
		for i := k; i < m; i++ {
			vn += v[i] * v[i]
		}
		if vn == 0 {
			continue
		}
		for j := k; j < n; j++ {
			s := 0.0
			for i := k; i < m; i++ {
				s += v[i] * w[i*n+j]
			}
			s = 2 * s / vn
			for i := k; i < m; i++ {
				w[i*n+j] -= s * v[i]
			}
		}
		vs[k] = v
	}
	r = &object.Matrix{Rows: n, Cols: n, Data: make([]float64, n*n)}
	for i := range n {
		for j := i; j < n; j++ {
			r.Data[i*n+j] = w[i*n+j]
		}
	}
	// Q is the reflections applied, last first, to the first n columns
	// of the identity.
	q = &object.Matrix{Rows: m, Cols: n, Data: make([]float64, m*n)}
	for i := range n {
		q.Data[i*n+i] = 1
	}
	for k := n - 1; k >= 0; k-- {
		v := vs[k]
		if v == nil {
			continue
		}
		vn := 0.0
		for i := k; i < m; i++ {
			vn += v[i] * v[i]
		}
		for j := range n {
			s := 0.0
			for i := k; i < m; i++ {
				s += v[i] * q.Data[i*n+j]
			}
			s = 2 * s / vn
			for i := k; i < m; i++ {
				q.Data[i*n+j] -= s * v[i]
			}
		}
	}
	// R's diagonal made positive (flipping Q's matching column), so the
	// answer is the one usually given.
	for i := range n {
		if r.Data[i*n+i] < 0 {
			for j := i; j < n; j++ {
				r.Data[i*n+j] = -r.Data[i*n+j]
			}
			for k := range m {
				q.Data[k*n+i] = -q.Data[k*n+i]
			}
		}
	}
	return q, r
}

// symmetricEigen finds a symmetric matrix's eigenvalues and eigenvectors:
// Householder reduction to tridiagonal form, then the implicit QL method
// (tred2 and tql2 from EISPACK, as the public-domain JAMA library gives
// them), about 9n³ steps in all. Values come largest first; vectors are
// the columns of the matrix, each of length 1, its first nonzero number
// positive.
func symmetricEigen(m *object.Matrix) ([]float64, *object.Matrix, bool) {
	n := m.Rows
	v := make([][]float64, n)
	for i := range n {
		v[i] = append([]float64(nil), m.Data[i*n:(i+1)*n]...)
	}
	d, e := make([]float64, n), make([]float64, n)
	if n > 0 {
		tred2(v, d, e)
		if !tql2(v, d, e) {
			return nil, nil, false
		}
	}
	order := make([]int, n)
	for i := range order {
		order[i] = i
	}
	sort.SliceStable(order, func(x, y int) bool { return d[order[x]] > d[order[y]] })
	values := make([]float64, n)
	vecs := &object.Matrix{Rows: n, Cols: n, Data: make([]float64, n*n)}
	for c, i := range order {
		values[c] = d[i]
		for k := range n {
			vecs.Data[k*n+c] = v[k][i]
		}
	}
	fixSigns(vecs)
	return values, vecs, true
}

// tred2 reduces the symmetric v to tridiagonal form (diagonal d,
// off-diagonal e), leaving in v the orthogonal transformation.
func tred2(v [][]float64, d, e []float64) {
	n := len(d)
	for j := range n {
		d[j] = v[n-1][j]
	}
	for i := n - 1; i > 0; i-- {
		scale, h := 0.0, 0.0
		for k := range i {
			scale += math.Abs(d[k])
		}
		if scale == 0 {
			e[i] = d[i-1]
			for j := range i {
				d[j] = v[i-1][j]
				v[i][j] = 0
				v[j][i] = 0
			}
		} else {
			for k := range i {
				d[k] /= scale
				h += d[k] * d[k]
			}
			f := d[i-1]
			g := math.Sqrt(h)
			if f > 0 {
				g = -g
			}
			e[i] = scale * g
			h -= f * g
			d[i-1] = f - g
			for j := range i {
				e[j] = 0
			}
			for j := range i {
				f = d[j]
				v[j][i] = f
				g = e[j] + v[j][j]*f
				for k := j + 1; k <= i-1; k++ {
					g += v[k][j] * d[k]
					e[k] += v[k][j] * f
				}
				e[j] = g
			}
			f = 0
			for j := range i {
				e[j] /= h
				f += e[j] * d[j]
			}
			hh := f / (h + h)
			for j := range i {
				e[j] -= hh * d[j]
			}
			for j := range i {
				f, g = d[j], e[j]
				for k := j; k <= i-1; k++ {
					v[k][j] -= f*e[k] + g*d[k]
				}
				d[j] = v[i-1][j]
				v[i][j] = 0
			}
		}
		d[i] = h
	}
	for i := 0; i < n-1; i++ {
		v[n-1][i] = v[i][i]
		v[i][i] = 1
		h := d[i+1]
		if h != 0 {
			for k := 0; k <= i; k++ {
				d[k] = v[k][i+1] / h
			}
			for j := 0; j <= i; j++ {
				g := 0.0
				for k := 0; k <= i; k++ {
					g += v[k][i+1] * v[k][j]
				}
				for k := 0; k <= i; k++ {
					v[k][j] -= g * d[k]
				}
			}
		}
		for k := 0; k <= i; k++ {
			v[k][i+1] = 0
		}
	}
	for j := range n {
		d[j] = v[n-1][j]
		v[n-1][j] = 0
	}
	v[n-1][n-1] = 1
	e[0] = 0
}

// tql2 finds the tridiagonal matrix's eigenvalues (into d) by implicit
// QL steps, turning v's columns into the eigenvectors. false if it
// doesn't settle, which a symmetric matrix of real numbers never does.
func tql2(v [][]float64, d, e []float64) bool {
	n := len(d)
	for i := 1; i < n; i++ {
		e[i-1] = e[i]
	}
	e[n-1] = 0
	f, tst1 := 0.0, 0.0
	const eps = 2.220446049250313e-16
	for l := range n {
		tst1 = math.Max(tst1, math.Abs(d[l])+math.Abs(e[l]))
		m := l
		for m < n && math.Abs(e[m]) > eps*tst1 {
			m++
		}
		if m == n {
			m = n - 1
		}
		if m > l {
			for iter := 0; ; iter++ {
				if iter > 100 {
					return false
				}
				g := d[l]
				p := (d[l+1] - g) / (2 * e[l])
				r := math.Hypot(p, 1)
				if p < 0 {
					r = -r
				}
				d[l] = e[l] / (p + r)
				d[l+1] = e[l] * (p + r)
				dl1 := d[l+1]
				h := g - d[l]
				for i := l + 2; i < n; i++ {
					d[i] -= h
				}
				f += h
				p = d[m]
				c, c2, c3 := 1.0, 1.0, 1.0
				el1 := e[l+1]
				s, s2 := 0.0, 0.0
				for i := m - 1; i >= l; i-- {
					c3 = c2
					c2 = c
					s2 = s
					g = c * e[i]
					h = c * p
					r = math.Hypot(p, e[i])
					e[i+1] = s * r
					s = e[i] / r
					c = p / r
					p = c*d[i] - s*g
					d[i+1] = h + s*(c*g+s*d[i])
					for k := range n {
						h = v[k][i+1]
						v[k][i+1] = s*v[k][i] + c*h
						v[k][i] = c*v[k][i] - s*h
					}
				}
				p = -s * s2 * c3 * el1 * e[l] / dl1
				e[l] = s * p
				d[l] = c * p
				if math.Abs(e[l]) <= eps*tst1 {
					break
				}
			}
		}
		d[l] += f
		e[l] = 0
	}
	return true
}

// fixSigns makes each column's first clearly nonzero number positive, so
// the same matrix always gives the same vectors.
func fixSigns(m *object.Matrix) {
	for c := range m.Cols {
		for r := range m.Rows {
			v := m.At(r, c)
			if math.Abs(v) > 1e-12 {
				if v < 0 {
					for k := range m.Rows {
						m.Set(k, c, -m.At(k, c))
					}
				}
				break
			}
		}
	}
}

// svdJacobi is the singular value decomposition A = U S Vᵀ by one-sided
// Jacobi rotations (Hestenes), for m >= n: U m × n, s (n, largest
// first), V n × n. The columns are kept one after another in memory, so
// each rotation reads them straight through.
func svdJacobi(a *object.Matrix) (u *object.Matrix, s []float64, v *object.Matrix) {
	m, n := a.Rows, a.Cols
	w := make([][]float64, n) // A's columns, rotated until at right angles
	vd := make([][]float64, n)
	for j := range n {
		w[j] = make([]float64, m)
		for i := range m {
			w[j][i] = a.Data[i*n+j]
		}
		vd[j] = make([]float64, n)
		vd[j][j] = 1
	}
	for sweep := 0; sweep < 100; sweep++ {
		rotated := false
		for p := range n {
			wp := w[p]
			for q := p + 1; q < n; q++ {
				wq := w[q]
				alpha, beta, gamma := 0.0, 0.0, 0.0
				for i, x := range wp {
					y := wq[i]
					alpha += x * x
					beta += y * y
					gamma += x * y
				}
				if gamma == 0 || math.Abs(gamma) <= 1e-15*math.Sqrt(alpha*beta) {
					continue
				}
				rotated = true
				zeta := (beta - alpha) / (2 * gamma)
				t := 1 / (math.Abs(zeta) + math.Sqrt(1+zeta*zeta))
				if zeta < 0 {
					t = -t
				}
				c := 1 / math.Sqrt(1+t*t)
				sn := c * t
				for i, x := range wp {
					y := wq[i]
					wp[i] = c*x - sn*y
					wq[i] = sn*x + c*y
				}
				vp, vq := vd[p], vd[q]
				for i, x := range vp {
					y := vq[i]
					vp[i] = c*x - sn*y
					vq[i] = sn*x + c*y
				}
			}
		}
		if !rotated {
			break
		}
	}
	sv := make([]float64, n)
	for j := range n {
		norm := 0.0
		for _, x := range w[j] {
			norm += x * x
		}
		sv[j] = math.Sqrt(norm)
	}
	order := make([]int, n)
	for i := range order {
		order[i] = i
	}
	sort.SliceStable(order, func(x, y int) bool { return sv[order[x]] > sv[order[y]] })
	u = &object.Matrix{Rows: m, Cols: n, Data: make([]float64, m*n)}
	v = &object.Matrix{Rows: n, Cols: n, Data: make([]float64, n*n)}
	s = make([]float64, n)
	for c, j := range order {
		s[c] = sv[j]
		for i := range m {
			if sv[j] > 0 {
				u.Data[i*n+c] = w[j][i] / sv[j]
			}
		}
		for i := range n {
			v.Data[i*n+c] = vd[j][i]
		}
	}
	// Flip each pair of columns together, so U S Vᵀ is unchanged.
	for c := range n {
		for i := range n {
			x := v.Data[i*n+c]
			if math.Abs(x) > 1e-12 {
				if x < 0 {
					for k := range n {
						v.Data[k*n+c] = -v.Data[k*n+c]
					}
					for k := range m {
						u.Data[k*n+c] = -u.Data[k*n+c]
					}
				}
				break
			}
		}
	}
	return u, s, v
}

// transposed is m with rows and columns swapped.
func transposed(m *object.Matrix) *object.Matrix {
	t := &object.Matrix{Rows: m.Cols, Cols: m.Rows, Data: make([]float64, len(m.Data)), Ints: m.Ints}
	for r := range m.Rows {
		for c := range m.Cols {
			t.Data[c*m.Rows+r] = m.Data[r*m.Cols+c]
		}
	}
	return t
}

// matMul is a × b; the caller has checked the sizes. The loop order
// (i, k, j) walks both matrices row by row, which keeps it cache-friendly,
// and a big product shares its rows out among the computer's cores (each
// row is worked out the same way either way, so the answer doesn't
// change).
func matMul(a, b *object.Matrix) *object.Matrix {
	out := &object.Matrix{Rows: a.Rows, Cols: b.Cols, Data: make([]float64, a.Rows*b.Cols), Ints: a.Ints && b.Ints}
	n, p := a.Cols, b.Cols
	rows := func(from, to int) {
		for i := from; i < to; i++ {
			row := out.Data[i*p : (i+1)*p]
			for k := range n {
				f := a.Data[i*n+k]
				if f == 0 {
					continue
				}
				brow := b.Data[k*p : (k+1)*p]
				for j, x := range brow {
					row[j] += f * x
				}
			}
		}
	}
	workers := min(runtime.GOMAXPROCS(0), a.Rows)
	if workers < 2 || a.Rows*n*p < 1<<20 {
		rows(0, a.Rows)
	} else {
		var wg sync.WaitGroup
		per := (a.Rows + workers - 1) / workers
		for from := 0; from < a.Rows; from += per {
			wg.Add(1)
			go func(from, to int) {
				defer wg.Done()
				rows(from, to)
			}(from, min(from+per, a.Rows))
		}
		wg.Wait()
	}
	out.CheckInts()
	return out
}
