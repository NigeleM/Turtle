// Copyright 2017-2026 Nigele McCoy
// Licensed under the Apache License, Version 2.0;
// see LICENSE and NOTICE.

// Package mysql talks to a MySQL (or MariaDB) server, written from
// scratch on the standard library: the client/server protocol, TLS, and
// the caching_sha2_password and mysql_native_password logins. Statements
// with ? values run as server-side prepared statements, so values never
// become part of the SQL text. Rows come back as Go values: int64,
// float64, string, []byte, time.Time, or nil for NULL.
//
//	db, err := mysql.Open("mysql://ann:secret@localhost:3306/shop")
//	cols, rows, err := db.Query("SELECT title FROM books WHERE price < ?", []any{1000})
package mysql

import (
	"bufio"
	"crypto/tls"
	"encoding/binary"
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
	Code     uint16 // MySQL error number, like 1062 for a duplicate key
	SQLState string
}

func (e *Error) Error() string { return e.Msg }

func errorf(format string, args ...any) error {
	return &Error{Msg: fmt.Sprintf(format, args...)}
}

// Capability flags.
const (
	clientLongPassword   = 1
	clientFoundRows      = 2
	clientLongFlag       = 4
	clientConnectWithDB  = 8
	clientProtocol41     = 512
	clientSSL            = 2048
	clientTransactions   = 8192
	clientSecureConn     = 32768
	clientMultiStmts     = 1 << 16
	clientMultiResults   = 1 << 17
	clientPSMultiResults = 1 << 18
	clientPluginAuth     = 1 << 19
	clientLenencAuth     = 1 << 21
)

const statusInTrans = 1

// DB is one connection to a server.
type DB struct {
	conn   net.Conn
	r      *bufio.Reader
	seq    byte
	status uint16
	closed bool
	name   string
	tlsOn  bool
}

// Open connects to mysql://user:password@host:port/database. Query
// parameters: tls (preferred (the default), true, skip-verify, false),
// timeout (seconds to connect, default 10).
func Open(dsn string) (*DB, error) {
	u, err := url.Parse(dsn)
	if err != nil || (u.Scheme != "mysql" && u.Scheme != "mariadb") {
		return nil, errorf("not a MySQL address: use mysql://user:password@host:port/database")
	}
	user := u.User.Username()
	password, _ := u.User.Password()
	host := u.Hostname()
	if host == "" {
		host = "localhost"
	}
	port := u.Port()
	if port == "" {
		port = "3306"
	}
	database := strings.TrimPrefix(u.Path, "/")
	q := u.Query()
	tlsMode := q.Get("tls")
	if tlsMode == "" {
		tlsMode = "preferred"
	}
	timeout := 10 * time.Second
	if t := q.Get("timeout"); t != "" {
		if n, err := strconv.Atoi(t); err == nil && n > 0 {
			timeout = time.Duration(n) * time.Second
		}
	}
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(host, port), timeout)
	if err != nil {
		return nil, errorf("can't connect to MySQL at %s:%s: %v (is the server running?)", host, port, err)
	}
	conn.SetDeadline(time.Now().Add(timeout))
	db := &DB{conn: conn, r: bufio.NewReaderSize(conn, 64*1024), name: host + "/" + database}
	if err := db.handshake(user, password, database, host, tlsMode); err != nil {
		db.conn.Close()
		return nil, err
	}
	db.conn.SetDeadline(time.Time{})
	return db, nil
}

// ---- packets ----

func (db *DB) readPacket() ([]byte, error) {
	var out []byte
	for {
		var h [4]byte
		if _, err := io.ReadFull(db.r, h[:]); err != nil {
			db.closed = true
			return nil, errorf("MySQL connection lost: %v", err)
		}
		n := int(h[0]) | int(h[1])<<8 | int(h[2])<<16
		db.seq = h[3] + 1
		buf := make([]byte, n)
		if _, err := io.ReadFull(db.r, buf); err != nil {
			db.closed = true
			return nil, errorf("MySQL connection lost: %v", err)
		}
		out = append(out, buf...)
		if n < 0xffffff {
			return out, nil
		}
	}
}

