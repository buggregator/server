package app

import (
	"flag"
	"fmt"
	"log/slog"
	"net"
	"os"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

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

// UIConfig bounds the event lists served to the UI.
//
// /api/events and /api/events/preview used to return everything stored for a
// project. By default they still do: the current UI loads the list once and
// does not page, so a cap would hide older events. default_limit and
// default_window opt in to smaller responses.
//
// The limits are pointers so an explicit 0 (no limit) differs from "not set".
type UIConfig struct {
	DefaultLimit  *int   `yaml:"default_limit"`  // events returned without a limit param; unset or 0 = unlimited
	MaxLimit      *int   `yaml:"max_limit"`      // ceiling for an explicit limit (default 5000); 0 = no ceiling
	DefaultWindow string `yaml:"default_window"` // time window without from/to: "24h", "7d"; empty or "all" = no window
}

// Config holds application configuration.
type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Database DatabaseConfig `yaml:"database"`
	Storage  StorageConfig  `yaml:"storage"`
	TCP      TCPConfig      `yaml:"tcp"`
	Metrics  MetricsConfig  `yaml:"metrics"`
	MCP      MCPConfig      `yaml:"mcp"`
	Xdebug   XdebugConfig   `yaml:"xdebug"`
	Auth     AuthConfig     `yaml:"auth"`
	UI       UIConfig       `yaml:"ui"`
	Sentry   SentryConfig   `yaml:"sentry"`
	Modules  ModulesConfig  `yaml:"modules"`
	Webhooks []WebhookDef   `yaml:"webhooks"`
	Projects []ProjectDef   `yaml:"projects"`
	Mocks    []MockDef      `yaml:"mocks"`
	// MocksOpenAPI are OpenAPI specs turned into read-only mock rules at start.
	MocksOpenAPI []MockOpenAPIDef `yaml:"mocks_openapi"`
	// UpdateCheck asks GitHub for the latest release so the UI can say a new
	// version is out.
	UpdateCheck UpdateCheckConfig `yaml:"update_check"`
	// Update controls installing a new release from the UI.
	Update UpdateConfig `yaml:"update"`
	// XdebugTrace bounds uploaded Xdebug function traces (module xdebug-trace).
	XdebugTrace XdebugTraceConfig `yaml:"xdebug_trace"`
	// Retention deletes old events in the background; unset keeps everything.
	Retention RetentionConfig `yaml:"retention"`
	// SMTPRelay is the outgoing SMTP server a caught email can be released
	// to from the UI; off while host is empty.
	SMTPRelay SMTPRelayConfig `yaml:"smtp_relay"`
	// Broadcast is the Pusher-compatible events catcher (Laravel Reverb,
	// Pusher, soketi): an optional upstream to forward to, app secrets.
	Broadcast BroadcastConfig `yaml:"broadcast"`
	Version   string          `yaml:"-"`

	// Shortcuts.
	HTTPAddr      string `yaml:"-"`
	DatabaseDSN   string `yaml:"-"`
	SMTPAddr      string `yaml:"-"`
	MonologAddr   string `yaml:"-"`
	VarDumperAddr string `yaml:"-"`
	ProxyAddr     string `yaml:"-"`
}

// MCPConfig controls the MCP (Model Context Protocol) server.
type MCPConfig struct {
	Enabled    bool   `yaml:"enabled"`     // Enable MCP server (default: false).
	Transport  string `yaml:"transport"`   // "socket" (Unix socket, default) or "http" (HTTP SSE for remote access).
	SocketPath string `yaml:"socket_path"` // Unix socket path (for transport=socket). Default: /tmp/buggregator-mcp.sock
	Addr       string `yaml:"addr"`        // HTTP listen address (for transport=http). Default: :8001
	AuthToken  string `yaml:"auth_token"`  // Bearer token for HTTP transport auth. Empty = no auth.
	// MockWrite exposes the MCP tools that create, change and delete mock
	// rules (default: true; false leaves the read and preview tools).
	MockWrite *bool `yaml:"mock_write"`
}

// MockWriteOn reports whether the MCP mock write tools are on (default true).
func (c MCPConfig) MockWriteOn() bool { return c.MockWrite == nil || *c.MockWrite }

// XdebugConfig controls the Xdebug (DBGp) proxy: Xdebug → Buggregator → IDE,
// or Buggregator drives the session itself when there is no IDE.
type XdebugConfig struct {
	Enabled          bool   `yaml:"enabled"`             // Enable the proxy (default: false).
	Addr             string `yaml:"addr"`                // Where Xdebug connects. Default: 127.0.0.1:9003
	IDEAddr          string `yaml:"ide_addr"`            // Where the IDE listens. Default: 127.0.0.1:9004
	AllowEval        bool   `yaml:"allow_eval"`          // Expose the xdebug_eval MCP tool (default: false).
	AllowControl     *bool  `yaml:"allow_control"`       // Expose the MCP control tools: step, continue, stop, wait (default: true).
	Mode             string `yaml:"mode"`                // proxy, auto or standalone. Default: auto
	PauseOnFirstLine bool   `yaml:"pause_on_first_line"` // A session driven here stops at the first line (default: false).
	IdleTimeout      string `yaml:"idle_timeout"`        // Release a session paused here after this long without a command. Default: 10m; 0 = never

	IdleTimeoutDur time.Duration `yaml:"-"` // parsed IdleTimeout: 0 = default, <0 = never
}

