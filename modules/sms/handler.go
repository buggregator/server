package sms

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/buggregator/go-buggregator/internal/event"
)

type handler struct{}

func (h *handler) Priority() int { return 35 }

func (h *handler) Match(r *http.Request) bool {
	if r.Method != http.MethodPost {
		return false
	}
	if r.Header.Get("X-Buggregator-Detected-Type") == "sms" {
		return true
	}
	return strings.HasPrefix(r.URL.Path, "/sms")
}

func (h *handler) Handle(r *http.Request) (*event.Incoming, error) {
	body, err := parseBody(r)
	if err != nil {
		return nil, err
	}

	// Parse URL segments: /sms, /sms/{gateway}, /sms/{gateway}/{project}
	segments := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	segment1 := "" // gateway or project
	segment2 := "" // project (if gateway in segment1)
	if len(segments) >= 2 {
		segment1 = segments[1]
	}
	if len(segments) >= 3 {
		segment2 = segments[2]
	}

	// Determine project.
	project := ""

	// Try explicit gateway mode: /sms/{gateway}
	if segment1 != "" {
		if gw := findByName(segment1); gw != nil {
			// Explicit gateway mode — validate and always store.
			project = segment2
			return handleExplicitGateway(gw, body, project)
		}
		// segment1 is not a known gateway — treat as project.
		project = segment1
	}

	// Auto-detect mode: /sms or /sms/{project}
	return handleAutoDetect(body, project)
}

// handleExplicitGateway validates and always stores (even with warnings).
func handleExplicitGateway(gw *gateway, body map[string]any, project string) (*event.Incoming, error) {
	warnings := gw.validate(body)
	msg := gw.parse(body)

	if len(warnings) > 0 {
		msg.Warnings = warnings
	}

	payload, _ := json.Marshal(msg)
	return &event.Incoming{
		Type:    "sms",
		Payload: json.RawMessage(payload),
		Project: project,
	}, nil
}

// handleAutoDetect finds matching gateway and only stores if to+message are non-empty.
func handleAutoDetect(body map[string]any, project string) (*event.Incoming, error) {
	gw := detectGatewayFromBody(body)
	if gw == nil {
		return nil, nil // no match, skip
	}

	msg := gw.parse(body)

	// Reject if essential fields are empty.
	if msg.To == "" && msg.Message == "" {
		return nil, nil
	}

	payload, _ := json.Marshal(msg)
	return &event.Incoming{
		Type:    "sms",
		Payload: json.RawMessage(payload),
		Project: project,
	}, nil
}

// SmsMessage is the stored payload matching PHP SmsMessage.
type SmsMessage struct {
	From     string   `json:"from"`
	To       string   `json:"to"`
	Message  string   `json:"message"`
	Gateway  string   `json:"gateway"`
	Warnings []string `json:"warnings,omitempty"`
}

// --- Body parsing helpers ---

func parseBody(r *http.Request) (map[string]any, error) {
	ct := r.Header.Get("Content-Type")

	if strings.Contains(ct, "application/json") {
		data, err := io.ReadAll(r.Body)
		if err != nil {
			return nil, err
		}
		defer r.Body.Close()

		var fields map[string]any
		if err := json.Unmarshal(data, &fields); err != nil {
			return nil, err
		}
		return fields, nil
	}

	// Form-encoded.
	if err := r.ParseForm(); err != nil {
		return nil, err
	}
	fields := make(map[string]any)
	for k, v := range r.Form {
		if len(v) == 1 {
			fields[k] = v[0]
		} else {
			fields[k] = v
		}
	}
	return fields, nil
}

// extractFirst returns the first non-empty string value from body matching candidate keys.
func extractFirst(body map[string]any, keys []string) string {
	for _, k := range keys {
		if v, ok := body[k]; ok {
			switch val := v.(type) {
			case string:
				if val != "" {
					return val
				}
			case float64:
				return fmt.Sprintf("%v", val)
			}
		}
	}
	return ""
}

func joinFields(fields []string) string {
	return strings.Join(fields, "|")
}
