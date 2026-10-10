package evaluator

import (
	"fmt"
	"slices"
	"sort"
	"strings"

	"Turtle/object"
)

// Help, version and stdlib: three core functions, there without an
// import. A name you define yourself comes first, as with typeof.
//
//	help[]              how to find things: the libraries, and how to ask
//	help["linear"]      a library, function or method, as turtle doc shows it
//	help[m]             what m is and what can be done with it
//	version             the version of Turtle running ("0.9.170")
//	stdlib[]            every library and its functions, as a map

// Version is Turtle's version; cmd/turtle sets it from the release tag.
var Version = "dev"

// versionText is Version without the tag's v: "0.9.170".
func versionText() string {
	return strings.TrimPrefix(Version, "v")
}

// typeMethods are the methods of each kind of value, in the order help
// lists them, each written the way it's called after "at".
var typeMethods = map[string][]string{
	"list": {"add[value]", "get[index]", "put[value, index]", "insert[value, index]", "remove[value]",
		"pop", "slice[start, end]", "contains[value]", "index[value]", "count[value]",
		"sort", "reverse", "clear", "len", "isempty", "tostring"},
	"set": {"add[value]", "remove[value]", "contains[value]", "union[other]", "intersection[other]",
		"difference[other]", "subset[other]", "superset[other]", "pop", "sort", "clear",
		"len", "isempty", "tostring"},
	"map": {"get[key]", "add[key, value]", "delete[key]", "contains[key]", "getkeys", "getvalues",
		"invert", "len", "isempty", "tostring"},
	"string": {"upper", "lower", "trim", "get[index]", "slice[start, end]", "split[separator]",
		"contains[text]", "indexof[text]", "replace[old, new]", "padleft[width]", "padright[width]",
		"isnumber", "len", "isempty", "tostring"},
	"integer": {"fixed[places]", "commas", "commas[places]", "sqrt", "abs", "round", "floor", "ceil", "pow[power]", "random"},
	"matrix": {"rows", "columns", "shape", "get[r, c]", "put[value, r, c]", "row[r]", "column[c]",
		"flatten", "reshape[rows, columns]", "isempty", "tostring"},
	"date": {"tostring"},
}

// methodName is "get" for "get[index]".
func methodName(m string) string {
	name, _, _ := strings.Cut(m, "[")
	return name
}

// coreFunctions are the functions every program has, without an import.
var coreFunctions = map[string]string{
	"typeof":  "typeof[x] is the kind of x as text: \"integer\", \"list\", \"matrix\", ...\n  Example: show typeof[3.5] .          // float",
	"help":    "help[x] shows what x is and what it can do; help[\"linear\"] a library,\n  help[\"reshape\"] a function, help[\"if\"] a keyword, help[] where to start.",
	"stdlib":  "stdlib[] is a map of every library's name to its function names.\n  Example: show stdlib[] at getkeys .",
	"version": "version (or version[]) is the version of Turtle running, as text.\n  Example: show version .              // 0.9.171",
}

// typeLibraries are the libraries that work on each kind of value.
var typeLibraries = map[string][]string{
	"list":     {"data", "sort", "search", "random"},
	"set":      {"data", "sort", "search"},
	"map":      {"data", "json", "config", "search"},
	"string":   {"strings", "pattern", "crypt"},
	"integer":  {"math"},
	"matrix":   {"linear"},
	"date":     {"time"},
	"database": {"sql"},
}

func init() {
	typeMethods["float"] = typeMethods["integer"]
	typeLibraries["float"] = typeLibraries["integer"]
}

// helpOverview is help with nothing to look up.
func helpOverview() string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "Turtle %s\n\n", versionText())
	sb.WriteString("help[\"linear\"]     a library: what it's for and its functions\n")
	sb.WriteString("help[\"reshape\"]    one function or method, with an example\n")
	sb.WriteString("help[\"if\"]         a keyword; help[\"keywords\"] lists them all\n")
	sb.WriteString("help[x]            what x is and what you can do with it\n")
	sb.WriteString("stdlib[]           every library and its functions, as a map\n")
	sb.WriteString("version            the version of Turtle\n\nLibraries (import one to use it):\n")
	for _, m := range moduleNames() {
		intro, _ := parseModuleDoc(m, moduleDocs[m])
		fmt.Fprintf(&sb, "  %-9s %s\n", m, firstSentence(intro))
	}
	return strings.TrimRight(sb.String(), "\n")
}

