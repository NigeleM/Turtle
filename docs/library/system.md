# `system` — the command line, environment and files

[Library index](index.md) · `import system`

Arguments, environment variables, files and folders, and the program itself.


`import system` (or `import system [args, exists]`):

| Function | Args | Returns |
|---|---|---|
| `args[]` | — | a `list` of the command-line arguments after the script path, as strings (empty if none) |
| `exists[path]` | path string | Boolean: is there a file or folder at `path` |
| `isfile[path]` | path string | Boolean: is `path` a regular file |
| `isfolder[path]` | path string | Boolean: is `path` a folder |
| `exit[code]` | optional integer (default `0`) | ends the program immediately with that exit code; `0` means success, anything else failure |
| `env[name]` | variable name string | the environment variable's value as a string, or `none` if it isn't set |
| `scriptfolder[]` | — | the full path of the folder the running script is in |
| `erase[path]` | path string | deletes the file, or the folder and **everything in it** (no undo); returns `none`. A missing path is a `file` error. It refuses the folder `turtle` runs in and any folder above it |
| `contents[path]` | optional folder path (default `"."`) | a `list` of the names of the files and folders inside, sorted (names only, not full paths); a missing folder is a fatal error. The same listing as the `[directory]` statement, usable inline |
| `walk[folder]` | optional folder path (default `"."`) | a `list` of every **file** under the folder, in its subfolders too, sorted. Each path starts with `folder` and uses `/` on every system (`"src/sub/b.txt"`), so it can go straight to `[read]` or `copyto` |
| `copyto[from, to, replace]` | paths, optional Boolean | copies a file, or a folder and everything in it, to exactly `to`, making missing folders on the way. Something already at `to` is a `file` error unless `replace` is `true`. Copying a folder into itself is refused |
| `moveto[from, to, replace]` | paths, optional Boolean | moves or renames a file or folder, the same rules as `copyto`; works across disks. It refuses to move the folder `turtle` runs in or one above it |
| `makefolder[path]` | path string | makes the folder and any missing folders above it; fine if it's already there |
| `pack[from, archive, replace]` | paths, optional Boolean | puts a file, or a folder and everything in it, into an archive whose kind comes from its name: `.zip`, `.tar`, `.tar.gz` or `.tgz`. A folder goes in under its own name |
| `unpack[archive, folder, replace]` | paths, optional Boolean | puts the archive's files into `folder` (made if missing). A file already there is a `file` error unless `replace` is `true`; an entry that would land outside `folder` (`../x`) is refused |
| `loadenv[file]` | optional path (default `".env"`) | reads `KEY=value` lines into a `map` (values are text) and sets each for `env[...]`, unless the environment already has it |
| `options[name: default, ...]` | option names and defaults | reads named options from the command line into a `map`; see below |
| `memory[]` | — | how many bytes the program's values hold now (it collects first, so values already let go don't count) |
| `sizeof[x]` | any value | about how many bytes `x` takes, with everything in it; an estimate, and a value in it twice counts once |
| `freememory[]` | — | collects what nothing refers to now and gives that memory back to the system; gives back the bytes freed |

**Files and folders:**

```
import system

copyto["report.csv", "backup/report.csv"]
moveto["old.txt", "archive/old.txt"]     // or rename in place
makefolder["out/2026/october"]
[loop][f in walk["src"]]
    show f .                             // src/a.txt, src/sub/b.txt, ...
[loop][end]

pack["src", "src.zip"]                   // src.zip holds src/...
unpack["src.zip", "restored"]            // restored/src/...
copyto["src", "backup", true]            // true: replace the old backup
```

**`.env` files** hold settings, often secrets, outside the code: one
`KEY=value` per line, `#` comments, values in `"double"` quotes (with
`\n`, `\"`) or `'single'` quotes (as typed), an optional `export ` in
front. What's already set in the environment wins over the file, as with
every `.env` tool, so a server's real settings aren't overridden.

