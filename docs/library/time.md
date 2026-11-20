# `time` — dates, clocks and waiting

`import time`

Dates, clocks and waiting: the current time, time zones, adding and comparing dates, and waiting until a moment.


## Functions

`now`, `sleep`, `today`, `to_zone`, `today_utc`, `make_date`, `to_date`, `add_time`, `time_between`, `format_date`, `wait_until`, `every`.

### `now[]`

Milliseconds since 1 January 1970: for measuring how long something takes.

**Example**

```turtle
start = now[]
show "took ", now[] - start, " ms" .
```

**Output**

```text
took 0 ms
```

### `sleep[amount, unit]`

Pauses the program. amount   how long, a number (0.25 is fine) unit     optional: "seconds" (the default) or "ms"

**Example**

```turtle
show sleep[250, "ms"] .
```

This needs a running server, database or network, so it is shown without output.

### `today[zone]`

The date and time now, in local time, or in a time zone. zone     optional: "Asia/Tokyo", "UTC", ...

**Example**

```turtle
show weekday of today[] .
tokyo = today["Asia/Tokyo"]
```

**Output**

```text
Thursday
```

### `to_zone[date, zone]`

The same moment, on another zone's clock.

**Example**

```turtle
d = make_date[2026, 1, 15, 12, 0]
ny = to_zone[d, "America/New_York"]
show zone of ny .
```

**Output**

```text
America/New_York
```

### `today_utc[]`

The date and time now, in UTC.

**Example**

```turtle
show today_utc[] .
```

**Output**

```text
2026-11-19 06:42:24 UTC
```

### `make_date[year, month, day, hour, minute, second, zone]`

A date from its parts.

**Example**

```turtle
import time
d = make_date[2026, 12, 25]
d = make_date[2026, 12, 25, 9, 30]
d = make_date[2026, 12, 25, 9, 0, 0, "Europe/London"]
show d .
```

**Output**

```text
2026-12-25 09:00:00 GMT
```

### `to_date[text, zone]`

Reads a date from text: "2026-10-03", "2026-10-03 14:05", "2026-10-03 14:05:00", or ISO 8601 ("2026-10-03T14:05:00Z", "2026-10-03T14:05:00-04:00", which keep their offset). zone     optional: the zone for text that names none (local otherwise) Gives back: a date.

**Example**

```turtle
import time
d = to_date["2026-10-03"]
show d .
```

**Output**

```text
2026-10-03 00:00:00
```

### `add_time[date, amount, unit]`

A date moved forward (or back, with a negative amount).

**Example**

```turtle
import time
due = add_time[today[], 30, "days"]
show due .
```

**Output**

```text
2026-12-19 14:58:18
```

The output is 30 days after today, so it shows the date on the day you run it.

### `time_between[a, b, unit]`

How many whole units from date a to date b (negative if b is earlier).

**Example**

```turtle
due = add_time[today[], 7, "days"]
left = time_between[today[], due, "days"]
show left .
```

**Output**

```text
7
```

### `format_date[date, pattern]`

Writes a date with a pattern: YYYY year, MM or M month, DD or D day, hh hour (00-23), mm minute, ss second, Month (March), Mon (Mar), Weekday (Thursday), Wkd (Thu), Zone (EST), Offset (-05:00).

**Example**

```turtle
show format_date[today[], "DD/MM/YYYY hh:mm"] .
```

**Output**

```text
19/11/2026 01:42
```

### `wait_until[date]`

Pauses until that moment (at once if it's past).

**Example**

```turtle
wait_until[today[]]
show "the date has come" .
```

This needs a running server, database or network, so it is shown without output.

### `every[amount, unit, job]`

Runs job now, and then again every amount units, until job returns false. amount   a whole number unit     "minutes", "days", ... job      a function with no parameters Illustrative: runs until the job returns false, so it is not run as a check.

**Example**

```turtle
def backup[]
    show "backing up" .
    return true
def [end]
every[7, "days", backup]
```

This runs until it is stopped, so it is shown without output.
