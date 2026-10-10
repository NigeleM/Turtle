// The Turtle extension for VS Code. Colors come from the grammar
// (syntaxes/turtle.tmLanguage.json) and need nothing else; the color
// scheme (turtle.colorScheme) picks them, for Turtle files only. Everything
// else (errors as you type, completion, hover help, go to definition, the
// outline, Format Document, exact colors) comes from turtle lsp, which this file starts and
// talks to: a small Language Server Protocol client written against VS
// Code's own API, with no npm packages. Turtle: Run File (the ▶ button)
// runs the open file in a terminal.

"use strict";

const vscode = require("vscode");
const { spawn } = require("child_process");
const path = require("path");

let server = null; // the running turtle lsp
let active = []; // what start registered, undone on restart

// ---- JSON-RPC over the server's stdin and stdout ----

class Connection {
  constructor(command, output) {
    this.output = output;
    this.nextId = 1;
    this.pending = new Map(); // id -> {resolve, reject}
    this.handlers = new Map(); // notification method -> function
    this.buffer = Buffer.alloc(0);
    this.ready = false; // set once the server has answered initialize
    this.lastError = ""; // the last thing it wrote to stderr
    this.process = spawn(command, ["lsp"], { stdio: ["pipe", "pipe", "pipe"] });
    this.process.stdout.on("data", (chunk) => this.receive(chunk));
    this.process.stderr.on("data", (chunk) => {
      output.append(chunk.toString());
      const text = chunk.toString().trim();
      if (text) this.lastError = text.split("\n").pop();
    });
    this.process.on("error", (err) => {
      this.reported = true;
      output.appendLine(`turtle lsp didn't start: ${err.message}`);
      const missing = err.code === "ENOENT";
      showProblem(
        output,
        missing
          ? `Turtle: no "${command}" command found. Install turtle (github.com/NigeleM/Turtle/releases), or set turtle.path to it. Colors still work without it.`
          : `Turtle: couldn't run "${command} lsp" (${err.message}). Colors still work.`
      );
      this.fail(err);
    });
    this.process.on("exit", (code) => {
      output.appendLine(`turtle lsp stopped (${code})`);
      if (!this.ready && !this.reported && !this.stopping) {
        // It ran but never started up: most likely a turtle from before
        // turtle lsp existed (v0.9.151), which takes "lsp" for a file.
        const said = this.lastError ? ` It said: "${this.lastError}".` : "";
        showProblem(
          output,
          `Turtle: "${command} lsp" stopped before starting.${said} Is turtle up to date? "turtle version" should say v0.9.151 or later. Colors still work without it.`
        );
      }
      this.fail(new Error("turtle lsp stopped"));
    });
  }

  fail(err) {
    for (const p of this.pending.values()) p.reject(err);
    this.pending.clear();
    this.closed = true;
  }

  send(message) {
    if (this.closed) return;
    const body = Buffer.from(JSON.stringify(Object.assign({ jsonrpc: "2.0" }, message)), "utf8");
    this.process.stdin.write(`Content-Length: ${body.length}\r\n\r\n`);
    this.process.stdin.write(body);
  }

  request(method, params) {
    if (this.closed) return Promise.reject(new Error("turtle lsp isn't running"));
    const id = this.nextId++;
    return new Promise((resolve, reject) => {
      this.pending.set(id, { resolve, reject });
      this.send({ id, method, params });
    });
  }

  notify(method, params) {
    this.send({ method, params });
  }

  receive(chunk) {
    this.buffer = Buffer.concat([this.buffer, chunk]);
    for (;;) {
      const headerEnd = this.buffer.indexOf("\r\n\r\n");
      if (headerEnd < 0) return;
      const header = this.buffer.slice(0, headerEnd).toString("ascii");
      const match = /Content-Length:\s*(\d+)/i.exec(header);
      if (!match) {
        this.buffer = this.buffer.slice(headerEnd + 4);
        continue;
      }
      const length = parseInt(match[1], 10);
      const start = headerEnd + 4;
      if (this.buffer.length < start + length) return;
      const body = this.buffer.slice(start, start + length).toString("utf8");
      this.buffer = this.buffer.slice(start + length);
      let message;
      try {
        message = JSON.parse(body);
      } catch (e) {
        continue;
      }
      if (message.id !== undefined && this.pending.has(message.id)) {
        const p = this.pending.get(message.id);
        this.pending.delete(message.id);
        if (message.error) p.reject(new Error(message.error.message));
        else p.resolve(message.result);
      } else if (message.method && this.handlers.has(message.method)) {
        this.handlers.get(message.method)(message.params);
      }
    }
  }

