package parser

import (
	"strings"

	"Turtle/ast"
	"Turtle/token"
)

// hypothesis[...] (core, no import): a claim tried without failing. Inside
// the brackets it's written like a check, a verify, a validate, or about a
// theory:
//
//	hypothesis[sum[xs] == 1]
//	hypothesis[n is integer]
//	hypothesis[prices each p give p > 0]
//	hypothesis[evens[nums] with nums as list of integer that result each x give x % 2 == 0]
//	hypothesis[share]
//	hypothesis[share, theorem result <= 100]
//	hypothesis[share 1 of 4 . is 25.0]
//	hypothesis[share 2 of 1]
//
// Anywhere else hypothesis is an ordinary name.

// startsHypothesis: curToken is hypothesis, with [ right after it.
func (p *Parser) startsHypothesis() bool {
	return p.curTokenIs(token.IDENT) && p.curToken.Literal == "hypothesis" &&
		p.peekTokenIs(token.LBRACKET) && p.peekToken.Line == p.curToken.Line
}

var hypothesisStops = []string{"fails", "each", "any", "not", "exactly", "with", "that", "matches"}

func (p *Parser) parseHypothesis() ast.Expression {
	tok := p.curToken
	if p.inTheory != "" {
		p.errorf("hypothesis[...] isn't written in a theory: a theory's claims are its theorems (theorem ...). Try one from outside it: hypothesis[%s, theorem ...]", p.inTheory)
	}
	h := &ast.HypothesisExpression{Token: tok}
	p.nextToken() // -> [
	if p.peekTokenIs(token.RBRACKET) {
		p.errorf("hypothesis[...]: write the claim to try in the brackets, e.g. hypothesis[sum[xs] == 1]")
		return nil
	}
	start := p.peekToken.Pos
	p.pushStops(hypothesisStops...)
	p.inVerifyValue++
	p.nextToken()
	value := p.parseExpression(LOWEST)
	p.inVerifyValue--
	p.popStops(hypothesisStops...)
	if value == nil {
		return nil
	}

	use, isUse := value.(*ast.TheoryCall)
	id, isName := value.(*ast.Identifier)
	switch {
	case isUse:
		// share 1 of 4 . is 25.0, or share 2 of 1 (its theorems on it).
		h.Use = use
		if p.peekTokenIs(token.PERIOD) {
			p.nextToken()
		}
		if p.peekTokenIs(token.IS) {
			p.nextToken()
			p.nextToken()
			h.Want = p.parseExpression(LOWEST)
		}
	case isName && id.Module == "" && p.peekTokenIs(token.COMMA):
		// share, theorem result <= 100
		p.nextToken() // -> ,
		if !p.peekTokenIs(token.IDENT) || p.peekToken.Literal != "theorem" {
			p.errorf("hypothesis[%s, ...]: after the theory's name, write theorem and the claim to try, e.g. hypothesis[%s, theorem result >= 0]", id.Value, id.Value)
			return nil
		}
		p.nextToken() // -> theorem
		line := p.curToken.Line
		p.nextToken()
		from := p.curToken.Pos
		expr := p.parseExpression(LOWEST)
		if expr == nil {
			return nil
		}
		h.Theory = id
		h.Theorem = &ast.Theorem{Line: line, Expr: expr, Text: strings.TrimSpace(p.l.Source()[from:p.curToken.End])}
	case p.peekIsQuant():
		v := &ast.VerifyStatement{Token: tok, Collection: value}
		if v.Rule = p.parseRule("hypothesis"); v.Rule == nil {
			return nil
		}
		h.Verify = v
	case p.peekTokenIs(token.TO) || p.peekTokenIs(token.IDENT) && (p.peekToken.Literal == "with" || p.peekToken.Literal == "that" || p.peekToken.Literal == "matches"):
		call, ok := value.(*ast.CallExpression)
		if !ok || call.Subject != nil {
			p.errorf("hypothesis[... with ...]: random inputs need a function call to try, e.g. hypothesis[evens[nums] with nums as list of integer that ...]")
			return nil
		}
		v := &ast.ValidateStatement{Token: tok, Call: call, ResultName: "result"}
		if !p.parseValidateRest(v, "hypothesis") {
			return nil
		}
		if len(v.Inputs) == 0 {
			p.errorf("hypothesis[%s[...] ...]: say what inputs to try, e.g. with %s as list of integer", call.Name, placeholderOf(call))
			return nil
		}
		h.Validate = v
	default:
		c := &ast.CheckStatement{Token: tok, Value: value, Form: "true"}
		if !p.parseCheckForm(c) {
			return nil
		}
		h.Check = c
	}
	end := p.curToken.End
	if !p.peekTokenIs(token.RBRACKET) {
		p.errorf("hypothesis[...]: one claim, then ], e.g. hypothesis[xs each x give x > 0]")
		return nil
	}
	p.nextToken()
	h.Text = strings.Join(strings.Fields(p.l.Source()[start:end]), " ")
	switch {
	case h.Check != nil:
		h.Check.Text = h.Text
	case h.Verify != nil:
		h.Verify.Text = h.Text
	case h.Validate != nil:
		h.Validate.Text = h.Text
	}
	return h
}

// placeholderOf is a name for a call's first input, for messages.
func placeholderOf(c *ast.CallExpression) string {
	if len(c.Arguments) > 0 {
		if id, ok := c.Arguments[0].(*ast.Identifier); ok {
			return id.Value
		}
	}
	return "x"
}