// WebhookDef defines a webhook from config.
// MockDef is a read-only mock rule from the config file (see #332): requests
// to /mock<path> get this response.
// MockOpenAPIDef is one OpenAPI spec file of the mocks_openapi section.
type MockOpenAPIDef struct {
	File     string  `yaml:"file"`      // path to the spec, YAML or JSON
	Project  string  `yaml:"project"`   // empty = every project
	Group    string  `yaml:"group"`     // default openapi:<info.title>, openapi:<project>:<info.title> with a project
	BasePath *string `yaml:"base_path"` // default the path of servers[0].url; "" = none
}

type MockDef struct {
	Name    string            `yaml:"name"`
	Method  string            `yaml:"method"` // GET, POST, … or ANY (default)
	Path    string            `yaml:"path"`   // /users/me, /users/{id}, /files/*
	Status  int               `yaml:"status"` // default 200
	Headers map[string]string `yaml:"headers"`
	Body    string            `yaml:"body"`
	Delay   string            `yaml:"delay"`   // e.g. "150ms", at most 30s
	Project string            `yaml:"project"` // empty = every project
	// Template makes body and header values Go templates (request values, fake data).
	Template bool  `yaml:"template"`
	Seed     int64 `yaml:"seed"` // not 0: the same fake data on every request
}

type WebhookDef struct {
	Event     string            `yaml:"event"`      // Event type to match ("sentry", "ray", "*" for all).
	URL       string            `yaml:"url"`        // Target URL.
	Headers   map[string]string `yaml:"headers"`    // Custom HTTP headers.
	VerifySSL *bool             `yaml:"verify_ssl"` // Verify SSL certificates (default: false).
	Retry     *bool             `yaml:"retry"`      // Retry on failure (default: true).
}

// XdebugTraceConfig bounds uploaded Xdebug traces. Sizes take a plain byte
// count or a KB/MB/GB suffix (1024-based).
type XdebugTraceConfig struct {
	MaxUpload string `yaml:"max_upload"` // request body, gzip or not. Default: 100MB
	MaxSize   string `yaml:"max_size"`   // uncompressed bytes read; a longer trace is cut. Default: 256MB
	Keep      int    `yaml:"keep"`       // traces kept, the oldest go first. Default: 20

	MaxUploadBytes int64 `yaml:"-"`
	MaxSizeBytes   int64 `yaml:"-"`
}

// UpdateCheckConfig controls the new-version check (GitHub releases/latest).
type UpdateCheckConfig struct {
	Enabled  *bool  `yaml:"enabled"`  // default true; false = no network call at all
	Interval string `yaml:"interval"` // Go duration between checks, default 12h, at least 1h

	IntervalDur time.Duration `yaml:"-"`
}

// On reports whether the check runs (default true).
func (c UpdateCheckConfig) On() bool { return c.Enabled == nil || *c.Enabled }

// UpdateConfig controls updating from the UI (binary installs only).
type UpdateConfig struct {
	// SelfUpdate: "auto" (default) the UI may update a binary install unless
	// the server listens beyond loopback without auth; "on" also then; "off"
	// never.
	SelfUpdate string `yaml:"self_update"`
	// ReleaseURL replaces the GitHub releases/latest API (a mirror); empty =
	// GitHub. Downloads are still checked against the built-in release key.
	ReleaseURL string `yaml:"release_url"`
}

// SMTPRelayConfig is the outgoing SMTP server for releasing a caught email
// to a real inbox. The password comes from the env or a file, never back out
// through the API.
type SMTPRelayConfig struct {
	Host               string   `yaml:"host"`                 // empty = release off
	Port               int      `yaml:"port"`                 // default 587 (starttls), 465 (tls), 25 (none)
	TLS                string   `yaml:"tls"`                  // starttls (default), tls or none
	Username           string   `yaml:"username"`             // empty = no AUTH
	Password           string   `yaml:"password"`             // prefer SMTP_RELAY_PASSWORD or password_file
	PasswordFile       string   `yaml:"password_file"`        // file holding the password (a Docker/K8s secret)
	From               string   `yaml:"from"`                 // MAIL FROM; empty = the message's From
	AllowedRecipients  []string `yaml:"allowed_recipients"`   // patterns like "*@example.com"; empty = any
	InsecureSkipVerify bool     `yaml:"insecure_skip_verify"` // accept any TLS certificate
}

