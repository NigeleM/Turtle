package evaluator

import (
	"fmt"
	"os"
	"strings"
	"time"
	"unicode/utf8"

	"Turtle/ast"
	"Turtle/object"
	"Turtle/token"
)

// Scrolls (see parser/scroll.go): a value going through steps in order,
// each step's result feeding the next.
//
// Each step runs in a scope of its own where here is the value reaching
// it, so here works anywhere in the step, even in a give made there, and
// means the nearest scroll's value. A saved scroll used as a step runs
// its steps right there, in order (numbered 3.1, 3.2, ...).
//
// Every step's result is recorded, cheaply (the values themselves), so an
// error can carry them for diagnose[e], and diagnose and turtle trace can
// show them. none is a value like any other: it goes on to the next step
// (diagnose marks it, to tell a forgotten return from a real answer).
// An error ends the scroll at its step, keeping its own kind, its message
// saying which step it was: the steps after it depend on it.

// stepRecord is what one step did.
type stepRecord struct {
	shown  string // with snapshot: the value as it showed when the step returned
	num    string // "2", or "3.1" inside a saved scroll used as step 3
	label  string // the step as written: "splitby \",\""
	status string // returned, returned none, failed, not reached
	value  object.Object
	depth  int
	err    string // for failed: the error's message
	// noneFrom is where the none a step was given came from ("2", or
	// "the start"), so a failure can point back at it.
	noneFrom string
}

// scrollRun is one scroll running, with what its steps did so far.
type scrollRun struct {
	records []stepRecord
	from    string // what made the value the next step gets: "the start", or a step's number
	// snapshot notes how each step's value showed as it returned, for
	// diagnose: a later step that changes a list or map in place would
	// otherwise change what an earlier step shows. startShown is the
	// starting value, likewise.
	snapshot   bool
	startShown string
}

const (
	stepReturned   = "returned"
	stepNone       = "returned none"
	stepFailed     = "failed"
	stepNotReached = "not reached"
)

// hereName is the name a step's value has inside it.
const hereName = "here"

var hereIdent = &ast.Identifier{Token: token.Token{Type: token.IDENT, Literal: hereName}, Value: hereName}

// evalScroll is a scroll in an expression: a saved scroll (a value) when
// it has no starting value, or else its result.
func (it *Interpreter) evalScroll(se *ast.ScrollExpression, env *object.Environment) object.Object {
	if se.Start == nil {
		env.Capture()
		return &object.Function{Parameters: []string{"value"}, Scroll: se, Env: env}
	}
	start := it.evalExpression(se.Start, env)
	run := &scrollRun{from: "the start"}
	return it.runScroll(run, se.Steps, env, start)
}

// runScroll runs steps on value from the top: an error coming out of it
// carries the steps' record.
func (it *Interpreter) runScroll(run *scrollRun, steps []*ast.ScrollStep, env *object.Environment, value object.Object) object.Object {
	defer func() {
		if r := recover(); r != nil {
			if fe, ok := r.(fatalError); ok && fe.steps == nil {
				fe.steps = run.list()
				panic(fe)
			}
			panic(r)
		}
	}()
	return it.runSteps(run, steps, env, value, "", 0)
}

// runSteps runs steps in order, numbering them after prefix.
func (it *Interpreter) runSteps(run *scrollRun, steps []*ast.ScrollStep, env *object.Environment, value object.Object, prefix string, depth int) object.Object {
	for i, st := range steps {
		num := prefix + itoa(i+1)
		at := len(run.records)
		run.records = append(run.records, stepRecord{num: num, label: st.Label, depth: depth})
		if _, none := value.(*object.None); none {
			run.records[at].noneFrom = run.from
		}
		value = it.runStep(run, st, env, value, num, len(steps), depth, at, steps[i+1:])
		run.from = "step " + num
		run.records[at].status, run.records[at].value = stepReturned, value
		if run.snapshot {
			run.records[at].shown = briefValue(value)
		}
		if _, none := value.(*object.None); none {
			run.records[at].status = stepNone
		}
		it.traceStep(run.records[at])
	}
	return value
}

