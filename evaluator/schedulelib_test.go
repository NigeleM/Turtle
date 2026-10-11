// Copyright 2017-2026 Nigele McCoy
// Licensed under the Apache License, Version 2.0;
// see LICENSE and NOTICE.

package evaluator

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// The schedule library: results in order, never more than the limit at
// once, the first failure stopping the rest, and every misuse an error.

// slowServer answers /n with "page n" after a pause, /fail/n with a 404,
// and counts requests and the most it was answering at once.
func slowServer(t *testing.T) (srv *httptest.Server, most, total *atomic.Int64) {
	most, total = new(atomic.Int64), new(atomic.Int64)
	var now atomic.Int64
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		total.Add(1)
		n := now.Add(1)
		defer now.Add(-1)
		for {
			m := most.Load()
			if n <= m || most.CompareAndSwap(m, n) {
				break
			}
		}
		time.Sleep(40 * time.Millisecond)
		if strings.HasPrefix(r.URL.Path, "/fail/") {
			http.Error(w, "gone", http.StatusNotFound)
			return
		}
		fmt.Fprint(w, "page "+strings.TrimPrefix(r.URL.Path, "/"))
	}))
	t.Cleanup(srv.Close)
	return srv, most, total
}

// urlList is a Turtle list of n addresses on srv; fail marks which fail.
func urlList(srv *httptest.Server, n int, fail ...int) string {
	var parts []string
	for i := 0; i < n; i++ {
		p := fmt.Sprint(i)
		for _, f := range fail {
			if f == i {
				p = "fail/" + p
			}
		}
		parts = append(parts, fmt.Sprintf("%q", srv.URL+"/"+p))
	}
	return "urls = list [" + strings.Join(parts, ", ") + "]\n"
}

func TestFetchallLimit(t *testing.T) {
	cases := []struct {
		setting string
		want    int64
	}{
		{"", 5},                           // the default
		{"schedulelimit = 2\n", 2},        // the file's setting
		{"schedulelimit = none\n", 12},    // no cap
		{"schedulelimit = 50\n", 12},      // more than there are
		{"x = 0\nschedulelimit = 3\n", 3}, // any line can set it
	}
	for _, c := range cases {
		srv, most, _ := slowServer(t)
		src := "import schedule\n" + c.setting + urlList(srv, 12) + "pages = fetchall[urls]\nshow pages at len, \" \", pages at get[0], \" \", pages at get[11] ."
		got, err := run(t, src, "")
		if err != nil {
			t.Fatalf("%q: %v", c.setting, err)
		}
		if strings.TrimSpace(got) != "12 page 0 page 11" {
			t.Errorf("%q: got %q", c.setting, got)
		}
		if most.Load() != c.want {
			t.Errorf("%q: at most %d at once, want %d", c.setting, most.Load(), c.want)
		}
	}
	// A call's own limit, and the sentence form.
	srv, most, _ := slowServer(t)
	got, err := run(t, "import schedule\n"+urlList(srv, 6)+"a = fetchall[urls, map [\"limit\": 1]]\nb = urls fetchall\nshow a == b, a at get[5] .", "")
	if err != nil || strings.TrimSpace(got) != "truepage 5" {
		t.Errorf("got %q, %v", got, err)
	}
	if most.Load() != 5 {
		t.Errorf("most at once %d, want 5 (the sentence call uses the default)", most.Load())
	}
}

func TestFetchallErrors(t *testing.T) {
	srv, _, total := slowServer(t)
	// One at a time: item 1 fails, so 2..5 never start.
	src := "import schedule\nschedulelimit = 1\n" + urlList(srv, 6, 1) +
		"safe\n    pages = fetchall[urls]\nhandle [] e .\n    show kind of e, \"|\", message of e .\nsafe [end]"
	got, err := run(t, src, "")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(got, "http|fetchall item 1: http_get "+srv.URL+"/fail/1: 404 Not Found") {
		t.Errorf("got %q", got)
	}
	if total.Load() != 2 {
		t.Errorf("%d requests, want 2 (the rest never start)", total.Load())
	}

	// Skipping: the failed ones are none, the rest come back.
	for _, skip := range []string{"skipschedule_error = true\npages = fetchall[urls]", "pages = fetchall[urls, map [\"skip_errors\": true]]"} {
		srv, _, _ := slowServer(t)
		got, err := run(t, "import schedule\n"+urlList(srv, 4, 0, 2)+skip+"\nshow pages .", "")
		if err != nil {
			t.Fatalf("%s: %v", skip, err)
		}
		if strings.TrimSpace(got) != `[ none, "page 1", none, "page 3" ]` {
			t.Errorf("%s: got %q", skip, got)
		}
	}
	// The call's map wins over the file's setting.
	srv, _, _ = slowServer(t)
	got, _ = run(t, "import schedule\nskipschedule_error = true\n"+urlList(srv, 3, 2)+
		"safe\n    x = fetchall[urls, map [\"skip_errors\": false]]\nhandle [http] e .\n    show \"stopped\" .\nsafe [end]", "")
	if strings.TrimSpace(got) != "stopped" {
		t.Errorf("got %q", got)
	}
	// Nothing to do.
	got, err = run(t, "import schedule\nshow fetchall[list []] .", "")
	if err != nil || strings.TrimSpace(got) != "[  ]" {
		t.Errorf("empty: got %q, %v", got, err)
	}
}