// BroadcastConfig is the module broadcast. Without an upstream events are
// only captured; with one, the request also goes to the real server.
type BroadcastConfig struct {
	Upstream string               `yaml:"upstream"` // e.g. http://127.0.0.1:8080 (Reverb); empty = capture only
	Apps     []BroadcastAppConfig `yaml:"apps"`     // secrets to check signatures; none = not checked
}

// BroadcastAppConfig is one app's key and secret.
type BroadcastAppConfig struct {
	Key    string `yaml:"key"`
	Secret string `yaml:"secret"`
}

// Secrets maps each app key to its secret.
func (b BroadcastConfig) Secrets() map[string]string {
	m := map[string]string{}
	for _, a := range b.Apps {
		if k := strings.TrimSpace(a.Key); k != "" {
			m[k] = a.Secret
		}
	}
	return m
}

// RetentionConfig is the optional auto-cleanup. Pinned events are never
// deleted and do not count towards max_events.
type RetentionConfig struct {
	MaxAge    string `yaml:"max_age"`    // delete events older than this: "30m", "24h", "7d"; empty or 0 = keep
	MaxEvents int    `yaml:"max_events"` // keep at most this many events in total, newest first; 0 = no cap

	MaxAgeDur time.Duration `yaml:"-"` // parsed MaxAge, 0 = off
}

// SentryConfig controls the Sentry module.
type SentryConfig struct {
	// SourceFetch: where the server may fetch the source of JS stack frames
	// (frame filename is a URL): "private" (default) loopback and private
	// hosts only, "any" any host, "off" never.
	SourceFetch string `yaml:"source_fetch"`
}

// MetricsConfig controls Prometheus metrics.
type MetricsConfig struct {
	Enabled bool   `yaml:"enabled"`
	Addr    string `yaml:"addr"` // Separate metrics server address (e.g., ":9090"). Empty = use main HTTP server.
}

type ServerConfig struct {
	Addr        string   `yaml:"addr"`
	CORSOrigins []string `yaml:"cors_origins"` // Allowed CORS origins. Empty = all ("*"), so browser SDKs (Sentry JS) work from any page.
}

type StorageConfig struct {
	Mode string `yaml:"mode"` // "memory" (default) or "filesystem"
	Path string `yaml:"path"` // Directory for filesystem mode
}

type DatabaseConfig struct {
	Driver string `yaml:"driver"`
	DSN    string `yaml:"dsn"`

	// MaxOpenConns is the size of the SQLite connection pool (default 4).
	//
	// With a single connection ingestion and UI queries share one queue: a heavy
	// list slows down writes, and writes slow down the list. In WAL mode readers
	// disturb neither each other nor the writer, so a pool is safe; the writer is
	// still serialized by SQLite itself, and busy_timeout in the DSN makes a
	// competing write wait instead of failing.
	MaxOpenConns int `yaml:"max_open_conns"`
}

type TCPConfig struct {
	SMTP      TCPServerConfig `yaml:"smtp"`
	Monolog   TCPServerConfig `yaml:"monolog"`
	VarDumper TCPServerConfig `yaml:"var-dumper"`
	Proxy     TCPServerConfig `yaml:"proxy"`
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
	CSP       *bool `yaml:"csp"`       // browser CSP / Reporting API reports
	Broadcast *bool `yaml:"broadcast"` // Pusher HTTP API events (Laravel Reverb, Pusher)
	HttpDump  *bool `yaml:"http-dump"`
	Profiler  *bool `yaml:"profiler"`
	// XdebugTrace is the Xdebug function trace upload (type xdebug-trace).
	XdebugTrace *bool `yaml:"xdebug-trace"`
}

// field is the setting of a module type; nil for an unknown type.
func (m *ModulesConfig) field(moduleType string) **bool {
	switch moduleType {
	case "sentry":
		return &m.Sentry
	case "ray":
		return &m.Ray
	case "var-dump":
		return &m.VarDump
	case "inspector":
		return &m.Inspector
	case "monolog":
		return &m.Monolog
	case "smtp":
		return &m.SMTP
	case "sms":
		return &m.SMS
	case "csp":
		return &m.CSP
	case "broadcast":
		return &m.Broadcast
	case "http-dump":
		return &m.HttpDump
	case "profiler":
		return &m.Profiler
	case "xdebug-trace":
		return &m.XdebugTrace
	}
	return nil
}

// IsEnabled returns true if a module is enabled (default: true).
func (m ModulesConfig) IsEnabled(moduleType string) bool {
	f := m.field(moduleType)
	if f == nil || *f == nil {
		return true // enabled by default
	}
	return **f
}

// Disable turns a module off, e.g. when it cannot start, so EnabledTypes
// (the UI menu) no longer lists it.
func (m *ModulesConfig) Disable(moduleType string) {
	if f := m.field(moduleType); f != nil {
		off := false
		*f = &off
	}
}

