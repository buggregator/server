package sentry

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

func (m *Module) Name() string { return "Sentry" }
func (m *Module) Type() string { return "sentry" }

func (m *Module) RegisterMigrations(migrator *storage.Migrator) error {
	return migrator.AddFromFS("sentry", migrations, "migrations")
}

func (m *Module) OnInit(db *sql.DB) error {
	m.db = db
	return nil
}

func (m *Module) RegisterRoutes(mux *http.ServeMux, store event.Store) {
	if m.db != nil {
		registerAPI(mux, m.db)
	}
}

func (m *Module) HTTPHandler() module.HTTPIngestionHandler {
	return &handler{db: m.db}
}

func (m *Module) PreviewMapper() event.PreviewMapper {
	return &previewMapper{}
}
