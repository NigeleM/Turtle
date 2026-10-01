package evaluator

import (
	"math"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"Turtle/ast"
	"Turtle/object"
)

func (it *Interpreter) evalExpression(expr ast.Expression, env *object.Environment) object.Object {
	switch e := expr.(type) {
	case *ast.IntegerLiteral:
		return &object.Integer{Value: e.Value}
	case *ast.FloatLiteral:
		return &object.Float{Value: e.Value}
	case *ast.StringLiteral:
		return &object.String{Value: e.Value}
	case *ast.BooleanLiteral:
		return &object.Boolean{Value: e.Value}
	case *ast.NoneLiteral:
		return object.NoneValue

	case *ast.Identifier:
		// A variable, or failing that a function by name — so "f = add"
		// or "apply[mylib binary, 3]" passes the function itself.
		// "a b" is either module a's function b (as a value) or the
		// sentence-style call b[a] with no further arguments.
		if e.Module != "" {
			if subject, ok := sentenceSubject(env, e.Module); ok {
				return it.callByName(e.Value, []object.Object{subject}, env)
			}
			return importedFunctionValue(qualifiedImport(env, e.Module, e.Value), e.Value)
		}
		if val, ok := env.Get(e.Value); ok {
			return val
		}
		if fn, ok := env.GetFunction(e.Value); ok {
			return fn
		}
		if im := resolveImported(env, e.Value); im != nil {
			return importedFunctionValue(im, e.Value)
		}
		fatalf("undefined variable %q", e.Value)
		return nil

	case *ast.PrefixExpression:
		right := it.evalExpression(e.Right, env)
		return evalPrefix(e.Operator, right)

	case *ast.InfixExpression:
		left := it.evalExpression(e.Left, env)
		right := it.evalExpression(e.Right, env)
		return evalInfix(e.Operator, left, right)

	case *ast.CallExpression:
		return it.evalCall(e, env)

	case *ast.FunctionLiteral:
		return &object.Function{Parameters: e.Parameters, Body: e.Body, Env: env}

	case *ast.FieldExpression:
		a, i := it.assemblyField(e, env)
		return a.Values[i]

	case *ast.ListLiteral:
		list := &object.List{}
		for _, el := range e.Elements {
			list.Elements = append(list.Elements, it.evalExpression(el, env))
		}
		return list

	case *ast.SetLiteral:
		set := &object.Set{}
		for _, el := range e.Elements {
			v := it.evalExpression(el, env)
			if !set.Contains(v) {
				set.Elements = append(set.Elements, v)
			}
		}
		return set

	case *ast.MapLiteral:
		m := object.NewMap()
		for i, k := range e.Keys {
			key := object.Key(it.evalExpression(k, env))
			m.Set(key, it.evalExpression(e.Values[i], env))
		}
		return m

	case *ast.MethodCallExpression:
		return it.evalMethodCall(e, env)

	case *ast.MinExpression:
		return it.reduceExtreme(e.Arg, env, true)
	case *ast.MaxExpression:
		return it.reduceExtreme(e.Arg, env, false)
	case *ast.LengthExpression:
		return &object.Integer{Value: int64(it.lengthOf(it.evalExpression(e.Arg, env)))}
	case *ast.ChangeExpression:
		return evalChange(e.TypeName, it.evalExpression(e.Source, env))

	default:
		fatalf("no evaluator for expression type %T", expr)
		return nil
	}
}

func evalPrefix(op string, right object.Object) object.Object {
	switch op {
	case "-":
		switch v := right.(type) {
		case *object.Integer:
			return &object.Integer{Value: -v.Value}
		case *object.Float:
			return &object.Float{Value: -v.Value}
		}
		fatalf("unary '-' needs a number, got %s", right.Type())
	case "!":
		return &object.Boolean{Value: !isTruthy(right)}
	}
	fatalf("unknown prefix operator %q", op)
	return nil
}

func numeric(obj object.Object) (float64, bool, bool) {
	switch v := obj.(type) {
	case *object.Integer:
		return float64(v.Value), true, true
	case *object.Float:
		return v.Value, false, true
	}
	return 0, false, false
}