// EnabledTypes returns a list of enabled module type strings.
func (m ModulesConfig) EnabledTypes() []string {
	all := []string{"sentry", "ray", "var-dump", "inspector", "monolog", "smtp", "sms", "csp", "broadcast", "http-dump", "profiler", "xdebug-trace"}
	var enabled []string
	for _, t := range all {
		if m.IsEnabled(t) {
			enabled = append(enabled, t)
		}
	}
	return enabled
}

// ListenPorts gives the port of each listener that is on, for /api/settings:
// the setup help fills real values. A listener that is off, or whose address
// has no fixed port, is left out; the UI then shows its default.
func (c Config) ListenPorts() map[string]int {
	ports := map[string]int{}
	add := func(name, addr string, on bool) {
		if !on {
			return
		}
		_, p, err := net.SplitHostPort(addr)
		if err != nil {
			return
		}
		if n, err := strconv.Atoi(p); err == nil && n > 0 {
			ports[name] = n
		}
	}
	add("http", c.Server.Addr, true)
	add("smtp", c.TCP.SMTP.Addr, c.Modules.IsEnabled("smtp"))
	add("monolog", c.TCP.Monolog.Addr, c.Modules.IsEnabled("monolog"))
	// VarDumper has no Windows build: main.go never starts its listener there.
	add("var_dumper", c.TCP.VarDumper.Addr, c.Modules.IsEnabled("var-dump") && runtime.GOOS != "windows")
	add("proxy", c.TCP.Proxy.Addr, true)
	add("xdebug", c.Xdebug.Addr, c.Xdebug.Enabled)
	add("mcp", c.MCP.Addr, c.MCP.Enabled && c.MCP.Transport == "http")
	return ports
}

// ProjectDef defines a project from config.
type ProjectDef struct {
	Key  string `yaml:"key"`
	Name string `yaml:"name"`
}

// LoadConfig reads configuration with priority: flags > env > config file > defaults.
// corsOrigins picks the allowed origins: CORS_ORIGINS (comma-separated),
// else the file's list, else all. Allowing all by default is what lets a
// browser SDK (Sentry JS) on any dev origin send to Buggregator.
func corsOrigins(file []string, env string) []string {
	if env != "" {
		return strings.Split(env, ",")
	}
	if len(file) > 0 {
		return file
	}
	return []string{"*"}
}

