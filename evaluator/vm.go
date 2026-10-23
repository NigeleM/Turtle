// Copyright 2017-2026 Nigele McCoy
// Licensed under the Apache License, Version 2.0;
// see LICENSE and NOTICE.

package evaluator

// A function body made into Go functions once, the first time it is
// called: its names become numbered slots in a frame, so nothing is looked
// up by name; whole numbers stay plain int64s between steps, so none is
// made into a value; and no scope is made for a call. Only bodies made of
// the parts below are compiled (numbers, comparisons, its own variables,
// if, loops, calls by name); any other body runs as before, so a program
// does the same either way, only sooner.

import (
	"fmt"
	"maps"
	"math"
	"os"
	"slices"
	"strings"
	"sync/atomic"

	"Turtle/ast"
	"Turtle/object"
)

// vmEnabled turns compiled bodies on; TURTLE_NOVM=1 (or a test) turns
// them off, to compare.
var vmEnabled = os.Getenv("TURTLE_NOVM") == ""

// val is a value in a compiled body: a whole number as itself (o nil),
// or any other value.
type val struct {
	i int64
	o object.Object
}

func (v val) box() object.Object {
	if v.o == nil {
		return object.Int(v.i)
	}
	return v.o
}

func valOf(o object.Object) val {
	if x, ok := o.(*object.Integer); ok {
		return val{i: x.Value}
	}
	return val{o: o}
}

// vmUnset marks a slot of the function's own not set yet: reading it
// reads the outer name instead, as the tree does before the function sets it.
type vmUnset struct{ _ byte }

func (*vmUnset) Type() object.Type { return object.NONE }
func (*vmUnset) Inspect() string   { return "none" }

var unset = val{o: &vmUnset{}}

// newFrame fills a frame for c: the parameters set, the rest not yet.
func newFrame(s []val, nparams int) {
	for i := nparams; i < len(s); i++ {
		s[i] = unset
	}
}

func (v val) truthy() bool {
	if v.o == nil {
		return v.i != 0
	}
	return isTruthy(v.o)
}

// How a statement ends.
type ctl uint8

const (
	ctlNext ctl = iota
	ctlBreak
	ctlContinue
	ctlReturn
)

type (
	exprFn func(it *Interpreter, s []val) val
	condFn func(it *Interpreter, s []val) bool
	stmtFn func(it *Interpreter, s []val) ctl
)

// compiled is a function body made into Go functions: nslots slots, the
// parameters first.
type compiled struct {
	nslots  int
	nparams int
	body    []stmtFn
}

// notCompiled marks a body that has parts compiling doesn't cover.
var notCompiled = &compiled{}

// vmStack holds the frames of compiled calls, in chunks that never move,
// so a frame's slice stays good while calls above it come and go.
type vmStack struct {
	chunks [][]val
	ci, sp int
}

type vmMark struct{ ci, sp int }

func (st *vmStack) mark() vmMark { return vmMark{st.ci, st.sp} }

func (st *vmStack) release(m vmMark) { st.ci, st.sp = m.ci, m.sp }

func (st *vmStack) alloc(n int) []val {
	if st.chunks == nil {
		st.chunks = [][]val{make([]val, 4096)}
	}
	if st.sp+n > len(st.chunks[st.ci]) {
		st.ci++
		if st.ci == len(st.chunks) {
			st.chunks = append(st.chunks, make([]val, max(65536, n)))
		} else if len(st.chunks[st.ci]) < n {
			st.chunks[st.ci] = make([]val, n)
		}
		st.sp = 0
	}
	s := st.chunks[st.ci][st.sp : st.sp+n : st.sp+n]
	st.sp += n
	return s
}

