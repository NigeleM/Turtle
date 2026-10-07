function fresh() { warnings = []; registered = []; procs = []; return { subscriptions: { push: function () {} } }; }
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
main().catch(function (e) { print("ERROR " + e + "\n" + e.stack); });
