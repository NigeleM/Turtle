// Copyright 2017-2026 Nigele McCoy
// Licensed under the Apache License, Version 2.0;
// see LICENSE and NOTICE.

package parser

import (
	"strings"

	"Turtle/ast"
	"Turtle/token"
)

// Scrolls: a value going through steps in order, each step's result
// feeding the next, ending with a period, like a sentence.
//
//	x is scroll 3 into add1, double, half .         // runs now
//	s = scroll add1, double, half .                 // a saved scroll
//	names is scroll raw into
//	    splitby ",",                                // one extra value
//	    add_time[here, 30, "days"],                 // here is the value
//	    at trim,                                    // a method
//	    n give n * 5 .                              // a function
//
// Outside [ ], a comma starts the next step; inside, it separates a
// call's arguments. A step without here gets the value first; one with
// here gets it where here is.

// parseScrollExpression parses a scroll. curToken is "scroll". It ends
// on its last step's last token with the period next, so a statement
// takes the period as its own end; inside [ ] the scroll takes its
// period itself.
func (p *Parser) parseScrollExpression() ast.Expression {
	se := &ast.ScrollExpression{Token: p.curToken}
	outerBrackets := p.inBrackets
	defer func(b, r, f, s bool) {
		p.inBrackets, p.inIsReceiver, p.inFieldObject, p.inShape = b, r, f, s
	}(p.inBrackets, p.inIsReceiver, p.inFieldObject, p.inShape)
	// A step's sentence calls take one argument: the commas are the scroll's.
	p.inBrackets, p.inIsReceiver, p.inFieldObject, p.inShape = true, false, false, false

	if p.peekTokenIs(token.PERIOD) || p.peekTokenIs(token.EOF) {
		p.errorAt(p.curToken.Line, p.curToken.End, "a scroll needs at least one step: scroll value into step, step .  or  s = scroll step, step .")
		return nil
	}
	run := p.scrollHasInto()
	p.nextToken()
	if run {
		p.pushStops("into")
		se.Start = p.parseExpression(LOWEST)
		p.popStops("into")
		if !(p.peekTokenIs(token.IDENT) && p.peekToken.Literal == "into") {
			p.errorAt(p.curToken.Line, p.curToken.End, "a scroll with a starting value needs \"into\" before its steps")
			return nil
		}
		p.nextToken() // -> into
		if p.peekTokenIs(token.PERIOD) || p.peekTokenIs(token.EOF) {
			p.errorAt(p.curToken.Line, p.curToken.End, "a scroll needs at least one step after \"into\"")
			return nil
		}
		p.nextToken() // -> the first step (on this line or the next)
	}

	saved := p.hereUses
	defer func() { p.hereUses = saved }()
	callHint := "" // the last step's "name value", for the two-values hint
	for n := 1; ; n++ {
		if p.curTokenIs(token.COMMA) || p.curTokenIs(token.PERIOD) {
			if p.curTokenIs(token.PERIOD) {
				p.errorf("a comma after the last step: remove it, or add a step")
			} else {
				p.errorf("scroll step %d is empty", n)
			}
			return nil
		}
		step, nameValue := p.parseScrollStep(n, se.Start == nil && n == 1, callHint)
		if step == nil {
			return nil
		}
		se.Steps = append(se.Steps, step)
		callHint = ""
		if nameValue {
			callHint = step.Label
		}
		switch {
		case p.peekTokenIs(token.COMMA):
			p.nextToken()
			if p.peekTokenIs(token.EOF) {
				p.errorf("a comma after the last step: remove it, or add a step")
				return nil
			}
			p.nextToken()
			continue
		case p.peekTokenIs(token.PERIOD):
			if outerBrackets {
				p.nextToken() // the period is the scroll's: ] or , follows
			}
			return se
		case p.peekTokenIs(token.EOF) || p.peekToken.Line != p.endLine(p.curToken):
			p.errorAt(p.endLine(p.curToken), p.curToken.End, "a scroll ends with '.' (after its last step), or goes on with ',' and the next step")
		case outerBrackets && p.peekTokenIs(token.RBRACKET):
			p.errorAt(p.peekToken.Line, p.peekToken.Pos, "a scroll ends with '.', inside [ ] too: put a period after its last step, before the ]")
		default:
			p.errorAt(p.peekToken.Line, p.peekToken.Pos, "after scroll step %d, expected ',' and the next step, or '.' to end the scroll; found %s", n, describe(p.peekToken))
		}
		return nil
	}
}

