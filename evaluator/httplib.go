package evaluator

import (
	"bytes"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"Turtle/object"
)

// The "http" builtin module: web requests, on Go's standard library.
//
//	body = http_get["https://api.example.com/users"]     // text; pass to load
//	reply = http_post[url, map ["name": "Ann"]]          // a map or list is sent as JSON
//	r = http_request["PUT", url, body, map ["Authorization": "Bearer x"]]
//	code = r at get["status"]                            // r has status, body, headers
//
// http_get and http_post fail on a network problem or a 4xx/5xx status;
// http_request only fails on a network problem and hands back any status.
// Failures are errors of kind http.
func (it *Interpreter) callHTTP(name string, args []object.Object) object.Object {
	switch name {
	case "http_get":
		requireFuncArgs(name, args, 1)
		return &object.String{Value: it.httpSimple(name, "GET", asStringArg(name, args[0]), nil)}
	case "http_post":
		requireFuncArgs(name, args, 2)
		return &object.String{Value: it.httpSimple(name, "POST", asStringArg(name, args[0]), args[1])}
	case "http_request":
		if len(args) < 2 || len(args) > 4 {
			fatalf("'http_request' expects 2 to 4 arguments (method, url [, body [, headers]]), got %d", len(args))
		}
		method := strings.ToUpper(asStringArg(name, args[0]))
		var body object.Object
		if len(args) >= 3 {
			body = args[2]
		}
		var headers *object.Map
		if len(args) == 4 {
			m, ok := args[3].(*object.Map)
			if !ok {
				fatalf("'http_request' headers must be a map, e.g. map [\"Accept\": \"application/json\"], got %s", typeName(args[3]))
			}
			headers = m
		}
		resp, text := doHTTP(name, method, asStringArg(name, args[1]), body, headers)
		result := object.NewMap()
		result.Put(&object.String{Value: "status"}, &object.Integer{Value: int64(resp.StatusCode)})
		result.Put(&object.String{Value: "body"}, &object.String{Value: text})
		result.Put(&object.String{Value: "headers"}, responseHeaders(resp.Header))
		return result
	}
	fatalKind(kindName, "no http function %q", name)
	return nil
}

// httpSimple is http_get / http_post: the body text, or an error for a
// 4xx/5xx status.
func (it *Interpreter) httpSimple(who, method, address string, body object.Object) string {
	resp, text := doHTTP(who, method, address, body, nil)
	if resp.StatusCode >= 400 {
		snippet := strings.TrimSpace(text)
		if len(snippet) > 200 {
			snippet = snippet[:200] + "..."
		}
		if snippet != "" {
			snippet = ": " + snippet
		}
		fatalKind(kindHTTP, "%s %s: %s%s", who, address, resp.Status, snippet)
	}
	return text
}

// httpClient is shared so connections are reused between requests.
var httpClient = &http.Client{Timeout: 30 * time.Second}

// maxResponse caps how much of a response is read, so a huge download
// can't exhaust memory.
const maxResponse = 64 << 20

func doHTTP(who, method, address string, body object.Object, headers *object.Map) (*http.Response, string) {
	u, err := url.Parse(address)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		fatalKind(kindHTTP, "%s: %q isn't a web address; it needs to start with http:// or https://", who, address)
	}
	var reader io.Reader
	contentType := ""
	switch b := body.(type) {
	case nil, *object.None:
	case *object.String:
		reader = strings.NewReader(b.Value)
		contentType = "text/plain; charset=utf-8"
	case *object.List, *object.Map, *object.Set, *object.Assembly:
		reader = bytes.NewReader(toJSON(who, b, ""))
		contentType = "application/json"
	default:
		fatalf("'%s' body must be text, or a map or list to send as JSON, got %s", who, typeName(body))
	}
	req, err := http.NewRequest(method, address, reader)
	if err != nil {
		fatalKind(kindHTTP, "%s %s: %v", who, address, err)
	}
	req.Header.Set("User-Agent", "Turtle")
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	if headers != nil {
		for _, k := range headers.Keys {
			v := headers.Values[k]
			req.Header.Set(headers.KeyOf(k).Inspect(), v.Inspect())
		}
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		fatalKind(kindHTTP, "%s %s: %s", who, address, networkProblem(err))
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxResponse))
	if err != nil {
		fatalKind(kindHTTP, "%s %s: reading the response: %s", who, address, networkProblem(err))
	}
	return resp, string(data)
}

// networkProblem says briefly why a request failed, without Go's
// repetition of the method and address.
func networkProblem(err error) string {
	if ue, ok := err.(*url.Error); ok {
		if ue.Timeout() {
			return "no answer within 30 seconds"
		}
		err = ue.Err
	}
	msg := err.Error()
	switch {
	case strings.Contains(msg, "connection refused"), strings.Contains(msg, "actively refused"): // Windows says the latter
		return "connection refused (is the server running?)"
	case strings.Contains(msg, "no such host"):
		return "no such host"
	}
	return msg
}

// responseHeaders is a map of lowercase header names to their values
// (several values joined with ", "), sorted by name.
func responseHeaders(h http.Header) *object.Map {
	names := make([]string, 0, len(h))
	for k := range h {
		names = append(names, k)
	}
	sort.Strings(names)
	m := object.NewMap()
	for _, k := range names {
		m.Put(&object.String{Value: strings.ToLower(k)}, &object.String{Value: strings.Join(h[k], ", ")})
	}
	return m
}
