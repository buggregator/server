package http

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/buggregator/go-buggregator/internal/event"
)

// AuthSettings holds auth info exposed via /api/settings.
type AuthSettings struct {
	Enabled  bool
	LoginURL string
}

// ListLimits bounds what /api/events and /api/events/preview return.
//
// Both endpoints used to return every event matching type/project: Limit and
// Offset existed in FindOptions but were never filled in. On a busy project
// that is a very large response — measured on a production instance, a single
// project preview weighed 25.8 MB — which the frontend then filters in the
// browser.
type ListLimits struct {
	DefaultLimit  int           // how many events to return when limit is absent; 0 = unlimited
	MaxLimit      int           // ceiling for an explicit limit; 0 = no ceiling
	DefaultWindow time.Duration // time window applied when neither from/to nor window is given; 0 = no window
}

// parseListOptions builds FindOptions from query parameters.
//
//	type, project — unchanged;
//	limit         — how many events to return (capped by MaxLimit);
//	offset / page — offset (page is counted from limit, 1-based);
//	from, to      — window bounds: unix seconds or RFC3339 ("2026-09-01T10:00:00Z");
//	window        — window relative to now: "24h", "15m", "7d";
//	                "all" or "0" opt out of DefaultWindow.
func parseListOptions(r *http.Request, lim ListLimits) event.FindOptions {
	q := r.URL.Query()
	opts := event.FindOptions{
		Type:    q.Get("type"),
		Project: q.Get("project"),
		Limit:   lim.DefaultLimit,
	}

	if v := q.Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			opts.Limit = n
		}
	}
	if lim.MaxLimit > 0 && opts.Limit > lim.MaxLimit {
		opts.Limit = lim.MaxLimit
	}

	if v := q.Get("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			opts.Offset = n
		}
	} else if v := q.Get("page"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 1 {
			opts.Offset = (n - 1) * opts.Limit
		}
	}

	opts.From = parseTimeParam(q.Get("from"))
	opts.To = parseTimeParam(q.Get("to"))

	// The default window only applies when the caller set no bounds itself.
	if opts.From == 0 && opts.To == 0 {
		switch w := strings.TrimSpace(q.Get("window")); w {
		case "":
			if lim.DefaultWindow > 0 {
				opts.From = epochSeconds(time.Now().Add(-lim.DefaultWindow))
			}
		case "all", "0":
			// explicit opt-out
		default:
			if d, err := ParseWindow(w); err == nil && d > 0 {
				opts.From = epochSeconds(time.Now().Add(-d))
			} else if lim.DefaultWindow > 0 {
				opts.From = epochSeconds(time.Now().Add(-lim.DefaultWindow))
			}
		}
	}

	return opts
}

// parseTimeParam reads a window bound: unix seconds (fraction allowed) or
// RFC3339. Returns 0 for an empty or unreadable value, which leaves the bound
// unset instead of failing the request.
func parseTimeParam(v string) float64 {
	v = strings.TrimSpace(v)
	if v == "" {
		return 0
	}
	if f, err := strconv.ParseFloat(v, 64); err == nil && f > 0 {
		return f
	}
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02T15:04:05", "2006-01-02"} {
		if t, err := time.Parse(layout, v); err == nil {
			return epochSeconds(t)
		}
	}
	return 0
}

// ParseWindow is time.ParseDuration plus the "d" suffix for days, which Go does
// not support but is the most common unit in a UI ("7d").
func ParseWindow(v string) (time.Duration, error) {
	if strings.HasSuffix(v, "d") {
		if n, err := strconv.ParseFloat(strings.TrimSuffix(v, "d"), 64); err == nil {
			return time.Duration(n * float64(24*time.Hour)), nil
		}
	}
	return time.ParseDuration(v)
}

func epochSeconds(t time.Time) float64 {
	return float64(t.UnixMicro()) / 1e6
}

// listMeta reports the applied limits, so a client can tell a truncated
// response from an exhausted one.
func listMeta(opts event.FindOptions, returned int) map[string]any {
	meta := map[string]any{
		"limit":    opts.Limit,
		"offset":   opts.Offset,
		"returned": returned,
	}
	if opts.From > 0 {
		meta["from"] = opts.From
	}
	if opts.To > 0 {
		meta["to"] = opts.To
	}
	return meta
}

// RegisterAPI registers core API routes on the given mux.
// authMiddleware wraps protected routes; pass a no-op when auth is disabled.
func RegisterAPI(mux *http.ServeMux, store event.Store, previews *event.PreviewRegistry, es *EventService, version string, db *sql.DB, enabledEvents []string, authSettings AuthSettings, authMiddleware func(http.Handler) http.Handler, listLimits ListLimits) {
	// Public routes (no auth required).
	mux.HandleFunc("GET /api/version", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]string{"version": version})
	})

	mux.HandleFunc("GET /api/settings", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{
			"auth": map[string]any{
				"enabled":   authSettings.Enabled,
				"login_url": authSettings.LoginURL,
			},
			"version": version,
			"events":  enabledEvents,
		})
	})

	// Protected routes (require auth when enabled).
	protect := func(pattern string, handler http.HandlerFunc) {
		mux.Handle(pattern, authMiddleware(handler))
	}

	// List events.
	protect("GET /api/events", func(w http.ResponseWriter, r *http.Request) {
		opts := parseListOptions(r, listLimits)
		events, err := store.FindAll(r.Context(), opts)
		if err != nil {
			writeError(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if events == nil {
			events = []event.Event{}
		}
		writeJSON(w, map[string]any{"data": events, "meta": listMeta(opts, len(events))})
	})

	// List event previews.
	protect("GET /api/events/preview", func(w http.ResponseWriter, r *http.Request) {
		opts := parseListOptions(r, listLimits)
		events, err := store.FindAll(r.Context(), opts)
		if err != nil {
			writeError(w, err.Error(), http.StatusInternalServerError)
			return
		}
		result := make([]event.Preview, 0, len(events))
		for _, ev := range events {
			result = append(result, previews.BuildPreview(ev))
		}
		writeJSON(w, map[string]any{"data": result, "meta": listMeta(opts, len(result))})
	})

	// Get single event.
	protect("GET /api/event/{uuid}", func(w http.ResponseWriter, r *http.Request) {
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
	protect("DELETE /api/event/{uuid}", func(w http.ResponseWriter, r *http.Request) {
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
	protect("DELETE /api/events", func(w http.ResponseWriter, r *http.Request) {
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
	protect("POST /api/event/{uuid}/pin", func(w http.ResponseWriter, r *http.Request) {
		uuid := r.PathValue("uuid")
		if err := store.Pin(r.Context(), uuid); err != nil {
			writeError(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, map[string]any{"status": "pinned"})
	})

	// Unpin event.
	protect("DELETE /api/event/{uuid}/pin", func(w http.ResponseWriter, r *http.Request) {
		uuid := r.PathValue("uuid")
		if err := store.Unpin(r.Context(), uuid); err != nil {
			writeError(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, map[string]any{"status": "unpinned"})
	})

	// List projects.
	protect("GET /api/projects", func(w http.ResponseWriter, r *http.Request) {
		rows, err := db.QueryContext(r.Context(), `SELECT key, name FROM projects`)
		if err != nil {
			writeError(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		var projects []map[string]any
		for rows.Next() {
			var key, name string
			rows.Scan(&key, &name)
			projects = append(projects, map[string]any{
				"key":        key,
				"name":       name,
				"is_default": key == "default",
			})
		}
		if projects == nil {
			projects = []map[string]any{}
		}
		writeJSON(w, map[string]any{"data": projects, "meta": map[string]any{}})
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