func evalInfix(op string, left, right object.Object) object.Object {
	if op == "+" || op == "-" {
		if res, ok := collectionOp(op, left, right); ok {
			return res
		}
	}
	// '+' overloads to string concatenation whenever either side isn't a
	// number (matches the legacy dual behavior).
	if op == "+" {
		lf, lIsInt, lIsNum := numeric(left)
		rf, rIsInt, rIsNum := numeric(right)
		if lIsNum && rIsNum {
			if lIsInt && rIsInt {
				return &object.Integer{Value: left.(*object.Integer).Value + right.(*object.Integer).Value}
			}
			return &object.Float{Value: lf + rf}
		}
		return &object.String{Value: left.Inspect() + right.Inspect()}
	}

	lf, lIsInt, lIsNum := numeric(left)
	rf, rIsInt, rIsNum := numeric(right)

	switch op {
	case "-", "*", "/", "%":
		if !lIsNum || !rIsNum {
			fatalf("operator %q needs two numbers, got %s and %s", op, left.Type(), right.Type())
		}
		bothInt := lIsInt && rIsInt
		switch op {
		case "-":
			if bothInt {
				return &object.Integer{Value: left.(*object.Integer).Value - right.(*object.Integer).Value}
			}
			return &object.Float{Value: lf - rf}
		case "*":
			if bothInt {
				return &object.Integer{Value: left.(*object.Integer).Value * right.(*object.Integer).Value}
			}
			return &object.Float{Value: lf * rf}
		case "/":
			if bothInt {
				r := right.(*object.Integer).Value
				if r == 0 {
					fatalf("division by zero")
				}
				return &object.Integer{Value: left.(*object.Integer).Value / r}
			}
			if rf == 0 {
				fatalf("division by zero")
			}
			return &object.Float{Value: lf / rf}
		case "%":
			if bothInt {
				r := right.(*object.Integer).Value
				if r == 0 {
					fatalf("modulo by zero")
				}
				return &object.Integer{Value: left.(*object.Integer).Value % r}
			}
			if rf == 0 {
				fatalf("modulo by zero")
			}
			return &object.Float{Value: math.Mod(lf, rf)}
		}
	case "<", ">", "<=", ">=":
		if lIsNum && rIsNum {
			return &object.Boolean{Value: compareNum(op, lf, rf)}
		}
		ls, lok := left.(*object.String)
		rs, rok := right.(*object.String)
		if lok && rok {
			return &object.Boolean{Value: compareStr(op, ls.Value, rs.Value)}
		}
		fatalf("operator %q needs two numbers or two strings, got %s and %s", op, left.Type(), right.Type())
	case "==":
		return &object.Boolean{Value: valuesEqual(left, right)}
	case "!=":
		return &object.Boolean{Value: !valuesEqual(left, right)}
	case "&&":
		return &object.Boolean{Value: isTruthy(left) && isTruthy(right)}
	case "||":
		return &object.Boolean{Value: isTruthy(left) || isTruthy(right)}
	}
	fatalf("unknown infix operator %q", op)
	return nil
}

func compareNum(op string, l, r float64) bool {
	switch op {
	case "<":
		return l < r
	case ">":
		return l > r
	case "<=":
		return l <= r
	case ">=":
		return l >= r
	}
	return false
}

func compareStr(op string, l, r string) bool {
	switch op {
	case "<":
		return l < r
	case ">":
		return l > r
	case "<=":
		return l <= r
	case ">=":
		return l >= r
	}
	return false
}

func valuesEqual(left, right object.Object) bool {
	return object.Equal(left, right)
}

// evalChange converts val to typeName ("integer", "float", "string",
// "ascii", "char", or "hex"). The direction between a pair of types (e.g.
// integer <-> hex string) is inferred from val's actual type, so one target
// name covers both directions.
func evalChange(typeName string, val object.Object) object.Object {
	switch typeName {
	case "integer":
		switch v := val.(type) {
		case *object.Integer:
			return v
		case *object.Float:
			return &object.Integer{Value: int64(v.Value)}
		case *object.String:
			n, err := strconv.ParseInt(strings.TrimSpace(v.Value), 10, 64)
			if err != nil {
				fatalf("change ... to integer: %q is not a valid integer", v.Value)
			}
			return &object.Integer{Value: n}
		}
	case "float":
		switch v := val.(type) {
		case *object.Float:
			return v
		case *object.Integer:
			return &object.Float{Value: float64(v.Value)}
		case *object.String:
			f, err := strconv.ParseFloat(strings.TrimSpace(v.Value), 64)
			if err != nil {
				fatalf("change ... to float: %q is not a valid float", v.Value)
			}
			return &object.Float{Value: f}
		}
	case "string":
		if s, ok := val.(*object.String); ok {
			return s
		}
		return &object.String{Value: val.Inspect()}
	case "ascii":
		s, ok := val.(*object.String)
		if !ok {
			fatalf("change ... to ascii needs a single-character string, got %s", val.Type())
		}
		runes := []rune(s.Value)
		if len(runes) != 1 {
			fatalf("change ... to ascii needs exactly one character, got %q", s.Value)
		}
		return &object.Integer{Value: int64(runes[0])}
	case "char":
		n, ok := val.(*object.Integer)
		if !ok {
			fatalf("change ... to char needs an integer code point, got %s", val.Type())
		}
		if n.Value < 0 || n.Value > utf8.MaxRune {
			fatalf("change ... to char: %d is not a valid code point", n.Value)
		}
		return &object.String{Value: string(rune(n.Value))}
	case "hex":
		switch v := val.(type) {
		case *object.Integer:
			return &object.String{Value: strconv.FormatInt(v.Value, 16)}
		case *object.String:
			s := strings.TrimSpace(v.Value)
			s = strings.TrimPrefix(strings.TrimPrefix(s, "0x"), "0X")
			n, err := strconv.ParseInt(s, 16, 64)
			if err != nil {
				fatalf("change ... to hex: %q is not valid hex", v.Value)
			}
			return &object.Integer{Value: n}
		}
	default:
		fatalf("change: unknown target type %q (want integer, float, string, ascii, char, or hex)", typeName)
	}
	fatalf("change: can't convert %s to %s", val.Type(), typeName)
	return nil
}

