package sentry

import (
	"database/sql"
	"net/http"

	"github.com/buggregator/server/v2/internal/event"
	"github.com/buggregator/server/v2/internal/module"
	"github.com/buggregator/server/v2/internal/storage"
)

type Module struct {
	module.BaseModule
	db          *sql.DB
	broadcaster module.Broadcaster
}

// StoredEvent is the WebSocket notice that structured rows were stored: logs,
// metrics or traces, which create no feed card and so no "event.received".
// Its payload names the project and the kinds - "log", "metric", "trace" -,
// never the rows, so the Sentry tabs know which list to fetch again.
const StoredEvent = "sentry.stored"

// SetBroadcaster implements module.Broadcasting.
func (m *Module) SetBroadcaster(b module.Broadcaster) { m.broadcaster = b }

// notifyStored sends StoredEvent on the project's channel, the one its event
// cards are broadcast on.
func (m *Module) notifyStored(project string, kinds []string) {
	if m.broadcaster == nil || len(kinds) == 0 {
		return
	}
	project = projectKey(project)
	m.broadcaster.Broadcast("events.project."+project, StoredEvent, map[string]any{
		"project": project,
		"kinds":   kinds,
	})
}

func New() *Module { return &Module{} }

func (m *Module) Name() string { return "Sentry" }
func (m *Module) Type() string { return "sentry" }

func (m *Module) RegisterMigrations(migrator *storage.Migrator) error {
	return migrator.AddFromFS("sentry", migrations, "migrations")
}

func (m *Module) OnInit(db *sql.DB) error {
	m.db = db
	return nil
}

// RegisterProtectedRoutes adds the Sentry UI API behind the API auth. SDK
// ingestion (envelope/store) goes through the HTTP handler and stays open.
func (m *Module) RegisterProtectedRoutes(mux *http.ServeMux, protect func(http.Handler) http.Handler) {
	if m.db != nil {
		registerAPI(mux, m.db, protect)
	}
}

func (m *Module) HTTPHandler() module.HTTPIngestionHandler {
	return &handler{db: m.db, notifyStored: m.notifyStored}
}

func (m *Module) PreviewMapper() event.PreviewMapper {
	return &previewMapper{}
}