func (db *DB) writePacket(payload []byte) error {
	for {
		n := len(payload)
		if n > 0xffffff {
			n = 0xffffff
		}
		h := []byte{byte(n), byte(n >> 8), byte(n >> 16), db.seq}
		db.seq++
		if _, err := db.conn.Write(append(h, payload[:n]...)); err != nil {
			db.closed = true
			return errorf("MySQL connection lost: %v", err)
		}
		payload = payload[n:]
		if n < 0xffffff {
			return nil
		}
	}
}

// command starts a new command (sequence 0).
func (db *DB) command(payload []byte) error {
	if db.closed {
		return errorf("the MySQL connection is closed")
	}
	db.seq = 0
	return db.writePacket(payload)
}

func lenenc(b []byte) (uint64, int) {
	if len(b) == 0 {
		return 0, 0
	}
	switch b[0] {
	case 0xfc:
		return uint64(binary.LittleEndian.Uint16(b[1:])), 3
	case 0xfd:
		return uint64(b[1]) | uint64(b[2])<<8 | uint64(b[3])<<16, 4
	case 0xfe:
		return binary.LittleEndian.Uint64(b[1:]), 9
	}
	return uint64(b[0]), 1
}

func lenencStr(b []byte) ([]byte, int) {
	n, k := lenenc(b)
	return b[k : k+int(n)], k + int(n)
}

func appendLenenc(b []byte, n uint64) []byte {
	switch {
	case n < 251:
		return append(b, byte(n))
	case n < 1<<16:
		return append(b, 0xfc, byte(n), byte(n>>8))
	case n < 1<<24:
		return append(b, 0xfd, byte(n), byte(n>>8), byte(n>>16))
	}
	b = append(b, 0xfe)
	return binary.LittleEndian.AppendUint64(b, n)
}

func parseErr(p []byte) *Error {
	e := &Error{}
	if len(p) >= 3 {
		e.Code = binary.LittleEndian.Uint16(p[1:])
	}
	msg := p[3:]
	if len(msg) > 0 && msg[0] == '#' && len(msg) >= 6 {
		e.SQLState = string(msg[1:6])
		msg = msg[6:]
	}
	e.Msg = string(msg)
	return e
}

// okPacket reads an OK packet: rows changed, last insert id, status.
func (db *DB) okPacket(p []byte) int64 {
	affected, k := lenenc(p[1:])
	_, k2 := lenenc(p[1+k:])
	if 1+k+k2+2 <= len(p) {
		db.status = binary.LittleEndian.Uint16(p[1+k+k2:])
	}
	return int64(affected)
}

// ---- connecting ----

func (db *DB) handshake(user, password, database, host, tlsMode string) error {
	p, err := db.readPacket()
	if err != nil {
		return err
	}
	if p[0] == 0xff {
		return parseErr(p)
	}
	if p[0] != 10 {
		return errorf("MySQL: unknown protocol version %d", p[0])
	}
	pos := 1 + strings.IndexByte(string(p[1:]), 0) + 1
	pos += 4 // connection id
	nonce := append([]byte{}, p[pos:pos+8]...)
	pos += 9
	caps := uint32(binary.LittleEndian.Uint16(p[pos:]))
	pos += 2
	pos += 1 + 2 // charset, status
	caps |= uint32(binary.LittleEndian.Uint16(p[pos:])) << 16
	pos += 2
	authLen := int(p[pos])
	pos += 1 + 10
	plugin := "mysql_native_password"
	if caps&clientSecureConn != 0 {
		n := authLen - 8
		if n < 13 {
			n = 13
		}
		nonce = append(nonce, p[pos:pos+n-1]...)
		pos += n
	}
	if caps&clientPluginAuth != 0 && pos < len(p) {
		end := strings.IndexByte(string(p[pos:]), 0)
		if end < 0 {
			end = len(p) - pos
		}
		plugin = string(p[pos : pos+end])
	}
	flags := uint32(clientLongPassword | clientFoundRows | clientLongFlag | clientProtocol41 | clientTransactions |
		clientSecureConn | clientMultiStmts | clientMultiResults | clientPSMultiResults | clientPluginAuth | clientLenencAuth)
	if database != "" {
		flags |= clientConnectWithDB
	}
	wantTLS := tlsMode != "false"
	if wantTLS && caps&clientSSL == 0 {
		if tlsMode != "preferred" {
			return errorf("MySQL server doesn't offer TLS (tls=%s); use tls=false for a local server", tlsMode)
		}
		wantTLS = false
	}
	if wantTLS {
		flags |= clientSSL
		req := binary.LittleEndian.AppendUint32(nil, flags)
		req = binary.LittleEndian.AppendUint32(req, 16<<20)
		req = append(req, 255)
		req = append(req, make([]byte, 23)...)
		if err := db.writePacket(req); err != nil {
			return err
		}
		cfg := &tls.Config{ServerName: host, InsecureSkipVerify: tlsMode != "true"}
		tc := tls.Client(db.conn, cfg)
		if err := tc.Handshake(); err != nil {
			return errorf("MySQL TLS: %v", err)
		}
		db.conn = tc
		db.r = bufio.NewReaderSize(tc, 64*1024)
		db.tlsOn = true
	}
	auth, err := db.scramble(plugin, password, nonce)
	if err != nil {
		return err
	}
	resp := binary.LittleEndian.AppendUint32(nil, flags)
	resp = binary.LittleEndian.AppendUint32(resp, 16<<20)
	resp = append(resp, 255) // utf8mb4
	resp = append(resp, make([]byte, 23)...)
	resp = append(resp, user...)
	resp = append(resp, 0)
	resp = appendLenenc(resp, uint64(len(auth)))
	resp = append(resp, auth...)
	if database != "" {
		resp = append(resp, database...)
		resp = append(resp, 0)
	}
	resp = append(resp, plugin...)
	resp = append(resp, 0)
	if err := db.writePacket(resp); err != nil {
		return err
	}
	return db.authResult(user, plugin, password, nonce)
}

