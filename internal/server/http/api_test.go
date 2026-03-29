package http_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/buggregator/go-buggregator/internal/event"
	"github.com/buggregator/go-buggregator/internal/module"
	serverhttp "github.com/buggregator/go-buggregator/internal/server/http"
	"github.com/buggregator/go-buggregator/internal/server/ws"
	"github.com/buggregator/go-buggregator/internal/storage"
)

func setupAPI(t *testing.T) (*http.ServeMux, *storage.SQLiteStore) {
	t.Helper()
	db, err := storage.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	// Create required tables
	for _, sql := range []string{
		`CREATE TABLE IF NOT EXISTS events (
			uuid TEXT PRIMARY KEY, type TEXT NOT NULL, payload TEXT NOT NULL,
			timestamp TEXT NOT NULL, project TEXT, is_pinned INTEGER NOT NULL DEFAULT 0
		)`,
		`CREATE TABLE IF NOT EXISTS projects (key TEXT PRIMARY KEY, name TEXT NOT NULL)`,
		`INSERT OR IGNORE INTO projects (key, name) VALUES ('default', 'Default')`,
	} {
		if _, err := db.Exec(sql); err != nil {
			t.Fatal(err)
		}
	}
	t.Cleanup(func() { db.Close() })

	store := storage.NewSQLiteStore(db)
	hub := ws.NewHub()
	registry := module.NewRegistry()
	es := serverhttp.NewEventService(store, hub, registry, nil)

	mux := http.NewServeMux()
	serverhttp.RegisterAPI(mux, store, event.NewPreviewRegistry(), es, "test-version", db, []string{"sentry", "ray"})

	return mux, store
}

func TestAPI_Version(t *testing.T) {
	mux, _ := setupAPI(t)

	r := httptest.NewRequest("GET", "/api/version", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)

	if w.Code != 200 {
		t.Fatalf("status = %d", w.Code)
	}
	var resp map[string]string
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["version"] != "test-version" {
		t.Errorf("version = %q", resp["version"])
	}
}

func TestAPI_Settings(t *testing.T) {
	mux, _ := setupAPI(t)

	r := httptest.NewRequest("GET", "/api/settings", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)

	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["version"] != "test-version" {
		t.Errorf("version = %q", resp["version"])
	}
	events, ok := resp["events"].([]any)
	if !ok || len(events) != 2 {
		t.Errorf("events = %v", resp["events"])
	}
}

func TestAPI_Events_Empty(t *testing.T) {
	mux, _ := setupAPI(t)

	r := httptest.NewRequest("GET", "/api/events", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)

	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)
	data, ok := resp["data"].([]any)
	if !ok {
		t.Fatal("data is not array")
	}
	if len(data) != 0 {
		t.Errorf("len = %d, want 0", len(data))
	}
}

func TestAPI_Events_CRUD(t *testing.T) {
	mux, store := setupAPI(t)
	ctx := context.Background()

	// Store an event
	ev := event.Event{
		UUID:      "test-uuid-1",
		Type:      "sentry",
		Payload:   json.RawMessage(`{"message":"error"}`),
		Timestamp: 1700000000.0,
		Project:   "default",
	}
	store.Store(ctx, ev)

	// GET /api/events
	t.Run("list events", func(t *testing.T) {
		r := httptest.NewRequest("GET", "/api/events", nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)

		var resp map[string]any
		json.NewDecoder(w.Body).Decode(&resp)
		data := resp["data"].([]any)
		if len(data) != 1 {
			t.Errorf("len = %d, want 1", len(data))
		}
	})

	// GET /api/events?type=sentry
	t.Run("filter by type", func(t *testing.T) {
		r := httptest.NewRequest("GET", "/api/events?type=ray", nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)

		var resp map[string]any
		json.NewDecoder(w.Body).Decode(&resp)
		data := resp["data"].([]any)
		if len(data) != 0 {
			t.Errorf("len = %d, want 0", len(data))
		}
	})

	// GET /api/event/{uuid}
	t.Run("get single event", func(t *testing.T) {
		r := httptest.NewRequest("GET", "/api/event/test-uuid-1", nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)

		if w.Code != 200 {
			t.Fatalf("status = %d", w.Code)
		}
		var resp map[string]any
		json.NewDecoder(w.Body).Decode(&resp)
		if resp["uuid"] != "test-uuid-1" {
			t.Errorf("uuid = %v", resp["uuid"])
		}
	})

	// GET /api/event/{uuid} not found
	t.Run("get nonexistent event", func(t *testing.T) {
		r := httptest.NewRequest("GET", "/api/event/nonexistent", nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)

		if w.Code != 404 {
			t.Errorf("status = %d, want 404", w.Code)
		}
	})

	// POST /api/event/{uuid}/pin
	t.Run("pin event", func(t *testing.T) {
		r := httptest.NewRequest("POST", "/api/event/test-uuid-1/pin", nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)

		if w.Code != 200 {
			t.Fatalf("status = %d", w.Code)
		}

		found, _ := store.FindByUUID(ctx, "test-uuid-1")
		if !found.IsPinned {
			t.Error("expected pinned")
		}
	})

	// DELETE /api/event/{uuid}/pin
	t.Run("unpin event", func(t *testing.T) {
		r := httptest.NewRequest("DELETE", "/api/event/test-uuid-1/pin", nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)

		found, _ := store.FindByUUID(ctx, "test-uuid-1")
		if found.IsPinned {
			t.Error("expected unpinned")
		}
	})

	// DELETE /api/event/{uuid}
	t.Run("delete event", func(t *testing.T) {
		r := httptest.NewRequest("DELETE", "/api/event/test-uuid-1", nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)

		if w.Code != 200 {
			t.Fatalf("status = %d", w.Code)
		}

		found, _ := store.FindByUUID(ctx, "test-uuid-1")
		if found != nil {
			t.Error("expected event to be deleted")
		}
	})
}

func TestAPI_DeleteAll(t *testing.T) {
	mux, store := setupAPI(t)
	ctx := context.Background()

	for _, ev := range []event.Event{
		{UUID: "d1", Type: "sentry", Payload: json.RawMessage(`{}`), Timestamp: 1.0, Project: "default"},
		{UUID: "d2", Type: "ray", Payload: json.RawMessage(`{}`), Timestamp: 2.0, Project: "default"},
	} {
		store.Store(ctx, ev)
	}

	body := `{"type":"sentry"}`
	r := httptest.NewRequest("DELETE", "/api/events", strings.NewReader(body))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)

	if w.Code != 200 {
		t.Fatalf("status = %d", w.Code)
	}

	all, _ := store.FindAll(ctx, event.FindOptions{})
	if len(all) != 1 {
		t.Errorf("len = %d, want 1", len(all))
	}
}

func TestAPI_Projects(t *testing.T) {
	mux, _ := setupAPI(t)

	r := httptest.NewRequest("GET", "/api/projects", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)

	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)
	data := resp["data"].([]any)
	if len(data) != 1 {
		t.Errorf("len = %d, want 1 (default project)", len(data))
	}
	proj := data[0].(map[string]any)
	if proj["key"] != "default" {
		t.Errorf("key = %v", proj["key"])
	}
}
