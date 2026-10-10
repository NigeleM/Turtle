package evaluator

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"Turtle/ast"
	"Turtle/object"
	"Turtle/syntax"
)

// Theories: a theory's word becomes a function of its notation's values,
// which its phrases call (see parser/theory.go for how they're read).

// coreWords are the functions every program has, which a theory's word
// can't be.
var coreWords = []string{"typeof", "help", "stdlib", "version", "diagnose"}

// defineTheory makes the theory's word, which must be new.
func (it *Interpreter) defineTheory(s *ast.TheoryStatement, env *object.Environment) {
	if !env.IsTopLevel() {
		fatalf("theory %s: a theory is written at the top level of a file, not inside other code", s.Name)
	}
	if where := existingWord(s.Name, env); where != "" {
		fatalKind(kindName, "%s is already a word in Turtle (%s): a theory needs a new word", s.Name, where)
	}
	env.Capture()
	env.DefineFunction(&object.Function{Name: s.Name, Parameters: s.Slots, Body: s.Definition, Env: env, Theory: s})
}

// existingWord says what name already is, or "" if it's new.
func existingWord(name string, env *object.Environment) string {
	for _, m := range builtinModules {
		if env.File() == builtinFilePrefix+m.Name+".turtle" && slices.Contains(m.Funcs, name) {
			continue // the library's own theory, which made the name its word
		}
		if slices.Contains(m.Funcs, name) || slices.Contains(m.Methods, name) {
			return "a function of the " + m.Name + " library"
		}
	}
	if slices.Contains(syntax.Methods, name) || oldMethodNames[name] != "" {
		return "a method"
	}
	if slices.Contains(coreWords, name) {
		return "a function every program has"
	}
	if fn, ok := env.GetFunction(name); ok {
		if fn.Theory != nil {
			return "a theory"
		}
		return "a function in this file"
	}
	if _, ok := env.Get(name); ok {
		return "a variable in this file"
	}
	return ""
}

// evalTheoryCall runs a phrase: the theory's definition, given the values.
func (it *Interpreter) evalTheoryCall(tc *ast.TheoryCall, env *object.Environment) object.Object {
	fn := it.theoryNamed(tc.Word, env)
	args := make([]object.Object, len(tc.Args))
	for i, a := range tc.Args {
		if a == nil {
			args[i] = object.NoneValue
			continue
		}
		args[i] = it.evalExpression(a, env)
	}
	return it.callFunction(fn, tc.Word, args)
}

// theoryNamed finds the theory word stands for.
func (it *Interpreter) theoryNamed(word string, env *object.Environment) *object.Function {
	if fn := theoryIn(word, env); fn != nil {
		return fn
	}
	fatalKind(kindName, "%s is a theory's word, but its theory isn't defined here", word)
	return nil
}

// theoremEnv is where a theorem is worked out: the theory's own file, its
// values by their names, and result.
func theoremEnv(fn *object.Function, args []object.Object, result object.Object) *object.Environment {
	env := object.NewEnclosedEnvironment(fn.Env)
	for i, name := range fn.Parameters {
		env.Set(name, args[i])
	}
	env.Set("result", result)
	return env
}

// checkTheorem works out a theorem for one use: whether it holds, and if
// not, why ("result is 5, which isn't <= 4"). An error in it is a fail.
func (it *Interpreter) checkTheorem(th *ast.Theorem, env *object.Environment) (holds bool, why string) {
	var details []string
	if fe := it.protect(func() { holds, details = it.explainTruth(th.Expr, env) }); fe != nil {
		return false, "it stopped with " + aKind(fe.kind) + " error: " + fe.text
	}
	return holds, strings.Join(details, "; ")
}

func aKind(kind string) string {
	if strings.ContainsRune("aeiou", rune(kind[0])) {
		return "an " + kind
	}
	return "a " + kind
}

// diagnoseTheoryUse is diagnose[tally 1 in nums]: the phrase's values,
// what it gave back, whether each theorem holds for it, and how long it
// took. It gives back the result, or the error.
func (it *Interpreter) diagnoseTheoryUse(tc *ast.TheoryCall, env *object.Environment) (result object.Object) {
	fn := it.theoryNamed(tc.Word, env)
	ts := fn.Theory
	args := make([]object.Object, len(tc.Args))
	for i, a := range tc.Args {
		args[i] = object.NoneValue
		if a != nil {
			args[i] = it.evalExpression(a, env)
		}
	}
	var b strings.Builder
	fmt.Fprintf(&b, "diagnose theory %s (line %d)\n", ts.Name, currentLine)
	fmt.Fprintf(&b, "  %-9s %s\n", "notation", ts.Notations[0].Text())
	for i, name := range ts.Slots {
		fmt.Fprintf(&b, "  %-9s %s\n", name, briefValue(deepCopy(args[i])))
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
		fmt.Fprintf(&b, "  %-9s %s\n", "returned", briefValue(result))
		width := 0
		for _, th := range ts.Theorems {
			width = max(width, len(th.Text))
		}
		for _, th := range ts.Theorems {
			holds, why := it.checkTheorem(th, theoremEnv(fn, args, result))
			mark := "holds"
			if !holds {
				mark = "✗ fails: " + why
			}
			fmt.Fprintf(&b, "  %-9s %-*s  %s\n", "theorem", width, th.Text, mark)
		}
		b.WriteString(took)
		fmt.Println(b.String())
	}()
	return it.callFunction(fn, tc.Word, args)
}

// theoryHelp is help for a theory: its abstract, how it's written, and
// what it promises.
func theoryHelp(ts *ast.TheoryStatement) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s        (a theory, line %d)\n", ts.Name, ts.Token.Line)
	for _, l := range strings.Split(ts.Abstract, "\n") {
		b.WriteString("  " + l + "\n")
	}
	for _, n := range ts.Notations {
		b.WriteString("  Written: " + n.Text() + "\n")
	}
	for _, th := range ts.Theorems {
		b.WriteString("  Theorem: " + th.Text + "\n")
	}
	return strings.TrimRight(b.String(), "\n")
}

// theoryIn is the theory word names here, this file's or an imported
// one, or nil.
func theoryIn(word string, env *object.Environment) *object.Function {
	if fn, ok := env.GetFunction(word); ok && fn.Theory != nil {
		return fn
	}
	if im := resolveImported(env, word); im != nil {
		if fn, ok := im.Module.Function(word); ok && fn.Theory != nil {
			return fn
		}
	}
	return nil
}