// authResult follows the server through the login: OK, an error, a
// switch to another method, or caching_sha2_password's extra steps.
func (db *DB) authResult(user, plugin, password string, nonce []byte) error {
	for {
		p, err := db.readPacket()
		if err != nil {
			return err
		}
		switch p[0] {
		case 0x00:
			db.okPacket(p)
			return nil
		case 0xff:
			e := parseErr(p)
			if e.Code == 1045 {
				return errorf("MySQL login failed for %q: %s", user, e.Msg)
			}
			return e
		case 0xfe: // switch method
			rest := p[1:]
			end := strings.IndexByte(string(rest), 0)
			if end < 0 {
				return errorf("MySQL: bad login switch")
			}
			plugin = string(rest[:end])
			nonce = append([]byte{}, rest[end+1:]...)
			if n := len(nonce); n > 0 && nonce[n-1] == 0 {
				nonce = nonce[:n-1]
			}
			auth, err := db.scramble(plugin, password, nonce)
			if err != nil {
				return err
			}
			if err := db.writePacket(auth); err != nil {
				return err
			}
		case 0x01: // more data
			if len(p) < 2 {
				return errorf("MySQL: bad login answer")
			}
			if plugin == "sha256_password" { // the public key asked for
				enc, err := rsaPassword(password, nonce, p[1:])
				if err != nil {
					return err
				}
				if err := db.writePacket(enc); err != nil {
					return err
				}
				continue
			}
			switch p[1] {
			case 3: // fast login worked; OK follows
			case 4: // full login: the password, over TLS or RSA-encrypted
				if db.tlsOn {
					if err := db.writePacket(append([]byte(password), 0)); err != nil {
						return err
					}
					continue
				}
				if err := db.writePacket([]byte{2}); err != nil {
					return err
				}
				keyPkt, err := db.readPacket()
				if err != nil {
					return err
				}
				if keyPkt[0] != 0x01 {
					return errorf("MySQL: expected the server's public key")
				}
				enc, err := rsaPassword(password, nonce, keyPkt[1:])
				if err != nil {
					return err
				}
				if err := db.writePacket(enc); err != nil {
					return err
				}
			default:
				return errorf("MySQL: unexpected login step %d", p[1])
			}
		default:
			return errorf("MySQL: unexpected answer while logging in")
		}
	}
}

// Close ends the connection.
func (db *DB) Close() error {
	if db.closed {
		return nil
	}
	db.command([]byte{0x01})
	db.closed = true
	return db.conn.Close()
}

// InTransaction reports whether a transaction is open.
func (db *DB) InTransaction() bool { return db.status&statusInTrans != 0 }

// Name is the server and database, for messages.
func (db *DB) Name() string { return db.name }

// QuoteIdent writes a table or column name for SQL.
func QuoteIdent(name string) string {
	return "`" + strings.ReplaceAll(name, "`", "``") + "`"
}
