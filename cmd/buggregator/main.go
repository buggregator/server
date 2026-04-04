package main

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/buggregator/go-buggregator/internal/app"
	mcpserver "github.com/buggregator/go-buggregator/internal/mcp"
	"github.com/buggregator/go-buggregator/internal/metrics"
	"github.com/buggregator/go-buggregator/internal/module"
	httpserver "github.com/buggregator/go-buggregator/internal/server/http"
	"github.com/buggregator/go-buggregator/internal/server/ws"
	"github.com/buggregator/go-buggregator/internal/storage"
	"github.com/buggregator/go-buggregator/modules/httpdumps"
	"github.com/buggregator/go-buggregator/modules/inspector"
	"github.com/buggregator/go-buggregator/modules/monolog"
	"github.com/buggregator/go-buggregator/modules/profiler"
	"github.com/buggregator/go-buggregator/modules/ray"
	"github.com/buggregator/go-buggregator/modules/sentry"
	"github.com/buggregator/go-buggregator/modules/sms"
	"github.com/buggregator/go-buggregator/modules/proxy"
	smtpmod "github.com/buggregator/go-buggregator/modules/smtp"
	"github.com/buggregator/go-buggregator/modules/vardumper"
	"github.com/buggregator/go-buggregator/modules/webhooks"
)

// version is set at build time via -ldflags "-X main.version=...".
var version = "dev"

func main() {
	// MCP proxy subcommand: "buggregator mcp" bridges stdio to the running instance.
	if len(os.Args) > 1 && os.Args[1] == "mcp" {
		socketPath := os.Getenv("MCP_SOCKET_PATH")
		if socketPath == "" {
			socketPath = "/tmp/buggregator-mcp.sock"
		}
		if err := mcpserver.RunProxy(socketPath); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))

	cfg := app.LoadConfig()
	cfg.Version = version

	db, err := storage.Open(cfg.DatabaseDSN)
	if err != nil {
		slog.Error("failed to open database", "err", err)
		os.Exit(1)
	}
	defer db.Close()

	// Seed projects from config file.
	for _, p := range cfg.Projects {
		db.Exec(`INSERT OR IGNORE INTO projects (key, name) VALUES (?, ?)`, p.Key, p.Name)
	}

	// Create metrics collector if enabled.
	var collector *metrics.Collector
	if cfg.Metrics.Enabled {
		collector = metrics.NewCollector()
		slog.Info("prometheus metrics enabled")
	}

	store := storage.NewSQLiteStore(db)
	attachments := storage.NewAttachmentStore(cfg.Storage.Mode, cfg.Storage.Path)
	hub := ws.NewHub()
	registry := module.NewRegistry()
	enabled := cfg.Modules

	// TCP modules.
	monologMod := monolog.New(cfg.MonologAddr)
	smtpMod := smtpmod.New(cfg.SMTPAddr, attachments, db)
	vardumperMod := vardumper.New(cfg.VarDumperAddr)
	proxyMod := proxy.New(cfg.ProxyAddr)

	// Start VarDumper PHP parser (only if enabled).
	if enabled.IsEnabled("var-dump") {
		if err := vardumperMod.StartPHP(); err != nil {
			slog.Error("failed to start VarDumper PHP parser", "err", err)
			os.Exit(1)
		}
		defer vardumperMod.StopPHP()
	}

	// Register enabled modules.
	if enabled.IsEnabled("sentry") {
		registry.Register(sentry.New())
	}
	if enabled.IsEnabled("ray") {
		registry.Register(ray.New())
	}
	if enabled.IsEnabled("inspector") {
		registry.Register(inspector.New())
	}
	if enabled.IsEnabled("profiler") {
		registry.Register(profiler.New())
	}
	if enabled.IsEnabled("monolog") {
		registry.Register(monologMod)
	}
	if enabled.IsEnabled("smtp") {
		registry.Register(smtpMod)
	}
	if enabled.IsEnabled("var-dump") {
		registry.Register(vardumperMod)
	}
	if enabled.IsEnabled("sms") {
		registry.Register(sms.New())
	}
	// HTTP proxy — stores events as http-dump with response data.
	registry.Register(proxyMod)

	// Register webhooks module if any webhooks are configured.
	if len(cfg.Webhooks) > 0 {
		whConfigs := make([]webhooks.WebhookConfig, len(cfg.Webhooks))
		for i, w := range cfg.Webhooks {
			retry := true
			if w.Retry != nil {
				retry = *w.Retry
			}
			verifySSL := false
			if w.VerifySSL != nil {
				verifySSL = *w.VerifySSL
			}
			whConfigs[i] = webhooks.WebhookConfig{
				Event:     w.Event,
				URL:       w.URL,
				Headers:   w.Headers,
				VerifySSL: verifySSL,
				Retry:     retry,
			}
		}
		registry.Register(webhooks.New(whConfigs, collector))
		slog.Info("webhooks registered", "count", len(whConfigs))
	}

	if enabled.IsEnabled("http-dump") {
		registry.Register(httpdumps.New(attachments, db)) // catch-all, must be last
	}

	// Build event service and inject into TCP modules.
	eventService := httpserver.NewEventService(store, hub, registry, collector)
	if enabled.IsEnabled("monolog") {
		monologMod.SetEventService(eventService)
	}
	if enabled.IsEnabled("smtp") {
		smtpMod.SetEventService(eventService)
	}
	if enabled.IsEnabled("var-dump") {
		vardumperMod.SetEventService(eventService)
	}
	proxyMod.SetEventService(eventService)

	application := app.New(cfg, db, registry, hub, store, attachments, collector)
	application.Run()
}
