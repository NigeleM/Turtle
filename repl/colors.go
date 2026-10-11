// Copyright 2017-2026 Nigele McCoy
// Licensed under the Apache License, Version 2.0;
// see LICENSE and NOTICE.

package repl

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"Turtle/syntax"
)

// The REPL's color schemes: the VS Code extension's twelve, color for
// color (a test reads editors/vscode/extension.js to keep them the same).
// A scheme colors eight groups, each with a shade for dark terminals, one
// for light, and a style: b bold, i italic, u underline.

// ---- schemes ----

const (
	gKw   = iota // keywords
	gImp         // imports
	gFn          // functions
	gDs          // data structures
	gShow        // show and warn
	gStr         // text
	gNum         // numbers
	gThy         // theories
)

type shade struct{ dark, light, style string }

type scheme struct {
	name   string // as VS Code names it
	groups [8]shade
}

var schemes = []scheme{
	{"Turtle", [8]shade{{"#4eba65", "#2c7a39", ""}, {"#4ec9b0", "#0b8577", ""}, {"#6cb6ff", "#1f63c7", ""}, {"#f0a04b", "#a85800", ""}, {"#c9a7ff", "#6a3fc0", "b"}, {"#e5865f", "#b5432a", ""}, {"#f27db8", "#b4307d", ""}, {"#ffd24d", "#956e00", "bi"}}},
	{"Classic", [8]shade{{"#569cd6", "#216197", ""}, {"#c586c0", "#7e3a78", ""}, {"#dcdcaa", "#797930", ""}, {"#4ec9b0", "#258370", ""}, {"#9cdcfe", "#0078b8", "b"}, {"#ce9178", "#89492f", ""}, {"#b5cea8", "#537741", ""}, {"#d7ba7d", "#8f6e29", "bi"}}},
	{"Ocean", [8]shade{{"#4fc1e9", "#117ca2", ""}, {"#3fd1c0", "#1c8276", ""}, {"#8c9cff", "#001ab8", ""}, {"#ffb86c", "#b35c00", ""}, {"#a6e3ff", "#007bb3", "b"}, {"#f4a988", "#ac3d0c", ""}, {"#ff8fb3", "#b8003b", ""}, {"#ffd866", "#946e00", "bi"}}},
	{"Sunset", [8]shade{{"#ff9e64", "#b84500", ""}, {"#e0af68", "#9b671c", ""}, {"#7aa2f7", "#063db1", ""}, {"#f7768e", "#b20626", ""}, {"#bb9af7", "#440aae", "b"}, {"#9ece6a", "#568027", ""}, {"#ff9eaf", "#b80020", ""}, {"#ffd75f", "#946f00", "bi"}}},
	{"Forest", [8]shade{{"#8fbf6a", "#578235", ""}, {"#8fbcbb", "#447473", ""}, {"#88c0d0", "#327386", ""}, {"#d08770", "#8d422b", ""}, {"#b48ead", "#6f4868", "b"}, {"#e0a5a0", "#8d322b", ""}, {"#d9707a", "#94242f", ""}, {"#ebcb8b", "#976c17", "bi"}}},
	{"Soft", [8]shade{{"#93b88a", "#4d7344", ""}, {"#8ab5b0", "#46726c", ""}, {"#8fa8c8", "#3b587d", ""}, {"#c9a27e", "#835a34", ""}, {"#b9a3c9", "#604375", "b"}, {"#c99a8c", "#7f4939", ""}, {"#c49aab", "#754257", ""}, {"#d6c08a", "#8a6f2d", "bi"}}},
	{"Dusk", [8]shade{{"#a7b77a", "#69783f", ""}, {"#87a9a0", "#4c6c63", ""}, {"#9aa6c9", "#3f4d79", ""}, {"#c7a06c", "#876231", ""}, {"#c4a7b9", "#6f4960", "b"}, {"#bf9277", "#7f5339", ""}, {"#b98f9c", "#724653", ""}, {"#d4b46a", "#917127", "bi"}}},
	{"Okabe-Ito", [8]shade{{"#009e73", "#00714f", ""}, {"#56b4e9", "#1f6f9e", ""}, {"#2f9be0", "#0060a0", ""}, {"#e69f00", "#8a5c00", ""}, {"#cc79a7", "#9c3d73", "b"}, {"#df6300", "#a64800", ""}, {"#f0e442", "#6e6600", ""}, {"#ffffff", "#000000", "biu"}}},
	{"Blue & Orange", [8]shade{{"#4da3ff", "#0059b8", "b"}, {"#82c4ff", "#0061b8", "i"}, {"#c4e0ff", "#0057b8", ""}, {"#ff9933", "#b85c00", ""}, {"#ffd166", "#996b00", "bu"}, {"#ffb380", "#b84a00", "i"}, {"#e6e6e6", "#5c5c5c", ""}, {"#ffe066", "#8f7200", "biu"}}},
	{"Teal & Rose", [8]shade{{"#2ec4b6", "#0f7f74", "b"}, {"#8be9e0", "#0d6e66", "i"}, {"#7fd8d0", "#14736b", ""}, {"#ff6b6b", "#c0392b", ""}, {"#ffffff", "#000000", "bu"}, {"#ff9e9e", "#b03a3a", "i"}, {"#ff4fa3", "#b3196a", ""}, {"#ff7a7a", "#a3201f", "biu"}}},
	{"High Contrast", [8]shade{{"#8cff66", "#1a6600", "b"}, {"#5cf0d8", "#076455", ""}, {"#7fc8ff", "#005a9e", ""}, {"#ffb84d", "#804c00", ""}, {"#e0b8ff", "#6700b8", "b"}, {"#ff9f80", "#a42800", ""}, {"#ff8fd0", "#ae0065", ""}, {"#ffe14d", "#665500", "biu"}}},
	{"No Color", [8]shade{{"#ffffff", "#000000", "b"}, {"#9a9a9a", "#6b6b6b", "i"}, {"#d9d9d9", "#2b2b2b", "u"}, {"#ffffff", "#000000", "bi"}, {"#ffffff", "#000000", "bu"}, {"#b3b3b3", "#5c5c5c", "i"}, {"#c6c6c6", "#454545", ""}, {"#ffffff", "#000000", "biu"}}},
}

