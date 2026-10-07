package evaluator

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/md5"
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"hash"
	"io"
	"os"
	"strconv"
	"strings"

	"Turtle/object"
)

// The "crypt" builtin module: hashes, signatures, encodings, random ids
// and tokens, passwords, and encryption, all on Go's standard library.
//
//	h = hash["hello"]                         // sha256, as hex
//	h = "hello" hash "sha512"                 // the sentence form
//	sig = hmac[body, secret]                  // a signature, as hex
//	b = encode["hi", "base64"]                // "aGk=" (and "base64url", "hex")
//	s = decode[b, "base64"]
//	id = uuid[]                               // a random (version 4) UUID
//	t = token[32]                             // 32 random letters and digits
//	stored = passwordhash["s3cret"]           // salted PBKDF2, to keep
//	ok = passwordcheck["s3cret", stored]
//	box = encrypt["the text", passphrase]     // AES-256-GCM, as text
//	text = decrypt[box, passphrase]
//
// A bad input (text that isn't base64, a wrong passphrase, an unknown
// algorithm) is an error of kind crypt.

// hashNames are the algorithms hash, filehash and hmac take.
var hashNames = map[string]func() hash.Hash{
	"sha256": sha256.New,
	"sha512": sha512.New,
	"sha1":   sha1.New,
	"md5":    md5.New,
}

func hashAlgorithm(fn string, args []object.Object, at int) (string, func() hash.Hash) {
	name := "sha256"
	if len(args) > at {
		name = strings.ToLower(asStringArg(fn, args[at]))
	}
	h, ok := hashNames[name]
	if !ok {
		fatalKind(kindCrypt, "%s: %q isn't an algorithm; use sha256 (the default), sha512, sha1 or md5", fn, name)
	}
	return name, h
}

// argCount checks a call has between min and max arguments.
func argCount(fn string, args []object.Object, min, max int, what string) {
	if len(args) < min || len(args) > max {
		fatalKind(kindType, "%s takes %s, got %d argument(s)", fn, what, len(args))
	}
}

// textArg is an argument that must be text.
func textArg(fn string, args []object.Object, i int, what string) string {
	s, ok := args[i].(*object.String)
	if !ok {
		fatalKind(kindType, "%s: %s must be text, got %s", fn, what, typeName(args[i]))
	}
	return s.Value
}

const (
	passwordIterations = 600000 // OWASP's 2023 advice for PBKDF2-SHA256
	encryptIterations  = 200000
	tokenLetters       = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
	maxTokenLength     = 4096
)

