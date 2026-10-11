// Copyright 2017-2026 Nigele McCoy
// Licensed under the Apache License, Version 2.0;
// see LICENSE and NOTICE.

package evaluator

import (
	"fmt"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"time"

	"Turtle/ast"
	"Turtle/object"
)

// hypothesis[...] (core, no import): a claim tried without failing. It
// gives true or false; on a line of its own it also shows a report: what
// was tried, whether it holds, and when it doesn't, where it breaks.
// Unlike check, verify and validate it never stops the program: an error
// while trying the claim is a "no", with the error as where it broke
// (only a misspelled name still stops, so a typo isn't read as a no).

// hypothesisReport is what trying a claim found.
type hypothesisReport struct {
	holds   bool
	summary string // after "holds": "no: 3 of 200 items (1.5%) break it"
	rows    []reportRow
}

// reportRow is a labeled line of a report; more lines go under it.
type reportRow struct {
	label string
	lines []string
}

func (r *hypothesisReport) add(label string, lines ...string) {
	if len(lines) > 0 {
		r.rows = append(r.rows, reportRow{label, lines})
	}
}

func (r hypothesisReport) text(h *ast.HypothesisExpression, line int, took time.Duration, mem uint64) string {
	var b strings.Builder
	fmt.Fprintf(&b, "hypothesis %s (line %d)\n", h.Text, line)
	yes := "no"
	if r.holds {
		yes = "yes"
	}
	if r.summary != "" {
		yes += ": " + r.summary
	}
	fmt.Fprintf(&b, "  %-9s %s\n", "holds", yes)
	for _, row := range r.rows {
		for i, l := range row.lines {
			label := row.label
			if i > 0 {
				label = ""
			}
			fmt.Fprintf(&b, "  %-9s %s\n", label, l)
		}
	}
	fmt.Fprintf(&b, "  %-9s %s\n%s", "took", fmtDuration(took), memoryLine(mem))
	return b.String()
}

// evalHypothesis tries the claim and gives true or false; shown (a line
// of its own) also shows the report.
func (it *Interpreter) evalHypothesis(h *ast.HypothesisExpression, env *object.Environment, shown bool) object.Object {
	line := currentLine
	start, mem := time.Now(), uint64(0)
	if shown { // measured only for the report: a value in a loop stays quick
		mem = allocatedBytes()
	}
	var r hypothesisReport
	if fe := it.protect(func() { r = it.tryHypothesis(h, env) }); fe != nil {
		if fe.kind == kindName {
			panic(*fe)
		}
		r = hypothesisReport{summary: "trying it stopped with " + aKind(fe.kind) + " error"}
		r.add("where", errorWhere(fe))
	}
	currentLine = line
	if shown {
		it.showReport(env, r.text(h, line, time.Since(start), mem))
	}
	return object.Bool(r.holds)
}

// errorWhere is an error's message and, when it has one, its line.
func errorWhere(fe *fatalError) string {
	if fe.line > 0 {
		return strings.TrimSuffix(place(fe.file, fe.line), ": ") + ": " + fe.text
	}
	return fe.text
}

func (it *Interpreter) tryHypothesis(h *ast.HypothesisExpression, env *object.Environment) hypothesisReport {
	switch {
	case h.Verify != nil:
		return it.hypothesizeData(h.Verify, env)
	case h.Validate != nil:
		return it.hypothesizeFunction(h.Validate, env)
	case h.Theorem != nil:
		return it.hypothesizeTheorem(it.hypothesisTheory(h.Theory, env), h.Theorem)
	case h.Use != nil && h.Want != nil:
		return it.hypothesizeProof(h, env)
	case h.Use != nil:
		return it.hypothesizeUse(h.Use, env)
	}
	if id, ok := h.Check.Value.(*ast.Identifier); ok && h.Check.Form == "true" && id.Module == "" {
		if fn := theoryIn(id.Value, env); fn != nil {
			return it.hypothesizeTheory(fn)
		}
	}
	return it.hypothesizeFact(h.Check, env)
}

// ---- a fact ----

