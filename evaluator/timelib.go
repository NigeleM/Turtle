package evaluator

import (
	"strings"
	"time"

	"Turtle/object"
)

// The "time" builtin module: the clock, dates, date arithmetic and
// waiting.
//
//	t = now[]                                  // milliseconds since 1970
//	sleep[2]  sleep[250, "ms"]                 // pause
//	d = today[]                                // local date and time
//	u = today_utc[]                            // the same moment in UTC
//	d = make_date[2026, 10, 3]                 // or [y, m, d, h, mi, s]
//	d = to_date["2026-10-03 14:05"]            // from text
//	due = add_time[d, 30, "days"]              // negative subtracts
//	left = time_between[today[], due, "days"]  // whole units, b - a
//	s = format_date[d, "Weekday, Month D YYYY"]
//	wait_until[due]                            // sleep until then
//	every[7, "days", job]                      // run job now and on a repeat
//
// Dates are whole seconds. Bad date text, or a date that doesn't exist
// (February 30), is an error of kind date.
func (it *Interpreter) callTime(name string, args []object.Object) object.Object {
	switch name {
	case "now":
		requireFuncArgs(name, args, 0)
		return &object.Integer{Value: time.Now().UnixMilli()}
	case "sleep":
		evalSleep(args)
		return object.NoneValue
	case "today":
		requireFuncArgs(name, args, 0)
		return &object.Date{Time: time.Now().Truncate(time.Second)}
	case "today_utc":
		requireFuncArgs(name, args, 0)
		return &object.Date{Time: time.Now().UTC().Truncate(time.Second)}
	case "make_date":
		return makeDate(args)
	case "to_date":
		requireFuncArgs(name, args, 1)
		return parseDate(asStringArg(name, args[0]))
	case "add_time":
		requireFuncArgs(name, args, 3)
		d := asDateArg(name, args[0])
		n := asIntArg(name, "amount", args[1])
		return &object.Date{Time: addTime(d.Time, n, asUnitArg(name, args[2]))}
	case "time_between":
		requireFuncArgs(name, args, 3)
		a, b := asDateArg(name, args[0]), asDateArg(name, args[1])
		return &object.Integer{Value: timeBetween(a.Time, b.Time, asUnitArg(name, args[2]))}
	case "format_date":
		requireFuncArgs(name, args, 2)
		return &object.String{Value: formatDate(asDateArg(name, args[0]).Time, asStringArg(name, args[1]))}
	case "wait_until":
		requireFuncArgs(name, args, 1)
		if d := time.Until(asDateArg(name, args[0]).Time); d > 0 {
			time.Sleep(d)
		}
		return object.NoneValue
	case "every":
		it.every(args)
		return object.NoneValue
	}
	fatalKind(kindName, "no builtin function %q in \"time\"", name)
	return nil
}

func asDateArg(fn string, obj object.Object) *object.Date {
	d, ok := obj.(*object.Date)
	if !ok {
		fatalf("'%s' needs a date (from today[], make_date or to_date), got %s", fn, obj.Type())
	}
	return d
}

func asIntArg(fn, what string, obj object.Object) int {
	n, ok := obj.(*object.Integer)
	if !ok {
		fatalf("'%s' %s must be a whole number, got %s", fn, what, obj.Type())
	}
	return int(n.Value)
}

// The units add_time, time_between and every take. A singular ("day") is
// accepted too.
var timeUnits = []string{"seconds", "minutes", "hours", "days", "weeks", "months", "years"}

func asUnitArg(fn string, obj object.Object) string {
	s, ok := obj.(*object.String)
	if ok {
		for _, u := range timeUnits {
			if s.Value == u || s.Value+"s" == u {
				return u
			}
		}
	}
	fatalf("'%s' unit must be one of %s, got %s", fn, strings.Join(timeUnits, ", "), object.Shown(obj))
	return ""
}

