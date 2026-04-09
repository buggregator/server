package profiler

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/buggregator/go-buggregator/internal/event"
)

type handler struct {
}

func (h *handler) Priority() int { return 40 }

func (h *handler) Match(r *http.Request) bool {
	if r.Method != http.MethodPost {
		return false
	}
	return r.Header.Get("X-Buggregator-Detected-Type") == "profiler" ||
		r.Header.Get("X-Profiler-Dump") != "" ||
		strings.HasSuffix(r.URL.Path, "/profiler/store")
}

func (h *handler) Handle(r *http.Request) (*event.Incoming, error) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	defer r.Body.Close()

	var incoming IncomingProfile
	if err := json.Unmarshal(body, &incoming); err != nil {
		return nil, err
	}

	// Process the profile (compute diffs, percentages, edges).
	peaks, edges := Process(&incoming)

	uuid := event.GenerateUUID()

	// Build event payload matching PHP Buggregator format.
	// peaks and edges are embedded so OnEventStored can store them
	// without re-parsing the original payload.
	payload := map[string]any{
		"profile_uuid": uuid,
		"app_name":     incoming.AppName,
		"hostname":     incoming.Hostname,
		"date":         incoming.Date,
		"tags":         incoming.Tags,
		"peaks":        peaks,
		"edges":        edges,
		"total_edges":  len(edges),
	}
	b, _ := json.Marshal(payload)

	return &event.Incoming{
		UUID:    uuid,
		Type:    "profiler",
		Payload: json.RawMessage(b),
	}, nil
}
