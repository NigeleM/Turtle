// The Turtle extension for VS Code. Colors come from the grammar
// (syntaxes/turtle.tmLanguage.json) and need nothing else. Everything
// else (errors as you type, completion, hover help, go to definition, the
// outline, exact colors) comes from turtle lsp, which this file starts and
// talks to: a small Language Server Protocol client written against VS
// Code's own API, with no npm packages.

"use strict";

const vscode = require("vscode");
const { spawn } = require("child_process");

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

const legend = new vscode.SemanticTokensLegend(
  ["keyword", "string", "number", "comment", "function", "type"],
  ["declaration", "defaultLibrary"]
);

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
      capabilities: {},
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

function stop() {
  for (const d of active) d.dispose();
  active = [];
  server = null;
}

function activate(context) {
  const output = vscode.window.createOutputChannel("Turtle");
  context.subscriptions.push(output);
  start(context, output);
  context.subscriptions.push(
    vscode.commands.registerCommand("turtle.restartServer", () => {
      stop();
      start(context, output);
    }),
    { dispose: stop }
  );
}

function deactivate() {
  stop();
}

module.exports = { activate, deactivate };
