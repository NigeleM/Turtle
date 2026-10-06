package sqlite

import (
	"sort"
	"strings"
)

// Window functions: f(...) OVER (PARTITION BY ... ORDER BY ... frame).
// They're computed after WHERE, GROUP BY and HAVING, on the rows (or
// groups) those leave: the rows are split into partitions, each sorted by
// the window's ORDER BY, and each function is computed for every row
// from the rows of its partition (for aggregates and first/last/nth
// value, the rows of its frame).

var rankingFuncs = map[string]bool{"row_number": true, "rank": true, "dense_rank": true, "percent_rank": true,
	"cume_dist": true, "ntile": true, "lag": true, "lead": true, "first_value": true, "last_value": true, "nth_value": true}

// winPlan is one window function call and its resolved window.
type winPlan struct {
	call      *callExpr
	partition []expr
	partColls []string
	order     []orderTerm
	ordColls  []string
	frame     frameSpec
}

// winSnap is one row of the window stage: the state that evaluates
// expressions for it (its source rows, and its group's aggregates).
type winSnap struct {
	cur   []*srcRow
	aggs  []*aggState
	final bool
}

// planWindows finds the window function calls in the result columns and
// ORDER BY, and resolves their windows.
func (c *corePlan) planWindows() {
	visit := func(e expr) {
		walk(e, func(e expr) {
			x, ok := e.(*callExpr)
			if !ok || x.over == nil || x.win > 0 {
				return
			}
			c.addWindow(x)
		})
	}
	for _, rc := range c.cols {
		visit(rc.e)
	}
	for _, t := range c.order {
		if t.pos == 0 {
			visit(t.e)
		}
	}
}

func (c *corePlan) addWindow(x *callExpr) {
	if !rankingFuncs[x.name] && !aggNames[x.name] {
		fail("SQL: %s() may not be used as a window function", x.name)
	}
	if x.distinct {
		fail("SQL: DISTINCT is not supported for window functions")
	}
	if x.filter != nil && rankingFuncs[x.name] {
		fail("SQL: FILTER clause may only be used with aggregate window functions")
	}
	need := func(lo, hi int) {
		if len(x.args) < lo || len(x.args) > hi || x.star && x.name != "count" {
			fail("SQL: wrong number of arguments to function %s()", x.name)
		}
	}
	switch x.name {
	case "row_number", "rank", "dense_rank", "percent_rank", "cume_dist":
		need(0, 0)
	case "ntile":
		need(1, 1)
	case "lag", "lead":
		need(1, 3)
	case "first_value", "last_value":
		need(1, 1)
	case "nth_value":
		need(2, 2)
	}
	// Merge in a named window: OVER w, or OVER (w ORDER BY ...).
	spec := *x.over
	if spec.name != "" {
		base, ok := c.s.windows[strings.ToLower(spec.name)]
		if !ok {
			fail("SQL: no such window: %s", spec.name)
		}
		if len(spec.partition) > 0 {
			fail("SQL: cannot override PARTITION clause of window: %s", spec.name)
		}
		if len(spec.order) > 0 && len(base.order) > 0 {
			fail("SQL: cannot override ORDER BY clause of window: %s", spec.name)
		}
		if base.frame != nil && (x.over.partition != nil || x.over.order != nil || x.over.frame != nil) {
			fail("SQL: cannot override frame specification of window: %s", spec.name)
		}
		merged := *base
		if len(spec.order) > 0 {
			merged.order = spec.order
		}
		if spec.frame != nil {
			merged.frame = spec.frame
		}
		spec = merged
	}
	w := &winPlan{call: x, partition: spec.partition, order: spec.order}
	for _, e := range w.partition {
		c.resolve(e, false)
		c.findAggs(e)
		coll, _ := exprCollation(e)
		w.partColls = append(w.partColls, coll)
	}
	for _, t := range w.order {
		c.resolve(t.e, false)
		c.findAggs(t.e)
		coll, _ := exprCollation(t.e)
		w.ordColls = append(w.ordColls, coll)
	}
	if spec.frame != nil {
		w.frame = *spec.frame
		c.resolve(w.frame.start.n, false)
		c.resolve(w.frame.end.n, false)
	} else {
		// The default: from the first row to the current row's last peer.
		w.frame = frameSpec{unit: "RANGE", start: frameBound{kind: "UNBOUNDED PRECEDING"}, end: frameBound{kind: "CURRENT ROW"}}
	}
	if w.frame.unit == "RANGE" && (w.frame.start.n != nil || w.frame.end.n != nil) && len(w.order) != 1 {
		fail("SQL: RANGE with offset PRECEDING/FOLLOWING requires one ORDER BY expression")
	}
	for _, a := range x.args {
		if hasWindow(a) {
			fail("SQL: misuse of window function %s()", x.name)
		}
	}
	c.wins = append(c.wins, w)
	x.win = len(c.wins)
}

