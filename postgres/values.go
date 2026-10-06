package postgres

import (
	"crypto/hmac"
	"crypto/md5"
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"math"
	"strconv"
	"strings"
	"time"
)

// ---- placeholders ----

// convertPlaceholders turns each ? outside quotes and comments into $1,
// $2, ... and counts them.
func convertPlaceholders(sql string) (string, int, error) {
	var sb strings.Builder
	n := 0
	for i := 0; i < len(sql); i++ {
		c := sql[i]
		switch {
		case c == '\'':
			// 'text', with '' inside; E'text' also allows backslash escapes.
			escapes := i > 0 && (sql[i-1] == 'E' || sql[i-1] == 'e')
			j := i + 1
			for j < len(sql) {
				if escapes && sql[j] == '\\' {
					j += 2
					continue
				}
				if sql[j] == '\'' {
					if j+1 < len(sql) && sql[j+1] == '\'' {
						j += 2
						continue
					}
					break
				}
				j++
			}
			if j >= len(sql) {
				return "", 0, errorf("a 'text' value isn't closed")
			}
			sb.WriteString(sql[i : j+1])
			i = j
		case c == '"':
			j := strings.IndexByte(sql[i+1:], '"')
			if j < 0 {
				return "", 0, errorf("a quoted name isn't closed")
			}
			sb.WriteString(sql[i : i+j+2])
			i += j + 1
		case c == '-' && i+1 < len(sql) && sql[i+1] == '-':
			j := strings.IndexByte(sql[i:], '\n')
			if j < 0 {
				j = len(sql) - i
			}
			sb.WriteString(sql[i : i+j])
			i += j - 1
		case c == '/' && i+1 < len(sql) && sql[i+1] == '*':
			depth, j := 1, i+2
			for j < len(sql) && depth > 0 {
				switch {
				case strings.HasPrefix(sql[j:], "/*"):
					depth++
					j += 2
				case strings.HasPrefix(sql[j:], "*/"):
					depth--
					j += 2
				default:
					j++
				}
			}
			sb.WriteString(sql[i:j])
			i = j - 1
		case c == '$':
			// $tag$ ... $tag$ quoting.
			end := strings.IndexByte(sql[i+1:], '$')
			tag := ""
			if end >= 0 {
				tag = sql[i : i+end+2]
			}
			if tag != "" && isTag(tag[1:len(tag)-1]) {
				close := strings.Index(sql[i+len(tag):], tag)
				if close < 0 {
					return "", 0, errorf("a %s quoted text isn't closed", tag)
				}
				stop := i + len(tag) + close + len(tag)
				sb.WriteString(sql[i:stop])
				i = stop - 1
				continue
			}
			sb.WriteByte(c)
		case c == '?':
			n++
			sb.WriteString("$" + strconv.Itoa(n))
		default:
			sb.WriteByte(c)
		}
	}
	return sb.String(), n, nil
}

func isTag(s string) bool {
	for i, c := range s {
		if !(c == '_' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || i > 0 && c >= '0' && c <= '9') {
			return false
		}
	}
	return true
}

// ---- values in and out ----

func encodeParam(v any) ([]byte, bool, error) {
	switch x := v.(type) {
	case nil:
		return nil, true, nil
	case int64:
		return []byte(strconv.FormatInt(x, 10)), false, nil
	case int:
		return []byte(strconv.Itoa(x)), false, nil
	case float64:
		if math.IsInf(x, 1) {
			return []byte("Infinity"), false, nil
		}
		if math.IsInf(x, -1) {
			return []byte("-Infinity"), false, nil
		}
		return []byte(strconv.FormatFloat(x, 'g', -1, 64)), false, nil
	case string:
		return []byte(x), false, nil
	case bool:
		if x {
			return []byte("true"), false, nil
		}
		return []byte("false"), false, nil
	case []byte:
		return []byte(`\x` + hex.EncodeToString(x)), false, nil
	case time.Time:
		return []byte(x.Format("2006-01-02 15:04:05.999999999-07:00")), false, nil
	}
	return nil, false, errorf("a value of type %T can't be sent to PostgreSQL", v)
}

// Type OIDs PostgreSQL gives for result columns.
const (
	oidBool        = 16
	oidBytea       = 17
	oidInt8        = 20
	oidInt2        = 21
	oidInt4        = 23
	oidOID         = 26
	oidFloat4      = 700
	oidFloat8      = 701
	oidNumeric     = 1700
	oidDate        = 1082
	oidTimestamp   = 1114
	oidTimestampTZ = 1184
)

