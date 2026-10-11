// Copyright 2017-2026 Nigele McCoy
// Licensed under the Apache License, Version 2.0;
// see LICENSE and NOTICE.

package parser

import (
	"strconv"

	"Turtle/ast"
	"Turtle/token"
)

// Random values (import random): "random" and a shape, a description of
// a kind of value in words, as in
//
//	random integer from 1 to 6
//	random list of 5 integers from 0 to 9
//	random map of string to integer
//	random list of 3 Order [string, integer from 1 to 10, float]
//
// See ast.Shape for every form. A count comes before the item's kind
// and is a whole number or a variable's name; limits come after it.

// shapeWords are the kinds of simple value, singular and plural.
var shapeWords = map[string]string{
	"integer": "integer", "integers": "integer",
	"float": "float", "floats": "float",
	"string": "string", "strings": "string",
	"digit": "digit", "digits": "digits",
	"letter": "letter", "letters": "letter",
	"boolean": "boolean", "booleans": "boolean",
	"date": "date", "dates": "date",
	"time": "time", "times": "time",
	"list": "list", "lists": "list",
	"set": "set", "sets": "set",
	"map": "map", "maps": "map",
}

// randomFuncs are the random library's functions: "random pick[x]" names
// the library, not an assembled type called pick.
var randomFuncs = map[string]bool{"pick": true, "shuffle": true, "sample": true, "chance": true}

// shapeConnectors are words that continue a shape after a value, so a
// name before them isn't a sentence-style call ("to n rounded to 2").
var shapeConnectors = map[string]bool{"rounded": true, "places": true}

// isShapeStart reports whether t (followed by next) begins a shape.
func isShapeStart(t, next token.Token) bool {
	switch t.Type {
	case token.LIST, token.SET, token.MAP:
		return true
	case token.IDENT:
		if _, ok := shapeWords[t.Literal]; ok {
			return true
		}
		return next.Type == token.LBRACKET && next.Line == t.Line && !randomFuncs[t.Literal]
	}
	return false
}

// startsRandom: curToken is "random", after import random, with a shape
// right after it on the same line.
func (p *Parser) startsRandom() bool {
	return p.randomImported && p.curTokenIs(token.IDENT) && p.curToken.Literal == "random" &&
		p.peekToken.Line == p.curToken.Line && isShapeStart(p.peekToken, p.peekN(2))
}

func (p *Parser) parseRandomExpression() ast.Expression {
	tok := p.curToken
	p.nextToken()
	shape := p.parseShape()
	if shape == nil {
		return nil
	}
	return &ast.RandomExpression{Token: tok, Shape: shape}
}

// parseShape parses a shape starting at curToken and leaves curToken on
// its last token.
func (p *Parser) parseShape() *ast.Shape {
	tok := p.curToken
	word := tok.Literal
	switch tok.Type {
	case token.LIST:
		word = "list"
	case token.SET:
		word = "set"
	case token.MAP:
		word = "map"
	}
	kind, simple := shapeWords[word]
	s := &ast.Shape{Token: tok, Kind: kind}
	if !simple {
		if tok.Type != token.IDENT || !p.peekTokenIs(token.LBRACKET) || p.peekToken.Line != tok.Line {
			p.errorf("random: %q isn't a kind of value; use integer, float, string, digits, digit, letter, boolean, date, time, list of ..., set of ..., map of ... to ..., or an assembled type with its field kinds, e.g. Order [string, integer]", tok.Literal)
			return nil
		}
		return p.parseAssembledShape(s)
	}
	switch kind {
	case "list", "set":
		if !p.expectPeek(token.OF) {
			return nil
		}
		p.parseShapeCount(s)
		p.nextToken()
		if s.Item = p.parseShape(); s.Item == nil {
			return nil
		}
	case "map":
		if !p.expectPeek(token.OF) {
			return nil
		}
		p.parseShapeCount(s)
		p.nextToken()
		if s.Key = p.parseShape(); s.Key == nil {
			return nil
		}
		if !p.expectPeek(token.TO) {
			p.errorf("random map of ...: say what the keys map to, e.g. map of string to integer")
			return nil
		}
		p.nextToken()
		if s.Item = p.parseShape(); s.Item == nil {
			return nil
		}
	case "integer", "float", "date", "time":
		if p.peekTokenIs(token.FROM) && p.peekToken.Line == p.curToken.Line {
			p.nextToken()
			p.nextToken()
			s.From = p.parseShapeValue()
			if !p.expectPeek(token.TO) {
				p.errorf("random %s from ...: give the range as from A to B", kind)
				return nil
			}
			p.nextToken()
			s.To = p.parseShapeValue()
		}
		if kind == "float" && p.peekToken.Literal == "rounded" && p.peekToken.Line == p.curToken.Line {
			p.nextToken()
			if !p.expectPeek(token.TO) {
				p.errorf("random float ... rounded: say how many places, e.g. rounded to 2")
				return nil
			}
			p.nextToken()
			s.Places = p.parseShapeValue()
			if p.peekToken.Literal == "places" && p.peekToken.Line == p.curToken.Line {
				p.nextToken()
			}
		}
	case "digits":
		if !p.peekTokenIs(token.OF) {
			s.Kind = "digit" // "list of 3 digits": three single digits
			break
		}
		p.parseTextOptions(s)
	case "string":
		p.parseTextOptions(s)
	}
	return s
}