// runStep runs one step on value. An error in it is marked as this
// step's (once: an enclosing saved scroll's step doesn't mark it again),
// and the steps after it as not reached.
func (it *Interpreter) runStep(run *scrollRun, st *ast.ScrollStep, env *object.Environment, value object.Object, num string, count, depth, at int, rest []*ast.ScrollStep) (result object.Object) {
	defer func() {
		r := recover()
		if r == nil {
			return
		}
		fe, ok := r.(fatalError)
		if !ok {
			panic(r)
		}
		if run.records[at].status == "" {
			run.records[at].status, run.records[at].err = stepFailed, fe.text
			it.traceStep(run.records[at])
		}
		prefix := ""
		if i := strings.LastIndex(num, "."); i >= 0 {
			prefix = num[:i+1]
		}
		run.notReached(rest, prefix, count-len(rest)+1, depth)
		last := num[len(prefix):]
		switch {
		case !fe.scrolled:
			fe.scrollWhere = fmt.Sprintf("scroll step %s of %d (%s)", last, count, st.Label)
			fe.scrollBase = fe.text
			fe.text = fe.scrollWhere + ": " + fe.scrollBase
			fe.scrolled, fe.scrollAt = true, num
		case fe.scrollAt == "":
			fe.scrollAt = num // scrollFail named this step already
		case strings.HasPrefix(fe.scrollAt, num+"."):
			// A step of a saved scroll failed: say which step it sits in.
			in := fmt.Sprintf(", in step %s of %d (%s)", last, count, st.Label)
			if fe.scrollWhere != "" {
				fe.scrollWhere += in
				fe.text = fe.scrollWhere + ": " + fe.scrollBase
			} else {
				fe.text += in
			}
			fe.scrollAt = num
		}
		fe.msg = place(fe.file, fe.line) + fe.text
		panic(fe)
	}()

	stepEnv := object.NewEnclosedEnvironment(env)
	stepEnv.Set(hereName, value)
	switch st.Kind {
	case ast.StepHere:
		return it.evalExpression(st.Expr, stepEnv)
	case ast.StepValue:
		return it.applyStepValue(run, it.evalExpression(st.Expr, stepEnv), st, value, num, depth)
	}
	call := st.Call
	if call.Module == "" && len(call.Arguments) == 0 {
		if v, ok := env.Get(call.Name); ok {
			if _, isFn := v.(*object.Function); isFn {
				return it.applyStepValue(run, v, st, value, num, depth)
			}
			if _, def := env.GetFunction(call.Name); !def {
				scrollFail("scroll step %s (%s) is %s, not a function or scroll", lastStep(num), st.Label, aValue(v))
			}
		}
	}
	if call.Module == "" && len(call.Arguments) == 0 {
		if fn, ok := env.GetFunction(call.Name); ok && fn.Shape == nil && len(fn.Parameters) != 1 {
			if _, isVar := env.Get(call.Name); !isVar {
				scrollFail("scroll step %s (%s) is a function of %d values; a step gets one (the value): write it with brackets and here, like %s[here, ...]", lastStep(num), st.Label, len(fn.Parameters), call.Name)
			}
		}
	}
	// "json load": module json's load, not json[value, load].
	if call.Module == "" && len(call.Arguments) == 1 {
		if id, ok := call.Arguments[0].(*ast.Identifier); ok && id.Module == "" {
			if _, isVar := env.Get(call.Name); !isVar {
				if im, isMod := env.FindImport(call.Name); isMod && im.Module.ExportsFunction(id.Value) {
					call = &ast.CallExpression{Token: call.Token, Module: call.Name, Name: id.Value}
				}
			}
		}
	}
	withValue := &ast.CallExpression{Token: call.Token, Module: call.Module, Name: call.Name,
		Arguments: append([]ast.Expression{hereIdent}, call.Arguments...)}
	return it.evalCall(withValue, stepEnv)
}

