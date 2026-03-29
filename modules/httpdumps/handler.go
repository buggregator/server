package httpdumps

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/buggregator/go-buggregator/internal/event"
)

type handler struct{}

func (h *handler) Priority() int { return 9999 } // Catch-all, runs last.

func (h *handler) Match(r *http.Request) bool {
	return true
}

func (h *handler) Handle(r *http.Request) (*event.Incoming, error) {
	body, _ := io.ReadAll(r.Body)
	defer r.Body.Close()

	// Headers as arrays (PSR-7 format) — matching PHP Buggregator.
	headers := make(map[string][]string)
	for k, v := range r.Header {
		headers[k] = v
	}

	// Parse query params.
	query := make(map[string]any)
	for k, v := range r.URL.Query() {
		if len(v) == 1 {
			query[k] = v[0]
		} else {
			query[k] = v
		}
	}

	// Parse POST body if form-encoded.
	post := make(map[string]any)
	ct := r.Header.Get("Content-Type")
	if strings.Contains(ct, "application/x-www-form-urlencoded") {
		r.ParseForm()
		for k, v := range r.PostForm {
			if len(v) == 1 {
				post[k] = v[0]
			} else {
				post[k] = v
			}
		}
	} else if strings.Contains(ct, "application/json") && len(body) > 0 {
		var parsed map[string]any
		if json.Unmarshal(body, &parsed) == nil {
			post = parsed
		}
	}

	// Parse cookies.
	cookies := make(map[string]string)
	for _, c := range r.Cookies() {
		cookies[c.Name] = c.Value
	}

	// URI without query string, without leading slash.
	uri := strings.TrimPrefix(r.URL.Path, "/")

	// Build payload matching PHP AnyHttpRequestDump format.
	payload := map[string]any{
		"received_at": time.Now().Format("2006-01-02 15:04:05"),
		"host":        r.Host,
		"request": map[string]any{
			"method":  r.Method,
			"uri":     uri,
			"headers": headers,
			"body":    string(body),
			"query":   query,
			"post":    post,
			"cookies": cookies,
			"files":   []any{},
		},
	}

	b, _ := json.Marshal(payload)

	return &event.Incoming{
		Type:    "http-dump",
		Payload: json.RawMessage(b),
	}, nil
}