func LoadConfig() Config {
	var configFile string
	flag.StringVar(&configFile, "config", "", "Path to config file (buggregator.yaml)")

	cfg := Config{}

	flag.StringVar(&cfg.Server.Addr, "http-addr", "", "HTTP listen address")
	flag.StringVar(&cfg.Database.DSN, "db", "", "SQLite DSN")
	flag.StringVar(&cfg.TCP.SMTP.Addr, "smtp-addr", "", "SMTP listen address")
	flag.StringVar(&cfg.TCP.Monolog.Addr, "monolog-addr", "", "Monolog TCP listen address")
	flag.StringVar(&cfg.TCP.VarDumper.Addr, "vardumper-addr", "", "VarDumper TCP listen address")
	var uiDefaultLimit, uiMaxLimit string
	flag.StringVar(&uiDefaultLimit, "ui-default-limit", "", "Events per list without a limit param (0 = unlimited)")
	flag.StringVar(&uiMaxLimit, "ui-max-limit", "", "Ceiling for an explicit list limit (0 = no ceiling)")
	var retentionMaxEvents string
	flag.StringVar(&cfg.Retention.MaxAge, "retention-max-age", "", "Delete events older than this (30m, 24h, 7d); empty = keep")
	flag.StringVar(&retentionMaxEvents, "retention-max-events", "", "Keep at most this many events (0 = no cap)")
	flag.StringVar(&cfg.Update.SelfUpdate, "update-self", "", "Update from the UI: auto (default), on, off")
	flag.StringVar(&cfg.Sentry.SourceFetch, "sentry-source-fetch", "", "Fetch JS frame sources from: private (default), any, off")
	flag.Parse()

	fileCfg := loadConfigFile(configFile)

	// Merge: flag > env > file > default.
	cfg.Server.Addr = coalesce(cfg.Server.Addr, os.Getenv("HTTP_ADDR"), fileCfg.Server.Addr, ":8000")
	cfg.Database.DSN = coalesce(cfg.Database.DSN, os.Getenv("DATABASE_DSN"), fileCfg.Database.DSN, ":memory:")
	cfg.Database.Driver = coalesce(fileCfg.Database.Driver, "sqlite")
	cfg.Database.MaxOpenConns = 4
	if v := fileCfg.Database.MaxOpenConns; v > 0 {
		cfg.Database.MaxOpenConns = v
	}
	if v, err := strconv.Atoi(strings.TrimSpace(os.Getenv("DATABASE_MAX_OPEN_CONNS"))); err == nil && v > 0 {
		cfg.Database.MaxOpenConns = v
	}
	cfg.TCP.SMTP.Addr = coalesce(cfg.TCP.SMTP.Addr, os.Getenv("SMTP_ADDR"), fileCfg.TCP.SMTP.Addr, ":1025")
	cfg.TCP.Monolog.Addr = coalesce(cfg.TCP.Monolog.Addr, os.Getenv("MONOLOG_ADDR"), fileCfg.TCP.Monolog.Addr, ":9913")
	cfg.TCP.VarDumper.Addr = coalesce(cfg.TCP.VarDumper.Addr, os.Getenv("VAR_DUMPER_ADDR"), fileCfg.TCP.VarDumper.Addr, ":9912")
	cfg.TCP.Proxy.Addr = coalesce(os.Getenv("PROXY_ADDR"), fileCfg.TCP.Proxy.Addr, ":8080")

	// Storage.
	cfg.Storage.Mode = coalesce(os.Getenv("STORAGE_MODE"), fileCfg.Storage.Mode, "memory")
	cfg.Storage.Path = coalesce(os.Getenv("STORAGE_PATH"), fileCfg.Storage.Path, "./storage")

	// Metrics.
	cfg.Metrics.Enabled = fileCfg.Metrics.Enabled || os.Getenv("METRICS_ENABLED") == "true"
	cfg.Metrics.Addr = coalesce(os.Getenv("METRICS_ADDR"), fileCfg.Metrics.Addr)

	// New-version check.
	updateCheck := envBool("UPDATE_CHECK_ENABLED", fileCfg.UpdateCheck.Enabled == nil || *fileCfg.UpdateCheck.Enabled)
	cfg.UpdateCheck.Enabled = &updateCheck
	cfg.UpdateCheck.Interval = coalesce(os.Getenv("UPDATE_CHECK_INTERVAL"), fileCfg.UpdateCheck.Interval, "12h")
	if d, err := time.ParseDuration(strings.TrimSpace(cfg.UpdateCheck.Interval)); err == nil && d > 0 {
		if d < time.Hour {
			slog.Warn("update_check.interval is below 1h, using 1h", "value", cfg.UpdateCheck.Interval)
			d = time.Hour
		}
		cfg.UpdateCheck.IntervalDur = d
	} else {
		slog.Warn("update_check.interval is not a positive duration, using 12h", "value", cfg.UpdateCheck.Interval)
	}

	// Update from the UI.
	cfg.Update.SelfUpdate = strings.ToLower(strings.TrimSpace(coalesce(cfg.Update.SelfUpdate, os.Getenv("UPDATE_SELF"), fileCfg.Update.SelfUpdate, "auto")))
	if v := cfg.Update.SelfUpdate; v != "auto" && v != "on" && v != "off" {
		slog.Warn("update.self_update must be auto, on or off, using auto", "value", v)
		cfg.Update.SelfUpdate = "auto"
	}
	cfg.Update.ReleaseURL = coalesce(os.Getenv("UPDATE_RELEASE_URL"), fileCfg.Update.ReleaseURL)

	// Xdebug traces.
	cfg.XdebugTrace.MaxUpload = coalesce(os.Getenv("XDEBUG_TRACE_MAX_UPLOAD"), fileCfg.XdebugTrace.MaxUpload, "100MB")
	cfg.XdebugTrace.MaxSize = coalesce(os.Getenv("XDEBUG_TRACE_MAX_SIZE"), fileCfg.XdebugTrace.MaxSize, "256MB")
	cfg.XdebugTrace.MaxUploadBytes = sizeOr("xdebug_trace.max_upload", cfg.XdebugTrace.MaxUpload, 100<<20)
	cfg.XdebugTrace.MaxSizeBytes = sizeOr("xdebug_trace.max_size", cfg.XdebugTrace.MaxSize, 256<<20)
	cfg.XdebugTrace.Keep = fileCfg.XdebugTrace.Keep
	if v := strings.TrimSpace(os.Getenv("XDEBUG_TRACE_KEEP")); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.XdebugTrace.Keep = n
		}
	}
	if cfg.XdebugTrace.Keep <= 0 {
		cfg.XdebugTrace.Keep = 20
	}

	// MCP.
	cfg.MCP.Enabled = fileCfg.MCP.Enabled || os.Getenv("MCP_ENABLED") == "true"
	cfg.MCP.Transport = coalesce(os.Getenv("MCP_TRANSPORT"), fileCfg.MCP.Transport, "socket")
	cfg.MCP.SocketPath = coalesce(os.Getenv("MCP_SOCKET_PATH"), fileCfg.MCP.SocketPath, "/tmp/buggregator-mcp.sock")
	cfg.MCP.Addr = coalesce(os.Getenv("MCP_ADDR"), fileCfg.MCP.Addr, ":8001")
	cfg.MCP.AuthToken = coalesce(os.Getenv("MCP_AUTH_TOKEN"), fileCfg.MCP.AuthToken)
	mockWrite := envBool("MCP_MOCK_WRITE", fileCfg.MCP.MockWriteOn())
	cfg.MCP.MockWrite = &mockWrite

	// Xdebug proxy.
	cfg.Xdebug.Enabled = envBool("XDEBUG_PROXY_ENABLED", fileCfg.Xdebug.Enabled)
	cfg.Xdebug.Addr = coalesce(os.Getenv("XDEBUG_PROXY_ADDR"), fileCfg.Xdebug.Addr, "127.0.0.1:9003")
	cfg.Xdebug.IDEAddr = coalesce(os.Getenv("XDEBUG_PROXY_IDE_ADDR"), fileCfg.Xdebug.IDEAddr, "127.0.0.1:9004")
	cfg.Xdebug.AllowEval = envBool("XDEBUG_PROXY_ALLOW_EVAL", fileCfg.Xdebug.AllowEval)
	allowControl := envBool("XDEBUG_PROXY_ALLOW_CONTROL", fileCfg.Xdebug.AllowControl == nil || *fileCfg.Xdebug.AllowControl)
	cfg.Xdebug.AllowControl = &allowControl
	cfg.Xdebug.Mode = coalesce(os.Getenv("XDEBUG_PROXY_MODE"), fileCfg.Xdebug.Mode, "auto")
	if m := cfg.Xdebug.Mode; m != "proxy" && m != "auto" && m != "standalone" {
		slog.Warn("xdebug.mode must be proxy, auto or standalone, using auto", "value", m)
		cfg.Xdebug.Mode = "auto"
	}
	cfg.Xdebug.PauseOnFirstLine = envBool("XDEBUG_PROXY_PAUSE_ON_FIRST_LINE", fileCfg.Xdebug.PauseOnFirstLine)
	cfg.Xdebug.IdleTimeout = coalesce(os.Getenv("XDEBUG_PROXY_IDLE_TIMEOUT"), fileCfg.Xdebug.IdleTimeout, "10m")
	cfg.Xdebug.IdleTimeoutDur = parseIdleTimeout(cfg.Xdebug.IdleTimeout)

	// Auth.
	cfg.Auth.Enabled = fileCfg.Auth.Enabled || os.Getenv("AUTH_ENABLED") == "true"
	cfg.Auth.Provider = coalesce(os.Getenv("AUTH_PROVIDER"), fileCfg.Auth.Provider, "oidc")
	cfg.Auth.ProviderURL = coalesce(os.Getenv("AUTH_PROVIDER_URL"), fileCfg.Auth.ProviderURL)
	cfg.Auth.ClientID = coalesce(os.Getenv("AUTH_CLIENT_ID"), fileCfg.Auth.ClientID)
	cfg.Auth.ClientSecret = coalesce(os.Getenv("AUTH_CLIENT_SECRET"), fileCfg.Auth.ClientSecret)
	cfg.Auth.CallbackURL = coalesce(os.Getenv("AUTH_CALLBACK_URL"), fileCfg.Auth.CallbackURL)
	cfg.Auth.Scopes = coalesce(os.Getenv("AUTH_SCOPES"), fileCfg.Auth.Scopes, "openid,email,profile")
	cfg.Auth.JWTSecret = coalesce(os.Getenv("AUTH_JWT_SECRET"), fileCfg.Auth.JWTSecret)

	// UI list limits.
	cfg.UI.DefaultLimit = resolveLimit(0, fileCfg.UI.DefaultLimit, uiDefaultLimit, os.Getenv("UI_DEFAULT_LIMIT"))
	cfg.UI.MaxLimit = resolveLimit(5000, fileCfg.UI.MaxLimit, uiMaxLimit, os.Getenv("UI_MAX_LIMIT"))
	cfg.UI.DefaultWindow = coalesce(os.Getenv("UI_DEFAULT_WINDOW"), fileCfg.UI.DefaultWindow)

	cfg.Retention.MaxAge = coalesce(cfg.Retention.MaxAge, os.Getenv("RETENTION_MAX_AGE"), fileCfg.Retention.MaxAge)
	cfg.Retention.MaxAgeDur = parseRetentionAge(cfg.Retention.MaxAge)
	cfg.Retention.MaxEvents = fileCfg.Retention.MaxEvents
	if v := coalesce(retentionMaxEvents, os.Getenv("RETENTION_MAX_EVENTS")); v != "" {
		n, err := strconv.Atoi(strings.TrimSpace(v))
		if err != nil || n < 0 {
			slog.Warn("retention.max_events is not a whole number, no cap", "value", v)
			n = 0
		}
		cfg.Retention.MaxEvents = n
	}
	if cfg.Retention.MaxEvents < 0 {
		cfg.Retention.MaxEvents = 0
	}
	cfg.SMTPRelay = smtpRelayConfig(fileCfg.SMTPRelay)
	cfg.Broadcast = broadcastConfig(fileCfg.Broadcast)
	cfg.Sentry.SourceFetch = sourceFetchMode(coalesce(cfg.Sentry.SourceFetch, os.Getenv("SENTRY_SOURCE_FETCH"), fileCfg.Sentry.SourceFetch, "private"))

	// CORS origins.
	cfg.Server.CORSOrigins = corsOrigins(fileCfg.Server.CORSOrigins, os.Getenv("CORS_ORIGINS"))

	// Modules: merge from file config, env override.
	cfg.Modules = fileCfg.Modules
	if env := os.Getenv("CLIENT_SUPPORTED_EVENTS"); env != "" {
		cfg.Modules = modulesFromCSV(env)
	}

	// Webhooks from config file.
	cfg.Webhooks = fileCfg.Webhooks

	// Projects from config file.
	cfg.Projects = fileCfg.Projects

	// Mock rules from config file (read-only in the UI).
	cfg.Mocks = fileCfg.Mocks
	cfg.MocksOpenAPI = fileCfg.MocksOpenAPI

	// Set shortcuts.
	cfg.HTTPAddr = cfg.Server.Addr
	cfg.DatabaseDSN = cfg.Database.DSN
	cfg.SMTPAddr = cfg.TCP.SMTP.Addr
	cfg.MonologAddr = cfg.TCP.Monolog.Addr
	cfg.VarDumperAddr = cfg.TCP.VarDumper.Addr
	cfg.ProxyAddr = cfg.TCP.Proxy.Addr

	return cfg
}

