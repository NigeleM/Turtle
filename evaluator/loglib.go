package evaluator

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"Turtle/ast"
	"Turtle/object"
)

// The log library (import log): log lines with a level, a time and where
// they came from, to the console (stderr) and/or a file, and a copy of
// what show and warn print.
//
//	log "server started on port ", port .
//	log warn "disk at ", pct, "%" .
//	log info "login", map ["user": name] .     a map adds fields: user=ann
//
// The settings are variables "import log" makes; set them like any
// variable (in a function, they hold for that function's code).

const (
	logLevelName   = "loglevel"
	logFileName    = "logfile"
	logConsoleName = "logconsole"
	logTimeName    = "logtime"
	logPartsName   = "logparts"
	logFormatName  = "logformat"
	logMaxSizeName = "logmaxsize"
	logKeepName    = "logkeep"
	outputFileName = "outputfile"
)

var logLevelRank = map[string]int{"debug": 0, "info": 1, "warn": 2, "error": 3, "off": 4}

// defineLogSettings gives the importing file its log settings, unless it
// already has variables of those names.
func defineLogSettings(env *object.Environment) {
	set := func(name string, v object.Object) {
		if _, ok := env.Get(name); !ok {
			env.Set(name, v)
		}
	}
	set(logLevelName, &object.String{Value: "info"})
	set(logFileName, object.NoneValue)
	set(logConsoleName, &object.Boolean{Value: true})
	set(logTimeName, &object.String{Value: "YYYY-MM-DD hh:mm:ss"})
	set(logPartsName, &object.List{Elements: []object.Object{
		&object.String{Value: "time"}, &object.String{Value: "level"}, &object.String{Value: "message"},
	}})
	set(logFormatName, &object.String{Value: "text"})
	set(logMaxSizeName, object.NoneValue)
	set(logKeepName, &object.Integer{Value: 3})
	set(outputFileName, object.NoneValue)
}

// logEntry is one log line before it's written.
type logEntry struct {
	level   string
	message string
	fields  *object.Map // from map values in the log statement
	file    string
	line    int
}

func (it *Interpreter) evalLog(s *ast.LogStatement, env *object.Environment) {
	e := logEntry{level: s.Level, fields: object.NewMap(), file: it.fileName(), line: currentLine}
	var msg strings.Builder
	for _, x := range s.Expressions {
		v := it.evalExpression(x, env)
		if m, ok := v.(*object.Map); ok {
			for _, k := range m.Keys {
				e.fields.Put(m.KeyOf(k), m.Values[k])
			}
			continue
		}
		msg.WriteString(v.Inspect())
	}
	e.message = msg.String()
	it.writeLog(env, e, true)
}

// fileName is the file the running code is in, for "file" and "where".
func (it *Interpreter) fileName() string {
	if currentFile != "" {
		return currentFile
	}
	if it.Script != "" {
		return it.Script
	}
	return "main"
}

// writeLog writes e to the console and the log file, as env's settings
// say, if its level is shown.
func (it *Interpreter) writeLog(env *object.Environment, e logEntry, console bool) {
	setting := func(name string) object.Object {
		v, _ := env.Get(name)
		return v
	}
	minLevel := "info"
	if v, ok := setting(logLevelName).(*object.String); ok {
		minLevel = v.Value
	}
	min, ok := logLevelRank[minLevel]
	if !ok {
		fatalf("loglevel must be \"debug\", \"info\", \"warn\", \"error\" or \"off\", got %s", object.Shown(setting(logLevelName)))
	}
	if logLevelRank[e.level] < min {
		return
	}
	line := it.formatLogLine(e, setting)
	if b, ok := setting(logConsoleName).(*object.Boolean); console && (!ok || b.Value) {
		fmt.Fprintln(os.Stderr, line)
	}
	switch f := setting(logFileName).(type) {
	case nil, *object.None:
	case *object.String:
		maxSize, keep := int64(-1), 3
		switch m := setting(logMaxSizeName).(type) {
		case nil, *object.None:
		case *object.Integer:
			if m.Value < 1 {
				fatalf("logmaxsize must be a number of bytes above 0, or none, got %s", m.Inspect())
			}
			maxSize = m.Value
		default:
			fatalf("logmaxsize must be a number of bytes, or none, got %s", object.Shown(m))
		}
		if k, ok := setting(logKeepName).(*object.Integer); ok {
			if k.Value < 0 || k.Value > 1000 {
				fatalf("logkeep must be from 0 to 1000, got %d", k.Value)
			}
			keep = int(k.Value)
		}
		it.appendLine(f.Value, line, maxSize, keep)
	default:
		fatalf("logfile must be a file's path, or none, got %s", object.Shown(f))
	}
}

