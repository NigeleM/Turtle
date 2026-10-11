// Copyright 2017-2026 Nigele McCoy
// Licensed under the Apache License, Version 2.0;
// see LICENSE and NOTICE.

package lsp

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"

	"Turtle/evaluator"
	"Turtle/token"
)

// session sends the server messages, as an editor would, and gives back
// what it answered: responses by id, and notifications in order.
type session struct {
	t       *testing.T
	in      bytes.Buffer
	nextID  int
	results map[int]json.RawMessage
	errors  map[int]rpcError
	notes   []message
	code    int
}

func newSession(t *testing.T) *session {
	return &session{t: t, results: map[int]json.RawMessage{}, errors: map[int]rpcError{}}
}

func (s *session) send(method string, params any, request bool) int {
	m := map[string]any{"jsonrpc": "2.0", "method": method}
	if params != nil {
		m["params"] = params
	}
	id := 0
	if request {
		s.nextID++
		id = s.nextID
		m["id"] = id
	}
	data, _ := json.Marshal(m)
	fmt.Fprintf(&s.in, "Content-Length: %d\r\n\r\n%s", len(data), data)
	return id
}

func (s *session) request(method string, params any) int { return s.send(method, params, true) }
func (s *session) notify(method string, params any)      { s.send(method, params, false) }

// run sends everything to a server, then reads all it wrote.
func (s *session) run() {
	var out bytes.Buffer
	s.code = Serve(&s.in, &out, "test")
	r := bufio.NewReader(&out)
	for {
		length := -1
		for {
			line, err := r.ReadString('\n')
			if err != nil {
				return
			}
			line = strings.TrimSpace(line)
			if line == "" {
				break
			}
			if v, ok := strings.CutPrefix(line, "Content-Length:"); ok {
				length, _ = strconv.Atoi(strings.TrimSpace(v))
			}
		}
		body := make([]byte, length)
		io.ReadFull(r, body)
		var m struct {
			ID     *int            `json:"id"`
			Method string          `json:"method"`
			Params json.RawMessage `json:"params"`
			Result json.RawMessage `json:"result"`
			Error  *rpcError       `json:"error"`
		}
		if err := json.Unmarshal(body, &m); err != nil {
			s.t.Fatalf("bad message %q: %v", body, err)
		}
		switch {
		case m.ID != nil && m.Error != nil:
			s.errors[*m.ID] = *m.Error
		case m.ID != nil:
			s.results[*m.ID] = m.Result
		default:
			s.notes = append(s.notes, message{Method: m.Method, Params: m.Params})
		}
	}
}

func (s *session) result(id int, v any) {
	s.t.Helper()
	raw, ok := s.results[id]
	if !ok {
		s.t.Fatalf("no result for request %d (errors %v)", id, s.errors)
	}
	if err := json.Unmarshal(raw, v); err != nil {
		s.t.Fatalf("result %d: %v (%s)", id, err, raw)
	}
}

func (s *session) start(utf8 bool) {
	caps := map[string]any{}
	if utf8 {
		caps["general"] = map[string]any{"positionEncodings": []string{"utf-8", "utf-16"}}
	}
	s.request("initialize", map[string]any{"capabilities": caps})
	s.notify("initialized", map[string]any{})
}

func (s *session) open(uri, text string) {
	s.notify("textDocument/didOpen", map[string]any{"textDocument": map[string]any{"uri": uri, "languageId": "turtle", "version": 1, "text": text}})
}

func (s *session) end() {
	s.request("shutdown", nil)
	s.notify("exit", nil)
	s.run()
}

func at(uri string, line, char int) map[string]any {
	return map[string]any{"textDocument": map[string]any{"uri": uri}, "position": map[string]any{"line": line, "character": char}}
}

// diagnostics are the last diagnostics published for uri.
func (s *session) diagnostics(uri string) []diagnostic {
	var out []diagnostic
	for _, n := range s.notes {
		if n.Method != "textDocument/publishDiagnostics" {
			continue
		}
		var p struct {
			URI         string       `json:"uri"`
			Diagnostics []diagnostic `json:"diagnostics"`
		}
		json.Unmarshal(n.Params, &p)
		if p.URI == uri {
			out = p.Diagnostics
		}
	}
	return out
}

const program = `import sql
import lib/shapes

// double makes x twice as big.
def double[x]
    return x * 2
def [end]

// Order is one line of a sale.
assemble Order [item, qty]

total = double[21]
nums = list [1, 2]
n = nums at length
db = sql_open["a.db"]
`

