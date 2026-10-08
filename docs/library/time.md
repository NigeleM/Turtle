# `time` — dates, clocks and waiting

[Library index](index.md) · `import time`

The clock, dates and time zones, date arithmetic and formatting, and waiting or repeating.


`import time` gives the clock, dates, time zones, date arithmetic, and waiting:

| Function | Args | Returns |
|---|---|---|
| `now[]` | — | milliseconds since the Unix epoch, as an Integer |
| `sleep[amount [, unit]]` | amount, optional `"seconds"` (default) or `"ms"` | pauses that long; returns `none` |
| `today[]` or `today[zone]` | optional time zone | the date and time now, in local time or in that zone |
| `today_utc[]` | — | the same moment, shown in UTC (`today["UTC"]`) |
| `to_zone[date, zone]` | date, time zone | the same moment on that zone's clock |
| `make_date[y, m, d]` or `[y, m, d, h, mi, s]` | whole numbers, and optionally a time zone last | that date and time, local or in the zone; one that doesn't exist (Feb 30) is a `date` error |
| `to_date[text [, zone]]` | `"YYYY-MM-DD"`, optionally with ` hh:mm` or ` hh:mm:ss`, or ISO 8601 (`2026-10-03T14:05:00Z`, `...-04:00`); optionally the zone for text that names none | the date; other text is a `date` error |
| `add_time[date, amount, unit]` | date, whole number, unit | a new date; a negative amount goes back |
| `time_between[a, b, unit]` | two dates, unit | how many whole units from `a` to `b` (negative if `b` is earlier) |
| `format_date[date, pattern]` | date, pattern text | the date written with the pattern (below) |
| `wait_until[date]` | date | sleeps until then (at once if it's past); returns `none` |
| `every[amount, unit, job]` | whole number, unit, a function with no parameters | runs `job` now and then on a repeat, until `job` returns `false` |

**Units:** `"seconds"`, `"minutes"`, `"hours"`, `"days"`, `"weeks"`,
`"months"`, `"years"` (or the singular, `"day"`).

**A date** shows as `2026-10-03 14:05:00`, whole seconds, in its own
clock: local, or another zone's, and then its zone shows too
(`2026-12-25 09:00:00 GMT`). Read its parts with `of`: `year`, `month`,
`day`, `hour`, `minute`, `second` (whole numbers), `weekday`
(`"Saturday"`) and `zone` (`"local"`, `"UTC"`, `"Asia/Tokyo"`, or an
offset like `"-04:00"` for a date read from text with one). Parts are read-only; make a new date
with `add_time` or `make_date`. Dates compare with `< > <= >= == !=` (the
same moment is equal whichever clock shows it), can be map keys and set
elements, join text with `+` or `{d}`, and are written to JSON as text:
a local date as `2026-10-03 14:05:00`, as always, and another zone's
with its offset (`2026-07-01T09:00:00+09:00`), so `to_date` reads it
back as the same moment. Files and databases get the same forms.

**Time zones** are names like `"America/New_York"`, `"Europe/London"`,
`"Asia/Tokyo"`, plus `"UTC"` and `"local"` (the computer's own). The
list of zones is built into Turtle, so the names work on every system,
Windows too. An unknown name is a `date` error.

```
import time

d = make_date[2026, 12, 25, 9, 0, 0, "Europe/London"]
ny = to_zone[d, "America/New_York"]        // or: d to_zone "America/New_York"
show ny .                                  // 2026-12-25 04:00:00 EST
show zone of ny, " ", ny == d .            // America/New_York true
show format_date[ny, "hh:mm Zone (Offset)"] .   // 04:00 EST (-05:00)
tokyo = today["Asia/Tokyo"]
```

```
import time

d = today[]
show d .                                     // 2026-10-03 14:05:00
show weekday of d .                          // Saturday

due = add_time[d, 30, "days"]                // 30 days from now
last_week = add_time[d, -7, "days"]
left = time_between[d, due, "days"]          // 30
show format_date[due, "Weekday, Month D YYYY"] .   // Monday, November 2 2026

if ] today[] > due [
    show "overdue" .
if [end]
```

**Months and years stay in the month:** January 31 + 1 month is
February 28 (29 in a leap year), not March 3; February 29 + 1 year is
February 28. `time_between` counts months the same way, so January 31 to
February 28 is 1 month.

**Days keep the clock time** across daylight-saving changes: noon + 1 day
is noon the next day, even when that day has 23 or 25 hours. `"hours"`
is exact: noon + 24 hours can be 11:00 or 13:00 on those days.

**`format_date` patterns:** `YYYY` year, `MM`/`M` month (`03`/`3`),
`DD`/`D` day, `hh` hour 00-23, `mm` minute, `ss` second, `Month`
(`March`), `Mon` (`Mar`), `Weekday` (`Thursday`), `Wkd` (`Thu`), `Zone`
(`EST`), `Offset` (`-05:00`). Anything else is copied as is: `format_date[d, "DD/MM/YYYY hh:mm"]` → `05/03/2026 14:07`.

**Automation.** `wait_until` and `every` run inside your script, so the
script has to keep running (in a terminal, or as a service). `every`
schedules each run from when it started, so a slow job doesn't push later
runs back. Return `false` from the job to stop; an error in the job stops
the program unless the job handles it with `safe`.

```
import time
import system

def backup[]
    sys cp data.db backups/
    warn "backed up at ", today[] .
def [end]

wait_until[add_time[today[], 1, "hours"]]    // run once, an hour from now
backup[]

every[7, "days", backup]                     // then every 7 days, forever
```

`sleep` examples:

```
sleep[1]            // 1 second
sleep[0.25]         // a quarter second
sleep[250, "ms"]    // 250 milliseconds
```

If you've already defined your own top-level function with one of these
names (`now`, `today`, ...), it always wins — the builtin only kicks in when no user function
of that name exists.

Plain `<result> is <receiver> .` (no `at`) is just assignment/aliasing —
`<result>` becomes another reference to the same underlying value.
