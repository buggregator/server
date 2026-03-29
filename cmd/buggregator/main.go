package main

import (
	"log/slog"
	"os"

	"github.com/buggregator/go-buggregator/internal/app"
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
	smtpmod "github.com/buggregator/go-buggregator/modules/smtp"
	"github.com/buggregator/go-buggregator/modules/vardumper"
)

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))

	cfg := app.LoadConfig()

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

	application := app.New(cfg, db, registry, hub, store, attachments, collector)
	application.Run()
}
