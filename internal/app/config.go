package app

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// AuthConfig controls authentication via OAuth2/OIDC providers.
type AuthConfig struct {
	Enabled      bool   `yaml:"enabled"`       // Enable authentication (default: false).
	Provider     string `yaml:"provider"`      // Provider type: "auth0", "google", "github", "keycloak", "gitlab", "oidc" (default: "oidc").
	ProviderURL  string `yaml:"provider_url"`  // OIDC issuer URL (e.g., https://xxx.us.auth0.com). Used for OIDC discovery.
	ClientID     string `yaml:"client_id"`     // OAuth2 client ID.
	ClientSecret string `yaml:"client_secret"` // OAuth2 client secret.
	CallbackURL  string `yaml:"callback_url"`  // OAuth2 callback URL (e.g., http://localhost:8000/auth/sso/callback).
	Scopes       string `yaml:"scopes"`        // Comma-separated scopes (default: "openid,email,profile").
	JWTSecret    string `yaml:"jwt_secret"`    // Secret for signing internal JWT tokens. Required when auth is enabled.
}

// Config holds application configuration.
type Config struct {
	Server   ServerConfig    `yaml:"server"`
	Database DatabaseConfig  `yaml:"database"`
	Storage  StorageConfig   `yaml:"storage"`
	TCP      TCPConfig       `yaml:"tcp"`
	Metrics  MetricsConfig   `yaml:"metrics"`
	MCP      MCPConfig       `yaml:"mcp"`
	Auth     AuthConfig      `yaml:"auth"`
	Modules  ModulesConfig   `yaml:"modules"`
	Webhooks []WebhookDef    `yaml:"webhooks"`
	Projects []ProjectDef    `yaml:"projects"`
	Version  string          `yaml:"-"`

	// Shortcuts.
	HTTPAddr      string `yaml:"-"`
	DatabaseDSN   string `yaml:"-"`
	SMTPAddr      string `yaml:"-"`
	MonologAddr   string `yaml:"-"`
	VarDumperAddr string `yaml:"-"`
}

// MCPConfig controls the MCP (Model Context Protocol) server.
type MCPConfig struct {
	Enabled    bool   `yaml:"enabled"`     // Enable MCP server (default: false).
	Transport  string `yaml:"transport"`   // "socket" (Unix socket, default) or "http" (HTTP SSE for remote access).
	SocketPath string `yaml:"socket_path"` // Unix socket path (for transport=socket). Default: /tmp/buggregator-mcp.sock
	Addr       string `yaml:"addr"`        // HTTP listen address (for transport=http). Default: :8001
	AuthToken  string `yaml:"auth_token"`  // Bearer token for HTTP transport auth. Empty = no auth.
}

// WebhookDef defines a webhook from config.
type WebhookDef struct {
	Event     string            `yaml:"event"`      // Event type to match ("sentry", "ray", "*" for all).
	URL       string            `yaml:"url"`        // Target URL.
	Headers   map[string]string `yaml:"headers"`    // Custom HTTP headers.
	VerifySSL *bool             `yaml:"verify_ssl"` // Verify SSL certificates (default: false).
	Retry     *bool             `yaml:"retry"`      // Retry on failure (default: true).
}

// MetricsConfig controls Prometheus metrics.
type MetricsConfig struct {
	Enabled bool   `yaml:"enabled"`
	Addr    string `yaml:"addr"` // Separate metrics server address (e.g., ":9090"). Empty = use main HTTP server.
}

type ServerConfig struct {
	Addr        string   `yaml:"addr"`
	CORSOrigins []string `yaml:"cors_origins"` // Allowed CORS origins. Empty = CORS disabled. Use ["*"] to allow all.
}

type StorageConfig struct {
	Mode string `yaml:"mode"` // "memory" (default) or "filesystem"
	Path string `yaml:"path"` // Directory for filesystem mode
}

type DatabaseConfig struct {
	Driver string `yaml:"driver"`
	DSN    string `yaml:"dsn"`
}

type TCPConfig struct {
	SMTP      TCPServerConfig `yaml:"smtp"`
	Monolog   TCPServerConfig `yaml:"monolog"`
	VarDumper TCPServerConfig `yaml:"var-dumper"`
}

type TCPServerConfig struct {
	Addr string `yaml:"addr"`
}

// ModulesConfig controls which modules are enabled.
// By default all modules are enabled. Set to false to disable.
type ModulesConfig struct {
	Sentry    *bool `yaml:"sentry"`
	Ray       *bool `yaml:"ray"`
	VarDump   *bool `yaml:"var-dump"`
	Inspector *bool `yaml:"inspector"`
	Monolog   *bool `yaml:"monolog"`
	SMTP      *bool `yaml:"smtp"`
	SMS       *bool `yaml:"sms"`
	HttpDump  *bool `yaml:"http-dump"`
	Profiler  *bool `yaml:"profiler"`
}

