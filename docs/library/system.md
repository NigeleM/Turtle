# `system` — the command line, environment and files

`import system`

The command line, environment and files: read arguments and environment variables, check and copy files, and pack or unpack archives.


## Functions

`args`, `exists`, `isfile`, `isfolder`, `exit`, `env`, `scriptfolder`, `contents`, `erase`, `warn`, `copyto`, `moveto`, `makefolder`, `walk`, `pack`, `unpack`, `loadenv`, `options`, `memory`, `sizeof`, `freememory`.

### `args[]`

The words typed after the script's name: turtle report.turtle a.txt b.txt Gives back: a list of text (empty if none).

**Example**

```turtle
show args[] .
```

**Output**

```text
[]
```

### `exists[path]`

Is there a file or folder at path?

**Example**

```turtle
show exists["notes.txt"] .
```

**Output**

```text
false
```

### `isfile[path]`

Is path a file?

**Example**

```turtle
show isfile["notes.txt"] .
```

**Output**

```text
false
```

### `isfolder[path]`

Is path a folder?

**Example**

```turtle
makefolder["out"]
show isfolder["out"] .
```

**Output**

```text
true
```

### `exit[code]`

Ends the program now. code is optional: 0 (the default) means it went well, anything else that it failed.

**Example**

```turtle
exit[1]
```

This example stops with an error on purpose or when run alone.

### `env[name]`

An environment variable's value.

**Example**

```turtle
home = env["HOME"]
```

**Output**

```text
(prints nothing)
```

### `scriptfolder[]`

The full path of the folder the running script is in.

**Example**

```turtle
data = "{scriptfolder[]}/data.csv"
```

**Output**

```text
(prints nothing)
```

### `contents[path]`

The names of the files and folders in a folder (path is optional, "." by default).

**Example**

```turtle
makefolder["out"]
[write] out/a.txt
"one"
[end]
show contents["out"] .
```

**Output**

```text
[ "a.txt" ]
```

### `erase[path]`

Deletes a file, or a folder and everything in it.

**Example**

```turtle
makefolder["out"]
erase["out"]
show exists["out"] .
```

**Output**

```text
false
```

### `warn ... .`

Like show, but to standard error, so it isn't mixed with output that's piped or saved.

**Example**

```turtle
import system
name = "report.csv"
warn "can't read ", name .
```

**Output**

```text
can't read report.csv
```

### `copyto[from, to, replace]`

Copies a file, or a folder and everything in it, to the path to (missing folders on the way are made). replace    optional: true replaces what's already at to; without it, something there is an error

**Example**

```turtle
[write] report.csv
"data"
[end]
copyto["report.csv", "backup/report.csv"]
show exists["backup/report.csv"] .
```

**Output**

```text
true
```

### `moveto[from, to, replace]`

Moves (or renames) a file or folder; replace as for copyto.

**Example**

```turtle
[write] old.txt
"data"
[end]
moveto["old.txt", "archive/old.txt"]
show exists["archive/old.txt"] .
```

**Output**

```text
true
```

### `makefolder[path]`

Makes a folder, and any missing folders above it.

**Example**

```turtle
show makefolder["out/2026/october"] .
```

**Output**

```text
none
```

### `walk[folder]`

Every file under the folder, in its subfolders too, sorted; each path starts with folder and uses /. "." if left out.

**Example**

```turtle
makefolder["src"]
[write] src/a.txt
"text"
[end]
[loop][f in walk["src"]]
    show f .
[loop][end]
```

**Output**

```text
src/a.txt
```

### `pack[from, archive, replace]`

Puts a file, or a folder and everything in it, into an archive.

**Example**

```turtle
makefolder["src"]
[write] src/a.txt
"text"
[end]
pack["src", "src.zip"]
unpack["src.zip", "restored"]
show walk["restored"] .
```

**Output**

```text
[ "restored/src/a.txt" ]
```

### `unpack[archive, folder, replace]`

Puts an archive's files into folder.

**Example**

```turtle
makefolder["src"]
[write] src/a.txt
"text"
[end]
pack["src", "src.zip"]
unpack["src.zip", "restored"]
show walk["restored"] .
```

**Output**

```text
[ "restored/src/a.txt" ]
```

### `loadenv[file]`

Reads a .env file (".env" if left out): KEY=value lines, # comments, "quoted" or 'quoted' values.

**Example**

```turtle
[write] app.env
"KEY=value"
[end]
settings = loadenv["app.env"]
show settings at get "KEY" .
```

**Output**

```text
value
```

### `options[name: default, ...]`

Named options from the command line.

**Example**

```turtle
opts = options["--out": "result.csv", "-v": false]
out = opts at get["--out"]
```

**Output**

```text
(prints nothing)
```

### `memory[]`

How much memory the program's values hold now, in bytes.

**Example**

```turtle
show memory[] / 1048576, " MB" .
```

**Output**

```text
0.44379425048828125 MB
```

### `sizeof[x]`

About how many bytes x takes, with everything in it (an estimate: a value inside it twice counts once).

**Example**

```turtle
import test
rows = list [1, 2, 3]
check sizeof[rows] < 5000000 .
```

**Output**

```text
(prints nothing)
```

### `freememory[]`

Collects the values nothing refers to now, and gives that memory back to the system: worth it between big steps, not after every small one.

**Example**

```turtle
rows = none
freed = freememory[]
```

**Output**

```text
(prints nothing)
```
