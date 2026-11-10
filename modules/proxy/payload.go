package proxy

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/buggregator/server/v2/internal/event"
)

const maxBodySize = 512 * 1024 // 512KB

// buildHTTPDumpEvent creates an http-dump event from a proxied request/response pair.
// The payload matches the http-dump format with additional "response", "proxy"
// and "scheme" fields. "scheme" ("http" or "https") is the scheme the client
// used; dumps stored before it was added have none (read them as "http").
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
		"scheme":      req.Scheme,
		"proxy":       true,
		"duration_ms": durationMs,
		"request": map[string]any{
			"method":  req.Method,
			"uri":     req.URI,
			"headers": headers,
			"body":    truncateBody(decodedBody(req.Body, req.Headers)),
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
			"body":        truncateBody(decodedBody(resp.Body, resp.Headers)),
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
	Headers http.Header
	Query   map[string][]string
	Body    []byte
}

type capturedResponse struct {
	StatusCode int
	Headers    http.Header
	Body       []byte
}

// truncateBody caps body at maxBodySize and marks it as truncated.
func truncateBody(body []byte) string {
	if len(body) <= maxBodySize {
		return string(body)
	}
	return string(body[:maxBodySize]) + "\n[truncated]"
}
