package evaluator

import (
	"strconv"
	"strings"

	"Turtle/ast"
)

// exprText writes an expression back as Turtle, for test messages that
// name the parts of what was checked ("nums at get[0] is 4").
func exprText(e ast.Expression) string {
	switch x := e.(type) {
	case nil:
		return ""
	case *ast.Identifier:
		if x.Module != "" {
			return x.Module + " " + x.Value
		}
		return x.Value
	case *ast.IntegerLiteral:
		return strconv.FormatInt(x.Value, 10)
	case *ast.FloatLiteral:
		return x.Token.Literal
	case *ast.StringLiteral:
		return strconv.Quote(x.Value)
	case *ast.InterpolatedString:
		return `"` + x.Token.Literal + `"`
	case *ast.BooleanLiteral:
		return strconv.FormatBool(x.Value)
	case *ast.NoneLiteral:
		return "none"
	case *ast.PrefixExpression:
		s := x.Operator + exprText(x.Right)
		if x.Grouped {
			return "(" + s + ")"
		}
		return s
	case *ast.InfixExpression:
		s := exprText(x.Left) + " " + x.Operator + " " + exprText(x.Right)
		if x.Grouped {
			return "(" + s + ")"
		}
		return s
	case *ast.CallExpression:
		name := x.Name
		if x.Module != "" {
			name = x.Module + " " + name
		}
		if x.Subject != nil {
			return exprText(x.Subject) + " " + name + " " + exprList(x.Arguments)
		}
		return name + "[" + exprList(x.Arguments) + "]"
	case *ast.MethodCallExpression:
		s := exprText(x.Receiver) + " at " + x.Method
		if x.Bracketed {
			return s + "[" + exprList(x.Arguments) + "]"
		}
		if len(x.Arguments) > 0 {
			return s + " " + exprList(x.Arguments)
		}
		return s
	case *ast.FieldExpression:
		return x.Field + " of " + exprText(x.Object)
	case *ast.ListLiteral:
		return "list [" + exprList(x.Elements) + "]"
	case *ast.SetLiteral:
		return "set [" + exprList(x.Elements) + "]"
	case *ast.MapLiteral:
		parts := make([]string, len(x.Keys))
		for i := range x.Keys {
			parts[i] = exprText(x.Keys[i]) + ": " + exprText(x.Values[i])
		}
		return "map [" + strings.Join(parts, ", ") + "]"
	case *ast.FunctionLiteral:
		params := strings.Join(x.Parameters, ", ")
		if len(x.Parameters) != 1 {
			params = "[" + params + "]"
		}
		if x.Body != nil && len(x.Body.Statements) == 1 {
			if r, ok := x.Body.Statements[0].(*ast.ReturnStatement); ok {
				return params + " gives " + exprText(r.Value)
			}
		}
		return params + " gives ..."
	case *ast.MinExpression:
		return "min of " + exprText(x.Arg)
	case *ast.MaxExpression:
		return "max of " + exprText(x.Arg)
	case *ast.LengthExpression:
		return "length of " + exprText(x.Arg)
	case *ast.ChangeExpression:
		return "change " + exprText(x.Source) + " to " + x.TypeName
	case *ast.RandomExpression:
		return "random " + x.Shape.Describe()
	}
	return e.TokenLiteral()
}

func exprList(es []ast.Expression) string {
	parts := make([]string, len(es))
	for i, e := range es {
		parts[i] = exprText(e)
	}
	return strings.Join(parts, ", ")
}

// isLiteral: showing this part's value would only repeat what's written.
func isLiteral(e ast.Expression) bool {
	switch e.(type) {
	case *ast.IntegerLiteral, *ast.FloatLiteral, *ast.StringLiteral, *ast.BooleanLiteral, *ast.NoneLiteral:
		return true
	case *ast.PrefixExpression:
		p := e.(*ast.PrefixExpression)
		return p.Operator == "-" && isLiteral(p.Right)
	}
	return false
}
