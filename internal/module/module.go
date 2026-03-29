package module

import (
	"database/sql"
	"net/http"

	"github.com/buggregator/go-buggregator/internal/event"
	"github.com/buggregator/go-buggregator/internal/server/tcp"
	"github.com/buggregator/go-buggregator/internal/storage"
)

// Module is the interface every event module must implement.
// Embed BaseModule to get no-op defaults for optional methods.
type Module interface {
	// Name returns a human-readable module name (e.g., "Sentry").
	Name() string

	// Type returns the event type string stored in the events table.
	Type() string

	// RegisterRoutes adds module-specific API routes to the router.
	RegisterRoutes(mux *http.ServeMux, store event.Store)

	// HTTPHandler returns a handler that inspects an incoming request
	// and either claims it or returns nil. Return nil if this module
	// has no HTTP ingestion (e.g., TCP-only modules).
	HTTPHandler() HTTPIngestionHandler

	// TCPServers returns TCP server configurations this module needs.
	TCPServers() []tcp.ServerConfig

	// PreviewMapper returns the mapper for converting payloads to previews.
	PreviewMapper() event.PreviewMapper

	// RegisterMigrations registers module-specific SQL migration files.
	RegisterMigrations(migrator *storage.Migrator) error

	// OnInit is called once at startup after migrations are applied.
	OnInit(db *sql.DB) error

	// OnEventStored is called after any event is persisted.
	OnEventStored(ev event.Event)
}

// HTTPIngestionHandler decides if an HTTP request belongs to a module.
type HTTPIngestionHandler interface {
	// Priority determines ordering. Lower runs first. HttpDumps = 9999.
	Priority() int

	// Match returns true if this module should handle the request.
	Match(r *http.Request) bool

	// Handle processes the request and returns the event to store.
	Handle(r *http.Request) (*event.Incoming, error)
}

// BaseModule provides no-op defaults for optional Module methods.
type BaseModule struct{}

func (b BaseModule) RegisterRoutes(_ *http.ServeMux, _ event.Store) {}
func (b BaseModule) HTTPHandler() HTTPIngestionHandler              { return nil }
func (b BaseModule) TCPServers() []tcp.ServerConfig                 { return nil }
func (b BaseModule) PreviewMapper() event.PreviewMapper             { return nil }
func (b BaseModule) RegisterMigrations(_ *storage.Migrator) error   { return nil }
func (b BaseModule) OnInit(_ *sql.DB) error                         { return nil }
func (b BaseModule) OnEventStored(_ event.Event)                    {}
