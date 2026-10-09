# Files, shell commands and modules

[Library index](index.md)

Reading and writing files, running shell commands, and importing your own modules. No import needed.

## Files

Paths are quoted strings or barewords (`file.txt`, `data/in.csv`),
resolved relative to the folder `turtle` was run in, like any command-line
tool (use `system`'s `scriptfolder[]` for files next to the script). A
single bare word with no `.` or `/` uses the variable of that name if one
exists. Any I/O failure (file not found, permission denied, etc.) is a
fatal error. Check first with `system`'s `exists[path]`.

```
[read] <path> to <ident> [end]
```

Reads the whole file, splits on newlines, assigns the result as a `list`
of strings (one per line) to `<ident>`.

```
[write] <path>
<line>
...
[end]

[append] <path>
<line>
...
[end]
```

Each `<line>` is either a quoted string (written verbatim) or a bare
identifier (replaced with that variable's current display value). Lines
are newline-joined and written with a trailing newline. `[write]` creates
or overwrites the file; `[append]` creates it if missing, otherwise adds to
the end.

```
[directory] <path> to <ident> [end]
```

Lists the directory's entries (names only, not full paths) into `<ident>`
as a `list`.

## Shell escape: `sys`

```
sys <rest of line>
```

`sys` must be the very first word of the statement — this is a real
keyword, not a substring match, so it only triggers there (unlike the
legacy interpreter, where any line merely *containing* "sys" anywhere
would misfire into shell execution). Everything after it, verbatim to the
end of the line, is passed to `sh -c` (`cmd /c` on Windows, with any
quotes in the command reaching `cmd` as written). It runs in the folder turtle was
run in, where file paths resolve too. The child process inherits stdin,
stdout, and stderr; its exit status is not checked or reported back to
the Turtle program (`runall` in `import schedule` gives back the output
and exit code).

This is a deliberately dangerous, unsandboxed feature — the Turtle
equivalent of Python's `os.system`. Only use it with trusted script
content.

## Modules: `import`

```
import <name>
```

```
import <name> [<f>, <g>]
```

Reads `<name>.turtle` (relative to the script's own folder, wherever `turtle`
was run from) and runs it
once, in its own scope. Its top-level functions become available to your
program: all of them, or only the ones listed in `[...]`. Its top-level
variables stay private to it. When two imports export the same function
name, call it qualified by module, as in `mylib now[]` or `time now[]`.
An unqualified call to a name that clashes is a fatal error. Full rules
are in [`reference.md`](../reference.md#modules).
