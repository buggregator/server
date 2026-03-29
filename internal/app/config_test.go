package app

import (
	"testing"
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
