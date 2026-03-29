package ws

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
)

// MuxRPCHandler routes Centrifugo RPC calls to the HTTP mux.
type MuxRPCHandler struct {
	mux *http.ServeMux
}

func NewMuxRPCHandler(mux *http.ServeMux) *MuxRPCHandler {
	return &MuxRPCHandler{mux: mux}
}

// HandleRPC converts an RPC call (e.g., DELETE api/event/{id}) into
// an HTTP request, executes it against the mux, and returns the response.
func (h *MuxRPCHandler) HandleRPC(method, uri string, data json.RawMessage) (json.RawMessage, error) {
	// Ensure URI starts with /.
	if !strings.HasPrefix(uri, "/") {
		uri = "/" + uri
	}

	// Parse RPC payload — extract token and build request body.
	var payload map[string]any
	if len(data) > 0 {
		json.Unmarshal(data, &payload)
	}
	if payload == nil {
		payload = make(map[string]any)
	}

	// Extract auth token from payload.
	token, _ := payload["token"].(string)
	delete(payload, "token")

	var body io.Reader
	if method == "GET" || method == "HEAD" {
		// For GET/HEAD, add payload as query params.
		if len(payload) > 0 {
			params := "?"
			for k, v := range payload {
				params += k + "=" + toString(v) + "&"
			}
			uri += strings.TrimRight(params, "&")
		}
	} else {
		// For POST/PUT/DELETE, encode as JSON body.
		b, _ := json.Marshal(payload)
		body = bytes.NewReader(b)
	}

	req := httptest.NewRequest(method, uri, body)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("X-Auth-Token", token)
	}

	rec := httptest.NewRecorder()
	h.mux.ServeHTTP(rec, req)

	resp := rec.Result()
	respBody, _ := io.ReadAll(resp.Body)
	resp.Body.Close()

	// Wrap response with status code as the frontend expects.
	var respData any
	json.Unmarshal(respBody, &respData)

	result := map[string]any{
		"code": resp.StatusCode,
		"data": respData,
	}

	return json.Marshal(result)
}

func toString(v any) string {
	switch val := v.(type) {
	case string:
		return val
	default:
		b, _ := json.Marshal(val)
		return string(b)
	}
}