// applyStepValue runs a step that is a function or a saved scroll.
func (it *Interpreter) applyStepValue(run *scrollRun, v object.Object, st *ast.ScrollStep, value object.Object, num string, depth int) object.Object {
	fn, ok := v.(*object.Function)
	if !ok {
		scrollFail("scroll step %s (%s) is %s, not a function or scroll", num, st.Label, aValue(v))
	}
	if fn.Scroll == nil {
		if fn.Shape == nil && len(fn.Parameters) != 1 {
			scrollFail("scroll step %s (%s) is a function of %d values; a step gets one (the value): write it with brackets and here", num, st.Label, len(fn.Parameters))
		}
		return it.callFunction(fn, st.Label, []object.Object{value})
	}
	// A saved scroll as a step: its steps, right here.
	for _, f := range it.scrollStack {
		if f == fn {
			scrollFail("scroll %s contains itself (%s)", scrollName(fn), it.scrollChain(fn))
		}
	}
	it.scrollStack = append(it.scrollStack, fn)
	defer func() { it.scrollStack = it.scrollStack[:len(it.scrollStack)-1] }()
	return it.runSteps(run, fn.Scroll.Steps, scrollEnv(fn, it), value, num+".", depth+1)
}

// runSavedScroll is a saved scroll called on a value: s[3], or given to
// process or keep.
func (it *Interpreter) runSavedScroll(fn *object.Function, value object.Object) object.Object {
	for _, f := range it.scrollStack {
		if f == fn {
			scrollFail("scroll %s contains itself (%s)", scrollName(fn), it.scrollChain(fn))
		}
	}
	it.scrollStack = append(it.scrollStack, fn)
	defer func() { it.scrollStack = it.scrollStack[:len(it.scrollStack)-1] }()
	return it.runScroll(&scrollRun{from: "the start"}, fn.Scroll.Steps, scrollEnv(fn, it), value)
}

func scrollEnv(fn *object.Function, it *Interpreter) *object.Environment {
	if fn.Env != nil {
		return fn.Env
	}
	return it.Global
}

func scrollName(fn *object.Function) string {
	if fn.Name == "" {
		return "(unnamed)"
	}
	return fn.Name
}

// scrollChain is "s -> t -> s" for a scroll found inside itself.
func (it *Interpreter) scrollChain(fn *object.Function) string {
	var names []string
	from := 0
	for i, f := range it.scrollStack {
		if f == fn {
			from = i
			break
		}
	}
	for _, f := range it.scrollStack[from:] {
		names = append(names, scrollName(f))
	}
	return strings.Join(append(names, scrollName(fn)), " -> ")
}

// notReached records steps that never ran: rest, numbered from first.
func (run *scrollRun) notReached(rest []*ast.ScrollStep, prefix string, first, depth int) {
	for i, st := range rest {
		run.records = append(run.records, stepRecord{num: prefix + itoa(first+i), label: st.Label, status: stepNotReached, depth: depth})
	}
}

// list is the record as Turtle values, which diagnose[e] gives back: one
// map per step, with step, name, returned and status.
func (run *scrollRun) list() *object.List {
	out := &object.List{}
	for _, r := range run.records {
		m := object.NewMap()
		m.Put(&object.String{Value: "step"}, &object.String{Value: r.num})
		m.Put(&object.String{Value: "name"}, &object.String{Value: r.label})
		v := r.value
		if v == nil {
			v = object.NoneValue
		}
		m.Put(&object.String{Value: "returned"}, v)
		status := r.status
		if status == "" {
			status = stepFailed
		}
		m.Put(&object.String{Value: "status"}, &object.String{Value: status})
		m.Put(&object.String{Value: "nonefrom"}, &object.String{Value: r.noneFrom})
		out.Elements = append(out.Elements, m)
	}
	return out
}