// formatLogLine lays the entry out as text (logparts) or as JSON.
func (it *Interpreter) formatLogLine(e logEntry, setting func(string) object.Object) string {
	now := time.Now()
	stamp := ""
	switch t := setting(logTimeName).(type) {
	case nil, *object.None:
	case *object.String:
		stamp = formatDate(now, t.Value)
	default:
		fatalf("logtime must be a date pattern like \"YYYY-MM-DD hh:mm:ss\", or none, got %s", object.Shown(t))
	}
	format := "text"
	if f, ok := setting(logFormatName).(*object.String); ok {
		format = f.Value
	}
	switch format {
	case "json":
		obj := object.NewMap()
		if stamp != "" {
			obj.Put(&object.String{Value: "time"}, &object.String{Value: stamp})
		}
		obj.Put(&object.String{Value: "level"}, &object.String{Value: e.level})
		obj.Put(&object.String{Value: "message"}, &object.String{Value: e.message})
		obj.Put(&object.String{Value: "file"}, &object.String{Value: e.file})
		obj.Put(&object.String{Value: "line"}, &object.Integer{Value: int64(e.line)})
		for _, k := range e.fields.Keys {
			obj.Put(&object.String{Value: e.fields.KeyOf(k).Inspect()}, e.fields.Values[k])
		}
		return string(toJSON("log", obj, ""))
	case "text":
	default:
		fatalf("logformat must be \"text\" or \"json\", got %s", object.Shown(setting(logFormatName)))
	}
	parts := []string{"time", "level", "message"}
	if l, ok := setting(logPartsName).(*object.List); ok {
		parts = parts[:0]
		for _, p := range l.Elements {
			s, ok := p.(*object.String)
			if !ok {
				fatalf("logparts must be a list of words: time, level, message, file, line, where; got %s", object.Shown(p))
			}
			parts = append(parts, s.Value)
		}
	}
	var out []string
	for _, p := range parts {
		switch p {
		case "time":
			if stamp != "" {
				out = append(out, stamp)
			}
		case "level":
			out = append(out, fmt.Sprintf("%-5s", strings.ToUpper(e.level)))
		case "message":
			msg := e.message
			for _, k := range e.fields.Keys {
				msg += " " + e.fields.KeyOf(k).Inspect() + "=" + fieldText(e.fields.Values[k])
			}
			out = append(out, strings.TrimSpace(msg))
		case "file":
			out = append(out, e.file)
		case "line":
			out = append(out, strconv.Itoa(e.line))
		case "where":
			out = append(out, fmt.Sprintf("%s:%d", e.file, e.line))
		default:
			fatalf("logparts: %q isn't a part; use time, level, message, file, line or where", p)
		}
	}
	return strings.Join(out, " ")
}

// fieldText writes a field's value: text as it is, quoted if it has a
// space or an = in it, anything else as show prints it.
func fieldText(v object.Object) string {
	if s, ok := v.(*object.String); ok {
		if s.Value == "" || strings.ContainsAny(s.Value, " =\"\t\n") {
			return strconv.Quote(s.Value)
		}
		return s.Value
	}
	return v.Inspect()
}

// ---- files: log files and the copy of the console ----

type openFile struct {
	f    *os.File
	size int64
}

// appendLine adds line to the file at path, opening it once (it stays
// open until the program ends). With maxSize set, a file that would grow
// past it is rotated first: app.log becomes app.log.1, app.log.1 becomes
// app.log.2, ..., keeping keep of them.
func (it *Interpreter) appendLine(path, line string, maxSize int64, keep int) {
	full := it.resolvePath(path)
	of := it.openForAppend(path, full)
	data := line + "\n"
	if maxSize > 0 && of.size > 0 && of.size+int64(len(data)) > maxSize {
		of.f.Close()
		delete(it.files, full)
		os.Remove(fmt.Sprintf("%s.%d", full, keep))
		for n := keep - 1; n >= 1; n-- {
			os.Rename(fmt.Sprintf("%s.%d", full, n), fmt.Sprintf("%s.%d", full, n+1))
		}
		if keep > 0 {
			os.Rename(full, full+".1")
		} else {
			os.Remove(full)
		}
		of = it.openForAppend(path, full)
	}
	n, err := of.f.WriteString(data)
	if err != nil {
		fatalKind(kindFile, "%s: %s", path, fileProblem(err))
	}
	of.size += int64(n)
}

func (it *Interpreter) openForAppend(path, full string) *openFile {
	if it.files == nil {
		it.files = map[string]*openFile{}
	}
	if of, ok := it.files[full]; ok {
		return of
	}
	f, err := os.OpenFile(full, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		fatalKind(kindFile, "%s: %s", path, fileProblem(err))
	}
	of := &openFile{f: f}
	if info, err := f.Stat(); err == nil {
		of.size = info.Size()
	}
	it.files[full] = of
	return of
}

func (it *Interpreter) closeFiles() {
	for _, of := range it.files {
		of.f.Close()
	}
	it.files = nil
}

// copyOutput adds what show or warn printed to outputfile, when the
// program has one.
func (it *Interpreter) copyOutput(env *object.Environment, text string) {
	switch f := mustGet(env, outputFileName).(type) {
	case nil, *object.None:
	case *object.String:
		it.appendLine(f.Value, text, -1, 0)
	default:
		fatalf("outputfile must be a file's path, or none, got %s", object.Shown(f))
	}
}

func mustGet(env *object.Environment, name string) object.Object {
	v, _ := env.Get(name)
	return v
}

// logStop writes the error that stopped the program to the log file, if
// the main file imports log and has one.
func (it *Interpreter) logStop(fe fatalError) {
	if _, ok := it.Global.Get(logLevelName); !ok {
		return
	}
	if _, ok := mustGet(it.Global, logFileName).(*object.String); !ok {
		return
	}
	defer func() { recover() }() // a bad setting mustn't hide the real error
	file := fe.file
	if file == "" {
		file = it.fileName()
	}
	it.writeLog(it.Global, logEntry{level: "error", message: "stopped: " + fe.text, fields: object.NewMap(), file: file, line: fe.line}, false)
}
