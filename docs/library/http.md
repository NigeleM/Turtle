# `http` — web requests

`import http`

Get and send data over the web: fetch a page, post data, or send any request with its own method and headers.


Get and send data over the web. Addresses start with `http://` or `https://`. (A local test server, for the examples: `GET /data` gives the text `some text`, `POST /users` gives `made`, `PUT /users/7` gives `updated`.)

## Functions

`http_get`, `http_post`, `http_request`.

### `http_get[url]`

Fetches a web address and gives back the page's text. An error status (4xx or 5xx) is an error.

**Example**

```turtle
import http
body = http_get["http://127.0.0.1:8765/data"]
show body .
```

**Output**

```text
some text
```

### `http_post[url, body]`

Sends `body` to a web address and gives back the reply's text.

**Example**

```turtle
import http
reply = http_post["http://127.0.0.1:8765/users", map ["name": "Ann"]]
show reply .
```

**Output**

```text
made
```

### `http_request[method, url, body, headers]`

Any request: `"GET"`, `"PUT"`, `"DELETE"` and so on. Gives back a map with `status`, `body` and `headers`. Only a network problem is an error; check the status yourself.

**Example**

```turtle
import http
reply = http_request["PUT", "http://127.0.0.1:8765/users/7", map ["name": "Ann"], map ["Content-Type": "application/json"]]
show status of reply .
show reply at get["body"] .
```

**Output**

```text
200
updated
```
