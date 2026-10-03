package evaluator

import (
	"errors"
	"io/fs"
	"strings"

	"Turtle/object"
	"Turtle/sqlite"
)

// The "sql" builtin module: databases, through drivers written from
// scratch (no third-party code). SQLite files today; PostgreSQL later,
// behind the same functions.
//
//	db = sql_open["shop.db"]                        // or "sqlite:shop.db"
//	rows = sql_query[db, "SELECT title, price FROM books WHERE price < ?", list [1000]]
//	// rows: a list of maps, one per row: [ { "title": "Dune", "price": 950 } ]
//	names = sql_tables[db]
//	sql_close[db]
//
// NULL is none; integers, reals and text come back as integers, floats
// and text; a BLOB comes back as text. Parameters can be integers,
// floats, text, booleans (1/0), none (NULL) and dates (as text). A bad
// query or database is an error of kind sql.
func (it *Interpreter) callSQL(name string, args []object.Object) object.Object {
	switch name {
	case "sql_open":
		requireFuncArgs(name, args, 1)
		return it.sqlOpen(asStringArg(name, args[0]))
	case "sql_query":
		if len(args) != 2 && len(args) != 3 {
			fatalf("'sql_query' expects 2 or 3 arguments (database, query [, list of values for the ? placeholders]), got %d", len(args))
		}
		conn := asDatabaseArg(name, args[0])
		params := sqlParams(name, args[2:])
		cols, rows, err := conn.Query(asStringArg(name, args[1]), params)
		if err != nil {
			fatalKind(kindSQL, "sql_query: %v", err)
		}
		out := &object.List{}
		for _, r := range rows {
			m := object.NewMap()
			for i, c := range cols {
				m.Put(&object.String{Value: c}, fromSQL(r[i]))
			}
			out.Elements = append(out.Elements, m)
		}
		return out
	case "sql_tables":
		requireFuncArgs(name, args, 1)
		out := &object.List{}
		for _, t := range asDatabaseArg(name, args[0]).Tables() {
			out.Elements = append(out.Elements, &object.String{Value: t})
		}
		return out
	case "sql_close":
		requireFuncArgs(name, args, 1)
		d, ok := args[0].(*object.Database)
		if !ok {
			fatalf("'sql_close' needs a database (from sql_open), got %s", args[0].Type())
		}
		if !d.Closed {
			d.Conn.(*sqlite.DB).Close()
			d.Closed = true
		}
		return object.NoneValue
	}
	fatalKind(kindName, "no sql function %q", name)
	return nil
}

func (it *Interpreter) sqlOpen(target string) object.Object {
	path := target
	switch {
	case strings.HasPrefix(target, "postgres://"), strings.HasPrefix(target, "postgresql://"):
		fatalKind(kindSQL, "sql_open: PostgreSQL isn't supported yet; only SQLite files for now")
	case strings.HasPrefix(target, "sqlite:"):
		path = strings.TrimPrefix(target, "sqlite:")
	}
	db, err := sqlite.Open(it.resolvePath(path))
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			fatalKind(kindFile, "sql_open %s: no such file (creating new databases comes with writing, in a later version)", path)
		}
		var se *sqlite.Error
		if errors.As(err, &se) {
			// Name the file as the program wrote it, not the full path.
			fatalKind(kindSQL, "sql_open %s: %s", path, strings.ReplaceAll(se.Msg, it.resolvePath(path), path))
		}
		fatalKind(kindFile, "sql_open %s: %s", path, fileProblem(err))
	}
	return &object.Database{Name: path, Conn: db}
}

func asDatabaseArg(fn string, obj object.Object) *sqlite.DB {
	d, ok := obj.(*object.Database)
	if !ok {
		fatalf("'%s' needs a database (from sql_open), got %s", fn, obj.Type())
	}
	if d.Closed {
		fatalKind(kindSQL, "%s: database %s is closed", fn, d.Name)
	}
	return d.Conn.(*sqlite.DB)
}

// sqlParams turns the optional list of ? values into SQL values.
func sqlParams(fn string, rest []object.Object) []sqlite.Value {
	if len(rest) == 0 {
		return nil
	}
	var elems []object.Object
	switch l := rest[0].(type) {
	case *object.List:
		elems = l.Elements
	default:
		fatalf("'%s' values for the ? placeholders must be a list, e.g. list [1000], got %s", fn, rest[0].Type())
	}
	out := make([]sqlite.Value, len(elems))
	for i, e := range elems {
		switch v := e.(type) {
		case *object.None:
			out[i] = nil
		case *object.Integer:
			out[i] = v.Value
		case *object.Float:
			out[i] = v.Value
		case *object.String:
			out[i] = v.Value
		case *object.Boolean:
			if v.Value {
				out[i] = int64(1)
			} else {
				out[i] = int64(0)
			}
		case *object.Date:
			out[i] = v.Inspect()
		default:
			fatalf("'%s' value %d for a ? placeholder can't be a %s (use integers, floats, text, booleans, none or dates)", fn, i+1, e.Type())
		}
	}
	return out
}

func fromSQL(v sqlite.Value) object.Object {
	switch x := v.(type) {
	case nil:
		return object.NoneValue
	case int64:
		return &object.Integer{Value: x}
	case float64:
		return &object.Float{Value: x}
	case string:
		return &object.String{Value: x}
	case []byte:
		return &object.String{Value: string(x)}
	}
	return object.NoneValue
}
