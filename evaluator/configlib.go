package evaluator

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	"Turtle/object"
	"Turtle/toml"
)

// The "config" builtin module: settings files, the format chosen by the
// file's extension (.toml, .json, .env), as a map.
//
//	settings = config_read["app.toml"]
//	port = settings at get["server"] at get["port"]
//	config_write["app.toml", settings]
//
// TOML's [sections] become maps inside the map; numbers, booleans, dates
// and lists keep their kinds. A file that isn't well written is an error
// of kind config naming its line; a missing one is a file error.

func (it *Interpreter) callConfig(name string, args []object.Object) object.Object {
	switch name {
	case "config_read":
		argCount(name, args, 1, 1, "a file: .toml, .json or .env")
		path := textArg(name, args, 0, "the file")
		format := configFormat(name, path)
		data, err := os.ReadFile(it.resolvePath(path))
		if err != nil {
			fatalKind(kindFile, "config_read %s: %s", path, fileProblem(err))
		}
		return readConfig(path, format, string(data))
	case "config_write":
		argCount(name, args, 2, 2, "a file (.toml, .json or .env) and a map of settings")
		path := textArg(name, args, 0, "the file")
		format := configFormat(name, path)
		m, ok := args[1].(*object.Map)
		if !ok {
			fatalf("config_write: the settings must be a map, got %s", typeName(args[1]))
		}
		text := writeConfig(path, format, m)
		if err := os.WriteFile(it.resolvePath(path), []byte(text), 0o644); err != nil {
			fatalKind(kindFile, "config_write %s: %s", path, fileProblem(err))
		}
		return object.NoneValue
	}
	fatalKind(kindName, "no config function %q", name)
	return nil
}

// configFormat is "toml", "json" or "env", from the file's extension.
func configFormat(fn, path string) string {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".toml":
		return "toml"
	case ".json":
		return "json"
	case ".env":
		return "env"
	}
	fatalKind(kindConfig, "%s %s: the file's extension picks the format: .toml, .json or .env", fn, path)
	return ""
}

func readConfig(path, format, text string) object.Object {
	switch format {
	case "toml":
		t, err := toml.Parse(text)
		if err != nil {
			fatalKind(kindConfig, "config_read %s %s", path, err)
		}
		return fromTOML(t)
	case "json":
		v := configJSON(path, text)
		if _, ok := v.(*object.Map); !ok {
			fatalKind(kindConfig, "config_read %s: the file holds %s, not an object { ... } of settings", path, typeName(v))
		}
		return v
	}
	m := object.NewMap()
	for n, line := range strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n") {
		line = strings.TrimSpace(line)
		if n == 0 {
			line = strings.TrimPrefix(line, "\ufeff")
		}
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimPrefix(line, "export ")
		key, value, ok := strings.Cut(line, "=")
		key = strings.TrimSpace(key)
		if !ok || key == "" || strings.ContainsAny(key, " \t") {
			fatalKind(kindConfig, "config_read %s line %d isn't KEY=value: %s", path, n+1, line)
		}
		value, ok = envValue(strings.TrimSpace(value))
		if !ok {
			fatalKind(kindConfig, "config_read %s line %d: a quote isn't closed", path, n+1)
		}
		m.Put(&object.String{Value: key}, &object.String{Value: value})
	}
	return m
}

// configJSON reads JSON, a bad file being a config error.
func configJSON(path, text string) (v object.Object) {
	defer func() {
		if r := recover(); r != nil {
			fe, ok := r.(fatalError)
			if !ok || fe.kind != kindJSON {
				panic(r)
			}
			fatalKind(kindConfig, "config_read %s: %s", path, strings.TrimPrefix(fe.text, "config_read: "))
		}
	}()
	return parseJSON("config_read", text)
}

