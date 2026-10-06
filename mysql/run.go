package mysql

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/x509"
	"encoding/binary"
	"encoding/pem"
	"math"
	"strconv"
	"strings"
	"time"
)

// ---- logins ----

func (db *DB) scramble(plugin, password string, nonce []byte) ([]byte, error) {
	if password == "" {
		return nil, nil
	}
	switch plugin {
	case "sha256_password":
		// Over TLS the password itself; otherwise ask for the server's
		// public key (the answer comes in authResult).
		if db.tlsOn {
			return append([]byte(password), 0), nil
		}
		return []byte{1}, nil
	case "mysql_native_password":
		// SHA1(pw) XOR SHA1(nonce + SHA1(SHA1(pw)))
		h1 := sha1.Sum([]byte(password))
		h2 := sha1.Sum(h1[:])
		s := sha1.New()
		s.Write(nonce[:20])
		s.Write(h2[:])
		h3 := s.Sum(nil)
		out := make([]byte, 20)
		for i := range out {
			out[i] = h1[i] ^ h3[i]
		}
		return out, nil
	case "caching_sha2_password":
		// SHA256(pw) XOR SHA256(SHA256(SHA256(pw)) + nonce)
		h1 := sha256.Sum256([]byte(password))
		h2 := sha256.Sum256(h1[:])
		s := sha256.New()
		s.Write(h2[:])
		s.Write(nonce[:20])
		h3 := s.Sum(nil)
		out := make([]byte, 32)
		for i := range out {
			out[i] = h1[i] ^ h3[i]
		}
		return out, nil
	case "mysql_clear_password":
		return append([]byte(password), 0), nil
	}
	return nil, errorf("MySQL wants a login method Turtle doesn't know (%s)", plugin)
}

// rsaPassword encrypts the password for caching_sha2_password's full
// login without TLS: (password + NUL) XOR nonce, RSA-OAEP.
func rsaPassword(password string, nonce, pemKey []byte) ([]byte, error) {
	block, _ := pem.Decode(pemKey)
	if block == nil {
		return nil, errorf("MySQL: the server's public key can't be read")
	}
	key, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, errorf("MySQL: the server's public key can't be read: %v", err)
	}
	pub, ok := key.(*rsa.PublicKey)
	if !ok {
		return nil, errorf("MySQL: the server's key isn't RSA")
	}
	plain := append([]byte(password), 0)
	for i := range plain {
		plain[i] ^= nonce[i%20]
	}
	return rsa.EncryptOAEP(sha1.New(), rand.Reader, pub, plain, nil)
}

// ---- results ----

// column is one column of a result.
type column struct {
	name    string
	typ     byte
	flags   uint16
	charset uint16
}

const (
	tDecimal    = 0
	tTiny       = 1
	tShort      = 2
	tLong       = 3
	tFloat      = 4
	tDouble     = 5
	tNull       = 6
	tTimestamp  = 7
	tLongLong   = 8
	tInt24      = 9
	tDate       = 10
	tTime       = 11
	tDatetime   = 12
	tYear       = 13
	tVarchar    = 15
	tBit        = 16
	tJSON       = 245
	tNewDecimal = 246
	tVarString  = 253
	tString     = 254

	flagUnsigned  = 32
	binaryCharset = 63
)

func parseColumn(p []byte) column {
	pos := 0
	var name []byte
	for i := 0; i < 6; i++ {
		s, n := lenencStr(p[pos:])
		if i == 4 {
			name = s
		}
		pos += n
	}
	_, n := lenenc(p[pos:])
	pos += n
	c := column{name: string(name)}
	c.charset = binary.LittleEndian.Uint16(p[pos:])
	c.typ = p[pos+6]
	c.flags = binary.LittleEndian.Uint16(p[pos+7:])
	return c
}

// readColumns reads n column definitions and the EOF after them.
func (db *DB) readColumns(n int) ([]column, error) {
	cols := make([]column, n)
	for i := range cols {
		p, err := db.readPacket()
		if err != nil {
			return nil, err
		}
		cols[i] = parseColumn(p)
	}
	if _, err := db.readPacket(); err != nil { // EOF
		return nil, err
	}
	return cols, nil
}

