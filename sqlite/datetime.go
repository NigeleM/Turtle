package sqlite

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

// Date and time functions, following SQLite's date.c: a moment is kept
// as a Julian day number in milliseconds (iJD), proleptic Gregorian
// calendar, UTC unless "localtime" says otherwise.
//
//	date(value, modifiers...)      2026-10-05
//	time(value, modifiers...)      14:05:00
//	datetime(value, modifiers...)  2026-10-05 14:05:00
//	julianday(value, ...)          2461318.08680556
//	unixepoch(value, ...)          1791209100
//	strftime(format, value, ...)   any of the above, and more
//
// value is 'now', text like '2026-10-05', '2026-10-05 14:05[:00[.000]]'
// (T instead of the space, a Z or ±HH:MM zone), 'HH:MM[:SS]', or a
// number (a Julian day, or seconds since 1970 with 'unixepoch' or
// 'auto'). Modifiers: '+3 days' (days, hours, minutes, seconds, months,
// years; - too), '+HH:MM', 'start of month' (year, day), 'weekday 0'..
// 'weekday 6', 'unixepoch', 'julianday', 'auto', 'localtime', 'utc',
// 'subsec'.

type dateTime struct {
	jd                                   int64 // milliseconds
	y, mo, d, h, mi                      int
	s                                    float64
	tz                                   int // minutes
	validJD, validYMD, validHMS, validTZ bool
	rawS                                 bool // a plain number, not yet interpreted
	raw                                  float64
	isError                              bool
	subsec                               bool
}

const unixEpochJD = 210866760000000

// nowFunc is the clock (tests replace it).
var nowFunc = time.Now

func (p *dateTime) computeJD() {
	if p.validJD {
		return
	}
	y, m, d := 2000, 1, 1
	if p.validYMD {
		y, m, d = p.y, p.mo, p.d
	}
	if m <= 2 {
		y--
		m += 12
	}
	a := y / 100
	b := 2 - a + a/4
	x1 := 36525 * (y + 4716) / 100
	x2 := 306001 * (m + 1) / 10000
	p.jd = int64((float64(x1+x2+d+b) - 1524.5) * 86400000)
	p.validJD = true
	if p.validHMS {
		p.jd += int64(p.h)*3600000 + int64(p.mi)*60000 + int64(p.s*1000+0.5)
		if p.validTZ {
			p.jd -= int64(p.tz) * 60000
			p.validYMD, p.validHMS, p.validTZ = false, false, false
		}
	}
}

func (p *dateTime) computeYMD() {
	if p.validYMD {
		return
	}
	if !p.validJD {
		p.y, p.mo, p.d = 2000, 1, 1
	} else if p.jd < 0 || p.jd > 464269060799999 {
		p.isError = true
		return
	} else {
		z := int((p.jd + 43200000) / 86400000)
		a := int((float64(z) - 1867216.25) / 36524.25)
		a = z + 1 + a - a/4
		b := a + 1524
		c := int((float64(b) - 122.1) / 365.25)
		d := (36525 * (c & 32767)) / 100
		e := int(float64(b-d) / 30.6001)
		x1 := int(30.6001 * float64(e))
		p.d = b - d - x1
		if e < 14 {
			p.mo = e - 1
		} else {
			p.mo = e - 13
		}
		if p.mo > 2 {
			p.y = c - 4716
		} else {
			p.y = c - 4715
		}
	}
	p.validYMD = true
}

func (p *dateTime) computeHMS() {
	if p.validHMS {
		return
	}
	p.computeJD()
	dayMs := int((p.jd + 43200000) % 86400000)
	p.s = float64(dayMs%60000) / 1000.0
	dayMin := dayMs / 60000
	p.mi = dayMin % 60
	p.h = dayMin / 60
	p.rawS = false
	p.validHMS = true
}

func (p *dateTime) computeYMDHMS() {
	p.computeYMD()
	p.computeHMS()
}

func (p *dateTime) clearYMDHMS() {
	p.validYMD, p.validHMS, p.validTZ = false, false, false
}

func (p *dateTime) setRawNumber(r float64) {
	p.raw, p.rawS = r, true
	if r >= 0.0 && r < 5373484.5 {
		p.jd = int64(r*86400000.0 + 0.5)
		p.validJD = true
	}
}

// parseDateTime reads a time value.
func parseDateTime(v Value, p *dateTime) bool {
	switch x := v.(type) {
	case nil:
		return false
	case int64:
		p.setRawNumber(float64(x))
		return true
	case float64:
		p.setRawNumber(x)
		return true
	}
	s := strings.TrimSpace(textValue(v))
	if strings.EqualFold(s, "now") {
		p.jd = nowFunc().UnixMilli() + unixEpochJD
		p.validJD = true
		return true
	}
	if parseYMD(s, p) || parseHMS(s, p) {
		return true
	}
	if f, err := strconv.ParseFloat(s, 64); err == nil && looksNumeric(s) {
		p.setRawNumber(f)
		return true
	}
	return false
}

