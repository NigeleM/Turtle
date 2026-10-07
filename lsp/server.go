// Package lsp is turtle lsp: a language server, so editors that speak the
// Language Server Protocol (VS Code, Neovim, Helix, Zed, Sublime ...) know
// Turtle: errors as you type, colors, completion, hover help, go to
// definition and an outline. It reads and writes JSON-RPC messages on
// stdin and stdout, using only Go's standard library.
package lsp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"

	"Turtle/evaluator"
	"Turtle/format"
	"Turtle/syntax"
	"Turtle/token"
)

// document is an open file: its text and what the server worked out.
type document struct {
	uri, path string
	text      *text
	analysis  *analysis
}

// Server is one editor's connection.
type Server struct {
	in          *bufio.Reader
	out         io.Writer
	mu          sync.Mutex
	docs        map[string]*document
	utf8        bool
	initialized bool
	shutdown    bool
	version     string
}

// Serve runs the server on in and out until the editor says exit. It
// returns the exit code: 0 after a shutdown request, 1 without one.
func Serve(in io.Reader, out io.Writer, version string) int {
	s := &Server{in: bufio.NewReader(in), out: out, docs: map[string]*document{}, version: version}
	for {
		msg, err := s.read()
		if err != nil {
			if s.shutdown {
				return 0
			}
			return 1
		}
		if msg.Method == "exit" {
			if s.shutdown {
				return 0
			}
			return 1
		}
		s.handle(msg)
	}
}

// read reads one message: headers, a blank line, then Content-Length
// bytes of JSON.
func (s *Server) read() (*message, error) {
	length := -1
	for {
		line, err := s.in.ReadString('\n')
		if err != nil {
			return nil, err
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			break
		}
		if v, ok := strings.CutPrefix(line, "Content-Length:"); ok {
			length, err = strconv.Atoi(strings.TrimSpace(v))
			if err != nil {
				return nil, err
			}
		}
	}
	if length < 0 {
		return nil, fmt.Errorf("a message without Content-Length")
	}
	body := make([]byte, length)
	if _, err := io.ReadFull(s.in, body); err != nil {
		return nil, err
	}
	var msg message
	if err := json.Unmarshal(body, &msg); err != nil {
		return &message{}, nil // not JSON: ignore it
	}
	return &msg, nil
}

func (s *Server) write(v any) {
	data, _ := json.Marshal(v)
	s.mu.Lock()
	defer s.mu.Unlock()
	fmt.Fprintf(s.out, "Content-Length: %d\r\n\r\n%s", len(data), data)
}

func (s *Server) reply(id json.RawMessage, result any) {
	s.write(map[string]any{"jsonrpc": "2.0", "id": id, "result": result})
}

func (s *Server) fail(id json.RawMessage, code int, msg string) {
	s.write(map[string]any{"jsonrpc": "2.0", "id": id, "error": rpcError{Code: code, Message: msg}})
}

func (s *Server) notify(method string, params any) {
	s.write(map[string]any{"jsonrpc": "2.0", "method": method, "params": params})
}

