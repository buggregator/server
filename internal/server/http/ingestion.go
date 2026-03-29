package http

import (
	"io/fs"
	"log/slog"
	"net/http"
	"strings"

	"github.com/buggregator/go-buggregator/internal/module"
)

const (
	// Internal headers set by the ingestion pipeline for handler matching.
	HeaderDetectedType    = "X-Buggregator-Detected-Type"
	HeaderDetectedProject = "X-Buggregator-Detected-Project"
)

// IngestionPipeline tries each registered handler in priority order.
// Requests that aren't claimed by any module are served as static frontend files.
type IngestionPipeline struct {
	handlers     []module.HTTPIngestionHandler
	eventService *EventService
	frontend     http.Handler
}

func NewIngestionPipeline(handlers []module.HTTPIngestionHandler, es *EventService, frontendFS fs.FS) *IngestionPipeline {
	fileServer := http.FileServer(http.FS(frontendFS))
	return &IngestionPipeline{
		handlers:     handlers,
		eventService: es,
		frontend:     fileServer,
	}
}

func (p *IngestionPipeline) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost || r.Method == http.MethodPut {
		// Detect event type from userinfo/headers/basic-auth.
		if detected := detectEventType(r); detected != nil {
			r.Header.Set(HeaderDetectedType, detected.Type)
			if detected.Project != "" {
				r.Header.Set(HeaderDetectedProject, detected.Project)
			}
		}

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

			// Override project from detection if handler didn't set it.
			if incoming.Project == "" {
				incoming.Project = r.Header.Get(HeaderDetectedProject)
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
	path := r.URL.Path
	if path != "/" && !strings.Contains(path, ".") {
		r.URL.Path = "/"
	}
	p.frontend.ServeHTTP(w, r)
}
