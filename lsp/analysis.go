package lsp

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"Turtle/ast"
	"Turtle/evaluator"
	"Turtle/lexer"
	"Turtle/parser"
	"Turtle/syntax"
	"Turtle/token"
)

// What the server knows about a file: its errors, what it defines
// (functions, assembled types, top-level variables), its imports, and
// the words in it, from parsing it the way turtle would.

type symbolKind int

const (
	symFunc symbolKind = iota
	symType
	symVar
)

// symbol is a name a file defines, and where.
type symbol struct {
	name   string
	kind   symbolKind
	detail string // "double[x]", "Order [item, qty, price]"
	doc    string // the // lines just above it
	uri    string
	line   int // from 0
	col    int // bytes into the line
	endCol int
}

type importInfo struct {
	path string // as written: lib/utils, sql
	line int    // from 0
}

type analysis struct {
	diags   []diagnostic
	symbols []symbol // this file's, then its imported files' functions and types
	imports []importInfo
	libs    map[string]bool // builtin libraries imported
	idents  map[string]bool // every name used in the file
}

// analyze parses a file (at path, for its imports) and gathers what the
// server needs.
func analyze(uri, path, src string, t *text) *analysis {
	a := &analysis{libs: map[string]bool{}, idents: map[string]bool{}}
	p := parser.New(lexer.New(src))
	program := p.ParseProgram()
	if errs := p.Errors(); len(errs) > 0 {
		// The first error is the real one; the parser trips over what
		// follows it.
		line, msg := errorLine(errs[0])
		a.diags = append(a.diags, diagnostic{Range: lineRange(t, line), Severity: severityError, Source: "turtle", Message: msg})
	}
	l := lexer.New(src)
	for tok := l.NextToken(); tok.Type != token.EOF; tok = l.NextToken() {
		if tok.Type == token.IDENT {
			a.idents[tok.Literal] = true
		}
	}
	builtins := map[string]bool{}
	for _, lib := range evaluator.Libraries() {
		builtins[lib] = true
	}
	dir := filepath.Dir(path)
	for _, st := range program.Statements {
		switch s := st.(type) {
		case *ast.ImportStatement:
			line := s.Token.Line - 1
			a.imports = append(a.imports, importInfo{path: s.Path, line: line})
			if builtins[s.Path] {
				a.libs[s.Path] = true
				continue
			}
			file := filepath.Join(dir, filepath.FromSlash(s.Path)+".trt")
			data, err := os.ReadFile(file)
			if err != nil {
				a.diags = append(a.diags, diagnostic{Range: lineRange(t, line), Severity: severityWarning, Source: "turtle",
					Message: fmt.Sprintf("import %s: there's no library called %s, and no file %s.trt", s.Path, s.Path, s.Path)})
				continue
			}
			ftext := newText(string(data), t.utf8)
			a.symbols = append(a.symbols, definitions(fileURI(file), string(data), ftext, false)...)
		}
	}
	a.symbols = append(definitions(uri, src, t, true), a.symbols...)
	return a
}

// errorLine takes "line 3: message" apart: line 2 (from 0) and the message.
func errorLine(e string) (int, string) {
	if rest, ok := strings.CutPrefix(e, "line "); ok {
		if n, msg, ok := strings.Cut(rest, ": "); ok {
			if line, err := strconv.Atoi(n); err == nil {
				return max(line-1, 0), msg
			}
		}
	}
	return 0, e
}

// lineRange covers a line's text, from its first non-space character.
func lineRange(t *text, line int) rangeLSP {
	s := t.lineText(line)
	start := len(s) - len(strings.TrimLeft(s, " \t"))
	if line >= len(t.lines) {
		line = len(t.lines) - 1
	}
	base := t.lines[line]
	return rangeLSP{Start: t.position(base + start), End: t.position(base + len(s))}
}

var (
	defLine      = regexp.MustCompile(`^(\s*)def\s+([A-Za-z_][A-Za-z0-9_]*)\s*\[([^\]]*)\]`)
	assembleLine = regexp.MustCompile(`^(\s*)assemble\s+([A-Za-z_][A-Za-z0-9_]*)\s*\[([^\]]*)\]`)
	assignLine   = regexp.MustCompile(`^([A-Za-z_][A-Za-z0-9_]*)\s*(=|is\s)`)
)

// definitions finds the top-level functions and assembled types in src
// (and, when vars, the top-level variables), each with its description:
// the comments above it and, for a function, the ones its body starts
// with (syntax.FunctionDoc).
func definitions(uri, src string, t *text, vars bool) []symbol {
	var out []symbol
	seen := map[string]bool{}
	lines := strings.Split(strings.ReplaceAll(src, "\r\n", "\n"), "\n")
	for n, line := range lines {
		if m := defLine.FindStringSubmatchIndex(line); m != nil && m[3]-m[2] == 0 {
			name := line[m[4]:m[5]]
			out = append(out, symbol{name: name, kind: symFunc, detail: name + "[" + line[m[6]:m[7]] + "]", doc: syntax.FunctionDoc(lines, n), uri: uri, line: n, col: m[4], endCol: m[5]})
			continue
		}
		if m := assembleLine.FindStringSubmatchIndex(line); m != nil && m[3]-m[2] == 0 {
			name := line[m[4]:m[5]]
			out = append(out, symbol{name: name, kind: symType, detail: name + " [" + line[m[6]:m[7]] + "]", doc: syntax.CommentAbove(lines, n), uri: uri, line: n, col: m[4], endCol: m[5]})
			continue
		}
		if m := assignLine.FindStringSubmatchIndex(line); vars && m != nil {
			name := line[m[2]:m[3]]
			if !seen[name] && token.LookupIdent(name) == token.IDENT {
				seen[name] = true
				out = append(out, symbol{name: name, kind: symVar, detail: strings.TrimSpace(line), uri: uri, line: n, col: m[2], endCol: m[3]})
			}
		}
	}
	return out
}

// symbolNamed finds a symbol by name: this file's first.
func (a *analysis) symbolNamed(name string) (symbol, bool) {
	for _, s := range a.symbols {
		if s.name == name {
			return s, true
		}
	}
	return symbol{}, false
}

// words for coloring: the imported libraries' words, and the library
// functions.
func (a *analysis) words() syntax.Words {
	w := syntax.Words{Context: map[string]bool{}, Builtin: map[string]bool{}}
	for lib := range a.libs {
		for _, word := range syntax.LibraryWords[lib] {
			w.Context[word] = true
		}
	}
	for _, f := range evaluator.LibraryFunctions() {
		w.Builtin[f] = true
	}
	return w
}