// compiledBody is fn's body compiled, or nil when it runs as a tree.
func (it *Interpreter) compiledBody(fn *object.Function) *compiled {
	if !vmEnabled || it.Trace != nil || it.debug != nil || fn.Body == nil {
		return nil
	}
	if c, ok := fn.Body.Compiled.Load().(*compiled); ok {
		if c == notCompiled {
			return nil
		}
		return c
	}
	c := compileFunc(fn)
	fn.Body.Compiled.Store(c)
	if c != notCompiled {
		vmCompiledCount.Add(1)
	}
	if vmStats {
		if c == notCompiled {
			fmt.Fprintf(os.Stderr, "vm: tree %s: %s\n", fn.Name, vmWhy)
		} else {
			fmt.Fprintf(os.Stderr, "vm: compiled %s\n", fn.Name)
		}
	}
	if c == notCompiled {
		return nil
	}
	return c
}

// vmFromTree runs a compiled body called from the tree: on the way out,
// by a return or an error, what the call changed is put back.
//
//go:noinline
func (it *Interpreter) vmFromTree(fn *object.Function, c *compiled, name string, args []object.Object) object.Object {
	st := &it.vm
	m := st.mark()
	prevFile, prevLine, prevBuiltin, prevFunc, prevDepth, prevEnv := currentFile, currentLine, it.inBuiltin, it.funcName, it.depth, it.vmEnv
	defer func() {
		currentFile, currentLine, it.inBuiltin, it.funcName, it.depth, it.vmEnv = prevFile, prevLine, prevBuiltin, prevFunc, prevDepth, prevEnv
		st.release(m)
	}()
	s := st.alloc(c.nslots)
	for i, a := range args {
		s[i] = valOf(a)
	}
	newFrame(s, c.nparams)
	r := it.vmEnter(fn, c, name, s)
	clear(s)
	return r.box()
}

// vmEnter runs a compiled body in the frame s, its arguments in place.
// What it changes it puts back on a return; after an error, vmFromTree
// does.
func (it *Interpreter) vmEnter(fn *object.Function, c *compiled, name string, s []val) val {
	if fn.Name != "" {
		name = fn.Name
	}
	if reason := it.stop.Load(); reason != stopNone {
		it.stopNow(reason)
	}
	defEnv := fn.Env
	if defEnv == nil {
		defEnv = it.Global
	}
	prevFile, prevLine, prevBuiltin, prevFunc, prevEnv := currentFile, currentLine, it.inBuiltin, it.funcName, it.vmEnv
	it.depth++
	if it.depth > maxCallDepth {
		tooManyCalls(name)
	}
	if it.inBuiltin = isBuiltinEnv(defEnv); !it.inBuiltin {
		currentFile = defEnv.File()
	}
	it.funcName, it.inPlace, it.vmEnv = name, false, defEnv
	how := runStmts(it, s, c.body)
	it.depth--
	currentFile, currentLine, it.inBuiltin, it.funcName, it.vmEnv = prevFile, prevLine, prevBuiltin, prevFunc, prevEnv
	if how == ctlReturn {
		r := it.vmRet
		it.vmRet = val{}
		return r
	}
	return val{o: object.NoneValue}
}

func runStmts(it *Interpreter, s []val, body []stmtFn) ctl {
	for _, st := range body {
		if how := st(it, s); how != ctlNext {
			return how
		}
	}
	return ctlNext
}

// ---- compiling ----

type compiler struct {
	locals  map[string]int   // the function's own names: their slots, the parameters first
	loops   []map[string]int // loop counters, innermost last
	defined map[int]bool     // slots surely set at this point
	// assigned are the names the body assigns anywhere: one of them read
	// where it isn't surely set might be the function's own or an outer
	// one, so such a body isn't compiled. Any other name is an outer one.
	assigned map[string]bool
	n        int
	failed   bool
}

// settingNames are variables libraries read from the caller's scope
// (tablerows for table, seed for random, ...): a body that names one keeps
// its scope, so a library it calls still finds it there.
var settingNames = map[string]bool{
	tableRowsName: true, seedName: true, reportFileName: true, hereName: true, suiteName: true,
	logLevelName: true, logFileName: true, logConsoleName: true, logTimeName: true, logPartsName: true,
	logFormatName: true, logMaxSizeName: true, logKeepName: true, outputFileName: true,
	scheduleLimitName: true, scheduleSkipName: true, serverLogName: true, serverHostName: true,
	serverTimeoutName: true, benchmarkName: true, runsName: true, benchtimeName: true, casesName: true,
	"range": true,
}