  stop() {
    this.stopping = true;
    if (this.closed) return;
    this.request("shutdown", null)
      .catch(() => {})
      .then(() => {
        this.notify("exit", null);
        setTimeout(() => this.process.kill(), 500);
      });
  }
}

// showProblem tells the user, with a button to see the details.
function showProblem(output, message) {
  vscode.window.showWarningMessage(message, "Show Details").then((choice) => {
    if (choice === "Show Details") output.show(true);
  });
}

// ---- converting between the protocol and VS Code ----

const toPosition = (p) => new vscode.Position(p.line, p.character);
const toRange = (r) => new vscode.Range(toPosition(r.start), toPosition(r.end));
const fromPosition = (p) => ({ line: p.line, character: p.character });
const docParams = (doc) => ({ textDocument: { uri: doc.uri.toString() } });
const posParams = (doc, pos) => ({ textDocument: { uri: doc.uri.toString() }, position: fromPosition(pos) });
// The protocol's kinds count from 1, VS Code's from 0.
const completionKind = (k) => (k ? k - 1 : vscode.CompletionItemKind.Text);
const symbolKind = (k) => (k ? k - 1 : vscode.SymbolKind.Variable);

// The colors turtle lsp sends, by index: the first six every editor
// takes, then Turtle's own groups (imports, data structures, show,
// theories, methods), which it sends once initialize lists them.
const tokenTypes = ["keyword", "string", "number", "comment", "function", "type", "namespace", "struct", "event", "macro", "method"];
const tokenModifiers = ["declaration", "defaultLibrary", "test"];
const legend = new vscode.SemanticTokensLegend(tokenTypes, tokenModifiers);

// ---- color schemes ----