func isEOF(p []byte) bool { return len(p) < 9 && len(p) > 0 && p[0] == 0xfe }

// textValue converts a text-protocol value by its column type.
func textValue(c column, s []byte) any {
	str := string(s)
	switch c.typ {
	case tTiny, tShort, tLong, tLongLong, tInt24, tYear:
		if n, err := strconv.ParseInt(str, 10, 64); err == nil {
			return n
		}
		if u, err := strconv.ParseUint(str, 10, 64); err == nil {
			return float64(u)
		}
	case tFloat, tDouble:
		if f, err := strconv.ParseFloat(str, 64); err == nil {
			return f
		}
	case tDecimal, tNewDecimal:
		return decimalValue(str)
	case tDate:
		if t, err := time.Parse("2006-01-02", str); err == nil {
			return t
		}
	case tDatetime, tTimestamp:
		if t, err := time.Parse("2006-01-02 15:04:05.999999", str); err == nil {
			return t
		}
	}
	if c.charset == binaryCharset && c.typ != tJSON {
		return append([]byte{}, s...)
	}
	return str
}

func decimalValue(s string) any {
	if !strings.ContainsAny(s, ".eE") {
		if n, err := strconv.ParseInt(s, 10, 64); err == nil {
			return n
		}
	}
	f, _ := strconv.ParseFloat(s, 64)
	return f
}

// binaryValue reads one binary-protocol value; it returns the value and
// how many bytes it took.
func binaryValue(c column, b []byte) (any, int) {
	unsigned := c.flags&flagUnsigned != 0
	switch c.typ {
	case tTiny:
		if unsigned {
			return int64(b[0]), 1
		}
		return int64(int8(b[0])), 1
	case tShort, tYear:
		v := binary.LittleEndian.Uint16(b)
		if unsigned || c.typ == tYear {
			return int64(v), 2
		}
		return int64(int16(v)), 2
	case tLong, tInt24:
		v := binary.LittleEndian.Uint32(b)
		if unsigned {
			return int64(v), 4
		}
		return int64(int32(v)), 4
	case tLongLong:
		v := binary.LittleEndian.Uint64(b)
		if unsigned && v > math.MaxInt64 {
			return float64(v), 8
		}
		return int64(v), 8
	case tFloat:
		return float64(math.Float32frombits(binary.LittleEndian.Uint32(b))), 4
	case tDouble:
		return math.Float64frombits(binary.LittleEndian.Uint64(b)), 8
	case tDate, tDatetime, tTimestamp:
		n := int(b[0])
		var y, mo, d, h, mi, s, us int
		if n >= 4 {
			y = int(binary.LittleEndian.Uint16(b[1:]))
			mo, d = int(b[3]), int(b[4])
		}
		if n >= 7 {
			h, mi, s = int(b[5]), int(b[6]), int(b[7])
		}
		if n >= 11 {
			us = int(binary.LittleEndian.Uint32(b[8:]))
		}
		if n == 0 || mo == 0 {
			return nil, 1 + n // 0000-00-00
		}
		return time.Date(y, time.Month(mo), d, h, mi, s, us*1000, time.UTC), 1 + n
	case tTime:
		n := int(b[0])
		if n == 0 {
			return "00:00:00", 1
		}
		neg := b[1] == 1
		days := int(binary.LittleEndian.Uint32(b[2:]))
		h, mi, s := int(b[6]), int(b[7]), int(b[8])
		str := strconv.Itoa(days*24+h) + ":" + two(mi) + ":" + two(s)
		if len(str) < 8 {
			str = "0" + str
		}
		if neg {
			str = "-" + str
		}
		return str, 1 + n
	}
	s, n := lenencStr(b)
	switch c.typ {
	case tDecimal, tNewDecimal:
		return decimalValue(string(s)), n
	}
	if c.charset == binaryCharset && c.typ != tJSON && c.typ != tBit {
		return append([]byte{}, s...), n
	}
	return string(s), n
}

func two(n int) string {
	if n < 10 {
		return "0" + strconv.Itoa(n)
	}
	return strconv.Itoa(n)
}

// result is what a command gave.
type result struct {
	cols    []string
	rows    [][]any
	changes int64
}

