package ray

import (
	"encoding/json"
	"fmt"
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
	if r.Header.Get("X-Buggregator-Detected-Type") == "ray" {
		return true
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
		Merge:   appendPayloads(body),
	}, nil
}

// A stored Ray event takes at most maxMergedPayloads payloads and
// maxMergedBytes: every follow-up rewrites and re-broadcasts the whole event.
const (
	maxMergedPayloads = 1000
	maxMergedBytes    = 8 << 20
)

// appendPayloads merges a follow-up request (same uuid: trace, color, size…)
// into the stored event: its payloads go after the stored ones, the rest of
// the stored event stays.
func appendPayloads(body []byte) func(json.RawMessage) (json.RawMessage, error) {
	return func(stored json.RawMessage) (json.RawMessage, error) {
		var next struct {
			Payloads []json.RawMessage `json:"payloads"`
		}
		if err := json.Unmarshal(body, &next); err != nil || len(next.Payloads) == 0 {
			return stored, nil
		}
		var doc map[string]json.RawMessage
		if err := json.Unmarshal(stored, &doc); err != nil {
			return nil, err
		}
		var payloads []json.RawMessage
		if raw, ok := doc["payloads"]; ok {
			if err := json.Unmarshal(raw, &payloads); err != nil {
				return nil, err
			}
		}
		if len(payloads)+len(next.Payloads) > maxMergedPayloads || len(stored)+len(body) > maxMergedBytes {
			return nil, fmt.Errorf("%w: at most %d payloads, %d MB", event.ErrMergeLimit, maxMergedPayloads, maxMergedBytes>>20)
		}
		merged, err := json.Marshal(append(payloads, next.Payloads...))
		if err != nil {
			return nil, err
		}
		doc["payloads"] = merged
		return json.Marshal(doc)
	}
}
