package main

import (
	"log/slog"
	"os"

	"github.com/buggregator/go-buggregator/internal/app"
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
	smtpmod "github.com/buggregator/go-buggregator/modules/smtp"
)

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))

	cfg := app.LoadConfig()

	// Open database.
	db, err := storage.Open(cfg.DatabaseDSN)
	if err != nil {
		slog.Error("failed to open database", "err", err)
		os.Exit(1)
	}
	defer db.Close()

	store := storage.NewSQLiteStore(db)
	hub := ws.NewHub()
	registry := module.NewRegistry()

	// TCP modules need the event service, which needs the registry.
	// We set it after construction.
	monologMod := monolog.New(cfg.MonologAddr)
	smtpMod := smtpmod.New(cfg.SMTPAddr)

	// Register modules — adding a new one is one line here.
	registry.Register(sentry.New())
	registry.Register(ray.New())
	registry.Register(inspector.New())
	registry.Register(profiler.New())
	registry.Register(monologMod)
	registry.Register(smtpMod)
	registry.Register(httpdumps.New()) // catch-all, must be last

	// Build the event service and inject into TCP modules.
	eventService := httpserver.NewEventService(store, hub, registry)
	monologMod.SetEventService(eventService)
	smtpMod.SetEventService(eventService)

	application := app.New(cfg, db, registry, hub, store)
	application.Run()
}