// aValue names a value's kind with an article: "a number", "an error".
func aValue(v object.Object) string {
	switch v.(type) {
	case *object.Integer, *object.Float:
		return "a number"
	case *object.String:
		return "text"
	case *object.None:
		return "none"
	}
	n := typeName(v)
	if strings.ContainsRune("aeiou", rune(n[0])) {
		return "an " + n
	}
	return "a " + n
}

// briefValue is a value shortened for one line of a report.
func briefValue(v object.Object) string {
	switch x := v.(type) {
	case nil:
		return ""
	case *object.None:
		return "nothing (none)"
	case *object.String:
		n := utf8.RuneCountInString(x.Value)
		if n <= 40 && !strings.ContainsAny(x.Value, "\n\r") {
			return fmt.Sprintf("%q", x.Value)
		}
		r := []rune(strings.Join(strings.Fields(x.Value), " "))
		if len(r) > 30 {
			r = r[:30]
		}
		return fmt.Sprintf("%q... (%d characters)", string(r), n)
	case *object.List:
		return fmt.Sprintf("list of %d: %s", len(x.Elements), preview(x))
	case *object.Set:
		return fmt.Sprintf("set of %d: %s", len(x.Elements), preview(x))
	case *object.Map:
		return fmt.Sprintf("map of %d: %s", x.Len(), preview(x))
	}
	return preview(v)
}

// preview is a value as it shows, on one line, cut short past 60
// characters.
func preview(v object.Object) string {
	s := strings.Join(strings.Fields(shortInspect(v, 80)), " ")
	if r := []rune(s); len(r) > 60 {
		s = string(r[:57]) + "..."
	}
	return s
}

// shortInspect is v as it shows, but for a list, set or map only as many
// items as fit in about limit characters: a preview of a big one costs
// little.
func shortInspect(v object.Object, limit int) string {
	var items []object.Object
	open, close := "[ ", " ]"
	switch x := v.(type) {
	case *object.List:
		items = x.Elements
	case *object.Set:
		items, open, close = x.Elements, "{ ", " }"
	case *object.Map:
		var b strings.Builder
		b.WriteString("{ ")
		for i, e := range x.Entries() {
			if i > 0 {
				b.WriteString(", ")
			}
			b.WriteString(shortInspect(e.Key, limit) + ": " + shortInspect(e.Val, limit))
			if b.Len() > limit {
				return b.String() + ", ..."
			}
		}
		return b.String() + " }"
	default:
		return object.Shown(v) // text quoted, as inside a list
	}
	if len(items) == 0 {
		return v.Inspect()
	}
	var b strings.Builder
	b.WriteString(open)
	for i, e := range items {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(shortInspect(e, limit))
		if b.Len() > limit && i < len(items)-1 {
			return b.String() + ", ..."
		}
	}
	return b.String() + close
}

// report writes the steps' record as a table, under a heading.
func (run *scrollRun) report(heading string, start object.Object, result object.Object, finished bool) string {
	var b strings.Builder
	b.WriteString(heading + "\n")
	width := 0
	for _, r := range run.records {
		if w := utf8.RuneCountInString(strings.Repeat("  ", r.depth) + r.num + " " + shortLabel(r.label)); w > width {
			width = w
		}
	}
	startText := run.startShown
	if startText == "" {
		startText = briefValue(start)
	}
	fmt.Fprintf(&b, "  %-*s   %s\n", width, "start", startText)
	for _, r := range run.records {
		name := strings.Repeat("  ", r.depth) + r.num + " " + shortLabel(r.label)
		switch r.status {
		case stepReturned:
			shown := r.shown
			if shown == "" {
				shown = briefValue(r.value)
			}
			fmt.Fprintf(&b, "  %-*s → returned %s\n", width, name, shown)
		case stepNone:
			fmt.Fprintf(&b, "  %-*s → returned nothing (none)   is none expected!?\n", width, name)
		case stepNotReached:
			fmt.Fprintf(&b, "  %-*s   not reached\n", width, name)
		default:
			fmt.Fprintf(&b, "  %-*s ✗ failed: %s\n", width, name, r.err)
			if r.noneFrom != "" {
				fmt.Fprintf(&b, "  %-*s   (it was given none, from %s)\n", width, "", r.noneFrom)
			}
		}
	}
	if finished {
		fmt.Fprintf(&b, "  %-*s   %s\n", width, "result", briefValue(result))
	}
	return strings.TrimRight(b.String(), "\n")
}