func fromTOML(v any) object.Object {
	switch x := v.(type) {
	case *toml.Table:
		m := object.NewMap()
		for _, k := range x.Keys {
			m.Put(&object.String{Value: k}, fromTOML(x.Values[k]))
		}
		return m
	case []any:
		l := &object.List{Elements: make([]object.Object, len(x))}
		for i, e := range x {
			l.Elements[i] = fromTOML(e)
		}
		return l
	case string:
		return &object.String{Value: x}
	case int64:
		return &object.Integer{Value: x}
	case float64:
		return &object.Float{Value: x}
	case bool:
		return &object.Boolean{Value: x}
	case toml.DateTime:
		if x.Kind == toml.LocalTime {
			// Turtle has no time of day without a date: text, "07:32:00".
			return &object.String{Value: x.Time.Format("15:04:05.999999999")}
		}
		return &object.Date{Time: x.Time.Truncate(time.Second)}
	}
	return object.NoneValue
}

func writeConfig(path, format string, m *object.Map) string {
	switch format {
	case "toml":
		t, err := toTOML(m, nil)
		if err == nil {
			var text string
			text, err = toml.Write(t.(*toml.Table))
			if err == nil {
				return text
			}
		}
		fatalKind(kindConfig, "config_write %s: %v", path, err)
	case "json":
		return string(toJSON("config_write", m, "  ")) + "\n"
	}
	var b strings.Builder
	for _, k := range m.Keys {
		key := m.KeyOf(k)
		ks, ok := key.(*object.String)
		if !ok || ks.Value == "" || strings.ContainsAny(ks.Value, " \t=\n#") {
			fatalKind(kindConfig, "config_write %s: %s can't be a .env name (one word, no spaces or =)", path, object.Shown(key))
		}
		var text string
		switch v := m.Values[k].(type) {
		case *object.Map, *object.List, *object.Set, *object.Assembly:
			fatalKind(kindConfig, "config_write %s: %s is %s; a .env file holds only single values (use .toml or .json for more)", path, ks.Value, aValue(v))
		case *object.None:
			fatalKind(kindConfig, "config_write %s: %s is none; leave it out", path, ks.Value)
		case *object.String:
			text = v.Value
		default:
			text = object.Exact(v)
		}
		if text == "" || strings.ContainsAny(text, " \t\n\r\"'#\\") {
			text = `"` + strings.NewReplacer(`\`, `\\`, `"`, `\"`, "\n", `\n`, "\r", `\r`, "\t", `\t`).Replace(text) + `"`
		}
		b.WriteString(ks.Value + "=" + text + "\n")
	}
	return b.String()
}

// toTOML turns a value into the toml package's kinds.
func toTOML(v object.Object, path []string) (any, error) {
	where := strings.Join(path, ".")
	switch x := v.(type) {
	case *object.Map:
		t := toml.NewTable()
		for _, k := range x.Keys {
			key := x.KeyOf(k)
			name := key.Inspect()
			if s, ok := key.(*object.String); ok {
				name = s.Value
			}
			val, err := toTOML(x.Values[k], append(path, name))
			if err != nil {
				return nil, err
			}
			t.Set(name, val)
		}
		return t, nil
	case *object.Assembly:
		t := toml.NewTable()
		for i, f := range x.Shape.Fields {
			val, err := toTOML(x.Values[i], append(path, f))
			if err != nil {
				return nil, err
			}
			t.Set(f, val)
		}
		return t, nil
	case *object.List, *object.Set:
		var elems []object.Object
		if l, ok := x.(*object.List); ok {
			elems = l.Elements
		} else {
			elems = x.(*object.Set).Elements
		}
		out := make([]any, len(elems))
		for i, e := range elems {
			val, err := toTOML(e, path)
			if err != nil {
				return nil, err
			}
			out[i] = val
		}
		return out, nil
	case *object.String:
		return x.Value, nil
	case *object.Integer:
		return x.Value, nil
	case *object.Float:
		return x.Value, nil
	case *object.Boolean:
		return x.Value, nil
	case *object.Date:
		switch {
		case !x.Local():
			return toml.DateTime{Time: x.Time, Kind: toml.OffsetDateTime}, nil
		case x.Time.Hour() == 0 && x.Time.Minute() == 0 && x.Time.Second() == 0:
			return toml.DateTime{Time: x.Time, Kind: toml.LocalDate}, nil
		}
		return toml.DateTime{Time: x.Time, Kind: toml.LocalDateTime}, nil
	case *object.None:
		return nil, errors.New(where + " is none, and TOML has no none: leave the key out")
	}
	return nil, errors.New(where + " is " + aValue(v) + ", which a settings file can't hold")
}