// scrollHasInto looks ahead (curToken is "scroll") for "into" before the
// scroll's period: with it, the scroll has a starting value and runs.
func (p *Parser) scrollHasInto() bool {
	depth := 0
	for i := 1; i < 4000; i++ {
		t := p.peekN(i)
		switch t.Type {
		case token.EOF:
			return false
		case token.LBRACKET:
			depth++
		case token.RBRACKET:
			depth--
		case token.PERIOD, token.SCROLL, token.COMMA:
			if depth <= 0 {
				return false
			}
		case token.IDENT:
			if depth <= 0 && t.Literal == "into" {
				return true
			}
		}
	}
	return false
}

// parseScrollStep parses one step; curToken is its first token, and it
// ends on the step's last token. first is true for a saved scroll's first
// step, where a plain value means a forgotten "into". nameValue reports a
// "name value" step, for the hint when a second value follows it.
func (p *Parser) parseScrollStep(n int, first bool, callHint string) (step *ast.ScrollStep, nameValue bool) {
	start := p.curToken
	step = &ast.ScrollStep{Token: start}
	p.hereUses = 0
	here := &ast.Identifier{Token: token.Token{Type: token.IDENT, Literal: "here", Line: start.Line, Pos: start.Pos, End: start.Pos}, Value: "here"}

	switch {
	case p.curTokenIs(token.AT):
		// "at trim", "at round[2]", "at get 0": a method on the value.
		mc := &ast.MethodCallExpression{Token: start, Receiver: here}
		p.nextToken()
		mc.Method = p.curToken.Literal
		if p.peekTokenIs(token.LBRACKET) && p.peekToken.Line == p.curToken.Line {
			p.nextToken()
			mc.Arguments = p.parseExpressionList(token.RBRACKET)
			mc.Bracketed = true
		} else if p.peekStartsArgument() {
			p.nextToken()
			mc.Arguments = []ast.Expression{p.parseExpression(LOWEST)}
		}
		step.Kind, step.Expr = ast.StepHere, p.continueExpression(mc)

	case p.curTokenIs(token.IDENT) && p.peekToken.Line == p.curToken.Line && p.peekStartsArgument() &&
		!p.peekTokenIs(token.LBRACKET) && !p.typeCheckAhead():
		// "splitby ",""/"keep u give ...": a function and one extra value.
		name := p.curToken
		nameValue = true
		p.nextToken()
		arg := p.parseExpression(LOWEST)
		call := &ast.CallExpression{Token: name, Name: name.Literal, Arguments: []ast.Expression{arg}}
		if p.hereUses > 0 {
			step.Kind, step.Expr = ast.StepHere, call
		} else {
			step.Kind, step.Call = ast.StepCall, call
		}

	case first && startsValue(start.Type):
		// "scroll 3 add1 .": a value where a step should be.
		p.errorAt(start.Line, start.Pos, "a scroll with a starting value needs \"into\" before its steps: scroll value into step, step .")
		return nil, false

	default:
		expr := p.parseExpression(LOWEST)
		if expr == nil {
			return nil, false
		}
		switch expr.(type) {
		case *ast.StringLiteral, *ast.IntegerLiteral, *ast.FloatLiteral, *ast.BooleanLiteral, *ast.NoneLiteral:
			if p.hereUses == 0 {
				what := describe(start)
				if start.Type == token.INT || start.Type == token.FLOAT {
					what = "the number " + start.Literal
				}
				switch {
				case first:
					p.errorAt(start.Line, start.Pos, "a scroll with a starting value needs \"into\" before its steps: scroll %s into step, step .", start.Literal)
				case callHint != "":
					fn, extra, _ := strings.Cut(callHint, " ")
					p.errorAt(start.Line, start.Pos, "scroll step %d is %s, not a step; for two or more values use brackets: %s[here, %s, %s]", n, what, fn, extra, p.stepText(start, start))
				default:
					p.errorAt(start.Line, start.Pos, "scroll step %d is %s, not a step (a function, a method with at, or an expression with here)", n, what)
				}
				return nil, false
			}
		}
		switch {
		case p.hereUses > 0:
			step.Kind, step.Expr = ast.StepHere, expr
		default:
			switch e := expr.(type) {
			case *ast.CallExpression:
				if e.Subject == nil {
					step.Kind, step.Call = ast.StepCall, e
					break
				}
				step.Kind, step.Expr = ast.StepValue, expr
			case *ast.Identifier:
				step.Kind, step.Call = ast.StepCall, &ast.CallExpression{Token: e.Token, Module: e.Module, Name: e.Value}
			default:
				step.Kind, step.Expr = ast.StepValue, expr
			}
		}
	}
	step.Label = p.stepText(start, p.curToken)
	return step, nameValue
}