func compileFunc(fn *object.Function) *compiled {
	c := &compiler{locals: map[string]int{}, defined: map[int]bool{}, assigned: map[string]bool{}}
	assignedNames(fn.Body.Statements, c.assigned)
	for _, p := range fn.Parameters {
		if _, dup := c.locals[p]; dup || settingNames[p] {
			vmWhy = "parameter"
			return notCompiled
		}
		c.locals[p] = c.n
		c.defined[c.n] = true
		c.n++
	}
	body := c.block(fn.Body.Statements)
	if c.failed {
		return notCompiled
	}
	return &compiled{nslots: c.n, nparams: len(fn.Parameters), body: body}
}

// assignedNames adds the names stmts assign, at any depth, to into.
func assignedNames(stmts []ast.Statement, into map[string]bool) {
	for _, st := range stmts {
		switch s := st.(type) {
		case *ast.AssignStatement:
			into[s.Name] = true
		case *ast.IfStatement:
			for _, cl := range s.Clauses {
				if cl.Body != nil {
					assignedNames(cl.Body.Statements, into)
				}
			}
		case *ast.LoopStatement:
			if s.Init != nil {
				assignedNames([]ast.Statement{s.Init}, into)
			}
			if s.Post != nil {
				assignedNames([]ast.Statement{s.Post}, into)
			}
			if s.Body != nil {
				assignedNames(s.Body.Statements, into)
			}
		}
	}
}

// local is the slot of a name the function sets: where it is in scope,
// else a new slot of the function's own.
func (c *compiler) local(name string) int {
	if slot, ok := c.resolve(name); ok {
		return slot
	}
	slot := c.n
	c.n++
	c.locals[name] = slot
	return slot
}

func (c *compiler) resolve(name string) (int, bool) {
	for i := len(c.loops) - 1; i >= 0; i-- {
		if slot, ok := c.loops[i][name]; ok {
			return slot, true
		}
	}
	slot, ok := c.locals[name]
	return slot, ok
}

// fail marks the body as not compiled; what's returned is never run.
func (c *compiler) fail(why string) {
	if !c.failed && vmStats {
		vmWhy = why
	}
	c.failed = true
}

// vmStats (TURTLE_VMSTATS=1) writes, for each body, whether it compiled
// and if not what stopped it.
var (
	// vmCompiledCount counts bodies compiled, for tests.
	vmCompiledCount atomic.Int64
	vmStats         = os.Getenv("TURTLE_VMSTATS") != ""
	vmWhy           string
)

func (c *compiler) block(stmts []ast.Statement) []stmtFn {
	out := make([]stmtFn, 0, len(stmts))
	for _, st := range stmts {
		out = append(out, c.stmt(st))
		if c.failed {
			return nil
		}
	}
	return out
}

// branch compiles stmts that may not run: what they set isn't surely set
// after them.
func (c *compiler) branch(stmts []ast.Statement) []stmtFn {
	saved := maps.Clone(c.defined)
	out := c.block(stmts)
	c.defined = saved
	return out
}

func lineOf(st ast.Statement) int { return st.Line() }

