package evaluator

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	"Turtle/ast"
	"Turtle/object"
)

// The test library (import test): check, verify and validate, and the
// settings turtle test reads. A failed check, verify or validate is an
// error of kind test: inside turtle test it fails that test; elsewhere
// it stops the program like any error (safe ... handle [test] catches it).

const (
	suiteName     = "suite"
	benchmarkName = "benchmark"
	runsName      = "runs"
	benchtimeName = "benchtime"
	casesName     = "cases"
	defaultCases  = 100
)

// defineTestSettings gives the importing file its test settings, unless
// it already has variables of those names.
func defineTestSettings(env *object.Environment) {
	set := func(name string, v object.Object) {
		if _, ok := env.Get(name); !ok {
			env.Set(name, v)
		}
	}
	set(suiteName, &object.Boolean{Value: false})
	set(benchmarkName, &object.Boolean{Value: false})
	set(runsName, object.NoneValue)
	set(benchtimeName, &object.Integer{Value: 1})
	set(casesName, &object.Integer{Value: defaultCases})
	defineSeed(env)
}

// testFail stops with a test error: what was written, then the details.
func testFail(text string, details []string) {
	var b strings.Builder
	b.WriteString("failed: " + text)
	for _, d := range details {
		b.WriteString("\n    " + d)
	}
	fatalKind(kindTest, "%s", b.String())
}

// tryEval evaluates e, returning the error it stopped with instead of
// stopping (system's exit[] still exits).
func (it *Interpreter) tryEval(e ast.Expression, env *object.Environment) (v object.Object, fe *fatalError) {
	line, file := currentLine, currentFile
	defer func() {
		if r := recover(); r != nil {
			if f, ok := r.(fatalError); ok && !f.parse {
				fe = &f
				currentLine, currentFile = line, file
				return
			}
			panic(r)
		}
	}()
	return it.evalExpression(e, env), nil
}

// ---- check ----

func (it *Interpreter) evalCheck(c *ast.CheckStatement, env *object.Environment) {
	switch c.Form {
	case "true":
		if ok, details := it.explainTruth(c.Value, env); !ok {
			testFail(c.Text, details)
		}
	case "is":
		v := it.evalExpression(c.Value, env)
		if it.isKind(v, c.Kind, env) == c.Negate {
			subject := exprText(c.Value) + " is " + object.Shown(v) + ", "
			if isLiteral(c.Value) {
				subject = exprText(c.Value) + " is "
			}
			if c.Negate {
				testFail(c.Text, []string{fmt.Sprintf("%s%s", subject, kindWithArticle(c.Kind))})
			}
			testFail(c.Text, []string{fmt.Sprintf("%s%s, not %s", subject, valueKind(v), kindWithArticle(c.Kind))})
		}
	case "close":
		v := it.evalExpression(c.Value, env)
		other := it.evalExpression(c.Other, env)
		a, _, ok1 := numeric(v)
		b, _, ok2 := numeric(other)
		if !ok1 || !ok2 {
			fatalf("check ... is close to: needs two numbers, got %s and %s", object.Shown(v), object.Shown(other))
		}
		tol := -1.0
		if c.Within != nil {
			w := it.evalExpression(c.Within, env)
			t, _, ok := numeric(w)
			if !ok || t < 0 {
				fatalf("check ... within: needs a number of 0 or more, got %s", object.Shown(w))
			}
			tol = t
		}
		if closeEnough(a, b, tol) == c.Negate {
			allowed := "a tiny rounding difference"
			if tol >= 0 {
				allowed = (&object.Float{Value: tol}).Inspect()
			}
			diff := (&object.Float{Value: a - b}).Inspect()
			if c.Negate {
				testFail(c.Text, []string{fmt.Sprintf("got %s, which is within %s of %s", v.Inspect(), allowed, other.Inspect())})
			}
			testFail(c.Text, []string{fmt.Sprintf("got %s, want %s (off by %s; allowed: %s)", object.Exact(v), object.Exact(other), diff, allowed)})
		}
	case "fails":
		v, fe := it.tryEval(c.Value, env)
		if fe == nil {
			testFail(c.Text, []string{fmt.Sprintf("expected an error, but it gave %s", object.Shown(v))})
		}
		if len(c.Errors) > 0 && !containsString(c.Errors, fe.kind) {
			testFail(c.Text, []string{fmt.Sprintf("expected a %s error, got a %s error: %s", strings.Join(c.Errors, " or "), fe.kind, fe.text)})
		}
	}
}

