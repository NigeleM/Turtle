function fresh() { warnings = []; registered = []; procs = []; sent = []; terminals.length = 0; return { subscriptions: { push: function () {} } }; }
async function settle() { for (var i = 0; i < 50; i++) await null; }

async function main() {
  // 1. turtle missing
  module.exports.activate(fresh());
  procs[0].emit("error", { code: "ENOENT", message: "spawn turtle ENOENT" });
  await settle();
  print("missing:  " + warnings.join(" | "));
  module.exports.deactivate();

  // 2. an old turtle: says something and quits before answering
  module.exports.activate(fresh());
  procs[0].stderr.emit("data", "turtle: open lsp: no such file or directory\n");
  procs[0].emit("exit", 1);
  await settle();
  print("old:      " + warnings.join(" | "));
  module.exports.deactivate();

  // 3. a working server: answers initialize; stopping it later says nothing
  module.exports.activate(fresh());
  var body = JSON.stringify({ jsonrpc: "2.0", id: 1, result: { capabilities: {} } });
  procs[0].stdout.emit("data", new FakeBuf("Content-Length: " + body.length + "\r\n\r\n" + body));
  await settle();
  module.exports.deactivate();
  procs[0].emit("exit", 0);
  await settle();
  print("working:  warnings=" + warnings.length + " providers=" + registered.join(",") + " sent initialized=" + (procs[0].written.indexOf('"initialized"') >= 0));
}
// 4. Run File: saves, then runs in one Turtle terminal, quoted for the shell.
async function runCases() {
  module.exports.activate(fresh());
  var saved = 0;
  function doc(file, extra) {
    var d = { languageId: "turtle", fileName: file, isUntitled: false, isDirty: false, save: async function () { saved++; return true; } };
    for (var k in extra) d[k] = extra[k];
    return { document: d };
  }
  fakeVscode.window.activeTextEditor = doc("/Users/bo/my code/it's.turtle", { isDirty: true });
  await commands["turtle.run"]();
  await commands["turtle.run"]();
  print("run:      saved=" + saved + " terminals=" + terminals.length + " sent=" + sent[0]);
  fakeVscode.window.activeTextEditor = doc("/tmp/x.turtle", { isUntitled: true });
  await commands["turtle.run"]();
  fakeVscode.window.activeTextEditor = { document: { languageId: "python" } };
  await commands["turtle.run"]();
  print("refused:  " + warnings.join(" | "));
  var f = module.exports.shellCommand;
  print("pwsh:     " + f("C:\\Program Files\\PowerShell\\7\\pwsh.exe", "turtle", "C:\\a b", "C:\\a b\\it's.turtle"));
  print("cmd:      " + f("C:\\Windows\\System32\\cmd.exe", "turtle", "C:\\a b", "C:\\a b\\x.turtle"));
  print("bash:     " + f("/bin/bash", "turtle", "/a b", "/a b/it's.turtle"));
  module.exports.deactivate();
}

main().then(runCases).catch(function (e) { print("ERROR " + e + "\n" + e.stack); });
