package httpdumps

import (
	"github.com/buggregator/go-buggregator/internal/event"
	"github.com/buggregator/go-buggregator/internal/module"
)

type Module struct {
	module.BaseModule
}

func New() *Module { return &Module{} }

func (m *Module) Name() string { return "HTTP Dumps" }
func (m *Module) Type() string { return "http-dump" }

func (m *Module) HTTPHandler() module.HTTPIngestionHandler {
	return &handler{}
}

func (m *Module) PreviewMapper() event.PreviewMapper {
	return &previewMapper{}
}