func (it *Interpreter) hypothesizeFact(c *ast.CheckStatement, env *object.Environment) hypothesisReport {
	ok, details := it.checkHolds(c, env)
	r := hypothesisReport{holds: ok}
	if ok {
		return r
	}
	for i, d := range details {
		details[i] = strings.TrimRight(d, " ")
	}
	r.add("where", details...)
	for _, name := range namesIn(c.Value) {
		if v, found := env.Get(name); found {
			if _, isFn := v.(*object.Function); !isFn {
				r.add(name, briefValue(v))
			}
		}
	}
	return r
}

// namesIn are the names e reads, each once, in the order written; not
// inside a function written in it (x give ...), whose names are its own.
func namesIn(e ast.Node) []string {
	var names []string
	seen := map[string]bool{}
	var walk func(v reflect.Value)
	walk = func(v reflect.Value) {
		if !v.IsValid() {
			return
		}
		switch v.Kind() {
		case reflect.Interface, reflect.Pointer:
			if v.IsNil() {
				return
			}
			switch x := v.Interface().(type) {
			case *ast.FunctionLiteral:
				return
			case *ast.Identifier:
				if x.Module == "" && !seen[x.Value] {
					seen[x.Value] = true
					names = append(names, x.Value)
				}
				return
			}
			walk(v.Elem())
		case reflect.Struct:
			for i := 0; i < v.NumField(); i++ {
				if v.Type().Field(i).IsExported() {
					walk(v.Field(i))
				}
			}
		case reflect.Slice:
			for i := 0; i < v.Len(); i++ {
				walk(v.Index(i))
			}
		}
	}
	walk(reflect.ValueOf(e))
	return names
}

// ---- data: each, any, not, at least, at most, exactly ----

func (it *Interpreter) hypothesizeData(v *ast.VerifyStatement, env *object.Environment) hypothesisReport {
	coll := it.evalExpression(v.Collection, env)
	run := it.tryRule(v.Rule, coll, env)
	ok, _ := run.verdict(v.Rule)
	r := hypothesisReport{holds: ok}
	total, nf, nb := len(run.items), len(run.follow), len(run.breaks)
	units := plural(total, run.unit)
	share := func(n int) string {
		if total == 0 {
			return ""
		}
		return " (" + percent(n, total) + ")"
	}
	follows := fmt.Sprintf("%d of %d %s%s %s it", nf, total, units, share(nf), verbFor(nf, "follows", "follow"))
	var shown []ruleItem // the items to list under where
	explain := false     // whether each one's why is worth showing
	switch q := v.Rule.Quant; {
	case q == "each" && ok:
		r.summary = fmt.Sprintf("all %d %s follow it", total, units)
	case q == "each":
		r.summary = fmt.Sprintf("%d of %d %s%s %s it", nb, total, units, share(nb), verbFor(nb, "breaks", "break"))
		shown, explain = run.breaks, true
	case q == "any" && ok:
		r.summary = follows
	case q == "any" && total == 0:
		r.summary = "there are no " + run.unit + "s"
	case q == "any":
		r.summary = fmt.Sprintf("none of the %d %s follows it", total, units)
	case q == "not" && ok:
		r.summary = fmt.Sprintf("none of the %d %s follows it", total, units)
	case q == "not":
		r.summary = follows + ", and none should"
		shown = run.follow
	default:
		how := map[string]string{"atleast": "at least", "atmost": "at most", "exactly": "exactly"}[q]
		r.summary = fmt.Sprintf("%s; %s %d should", follows, how, run.count)
		if !ok && nf > run.count {
			shown = run.follow
		}
	}
	r.add(dataLabel(v.Collection), briefValue(coll))
	r.add("where", it.itemLines(run, shown, explain)...)
	if ruleChecksType(v.Rule.Fn) || ruleChecksType(run.fn.Body) {
		r.add("types", kindCounts(run.items))
	}
	return r
}

// dataLabel names the data in a report: its name, or "data".
func dataLabel(e ast.Expression) string {
	if id, ok := e.(*ast.Identifier); ok && len(id.Value) <= 9 {
		return id.Value
	}
	return "data"
}

// percent is n of total as a percent, to one place at most: 1.5%, 4%.
func percent(n, total int) string {
	return strings.TrimSuffix(fmt.Sprintf("%.1f", 100*float64(n)/float64(total)), ".0") + "%"
}

