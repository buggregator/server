package http

import (
	"encoding/json"
	"net/http"

	"github.com/buggregator/go-buggregator/internal/event"
)

// RegisterAPI registers core API routes on the given mux.
func RegisterAPI(mux *http.ServeMux, store event.Store, previews *event.PreviewRegistry, es *EventService, version string) {
	mux.HandleFunc("GET /api/version", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]string{"version": version})
	})

	mux.HandleFunc("GET /api/settings", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{
			"auth":    map[string]any{"enabled": false},
			"version": version,
			"events":  []string{"sentry", "ray", "var-dump", "inspector", "monolog", "smtp", "http-dumps", "profiler"},
		})
	})

	// List events.
	mux.HandleFunc("GET /api/events", func(w http.ResponseWriter, r *http.Request) {
		opts := event.FindOptions{
			Type:    r.URL.Query().Get("type"),
			Project: r.URL.Query().Get("project"),
		}
		events, err := store.FindAll(r.Context(), opts)
		if err != nil {
			writeError(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if events == nil {
			events = []event.Event{}
		}
		writeJSON(w, map[string]any{"data": events, "meta": map[string]any{}})
	})

	// List event previews.
	mux.HandleFunc("GET /api/events/preview", func(w http.ResponseWriter, r *http.Request) {
		opts := event.FindOptions{
			Type:    r.URL.Query().Get("type"),
			Project: r.URL.Query().Get("project"),
		}
		events, err := store.FindAll(r.Context(), opts)
		if err != nil {
			writeError(w, err.Error(), http.StatusInternalServerError)
			return
		}
		result := make([]event.Preview, 0, len(events))
		for _, ev := range events {
			result = append(result, previews.BuildPreview(ev))
		}
		writeJSON(w, map[string]any{"data": result, "meta": map[string]any{}})
	})

	// Get single event.
	mux.HandleFunc("GET /api/event/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		uuid := r.PathValue("uuid")
		ev, err := store.FindByUUID(r.Context(), uuid)
		if err != nil {
			writeError(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if ev == nil {
			writeError(w, "event not found", http.StatusNotFound)
			return
		}
		writeJSON(w, ev)
	})

	// Delete single event.
	mux.HandleFunc("DELETE /api/event/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		uuid := r.PathValue("uuid")
		ev, _ := store.FindByUUID(r.Context(), uuid)
		if err := store.Delete(r.Context(), uuid); err != nil {
			writeError(w, err.Error(), http.StatusInternalServerError)
			return
		}
		project := ""
		if ev != nil {
			project = ev.Project
		}
		es.BroadcastDeleted(uuid, project)
		writeJSON(w, map[string]any{"status": true})
	})

	// Clear events.
	mux.HandleFunc("DELETE /api/events", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Type    string   `json:"type"`
			Project string   `json:"project"`
			UUIDs   []string `json:"uuids"`
		}
		json.NewDecoder(r.Body).Decode(&body)

		opts := event.DeleteOptions{
			Type:    body.Type,
			Project: body.Project,
			UUIDs:   body.UUIDs,
		}
		if err := store.DeleteAll(r.Context(), opts); err != nil {
			writeError(w, err.Error(), http.StatusInternalServerError)
			return
		}
		es.BroadcastCleared(body.Type, body.Project)
		writeJSON(w, map[string]any{"status": true})
	})

	// Pin event.
	mux.HandleFunc("POST /api/event/{uuid}/pin", func(w http.ResponseWriter, r *http.Request) {
		uuid := r.PathValue("uuid")
		if err := store.Pin(r.Context(), uuid); err != nil {
			writeError(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, map[string]any{"status": "pinned"})
	})

	// Unpin event.
	mux.HandleFunc("DELETE /api/event/{uuid}/pin", func(w http.ResponseWriter, r *http.Request) {
		uuid := r.PathValue("uuid")
		if err := store.Unpin(r.Context(), uuid); err != nil {
			writeError(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, map[string]any{"status": "unpinned"})
	})
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]any{"message": msg, "code": code})
}
