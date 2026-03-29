package app

import (
	"context"
	"database/sql"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/buggregator/go-buggregator/internal/event"
	"github.com/buggregator/go-buggregator/internal/frontend"
	"github.com/buggregator/go-buggregator/internal/metrics"
	"github.com/buggregator/go-buggregator/internal/module"
	httpserver "github.com/buggregator/go-buggregator/internal/server/http"
	"github.com/buggregator/go-buggregator/internal/server/tcp"
	"github.com/buggregator/go-buggregator/internal/server/ws"
	"github.com/buggregator/go-buggregator/internal/storage"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// App orchestrates all servers and modules.
type App struct {
	cfg         Config
	db          *sql.DB
	registry    *module.Registry
	hub         *ws.Hub
	store       event.Store
	attachments *storage.AttachmentStore
	metrics     *metrics.Collector
}

func New(cfg Config, db *sql.DB, registry *module.Registry, hub *ws.Hub, store event.Store, attachments *storage.AttachmentStore, m *metrics.Collector) *App {
	return &App{
		cfg:         cfg,
		db:          db,
		registry:    registry,
		hub:         hub,
		store:       store,
		attachments: attachments,
		metrics:     m,
	}
}

// Run starts all servers and blocks until shutdown signal.
func (a *App) Run() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// Wrap store with metrics instrumentation.
	var store event.Store = a.store
	if a.metrics != nil {
		store = metrics.NewInstrumentedStore(a.store, a.metrics, a.db)
	}

	mux := http.NewServeMux()

	// Initialize modules (migrations, routes, handlers).
	if err := a.registry.Init(a.db, mux, store); err != nil {
		slog.Error("failed to initialize modules", "err", err)
		os.Exit(1)
	}

	// Build the event service that ties ingestion -> store -> broadcast.
	eventService := httpserver.NewEventService(store, a.hub, a.registry, a.metrics)

	// Register core API routes.
	httpserver.RegisterAPI(mux, store, a.registry.Previews(), eventService, a.cfg.Version, a.db, a.cfg.Modules.EnabledTypes())

	// Register attachment API endpoints.
	httpserver.RegisterAttachmentAPI(mux, a.db, a.attachments)

	// Register metrics endpoint.
	if a.metrics != nil {
		if a.cfg.Metrics.Addr != "" {
			// Separate metrics server.
			metricsMux := http.NewServeMux()
			metricsMux.Handle("GET /metrics", promhttp.Handler())
			go func() {
				slog.Info("metrics server started", "addr", a.cfg.Metrics.Addr)
				if err := http.ListenAndServe(a.cfg.Metrics.Addr, metricsMux); err != nil {
					slog.Error("metrics server error", "err", err)
				}
			}()
		} else {
			mux.Handle("GET /metrics", promhttp.Handler())
		}
	}

	// Wire RPC handler so Centrifugo RPC calls route to our HTTP handlers.
	a.hub.SetRPCHandler(ws.NewMuxRPCHandler(mux))

	// Provide project list for server-side WebSocket subscriptions.
	a.hub.SetProjectProvider(func() []string {
		rows, err := a.db.Query(`SELECT key FROM projects`)
		if err != nil {
			return []string{"default"}
		}
		defer rows.Close()
		var keys []string
		for rows.Next() {
			var key string
			rows.Scan(&key)
			keys = append(keys, key)
		}
		if len(keys) == 0 {
			keys = []string{"default"}
		}
		return keys
	})

	// Set metrics on hub.
	if a.metrics != nil {
		a.hub.SetMetrics(a.metrics)
	}

	// Register WebSocket endpoints (Centrifugo-compatible + simple).
	mux.HandleFunc("GET /connection/websocket", a.hub.HandleUpgrade)
	mux.HandleFunc("GET /ws", a.hub.HandleUpgrade)

	// Prepare embedded frontend filesystem.
	frontendFS, err := fs.Sub(frontend.Dist, "dist")
	if err != nil {
		slog.Error("failed to load frontend", "err", err)
		os.Exit(1)
	}

	// Register ingestion pipeline + frontend fallback as catch-all.
	pipeline := httpserver.NewIngestionPipeline(a.registry.Handlers(), eventService, frontendFS)
	mux.Handle("/", pipeline)

	// Start TCP servers.
	tcpManager := tcp.NewManager(a.registry.TCPServers(), a.metrics)
	if err := tcpManager.Start(ctx); err != nil {
		slog.Error("failed to start TCP servers", "err", err)
		os.Exit(1)
	}

	// Start WebSocket hub.
	go a.hub.Run(ctx)

	// Start HTTP server (with optional metrics middleware).
	var handler http.Handler = mux
	if a.metrics != nil {
		handler = metrics.HTTPMiddleware(mux, a.metrics)
	}

	srv := &http.Server{Addr: a.cfg.HTTPAddr, Handler: handler}
	go func() {
		slog.Info("HTTP server started", "addr", a.cfg.HTTPAddr)
		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			slog.Error("HTTP server error", "err", err)
		}
	}()

	<-ctx.Done()
	slog.Info("shutting down...")
	_ = srv.Shutdown(context.Background())
	tcpManager.Wait()
}
