package httpdumps

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
)

// registerMockAPI adds the rule CRUD and the mock request list. protect wraps
// each route with the API's auth middleware.
func registerMockAPI(mux *http.ServeMux, rules *mockStore, db *sql.DB, protect func(http.Handler) http.Handler) {
	route := func(pattern string, h http.HandlerFunc) { mux.Handle(pattern, protect(h)) }

	route("GET /api/mocks", func(w http.ResponseWriter, r *http.Request) {
		mockJSON(w, http.StatusOK, map[string]any{"data": rules.list()})
	})

	route("POST /api/mocks", func(w http.ResponseWriter, r *http.Request) {
		in, ok := readRule(w, r)
		if !ok {
			return
		}
		rule, err := rules.create(in)
		if err != nil {
			mockError(w, err)
			return
		}
		mockJSON(w, http.StatusCreated, rule)
	})

	route("GET /api/mocks/requests", func(w http.ResponseWriter, r *http.Request) {
		limit := 50
		if v, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil && v > 0 && v <= 500 {
			limit = v
		}
		data, err := listMockRequests(db, r.URL.Query().Get("project"), r.URL.Query().Get("rule_id"), limit)
		if err != nil {
			mockError(w, err)
			return
		}
		mockJSON(w, http.StatusOK, map[string]any{"data": data})
	})

	route("POST /api/mocks/order", func(w http.ResponseWriter, r *http.Request) {
		var in struct {
			IDs []string `json:"ids"`
		}
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&in); err != nil {
			mockJSON(w, http.StatusBadRequest, map[string]any{"message": "invalid JSON body", "code": 400})
			return
		}
		if err := rules.reorder(in.IDs); err != nil {
			mockError(w, err)
			return
		}
		mockJSON(w, http.StatusOK, map[string]any{"data": rules.list()})
	})

	route("GET /api/mocks/{id}", func(w http.ResponseWriter, r *http.Request) {
		rule, err := rules.get(r.PathValue("id"))
		if err != nil {
			mockError(w, err)
			return
		}
		mockJSON(w, http.StatusOK, rule)
	})

	route("PUT /api/mocks/{id}", func(w http.ResponseWriter, r *http.Request) {
		in, ok := readRule(w, r)
		if !ok {
			return
		}
		rule, err := rules.update(r.PathValue("id"), in)
		if err != nil {
			mockError(w, err)
			return
		}
		mockJSON(w, http.StatusOK, rule)
	})

	route("DELETE /api/mocks/{id}", func(w http.ResponseWriter, r *http.Request) {
		if err := rules.delete(r.PathValue("id")); err != nil {
			mockError(w, err)
			return
		}
		mockJSON(w, http.StatusOK, map[string]any{"status": true})
	})
}

// readRule decodes a rule from the body. "enabled" defaults to true when absent.
func readRule(w http.ResponseWriter, r *http.Request) (MockRule, bool) {
	var in struct {
		MockRule
		Enabled *bool `json:"enabled"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 2*MaxMockBody)).Decode(&in); err != nil {
		mockJSON(w, http.StatusBadRequest, map[string]any{"message": "invalid JSON body", "code": 400})
		return MockRule{}, false
	}
	rule := in.MockRule
	rule.Enabled = in.Enabled == nil || *in.Enabled
	return rule, true
}

// listMockRequests returns the latest HTTP dumps captured under /mock/, newest
// first, optionally for one project or one rule.
func listMockRequests(db *sql.DB, project, ruleID string, limit int) ([]map[string]any, error) {
	query := `SELECT uuid, payload, timestamp, project FROM events
		WHERE type = 'http-dump' AND json_type(payload, '$.mock') IS NOT NULL`
	var args []any
	if project != "" {
		query += ` AND project = ?`
		args = append(args, project)
	}
	if ruleID != "" {
		query += ` AND json_extract(payload, '$.mock.rule_id') = ?`
		args = append(args, ruleID)
	}
	query += ` ORDER BY timestamp DESC LIMIT ?`
	args = append(args, limit)

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []map[string]any{}
	for rows.Next() {
		var uuid, payload, ts string
		var proj sql.NullString
		if err := rows.Scan(&uuid, &payload, &ts, &proj); err != nil {
			return nil, err
		}
		tsf, _ := strconv.ParseFloat(ts, 64)
		out = append(out, map[string]any{
			"uuid": uuid, "type": "http-dump", "project": proj.String, "timestamp": tsf,
			"payload": json.RawMessage(payload),
		})
	}
	return out, rows.Err()
}

func mockJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}

func mockError(w http.ResponseWriter, err error) {
	var verr ValidationError
	switch {
	case errors.As(err, &verr):
		mockJSON(w, http.StatusUnprocessableEntity, map[string]any{"message": err.Error(), "code": 422, "errors": verr})
	case errors.Is(err, errMockNotFound):
		mockJSON(w, http.StatusNotFound, map[string]any{"message": err.Error(), "code": 404})
	case errors.Is(err, errMockReadOnly):
		mockJSON(w, http.StatusConflict, map[string]any{"message": err.Error(), "code": 409})
	default:
		mockJSON(w, http.StatusInternalServerError, map[string]any{"message": err.Error(), "code": 500})
	}
}