// Each scheme colors the eight groups: [dark themes, light themes, style],
// the style b bold, i italic, u underline. Every color has 4.5:1 contrast
// or more on its theme's background (7:1 in High Contrast).
const SCHEMES = {
  "Turtle": {
    kw: ["#4eba65", "#2c7a39", ""],
    imp: ["#4ec9b0", "#0b8577", ""],
    fn: ["#6cb6ff", "#1f63c7", ""],
    ds: ["#f0a04b", "#a85800", ""],
    show: ["#c9a7ff", "#6a3fc0", "b"],
    str: ["#e5865f", "#b5432a", ""],
    num: ["#f27db8", "#b4307d", ""],
    thy: ["#ffd24d", "#956e00", "bi"]
  },
  "Classic": {
    kw: ["#569cd6", "#216197", ""],
    imp: ["#c586c0", "#7e3a78", ""],
    fn: ["#dcdcaa", "#797930", ""],
    ds: ["#4ec9b0", "#258370", ""],
    show: ["#9cdcfe", "#0078b8", "b"],
    str: ["#ce9178", "#89492f", ""],
    num: ["#b5cea8", "#537741", ""],
    thy: ["#d7ba7d", "#8f6e29", "bi"]
  },
  "Ocean": {
    kw: ["#4fc1e9", "#117ca2", ""],
    imp: ["#3fd1c0", "#1c8276", ""],
    fn: ["#8c9cff", "#001ab8", ""],
    ds: ["#ffb86c", "#b35c00", ""],
    show: ["#a6e3ff", "#007bb3", "b"],
    str: ["#f4a988", "#ac3d0c", ""],
    num: ["#ff8fb3", "#b8003b", ""],
    thy: ["#ffd866", "#946e00", "bi"]
  },
  "Sunset": {
    kw: ["#ff9e64", "#b84500", ""],
    imp: ["#e0af68", "#9b671c", ""],
    fn: ["#7aa2f7", "#063db1", ""],
    ds: ["#f7768e", "#b20626", ""],
    show: ["#bb9af7", "#440aae", "b"],
    str: ["#9ece6a", "#568027", ""],
    num: ["#ff9eaf", "#b80020", ""],
    thy: ["#ffd75f", "#946f00", "bi"]
  },
  "Forest": {
    kw: ["#8fbf6a", "#578235", ""],
    imp: ["#8fbcbb", "#447473", ""],
    fn: ["#88c0d0", "#327386", ""],
    ds: ["#d08770", "#8d422b", ""],
    show: ["#b48ead", "#6f4868", "b"],
    str: ["#e0a5a0", "#8d322b", ""],
    num: ["#d9707a", "#94242f", ""],
    thy: ["#ebcb8b", "#976c17", "bi"]
  },
  "Soft": {
    kw: ["#93b88a", "#4d7344", ""],
    imp: ["#8ab5b0", "#46726c", ""],
    fn: ["#8fa8c8", "#3b587d", ""],
    ds: ["#c9a27e", "#835a34", ""],
    show: ["#b9a3c9", "#604375", "b"],
    str: ["#c99a8c", "#7f4939", ""],
    num: ["#c49aab", "#754257", ""],
    thy: ["#d6c08a", "#8a6f2d", "bi"]
  },
  "Dusk": {
    kw: ["#a7b77a", "#69783f", ""],
    imp: ["#87a9a0", "#4c6c63", ""],
    fn: ["#9aa6c9", "#3f4d79", ""],
    ds: ["#c7a06c", "#876231", ""],
    show: ["#c4a7b9", "#6f4960", "b"],
    str: ["#bf9277", "#7f5339", ""],
    num: ["#b98f9c", "#724653", ""],
    thy: ["#d4b46a", "#917127", "bi"]
  },
  "Okabe-Ito": {
    kw: ["#009e73", "#00714f", ""],
    imp: ["#56b4e9", "#1f6f9e", ""],
    fn: ["#2f9be0", "#0060a0", ""],
    ds: ["#e69f00", "#8a5c00", ""],
    show: ["#cc79a7", "#9c3d73", "b"],
    str: ["#df6300", "#a64800", ""],
    num: ["#f0e442", "#6e6600", ""],
    thy: ["#ffffff", "#000000", "biu"]
  },
  "Blue & Orange": {
    kw: ["#4da3ff", "#0059b8", "b"],
    imp: ["#82c4ff", "#0061b8", "i"],
    fn: ["#c4e0ff", "#0057b8", ""],
    ds: ["#ff9933", "#b85c00", ""],
    show: ["#ffd166", "#996b00", "bu"],
    str: ["#ffb380", "#b84a00", "i"],
    num: ["#e6e6e6", "#5c5c5c", ""],
    thy: ["#ffe066", "#8f7200", "biu"]
  },
  "Teal & Rose": {
    kw: ["#2ec4b6", "#0f7f74", "b"],
    imp: ["#8be9e0", "#0d6e66", "i"],
    fn: ["#7fd8d0", "#14736b", ""],
    ds: ["#ff6b6b", "#c0392b", ""],
    show: ["#ffffff", "#000000", "bu"],
    str: ["#ff9e9e", "#b03a3a", "i"],
    num: ["#ff4fa3", "#b3196a", ""],
    thy: ["#ff7a7a", "#a3201f", "biu"]
  },
  "High Contrast": {
    kw: ["#8cff66", "#1a6600", "b"],
    imp: ["#5cf0d8", "#076455", ""],
    fn: ["#7fc8ff", "#005a9e", ""],
    ds: ["#ffb84d", "#804c00", ""],
    show: ["#e0b8ff", "#6700b8", "b"],
    str: ["#ff9f80", "#a42800", ""],
    num: ["#ff8fd0", "#ae0065", ""],
    thy: ["#ffe14d", "#665500", "biu"]
  },
  "No Color": {
    kw: ["#ffffff", "#000000", "b"],
    imp: ["#9a9a9a", "#6b6b6b", "i"],
    fn: ["#d9d9d9", "#2b2b2b", "u"],
    ds: ["#ffffff", "#000000", "bi"],
    show: ["#ffffff", "#000000", "bu"],
    str: ["#b3b3b3", "#5c5c5c", "i"],
    num: ["#c6c6c6", "#454545", ""],
    thy: ["#ffffff", "#000000", "biu"]
  },
};

