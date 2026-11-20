# `json` — JSON

`import json`

Turn JSON text into Turtle values, and Turtle values back into JSON text, for web APIs and saved data.


## Functions

`load`, `json_text`, `json_read`, `json_write`, `json_get`.

### `load[text]`

Reads JSON text.

**Example**

```turtle
import json
user = load['{"name": "Ann"}']
show user .
```

**Output**

```text
{ "name": "Ann" }
```

### `json_text[value]`

Writes a value as JSON, on one line.

**Example**

```turtle
show json_text[map ["id": 7, "tags": list ["a", "b"]]] .
```

**Output**

```text
{"id":7,"tags":["a","b"]}
```

### `json_read[path]`

Reads a JSON file.

**Example**

```turtle
json_write["user.json", map ["name": "Ann"]]
show json_read["user.json"] .
```

**Output**

```text
{ "name": "Ann" }
```

### `json_write[path, value]`

Saves a value to a file as indented JSON.

**Example**

```turtle
json_write["out.json", list [1, 2]]
show json_read["out.json"] .
```

**Output**

```text
[ 1, 2 ]
```

### `json_get[value, step, step, ...]`

Follows a path of map keys and list positions into a value.

**Example**

```turtle
config = load['{"server": {"port": 8080}}']
port = json_get[config, "server", "port"]
show port .
```

**Output**

```text
8080
```