func digits(s string, n int) (int, string, bool) {
	if len(s) < n {
		return 0, s, false
	}
	v := 0
	for i := 0; i < n; i++ {
		if s[i] < '0' || s[i] > '9' {
			return 0, s, false
		}
		v = v*10 + int(s[i]-'0')
	}
	return v, s[n:], true
}

// parseYMD reads YYYY-MM-DD with an optional time after a space or T.
func parseYMD(s string, p *dateTime) bool {
	neg := false
	if strings.HasPrefix(s, "-") {
		neg, s = true, s[1:]
	}
	y, rest, ok := digits(s, 4)
	if !ok || !strings.HasPrefix(rest, "-") {
		return false
	}
	m, rest, ok := digits(rest[1:], 2)
	if !ok || !strings.HasPrefix(rest, "-") {
		return false
	}
	d, rest, ok := digits(rest[1:], 2)
	if !ok || m < 1 || m > 12 || d < 1 || d > 31 {
		return false
	}
	rest = strings.TrimLeft(rest, " ")
	if strings.HasPrefix(rest, "T") {
		rest = rest[1:]
	}
	if rest != "" {
		if !parseHMS(rest, p) {
			return false
		}
	} else {
		p.validHMS = false
	}
	p.validJD = false
	p.validYMD = true
	if neg {
		y = -y
	}
	p.y, p.mo, p.d = y, m, d
	if p.validTZ {
		p.computeJD()
	}
	return true
}

// parseHMS reads HH:MM[:SS[.fff]] with an optional zone.
func parseHMS(s string, p *dateTime) bool {
	h, rest, ok := digits(s, 2)
	if !ok || !strings.HasPrefix(rest, ":") {
		return false
	}
	m, rest, ok := digits(rest[1:], 2)
	if !ok {
		return false
	}
	sec := 0.0
	if strings.HasPrefix(rest, ":") {
		var si int
		si, rest, ok = digits(rest[1:], 2)
		if !ok {
			return false
		}
		sec = float64(si)
		if strings.HasPrefix(rest, ".") && len(rest) > 1 && rest[1] >= '0' && rest[1] <= '9' {
			frac := 0.0
			scale := 1.0
			i := 1
			for i < len(rest) && rest[i] >= '0' && rest[i] <= '9' {
				frac = frac*10 + float64(rest[i]-'0')
				scale *= 10
				i++
			}
			sec += frac / scale
			rest = rest[i:]
		}
	}
	if h > 24 || m > 59 || sec >= 60 {
		return false
	}
	p.validJD = false
	p.rawS = false
	p.validHMS = true
	p.h, p.mi, p.s = h, m, sec
	rest = strings.TrimLeft(rest, " ")
	p.validTZ = false
	p.tz = 0
	switch {
	case rest == "":
	case rest == "Z" || rest == "z":
		p.validTZ = true
	case rest[0] == '+' || rest[0] == '-':
		sign := 1
		if rest[0] == '-' {
			sign = -1
		}
		zh, r2, ok := digits(rest[1:], 2)
		if !ok || !strings.HasPrefix(r2, ":") {
			return false
		}
		zm, r2, ok := digits(r2[1:], 2)
		if !ok || strings.TrimSpace(r2) != "" {
			return false
		}
		p.tz = sign * (zh*60 + zm)
		p.validTZ = true
	default:
		return false
	}
	return true
}

