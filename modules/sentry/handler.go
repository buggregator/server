package sentry

import (
	"bytes"
	"compress/gzip"
	"compress/zlib"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/buggregator/go-buggregator/internal/event"
)

type handler struct{}

func (h *handler) Priority() int { return 10 }

func (h *handler) Match(r *http.Request) bool {
	if r.Method != http.MethodPost {
		return false
	}
	if r.Header.Get("X-Buggregator-Detected-Type") == "sentry" {
		return true
	}
	if r.Header.Get("X-Sentry-Auth") != "" {
		return true
	}
	path := r.URL.Path
	return strings.HasSuffix(path, "/store") || strings.HasSuffix(path, "/envelope")
}

func (h *handler) Handle(r *http.Request) (*event.Incoming, error) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	defer r.Body.Close()

	// Sentry SDKs often send gzip or deflate compressed payloads.
	body = decompress(body, r.Header.Get("Content-Encoding"))

	// Extract project from path: /api/{project}/store
	project := extractProject(r.URL.Path)

	// Handle Sentry envelope format (multiline: header\nitem_header\npayload).
	if strings.HasSuffix(r.URL.Path, "/envelope") {
		return handleEnvelope(body, project)
	}

	// Standard JSON store format.
	var parsed map[string]any
	uuid := ""
	if json.Unmarshal(body, &parsed) == nil {
		if id, ok := parsed["event_id"].(string); ok {
			uuid = id
		}
	} else {
		// If still not valid JSON, wrap it as a raw string.
		wrapped, _ := json.Marshal(map[string]any{"raw": string(body)})
		body = wrapped
	}

	return &event.Incoming{
		UUID:    uuid,
		Type:    "sentry",
		Payload: json.RawMessage(body),
		Project: project,
	}, nil
}

// decompress handles gzip and deflate Content-Encoding.
// Also auto-detects gzip/zlib by magic bytes if header is missing.
func decompress(data []byte, encoding string) []byte {
	if len(data) == 0 {
		return data
	}

	// Try by Content-Encoding header first.
	switch strings.ToLower(encoding) {
	case "gzip":
		if d, err := decompressGzip(data); err == nil {
			return d
		}
	case "deflate":
		if d, err := decompressZlib(data); err == nil {
			return d
		}
	}

	// Auto-detect by magic bytes (Sentry SDKs sometimes omit the header).
	if len(data) >= 2 {
		// Gzip magic: 0x1f 0x8b
		if data[0] == 0x1f && data[1] == 0x8b {
			if d, err := decompressGzip(data); err == nil {
				return d
			}
		}
		// Zlib magic: 0x78 (0x01, 0x5e, 0x9c, 0xda)
		if data[0] == 0x78 {
			if d, err := decompressZlib(data); err == nil {
				return d
			}
		}
	}

	return data
}

func decompressGzip(data []byte) ([]byte, error) {
	r, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer r.Close()
	return io.ReadAll(r)
}

func decompressZlib(data []byte) ([]byte, error) {
	r, err := zlib.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer r.Close()
	return io.ReadAll(r)
}

// handleEnvelope parses Sentry envelope format.
// Format: envelope_header\nitem_header\npayload[\nitem_header\npayload...]
func handleEnvelope(body []byte, project string) (*event.Incoming, error) {
	lines := strings.SplitN(string(body), "\n", 3)

	// Parse envelope header for event_id.
	uuid := ""
	if len(lines) >= 1 {
		var header map[string]any
		if json.Unmarshal([]byte(lines[0]), &header) == nil {
			if id, ok := header["event_id"].(string); ok {
				uuid = id
			}
		}
	}

	// Try to extract the main event payload (third line onwards).
	var payload json.RawMessage
	if len(lines) >= 3 {
		// Try parsing the payload as JSON.
		trimmed := strings.TrimSpace(lines[2])
		if json.Valid([]byte(trimmed)) {
			payload = json.RawMessage(trimmed)
		}
	}

	// Fallback: store the whole envelope as JSON.
	if payload == nil {
		wrapped, _ := json.Marshal(map[string]any{
			"envelope": string(body),
		})
		payload = wrapped
	}

	return &event.Incoming{
		UUID:    uuid,
		Type:    "sentry",
		Payload: payload,
		Project: project,
	}, nil
}

func extractProject(path string) string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) >= 2 {
		return parts[1]
	}
	return ""
}
