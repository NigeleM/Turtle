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

**Turtle: Restart Language Server** (in the command palette) restarts it
after updating turtle. Messages from the server are in the Output panel,
under Turtle.

## Checking the extension

`test/run.sh` runs `extension.js` against stand-ins for VS Code and for
starting `turtle lsp` (turtle missing, an old turtle, a working server)
and running a file, and prints what the user would see. On macOS it uses the built-in
JavaScript engine; elsewhere, Node.js.