// hasWindow reports whether e calls a window function.
func hasWindow(e expr) bool {
	found := false
	walk(e, func(e expr) {
		if x, ok := e.(*callExpr); ok && x.over != nil {
			found = true
		}
	})
	return found
}

func (r *runner) snapshot() winSnap {
	return winSnap{cur: append([]*srcRow{}, r.cur...), aggs: r.aggs, final: r.final}
}

func (r *runner) restore(s winSnap) {
	copy(r.cur, s.cur)
	r.aggs, r.final = s.aggs, s.final
}

// computeWindows works out every window function for every row.
func (c *corePlan) computeWindows(r *runner, snaps []winSnap) [][]Value {
	n := len(snaps)
	out := make([][]Value, n)
	for i := range out {
		out[i] = make([]Value, len(c.wins))
	}
	for wi, w := range c.wins {
		x := w.call
		// Each row's partition key, order keys, arguments and FILTER.
		parts := make([][]Value, n)
		ords := make([][]Value, n)
		args := make([][]Value, n)
		keep := make([]bool, n)
		for i, s := range snaps {
			r.restore(s)
			parts[i] = make([]Value, len(w.partition))
			for k, e := range w.partition {
				parts[i][k] = r.eval(e)
			}
			ords[i] = make([]Value, len(w.order))
			for k, t := range w.order {
				ords[i][k] = r.eval(t.e)
			}
			args[i] = make([]Value, len(x.args))
			for k, a := range x.args {
				args[i][k] = r.eval(a)
			}
			keep[i] = x.filter == nil || truthy(r.eval(x.filter))
		}
		idx := make([]int, n)
		for i := range idx {
			idx[i] = i
		}
		sort.SliceStable(idx, func(a, b int) bool {
			pa, pb := parts[idx[a]], parts[idx[b]]
			for k := range pa {
				if d := compareColl(pa[k], pb[k], w.partColls[k]); d != 0 {
					return d < 0
				}
			}
			return orderLess(ords[idx[a]], ords[idx[b]], w.order, w.ordColls)
		})
		samePart := func(a, b int) bool {
			for k := range parts[a] {
				if compareColl(parts[a][k], parts[b][k], w.partColls[k]) != 0 {
					return false
				}
			}
			return true
		}
		peers := func(a, b int) bool {
			for k := range ords[a] {
				x, y := ords[a][k], ords[b][k]
				if (x == nil) != (y == nil) || (x != nil && compareColl(x, y, w.ordColls[k]) != 0) {
					return false
				}
			}
			return true
		}
		for start := 0; start < n; {
			end := start + 1
			for end < n && samePart(idx[start], idx[end]) {
				end++
			}
			c.windowPartition(r, snaps, w, wi, idx[start:end], ords, args, keep, peers, out)
			start = end
		}
	}
	return out
}

