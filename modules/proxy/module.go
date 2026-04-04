package proxy

import (
	"context"

	"github.com/buggregator/go-buggregator/internal/event"
	"github.com/buggregator/go-buggregator/internal/module"
	"github.com/buggregator/go-buggregator/internal/server/tcp"
)

// Module implements an HTTP forward proxy that captures request/response pairs
// and stores them as "http-dump" events with an additional response and proxy flag.
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

func (m *Module) Name() string { return "HTTP Proxy" }
func (m *Module) Type() string { return "http-dump" }

func (m *Module) TCPServers() []tcp.ServerConfig {
	if m.eventService == nil {
		return nil
	}
	return []tcp.ServerConfig{
		{
			Name:    "http-proxy",
			Address: m.addr,
			Starter: newProxyServer(m.addr, m.eventService),
		},
	}
}
