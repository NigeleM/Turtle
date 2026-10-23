// Copyright 2017-2026 Nigele McCoy
// Licensed under the Apache License, Version 2.0;
// see LICENSE and NOTICE.

package evaluator

import (
	"bytes"
	"encoding/json"
	"math/rand"
	"strings"
	"testing"
)

// writeJSONString writes exactly what Go's encoder writes with HTML escaping
// off, for every byte, the characters JSON treats specially, text that
// isn't UTF-8, and random text.
func TestWriteJSONStringMatchesEncoder(t *testing.T) {
	want := func(s string) string {
		var b bytes.Buffer
		enc := json.NewEncoder(&b)
		enc.SetEscapeHTML(false)
		_ = enc.Encode(s)
		return strings.TrimSuffix(b.String(), "\n")
	}
	check := func(s string) {
		t.Helper()
		var b bytes.Buffer
		writeJSONString(&b, s)
		if got, w := b.String(), want(s); got != w {
			t.Fatalf("%q: got %s, want %s", s, got, w)
		}
	}
	for c := 0; c < 256; c++ {
		check(string([]byte{byte(c)}))
		check("a" + string([]byte{byte(c)}) + "z")
	}
	for _, s := range []string{
		"", "plain", `say "hi"`, `back\slash`, "<b>&amp;</b>", "tab\there", "line\nbreak\r\n",
		"  ", "x y", "é ü ß 中文 🐢", "�", "\xff\xfe", "ok\xc3", "\xe2\x80", "\xed\xa0\x80",
		"\x00\x01\x1f\x7f", "\b\f", strings.Repeat("a", 10000),
	} {
		check(s)
	}
	r := rand.New(rand.NewSource(1))
	pieces := []string{"a", "Z", "0", " ", `"`, `\`, "\n", "\t", "\x00", "\x1f", "\x7f", "é", "€", "🐢", " ", "\xff", "\xc3", "<", "&"}
	for range 200000 {
		var sb strings.Builder
		for range r.Intn(12) {
			sb.WriteString(pieces[r.Intn(len(pieces))])
		}
		check(sb.String())
	}
}