// windowPartition computes one window function over one partition, its
// rows (as indexes into the stage's rows) already in window order.
func (c *corePlan) windowPartition(r *runner, snaps []winSnap, w *winPlan, wi int, rows []int, ords, args [][]Value, keep []bool, peers func(a, b int) bool, out [][]Value) {
	x := w.call
	m := len(rows)
	// Peer groups: group[p] is the group of position p; first/last rows.
	group := make([]int, m)
	var gStart, gEnd []int
	for p := 0; p < m; p++ {
		if p == 0 || !peers(rows[p-1], rows[p]) {
			gStart = append(gStart, p)
			if p > 0 {
				gEnd = append(gEnd, p-1)
			}
		}
		group[p] = len(gStart) - 1
	}
	gEnd = append(gEnd, m-1)
	set := func(p int, v Value) { out[rows[p]][wi] = v }
	switch x.name {
	case "row_number":
		for p := 0; p < m; p++ {
			set(p, int64(p+1))
		}
		return
	case "rank", "dense_rank", "percent_rank", "cume_dist":
		for p := 0; p < m; p++ {
			g := group[p]
			switch x.name {
			case "rank":
				set(p, int64(gStart[g]+1))
			case "dense_rank":
				set(p, int64(g+1))
			case "percent_rank":
				if m == 1 {
					set(p, 0.0)
				} else {
					set(p, float64(gStart[g])/float64(m-1))
				}
			default:
				set(p, float64(gEnd[g]+1)/float64(m))
			}
		}
		return
	case "ntile":
		for p := 0; p < m; p++ {
			r.restore(snaps[rows[p]])
			nb := r.eval(x.args[0])
			k, ok := applyAffinity(nb, affInteger).(int64)
			if !ok || k <= 0 {
				fail("SQL: argument of ntile must be a positive integer")
			}
			// k buckets; the first (m mod k) hold one more row.
			size, extra := int64(m)/k, int64(m)%k
			var bucket int64
			if size == 0 {
				bucket = int64(p) + 1
			} else if int64(p) < extra*(size+1) {
				bucket = int64(p)/(size+1) + 1
			} else {
				bucket = extra + (int64(p)-extra*(size+1))/size + 1
			}
			set(p, bucket)
		}
		return
	case "lag", "lead":
		for p := 0; p < m; p++ {
			off := int64(1)
			if len(x.args) > 1 {
				o, ok := applyAffinity(args[rows[p]][1], affInteger).(int64)
				if !ok {
					fail("SQL: second argument to %s must be a non-negative integer", x.name)
				}
				off = o
			}
			q := int64(p) - off
			if x.name == "lead" {
				q = int64(p) + off
			}
			if q >= 0 && q < int64(m) {
				set(p, args[rows[q]][0])
			} else if len(x.args) > 2 {
				set(p, args[rows[p]][2])
			}
		}
		return
	}
	// The rest use the frame: aggregates, first/last/nth_value.
	frame := func(p int) (int, int) { return c.frameBounds(r, snaps, w, rows, p, ords, group, gStart, gEnd) }
	excluded := func(p, q int) bool {
		switch w.frame.exclude {
		case "CURRENT ROW":
			return q == p
		case "GROUP":
			return group[q] == group[p]
		case "TIES":
			return q != p && group[q] == group[p]
		}
		return false
	}
	if rankingFuncs[x.name] { // first_value, last_value, nth_value
		for p := 0; p < m; p++ {
			lo, hi := frame(p)
			var inFrame []int
			for q := lo; q <= hi; q++ {
				if !excluded(p, q) {
					inFrame = append(inFrame, q)
				}
			}
			switch x.name {
			case "first_value":
				if len(inFrame) > 0 {
					set(p, args[rows[inFrame[0]]][0])
				}
			case "last_value":
				if len(inFrame) > 0 {
					set(p, args[rows[inFrame[len(inFrame)-1]]][0])
				}
			default:
				k, ok := applyAffinity(args[rows[p]][1], affInteger).(int64)
				if !ok || k <= 0 {
					fail("SQL: second argument to nth_value must be a positive integer")
				}
				if int(k) <= len(inFrame) {
					set(p, args[rows[inFrame[k-1]]][0])
				}
			}
		}
		return
	}
	// Aggregates over the frame. From the partition's first row on (the
	// usual case: running totals), add rows as the frame grows; otherwise
	// count each frame afresh.
	addRow := func(a *aggState, q int) {
		i := rows[q]
		if !keep[i] {
			return
		}
		if x.star {
			a.count++
			return
		}
		v := args[i][0]
		if v == nil && !strings.HasPrefix(x.name, "json_group") {
			return
		}
		var second Value = ","
		if len(x.args) > 1 {
			second = args[i][1]
		}
		a.add(v, second)
	}
	running := w.frame.start.kind == "UNBOUNDED PRECEDING" && w.frame.exclude == ""
	var acc *aggState
	added := -1
	for p := 0; p < m; p++ {
		lo, hi := frame(p)
		var a *aggState
		if running && hi >= added {
			if acc == nil {
				acc = newAggStates([]*callExpr{x})[0]
			}
			for q := added + 1; q <= hi; q++ {
				addRow(acc, q)
			}
			added = hi
			a = acc
		} else {
			a = newAggStates([]*callExpr{x})[0]
			for q := lo; q <= hi; q++ {
				if !excluded(p, q) {
					addRow(a, q)
				}
			}
		}
		set(p, r.aggResult(a))
	}
}

