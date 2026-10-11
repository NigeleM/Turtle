// Copyright 2017-2026 Nigele McCoy
// Licensed under the Apache License, Version 2.0;
// see LICENSE and NOTICE.

package evaluator

import (
	"reflect"

	"Turtle/ast"
	"Turtle/token"
)

// A function can read a global (or its enclosing function's variables)
// but assigning one makes a local instead, so a name collision can't
// change shared state. count = count + 1 in a function, with no local
// count yet, would read the global and quietly change only a new local:
// that's always a mistake, so it's an error that says what to do.

func outerAssignError(name, fn string, global bool) {
	if fn == "" {
		fn = "f"
	}
	if global {
		fatalKind(kindName, "%s is a global, and a function can't change a global by assigning to it. "+
			"Give %s the value and return the new one (%s = %s[%s]), or keep it in a list or map, which a function can change",
			name, fn, name, fn, name)
	}
	fatalKind(kindName, "%s belongs to the function around %s, which %s can read but not assign to. "+
		"Return the new value instead, or keep it in a list or map, which a function can change", name, fn, fn)
}

var (
	identifierType = reflect.TypeOf(&ast.Identifier{})
	tokenType      = reflect.TypeOf(token.Token{})
)

// mentionsName reports whether expression e reads the variable name.
// It only runs when an assignment would hide an outer variable, which is
// rare, so a general walk over the tree is fine.
func mentionsName(e ast.Expression, name string) bool {
	return walkMentions(reflect.ValueOf(e), name)
}

func walkMentions(v reflect.Value, name string) bool {
	switch v.Kind() {
	case reflect.Interface:
		if v.IsNil() {
			return false
		}
		return walkMentions(v.Elem(), name)
	case reflect.Pointer:
		if v.IsNil() {
			return false
		}
		if v.Type() == identifierType {
			id := v.Interface().(*ast.Identifier)
			return (id.Value == name && id.Module == "") || id.Module == name
		}
		return walkMentions(v.Elem(), name)
	case reflect.Struct:
		if v.Type() == tokenType {
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
	case reflect.Map:
		for _, k := range v.MapKeys() {
			if walkMentions(v.MapIndex(k), name) {
				return true
			}
		}
	}
	return false
}
