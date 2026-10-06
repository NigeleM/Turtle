package evaluator

import (
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// runLog runs src in a fresh folder and returns stdout, what went to
// stderr, the error, and the folder (for the files it wrote).
func runLog(t *testing.T, src string) (stdout, stderr string, err error, dir string) {
	t.Helper()
	dir = t.TempDir()
	r, w, perr := os.Pipe()
	if perr != nil {
		t.Fatal(perr)
	}
	saved := os.Stderr
	os.Stderr = w
	done := make(chan string)
	go func() {
		b, _ := io.ReadAll(r)
		done <- string(b)
	}()
	scriptName = "main.trt"
	stdout, err = runIn(t, dir, src, "")
	scriptName = ""
	os.Stderr = saved
	w.Close()
	stderr = <-done
	return stdout, stderr, err, dir
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

var stamp = regexp.MustCompile(`^\d{4}-\d\d-\d\d \d\d:\d\d:\d\d `)

func TestLogConsoleAndLevels(t *testing.T) {
	out, errOut, err, _ := runLog(t, `import log
show "program output" .
log "started on port ", 8080 .
log debug "hidden" .
log warn "disk at ", 91, "%" .
log error "lost it" .
loglevel = "warn"
log info "hidden too" .
log warn "still shown" .
loglevel = "off"
log error "nothing" .`)
	if err != nil {
		t.Fatal(err)
	}
	if out != "program output\n" {
		t.Errorf("stdout: %q (log lines go to stderr)", out)
	}
	lines := strings.Split(strings.TrimSpace(errOut), "\n")
	want := []string{"INFO  started on port 8080", "WARN  disk at 91%", "ERROR lost it", "WARN  still shown"}
	if len(lines) != len(want) {
		t.Fatalf("stderr: %q", errOut)
	}
	for i, l := range lines {
		if !stamp.MatchString(l) || !strings.HasSuffix(l, want[i]) {
			t.Errorf("line %d: %q, want time + %q", i, l, want[i])
		}
	}
}

func TestLogFileAndParts(t *testing.T) {
	_, errOut, err, dir := runLog(t, `import log
logconsole = false
logfile = "app.log"
logtime = none
log "plain" .
logparts = list ["level", "file", "line", "message"]
log info "with file and line" .
logparts = list ["where", "message"]
log warn "where" .
logparts = list ["message"]
log info "login", map ["user": "ann", "from": "home office", "tries": 2, "note": ""] .
logtime = "hh:mm"
logparts = list ["time", "message"]
log info "short time" .`)
	if err != nil {
		t.Fatal(err)
	}
	if errOut != "" {
		t.Errorf("logconsole = false still printed %q", errOut)
	}
	lines := strings.Split(strings.TrimSpace(readFile(t, filepath.Join(dir, "app.log"))), "\n")
	want := []string{
		"INFO  plain",
		"INFO  main.trt 7 with file and line",
		"main.trt:9 where",
		`login user=ann from="home office" tries=2 note=""`,
	}
	for i, w := range want {
		if lines[i] != w {
			t.Errorf("line %d: %q, want %q", i, lines[i], w)
		}
	}
	if !regexp.MustCompile(`^\d\d:\d\d short time$`).MatchString(lines[4]) {
		t.Errorf("custom time: %q", lines[4])
	}
}

func TestLogJSON(t *testing.T) {
	_, _, err, dir := runLog(t, `import log
logconsole = false
logfile = "app.json"
logformat = "json"
log warn "low stock", map ["sku": "B1", "left": 2] .`)
	if err != nil {
		t.Fatal(err)
	}
	got := readFile(t, filepath.Join(dir, "app.json"))
	if !regexp.MustCompile(`^\{"time":"\d{4}-\d\d-\d\d \d\d:\d\d:\d\d","level":"warn","message":"low stock","file":"main.trt","line":5,"sku":"B1","left":2\}\n$`).MatchString(got) {
		t.Errorf("json line: %q", got)
	}
}

func TestLogFileAppendsAcrossRuns(t *testing.T) {
	dir := t.TempDir()
	src := "import log\nlogconsole = false\nlogfile = \"app.log\"\nlogtime = none\nlog \"run\" .\n"
	for i := 0; i < 2; i++ {
		if _, err := runIn(t, dir, src, ""); err != nil {
			t.Fatal(err)
		}
	}
	if got := readFile(t, filepath.Join(dir, "app.log")); got != "INFO  run\nINFO  run\n" {
		t.Errorf("got %q", got)
	}
}

func TestLogRotation(t *testing.T) {
	_, _, err, dir := runLog(t, `import log
logconsole = false
logfile = "app.log"
logtime = none
logparts = list ["message"]
logmaxsize = 10
logkeep = 2
[loop][i = 1; i <= 5; i++]
    log "line {i}" .
[loop][end]`)
	if err != nil {
		t.Fatal(err)
	}
	// Each line is 7 bytes, so every file holds one: the newest in app.log,
	// the two before it in .1 and .2, the rest dropped.
	for name, want := range map[string]string{"app.log": "line 5\n", "app.log.1": "line 4\n", "app.log.2": "line 3\n"} {
		if got := readFile(t, filepath.Join(dir, name)); got != want {
			t.Errorf("%s: %q, want %q", name, got, want)
		}
	}
	if _, err := os.Stat(filepath.Join(dir, "app.log.3")); err == nil {
		t.Error("app.log.3 should be gone (logkeep = 2)")
	}
}

func TestOutputFile(t *testing.T) {
	out, _, err, dir := runLog(t, `import log
import system
show "before" .
outputfile = "run.txt"
show "total: ", 42 .
warn "careful" .
outputfile = none
show "after" .`)
	if err != nil {
		t.Fatal(err)
	}
	if out != "before\ntotal: 42\nafter\n" {
		t.Errorf("screen: %q", out)
	}
	if got := readFile(t, filepath.Join(dir, "run.txt")); got != "total: 42\ncareful\n" {
		t.Errorf("run.txt: %q", got)
	}
}

func TestLogTheErrorThatStops(t *testing.T) {
	_, _, err, dir := runLog(t, `import log
logconsole = false
logfile = "app.log"
logtime = none
logparts = list ["level", "where", "message"]
log "starting" .
x = 1 div 0`)
	if err == nil {
		t.Fatal("expected the division error")
	}
	if got := readFile(t, filepath.Join(dir, "app.log")); got != "INFO  main.trt:6 starting\nERROR main.trt:7 stopped: division by zero\n" {
		t.Errorf("got %q", got)
	}
}

func TestLogInFunctionsAndModules(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "lib"), 0o755)
	os.WriteFile(filepath.Join(dir, "lib", "worker.trt"), []byte("import log\nlogconsole = false\nlogfile = \"worker.log\"\nlogtime = none\nlogparts = list [\"where\", \"message\"]\ndef work[]\n    log \"working\" .\ndef [end]\n"), 0o644)
	_, err := runIn(t, dir, `import log
import lib/worker
logconsole = false
logfile = "main.log"
logtime = none
def quiet[]
    loglevel = "error"
    log "not shown: this function's own setting" .
def [end]
quiet[]
log "shown" .
work[]`, "")
	if err != nil {
		t.Fatal(err)
	}
	if got := readFile(t, filepath.Join(dir, "main.log")); got != "INFO  shown\n" {
		t.Errorf("main.log: %q", got)
	}
	if got := readFile(t, filepath.Join(dir, "worker.log")); got != "lib/worker.trt:7 working\n" {
		t.Errorf("worker.log (a module logs with its own settings): %q", got)
	}
}

