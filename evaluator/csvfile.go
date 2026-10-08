package evaluator

import (
	"strconv"
	"strings"

	"Turtle/object"
)

// .csv and .tsv files (RFC 4180). As in pandas and spreadsheets, what a
// cell holds decides its kind, not its quotes: quotes only let a value
// hold the separator, a quote or a line break.
//
//	price,code,note
//	950,"950",007        -> 950, 950 (integers), "007" (text)
//
// Only a plain number is read as one: digits, maybe a minus sign, a
// decimal point and an exponent (7, -3, 2.5, 1e5, 6.02e-23). 007, +5,
// .5, 1,000 and a whole number
// too big for an integer stay text, since each is as likely a code or an
// id (unlike pandas, which drops 007's zeros). An empty cell is none.
// Column types (columntypes.go) say otherwise column by column.

// csvCell is one cell's text, without its quotes.
type csvCell struct {
	text string
}

// csvError is a file that isn't well formed, at a line of the file.
type csvError struct {
	line int
	msg  string
}

func (e *csvError) Error() string { return "line " + strconv.Itoa(e.line) + ": " + e.msg }

// csvRecord is one line's cells, and the line it starts on.
type csvRecord struct {
	line  int
	cells []csvCell
}

// parseCSV splits text into records. sep is ',' or '\t'. A line break
// ends a record (\n or \r\n), except inside quotes.
func parseCSV(text string, sep byte) ([]csvRecord, error) {
	var out []csvRecord
	line := 1
	i := 0
	for i < len(text) {
		rec := csvRecord{line: line}
		for {
			var cell csvCell
			if i < len(text) && text[i] == '"' {
				start := line
				i++
				var b strings.Builder
				for {
					if i >= len(text) {
						return nil, &csvError{start, `a quoted value has no closing "`}
					}
					c := text[i]
					if c == '"' {
						if i+1 < len(text) && text[i+1] == '"' {
							b.WriteByte('"')
							i += 2
							continue
						}
						i++
						break
					}
					if c == '\n' {
						line++
					}
					b.WriteByte(c)
					i++
				}
				cell.text = b.String()
				if i < len(text) && text[i] != sep && text[i] != '\n' && !(text[i] == '\r' && i+1 < len(text) && text[i+1] == '\n') {
					return nil, &csvError{line, `text after a quoted value's closing "; quote the whole value, with any " inside doubled ("")`}
				}
			} else {
				start := i
				for i < len(text) && text[i] != sep && text[i] != '\n' {
					if text[i] == '"' {
						return nil, &csvError{line, `a " inside a value that isn't quoted; quote the whole value, with any " inside doubled ("")`}
					}
					i++
				}
				cell.text = strings.TrimSuffix(text[start:i], "\r")
			}
			rec.cells = append(rec.cells, cell)
			if i < len(text) && text[i] == sep {
				i++
				continue
			}
			break
		}
		if i < len(text) && text[i] == '\r' {
			i++
		}
		if i < len(text) && text[i] == '\n' {
			i++
			line++
		}
		out = append(out, rec)
	}
	return out, nil
}

// isPlainNumber reports whether s is a number as a file writes one,
// -?(0|[1-9][0-9]*)(.[0-9]+)?([eE][+-]?[0-9]+)?: what a cell must be to
// be read as a number. (By hand, not a regular expression: it runs on
// every cell.)
func isPlainNumber(s string) bool {
	i := 0
	if i < len(s) && s[i] == '-' {
		i++
	}
	start := i
	for i < len(s) && s[i] >= '0' && s[i] <= '9' {
		i++
	}
	if i == start || s[start] == '0' && i-start > 1 {
		return false // no digits, or a leading zero (007)
	}
	if i == len(s) {
		return true
	}
	if s[i] != '.' {
		return isExponent(s[i:])
	}
	i++
	frac := i
	for i < len(s) && s[i] >= '0' && s[i] <= '9' {
		i++
	}
	if i == frac {
		return false
	}
	return i == len(s) || isExponent(s[i:])
}

// isExponent reports whether s is a number's exponent: e or E, maybe a
// sign, then digits (e5, E-18, e+3).
func isExponent(s string) bool {
	if len(s) < 2 || s[0] != 'e' && s[0] != 'E' {
		return false
	}
	i := 1
	if s[i] == '+' || s[i] == '-' {
		i++
	}
	if i == len(s) {
		return false
	}
	for ; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

// cellValue is what a cell reads as: a plain number is a number, an
// empty cell none, anything else text.
func cellValue(c csvCell) object.Object {
	if c.text == "" {
		return object.NoneValue
	}
	if isPlainNumber(c.text) {
		if !strings.ContainsAny(c.text, ".eE") {
			if n, err := strconv.ParseInt(c.text, 10, 64); err == nil {
				return object.Int(n)
			}
			return &object.String{Value: c.text} // too big: an id, not a quantity
		}
		if f, err := strconv.ParseFloat(c.text, 64); err == nil {
			return &object.Float{Value: f}
		}
	}
	return &object.String{Value: c.text}
}

// csvQuoting is how table_write quotes text: "needed" (the default) only
// when it has to; "text" every text value, as some programs want.
type csvQuoting string

const (
	quoteNeeded csvQuoting = "needed"
	quoteText   csvQuoting = "text"
)

// csvField writes one value. Numbers are never quoted. Text is quoted
// when it holds the separator, a quote or a line break, or starts or
// ends with a space; with quoteText, always.
func csvField(v object.Object, sep byte, q csvQuoting) string {
	s, isText := v.(*object.String)
	text := fileCell(v)
	if !isText {
		return text
	}
	text = s.Value
	need := q == quoteText && text != "" ||
		strings.ContainsAny(text, "\"\r\n"+string(sep)) ||
		strings.TrimSpace(text) != text
	if !need {
		return text
	}
	return `"` + strings.ReplaceAll(text, `"`, `""`) + `"`
}

// csvHeader writes a header name: quoted only when it has to be.
func csvHeader(name string, sep byte) string {
	if name != "" && !strings.ContainsAny(name, "\"\r\n"+string(sep)) && strings.TrimSpace(name) == name {
		return name
	}
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}
