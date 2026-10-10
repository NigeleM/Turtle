package evaluator

import (
	"fmt"
	"io"
	"math/rand"
	"regexp"
	"sort"
	"strings"
	"time"

	"Turtle/ast"
	"Turtle/object"
)

// Proving a theory, for turtle test and diagnose[theory]: each proof case
// must give what it says, and each theorem must hold for the proof cases'
// results and for random inputs made like the proof cases' values.

// theoryProof is what checking one theory found.
type theoryProof struct {
	ts       *ast.TheoryStatement
	file     string // where it's written, "" for the file being run
	cases    []caseCheck
	theorems []theoremCheck
	tried    int   // random inputs tried
	seed     int64 // the random inputs' seed, to repeat a run
	warnings []string
}

type caseCheck struct {
	pc   *ast.ProofCase
	ok   bool
	what string // when it isn't: "gave 200, expected 1800", or the error
}

type theoremCheck struct {
	th    *ast.Theorem
	holds bool
	why   string // when it doesn't: the input and why
}

// failed reports whether a proof case or a theorem failed.
func (p theoryProof) failed() bool {
	for _, c := range p.cases {
		if !c.ok {
			return true
		}
	}
	for _, t := range p.theorems {
		if !t.holds {
			return true
		}
	}
	return false
}

// proveTheory checks a theory: its proof, then its theorems on the proof
// cases and on random inputs.
func (it *Interpreter) proveTheory(fn *object.Function, seed int64) theoryProof {
	ts := fn.Theory
	pr := theoryProof{ts: ts, file: fn.Env.File(), seed: seed}
	if len(ts.Proofs) == 0 {
		pr.warnings = append(pr.warnings, "unproven: it has no proof")
	}
	if !mentionsWord(ts.Abstract, ts.Name) {
		pr.warnings = append(pr.warnings, fmt.Sprintf("its abstract doesn't say what %s is: name %s in it", ts.Name, ts.Name))
	}

	// The proof cases: each gives what it says.
	var examples [][]object.Object // each case's values, for the theorems and for the random inputs' kinds
	var results []object.Object
	for _, pc := range ts.Proofs {
		c := caseCheck{pc: pc, ok: true}
		var args []object.Object
		var got, want object.Object
		fe := it.protect(func() {
			args = it.theoryArgs(pc.Use, fn.Env)
			got = it.callFunction(fn, ts.Name, copies(args))
			want = it.evalExpression(pc.Want, fn.Env)
		})
		switch {
		case fe != nil:
			c.ok, c.what = false, "stopped with "+aKind(fe.kind)+" error: "+fe.text
		case !valuesEqual(got, want):
			c.ok, c.what = false, fmt.Sprintf("gave %s, expected %s", object.Shown(got), object.Shown(want))
		default:
			examples = append(examples, args)
			results = append(results, got)
		}
		pr.cases = append(pr.cases, c)
	}

	// The theorems, on the proof cases' results first.
	for _, th := range ts.Theorems {
		tc := theoremCheck{th: th, holds: true}
		for i, args := range examples {
			if holds, why := it.checkTheorem(th, theoremEnv(fn, args, results[i])); !holds {
				tc.holds, tc.why = false, "on "+describeArgs(ts, args)+": "+why
				break
			}
		}
		pr.theorems = append(pr.theorems, tc)
	}

	// Then on random inputs, made like the proof cases' values.
	inputs := theoryInputs(ts, examples)
	if len(ts.Theorems) == 0 || inputs == nil {
		return pr
	}
	rng := rand.New(rand.NewSource(seed))
	for n := 0; n < defaultCases; n++ {
		vals := make([]object.Object, len(inputs))
		for i, in := range inputs {
			vals[i] = it.randomValue(in.Shape, fn.Env, rng)
		}
		pr.tried++
		for k, th := range ts.Theorems {
			if !pr.theorems[k].holds {
				continue
			}
			fails := func(try []object.Object) bool {
				holds, _, ran := it.theoremOn(fn, th, try)
				return ran && !holds
			}
			if !fails(vals) {
				continue
			}
			vals = it.shrink(inputs, vals, fn.Env, fails)
			_, why, _ := it.theoremOn(fn, th, vals)
			pr.theorems[k].holds, pr.theorems[k].why = false, "on "+describeArgs(ts, vals)+": "+why
		}
	}
	return pr
}

