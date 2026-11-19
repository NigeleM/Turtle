# `config` — settings files

`import config`

Read and write settings files, so a program can keep its options outside its code. Use it when a value should change without editing the program.


## Functions

`config_read`, `config_write`.

### `config_read[path]`

Reads a settings file. path      a .toml, .json or .env file Gives back: a map ([sections] and objects as maps inside it; a .env file's values are text).

**Example**

```turtle
settings = map ["server": map ["port": 8080]]
config_write["app.toml", settings]
settings = config_read["app.toml"]
show settings at get "server" .
```

**Output**

```text
{ "port": 8080 }
```

### `config_write[path, settings]`

Writes a map as a settings file, by the file's extension.

**Example**

```turtle
settings = map ["server": map ["port": 8080]]
config_write["app.toml", settings]
show "saved" .
```

**Output**

```text
saved
```
