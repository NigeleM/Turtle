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
