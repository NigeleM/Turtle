package sqlite

import (
	"strings"
)

// Table is one table from the schema.
type Table struct {
	Name     string
	Root     uint32 // root page of its B-tree
	Columns  []Column
	RowidCol int // index of the INTEGER PRIMARY KEY column (which holds the rowid), or -1
	SQL      string
}

// Column is one column of a table.
type Column struct {
	Name     string
	Type     string // as declared: "INTEGER", "VARCHAR(20)", "" ...
	Affinity affinity
}

// Column affinity: how SQLite treats values compared with a column.
type affinity int

const (
	affBlob affinity = iota // no preference (also: no declared type)
	affText
	affNumeric
	affInteger
	affReal
)

// affinityOf applies SQLite's rules to a declared column type.
func affinityOf(declared string) affinity {
	t := strings.ToUpper(declared)
	switch {
	case strings.Contains(t, "INT"):
		return affInteger
	case strings.Contains(t, "CHAR"), strings.Contains(t, "CLOB"), strings.Contains(t, "TEXT"):
		return affText
	case strings.Contains(t, "BLOB"), t == "":
		return affBlob
	case strings.Contains(t, "REAL"), strings.Contains(t, "FLOA"), strings.Contains(t, "DOUB"):
		return affReal
	}
	return affNumeric
}

// The schema table itself, queryable as sqlite_schema or sqlite_master.
var schemaTable = &Table{
	Name: "sqlite_schema",
	Root: 1,
	Columns: []Column{
		{Name: "type", Type: "text", Affinity: affText},
		{Name: "name", Type: "text", Affinity: affText},
		{Name: "tbl_name", Type: "text", Affinity: affText},
		{Name: "rootpage", Type: "int", Affinity: affInteger},
		{Name: "sql", Type: "text", Affinity: affText},
	},
	RowidCol: -1,
}

// loadSchema reads the tables listed in sqlite_schema (page 1).
func (db *DB) loadSchema() error {
	db.tables = map[string]*Table{}
	db.names = nil
	return db.scanTable(1, func(_ int64, payload []byte) error {
		vals, err := db.decodeRecord(payload)
		if err != nil {
			return err
		}
		if len(vals) < 5 {
			return nil
		}
		kind, _ := vals[0].(string)
		name, _ := vals[1].(string)
		root, _ := vals[3].(int64)
		sqlText, _ := vals[4].(string)
		if kind != "table" || strings.HasPrefix(name, "sqlite_") {
			return nil
		}
		t, err := parseCreateTable(sqlText)
		if err != nil {
			return errorf("table %s: %v", name, err)
		}
		t.Name, t.Root, t.SQL = name, uint32(root), sqlText
		db.tables[strings.ToLower(name)] = t
		db.names = append(db.names, name)
		return nil
	})
}

// Tables lists the database's tables, in the order they were created.
func (db *DB) Tables() []string { return append([]string{}, db.names...) }

// table finds a table by name, ignoring case like SQLite.
func (db *DB) table(name string) (*Table, error) {
	lower := strings.ToLower(name)
	if lower == "sqlite_schema" || lower == "sqlite_master" {
		return schemaTable, nil
	}
	if t, ok := db.tables[lower]; ok {
		return t, nil
	}
	return nil, errorf("no such table: %s", name)
}