func shortLabel(s string) string {
	if r := []rune(s); len(r) > 32 {
		return string(r[:29]) + "..."
	}
	return s
}

// traceStep writes a step's line under turtle trace.
func (it *Interpreter) traceStep(r stepRecord) {
	if it.Trace == nil {
		return
	}
	it.traceClose()
	indent := strings.Repeat("  ", r.depth)
	switch r.status {
	case stepReturned:
		fmt.Fprintf(it.Trace, "          %sstep %s %s → returned %s\n", indent, r.num, shortLabel(r.label), briefValue(r.value))
	case stepNone:
		fmt.Fprintf(it.Trace, "          %sstep %s %s → returned nothing (none)   is none expected!?\n", indent, r.num, shortLabel(r.label))
	case stepFailed:
		fmt.Fprintf(it.Trace, "          %sstep %s %s ✗ failed: %s\n", indent, r.num, shortLabel(r.label), r.err)
		if r.noneFrom != "" {
			fmt.Fprintf(it.Trace, "          %s  (it was given none, from %s)\n", indent, r.noneFrom)
		}
	}
}

// ---- diagnose ----------------------------------------------------------

// diagnose shows where a scroll or a function went wrong, without
// stopping the program: it shows what happened and gives back what it
// ended with.
//
//	x is diagnose[s, 3] .                    // a saved scroll on 3
//	x is s diagnose 3 .                      // the same, as a sentence
//	x is diagnose[scroll 3 into a, b .] .    // a scroll written there
//	y is diagnose[double, 4] .               // a function: given, returned, time
//	diagnose[e]                              // an error: its message and steps
//
// A scroll gives back its result (none too, if that's what it ends
// with), or the error, as a value like the one handle gives, when a step
// failed: the scroll ends at that step, since the steps after it depend
// on it. A function gives back its result or its error. An error gives back
// itself. Only Turtle errors are caught: exit[] and Ctrl-C go on.
func (it *Interpreter) evalDiagnose(ce *ast.CallExpression, env *object.Environment) object.Object {
	if len(ce.Arguments) == 0 {
		fatalf("diagnose needs a scroll and a value, a scroll written in it, a function and its values, or an error")
	}
	if tc, ok := ce.Arguments[0].(*ast.TheoryCall); ok && len(ce.Arguments) == 1 {
		return it.diagnoseTheoryUse(tc, env)
	}
	if se, ok := ce.Arguments[0].(*ast.ScrollExpression); ok && se.Start != nil && len(ce.Arguments) == 1 {
		return it.diagnoseScroll("diagnose scroll", se.Steps, env, it.evalExpression(se.Start, env))
	}
	// diagnose[discount]: a theory, this file's or an imported one.
	if id, ok := ce.Arguments[0].(*ast.Identifier); ok && id.Module == "" && len(ce.Arguments) == 1 {
		if fn := theoryIn(id.Value, env); fn != nil {
			fmt.Println(it.proveTheory(fn, theorySeed(it.Global)).report())
			return fn
		}
	}
	// diagnose[load, text]: a library function, by name.
	if id, ok := ce.Arguments[0].(*ast.Identifier); ok {
		_, isVar := env.Get(id.Value)
		_, isDef := env.GetFunction(id.Value)
		if !isVar && !isDef && id.Module == "" || id.Module != "" {
			args := make([]object.Object, len(ce.Arguments)-1)
			for i, a := range ce.Arguments[1:] {
				args[i] = it.evalExpression(a, env)
			}
			call := &ast.CallExpression{Token: ce.Token, Module: id.Module, Name: id.Value}
			return it.diagnoseCall(id.Value, args, func() object.Object { return it.applyCall(call, args, env) })
		}
	}
	args := make([]object.Object, len(ce.Arguments))
	for i, a := range ce.Arguments {
		args[i] = it.evalExpression(a, env)
	}
	return it.diagnoseValues(args)
}

