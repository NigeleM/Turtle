# Turtle for VS Code

Run a file with the ▶ button at the top right (or Ctrl-F5, or **Turtle:
Run File** in the command palette): it saves the file and runs it in a
Turtle terminal, in the file's own folder, so `?` input works there too.

Colors for `.turtle` (and `.trt`) files, and through `turtle lsp`: errors as you type,
completion, hover help (your functions' `//` comments and the standard
library's docs), go to definition (F12), the outline, and Format Document (Shift-Alt-F, the
same as `turtle fmt`). No npm packages:
the extension is three JSON files and one small JavaScript file.

## Install

You need `turtle` installed (`turtle version` in a terminal should work).

**Copy the folder** into VS Code's extensions folder, then restart VS Code:

```sh
# macOS / Linux
cp -r editors/vscode ~/.vscode/extensions/turtle

# Windows (PowerShell)
Copy-Item -Recurse editors\vscode "$env:USERPROFILE\.vscode\extensions\turtle"
```

Or build a `.vsix` to share (needs Node.js once, for the packaging tool):

```sh
cd editors/vscode
npx @vscode/vsce package
code --install-extension turtle-*.vsix      # the file vsce just made
```

## Settings

| Setting | Default | Means |
|---|---|---|
| `turtle.path` | `turtle` | the turtle command; a full path if it isn't on your PATH |
| `turtle.languageServer` | `true` | run `turtle lsp`; colors work without it |
| `turtle.colorScheme` | `Turtle` | how Turtle files are colored (below) |
| `turtle.colors` | `{}` | your own color for any group, over the scheme's |

## Colors

Turtle colors words by what they do, in eight groups:

| Group | Words |
|---|---|
| keywords | `if`, `loop`, `def`, `return`, `safe`, `give`, `is`, `of`, `at`, ... |
| imports | the whole `import` line: `import time [now, today]` |
| functions | function names (bold where defined; a `test_` function bold and italic), calls, methods after `at` |
| data | `list`, `set`, `map`, `matrix`, `assemble` and your types (bold where made) |
| show | `show` and `warn` |
| text | `"text"`, `'text'`, backtick text |
| numbers | numbers, `true`, `false`, `none` |
| theories | `theory`, its sections, and the theory's word wherever it's used |

**Turtle: Color Scheme** in the settings picks one of twelve. Each has a
shade for dark themes and one for light, chosen when you switch themes,
and every color has 4.5:1 contrast or more on its background.

- **Traditional:** Turtle (the default), Classic (close to VS Code's own
  colors), Ocean, Sunset, Forest.
- **Easy on the eyes:** Soft, Dusk.
- **Color vision:** Okabe-Ito (the palette scientists use, readable with
  every common kind of color blindness), Blue & Orange (red-green color
  blindness), Teal & Rose (blue-yellow), High Contrast (low vision: 7:1),
  No Color (brightness and style alone). These also use bold, italic and
  underline, so no group depends on color alone.
- **Theme colors:** your theme's own colors, nothing from Turtle.

The scheme colors Turtle files only: other languages keep your theme. It's
written into your user settings (`editor.tokenColorCustomizations` and
`editor.semanticTokenColorCustomizations`), as rules for Turtle alone;
choose **Theme colors** before uninstalling to take them out.

To change one group, set it in `turtle.colors`:

```json
"turtle.colors": { "theories": "#E5484D", "keywords": "#3FB950" }
```

The groups are `keywords`, `imports`, `functions`, `data`, `show`,
`text`, `numbers` and `theories`.

**Turtle: Restart Language Server** (in the command palette) restarts it
after updating turtle. Messages from the server are in the Output panel,
under Turtle.

## Checking the extension

`test/run.sh` runs `extension.js` against stand-ins for VS Code and for
starting `turtle lsp` (turtle missing, an old turtle, a working server)
and running a file, and prints what the user would see. On macOS it uses the built-in
JavaScript engine; elsewhere, Node.js.
