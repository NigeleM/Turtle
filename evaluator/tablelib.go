package evaluator

import (
	"strings"
	"unicode"

	"Turtle/object"
)

// table[x] (in the "data" module) lays a collection out as a text table,
// so "show table[rows] ." prints one row per line instead of everything
// on one line. It returns a string, which can also be written to a file.
//
//	list of maps            one row per map, one column per key (SQL rows
//	                        from sql_query are this shape)
//	list of assembled       one row per value, one column per field
//	list of lists           one row per inner list, columns 0, 1, 2, ...
//	any other list or set   columns "#" (counting from 0) and "value"
//	one map                 columns "key" and "value"
//	one assembled value     columns "field" and "value"
//
// Columns appear in the order their keys or fields are first seen. A row
// without a column's key leaves that cell blank, which is different from
// a value of none. Columns of numbers are right-aligned.
//
// At most tablerows rows are shown, then a "... N more rows" line.
// "import data" sets tablerows to defaultTableRows in the importing file;
// the program changes it like any variable (tablerows = 50), and none
// shows every row. table[x, n] shows at most n rows for that call only.
func dataTable(args []object.Object, env *object.Environment) object.Object {
	if len(args) != 1 && len(args) != 2 {
		fatalf("'table' expects 1 or 2 arguments (a list, set, map, or assembled value, and how many rows to show), got %d", len(args))
	}
	limit := defaultTableRows
	if v, ok := env.Get(tableRowsName); ok {
		limit = rowLimit(v, "tablerows")
	}
	if len(args) == 2 {
		limit = rowLimit(args[1], "'table' row count")
	}
	t := tableShape("table", args[0])
	return &object.String{Value: t.render(limit)}
}

// tableShape lays any collection out as a header and rows, the one
// layout table[], table_write and sql's file functions all share.
func tableShape(fn string, x object.Object) textTable {
	var t textTable
	switch v := x.(type) {
	case *object.List:
		t = elementsTable(v.Elements)
	case *object.Set:
		t = elementsTable(v.Elements)
	case *object.Map:
		t = textTable{header: []string{"key", "value"}}
		for _, k := range v.Keys {
			t.add([]object.Object{v.KeyOf(k), v.Values[k]})
		}
	case *object.Assembly:
		t = textTable{header: []string{"field", "value"}}
		for i, f := range v.Shape.Fields {
			t.add([]object.Object{&object.String{Value: f}, v.Values[i]})
		}
	default:
		fatalf("'%s' needs a list, set, map, or assembled value, got %s", fn, typeName(x))
	}
	return t
}

const (
	tableRowsName    = "tablerows"
	defaultTableRows = 20
)

// defineTableRows gives the importing file its tablerows setting, unless
// the file already has a variable of that name.
func defineTableRows(env *object.Environment) {
	if _, ok := env.Get(tableRowsName); !ok {
		env.Set(tableRowsName, &object.Integer{Value: defaultTableRows})
	}
}

// rowLimit reads a row count: a whole number of 0 or more, or none for
// no limit (returned as -1).
func rowLimit(v object.Object, what string) int {
	switch n := v.(type) {
	case *object.None:
		return -1
	case *object.Integer:
		if n.Value >= 0 {
			if n.Value > int64(^uint(0)>>1) {
				return -1
			}
			return int(n.Value)
		}
	}
	fatalf("%s must be a whole number of 0 or more, or none to show every row, got %s", what, v.Inspect())
	return 0
}

// elementsTable picks the layout for the elements of a list or set: rows
// of maps or assembled values (mixed freely), rows of lists, or else one
// numbered row per element.
func elementsTable(elems []object.Object) textTable {
	if len(elems) == 0 {
		return textTable{empty: true}
	}
	if allRecords(elems) {
		header, rows := recordRows(elems)
		return textTable{header: header, rows: rows}
	}
	if allLists(elems) {
		var t textTable
		for _, e := range elems {
			inner := e.(*object.List).Elements
			for len(t.header) < len(inner) {
				t.header = append(t.header, itoa(len(t.header)))
			}
			t.rows = append(t.rows, inner)
		}
		for i, r := range t.rows {
			t.rows[i] = append(append([]object.Object{}, r...), make([]object.Object, len(t.header)-len(r))...)
		}
		return t
	}
	t := textTable{header: []string{"#", "value"}}
	for i, e := range elems {
		t.add([]object.Object{&object.Integer{Value: int64(i)}, e})
	}
	return t
}

func allRecords(elems []object.Object) bool {
	for _, e := range elems {
		switch e.(type) {
		case *object.Map, *object.Assembly:
		default:
			return false
		}
	}
	return true
}

func allLists(elems []object.Object) bool {
	for _, e := range elems {
		if _, ok := e.(*object.List); !ok {
			return false
		}
	}
	return true
}

// recordRows lines up maps and assembled values as rows: one column per
// key or field, in the order first seen, so rows with different keys
// still line up. A row without a column's key has a nil cell there.
func recordRows(elems []object.Object) ([]string, [][]object.Object) {
	var header []string
	col := map[string]int{}
	for _, e := range elems {
		names, _ := recordFields(e)
		for _, n := range names {
			if _, ok := col[n]; !ok {
				col[n] = len(header)
				header = append(header, n)
			}
		}
	}
	rows := make([][]object.Object, len(elems))
	for i, e := range elems {
		names, vals := recordFields(e)
		rows[i] = make([]object.Object, len(header))
		for j, n := range names {
			rows[i][col[n]] = vals[j]
		}
	}
	return header, rows
}

