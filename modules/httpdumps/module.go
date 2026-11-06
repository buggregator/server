package httpdumps

import (
	"context"
	"database/sql"
	"log/slog"
	"net/http"

	"github.com/buggregator/go-buggregator/internal/event"
	"github.com/buggregator/go-buggregator/internal/module"
	"github.com/buggregator/go-buggregator/internal/storage"
)

type Module struct {
	module.BaseModule
	attachments *storage.AttachmentStore
	db          *sql.DB
	mocks       *mockStore
	events      EventStorer
}

func New(attachments *storage.AttachmentStore, db *sql.DB) *Module {
	return &Module{attachments: attachments, db: db, mocks: newMockStore(db)}
}

// SetEventService gives the mock server the service that stores captured requests.
func (m *Module) SetEventService(es EventStorer) { m.events = es }

// SetMockRules sets the read-only mock rules from buggregator.yaml. Invalid
// rules are logged and skipped.
func (m *Module) SetMockRules(rules []MockRule) {
	valid := make([]MockRule, 0, len(rules))
	for i, r := range rules {
		r.Enabled = true
		if err := r.Normalize(); err != nil {
			slog.Warn("mock: skipping config rule", "index", i, "path", r.Path, "err", err)
			continue
		}
		valid = append(valid, r)
	}
	m.mocks.setConfig(valid)
}

func (m *Module) RegisterMigrations(migrator *storage.Migrator) error {
	return migrator.AddFromFS("http-dump", migrations, "migrations")
}

func (m *Module) OnInit(db *sql.DB) error {
	return m.mocks.reload()
}

// RegisterRoutes mounts the mock server under /mock/. It is public: the mocked
// API's clients call it. The route is more specific than the ingestion
// pipeline at "/", so no rule can shadow SDK ingestion, /api or the UI.
func (m *Module) RegisterRoutes(mux *http.ServeMux, _ event.Store) {
	srv := &mockServer{rules: m.mocks, capture: &handler{attachments: m.attachments, db: m.db}, events: m}
	mux.Handle(MockPrefix+"/", srv)
}

// HandleIncoming forwards to the event service set later by main (routes are
// registered before it exists).
func (m *Module) HandleIncoming(ctx context.Context, inc *event.Incoming) error {
	if m.events == nil {
		return nil
	}
	return m.events.HandleIncoming(ctx, inc)
}

// RegisterProtectedRoutes adds the mock rule API behind the API auth.
func (m *Module) RegisterProtectedRoutes(mux *http.ServeMux, protect func(http.Handler) http.Handler) {
	registerMockAPI(mux, m.mocks, m.db, protect)
}

func (m *Module) Name() string { return "HTTP Dumps" }
func (m *Module) Type() string { return "http-dump" }

func (m *Module) HTTPHandler() module.HTTPIngestionHandler {
	return &handler{attachments: m.attachments, db: m.db}
}

func (m *Module) PreviewMapper() event.PreviewMapper {
	return &previewMapper{}
}
