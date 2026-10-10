# The REPL

Run `turtle` with no file to get Turtle's interactive prompt: type a line,
press Enter, see the result. The code is colored as you type.

```
$ turtle
Turtle v0.9.166 — type help for help, quit to leave
>>> nums = list [3, 1, 2]
>>> nums at length * 2
6
>>> def double[x]
...     return x * 2
... def [end]
>>> double[21]
42
>>> 1 div 0
error (math): division by zero
>>> quit
```

- **A lone expression or call shows its value** (`nums`, `1 + 2`,
  `double[21]`), text in quotes: `"a" + "b"` shows `"ab"`. Anything
  else runs as in a program (`show nums .` prints `[ 3, 1, 2 ]`).
- **A block waits for its end.** After a line that opens one (`def`,
  `if ] ... [`, `[loop][...]`, `safe`, a `give` function whose body is on
  the next lines, `[write] ...`), the prompt becomes `...` and the next
  line is indented for you; the entry runs once the block's end line
  (`def [end]`, `if [end]` ...) is typed. A `validate` sentence waits for
  its period. An empty line runs an unfinished entry anyway (to see what's
  wrong with it).
- **Errors don't end the session**: you see `error (kind): message`, and
  everything you defined is still there.
- **Ctrl-C** cancels the line being typed, or stops code that's running
  (an endless loop, say) and comes back to `>>>`.
- **Imports carry on**: after `import random`, `random integer from 1 to
  6` works on later lines, and so do `check` after `import test` and `log`
  after `import log`.

## Colors

Colors come from Turtle's own lexer, so they always match the language:

They're grouped by what a word does, as in the VS Code extension's
Turtle scheme:

| Color | What |
|---|---|
| green | keywords: `def`, `if`, `loop`, `at`, `div` ..., and a library's words once imported (`random`, `check`, `log`) |
| teal | an `import` line, all of it |
| blue | functions: calls `name[...]`, library functions, methods after `at`; bold where defined, bold italic for a `test_` function |
| amber | data structures: `list`, `set`, `map`, `matrix`, `assemble` and your types (bold where made) |
| bold violet | `show` and `warn` |
| red | text |
| pink | numbers, `true`, `false`, `none` |
| bold italic gold | theories: `theory`, its sections, and its word where it's used |
| italic gray | comments |

They work in macOS Terminal and iTerm, Linux terminals, and Windows
Terminal / PowerShell on Windows 10 and later. Set the environment
variable `NO_COLOR` (to anything) for no colors.

## Keys

| Keys | Do |
|---|---|
| Left / Right, Ctrl-B / Ctrl-F | move |
| Ctrl-Left / Ctrl-Right, Alt-B / Alt-F | move a word |
| Home / End, Ctrl-A / Ctrl-E | start / end of the line |
| Up / Down, Ctrl-P / Ctrl-N | earlier / later lines (history) |
| Backspace, Delete | delete a character |
| Ctrl-W, Alt-Backspace | delete the word before the cursor |
| Ctrl-K / Ctrl-U | delete to the end / start of the line |
| Tab | indent (four spaces) |
| Ctrl-L | clear the screen |
| Ctrl-C | cancel the line, or stop running code |
| Ctrl-D | leave (on an empty line) |

History is kept between sessions in `~/.turtle_history` (the last 1,000
lines).

## Commands

A line with just one of these words (and no variable or function of
that name) is a command:

| Command | Does |
|---|---|
| `help` | the commands and keys |
| `help linear` | a library: what it's for, and each function in a line |
| `help reshape` | one function or method in full, with an example |
| `help m` | what your variable `m` is, and its methods and libraries |
| `help list` | a kind of value's methods (`list`, `set`, `map`, `string`, `integer`, `float`, `matrix`) |
| `help if` | a keyword; `help keywords` lists them all |
| `quit` (or `exit`, or Ctrl-D) | leave |
| `clear` | clear the screen |
| `names` | list your variables and functions |
| `load file.turtle` | run a file into this session: its functions and values are then yours to use |
| `save file.turtle` | write the entries you've run (the ones without errors) to a file |

`exit[code]` from `import system` leaves too. The same help works in a
program as `help["linear"]` or `help[m]`; see
[Finding things out](reference.md#finding-things-out-help-stdlib-version).

## Not a terminal

With input piped in, `turtle` runs it as a program, the way `python`
does: `echo 'show 1 + 2 .' | turtle` prints `3`.