func TestRunall(t *testing.T) {
	if _, err := os.Stat("/bin/sh"); err != nil {
		t.Skip("no sh")
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "here.txt"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	src := `import schedule
outs = runall[list ["echo hi", "echo oops 1>&2; exit 3", "ls", "printf 'a\nb\n\n'"]]
[loop][o in outs]
    show o at get["output"], "|", o at get["errors"], "|", o at get["code"] .
[loop][end]`
	got, err := runIn(t, dir, src, "")
	if err != nil {
		t.Fatal(err)
	}
	want := "hi||0\n|oops|3\nhere.txt||0\na\nb||0"
	if strings.TrimSpace(got) != want {
		t.Errorf("got %q, want %q", got, want)
	}
	// Side by side: four 0.3 s sleeps take about 0.3 s, not 1.2 s.
	start := time.Now()
	if _, err := run(t, "import schedule\nx = runall[list [\"sleep 0.3\", \"sleep 0.3\", \"sleep 0.3\", \"sleep 0.3\"]]", ""); err != nil {
		t.Fatal(err)
	}
	if d := time.Since(start); d > 900*time.Millisecond {
		t.Errorf("took %v; not side by side", d)
	}
}

func TestScheduleMisuse(t *testing.T) {
	dir := t.TempDir()
	cases := []struct{ src, kind, want string }{
		{`fetchall[]`, "type", "fetchall takes a list of web addresses"},
		{`fetchall["http://x"]`, "type", "fetchall needs a list of web addresses, got string"},
		{`fetchall[list [1]]`, "type", "fetchall: item 0 must be text, got integer"},
		{`fetchall[list ["nope"]]`, "http", `fetchall item 0: http_get: "nope" isn't a web address`},
		{`fetchall[list [], map ["fast": true]]`, "schedule", `"fast" isn't a setting`},
		{`fetchall[list [], map ["limit": 0]]`, "schedule", "fetchall's limit must be 1 or more"},
		{`fetchall[list [], map ["limit": "8"]]`, "type", "fetchall's limit must be a whole number"},
		{`fetchall[list [], 5]`, "type", "the settings must be a map"},
		{"runall[list []]\n    schedulelimit = -1\n    x = runall[list []]", "schedule", "schedulelimit must be 1 or more"},
		{"runall[list []]\n    skipschedule_error = \"yes\"\n    x = runall[list []]", "type", "skipschedule_error must be true or false"},
		{`queryall[5, list []]`, "type", "queryall needs a database"},
		{`queryall[sql_create["a.db"], list ["SELECT 1"]]`, "schedule", "queryall works with postgres and mysql databases"},
	}
	for _, c := range cases {
		os.Remove(filepath.Join(dir, "a.db"))
		src := "import schedule\nimport sql\nsafe\n    x = " + c.src + "\nhandle [] e .\n    show kind of e, \"|\", message of e .\nsafe [end]"
		got, err := runIn(t, dir, src, "")
		if err != nil {
			t.Errorf("%s: %v", c.src, err)
			continue
		}
		if !strings.HasPrefix(got, c.kind+"|") || !strings.Contains(got, c.want) {
			t.Errorf("%s: got %q, want kind %s and %q", c.src, got, c.kind, c.want)
		}
	}
	// Settings already in the file are kept; schedule is a kind safe can name.
	got, err := run(t, "schedulelimit = 9\nimport schedule\nshow schedulelimit .\nsafe\n    x = runall[list [], map [\"limit\": 0]]\nhandle [schedule] e .\n    show \"caught\" .\nsafe [end]", "")
	if err != nil || strings.TrimSpace(got) != "9\ncaught" {
		t.Errorf("got %q, %v", got, err)
	}
	// Without the import, a clear message.
	if _, err := run(t, "x = fetchall[list []]", ""); err == nil || !strings.Contains(err.Error(), `import schedule`) {
		t.Errorf("got %v", err)
	}
}

// TestQueryall runs against real servers when TURTLE_PG_URL or
// TURTLE_MYSQL_URL is set, as the drivers' own tests do.
func TestQueryall(t *testing.T) {
	ran := false
	for _, env := range []string{"TURTLE_PG_URL", "TURTLE_MYSQL_URL"} {
		dsn := os.Getenv(env)
		if dsn == "" {
			continue
		}
		ran = true
		src := fmt.Sprintf(`import schedule
import sql
db = sql_open[%q]
r = queryall[db, list ["SELECT 1 AS n", list ["SELECT ? AS n", list [2]], "SELECT 3 AS n", "SELECT 4 AS n", "SELECT 5 AS n", "SELECT 6 AS n"]]
[loop][rows in r]
    show rows at get[0] at get["n"], " " .
[loop][end]
skipschedule_error = true
r = queryall[db, list ["SELECT 1 AS n", "SELECT FROM nowhere"]]
show r at get[1] .`, dsn)
		got, err := run(t, src, "")
		if err != nil {
			t.Errorf("%s: %v", env, err)
		} else if strings.Join(strings.Fields(got), " ") != "1 2 3 4 5 6 none" {
			t.Errorf("%s: got %q", env, got)
		}
	}
	if !ran {
		t.Skip("set TURTLE_PG_URL or TURTLE_MYSQL_URL to run against a server")
	}
}
