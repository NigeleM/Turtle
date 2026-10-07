#!/bin/sh
# Runs extension.js against stand-ins (see stubs.js) in three cases.
cd "$(dirname "$0")"
cat stubs.js ../extension.js cases.js > /tmp/turtle-vscode-sim.js
JSC=/System/Library/Frameworks/JavaScriptCore.framework/Versions/Current/Helpers/jsc
if [ -x "$JSC" ]; then "$JSC" /tmp/turtle-vscode-sim.js; else node /tmp/turtle-vscode-sim.js; fi
