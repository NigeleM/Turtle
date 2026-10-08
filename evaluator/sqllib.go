package evaluator

import (
	"errors"
	"io/fs"
	"net/url"
	"strings"
	"time"

	"Turtle/mysql"
	"Turtle/object"
	"Turtle/postgres"
	"Turtle/sqlite"
)

// The "sql" builtin module: databases, through drivers written from
// scratch (no third-party code): SQLite files, PostgreSQL and MySQL
// servers, all behind the same functions. The address picks which.
//
//	db = sql_open["shop.db"]                        // or "sqlite:shop.db"
//	db = sql_open["postgres://ann:pw@localhost:5432/shop"]
//	db = sql_open["mysql://ann:pw@localhost:3306/shop"]
//	db = sql_create["new.db"]                       // a new, empty database
//	rows = sql_query[db, "SELECT title, price FROM books WHERE price < ?", list [1000]]
//	// rows: a list of maps, one per row: [ { "title": "Dune", "price": 950 } ]
//	n = sql_run[db, "UPDATE books SET price = ? WHERE sku = ?", list [900, "B1"]]
//	// n: how many rows the change touched
//	names = sql_tables[db]
//	sql_save[db, "SELECT * FROM books", "books.csv"] // query results to a file
//	sql_load[db, "books", "new.csv"]                 // a file's rows into a table
//	sql_update[db, "books", "sku", "prices.csv"]     // change records by key
//	sql_delete[db, "books", "sku", "gone.csv"]       // remove records by key
//	sql_upsert[db, "books", "sku", "restock.csv"]    // add, or change if there
//	sql_close[db]
//
// NULL is none; integers, reals and text come back as integers, floats
// and text; a BLOB comes back as text; a server's booleans and dates
// come back as booleans and dates. Parameters can be integers,
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
		return rowMaps(cols, rows)
	case "sql_run":
		if len(args) != 2 && len(args) != 3 {
			fatalf("'sql_run' expects 2 or 3 arguments (database, statement [, list of values for the ? placeholders]), got %d", len(args))
		}
		conn := asDatabaseArg(name, args[0])
		params := sqlParams(name, args[2:])
		res, err := conn.Exec(asStringArg(name, args[1]), params)
		if err != nil {
			fatalKind(kindSQL, "sql_run: %v", err)
		}
		return &object.Integer{Value: res}
	case "sql_create":
		requireFuncArgs(name, args, 1)
		path := asStringArg(name, args[0])
		if serverAddress(path) {
			fatalKind(kindSQL, "sql_create makes SQLite files; a PostgreSQL or MySQL database is made on the server (CREATE DATABASE), then opened with sql_open")
		}
		db, err := sqlite.Create(it.resolvePath(path))
		if err != nil {
			if errors.Is(err, fs.ErrExist) {
				fatalKind(kindFile, "sql_create %s: the file already exists (sql_open opens it)", path)
			}
			fatalKind(kindFile, "sql_create %s: %s", path, fileProblem(err))
		}
		return it.track(&object.Database{Name: path, Conn: sqliteConn{db}})
	case "sql_tables":
		requireFuncArgs(name, args, 1)
		names, err := asDatabaseArg(name, args[0]).Tables()
		if err != nil {
			fatalKind(kindSQL, "sql_tables: %v", err)
		}
		out := &object.List{}
		for _, t := range names {
			out.Elements = append(out.Elements, &object.String{Value: t})
		}
		return out
	case "sql_save":
		if len(args) != 3 && len(args) != 4 {
			fatalf("'sql_save' expects 3 or 4 arguments (database, query, file [, list of values for the ? placeholders]), got %d", len(args))
		}
		conn := asDatabaseArg(name, args[0])
		params := sqlParams(name, args[3:])
		cols, rows, err := conn.Query(asStringArg(name, args[1]), params)
		if err != nil {
			fatalKind(kindSQL, "sql_save: %v", err)
		}
		t := textTable{header: cols}
		for _, r := range rows {
			row := make([]object.Object, len(r))
			for i, v := range r {
				row[i] = fromSQL(v)
			}
			t.add(row)
		}
		return &object.Integer{Value: int64(it.writeTableFile(name, asStringArg(name, args[2]), t))}
	case "sql_load":
		if len(args) != 3 && len(args) != 4 {
			fatalf("'sql_load' expects 3 or 4 arguments (database, table, file [, map of column types]), got %d", len(args))
		}
		conn := asDatabaseArg(name, args[0])
		table := asStringArg(name, args[1])
		path := asStringArg(name, args[2])
		var ct *columnTypes
		if len(args) == 4 {
			ct = parseColumnTypes(name, args[3])
		}
		header, rows := it.readTableFile(name, path)
		if len(header) == 0 {
			return &object.Integer{Value: 0}
		}
		types := applyColumnTypes(name, path, header, rows, ct)
		for _, row := range rows {
			for i, t := range types {
				if t.kind == colDate {
					row[i] = dateForSQL(row[i])
				}
			}
		}
		marks := strings.TrimSuffix(strings.Repeat("?, ", len(header)), ", ")
		stmt := "INSERT INTO " + conn.Quote(table) + " (" + quoteSQLNames(conn, header) + ") VALUES (" + marks + ")"
		if sqlHasTable(name, conn, table) {
			return it.sqlFileRows(name, conn, stmt, rows, nil)
		}
		key := ""
		if ct != nil {
			key = ct.key
		}
		var cols []string
		for i, h := range header {
			isKey := key != "" && headerIndex(header, key) == i
			col := conn.Quote(h) + " " + createColumnType(conn, types[i], isKey)
			if isKey {
				col += " PRIMARY KEY"
			}
			cols = append(cols, col)
		}
		if _, err := conn.Exec("CREATE TABLE "+conn.Quote(table)+" ("+strings.Join(cols, ", ")+")", nil); err != nil {
			fatalKind(kindSQL, "%s: making table %s: %v", name, table, err)
		}
		// The table is new: if the file is refused, it goes too.
		defer func() {
			if r := recover(); r != nil {
				conn.Exec("DROP TABLE "+conn.Quote(table), nil)
				panic(r)
			}
		}()
		return it.sqlFileRows(name, conn, stmt, rows, nil)
	case "sql_update", "sql_delete", "sql_upsert":
		requireFuncArgs(name, args, 4)
		conn := asDatabaseArg(name, args[0])
		table := asStringArg(name, args[1])
		key := asStringArg(name, args[2])
		path := asStringArg(name, args[3])
		header, rows := it.readTableFile(name, path)
		if len(header) == 0 {
			return &object.Integer{Value: 0}
		}
		ki := -1
		var others []string
		var order []int
		for i, h := range header {
			if strings.EqualFold(h, key) {
				ki = i
			} else {
				others = append(others, h)
				order = append(order, i)
			}
		}
		if ki < 0 {
			fatalKind(kindSQL, "%s %s: the file has no %q column (its columns: %s)", name, path, key, strings.Join(header, ", "))
		}
		t, k := conn.Quote(table), conn.Quote(key)
		switch name {
		case "sql_delete":
			return it.sqlFileRows(name, conn, "DELETE FROM "+t+" WHERE "+k+" = ?", rows, []int{ki})
		case "sql_update":
			if len(others) == 0 {
				fatalKind(kindSQL, "sql_update %s: the file has only the key column, so there's nothing to change", path)
			}
			var sets []string
			for _, c := range others {
				sets = append(sets, conn.Quote(c)+" = ?")
			}
			return it.sqlFileRows(name, conn, "UPDATE "+t+" SET "+strings.Join(sets, ", ")+" WHERE "+k+" = ?", rows, append(order, ki))
		}
		marks := strings.TrimSuffix(strings.Repeat("?, ", len(header)), ", ")
		stmt := "INSERT INTO " + t + " (" + quoteSQLNames(conn, header) + ") VALUES (" + marks + ") " + conn.Upsert(key, others)
		return it.sqlFileRows(name, conn, stmt, rows, nil)
	case "sql_close":
		requireFuncArgs(name, args, 1)
		d, ok := args[0].(*object.Database)
		if !ok {
			fatalf("'sql_close' needs a database (from sql_open), got %s", args[0].Type())
		}
		if !d.Closed {
			d.Conn.(sqlConn).Close()
			d.Closed = true
		}
		return object.NoneValue
	}
	fatalKind(kindName, "no sql function %q", name)
	return nil
}