// itemLines lists items (10 at most) by where they are, each with its
// value and, when explain, why the rule is false for it.
func (it *Interpreter) itemLines(run ruleRun, items []ruleItem, explain bool) []string {
	width := 0
	for i, item := range items {
		if i < 10 {
			width = max(width, len(item.label))
		}
	}
	var out []string
	for i, item := range items {
		if i == 10 {
			out = append(out, fmt.Sprintf("... and %d more", len(items)-10))
			break
		}
		vals := make([]string, len(item.args))
		for k, a := range item.args {
			vals[k] = briefValue(a)
		}
		label := item.label
		if label == "" {
			label = "item"
			width = max(width, 4)
		}
		line := fmt.Sprintf("%-*s  %s", width, label, strings.Join(vals, ", "))
		if explain {
			if why := it.ruleWhy(run.fn, item.args); why != "" {
				line += "   (" + why + ")"
			}
		}
		out = append(out, line)
	}
	return out
}

// ruleWhy says why a rule of one expression (x give qty of x > 0) is
// false for args: "qty of x is 0, which isn't > 0". "" when it can't tell.
func (it *Interpreter) ruleWhy(fn *object.Function, args []object.Object) string {
	if fn.Body == nil || len(fn.Body.Statements) != 1 || len(fn.Parameters) != len(args) {
		return ""
	}
	ret, ok := fn.Body.Statements[0].(*ast.ReturnStatement)
	if !ok || ret.Value == nil {
		return ""
	}
	env := object.NewEnclosedEnvironment(fn.Env)
	for i, name := range fn.Parameters {
		env.Set(name, args[i])
	}
	var details []string
	if it.protect(func() { _, details = it.explainTruth(ret.Value, env) }) != nil {
		return ""
	}
	for i, d := range details {
		details[i] = strings.TrimSpace(d)
	}
	return strings.Join(details, "; ")
}

// ruleChecksType reports whether a rule is about kinds: x give x type Sale.
func ruleChecksType(n ast.Node) bool {
	found := false
	var walk func(v reflect.Value)
	walk = func(v reflect.Value) {
		if found || !v.IsValid() {
			return
		}
		switch v.Kind() {
		case reflect.Interface, reflect.Pointer:
			if v.IsNil() {
				return
			}
			if _, ok := v.Interface().(*ast.TypeCheckExpression); ok {
				found = true
				return
			}
			walk(v.Elem())
		case reflect.Struct:
			for i := 0; i < v.NumField(); i++ {
				if v.Type().Field(i).IsExported() {
					walk(v.Field(i))
				}
			}
		case reflect.Slice:
			for i := 0; i < v.Len(); i++ {
				walk(v.Index(i))
			}
		}
	}
	walk(reflect.ValueOf(n))
	return found
}

// kindCounts is how many items are of each kind, most first: 197 Sale,
// 2 map, 1 string.
func kindCounts(items []ruleItem) string {
	counts := map[string]int{}
	var kinds []string
	for _, item := range items {
		if len(item.args) == 0 {
			continue
		}
		k := typeName(item.args[len(item.args)-1])
		if counts[k] == 0 {
			kinds = append(kinds, k)
		}
		counts[k]++
	}
	sort.SliceStable(kinds, func(i, j int) bool { return counts[kinds[i]] > counts[kinds[j]] })
	parts := make([]string, len(kinds))
	for i, k := range kinds {
		parts[i] = fmt.Sprintf("%d %s", counts[k], k)
	}
	return strings.Join(parts, ", ")
}

// ---- a function, on random inputs ----

func (it *Interpreter) hypothesizeFunction(v *ast.ValidateStatement, env *object.Environment) hypothesisReport {
	found := it.searchValidate(v, env, true)
	if !found.failed {
		r := hypothesisReport{holds: true, summary: fmt.Sprintf("on %d random inputs", found.tried)}
		r.add("seed", fmt.Sprint(found.seed))
		return r
	}
	r := hypothesisReport{summary: fmt.Sprintf("failed on case %d of %d", found.caseN, found.cases)}
	var where []string
	for i, in := range found.inputs {
		line := fmt.Sprintf("%s = %s", in.Name, object.Shown(found.vals[i]))
		if !valuesEqual(found.vals[i], found.first[i]) {
			line += fmt.Sprintf("   (smallest found; first failed on %s)", object.Shown(found.first[i]))
		}
		where = append(where, line)
	}
	r.add("where", where...)
	if found.result != nil {
		r.add(v.ResultName, briefValue(found.result))
	}
	why := make([]string, len(found.details))
	for i, d := range found.details {
		why[i] = strings.TrimRight(d, " ")
	}
	r.add("why", why...)
	r.add("seed", fmt.Sprintf("%d   (seed = %d repeats this run)", found.seed, found.seed))
	return r
}