// What each group colors: the grammar's scopes (colors before turtle lsp
// starts, or without it) and turtle lsp's tokens, Turtle files only.
const GROUPS = {
  kw: { setting: "keywords", scopes: ["keyword.control.turtle", "keyword.other.turtle", "keyword.other.library.turtle"], tokens: ["keyword"] },
  imp: { setting: "imports", scopes: ["keyword.other.import.turtle"], tokens: ["namespace"] },
  fn: { setting: "functions", scopes: ["entity.name.function.call.turtle", "entity.name.function.method.turtle", "support.function.turtle"], tokens: ["function", "method"] },
  ds: { setting: "data", scopes: ["storage.type.data.turtle"], tokens: ["struct", "type"] },
  show: { setting: "show", scopes: ["keyword.other.show.turtle"], tokens: ["event"] },
  str: { setting: "text", scopes: ["string.quoted.double.turtle", "string.quoted.single.turtle", "string.quoted.other.raw.turtle"], tokens: ["string"] },
  num: { setting: "numbers", scopes: ["constant.numeric.turtle", "constant.language.turtle"], tokens: ["number"] },
  thy: { setting: "theories", scopes: ["keyword.other.theory.turtle"], tokens: ["macro"] },
};
// Where a function or a type is made: its group's color, in bold (a test
// function bold and italic).
const DEFINITIONS = [
  { group: "fn", style: "b", scopes: ["entity.name.function.definition.turtle"], tokens: ["function.declaration"] },
  { group: "fn", style: "bi", scopes: ["entity.name.function.test.turtle"], tokens: ["function.declaration.test"] },
  { group: "ds", style: "b", scopes: ["entity.name.type.definition.turtle"], tokens: ["struct.declaration", "type.declaration"] },
];
const RULE_NAME = "Turtle: "; // the start of every text-color rule this extension writes

const fontStyle = (style) => [style.includes("b") && "bold", style.includes("i") && "italic", style.includes("u") && "underline"].filter(Boolean).join(" ");

// schemeRules are the color rules for a scheme on a light or dark theme,
// with the user's own colors (turtle.colors) over the scheme's: the
// grammar's (textMateRules) and turtle lsp's (semantic rules).
function schemeRules(name, light, own) {
  const scheme = SCHEMES[name];
  if (!scheme) return { textMate: [], semantic: {} };
  const textMate = [];
  const semantic = {};
  const add = (label, color, style, scopes, tokens) => {
    textMate.push({ name: RULE_NAME + label, scope: scopes, settings: { foreground: color, fontStyle: fontStyle(style) } });
    for (const t of tokens) {
      semantic[t + ":turtle"] = { foreground: color, bold: style.includes("b"), italic: style.includes("i"), underline: style.includes("u") };
    }
  };
  const colorOf = (group) => (own && own[GROUPS[group].setting]) || scheme[group][light ? 1 : 0];
  for (const group of Object.keys(GROUPS)) {
    add(GROUPS[group].setting, colorOf(group), scheme[group][2], GROUPS[group].scopes, GROUPS[group].tokens);
  }
  for (const d of DEFINITIONS) {
    add(GROUPS[d.group].setting + " (where made)", colorOf(d.group), d.style + scheme[d.group][2], d.scopes, d.tokens);
  }
  return { textMate, semantic };
}

// The semantic rule names this extension writes: only these are replaced,
// so a user's own rules for Turtle stay.
const OWN_TOKENS = new Set(
  Object.values(GROUPS).flatMap((g) => g.tokens).concat(DEFINITIONS.flatMap((d) => d.tokens)).map((t) => t + ":turtle")
);

// applyColors writes the chosen scheme into the user's color settings,
// for Turtle files only, in the shades for the theme now in use: light or
// dark. It replaces only what it wrote before.
async function applyColors() {
  const turtle = vscode.workspace.getConfiguration("turtle");
  const name = turtle.get("colorScheme", "Turtle");
  const kind = vscode.window.activeColorTheme ? vscode.window.activeColorTheme.kind : 2;
  const light = kind === 1 || kind === 4; // Light, HighContrastLight
  const rules = schemeRules(name, light, turtle.get("colors", {}));

  const editor = vscode.workspace.getConfiguration("editor");
  const target = vscode.ConfigurationTarget.Global;

  const tm = Object.assign({}, (editor.inspect("tokenColorCustomizations") || {}).globalValue || {});
  const kept = (tm.textMateRules || []).filter((r) => !(r.name || "").startsWith(RULE_NAME));
  const textMateRules = kept.concat(rules.textMate);
  if (JSON.stringify(textMateRules) !== JSON.stringify(tm.textMateRules || [])) {
    if (textMateRules.length) tm.textMateRules = textMateRules;
    else delete tm.textMateRules;
    await editor.update("tokenColorCustomizations", Object.keys(tm).length ? tm : undefined, target);
  }

  const sem = Object.assign({}, (editor.inspect("semanticTokenColorCustomizations") || {}).globalValue || {});
  const semRules = {};
  for (const [k, v] of Object.entries(sem.rules || {})) if (!OWN_TOKENS.has(k)) semRules[k] = v;
  Object.assign(semRules, rules.semantic);
  if (JSON.stringify(semRules) !== JSON.stringify(sem.rules || {})) {
    if (Object.keys(semRules).length) sem.rules = semRules;
    else delete sem.rules;
    await editor.update("semanticTokenColorCustomizations", Object.keys(sem).length ? sem : undefined, target);
  }
}

