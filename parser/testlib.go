package parser

import (
	"strings"

	"Turtle/ast"
	"Turtle/token"
)

// The test library's sentences (import test). In a file that imports
// test, check, verify and validate begin statements:
//
//	check total[order] == 45 .
//	check x is integer .            check x is not none .
//	check 0.1 + 0.2 is close to 0.3 .
//	check divide[1, 0] fails [math] .
//	verify evens[nums] each x give x % 2 == 0 .
//	verify rolls at least 2 r give r == 6 .
//	verify prices each pair [a, b] give a <= b .
//	validate evens[nums] with nums as list of integer
//	    that result each x give x % 2 == 0 .
//
// Elsewhere the three are ordinary names.

var testWords = map[string]bool{"check": true, "verify": true, "validate": true}

// quantWords begin a rule after a collection (at least / at most start
// with the keyword at, handled in peekIsQuant).
var quantWords = []string{"each", "any", "not", "exactly"}

// pushStops makes words end the expression being parsed: a value right
// before one isn't a sentence-style call ("nums each ..." is nums, then
// the rule).
func (p *Parser) pushStops(words ...string) {
	if p.stopWords == nil {
		p.stopWords = map[string]int{}
	}
	for _, w := range words {
		p.stopWords[w]++
	}
}

func (p *Parser) popStops(words ...string) {
	for _, w := range words {
		p.stopWords[w]--
	}
}

// isStop reports whether lit ends the value being parsed.
func (p *Parser) isStop(lit string) bool {
	return p.stopWords[lit] > 0 || p.inShape && shapeConnectors[lit]
}

// startsTestStatement: curToken is check, verify or validate, in a file
// that imports test.
func (p *Parser) startsTestStatement() bool {
	return p.testImported && p.curTokenIs(token.IDENT) && testWords[p.curToken.Literal]
}

func (p *Parser) parseTestStatement() ast.Statement {
	switch p.curToken.Literal {
	case "check":
		return p.parseCheckStatement()
	case "verify":
		return p.parseVerifyStatement()
	}
	return p.parseValidateStatement()
}

// sourceText is the program's lines from..to, joined, for messages.
func (p *Parser) sourceText(from, to int) string {
	if p.lines == nil {
		p.lines = strings.Split(p.l.Input(), "\n")
	}
	var parts []string
	for n := from; n <= to && n-1 < len(p.lines); n++ {
		if n >= 1 {
			parts = append(parts, strings.TrimSpace(p.lines[n-1]))
		}
	}
	return strings.Join(parts, " ")
}

// endStatement consumes the closing period and records the statement's
// text.
func (p *Parser) endStatement(start int, text *string) bool {
	end := p.peekToken.Line
	if !p.requirePeriod() {
		return false
	}
	*text = p.sourceText(start, end)
	return true
}

func (p *Parser) parseCheckStatement() ast.Statement {
	tok := p.curToken
	p.pushStops("fails")
	p.nextToken()
	value := p.parseExpression(LOWEST)
	p.popStops("fails")
	c := &ast.CheckStatement{Token: tok, Value: value, Form: "true"}
	if !p.parseCheckForm(c) {
		return nil
	}
	if !p.endStatement(tok.Line, &c.Text) {
		return nil
	}
	return c
}

// parseCheckForm reads what comes after a check's value: is <kind>,
// is close to ..., fails [...], or nothing (a true/false value).
func (p *Parser) parseCheckForm(c *ast.CheckStatement) bool {
	switch {
	case p.peekTokenIs(token.IS):
		p.nextToken()
		if p.peekTokenIs(token.IDENT) && p.peekToken.Literal == "not" {
			p.nextToken()
			c.Negate = true
		}
		p.nextToken()
		if p.curToken.Literal == "close" && p.peekTokenIs(token.TO) {
			c.Form = "close"
			p.nextToken() // -> to
			p.nextToken()
			p.pushStops("within")
			c.Other = p.parseExpression(LOWEST)
			p.popStops("within")
			if p.peekTokenIs(token.IDENT) && p.peekToken.Literal == "within" {
				p.nextToken()
				p.nextToken()
				c.Within = p.parseExpression(LOWEST)
			}
			break
		}
		c.Form = "is"
		switch p.curToken.Type {
		case token.LIST:
			c.Kind = "list"
		case token.SET:
			c.Kind = "set"
		case token.MAP:
			c.Kind = "map"
		case token.NONE:
			c.Kind = "none"
		case token.IDENT:
			c.Kind = p.curToken.Literal
		default:
			p.errorf("check ... is: say what kind of value, e.g. check x is integer . (integer, float, number, string, boolean, list, set, map, date, none, function, empty, or an assembled type)")
			return false
		}
	case p.peekTokenIs(token.IDENT) && p.peekToken.Literal == "fails":
		p.nextToken()
		c.Form = "fails"
		if p.peekTokenIs(token.LBRACKET) && p.peekToken.Line == p.curToken.Line {
			p.nextToken() // -> [
			for !p.peekTokenIs(token.RBRACKET) {
				if !p.expectPeek(token.IDENT) {
					p.errorf("check ... fails [...]: list error kinds, e.g. fails [math, type]")
					return false
				}
				c.Errors = append(c.Errors, p.curToken.Literal)
				if p.peekTokenIs(token.COMMA) {
					p.nextToken()
				}
			}
			p.nextToken() // -> ]
		}
	}
	return true
}

