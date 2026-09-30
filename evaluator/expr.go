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

	case *ast.Identifier:
		val, ok := env.Get(e.Value)
		if !ok {
			fatalf("undefined variable %q", e.Value)
		}
		return val

	case *ast.PrefixExpression:
		right := it.evalExpression(e.Right, env)
		return evalPrefix(e.Operator, right)

	case *ast.InfixExpression:
		left := it.evalExpression(e.Left, env)
		right := it.evalExpression(e.Right, env)
		return evalInfix(e.Operator, left, right)

	case *ast.CallExpression:
		return it.evalCall(e, env)

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
			key := it.evalExpression(k, env).Inspect()
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
	lf, _, lIsNum := numeric(left)
	rf, _, rIsNum := numeric(right)
	if lIsNum && rIsNum {
		return lf == rf
	}
	return left.Inspect() == right.Inspect() && left.Type() == right.Type()
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

// evalCall runs a user-defined function in a fresh, isolated call
// environment (fixing the legacy bug of one shared mutable scope per
// function definition, which broke recursion).
func (it *Interpreter) evalCall(ce *ast.CallExpression, env *object.Environment) object.Object {
	fn, ok := env.GetFunction(ce.Name)
	if !ok {
		if result, handled := it.callBuiltin(ce, env); handled {
			return result
		}
		fatalf("undefined function %q", ce.Name)
	}
	if len(ce.Arguments) != len(fn.Parameters) {
		fatalf("function %q expects %d argument(s), got %d", ce.Name, len(fn.Parameters), len(ce.Arguments))
	}
	callEnv := env.NewCallEnvironment()
	for i, param := range fn.Parameters {
		callEnv.Set(param, it.evalExpression(ce.Arguments[i], env))
	}
	res := it.evalBlock(fn.Body, callEnv)
	if res.Signal == SigReturn {
		return res.Value
	}
	return &object.Integer{Value: 0}
}

// callBuiltin handles "now[]" and "sleep[amount [, unit]]" — the
// functions unlocked by "import time" (see builtinModules in
// evaluator.go). Only reached when no user-defined function of that name
// exists, so a user's own function of the same name always takes
// precedence and is never shadowed.
func (it *Interpreter) callBuiltin(ce *ast.CallExpression, env *object.Environment) (object.Object, bool) {
	switch ce.Name {
	case "now":
		requireModule(it, "time", "now")
		if len(ce.Arguments) != 0 {
			fatalf("'now' expects 0 arguments, got %d", len(ce.Arguments))
		}
		return &object.Integer{Value: time.Now().UnixMilli()}, true
	case "sleep":
		requireModule(it, "time", "sleep")
		it.evalSleep(ce, env)
		return &object.Integer{Value: 0}, true
	}
	return nil, false
}

// evalSleep implements "sleep[amount]" / "sleep[amount, unit]". unit is
// "seconds" (the default, when omitted) or "ms". Arguments are evaluated
// exactly once each — evaluating twice would double any side effect, e.g.
// a function call with output.
func (it *Interpreter) evalSleep(ce *ast.CallExpression, env *object.Environment) {
	if len(ce.Arguments) < 1 || len(ce.Arguments) > 2 {
		fatalf("'sleep' expects 1 or 2 arguments (amount [, unit]), got %d", len(ce.Arguments))
	}
	amount, _, ok := numeric(it.evalExpression(ce.Arguments[0], env))
	if !ok {
		fatalf("'sleep' amount must be a number")
	}
	unit := "seconds"
	if len(ce.Arguments) == 2 {
		u, ok := it.evalExpression(ce.Arguments[1], env).(*object.String)
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
