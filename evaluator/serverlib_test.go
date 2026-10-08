package evaluator

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"Turtle/lexer"
	"Turtle/parser"
)

// startServer runs src (which calls serve on port 0) and gives back its
// address; the server stops when the test ends.
func startServer(t *testing.T, dir, src string) string {
	t.Helper()
	p := parser.New(lexer.New(src))
	program := p.ParseProgram()
	if errs := p.Errors(); len(errs) > 0 {
		t.Fatalf("parse: %v", errs)
	}
	it := New(dir)
	it.WorkDir = dir
	addr := make(chan string, 1)
	done := make(chan error, 1)
	it.OnServe = func(a string) { addr <- a }
	go func() { done <- it.Run(program) }()
	select {
	case a := <-addr:
		t.Cleanup(func() {
			it.StopServer()
			<-done
		})
		return a
	case err := <-done:
		t.Fatalf("the program ended: %v", err)
	case <-time.After(5 * time.Second):
		t.Fatal("the server didn't start")
	}
	return ""
}

func get(t *testing.T, method, url, body, contentType string) (int, string, string) {
	t.Helper()
	req, _ := http.NewRequest(method, url, strings.NewReader(body))
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, resp.Header.Get("Content-Type"), string(data)
}

const serverApp = `import server
serverlog = false
def home[req]
    return "<h1>Hello!</h1>"
def [end]
def showuser[req]
    return map ["id": id of params of req, "q": query of req, "method": method of req]
def [end]
def adduser[req]
    return reply[201, map ["made": name of json of req], map ["X-Made": "yes"]]
def [end]
def signup[req]
    return "thanks " + who of form of req
def [end]
def old[]
    return redirect["/"]
def [end]
def broken[req]
    return 1 / 0
def [end]
def nothing[req]
    return none
def [end]
def rest[req]
    p = params of req
    return p at get["*"]
def [end]
upper = scroll path of here, at upper .
app = map [
    "GET /": home,
    "GET /users/:id": showuser,
    "GET /users/new": "the form",
    "POST /users": adduser,
    "POST /signup": signup,
    "GET /old": old,
    "GET /broken": broken,
    "DELETE /nothing": nothing,
    "GET /files/*": rest,
    "GET /shout": upper,
    "GET /static/*": "public"
]
serve[app, 0]`

func TestServer(t *testing.T) {
	dir := t.TempDir()
	os.Mkdir(filepath.Join(dir, "public"), 0o755)
	os.WriteFile(filepath.Join(dir, "public", "a.txt"), []byte("hello file"), 0o644)
	os.WriteFile(filepath.Join(dir, "secret.txt"), []byte("secret"), 0o644)
	base := startServer(t, dir, serverApp)

	cases := []struct {
		method, path, body, ctype string
		status                    int
		wantType, wantBody        string
	}{
		{"GET", "/", "", "", 200, "text/html", "<h1>Hello!</h1>"},
		{"GET", "/users/7?x=1&x=2", "", "", 200, "application/json", `{"id":"7","q":{"x":["1","2"]},"method":"GET"}`},
		{"GET", "/users/new", "", "", 200, "text/plain", "the form"}, // a fixed part beats :id
		{"POST", "/users", `{"name":"Bo"}`, "application/json", 201, "application/json", `{"made":"Bo"}`},
		{"POST", "/signup", "who=Ann", "application/x-www-form-urlencoded", 200, "text/plain", "thanks Ann"},
		{"GET", "/broken", "", "", 500, "text/plain", "server error"},
		{"GET", "/", "", "", 200, "text/html", "<h1>Hello!</h1>"}, // still running after the 500
		{"DELETE", "/nothing", "", "", 204, "", ""},
		{"GET", "/files/a/b.txt", "", "", 200, "text/plain", "a/b.txt"},
		{"GET", "/shout", "", "", 200, "text/plain", "/SHOUT"}, // a saved scroll as a handler
		{"GET", "/static/a.txt", "", "", 200, "text/plain", "hello file"},
		{"GET", "/static/../secret.txt", "", "", 404, "text/plain", "not found"},
		{"GET", "/static/%2e%2e/secret.txt", "", "", 404, "text/plain", "not found"},
		{"GET", "/nope", "", "", 404, "text/plain", "not found"},
		{"PUT", "/users/3", "", "", 405, "text/plain", "method not allowed"},
	}
	for _, c := range cases {
		status, ctype, body := get(t, c.method, base+c.path, c.body, c.ctype)
		if status != c.status || !strings.HasPrefix(ctype, c.wantType) || body != c.wantBody {
			t.Errorf("%s %s: got %d %q %q, want %d %q %q", c.method, c.path, status, ctype, body, c.status, c.wantType, c.wantBody)
		}
	}
	// redirect: 302 with Location (don't follow it).
	client := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	resp, err := client.Get(base + "/old")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != 302 || resp.Header.Get("Location") != "/" {
		t.Errorf("redirect: %d %q", resp.StatusCode, resp.Header.Get("Location"))
	}
	req, _ := http.NewRequest("POST", base+"/users", strings.NewReader(`{"name":"Cy"}`))
	req.Header.Set("Content-Type", "application/json")
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.Header.Get("X-Made") != "yes" {
		t.Errorf("reply headers: %v", resp.Header)
	}
}