// peekIsQuant: peekToken begins a rule (each, any, not, exactly N, at
// least N, at most N).
func (p *Parser) peekIsQuant() bool {
	if p.peekTokenIs(token.AT) {
		lit := p.peekN(2).Literal
		return lit == "least" || lit == "most"
	}
	if !p.peekTokenIs(token.IDENT) {
		return false
	}
	for _, w := range quantWords {
		if p.peekToken.Literal == w {
			return true
		}
	}
	return false
}

// parseRule reads a rule; peekToken is its first word.
func (p *Parser) parseRule(what string) *ast.Rule {
	if !p.peekIsQuant() {
		p.errorf("%s: after the values, say how many must follow the rule: each, any, not, at least N, at most N or exactly N, e.g. %s nums each x give x > 0 .", what, what)
		return nil
	}
	r := &ast.Rule{}
	p.nextToken()
	switch {
	case p.curTokenIs(token.AT):
		p.nextToken()
		r.Quant = "at" + p.curToken.Literal
		p.nextToken()
		r.Count = p.countValue()
	case p.curToken.Literal == "exactly":
		r.Quant = "exactly"
		p.nextToken()
		r.Count = p.countValue()
	default:
		r.Quant = p.curToken.Literal
	}
	if r.Count == nil && r.Quant != "each" && r.Quant != "any" && r.Quant != "not" {
		return nil
	}
	if p.peekTokenIs(token.IDENT) && p.peekToken.Literal == "pair" && p.peekN(2).Type == token.LBRACKET {
		p.nextToken()
		r.Pair = true
	}
	if p.peekTokenIs(token.PERIOD) || p.peekTokenIs(token.EOF) || p.peekTokenIs(token.RBRACKET) {
		p.errorf("%s: give the rule after %q, e.g. x give x > 0", what, p.curToken.Literal)
		return nil
	}
	p.nextToken()
	r.Fn = p.parseExpression(LOWEST)
	return r
}

func (p *Parser) parseVerifyStatement() ast.Statement {
	tok := p.curToken
	p.pushStops(quantWords...)
	p.inVerifyValue++
	p.nextToken()
	coll := p.parseExpression(LOWEST)
	p.inVerifyValue--
	p.popStops(quantWords...)
	v := &ast.VerifyStatement{Token: tok, Collection: coll}
	if v.Rule = p.parseRule("verify"); v.Rule == nil {
		return nil
	}
	if !p.endStatement(tok.Line, &v.Text) {
		return nil
	}
	return v
}

func (p *Parser) parseValidateStatement() ast.Statement {
	tok := p.curToken
	words := []string{"with", "that", "matches", "as"}
	p.pushStops(words...)
	p.nextToken()
	expr := p.parseExpression(LOWEST)
	p.popStops(words...)
	call, ok := expr.(*ast.CallExpression)
	if !ok || call.Subject != nil {
		p.errorf("validate needs a function call to test, e.g. validate evens[nums] with nums as list of integer that ...")
		return nil
	}
	v := &ast.ValidateStatement{Token: tok, Call: call, ResultName: "result"}
	if !p.parseValidateRest(v, "validate") {
		return nil
	}
	if !p.endStatement(tok.Line, &v.Text) {
		return nil
	}
	return v
}

