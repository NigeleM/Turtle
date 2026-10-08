package evaluator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const appTOML = `# My app
name = "Shop"
debug = false
started = 2026-10-07
ratio = 0.5

[server]
port = 8080
hosts = ["localhost", "0.0.0.0"]

[[users]]
name = "Ann"

[[users]]
name = "Bo"
`

func TestConfig(t *testing.T) {
	dir := t.TempDir()
	write := func(name, text string) {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("app.toml", appTOML)
	write("app.json", `{"name": "Shop", "server": {"port": 8080}}`)
	write(".env", "# keys\nAPI_KEY=abc123\nexport NAME=\"Ann Lee\"\n")
	src := `import config
s = config_read["app.toml"]
show s at get["name"], "|", port of server of s, "|", typeof[s at get["started"]], "|", s at get["ratio"], "|", s at get["users"] at get[1] at get["name"] .
j = config_read["app.json"]
show port of server of j .
e = config_read[".env"]
show e .
srv = server of s
port of srv = 9090
config_write["out.toml", s]
config_write["out.json", s]
config_write["out.env", map ["KEY": "a b", "N": 5]]
show config_read["out.toml"] == s, "|", port of server of config_read["out.json"], "|", config_read["out.env"] .`
	got, err := runIn(t, dir, src, "")
	if err != nil {
		t.Fatal(err)
	}
	want := `Shop|8080|date|0.5|Bo
8080
{ "API_KEY": "abc123", "NAME": "Ann Lee" }
true|9090|{ "KEY": "a b", "N": "5" }`
	if strings.TrimSpace(got) != want {
		t.Errorf("got\n%s\nwant\n%s", got, want)
	}
	out, _ := os.ReadFile(filepath.Join(dir, "out.toml"))
	for _, w := range []string{"name = \"Shop\"\n", "started = 2026-10-07\n", "\n[server]\nport = 9090\n", "\n[[users]]\nname = \"Ann\"\n"} {
		if !strings.Contains(string(out), w) {
			t.Errorf("out.toml missing %q:\n%s", w, out)
		}
	}
}

func TestConfigErrors(t *testing.T) {
	dir := t.TempDir()
	for name, text := range map[string]string{
		"bad.toml":   "name = Shop\n",
		"twice.toml": "a = 1\na = 2\n",
		"bad.json":   `{"a": }`,
		"list.json":  `[1, 2]`,
		"bad.env":    "JUST WORDS\n",
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	cases := []struct{ src, kind, want string }{
		{`config_read["bad.toml"]`, "config", `config_read bad.toml line 1: "Shop" isn't a value: text needs quotes`},
		{`config_read["twice.toml"]`, "config", "config_read twice.toml line 2: a is already set"},
		{`config_read["bad.json"]`, "config", "config_read bad.json:"},
		{`config_read["list.json"]`, "config", "the file holds list, not an object"},
		{`config_read["bad.env"]`, "config", "config_read bad.env line 1 isn't KEY=value"},
		{`config_read["app.yaml"]`, "config", "the file's extension picks the format: .toml, .json or .env"},
		{`config_read["missing.toml"]`, "file", "config_read missing.toml"},
		{`config_write["x.toml", map ["a": none]]`, "config", "a is none, and TOML has no none"},
		{`config_write["x.env", map ["a": list [1]]]`, "config", "a is a list; a .env file holds only single values"},
		{`config_write["x.toml", 5]`, "type", "config_write: the settings must be a map"},
		{`config_read[]`, "type", "config_read takes a file"},
	}
	for _, c := range cases {
		src := "import config\nsafe\n    x = " + c.src + "\nhandle [] e .\n    show kind of e, \"|\", message of e .\nsafe [end]"
		got, err := runIn(t, dir, src, "")
		if err != nil {
			t.Errorf("%s: %v", c.src, err)
			continue
		}
		if !strings.HasPrefix(got, c.kind+"|") || !strings.Contains(got, c.want) {
			t.Errorf("%s: got %q, want kind %s and %q", c.src, got, c.kind, c.want)
		}
	}
}