func (s *Server) handle(msg *message) {
	isRequest := len(msg.ID) > 0
	if !s.initialized && msg.Method != "initialize" {
		if isRequest {
			s.fail(msg.ID, errNotInitialized, "initialize first")
		}
		return
	}
	switch msg.Method {
	case "initialize":
		s.initialize(msg)
	case "initialized", "$/cancelRequest", "$/setTrace", "textDocument/didSave", "workspace/didChangeConfiguration":
	case "shutdown":
		s.shutdown = true
		s.reply(msg.ID, nil)
	case "textDocument/didOpen":
		var p struct {
			TextDocument struct {
				URI  string `json:"uri"`
				Text string `json:"text"`
			} `json:"textDocument"`
		}
		if json.Unmarshal(msg.Params, &p) == nil {
			s.update(p.TextDocument.URI, p.TextDocument.Text)
		}
	case "textDocument/didChange":
		var p struct {
			TextDocument   textDocumentID `json:"textDocument"`
			ContentChanges []struct {
				Text string `json:"text"`
			} `json:"contentChanges"`
		}
		if json.Unmarshal(msg.Params, &p) == nil && len(p.ContentChanges) > 0 {
			// Full sync: each change is the whole text.
			s.update(p.TextDocument.URI, p.ContentChanges[len(p.ContentChanges)-1].Text)
		}
	case "textDocument/didClose":
		var p struct {
			TextDocument textDocumentID `json:"textDocument"`
		}
		if json.Unmarshal(msg.Params, &p) == nil {
			delete(s.docs, p.TextDocument.URI)
			s.notify("textDocument/publishDiagnostics", map[string]any{"uri": p.TextDocument.URI, "diagnostics": []diagnostic{}})
		}
	case "textDocument/completion":
		s.withPosition(msg, s.completion)
	case "textDocument/hover":
		s.withPosition(msg, s.hover)
	case "textDocument/definition":
		s.withPosition(msg, s.definition)
	case "textDocument/documentSymbol":
		s.withDocument(msg, s.documentSymbols)
	case "textDocument/semanticTokens/full":
		s.withDocument(msg, s.semanticTokens)
	case "textDocument/formatting":
		s.withDocument(msg, s.formatting)
	default:
		if isRequest {
			s.fail(msg.ID, errMethodNotFound, "turtle lsp doesn't do "+msg.Method)
		}
	}
}

func (s *Server) initialize(msg *message) {
	var p struct {
		Capabilities struct {
			General struct {
				PositionEncodings []string `json:"positionEncodings"`
			} `json:"general"`
		} `json:"capabilities"`
	}
	json.Unmarshal(msg.Params, &p)
	encoding := "utf-16"
	for _, e := range p.Capabilities.General.PositionEncodings {
		if e == "utf-8" {
			encoding, s.utf8 = "utf-8", true
		}
	}
	s.initialized = true
	s.reply(msg.ID, map[string]any{
		"capabilities": map[string]any{
			"positionEncoding":           encoding,
			"textDocumentSync":           map[string]any{"openClose": true, "change": 1},
			"completionProvider":         map[string]any{"triggerCharacters": []string{}},
			"hoverProvider":              true,
			"definitionProvider":         true,
			"documentSymbolProvider":     true,
			"documentFormattingProvider": true,
			"semanticTokensProvider": map[string]any{
				"legend": map[string]any{"tokenTypes": tokenTypes, "tokenModifiers": tokenModifiers},
				"full":   true,
			},
		},
		"serverInfo": map[string]any{"name": "turtle", "version": s.version},
	})
}

// update takes a document's new text, works it out, and sends its errors.
func (s *Server) update(uri, src string) {
	path := uriPath(uri)
	t := newText(src, s.utf8)
	d := &document{uri: uri, path: path, text: t, analysis: analyze(uri, path, src, t)}
	s.docs[uri] = d
	diags := d.analysis.diags
	if diags == nil {
		diags = []diagnostic{}
	}
	s.notify("textDocument/publishDiagnostics", map[string]any{"uri": uri, "diagnostics": diags})
}

func (s *Server) withDocument(msg *message, f func(*document) any) {
	var p struct {
		TextDocument textDocumentID `json:"textDocument"`
	}
	if json.Unmarshal(msg.Params, &p) != nil {
		s.fail(msg.ID, errInvalidParams, "no textDocument")
		return
	}
	d, ok := s.docs[p.TextDocument.URI]
	if !ok {
		s.reply(msg.ID, nil)
		return
	}
	s.reply(msg.ID, f(d))
}