// parseValidateRest reads what comes after validate's call: [to name]
// with x as <shape>, ... and that ... or matches ....
func (p *Parser) parseValidateRest(v *ast.ValidateStatement, what string) bool {
	call := v.Call
	if p.peekTokenIs(token.TO) {
		p.nextToken()
		if !p.expectPeek(token.IDENT) {
			return false
		}
		v.ResultName = p.curToken.Literal
	}
	if p.peekTokenIs(token.IDENT) && p.peekToken.Literal == "with" {
		p.nextToken()
		for {
			if !p.expectPeek(token.IDENT) {
				return false
			}
			name := p.curToken.Literal
			if !p.peekTokenIs(token.IDENT) || p.peekToken.Literal != "as" {
				p.errorf("%s ... with %s: say what %s is, e.g. with %s as list of integer", what, name, name, name)
				return false
			}
			p.nextToken() // -> as
			p.nextToken() // -> the shape
			p.pushStops("that", "matches")
			shape := p.parseShape()
			p.popStops("that", "matches")
			if shape == nil {
				return false
			}
			v.Inputs = append(v.Inputs, &ast.ValidateInput{Name: name, Shape: shape})
			if !p.peekTokenIs(token.COMMA) {
				break
			}
			p.nextToken()
		}
	}
	switch {
	case p.peekTokenIs(token.IDENT) && p.peekToken.Literal == "that":
		p.nextToken()
		p.nextToken()
		p.pushStops(quantWords...)
		p.inVerifyValue++
		e := p.parseExpression(LOWEST)
		p.inVerifyValue--
		p.popStops(quantWords...)
		if p.peekIsQuant() {
			v.Collection = e
			if v.Rule = p.parseRule(what); v.Rule == nil {
				return false
			}
		} else {
			v.That = e
		}
	case p.peekTokenIs(token.IDENT) && p.peekToken.Literal == "matches":
		p.nextToken()
		p.nextToken()
		v.Matches = p.parseExpression(LOWEST)
	default:
		p.errorf("%s %s[...]: give the rule every answer must follow: that ... (true or false), that %s each x give ..., or matches another_function[...]", what, call.Name, v.ResultName)
		return false
	}
	return true
}

// linkValidateExamples gives each validate without "with" the calls to
// the same function in the checks of its test (the function it's in, or
// the file's top level), so it can make inputs like theirs.
func linkValidateExamples(stmts []ast.Statement) {
	var checks []*ast.CheckStatement
	var validates []*ast.ValidateStatement
	var walk func([]ast.Statement)
	walk = func(stmts []ast.Statement) {
		for _, s := range stmts {
			switch x := s.(type) {
			case *ast.FunctionDefStatement:
				if x.Body != nil {
					linkValidateExamples(x.Body.Statements)
				}
			case *ast.CheckStatement:
				checks = append(checks, x)
			case *ast.ValidateStatement:
				if len(x.Inputs) == 0 {
					validates = append(validates, x)
				}
			case *ast.BlockStatement:
				walk(x.Statements)
			case *ast.IfStatement:
				for _, c := range x.Clauses {
					if c.Body != nil {
						walk(c.Body.Statements)
					}
				}
			case *ast.LoopStatement:
				if x.Body != nil {
					walk(x.Body.Statements)
				}
			case *ast.SafeStatement:
				if x.Body != nil {
					walk(x.Body.Statements)
				}
				if x.Handler != nil {
					walk(x.Handler.Statements)
				}
			}
		}
	}
	walk(stmts)
	for _, v := range validates {
		for _, c := range checks {
			findCalls(c.Value, v.Call.Name, len(v.Call.Arguments), &v.Examples)
		}
	}
}

// findCalls collects the calls to name with n arguments inside e.
func findCalls(e ast.Expression, name string, n int, out *[]*ast.CallExpression) {
	switch x := e.(type) {
	case *ast.CallExpression:
		if x.Name == name && len(x.Arguments) == n && x.Subject == nil {
			*out = append(*out, x)
		}
		for _, a := range x.Arguments {
			findCalls(a, name, n, out)
		}
	case *ast.InfixExpression:
		findCalls(x.Left, name, n, out)
		findCalls(x.Right, name, n, out)
	case *ast.PrefixExpression:
		findCalls(x.Right, name, n, out)
	case *ast.MethodCallExpression:
		findCalls(x.Receiver, name, n, out)
		for _, a := range x.Arguments {
			findCalls(a, name, n, out)
		}
	case *ast.FieldExpression:
		findCalls(x.Object, name, n, out)
	}
}