// readResult reads a text-protocol answer (COM_QUERY), every result set
// of it, keeping the last one's rows.
func (db *DB) readResult() (*result, error) {
	res := &result{}
	for {
		p, err := db.readPacket()
		if err != nil {
			return nil, err
		}
		switch p[0] {
		case 0x00:
			res.changes += db.okPacket(p)
		case 0xff:
			return nil, parseErr(p)
		case 0xfb:
			return nil, errorf("LOAD DATA LOCAL isn't supported; use sql_load")
		default:
			n, _ := lenenc(p)
			cols, err := db.readColumns(int(n))
			if err != nil {
				return nil, err
			}
			res.cols = nil
			for _, c := range cols {
				res.cols = append(res.cols, c.name)
			}
			res.rows = nil
			for {
				p, err := db.readPacket()
				if err != nil {
					return nil, err
				}
				if isEOF(p) {
					db.status = binary.LittleEndian.Uint16(p[3:])
					break
				}
				if p[0] == 0xff {
					return nil, parseErr(p)
				}
				row := make([]any, len(cols))
				pos := 0
				for i, c := range cols {
					if p[pos] == 0xfb {
						pos++
						continue
					}
					s, n := lenencStr(p[pos:])
					row[i] = textValue(c, s)
					pos += n
				}
				res.rows = append(res.rows, row)
			}
		}
		if db.status&8 == 0 { // no more results
			return res, nil
		}
	}
}

// ---- running statements ----

func (db *DB) query(sql string) (*result, error) {
	if err := db.command(append([]byte{0x03}, sql...)); err != nil {
		return nil, err
	}
	return db.readResult()
}

// stmt is a prepared statement.
type stmt struct {
	id      uint32
	nparams int
	cols    []column
}

func (db *DB) prepare(sql string) (*stmt, error) {
	if err := db.command(append([]byte{0x16}, sql...)); err != nil {
		return nil, err
	}
	p, err := db.readPacket()
	if err != nil {
		return nil, err
	}
	if p[0] == 0xff {
		return nil, parseErr(p)
	}
	s := &stmt{id: binary.LittleEndian.Uint32(p[1:])}
	ncols := int(binary.LittleEndian.Uint16(p[5:]))
	s.nparams = int(binary.LittleEndian.Uint16(p[7:]))
	if s.nparams > 0 {
		if _, err := db.readColumns(s.nparams); err != nil {
			return nil, err
		}
	}
	if ncols > 0 {
		if s.cols, err = db.readColumns(ncols); err != nil {
			return nil, err
		}
	}
	return s, nil
}

func (db *DB) closeStmt(s *stmt) {
	db.command(binary.LittleEndian.AppendUint32([]byte{0x19}, s.id))
}