func (s *Server) withPosition(msg *message, f func(*document, int) any) {
	var p positionParams
	if json.Unmarshal(msg.Params, &p) != nil {
		s.fail(msg.ID, errInvalidParams, "no position")
		return
	}
	d, ok := s.docs[p.TextDocument.URI]
	if !ok {
		s.reply(msg.ID, nil)
		return
	}
	s.reply(msg.ID, f(d, d.text.offset(p.Position)))
}

// ---- completion ----

func (s *Server) completion(d *document, offset int) any {
	a := d.analysis
	before := d.text.src[d.text.lines[d.text.position(offset).Line]:offset]
	word, start := d.text.wordAt(offset)
	_ = word
	prefix := strings.TrimRight(before[:len(before)-(offset-start)], " \t")
	var items []completionItem
	add := func(label string, kind int, detail, doc, sort string) {
		items = append(items, completionItem{Label: label, Kind: kind, Detail: detail, Documentation: doc, SortText: sort + label})
	}
	switch {
	case strings.HasSuffix(prefix, " at") || prefix == "at":
		for _, m := range syntax.Methods {
			add(m, completionMethod, "method", "", "1")
		}
		return items
	case strings.HasPrefix(strings.TrimLeft(before, " \t"), "import ") && !strings.ContainsAny(before, "[]"):
		for _, lib := range evaluator.Libraries() {
			add(lib, completionModule, "library", "", "1")
		}
		for _, f := range localModules(filepath.Dir(d.path)) {
			add(f, completionFile, "your file", "", "2")
		}
		return items
	case strings.HasSuffix(prefix, "random") && a.libs["random"]:
		for _, w := range syntax.ShapeWords {
			add(w, completionKeyword, "a kind of value", "", "1")
		}
		return items
	}
	seen := map[string]bool{}
	for _, sym := range a.symbols {
		if seen[sym.name] {
			continue
		}
		seen[sym.name] = true
		switch sym.kind {
		case symFunc:
			add(sym.name, completionFunction, sym.detail, sym.doc, "1")
		case symType:
			add(sym.name, completionClass, sym.detail, sym.doc, "1")
		case symVar:
			add(sym.name, completionVariable, "", "", "2")
		}
	}
	for name := range a.idents {
		if !seen[name] && name != word {
			seen[name] = true
			add(name, completionVariable, "", "", "2")
		}
	}
	for lib := range a.libs {
		for _, w := range syntax.LibraryWords[lib] {
			if !seen[w] {
				seen[w] = true
				add(w, completionKeyword, "import "+lib, "", "3")
			}
		}
	}
	for _, kw := range token.Keywords() {
		if !seen[kw] {
			add(kw, completionKeyword, "", keywordHelp[kw], "3")
		}
	}
	for _, f := range evaluator.LibraryFunctions() {
		if seen[f] {
			continue
		}
		call, mod, _, _ := evaluator.LibraryDoc(f)
		order := "5"
		if a.libs[mod] {
			order = "4"
		}
		add(f, completionFunction, "import "+mod, call, order)
	}
	return items
}

// localModules are the .trt files next to path (and in folders below),
// as import names: utils, lib/shop.
func localModules(dir string) []string {
	var out []string
	filepath.WalkDir(dir, func(p string, e os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if e.IsDir() && p != dir && (strings.HasPrefix(e.Name(), ".") || strings.Count(strings.TrimPrefix(p, dir), string(filepath.Separator)) > 3) {
			return filepath.SkipDir
		}
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".trt") {
			rel, _ := filepath.Rel(dir, p)
			out = append(out, strings.TrimSuffix(filepath.ToSlash(rel), ".trt"))
		}
		return nil
	})
	sort.Strings(out)
	return out
}

// ---- hover ----