// evalCall resolves "name[...]" and runs it. Resolution order:
//  1. a variable holding a function (a parameter, a nested def, an
//     anonymous "gives" function, or a function returned from another
//     call) — a variable holding a non-function doesn't block the lookup,
//     so existing scripts with a variable and a function of the same name
//     keep working;
//  2. a top-level def in the current file — your own function always
//     wins over an imported one of the same name;
//  3. a function from this file's imports, which must be unambiguous:
//     if two imports both provide it, it has to be called qualified
//     ("time now[]").
//
// "a name[...]" / "a name args" is module a's function when a is an
// imported module, or the sentence-style call name[a, args...] when a is a
// variable.
func (it *Interpreter) evalCall(ce *ast.CallExpression, env *object.Environment) object.Object {
	args := make([]object.Object, len(ce.Arguments))
	for i, a := range ce.Arguments {
		args[i] = it.evalExpression(a, env)
	}
	if ce.Module != "" {
		if subject, ok := sentenceSubject(env, ce.Module); ok {
			return it.callByName(ce.Name, append([]object.Object{subject}, args...), env)
		}
		return it.callImported(qualifiedImport(env, ce.Module, ce.Name), ce.Name, args, env)
	}
	return it.callByName(ce.Name, args, env)
}

// sentenceSubject reports whether name, in front of a function name, is
// a variable (so "name verb ..." is a sentence-style call) rather than an
// imported module. A name that's both is refused rather than guessed.
func sentenceSubject(env *object.Environment, name string) (object.Object, bool) {
	v, isVar := env.Get(name)
	if _, isModule := env.FindImport(name); isModule {
		if isVar {
			fatalf("%q is both a variable and an imported module — rename the variable", name)
		}
		return nil, false
	}
	if !isVar {
		fatalf("unknown module or variable %q in front of a function name", name)
	}
	return v, true
}

func (it *Interpreter) callByName(name string, args []object.Object, env *object.Environment) object.Object {
	if v, ok := env.Get(name); ok {
		if fn, ok := v.(*object.Function); ok {
			return it.callFunction(fn, name, args)
		}
	}
	if fn, ok := env.GetFunction(name); ok {
		return it.callFunction(fn, name, args)
	}
	if im := resolveImported(env, name); im != nil {
		return it.callImported(im, name, args, env)
	}
	for _, mod := range builtinModules {
		if mod.ExportsFunction(name) {
			requireModule(env, mod.Name, name)
		}
	}
	fatalf("undefined function %q", name)
	return nil
}

// callFunction runs fn on already-evaluated args in a fresh, isolated
// call environment (fixing the legacy bug of one shared mutable scope per
// function definition, which broke recursion) that encloses the
// function's defining scope, not the caller's — lexical scoping, which is
// what lets a closure see the locals it captured, and an imported
// function its own module's globals.
func (it *Interpreter) callFunction(fn *object.Function, name string, args []object.Object) object.Object {
	if fn.Name != "" {
		name = fn.Name
	}
	if fn.Shape != nil {
		if len(args) != len(fn.Shape.Fields) {
			fatalf("%s needs %d value(s), one per field (%s), got %d",
				name, len(fn.Shape.Fields), strings.Join(fn.Shape.Fields, ", "), len(args))
		}
		return &object.Assembly{Shape: fn.Shape, Values: append([]object.Object{}, args...)}
	}
	if len(args) != len(fn.Parameters) {
		fatalf("function %q expects %d argument(s), got %d", name, len(fn.Parameters), len(args))
	}
	defEnv := fn.Env
	if defEnv == nil {
		defEnv = it.Global
	}
	callEnv := object.NewEnclosedEnvironment(defEnv)
	for i, param := range fn.Parameters {
		callEnv.Set(param, args[i])
	}
	res := it.evalBlock(fn.Body, callEnv)
	if res.Signal == SigReturn {
		return res.Value
	}
	return object.NoneValue
}

