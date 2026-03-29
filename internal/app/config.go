package app

import (
	"flag"
	"os"
)

// Config holds application configuration.
type Config struct {
	HTTPAddr    string
	DatabaseDSN string
	SMTPAddr    string
	MonologAddr string
	Version     string
}

// LoadConfig reads configuration from flags and environment variables.
func LoadConfig() Config {
	cfg := Config{
		Version: "1.0.0",
	}

	flag.StringVar(&cfg.HTTPAddr, "http-addr", envOr("HTTP_ADDR", ":8000"), "HTTP listen address")
	flag.StringVar(&cfg.DatabaseDSN, "db", envOr("DATABASE_DSN", ":memory:"), "SQLite DSN (:memory: or file path)")
	flag.StringVar(&cfg.SMTPAddr, "smtp-addr", envOr("SMTP_ADDR", ":1025"), "SMTP listen address")
	flag.StringVar(&cfg.MonologAddr, "monolog-addr", envOr("MONOLOG_ADDR", ":9913"), "Monolog TCP listen address")
	flag.Parse()

	return cfg
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
