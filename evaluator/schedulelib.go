package evaluator

import (
	"bytes"
	"errors"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"

	"Turtle/mysql"
	"Turtle/object"
	"Turtle/postgres"
)

// The "schedule" builtin module: many web requests, shell commands or
// database queries at once. Only the work itself runs side by side, inside
// Go; Turtle code still runs one line at a time.
//
//	pages = fetchall[urls]                    // http_get each; texts, in order
//	pages = urls fetchall                     // the sentence form
//	outs = runall[list ["git pull", "make"]]  // [{ output, errors, code }, ...]
//	results = queryall[db, list ["SELECT ...", "SELECT ..."]]
//	pages = fetchall[urls, map ["limit": 10, "skip_errors": true]]
//
// It works like a semaphore in an async pool: at most schedulelimit items
// run at once (5 unless changed; none means no cap), and each one that
// finishes starts the next straight away. Results come back in the list's
// order, whatever order they finish in.
//
// The first failure stops the call with that item's error, of its own
// kind (http, sql, schedule): the running items finish, the waiting ones
// never start. With skipschedule_error = true, a failed item becomes none
// and the rest carry on. "import schedule" sets both in the importing
// file; a call's own map ["limit": n, "skip_errors": b] overrides them
// for that call.

const (
	scheduleLimitName = "schedulelimit"
	scheduleSkipName  = "skipschedule_error"
	defaultSchedule   = 5
)

// defineScheduleSettings gives the importing file its schedule settings,
// unless it already has variables of those names.
func defineScheduleSettings(env *object.Environment) {
	if _, ok := env.Get(scheduleLimitName); !ok {
		env.Set(scheduleLimitName, &object.Integer{Value: defaultSchedule})
	}
	if _, ok := env.Get(scheduleSkipName); !ok {
		env.Set(scheduleSkipName, &object.Boolean{Value: false})
	}
}

// scheduleOptions are a call's limit (0 for no cap) and whether a failed
// item becomes none.
type scheduleOptions struct {
	limit int
	skip  bool
}

// scheduleSettings reads the file's settings, then the call's own map.
func scheduleSettings(fn string, env *object.Environment, args []object.Object, at int) scheduleOptions {
	o := scheduleOptions{limit: defaultSchedule}
	if v, ok := env.Get(scheduleLimitName); ok {
		o.limit = scheduleLimit(v, scheduleLimitName)
	}
	if v, ok := env.Get(scheduleSkipName); ok {
		o.skip = scheduleSkip(v, scheduleSkipName)
	}
	if len(args) <= at {
		return o
	}
	m, ok := args[at].(*object.Map)
	if !ok {
		fatalf("%s: the settings must be a map, e.g. map [\"limit\": 10, \"skip_errors\": true], got %s", fn, typeName(args[at]))
	}
	for _, k := range m.Keys {
		v := m.Values[k]
		switch key := m.KeyOf(k).Inspect(); key {
		case "limit":
			o.limit = scheduleLimit(v, fn+"'s limit")
		case "skip_errors":
			o.skip = scheduleSkip(v, fn+"'s skip_errors")
		default:
			fatalKind(kindSchedule, "%s: %q isn't a setting; use \"limit\" or \"skip_errors\"", fn, key)
		}
	}
	return o
}

// scheduleLimit reads how many may run at once: a whole number of 1 or
// more, or none for no cap (returned as 0).
func scheduleLimit(v object.Object, what string) int {
	switch n := v.(type) {
	case *object.None:
		return 0
	case *object.Integer:
		if n.Value >= 1 {
			if n.Value > 1<<20 {
				return 0
			}
			return int(n.Value)
		}
		fatalKind(kindSchedule, "%s must be 1 or more (or none for no limit), got %d", what, n.Value)
	}
	fatalf("%s must be a whole number, or none for no limit, got %s", what, typeName(v))
	return 0
}

func scheduleSkip(v object.Object, what string) bool {
	b, ok := v.(*object.Boolean)
	if !ok {
		fatalf("%s must be true or false, got %s", what, typeName(v))
	}
	return b.Value
}

// scheduleList is the list of items a call works through.
func scheduleList(fn string, v object.Object, what string) []object.Object {
	l, ok := v.(*object.List)
	if !ok {
		fatalf("%s needs a list of %s, got %s", fn, what, typeName(v))
	}
	return l.Elements
}

