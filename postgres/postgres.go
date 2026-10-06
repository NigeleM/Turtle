// Package postgres talks to a PostgreSQL server, written from scratch on
// the standard library: the frontend/backend protocol (version 3), TLS,
// and SCRAM-SHA-256, MD5 and cleartext logins. It runs statements with ?
// placeholders (turned into $1, $2, ...) and gives rows back as Go values:
// int64, float64, string, bool, []byte, time.Time, or nil for NULL.
//
//	db, err := postgres.Open("postgres://ann:secret@localhost:5432/shop?sslmode=prefer")
//	cols, rows, err := db.Query("SELECT title FROM books WHERE price < ?", []any{1000})
//	n, err := db.Exec("UPDATE books SET price = ? WHERE sku = ?", []any{900, "B1"})
package postgres

import (
	"bufio"
	"crypto/tls"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Error is a problem the server reported, or a connection problem.
type Error struct {
	Msg      string
	Code     string // SQLSTATE, like 23505 for a unique violation
	Severity string
	Detail   string
	Hint     string
}

func (e *Error) Error() string {
	msg := e.Msg
	if e.Detail != "" {
		msg += " (" + e.Detail + ")"
	}
	if e.Hint != "" {
		msg += "; " + e.Hint
	}
	return msg
}

func errorf(format string, args ...any) error {
	return &Error{Msg: fmt.Sprintf(format, args...)}
}

// DB is one connection to a server.
type DB struct {
	conn     net.Conn
	r        *bufio.Reader
	w        []byte // the message being built
	status   byte   // from ReadyForQuery: I idle, T in a transaction, E failed transaction
	params   map[string]string
	closed   bool
	name     string
	nextStmt int
}

// Open connects to postgres://user:password@host:port/database. Query
// parameters: sslmode (disable, prefer (the default), require,
// verify-full), connect_timeout (seconds, default 10),
// application_name.
func Open(dsn string) (*DB, error) {
	u, err := url.Parse(dsn)
	if err != nil || (u.Scheme != "postgres" && u.Scheme != "postgresql") {
		return nil, errorf("not a PostgreSQL address: use postgres://user:password@host:port/database")
	}
	user := u.User.Username()
	password, _ := u.User.Password()
	host := u.Hostname()
	if host == "" {
		host = "localhost"
	}
	port := u.Port()
	if port == "" {
		port = "5432"
	}
	database := strings.TrimPrefix(u.Path, "/")
	if database == "" {
		database = user
	}
	q := u.Query()
	sslmode := q.Get("sslmode")
	if sslmode == "" {
		sslmode = "prefer"
	}
	timeout := 10 * time.Second
	if t := q.Get("connect_timeout"); t != "" {
		if n, err := strconv.Atoi(t); err == nil && n > 0 {
			timeout = time.Duration(n) * time.Second
		}
	}
	appName := q.Get("application_name")
	if appName == "" {
		appName = "turtle"
	}
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(host, port), timeout)
	if err != nil {
		return nil, errorf("can't connect to PostgreSQL at %s:%s: %v (is the server running?)", host, port, err)
	}
	conn.SetDeadline(time.Now().Add(timeout))
	db := &DB{conn: conn, params: map[string]string{}, name: host + "/" + database}
	if sslmode != "disable" {
		if err := db.startTLS(host, sslmode); err != nil {
			conn.Close()
			return nil, err
		}
	}
	db.r = bufio.NewReaderSize(db.conn, 64*1024)
	if err := db.startup(user, password, database, appName); err != nil {
		db.conn.Close()
		return nil, err
	}
	db.conn.SetDeadline(time.Time{})
	return db, nil
}

func (db *DB) startTLS(host, mode string) error {
	var req [8]byte
	binary.BigEndian.PutUint32(req[0:], 8)
	binary.BigEndian.PutUint32(req[4:], 80877103)
	if _, err := db.conn.Write(req[:]); err != nil {
		return errorf("PostgreSQL: %v", err)
	}
	var answer [1]byte
	if _, err := io.ReadFull(db.conn, answer[:]); err != nil {
		return errorf("PostgreSQL: %v", err)
	}
	if answer[0] != 'S' {
		if mode == "prefer" {
			return nil // the server has no TLS; carry on without
		}
		return errorf("PostgreSQL server doesn't offer TLS (sslmode=%s); use sslmode=disable for a local server", mode)
	}
	cfg := &tls.Config{ServerName: host, InsecureSkipVerify: mode != "verify-full"}
	tc := tls.Client(db.conn, cfg)
	if err := tc.Handshake(); err != nil {
		return errorf("PostgreSQL TLS: %v", err)
	}
	db.conn = tc
	return nil
}

