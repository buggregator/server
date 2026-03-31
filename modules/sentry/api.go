package sentry

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
)

func registerAPI(mux *http.ServeMux, db *sql.DB) {
	mux.HandleFunc("GET /api/sentry/exceptions", handleExceptionsList(db))
	mux.HandleFunc("GET /api/sentry/exceptions/{id}", handleExceptionDetail(db))
	mux.HandleFunc("GET /api/sentry/traces", handleTracesList(db))
	mux.HandleFunc("GET /api/sentry/traces/{traceId}", handleTraceDetail(db))
	mux.HandleFunc("GET /api/sentry/logs", handleLogsList(db))
	mux.HandleFunc("GET /api/sentry/service-map", handleServiceMap(db))
	mux.HandleFunc("GET /api/sentry/counts", handleCounts(db))
}

// pagination extracts page/limit from query params with defaults.
func pagination(r *http.Request, defaultLimit int) (limit, offset int) {
	limit = defaultLimit
	page := 1

	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 500 {
			limit = n
		}
	}
	if v := r.URL.Query().Get("page"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			page = n
		}
	}

	offset = (page - 1) * limit
	return
}

func apiJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

func apiError(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]any{"message": msg, "code": code})
}

// listResponse wraps data with pagination metadata.
func listResponse(data any, total, page, limit int) map[string]any {
	return map[string]any{
		"data": data,
		"meta": map[string]any{
			"total": total,
			"page":  page,
			"limit": limit,
		},
	}
}

// currentPage derives page number from offset and limit.
func currentPage(offset, limit int) int {
	if limit == 0 {
		return 1
	}
	return (offset / limit) + 1
}

// scanNullString returns empty string for NULL columns.
func scanNullString(ns sql.NullString) string {
	if ns.Valid {
		return ns.String
	}
	return ""
}

// scanNullBool returns nil for NULL, pointer to bool otherwise.
func scanNullBool(nb sql.NullBool) *bool {
	if nb.Valid {
		return &nb.Bool
	}
	return nil
}

// scanNullInt returns nil for NULL, pointer to int otherwise.
func scanNullInt(ni sql.NullInt64) *int {
	if ni.Valid {
		v := int(ni.Int64)
		return &v
	}
	return nil
}