func (it *Interpreter) callImported(im *object.Import, name string, args []object.Object, env *object.Environment) object.Object {
	if fn, ok := im.Module.Function(name); ok {
		return it.callFunction(fn, name, args)
	}
	return it.callBuiltin(name, args)
}

// resolveImported finds the import that provides function name to an
// unqualified use, or nil if none does. More than one is a clash: rather
// than guess, it fails and asks for the module to be named.
func resolveImported(env *object.Environment, name string) *object.Import {
	var found []*object.Import
	for _, im := range env.Imports() {
		if im.Allows(name) && im.Module.ExportsFunction(name) {
			found = append(found, im)
		}
	}
	if len(found) <= 1 {
		if len(found) == 1 {
			return found[0]
		}
		return nil
	}
	mods := make([]string, len(found))
	for i, im := range found {
		mods[i] = im.Module.Name
	}
	fatalf("%q is provided by more than one import (%s) — say which one, e.g. %s %s[...]",
		name, strings.Join(mods, ", "), mods[0], name)
	return nil
}

// qualifiedImport resolves "module name" against this file's imports.
func qualifiedImport(env *object.Environment, module, name string) *object.Import {
	im, ok := env.FindImport(module)
	if !ok {
		fatalf("unknown module %q in \"%s %s\" — import it first", module, module, name)
	}
	if !im.Module.ExportsFunction(name) {
		fatalf("module %q has no function %q", module, name)
	}
	if !im.Allows(name) {
		fatalf("%q isn't imported — add it to \"import %s [...]\"", name, module)
	}
	return im
}

func importedFunctionValue(im *object.Import, name string) object.Object {
	fn, ok := im.Module.Function(name)
	if !ok {
		fatalf("%q is a builtin of %q and can only be called, not used as a value", name, im.Module.Name)
	}
	return fn
}

// callBuiltin runs a builtin module function ("now[]", "sleep[amount [,
// unit]]", data's process/keep/copy) once resolution has already
// confirmed it's imported.
func (it *Interpreter) callBuiltin(name string, args []object.Object) object.Object {
	switch name {
	case "now":
		if len(args) != 0 {
			fatalf("'now' expects 0 arguments, got %d", len(args))
		}
		return &object.Integer{Value: time.Now().UnixMilli()}
	case "sleep":
		evalSleep(args)
		return object.NoneValue
	case "process":
		return it.dataProcess(args)
	case "keep":
		return it.dataKeep(args)
	case "copy":
		return dataCopy(args)
	case "args", "exists", "isFile", "isFolder":
		return it.callSystem(name, args)
	}
	fatalf("no builtin function %q", name)
	return nil
}

// evalSleep implements "sleep[amount]" / "sleep[amount, unit]". unit is
// "seconds" (the default, when omitted) or "ms".
func evalSleep(args []object.Object) {
	if len(args) < 1 || len(args) > 2 {
		fatalf("'sleep' expects 1 or 2 arguments (amount [, unit]), got %d", len(args))
	}
	amount, _, ok := numeric(args[0])
	if !ok {
		fatalf("'sleep' amount must be a number")
	}
	unit := "seconds"
	if len(args) == 2 {
		u, ok := args[1].(*object.String)
		if !ok {
			fatalf(`'sleep' unit must be a string, "seconds" or "ms"`)
		}
		unit = u.Value
	}
	switch unit {
	case "seconds":
		time.Sleep(time.Duration(amount * float64(time.Second)))
	case "ms":
		time.Sleep(time.Duration(amount * float64(time.Millisecond)))
	default:
		fatalf(`'sleep' unit must be "seconds" or "ms", got %q`, unit)
	}
}

// assemblyField evaluates the value in "field of <value>" and finds the
// field in it, failing clearly if the value isn't assembled or has no
// such field.
func (it *Interpreter) assemblyField(fe *ast.FieldExpression, env *object.Environment) (*object.Assembly, int) {
	obj := it.evalExpression(fe.Object, env)
	a, ok := obj.(*object.Assembly)
	if !ok {
		fatalf("'%s of' needs an assembled value, got %s", fe.Field, obj.Type())
	}
	i := a.Shape.Index(fe.Field)
	if i < 0 {
		fatalf("%s has no field %q (its fields: %s)", a.Shape.Name, fe.Field, strings.Join(a.Shape.Fields, ", "))
	}
	return a, i
}
