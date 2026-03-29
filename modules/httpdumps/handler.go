package httpdumps

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/buggregator/go-buggregator/internal/event"
)

type handler struct{}

func (h *handler) Priority() int { return 9999 } // Catch-all, runs last.

func (h *handler) Match(r *http.Request) bool {
	// Matches everything that wasn't claimed by another module.
	return true
}

func (h *handler) Handle(r *http.Request) (*event.Incoming, error) {
	body, _ := io.ReadAll(r.Body)
	defer r.Body.Close()

	headers := make(map[string]string)
	for k, v := range r.Header {
		if len(v) > 0 {
			headers[k] = v[0]
		}
	}

	payload := map[string]any{
		"method":  r.Method,
		"uri":     r.RequestURI,
		"host":    r.Host,
		"headers": headers,
		"body":    string(body),
		"query":   r.URL.RawQuery,
	}

	b, _ := json.Marshal(payload)

	return &event.Incoming{
		Type:    "http-dumps",
		Payload: json.RawMessage(b),
	}, nil
}