func (c *compiler) stmt(st ast.Statement) stmtFn {
	line := lineOf(st)
	switch s := st.(type) {
	case *ast.AssignStatement:
		if s.Name == seedName {
			c.fail("seed")
			return nil
		}
		if settingNames[s.Name] {
			c.fail("setting")
			return nil
		}
		if sl, ok := c.resolve(s.Name); (!ok || !c.defined[sl]) && assignReads(s) {
			c.fail("assignment may change an outer name")
			return nil
		}
		value, name := c.expr(s.Value), s.Name
		slot, ok := c.resolve(s.Name)
		if !ok {
			slot = c.local(s.Name)
		}
		c.defined[slot] = true
		return func(it *Interpreter, sl []val) ctl {
			if !it.inBuiltin {
				currentLine = line
			}
			v := value(it, sl)
			if fn, ok := v.o.(*object.Function); ok && fn.Scroll != nil && fn.Name == "" {
				fn.Name = name // a saved scroll is named after its variable
			}
			sl[slot] = v
			return ctlNext
		}

	case *ast.ReturnStatement:
		if s.Value == nil {
			return func(it *Interpreter, sl []val) ctl {
				it.vmRet = val{o: object.NoneValue}
				return ctlReturn
			}
		}
		value := c.expr(s.Value)
		return func(it *Interpreter, sl []val) ctl {
			if !it.inBuiltin {
				currentLine = line
			}
			it.vmRet = value(it, sl)
			return ctlReturn
		}

	case *ast.BreakStatement:
		return func(*Interpreter, []val) ctl { return ctlBreak }
	case *ast.ContinueStatement:
		return func(*Interpreter, []val) ctl { return ctlContinue }

	case *ast.CallStatement:
		call := c.call(s.Call, true)
		return func(it *Interpreter, sl []val) ctl {
			if !it.inBuiltin {
				currentLine = line
			}
			call(it, sl)
			return ctlNext
		}

	case *ast.IfStatement:
		conds := make([]condFn, len(s.Clauses))
		bodies := make([][]stmtFn, len(s.Clauses))
		for i, cl := range s.Clauses {
			if cl.Condition != nil {
				conds[i] = c.cond(cl.Condition)
			}
			if cl.Body != nil {
				bodies[i] = c.branch(cl.Body.Statements)
			}
		}
		return func(it *Interpreter, sl []val) ctl {
			if !it.inBuiltin {
				currentLine = line
			}
			for i, cond := range conds {
				if cond == nil || cond(it, sl) {
					return runStmts(it, sl, bodies[i])
				}
			}
			return ctlNext
		}

	case *ast.LoopStatement:
		if s.Kind == ast.LoopEach {
			return c.each(s, line)
		}
		return c.loop(s, line)

	case *ast.DataOpStatement:
		target := c.name(s.Target)
		var v, idx, idx2 exprFn
		if s.Value != nil {
			v = c.expr(s.Value)
		}
		if s.Index != nil {
			idx = c.expr(s.Index)
		}
		if s.Index2 != nil {
			idx2 = c.expr(s.Index2)
		}
		part := func(f exprFn, it *Interpreter, sl []val) object.Object {
			if f == nil {
				return nil
			}
			return f(it, sl).box()
		}
		return func(it *Interpreter, sl []val) ctl {
			if !it.inBuiltin {
				currentLine = line
			}
			t := target(it, sl).box()
			dataOp(s, t, part(v, it, sl), part(idx, it, sl), part(idx2, it, sl))
			return ctlNext
		}

	case *ast.ShowStatement:
		exprs := c.exprs(s.Expressions)
		return func(it *Interpreter, sl []val) ctl {
			if !it.inBuiltin {
				currentLine = line
			}
			if s.Stderr {
				requireModule(it.vmEnv, "system", "warn")
			}
			var buf [4]object.Object
			vals := buf[:0]
			for _, f := range exprs {
				vals = append(vals, f(it, sl).box())
			}
			it.showValues(s, vals, it.vmEnv)
			return ctlNext
		}

	case *ast.FailStatement:
		value := c.expr(s.Value)
		return func(it *Interpreter, sl []val) ctl {
			if !it.inBuiltin {
				currentLine = line
			}
			v := value(it, sl)
			fatalKind(kindCustom, "%s", v.box().Inspect())
			return ctlNext
		}

	case *ast.ExpressionStatement:
		if _, ok := s.Expression.(*ast.HypothesisExpression); ok {
			break
		}
		value, print := c.expr(s.Expression), s.Print
		return func(it *Interpreter, sl []val) ctl {
			if !it.inBuiltin {
				currentLine = line
			}
			v := value(it, sl)
			if print {
				fmt.Fprintln(stdoutBuf, v.box().Inspect())
			}
			return ctlNext
		}
	}
	c.fail(fmt.Sprintf("statement %T", st))
	return nil
}

