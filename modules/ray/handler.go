package ray

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/buggregator/go-buggregator/internal/event"
)

type handler struct{}

func (h *handler) Priority() int { return 20 }

func (h *handler) Match(r *http.Request) bool {
	if r.Method != http.MethodPost {
		return false
	}
	ua := strings.ToLower(r.Header.Get("User-Agent"))
	return strings.HasPrefix(ua, "ray")
}

func (h *handler) Handle(r *http.Request) (*event.Incoming, error) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	defer r.Body.Close()

	// Extract UUID from payload if present.
	var parsed map[string]any
	uuid := ""
	if json.Unmarshal(body, &parsed) == nil {
		if id, ok := parsed["uuid"].(string); ok {
			uuid = id
		}
	}

	return &event.Incoming{
		UUID:    uuid,
		Type:    "ray",
		Payload: json.RawMessage(body),
	}, nil
}