// ---- theories ----

// hypothesisTheory is the theory a hypothesis names.
func (it *Interpreter) hypothesisTheory(id *ast.Identifier, env *object.Environment) *object.Function {
	if fn := theoryIn(id.Value, env); fn != nil {
		return fn
	}
	if _, ok := env.Get(id.Value); ok || userFunction(env, id.Value) {
		fatalf("hypothesis[%s, theorem ...]: %s isn't a theory", id.Value, id.Value)
	}
	fatalKind(kindName, "hypothesis[%s, theorem ...]: there's no theory called %s", id.Value, id.Value)
	return nil
}

// hypothesizeTheory is hypothesis[share]: its proof and theorems, now.
func (it *Interpreter) hypothesizeTheory(fn *object.Function) hypothesisReport {
	p := it.proveTheory(fn, theorySeed(it.Global))
	r := hypothesisReport{holds: !p.failed()}
	held := 0
	for _, c := range p.cases {
		if c.ok {
			held++
		}
	}
	thHeld := 0
	for _, t := range p.theorems {
		if t.holds {
			thHeld++
		}
	}
	r.summary = fmt.Sprintf("proof %d of %d, %d of %d %s", held, len(p.cases), thHeld, len(p.theorems), plural(len(p.theorems), "theorem"))
	if len(p.cases) == 0 {
		r.summary = fmt.Sprintf("no proof, %d of %d %s", thHeld, len(p.theorems), plural(len(p.theorems), "theorem"))
	}
	var proof, theorems, where []string
	for _, c := range p.cases {
		mark := "✓"
		if !c.ok {
			mark = "✗"
			where = append(where, fmt.Sprintf("line %d: %s  %s", c.pc.Line, c.pc.Text, c.what))
		}
		proof = append(proof, mark+" "+c.pc.Text)
	}
	for _, t := range p.theorems {
		mark := "✓"
		if !t.holds {
			mark = "✗"
			where = append(where, fmt.Sprintf("line %d: theorem %s fails %s", t.th.Line, t.th.Text, t.why))
		}
		theorems = append(theorems, mark+" "+t.th.Text)
	}
	r.add("where", where...)
	r.add("proof", proof...)
	r.add("theorems", theorems...)
	if p.tried > 0 {
		r.add("tried", fmt.Sprintf("the proof cases and %d random inputs%s (seed %d)", p.tried, refusedNote(p), p.seed))
	}
	r.add("warning", p.warnings...)
	return r
}

// hypothesizeTheorem is hypothesis[share, theorem ...]: a theorem tried
// on the theory before it's written into it.
func (it *Interpreter) hypothesizeTheorem(fn *object.Function, th *ast.Theorem) hypothesisReport {
	ts := *fn.Theory
	ts.Theorems = []*ast.Theorem{th}
	trial := *fn
	trial.Theory = &ts
	p := it.proveTheory(&trial, theorySeed(it.Global))
	t := p.theorems[0]
	held := 0
	for _, c := range p.cases {
		if c.ok {
			held++
		}
	}
	r := hypothesisReport{holds: t.holds && held > 0}
	on := fmt.Sprintf("the %d proof %s", held, plural(held, "case"))
	if p.tried > 0 {
		on += fmt.Sprintf(" and %d random inputs%s", p.tried, refusedNote(p))
	}
	switch {
	case len(p.cases) == 0:
		r.summary = fmt.Sprintf("not tried: %s has no proof cases to learn its inputs from", ts.Name)
	case held == 0:
		r.summary = fmt.Sprintf("not tried: none of %s's proof cases holds, to learn its inputs from", ts.Name)
	case t.holds:
		r.summary = "on " + on
	default:
		r.summary = "it fails " + strings.SplitN(t.why, ":", 2)[0]
		r.add("where", t.why)
	}
	if len(p.cases) > 0 {
		r.add("proof", fmt.Sprintf("%d of %d %s", held, len(p.cases), verbFor(len(p.cases), "holds", "hold")))
	}
	if t.holds && held > 0 && p.tried == 0 {
		r.add("note", "only the proof cases: their values' kinds couldn't be learned for random inputs")
	}
	if p.tried > 0 {
		r.add("seed", fmt.Sprint(p.seed))
	}
	return r
}

