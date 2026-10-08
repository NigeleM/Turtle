package evaluator

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"Turtle/object"
)

// The "server" builtin module: a web server. Routes are a map from
// "METHOD /path" to what answers it; handlers are ordinary functions.
//
//	def showuser[req]
//	    id = id of params of req
//	    return map ["id": id]               // a map or list goes as JSON
//	def [end]
//
//	app = map [
//	    "GET /": home,
//	    "GET /users/:id": showuser,         // :id lands in params
//	    "POST /users": adduser,
//	    "GET /static/*": "public"           // text on a /* route: a folder's files
//	]
//	serve[app, 8080]                        // or: app serve 8080
//
// A handler gets the request as a map (method, path, params, query,
// headers, body, json, form, ip) and gives back text (a page), a map or
// list (JSON), reply[status, body, headers] or redirect[address]. A
// handler's error answers 500 and is shown; the server goes on. Requests
// take turns: Turtle code runs one line at a time.

const (
	serverLogName  = "serverlog"
	serverHostName = "serverhost"
)

// replyShape is reply's assembled type: Reply { status, body, headers }.
var replyShape = &object.Shape{Name: "Reply", Fields: []string{"status", "body", "headers"}}

// defineServerSettings gives the importing file its server settings,
// unless it already has variables of those names.
func defineServerSettings(env *object.Environment) {
	if _, ok := env.Get(serverLogName); !ok {
		env.Set(serverLogName, object.Bool(true))
	}
	if _, ok := env.Get(serverHostName); !ok {
		env.Set(serverHostName, &object.String{Value: "localhost"})
	}
}

func (it *Interpreter) callServer(name string, args []object.Object, env *object.Environment) object.Object {
	switch name {
	case "reply":
		argCount(name, args, 1, 3, "a status (200, 404, ...), and optionally a body and a map of headers")
		status, ok := args[0].(*object.Integer)
		if !ok || status.Value < 100 || status.Value > 599 {
			fatalKind(kindServer, "reply: the status must be a whole number from 100 to 599 (200 ok, 404 not found, ...), got %s", object.Shown(args[0]))
		}
		var body object.Object = &object.String{Value: ""}
		if len(args) > 1 {
			body = args[1]
		}
		headers := object.Object(object.NewMap())
		if len(args) > 2 {
			if _, ok := args[2].(*object.Map); !ok {
				fatalf("reply: the headers must be a map, like map [\"Cache-Control\": \"no-store\"], got %s", typeName(args[2]))
			}
			headers = args[2]
		}
		return &object.Assembly{Shape: replyShape, Values: []object.Object{status, body, headers}}
	case "redirect":
		argCount(name, args, 1, 2, "an address, and optionally a status (302 by default, or 301, 303, 307, 308)")
		to := textArg(name, args, 0, "the address")
		code := int64(http.StatusFound)
		if len(args) == 2 {
			n, ok := args[1].(*object.Integer)
			if !ok || n.Value < 300 || n.Value > 399 {
				fatalKind(kindServer, "redirect: the status must be 300 to 399, got %s", object.Shown(args[1]))
			}
			code = n.Value
		}
		h := object.NewMap()
		h.Put(&object.String{Value: "Location"}, &object.String{Value: to})
		return &object.Assembly{Shape: replyShape, Values: []object.Object{object.Int(code), &object.String{Value: ""}, h}}
	case "serve":
		argCount(name, args, 1, 2, "a map of routes, and a port (8080 if left out)")
		routes, ok := args[0].(*object.Map)
		if !ok {
			fatalf("serve: the routes must be a map, like map [\"GET /\": home], got %s", typeName(args[0]))
		}
		port := int64(8080)
		if len(args) == 2 {
			n, ok := args[1].(*object.Integer)
			if !ok || n.Value < 0 || n.Value > 65535 {
				fatalKind(kindServer, "serve: the port must be a whole number from 1 to 65535, got %s", object.Shown(args[1]))
			}
			port = n.Value
		}
		it.serve(it.parseRoutes(routes), port, env)
		return object.NoneValue
	}
	fatalKind(kindName, "no server function %q", name)
	return nil
}

// route is one "METHOD /path" and what answers it.
type route struct {
	method   string // "" for any method
	parts    []string
	wildcard bool // ends with /*
	key      string
	answer   object.Object
}