// helpTopic is the documentation for a library, function or method name.
func helpTopic(topic string) (string, bool) {
	topic = strings.TrimSpace(topic)
	if topic == "" {
		return helpOverview(), true
	}
	if strings.ToLower(topic) == "keywords" {
		return keywordsOverview(), true
	}
	var found []string
	if k := keywordHelp(topic); k != "" {
		found = append(found, k)
	}
	if _, ok := moduleDocs[strings.ToLower(topic)]; ok {
		found = append(found, libraryHelp(strings.ToLower(topic)))
		return strings.Join(found, "\n\n"), true // a library's functions are in its help
	}
	if c, ok := coreFunctions[topic]; ok {
		found = append(found, topic+"        (no import needed)\n  "+c)
	}
	if _, ok := typeMethods[strings.ToLower(topic)]; ok {
		found = append(found, strings.TrimRight(typeHelp(strings.ToLower(topic), "x"), "\n"))
	}
	for _, e := range allEntries() {
		if e.name() == topic {
			found = append(found, strings.TrimRight(e.String(), "\n"))
		}
	}
	for _, t := range sortedTypes() {
		for _, m := range typeMethods[t] {
			if methodName(m) == topic {
				found = append(found, fmt.Sprintf("%s at %s        (a %s method; help[\"%s\"] for the rest)", t, m, t, t))
			}
		}
	}
	if len(found) > 0 {
		return strings.Join(found, "\n\n"), true
	}
	return "", false
}

// libraryHelp is a library's introduction and one line per function;
// help["sql_load"] then shows a function in full.
func libraryHelp(module string) string {
	intro, es := parseModuleDoc(module, moduleDocs[module])
	var sb strings.Builder
	fmt.Fprintf(&sb, "import %s\n\n%s\n\n", module, intro)
	for _, e := range es {
		fmt.Fprintf(&sb, "  %s\n      %s\n", e.call, firstSentence(e.body))
	}
	fmt.Fprintf(&sb, "\nOne in full: help[\"%s\"]", es[0].name())
	return sb.String()
}

// firstSentence is the start of a description, up to its first full
// stop, on one line.
func firstSentence(body string) string {
	para, _, _ := strings.Cut(strings.TrimSpace(body), "\n  Example")
	para, _, _ = strings.Cut(para, "\n\n")
	para, _, _ = strings.Cut(para, ":\n") // examples follow
	text := strings.Join(strings.Fields(para), " ")
	for i := 0; i+1 < len(text); i++ {
		if text[i] == '.' && text[i+1] == ' ' {
			return text[:i+1]
		}
	}
	return strings.TrimSuffix(text, ":")
}

func sortedTypes() []string {
	var ts []string
	for t := range typeMethods {
		ts = append(ts, t)
	}
	sort.Strings(ts)
	return ts
}

// valueHelp says what v is and what can be done with it. name is what
// it's called, when that's known (the REPL's help m), else "".
func valueHelp(v object.Object, name string) string {
	kind := typeName(v)
	var sb strings.Builder
	if name == "" {
		name = "x"
		sb.WriteString("This is " + describeValue(v, kind) + ".\n")
	} else {
		fmt.Fprintf(&sb, "%s is %s.\n", name, describeValue(v, kind))
	}
	switch x := v.(type) {
	case *object.Function:
		switch {
		case x.Theory != nil:
			return theoryHelp(x.Theory)
		case x.Shape != nil:
			fmt.Fprintf(&sb, "Make one: %s[%s]. Read a field: %s of value.\n",
				x.Shape.Name, strings.Join(x.Shape.Fields, ", "), x.Shape.Fields[0])
		case x.Scroll == nil:
			fmt.Fprintf(&sb, "Call it: %s[%s]\n", functionName(x, name), strings.Join(x.Parameters, ", "))
		}
	case *object.Assembly:
		fmt.Fprintf(&sb, "Its fields: %s. Read one: %s of %s\n",
			strings.Join(x.Shape.Fields, ", "), x.Shape.Fields[0], name)
	}
	sb.WriteString(typeHelp(kind, name))
	return strings.TrimRight(sb.String(), "\n")
}

// typeHelp is the methods of a kind of value, written on name, and the
// libraries that work on it.
func typeHelp(kind, name string) string {
	var sb strings.Builder
	if ms := typeMethods[kind]; len(ms) > 0 {
		article := "A"
		if strings.ContainsRune("aeiou", rune(kind[0])) {
			article = "An"
		}
		fmt.Fprintf(&sb, "%s %s's methods, as %s at %s:\n", article, kind, name, ms[0])
		sb.WriteString(wrapList(ms, "  ", 72))
		if kind == "integer" || kind == "float" {
			sb.WriteString("  (sqrt to random need import math)\n")
		}
	}
	if libs := typeLibraries[kind]; len(libs) > 0 {
		var asks []string
		for _, l := range libs {
			asks = append(asks, `help["`+l+`"]`)
		}
		sb.WriteString("Libraries for it: " + strings.Join(asks, ", ") + "\n")
	}
	return sb.String()
}