// sqlHasTable reports whether the database has the table, by name in
// any case.
func sqlHasTable(fn string, conn sqlConn, table string) bool {
	names, err := conn.Tables()
	if err != nil {
		fatalKind(kindSQL, "%s: %v", fn, err)
	}
	for _, n := range names {
		if strings.EqualFold(n, table) {
			return true
		}
	}
	return false
}

// track remembers a database so Run can close it if the program doesn't.
func (it *Interpreter) track(d *object.Database) *object.Database {
	it.dbs = append(it.dbs, d)
	return d
}

func (it *Interpreter) closeDatabases() {
	for _, d := range it.dbs {
		if !d.Closed {
			d.Conn.(sqlConn).Close()
			d.Closed = true
		}
	}
	it.dbs = nil
}

// sqlConn is an open database of any kind.
type sqlConn interface {
	Query(sql string, params []any) ([]string, [][]any, error)
	Exec(sql string, params []any) (int64, error)
	ExecRows(sql string, rows [][]any) (int64, error)
	Tables() ([]string, error)
	Close() error
	Quote(name string) string                  // a table or column name, quoted
	Upsert(key string, others []string) string // the ON CONFLICT part of an upsert
}

type sqliteConn struct{ *sqlite.DB }

func (c sqliteConn) Exec(sql string, params []any) (int64, error) {
	res, err := c.DB.Exec(sql, params)
	return res.Changes, err
}

