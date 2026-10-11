// Copyright 2017-2026 Nigele McCoy
// Licensed under the Apache License, Version 2.0;
// see LICENSE and NOTICE.

package repl

import (
	"bytes"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"Turtle/syntax"
)

// The schemes are the VS Code extension's, color for color.
func TestSchemesMatchVSCode(t *testing.T) {
	data, err := os.ReadFile("../editors/vscode/extension.js")
	if err != nil {
		t.Fatal(err)
	}
	src := string(data)
	start := strings.Index(src, "const SCHEMES = {")
	end := strings.Index(src[start:], "\n};")
	if start < 0 || end < 0 {
		t.Fatal("no SCHEMES in extension.js")
	}
	block := src[start : start+end]
	nameRe := regexp.MustCompile(`(?m)^  "([^"]+)": \{`)
	groupRe := regexp.MustCompile(`(kw|imp|fn|ds|show|str|num|thy): \["(#[0-9a-fA-F]{6})", "(#[0-9a-fA-F]{6})", "([biu]*)"\]`)
	order := map[string]int{"kw": gKw, "imp": gImp, "fn": gFn, "ds": gDs, "show": gShow, "str": gStr, "num": gNum, "thy": gThy}
	names := nameRe.FindAllStringSubmatchIndex(block, -1)
	if len(names) != len(schemes) {
		t.Fatalf("extension.js has %d schemes, the REPL %d", len(names), len(schemes))
	}
	for i, m := range names {
		name := block[m[2]:m[3]]
		body := block[m[1]:]
		if i+1 < len(names) {
			body = block[m[1]:names[i+1][0]]
		}
		if schemes[i].name != name {
			t.Errorf("scheme %d: extension.js %q, the REPL %q", i, name, schemes[i].name)
			continue
		}
		var got [8]shade
		for _, g := range groupRe.FindAllStringSubmatch(body, -1) {
			got[order[g[1]]] = shade{strings.ToLower(g[2]), strings.ToLower(g[3]), g[4]}
		}
		if got != schemes[i].groups {
			t.Errorf("%s differs:\nextension.js %v\nREPL         %v", name, got, schemes[i].groups)
		}
	}
}

func TestSchemeKeys(t *testing.T) {
	want := "turtle classic ocean sunset forest soft dusk okabe-ito blue-orange teal-rose high-contrast no-color"
	if got := strings.Join(schemeKeys(), " "); got != want {
		t.Errorf("got %s", got)
	}
}

func TestColorCode(t *testing.T) {
	cases := []struct {
		hex, style string
		truecolor  bool
		want       string
	}{
		{"#ffd24d", "bi", true, "\x1b[1;3;38;2;255;210;77m"},
		{"#000000", "biu", true, "\x1b[1;3;4;38;2;0;0;0m"},
		{"#ffffff", "", false, "\x1b[38;5;231m"},
		{"#000000", "", false, "\x1b[38;5;16m"},
		{"#808080", "", false, "\x1b[38;5;244m"},
		{"#ff0000", "b", false, "\x1b[1;38;5;196m"},
	}
	for _, c := range cases {
		if got := colorCode(c.hex, c.style, c.truecolor); got != c.want {
			t.Errorf("colorCode(%s, %s, %v) = %q, want %q", c.hex, c.style, c.truecolor, got, c.want)
		}
	}
}

func TestLightOrDark(t *testing.T) {
	replies := map[string][2]bool{
		"\x1b]11;rgb:ffff/ffff/ffff\x1b\\\x1b[?62;22c": {true, true},
		"\x1b]11;rgb:1e1e/1e1e/1e1e\x07":               {false, true},
		"\x1b]11;rgb:fd/f6/e3\x1b\\":                   {true, true}, // Solarized Light, two digits
		"\x1b[?1;2c":                                   {false, false},
		"":                                             {false, false},
	}
	for r, want := range replies {
		light, ok := lightFromReply(r)
		if [2]bool{light, ok} != want {
			t.Errorf("lightFromReply(%q) = %v %v", r, light, ok)
		}
	}
	fgbg := map[string][2]bool{"15;0": {false, true}, "0;15": {true, true}, "0;default;7": {true, true}, "": {false, false}, "x": {false, false}}
	for v, want := range fgbg {
		light, ok := lightFromColorFGBG(v)
		if [2]bool{light, ok} != want {
			t.Errorf("lightFromColorFGBG(%q) = %v %v", v, light, ok)
		}
	}
}

