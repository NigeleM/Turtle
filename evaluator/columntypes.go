package evaluator

import (
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"

	"Turtle/object"
)

// Column types: the map table_read and sql_load take after the file, so
// a file's text cells come in as the right kind of value and "007" can
// stay text while "42" becomes a number:
//
//	types = map ["sku": "text", "qty": "integer", "price": "float", "primary_key": "sku"]
//	rows = table_read["books.csv", types]
//	sql_load[db, "books", "books.csv", types]
//
// A type is a Turtle word (string, text, integer, float, boolean, date)
// or a SQL one (VARCHAR(20), BIGINT, DOUBLE PRECISION, NUMERIC(10, 2),
// ...), in any case. Columns the map leaves out are text. "primary_key"
// names the key column: sql_load makes it the new table's PRIMARY KEY,
// and table_read checks every row has a different one.

type columnKind int

const (
	colText columnKind = iota
	colInteger
	colFloat
	colNumber // a SQL NUMERIC or DECIMAL: an integer when whole, else a float
	colBoolean
	colDate
)

var columnKindNames = map[columnKind]string{
	colText: "text", colInteger: "an integer", colFloat: "a float",
	colNumber: "a number", colBoolean: "true or false", colDate: "a date",
}

// columnType is one column's type: its kind, and the SQL type to create
// it with when it was written as SQL ("" for a Turtle word, which each
// database spells its own way).
type columnType struct {
	kind columnKind
	sql  string
}

type columnTypes struct {
	cols  map[string]columnType // by the name as the map wrote it
	order []string
	key   string // the primary_key column, or ""
}

const primaryKeyEntry = "primary_key"

var turtleTypeWords = map[string]columnKind{
	"string": colText, "text": colText,
	"integer": colInteger, "int": colInteger,
	"float":   colFloat,
	"boolean": colBoolean, "bool": colBoolean,
	"date": colDate,
}