// applyModifier changes p by one modifier; false means a bad modifier.
func applyModifier(z string, p *dateTime, first bool) bool {
	low := strings.ToLower(strings.TrimSpace(z))
	switch {
	case low == "unixepoch":
		if !p.rawS || !first {
			return false
		}
		r := p.raw*1000.0 + 210866760000000.0
		if r < 0 || r >= 464269060800000 {
			return false
		}
		p.jd = int64(r + 0.5)
		p.validJD = true
		p.rawS = false
		p.clearYMDHMS()
		return true
	case low == "julianday":
		if !p.rawS || !first {
			return false
		}
		p.rawS = false
		return p.validJD
	case low == "auto":
		if !p.rawS || !first {
			return false
		}
		if p.raw >= 0 && p.raw < 5373484.5 {
			p.rawS = false
			return true
		}
		return applyModifier("unixepoch", p, true)
	case low == "localtime":
		p.computeJD()
		ms := p.jd - unixEpochJD
		t := time.UnixMilli(ms).In(time.Local)
		_, off := t.Zone()
		p.jd += int64(off) * 1000
		p.clearYMDHMS()
		p.rawS = false
		return true
	case low == "utc":
		p.computeJD()
		ms := p.jd - unixEpochJD
		// The offset in force at that local time.
		t := time.UnixMilli(ms).In(time.Local)
		_, off := t.Zone()
		t2 := time.UnixMilli(ms - int64(off)*1000).In(time.Local)
		_, off2 := t2.Zone()
		p.jd -= int64(off2) * 1000
		p.clearYMDHMS()
		p.rawS = false
		return true
	case low == "subsec" || low == "subsecond":
		p.subsec = true
		return true
	case strings.HasPrefix(low, "weekday "):
		n, err := strconv.ParseFloat(strings.TrimSpace(low[8:]), 64)
		if err != nil || n < 0 || n >= 7 || n != math.Trunc(n) {
			return false
		}
		p.computeYMDHMS()
		p.validTZ = false
		p.validJD = false
		p.computeJD()
		z := ((p.jd + 129600000) / 86400000) % 7
		if z > int64(n) {
			z -= 7
		}
		p.jd += (int64(n) - z) * 86400000
		p.clearYMDHMS()
		return true
	case strings.HasPrefix(low, "start of "):
		if !p.validJD && !p.validYMD && !p.validHMS {
			return false
		}
		p.computeYMD()
		p.validHMS = true
		p.h, p.mi, p.s = 0, 0, 0
		p.rawS = false
		p.validTZ = false
		p.validJD = false
		switch strings.TrimSpace(low[9:]) {
		case "month":
			p.d = 1
		case "year":
			p.mo, p.d = 1, 1
		case "day":
		default:
			return false
		}
		return true
	}
	// ±NNN unit, or ±HH:MM[:SS].
	if low == "" {
		return false
	}
	num := low
	sign := 1.0
	if num[0] == '+' || num[0] == '-' {
		if num[0] == '-' {
			sign = -1
		}
		num = num[1:]
	}
	if hms := (&dateTime{}); parseHMS(num, hms) && !hms.validTZ && (low[0] == '+' || low[0] == '-') {
		p.computeJD()
		p.clearYMDHMS()
		ms := int64(hms.h)*3600000 + int64(hms.mi)*60000 + int64(hms.s*1000+0.5)
		p.jd += int64(sign) * ms
		return true
	}
	i := 0
	for i < len(num) && (num[i] >= '0' && num[i] <= '9' || num[i] == '.') {
		i++
	}
	if i == 0 {
		return false
	}
	r, err := strconv.ParseFloat(num[:i], 64)
	if err != nil {
		return false
	}
	r *= sign
	unit := strings.TrimSpace(num[i:])
	unit = strings.TrimSuffix(unit, "s")
	p.computeJD()
	rounder := 0.5
	if r < 0 {
		rounder = -0.5
	}
	switch unit {
	case "day":
		p.jd += int64(r*86400000.0 + rounder)
	case "hour":
		p.jd += int64(r*3600000.0 + rounder)
	case "minute":
		p.jd += int64(r*60000.0 + rounder)
	case "second":
		p.jd += int64(r*1000.0 + rounder)
	case "month":
		p.computeYMDHMS()
		p.mo += int(r)
		var x int
		if p.mo > 0 {
			x = (p.mo - 1) / 12
		} else {
			x = (p.mo - 12) / 12
		}
		p.y += x
		p.mo -= x * 12
		p.validJD = false
		p.computeJD()
		frac := r - float64(int(r))
		if frac != 0 {
			p.jd += int64(frac*30*86400000.0 + rounder)
		}
	case "year":
		p.computeYMDHMS()
		y := int(r)
		p.y += y
		p.validJD = false
		p.computeJD()
		frac := r - float64(y)
		if frac != 0 {
			p.jd += int64(frac*365*86400000.0 + rounder)
		}
	default:
		return false
	}
	p.clearYMDHMS()
	return true
}

// dateArgs reads the time value and modifiers of a date function.
func dateArgs(args []Value) (*dateTime, bool) {
	p := &dateTime{}
	if len(args) == 0 {
		p.jd = nowFunc().UnixMilli() + unixEpochJD
		p.validJD = true
	} else if !parseDateTime(args[0], p) {
		return nil, false
	}
	for i := 1; i < len(args); i++ {
		if args[i] == nil {
			return nil, false
		}
		if !applyModifier(textValue(args[i]), p, i == 1) {
			return nil, false
		}
	}
	p.computeJD()
	if p.isError || p.jd < 0 || p.jd > 464269060799999 {
		return nil, false
	}
	return p, true
}

func dateFunc(name string, args []Value) Value {
	if name == "strftime" {
		if len(args) == 0 || args[0] == nil {
			return nil
		}
		p, ok := dateArgs(args[1:])
		if !ok {
			return nil
		}
		return strftime(textValue(args[0]), p)
	}
	p, ok := dateArgs(args)
	if !ok {
		return nil
	}
	switch name {
	case "julianday":
		return float64(p.jd) / 86400000.0
	case "unixepoch":
		if p.subsec {
			return float64(p.jd-unixEpochJD) / 1000.0
		}
		return (p.jd - unixEpochJD) / 1000
	case "date":
		p.computeYMD()
		return ymd(p)
	case "time":
		p.computeHMS()
		return hms(p)
	}
	p.computeYMDHMS()
	return ymd(p) + " " + hms(p)
}