// refusedNote says how many random inputs the theory refused, when it
// refused any: those tried nothing.
func refusedNote(p theoryProof) string {
	if p.refused == 0 {
		return ""
	}
	return fmt.Sprintf(", %d of them refused by %s", p.refused, p.ts.Name)
}

// hypothesizeProof is hypothesis[share 1 of 4 . is 25.0].
func (it *Interpreter) hypothesizeProof(h *ast.HypothesisExpression, env *object.Environment) hypothesisReport {
	fn := it.theoryNamed(h.Use.Word, env)
	args := it.theoryArgs(h.Use, env)
	got := it.callFunction(fn, fn.Theory.Name, copies(args))
	want := it.evalExpression(h.Want, env)
	r := hypothesisReport{holds: valuesEqual(got, want)}
	slotRows(&r, fn.Theory, args)
	if !r.holds {
		where := diffLines(got, want)
		if len(where) == 0 {
			where = []string{fmt.Sprintf("gave %s, expected %s", object.Shown(got), object.Shown(want))}
		}
		r.add("where", where...)
	}
	return r
}

// hypothesizeUse is hypothesis[share 2 of 1]: the theory's theorems on
// one use of it.
func (it *Interpreter) hypothesizeUse(tc *ast.TheoryCall, env *object.Environment) hypothesisReport {
	fn := it.theoryNamed(tc.Word, env)
	ts := fn.Theory
	args := it.theoryArgs(tc, env)
	var r hypothesisReport
	slotRows(&r, ts, args)
	if len(ts.Theorems) == 0 {
		r.summary = ts.Name + " has no theorems to try it on; to try what it gives, write hypothesis[" + tc.Word + " ... . is ...]"
		return r
	}
	var result object.Object
	if fe := it.protect(func() { result = it.callFunction(fn, ts.Name, copies(args)) }); fe != nil {
		r.summary = ts.Name + " stopped with " + aKind(fe.kind) + " error"
		r.add("where", errorWhere(fe))
		return r
	}
	r.add("returned", briefValue(result))
	r.holds = true
	var lines []string
	for _, th := range ts.Theorems {
		holds, why := it.checkTheorem(th, theoremEnv(fn, args, result))
		if holds {
			lines = append(lines, "✓ "+th.Text)
			continue
		}
		r.holds = false
		lines = append(lines, "✗ "+th.Text+": "+why)
	}
	r.add("theorems", lines...)
	return r
}

func slotRows(r *hypothesisReport, ts *ast.TheoryStatement, args []object.Object) {
	for i, name := range ts.Slots {
		r.add(name, briefValue(args[i]))
	}
}

// ---- reports, on the screen and in reportfile ----

const reportFileName = "reportfile"

var ansiCodes = regexp.MustCompile("\x1b\\[[0-9;]*m")

// showReport shows a diagnose or hypothesis report, and adds it to the
// program's reportfile when it has one (reportfile = "checks.txt"), with
// the time and the line it came from.
func (it *Interpreter) showReport(env *object.Environment, text string) {
	fmt.Println(text)
	it.fileReport(env, text)
}

// fileReport adds a report already shown to the program's reportfile,
// when it has one.
func (it *Interpreter) fileReport(env *object.Environment, text string) {
	if env == nil {
		env = it.Global
	}
	v, ok := env.Get(reportFileName)
	if !ok && it.Global != nil {
		v, ok = it.Global.Get(reportFileName)
	}
	if !ok {
		return
	}
	switch f := v.(type) {
	case *object.None:
	case *object.String:
		file := currentFile
		if file == "" {
			file = it.Script
		}
		head := fmt.Sprintf("== %s  %s", time.Now().Format("2006-01-02 15:04:05"), strings.TrimSuffix(place(file, currentLine), ": "))
		it.appendLine(f.Value, head+"\n"+ansiCodes.ReplaceAllString(text, "")+"\n", -1, 0)
	default:
		fatalf("reportfile must be a file's path, or none, got %s", object.Shown(f))
	}
}