// parseRetentionAge reads retention.max_age: a Go duration or whole days
// ("7d"). Empty, "0" or a bad value keep everything (a bad one with a warning).
func parseRetentionAge(v string) time.Duration {
	v = strings.TrimSpace(v)
	if v == "" || v == "0" {
		return 0
	}
	var d time.Duration
	var err error
	if days, ok := strings.CutSuffix(v, "d"); ok {
		var n int
		n, err = strconv.Atoi(days)
		d = time.Duration(n) * 24 * time.Hour
	} else {
		d, err = time.ParseDuration(v)
	}
	if err != nil || d <= 0 {
		slog.Warn("retention.max_age is not a duration like 24h or 7d; keeping all events", "value", v)
		return 0
	}
	return d
}

// sourceFetchMode checks sentry.source_fetch; an unknown value is the safe
// default, private.
func sourceFetchMode(v string) string {
	switch m := strings.ToLower(strings.TrimSpace(v)); m {
	case "private", "any", "off":
		return m
	default:
		slog.Warn("sentry.source_fetch must be private, any or off; using private", "value", v)
		return "private"
	}
}

// parseIdleTimeout reads xdebug.idle_timeout: a Go duration ("10m", "90s");
// "0" switches the guard off. A bad value keeps the default.
func parseIdleTimeout(v string) time.Duration {
	d, err := time.ParseDuration(strings.TrimSpace(v))
	switch {
	case strings.TrimSpace(v) == "0":
		return -1
	case err != nil:
		slog.Warn("xdebug.idle_timeout is not a duration, using 10m", "value", v)
		return 0
	case d <= 0:
		return -1
	}
	return d
}