func (s *Server) hover(d *document, offset int) any {
	word, start := d.text.wordAt(offset)
	if word == "" {
		return nil
	}
	r := rangeLSP{Start: d.text.position(start), End: d.text.position(start + len(word))}
	var text string
	if sym, ok := d.analysis.symbolNamed(word); ok {
		head := "def " + sym.detail
		switch sym.kind {
		case symType:
			head = "assemble " + sym.detail
		case symVar:
			head = sym.detail
		}
		text = "```turtle\n" + head + "\n```"
		if sym.doc != "" {
			text += "\n\n" + sym.doc
		}
		if sym.uri != d.uri {
			text += "\n\n_from " + filepath.Base(uriPath(sym.uri)) + "_"
		}
	} else if call, mod, body, ok := evaluator.LibraryDoc(word); ok {
		text = "```turtle\n" + call + "\n```\n_import " + mod + "_\n\n```\n" + strings.TrimRight(dedent(body), "\n") + "\n```"
	} else if help, ok := keywordHelp[word]; ok && token.LookupIdent(word) != token.IDENT {
		text = "**" + word + "** — " + help
	} else {
		return nil
	}
	return map[string]any{"contents": markup{Kind: "markdown", Value: text}, "range": r}
}

// dedent removes the indentation turtle doc's text starts its lines with.
func dedent(s string) string {
	lines := strings.Split(s, "\n")
	for i, l := range lines {
		lines[i] = strings.TrimPrefix(l, "  ")
	}
	return strings.Join(lines, "\n")
}

// keywordHelp is a line about each keyword, for hover and completion.
var keywordHelp = map[string]string{
	"show": "prints values: `show \"total: \", n .`", "warn": "prints to stderr: `warn \"careful\" .` (import system)",
	"if": "`if ] condition [` ... `if [end]`", "else": "`else if ] condition [` or `else ]` in an if chain",
	"def": "defines a function: `def name[a, b]` ... `def [end]`", "return": "gives a function's answer back",
	"end": "closes a block: `def [end]`, `if [end]`, `[loop][end]` ...", "loop": "`[loop][x in nums]`, `[loop][i = 0; i < 3; i++]`, `[loop][condition]` ... `[loop][end]`",
	"list": "`list [1, 2, 3]`", "set": "`set [1, 2]`: no repeats", "map": "`map [\"a\": 1]`: keys to values",
	"import": "`import sql`, `import lib/utils`, `import time [now]`", "sys": "runs a shell command: `sys ls -la`",
	"to": "in sentences: `add 4 to nums .`, `change x to integer`", "from": "in sentences: `remove 1 from nums .`",
	"at": "calls a method: `nums at get[0]`, `name at upper`", "of": "a field or key: `qty of order`",
	"is": "assigns, in sentence form: `r is nums at get 0 .`", "add": "`add 4 to nums .`", "change": "converts: `change \"42\" to integer`",
	"remove": "`remove 4 from nums .`", "delete": "`delete \"key\" from ages .` (maps)", "sort": "`sort nums .` sorts in place; `import sort` for more",
	"reverse": "`reverse nums .`", "insert": "`insert 9 to nums at 0 .`", "min": "`min of nums`", "max": "`max of nums`",
	"length": "`length of nums`", "read": "`[read] notes.txt to lines [end]`", "write": "`[write] notes.txt` ... `[end]`",
	"append": "`[append] notes.txt` ... `[end]`", "directory": "`[directory] path to names [end]`", "break": "leaves a loop",
	"continue": "goes on to the loop's next pass", "give": "makes a function: `x give x + 1`, `[a, b] give a + b`",
	"in": "`[loop][x in nums]`", "assemble": "names a kind of value with fields: `assemble Order [item, qty]`",
	"safe": "`safe` ... `handle [kinds] e .` ... `safe [end]`: catches errors", "handle": "the part of a safe block that runs on an error",
	"fail": "raises your own error: `fail \"message\"`", "div": "whole-number division: `7 div 2` is 3",
	"true": "the boolean true", "false": "the boolean false", "none": "no value",
}

// ---- definition and outline ----

func (s *Server) definition(d *document, offset int) any {
	word, _ := d.text.wordAt(offset)
	sym, ok := d.analysis.symbolNamed(word)
	if !ok {
		return nil
	}
	return location{URI: sym.uri, Range: s.symbolRange(d, sym)}
}