```
# .env
API_KEY=abc123
NAME="Ann Lee"
```

```
settings = loadenv[]                     // { "API_KEY": "abc123", "NAME": "Ann Lee" }
key = env["API_KEY"]
```

**Memory.** Turtle frees a value by itself once nothing refers to it:
a function's own values when it's done, a variable's old value when it
changes. Go collects them in batches, though, so between big steps (a
whole file loaded, then a report built) `freememory[]` collects now and
hands the memory back. Set what you're done with to `none` first; a value
a variable still holds can't be freed. `diagnose`, `hypothesis` and
`turtle test` show how much each step allocated.

```
import system

rows = table_read["sales.csv"]
show sizeof[rows] / 1048576, " MB" .
count = length of rows
rows = none                              // done with the rows
freed = freememory[]
show memory[] / 1048576, " MB held" .
```

**Named options.** `options` takes each option with its default, written
as `name: default` pairs, and gives back a map of every option's value.
The default sets the option's kind:

| Default | Typed as | Value |
|---|---|---|
| `false` (or `true`) | `-v` alone, or `-v=false` | an on/off switch: `true` when given |
| an integer / a float | `--count 5` or `--count=5` | a number; anything else is an error |
| text | `--out file.csv` or `--out=file.csv` | the text |

What isn't an option (`data.csv`) stays for `args[]`, which then gives
only those. `--` ends the options: everything after it goes to `args[]`.
An option that isn't listed is an error naming the ones that are.
`--help` (or `-h`) shows the options and ends the program, unless you
list your own.

```
// turtle report.turtle data.csv --out result.csv -v
import system

opts = options["--out": "result.csv", "-v": false, "--count": 10]
out = opts at get["--out"]                // "result.csv"
loud = opts at get["-v"]                  // true
files = args[]                            // [ "data.csv" ]
```

```
$ turtle report.turtle --help
usage: turtle report.turtle [options]

options:
  --out    default result.csv
  -v       on or off (off unless given)
  --count  default 10
```

`warn <expr> {, <expr>} .` is `show` for stderr: same pieces, same
period, but the line goes to standard error, so it isn't mixed into
output that's piped or saved to a file. It needs `import system` (or
`import system [warn]`).

```
import system
warn "can't read ", name, ", skipping" .
erase["build"]
```

**Paths resolve from the folder you ran `turtle` in**, like any
command-line tool, unless absolute. That applies here and to `[read]`,
`[write]`, `[append]` and `[directory]`. So
`turtle ~/tools/count.turtle notes.txt` reads `./notes.txt`. (`import` is
different: it always looks next to the script, so a program and its
libraries can be moved together.) To use a file that sits next to the
script, build its path from `scriptfolder[]`:

```
import system
config = scriptfolder[] + "/config.txt"
[read] config to lines [end]
```

A complete tool, with usage message and exit codes:

```
// turtle count.turtle notes.txt
import system
import strings

a = args[]
if ] length of a == 0 [
    show "usage: count.turtle <file>" .
    exit[2]
if [end]
name is a at get 0 .
if ] !name exists [
    show "missing: " + name .
    exit[1]
if [end]
[read] name to lines [end]
show name, ": ", length of lines, " lines" .
show lines join " | " .
```

Listing a folder:

```
import system
show contents[] .                    // the folder turtle was run in
[loop][name in contents["sub"]]
    if ] isfolder["sub/" + name] [
        show name, "/" .
    else ]
        show name .
    if [end]
[loop][end]
```

Check a file before reading it, since a missing file is a fatal error for
`[read]`:

```
// turtle tool.turtle notes.txt sub missing.txt
import system

[loop][name in args[]]
    if ] name exists [
        if ] isfolder[name] [
            show name, " is a folder" .
        else ]
            show name, " is a file" .
        if [end]
    else ]
        show name, " does not exist" .
    if [end]
[loop][end]
```