// scheduleTexts is scheduleList for items that must all be text.
func scheduleTexts(fn string, v object.Object, what string) []string {
	elems := scheduleList(fn, v, what)
	out := make([]string, len(elems))
	for i, e := range elems {
		s, ok := e.(*object.String)
		if !ok {
			fatalf("%s: item %d must be text, got %s", fn, i, typeName(e))
		}
		out[i] = s.Value
	}
	return out
}

// jobResult is one item's outcome: its value, or the error it stopped
// with. bug holds any other panic, a fault in the interpreter itself,
// which is raised again on the main goroutine rather than lost.
type jobResult struct {
	value any
	err   *fatalError
	bug   any
	done  bool
}

// scheduleRun runs job for items 0..n-1, at most o.limit at a time, on
// that many workers: each takes the next waiting item as soon as it's
// free, which is what makes it a semaphore and not a batch. worker
// numbers each worker 0..limit-1, so a job can keep a connection per
// worker. Unless o.skip, the first failure stops new items starting.
//
// A job reports errors the usual way (fatalKind panics); each one is
// caught on its own goroutine. Jobs must not touch the interpreter's
// state: they only read their inputs, and the main goroutine turns their
// results into Turtle values afterwards.
func scheduleRun(n int, o scheduleOptions, job func(i, worker int) any) []jobResult {
	results := make([]jobResult, n)
	workers := o.limit
	if workers == 0 || workers > n {
		workers = n
	}
	var stopped atomic.Bool
	next := make(chan int)
	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			for i := range next {
				if stopped.Load() { // handed over just as another failed
					continue
				}
				results[i] = runJob(i, w, job)
				if results[i].err != nil && !o.skip || results[i].bug != nil {
					stopped.Store(true)
				}
			}
		}(w)
	}
	for i := 0; i < n && !stopped.Load(); i++ {
		next <- i
	}
	close(next)
	wg.Wait()
	return results
}

func runJob(i, w int, job func(i, worker int) any) (r jobResult) {
	defer func() {
		if p := recover(); p != nil {
			if fe, ok := p.(fatalError); ok {
				r = jobResult{err: &fe, done: true}
			} else {
				r = jobResult{bug: p, done: true}
			}
		}
	}()
	return jobResult{value: job(i, w), done: true}
}

// scheduleResults turns the outcomes into a list, through value for the
// ones that worked. A failure is none when skipping, and otherwise stops
// the program with the failed item's error (the earliest in the list, if
// several failed), naming which item it was.
func scheduleResults(fn string, results []jobResult, o scheduleOptions, value func(any) object.Object) *object.List {
	for _, r := range results {
		if r.bug != nil {
			panic(r.bug)
		}
	}
	out := &object.List{Elements: make([]object.Object, len(results))}
	for i, r := range results {
		switch {
		case r.err != nil && !o.skip:
			fatalKind(r.err.kind, "%s item %d: %s", fn, i, r.err.text)
		case r.err != nil || !r.done:
			out.Elements[i] = object.NoneValue
		default:
			out.Elements[i] = value(r.value)
		}
	}
	return out
}

func (it *Interpreter) callSchedule(name string, args []object.Object, env *object.Environment) object.Object {
	switch name {
	case "fetchall":
		argCount(name, args, 1, 2, "a list of web addresses, and optionally a map of settings")
		urls := scheduleTexts(name, args[0], "web addresses")
		o := scheduleSettings(name, env, args, 1)
		results := scheduleRun(len(urls), o, func(i, _ int) any {
			return it.httpSimple("http_get", "GET", urls[i], nil)
		})
		return scheduleResults(name, results, o, func(v any) object.Object {
			return &object.String{Value: v.(string)}
		})

	case "runall":
		argCount(name, args, 1, 2, "a list of shell commands, and optionally a map of settings")
		cmds := scheduleTexts(name, args[0], "shell commands")
		o := scheduleSettings(name, env, args, 1)
		dir := it.Dir
		results := scheduleRun(len(cmds), o, func(i, _ int) any {
			return runCommand(cmds[i], dir)
		})
		return scheduleResults(name, results, o, func(v any) object.Object {
			c := v.(commandResult)
			m := object.NewMap()
			m.Put(&object.String{Value: "output"}, &object.String{Value: c.output})
			m.Put(&object.String{Value: "errors"}, &object.String{Value: c.errors})
			m.Put(&object.String{Value: "code"}, &object.Integer{Value: int64(c.code)})
			return m
		})

	case "queryall":
		argCount(name, args, 2, 3, "a database, a list of queries, and optionally a map of settings")
		d, ok := args[0].(*object.Database)
		if !ok {
			fatalf("queryall needs a database (from sql_open), got %s", typeName(args[0]))
		}
		asDatabaseArg(name, d) // closed?
		if d.Address == "" {
			fatalKind(kindSchedule, "queryall works with postgres and mysql databases; for SQLite, use sql_query one query at a time")
		}
		queries := scheduleQueries(name, args[1])
		o := scheduleSettings(name, env, args, 2)
		return it.queryAll(name, d.Address, queries, o)
	}
	fatalKind(kindName, "no schedule function %q", name)
	return nil
}