func (it *Interpreter) parseRoutes(m *object.Map) []route {
	var rs []route
	for _, k := range m.Keys {
		ks, ok := m.KeyOf(k).(*object.String)
		if !ok {
			fatalKind(kindServer, "serve: a route is text like \"GET /users/:id\", got %s", object.Shown(m.KeyOf(k)))
		}
		method, path := "", strings.TrimSpace(ks.Value)
		if f := strings.Fields(path); len(f) == 2 {
			method, path = strings.ToUpper(f[0]), f[1]
		}
		if !strings.HasPrefix(path, "/") {
			fatalKind(kindServer, "serve: route %q: write it as \"METHOD /path\", like \"GET /\" or \"POST /users\"", ks.Value)
		}
		r := route{method: method, key: ks.Value, answer: m.Values[k]}
		for _, p := range strings.Split(strings.Trim(path, "/"), "/") {
			if p != "" {
				r.parts = append(r.parts, p)
			}
		}
		if n := len(r.parts); n > 0 && r.parts[n-1] == "*" {
			r.wildcard, r.parts = true, r.parts[:n-1]
		}
		if fn, ok := r.answer.(*object.Function); ok && fn.Shape == nil && fn.Scroll == nil && len(fn.Parameters) > 1 {
			fatalKind(kindServer, "serve: route %q: a handler takes the request (one name), or nothing, got a function of %d", ks.Value, len(fn.Parameters))
		}
		if s, ok := r.answer.(*object.String); ok && r.wildcard {
			info, err := os.Stat(it.resolvePath(s.Value))
			if err != nil || !info.IsDir() {
				fatalKind(kindServer, "serve: route %q: %q isn't a folder (a text on a /* route is the folder whose files it serves)", ks.Value, s.Value)
			}
		}
		rs = append(rs, r)
	}
	// The most exact route wins: fixed parts before :params before /*.
	sort.SliceStable(rs, func(i, j int) bool { return routeScore(rs[i]) > routeScore(rs[j]) })
	return rs
}

func routeScore(r route) int {
	score := 0
	for _, p := range r.parts {
		score += 2
		if !strings.HasPrefix(p, ":") {
			score++
		}
	}
	if r.wildcard {
		score -= 1
	}
	if r.method != "" {
		score++
	}
	return score*2 + len(r.parts)
}

// match reports whether r answers path, and its params.
func (r route) match(path []string) (map[string]string, bool) {
	if len(path) < len(r.parts) || !r.wildcard && len(path) != len(r.parts) {
		return nil, false
	}
	params := map[string]string{}
	for i, p := range r.parts {
		switch {
		case strings.HasPrefix(p, ":"):
			params[p[1:]] = path[i]
		case p != path[i]:
			return nil, false
		}
	}
	if r.wildcard {
		params["*"] = strings.Join(path[len(r.parts):], "/")
	}
	return params, true
}

// serve answers requests until the program is stopped (Ctrl+C), or a
// handler calls exit[].
func (it *Interpreter) serve(routes []route, port int64, env *object.Environment) {
	host := "localhost"
	if v, ok := env.Get(serverHostName); ok {
		s, isText := v.(*object.String)
		if !isText {
			fatalf("serverhost must be text, like \"localhost\" or \"0.0.0.0\", got %s", typeName(v))
		}
		host = s.Value
	}
	log := true
	if v, ok := env.Get(serverLogName); ok {
		log = isTruthy(v)
	}
	ln, err := net.Listen("tcp", net.JoinHostPort(host, strconv.FormatInt(port, 10)))
	if err != nil {
		if errors.Is(err, syscall.EADDRINUSE) {
			fatalKind(kindServer, "serve: port %d is in use (is another server running?); pick another port", port)
		}
		fatalKind(kindServer, "serve: %v", err)
	}
	addr := "http://" + ln.Addr().String()
	if host == "localhost" {
		addr = fmt.Sprintf("http://localhost:%d", ln.Addr().(*net.TCPAddr).Port)
	}
	fmt.Printf("serving on %s (Ctrl+C stops it)\n", addr)

	var turn sync.Mutex // Turtle code runs one request at a time
	var stopWith any    // exit[] from a handler: stops the server, then the program
	srv := &http.Server{ReadHeaderTimeout: 30 * time.Second}
	file, line := currentFile, currentLine
	srv.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		turn.Lock()
		defer turn.Unlock()
		currentFile, currentLine = file, line
		status := it.answer(w, r, routes, func(p any) {
			stopWith = p
			go srv.Close()
		})
		if log {
			fmt.Printf("%s %s %d %s\n", r.Method, r.URL.RequestURI(), status, fmtDuration(time.Since(start)))
		}
	})
	it.serverMu.Lock()
	it.server = srv
	it.serverMu.Unlock()
	if it.OnServe != nil {
		it.OnServe(addr)
	}
	err = srv.Serve(ln)
	turn.Lock() // wait for a request still being answered
	turn.Unlock()
	currentFile, currentLine = file, line
	if stopWith != nil {
		panic(stopWith)
	}
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		fatalKind(kindServer, "serve: %v", err)
	}
}