// sqlTypeWord is a SQL type as people write them: words, then maybe a
// size, as in VARCHAR(20), DOUBLE PRECISION or NUMERIC(10, 2). Nothing
// else gets into a CREATE TABLE.
var sqlTypeWord = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_]*( [A-Za-z][A-Za-z0-9_]*)*( ?\( *[0-9]+ *(, *[0-9]+ *)?\))?$`)

// sqlTypeKind sorts a SQL type the way SQLite does (by the words in its
// name), adding booleans and dates.
func sqlTypeKind(word string) (columnKind, bool) {
	u := strings.ToUpper(word)
	switch {
	case strings.Contains(u, "INT"):
		return colInteger, true
	case strings.Contains(u, "CHAR"), strings.Contains(u, "CLOB"), strings.Contains(u, "TEXT"), strings.Contains(u, "BLOB"):
		return colText, true
	case strings.Contains(u, "REAL"), strings.Contains(u, "FLOA"), strings.Contains(u, "DOUB"):
		return colFloat, true
	case strings.Contains(u, "BOOL"):
		return colBoolean, true
	case strings.Contains(u, "DATE"), strings.Contains(u, "TIME"):
		return colDate, true
	case strings.Contains(u, "NUMERIC"), strings.Contains(u, "DECIMAL"), strings.Contains(u, "NUMBER"):
		return colNumber, true
	}
	return colText, false
}

// parseColumnTypes reads the types map; none means no map.
func parseColumnTypes(fn string, v object.Object) *columnTypes {
	if _, ok := v.(*object.None); ok {
		return nil
	}
	m, ok := v.(*object.Map)
	if !ok {
		fatalf("'%s' column types must be a map of column name to type, e.g. map [\"qty\": \"integer\"], got %s", fn, typeName(v))
	}
	ct := &columnTypes{cols: map[string]columnType{}}
	for _, k := range m.Keys {
		name, ok := m.KeyOf(k).(*object.String)
		if !ok {
			fatalf("'%s' column types: a column name must be text, got %s", fn, typeName(m.KeyOf(k)))
		}
		val, ok := m.Values[k].(*object.String)
		if !ok {
			fatalf("'%s' column types: the type of %q must be text, like \"integer\", got %s", fn, name.Value, typeName(m.Values[k]))
		}
		word := strings.Join(strings.Fields(val.Value), " ")
		if name.Value == primaryKeyEntry {
			if word == "" {
				fatalf("'%s' column types: \"primary_key\" names the key column, e.g. \"primary_key\": \"sku\"", fn)
			}
			ct.key = val.Value
			continue
		}
		if kind, ok := turtleTypeWords[strings.ToLower(word)]; ok {
			ct.add(name.Value, columnType{kind: kind})
			continue
		}
		kind, known := sqlTypeKind(word)
		if !known || !sqlTypeWord.MatchString(word) {
			fatalf("'%s' column types: %q isn't a type for column %q; use string, text, integer, float, boolean or date, or a SQL type such as VARCHAR(20) or NUMERIC(10, 2)", fn, val.Value, name.Value)
		}
		ct.add(name.Value, columnType{kind: kind, sql: word})
	}
	return ct
}

func (ct *columnTypes) add(name string, t columnType) {
	ct.cols[name] = t
	ct.order = append(ct.order, name)
}

// headerIndex finds a column in a file's header: its exact name, or one
// that differs only in case (as SQL column names do).
func headerIndex(header []string, name string) int {
	for i, h := range header {
		if h == name {
			return i
		}
	}
	for i, h := range header {
		if strings.EqualFold(h, name) {
			return i
		}
	}
	return -1
}

// perColumn gives each of the header's columns its type: text unless the
// map says otherwise. A column the map names that the file doesn't have
// is an error, since it's almost always a typo.
func (ct *columnTypes) perColumn(fn, path string, header []string) []columnType {
	out := make([]columnType, len(header))
	if ct == nil {
		return out
	}
	missing := func(name string) {
		fatalKind(kindName, "%s %s: the file has no column %q (its columns: %s)", fn, path, name, strings.Join(header, ", "))
	}
	for _, name := range ct.order {
		i := headerIndex(header, name)
		if i < 0 {
			missing(name)
		}
		out[i] = ct.cols[name]
	}
	if ct.key != "" && headerIndex(header, ct.key) < 0 {
		missing(ct.key)
	}
	return out
}

// applyColumnTypes turns each cell into its column's kind of value, in
// place. A cell that can't be one is an error naming the row and column.
func applyColumnTypes(fn, path string, header []string, rows [][]object.Object, ct *columnTypes) []columnType {
	types := ct.perColumn(fn, path, header)
	for r, row := range rows {
		for i := range row {
			row[i] = convertCell(fn, path, r+1, header[i], types[i].kind, row[i])
		}
	}
	return types
}

func convertCell(fn, path string, row int, col string, kind columnKind, v object.Object) object.Object {
	if _, ok := v.(*object.None); ok || v == nil {
		return object.NoneValue
	}
	bad := func(errKind string) object.Object {
		fatalKind(errKind, "%s %s: row %d, column %q: %s isn't %s", fn, path, row, col, object.Shown(v), columnKindNames[kind])
		return nil
	}
	s, isText := v.(*object.String)
	text := ""
	if isText {
		text = strings.TrimSpace(s.Value)
	}
	switch kind {
	case colText:
		switch x := v.(type) {
		case *object.String:
			return x
		case *object.List, *object.Map:
			return &object.String{Value: string(toJSON(fn, v, ""))}
		}
		return &object.String{Value: fileCell(v)}
	case colInteger:
		switch x := v.(type) {
		case *object.Integer:
			return x
		case *object.Float:
			if x.Value == math.Trunc(x.Value) && math.Abs(x.Value) < 1<<63 {
				return &object.Integer{Value: int64(x.Value)}
			}
		case *object.String:
			if n, err := strconv.ParseInt(text, 10, 64); err == nil {
				return &object.Integer{Value: n}
			}
			if f, err := strconv.ParseFloat(text, 64); err == nil && f == math.Trunc(f) && math.Abs(f) < 1<<63 {
				return &object.Integer{Value: int64(f)}
			}
		}
		return bad(kindNumber)
	case colFloat:
		switch x := v.(type) {
		case *object.Float:
			return x
		case *object.Integer:
			return &object.Float{Value: float64(x.Value)}
		case *object.String:
			if f, err := strconv.ParseFloat(text, 64); err == nil && !math.IsInf(f, 0) && !math.IsNaN(f) {
				return &object.Float{Value: f}
			}
		}
		return bad(kindNumber)
	case colNumber:
		switch x := v.(type) {
		case *object.Integer, *object.Float:
			return x
		case *object.String:
			if n, err := strconv.ParseInt(text, 10, 64); err == nil {
				return &object.Integer{Value: n}
			}
			if f, err := strconv.ParseFloat(text, 64); err == nil && !math.IsInf(f, 0) && !math.IsNaN(f) {
				return &object.Float{Value: f}
			}
		}
		return bad(kindNumber)
	case colBoolean:
		switch x := v.(type) {
		case *object.Boolean:
			return x
		case *object.Integer:
			if x.Value == 0 || x.Value == 1 {
				return &object.Boolean{Value: x.Value == 1}
			}
		case *object.String:
			switch strings.ToLower(text) {
			case "true", "yes", "1", "t", "y":
				return &object.Boolean{Value: true}
			case "false", "no", "0", "f", "n":
				return &object.Boolean{Value: false}
			}
		}
		return bad(kindType)
	case colDate:
		switch x := v.(type) {
		case *object.Date:
			return x
		case *object.String:
			if d, ok := tryParseDate(text); ok {
				return d
			}
		}
		return bad(kindDate)
	}
	return v
}

// checkKeyColumn makes sure every row has a key, and no two the same.
func checkKeyColumn(fn, path string, header []string, rows [][]object.Object, key string) {
	i := headerIndex(header, key)
	seen := map[string]int{}
	for r, row := range rows {
		v := row[i]
		if _, ok := v.(*object.None); ok {
			fatalKind(kindKey, "%s %s: row %d has no %q, the primary key", fn, path, r+1, key)
		}
		k := object.Key(v)
		if first, dup := seen[k]; dup {
			fatalKind(kindKey, "%s %s: rows %d and %d have the same %q, %s (the primary key)", fn, path, first, r+1, key, object.Shown(v))
		}
		seen[k] = r + 1
	}
}

// createColumnType is how the database spells a column's type in CREATE
// TABLE: SQL as it was written, a Turtle word as the database's own.
func createColumnType(conn sqlConn, t columnType, isKey bool) string {
	if t.sql != "" {
		return t.sql
	}
	_, isPG := conn.(postgresConn)
	_, isMy := conn.(mysqlConn)
	switch t.kind {
	case colInteger:
		if isPG || isMy {
			return "BIGINT"
		}
		return "INTEGER"
	case colFloat:
		switch {
		case isPG:
			return "DOUBLE PRECISION"
		case isMy:
			return "DOUBLE"
		}
		return "REAL"
	case colBoolean:
		return "BOOLEAN"
	case colDate:
		return "DATE"
	}
	if isMy && isKey {
		return "VARCHAR(255)" // MySQL can't key a TEXT column
	}
	return "TEXT"
}

// dateForSQL writes a date for a date column: just the day when it has
// no time of day, so DATE columns take it everywhere.
func dateForSQL(v object.Object) object.Object {
	d, ok := v.(*object.Date)
	if !ok {
		return v
	}
	t := d.Time
	if t.Hour() == 0 && t.Minute() == 0 && t.Second() == 0 {
		return &object.String{Value: t.Format(time.DateOnly)}
	}
	return &object.String{Value: sqlDateText(d)}
}

// sqlDateText is a date for a database: "2026-10-03 14:05:00", and for a
// date in another zone its offset too ("2026-12-25 09:00:00+00:00"),
// which PostgreSQL, MySQL and to_date all read.
func sqlDateText(d *object.Date) string {
	if d.Local() {
		return d.Text()
	}
	return d.Time.Format("2006-01-02 15:04:05-07:00")
}