func folder(t *testing.T) (string, string) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "lib"), 0o755)
	os.WriteFile(filepath.Join(dir, "lib", "shapes.trt"), []byte("// area of a rectangle.\ndef area[w, h]\n    return w * h\ndef [end]\n"), 0o644)
	return dir, fileURI(filepath.Join(dir, "main.trt"))
}

func TestInitializeAndShutdown(t *testing.T) {
	s := newSession(t)
	s.start(true)
	s.end()
	var r struct {
		Capabilities map[string]any        `json:"capabilities"`
		ServerInfo   struct{ Name string } `json:"serverInfo"`
	}
	s.result(1, &r)
	if r.Capabilities["positionEncoding"] != "utf-8" || r.ServerInfo.Name != "turtle" || r.Capabilities["hoverProvider"] != true {
		t.Errorf("capabilities: %+v", r)
	}
	if s.code != 0 {
		t.Errorf("exit code after shutdown: %d", s.code)
	}
	// Without shutdown first, exit is an error exit; requests before
	// initialize are refused.
	s2 := newSession(t)
	id := s2.request("textDocument/hover", at("file:///x.trt", 0, 0))
	s2.notify("exit", nil)
	s2.run()
	if s2.code != 1 || s2.errors[id].Code != errNotInitialized {
		t.Errorf("code %d, errors %v", s2.code, s2.errors)
	}
}

func TestDiagnostics(t *testing.T) {
	dir, uri := folder(t)
	s := newSession(t)
	s.start(false)
	s.open(uri, program)
	bad := fileURI(filepath.Join(dir, "bad.trt"))
	s.open(bad, "x = 1\nshow x\nimport nowhere\n")
	s.end()
	if d := s.diagnostics(uri); len(d) != 0 {
		t.Errorf("a good file has no errors: %+v", d)
	}
	d := s.diagnostics(bad)
	if len(d) != 2 {
		t.Fatalf("want a parse error and a missing import, got %+v", d)
	}
	if d[0].Severity != severityError || d[0].Range.Start.Line != 1 || !strings.Contains(d[0].Message, "this line needs a '.' at the end") {
		t.Errorf("parse error: %+v", d[0])
	}
	if d[1].Severity != severityWarning || d[1].Range.Start.Line != 2 || !strings.Contains(d[1].Message, "no library called nowhere, and no file nowhere.turtle (or .trt)") {
		t.Errorf("missing import: %+v", d[1])
	}
}

func TestDiagnosticsFollowChanges(t *testing.T) {
	_, uri := folder(t)
	s := newSession(t)
	s.start(false)
	s.open(uri, "show 1\n")
	s.notify("textDocument/didChange", map[string]any{"textDocument": map[string]any{"uri": uri, "version": 2}, "contentChanges": []map[string]any{{"text": "show 1 .\n"}}})
	s.end()
	if d := s.diagnostics(uri); len(d) != 0 {
		t.Errorf("fixed file still has errors: %+v", d)
	}
}

func labels(items []completionItem) map[string]completionItem {
	out := map[string]completionItem{}
	for _, i := range items {
		out[i.Label] = i
	}
	return out
}

func TestCompletion(t *testing.T) {
	_, uri := folder(t)
	s := newSession(t)
	s.start(false)
	s.open(uri, program+"x = nums at \nimport \nnext = do\n")
	general := s.request("textDocument/completion", at(uri, 18, 9))
	methods := s.request("textDocument/completion", at(uri, 15, 12))
	imports := s.request("textDocument/completion", at(uri, 16, 7))
	s.end()
	var g []completionItem
	s.result(general, &g)
	got := labels(g)
	for name, kind := range map[string]int{"double": completionFunction, "Order": completionClass, "area": completionFunction, "total": completionVariable, "show": completionKeyword, "give": completionKeyword, "sql_query": completionFunction} {
		if got[name].Kind != kind {
			t.Errorf("%s: kind %d, want %d", name, got[name].Kind, kind)
		}
	}
	if got["double"].Detail != "double[x]" || got["double"].Documentation != "double makes x twice as big." {
		t.Errorf("double: %+v", got["double"])
	}
	if got["sql_query"].Detail != "import sql" || got["sql_query"].SortText >= got["min_sort"].SortText {
		t.Errorf("an imported library's functions come first: %+v %+v", got["sql_query"], got["min_sort"])
	}
	var m []completionItem
	s.result(methods, &m)
	if ms := labels(m); ms["upper"].Kind != completionMethod || ms["double"].Label != "" {
		t.Errorf("after at, methods only: %d items", len(m))
	}
	var im []completionItem
	s.result(imports, &im)
	if is := labels(im); is["sql"].Kind != completionModule || is["lib/shapes"].Kind != completionFile {
		t.Errorf("after import: %v", is)
	}
}

