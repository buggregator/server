package module

import (
	"database/sql"
	"log/slog"
	"net/http"
	"sort"

	"github.com/buggregator/go-buggregator/internal/event"
	"github.com/buggregator/go-buggregator/internal/server/tcp"
	"github.com/buggregator/go-buggregator/internal/storage"
)

// Registry collects modules and provides access to their handlers.
type Registry struct {
	modules  []Module
	handlers []HTTPIngestionHandler
	previews *event.PreviewRegistry
	tcpCfgs  []tcp.ServerConfig
}

func NewRegistry() *Registry {
	return &Registry{
		previews: event.NewPreviewRegistry(),
	}
}

// Register adds a module to the registry.
func (r *Registry) Register(m Module) {
	r.modules = append(r.modules, m)
}

// Init initializes all modules: runs migrations, routes, handlers.
func (r *Registry) Init(db *sql.DB, mux *http.ServeMux, store event.Store) error {
	// Phase 1: Collect all migrations.
	migrator := storage.NewMigrator(db)

	// Register core migrations.
	if err := migrator.AddFromFS("core", storage.CoreMigrations, "migrations"); err != nil {
		return err
	}

	// Register module migrations.
	for _, m := range r.modules {
		if err := m.RegisterMigrations(migrator); err != nil {
			return err
		}
	}

	// Run all migrations sorted by filename.
	if err := migrator.Run(); err != nil {
		return err
	}

	// Phase 2: Initialize modules.
	for _, m := range r.modules {
		slog.Info("initializing module", "name", m.Name(), "type", m.Type())

		if err := m.OnInit(db); err != nil {
			return err
		}

		m.RegisterRoutes(mux, store)

		if h := m.HTTPHandler(); h != nil {
			r.handlers = append(r.handlers, h)
		}

		if cfgs := m.TCPServers(); cfgs != nil {
			r.tcpCfgs = append(r.tcpCfgs, cfgs...)
		}

		if pm := m.PreviewMapper(); pm != nil {
			r.previews.Register(m.Type(), pm)
		}
	}

	// Sort handlers by priority (lower = first).
	sort.Slice(r.handlers, func(i, j int) bool {
		return r.handlers[i].Priority() < r.handlers[j].Priority()
	})

	return nil
}

// Handlers returns HTTP ingestion handlers sorted by priority.
func (r *Registry) Handlers() []HTTPIngestionHandler {
	return r.handlers
}

// TCPServers returns all TCP server configs from all modules.
func (r *Registry) TCPServers() []tcp.ServerConfig {
	return r.tcpCfgs
}

// Previews returns the preview registry.
func (r *Registry) Previews() *event.PreviewRegistry {
	return r.previews
}

// NotifyEventStored calls OnEventStored on all modules.
func (r *Registry) NotifyEventStored(ev event.Event) {
	for _, m := range r.modules {
		m.OnEventStored(ev)
	}
}
