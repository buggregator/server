package smtp

import (
	"context"
	"database/sql"

	"github.com/buggregator/go-buggregator/internal/event"
	"github.com/buggregator/go-buggregator/internal/module"
	"github.com/buggregator/go-buggregator/internal/server/tcp"
	"github.com/buggregator/go-buggregator/internal/storage"
)

type Module struct {
	module.BaseModule
	addr         string
	eventService EventStorer
	attachments  *storage.AttachmentStore
	db           *sql.DB
}

type EventStorer interface {
	HandleIncoming(ctx context.Context, inc *event.Incoming) error
}

func New(addr string, attachments *storage.AttachmentStore, db *sql.DB) *Module {
	return &Module{addr: addr, attachments: attachments, db: db}
}

func (m *Module) SetEventService(es EventStorer) {
	m.eventService = es
}

func (m *Module) Name() string { return "SMTP" }
func (m *Module) Type() string { return "smtp" }

func (m *Module) TCPServers() []tcp.ServerConfig {
	if m.eventService == nil {
		return nil
	}
	return []tcp.ServerConfig{
		{
			Name:    "smtp",
			Address: m.addr,
			Starter: newSMTPServer(m.addr, m.eventService, m.attachments, m.db),
		},
	}
}

func (m *Module) PreviewMapper() event.PreviewMapper {
	return &previewMapper{}
}