// continueExpression goes on parsing operators after a value already
// parsed (a method step: "at len + 1", "at trim at upper").
func (p *Parser) continueExpression(left ast.Expression) ast.Expression {
	for LOWEST < p.peekPrecedence() {
		infix := p.infixParseFns[p.peekToken.Type]
		if infix == nil {
			return left
		}
		p.nextToken()
		left = infix(left)
	}
	return left
}

// stepText is the program's text from token a to token b, on one line
// (spaces squeezed), for a step's label.
func (p *Parser) stepText(a, b token.Token) string {
	src := p.l.Source()
	if a.Pos < 0 || b.End > len(src) || a.Pos >= b.End {
		return a.Literal
	}
	return strings.Join(strings.Fields(src[a.Pos:b.End]), " ")
}

// endScroll is for a statement whose value was just parsed and stepped
// past: a scroll's period is the statement's end too.
func (p *Parser) endScroll(val ast.Expression) {
	if _, ok := val.(*ast.ScrollExpression); ok && p.curTokenIs(token.PERIOD) {
		p.nextToken()
	}
}

// startsValue: a token that begins a plain value, never a step.
func startsValue(t token.Type) bool {
	switch t {
	case token.INT, token.FLOAT, token.STRING, token.RAWSTRING, token.TRUE, token.FALSE,
		token.NONE, token.LIST, token.SET, token.MAP:
		return true
	}
	return false
}

// parseDiagnoseBlock parses "diagnose" on a line of its own, the lines
// under it, and "diagnose [end]". curToken is "diagnose".
func (p *Parser) parseDiagnoseBlock() ast.Statement {
	tok := p.curToken
	body := p.parseBlockUntil(p.isDiagnoseEnd)
	if !p.isDiagnoseEnd() {
		p.unclosedError(tok, "this diagnose", "diagnose [end]")
		return nil
	}
	end := p.curToken.Line
	p.nextToken() // -> [
	p.nextToken() // -> end
	p.nextToken() // -> ]
	p.nextToken()
	if p.lines == nil {
		p.lines = strings.Split(p.l.Input(), "\n")
	}
	return &ast.DiagnoseStatement{Token: tok, Body: body, End: end, Lines: p.lines}
}

func (p *Parser) isDiagnoseEnd() bool {
	return p.curTokenIs(token.IDENT) && p.curToken.Literal == "diagnose" &&
		p.peekN(1).Type == token.LBRACKET && p.peekN(2).Type == token.END && p.peekN(3).Type == token.RBRACKET
}
