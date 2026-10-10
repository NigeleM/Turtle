package evaluator

import (
	"Turtle/ast"
	"Turtle/lexer"
	"Turtle/parser"
	"Turtle/syntax"

	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// "turtle doc" shows documentation:
//
//	turtle doc              every library and its functions
//	turtle doc sql          one library: every function, in full
//	turtle doc sql_load     one function
//	turtle doc lib/shop.turtle   the functions of a Turtle file, from the
//	                        // comments written just above each def
//
// A Turtle file is documented the way Go code is: the // lines directly
// above a def (or assemble) describe it, and the // lines at the top of
// the file describe the file.

type funcEntry struct {
	module string
	call   string // the first line: how it's called
	body   string // the rest, indented
}

func (e funcEntry) name() string {
	if i := strings.Index(e.call, " at "); i >= 0 {
		return strings.Fields(e.call[i+4:])[0]
	}
	if i := strings.IndexAny(e.call, "[ "); i >= 0 {
		return e.call[:i]
	}
	return e.call
}

// parseModuleDoc splits a module's text into its introduction and its
// function entries.
func parseModuleDoc(module, text string) (string, []funcEntry) {
	parts := strings.Split(text, "\n### ")
	intro := strings.TrimSpace(parts[0])
	var entries []funcEntry
	for _, p := range parts[1:] {
		call, body, _ := strings.Cut(p, "\n")
		entries = append(entries, funcEntry{module: module, call: strings.TrimSpace(call), body: strings.TrimRight(body, "\n ")})
	}
	return intro, entries
}

func moduleNames() []string {
	var names []string
	for m := range moduleDocs {
		names = append(names, m)
	}
	sort.Strings(names)
	return names
}

func allEntries() []funcEntry {
	var out []funcEntry
	for _, m := range moduleNames() {
		_, es := parseModuleDoc(m, moduleDocs[m])
		out = append(out, es...)
	}
	return out
}

func (e funcEntry) String() string {
	return fmt.Sprintf("%s        (import %s)\n%s\n", e.call, e.module, e.body)
}

// Doc returns the documentation for topic (see above). dir is where a
// file topic's path starts.
func Doc(topic, dir string) (string, error) {
	topic = strings.TrimSpace(topic)
	if topic == "" {
		var sb strings.Builder
		sb.WriteString("Turtle's libraries. Use one with import, e.g. \"import sql\".\n")
		sb.WriteString("turtle doc <library> shows it in full; turtle doc <function> one function;\n")
		sb.WriteString("turtle doc <file.turtle> the functions of your own file.\n")
		for _, m := range moduleNames() {
			intro, es := parseModuleDoc(m, moduleDocs[m])
			first, _, _ := strings.Cut(intro, "\n")
			fmt.Fprintf(&sb, "\n%s: %s\n", m, first)
			for _, e := range es {
				fmt.Fprintf(&sb, "    %s\n", e.call)
			}
		}
		return sb.String(), nil
	}
	if text, ok := moduleDocs[strings.ToLower(topic)]; ok {
		intro, es := parseModuleDoc(topic, text)
		var sb strings.Builder
		fmt.Fprintf(&sb, "import %s\n\n%s\n", strings.ToLower(topic), intro)
		for _, e := range es {
			sb.WriteString("\n" + e.String())
		}
		return sb.String(), nil
	}
	for _, e := range allEntries() {
		if e.name() == topic {
			return e.String(), nil
		}
	}
	path := topic
	if !filepath.IsAbs(path) {
		path = filepath.Join(dir, path)
	}
	if !syntax.IsTurtleFile(path) {
		ext, err := syntax.FindModule(topic, func(ext string) bool {
			_, err := os.Stat(path + ext)
			return err == nil
		})
		if err != nil {
			return "", err
		}
		path += ext
	}
	if data, err := os.ReadFile(path); err == nil {
		return fileDoc(topic, string(data)), nil
	}
	var close []string
	for _, e := range allEntries() {
		if strings.Contains(e.name(), topic) || strings.Contains(topic, e.name()) || editDistance(e.name(), topic) <= 2 {
			close = append(close, e.name())
		}
	}
	msg := fmt.Sprintf("no library, function or file called %q", topic)
	if len(close) > 0 {
		msg += " (did you mean " + strings.Join(close, ", ") + "?)"
	}
	return "", fmt.Errorf("%s; turtle doc lists them all", msg)
}

// fileDoc documents a Turtle file from its comments: the // lines at its
// top, and the // lines directly above each top-level def and assemble.
func fileDoc(name, src string) string {
	lines := strings.Split(strings.ReplaceAll(src, "\r\n", "\n"), "\n")
	var sb strings.Builder
	sb.WriteString(name + "\n")
	top, i := syntax.FileDoc(lines)
	if top != "" {
		sb.WriteString("\n" + top + "\n")
	}
	found := 0
	var private []string
	for n, line := range lines {
		if n < i || (!strings.HasPrefix(line, "def ") && !strings.HasPrefix(line, "assemble ")) {
			continue
		}
		head := strings.TrimSpace(line)
		// A private (~) function is for the file itself: named at the end,
		// not documented for those who import it.
		if _, rest, _ := strings.Cut(head, " "); strings.HasPrefix(rest, "~") {
			name, _, _ := strings.Cut(rest, "[")
			private = append(private, strings.TrimSpace(name))
			continue
		}
		if strings.HasPrefix(head, "def ") {
			head = strings.TrimSpace(strings.TrimPrefix(head, "def "))
			if strings.HasPrefix(head, "[") {
				continue // def [end]
			}
		}
		text := syntax.CommentAbove(lines, n)
		if strings.HasPrefix(line, "def ") {
			text = syntax.FunctionDoc(lines, n)
		}
		var doc []string
		for _, l := range strings.Split(text, "\n") {
			if text != "" {
				doc = append(doc, "  "+l)
			}
		}
		if len(doc) == 0 {
			doc = []string{"  (no description: write // lines just above it, or first in its body)"}
		}
		fmt.Fprintf(&sb, "\n%s\n%s\n", head, strings.Join(doc, "\n"))
		found++
	}
	// Its theories: how each is written, its abstract and its theorems.
	program := parser.New(lexer.New(strings.Join(lines, "\n"))).ParseProgram()
	for _, st := range program.Statements {
		if ts, ok := st.(*ast.TheoryStatement); ok {
			sb.WriteString("\n" + strings.Replace(theoryHelp(ts), "        (a theory, line", "        (theory, line", 1) + "\n")
			found++
		}
	}
	if found == 0 {
		sb.WriteString("\n(no functions or assembled types at the top level)\n")
	}
	if len(private) > 0 {
		fmt.Fprintf(&sb, "\nPrivate to this file: %s\n", strings.Join(private, ", "))
	}
	return sb.String()
}

func commentText(line string) string {
	t := strings.TrimPrefix(strings.TrimSpace(line), "//")
	return strings.TrimPrefix(t, " ")
}

// editDistance counts the letters to add, remove or change to turn a
// into b, to suggest names for a misspelling.
func editDistance(a, b string) int {
	prev := make([]int, len(b)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(a); i++ {
		cur := make([]int, len(b)+1)
		cur[0] = i
		for j := 1; j <= len(b); j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			cur[j] = min(prev[j]+1, cur[j-1]+1, prev[j-1]+cost)
		}
		prev = cur
	}
	return prev[len(b)]
}

// LibraryDoc is a library function's documentation, as turtle doc shows
// it: how it's called ("sql_load[db, table, path [, types]]"), its
// library, and the description. For the language server's hover.
func LibraryDoc(name string) (call, module, body string, ok bool) {
	for _, e := range allEntries() {
		if e.name() == name && !strings.Contains(e.call, " at ") {
			return e.call, e.module, e.body, true
		}
	}
	return "", "", "", false
}

// Libraries are the builtin libraries' names (what import takes).
func Libraries() []string {
	var out []string
	for name := range builtinModules {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

// LibraryOf is the library a function comes from ("" if none).
func LibraryOf(name string) string {
	for _, m := range builtinModules {
		for _, f := range m.Funcs {
			if f == name {
				return m.Name
			}
		}
	}
	return ""
}

// IsLibrary reports whether name is a builtin library (import json), not
// a .turtle file.
func IsLibrary(name string) bool {
	_, ok := builtinModules[name]
	return ok
}