// diagnoseValues is diagnose on values already worked out.
func (it *Interpreter) diagnoseValues(args []object.Object) object.Object {
	if len(args) == 0 {
		fatalf("diagnose needs a scroll and a value, a scroll written in it, a function and its values, or an error")
	}
	switch v := args[0].(type) {
	case *object.Error:
		if len(args) != 1 {
			fatalf("diagnose of an error takes just the error, got %d values", len(args))
		}
		out := fmt.Sprintf("diagnose error (line %d)\n  %s %s", currentLine, v.Kind+":", v.Message)
		if v.Steps != nil && len(v.Steps.Elements) > 0 {
			run := &scrollRun{}
			for _, e := range v.Steps.Elements {
				m := e.(*object.Map)
				get := func(k string) object.Object { x, _ := m.Get(&object.String{Value: k}); return x }
				r := stepRecord{num: get("step").Inspect(), label: get("name").Inspect(), status: get("status").Inspect(), value: get("returned"), noneFrom: get("nonefrom").Inspect()}
				r.depth = strings.Count(r.num, ".")
				if r.status == stepFailed {
					r.err = "(see the message above)"
				}
				run.records = append(run.records, r)
			}
			out += "\n" + strings.SplitN(run.report("", nil, nil, false), "\n", 3)[2]
		}
		fmt.Println(out)
		return v
	case *object.Function:
		if v.Theory != nil && len(args) == 1 {
			fmt.Println(it.proveTheory(v, theorySeed(it.Global)).report())
			return v
		}
		if v.Scroll != nil {
			if len(args) != 2 {
				fatalf("diagnose of a scroll takes the scroll and the value to run it on: diagnose[%s, value], got %d values", scrollName(v), len(args))
			}
			return it.diagnoseScroll("diagnose "+scrollShown(v), v.Scroll.Steps, scrollEnv(v, it), args[1])
		}
		name := v.Name
		if name == "" {
			name = "function"
		}
		return it.diagnoseCall(name, args[1:], func() object.Object { return it.callFunction(v, name, args[1:]) })
	}
	fatalf("diagnose needs a scroll, a function or an error, got %s", aValue(args[0]))
	return nil
}

func scrollShown(fn *object.Function) string {
	if fn.Name == "" {
		return "scroll"
	}
	return fn.Name
}

// diagnoseScroll runs steps on value and shows each step's result: its
// value, none (marked), or the error it failed with. It gives back the
// result, or the error.
func (it *Interpreter) diagnoseScroll(heading string, steps []*ast.ScrollStep, env *object.Environment, value object.Object) (result object.Object) {
	heading = fmt.Sprintf("%s (line %d)", heading, currentLine)
	run := &scrollRun{from: "the start", snapshot: true}
	run.startShown = briefValue(value) // as it was, before any step changes it
	start := value
	defer func() {
		if r := recover(); r != nil {
			fe, ok := r.(fatalError)
			if !ok || fe.parse {
				panic(r)
			}
			fmt.Println(run.report(heading, start, nil, false))
			result = it.errorValue(fe)
			return
		}
		fmt.Println(run.report(heading, start, result, true))
	}()
	return it.runScroll(run, steps, env, value)
}

