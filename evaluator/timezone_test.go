package evaluator

import (
	"strings"
	"testing"
)

// Time zones: dates keep their zone, show it when it isn't the
// computer's, compare as moments, and go to data in a form that reads
// back as the same moment.
func TestTimeZones(t *testing.T) {
	cases := []struct{ src, want string }{
		{`d = make_date[2026, 12, 25, 9, 0, 0, "Europe/London"]
show d .`, "2026-12-25 09:00:00 GMT"},
		{`d = make_date[2026, 12, 25, 9, 0, 0, "Europe/London"]
ny = to_zone[d, "America/New_York"]
show ny, "|", zone of ny, "|", hour of ny, "|", ny == d .`, "2026-12-25 04:00:00 EST|America/New_York|4|true"},
		{`d = make_date[2026, 12, 25, 9, 0, 0, "Europe/London"]
ny = d to_zone "America/New_York"
show format_date[ny, "hh:mm Zone (Offset)"] .`, "04:00 EST (-05:00)"},
		{`d = make_date[2026, 7, 1, "Asia/Tokyo"]
show d, "|", zone of d .`, "2026-07-01 00:00:00 JST|Asia/Tokyo"},
		{`show to_date["2026-10-03 14:05", "Asia/Tokyo"] .`, "2026-10-03 14:05:00 JST"},
		{`d = to_date["2026-10-03T14:05:00+05:45"]
show zone of d, "|", hour of d .`, "+05:45|14"},
		{`show zone of today[], "|", zone of today_utc[], "|", zone of today["UTC"], "|", zone of today["local"] .`, "local|UTC|UTC|local"},
		{`show time_between[make_date[2026, 1, 1, 0, 0, 0, "Asia/Tokyo"], make_date[2026, 1, 1, 0, 0, 0, "UTC"], "hours"] .`, "9"},
		// add_time keeps the zone's clock, across its daylight-saving change.
		{`d = make_date[2026, 3, 7, 12, 0, 0, "America/New_York"]
show add_time[d, 1, "days"] .`, "2026-03-08 12:00:00 EDT"},
		// A local date shows and goes to data just as before.
		{`d = make_date[2026, 1, 2, 3, 4, 5]
show d, "|", json_text[d] .`, `2026-01-02 03:04:05|"2026-01-02 03:04:05"`},
		// Another zone's date goes to data with its offset, and reads back.
		{`d = make_date[2026, 7, 1, 9, 0, 0, "Asia/Tokyo"]
s = json_text[d]
back = load[s]
show s, "|", to_date[back] == d .`, `"2026-07-01T09:00:00+09:00"|true`},
		// The same moment is the same set member and map key.
		{`d = make_date[2026, 7, 1, 9, 0, 0, "Asia/Tokyo"]
s = set [d, to_zone[d, "UTC"]]
show s at len .`, "1"},
	}
	for _, c := range cases {
		got, err := run(t, "import time\nimport json\n"+c.src, "")
		if err != nil {
			t.Errorf("%s: %v", c.src, err)
		} else if strings.TrimSpace(got) != c.want {
			t.Errorf("%s:\n got  %q\n want %q", c.src, strings.TrimSpace(got), c.want)
		}
	}
}

func TestTimeZoneErrors(t *testing.T) {
	cases := []struct{ src, kind, want string }{
		{`today["Mars/Base"]`, "date", `today: "Mars/Base" isn't a time zone`},
		{`to_zone[today[], ""]`, "date", `isn't a time zone`},
		{`to_zone[today[], 5]`, "type", `'to_zone' time zone must be text`},
		{`today[1, 2]`, "type", `'today' expects nothing, or a time zone`},
		{`make_date[2026, 2, 30, "UTC"]`, "date", `isn't a real date`},
		{`to_zone[5, "UTC"]`, "type", `'to_zone' needs a date`},
	}
	for _, c := range cases {
		src := "import time\nsafe\n    x = " + c.src + "\nhandle [] e .\n    show kind of e, \"|\", message of e .\nsafe [end]"
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
