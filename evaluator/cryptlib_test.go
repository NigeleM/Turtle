// Copyright 2017-2026 Nigele McCoy
// Licensed under the Apache License, Version 2.0;
// see LICENSE and NOTICE.

package evaluator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The crypt library: results checked against known values (Python's
// hashlib and hmac give the same), and every misuse an error.

func TestCrypt(t *testing.T) {
	cases := []struct{ src, want string }{
		{`show hash["hello"] .`, "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824"},
		{`show hash["hello", "md5"] .`, "5d41402abc4b2a76b9719d911017c592"},
		{`show hash["hello", "SHA1"] .`, "aaf4c61ddcc5e8a2dabede0f3b482cd9aea9434d"},
		{`h = "hello" hash "sha512"` + "\nshow h at len .", "128"},
		{`show hmac["abc", "key"] .`, "9c196e32dc0175f86f4b1cb89289d6619de6bee699e4c378e68309ed97a1a6ab"},
		{`s = "abc" hmac "key"` + "\nshow s .", "9c196e32dc0175f86f4b1cb89289d6619de6bee699e4c378e68309ed97a1a6ab"},
		{`show encode["hi there"] .`, "aGkgdGhlcmU="},
		{`show decode["aGkgdGhlcmU="] .`, "hi there"},
		{`show encode["??>", "base64url"] .`, "Pz8-"},
		{`show decode["Pz8-", "base64url"], decode["Pz8-", "base64url"] .`, "??>??>"},
		{`show encode["hi", "hex"], decode["6869", "HEX"] .`, "6869hi"},
		{`id = uuid[]` + "\nshow id at len, id at get[14], id at get[8] .", "364-"},
		{`show token[] at len, " ", token[5] at len .`, "32 5"},
		{`s = passwordhash["pw"]` + "\nshow passwordcheck[\"pw\", s], passwordcheck[\"pW\", s], s at slice[0, 13] .", "truefalsepbkdf2-sha256"},
		{`b = encrypt["meet at 9", "phrase"]` + "\nshow decrypt[b, \"phrase\"], \" \", b at slice[0, 8] .", "meet at 9 turtle1:"},
	}
	for _, c := range cases {
		got, err := run(t, "import crypt\n"+c.src, "")
		if err != nil {
			t.Errorf("%s: %v", c.src, err)
		} else if strings.TrimSpace(got) != c.want {
			t.Errorf("%s:\n got  %q\n want %q", c.src, strings.TrimSpace(got), c.want)
		}
	}
	// Random each time.
	got, _ := run(t, "import crypt\nshow uuid[] == uuid[], token[] == token[], passwordhash[\"a\"] == passwordhash[\"a\"], encrypt[\"a\", \"k\"] == encrypt[\"a\", \"k\"] .", "")
	if strings.TrimSpace(got) != "falsefalsefalsefalse" {
		t.Errorf("not random: %s", got)
	}
}

func TestCryptMisuse(t *testing.T) {
	cases := []struct{ src, kind, want string }{
		{`hash[]`, "type", "hash takes text, and optionally an algorithm"},
		{`hash[5]`, "type", "hash: what to hash must be text, got integer"},
		{`hash["a", "sha999"]`, "crypt", `"sha999" isn't an algorithm`},
		{`hmac["a"]`, "type", "hmac takes text, a secret key"},
		{`encode["a", "rot13"]`, "crypt", `"rot13" isn't a way to encode`},
		{`decode["%%%"]`, "crypt", "decode: that isn't base64 text"},
		{`decode["zz", "hex"]`, "crypt", "decode: that isn't hex text"},
		{`uuid[1]`, "type", "uuid takes nothing"},
		{`token[0]`, "crypt", "the length must be 1 to 4096"},
		{`token["8"]`, "type", "the length must be a whole number"},
		{`passwordcheck["a", "plain"]`, "crypt", "isn't something passwordhash made"},
		{`decrypt["hello", "k"]`, "crypt", "isn't something encrypt made"},
		{`decrypt[encrypt["a", "k"], "x"]`, "crypt", "wrong passphrase"},
		{`encrypt["a", ""]`, "crypt", "the passphrase is empty"},
		{`filehash["nope.txt"]`, "file", "filehash nope.txt"},
	}
	for _, c := range cases {
		src := "import crypt\nsafe\n    x = " + c.src + "\nhandle [] e .\n    show kind of e, \"|\", message of e .\nsafe [end]"
		got, err := run(t, src, "")
		if err != nil {
			t.Errorf("%s: %v", c.src, err)
			continue
		}
		if !strings.HasPrefix(got, c.kind+"|") || !strings.Contains(got, c.want) {
			t.Errorf("%s: got %q, want kind %s and %q", c.src, got, c.kind, c.want)
		}
	}
}

func TestFilehash(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := runIn(t, dir, "import crypt\nshow filehash[\"a.txt\"] == hash[\"hello\"], filehash[\"a.txt\", \"md5\"] .", "")
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(got) != "true5d41402abc4b2a76b9719d911017c592" {
		t.Errorf("got %q", got)
	}
}
