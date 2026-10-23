// Copyright 2017-2026 Nigele McCoy
// Licensed under the Apache License, Version 2.0;
// see LICENSE and NOTICE.

package object

import "testing"

func TestGetFallsBackToGlobal(t *testing.T) {
	global := NewGlobalEnvironment()
	global.Set("x", &Integer{Value: 1})

	call := global.NewCallEnvironment()
	v, ok := call.Get("x")
	if !ok {
		t.Fatal("expected call scope to read the global 'x', found nothing")
	}
	if v.(*Integer).Value != 1 {
		t.Errorf("got %v, want 1", v)
	}
}

func TestLocalShadowsGlobalWithoutMutatingIt(t *testing.T) {
	global := NewGlobalEnvironment()
	global.Set("x", &Integer{Value: 1})

	call := global.NewCallEnvironment()
	call.Set("x", &Integer{Value: 99}) // a local write inside a "function call"

	localVal, _ := call.Get("x")
	if localVal.(*Integer).Value != 99 {
		t.Errorf("local scope: got %v, want 99", localVal)
	}

	globalVal, _ := global.Get("x")
	if globalVal.(*Integer).Value != 1 {
		t.Errorf("global scope should be untouched: got %v, want 1", globalVal)
	}
}

func TestGetMissingEverywhere(t *testing.T) {
	global := NewGlobalEnvironment()
	call := global.NewCallEnvironment()
	if _, ok := call.Get("nope"); ok {
		t.Error("expected 'nope' to be missing in both scopes")
	}
}

func TestFunctionsAreGlobalRegardlessOfDefiningScope(t *testing.T) {
	global := NewGlobalEnvironment()
	call := global.NewCallEnvironment()

	fn := &Function{Name: "f", Parameters: []string{"a"}}
	call.DefineFunction(fn) // defining "from" a call scope still lands globally

	if _, ok := global.GetFunction("f"); !ok {
		t.Error("function defined via a call scope should be visible from global")
	}
	if _, ok := call.GetFunction("f"); !ok {
		t.Error("function should also be visible from the call scope that defined it")
	}
}

func TestGetWalksEnclosingScopes(t *testing.T) {
	global := NewGlobalEnvironment()
	global.Set("g", &Integer{Value: 1})
	outer := NewEnclosedEnvironment(global)
	outer.Set("o", &Integer{Value: 2})
	inner := NewEnclosedEnvironment(outer)

	if v, ok := inner.Get("o"); !ok || v.(*Integer).Value != 2 {
		t.Errorf("inner should read enclosing call's 'o', got %v, %v", v, ok)
	}
	if v, ok := inner.Get("g"); !ok || v.(*Integer).Value != 1 {
		t.Errorf("inner should read global 'g', got %v, %v", v, ok)
	}
	inner.Set("o", &Integer{Value: 99})
	if v, _ := outer.Get("o"); v.(*Integer).Value != 2 {
		t.Errorf("inner Set must not write through to outer, got %v", v)
	}
}

// StoreAt stores again where SetUnlessShadows stored, and refuses whenever
// that place may no longer be right, so the caller looks again.
func TestStoreAtFollowsSetUnlessShadows(t *testing.T) {
	global := NewGlobalEnvironment()
	global.Set("lo", Int(0))
	loop := NewLoopEnvironment(global)
	loop.Define("i", Int(0))

	// lo, assigned inside the loop, lives in the global scope one loop out.
	_, _, p := loop.SetUnlessShadows("lo", Int(1))
	if !loop.StoreAt("lo", p, Int(2)) {
		t.Fatal("StoreAt refused the place it was given")
	}
	if v, _ := global.Get("lo"); v.(*Integer).Value != 2 {
		t.Errorf("lo = %v, want 2", v)
	}
	// The loop's own name is stored in the loop's scope.
	_, _, pi := loop.SetUnlessShadows("i", Int(5))
	if !loop.StoreAt("i", pi, Int(6)) {
		t.Fatal("StoreAt refused the loop name's place")
	}
	if v, _ := loop.Get("i"); v.(*Integer).Value != 6 {
		t.Errorf("i = %v, want 6", v)
	}
	if _, ok := global.Get("i"); ok {
		t.Error("the loop name leaked into the global scope")
	}
	// A loop scope that has the name, between here and the place: refuse.
	inner := NewLoopEnvironment(loop)
	inner.Define("lo", Int(9))
	if inner.StoreAt("lo", place(1, 0), Int(3)) {
		t.Error("StoreAt passed a loop scope that has the name")
	}
	// A call scope between here and the place: an assignment would make a
	// local there, so refuse.
	call := global.NewCallEnvironment()
	if call.StoreAt("lo", p, Int(4)) {
		t.Error("StoreAt passed a call scope")
	}
	// The place no longer holds the name (a reused scope): refuse.
	fresh := NewGlobalEnvironment()
	fresh.Set("other", Int(0))
	freshLoop := NewLoopEnvironment(fresh)
	if freshLoop.StoreAt("lo", p, Int(5)) {
		t.Error("StoreAt stored where the name no longer is")
	}
	if StoreZero := (&Environment{}); StoreZero.StoreAt("lo", 0, Int(1)) {
		t.Error("StoreAt took no place as a place")
	}
}