// decodeValue turns a column's text into a Go value by its type.
func decodeValue(typ uint32, b []byte) (any, error) {
	s := string(b)
	switch typ {
	case oidBool:
		return s == "t", nil
	case oidInt2, oidInt4, oidInt8, oidOID:
		return strconv.ParseInt(s, 10, 64)
	case oidFloat4, oidFloat8:
		switch s {
		case "NaN":
			return math.NaN(), nil
		case "Infinity":
			return math.Inf(1), nil
		case "-Infinity":
			return math.Inf(-1), nil
		}
		return strconv.ParseFloat(s, 64)
	case oidNumeric:
		if !strings.ContainsAny(s, ".eEN") {
			if n, err := strconv.ParseInt(s, 10, 64); err == nil {
				return n, nil
			}
		}
		if s == "NaN" {
			return math.NaN(), nil
		}
		return strconv.ParseFloat(s, 64)
	case oidBytea:
		if strings.HasPrefix(s, `\x`) {
			return hex.DecodeString(s[2:])
		}
		return b, nil
	case oidDate:
		if t, err := time.Parse("2006-01-02", s); err == nil {
			return t, nil
		}
		return s, nil // infinity, BC dates
	case oidTimestamp:
		if t, err := time.Parse("2006-01-02 15:04:05.999999999", s); err == nil {
			return t, nil
		}
		return s, nil
	case oidTimestampTZ:
		for _, layout := range []string{"2006-01-02 15:04:05.999999999-07", "2006-01-02 15:04:05.999999999-07:00", "2006-01-02 15:04:05.999999999-07:00:00"} {
			if t, err := time.Parse(layout, s); err == nil {
				return t, nil
			}
		}
		return s, nil
	}
	return s, nil
}

// ---- logins ----

func md5Password(user, password string, salt []byte) string {
	inner := md5.Sum([]byte(password + user))
	h := hex.EncodeToString(inner[:])
	outer := md5.Sum(append([]byte(h), salt...))
	return "md5" + hex.EncodeToString(outer[:])
}

// scramState is a SCRAM-SHA-256 login (RFC 5802, RFC 7677).
type scramState struct {
	password    string
	clientNonce string
	firstBare   string
	authMessage string
	salted      []byte
}

func newScram(password string) *scramState {
	b := make([]byte, 18)
	rand.Read(b)
	return &scramState{password: password, clientNonce: base64.StdEncoding.EncodeToString(b)}
}

func (s *scramState) clientFirst() string {
	s.firstBare = "n=,r=" + s.clientNonce
	return "n,," + s.firstBare
}

func hmacSHA256(key []byte, msg string) []byte {
	m := hmac.New(sha256.New, key)
	m.Write([]byte(msg))
	return m.Sum(nil)
}

func (s *scramState) clientFinal(serverFirst string) (string, error) {
	var nonce, salt string
	iter := 0
	for _, part := range strings.Split(serverFirst, ",") {
		if len(part) < 2 {
			continue
		}
		switch part[:2] {
		case "r=":
			nonce = part[2:]
		case "s=":
			salt = part[2:]
		case "i=":
			iter, _ = strconv.Atoi(part[2:])
		}
	}
	if !strings.HasPrefix(nonce, s.clientNonce) || salt == "" || iter <= 0 {
		return "", errorf("PostgreSQL login: the server's SCRAM answer is malformed")
	}
	saltBytes, err := base64.StdEncoding.DecodeString(salt)
	if err != nil {
		return "", errorf("PostgreSQL login: bad SCRAM salt")
	}
	s.salted, err = pbkdf2.Key(sha256.New, s.password, saltBytes, iter, 32)
	if err != nil {
		return "", errorf("PostgreSQL login: %v", err)
	}
	clientKey := hmacSHA256(s.salted, "Client Key")
	stored := sha256.Sum256(clientKey)
	withoutProof := "c=biws,r=" + nonce
	s.authMessage = s.firstBare + "," + serverFirst + "," + withoutProof
	sig := hmacSHA256(stored[:], s.authMessage)
	proof := make([]byte, len(clientKey))
	for i := range proof {
		proof[i] = clientKey[i] ^ sig[i]
	}
	return withoutProof + ",p=" + base64.StdEncoding.EncodeToString(proof), nil
}

func (s *scramState) verify(serverFinal string) error {
	if strings.HasPrefix(serverFinal, "e=") {
		return errorf("PostgreSQL login failed: %s", serverFinal[2:])
	}
	if !strings.HasPrefix(serverFinal, "v=") {
		return errorf("PostgreSQL login: the server's SCRAM answer is malformed")
	}
	got, err := base64.StdEncoding.DecodeString(serverFinal[2:])
	if err != nil {
		return errorf("PostgreSQL login: bad server signature")
	}
	serverKey := hmacSHA256(s.salted, "Server Key")
	want := hmacSHA256(serverKey, s.authMessage)
	if !hmac.Equal(got, want) {
		return errorf("PostgreSQL login: the server couldn't prove it knows the password; not trusting it")
	}
	return nil
}
