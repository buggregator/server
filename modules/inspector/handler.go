package inspector

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"

	"github.com/buggregator/go-buggregator/internal/event"
)

type handler struct{}

func (h *handler) Priority() int { return 30 }

func (h *handler) Match(r *http.Request) bool {
	if r.Method != http.MethodPost {
		return false
	}
	return r.Header.Get("X-Inspector-Key") != "" || r.Header.Get("X-Inspector-Version") != ""
}

func (h *handler) Handle(r *http.Request) (*event.Incoming, error) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	defer r.Body.Close()

	// Inspector sends base64-encoded JSON array.
	decoded, err := base64.StdEncoding.DecodeString(string(body))
	if err != nil {
		// Try raw JSON.
		decoded = body
	}

	// Validate it's valid JSON.
	var payload json.RawMessage
	if err := json.Unmarshal(decoded, &payload); err != nil {
		return nil, err
	}

	return &event.Incoming{
		Type:    "inspector",
		Payload: payload,
	}, nil
}