// StopServer stops a running serve (for tests, and the REPL).
func (it *Interpreter) StopServer() {
	it.serverMu.Lock()
	defer it.serverMu.Unlock()
	if it.server != nil {
		it.server.Close()
	}
}

// answer finds the route for a request, runs it, and writes the
// response, giving back its status.
func (it *Interpreter) answer(w http.ResponseWriter, r *http.Request, routes []route, stop func(any)) (status int) {
	var path []string
	for _, p := range strings.Split(strings.Trim(r.URL.Path, "/"), "/") {
		if p != "" {
			path = append(path, p)
		}
	}
	allowed := false
	for _, rt := range routes {
		params, ok := rt.match(path)
		if !ok {
			continue
		}
		if rt.method != "" && rt.method != r.Method && !(rt.method == "GET" && r.Method == "HEAD") {
			allowed = true
			continue
		}
		if s, ok := rt.answer.(*object.String); ok && rt.wildcard {
			return it.serveFile(w, r, s.Value, params["*"])
		}
		return it.runHandler(w, r, rt, params, stop)
	}
	if allowed {
		return writeText(w, http.StatusMethodNotAllowed, "method not allowed")
	}
	return writeText(w, http.StatusNotFound, "not found")
}

// serveFile answers with a file from folder, never outside it.
func (it *Interpreter) serveFile(w http.ResponseWriter, r *http.Request, folder, rest string) int {
	root, err := filepath.Abs(it.resolvePath(folder))
	if err != nil {
		return writeText(w, http.StatusInternalServerError, "server error")
	}
	full := filepath.Join(root, filepath.FromSlash(rest))
	if rel, err := filepath.Rel(root, full); err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return writeText(w, http.StatusNotFound, "not found")
	}
	info, err := os.Stat(full)
	if err == nil && info.IsDir() {
		full = filepath.Join(full, "index.html")
		info, err = os.Stat(full)
	}
	if err != nil || info.IsDir() {
		return writeText(w, http.StatusNotFound, "not found")
	}
	sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
	http.ServeFile(sw, r, full)
	return sw.status
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (s *statusWriter) WriteHeader(code int) {
	s.status = code
	s.ResponseWriter.WriteHeader(code)
}

// runHandler runs a route's answer and writes what it gives back. An
// error in it answers 500 and is shown; exit[] stops the server.
func (it *Interpreter) runHandler(w http.ResponseWriter, r *http.Request, rt route, params map[string]string, stop func(any)) (status int) {
	defer func() {
		p := recover()
		if p == nil {
			return
		}
		if fe, ok := p.(fatalError); ok && !fe.parse {
			fmt.Fprintf(os.Stderr, "%s %s: %s\n", r.Method, r.URL.Path, fe.msg)
			status = writeText(w, http.StatusInternalServerError, "server error")
			return
		}
		status = writeText(w, http.StatusInternalServerError, "server error")
		stop(p)
	}()
	var result object.Object
	switch a := rt.answer.(type) {
	case *object.Function:
		var args []object.Object
		if a.Scroll != nil || a.Shape != nil || len(a.Parameters) == 1 {
			req, err := requestMap(r, params)
			if err != nil {
				return writeText(w, http.StatusBadRequest, err.Error())
			}
			args = []object.Object{req}
		}
		result = it.callFunction(a, rt.key, args)
	default:
		result = rt.answer // a fixed answer: "GET /health": "ok"
	}
	return it.writeResult(w, r, result)
}

const maxRequestBody = 10 << 20

