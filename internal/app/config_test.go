package app

import (
	"os"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestModulesConfig_IsEnabled(t *testing.T) {
	t.Run("nil pointer means enabled by default", func(t *testing.T) {
		cfg := ModulesConfig{}
		for _, typ := range []string{"sentry", "ray", "var-dump", "inspector", "monolog", "smtp", "sms", "http-dump", "profiler"} {
			if !cfg.IsEnabled(typ) {
				t.Errorf("%q should be enabled by default", typ)
			}
		}
	})

	t.Run("unknown type returns true", func(t *testing.T) {
		cfg := ModulesConfig{}
		if !cfg.IsEnabled("unknown-module") {
			t.Error("unknown module should return true")
		}
	})

	t.Run("explicitly disabled", func(t *testing.T) {
		f := false
		cfg := ModulesConfig{Sentry: &f, SMTP: &f}
		if cfg.IsEnabled("sentry") {
			t.Error("sentry should be disabled")
		}
		if cfg.IsEnabled("smtp") {
			t.Error("smtp should be disabled")
		}
		if !cfg.IsEnabled("ray") {
			t.Error("ray should still be enabled")
		}
	})

	t.Run("explicitly enabled", func(t *testing.T) {
		tr := true
		cfg := ModulesConfig{Sentry: &tr}
		if !cfg.IsEnabled("sentry") {
			t.Error("sentry should be enabled")
		}
	})
}

func TestModulesConfig_EnabledTypes(t *testing.T) {
	t.Run("all enabled by default", func(t *testing.T) {
		cfg := ModulesConfig{}
		types := cfg.EnabledTypes()
		if len(types) != 12 {
			t.Errorf("len = %d, want 12", len(types))
		}
	})

	t.Run("some disabled", func(t *testing.T) {
		f := false
		cfg := ModulesConfig{Sentry: &f, SMTP: &f}
		types := cfg.EnabledTypes()
		if len(types) != 10 {
			t.Errorf("len = %d, want 10", len(types))
		}
		for _, typ := range types {
			if typ == "sentry" || typ == "smtp" {
				t.Errorf("disabled type %q should not be in list", typ)
			}
		}
	})
}

func TestModulesFromCSV(t *testing.T) {
	cfg := modulesFromCSV("sentry,ray,profiler")

	if !cfg.IsEnabled("sentry") {
		t.Error("sentry should be enabled")
	}
	if !cfg.IsEnabled("ray") {
		t.Error("ray should be enabled")
	}
	if !cfg.IsEnabled("profiler") {
		t.Error("profiler should be enabled")
	}

	// Types not in CSV should be disabled
	if cfg.IsEnabled("smtp") {
		t.Error("smtp should be disabled")
	}
	if cfg.IsEnabled("sms") {
		t.Error("sms should be disabled")
	}
	if cfg.IsEnabled("monolog") {
		t.Error("monolog should be disabled")
	}
}

func TestExpandEnvVars(t *testing.T) {
	t.Setenv("TEST_VAR", "hello")

	tests := []struct {
		input string
		want  string
	}{
		{"${TEST_VAR}", "hello"},
		{"${TEST_VAR:fallback}", "hello"},
		{"${UNSET_VAR:default_val}", "default_val"},
		{"${UNSET_VAR}", ""},
		{"addr: ${TEST_VAR}:8000", "addr: hello:8000"},
		{"no vars here", "no vars here"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := expandEnvVars(tt.input)
			if got != tt.want {
				t.Errorf("expandEnvVars(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestCoalesce(t *testing.T) {
	tests := []struct {
		values []string
		want   string
	}{
		{[]string{"a", "b"}, "a"},
		{[]string{"", "b"}, "b"},
		{[]string{"", "", "c"}, "c"},
		{[]string{"", ""}, ""},
		{[]string{}, ""},
	}

	for _, tt := range tests {
		got := coalesce(tt.values...)
		if got != tt.want {
			t.Errorf("coalesce(%v) = %q, want %q", tt.values, got, tt.want)
		}
	}
}

func TestConfig_String(t *testing.T) {
	cfg := Config{
		Server:   ServerConfig{Addr: ":8000"},
		Database: DatabaseConfig{DSN: ":memory:"},
		TCP: TCPConfig{
			SMTP:      TCPServerConfig{Addr: ":1025"},
			Monolog:   TCPServerConfig{Addr: ":9913"},
			VarDumper: TCPServerConfig{Addr: ":9912"},
		},
	}
	s := cfg.String()
	if s == "" {
		t.Error("String() should not be empty")
	}
}

func TestConfig_MCPMockWrite(t *testing.T) {
	if !(MCPConfig{}).MockWriteOn() {
		t.Error("mock_write is off by default")
	}
	path := t.TempDir() + "/buggregator.yaml"
	os.WriteFile(path, []byte("mcp:\n  mock_write: false\n"), 0o600)
	if loadConfigFile(path).MCP.MockWriteOn() {
		t.Error("mock_write: false is on")
	}
}

func TestConfig_MocksOpenAPIFromYAML(t *testing.T) {
	path := t.TempDir() + "/buggregator.yaml"
	os.WriteFile(path, []byte("mocks_openapi:\n  - file: ./api.yaml\n    project: shop\n    group: shop-api\n    base_path: \"\"\n  - file: ./b.json\n"), 0o600)
	defs := loadConfigFile(path).MocksOpenAPI
	if len(defs) != 2 || defs[0].File != "./api.yaml" || defs[0].Project != "shop" || defs[0].Group != "shop-api" ||
		defs[0].BasePath == nil || *defs[0].BasePath != "" || defs[1].BasePath != nil {
		t.Errorf("defs = %+v", defs)
	}
}

func TestConfig_MocksFromYAML(t *testing.T) {
	path := t.TempDir() + "/buggregator.yaml"
	os.WriteFile(path, []byte(`
mocks:
  - method: GET
    path: /users/{id}
    status: 200
    headers: { Content-Type: application/json }
    body: '{"id": 1}'
    delay: 150ms
    project: shop
  - path: /orders/{id}
    template: true
    seed: 42
    body: '[{{ range $i := repeat 2 }}{{ if $i }},{{ end }}{{ fake "name" | json }}{{ end }}]'
`), 0o600)
	cfg := loadConfigFile(path)
	if len(cfg.Mocks) != 2 {
		t.Fatalf("mocks = %+v", cfg.Mocks)
	}
	m := cfg.Mocks[0]
	if m.Method != "GET" || m.Path != "/users/{id}" || m.Status != 200 || m.Headers["Content-Type"] != "application/json" ||
		m.Body != `{"id": 1}` || m.Delay != "150ms" || m.Project != "shop" {
		t.Errorf("mock = %+v", m)
	}
	if m := cfg.Mocks[1]; !m.Template || m.Seed != 42 || !strings.Contains(m.Body, `{{ range $i := repeat 2 }}`) {
		t.Errorf("template mock = %+v", m)
	}
}

// 0 is a real limit value (unlimited) from any source; only empty or
// unreadable values fall through to the next source.
func TestResolveLimit(t *testing.T) {
	zero, ten := 0, 10
	tests := []struct {
		name   string
		file   *int
		values []string
		want   int
	}{
		{"nothing set", nil, []string{"", ""}, 1000},
		{"flag 0", &ten, []string{"0", "5"}, 0},
		{"env 0", &ten, []string{"", "0"}, 0},
		{"file 0", &zero, []string{"", ""}, 0},
		{"env over file", &ten, []string{"", "7"}, 7},
		{"flag over env", nil, []string{"3", "7"}, 3},
		{"unreadable env", &ten, []string{"", "lots"}, 10},
		{"negative env", nil, []string{"", "-1"}, 1000},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := *resolveLimit(1000, tt.file, tt.values...); got != tt.want {
				t.Errorf("resolveLimit = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestEnvBool(t *testing.T) {
	tests := []struct {
		env  string
		def  bool
		want bool
	}{
		{"", true, true},
		{"", false, false},
		{"false", true, false}, // env switches off a yaml true
		{"true", false, true},
		{"0", true, false},
		{"junk", true, true},
	}
	for _, tt := range tests {
		t.Setenv("BUGGREGATOR_TEST_BOOL", tt.env)
		if got := envBool("BUGGREGATOR_TEST_BOOL", tt.def); got != tt.want {
			t.Errorf("env=%q def=%v: got %v", tt.env, tt.def, got)
		}
	}
}

func TestParseIdleTimeout(t *testing.T) {
	for in, want := range map[string]time.Duration{
		"10m": 10 * time.Minute, " 90s ": 90 * time.Second, "0": -1, "0s": -1, "-5m": -1,
		"600": 0, "soon": 0, // not a duration: default
	} {
		if got := parseIdleTimeout(in); got != want {
			t.Errorf("%q: %v, want %v", in, got, want)
		}
	}
}

func TestParseSize(t *testing.T) {
	for in, want := range map[string]int64{
		"123": 123, "64KB": 64 << 10, "100MB": 100 << 20, "100mb": 100 << 20, " 1 GB ": 1 << 30,
		"256MiB": 256 << 20, "10M": 10 << 20, "512B": 512,
	} {
		if got, err := parseSize(in); err != nil || got != want {
			t.Errorf("parseSize(%q) = %d, %v; want %d", in, got, err, want)
		}
	}
	for _, in := range []string{"", "0", "-1MB", "MB", "1.5GB", "10TB", "99999999999GB"} {
		if _, err := parseSize(in); err == nil {
			t.Errorf("parseSize(%q) accepted", in)
		}
	}
	if got := sizeOr("x", "junk", 7); got != 7 {
		t.Errorf("sizeOr fallback = %d", got)
	}
}

func TestModulesConfig_XdebugTrace(t *testing.T) {
	if !(ModulesConfig{}).IsEnabled("xdebug-trace") {
		t.Error("xdebug-trace is off by default")
	}
	if modulesFromCSV("sentry").IsEnabled("xdebug-trace") {
		t.Error("xdebug-trace on although not listed")
	}
	if !modulesFromCSV("sentry,xdebug-trace").IsEnabled("xdebug-trace") {
		t.Error("xdebug-trace off although listed")
	}
}

func TestBroadcastConfig(t *testing.T) {
	t.Setenv("BROADCAST_UPSTREAM", "http://127.0.0.1:8080")
	t.Setenv("BROADCAST_APPS", "app-key:app-secret, bad ,k2:s:with:colons")
	c := broadcastConfig(BroadcastConfig{Upstream: "http://file", Apps: []BroadcastAppConfig{{Key: "file", Secret: "x"}}})
	if c.Upstream != "http://127.0.0.1:8080" {
		t.Errorf("upstream = %q", c.Upstream)
	}
	sec := c.Secrets()
	if len(sec) != 2 || sec["app-key"] != "app-secret" || sec["k2"] != "s:with:colons" {
		t.Errorf("secrets = %v", sec)
	}
}

func TestModulesConfig_Disable(t *testing.T) {
	cfg := Config{}
	cfg.Modules.Disable("xdebug-trace")
	cfg.Modules.Disable("unknown-module") // no-op
	if cfg.Modules.IsEnabled("xdebug-trace") {
		t.Error("xdebug-trace should be disabled")
	}
	types := cfg.Modules.EnabledTypes()
	if strings.Contains(strings.Join(types, ","), "xdebug-trace") || len(types) != 11 {
		t.Errorf("EnabledTypes = %v", types)
	}
}

func TestCORSOrigins(t *testing.T) {
	for _, tt := range []struct {
		name string
		file []string
		env  string
		want string
	}{
		{"default allows all", nil, "", "*"},
		{"file", []string{"http://localhost:3000"}, "", "http://localhost:3000"},
		{"env wins", []string{"http://a"}, "http://b,http://c", "http://b,http://c"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := strings.Join(corsOrigins(tt.file, tt.env), ","); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParseRetentionAge(t *testing.T) {
	for in, want := range map[string]time.Duration{
		"": 0, "0": 0, "24h": 24 * time.Hour, "30m": 30 * time.Minute, "7d": 7 * 24 * time.Hour,
		"-1h": 0, "week": 0, "1.5d": 0,
	} {
		if got := parseRetentionAge(in); got != want {
			t.Errorf("parseRetentionAge(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestSourceFetchMode(t *testing.T) {
	for in, want := range map[string]string{
		"private": "private", "ANY": "any", " off ": "off", "": "private", "all": "private",
	} {
		if got := sourceFetchMode(in); got != want {
			t.Errorf("sourceFetchMode(%q) = %q, want %q", in, got, want)
		}
	}
}

// /api/settings gives the setup help the real listen ports: only listeners
// that are on, only addresses with a fixed port.
func TestConfig_ListenPorts(t *testing.T) {
	off := false
	base := func() Config {
		var c Config
		c.Server.Addr = ":8000"
		c.TCP.SMTP.Addr = "0.0.0.0:2525"
		c.TCP.Monolog.Addr = ":9913"
		c.TCP.VarDumper.Addr = "127.0.0.1:9912"
		c.TCP.Proxy.Addr = ":8080"
		c.Xdebug.Addr = "127.0.0.1:9003"
		c.MCP.Addr = ":8001"
		c.MCP.Transport = "socket"
		return c
	}
	tests := []struct {
		name string
		mod  func(*Config)
		want map[string]int
	}{
		{"defaults: xdebug and mcp off", func(*Config) {},
			map[string]int{"http": 8000, "smtp": 2525, "monolog": 9913, "var_dumper": 9912, "proxy": 8080}},
		{"xdebug on, mcp over http", func(c *Config) { c.Xdebug.Enabled = true; c.MCP.Enabled = true; c.MCP.Transport = "http" },
			map[string]int{"http": 8000, "smtp": 2525, "monolog": 9913, "var_dumper": 9912, "proxy": 8080, "xdebug": 9003, "mcp": 8001}},
		{"mcp over socket has no port", func(c *Config) { c.MCP.Enabled = true },
			map[string]int{"http": 8000, "smtp": 2525, "monolog": 9913, "var_dumper": 9912, "proxy": 8080}},
		{"modules off", func(c *Config) { c.Modules.SMTP = &off; c.Modules.Monolog = &off; c.Modules.VarDump = &off },
			map[string]int{"http": 8000, "proxy": 8080}},
		{"no fixed port", func(c *Config) { c.Server.Addr = ":0"; c.TCP.SMTP.Addr = "bad"; c.TCP.Proxy.Addr = ":http" },
			map[string]int{"monolog": 9913, "var_dumper": 9912}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := base()
			tt.mod(&c)
			got := c.ListenPorts()
			if len(got) != len(tt.want) {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
			for k, v := range tt.want {
				if got[k] != v {
					t.Errorf("%s = %d, want %d (all: %v)", k, got[k], v, got)
				}
			}
		})
	}
}

func TestSMTPRelayConfig(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/buggregator.yaml"
	os.WriteFile(path, []byte("smtp_relay:\n  host: smtp.file.test\n  port: 2525\n  tls: none\n  username: file-user\n  password: from-yaml\n  allowed_recipients: [\"*@company.test\"]\n"), 0o600)
	secret := dir + "/relay-password"
	os.WriteFile(secret, []byte("from-file\n"), 0o600)
	file := loadConfigFile(path).SMTPRelay

	for _, tc := range []struct {
		name string
		env  map[string]string
		want SMTPRelayConfig
	}{
		{"file only", nil, SMTPRelayConfig{Host: "smtp.file.test", Port: 2525, TLS: "none", Username: "file-user",
			Password: "from-yaml", AllowedRecipients: []string{"*@company.test"}}},
		{"password file wins over yaml", map[string]string{"SMTP_RELAY_PASSWORD_FILE": secret},
			SMTPRelayConfig{Host: "smtp.file.test", Port: 2525, TLS: "none", Username: "file-user",
				Password: "from-file", PasswordFile: secret, AllowedRecipients: []string{"*@company.test"}}},
		{"env over everything", map[string]string{
			"SMTP_RELAY_HOST": "smtp.env.test", "SMTP_RELAY_PORT": "465", "SMTP_RELAY_TLS": "TLS",
			"SMTP_RELAY_USERNAME": "env-user", "SMTP_RELAY_PASSWORD": "from-env", "SMTP_RELAY_PASSWORD_FILE": secret,
			"SMTP_RELAY_FROM": "qa@company.test", "SMTP_RELAY_ALLOWED_RECIPIENTS": "*@a.test, b@b.test",
			"SMTP_RELAY_INSECURE_SKIP_VERIFY": "true",
		}, SMTPRelayConfig{Host: "smtp.env.test", Port: 465, TLS: "tls", Username: "env-user", Password: "from-env",
			PasswordFile: secret, From: "qa@company.test", AllowedRecipients: []string{"*@a.test", "b@b.test"}, InsecureSkipVerify: true}},
		{"bad port and tls fall back", map[string]string{"SMTP_RELAY_PORT": "99999", "SMTP_RELAY_TLS": "ssl"},
			SMTPRelayConfig{Host: "smtp.file.test", Port: 2525, TLS: "starttls", Username: "file-user",
				Password: "from-yaml", AllowedRecipients: []string{"*@company.test"}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for k, v := range tc.env {
				t.Setenv(k, v)
			}
			got := smtpRelayConfig(file)
			if strings.Join(got.AllowedRecipients, ",") != strings.Join(tc.want.AllowedRecipients, ",") {
				t.Errorf("allowed = %v, want %v", got.AllowedRecipients, tc.want.AllowedRecipients)
			}
			got.AllowedRecipients, tc.want.AllowedRecipients = nil, nil
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("got  %+v\nwant %+v", got, tc.want)
			}
		})
	}

	// Off by default; the password never shows in the config summary.
	if c := smtpRelayConfig(SMTPRelayConfig{}); c.Host != "" || c.TLS != "starttls" {
		t.Errorf("default = %+v", c)
	}
	if s := (Config{SMTPRelay: SMTPRelayConfig{Host: "h", Password: "from-env"}}).String(); strings.Contains(s, "from-env") {
		t.Errorf("String() shows the password: %s", s)
	}
}

func TestEnvList(t *testing.T) {
	file := []string{"file.com"}
	if got := envList("BUGGREGATOR_TEST_LIST", file); len(got) != 1 || got[0] != "file.com" {
		t.Fatalf("no env: %v", got)
	}
	t.Setenv("BUGGREGATOR_TEST_LIST", " a@x.com, ,acme.com ")
	if got := envList("BUGGREGATOR_TEST_LIST", file); len(got) != 2 || got[0] != "a@x.com" || got[1] != "acme.com" {
		t.Fatalf("env: %v", got)
	}
}