const defaultScheme = "turtle"

// schemeKey is how a scheme is typed: lowercase, a dash for spaces and
// "&" (Blue & Orange is blue-orange), so no Shift key.
func schemeKey(name string) string {
	return strings.Join(strings.Fields(strings.ToLower(strings.ReplaceAll(name, "&", " "))), "-")
}

func findScheme(key string) (scheme, bool) {
	for _, s := range schemes {
		if schemeKey(s.name) == key {
			return s, true
		}
	}
	return scheme{}, false
}

func schemeKeys() []string {
	var keys []string
	for _, s := range schemes {
		keys = append(keys, schemeKey(s.name))
	}
	return keys
}

// ---- terminal codes ----

// colorCode is the terminal code for a color and style: the exact color
// on terminals with 24-bit color, else the nearest of the 256.
func colorCode(hex, style string, truecolor bool) string {
	var b strings.Builder
	b.WriteString("\x1b[")
	for _, c := range []struct {
		letter, code string
	}{{"b", "1;"}, {"i", "3;"}, {"u", "4;"}} {
		if strings.Contains(style, c.letter) {
			b.WriteString(c.code)
		}
	}
	r, g, bl := rgb(hex)
	if truecolor {
		fmt.Fprintf(&b, "38;2;%d;%d;%dm", r, g, bl)
	} else {
		fmt.Fprintf(&b, "38;5;%dm", nearest256(r, g, bl))
	}
	return b.String()
}

func rgb(hex string) (int, int, int) {
	v, _ := strconv.ParseUint(strings.TrimPrefix(hex, "#"), 16, 32)
	return int(v >> 16 & 0xff), int(v >> 8 & 0xff), int(v & 0xff)
}

// nearest256 is the 256-color palette's closest color: from its 6×6×6
// cube (16-231) or its grays (232-255).
func nearest256(r, g, b int) int {
	levels := []int{0, 95, 135, 175, 215, 255}
	best, bestDist := 16, -1
	try := func(n, cr, cg, cb int) {
		d := (r-cr)*(r-cr) + (g-cg)*(g-cg) + (b-cb)*(b-cb)
		if bestDist < 0 || d < bestDist {
			best, bestDist = n, d
		}
	}
	for i, cr := range levels {
		for j, cg := range levels {
			for k, cb := range levels {
				try(16+36*i+6*j+k, cr, cg, cb)
			}
		}
	}
	for i := 0; i < 24; i++ {
		v := 8 + 10*i
		try(232+i, v, v, v)
	}
	return best
}

// palette is what each kind of code gets in a scheme, and the color of a
// shown value (the scheme's function color).
func palette(s scheme, light, truecolor bool) (map[syntax.Class]string, string) {
	code := func(group int, extra string) string {
		sh := s.groups[group]
		hex := sh.dark
		if light {
			hex = sh.light
		}
		return colorCode(hex, sh.style+extra, truecolor)
	}
	codes := map[syntax.Class]string{
		syntax.Keyword:        code(gKw, ""),
		syntax.Import:         code(gImp, ""),
		syntax.Definition:     code(gFn, "b"),
		syntax.TestDefinition: code(gFn, "bi"),
		syntax.Call:           code(gFn, ""),
		syntax.Builtin:        code(gFn, ""),
		syntax.Method:         code(gFn, ""),
		syntax.Data:           code(gDs, ""),
		syntax.DataDefinition: code(gDs, "b"),
		syntax.Show:           code(gShow, ""),
		syntax.String:         code(gStr, ""),
		syntax.Constant:       code(gNum, ""),
		syntax.Theory:         code(gThy, ""),
		syntax.Comment:        "\x1b[3;38;5;244m", // italic gray, in every scheme
	}
	return codes, code(gFn, "")
}

// hasTrueColor reports whether the terminal shows 24-bit colors.
func hasTrueColor() bool {
	ct := strings.ToLower(os.Getenv("COLORTERM"))
	return ct == "truecolor" || ct == "24bit" || os.Getenv("WT_SESSION") != ""
}

