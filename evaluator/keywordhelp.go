// Copyright 2017-2026 Nigele McCoy
// Licensed under the Apache License, Version 2.0;
// see LICENSE and NOTICE.

package evaluator

import (
	"fmt"
	"regexp"
	"strings"
	"sync"

	"Turtle/docs"
)

// Keyword help comes from the Keywords tables in docs/reference.md, so
// help["if"] and the reference always say the same thing.

type keywordEntry struct {
	word, what, example, see, where string
	special                         bool // special only in one place, an ordinary name elsewhere
}

var (
	keywordsOnce sync.Once
	keywordList  []keywordEntry
)

var (
	mdLink = regexp.MustCompile(`\[([^\]]*)\]\(([^)]*)\)`)
	mdWord = regexp.MustCompile("`([^`]+)`")
)

// plainText drops markdown's backticks and turns [text](#anchor) into
// text (docs/reference.md#anchor).
func plainText(s string) string {
	s = mdLink.ReplaceAllStringFunc(s, func(m string) string {
		p := mdLink.FindStringSubmatch(m)
		target := p[2]
		switch {
		case strings.HasPrefix(target, "#"):
			target = "docs/reference.md" + target
		case !strings.Contains(target, "://"):
			target = "docs/" + target
		}
		return strings.Trim(p[1], "`") + " (" + target + ")"
	})
	return strings.ReplaceAll(s, "`", "")
}

// keywords reads the tables once.
func keywords() []keywordEntry {
	keywordsOnce.Do(func() {
		text := strings.ReplaceAll(docs.Reference, "\r\n", "\n")
		start := strings.Index(text, "\n## Keywords\n")
		if start < 0 {
			return
		}
		section := text[start+1:]
		if end := strings.Index(section[3:], "\n## "); end >= 0 {
			section = section[:end+3]
		}
		var header []string
		for _, line := range strings.Split(section, "\n") {
			if !strings.HasPrefix(line, "|") {
				header = nil
				continue
			}
			cells := strings.Split(strings.Trim(line, "|"), " | ")
			for i := range cells {
				cells[i] = strings.TrimSpace(cells[i])
			}
			if header == nil {
				header = cells
				continue
			}
			if strings.HasPrefix(cells[0], "---") {
				continue
			}
			var e keywordEntry
			for i, h := range header {
				if i >= len(cells) {
					break
				}
				switch h {
				case "What it does":
					e.what = plainText(cells[i])
				case "Example":
					e.example = plainText(cells[i])
				case "See":
					e.see = plainText(cells[i])
				case "Import":
					e.where = "after " + plainText(cells[i])
				case "Where":
					e.what = plainText(cells[i])
					e.special = true
				}
			}
			for _, w := range mdWord.FindAllStringSubmatch(cells[0], -1) {
				k := e
				k.word = w[1]
				keywordList = append(keywordList, k)
			}
		}
	})
	return keywordList
}

// keywordHelp is help for a keyword, or "" when word isn't one.
func keywordHelp(word string) string {
	for _, e := range keywords() {
		if e.word != word {
			continue
		}
		kind := "a keyword"
		switch {
		case e.where != "":
			kind = "a keyword " + e.where
		case e.special:
			kind = "a word, special only in one place"
		}
		var sb strings.Builder
		fmt.Fprintf(&sb, "%s        (%s)\n  %s.\n", e.word, kind, upperFirst(e.what))
		if e.example != "" {
			fmt.Fprintf(&sb, "  Example: %s\n", e.example)
		}
		if e.see != "" {
			fmt.Fprintf(&sb, "  More: %s\n", e.see)
		}
		return strings.TrimRight(sb.String(), "\n")
	}
	return ""
}

// keywordsOverview is help["keywords"]: every keyword in a line.
func keywordsOverview() string {
	var sb strings.Builder
	sb.WriteString("Turtle's keywords (help[\"if\"] shows one):\n")
	// Words that share a row (integer, float, string, ...) share a line.
	ks := keywords()
	for i := 0; i < len(ks); {
		j := i + 1
		for j < len(ks) && ks[j].what == ks[i].what && ks[j].where == ks[i].where {
			j++
		}
		var words []string
		for _, e := range ks[i:j] {
			words = append(words, e.word)
		}
		what := ks[i].what
		if ks[i].where != "" {
			what += " (" + ks[i].where + ")"
		}
		names := strings.Join(words, ", ")
		if len(names) > 10 {
			fmt.Fprintf(&sb, "  %s\n  %-10s %s\n", names, "", what)
		} else {
			fmt.Fprintf(&sb, "  %-10s %s\n", names, what)
		}
		i = j
	}
	return strings.TrimRight(sb.String(), "\n")
}

func upperFirst(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
