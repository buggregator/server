package profiler

import (
	"database/sql"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/buggregator/go-buggregator/internal/event"
	"github.com/buggregator/go-buggregator/internal/module"
	"github.com/buggregator/go-buggregator/internal/storage"
)

type Module struct {
	module.BaseModule
	db *sql.DB
}

func New() *Module { return &Module{} }

func (m *Module) Name() string { return "Profiler" }
func (m *Module) Type() string { return "profiler" }

func (m *Module) RegisterMigrations(migrator *storage.Migrator) error {
	return migrator.AddFromFS("profiler", migrations, "migrations")
}

func (m *Module) OnInit(db *sql.DB) error {
	m.db = db
	return nil
}

func (m *Module) HTTPHandler() module.HTTPIngestionHandler {
	return &handler{}
}

func (m *Module) RegisterRoutes(mux *http.ServeMux, store event.Store) {
	registerAPI(mux, m.db, store)
}

func (m *Module) PreviewMapper() event.PreviewMapper {
	return &previewMapper{}
}

func (m *Module) OnEventStored(ev event.Event) {
	if ev.Type != "profiler" || m.db == nil {
		return
	}

	var payload struct {
		ProfileUUID string          `json:"profile_uuid"`
		AppName     string          `json:"app_name"`
		Peaks       Metrics         `json:"peaks"`
		Edges       map[string]Edge `json:"edges"`
	}
	if err := json.Unmarshal(ev.Payload, &payload); err != nil {
		slog.Warn("profiler: failed to parse event payload", "err", err)
		return
	}

	if err := storeProfile(m.db, payload.ProfileUUID, payload.AppName, payload.Peaks, payload.Edges); err != nil {
		slog.Warn("profiler: failed to store profile", "err", err)
	}
}