// ---- messages ----

func (db *DB) begin(t byte) {
	db.w = db.w[:0]
	if t != 0 {
		db.w = append(db.w, t)
	}
	db.w = append(db.w, 0, 0, 0, 0)
}

func (db *DB) str(s string) { db.w = append(db.w, s...); db.w = append(db.w, 0) }
func (db *DB) int32(n int32) {
	db.w = binary.BigEndian.AppendUint32(db.w, uint32(n))
}
func (db *DB) int16(n int16) {
	db.w = binary.BigEndian.AppendUint16(db.w, uint16(n))
}

// end fills in the length and appends the message to out.
func (db *DB) end(out []byte) []byte {
	start := 0
	if len(db.w) > 0 && db.w[0] != 0 && len(db.w) >= 5 {
		start = 1
	}
	binary.BigEndian.PutUint32(db.w[start:], uint32(len(db.w)-start))
	return append(out, db.w...)
}

func (db *DB) send(msg []byte) error {
	if _, err := db.conn.Write(msg); err != nil {
		db.closed = true
		return errorf("PostgreSQL connection lost: %v", err)
	}
	return nil
}

// recv reads one message: its type and body.
func (db *DB) recv() (byte, []byte, error) {
	var h [5]byte
	if _, err := io.ReadFull(db.r, h[:]); err != nil {
		db.closed = true
		return 0, nil, errorf("PostgreSQL connection lost: %v", err)
	}
	n := int(binary.BigEndian.Uint32(h[1:])) - 4
	if n < 0 || n > 1<<30 {
		db.closed = true
		return 0, nil, errorf("PostgreSQL: bad message length")
	}
	body := make([]byte, n)
	if _, err := io.ReadFull(db.r, body); err != nil {
		db.closed = true
		return 0, nil, errorf("PostgreSQL connection lost: %v", err)
	}
	return h[0], body, nil
}

func serverError(body []byte) *Error {
	e := &Error{}
	for len(body) > 1 {
		code := body[0]
		end := 1
		for end < len(body) && body[end] != 0 {
			end++
		}
		v := string(body[1:end])
		switch code {
		case 'S':
			e.Severity = v
		case 'C':
			e.Code = v
		case 'M':
			e.Msg = v
		case 'D':
			e.Detail = v
		case 'H':
			e.Hint = v
		}
		body = body[end+1:]
	}
	return e
}

// ---- starting up ----

func (db *DB) startup(user, password, database, appName string) error {
	db.begin(0)
	db.int32(196608) // protocol 3.0
	for _, kv := range [][2]string{{"user", user}, {"database", database}, {"application_name", appName},
		{"client_encoding", "UTF8"}, {"DateStyle", "ISO, MDY"}} {
		db.str(kv[0])
		db.str(kv[1])
	}
	db.w = append(db.w, 0)
	binary.BigEndian.PutUint32(db.w, uint32(len(db.w)))
	if err := db.send(db.w); err != nil {
		return err
	}
	var scram *scramState
	for {
		t, body, err := db.recv()
		if err != nil {
			return err
		}
		switch t {
		case 'R':
			kind := binary.BigEndian.Uint32(body)
			switch kind {
			case 0: // ok
			case 3: // cleartext
				db.begin('p')
				db.str(password)
				if err := db.send(db.end(nil)); err != nil {
					return err
				}
			case 5: // md5
				db.begin('p')
				db.str(md5Password(user, password, body[4:8]))
				if err := db.send(db.end(nil)); err != nil {
					return err
				}
			case 10: // SASL
				mechs := strings.Split(strings.TrimRight(string(body[4:]), "\x00"), "\x00")
				ok := false
				for _, m := range mechs {
					if m == "SCRAM-SHA-256" {
						ok = true
					}
				}
				if !ok {
					return errorf("PostgreSQL wants a login method Turtle doesn't know (%s)", strings.Join(mechs, ", "))
				}
				scram = newScram(password)
				first := scram.clientFirst()
				db.begin('p')
				db.str("SCRAM-SHA-256")
				db.int32(int32(len(first)))
				db.w = append(db.w, first...)
				if err := db.send(db.end(nil)); err != nil {
					return err
				}
			case 11: // SASL continue
				final, err := scram.clientFinal(string(body[4:]))
				if err != nil {
					return err
				}
				db.begin('p')
				db.w = append(db.w, final...)
				if err := db.send(db.end(nil)); err != nil {
					return err
				}
			case 12: // SASL final
				if err := scram.verify(string(body[4:])); err != nil {
					return err
				}
			default:
				return errorf("PostgreSQL wants a login method Turtle doesn't know (code %d)", kind)
			}
		case 'S':
			parts := strings.SplitN(string(body), "\x00", 3)
			if len(parts) >= 2 {
				db.params[parts[0]] = parts[1]
			}
		case 'K', 'N':
		case 'Z':
			db.status = body[0]
			return nil
		case 'E':
			e := serverError(body)
			if e.Code == "28P01" || e.Code == "28000" {
				return errorf("PostgreSQL login failed for %q: %s", user, e.Msg)
			}
			return e
		default:
			return errorf("PostgreSQL: unexpected message %q while connecting", t)
		}
	}
}

