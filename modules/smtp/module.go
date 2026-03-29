package smtp

import (
	"context"

	"github.com/buggregator/go-buggregator/internal/event"
	"github.com/buggregator/go-buggregator/internal/module"
	"github.com/buggregator/go-buggregator/internal/server/tcp"
)

type Module struct {
	module.BaseModule
	addr         string
	eventService EventStorer
}

type EventStorer interface {
	HandleIncoming(ctx context.Context, inc *event.Incoming) error
}

func New(addr string) *Module {
	return &Module{addr: addr}
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
			Starter: newSMTPServer(m.addr, m.eventService),
		},
	}
}

func (m *Module) PreviewMapper() event.PreviewMapper {
	return &previewMapper{}
}
