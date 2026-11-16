package sentry

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// The window must cut spans by time, not by date: a span from the start of
// the same UTC day is outside a 60-minute window at noon.
func TestServiceMapWindowSameDate(t *testing.T) {
	db := setupTestDB(t)
	mux := http.NewServeMux()
	registerAPI(mux, db, noAuth)

	noon := time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)
	now = func() time.Time { return noon }
	t.Cleanup(func() { now = time.Now })

	for _, s := range []struct{ id, peer, start string }{
		{"recent", "db-recent", "2026-11-16T11:30:00.250Z"},
		{"same-date-old", "db-old", "2026-11-16T00:05:00Z"},
		{"yesterday", "db-yesterday", "2026-11-15T11:30:00Z"},
	} {
		if _, err := db.Exec(`INSERT INTO sentry_spans (id, span_id, trace_id, service_name, peer_address, peer_type, start_ts)
			VALUES (?, ?, 't', 'app', ?, 'db', ?)`, s.id, s.id, s.peer, s.start); err != nil {
			t.Fatal(err)
		}
	}

	req := httptest.NewRequest("GET", "/api/sentry/service-map?window=60", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("status = %d: %s", w.Code, w.Body)
	}
	var resp struct {
		Edges []struct {
			Target string `json:"target"`
		} `json:"edges"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if len(resp.Edges) != 1 || resp.Edges[0].Target != "db-recent" {
		t.Fatalf("edges = %+v, want only db-recent", resp.Edges)
	}
}