func TestLogNamesAndMistakes(t *testing.T) {
	// Without import log, log is an ordinary name.
	out, err := run(t, "log = 5\ndef logs[]\n    return 1\ndef [end]\nshow log + logs[] .", "")
	if err != nil || out != "6\n" {
		t.Fatalf("got %q, %v", out, err)
	}
	// With it, log = ... still assigns.
	out, err = run(t, "import log\nlogconsole = false\nlog = 5\nshow log .", "")
	if err != nil || out != "5\n" {
		t.Fatalf("assign with import log: %q, %v", out, err)
	}
	mistakes := map[string]string{
		`loglevel = "loud"` + "\nlog \"x\" .":                 `loglevel must be "debug", "info", "warn", "error" or "off"`,
		`logformat = "xml"` + "\nlog \"x\" .":                 `logformat must be "text" or "json"`,
		`logparts = list ["when"]` + "\nlog \"x\" .":          `"when" isn't a part`,
		`logfile = 5` + "\nlog \"x\" .":                       "logfile must be a file's path",
		`logfile = "a.log"` + "\nlogmaxsize = 0\nlog \"x\" .": "logmaxsize must be a number of bytes above 0",
		`logfile = "no/such/folder/a.log"` + "\nlog \"x\" .":  "no/such/folder/a.log",
	}
	for setup, want := range mistakes {
		_, _, err, _ := runLog(t, "import log\nlogconsole = false\n"+setup)
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%q: want error containing %q, got %v", setup, want, err)
		}
	}
}