func TestServerExit(t *testing.T) {
	src := `import server
import system
serverlog = false
def stop[req]
    exit[3]
def [end]
serve[map ["GET /stop": stop], 0]`
	p := parser.New(lexer.New(src))
	program := p.ParseProgram()
	it := New(t.TempDir())
	addr := make(chan string, 1)
	done := make(chan error, 1)
	it.OnServe = func(a string) { addr <- a }
	go func() { done <- it.Run(program) }()
	base := <-addr
	http.Get(base + "/stop")
	select {
	case err := <-done:
		if ex, ok := err.(ExitRequest); !ok || ex.Code != 3 {
			t.Errorf("want exit 3, got %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("exit[] in a handler didn't stop the server")
	}
}

func TestServerMisuse(t *testing.T) {
	cases := []struct{ src, kind, want string }{
		{`serve[5]`, "type", "serve: the routes must be a map"},
		{`serve[map ["home": 1], 0]`, "server", `route "home": write it as "METHOD /path"`},
		{`serve[map ["GET /x/*": "nofolder"], 0]`, "server", `"nofolder" isn't a folder`},
		{`serve[map ["GET /": [a, b] give a], 0]`, "server", "a handler takes the request (one name), or nothing"},
		{`serve[map [], 70000]`, "server", "the port must be a whole number from 1 to 65535"},
		{`reply[700]`, "server", "the status must be a whole number from 100 to 599"},
		{`redirect["/", 200]`, "server", "the status must be 300 to 399"},
		{`reply[200, "x", 5]`, "type", "reply: the headers must be a map"},
	}
	for _, c := range cases {
		src := "import server\nsafe\n    x = " + c.src + "\nhandle [] e .\n    show kind of e, \"|\", message of e .\nsafe [end]"
		got, err := run(t, src, "")
		if err != nil {
			t.Errorf("%s: %v", c.src, err)
			continue
		}
		if !strings.HasPrefix(got, c.kind+"|") || !strings.Contains(got, c.want) {
			t.Errorf("%s: got %q, want kind %s and %q", c.src, got, c.kind, c.want)
		}
	}
	got, _ := run(t, "import server\nr = reply[404, \"gone\"]\nshow r, \"|\", status of r, \"|\", typeof[r], \"|\", serverlog, \"|\", serverhost .", "")
	if want := `Reply { status: 404, body: "gone", headers: {  } }|404|Reply|true|localhost`; strings.TrimSpace(got) != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
