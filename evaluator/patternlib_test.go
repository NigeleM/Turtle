package evaluator

import (
	"regexp"
	"strconv"
	"strings"
	"testing"

	"Turtle/object"
)

// Wherever Go's own replacement is right (every $ names a group the
// pattern has, and a number isn't run into letters), replaceAll gives the
// same text.
func TestReplaceAllAgreesWithGo(t *testing.T) {
	pats := []string{`(\w)-(\w)`, `(\d+)`, `a*`, `(?P<first>\w+) (?P<last>\w+)`, `(a)(x)?(b)`, `x`, `^`, `$`, `\b`, `(é+)`}
	texts := []string{"a-b c-d", "price 5 and 10", "aaa", "ann lee, bo ray", "ab axb", "héllo éé", "", "x x"}
	compared := 0
	withs := []string{"[$1]", "${1}", "$$", "<$0>", "${last}, ${first}", "$2/$1", "plain", "", "${1}-${2}", "$first"}
	for _, p := range pats {
		re := regexp.MustCompile(p)
		for _, text := range texts {
			for _, with := range withs {
				if !namesOnlyItsGroups(re, with) {
					continue
				}
				compared++
				if got, want := replaceAll(re, text, with), re.ReplaceAllString(text, with); got != want {
					t.Errorf("%q in %q with %q: got %q, Go gives %q", p, text, with, got, want)
				}
			}
		}
	}
	if compared < 300 {
		t.Errorf("only %d cases compared", compared)
	}
	t.Logf("%d cases agree with Go", compared)
}

// namesOnlyItsGroups: every $... in with names a group re has.
func namesOnlyItsGroups(re *regexp.Regexp, with string) bool {
	for _, m := range regexp.MustCompile(`\$(\$|\{([^}]*)\}|([0-9]+|[A-Za-z_]\w*))`).FindAllStringSubmatch(with, -1) {
		name := m[2] + m[3]
		if m[1] == "$" {
			continue
		}
		if n := strings.TrimLeft(name, "0123456789"); n == "" {
			if len(name) > 2 || name[0]-'0' > byte(re.NumSubexp()) {
				return false
			}
		} else if re.SubexpIndex(name) < 0 {
			return false
		}
	}
	return true
}

func TestReplaceAllFixes(t *testing.T) {
	cases := []struct{ text, pat, with, want string }{
		{"price: 5", `\d+`, "$10", "price: $10"}, // no group 10: as written
		{"a-b", `(\w)-(\w)`, "$1x$2", "axb"},     // a number ends at a letter
		{"a-b", `(\w)-(\w)`, "${1}x$2", "axb"},
		{"cost", `cost`, "$$ cost", "$ cost"},
		{"ann lee", `(?P<first>\w+) (?P<last>\w+)`, "$last, ${first}", "lee, ann"},
		{"ab", `(a)(x)?(b)`, "[$2]", "[]"}, // took no part in the match
		{"a.b", `\.`, "$", "a$b"},
		{"x", `x`, "${nope} $nope $", "${nope} $nope $"},
	}
	for _, c := range cases {
		if got := replaceAll(regexp.MustCompile(c.pat), c.text, c.with); got != c.want {
			t.Errorf("replaceall[%q, %q, %q] = %q, want %q", c.text, c.pat, c.with, got, c.want)
		}
	}
}

// Patterns made as a program goes don't pile up.
func TestPatternCacheIsCapped(t *testing.T) {
	for i := 0; i < 3*maxPatterns; i++ {
		compilePattern("matches", &object.String{Value: "^item-" + strconv.Itoa(i) + "$"})
	}
	patternsMu.Lock()
	n := len(patterns)
	patternsMu.Unlock()
	if n > maxPatterns {
		t.Errorf("%d patterns kept, at most %d", n, maxPatterns)
	}
}