func (it *Interpreter) callCrypt(name string, args []object.Object) object.Object {
	switch name {
	case "hash":
		argCount(name, args, 1, 2, "text, and optionally an algorithm (sha256, sha512, sha1, md5)")
		text := textArg(name, args, 0, "what to hash")
		_, newHash := hashAlgorithm(name, args, 1)
		h := newHash()
		h.Write([]byte(text))
		return &object.String{Value: hex.EncodeToString(h.Sum(nil))}

	case "filehash":
		argCount(name, args, 1, 2, "a file path, and optionally an algorithm (sha256, sha512, sha1, md5)")
		path := textArg(name, args, 0, "the file path")
		_, newHash := hashAlgorithm(name, args, 1)
		f, err := os.Open(it.resolvePath(path))
		if err != nil {
			fatalKind(kindFile, "filehash %s: %s", path, fileProblem(err))
		}
		defer f.Close()
		h := newHash()
		if _, err := io.Copy(h, f); err != nil {
			fatalKind(kindFile, "filehash %s: %s", path, fileProblem(err))
		}
		return &object.String{Value: hex.EncodeToString(h.Sum(nil))}

	case "hmac":
		argCount(name, args, 2, 3, "text, a secret key, and optionally an algorithm (sha256, sha512, sha1, md5)")
		text := textArg(name, args, 0, "what to sign")
		key := textArg(name, args, 1, "the secret key")
		_, newHash := hashAlgorithm(name, args, 2)
		m := hmac.New(newHash, []byte(key))
		m.Write([]byte(text))
		return &object.String{Value: hex.EncodeToString(m.Sum(nil))}

	case "encode", "decode":
		argCount(name, args, 1, 2, `text, and optionally how: "base64" (the default), "base64url" or "hex"`)
		text := textArg(name, args, 0, "the text")
		how := "base64"
		if len(args) == 2 {
			how = strings.ToLower(textArg(name, args, 1, "how"))
		}
		var enc *base64.Encoding
		switch how {
		case "base64":
			enc = base64.StdEncoding
		case "base64url":
			enc = base64.RawURLEncoding
		case "hex":
		default:
			fatalKind(kindCrypt, `%s: %q isn't a way to %s; use "base64", "base64url" or "hex"`, name, how, name)
		}
		if name == "encode" {
			if enc == nil {
				return &object.String{Value: hex.EncodeToString([]byte(text))}
			}
			return &object.String{Value: enc.EncodeToString([]byte(text))}
		}
		var out []byte
		var err error
		if enc == nil {
			out, err = hex.DecodeString(strings.TrimSpace(text))
		} else {
			out, err = enc.DecodeString(strings.TrimSpace(text))
			if err != nil && how == "base64url" { // padded base64url decodes too
				out, err = base64.URLEncoding.DecodeString(strings.TrimSpace(text))
			}
		}
		if err != nil {
			fatalKind(kindCrypt, "decode: that isn't %s text", how)
		}
		return &object.String{Value: string(out)}

	case "uuid":
		argCount(name, args, 0, 0, "nothing")
		var b [16]byte
		rand.Read(b[:])
		b[6] = b[6]&0x0f | 0x40 // version 4
		b[8] = b[8]&0x3f | 0x80 // the RFC 4122 variant
		return &object.String{Value: fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])}

	case "token":
		argCount(name, args, 0, 1, "nothing, or a length (32 if left out)")
		n := int64(32)
		if len(args) == 1 {
			v, ok := args[0].(*object.Integer)
			if !ok {
				fatalKind(kindType, "token: the length must be a whole number, got %s", typeName(args[0]))
			}
			n = v.Value
		}
		if n < 1 || n > maxTokenLength {
			fatalKind(kindCrypt, "token: the length must be 1 to %d, got %d", maxTokenLength, n)
		}
		return &object.String{Value: randomLetters(int(n))}

	case "passwordhash":
		argCount(name, args, 1, 1, "a password")
		pw := textArg(name, args, 0, "the password")
		salt := make([]byte, 16)
		rand.Read(salt)
		key, err := pbkdf2.Key(sha256.New, pw, salt, passwordIterations, 32)
		if err != nil {
			fatalKind(kindCrypt, "passwordhash: %v", err)
		}
		return &object.String{Value: fmt.Sprintf("pbkdf2-sha256$%d$%s$%s", passwordIterations,
			base64.RawStdEncoding.EncodeToString(salt), base64.RawStdEncoding.EncodeToString(key))}

	case "passwordcheck":
		argCount(name, args, 2, 2, "a password and what passwordhash gave for it")
		pw := textArg(name, args, 0, "the password")
		stored := textArg(name, args, 1, "the stored hash")
		parts := strings.Split(stored, "$")
		if len(parts) != 4 || parts[0] != "pbkdf2-sha256" {
			fatalKind(kindCrypt, "passwordcheck: the second argument isn't something passwordhash made")
		}
		iter, err1 := strconv.Atoi(parts[1])
		salt, err2 := base64.RawStdEncoding.DecodeString(parts[2])
		want, err3 := base64.RawStdEncoding.DecodeString(parts[3])
		if err1 != nil || err2 != nil || err3 != nil || iter < 1 || len(want) == 0 {
			fatalKind(kindCrypt, "passwordcheck: the second argument isn't something passwordhash made")
		}
		got, err := pbkdf2.Key(sha256.New, pw, salt, iter, len(want))
		return &object.Boolean{Value: err == nil && subtle.ConstantTimeCompare(got, want) == 1}

	case "encrypt":
		argCount(name, args, 2, 2, "text and a passphrase")
		text := textArg(name, args, 0, "what to encrypt")
		pass := textArg(name, args, 1, "the passphrase")
		if pass == "" {
			fatalKind(kindCrypt, "encrypt: the passphrase is empty")
		}
		salt := make([]byte, 16)
		rand.Read(salt)
		gcm := encryptionKey(pass, salt)
		nonce := make([]byte, gcm.NonceSize())
		rand.Read(nonce)
		sealed := gcm.Seal(nil, nonce, []byte(text), nil)
		box := append(append(append([]byte{1}, salt...), nonce...), sealed...) // 1: this format
		return &object.String{Value: "turtle1:" + base64.RawURLEncoding.EncodeToString(box)}

	case "decrypt":
		argCount(name, args, 2, 2, "what encrypt gave, and the passphrase")
		boxText := textArg(name, args, 0, "what encrypt gave")
		pass := textArg(name, args, 1, "the passphrase")
		raw, err := base64.RawURLEncoding.DecodeString(strings.TrimPrefix(strings.TrimSpace(boxText), "turtle1:"))
		if err != nil || !strings.HasPrefix(strings.TrimSpace(boxText), "turtle1:") || len(raw) < 1+16+12+16 || raw[0] != 1 {
			fatalKind(kindCrypt, "decrypt: that isn't something encrypt made")
		}
		salt := raw[1:17]
		gcm := encryptionKey(pass, salt)
		nonce := raw[17 : 17+gcm.NonceSize()]
		plain, err := gcm.Open(nil, nonce, raw[17+gcm.NonceSize():], nil)
		if err != nil {
			fatalKind(kindCrypt, "decrypt: wrong passphrase, or the text was changed")
		}
		return &object.String{Value: string(plain)}
	}
	fatalKind(kindName, "crypt has no function %q", name)
	return nil
}

// encryptionKey turns a passphrase and salt into AES-256-GCM.
func encryptionKey(pass string, salt []byte) cipher.AEAD {
	key, err := pbkdf2.Key(sha256.New, pass, salt, encryptIterations, 32)
	if err != nil {
		fatalKind(kindCrypt, "encrypt: %v", err)
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		fatalKind(kindCrypt, "encrypt: %v", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		fatalKind(kindCrypt, "encrypt: %v", err)
	}
	return gcm
}

// randomLetters is n letters and digits from the system's secure random
// source, each equally likely.
func randomLetters(n int) string {
	out := make([]byte, n)
	buf := make([]byte, 1)
	limit := byte(256 - 256%len(tokenLetters)) // reject bytes past it: no bias
	for i := 0; i < n; {
		rand.Read(buf)
		if buf[0] < limit {
			out[i] = tokenLetters[int(buf[0])%len(tokenLetters)]
			i++
		}
	}
	return string(out)
}