// Close ends the connection.
func (db *DB) Close() error {
	if db.closed {
		return nil
	}
	db.closed = true
	db.begin('X')
	db.send(db.end(nil))
	return db.conn.Close()
}

// InTransaction reports whether a transaction is open (after BEGIN).
func (db *DB) InTransaction() bool { return db.status == 'T' || db.status == 'E' }

// ---- running statements ----

// result is what one statement gave: columns, rows, how many rows changed.
type result struct {
	cols    []string
	types   []uint32
	rows    [][]any
	changes int64
}

func (db *DB) check() error {
	if db.closed {
		return errorf("the PostgreSQL connection is closed")
	}
	return nil
}

// run sends one statement with parameters (extended protocol) and reads
// its result.
func (db *DB) run(sql string, params []any) (*result, error) {
	if err := db.check(); err != nil {
		return nil, err
	}
	text, n, err := convertPlaceholders(sql)
	if err != nil {
		return nil, err
	}
	if n != len(params) {
		return nil, errorf("the statement has %d ? placeholder(s) but %d value(s) were given", n, len(params))
	}
	var out []byte
	db.begin('P')
	db.str("")
	db.str(text)
	db.int16(0)
	out = db.end(out)
	db.begin('B')
	db.str("")
	db.str("")
	db.int16(0) // all parameters as text
	db.int16(int16(len(params)))
	for _, p := range params {
		v, isNull, err := encodeParam(p)
		if err != nil {
			return nil, err
		}
		if isNull {
			db.int32(-1)
			continue
		}
		db.int32(int32(len(v)))
		db.w = append(db.w, v...)
	}
	db.int16(0) // all results as text
	out = db.end(out)
	db.begin('D')
	db.w = append(db.w, 'P')
	db.str("")
	out = db.end(out)
	db.begin('E')
	db.str("")
	db.int32(0)
	out = db.end(out)
	db.begin('S')
	out = db.end(out)
	if err := db.send(out); err != nil {
		return nil, err
	}
	return db.readResults()
}

// simple runs SQL with the simple protocol, which allows several
// statements; it returns the last statement's result.
func (db *DB) simple(sql string) (*result, error) {
	if err := db.check(); err != nil {
		return nil, err
	}
	db.begin('Q')
	db.str(sql)
	if err := db.send(db.end(nil)); err != nil {
		return nil, err
	}
	return db.readResults()
}