func (c *compiler) loop(s *ast.LoopStatement, line int) stmtFn {
	if s.Body == nil {
		c.fail("empty loop")
		return nil
	}
	saved := maps.Clone(c.defined)
	defer func() { c.defined = saved }()
	var init exprFn
	counter := -1
	if s.Kind == ast.LoopCStyle && s.Init != nil {
		a, ok := s.Init.(*ast.AssignStatement)
		if !ok || settingNames[a.Name] {
			c.fail("loop init")
			return nil
		}
		init = c.expr(a.Value) // worked out outside the loop's scope
		counter = c.n
		c.n++
		c.loops = append(c.loops, map[string]int{a.Name: counter})
		c.defined[counter] = true
		defer func() { c.loops = c.loops[:len(c.loops)-1] }()
	}
	var cond condFn
	if s.Condition != nil {
		cond = c.cond(s.Condition)
	}
	body := c.block(s.Body.Statements)
	var post stmtFn
	if s.Kind == ast.LoopCStyle && s.Post != nil {
		post = c.stmt(s.Post)
	}
	return func(it *Interpreter, sl []val) ctl {
		if !it.inBuiltin {
			currentLine = line
		}
		if init != nil {
			sl[counter] = init(it, sl)
		}
		for {
			if reason := it.stop.Load(); reason != stopNone {
				it.stopNow(reason)
			}
			if cond != nil && !cond(it, sl) {
				return ctlNext
			}
			switch runStmts(it, sl, body) {
			case ctlBreak:
				return ctlNext
			case ctlReturn:
				return ctlReturn
			}
			if post != nil {
				post(it, sl)
			}
		}
	}
}

// each is [loop][x in c] or [loop][a, b in c], as evalEach runs it: over
// a copy of c, its names in a scope of the loop's own.
func (c *compiler) each(s *ast.LoopStatement, line int) stmtFn {
	if s.Body == nil || len(s.Vars) < 1 || len(s.Vars) > 2 {
		c.fail("loop form")
		return nil
	}
	iter := c.expr(s.Iterable)
	saved := maps.Clone(c.defined)
	defer func() { c.defined = saved }()
	scope := map[string]int{}
	slots := make([]int, len(s.Vars))
	for i, v := range s.Vars {
		if settingNames[v] {
			c.fail("setting")
			return nil
		}
		slots[i] = c.n
		scope[v] = c.n
		c.defined[c.n] = true
		c.n++
	}
	c.loops = append(c.loops, scope)
	body := c.block(s.Body.Statements)
	c.loops = c.loops[:len(c.loops)-1]
	two := len(s.Vars) == 2
	return func(it *Interpreter, sl []val) ctl {
		if !it.inBuiltin {
			currentLine = line
		}
		var firsts, seconds []object.Object
		isMap := false
		switch v := iter(it, sl).box().(type) {
		case *object.List:
			seconds = slices.Clone(v.Elements)
		case *object.Set:
			seconds = slices.Clone(v.Elements)
		case *object.String:
			for _, r := range v.Value {
				seconds = append(seconds, &object.String{Value: string(r)})
			}
		case *object.Map:
			isMap = true
			for _, me := range v.Entries() {
				firsts = append(firsts, me.Key)
				seconds = append(seconds, me.Val)
			}
		default:
			fatalf("'[loop][... in ...]' needs a list, set, map, or string, got %s", typeName(v))
		}
		walk := seconds
		if !two && isMap {
			walk = firsts
		}
		for i, x := range walk {
			if reason := it.stop.Load(); reason != stopNone {
				it.stopNow(reason)
			}
			if two {
				if isMap {
					sl[slots[0]] = val{o: firsts[i]}
				} else {
					sl[slots[0]] = val{i: int64(i)}
				}
				sl[slots[1]] = valOf(x)
			} else {
				sl[slots[0]] = valOf(x)
			}
			switch runStmts(it, sl, body) {
			case ctlBreak:
				return ctlNext
			case ctlReturn:
				return ctlReturn
			}
		}
		return ctlNext
	}
}