// ---- the server's life ----

function start(context, output) {
  const config = vscode.workspace.getConfiguration("turtle");
  if (!config.get("languageServer", true)) return;
  const command = config.get("path", "turtle");
  const conn = new Connection(command, output);
  server = conn;

  const diagnostics = vscode.languages.createDiagnosticCollection("turtle");
  active.push(diagnostics);
  conn.handlers.set("textDocument/publishDiagnostics", (params) => {
    const uri = vscode.Uri.parse(params.uri);
    diagnostics.set(
      uri,
      params.diagnostics.map((d) => {
        const diag = new vscode.Diagnostic(
          toRange(d.range),
          d.message,
          d.severity === 2 ? vscode.DiagnosticSeverity.Warning : vscode.DiagnosticSeverity.Error
        );
        diag.source = d.source || "turtle";
        return diag;
      })
    );
  });

  const isTurtle = (doc) => doc.languageId === "turtle";
  const open = (doc) => {
    if (!isTurtle(doc)) return;
    conn.notify("textDocument/didOpen", {
      textDocument: { uri: doc.uri.toString(), languageId: "turtle", version: doc.version, text: doc.getText() },
    });
  };

  conn
    .request("initialize", {
      processId: process.pid,
      rootUri: vscode.workspace.workspaceFolders ? vscode.workspace.workspaceFolders[0].uri.toString() : null,
      capabilities: { textDocument: { semanticTokens: { tokenTypes, tokenModifiers } } },
      clientInfo: { name: "vscode-turtle" },
    })
    .then(() => {
      conn.ready = true;
      conn.notify("initialized", {});
      vscode.workspace.textDocuments.forEach(open);
    })
    .catch((err) => output.appendLine(`initialize failed: ${err.message}`));

  const selector = { language: "turtle" };
  active.push(
    vscode.workspace.onDidOpenTextDocument(open),
    vscode.workspace.onDidChangeTextDocument((e) => {
      if (!isTurtle(e.document)) return;
      conn.notify("textDocument/didChange", {
        textDocument: { uri: e.document.uri.toString(), version: e.document.version },
        contentChanges: [{ text: e.document.getText() }],
      });
    }),
    vscode.workspace.onDidCloseTextDocument((doc) => {
      if (isTurtle(doc)) conn.notify("textDocument/didClose", docParams(doc));
    }),

    vscode.languages.registerCompletionItemProvider(selector, {
      provideCompletionItems(doc, pos) {
        return conn.request("textDocument/completion", posParams(doc, pos)).then((items) =>
          (items || []).map((i) => {
            const item = new vscode.CompletionItem(i.label, completionKind(i.kind));
            item.detail = i.detail;
            if (i.documentation) item.documentation = new vscode.MarkdownString(i.documentation);
            item.sortText = i.sortText;
            return item;
          })
        );
      },
    }),

    vscode.languages.registerHoverProvider(selector, {
      provideHover(doc, pos) {
        return conn.request("textDocument/hover", posParams(doc, pos)).then((h) => {
          if (!h) return null;
          const md = new vscode.MarkdownString(h.contents.value);
          return new vscode.Hover(md, h.range ? toRange(h.range) : undefined);
        });
      },
    }),

    vscode.languages.registerDefinitionProvider(selector, {
      provideDefinition(doc, pos) {
        return conn.request("textDocument/definition", posParams(doc, pos)).then((loc) =>
          loc ? new vscode.Location(vscode.Uri.parse(loc.uri), toRange(loc.range)) : null
        );
      },
    }),

    vscode.languages.registerDocumentSymbolProvider(selector, {
      provideDocumentSymbols(doc) {
        return conn.request("textDocument/documentSymbol", docParams(doc)).then((syms) =>
          (syms || []).map(
            (s) => new vscode.DocumentSymbol(s.name, s.detail || "", symbolKind(s.kind), toRange(s.range), toRange(s.selectionRange))
          )
        );
      },
    }),

    vscode.languages.registerDocumentFormattingEditProvider(selector, {
      provideDocumentFormattingEdits(doc) {
        return conn
          .request("textDocument/formatting", { textDocument: { uri: doc.uri.toString() }, options: { tabSize: 4, insertSpaces: true } })
          .then((edits) => (edits || []).map((e) => new vscode.TextEdit(toRange(e.range), e.newText)));
      },
    }),

    vscode.languages.registerDocumentSemanticTokensProvider(
      selector,
      {
        provideDocumentSemanticTokens(doc) {
          return conn
            .request("textDocument/semanticTokens/full", docParams(doc))
            .then((r) => new vscode.SemanticTokens(new Uint32Array(r ? r.data : [])));
        },
      },
      legend
    ),

    { dispose: () => conn.stop() }
  );
}

