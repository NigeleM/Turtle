# `http` — web requests

[Library index](index.md) · `import http`

GET, POST and any other web request.


`import http` makes web requests:

| Function | Args | Returns |
|---|---|---|
| `http_get[url]` | web address | the response body as text |
| `http_post[url, body]` | web address, body | the response body as text |
| `http_request[method, url [, body [, headers]]]` | `"GET"`, `"PUT"`, ...; address; optional body and headers map | a map with `status` (integer), `body` (text), `headers` (map, lowercase names) |

**Bodies:** text is sent as is (`text/plain`); a map, list, set or
assembled value is sent as JSON (`application/json`); `none` sends no body.

**Errors** are kind `http`. `http_get` and `http_post` fail on a network
problem or a 4xx/5xx status: `http_get https://api.example.com/x: 404 Not
Found: no such page`. `http_request` only fails on a network problem, and
hands back any status for you to check. A request with no answer in 30
seconds fails, and so does a response larger than 64 MB (it isn't cut
short).

> *Illustrative.* This example needs a network address or a running server, so it is shown for reading and is not run as a test.

```
import http
import json
import system

safe
    users = load[http_get["https://api.example.com/users"]]   // a list of maps
handle [http, json] e .
    warn "couldn't fetch users: ", e .
    users = list []
safe [end]

reply = http_post["https://api.example.com/users", map ["name": "Ann"]]

r = http_request["DELETE", "https://api.example.com/users/7", none, map ["Authorization": "Bearer {token}"]]
if ] r at get["status"] != 204 [
    show "delete failed: ", r at get["body"] .
if [end]
```

---

Copyright 2017-2026 Nigele McCoy. Licensed under the
[Apache License 2.0](../../LICENSE); see [NOTICE](../../NOTICE).

<!-- generated: turtle doc `http` (regenerate with the script; do not edit by hand) -->
## Function reference

The entries below are copied from the binary's own documentation (`turtle doc http`), so they match the version you run.

```text
import http
```

### `Web requests. A map, list or assembled value sent as a body goes as JSON.`

```text
Web requests. A map, list or assembled value sent as a body goes as JSON.
```

### `http_get[url]`

```text
http_get[url]        (import http)
  Fetches a web address.
  Gives back: the response's body as text. A network problem or an
  error status (4xx, 5xx) is an http error.
```

### `http_post[url, body]`

```text
http_post[url, body]        (import http)
  Sends body to a web address.
  Gives back: the response's body as text.
```

### `http_request[method, url, body, headers]`

```text
http_request[method, url, body, headers]        (import http)
  Any request: "GET", "PUT", "DELETE", ... body and headers (a map) are
  optional.
  Gives back: a map with "status" (a number), "body" (text) and
  "headers" (a map). Only a network problem is an error; check the
  status yourself.
```

<!-- end generated -->