func (c *compiler) cond(e ast.Expression) condFn {
	switch x := e.(type) {
	case *ast.BooleanLiteral:
		b := x.Value
		return func(*Interpreter, []val) bool { return b }
	case *ast.PrefixExpression:
		if x.Operator == "!" {
			r := c.cond(x.Right)
			return func(it *Interpreter, s []val) bool { return !r(it, s) }
		}
	case *ast.InfixExpression:
		switch x.Kind {
		case ast.InfixAnd:
			l, r := c.cond(x.Left), c.cond(x.Right)
			return func(it *Interpreter, s []val) bool { return l(it, s) && r(it, s) }
		case ast.InfixOr:
			l, r := c.cond(x.Left), c.cond(x.Right)
			return func(it *Interpreter, s []val) bool { return l(it, s) || r(it, s) }
		}
		if cmp := c.compare(x); cmp != nil {
			return cmp
		}
	}
	v := c.expr(e)
	return func(it *Interpreter, s []val) bool { return v(it, s).truthy() }
}

// compare is a comparison as a condition, or nil when x isn't one.
func (c *compiler) compare(x *ast.InfixExpression) condFn {
	op := x.Operator
	var fast func(a, b int64) bool
	switch op {
	case "<":
		fast = func(a, b int64) bool { return a < b }
	case ">":
		fast = func(a, b int64) bool { return a > b }
	case "<=":
		fast = func(a, b int64) bool { return a <= b }
	case ">=":
		fast = func(a, b int64) bool { return a >= b }
	case "==":
		fast = func(a, b int64) bool { return a == b }
	case "!=":
		fast = func(a, b int64) bool { return a != b }
	default:
		return nil
	}
	l, r := c.expr(x.Left), c.expr(x.Right)
	// The usual ones written out, so the common case is one call less.
	if op == "<" {
		return func(it *Interpreter, s []val) bool {
			a, b := l(it, s), r(it, s)
			if a.o == nil && b.o == nil {
				return a.i < b.i
			}
			return isTruthy(evalInfix(op, a.box(), b.box()))
		}
	}
	return func(it *Interpreter, s []val) bool {
		a, b := l(it, s), r(it, s)
		if a.o == nil && b.o == nil {
			return fast(a.i, b.i)
		}
		return isTruthy(evalInfix(op, a.box(), b.box()))
	}
}

func (c *compiler) expr(e ast.Expression) exprFn {
	return c.exprIn(e, false)
}

