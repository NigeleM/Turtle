// Copyright 2017-2026 Nigele McCoy
// Licensed under the Apache License, Version 2.0;
// see LICENSE and NOTICE.

package toml

import (
	"math"
	"strings"
	"testing"
	"time"
)

const spec = `# This is a TOML document
title = "TOML Example"
"quoted key" = 'C:\Users\nodejs'
site."google.com" = true

[owner]
name = "Tom Preston-Werner"
dob = 1979-05-27T07:32:00-08:00

[database]
enabled = true
ports = [ 8000, 8001, 8002 ]
data = [ ["delta", "phi"], [3.14] ]
temp_targets = { cpu = 79.5, case = 72.0 }

[servers]

[servers.alpha]
ip = "10.0.0.1"
role = "frontend"

[[products]]
name = "Hammer"
sku = 738594937

[[products]]  # empty table within the array

[[products]]
name = "Nail"
sku = 284758393
color = "gray"
`

func TestParseSpec(t *testing.T) {
	tb, err := Parse(spec)
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(tb.Keys, ","); got != "title,quoted key,site,owner,database,servers,products" {
		t.Errorf("keys in order: %s", got)
	}
	if tb.Values["quoted key"] != `C:\Users\nodejs` {
		t.Errorf("literal string: %v", tb.Values["quoted key"])
	}
	if tb.Values["site"].(*Table).Values["google.com"] != true {
		t.Error("quoted part of a dotted key")
	}
	dob := tb.Values["owner"].(*Table).Values["dob"].(DateTime)
	if dob.Kind != OffsetDateTime || !dob.Time.Equal(time.Date(1979, 5, 27, 15, 32, 0, 0, time.UTC)) {
		t.Errorf("dob: %v", dob)
	}
	db := tb.Values["database"].(*Table)
	if ports := db.Values["ports"].([]any); len(ports) != 3 || ports[1] != int64(8001) {
		t.Errorf("ports: %v", ports)
	}
	if db.Values["temp_targets"].(*Table).Values["case"] != 72.0 {
		t.Error("inline table")
	}
	products := tb.Values["products"].([]any)
	if len(products) != 3 || len(products[1].(*Table).Keys) != 0 || products[2].(*Table).Values["color"] != "gray" {
		t.Errorf("array of tables: %v", products)
	}
	if tb.Values["servers"].(*Table).Values["alpha"].(*Table).Values["role"] != "frontend" {
		t.Error("nested table")
	}
}

func TestParseValues(t *testing.T) {
	cases := []struct {
		src  string
		want any
	}{
		{`v = "tab\there \u00e9 \U0001F600 \"q\""`, "tab\there é 😀 \"q\""},
		{"v = \"\"\"\nRoses are red\nViolets are blue\"\"\"", "Roses are red\nViolets are blue"},
		{"v = \"\"\"\nThe quick \\\n\n   brown fox.\"\"\"", "The quick brown fox."},
		{`v = """Here are two quotation marks: "". Simple enough."""`, `Here are two quotation marks: "". Simple enough.`},
		{`v = """"This," she said, "is just a pointless statement.""""`, `"This," she said, "is just a pointless statement."`},
		{"v = '''\nThe first newline is\ntrimmed in raw strings.'''", "The first newline is\ntrimmed in raw strings."},
		{`v = +99`, int64(99)},
		{`v = -17`, int64(-17)},
		{`v = 1_000_000`, int64(1000000)},
		{`v = 0xDEAD_beef`, int64(0xdeadbeef)},
		{`v = 0o755`, int64(0o755)},
		{`v = 0b1101`, int64(13)},
		{`v = 6.626e-34`, 6.626e-34},
		{`v = -0.01`, -0.01},
		{`v = 224_617.445_991`, 224617.445991},
		{`v = 5e+22`, 5e+22},
		{`v = -inf`, math.Inf(-1)},
		{`v = false`, false},
	}
	for _, c := range cases {
		tb, err := Parse(c.src)
		if err != nil {
			t.Errorf("%s: %v", c.src, err)
			continue
		}
		if got := tb.Values["v"]; got != c.want {
			t.Errorf("%s: got %#v, want %#v", c.src, got, c.want)
		}
	}
	tb, err := Parse("a = nan\nb = 1979-05-27\nc = 07:32:00\nd = 1979-05-27 07:32:00\ne = 1979-05-27T00:32:00.999999-07:00")
	if err != nil {
		t.Fatal(err)
	}
	if f := tb.Values["a"].(float64); !math.IsNaN(f) {
		t.Error("nan")
	}
	kinds := map[string]DateTimeKind{"b": LocalDate, "c": LocalTime, "d": LocalDateTime, "e": OffsetDateTime}
	for k, want := range kinds {
		if dt := tb.Values[k].(DateTime); dt.Kind != want {
			t.Errorf("%s: kind %v, want %v", k, dt.Kind, want)
		}
	}
}

func TestParseErrors(t *testing.T) {
	cases := []struct{ src, want string }{
		{"a = 1\na = 2", "line 2: a is already set"},
		{"[t]\nx = 1\n[t]\ny = 2", "line 3: table [t] is defined twice"},
		{"a.b.c = 1\n[a.b]", "is defined twice"},
		{"a.b = 1\n[a.b]", "a.b is already a value"},
		{"a = { x = 1 }\n[a]", "inline table"},
		{"a = { x = 1 }\na.y = 2", "a is a table defined elsewhere"},
		{"a = [1]\n[[a]]", "already a table or a value"},
		{"x = hello", `"hello" isn't a value: text needs quotes`},
		{`x = "open`, "isn't closed on its line"},
		{"x = 1 2", "unexpected '2' after a value"},
		{"x =", "x = needs a value"},
		{"x = 012", `"012" isn't a value`},
		{"x = 1__0", `"1__0" isn't a value`},
		{"x = 2026-02-30", "isn't a date or time TOML knows"},
		{`x = "\q"`, `\q isn't an escape`},
		{"x = { a = 1,\n b = 2 }", "goes on one line"},
		{"[a", "a [table] header ends with ]"},
		{"= 1", "expected a key"},
	}
	for _, c := range cases {
		_, err := Parse(c.src)
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%q: got %v, want %q", c.src, err, c.want)
		}
	}
}

func TestWriteRoundTrip(t *testing.T) {
	tb, err := Parse(spec)
	if err != nil {
		t.Fatal(err)
	}
	text, err := Write(tb)
	if err != nil {
		t.Fatal(err)
	}
	again, err := Parse(text)
	if err != nil {
		t.Fatalf("written TOML doesn't parse: %v\n%s", err, text)
	}
	text2, _ := Write(again)
	if text != text2 {
		t.Errorf("not stable:\n%s\n---\n%s", text, text2)
	}
	for _, want := range []string{`title = "TOML Example"`, `"quoted key" = "C:\\Users\\nodejs"`, "[owner]", "dob = 1979-05-27T07:32:00-08:00", "[[products]]", "[database.temp_targets]\ncpu = 79.5\ncase = 72.0", "[servers.alpha]"} {
		if !strings.Contains(text, want) {
			t.Errorf("missing %q in:\n%s", want, text)
		}
	}
	if _, err := Write(&Table{Keys: []string{"x"}, Values: map[string]any{"x": nil}}); err == nil || !strings.Contains(err.Error(), "TOML has no none") {
		t.Errorf("none: %v", err)
	}
}