// AllowControlOn reports whether the MCP control tools are on (default true).
func (c XdebugConfig) AllowControlOn() bool { return c.AllowControl == nil || *c.AllowControl }

// parseSize reads "123", "64KB", "100MB", "1GB" (1024-based, case does
// not matter, "B" and "iB" accepted).
func parseSize(v string) (int64, error) {
	v = strings.ToUpper(strings.TrimSpace(v))
	mult := int64(1)
	for _, u := range []struct {
		suffix string
		mult   int64
	}{{"GIB", 1 << 30}, {"MIB", 1 << 20}, {"KIB", 1 << 10}, {"GB", 1 << 30}, {"MB", 1 << 20}, {"KB", 1 << 10}, {"G", 1 << 30}, {"M", 1 << 20}, {"K", 1 << 10}, {"B", 1}} {
		if strings.HasSuffix(v, u.suffix) {
			v, mult = strings.TrimSpace(strings.TrimSuffix(v, u.suffix)), u.mult
			break
		}
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil || n <= 0 || n > (1<<62)/mult {
		return 0, fmt.Errorf("not a positive size: %q", v)
	}
	return n * mult, nil
}

// sizeOr parses a size setting, logging and using def when it is invalid.
func sizeOr(name, v string, def int64) int64 {
	n, err := parseSize(v)
	if err != nil {
		slog.Warn(name+" is not a valid size, using the default", "value", v, "default", def)
		return def
	}
	return n
}

// envBool returns the env var as a bool when it is set ("true"/"false"),
// otherwise def. Unlike "file || env", the env can also switch a setting off.
func envBool(name string, def bool) bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(name))) {
	case "true", "1", "yes":
		return true
	case "false", "0", "no":
		return false
	}
	return def
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
		CSP:       boolPtr(enabled["csp"], t, f),
		Broadcast: boolPtr(enabled["broadcast"], t, f),
		HttpDump:  boolPtr(enabled["http-dump"], t, f),
		Profiler:  boolPtr(enabled["profiler"], t, f),

		XdebugTrace: boolPtr(enabled["xdebug-trace"], t, f),
	}
}