// exprIn compiles e; numeric is whether it's inside arithmetic the tree
// works out as plain numbers (evalNumeric), whose slow path it then takes.
func (c *compiler) exprIn(e ast.Expression, numeric bool) exprFn {
	switch x := e.(type) {
	case *ast.IntegerLiteral:
		v := val{i: x.Value}
		if x.Big != nil {
			v = val{o: object.IntOf(x.Big)}
		}
		return func(*Interpreter, []val) val { return v }
	case *ast.FloatLiteral:
		v := val{o: &object.Float{Value: x.Value}}
		return func(*Interpreter, []val) val { return v }
	case *ast.StringLiteral:
		v := val{o: &object.String{Value: x.Value}}
		return func(*Interpreter, []val) val { return v }
	case *ast.BooleanLiteral:
		v := val{o: object.Bool(x.Value)}
		return func(*Interpreter, []val) val { return v }
	case *ast.NoneLiteral:
		v := val{o: object.NoneValue}
		return func(*Interpreter, []val) val { return v }

	case *ast.Identifier:
		if x.Module != "" {
			break // "n f", which may read a local
		}
		slot, ok := c.resolve(x.Value)
		if ok && c.defined[slot] {
			return func(_ *Interpreter, s []val) val { return s[slot] }
		}
		if ok || c.assigned[x.Value] {
			slot = c.local(x.Value)
			// The function's own once set; until then the outer one.
			return func(it *Interpreter, s []val) val {
				if v := s[slot]; v != unset {
					return v
				}
				return valOf(it.evalExpression(x, it.vmEnv))
			}
		}
		// An outer name (a global, pi, a function as a value): as the
		// tree reads it, from the scope the function was made in.
		return func(it *Interpreter, _ []val) val { return valOf(it.evalExpression(x, it.vmEnv)) }

	case *ast.PrefixExpression:
		r := c.expr(x.Right)
		op := x.Operator
		if op == "-" {
			return func(it *Interpreter, s []val) val {
				v := r(it, s)
				if v.o == nil && v.i != math.MinInt64 {
					return val{i: -v.i}
				}
				return valOf(evalPrefix(op, v.box()))
			}
		}
		return func(it *Interpreter, s []val) val { return valOf(evalPrefix(op, r(it, s).box())) }

	case *ast.InfixExpression:
		switch x.Kind {
		case ast.InfixAnd, ast.InfixOr:
			cond := c.cond(x)
			return func(it *Interpreter, s []val) val { return val{o: object.Bool(cond(it, s))} }
		}
		if cmp := c.compare(x); cmp != nil {
			return func(it *Interpreter, s []val) val { return val{o: object.Bool(cmp(it, s))} }
		}
		inNumeric := numeric || x.Kind == ast.InfixNumeric
		l, r := c.exprIn(x.Left, inNumeric), c.exprIn(x.Right, inNumeric)
		return arith(x.Operator, l, r, inNumeric)

	case *ast.CallExpression:
		return c.call(x, false)

	case *ast.MethodCallExpression:
		if x.Hint != "" {
			break
		}
		recv := c.expr(x.Receiver)
		args := c.exprs(x.Arguments)
		return func(it *Interpreter, s []val) val {
			r := recv(it, s).box()
			var buf [2]object.Object
			a := buf[:0]
			for _, f := range args {
				a = append(a, f(it, s).box())
			}
			return valOf(it.applyMethod(x, r, a, it.vmEnv))
		}

	case *ast.ListLiteral:
		els := c.exprs(x.Elements)
		return func(it *Interpreter, s []val) val {
			l := &object.List{Elements: make([]object.Object, len(els))}
			for i, f := range els {
				l.Elements[i] = f(it, s).box()
			}
			return val{o: l}
		}
	case *ast.SetLiteral:
		els := c.exprs(x.Elements)
		return func(it *Interpreter, s []val) val {
			set := &object.Set{}
			for _, f := range els {
				set.Add(f(it, s).box())
			}
			return val{o: set}
		}
	case *ast.MapLiteral:
		keys, vals := c.exprs(x.Keys), c.exprs(x.Values)
		return func(it *Interpreter, s []val) val {
			m := object.NewMap()
			for i, k := range keys {
				m.Put(k(it, s).box(), vals[i](it, s).box())
			}
			return val{o: m}
		}
	case *ast.InterpolatedString:
		parts := c.exprs(x.Parts)
		return func(it *Interpreter, s []val) val {
			var b strings.Builder
			for _, f := range parts {
				v := f(it, s).box()
				writeShown(&b, v, v.Inspect())
			}
			return val{o: &object.String{Value: b.String()}}
		}
	case *ast.LengthExpression:
		arg := c.expr(x.Arg)
		return func(it *Interpreter, s []val) val { return val{i: int64(it.lengthOf(arg(it, s).box()))} }
	case *ast.ChangeExpression:
		if x.TypeName == "matrix" {
			break
		}
		src, to := c.expr(x.Source), x.TypeName
		return func(it *Interpreter, s []val) val { return valOf(evalChange(to, src(it, s).box())) }
	}
	if id, ok := e.(*ast.Identifier); ok {
		c.fail("name not local: " + id.Value)
		return nil
	}
	c.fail(fmt.Sprintf("expression %T", e))
	return nil
}

