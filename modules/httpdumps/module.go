package httpdumps

import (
	"database/sql"

	"github.com/buggregator/go-buggregator/internal/event"
	"github.com/buggregator/go-buggregator/internal/module"
	"github.com/buggregator/go-buggregator/internal/storage"
)

type Module struct {
	module.BaseModule
	attachments *storage.AttachmentStore
	db          *sql.DB
}

func New(attachments *storage.AttachmentStore, db *sql.DB) *Module {
	return &Module{attachments: attachments, db: db}
}

func (m *Module) Name() string { return "HTTP Dumps" }
func (m *Module) Type() string { return "http-dump" }

func (m *Module) HTTPHandler() module.HTTPIngestionHandler {
	return &handler{attachments: m.attachments, db: m.db}
}

func (m *Module) PreviewMapper() event.PreviewMapper {
	return &previewMapper{}
}