// kindWithArticle writes a check's kind word as words: "an integer".
func kindWithArticle(kind string) string {
	switch kind {
	case "none", "empty":
		return kind
	case "int":
		kind = "integer"
	case "bool":
		kind = "boolean"
	}
	if strings.ContainsRune("AEIOUaeiou", rune(kind[0])) {
		return "an " + kind
	}
	return "a " + kind
}

func containsString(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

// isKind reports whether v is of the kind named in "check x is <kind>".
func (it *Interpreter) isKind(v object.Object, kind string, env *object.Environment) bool {
	switch kind {
	case "integer", "int":
		_, ok := v.(*object.Integer)
		return ok
	case "float":
		_, ok := v.(*object.Float)
		return ok
	case "number":
		_, _, ok := numeric(v)
		return ok
	case "string":
		_, ok := v.(*object.String)
		return ok
	case "boolean", "bool":
		_, ok := v.(*object.Boolean)
		return ok
	case "list":
		_, ok := v.(*object.List)
		return ok
	case "set":
		_, ok := v.(*object.Set)
		return ok
	case "map":
		_, ok := v.(*object.Map)
		return ok
	case "date":
		_, ok := v.(*object.Date)
		return ok
	case "none":
		_, ok := v.(*object.None)
		return ok
	case "function":
		f, ok := v.(*object.Function)
		return ok && f.Shape == nil
	case "empty":
		n, ok := lengthOf(v)
		return ok && n == 0
	}
	shape := findShape(env, kind)
	if shape == nil {
		fatalKind(kindName, "%s isn't a kind (integer, float, number, string, boolean, list, set, map, date, none, function, empty) or an assembled type", kind)
	}
	a, ok := v.(*object.Assembly)
	return ok && a.Shape == shape
}

// findShape is assembledShape without the error: nil if there's no
// assembled type called name.
func findShape(env *object.Environment, name string) *object.Shape {
	var fn *object.Function
	if v, ok := env.Get(name); ok {
		fn, _ = v.(*object.Function)
	}
	if fn == nil {
		fn, _ = env.GetFunction(name)
	}
	if fn == nil {
		if im := resolveImported(env, name); im != nil {
			fn, _ = im.Module.Function(name)
		}
	}
	if fn == nil {
		return nil
	}
	return fn.Shape
}

// explainTruth works out e, which must be true or false, and when it's
// false says why: how the two sides of == differ, or what the parts of a
// method or function call were. Each part is worked out once.
func (it *Interpreter) explainTruth(e ast.Expression, env *object.Environment) (bool, []string) {
	var v object.Object
	var details []string
	switch x := e.(type) {
	case *ast.InfixExpression:
		if x.Operator == "&&" || x.Operator == "||" {
			lok, ld := it.explainTruth(x.Left, env)
			rok, rd := it.explainTruth(x.Right, env)
			if x.Operator == "&&" {
				if lok && rok {
					return true, nil
				}
				if !lok {
					details = append(details, prefixed(exprText(x.Left)+" is false", ld)...)
				}
				if !rok {
					details = append(details, prefixed(exprText(x.Right)+" is false", rd)...)
				}
				return false, details
			}
			if lok || rok {
				return true, nil
			}
			details = append(prefixed(exprText(x.Left)+" is false", ld), prefixed(exprText(x.Right)+" is false", rd)...)
			return false, details
		}
		l := it.evalExpression(x.Left, env)
		r := it.evalExpression(x.Right, env)
		v = evalInfix(x.Operator, l, r)
		switch x.Operator {
		case "==":
			details = diffLines(l, r)
		case "!=":
			details = []string{fmt.Sprintf("both are %s", object.Shown(l))}
		default:
			details = []string{fmt.Sprintf("%s is %s, which isn't %s %s", exprText(x.Left), object.Shown(l), x.Operator, object.Shown(r))}
			if !isLiteral(x.Right) {
				details = append(details, fmt.Sprintf("%s is %s", exprText(x.Right), object.Shown(r)))
			}
		}
	case *ast.PrefixExpression:
		if x.Operator == "!" {
			ok, _ := it.explainTruth(x.Right, env)
			if ok {
				return false, []string{exprText(x.Right) + " is true"}
			}
			return true, nil
		}
		v = it.evalExpression(e, env)
	case *ast.MethodCallExpression:
		recv := it.evalExpression(x.Receiver, env)
		args := make([]object.Object, len(x.Arguments))
		for i, a := range x.Arguments {
			args[i] = it.evalExpression(a, env)
		}
		v = it.applyMethod(x, recv, args, env)
		details = append(details, fmt.Sprintf("%s is %s", exprText(x.Receiver), object.Shown(recv)))
		for i, a := range x.Arguments {
			if !isLiteral(a) {
				details = append(details, fmt.Sprintf("%s is %s", exprText(a), object.Shown(args[i])))
			}
		}
	case *ast.CallExpression:
		args := make([]object.Object, len(x.Arguments))
		for i, a := range x.Arguments {
			args[i] = it.evalExpression(a, env)
		}
		v = it.applyCall(x, args, env)
		for i, a := range x.Arguments {
			if !isLiteral(a) {
				details = append(details, fmt.Sprintf("%s is %s", exprText(a), object.Shown(args[i])))
			}
		}
	default:
		v = it.evalExpression(e, env)
		if !isLiteral(e) {
			details = []string{fmt.Sprintf("%s is %s", exprText(e), object.Shown(v))}
		}
	}
	b, ok := v.(*object.Boolean)
	if !ok {
		fatalf("check needs something true or false, but %s is %s; to compare it, write check %s == ... .", exprText(e), object.Shown(v), exprText(e))
	}
	if b.Value {
		return true, nil
	}
	return false, details
}

func prefixed(head string, lines []string) []string {
	out := []string{head}
	for _, l := range lines {
		out = append(out, "  "+l)
	}
	return out
}

// ---- verify ----

func (it *Interpreter) evalVerify(v *ast.VerifyStatement, env *object.Environment) {
	coll := it.evalExpression(v.Collection, env)
	if ok, details := it.checkRule(v.Rule, coll, env); !ok {
		testFail(v.Text, details)
	}
}

// ruleItem is one thing a rule is tried on.
type ruleItem struct {
	label string // "at 3", "at key \"a\"", "at 2 and 3", or "" for a set
	args  []object.Object
}

func (r ruleItem) String() string {
	parts := make([]string, len(r.args))
	for i, a := range r.args {
		parts[i] = object.Shown(a)
	}
	if r.label == "" {
		return strings.Join(parts, ", ")
	}
	return r.label + ": " + strings.Join(parts, ", ")
}

// ruleItems are what a rule is tried on: a list's or set's items, a
// string's characters, a map's values (or key and value, for a rule of
// two names), or neighbors two at a time (pair).
func ruleItems(coll object.Object, fn *object.Function, pair bool) []ruleItem {
	n := len(fn.Parameters)
	var elems []object.Object
	var labels []string
	switch c := coll.(type) {
	case *object.List:
		elems = c.Elements
		for i := range elems {
			labels = append(labels, fmt.Sprintf("at %d", i))
		}
	case *object.Set:
		elems = c.Elements
		labels = make([]string, len(elems))
	case *object.String:
		for _, r := range c.Value {
			elems = append(elems, &object.String{Value: string(r)})
		}
		for i := range elems {
			labels = append(labels, fmt.Sprintf("at %d", i))
		}
	case *object.Map:
		if pair {
			fatalf("each pair works on a list, set or string, in order; a map has no neighbors")
		}
		var out []ruleItem
		for _, k := range c.Keys {
			key := c.KeyOf(k)
			label := "at key " + object.Shown(key)
			switch n {
			case 1:
				out = append(out, ruleItem{label, []object.Object{c.Values[k]}})
			case 2:
				out = append(out, ruleItem{label, []object.Object{key, c.Values[k]}})
			default:
				fatalf("a rule for a map takes one name (the value) or two ([key, value] give ...), got %d", n)
			}
		}
		return out
	default:
		fatalf("the rule needs a list, set, map or string to go through, got %s", valueKind(coll))
	}
	if pair {
		if n != 2 {
			fatalf("each pair needs a rule of two names: each pair [a, b] give a <= b")
		}
		var out []ruleItem
		for i := 0; i+1 < len(elems); i++ {
			label := fmt.Sprintf("at %d and %d", i, i+1)
			if labels[i] == "" {
				label = ""
			}
			out = append(out, ruleItem{label, []object.Object{elems[i], elems[i+1]}})
		}
		return out
	}
	if n != 1 {
		fatalf("the rule takes %d names; for one item at a time use one (x give ...), for neighbors write each pair [a, b] give ...", n)
	}
	out := make([]ruleItem, len(elems))
	for i, e := range elems {
		out[i] = ruleItem{labels[i], []object.Object{e}}
	}
	return out
}

// checkRule tries the rule on every item and counts the ones that follow
// it; false, with the reason, when there are too many or too few.
func (it *Interpreter) checkRule(r *ast.Rule, coll object.Object, env *object.Environment) (bool, []string) {
	f := it.evalExpression(r.Fn, env)
	fn, ok := f.(*object.Function)
	if !ok || fn.Shape != nil {
		fatalf("the rule must be a function, e.g. x give x > 0, or a function's name; %s is %s", exprText(r.Fn), object.Shown(f))
	}
	items := ruleItems(coll, fn, r.Pair)
	var follow, breaks []ruleItem
	for _, item := range items {
		res := it.callFunction(fn, "rule", item.args)
		b, ok := res.(*object.Boolean)
		if !ok {
			fatalf("the rule must give true or false; for %s it gave %s", item, object.Shown(res))
		}
		if b.Value {
			follow = append(follow, item)
		} else {
			breaks = append(breaks, item)
		}
	}
	unit := "item"
	if r.Pair {
		unit = "pair"
	}
	total := len(items)
	count := -1
	if r.Count != nil {
		cv := it.evalExpression(r.Count, env)
		n, ok := cv.(*object.Integer)
		if !ok || n.Value < 0 {
			fatalf("how many must follow the rule: a whole number of 0 or more, got %s", object.Shown(cv))
		}
		count = int(n.Value)
	}
	listed := func(head string, items []ruleItem) []string {
		out := []string{head}
		for i, item := range items {
			if i == 10 {
				out = append(out, fmt.Sprintf("  ... and %d more", len(items)-10))
				break
			}
			out = append(out, "  "+item.String())
		}
		return out
	}
	follows := fmt.Sprintf("%d of %d %s follow the rule", len(follow), total, plural(total, unit))
	if len(follow) == 1 {
		follows = fmt.Sprintf("1 of %d %s follows the rule", total, plural(total, unit))
	}
	switch r.Quant {
	case "each":
		if len(breaks) == 0 {
			return true, nil
		}
		return false, listed(fmt.Sprintf("%d of %d %s %s the rule:", len(breaks), total, plural(total, unit), verbFor(len(breaks), "breaks", "break")), breaks)
	case "any":
		if len(follow) > 0 {
			return true, nil
		}
		if total == 0 {
			return false, []string{"there are no " + unit + "s, so none follows the rule"}
		}
		return false, []string{fmt.Sprintf("none of the %d %s follows the rule", total, plural(total, unit))}
	case "not":
		if len(follow) == 0 {
			return true, nil
		}
		return false, listed(follows+", and none should:", follow)
	case "atleast":
		if len(follow) >= count {
			return true, nil
		}
		return false, listed(fmt.Sprintf("%s; at least %d should", follows, count)+colonIf(len(follow) > 0), follow)
	case "atmost":
		if len(follow) <= count {
			return true, nil
		}
		return false, listed(fmt.Sprintf("%s; at most %d should:", follows, count), follow)
	case "exactly":
		if len(follow) == count {
			return true, nil
		}
		return false, listed(fmt.Sprintf("%s; exactly %d should", follows, count)+colonIf(len(follow) > 0), follow)
	}
	fatalf("unknown rule %q", r.Quant)
	return false, nil
}

func verbFor(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}

func colonIf(b bool) string {
	if b {
		return ":"
	}
	return ""
}

// ---- validate ----

func (it *Interpreter) evalValidate(v *ast.ValidateStatement, env *object.Environment) {
	inputs := v.Inputs
	learned := false
	if len(inputs) == 0 {
		inputs = it.learnInputs(v, env)
		learned = true
	}
	cases := defaultCases
	if c, ok := env.Get(casesName); ok {
		n, isInt := c.(*object.Integer)
		if !isInt || n.Value < 1 {
			fatalf("cases (how many random inputs validate tries) must be a whole number of 1 or more, got %s", object.Shown(c))
		}
		cases = int(n.Value)
	}
	var seed int64
	if s, ok := env.Get(seedName); ok {
		switch x := s.(type) {
		case *object.Integer:
			seed = x.Value
		case *object.None:
			seed = rand.New(rand.NewSource(time.Now().UnixNano())).Int63n(1_000_000_000)
		default:
			fatalf("seed must be a whole number or none, got %s", object.Shown(s))
		}
	} else {
		seed = rand.New(rand.NewSource(time.Now().UnixNano())).Int63n(1_000_000_000)
	}
	rng := rand.New(rand.NewSource(seed))

	for n := 1; n <= cases; n++ {
		vals := make([]object.Object, len(inputs))
		for i, in := range inputs {
			vals[i] = it.randomValue(in.Shape, env, rng)
		}
		if failed, _, _ := it.validateRun(v, inputs, vals, env); !failed {
			continue
		}
		original := vals
		vals = it.shrink(inputs, vals, env, func(try []object.Object) bool {
			failed, _, _ := it.validateRun(v, inputs, try, env)
			return failed
		})
		_, details, result := it.validateRun(v, inputs, vals, env)
		lines := []string{fmt.Sprintf("failed on case %d of %d (seed %d)", n, cases, seed)}
		for i, in := range inputs {
			line := fmt.Sprintf("%s = %s", in.Name, object.Shown(vals[i]))
			if !valuesEqual(vals[i], original[i]) {
				line += fmt.Sprintf("   (smallest found; first failed on %s)", object.Shown(original[i]))
			}
			lines = append(lines, line)
		}
		if result != nil {
			lines = append(lines, fmt.Sprintf("%s = %s", v.ResultName, object.Shown(result)))
		}
		lines = append(lines, details...)
		if learned {
			lines = append(lines, "inputs made like the ones in this test's checks")
		}
		lines = append(lines, fmt.Sprintf("to repeat this run: seed = %d", seed))
		testFail(v.Text, lines)
	}
}

// validateRun calls the function on one set of inputs and tries the rule
// on its answer: true (with why) when the rule fails or the call stops
// with an error.
func (it *Interpreter) validateRun(v *ast.ValidateStatement, inputs []*ast.ValidateInput, vals []object.Object, env *object.Environment) (failed bool, details []string, result object.Object) {
	// The function gets its own copy of each input, and so does the rule:
	// a function that changes its list in place (keep, sort, add ...)
	// mustn't change what the rule sees, or the next try.
	callEnv := object.NewEnclosedEnvironment(env)
	child := object.NewEnclosedEnvironment(env)
	for i, in := range inputs {
		callEnv.Set(in.Name, deepCopy(vals[i]))
		child.Set(in.Name, deepCopy(vals[i]))
	}
	res, fe := it.tryEval(v.Call, callEnv)
	if fe != nil {
		return true, []string{"it stopped with an error: " + fe.text}, nil
	}
	child.Set(v.ResultName, res)
	var ok bool
	var why []string
	stopped := it.protect(func() {
		switch {
		case v.That != nil:
			ok, why = it.explainTruth(v.That, child)
			if !ok {
				why = append([]string{"the rule is false: " + exprText(v.That)}, why...)
			}
		case v.Rule != nil:
			coll := it.evalExpression(v.Collection, child)
			ok, why = it.checkRule(v.Rule, coll, child)
		case v.Matches != nil:
			other := it.evalExpression(v.Matches, child)
			ok = valuesEqual(res, other)
			if !ok {
				why = append([]string{fmt.Sprintf("%s gave %s", exprText(v.Matches), object.Shown(other))}, diffLines(res, other)...)
			}
		}
	})
	if stopped != nil {
		return true, []string{"the rule stopped with an error: " + stopped.text}, res
	}
	return !ok, why, res
}

// deepCopy copies a value and everything inside it.
func deepCopy(v object.Object) object.Object {
	switch x := v.(type) {
	case *object.List:
		out := &object.List{Elements: make([]object.Object, len(x.Elements))}
		for i, e := range x.Elements {
			out.Elements[i] = deepCopy(e)
		}
		return out
	case *object.Set:
		out := &object.Set{}
		for _, e := range x.Elements {
			out.Add(deepCopy(e))
		}
		return out
	case *object.Map:
		out := object.NewMap()
		for _, k := range x.Keys {
			out.Put(deepCopy(x.KeyOf(k)), deepCopy(x.Values[k]))
		}
		return out
	case *object.Assembly:
		vals := make([]object.Object, len(x.Values))
		for i, e := range x.Values {
			vals[i] = deepCopy(e)
		}
		return &object.Assembly{Shape: x.Shape, Values: vals}
	}
	return v
}

// protect runs f, returning the error it stopped with, if any.
func (it *Interpreter) protect(f func()) (fe *fatalError) {
	line, file := currentLine, currentFile
	defer func() {
		if r := recover(); r != nil {
			if e, ok := r.(fatalError); ok && !e.parse {
				fe = &e
				currentLine, currentFile = line, file
				return
			}
			panic(r)
		}
	}()
	f()
	return nil
}

// learnInputs works out what kind of inputs to make from the test's
// checks on the same function: check evens[list [1, 2, 3]] == ... means
// nums is a list of integers.
func (it *Interpreter) learnInputs(v *ast.ValidateStatement, env *object.Environment) []*ast.ValidateInput {
	how := fmt.Sprintf("say what each input is, e.g. validate %s with %s as list of integer that ...", exprText(v.Call), placeholderName(v.Call))
	var names []string
	for _, a := range v.Call.Arguments {
		id, ok := a.(*ast.Identifier)
		if !ok || id.Module != "" {
			fatalf("validate %s: to make inputs, each argument must be a name, like nums", exprText(v.Call))
		}
		names = append(names, id.Value)
	}
	for _, ex := range v.Examples {
		inputs := make([]*ast.ValidateInput, len(names))
		ok := true
		for i, a := range ex.Arguments {
			val, fe := it.tryEval(a, env)
			if fe != nil {
				ok = false
				break
			}
			shape := shapeOf(val)
			if shape == nil {
				ok = false
				break
			}
			inputs[i] = &ast.ValidateInput{Name: names[i], Shape: shape}
		}
		if ok {
			return inputs
		}
	}
	if len(v.Examples) == 0 {
		fatalf("validate %s: there's no check on %s in this test to learn its inputs from; %s", exprText(v.Call), v.Call.Name, how)
	}
	fatalf("validate %s: couldn't tell the kind of input from the checks (empty collections, none, or mixed kinds); %s", exprText(v.Call), how)
	return nil
}

func placeholderName(c *ast.CallExpression) string {
	if len(c.Arguments) > 0 {
		return exprText(c.Arguments[0])
	}
	return "x"
}

// shapeOf is the shape a value is an example of: list [1, 2] is a list
// of integer. nil if it can't tell (none, an empty list, mixed kinds).
func shapeOf(v object.Object) *ast.Shape {
	switch x := v.(type) {
	case *object.Integer:
		return &ast.Shape{Kind: "integer"}
	case *object.Float:
		return &ast.Shape{Kind: "float"}
	case *object.String:
		return &ast.Shape{Kind: "string"}
	case *object.Boolean:
		return &ast.Shape{Kind: "boolean"}
	case *object.Date:
		return &ast.Shape{Kind: "date"}
	case *object.List:
		if item := commonShape(x.Elements); item != nil {
			return &ast.Shape{Kind: "list", Item: item}
		}
	case *object.Set:
		if item := commonShape(x.Elements); item != nil {
			return &ast.Shape{Kind: "set", Item: item}
		}
	case *object.Map:
		var keys, vals []object.Object
		for _, k := range x.Keys {
			keys = append(keys, x.KeyOf(k))
			vals = append(vals, x.Values[k])
		}
		key, val := commonShape(keys), commonShape(vals)
		if key != nil && val != nil {
			return &ast.Shape{Kind: "map", Key: key, Item: val}
		}
	case *object.Assembly:
		s := &ast.Shape{Kind: "assembled", TypeName: x.Shape.Name}
		for _, f := range x.Values {
			fs := shapeOf(f)
			if fs == nil {
				return nil
			}
			s.Fields = append(s.Fields, fs)
		}
		return s
	}
	return nil
}

// commonShape is the shape every one of vals fits: integers and floats
// together are floats.
func commonShape(vals []object.Object) *ast.Shape {
	var out *ast.Shape
	for _, v := range vals {
		s := shapeOf(v)
		if s == nil {
			continue
		}
		switch {
		case out == nil:
			out = s
		case out.Describe() == s.Describe():
		case out.Kind == "integer" && s.Kind == "float", out.Kind == "float" && s.Kind == "integer":
			out = &ast.Shape{Kind: "float"}
		default:
			return nil
		}
	}
	return out
}

// typeName is typeof's answer: "integer", "list", "Order" ...
func typeName(v object.Object) string {
	switch x := v.(type) {
	case *object.Integer:
		return "integer"
	case *object.Float:
		return "float"
	case *object.String:
		return "string"
	case *object.Boolean:
		return "boolean"
	case *object.List:
		return "list"
	case *object.Set:
		return "set"
	case *object.Map:
		return "map"
	case *object.Date:
		return "date"
	case *object.None:
		return "none"
	case *object.Function:
		if x.Shape != nil {
			return "assembled type"
		}
		return "function"
	case *object.Assembly:
		return x.Shape.Name
	case *object.Error:
		return "error"
	case *object.Database:
		return "database"
	}
	return strings.ToLower(string(v.Type()))
}