// name is the variable a data statement works on, as getVar finds it.
func (c *compiler) name(n string) exprFn {
	slot, ok := c.resolve(n)
	if ok && c.defined[slot] {
		return func(_ *Interpreter, s []val) val { return s[slot] }
	}
	if ok || c.assigned[n] {
		slot = c.local(n)
		return func(it *Interpreter, s []val) val {
			if v := s[slot]; v != unset {
				return v
			}
			return valOf(it.getVar(it.vmEnv, n))
		}
	}
	return func(it *Interpreter, _ []val) val { return valOf(it.getVar(it.vmEnv, n)) }
}

func (c *compiler) exprs(es []ast.Expression) []exprFn {
	out := make([]exprFn, len(es))
	for i, e := range es {
		out[i] = c.expr(e)
	}
	return out
}

// slowInfix is op on values that aren't both plain whole numbers in
// int64 range, the way the tree works it out.
func slowInfix(op string, a, b val, numeric bool) val {
	if numeric {
		an, aok := asNum(a.box())
		bn, bok := asNum(b.box())
		if aok && bok {
			n := numArith(op, an, bn)
			if n.isInt && n.big == nil {
				return val{i: n.i}
			}
			return val{o: n.box()}
		}
	}
	return valOf(evalInfix(op, a.box(), b.box()))
}

func arith(op string, l, r exprFn, numeric bool) exprFn {
	switch op {
	case "+":
		return func(it *Interpreter, s []val) val {
			a, b := l(it, s), r(it, s)
			if a.o == nil && b.o == nil {
				if sum := a.i + b.i; (sum > a.i) == (b.i > 0) {
					return val{i: sum}
				}
			}
			return slowInfix(op, a, b, numeric)
		}
	case "-":
		return func(it *Interpreter, s []val) val {
			a, b := l(it, s), r(it, s)
			if a.o == nil && b.o == nil {
				if d := a.i - b.i; (d < a.i) == (b.i > 0) {
					return val{i: d}
				}
			}
			return slowInfix(op, a, b, numeric)
		}
	case "*":
		return func(it *Interpreter, s []val) val {
			a, b := l(it, s), r(it, s)
			if a.o == nil && b.o == nil {
				if p, ok := mulFits(a.i, b.i); ok {
					return val{i: p}
				}
			}
			return slowInfix(op, a, b, numeric)
		}
	case "%":
		return func(it *Interpreter, s []val) val {
			a, b := l(it, s), r(it, s)
			if a.o == nil && b.o == nil && b.i != 0 {
				return val{i: a.i % b.i}
			}
			return slowInfix(op, a, b, numeric)
		}
	}
	return func(it *Interpreter, s []val) val {
		return slowInfix(op, l(it, s), r(it, s), numeric)
	}
}

// call is a call by name: to a compiled body, its arguments go straight
// into the new frame; to anything else, as the tree calls it.
func (c *compiler) call(ce *ast.CallExpression, statement bool) exprFn {
	if ce.Subject != nil || ce.Module != "" || ce.Name == "diagnose" {
		c.fail("call form")
		return nil
	}
	if _, local := c.resolve(ce.Name); local || c.assigned[ce.Name] {
		c.fail("call of a variable") // a variable holding a function
		return nil
	}
	args := make([]exprFn, len(ce.Arguments))
	for i, a := range ce.Arguments {
		args[i] = c.expr(a)
	}
	n := len(args)
	return func(it *Interpreter, s []val) val {
		fn := knownFunction(ce, it.vmEnv)
		if fn != nil && fn.Shape == nil && fn.Scroll == nil && len(fn.Parameters) == n {
			if cb := it.compiledBody(fn); cb != nil {
				st := &it.vm
				m := st.mark()
				frame := st.alloc(cb.nslots)
				for i, a := range args {
					frame[i] = a(it, s)
				}
				newFrame(frame, cb.nparams)
				r := it.vmEnter(fn, cb, ce.Name, frame)
				clear(frame)
				st.release(m)
				return r
			}
		}
		boxed := make([]object.Object, n)
		for i, a := range args {
			boxed[i] = a(it, s).box()
		}
		if statement {
			it.inPlace = true
			defer func() { it.inPlace = false }()
		}
		return valOf(it.applyCall(ce, boxed, it.vmEnv))
	}
}