// sqlJob is one query and the values for its ? placeholders.
type sqlJob struct {
	query  string
	params []any
}

// scheduleQueries reads queryall's list: each item is a query's text, or
// a list of the text and a list of values for its ? placeholders.
func scheduleQueries(fn string, v object.Object) []sqlJob {
	elems := scheduleList(fn, v, "queries")
	jobs := make([]sqlJob, len(elems))
	for i, e := range elems {
		switch q := e.(type) {
		case *object.String:
			jobs[i] = sqlJob{query: q.Value}
		case *object.List:
			if len(q.Elements) != 2 {
				fatalf("%s: item %d must be a query, or list [query, list of values], got a list of %d", fn, i, len(q.Elements))
			}
			s, ok := q.Elements[0].(*object.String)
			if !ok {
				fatalf("%s: item %d's query must be text, got %s", fn, i, typeName(q.Elements[0]))
			}
			jobs[i] = sqlJob{query: s.Value, params: sqlParams(fn, q.Elements[1:])}
		default:
			fatalf("%s: item %d must be a query, or list [query, list of values], got %s", fn, i, typeName(e))
		}
	}
	return jobs
}

// queryResult is a query's columns and rows, before they become maps.
type queryResult struct {
	cols []string
	rows [][]any
}

// queryAll runs the queries over one new connection per worker, since a
// connection answers one query at a time; each is closed at the end. They
// don't see the program's own connection's open transaction.
func (it *Interpreter) queryAll(fn, address string, queries []sqlJob, o scheduleOptions) object.Object {
	workers := o.limit
	if workers == 0 || workers > len(queries) {
		workers = len(queries)
	}
	conns := make([]sqlConn, workers)
	defer func() {
		for _, c := range conns {
			if c != nil {
				c.Close()
			}
		}
	}()
	results := scheduleRun(len(queries), o, func(i, w int) any {
		if conns[w] == nil {
			conns[w] = openServer(fn, address)
		}
		q := queries[i]
		cols, rows, err := conns[w].Query(q.query, q.params)
		if err != nil {
			fatalKind(kindSQL, "%v", err)
		}
		return queryResult{cols, rows}
	})
	return scheduleResults(fn, results, o, func(v any) object.Object {
		r := v.(queryResult)
		return rowMaps(r.cols, r.rows)
	})
}

// openServer opens another connection to a PostgreSQL or MySQL server.
func openServer(fn, address string) sqlConn {
	if strings.HasPrefix(address, "mysql://") || strings.HasPrefix(address, "mariadb://") {
		db, err := mysql.Open(address)
		if err != nil {
			fatalKind(kindSQL, "%s: opening another connection: %v", fn, err)
		}
		return mysqlConn{db}
	}
	db, err := postgres.Open(address)
	if err != nil {
		fatalKind(kindSQL, "%s: opening another connection: %v", fn, err)
	}
	return postgresConn{db}
}

// commandResult is what one shell command printed, and its exit code.
type commandResult struct {
	output, errors string
	code           int
}

// runCommand runs a command through the shell (sh -c, or cmd /C on
// Windows) in the script's folder, capturing what it prints. A nonzero
// exit code isn't an error: it's in the result. Trailing newlines are
// dropped, as the shell's $(...) does.
func runCommand(command, dir string) commandResult {
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/C", command)
	} else {
		cmd = exec.Command("sh", "-c", command)
	}
	cmd.Dir = dir
	var out, errs bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errs
	err := cmd.Run()
	code := 0
	if err != nil {
		var ee *exec.ExitError
		if !errors.As(err, &ee) {
			fatalKind(kindSchedule, "%q couldn't start: %v", command, err)
		}
		code = ee.ExitCode()
	}
	return commandResult{output: strings.TrimRight(out.String(), "\r\n"), errors: strings.TrimRight(errs.String(), "\r\n"), code: code}
}
