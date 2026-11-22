package ray

import (
	"context"
	"net/http"

	"github.com/buggregator/server/v2/internal/event"
	"github.com/buggregator/server/v2/internal/module"
)

type Module struct {
	module.BaseModule
	locks  *locks
	events EventStorer
	store  event.Store
}

// EventStorer stores and broadcasts events (the HTTP event service).
type EventStorer interface {
	HandleIncoming(ctx context.Context, inc *event.Incoming) error
}

func New() *Module { return &Module{locks: newLocks()} }

// SetEventService lets the module save a pause's state into its event.
func (m *Module) SetEventService(es EventStorer) { m.events = es }

func (m *Module) Name() string { return "Ray" }
func (m *Module) Type() string { return "ray" }

func (m *Module) HTTPHandler() module.HTTPIngestionHandler {
	return &handler{locks: m.locks}
}

func (m *Module) RegisterRoutes(mux *http.ServeMux, store event.Store) {
	m.store = store
	// Ray PHP client checks this endpoint to detect if server is available.
	// Return 400 = "I exist but this isn't a real endpoint" — Ray treats this as "available".
	mux.HandleFunc("GET /_availability_check", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	})
	// ray()->pause(): the PHP client polls this every second while paused.
	// Public like the availability check: the client sends no credentials.
	mux.HandleFunc("GET /locks/{name}", m.handleLockStatus)
}

// RegisterProtectedRoutes adds the UI's Continue / Stop execution, called
// over the WebSocket RPC as post:api/ray/locks/{name}.
func (m *Module) RegisterProtectedRoutes(mux *http.ServeMux, protect func(http.Handler) http.Handler) {
	mux.Handle("POST /api/ray/locks/{name}", protect(http.HandlerFunc(m.handleLockRelease)))
}

func (m *Module) PreviewMapper() event.PreviewMapper {
	return &previewMapper{}
}
