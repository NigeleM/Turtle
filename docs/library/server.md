# `server` — a web server

`import server`

Answer web requests from your program: define routes, send replies with their own status, and redirect visitors.


Answer web requests from your Turtle program. `serve` runs until you stop it with Ctrl+C.

## Functions

`serve`, `reply`, `redirect`.

### `serve[routes, port]`

Starts a web server. `routes` is a map from `"METHOD /path"` to a function or a fixed text. `port` is optional and is 8080 by default.

**Example**

```turtle
import server
def hello[req]
    return "hello"
def [end]
routes = map ["GET /": hello, "GET /health": "ok"]
serve[routes, 8080]
```

**Output**

```text
Then, in another window:

$ curl http://127.0.0.1:8080/
hello
$ curl http://127.0.0.1:8080/health
ok
```

### `reply[status, body, headers]`

Gives back an answer with its own status code. Use it when a route must answer with something other than 200.

**Example**

```turtle
import server
def find[req]
    return reply[404, "no such user"]
def [end]
routes = map ["GET /users/x": find]
serve[routes, 8080]
```

**Output**

```text
$ curl -i http://127.0.0.1:8080/users/x
HTTP/1.1 404 Not Found
Content-Type: text/plain; charset=utf-8
...

no such user
```

### `redirect[address, status]`

Sends the visitor to another address. The default status is 302.

**Example**

```turtle
import server
def go[req]
    return redirect["/login"]
def [end]
routes = map ["GET /old": go]
serve[routes, 8080]
```

**Output**

```text
$ curl -i http://127.0.0.1:8080/old
HTTP/1.1 302 Found
Location: /login
```

The examples use `serve` in a file of its own, because it runs until it is stopped. Start it, then open the address in a browser or with `curl`.
