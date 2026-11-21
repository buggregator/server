package ws

import (
	"encoding/json"
	"net/http"
	"testing"
)

// A handler panic over the RPC bridge answers that call with 500; the bridge
// and the next call keep working.
func TestHandleRPC_RecoversHandlerPanic(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/boom", func(http.ResponseWriter, *http.Request) {
		var s []int
		_ = s[5] // index out of range, as a makeslice/len bug would
	})
	mux.HandleFunc("GET /api/ok", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"ok":true}`))
	})
	h := NewMuxRPCHandler(mux)

	var resp struct {
		Code int `json:"code"`
		Data struct {
			Message string `json:"message"`
			OK      bool   `json:"ok"`
		} `json:"data"`
	}
	raw, err := h.HandleRPC("GET", "api/boom", nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, &resp); err != nil {
		t.Fatal(err)
	}
	if resp.Code != http.StatusInternalServerError || resp.Data.Message != "internal server error" {
		t.Errorf("panicking route = %s", raw)
	}

	raw, err = h.HandleRPC("GET", "api/ok", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp.Data.Message = ""
	if err := json.Unmarshal(raw, &resp); err != nil {
		t.Fatal(err)
	}
	if resp.Code != http.StatusOK || !resp.Data.OK {
		t.Errorf("next call = %s", raw)
	}
}

// Any page can open the WebSocket: the bridge must not reach the release of
// a caught email (sends to a real mailbox), however the path is spelled.
func TestHandleRPC_RefusesRelease(t *testing.T) {
	reached := false
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/smtp/release/{uuid}", func(http.ResponseWriter, *http.Request) { reached = true })
	h := NewMuxRPCHandler(mux)
	for _, uri := range []string{"api/smtp/release/m1", "/api/smtp/./release/m1", "/api/smtp/%72elease/m1"} {
		raw, err := h.HandleRPC("POST", uri, json.RawMessage(`{"to":["a@example.com"]}`))
		if err != nil {
			t.Fatal(err)
		}
		var resp struct {
			Code int `json:"code"`
		}
		json.Unmarshal(raw, &resp)
		if resp.Code != http.StatusForbidden {
			t.Errorf("%s: code %d, want 403", uri, resp.Code)
		}
	}
	if reached {
		t.Error("the release handler was reached over RPC")
	}
}
