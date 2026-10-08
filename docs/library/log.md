# `log` — logging

[Library index](index.md) · `import log`

Log lines with levels and timestamps, to the screen or a file.


`import log` adds the `log` statement: a line with a level, a time and
where it came from, written to the console and, if you like, a file.

```
import log

log "server started on port ", port .            // info, the default level
log debug "row ", row .
log warn "disk at ", pct, "%" .
log error "lost connection to ", host .
log info "login", map ["user": name, "ip": ip] .  // a map adds fields
```

```
2026-10-06 19:48:04 INFO  server started on port 8080
2026-10-06 19:48:04 WARN  disk at 91%
2026-10-06 19:48:04 ERROR lost connection to db1
2026-10-06 19:48:04 INFO  login user=ann ip=10.0.0.4
```

The values are written one after another, as `show` writes them; a map
among them becomes `key=value` fields (a value with a space is quoted:
`from="home office"`).

**Levels**: `debug` < `info` < `warn` < `error`. Lines below `loglevel`
are skipped, so `log debug` lines can stay in the code and be turned on
when needed.

**Settings.** `import log` makes these variables; set them like any
variable. Set at the top of a file, they hold for the file; set inside a
function, for that function. An imported library that logs uses its own.

| Variable | Default | Means |
|---|---|---|
| `loglevel` | `"info"` | the lowest level shown: `"debug"`, `"info"`, `"warn"`, `"error"`, or `"off"` for nothing |
| `logconsole` | `true` | print log lines to the console. They go to stderr, so a program's own output (`show`) stays clean, and `turtle report.trt > out.txt` captures only that |
| `logfile` | `none` | a file to add each line to. It's added to across runs, never replaced, and not held open, so it can be read, moved or deleted while the program runs |
| `logtime` | `"YYYY-MM-DD hh:mm:ss"` | the time, in `format_date`'s patterns (`"hh:mm"`, `"DD Mon hh:mm"` ...); `none` leaves it out |
| `logparts` | `list ["time", "level", "message"]` | what a line shows, in order: `time`, `level`, `message`, `file`, `line`, `where` (`file:line`) |
| `logformat` | `"text"` | `"json"`: one JSON object per line, for tools that read logs: `{"time": ..., "level": "warn", "message": ..., "file": ..., "line": 12, "sku": "B1"}` (fields are keys of their own) |
| `logmaxsize` | `none` | rotate the log file before it grows past this many bytes: `app.log` becomes `app.log.1`, `.1` becomes `.2`, ... |
| `logkeep` | `3` | how many rotated files to keep (the oldest goes) |
| `outputfile` | `none` | a file that also gets everything `show` and `warn` print, to look at a run later |

```
logfile = "app.log"
logparts = list ["time", "level", "where", "message"]
log warn "retrying" .                  // 2026-10-06 19:48:04 WARN  main.trt:14 retrying

logmaxsize = 1000000                   // about 1 MB per file
logkeep = 5

outputfile = "run.txt"                 // show still prints; run.txt keeps a copy
```

- **When a program stops with an error** and has a `logfile`, the error
  is written there too: `ERROR main.trt:18 stopped: division by zero`.
- `log` begins a statement only in a file with `import log`; elsewhere it's
  an ordinary name, and `log = 5` still assigns.
- A log file or `outputfile` that can't be written is an error of kind
  `file`; a setting of the wrong kind is a `type` error naming it.
