// simulate.js: runs extension.js against stand-ins for VS Code and for
// starting turtle lsp, in three cases: turtle missing, an old turtle,
// a working server. Run with ./run.sh (macOS: the built-in JavaScript
// engine; elsewhere: node). Not part of the extension itself.
var print = typeof print === "function" ? print : console.log;
// Stand-ins for VS Code, Node's child_process and Buffer, to run extension.js.
var warnings = [], registered = [], procs = [];
function FakeBuf(s) { this.s = s; this.length = s.length; }
FakeBuf.prototype.indexOf = function (x) { return this.s.indexOf(x); };
FakeBuf.prototype.slice = function (a, b) { return new FakeBuf(this.s.slice(a, b)); };
FakeBuf.prototype.toString = function () { return this.s; };
var Buffer = { alloc: function () { return new FakeBuf(""); }, from: function (s) { return new FakeBuf(s); },
  concat: function (l) { return new FakeBuf(l.map(function (b) { return b.s; }).join("")); } };
function Emitter() { this.h = {}; }
Emitter.prototype.on = function (e, f) { this.h[e] = f; };
Emitter.prototype.emit = function (e, a) { if (this.h[e]) this.h[e](a); };
function fakeSpawn(cmd, args) {
  var p = new Emitter(); p.stdout = new Emitter(); p.stderr = new Emitter(); p.written = "";
  p.stdin = { write: function (s) { p.written += (s.s !== undefined ? s.s : s); } };
  p.kill = function () {}; p.cmd = cmd; p.args = args; procs.push(p); return p;
}
function Disposable() { return { dispose: function () {} }; }
var fakeVscode = {
  window: {
    createOutputChannel: function () { return { append: function () {}, appendLine: function () {}, show: function () {}, dispose: function () {} }; },
    showWarningMessage: function (m) { warnings.push(m); return { then: function () {} }; },
  },
  workspace: { getConfiguration: function () { return { get: function (k, d) { return d; } }; },
    textDocuments: [], workspaceFolders: null,
    onDidOpenTextDocument: Disposable, onDidChangeTextDocument: Disposable, onDidCloseTextDocument: Disposable },
  languages: { createDiagnosticCollection: function () { return { set: function () {}, dispose: function () {} }; },
    registerCompletionItemProvider: function () { registered.push("completion"); return Disposable(); },
    registerHoverProvider: function () { registered.push("hover"); return Disposable(); },
    registerDefinitionProvider: function () { registered.push("definition"); return Disposable(); },
    registerDocumentSymbolProvider: function () { registered.push("symbols"); return Disposable(); },
    registerDocumentSemanticTokensProvider: function () { registered.push("tokens"); return Disposable(); } },
  commands: { registerCommand: function () { return Disposable(); } },
  SemanticTokensLegend: function () {}, Position: function () {}, Range: function () {},
};
var process = { pid: 1 };
function require(name) {
  if (name === "vscode") return fakeVscode;
  if (name === "child_process") return { spawn: fakeSpawn };
}
var module = { exports: {} };
