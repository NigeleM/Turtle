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

Colors come from Turtle's own lexer, so they always match the language.
They're grouped by what a word does, in the same twelve schemes as the
VS Code extension, color for color:

| Group | What |
|---|---|
| keywords | `def`, `if`, `loop`, `at`, `div` ..., and a library's words once imported (`random`, `check`, `log`) |
| imports | an `import` line, all of it |
| functions | calls `name[...]`, library functions, methods after `at`; bold where defined, bold italic for a `test_` function |
| data | `list`, `set`, `map`, `matrix`, `assemble` and your types (bold where made) |
| show | `show` and `warn` |
| text | `"text"` |
| numbers | numbers, `true`, `false`, `none` |
| theories | `theory`, its sections, and its word where it's used |

Comments are italic gray in every scheme, and a value the REPL shows
takes the functions color.

### Picking a scheme

Type `colors` for the picker: the schemes down the left, and a short
program on the right in the colors of the one the cursor is on.

```
>>> colors
  shade: ← auto (dark) →

  turtle (now)        │  import time [today]
  classic             │  assemble Order [item, qty]
> ocean               │
  sunset              │  // the items, and how many
  forest              │  def total[orders]
  soft                │      sum = 0
  dusk                │      [loop][o in orders]
  okabe-ito           │          sum = sum + o at qty
  blue-orange         │      [loop][end]
  teal-rose           │      show "total: " . sum
  high-contrast       │      return tally orders
  no-color            │  def [end]

  ↑↓ scheme   ←→ shade   Enter keeps it   Esc leaves it
```

| Keys | Do |
|---|---|
| Up / Down | the scheme; the program recolors as you go |
| Left / Right | the shade: `auto`, `dark`, `light` |
| Enter | keep it (and for next time) |
| Esc, `q` or Ctrl-C | leave the colors as they were |

Or name it, with no picker:

```
>>> colors okabe-ito
colors: okabe-ito, dark (auto)
>>> colors classic light
colors: classic, light
>>> colors auto
colors: classic, dark (auto)
```

Without keys to press (input piped in, or a window narrower than the
picker's 55 columns) `colors` lists the schemes instead, each in its own
colors.

- **The names** are VS Code's, in lowercase with a dash for spaces and
  `&`: *Okabe-Ito* is `okabe-ito`, *Blue & Orange* is `blue-orange`,
  *High Contrast* is `high-contrast`. (VS Code's *Theme colors* has no
  REPL twin: your terminal's theme doesn't reach Turtle.) See the
  [extension's README](../editors/vscode/README.md) for what each scheme
  is for.
- **Light or dark.** Each scheme has a shade for dark backgrounds and one
  for light. On `auto` (the default) the REPL asks the terminal for its
  background color when it starts, or reads `COLORFGBG` if the terminal
  sets it, and picks the shade that reads. Terminals that don't answer
  (and Windows, which isn't asked) get dark; `colors light` fixes that.
  `colors light`, `colors dark` and `colors auto` change only the shade.
- **It's kept** in `~/.config/turtle/repl`, one line (`okabe-ito auto`),
  and used every time the REPL starts. Delete the file to go back to
  `turtle auto`.
- **Exact colors** where the terminal shows 24-bit color (it sets
  `COLORTERM=truecolor`: iTerm2, VS Code's terminal, Ghostty, WezTerm,
  Windows Terminal ...); elsewhere (macOS Terminal) the closest of the
  256 colors every terminal has.
- **To match VS Code**, pick the same scheme in both: **Turtle: Color
  Scheme** in VS Code's settings, `colors` in the REPL. The REPL doesn't
  read VS Code's settings; each keeps its own.

They work in macOS Terminal and iTerm, Linux terminals, and Windows
Terminal / PowerShell on Windows 10 and later. Set the environment
variable `NO_COLOR` (to anything) for no colors at all; `colors` still
records a choice for when they're back.

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
| `colors` | pick a color scheme: Up/Down the scheme, Left/Right the shade, Enter keeps it ([Colors](#picking-a-scheme)) |
| `colors okabe-ito` | use a scheme; add `light`, `dark` or `auto` for the shade. Kept for next time |

`exit[code]` from `import system` leaves too. The same help works in a
program as `help["linear"]` or `help[m]`; see
[Finding things out](reference.md#finding-things-out-help-stdlib-version).

## Not a terminal

With input piped in, `turtle` runs it as a program, the way `python`
does: `echo 'show 1 + 2 .' | turtle` prints `3`.

---

Copyright 2017-2026 Nigele McCoy. Licensed under the
[Apache License 2.0](../LICENSE); see [NOTICE](../NOTICE).
