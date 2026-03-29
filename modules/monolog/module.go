package monolog

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

// EventStorer is the interface for storing events from TCP connections.
type EventStorer interface {
	HandleIncoming(ctx context.Context, inc *event.Incoming) error
}

func New(addr string) *Module {
	return &Module{addr: addr}
}

// SetEventService injects the event service after construction.
func (m *Module) SetEventService(es EventStorer) {
	m.eventService = es
}

func (m *Module) Name() string { return "Monolog" }
func (m *Module) Type() string { return "monolog" }

func (m *Module) TCPServers() []tcp.ServerConfig {
	if m.eventService == nil {
		return nil
	}
	return []tcp.ServerConfig{
		{
			Name:    "monolog",
			Address: m.addr,
			Handler: &tcpHandler{eventService: m.eventService},
		},
	}
}

func (m *Module) PreviewMapper() event.PreviewMapper {
	return &previewMapper{}
}