func TestHover(t *testing.T) {
	_, uri := folder(t)
	s := newSession(t)
	s.start(false)
	s.open(uri, program+"a = area[2, 3]\n")
	user := s.request("textDocument/hover", at(uri, 11, 10))
	lib := s.request("textDocument/hover", at(uri, 14, 7))
	kw := s.request("textDocument/hover", at(uri, 4, 1))
	imported := s.request("textDocument/hover", at(uri, 15, 5))
	nothing := s.request("textDocument/hover", at(uri, 12, 0))
	s.end()
	text := func(id int) string {
		var h *struct{ Contents markup }
		s.result(id, &h)
		if h == nil {
			return ""
		}
		return h.Contents.Value
	}
	if v := text(user); !strings.Contains(v, "def double[x]") || !strings.Contains(v, "double makes x twice as big.") {
		t.Errorf("user function: %q", v)
	}
	if v := text(lib); !strings.Contains(v, "sql_open[address]") || !strings.Contains(v, "_import sql_") {
		t.Errorf("library function: %q", v)
	}
	if v := text(kw); !strings.Contains(v, "**def**") {
		t.Errorf("keyword: %q", v)
	}
	if v := text(imported); !strings.Contains(v, "def area[w, h]") || !strings.Contains(v, "_from shapes.trt_") {
		t.Errorf("imported function: %q", v)
	}
	if v := text(nothing); v != "" && !strings.Contains(v, "nums") {
		t.Errorf("a plain variable: %q", v)
	}
}

func TestDefinitionAndSymbols(t *testing.T) {
	dir, uri := folder(t)
	s := newSession(t)
	s.start(false)
	s.open(uri, program+"a = area[2, 3]\n")
	here := s.request("textDocument/definition", at(uri, 11, 9))
	there := s.request("textDocument/definition", at(uri, 15, 5))
	syms := s.request("textDocument/documentSymbol", map[string]any{"textDocument": map[string]any{"uri": uri}})
	s.end()
	var l location
	s.result(here, &l)
	if l.URI != uri || l.Range.Start.Line != 4 || l.Range.Start.Character != 4 || l.Range.End.Character != 10 {
		t.Errorf("double is defined on line 5: %+v", l)
	}
	s.result(there, &l)
	if l.URI != fileURI(filepath.Join(dir, "lib", "shapes.trt")) || l.Range.Start.Line != 1 {
		t.Errorf("area is in lib/shapes.trt: %+v", l)
	}
	var ds []documentSymbol
	s.result(syms, &ds)
	var names []string
	for _, d := range ds {
		names = append(names, d.Name)
	}
	if strings.Join(names, ",") != "double,Order,total,nums,n,db,a" {
		t.Errorf("outline: %v", names)
	}
}

func TestSemanticTokens(t *testing.T) {
	_, uri := folder(t)
	s := newSession(t)
	s.start(false)
	s.open(uri, "def f[x] // é\n    return \"é\" + 1\n//* a\nb *//\n")
	id := s.request("textDocument/semanticTokens/full", map[string]any{"textDocument": map[string]any{"uri": uri}})
	s.end()
	var r struct{ Data []int }
	s.result(id, &r)
	var got [][]int
	for i := 0; i+5 <= len(r.Data); i += 5 {
		got = append(got, r.Data[i:i+5])
	}
	want := [][]int{
		{0, 0, 3, 0, 0}, // def
		{0, 4, 1, 4, 1}, // f, a definition
		{0, 5, 4, 3, 0}, // // é (UTF-16: 4 units)
		{1, 4, 6, 0, 0}, // return
		{0, 7, 3, 1, 0}, // "é"
		{0, 6, 1, 2, 0}, // 1
		{1, 0, 5, 3, 0}, // //* a
		{1, 0, 5, 3, 0}, // b *//
	}
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Errorf("got  %v\nwant %v", got, want)
	}
}