func boolPtr(cond bool, yes, no bool) *bool {
	if cond {
		return &yes
	}
	return &no
}

// broadcastConfig fills the module from BROADCAST_UPSTREAM and
// BROADCAST_APPS ("key:secret,key2:secret2") over the file.
func broadcastConfig(file BroadcastConfig) BroadcastConfig {
	c := file
	c.Upstream = coalesce(os.Getenv("BROADCAST_UPSTREAM"), file.Upstream)
	if v := os.Getenv("BROADCAST_APPS"); v != "" {
		c.Apps = nil
		for _, kv := range strings.Split(v, ",") {
			k, sec, ok := strings.Cut(strings.TrimSpace(kv), ":")
			if !ok || strings.TrimSpace(k) == "" {
				slog.Warn("BROADCAST_APPS: want key:secret pairs, skipping one")
				continue
			}
			c.Apps = append(c.Apps, BroadcastAppConfig{Key: strings.TrimSpace(k), Secret: sec})
		}
	}
	return c
}

// smtpRelayConfig fills the relay from SMTP_RELAY_* env vars over the file.
// The password: SMTP_RELAY_PASSWORD, else the file named by
// SMTP_RELAY_PASSWORD_FILE or password_file, else the YAML value.
func smtpRelayConfig(file SMTPRelayConfig) SMTPRelayConfig {
	c := file
	c.Host = coalesce(os.Getenv("SMTP_RELAY_HOST"), file.Host)
	if v := strings.TrimSpace(os.Getenv("SMTP_RELAY_PORT")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n < 65536 {
			c.Port = n
		} else {
			slog.Warn("SMTP_RELAY_PORT is not a port, using the default", "value", v)
		}
	}
	c.TLS = strings.ToLower(coalesce(os.Getenv("SMTP_RELAY_TLS"), file.TLS, "starttls"))
	switch c.TLS {
	case "starttls", "tls", "none":
	default:
		slog.Warn("smtp_relay.tls must be starttls, tls or none; using starttls", "value", c.TLS)
		c.TLS = "starttls"
	}
	c.Username = coalesce(os.Getenv("SMTP_RELAY_USERNAME"), file.Username)
	c.From = coalesce(os.Getenv("SMTP_RELAY_FROM"), file.From)
	if v := os.Getenv("SMTP_RELAY_ALLOWED_RECIPIENTS"); v != "" {
		c.AllowedRecipients = nil
		for _, p := range strings.Split(v, ",") {
			if p = strings.TrimSpace(p); p != "" {
				c.AllowedRecipients = append(c.AllowedRecipients, p)
			}
		}
	}
	c.InsecureSkipVerify = envBool("SMTP_RELAY_INSECURE_SKIP_VERIFY", file.InsecureSkipVerify)
	c.PasswordFile = coalesce(os.Getenv("SMTP_RELAY_PASSWORD_FILE"), file.PasswordFile)
	if v := os.Getenv("SMTP_RELAY_PASSWORD"); v != "" {
		c.Password = v
	} else if c.PasswordFile != "" {
		b, err := os.ReadFile(c.PasswordFile)
		if err != nil {
			slog.Warn("smtp_relay: cannot read the password file", "file", c.PasswordFile, "err", err)
		} else {
			c.Password = strings.TrimRight(string(b), "\r\n")
		}
	}
	return c
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

// resolveLimit picks a list limit by priority: the first of values (flag, then
// env) that is a number >= 0, then the config file, then def. 0 is a real
// value (no limit); only an empty or unreadable value counts as "not set".
func resolveLimit(def int, file *int, values ...string) *int {
	for _, s := range values {
		if n, err := strconv.Atoi(strings.TrimSpace(s)); err == nil && n >= 0 {
			return &n
		}
	}
	if file != nil && *file >= 0 {
		n := *file
		return &n
	}
	return &def
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
	return fmt.Sprintf("http=%s db=%s smtp=%s monolog=%s vardumper=%s proxy=%s modules=%v",
		c.Server.Addr, c.Database.DSN, c.TCP.SMTP.Addr, c.TCP.Monolog.Addr, c.TCP.VarDumper.Addr,
		c.TCP.Proxy.Addr, c.Modules.EnabledTypes())
}