// ---- running a file ----

// shellCommand is "cd to dir, then run turtle on file", quoted for the
// shell VS Code's terminal uses: PowerShell, cmd, or a Unix shell.
function shellCommand(shell, turtle, dir, file) {
  const name = (shell || "").split(/[\\/]/).pop().toLowerCase(); // either separator, on any system
  if (name.startsWith("pwsh") || name.startsWith("powershell")) {
    const q = (s) => "'" + s.replace(/'/g, "''") + "'";
    return `Set-Location -LiteralPath ${q(dir)}; & ${q(turtle)} ${q(file)}`;
  }
  if (name === "cmd.exe" || name === "cmd") {
    const q = (s) => '"' + s + '"';
    return `cd /d ${q(dir)} && ${q(turtle)} ${q(file)}`;
  }
  const q = (s) => "'" + s.replace(/'/g, "'\\''") + "'";
  return `cd ${q(dir)} && ${q(turtle)} ${q(file)}`;
}

// runFile saves the open Turtle file and runs it in the Turtle terminal,
// in the file's own folder, as "turtle file.turtle" would.
async function runFile() {
  const editor = vscode.window.activeTextEditor;
  const doc = editor && editor.document;
  if (!doc || doc.languageId !== "turtle") {
    vscode.window.showWarningMessage("Turtle: open a .turtle file to run it.");
    return;
  }
  if (doc.isUntitled) {
    vscode.window.showWarningMessage("Turtle: save the file first, then run it.");
    return;
  }
  if (doc.isDirty && !(await doc.save())) return;
  const turtle = vscode.workspace.getConfiguration("turtle").get("path", "turtle");
  let term = vscode.window.terminals.find((t) => t.name === "Turtle" && t.exitStatus === undefined);
  if (!term) term = vscode.window.createTerminal({ name: "Turtle" });
  term.show(true);
  term.sendText(shellCommand(vscode.env.shell, turtle, path.dirname(doc.fileName), doc.fileName));
}

function stop() {
  for (const d of active) d.dispose();
  active = [];
  server = null;
}

function activate(context) {
  const output = vscode.window.createOutputChannel("Turtle");
  context.subscriptions.push(output);
  start(context, output);
  const recolor = () => applyColors().catch((err) => output.appendLine(`colors: ${err.message}`));
  recolor();
  context.subscriptions.push(
    vscode.window.onDidChangeActiveColorTheme(recolor),
    vscode.workspace.onDidChangeConfiguration((e) => {
      if (e.affectsConfiguration("turtle.colorScheme") || e.affectsConfiguration("turtle.colors")) recolor();
    })
  );
  context.subscriptions.push(
    vscode.commands.registerCommand("turtle.restartServer", () => {
      stop();
      start(context, output);
    }),
    vscode.commands.registerCommand("turtle.run", runFile),
    { dispose: stop }
  );
}

function deactivate() {
  stop();
}

module.exports = { activate, deactivate, shellCommand, schemeRules, applyColors, SCHEMES };