// An editor that lists Turtle's own groups gets them; one that doesn't
// gets the six every editor takes, as before.
func TestSemanticTokenGroups(t *testing.T) {
	src := "import data\nassemble Sale [item]\ntheory twice\n    abstract\n        twice doubles n.\n    notation twice n .\n    definition\n        return n * 2\ntheory [end]\ndef test_x[]\n    show twice 2 . at len\ndef [end]\n"
	kinds := func(groups bool) []string {
		_, uri := folder(t)
		s := newSession(t)
		caps := map[string]any{}
		if groups {
			caps["textDocument"] = map[string]any{"semanticTokens": map[string]any{"tokenTypes": tokenTypes, "tokenModifiers": tokenModifiers}}
		}
		s.request("initialize", map[string]any{"capabilities": caps})
		s.notify("initialized", map[string]any{})
		s.open(uri, src)
		id := s.request("textDocument/semanticTokens/full", map[string]any{"textDocument": map[string]any{"uri": uri}})
		s.end()
		var r struct{ Data []int }
		s.result(id, &r)
		var out []string
		for i := 0; i+5 <= len(r.Data); i += 5 {
			k := tokenTypes[r.Data[i+3]]
			for b, m := range tokenModifiers {
				if r.Data[i+4]&(1<<b) != 0 {
					k += "." + m
				}
			}
			out = append(out, k)
		}
		return out
	}
	with := strings.Join(kinds(true), " ")
	for _, want := range []string{"namespace namespace", "struct struct.declaration", "macro macro macro", "function.declaration.test", "event macro", "method"} {
		if !strings.Contains(with, want) {
			t.Errorf("with groups: missing %q in %s", want, with)
		}
	}
	for _, k := range kinds(false) {
		base, _, _ := strings.Cut(k, ".")
		if base != "keyword" && base != "string" && base != "number" && base != "comment" && base != "function" && base != "type" || strings.Contains(k, "test") {
			t.Errorf("without groups: an editor got %q", k)
		}
	}
}

// testdata/colors/colors.turtle has every color group, and the server
// finds each one in it.
func TestColorsFileHasEveryGroup(t *testing.T) {
	path, err := filepath.Abs("../testdata/colors/colors.turtle")
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	s := newSession(t)
	s.request("initialize", map[string]any{"capabilities": map[string]any{"textDocument": map[string]any{"semanticTokens": map[string]any{"tokenTypes": tokenTypes, "tokenModifiers": tokenModifiers}}}})
	s.notify("initialized", map[string]any{})
	uri := fileURI(path)
	s.open(uri, strings.ReplaceAll(string(data), "\r\n", "\n"))
	id := s.request("textDocument/semanticTokens/full", map[string]any{"textDocument": map[string]any{"uri": uri}})
	s.end()
	var r struct{ Data []int }
	s.result(id, &r)
	seen := map[string]bool{}
	for i := 0; i+5 <= len(r.Data); i += 5 {
		k := tokenTypes[r.Data[i+3]]
		seen[k] = true
		for b, m := range tokenModifiers {
			if r.Data[i+4]&(1<<b) != 0 {
				seen[k+"."+m] = true
			}
		}
	}
	for _, want := range []string{"keyword", "string", "number", "comment", "function", "namespace", "struct", "event", "macro", "method",
		"function.declaration", "function.defaultLibrary", "function.test", "struct.declaration"} {
		if !seen[want] {
			t.Errorf("colors.turtle: no %s token", want)
		}
	}
}

func TestPositions(t *testing.T) {
	src := "aé𝄞b\nxy"
	u16 := newText(src, false)
	if p := u16.position(strings.Index(src, "b")); p != (position{0, 4}) {
		t.Errorf("utf-16: %+v (é is 1 unit, 𝄞 is 2)", p)
	}
	if o := u16.offset(position{0, 4}); src[o] != 'b' {
		t.Errorf("utf-16 offset: %d", o)
	}
	u8 := newText(src, true)
	if p := u8.position(strings.Index(src, "b")); p != (position{0, 7}) {
		t.Errorf("utf-8: %+v", p)
	}
	if p := u8.position(len(src)); p != (position{1, 2}) {
		t.Errorf("end: %+v", p)
	}
	if w, start := u8.wordAt(len("aé𝄞b\nx")); w != "xy" || start != len("aé𝄞b\n") {
		t.Errorf("wordAt: %q %d", w, start)
	}
	// A real path on this system (C:\... on Windows), with a space.
	path := filepath.Join(t.TempDir(), "a b", "c.trt")
	if got := uriPath(fileURI(path)); got != path {
		t.Errorf("uri round trip: %q, want %q", got, path)
	}
}