// frameBounds gives the first and last positions of p's frame (first >
// last: empty).
func (c *corePlan) frameBounds(r *runner, snaps []winSnap, w *winPlan, rows []int, p int, ords [][]Value, group, gStart, gEnd []int) (int, int) {
	m := len(rows)
	f := w.frame
	offset := func(b frameBound) int64 {
		r.restore(snaps[rows[p]])
		v := r.eval(b.n)
		if f.unit == "RANGE" {
			return 0
		}
		n, ok := applyAffinity(v, affInteger).(int64)
		if !ok || n < 0 {
			fail("SQL: frame starting offset must be a non-negative integer")
		}
		return n
	}
	rangeValue := func(b frameBound) Value {
		r.restore(snaps[rows[p]])
		v := numericValue(r.eval(b.n))
		if v == nil || toFloat(v) < 0 {
			fail("SQL: frame starting offset must be a non-negative number")
		}
		return v
	}
	bound := func(b frameBound, isStart bool) int {
		switch b.kind {
		case "UNBOUNDED PRECEDING":
			return 0
		case "UNBOUNDED FOLLOWING":
			return m - 1
		case "CURRENT ROW":
			switch f.unit {
			case "ROWS":
				return p
			}
			if isStart {
				return gStart[group[p]]
			}
			return gEnd[group[p]]
		}
		sign := int64(1)
		if b.kind == "PRECEDING" {
			sign = -1
		}
		switch f.unit {
		case "ROWS":
			q := int64(p) + sign*offset(b)
			return clampPos(q, m, isStart)
		case "GROUPS":
			g := int64(group[p]) + sign*offset(b)
			if g < 0 {
				if isStart {
					return 0
				}
				return -1
			}
			if g >= int64(len(gStart)) {
				if isStart {
					return m
				}
				return m - 1
			}
			if isStart {
				return gStart[g]
			}
			return gEnd[g]
		}
		// RANGE n PRECEDING / FOLLOWING: by the ORDER BY value.
		cur := ords[rows[p]][0]
		if cur == nil {
			// NULLs are their own range: the current row's peers.
			if isStart {
				return gStart[group[p]]
			}
			return gEnd[group[p]]
		}
		delta := rangeValue(b)
		desc := w.order[0].desc
		target := arith("+", numericValue(cur), delta)
		if (b.kind == "PRECEDING") != desc {
			target = arith("-", numericValue(cur), delta)
		}
		// within: does q's value fall inside the frame on this side?
		within := func(q int) bool {
			v := ords[rows[q]][0]
			if v == nil {
				return false
			}
			d := compare(numericValue(v), target)
			if isStart {
				if desc {
					return d <= 0
				}
				return d >= 0
			}
			if desc {
				return d >= 0
			}
			return d <= 0
		}
		if isStart {
			for q := 0; q < m; q++ {
				if ords[rows[q]][0] != nil && within(q) {
					return q
				}
			}
			return m
		}
		last := -1
		for q := 0; q < m; q++ {
			if ords[rows[q]][0] != nil && within(q) {
				last = q
			}
		}
		return last
	}
	return bound(f.start, true), bound(f.end, false)
}

func clampPos(q int64, m int, isStart bool) int {
	if q < 0 {
		if isStart {
			return 0
		}
		return -1
	}
	if q >= int64(m) {
		if isStart {
			return m
		}
		return m - 1
	}
	return int(q)
}
