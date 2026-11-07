package httpdumps

import (
	"database/sql"
	"encoding/json"
	"errors"
	"mime"
	"net/http"
	"net/url"
	"strconv"
	"strings"
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
		rule, err := rules.create(in.MockRule, "api")
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

	route("GET /api/mocks/helpers", func(w http.ResponseWriter, r *http.Request) {
		mockJSON(w, http.StatusOK, map[string]any{"helpers": MockHelpers, "fake": MockFakes()})
	})

	route("POST /api/mocks/preview", func(w http.ResponseWriter, r *http.Request) {
		var in mockPreviewInput
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 3*MaxMockBody)).Decode(&in); err != nil {
			mockJSON(w, http.StatusBadRequest, map[string]any{"message": "invalid JSON body", "code": 400})
			return
		}
		out, err := previewMock(in)
		if err != nil {
			mockError(w, err)
			return
		}
		mockJSON(w, http.StatusOK, out)
	})

	route("POST /api/mocks/import/openapi", func(w http.ResponseWriter, r *http.Request) {
		// JSON only: a cross-site form or text/plain POST (no preflight) must
		// not make the server download URLs.
		if mt, _, err := mime.ParseMediaType(r.Header.Get("Content-Type")); err != nil || mt != "application/json" {
			mockJSON(w, http.StatusUnsupportedMediaType, map[string]any{"message": "Content-Type must be application/json", "code": 415})
			return
		}
		var in OpenAPIImport
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, MaxOpenAPISpec+(1<<20))).Decode(&in); err != nil {
			mockJSON(w, http.StatusBadRequest, map[string]any{"message": "invalid JSON body", "code": 400})
			return
		}
		res, err := importOpenAPI(r.Context(), rules, in, "api")
		if err != nil {
			mockError(w, err)
			return
		}
		mockJSON(w, http.StatusOK, res)
	})

	route("DELETE /api/mocks/groups/{group}", func(w http.ResponseWriter, r *http.Request) {
		n, err := rules.deleteGroup(r.PathValue("group"))
		if err != nil {
			mockError(w, err)
			return
		}
		mockJSON(w, http.StatusOK, map[string]any{"deleted": n})
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
		// A client that does not know template/seed (an older UI) keeps them.
		if !in.hasTemplate || !in.hasSeed {
			if old, err := rules.get(r.PathValue("id")); err == nil {
				if !in.hasTemplate {
					in.Template = old.Template
				}
				if !in.hasSeed {
					in.Seed = old.Seed
				}
			}
		}
		rule, err := rules.update(r.PathValue("id"), in.MockRule)
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

// ruleInput is a decoded rule and which optional keys the body had.
type ruleInput struct {
	MockRule
	hasTemplate, hasSeed bool
}

// readRule decodes a rule from the body. "enabled" defaults to true when absent.
func readRule(w http.ResponseWriter, r *http.Request) (ruleInput, bool) {
	var in struct {
		MockRule
		Enabled  *bool  `json:"enabled"`
		Template *bool  `json:"template"`
		Seed     *int64 `json:"seed"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 2*MaxMockBody)).Decode(&in); err != nil {
		mockJSON(w, http.StatusBadRequest, map[string]any{"message": "invalid JSON body", "code": 400})
		return ruleInput{}, false
	}
	out := ruleInput{MockRule: in.MockRule, hasTemplate: in.Template != nil, hasSeed: in.Seed != nil}
	out.Enabled = in.Enabled == nil || *in.Enabled
	if in.Template != nil {
		out.Template = *in.Template
	}
	if in.Seed != nil {
		out.Seed = *in.Seed
	}
	return out, true
}

// mockPreviewInput is the body of POST /api/mocks/preview: a rule (not saved)
// and a sample request. Path may carry a query string.
type mockPreviewInput struct {
	Rule    MockRule `json:"rule"`
	Request struct {
		Method  string            `json:"method"`
		Path    string            `json:"path"`
		Headers map[string]string `json:"headers"`
		Body    string            `json:"body"`
	} `json:"request"`
}

// previewMock renders a rule for a sample request, the way the mock server
// would answer it. A template error is returned in "error", not as a failure.
func previewMock(in mockPreviewInput) (map[string]any, error) {
	rule := in.Rule
	if err := rule.Normalize(); err != nil {
		return nil, err
	}
	method := strings.ToUpper(strings.TrimSpace(in.Request.Method))
	if method == "" {
		method = http.MethodGet
		if rule.Method != "ANY" {
			method = rule.Method
		}
	}
	u, err := url.Parse(in.Request.Path)
	if err != nil || !strings.HasPrefix(u.Path, "/") {
		return nil, ValidationError{"request.path": "must be a path starting with /, e.g. /users/42?page=2"}
	}
	path := strings.TrimPrefix(u.Path, MockPrefix)
	if path == "" {
		path = "/"
	}
	header := http.Header{}
	for k, v := range in.Request.Headers {
		header.Set(k, v)
	}

	out := map[string]any{
		"matched": (rule.Method == "ANY" || rule.Method == method) && matchMockPath(rule.Path, path),
		"status":  rule.Status, "headers": rule.Headers, "body": rule.Body, "error": nil,
	}
	if rule.Template {
		req := &mockRequest{Method: method, Path: path, Params: mockPathParams(rule.Path, path),
			Query: u.Query(), Header: header, Body: []byte(in.Request.Body)}
		rendered, err := renderMockRule(&rule, req)
		if err != nil {
			out["status"], out["headers"], out["body"] = http.StatusInternalServerError, map[string]string{}, ""
			out["error"] = "mock template: " + err.Error()
		} else {
			out["headers"], out["body"] = rendered.Headers, string(rendered.Body)
		}
	}
	return out, nil
}

// listMockRequests returns the latest HTTP dumps captured under /mock/, newest
// first, optionally for one project or one rule.
// mockRequestsQuery builds the query of listMockRequests. It reads the partial
// index idx_events_mock_requests (migration 000031) newest first and stops at
// the limit. INDEXED BY: left alone, the planner picks events_idx_type and
// sorts every http-dump. The first WHERE terms must stay the index's.
func mockRequestsQuery(project, ruleID string, limit int) (string, []any) {
	query := `SELECT uuid, payload, timestamp, project FROM events INDEXED BY idx_events_mock_requests
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
	return query, append(args, limit)
}

func listMockRequests(db *sql.DB, project, ruleID string, limit int) ([]map[string]any, error) {
	query, args := mockRequestsQuery(project, ruleID, limit)
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