// TestVSCodeGrammarIsCurrent: the VS Code grammar colors exactly
// Turtle's keywords and library functions. If this fails after adding a
// keyword or a library function, add it to
// editors/vscode/syntaxes/turtle.tmLanguage.json.
func TestVSCodeGrammarIsCurrent(t *testing.T) {
	data, err := os.ReadFile("../editors/vscode/syntaxes/turtle.tmLanguage.json")
	if err != nil {
		t.Fatal(err)
	}
	var g struct {
		Repository map[string]struct {
			Match    string `json:"match"`
			Patterns []struct {
				Match string `json:"match"`
			} `json:"patterns"`
		} `json:"repository"`
	}
	if err := json.Unmarshal(data, &g); err != nil {
		t.Fatal(err)
	}
	wordsIn := func(pattern string) map[string]bool {
		out := map[string]bool{}
		inner := pattern[strings.Index(pattern, "(")+1 : strings.Index(pattern, ")")]
		for _, w := range strings.Split(inner, "|") {
			out[w] = true
		}
		return out
	}
	grammar := map[string]bool{}
	for _, p := range g.Repository["keywords"].Patterns {
		for w := range wordsIn(p.Match) {
			grammar[w] = true
		}
	}
	for w := range wordsIn(g.Repository["constants"].Match) {
		grammar[w] = true
	}
	want := map[string]bool{}
	for _, k := range token.Keywords() {
		want[k] = true
	}
	if fmt.Sprint(sortedKeys(grammar)) != fmt.Sprint(sortedKeys(want)) {
		t.Errorf("grammar keywords %v\nTurtle's          %v", sortedKeys(grammar), sortedKeys(want))
	}
	lib := wordsIn(g.Repository["library"].Match)
	funcs := map[string]bool{}
	for _, f := range evaluator.LibraryFunctions() {
		funcs[f] = true
	}
	if fmt.Sprint(sortedKeys(lib)) != fmt.Sprint(sortedKeys(funcs)) {
		t.Errorf("grammar library functions %v\nTurtle's %v", sortedKeys(lib), sortedKeys(funcs))
	}
}

func sortedKeys(m map[string]bool) []string {
	var out []string
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// TestHoverDocInsideAndBlocks: a function's description can be above it
// or first in its body, as // lines or a //* *// block.
func TestHoverDocInsideAndBlocks(t *testing.T) {
	_, uri := folder(t)
	s := newSession(t)
	s.start(false)
	s.open(uri, `//*
  run calls hello.
*//
def run[c]
    // c is any number.
    show c .
def [end]

def quiet[]
    x = 1 // not a description
def [end]

run[1]
quiet[]
`)
	runHover := s.request("textDocument/hover", at(uri, 12, 1))
	quietHover := s.request("textDocument/hover", at(uri, 13, 1))
	s.end()
	text := func(id int) string {
		var h *struct{ Contents markup }
		s.result(id, &h)
		if h == nil {
			return ""
		}
		return h.Contents.Value
	}
	if v := text(runHover); !strings.Contains(v, "run calls hello.\nc is any number.") {
		t.Errorf("run: %q", v)
	}
	if v := text(quietHover); strings.Contains(v, "not a description") {
		t.Errorf("quiet: %q", v)
	}
}

// A private function's name, ~limit, is one word, for hover and go to
// definition.
func TestWordAtPrivateName(t *testing.T) {
	tx := newText("x = ~limit[3]", true)
	if w, start := tx.wordAt(7); w != "~limit" || start != 4 {
		t.Errorf("got %q at %d", w, start)
	}
}

// A theory, this file's or an imported one: hover shows how it's
// written and its abstract; completion offers its word; go to definition
// finds it.
func TestTheorySymbols(t *testing.T) {
	src := "theory tally\n    abstract\n        tally says how many times b is in a.\n    notation tally b in a .\n    definition\n        return 0\ntheory [end]\nshow tally 1 in list [1] .\n"
	tx := newText(src, true)
	syms := definitions("file:///x.turtle", src, tx, true)
	if len(syms) == 0 || syms[0].name != "tally" || syms[0].kind != symTheory || syms[0].detail != "tally b in a ." || syms[0].doc != "tally says how many times b is in a." {
		t.Fatalf("got %+v", syms)
	}
}

// A standard library's own Turtle file reads as Turtle reads it: its
// library's sentences (random integer from ...) aren't errors there.
func TestStandardLibraryFiles(t *testing.T) {
	path, err := filepath.Abs("../evaluator/lib/random.turtle")
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	src := strings.ReplaceAll(string(data), "\r\n", "\n")
	if a := analyze(fileURI(path), path, src, newText(src, false)); len(a.diags) > 0 {
		t.Errorf("random.turtle: %v", a.diags)
	}
	// The same lines in an ordinary file are still an error without the import.
	other := filepath.Join(t.TempDir(), "mine.turtle")
	if a := analyze(fileURI(other), other, "x = random integer from 0 to 3\n", newText("x = random integer from 0 to 3\n", false)); len(a.diags) == 0 {
		t.Errorf("an ordinary file without import random: want an error")
	}
}
