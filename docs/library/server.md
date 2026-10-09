# `server` — a web server

[Library index](index.md) · `import server`

Serving a web API or pages from a map of routes.


`import server` runs a web server. Routes are a map from `"METHOD /path"`
to what answers them, and handlers are ordinary functions: no new
keywords.

```
import server

def home[req]
    return "<h1>Hello!</h1>"                  // text: a page
def [end]

def showuser[req]
    id = id of params of req                  // from /users/:id
    return map ["id": id, "name": "Ann"]      // a map or list: JSON
def [end]

def adduser[req]
    data = json of req                        // the posted JSON
    return reply[201, map ["made": name of data]]
def [end]

app = map [
    "GET /": home,
    "GET /users/:id": showuser,
    "POST /users": adduser,
    "GET /old": redirect["/"],                // a fixed answer
    "GET /health": "ok",
    "GET /static/*": "public"                 // the files in the folder public
]
serve[app, 8080]                              // or: app serve 8080
```

```
serving on http://localhost:8080 (Ctrl+C stops it)
GET / 200 84µs
GET /users/7 200 251µs
```

| Function | Takes | Gives back |
|---|---|---|
| `serve[routes [, port]]` | a map of routes, a port (8080 if left out) | answers requests until the program is stopped |
| `reply[status [, body [, headers]]]` | a status (200, 404, ...), text or a map/list, a map of headers | a `Reply { status, body, headers }` |
| `redirect[address [, status]]` | where to send the visitor, 302 (or 301, 303, 307, 308) | a `Reply` |

**Routes.** `"GET /users/:id"`: a `:name` part matches any one part and
lands in `params`. `"GET /files/*"`: the rest of the path lands in
`params` as `"*"`. A route without a method (`"/ping"`) answers any.
The most exact route wins: fixed parts before `:names` before `*`, so
`"GET /users/new"` beats `"GET /users/:id"`. A path no route has is 404;
one with only other methods is 405. `GET` routes answer `HEAD` too.

**What answers.** A function of one name gets the request; one of none
is just called. A saved scroll works too, getting the request. Any other
value is the answer itself, and text on a `/*` route is a folder: its
files are served (`index.html` for a folder), never anything outside it.

**The request** is a map, read with `of`: `method`, `path`, `params`,
`query` (text, or a list when a name comes twice), `headers` (lowercase
names), `body` (text), `json` (the body as JSON when it's sent as JSON,
else `none`), `form` (a posted form's fields), `ip`.

**What a handler gives back:** text is a page (HTML when it starts with
`<`, plain text otherwise); a map, list or assembled value is JSON;
`reply[...]` sets the status and headers; `none` is 204 (no content).

**Errors.** An error in a handler answers `500 server error`, and the
error shows in the terminal (`GET /broken: line 17: division by zero`);
the server goes on. `exit[]` in a handler stops the server and the
program. A port in use, a route that isn't `"METHOD /path"`, or a `/*`
folder that doesn't exist is an error of kind `server`.

**Settings** (`import server` makes them; change them like any variable):

| Setting | Default | Meaning |
|---|---|---|
| `serverlog` | `true` | a line per request: method, path, status, time |
| `serverhost` | `"localhost"` | only this computer; `"0.0.0.0"` answers others on the network |

**One request at a time.** Turtle code runs one line at a time, so
requests take turns (the network work around them runs side by side).
That's fine for tools, dashboards and small sites.

## Safety

- **Only this computer, unless you say otherwise.** By default the server
  answers only on `localhost`. `serverhost = "0.0.0.0"` opens it to the
  network; do that only on purpose.
- **Hidden files are never served.** A static folder route won't serve
  a name starting with `.` (`.env`, `.git`, ...), anything outside the
  folder (`../`), or a link inside it that leads outside.
- **Slow clients wait alone.** A request must arrive within a minute,
  body and all, and only the Turtle handler itself takes a turn; reading
  a request, sending a file and sending an answer happen side by side,
  so one slow connection doesn't hold up the others. Bodies are limited
  to 10 MB.
- **Errors stay private.** A handler that fails answers `500 server
  error`; the details go to the terminal, not to the visitor.
- **On the internet, put a web server in front.** Turtle speaks plain
  HTTP. For a public site, run it behind one that handles HTTPS
  certificates (Caddy, nginx), with `serverhost` left as `localhost`.
