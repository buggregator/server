package http

import (
	"io/fs"
	"log/slog"
	"net/http"
	"strings"

	"github.com/buggregator/go-buggregator/internal/module"
)

// IngestionPipeline tries each registered handler in priority order.
// Requests that aren't claimed by any module are served as static frontend files.
type IngestionPipeline struct {
	handlers     []module.HTTPIngestionHandler
	eventService *EventService
	frontend     http.Handler
}

func NewIngestionPipeline(handlers []module.HTTPIngestionHandler, es *EventService, frontendFS fs.FS) *IngestionPipeline {
	// Create file server for embedded frontend.
	fileServer := http.FileServer(http.FS(frontendFS))

	return &IngestionPipeline{
		handlers:     handlers,
		eventService: es,
		frontend:     fileServer,
	}
}

func (p *IngestionPipeline) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Only try ingestion handlers for POST/PUT requests.
	// GET requests go straight to frontend (unless matched by API routes).
	if r.Method == http.MethodPost || r.Method == http.MethodPut {
		for _, h := range p.handlers {
			if !h.Match(r) {
				continue
			}

			incoming, err := h.Handle(r)
			if err != nil {
				slog.Error("ingestion handler error", "err", err)
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			if incoming == nil {
				continue
			}

			if err := p.eventService.HandleIncoming(r.Context(), incoming); err != nil {
				slog.Error("failed to store event", "err", err)
				http.Error(w, "internal error", http.StatusInternalServerError)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status":true}`))
			return
		}
	}

	// Serve frontend static files.
	// For SPA: if the file doesn't exist, serve index.html.
	path := r.URL.Path
	if path != "/" && !strings.Contains(path, ".") {
		// SPA route — serve index.html for paths without extensions.
		r.URL.Path = "/"
	}
	p.frontend.ServeHTTP(w, r)
}
