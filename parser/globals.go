// Copyright 2017-2026 Nigele McCoy
// Licensed under the Apache License, Version 2.0;
// see LICENSE and NOTICE.

package parser

import (
	"reflect"

	"Turtle/ast"
	"Turtle/token"
)

// warnLocalShadows warns where a function assigns a name that the file also
// sets at top level (and that isn't the function's parameter). Turtle makes
// that assignment a new local, so the top-level value doesn't change (see
// "Scoping" in the reference). A value that reads the name is already an
// error when it runs, so that case isn't repeated here.
func (p *Parser) warnLocalShadows(stmts []ast.Statement) {
	globals := map[string]bool{}
	collectTopLevel(stmts, globals)
	if len(globals) == 0 {
		return
	}
	for _, st := range stmts {
		if fd, ok := st.(*ast.FunctionDefStatement); ok {
			p.warnInFunction(fd, globals)
		}
	}
}

// collectTopLevel records every name assigned at top level, in blocks and
// loops at top level, but not inside a function.
func collectTopLevel(stmts []ast.Statement, names map[string]bool) {
	for _, st := range stmts {
		switch s := st.(type) {
		case *ast.AssignStatement:
			names[s.Name] = true
		case *ast.BlockStatement:
			collectTopLevel(s.Statements, names)
		case *ast.IfStatement:
			for _, c := range s.Clauses {
				collectTopLevel(c.Body.Statements, names)
			}
		case *ast.LoopStatement:
			collectTopLevel(loopParts(s), names)
		}
	}
}

// loopParts is a loop's statements: its start and step, and its body.
func loopParts(s *ast.LoopStatement) []ast.Statement {
	var out []ast.Statement
	if s.Init != nil {
		out = append(out, s.Init)
	}
	if s.Post != nil {
		out = append(out, s.Post)
	}
	return append(out, s.Body.Statements...)
}

func (p *Parser) warnInFunction(fd *ast.FunctionDefStatement, globals map[string]bool) {
	locals := map[string]bool{}
	for _, par := range fd.Parameters {
		locals[par] = true
	}
	p.warnInBody(fd.Body.Statements, fd.Name, globals, locals)
}

func (p *Parser) warnInBody(stmts []ast.Statement, fn string, globals, locals map[string]bool) {
	for _, st := range stmts {
		switch s := st.(type) {
		case *ast.AssignStatement:
			p.warnShadow(s, fn, globals, locals)
		case *ast.BlockStatement:
			p.warnInBody(s.Statements, fn, globals, locals)
		case *ast.IfStatement:
			for _, c := range s.Clauses {
				p.warnInBody(c.Body.Statements, fn, globals, locals)
			}
		case *ast.LoopStatement:
			if a, ok := s.Init.(*ast.AssignStatement); ok {
				p.warnShadow(a, fn, globals, locals)
			}
			p.warnInBody(s.Body.Statements, fn, globals, locals)
			if a, ok := s.Post.(*ast.AssignStatement); ok {
				p.warnShadow(a, fn, globals, locals)
			}
		case *ast.FunctionDefStatement:
			p.warnInFunction(s, globals)
		}
	}
}

// warnShadow warns once per name and function, at the first assignment that
// makes a local of a top-level name.
func (p *Parser) warnShadow(s *ast.AssignStatement, fn string, globals, locals map[string]bool) {
	if globals[s.Name] && !locals[s.Name] && !mentionsName(s.Value, s.Name) {
		p.warnAt(s.Token.Line, s.Token.Pos, "%s is also a top-level variable, so this makes a new local %s in %s and the top-level %s is not changed. To change it, return the new value and assign that, or keep it in a list or map",
			s.Name, s.Name, fn, s.Name)
	}
	locals[s.Name] = true
}

var (
	identNodeType = reflect.TypeOf(&ast.Identifier{})
	tokenNodeType = reflect.TypeOf(token.Token{})
)

// mentionsName reports whether expression e reads the variable name.
func mentionsName(e ast.Expression, name string) bool {
	return walkMentions(reflect.ValueOf(e), name)
}

func walkMentions(v reflect.Value, name string) bool {
	switch v.Kind() {
	case reflect.Interface:
		return !v.IsNil() && walkMentions(v.Elem(), name)
	case reflect.Pointer:
		if v.IsNil() {
			return false
		}
		if v.Type() == identNodeType {
			id := v.Interface().(*ast.Identifier)
			return (id.Value == name && id.Module == "") || id.Module == name
		}
		return walkMentions(v.Elem(), name)
	case reflect.Struct:
		if v.Type() == tokenNodeType {
			return false
		}
		for i := range v.NumField() {
			if v.Type().Field(i).IsExported() && walkMentions(v.Field(i), name) {
				return true
			}
		}
	case reflect.Slice, reflect.Array:
		for i := range v.Len() {
			if walkMentions(v.Index(i), name) {
				return true
			}
		}
	}
	return false
}