// parseCreateTable reads the column list of a CREATE TABLE statement:
// names, declared types, and which column (if any) is an INTEGER PRIMARY
// KEY, which SQLite stores as the rowid instead of in the record.
func parseCreateTable(src string) (*Table, error) {
	toks, err := lexSQL(src)
	if err != nil {
		return nil, err
	}
	// Skip to the opening parenthesis of the column list.
	i := 0
	for i < len(toks) && !(toks[i].kind == tOp && toks[i].text == "(") {
		if toks[i].kind == tIdent && !toks[i].quoted && strings.EqualFold(toks[i].text, "AS") {
			return nil, errorf("CREATE TABLE ... AS SELECT isn't supported")
		}
		i++
	}
	if i == len(toks) {
		return nil, errorf("no column list in %q", src)
	}
	t := &Table{RowidCol: -1}
	// Split the list on top-level commas.
	var defs [][]token
	depth, start := 0, i+1
	for j := i + 1; j < len(toks); j++ {
		tk := toks[j]
		if tk.kind != tOp {
			continue
		}
		switch tk.text {
		case "(":
			depth++
		case ")":
			if depth == 0 {
				defs = append(defs, toks[start:j])
				j = len(toks)
				break
			}
			depth--
		case ",":
			if depth == 0 {
				defs = append(defs, toks[start:j])
				start = j + 1
			}
		}
	}
	tableKeys := []string{}
	for _, d := range defs {
		if len(d) == 0 {
			continue
		}
		first := strings.ToUpper(d[0].text)
		if !d[0].quoted && (first == "CONSTRAINT" || first == "PRIMARY" || first == "UNIQUE" || first == "CHECK" || first == "FOREIGN") {
			// A table constraint: PRIMARY KEY (a) makes a an INTEGER
			// PRIMARY KEY when it's the only column and declared INTEGER.
			if first == "PRIMARY" {
				var cols []string
				for _, tk := range d {
					if tk.kind == tIdent && !isWord(tk, "PRIMARY", "KEY", "ASC", "DESC") {
						cols = append(cols, tk.text)
					}
				}
				tableKeys = cols
			}
			continue
		}
		col := Column{Name: d[0].text}
		// The type is the words after the name, up to the first
		// constraint keyword (and any parenthesised size).
		k := 1
		var typ []string
		for k < len(d) {
			tk := d[k]
			if tk.kind == tIdent && !tk.quoted && isConstraintWord(tk.text) {
				break
			}
			if tk.kind == tOp && tk.text == "(" {
				end := k
				for end < len(d) && !(d[end].kind == tOp && d[end].text == ")") {
					end++
				}
				typ = append(typ, src[tk.pos:min(d[min(end, len(d)-1)].end, len(src))])
				k = end + 1
				continue
			}
			typ = append(typ, tk.text)
			k++
		}
		col.Type = strings.Join(typ, " ")
		col.Affinity = affinityOf(col.Type)
		if strings.EqualFold(col.Type, "INTEGER") && hasPrimaryKey(d[k:]) {
			t.RowidCol = len(t.Columns)
		}
		t.Columns = append(t.Columns, col)
	}
	if len(tableKeys) == 1 && t.RowidCol < 0 {
		for ci, c := range t.Columns {
			if strings.EqualFold(c.Name, tableKeys[0]) && strings.EqualFold(c.Type, "INTEGER") {
				t.RowidCol = ci
			}
		}
	}
	if len(t.Columns) == 0 {
		return nil, errorf("no columns in %q", src)
	}
	// WITHOUT ROWID tables are stored differently (as an index B-tree).
	last := toks[len(toks)-2]
	if last.kind == tIdent && strings.EqualFold(last.text, "ROWID") {
		return nil, errorf("WITHOUT ROWID tables aren't supported yet")
	}
	return t, nil
}

func isConstraintWord(w string) bool {
	switch strings.ToUpper(w) {
	case "CONSTRAINT", "PRIMARY", "NOT", "NULL", "UNIQUE", "CHECK", "DEFAULT", "COLLATE", "REFERENCES", "GENERATED", "AS":
		return true
	}
	return false
}

// hasPrimaryKey reports whether column constraints include PRIMARY KEY
// (and not DESC, which keeps it an ordinary column, a SQLite quirk).
func hasPrimaryKey(d []token) bool {
	for k := 0; k+1 < len(d); k++ {
		if isWord(d[k], "PRIMARY") && isWord(d[k+1], "KEY") {
			return !(k+2 < len(d) && isWord(d[k+2], "DESC"))
		}
	}
	return false
}

// isWord reports whether tk is an unquoted identifier equal to one of
// words, ignoring case.
func isWord(tk token, words ...string) bool {
	if tk.kind != tIdent || tk.quoted {
		return false
	}
	for _, w := range words {
		if strings.EqualFold(tk.text, w) {
			return true
		}
	}
	return false
}