// theoremOn runs the theory on args and checks th on its result. ran is
// false when the theory itself stopped with an error on them (a random
// input it doesn't take is no counterexample).
func (it *Interpreter) theoremOn(fn *object.Function, th *ast.Theorem, args []object.Object) (holds bool, why string, ran bool) {
	var result object.Object
	if fe := it.protect(func() { result = it.callFunction(fn, fn.Theory.Name, copies(args)) }); fe != nil {
		return true, "", false
	}
	holds, why = it.checkTheorem(th, theoremEnv(fn, args, result))
	return holds, why, true
}

// theoryInputs are the random inputs' kinds, one per value, learned from
// the proof cases; nil when a value's kind can't be learned.
func theoryInputs(ts *ast.TheoryStatement, examples [][]object.Object) []*ast.ValidateInput {
	if len(examples) == 0 {
		return nil
	}
	var inputs []*ast.ValidateInput
	for i, name := range ts.Slots {
		var vals []object.Object
		for _, args := range examples {
			vals = append(vals, args[i])
		}
		shape := commonShape(vals)
		if shape == nil {
			return nil
		}
		inputs = append(inputs, &ast.ValidateInput{Name: name, Shape: shape})
	}
	return inputs
}

// theoryArgs works out a use's values, in the order of the theory's slots.
func (it *Interpreter) theoryArgs(tc *ast.TheoryCall, env *object.Environment) []object.Object {
	args := make([]object.Object, len(tc.Args))
	for i, a := range tc.Args {
		args[i] = object.NoneValue
		if a != nil {
			args[i] = it.evalExpression(a, env)
		}
	}
	return args
}

func copies(args []object.Object) []object.Object {
	out := make([]object.Object, len(args))
	for i, a := range args {
		out[i] = deepCopy(a)
	}
	return out
}

// describeArgs is "a = [ ], b = 4".
func describeArgs(ts *ast.TheoryStatement, args []object.Object) string {
	parts := make([]string, len(ts.Slots))
	for i, name := range ts.Slots {
		parts[i] = name + " = " + object.Shown(args[i])
	}
	return strings.Join(parts, ", ")
}

// mentionsWord reports whether text has word in it, as a word.
func mentionsWord(text, word string) bool {
	return regexp.MustCompile(`(^|[^A-Za-z0-9_~])` + regexp.QuoteMeta(word) + `([^A-Za-z0-9_]|$)`).MatchString(text)
}

// theoriesToProve are the theories a test file checks: its own, and those
// of the files it imports, in the order they're written.
func theoriesToProve(env *object.Environment) []*object.Function {
	var out []*object.Function
	add := func(fns []*object.Function) {
		var mine []*object.Function
		for _, fn := range fns {
			if fn.Theory != nil {
				mine = append(mine, fn)
			}
		}
		sort.Slice(mine, func(i, j int) bool { return mine[i].Theory.Token.Line < mine[j].Theory.Token.Line })
		out = append(out, mine...)
	}
	add(env.Functions())
	for _, im := range env.Imports() {
		if im.Module.Env != nil && !im.Module.Hybrid {
			add(im.Module.Env.Functions())
		}
	}
	return out
}

