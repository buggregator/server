package profiler

import (
	"database/sql"
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
	return &handler{db: m.db}
}

func (m *Module) RegisterRoutes(mux *http.ServeMux, store event.Store) {
	registerAPI(mux, m.db, store)
}

func (m *Module) PreviewMapper() event.PreviewMapper {
	return &previewMapper{}
}
