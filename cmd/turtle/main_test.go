package main

import (
	"os"
	"testing"
)

// TestCommandWords: "turtle test" (or lsp, doc ...) is the command even
// in a project with a folder of that name; only a script file of that
// name runs instead.
func TestCommandWords(t *testing.T) {
	dir := t.TempDir()
	wd, _ := os.Getwd()
	defer os.Chdir(wd)
	os.Chdir(dir)
	saved := os.Args
	defer func() { os.Args = saved }()
	os.Args = []string{"turtle", "lsp"}
	if !command("lsp") {
		t.Error("lsp with nothing called lsp here")
	}
	os.Mkdir("lsp", 0o755)
	if !command("lsp") {
		t.Error("lsp with a folder called lsp")
	}
	os.Args = []string{"turtle", "test"}
	os.WriteFile("test", []byte("show 1 ."), 0o644)
	if command("test") {
		t.Error("a script file called test should run")
	}
}