func ymd(p *dateTime) string {
	if p.y < 0 {
		return fmt.Sprintf("-%04d-%02d-%02d", -p.y, p.mo, p.d)
	}
	return fmt.Sprintf("%04d-%02d-%02d", p.y, p.mo, p.d)
}

func hms(p *dateTime) string {
	if p.subsec {
		return fmt.Sprintf("%02d:%02d:%06.3f", p.h, p.mi, p.s)
	}
	return fmt.Sprintf("%02d:%02d:%02d", p.h, p.mi, int(p.s))
}

func strftime(format string, p *dateTime) Value {
	p.computeYMDHMS()
	var sb strings.Builder
	dayOfYear := func() int {
		y := &dateTime{y: p.y, mo: 1, d: 1, h: p.h, mi: p.mi, s: p.s, validYMD: true, validHMS: true}
		y.computeJD()
		return int((p.jd-y.jd+43200000)/86400000) + 1
	}
	for i := 0; i < len(format); i++ {
		c := format[i]
		if c != '%' || i+1 >= len(format) {
			sb.WriteByte(c)
			continue
		}
		i++
		switch format[i] {
		case 'd':
			fmt.Fprintf(&sb, "%02d", p.d)
		case 'e':
			fmt.Fprintf(&sb, "%2d", p.d)
		case 'f':
			s := p.s
			if s > 59.999 {
				s = 59.999
			}
			fmt.Fprintf(&sb, "%06.3f", s)
		case 'F':
			sb.WriteString(ymd(p))
		case 'H':
			fmt.Fprintf(&sb, "%02d", p.h)
		case 'k':
			fmt.Fprintf(&sb, "%2d", p.h)
		case 'I', 'l':
			h := p.h % 12
			if h == 0 {
				h = 12
			}
			if format[i] == 'I' {
				fmt.Fprintf(&sb, "%02d", h)
			} else {
				fmt.Fprintf(&sb, "%2d", h)
			}
		case 'p':
			if p.h >= 12 {
				sb.WriteString("PM")
			} else {
				sb.WriteString("AM")
			}
		case 'P':
			if p.h >= 12 {
				sb.WriteString("pm")
			} else {
				sb.WriteString("am")
			}
		case 'j':
			fmt.Fprintf(&sb, "%03d", dayOfYear())
		case 'J':
			sb.WriteString(formatReal(float64(p.jd) / 86400000.0))
		case 'm':
			fmt.Fprintf(&sb, "%02d", p.mo)
		case 'M':
			fmt.Fprintf(&sb, "%02d", p.mi)
		case 's':
			if p.subsec {
				sb.WriteString(formatReal(float64(p.jd-unixEpochJD) / 1000.0))
			} else {
				fmt.Fprintf(&sb, "%d", (p.jd-unixEpochJD)/1000)
			}
		case 'S':
			fmt.Fprintf(&sb, "%02d", int(p.s))
		case 'T':
			fmt.Fprintf(&sb, "%02d:%02d:%02d", p.h, p.mi, int(p.s))
		case 'R':
			fmt.Fprintf(&sb, "%02d:%02d", p.h, p.mi)
		case 'u':
			w := int((p.jd+129600000)/86400000) % 7
			if w == 0 {
				w = 7
			}
			fmt.Fprintf(&sb, "%d", w)
		case 'w':
			fmt.Fprintf(&sb, "%d", int((p.jd+129600000)/86400000)%7)
		case 'U':
			w := int((p.jd+129600000)/86400000) % 7
			fmt.Fprintf(&sb, "%02d", (dayOfYear()-1+7-w)/7)
		case 'W':
			w := (int((p.jd+129600000)/86400000) + 6) % 7 // Monday = 0
			fmt.Fprintf(&sb, "%02d", (dayOfYear()-1+7-w)/7)
		case 'G', 'g', 'V':
			t := time.Date(p.y, time.Month(p.mo), p.d, 0, 0, 0, 0, time.UTC)
			y, wk := t.ISOWeek()
			switch format[i] {
			case 'G':
				fmt.Fprintf(&sb, "%04d", y)
			case 'g':
				fmt.Fprintf(&sb, "%02d", y%100)
			default:
				fmt.Fprintf(&sb, "%02d", wk)
			}
		case 'Y':
			fmt.Fprintf(&sb, "%04d", p.y)
		case '%':
			sb.WriteByte('%')
		default:
			return nil
		}
	}
	return sb.String()
}