func (c sqliteConn) ExecRows(sql string, rows [][]any) (int64, error) {
	res, err := c.DB.ExecRows(sql, rows)
	return res.Changes, err
}

func (c sqliteConn) Tables() ([]string, error) { return c.DB.Tables(), nil }
func (c sqliteConn) Quote(name string) string  { return quoteSQLName(name) }
func (c sqliteConn) Upsert(key string, others []string) string {
	return onConflict(key, others)
}

type postgresConn struct{ *postgres.DB }

func (c postgresConn) Quote(name string) string { return quoteSQLName(name) }
func (c postgresConn) Upsert(key string, others []string) string {
	return onConflict(key, others)
}

type mysqlConn struct{ *mysql.DB }

func (c mysqlConn) Quote(name string) string { return mysql.QuoteIdent(name) }
func (c mysqlConn) Upsert(key string, others []string) string {
	var sets []string
	for _, o := range others {
		sets = append(sets, mysql.QuoteIdent(o)+" = VALUES("+mysql.QuoteIdent(o)+")")
	}
	if len(sets) == 0 { // only the key: a record already there stays as it is
		sets = []string{mysql.QuoteIdent(key) + " = " + mysql.QuoteIdent(key)}
	}
	return "ON DUPLICATE KEY UPDATE " + strings.Join(sets, ", ")
}

// onConflict is the upsert clause of SQLite and PostgreSQL.
func onConflict(key string, others []string) string {
	if len(others) == 0 {
		return "ON CONFLICT (" + quoteSQLName(key) + ") DO NOTHING"
	}
	var sets []string
	for _, c := range others {
		sets = append(sets, quoteSQLName(c)+" = excluded."+quoteSQLName(c))
	}
	return "ON CONFLICT (" + quoteSQLName(key) + ") DO UPDATE SET " + strings.Join(sets, ", ")
}

func serverAddress(target string) bool {
	for _, p := range []string{"postgres://", "postgresql://", "mysql://", "mariadb://"} {
		if strings.HasPrefix(target, p) {
			return true
		}
	}
	return false
}

// hidePassword writes a server address without its password, for messages.
func hidePassword(target string) string {
	if u, err := url.Parse(target); err == nil && u.User != nil {
		if _, ok := u.User.Password(); ok {
			u.User = url.UserPassword(u.User.Username(), "xxxxx")
			return u.String()
		}
	}
	return target
}