// ---- light or dark ----

// lightFromColorFGBG reads COLORFGBG ("15;0": white on black), which some
// terminals set. ok is false without it.
func lightFromColorFGBG(v string) (light, ok bool) {
	parts := strings.Split(v, ";")
	bg, err := strconv.Atoi(parts[len(parts)-1])
	if v == "" || err != nil {
		return false, false
	}
	return bg == 7 || bg >= 9, true
}

// lightFromReply reads a terminal's answer to "what's your background"
// (OSC 11: "\x1b]11;rgb:ffff/ffff/ffff\x1b\\"). ok is false without one.
func lightFromReply(reply string) (light, ok bool) {
	i := strings.Index(reply, "rgb:")
	if i < 0 {
		return false, false
	}
	rest := reply[i+4:]
	var vals [3]float64
	for n := 0; n < 3; n++ {
		j := 0
		for j < len(rest) && j < 4 && strings.IndexByte("0123456789abcdefABCDEF", rest[j]) >= 0 {
			j++
		}
		if j == 0 {
			return false, false
		}
		v, _ := strconv.ParseUint(rest[:j], 16, 32)
		vals[n] = float64(v) / float64(uint64(1)<<(4*j)-1)
		rest = rest[j:]
		if n < 2 {
			if !strings.HasPrefix(rest, "/") {
				return false, false
			}
			rest = rest[1:]
		}
	}
	return 0.2126*vals[0]+0.7152*vals[1]+0.0722*vals[2] > 0.5, true
}

// The terminal's answers: its background (OSC 11), and what it is (DA1).
var (
	oscReply = regexp.MustCompile(`\x1b\]11;[^\x07\x1b]*(\x07|\x1b\\)`)
	da1Reply = regexp.MustCompile(`\x1b\[\?[0-9;]*c`)
)

// withoutReplies is what was read while asking the terminal, without its
// answers: keys typed (or pasted) meanwhile.
func withoutReplies(b []byte) []byte {
	return da1Reply.ReplaceAll(oscReply.ReplaceAll(b, nil), nil)
}

// ---- the setting ----

// settingsPath is where the REPL keeps your scheme: ~/.config/turtle/repl,
// one line, "okabe-ito auto".
func settingsPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".config", "turtle", "repl")
}

// readSetting reads a saved "scheme shade" line; the defaults where it's
// missing or wrong.
func readSetting(path string) (key, shadeName string) {
	key, shadeName = defaultScheme, "auto"
	if path == "" {
		return
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	f := strings.Fields(string(data))
	if len(f) > 0 {
		if _, ok := findScheme(f[0]); ok {
			key = f[0]
		}
	}
	if len(f) > 1 && (f[1] == "light" || f[1] == "dark") {
		shadeName = f[1]
	}
	return
}

func writeSetting(path, key, shadeName string) error {
	if path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(key+" "+shadeName+"\n"), 0o644)
}

// ---- the session's colors ----

// setColors makes a scheme and shade (auto, light, dark) the session's.
func (s *Session) setColors(key, shadeName string) {
	sc, _ := findScheme(key)
	s.scheme, s.shade = key, shadeName
	s.light = shadeName == "light"
	if shadeName == "auto" && s.detectLight != nil {
		if !s.detected {
			s.detectedLight, s.detected = s.detectLight(), true
		}
		s.light = s.detectedLight
	}
	s.codes, s.resultColor = palette(sc, s.light, s.truecolor)
}

// colorsCommand is the colors command: with nothing, the picker
// (picker.go); with a scheme and/or light, dark or auto, a change (kept for
// next time).
func (s *Session) colorsCommand(arg string) {
	if arg == "" {
		if !s.pickColors() {
			s.listColors()
		}
		return
	}
	key, shadeName := s.scheme, s.shade
	for _, w := range strings.Fields(strings.ToLower(arg)) {
		if w == "light" || w == "dark" || w == "auto" {
			shadeName = w
		} else if _, ok := findScheme(w); ok {
			key = w
		} else {
			fmt.Fprintln(s.out, s.paint(errorColor, "colors: there's no scheme "+w+"; colors lists them"))
			return
		}
	}
	s.chooseColors(key, shadeName)
}

// chooseColors makes a scheme and shade the session's, and keeps them for
// next time.
func (s *Session) chooseColors(key, shadeName string) {
	s.setColors(key, shadeName)
	if err := writeSetting(s.settings, key, shadeName); err != nil {
		fmt.Fprintln(s.out, s.paint(errorColor, "colors: couldn't keep it for next time: "+err.Error()))
	}
	msg := "colors: " + key + ", " + s.shadeText()
	if !s.color {
		msg += " (colors are off here: NO_COLOR, or not a color terminal)"
	}
	fmt.Fprintln(s.out, s.paint(s.resultColor, msg))
}

// shadeText is the shade in use: "light", or "dark (auto)" when found out.
func (s *Session) shadeText() string {
	t := "dark"
	if s.light {
		t = "light"
	}
	if s.shade == "auto" {
		t += " (auto)"
	}
	return t
}
