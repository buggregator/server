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

// Config holds application configuration.
type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Database DatabaseConfig `yaml:"database"`
	TCP      TCPConfig      `yaml:"tcp"`
	Version  string         `yaml:"-"`

	// Shortcuts for backward compatibility.
	HTTPAddr      string `yaml:"-"`
	DatabaseDSN   string `yaml:"-"`
	SMTPAddr      string `yaml:"-"`
	MonologAddr   string `yaml:"-"`
	VarDumperAddr string `yaml:"-"`
}

type ServerConfig struct {
	Addr string `yaml:"addr"` // HTTP listen address
}

type DatabaseConfig struct {
	Driver string `yaml:"driver"` // "sqlite" (default)
	DSN    string `yaml:"dsn"`    // ":memory:", file path, or connection string
}

type TCPConfig struct {
	SMTP      TCPServerConfig `yaml:"smtp"`
	Monolog   TCPServerConfig `yaml:"monolog"`
	VarDumper TCPServerConfig `yaml:"var-dumper"`
}

type TCPServerConfig struct {
	Addr string `yaml:"addr"`
}

// LoadConfig reads configuration with priority: flags > env > config file > defaults.
func LoadConfig() Config {
	var configFile string
	flag.StringVar(&configFile, "config", "", "Path to config file (buggregator.yaml)")

	cfg := Config{Version: "1.0.0"}

	// Parse flags first to get --config path.
	flag.StringVar(&cfg.Server.Addr, "http-addr", "", "HTTP listen address")
	flag.StringVar(&cfg.Database.DSN, "db", "", "SQLite DSN")
	flag.StringVar(&cfg.TCP.SMTP.Addr, "smtp-addr", "", "SMTP listen address")
	flag.StringVar(&cfg.TCP.Monolog.Addr, "monolog-addr", "", "Monolog TCP listen address")
	flag.StringVar(&cfg.TCP.VarDumper.Addr, "vardumper-addr", "", "VarDumper TCP listen address")
	flag.Parse()

	// Load config file (if specified or auto-detected).
	fileCfg := loadConfigFile(configFile)

	// Merge: flag > env > file > default.
	cfg.Server.Addr = coalesce(cfg.Server.Addr, os.Getenv("HTTP_ADDR"), fileCfg.Server.Addr, ":8000")
	cfg.Database.DSN = coalesce(cfg.Database.DSN, os.Getenv("DATABASE_DSN"), fileCfg.Database.DSN, ":memory:")
	cfg.Database.Driver = coalesce(fileCfg.Database.Driver, "sqlite")
	cfg.TCP.SMTP.Addr = coalesce(cfg.TCP.SMTP.Addr, os.Getenv("SMTP_ADDR"), fileCfg.TCP.SMTP.Addr, ":1025")
	cfg.TCP.Monolog.Addr = coalesce(cfg.TCP.Monolog.Addr, os.Getenv("MONOLOG_ADDR"), fileCfg.TCP.Monolog.Addr, ":9913")
	cfg.TCP.VarDumper.Addr = coalesce(cfg.TCP.VarDumper.Addr, os.Getenv("VAR_DUMPER_ADDR"), fileCfg.TCP.VarDumper.Addr, ":9912")

	// Set shortcuts for backward compat.
	cfg.HTTPAddr = cfg.Server.Addr
	cfg.DatabaseDSN = cfg.Database.DSN
	cfg.SMTPAddr = cfg.TCP.SMTP.Addr
	cfg.MonologAddr = cfg.TCP.Monolog.Addr
	cfg.VarDumperAddr = cfg.TCP.VarDumper.Addr

	return cfg
}

// loadConfigFile tries to load a YAML config file.
// If path is empty, auto-detects buggregator.yaml in current directory.
func loadConfigFile(path string) Config {
	var cfg Config

	if path == "" {
		// Auto-detect config file.
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

	// Expand environment variables: ${VAR} and ${VAR:default}.
	expanded := expandEnvVars(string(data))

	if err := yaml.Unmarshal([]byte(expanded), &cfg); err != nil {
		slog.Warn("failed to parse config file", "path", path, "err", err)
		return cfg
	}

	slog.Info("loaded config file", "path", path)
	return cfg
}

// envVarPattern matches ${VAR} and ${VAR:default} patterns.
var envVarPattern = regexp.MustCompile(`\$\{([^}:]+)(?::([^}]*))?\}`)

// expandEnvVars replaces ${VAR} and ${VAR:default} with environment variable values.
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

// coalesce returns the first non-empty string.
func coalesce(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

// String returns a human-readable config summary.
func (c Config) String() string {
	return fmt.Sprintf("http=%s db=%s smtp=%s monolog=%s vardumper=%s",
		c.Server.Addr, c.Database.DSN, c.TCP.SMTP.Addr, c.TCP.Monolog.Addr, c.TCP.VarDumper.Addr)
}
