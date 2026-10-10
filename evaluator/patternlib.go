package evaluator

import (
	"regexp"
	"strconv"
	"strings"
	"sync"
	"unicode"

	"Turtle/object"
)

// The pattern library (import pattern): regular expressions, through Go's
// regexp package (RE2 syntax: no backtracking, so no pattern can take
// forever). Write patterns in backticks, which keep { } and \ as typed:
//
//	matches["order 42", `\d+`]              true
//	findall["a1 b22", `\d+`]                [ "1", "22" ]
//	replaceall["a1 b2", `\d`, "#"]          "a# b#"
//	splitby["a, b;c", `[,;]\s*`]            [ "a", "b", "c" ]
//	groups["2026-10-06", `(\d+)-(\d+)`]     [ "2026", "10" ]

// patterns are the patterns compiled so far, each compiled once. A
// program that makes patterns as it goes ("^item-" + i + "$") would
// otherwise keep every one: past maxPatterns they start over.
var (
	patternsMu sync.Mutex
	patterns   = map[string]*regexp.Regexp{}
)

const maxPatterns = 1000

func compilePattern(fn string, v object.Object) *regexp.Regexp {
	text := asStringArg(fn, v)
	patternsMu.Lock()
	re, ok := patterns[text]
	patternsMu.Unlock()
	if ok {
		return re
	}
	re, err := regexp.Compile(text)
	if err != nil {
		fatalKind(kindPattern, "%s: %q isn't a valid pattern: %v", fn, text, err)
	}
	patternsMu.Lock()
	if len(patterns) >= maxPatterns {
		patterns = map[string]*regexp.Regexp{}
	}
	patterns[text] = re
	patternsMu.Unlock()
	return re
}

// replaceAll replaces every match of re in text with with, where $1 (or
// ${1}) is the first group, $name (or ${name}) a named one, and $$ a $.
// A group number ends at the first character that isn't a digit ($1x is
// group 1, then x), and a $ that names no group of the pattern stays as
// written ("$10" in a pattern without ten groups is $10).
func replaceAll(re *regexp.Regexp, text, with string) string {
	var b strings.Builder
	last := 0
	for _, m := range re.FindAllStringSubmatchIndex(text, -1) {
		b.WriteString(text[last:m[0]])
		expandReplacement(&b, re, text, m, with)
		last = m[1]
	}
	b.WriteString(text[last:])
	return b.String()
}

func expandReplacement(b *strings.Builder, re *regexp.Regexp, text string, m []int, with string) {
	group := func(name string) (string, bool) {
		i := -1
		if n, err := strconv.Atoi(name); err == nil {
			i = n
		} else if name != "" {
			i = re.SubexpIndex(name)
		}
		if i < 0 || i > re.NumSubexp() {
			return "", false
		}
		if m[2*i] < 0 {
			return "", true // a group that took no part in the match
		}
		return text[m[2*i]:m[2*i+1]], true
	}
	for i := 0; i < len(with); i++ {
		if with[i] != '$' || i+1 == len(with) {
			b.WriteByte(with[i])
			continue
		}
		rest := with[i+1:]
		switch {
		case rest[0] == '$':
			b.WriteByte('$')
			i++
		case rest[0] == '{':
			end := strings.IndexByte(rest, '}')
			if end < 0 {
				b.WriteByte('$')
				continue
			}
			if g, ok := group(rest[1:end]); ok {
				b.WriteString(g)
				i += end + 1
			} else {
				b.WriteByte('$')
			}
		default:
			n := 0
			if rest[0] >= '0' && rest[0] <= '9' {
				for n < len(rest) && rest[n] >= '0' && rest[n] <= '9' {
					n++
				}
			} else {
				for n < len(rest) && (rest[n] == '_' || unicode.IsLetter(rune(rest[n])) || rest[n] >= '0' && rest[n] <= '9') {
					n++
				}
			}
			if g, ok := group(rest[:n]); ok && n > 0 {
				b.WriteString(g)
				i += n
			} else {
				b.WriteByte('$')
			}
		}
	}
}

func strings2list(items []string) *object.List {
	out := &object.List{Elements: make([]object.Object, len(items))}
	for i, s := range items {
		out.Elements[i] = &object.String{Value: s}
	}
	return out
}

func (it *Interpreter) callPattern(name string, args []object.Object) object.Object {
	switch name {
	case "matches":
		requireFuncArgs(name, args, 2)
		text := asStringArg(name, args[0])
		return object.Bool(compilePattern(name, args[1]).MatchString(text))
	case "findall":
		requireFuncArgs(name, args, 2)
		text := asStringArg(name, args[0])
		return strings2list(compilePattern(name, args[1]).FindAllString(text, -1))
	case "replaceall":
		requireFuncArgs(name, args, 3)
		text := asStringArg(name, args[0])
		with := asStringArg(name, args[2])
		return &object.String{Value: replaceAll(compilePattern(name, args[1]), text, with)}
	case "splitby":
		requireFuncArgs(name, args, 2)
		text := asStringArg(name, args[0])
		return strings2list(compilePattern(name, args[1]).Split(text, -1))
	case "groups":
		requireFuncArgs(name, args, 2)
		text := asStringArg(name, args[0])
		m := compilePattern(name, args[1]).FindStringSubmatch(text)
		if m == nil {
			return object.NoneValue
		}
		return strings2list(m[1:])
	}
	fatalKind(kindName, "no pattern function %q", name)
	return nil
}
