# `log` — logging

`import log`

Write log lines with a level and a time, to the screen or a file. Use it to record what a program does, so problems can be found later.


## Statements

`log`, with a level before the message: `debug`, `info`, `warn` or `error`. With no level it is `info`.

## Example

### `log level "message", value`

Writes a line with the time, the level and the message. A map adds fields after the message. `debug` lines don't show unless the level is turned down.

**Example**

```turtle
import log
log "server started on port ", 8080 .
log warn "disk at ", 91, "%" .
log error "lost connection to ", "db1" .
log info "login", map ["user": "ann", "ip": "10.0.0.4"] .
```

**Output**

```text
2026-11-19 01:59:13 INFO  server started on port 8080
2026-11-19 01:59:13 WARN  disk at 91%
2026-11-19 01:59:13 ERROR lost connection to db1
2026-11-19 01:59:13 INFO  login user=ann ip=10.0.0.4
```

The times above are when the example was run; your lines will show your own time.
