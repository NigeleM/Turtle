# `http` — web requests

[Library index](index.md) · `import http`

GET, POST and any other web request.


`import http` makes web requests, using only Go's standard library:

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
seconds fails.

```
import http
import json

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
