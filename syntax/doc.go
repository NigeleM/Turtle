package syntax

import "strings"

// A function's description, for hover, completion and turtle doc: the
// comments directly above its def, and the comments that come first in
// its body (as Python's docstrings do), either kind, // lines or a //* *//
// block:
//
//	// hello divides b by ten.
//	def hello[b]
//	    //* b is a number; the answer
//	        is a float. *//
//	    return b / 10.0
//	def [end]
//
// To the interpreter they're ordinary comments.

// FunctionDoc is the description of the function whose def is on line
// def (from 0): the comment above it, then the one starting its body.
func FunctionDoc(lines []string, def int) string {
	var parts []string
	for _, p := range []string{CommentAbove(lines, def), CommentInside(lines, def)} {
		if p != "" {
			parts = append(parts, p)
		}
	}
	return strings.Join(parts, "\n")
}

// CommentAbove is the comment directly above line n: // lines and //* *//
// blocks right before it, with no blank line between.
func CommentAbove(lines []string, n int) string {
	var groups []string
	for k := n - 1; k >= 0; {
		t := strings.TrimSpace(lines[k])
		switch {
		case strings.HasSuffix(t, "*//"):
			start := k
			for start >= 0 && !strings.Contains(lines[start], "//*") {
				start--
			}
			if start < 0 {
				return strings.Join(groups, "\n")
			}
			groups = append([]string{blockText(lines[start : k+1])}, groups...)
			k = start - 1
		case strings.HasPrefix(t, "//"):
			groups = append([]string{lineText(t)}, groups...)
			k--
		default:
			return strings.Join(groups, "\n")
		}
	}
	return strings.Join(groups, "\n")
}

// CommentInside is the comment a function's body starts with, after its
// def on line def: // lines and //* *// blocks before the first line of
// code (blank lines before it are fine; a blank line after it ends it).
func CommentInside(lines []string, def int) string {
	var groups []string
	for k := def + 1; k < len(lines); {
		t := strings.TrimSpace(lines[k])
		switch {
		case t == "":
			if len(groups) > 0 {
				return strings.Join(groups, "\n")
			}
			k++
		case strings.HasPrefix(t, "//*"):
			end := -1 // the line the block closes on
			if strings.Contains(t[3:], "*//") {
				end = k
			} else {
				for j := k + 1; j < len(lines); j++ {
					if strings.Contains(lines[j], "*//") {
						end = j
						break
					}
				}
			}
			if end < 0 {
				return strings.Join(groups, "\n")
			}
			groups = append(groups, blockText(lines[k:end+1]))
			k = end + 1
		case strings.HasPrefix(t, "//"):
			groups = append(groups, lineText(t))
			k++
		default:
			return strings.Join(groups, "\n")
		}
	}
	return strings.Join(groups, "\n")
}

// lineText is a // line's words.
func lineText(t string) string {
	return strings.TrimPrefix(strings.TrimPrefix(t, "//"), " ")
}

// blockText is a //* *// block's words: the markers gone, each line
// trimmed, blank lines at either end dropped.
func blockText(lines []string) string {
	text := strings.Join(lines, "\n")
	if i := strings.Index(text, "//*"); i >= 0 {
		text = text[i+3:]
	}
	if i := strings.LastIndex(text, "*//"); i >= 0 {
		text = text[:i]
	}
	var out []string
	for _, l := range strings.Split(text, "\n") {
		out = append(out, strings.TrimSpace(l))
	}
	for len(out) > 0 && out[0] == "" {
		out = out[1:]
	}
	for len(out) > 0 && out[len(out)-1] == "" {
		out = out[:len(out)-1]
	}
	return strings.Join(out, "\n")
}

// FileDoc is a file's description, and the line after it: the comment the
// file starts with, // lines or a //* *// block. A comment directly above
// a def or assemble (no blank line between) is that one's, not the file's.
func FileDoc(lines []string) (string, int) {
	var groups []string
	k := 0
	for k < len(lines) {
		t := strings.TrimSpace(lines[k])
		switch {
		case strings.HasPrefix(t, "//*"):
			end := -1 // the line the block closes on
			if strings.Contains(t[3:], "*//") {
				end = k
			} else {
				for j := k + 1; j < len(lines); j++ {
					if strings.Contains(lines[j], "*//") {
						end = j
						break
					}
				}
			}
			if end < 0 {
				return "", 0 // never closed: not a description
			}
			groups = append(groups, blockText(lines[k:end+1]))
			k = end + 1
		case strings.HasPrefix(t, "//"):
			groups = append(groups, lineText(t))
			k++
		default:
			if len(groups) > 0 && (strings.HasPrefix(t, "def ") || strings.HasPrefix(t, "assemble ")) {
				return "", 0
			}
			return strings.Join(groups, "\n"), k
		}
	}
	return strings.Join(groups, "\n"), k
}