// makeDate is make_date[year, month, day] or [year, month, day, hour,
// minute, second], in local time. A date that doesn't exist is an error,
// not quietly moved to the next month.
func makeDate(args []object.Object) object.Object {
	if len(args) != 3 && len(args) != 6 {
		fatalf("'make_date' expects 3 arguments (year, month, day) or 6 (and hour, minute, second), got %d", len(args))
	}
	names := []string{"year", "month", "day", "hour", "minute", "second"}
	v := make([]int, 6)
	for i, a := range args {
		v[i] = asIntArg("make_date", names[i], a)
	}
	t := time.Date(v[0], time.Month(v[1]), v[2], v[3], v[4], v[5], 0, time.Local)
	if t.Year() != v[0] || int(t.Month()) != v[1] || t.Day() != v[2] || t.Hour() != v[3] || t.Minute() != v[4] || t.Second() != v[5] {
		fatalKind(kindDate, "make_date: %04d-%02d-%02d %02d:%02d:%02d isn't a real date and time", v[0], v[1], v[2], v[3], v[4], v[5])
	}
	return &object.Date{Time: t}
}

// dateLayouts are the text forms to_date reads: the way dates show, with
// or without seconds or a time, and ISO 8601 / RFC 3339 with a zone.
var dateLayouts = []string{
	"2006-01-02 15:04:05",
	"2006-01-02 15:04",
	"2006-01-02",
	"2006-01-02T15:04:05",
	time.RFC3339,
}

func parseDate(text string) object.Object {
	if d, ok := tryParseDate(text); ok {
		return d
	}
	fatalKind(kindDate, "to_date: %q isn't a date; use YYYY-MM-DD, optionally with hh:mm or hh:mm:ss", text)
	return nil
}

// tryParseDate is parseDate without the error: false if text isn't a date.
func tryParseDate(text string) (object.Object, bool) {
	s := strings.TrimSpace(text)
	for _, layout := range dateLayouts {
		var t time.Time
		var err error
		if strings.Contains(layout, "Z07") {
			t, err = time.Parse(layout, s)
		} else {
			t, err = time.ParseInLocation(layout, s, time.Local)
		}
		if err == nil {
			return &object.Date{Time: t.Truncate(time.Second)}, true
		}
	}
	return nil, false
}

// addTime adds n units. Days, weeks, months and years keep the clock time
// (across daylight-saving changes too); a month or year that lands past
// the end of a month stays on its last day: January 31 + 1 month is
// February 28 (or 29).
func addTime(t time.Time, n int, unit string) time.Time {
	switch unit {
	case "seconds":
		return t.Add(time.Duration(n) * time.Second)
	case "minutes":
		return t.Add(time.Duration(n) * time.Minute)
	case "hours":
		return t.Add(time.Duration(n) * time.Hour)
	case "days":
		return t.AddDate(0, 0, n)
	case "weeks":
		return t.AddDate(0, 0, 7*n)
	case "months":
		return addMonths(t, n)
	}
	return addMonths(t, 12*n) // years
}

func addMonths(t time.Time, n int) time.Time {
	first := time.Date(t.Year(), t.Month(), 1, t.Hour(), t.Minute(), t.Second(), 0, t.Location())
	target := first.AddDate(0, n, 0)
	last := target.AddDate(0, 1, -1).Day()
	day := t.Day()
	if day > last {
		day = last
	}
	return time.Date(target.Year(), target.Month(), day, t.Hour(), t.Minute(), t.Second(), 0, t.Location())
}