// IsEnabled returns true if a module is enabled (default: true).
func (m ModulesConfig) IsEnabled(moduleType string) bool {
	var val *bool
	switch moduleType {
	case "sentry":
		val = m.Sentry
	case "ray":
		val = m.Ray
	case "var-dump":
		val = m.VarDump
	case "inspector":
		val = m.Inspector
	case "monolog":
		val = m.Monolog
	case "smtp":
		val = m.SMTP
	case "sms":
		val = m.SMS
	case "http-dump":
		val = m.HttpDump
	case "profiler":
		val = m.Profiler
	}
	if val == nil {
		return true // enabled by default
	}
	return *val
}

// EnabledTypes returns a list of enabled module type strings.
func (m ModulesConfig) EnabledTypes() []string {
	all := []string{"sentry", "ray", "var-dump", "inspector", "monolog", "smtp", "sms", "http-dump", "profiler"}
	var enabled []string
	for _, t := range all {
		if m.IsEnabled(t) {
			enabled = append(enabled, t)
		}
	}
	return enabled
}

// ProjectDef defines a project from config.
type ProjectDef struct {
	Key  string `yaml:"key"`
	Name string `yaml:"name"`
}

// LoadConfig reads configuration with priority: flags > env > config file > defaults.
func LoadConfig() Config {
	var configFile string
	flag.StringVar(&configFile, "config", "", "Path to config file (buggregator.yaml)")

	cfg := Config{}

	flag.StringVar(&cfg.Server.Addr, "http-addr", "", "HTTP listen address")
	flag.StringVar(&cfg.Database.DSN, "db", "", "SQLite DSN")
	flag.StringVar(&cfg.TCP.SMTP.Addr, "smtp-addr", "", "SMTP listen address")
	flag.StringVar(&cfg.TCP.Monolog.Addr, "monolog-addr", "", "Monolog TCP listen address")
	flag.StringVar(&cfg.TCP.VarDumper.Addr, "vardumper-addr", "", "VarDumper TCP listen address")
	flag.Parse()

	fileCfg := loadConfigFile(configFile)

	// Merge: flag > env > file > default.
	cfg.Server.Addr = coalesce(cfg.Server.Addr, os.Getenv("HTTP_ADDR"), fileCfg.Server.Addr, ":8000")
	cfg.Database.DSN = coalesce(cfg.Database.DSN, os.Getenv("DATABASE_DSN"), fileCfg.Database.DSN, ":memory:")
	cfg.Database.Driver = coalesce(fileCfg.Database.Driver, "sqlite")
	cfg.TCP.SMTP.Addr = coalesce(cfg.TCP.SMTP.Addr, os.Getenv("SMTP_ADDR"), fileCfg.TCP.SMTP.Addr, ":1025")
	cfg.TCP.Monolog.Addr = coalesce(cfg.TCP.Monolog.Addr, os.Getenv("MONOLOG_ADDR"), fileCfg.TCP.Monolog.Addr, ":9913")
	cfg.TCP.VarDumper.Addr = coalesce(cfg.TCP.VarDumper.Addr, os.Getenv("VAR_DUMPER_ADDR"), fileCfg.TCP.VarDumper.Addr, ":9912")

	// Storage.
	cfg.Storage.Mode = coalesce(os.Getenv("STORAGE_MODE"), fileCfg.Storage.Mode, "memory")
	cfg.Storage.Path = coalesce(os.Getenv("STORAGE_PATH"), fileCfg.Storage.Path, "./storage")

	// Metrics.
	cfg.Metrics.Enabled = fileCfg.Metrics.Enabled || os.Getenv("METRICS_ENABLED") == "true"
	cfg.Metrics.Addr = coalesce(os.Getenv("METRICS_ADDR"), fileCfg.Metrics.Addr)

	// MCP.
	cfg.MCP.Enabled = fileCfg.MCP.Enabled || os.Getenv("MCP_ENABLED") == "true"
	cfg.MCP.Transport = coalesce(os.Getenv("MCP_TRANSPORT"), fileCfg.MCP.Transport, "socket")
	cfg.MCP.SocketPath = coalesce(os.Getenv("MCP_SOCKET_PATH"), fileCfg.MCP.SocketPath, "/tmp/buggregator-mcp.sock")
	cfg.MCP.Addr = coalesce(os.Getenv("MCP_ADDR"), fileCfg.MCP.Addr, ":8001")
	cfg.MCP.AuthToken = coalesce(os.Getenv("MCP_AUTH_TOKEN"), fileCfg.MCP.AuthToken)

	// Auth.
	cfg.Auth.Enabled = fileCfg.Auth.Enabled || os.Getenv("AUTH_ENABLED") == "true"
	cfg.Auth.Provider = coalesce(os.Getenv("AUTH_PROVIDER"), fileCfg.Auth.Provider, "oidc")
	cfg.Auth.ProviderURL = coalesce(os.Getenv("AUTH_PROVIDER_URL"), fileCfg.Auth.ProviderURL)
	cfg.Auth.ClientID = coalesce(os.Getenv("AUTH_CLIENT_ID"), fileCfg.Auth.ClientID)
	cfg.Auth.ClientSecret = coalesce(os.Getenv("AUTH_CLIENT_SECRET"), fileCfg.Auth.ClientSecret)
	cfg.Auth.CallbackURL = coalesce(os.Getenv("AUTH_CALLBACK_URL"), fileCfg.Auth.CallbackURL)
	cfg.Auth.Scopes = coalesce(os.Getenv("AUTH_SCOPES"), fileCfg.Auth.Scopes, "openid,email,profile")
	cfg.Auth.JWTSecret = coalesce(os.Getenv("AUTH_JWT_SECRET"), fileCfg.Auth.JWTSecret)

	// CORS origins.
	cfg.Server.CORSOrigins = fileCfg.Server.CORSOrigins
	if env := os.Getenv("CORS_ORIGINS"); env != "" {
		cfg.Server.CORSOrigins = strings.Split(env, ",")
	}
	if len(cfg.Server.CORSOrigins) == 0 {
		cfg.Server.CORSOrigins = []string{"*"}
	}

	// Modules: merge from file config, env override.
	cfg.Modules = fileCfg.Modules
	if env := os.Getenv("CLIENT_SUPPORTED_EVENTS"); env != "" {
		cfg.Modules = modulesFromCSV(env)
	}

	// Webhooks from config file.
	cfg.Webhooks = fileCfg.Webhooks

	// Projects from config file.
	cfg.Projects = fileCfg.Projects

	// Set shortcuts.
	cfg.HTTPAddr = cfg.Server.Addr
	cfg.DatabaseDSN = cfg.Database.DSN
	cfg.SMTPAddr = cfg.TCP.SMTP.Addr
	cfg.MonologAddr = cfg.TCP.Monolog.Addr
	cfg.VarDumperAddr = cfg.TCP.VarDumper.Addr

	return cfg
}