// recordFields gives a map's keys (as text) or an assembled value's field
// names, with the matching values.
func recordFields(o object.Object) ([]string, []object.Object) {
	switch r := o.(type) {
	case *object.Map:
		names := make([]string, len(r.Keys))
		vals := make([]object.Object, len(r.Keys))
		for i, k := range r.Keys {
			names[i] = r.KeyOf(k).Inspect()
			vals[i] = r.Values[k]
		}
		return names, vals
	case *object.Assembly:
		return r.Shape.Fields, r.Values
	}
	return nil, nil
}

// textTable holds the cells before layout. A nil cell is a missing one
// and stays blank.
type textTable struct {
	header []string
	rows   [][]object.Object
	empty  bool
}

func (t *textTable) add(row []object.Object) { t.rows = append(t.rows, row) }

func (t *textTable) render(limit int) string {
	if t.empty {
		return "no rows"
	}
	hidden := 0
	if limit >= 0 && len(t.rows) > limit {
		hidden = len(t.rows) - limit
		t.rows = t.rows[:limit]
	}
	n := len(t.header)
	cells := make([][]string, len(t.rows))
	widths := make([]int, n)
	numeric := make([]bool, n)
	for c, h := range t.header {
		widths[c] = displayWidth(h)
		numeric[c] = true
	}
	hasNumber := make([]bool, n)
	for r, row := range t.rows {
		cells[r] = make([]string, n)
		for c := 0; c < n; c++ {
			var v object.Object
			if c < len(row) {
				v = row[c]
			}
			switch v.(type) {
			case *object.Integer, *object.Float:
				hasNumber[c] = true
			case nil, *object.None:
			default:
				numeric[c] = false
			}
			cells[r][c] = cellText(v)
			if w := displayWidth(cells[r][c]); w > widths[c] {
				widths[c] = w
			}
		}
	}
	for c := range numeric {
		numeric[c] = numeric[c] && hasNumber[c]
	}

	var b strings.Builder
	line := func(texts []string) {
		var l strings.Builder
		for c, s := range texts {
			if c > 0 {
				l.WriteString("  ")
			}
			pad := strings.Repeat(" ", widths[c]-displayWidth(s))
			if numeric[c] {
				l.WriteString(pad + s)
			} else {
				l.WriteString(s + pad)
			}
		}
		b.WriteString(strings.TrimRight(l.String(), " "))
	}
	line(t.header)
	dashes := make([]string, n)
	for c, w := range widths {
		dashes[c] = strings.Repeat("-", w)
	}
	b.WriteString("\n")
	line(dashes)
	for _, row := range cells {
		b.WriteString("\n")
		line(row)
	}
	if hidden == 1 {
		b.WriteString("\n... 1 more row")
	} else if hidden > 1 {
		b.WriteString("\n... " + itoa(hidden) + " more rows")
	}
	return b.String()
}

// cellText is a value as one table cell: text without quotes (so a column
// of names reads cleanly), with line breaks and tabs written as \n and \t
// so a cell never breaks the layout; anything else as show prints it
// inside a list.
func cellText(v object.Object) string {
	switch s := v.(type) {
	case nil:
		return ""
	case *object.String:
		return strings.NewReplacer("\r", `\r`, "\n", `\n`, "\t", `\t`).Replace(s.Value)
	}
	return v.Inspect()
}

// displayWidth is how many terminal columns s takes: combining marks and
// joiners take none, wide East Asian characters and emoji take two.
func displayWidth(s string) int {
	w := 0
	for _, r := range s {
		switch {
		case unicode.Is(unicode.Mn, r) || unicode.Is(unicode.Me, r) || unicode.Is(unicode.Cf, r),
			r >= 0xFE00 && r <= 0xFE0F: // variation selectors
		case isWide(r):
			w += 2
		default:
			w++
		}
	}
	return w
}

func isWide(r rune) bool {
	switch {
	case r >= 0x1100 && r <= 0x115F, // Hangul Jamo
		r >= 0x2E80 && r <= 0x303E, // CJK radicals, punctuation
		r >= 0x3041 && r <= 0x33FF, // kana, CJK symbols
		r >= 0x3400 && r <= 0x4DBF, // CJK extension A
		r >= 0x4E00 && r <= 0x9FFF, // CJK ideographs
		r >= 0xA000 && r <= 0xA4CF, // Yi
		r >= 0xAC00 && r <= 0xD7A3, // Hangul syllables
		r >= 0xF900 && r <= 0xFAFF, // CJK compatibility
		r >= 0xFE30 && r <= 0xFE4F, // CJK compatibility forms
		r >= 0xFF00 && r <= 0xFF60, // fullwidth forms
		r >= 0xFFE0 && r <= 0xFFE6,
		r >= 0x1F300 && r <= 0x1F64F, // emoji: symbols, faces
		r >= 0x1F680 && r <= 0x1F6FF, // transport
		r >= 0x1F900 && r <= 0x1F9FF, // more emoji
		r >= 0x1FA70 && r <= 0x1FAFF,
		r >= 0x20000 && r <= 0x3FFFD: // CJK extensions B and on
		return true
	}
	return false
}
