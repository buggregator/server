package app

import (
	"os"
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
		if len(types) != 9 {
			t.Errorf("len = %d, want 9", len(types))
		}
	})

	t.Run("some disabled", func(t *testing.T) {
		f := false
		cfg := ModulesConfig{Sentry: &f, SMTP: &f}
		types := cfg.EnabledTypes()
		if len(types) != 7 {
			t.Errorf("len = %d, want 7", len(types))
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