func TestColorsCommand(t *testing.T) {
	var out bytes.Buffer
	s := NewSession(t.TempDir(), &out, true)
	s.truecolor = true
	s.settings = filepath.Join(t.TempDir(), "turtle", "repl")
	s.detectLight = func() bool { return true }

	s.Eval("colors")
	if list := stripCodes(out.String()); !strings.HasPrefix(list, "turtle, dark (auto)\n") || !strings.Contains(list, "  okabe-ito       if  import time  total[x]  list  show  \"text\"  42  tally\n") {
		t.Errorf("colors: %q", out.String())
	}
	out.Reset()
	s.Eval("colors okabe-ito")
	if !strings.Contains(out.String(), "okabe-ito, light (auto)") {
		t.Errorf("colors okabe-ito: %q", out.String())
	}
	// Okabe-Ito's light keywords are #00714f.
	if s.codes[syntax.Keyword] != "\x1b[38;2;0;113;79m" {
		t.Errorf("keyword code %q", s.codes[syntax.Keyword])
	}
	s.Eval("colors dark")
	if s.scheme != "okabe-ito" || s.light || s.codes[syntax.Keyword] != "\x1b[38;2;0;158;115m" {
		t.Errorf("colors dark: %s %v %q", s.scheme, s.light, s.codes[syntax.Keyword])
	}
	if key, shadeName := readSetting(s.settings); key != "okabe-ito" || shadeName != "dark" {
		t.Errorf("kept %s %s", key, shadeName)
	}
	out.Reset()
	s.Eval("colors purple")
	if !strings.Contains(out.String(), "there's no scheme purple") || s.scheme != "okabe-ito" {
		t.Errorf("colors purple: %q", out.String())
	}
	// A variable named colors is yours, not the command.
	out.Reset()
	s.Eval("colors = 3")
	s.Eval("colors")
	if strings.TrimSpace(stripCodes(out.String())) != "3" {
		t.Errorf("colors variable: %q", out.String())
	}
}

func TestReadSettingDefaults(t *testing.T) {
	dir := t.TempDir()
	if k, sh := readSetting(filepath.Join(dir, "none")); k != "turtle" || sh != "auto" {
		t.Errorf("missing: %s %s", k, sh)
	}
	p := filepath.Join(dir, "repl")
	os.WriteFile(p, []byte("nonsense sideways\n"), 0o644)
	if k, sh := readSetting(p); k != "turtle" || sh != "auto" {
		t.Errorf("wrong: %s %s", k, sh)
	}
}

func stripCodes(s string) string {
	return regexp.MustCompile("\x1b\\[[0-9;]*m").ReplaceAllString(s, "")
}

func TestWithoutReplies(t *testing.T) {
	got := withoutReplies([]byte("ab\x1b]11;rgb:0000/0000/0000\x1b\\c\x1b[?62;22cd\x1b]11;rgb:ff/ff/ff\x07e"))
	if string(got) != "abcde" {
		t.Errorf("got %q", got)
	}
}

func TestPickerLines(t *testing.T) {
	lines := pickerLines(7, 2, "turtle", false, true) // okabe-ito, light
	plain := make([]string, len(lines))
	for i, l := range lines {
		plain[i] = stripCodes(l)
	}
	text := strings.Join(plain, "\n")
	for _, want := range []string{"shade: ← light →", "> okabe-ito ", "  turtle (now) ", "│  def total[orders]", "↑↓ scheme"} {
		if !strings.Contains(text, want) {
			t.Errorf("no %q in\n%s", want, text)
		}
	}
	// The sample is in the cursor's scheme and shade: Okabe-Ito's light
	// keywords (#00714f).
	if !strings.Contains(strings.Join(lines, "\n"), "\x1b[38;2;0;113;79mdef") {
		t.Error("sample not in okabe-ito light")
	}
	// Every line fits the width the picker asks for.
	for _, l := range plain {
		if n := len([]rune(l)); n > pickerWidth() {
			t.Errorf("%d columns, more than %d: %q", n, pickerWidth(), l)
		}
	}
	if got := stripCodes(pickerLines(0, 0, "turtle", true, false)[0]); got != "  shade: ← auto (light) →" {
		t.Errorf("auto shade: %q", got)
	}
}

func TestLoneEscape(t *testing.T) {
	if k := decodeKeys([]byte{27}); len(k) != 1 || k[0].kind != keyEscape {
		t.Errorf("got %v", k)
	}
}