// symbolRange is where a symbol's name is, in its own file.
func (s *Server) symbolRange(d *document, sym symbol) rangeLSP {
	t := d.text
	if sym.uri != d.uri {
		data, _ := os.ReadFile(uriPath(sym.uri))
		t = newText(string(data), s.utf8)
	}
	if sym.line >= len(t.lines) {
		return rangeLSP{}
	}
	base := t.lines[sym.line]
	return rangeLSP{Start: t.position(base + sym.col), End: t.position(base + sym.endCol)}
}

func (s *Server) documentSymbols(d *document) any {
	out := []documentSymbol{}
	for _, sym := range d.analysis.symbols {
		if sym.uri != d.uri {
			continue
		}
		kind := symbolFunction
		switch sym.kind {
		case symType:
			kind = symbolStruct
		case symVar:
			kind = symbolVariable
		}
		r := s.symbolRange(d, sym)
		out = append(out, documentSymbol{Name: sym.name, Detail: sym.detail, Kind: kind, Range: lineRange(d.text, sym.line), SelectionRange: r})
	}
	return out
}

// ---- colors ----

// semanticTokens sends the colored spans, as the protocol packs them: per
// span, its line and column relative to the previous one, its length,
// its type and modifiers. A span over several lines (a block comment) is
// sent a line at a time.
func (s *Server) semanticTokens(d *document) any {
	t := d.text
	data := []int{}
	prevLine, prevChar := 0, 0
	emit := func(start, end, typ, mods int) {
		p := t.position(start)
		length := t.units(t.src[start:end])
		if length == 0 {
			return
		}
		dl := p.Line - prevLine
		dc := p.Character
		if dl == 0 {
			dc -= prevChar
		}
		data = append(data, dl, dc, length, typ, mods)
		prevLine, prevChar = p.Line, p.Character
	}
	for _, sp := range syntax.Highlight(t.src, d.analysis.words()) {
		typ, mods := 0, 0
		switch sp.Class {
		case syntax.Keyword:
			typ = 0
		case syntax.String:
			typ = 1
		case syntax.Constant:
			typ = 2
		case syntax.Comment:
			typ = 3
		case syntax.Definition:
			typ, mods = 4, 1
		case syntax.Call:
			typ = 4
		case syntax.Builtin:
			typ, mods = 4, 2
		default:
			continue
		}
		for start := sp.Start; start < sp.End; {
			end := sp.End
			if nl := strings.IndexByte(t.src[start:sp.End], '\n'); nl >= 0 {
				end = start + nl
			}
			emit(start, end, typ, mods)
			start = end + 1
		}
	}
	return map[string]any{"data": data}
}

// ---- URIs ----

// uriPath turns file:///Users/ann/a.trt (or file:///C:/x/a.trt) into a
// path.
func uriPath(uri string) string {
	u, err := url.Parse(uri)
	if err != nil || u.Scheme != "file" {
		return uri
	}
	p := u.Path
	if runtime.GOOS == "windows" {
		p = strings.TrimPrefix(p, "/")
	}
	return filepath.FromSlash(p)
}

// fileURI turns a path into a file:// URI.
func fileURI(path string) string {
	p := filepath.ToSlash(path)
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	return (&url.URL{Scheme: "file", Path: p}).String()
}

// formatting is Format Document: turtle fmt's layout, as one edit of the
// whole text. Code that doesn't parse yet is left alone (no edits); its
// error is already showing.
func (s *Server) formatting(d *document) any {
	formatted, err := format.Format(d.text.src)
	if err != nil || formatted == d.text.src {
		return []any{}
	}
	whole := rangeLSP{Start: d.text.position(0), End: d.text.position(len(d.text.src))}
	return []any{map[string]any{"range": whole, "newText": formatted}}
}
