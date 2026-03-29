package ray

import (
	"net/http"

	"github.com/buggregator/go-buggregator/internal/event"
	"github.com/buggregator/go-buggregator/internal/module"
)

type Module struct {
	module.BaseModule
}

func New() *Module { return &Module{} }

func (m *Module) Name() string { return "Ray" }
func (m *Module) Type() string { return "ray" }

func (m *Module) HTTPHandler() module.HTTPIngestionHandler {
	return &handler{}
}

func (m *Module) RegisterRoutes(mux *http.ServeMux, store event.Store) {
	// Ray PHP client checks this endpoint to detect if server is available.
	// Return 400 = "I exist but this isn't a real endpoint" — Ray treats this as "available".
	mux.HandleFunc("GET /_availability_check", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	})
}

func (m *Module) PreviewMapper() event.PreviewMapper {
	return &previewMapper{}
}
