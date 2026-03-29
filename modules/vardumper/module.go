package vardumper

import (
	"context"
	"log/slog"

	"github.com/buggregator/go-buggregator/internal/event"
	"github.com/buggregator/go-buggregator/internal/module"
	"github.com/buggregator/go-buggregator/internal/server/tcp"
)

type Module struct {
	module.BaseModule
	addr         string
	php          *PHPProcess
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

func (m *Module) Name() string { return "VarDumper" }
func (m *Module) Type() string { return "var-dump" }

func (m *Module) TCPServers() []tcp.ServerConfig {
	if m.eventService == nil || m.php == nil {
		return nil
	}
	return []tcp.ServerConfig{
		{
			Name:    "var-dumper",
			Address: m.addr,
			Handler: &tcpHandler{php: m.php, eventService: m.eventService},
		},
	}
}

func (m *Module) PreviewMapper() event.PreviewMapper {
	return &previewMapper{}
}

// StartPHP starts the embedded PHP parser process. Call before Init.
func (m *Module) StartPHP() error {
	php, err := StartPHPProcess()
	if err != nil {
		slog.Error("failed to start PHP VarDumper parser", "err", err)
		return err
	}
	m.php = php
	return nil
}

// StopPHP stops the embedded PHP parser process.
func (m *Module) StopPHP() {
	if m.php != nil {
		m.php.Stop()
	}
}