// requestMap is the request as a map for a handler.
func requestMap(r *http.Request, params map[string]string) (*object.Map, error) {
	str := func(s string) object.Object { return &object.String{Value: s} }
	m := object.NewMap()
	m.Put(str("method"), str(r.Method))
	m.Put(str("path"), str(r.URL.Path))
	pm := object.NewMap()
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		pm.Put(str(k), str(params[k]))
	}
	m.Put(str("params"), pm)
	m.Put(str("query"), valuesMap(r.URL.Query()))
	m.Put(str("headers"), responseHeaders(r.Header))
	data, err := io.ReadAll(io.LimitReader(r.Body, maxRequestBody+1))
	if err != nil {
		return nil, fmt.Errorf("couldn't read the request: %v", err)
	}
	if len(data) > maxRequestBody {
		return nil, errors.New("the request is too large (10 MB at most)")
	}
	body := string(data)
	m.Put(str("body"), str(body))
	var js object.Object = object.NoneValue
	ct := strings.ToLower(r.Header.Get("Content-Type"))
	if strings.Contains(ct, "json") && json.Valid(data) {
		js = parseJSON("request", body)
	}
	m.Put(str("json"), js)
	form := object.NewMap()
	if strings.HasPrefix(ct, "application/x-www-form-urlencoded") || strings.HasPrefix(ct, "multipart/form-data") {
		r.Body = io.NopCloser(strings.NewReader(body))
		if strings.HasPrefix(ct, "multipart/") {
			_ = r.ParseMultipartForm(maxRequestBody)
		} else {
			_ = r.ParseForm()
		}
		form = valuesMap(r.PostForm)
	}
	m.Put(str("form"), form)
	ip := r.RemoteAddr
	if h, _, err := net.SplitHostPort(ip); err == nil {
		ip = h
	}
	m.Put(str("ip"), str(ip))
	return m, nil
}

// valuesMap is query or form values as a map: text, or a list of text
// when a name comes more than once.
func valuesMap(v map[string][]string) *object.Map {
	keys := make([]string, 0, len(v))
	for k := range v {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	m := object.NewMap()
	for _, k := range keys {
		vals := v[k]
		if len(vals) == 1 {
			m.Put(&object.String{Value: k}, &object.String{Value: vals[0]})
			continue
		}
		l := &object.List{}
		for _, s := range vals {
			l.Elements = append(l.Elements, &object.String{Value: s})
		}
		m.Put(&object.String{Value: k}, l)
	}
	return m
}

// writeResult writes what a handler gave back.
func (it *Interpreter) writeResult(w http.ResponseWriter, r *http.Request, v object.Object) int {
	status := http.StatusOK
	if a, ok := v.(*object.Assembly); ok && a.Shape == replyShape {
		status = int(a.Values[0].(*object.Integer).Value)
		if h, ok := a.Values[2].(*object.Map); ok {
			for _, k := range h.Keys {
				w.Header().Set(h.KeyOf(k).Inspect(), headerText(h.Values[k]))
			}
		}
		v = a.Values[1]
	}
	switch x := v.(type) {
	case *object.None:
		if status == http.StatusOK {
			status = http.StatusNoContent
		}
		w.WriteHeader(status)
		return status
	case *object.String:
		if w.Header().Get("Content-Type") == "" {
			ct := "text/plain; charset=utf-8"
			if strings.HasPrefix(strings.TrimSpace(x.Value), "<") {
				ct = "text/html; charset=utf-8"
			}
			w.Header().Set("Content-Type", ct)
		}
		w.WriteHeader(status)
		if r.Method != http.MethodHead {
			io.WriteString(w, x.Value)
		}
		return status
	case *object.Map, *object.List, *object.Set, *object.Assembly:
		data := toJSON("serve", x, "")
		if w.Header().Get("Content-Type") == "" {
			w.Header().Set("Content-Type", "application/json")
		}
		w.WriteHeader(status)
		if r.Method != http.MethodHead {
			w.Write(data)
		}
		return status
	case *object.Error:
		fmt.Fprintf(os.Stderr, "%s %s: %s\n", r.Method, r.URL.Path, x.Inspect())
		return writeText(w, http.StatusInternalServerError, "server error")
	}
	return it.writeResult(w, r, &object.String{Value: v.Inspect()})
}

func headerText(v object.Object) string {
	if s, ok := v.(*object.String); ok {
		return s.Value
	}
	return v.Inspect()
}

func writeText(w http.ResponseWriter, status int, text string) int {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(status)
	io.WriteString(w, text)
	return status
}