// wrapList joins items with commas into lines no wider than width.
func wrapList(items []string, indent string, width int) string {
	var sb strings.Builder
	line := indent
	for i, it := range items {
		piece := it
		if i < len(items)-1 {
			piece += ","
		}
		if len(line) > len(indent) && len(line)+1+len(piece) > width {
			sb.WriteString(line + "\n")
			line = indent
		}
		if len(line) > len(indent) {
			line += " "
		}
		line += piece
	}
	return sb.String() + line + "\n"
}

func functionName(fn *object.Function, name string) string {
	if fn.Name != "" {
		return fn.Name
	}
	return name
}

// describeValue is "a matrix, 2 x 3", "a list of 4 items", ...
func describeValue(v object.Object, kind string) string {
	switch x := v.(type) {
	case *object.List:
		return fmt.Sprintf("a list of %d %s", len(x.Elements), plural(len(x.Elements), "item"))
	case *object.Set:
		return fmt.Sprintf("a set of %d %s", len(x.Elements), plural(len(x.Elements), "item"))
	case *object.Map:
		n := x.Len()
		if n == 1 {
			return "a map of 1 entry"
		}
		return fmt.Sprintf("a map of %d entries", n)
	case *object.String:
		n := len([]rune(x.Value))
		return fmt.Sprintf("a string of %d %s", n, plural(n, "character"))
	case *object.Matrix:
		return "a matrix, " + strings.TrimSuffix(x.SizeText(), " matrix")
	case *object.Assembly:
		return "an assembled " + x.Shape.Name
	case *object.None:
		return "none, the value for nothing"
	}
	switch kind[0] {
	case 'a', 'e', 'i', 'o', 'u':
		return "an " + kind
	}
	return "a " + kind
}

// HelpText is help for the REPL's help command: a topic, or, when name
// is one of the session's variables, its value.
func (it *Interpreter) HelpText(topic string) (string, error) {
	if v, ok := it.Global.Get(topic); ok {
		return valueHelp(v, topic), nil
	}
	if fn, ok := it.Global.GetFunction(topic); ok {
		return valueHelp(fn, topic), nil
	}
	if text, ok := helpTopic(topic); ok {
		return text, nil
	}
	return "", fmt.Errorf("%s", helpNotFound(topic))
}

func helpNotFound(topic string) string {
	var close []string
	for _, e := range allEntries() {
		if editDistance(e.name(), topic) <= 2 {
			close = append(close, e.name())
		}
	}
	for _, e := range keywords() {
		if editDistance(e.word, topic) <= 1 && len(topic) > 2 && !slices.Contains(close, e.word) {
			close = append(close, e.word)
		}
	}
	msg := fmt.Sprintf("no library, function, method or keyword called %q", topic)
	if len(close) > 0 {
		msg += " (did you mean " + strings.Join(close, ", ") + "?)"
	}
	return msg + `; help[] lists the libraries, help["keywords"] the keywords`
}

// callHelp is help[...]: it shows the help and gives none.
func (it *Interpreter) callHelp(args []object.Object, env *object.Environment) object.Object {
	if len(args) > 1 {
		fatalf("help takes one thing to look up, got %d", len(args))
	}
	var text string
	switch {
	case len(args) == 0:
		text = helpOverview()
	default:
		if s, ok := args[0].(*object.String); ok {
			if fn, ok := env.GetFunction(s.Value); ok && fn.Theory != nil {
				text = theoryHelp(fn.Theory)
				break
			}
			// Text names what to look up; help["string"] is about text.
			t, found := helpTopic(s.Value)
			if !found {
				t = helpNotFound(s.Value)
			}
			text = t
			break
		}
		text = valueHelp(args[0], "")
	}
	fmt.Println(text)
	it.copyOutput(env, text)
	return object.NoneValue
}

// stdlibMap is stdlib[]: each library's name, and its functions (math's
// are methods on numbers) in the order the docs give them.
func stdlibMap() object.Object {
	m := object.NewMap()
	for _, name := range moduleNames() {
		_, es := parseModuleDoc(name, moduleDocs[name])
		fs := &object.List{}
		for _, e := range es {
			fs.Elements = append(fs.Elements, &object.String{Value: e.name()})
		}
		m.Put(&object.String{Value: name}, fs)
	}
	return m
}