// timeBetween is how many whole units fit from a to b: negative when b
// is before a. Months and years count calendar months the way add_time
// adds them: January 31 to February 28 is 1 month (add_time[Jan 31, 1,
// "months"] is February 28), January 15 to February 14 is 0.
func timeBetween(a, b time.Time, unit string) int64 {
	switch unit {
	case "seconds":
		return int64(b.Sub(a) / time.Second)
	case "minutes":
		return int64(b.Sub(a) / time.Minute)
	case "hours":
		return int64(b.Sub(a) / time.Hour)
	case "days", "weeks":
		// Calendar days, so a daylight-saving day (23 or 25 hours) still
		// counts as one: estimate from the hours, then correct by a day.
		days := int(b.Sub(a) / (24 * time.Hour))
		if !b.Before(a) {
			for !addTime(a, days+1, "days").After(b) {
				days++
			}
			for addTime(a, days, "days").After(b) {
				days--
			}
		} else {
			for !addTime(a, days-1, "days").Before(b) {
				days--
			}
			for addTime(a, days, "days").Before(b) {
				days++
			}
		}
		if unit == "weeks" {
			return int64(days / 7)
		}
		return int64(days)
	}
	months := int64((b.Year()-a.Year())*12 + int(b.Month()) - int(a.Month()))
	// Not a whole month yet if adding it overshoots b (or undershoots,
	// going back).
	if months > 0 && addMonths(a, int(months)).After(b) {
		months--
	} else if months < 0 && addMonths(a, int(months)).Before(b) {
		months++
	}
	if unit == "years" {
		return months / 12
	}
	return months
}

// formatDate fills a pattern: YYYY year, MM/M month, DD/D day, hh hour
// (00-23), mm minute, ss second, Month (October), Mon (Oct), Weekday
// (Saturday), Wkd (Sat). Anything else is copied as is.
func formatDate(t time.Time, pattern string) string {
	two := func(n int) string {
		if n < 10 {
			return "0" + itoa(n)
		}
		return itoa(n)
	}
	tokens := []struct {
		tok string
		val func() string
	}{
		{"YYYY", func() string { return itoa(t.Year()) }},
		{"Weekday", func() string { return t.Weekday().String() }},
		{"Month", func() string { return t.Month().String() }},
		{"Mon", func() string { return t.Month().String()[:3] }},
		{"Wkd", func() string { return t.Weekday().String()[:3] }},
		{"MM", func() string { return two(int(t.Month())) }},
		{"DD", func() string { return two(t.Day()) }},
		{"hh", func() string { return two(t.Hour()) }},
		{"mm", func() string { return two(t.Minute()) }},
		{"ss", func() string { return two(t.Second()) }},
		{"M", func() string { return itoa(int(t.Month())) }},
		{"D", func() string { return itoa(t.Day()) }},
	}
	var sb strings.Builder
	for i := 0; i < len(pattern); {
		matched := false
		for _, tk := range tokens {
			if strings.HasPrefix(pattern[i:], tk.tok) {
				sb.WriteString(tk.val())
				i += len(tk.tok)
				matched = true
				break
			}
		}
		if !matched {
			sb.WriteByte(pattern[i])
			i++
		}
	}
	return sb.String()
}

func itoa(n int) string { return (&object.Integer{Value: int64(n)}).Inspect() }

// every runs job now, then again every amount units, until job returns
// false. Each run is scheduled from the start time, so a slow job doesn't
// make later runs drift. An error in job stops the program unless the
// job handles it with safe.
func (it *Interpreter) every(args []object.Object) {
	if len(args) != 3 {
		fatalf("'every' expects 3 arguments (amount, unit, function), e.g. every[7, \"days\", backup], got %d", len(args))
	}
	n := asIntArg("every", "amount", args[0])
	if n <= 0 {
		fatalf("'every' amount must be at least 1, got %d", n)
	}
	unit := asUnitArg("every", args[1])
	fn, ok := args[2].(*object.Function)
	if !ok {
		fatalf("'every' needs a function to run (e.g. backup, or [] give ...), got %s", args[2].Type())
	}
	if len(fn.Parameters) != 0 {
		fatalf("'every' runs a function with no parameters, but this one takes %d", len(fn.Parameters))
	}
	start := time.Now()
	for k := 1; ; k++ {
		if res, ok := it.callFunction(fn, "every", nil).(*object.Boolean); ok && !res.Value {
			return
		}
		next := addTime(start, n*k, unit)
		if d := time.Until(next); d > 0 {
			time.Sleep(d)
		}
	}
}
