# Editors

`turtle lsp` is a language server: editors that speak the Language Server
Protocol get Turtle support from it.

| What you get | |
|---|---|
| Errors as you type | the parser's own messages, under the spot; imports of missing files, as warnings |
| Colors | keywords, strings, numbers, comments, definitions, calls, library functions, from Turtle's lexer, in twelve schemes; the REPL has the same ones (`colors` there) |
| Completion | keywords, your functions, assembled types and variables, functions from your imported files, library functions (an imported library's first), methods after `at`, libraries and your files after `import`, kinds of value after `random` |
| Hover | your function's description (comments above its `def` or first in its body, `//` lines or a `//* *//` block), a library function's `turtle doc` text, a keyword's meaning |
| Go to definition | your functions and assembled types, also in imported files |
| Outline | the file's functions, assembled types and top-level variables |
| Format Document | `turtle fmt`'s layout |

It's built into `turtle` (no extra install). Files end in `.turtle` (or `.trt`, the earlier ending).

## VS Code

Install the extension in [`editors/vscode`](../editors/vscode/README.md)
(copy the folder into `~/.vscode/extensions/turtle`). It adds the language,
a color grammar, comment toggling (Ctrl-/), bracket matching and
indenting, a ▶ button that runs the open file (Ctrl-F5), and starts
`turtle lsp`.

## Neovim (0.11 and later)

```lua
vim.filetype.add({ extension = { turtle = "turtle", trt = "turtle" } })
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
file-types = ["turtle", "trt"]
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
(add-to-list 'auto-mode-alist '("\\.turtle\\'" . turtle-mode))
(add-to-list 'eglot-server-programs '(turtle-mode "turtle" "lsp"))
```

## Others

Any editor with an LSP client works: run `turtle lsp` for files ending in
`.turtle`. It talks on stdin and stdout, sends whole-file sync, and answers
initialize, didOpen / didChange / didClose, completion, hover, definition,
documentSymbol and semanticTokens/full.

---

Copyright 2017-2026 Nigele McCoy. Licensed under the
[Apache License 2.0](../LICENSE); see [NOTICE](../NOTICE).

## Example: a session with the language server

`turtle lsp` speaks the Language Server Protocol over standard input and output, so an editor starts it for you. This is the exchange an editor makes: it asks the server to start, and then asks it to shut down. Each message has a `Content-Length` header. The server's reply to `initialize` lists what it can do, such as completion, hover, formatting and semantic tokens.

```text
$ turtle lsp   (messages sent on standard input)
Content-Length: 564

{"id":1,"jsonrpc":"2.0","result":{"capabilities":{"completionProvider":{"triggerCharacters":[]},"definitionProvider":true,"documentFormattingProvider":true,"documentSymbolProvider":true,"hoverProvider":true,"positionEncoding":"utf-16","semanticTokensProvider":{"full":true,"legend":{"tokenModifiers":["declaration","defaultLibrary","test"],"tokenTypes":["keyword","string","number","comment","function","type","namespace","struct","event","macro","method"]}},"textDocumentSync":{"change":1,"openClose":true}},"serverInfo":{"name":"turtle","version":"v1.0.0-dev"}}}Content-Length: 38

{"id":2,"jsonrpc":"2.0","result":null}
```
