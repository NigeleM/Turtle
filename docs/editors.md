# Editors

`turtle lsp` is a language server: editors that speak the Language Server
Protocol get Turtle support from it.

| What you get | |
|---|---|
| Errors as you type | the parser's own messages, on the line; imports of missing files, as warnings |
| Colors | keywords, strings, numbers, comments, definitions, calls, library functions, from Turtle's lexer (the same as the REPL) |
| Completion | keywords, your functions, assembled types and variables, functions from your imported files, library functions (an imported library's first), methods after `at`, libraries and your files after `import`, kinds of value after `random` |
| Hover | your function's description (comments above its `def` or first in its body, `//` lines or a `//* *//` block), a library function's `turtle doc` text, a keyword's meaning |
| Go to definition | your functions and assembled types, also in imported files |
| Outline | the file's functions, assembled types and top-level variables |

It's built into `turtle` (no extra install) and uses only Go's standard
library. Files must end in `.trt`.

## VS Code

Install the extension in [`editors/vscode`](../editors/vscode/README.md)
(copy the folder into `~/.vscode/extensions/turtle`). It adds the language,
a color grammar, comment toggling (Ctrl-/), bracket matching and
indenting, and starts `turtle lsp`.

## Neovim (0.11 and later)

```lua
vim.filetype.add({ extension = { trt = "turtle" } })
vim.lsp.config("turtle", { cmd = { "turtle", "lsp" }, filetypes = { "turtle" }, root_markers = { ".git" } })
vim.lsp.enable("turtle")
```

## Helix

In `~/.config/helix/languages.toml`:

```toml
[language-server.turtle]
command = "turtle"
args = ["lsp"]

[[language]]
name = "turtle"
scope = "source.turtle"
file-types = ["trt"]
comment-token = "//"
language-servers = ["turtle"]
```

## Sublime Text

With the LSP package, in Preferences → Package Settings → LSP → Settings:

```json
{
  "clients": {
    "turtle": {
      "enabled": true,
      "command": ["turtle", "lsp"],
      "selector": "source.turtle"
    }
  }
}
```

(Sublime needs a syntax for `source.turtle`; the VS Code grammar,
`editors/vscode/syntaxes/turtle.tmLanguage.json`, works as a
`.tmLanguage` file.)

## Emacs

With eglot (built in since Emacs 29):

```elisp
(define-derived-mode turtle-mode prog-mode "Turtle")
(add-to-list 'auto-mode-alist '("\\.trt\\'" . turtle-mode))
(add-to-list 'eglot-server-programs '(turtle-mode "turtle" "lsp"))
```

## Others

Any editor with an LSP client works: run `turtle lsp` for files ending in
`.trt`. It talks on stdin and stdout, sends whole-file sync, and answers
initialize, didOpen / didChange / didClose, completion, hover, definition,
documentSymbol and semanticTokens/full.