// modulesFromCSV creates ModulesConfig from a comma-separated list of enabled types.
// Types not in the list are disabled.
func modulesFromCSV(csv string) ModulesConfig {
	enabled := make(map[string]bool)
	for _, t := range strings.Split(csv, ",") {
		enabled[strings.TrimSpace(t)] = true
	}

	t, f := true, false
	return ModulesConfig{
		Sentry:    boolPtr(enabled["sentry"], t, f),
		Ray:       boolPtr(enabled["ray"], t, f),
		VarDump:   boolPtr(enabled["var-dump"], t, f),
		Inspector: boolPtr(enabled["inspector"], t, f),
		Monolog:   boolPtr(enabled["monolog"], t, f),
		SMTP:      boolPtr(enabled["smtp"], t, f),
		SMS:       boolPtr(enabled["sms"], t, f),
		HttpDump:  boolPtr(enabled["http-dump"], t, f),
		Profiler:  boolPtr(enabled["profiler"], t, f),
	}
}

func boolPtr(cond bool, yes, no bool) *bool {
	if cond {
		return &yes
	}
	return &no
}

func loadConfigFile(path string) Config {
	var cfg Config
	if path == "" {
		for _, name := range []string{"buggregator.yaml", "buggregator.yml"} {
			if _, err := os.Stat(name); err == nil {
				path = name
				break
			}
		}
	}
	if path == "" {
		return cfg
	}

	data, err := os.ReadFile(path)
	if err != nil {
		slog.Warn("failed to read config file", "path", path, "err", err)
		return cfg
	}

	expanded := expandEnvVars(string(data))
	if err := yaml.Unmarshal([]byte(expanded), &cfg); err != nil {
		slog.Warn("failed to parse config file", "path", path, "err", err)
		return cfg
	}

	slog.Info("loaded config file", "path", path)
	return cfg
}

var envVarPattern = regexp.MustCompile(`\$\{([^}:]+)(?::([^}]*))?\}`)

func expandEnvVars(input string) string {
	return envVarPattern.ReplaceAllStringFunc(input, func(match string) string {
		groups := envVarPattern.FindStringSubmatch(match)
		if len(groups) < 2 {
			return match
		}
		varName := strings.TrimSpace(groups[1])
		defaultVal := ""
		if len(groups) >= 3 {
			defaultVal = groups[2]
		}
		if val := os.Getenv(varName); val != "" {
			return val
		}
		return defaultVal
	})
}

func coalesce(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

func (c Config) String() string {
	return fmt.Sprintf("http=%s db=%s smtp=%s monolog=%s vardumper=%s modules=%v",
		c.Server.Addr, c.Database.DSN, c.TCP.SMTP.Addr, c.TCP.Monolog.Addr, c.TCP.VarDumper.Addr,
		c.Modules.EnabledTypes())
}