// diagnoseCall runs a function, then shows what it was given, what it
// gave back, and how long it took.
func (it *Interpreter) diagnoseCall(name string, args []object.Object, call func() object.Object) (result object.Object) {
	var b strings.Builder
	fmt.Fprintf(&b, "diagnose %s (line %d)\n", name, currentLine)
	for i, a := range args {
		label := "given"
		if len(args) > 1 {
			label = fmt.Sprintf("given %d", i+1)
		}
		fmt.Fprintf(&b, "  %-9s %s\n", label, briefValue(a))
	}
	start := time.Now()
	defer func() {
		took := fmt.Sprintf("  %-9s %s", "took", fmtDuration(time.Since(start)))
		if r := recover(); r != nil {
			fe, ok := r.(fatalError)
			if !ok || fe.parse {
				panic(r)
			}
			fmt.Fprintf(&b, "  %-9s ✗ %s error: %s\n%s", "failed", fe.kind, fe.text, took)
			fmt.Println(b.String())
			result = it.errorValue(fe)
			return
		}
		fmt.Fprintf(&b, "  %-9s %s\n%s", "returned", briefValue(result), took)
		fmt.Println(b.String())
	}()
	return call()
}

// lastStep is a step's own number: "2" in "1.2".
func lastStep(num string) string {
	return num[strings.LastIndex(num, ".")+1:]
}

// scrollFail stops the program with a scroll's own error (kind scroll),
// whose message already says which step.
func scrollFail(format string, args ...any) {
	text := fmt.Sprintf(format, args...)
	panic(fatalError{msg: place(currentFile, currentLine) + text, text: text, kind: kindScroll, line: currentLine, file: currentFile, scrolled: true})
}

// ---- the diagnose block -------------------------------------------------

// maxDiagnoseLines is how many lines a diagnose block shows before it
// only counts the rest (a long loop).
const maxDiagnoseLines = 60

// evalDiagnoseBlock runs a diagnose block, showing each line as it runs,
// the value each assignment gave, and each loop pass; an error in it is
// shown, and the program goes on after diagnose [end].
func (it *Interpreter) evalDiagnoseBlock(s *ast.DiagnoseStatement, env *object.Environment) (res ExecResult) {
	fmt.Printf("diagnose (lines %d-%d)\n", s.Line()+1, s.End-1)
	out := &cappedWriter{max: maxDiagnoseLines}
	prevTrace, prevOpen := it.Trace, it.traceOpen
	if it.traceSrc == nil {
		it.traceSrc = map[string][]string{}
	}
	prevSrc, hadSrc := it.traceSrc[currentFile]
	if !hadSrc {
		it.traceSrc[currentFile] = s.Lines
	}
	it.Trace, it.traceOpen = out, nil
	file := currentFile
	defer func() {
		r := recover()
		it.traceClose()
		it.Trace, it.traceOpen = prevTrace, prevOpen
		if hadSrc {
			it.traceSrc[file] = prevSrc
		} else {
			delete(it.traceSrc, file)
		}
		out.flush()
		if out.dropped > 0 {
			fmt.Printf("  ... %d more lines\n", out.dropped)
		}
		if r != nil {
			fe, ok := r.(fatalError)
			if !ok || fe.parse {
				panic(r)
			}
			fmt.Printf("  ✗ failed: %s error: %s\n", fe.kind, fe.msg)
			res = noneResult
			return
		}
		fmt.Println("  finished")
	}()
	return it.evalBlock(s.Body, env)
}

// cappedWriter writes lines to stdout, two spaces in, up to max of them,
// then only counts.
type cappedWriter struct {
	max, lines, dropped int
	partial             []byte
}

func (c *cappedWriter) Write(b []byte) (int, error) {
	for _, ch := range b {
		c.partial = append(c.partial, ch)
		if ch == '\n' {
			c.emit()
		}
	}
	return len(b), nil
}

func (c *cappedWriter) emit() {
	if c.lines < c.max {
		os.Stdout.Write(append([]byte("  "), c.partial...))
		c.lines++
	} else {
		c.dropped++
	}
	c.partial = c.partial[:0]
}

func (c *cappedWriter) flush() {
	if len(c.partial) > 0 {
		c.partial = append(c.partial, '\n')
		c.emit()
	}
}
