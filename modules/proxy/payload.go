package proxy

import (
	"encoding/json"
	"time"

	"github.com/buggregator/go-buggregator/internal/event"
)

const maxBodySize = 512 * 1024 // 512KB

// buildHTTPDumpEvent creates an http-dump event from a proxied request/response pair.
// The payload matches the http-dump format with additional "response" and "proxy" fields.
func buildHTTPDumpEvent(req *capturedRequest, resp *capturedResponse, durationMs float64, proxyErr string) *event.Incoming {
	headers := make(map[string][]string)
	for k, v := range req.Headers {
		headers[k] = v
	}

	query := make(map[string]any)
	for k, v := range req.Query {
		if len(v) == 1 {
			query[k] = v[0]
		} else {
			query[k] = v
		}
	}

	payload := map[string]any{
		"received_at": time.Now().Format("2006-01-02 15:04:05"),
		"host":        req.Host,
		"proxy":       true,
		"duration_ms": durationMs,
		"request": map[string]any{
			"method":  req.Method,
			"uri":     req.URI,
			"headers": headers,
			"body":    truncateBody(req.Body),
			"query":   query,
			"post":    map[string]any{},
			"cookies": map[string]string{},
			"files":   []any{},
		},
	}

	if proxyErr != "" {
		payload["error"] = proxyErr
	}

	if resp != nil {
		respHeaders := make(map[string][]string)
		for k, v := range resp.Headers {
			respHeaders[k] = v
		}
		payload["response"] = map[string]any{
			"status_code": resp.StatusCode,
			"headers":     respHeaders,
			"body":        truncateBody(resp.Body),
		}
	}

	b, _ := json.Marshal(payload)

	return &event.Incoming{
		UUID:    event.GenerateUUID(),
		Type:    "http-dump",
		Payload: json.RawMessage(b),
	}
}

type capturedRequest struct {
	Method  string
	URI     string
	Host    string
	Scheme  string
	Headers map[string][]string
	Query   map[string][]string
	Body    []byte
}

type capturedResponse struct {
	StatusCode int
	Headers    map[string][]string
	Body       []byte
}

// truncateBody caps body at maxBodySize and marks it as truncated.
func truncateBody(body []byte) string {
	if len(body) <= maxBodySize {
		return string(body)
	}
	return string(body[:maxBodySize]) + "\n[truncated]"
}