func (it *Interpreter) sqlOpen(target string) object.Object {
	path := target
	switch {
	case strings.HasPrefix(target, "postgres://"), strings.HasPrefix(target, "postgresql://"):
		db, err := postgres.Open(target)
		if err != nil {
			fatalKind(kindSQL, "sql_open: %v", err)
		}
		return it.track(&object.Database{Name: hidePassword(target), Conn: postgresConn{db}, Address: target})
	case strings.HasPrefix(target, "mysql://"), strings.HasPrefix(target, "mariadb://"):
		db, err := mysql.Open(target)
		if err != nil {
			fatalKind(kindSQL, "sql_open: %v", err)
		}
		return it.track(&object.Database{Name: hidePassword(target), Conn: mysqlConn{db}, Address: target})
	case strings.HasPrefix(target, "sqlite:"):
		path = strings.TrimPrefix(target, "sqlite:")
	}
	db, err := sqlite.Open(it.resolvePath(path))
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			fatalKind(kindFile, "sql_open %s: no such file (sql_create makes a new database)", path)
		}
		var se *sqlite.Error
		if errors.As(err, &se) {
			// Name the file as the program wrote it, not the full path.
			fatalKind(kindSQL, "sql_open %s: %s", path, strings.ReplaceAll(se.Msg, it.resolvePath(path), path))
		}
		fatalKind(kindFile, "sql_open %s: %s", path, fileProblem(err))
	}
	return it.track(&object.Database{Name: path, Conn: sqliteConn{db}})
}

func asDatabaseArg(fn string, obj object.Object) sqlConn {
	d, ok := obj.(*object.Database)
	if !ok {
		fatalf("'%s' needs a database (from sql_open), got %s", fn, obj.Type())
	}
	if d.Closed {
		fatalKind(kindSQL, "%s: database %s is closed", fn, d.Name)
	}
	return d.Conn.(sqlConn)
}

// sqlParams turns the optional list of ? values into SQL values.
func sqlParams(fn string, rest []object.Object) []any {
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
	out := make([]any, len(elems))
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
			out[i] = sqlDateText(v)
		default:
			fatalf("'%s' value %d for a ? placeholder can't be a %s (use integers, floats, text, booleans, none or dates)", fn, i+1, e.Type())
		}
	}
	return out
}

// rowMaps turns query rows into a list of maps, one per row. A column
// name used twice (SELECT * over a join) gets :1, :2 ... after it, as
// SQLite names them in a subquery.
func rowMaps(cols []string, rows [][]any) *object.List {
	names := make([]string, len(cols))
	seen := map[string]bool{}
	for i, c := range cols {
		n := c
		for k := 1; seen[strings.ToLower(n)]; k++ {
			n = c + ":" + itoa(k)
		}
		seen[strings.ToLower(n)] = true
		names[i] = n
	}
	out := &object.List{}
	for _, r := range rows {
		m := object.NewMap()
		for i, n := range names {
			m.Put(&object.String{Value: n}, fromSQL(r[i]))
		}
		out.Elements = append(out.Elements, m)
	}
	return out
}

func fromSQL(v any) object.Object {
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
	case bool:
		return &object.Boolean{Value: x}
	case time.Time:
		return &object.Date{Time: x}
	}
	return object.NoneValue
}

// sqlFileRows runs stmt once per row of a file, as one statement: if
// any row fails, no row's change is kept. cols picks and orders each
// row's values (nil: all, in order). It returns how many records changed.
func (it *Interpreter) sqlFileRows(fn string, conn sqlConn, stmt string, rows [][]object.Object, cols []int) object.Object {
	vals := make([][]any, len(rows))
	for r, row := range rows {
		pick := cols
		if pick == nil {
			pick = make([]int, len(row))
			for i := range pick {
				pick[i] = i
			}
		}
		for _, i := range pick {
			vals[r] = append(vals[r], fileValueToSQL(fn, row[i]))
		}
	}
	n, err := conn.ExecRows(stmt, vals)
	if err != nil {
		fatalKind(kindSQL, "%s: %v", fn, err)
	}
	return &object.Integer{Value: n}
}

// fileValueToSQL turns a value read from a table file into a SQL value:
// .csv and .tsv cells are text or none; .json values keep their kind,
// and a list or object inside a row is stored as its JSON text.
func fileValueToSQL(fn string, v object.Object) any {
	switch x := v.(type) {
	case *object.None:
		return nil
	case *object.String:
		return x.Value
	case *object.Integer:
		return x.Value
	case *object.Float:
		return x.Value
	case *object.Boolean:
		if x.Value {
			return int64(1)
		}
		return int64(0)
	case *object.Date:
		return sqlDateText(x)
	}
	return string(toJSON(fn, v, ""))
}

// quoteSQLName writes a table or column name for SQL: in double quotes,
// so any name works and none can change the statement.
func quoteSQLName(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}

func quoteSQLNames(conn sqlConn, names []string) string {
	q := make([]string, len(names))
	for i, n := range names {
		q[i] = conn.Quote(n)
	}
	return strings.Join(q, ", ")
}
