# Files, shell commands and modules

Read, write and append files, list folders, run shell commands with `sys`, and import your own modules. No import is needed.


`import` is not needed for these.

## Statements

`[read]`, `[write]`, `[append]`, `[directory]`, `sys`, and `import`.

### `[read] path to lines [end]`

Reads the whole file into a list of text, one item per line.

**Example**

```turtle
[write] notes.txt
"first line"
"second line"
[end]
[append] notes.txt
"third line"
[end]
[read] notes.txt to lines [end]
show lines at get[2] .
```

**Output**

```text
third line
```

### `[write] path` and `[append] path`

`[write]` replaces the file. `[append]` adds to the end, or creates the file. Each line between them is a quoted text or a variable name.

**Example**

Shown above, in the `[read]` example.

### `[directory] path to names [end]`

Lists the names in a folder, into a list.

**Example**

```turtle
[directory] . to names [end]
show length of names > 0 .
```

**Output**

```text
true
```

### `sys command`

Runs a command in the shell. The rest of the line is the command.

**Example**

```turtle
sys echo from the shell
```

**Output**

```text
from the shell
```

### `import name`

Brings in your own `.turtle` file. Its functions are called with its name in front, or as listed.

**Example**

With `lib/utils.turtle` holding `def half[n]` that returns `n / 2`:

```turtle
import lib/utils
show utils half[4] .
```

**Output**

```text
2.0
```

Paths are relative to the folder you run `turtle` from. For files next to your program, see `scriptfolder[]` in the [system](system.md) library.