func (db *DB) readResults() (*result, error) {
	res := &result{}
	var firstErr error
	for {
		t, body, err := db.recv()
		if err != nil {
			return nil, err
		}
		switch t {
		case '1', '2', 'n', 'I', 's', 't':
		case 'T':
			res = &result{changes: res.changes}
			count := int(binary.BigEndian.Uint16(body))
			pos := 2
			for i := 0; i < count; i++ {
				end := pos
				for body[end] != 0 {
					end++
				}
				res.cols = append(res.cols, string(body[pos:end]))
				pos = end + 1
				res.types = append(res.types, binary.BigEndian.Uint32(body[pos+6:]))
				pos += 18
			}
		case 'D':
			count := int(binary.BigEndian.Uint16(body))
			pos := 2
			row := make([]any, count)
			for i := 0; i < count; i++ {
				n := int32(binary.BigEndian.Uint32(body[pos:]))
				pos += 4
				if n < 0 {
					continue
				}
				var typ uint32
				if i < len(res.types) {
					typ = res.types[i]
				}
				v, err := decodeValue(typ, body[pos:pos+int(n)])
				if err != nil && firstErr == nil {
					firstErr = err
				}
				row[i] = v
				pos += int(n)
			}
			res.rows = append(res.rows, row)
		case 'C':
			res.changes = commandChanges(strings.TrimRight(string(body), "\x00"))
		case 'E':
			if firstErr == nil {
				firstErr = serverError(body)
			}
		case 'N':
		case 'S':
			parts := strings.SplitN(string(body), "\x00", 3)
			if len(parts) >= 2 {
				db.params[parts[0]] = parts[1]
			}
		case 'A':
		case 'G', 'H':
			// COPY isn't used; refuse it politely.
			if firstErr == nil {
				firstErr = errorf("COPY isn't supported through sql_run; use sql_load for files")
			}
			db.begin('f')
			db.str("COPY isn't supported")
			db.send(db.end(nil))
		case 'Z':
			db.status = body[0]
			if firstErr != nil {
				return nil, firstErr
			}
			return res, nil
		default:
			return nil, errorf("PostgreSQL: unexpected message %q", t)
		}
	}
}

// commandChanges reads how many rows a command changed from its tag:
// INSERT 0 5, UPDATE 3, DELETE 2, MERGE 1.
func commandChanges(tag string) int64 {
	f := strings.Fields(tag)
	if len(f) < 2 {
		return 0
	}
	switch f[0] {
	case "INSERT", "UPDATE", "DELETE", "MERGE", "COPY":
		n, _ := strconv.ParseInt(f[len(f)-1], 10, 64)
		return n
	}
	return 0
}

// Query runs a statement that gives rows.
func (db *DB) Query(sql string, params []any) ([]string, [][]any, error) {
	res, err := db.run(sql, params)
	if err != nil {
		return nil, nil, err
	}
	return res.cols, res.rows, nil
}

// Exec runs a statement (or, with no values, several separated by ;)
// and reports how many rows it changed.
func (db *DB) Exec(sql string, params []any) (int64, error) {
	var res *result
	var err error
	if len(params) == 0 {
		res, err = db.simple(sql)
	} else {
		res, err = db.run(sql, params)
	}
	if err != nil {
		return 0, err
	}
	return res.changes, nil
}

// ExecRows runs one statement once per row of values, all or nothing:
// inside its own transaction, or a savepoint when one is open.
func (db *DB) ExecRows(sql string, rows [][]any) (int64, error) {
	if err := db.check(); err != nil {
		return 0, err
	}
	if db.status == 'E' {
		return 0, errorf("the transaction has already failed; ROLLBACK first")
	}
	inTx := db.InTransaction()
	open, commit, undo := "BEGIN", "COMMIT", "ROLLBACK"
	if inTx {
		open, commit, undo = "SAVEPOINT turtle_rows", "RELEASE SAVEPOINT turtle_rows", "ROLLBACK TO SAVEPOINT turtle_rows; RELEASE SAVEPOINT turtle_rows"
	}
	if _, err := db.simple(open); err != nil {
		return 0, err
	}
	var total int64
	for _, row := range rows {
		res, err := db.run(sql, row)
		if err != nil {
			db.simple(undo)
			return 0, err
		}
		total += res.changes
	}
	if _, err := db.simple(commit); err != nil {
		db.simple(undo)
		return 0, err
	}
	return total, nil
}

// Tables lists the tables in the current schema.
func (db *DB) Tables() ([]string, error) {
	_, rows, err := db.Query("SELECT table_name FROM information_schema.tables WHERE table_schema = current_schema() AND table_type = 'BASE TABLE' ORDER BY table_name", nil)
	if err != nil {
		return nil, err
	}
	out := make([]string, len(rows))
	for i, r := range rows {
		out[i], _ = r[0].(string)
	}
	return out, nil
}

// QuoteIdent writes a table or column name for SQL.
func QuoteIdent(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}

// Name is the server and database, for messages.
func (db *DB) Name() string { return db.name }

var errClosed = errors.New("closed")
