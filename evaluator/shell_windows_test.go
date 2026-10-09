package evaluator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestShellOnWindows: sys and runall go through cmd, with quotes in a
// command reaching it as written (TestRunall needs sh, so it skips here).
func TestShellOnWindows(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "here.txt"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	src := `import schedule
outs = runall[list ["echo hi", "echo oops>&2& exit /b 3", "dir /b", "echo \"two words\""]]
[loop][o in outs]
    show o at get["output"], "|", o at get["errors"], "|", o at get["code"] .
[loop][end]
sys echo "from sys"`
	got, err := runIn(t, dir, src, "")
	if err != nil {
		t.Fatal(err)
	}
	want := "hi||0\n|oops|3\nhere.txt||0\n\"two words\"||0\n\"from sys\""
	if got = strings.TrimSpace(strings.ReplaceAll(got, "\r\n", "\n")); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