// execute runs a prepared statement with values.
func (db *DB) execute(s *stmt, params []any) (*result, error) {
	if len(params) != s.nparams {
		return nil, errorf("the statement has %d ? placeholder(s) but %d value(s) were given", s.nparams, len(params))
	}
	b := []byte{0x17}
	b = binary.LittleEndian.AppendUint32(b, s.id)
	b = append(b, 0)
	b = binary.LittleEndian.AppendUint32(b, 1)
	if s.nparams > 0 {
		nulls := make([]byte, (s.nparams+7)/8)
		var types, values []byte
		for i, v := range params {
			switch x := v.(type) {
			case nil:
				nulls[i/8] |= 1 << (i % 8)
				types = append(types, tNull, 0)
			case int64:
				types = append(types, tLongLong, 0)
				values = binary.LittleEndian.AppendUint64(values, uint64(x))
			case int:
				types = append(types, tLongLong, 0)
				values = binary.LittleEndian.AppendUint64(values, uint64(int64(x)))
			case float64:
				types = append(types, tDouble, 0)
				values = binary.LittleEndian.AppendUint64(values, math.Float64bits(x))
			case bool:
				types = append(types, tTiny, 0)
				if x {
					values = append(values, 1)
				} else {
					values = append(values, 0)
				}
			case string:
				types = append(types, tVarString, 0)
				values = appendLenenc(values, uint64(len(x)))
				values = append(values, x...)
			case []byte:
				types = append(types, 252, 0)
				values = appendLenenc(values, uint64(len(x)))
				values = append(values, x...)
			case time.Time:
				str := x.Format("2006-01-02 15:04:05.999999")
				types = append(types, tVarString, 0)
				values = appendLenenc(values, uint64(len(str)))
				values = append(values, str...)
			default:
				return nil, errorf("a value of type %T can't be sent to MySQL", v)
			}
		}
		b = append(b, nulls...)
		b = append(b, 1)
		b = append(b, types...)
		b = append(b, values...)
	}
	if err := db.command(b); err != nil {
		return nil, err
	}
	res := &result{}
	for {
		p, err := db.readPacket()
		if err != nil {
			return nil, err
		}
		switch p[0] {
		case 0x00:
			res.changes += db.okPacket(p)
		case 0xff:
			return nil, parseErr(p)
		default:
			n, _ := lenenc(p)
			cols, err := db.readColumns(int(n))
			if err != nil {
				return nil, err
			}
			res.cols = nil
			for _, c := range cols {
				res.cols = append(res.cols, c.name)
			}
			res.rows = nil
			for {
				p, err := db.readPacket()
				if err != nil {
					return nil, err
				}
				if isEOF(p) {
					db.status = binary.LittleEndian.Uint16(p[3:])
					break
				}
				if p[0] == 0xff {
					return nil, parseErr(p)
				}
				row := make([]any, len(cols))
				nullMap := p[1 : 1+(len(cols)+9)/8]
				pos := 1 + len(nullMap)
				for i, c := range cols {
					bit := i + 2
					if nullMap[bit/8]&(1<<(bit%8)) != 0 {
						continue
					}
					v, n := binaryValue(c, p[pos:])
					row[i] = v
					pos += n
				}
				res.rows = append(res.rows, row)
			}
		}
		if db.status&8 == 0 {
			return res, nil
		}
	}
}

// run runs one statement: with values, prepared; without, as text.
func (db *DB) run(sql string, params []any) (*result, error) {
	if len(params) == 0 {
		return db.query(sql)
	}
	s, err := db.prepare(sql)
	if err != nil {
		return nil, err
	}
	defer db.closeStmt(s)
	return db.execute(s, params)
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
	res, err := db.run(sql, params)
	if err != nil {
		return 0, err
	}
	return res.changes, nil
}

// ExecRows runs one statement once per row of values, all or nothing:
// in its own transaction, or a savepoint when one is open. An INSERT
// ... ON DUPLICATE KEY UPDATE row counts once, whether it added a
// record or changed one (MySQL itself says 2 for a change).
func (db *DB) ExecRows(sql string, rows [][]any) (int64, error) {
	upsert := strings.Contains(strings.ToUpper(sql), "ON DUPLICATE KEY UPDATE")
	inTx := db.InTransaction()
	open, commit, undo := "START TRANSACTION", "COMMIT", "ROLLBACK"
	if inTx {
		open, commit, undo = "SAVEPOINT turtle_rows", "RELEASE SAVEPOINT turtle_rows", "ROLLBACK TO SAVEPOINT turtle_rows"
	}
	if _, err := db.query(open); err != nil {
		return 0, err
	}
	s, err := db.prepare(sql)
	if err != nil {
		db.query(undo)
		return 0, err
	}
	defer db.closeStmt(s)
	var total int64
	for _, row := range rows {
		res, err := db.execute(s, row)
		if err != nil {
			db.query(undo)
			if inTx {
				db.query("RELEASE SAVEPOINT turtle_rows")
			}
			return 0, err
		}
		if upsert && res.changes > 1 {
			res.changes = 1
		}
		total += res.changes
	}
	if _, err := db.query(commit); err != nil {
		db.query(undo)
		return 0, err
	}
	return total, nil
}

// Tables lists the tables of the current database.
func (db *DB) Tables() ([]string, error) {
	_, rows, err := db.Query("SELECT table_name FROM information_schema.tables WHERE table_schema = DATABASE() AND table_type = 'BASE TABLE' ORDER BY table_name", nil)
	if err != nil {
		return nil, err
	}
	out := make([]string, len(rows))
	for i, r := range rows {
		switch v := r[0].(type) {
		case string:
			out[i] = v
		case []byte:
			out[i] = string(v)
		}
	}
	return out, nil
}