// report is the proof as diagnose[theory] shows it.
func (p theoryProof) report() string {
	var b strings.Builder
	where := fmt.Sprintf("line %d", p.ts.Token.Line)
	if lib, ok := strings.CutPrefix(p.file, builtinFilePrefix); ok {
		where = "the " + strings.TrimSuffix(lib, ".turtle") + " library, " + where
	} else if p.file != "" {
		where = p.file + " " + where
	}
	fmt.Fprintf(&b, "diagnose theory %s (%s)\n", p.ts.Name, where)
	for _, n := range p.ts.Notations {
		fmt.Fprintf(&b, "  notation  %s\n", n.Text())
	}
	width := 0
	for _, c := range p.cases {
		width = max(width, len(c.pc.Text))
	}
	for _, t := range p.theorems {
		width = max(width, len(t.th.Text))
	}
	if len(p.cases) > 0 {
		b.WriteString("  proof\n")
		for _, c := range p.cases {
			mark := "✓"
			if !c.ok {
				mark = "✗ " + c.what
			}
			fmt.Fprintf(&b, "    %-*s  %s\n", width, c.pc.Text, mark)
		}
	}
	if len(p.theorems) > 0 {
		on := "on the proof cases"
		if p.tried > 0 {
			on += fmt.Sprintf(" and %d random inputs (seed %d)", p.tried, p.seed)
		}
		fmt.Fprintf(&b, "  theorems, %s\n", on)
		for _, t := range p.theorems {
			mark := "holds"
			if !t.holds {
				mark = "✗ fails " + t.why
			}
			fmt.Fprintf(&b, "    %-*s  %s\n", width, t.th.Text, mark)
		}
	}
	for _, w := range p.warnings {
		fmt.Fprintf(&b, "  warning: %s\n", w)
	}
	return strings.TrimRight(b.String(), "\n")
}

// theorySeed is the random inputs' seed: the program's seed setting if it
// has one, else a new one each run.
func theorySeed(env *object.Environment) int64 {
	if s, ok := env.Get(seedName); ok {
		if n, ok := s.(*object.Integer); ok {
			return n.Value
		}
	}
	return rand.New(rand.NewSource(time.Now().UnixNano())).Int63n(1_000_000_000)
}

// writeTheories proves a test file's theories and reports each in a line,
// with what failed under it.
func (it *Interpreter) writeTheories(out io.Writer, name string, env *object.Environment) testCounts {
	var counts testCounts
	fns := theoriesToProve(env)
	width := 0
	for _, fn := range fns {
		width = max(width, len("theory "+fn.Name))
	}
	seed := theorySeed(env)
	for _, fn := range fns {
		p := it.proveTheory(fn, seed)
		label := fmt.Sprintf("%-*s", width, "theory "+fn.Name)
		where := name
		if p.file != "" {
			where = p.file
		}
		if p.failed() {
			counts.theoriesFailed++
			fmt.Fprintf(out, "  FAIL  theory %s\n", fn.Name)
			for _, c := range p.cases {
				if !c.ok {
					fmt.Fprintf(out, "        %s line %d: proof: %s . %s\n", where, c.pc.Line, strings.SplitN(c.pc.Text, " . is ", 2)[0], c.what)
				}
			}
			for _, t := range p.theorems {
				if !t.holds {
					fmt.Fprintf(out, "        %s line %d: theorem %s fails %s\n", where, t.th.Line, t.th.Text, t.why)
				}
			}
			if p.tried > 0 {
				fmt.Fprintf(out, "        to repeat this run: seed = %d\n", p.seed)
			}
		} else {
			counts.theoriesProven++
			summary := fmt.Sprintf("proof %d of %d", len(p.cases), len(p.cases))
			if len(p.cases) == 0 {
				summary = "no proof"
			}
			if len(p.theorems) > 0 {
				summary += fmt.Sprintf(", %d %s", len(p.theorems), verbFor(len(p.theorems), "theorem holds", "theorems hold"))
				if p.tried > 0 {
					summary += fmt.Sprintf(" on %d random inputs", p.tried)
				} else {
					summary += " on the proof cases"
				}
			}
			fmt.Fprintf(out, "  PASS  %s   %s\n", label, summary)
		}
		for _, w := range p.warnings {
			counts.theoryWarnings++
			fmt.Fprintf(out, "  WARN  %s   %s\n", label, w)
		}
	}
	return counts
}