// parseTextOptions reads "of N [to M]" and, for a string, "from chars", in
// either order.
func (p *Parser) parseTextOptions(s *ast.Shape) {
	for p.peekToken.Line == p.curToken.Line {
		switch {
		case p.peekTokenIs(token.OF) && s.Length == nil:
			p.nextToken()
			p.nextToken()
			s.Length = p.countValue()
			if p.peekTokenIs(token.TO) && isCountToken(p.peekN(2), p.peekN(3)) {
				p.nextToken()
				p.nextToken()
				s.LengthTo = p.countValue()
			}
		case p.peekTokenIs(token.FROM) && s.Kind == "string" && s.Chars == nil:
			p.nextToken()
			p.nextToken()
			s.Chars = p.parseShapeValue()
		default:
			return
		}
	}
}

// parseShapeCount reads a collection's optional "N" or "N to M" after
// "of", leaving curToken on its last token.
func (p *Parser) parseShapeCount(s *ast.Shape) {
	if !isCountToken(p.peekToken, p.peekN(2)) {
		return
	}
	p.nextToken()
	s.Count = p.countValue()
	if p.peekTokenIs(token.TO) && isCountToken(p.peekN(2), p.peekN(3)) {
		p.nextToken()
		p.nextToken()
		s.CountTo = p.countValue()
	}
}

// isCountToken: a whole number, or a variable's name that isn't itself
// a kind of value.
func isCountToken(t, next token.Token) bool {
	return t.Type == token.INT || t.Type == token.IDENT && !isShapeStart(t, next)
}

func (p *Parser) countValue() ast.Expression {
	if p.curTokenIs(token.INT) {
		v, err := strconv.ParseInt(p.curToken.Literal, 10, 64)
		if err != nil {
			p.errorf("random: the count %s is too big", p.curToken.Literal)
			return nil
		}
		return &ast.IntegerLiteral{Token: p.curToken, Value: v}
	}
	if !p.curTokenIs(token.IDENT) {
		p.errorf("random: a count is a whole number or a variable's name, got %q", p.curToken.Literal)
		return nil
	}
	return &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
}

// parseShapeValue parses a limit: arithmetic belongs to it (to n + 1),
// but a comparison or "at" applies to the random value itself, so
// random integer from 1 to 6 == 6 compares the roll.
func (p *Parser) parseShapeValue() ast.Expression {
	saved := p.inShape
	p.inShape = true
	defer func() { p.inShape = saved }()
	return p.parseExpression(LESSGREATER)
}

// parseAssembledShape reads "Order [shape, shape, ...]".
func (p *Parser) parseAssembledShape(s *ast.Shape) *ast.Shape {
	s.Kind = "assembled"
	s.TypeName = p.curToken.Literal
	p.nextToken() // -> '['
	if p.peekTokenIs(token.RBRACKET) {
		p.errorf("random %s []: give a kind for each field, e.g. %s [string, integer]", s.TypeName, s.TypeName)
		return nil
	}
	for {
		p.nextToken()
		f := p.parseShape()
		if f == nil {
			return nil
		}
		s.Fields = append(s.Fields, f)
		if !p.peekTokenIs(token.COMMA) {
			break
		}
		p.nextToken()
	}
	if !p.expectPeek(token.RBRACKET) {
		return nil
	}
	return s
}
