package sentry

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	httpserver "github.com/buggregator/server/v2/internal/server/http"
)

// registerAPI adds the /api/sentry routes; protect is the API auth middleware.
func registerAPI(mux *http.ServeMux, db *sql.DB, protect func(http.Handler) http.Handler) {
	route := func(pattern string, h http.HandlerFunc) { mux.Handle(pattern, protect(h)) }
	route("GET /api/sentry/exceptions", handleExceptionsList(db))
	route("GET /api/sentry/exceptions/{id}", handleExceptionDetail(db))
	route("GET /api/sentry/traces", handleTracesList(db))
	route("GET /api/sentry/traces/{traceId}", handleTraceDetail(db))
	route("GET /api/sentry/logs", handleLogsList(db))
	route("GET /api/sentry/metrics", handleMetricsList(db))
	route("GET /api/sentry/service-map", handleServiceMap(db))
	route("GET /api/sentry/counts", handleCounts(db))
	route("DELETE /api/sentry/all", handleClearAll(db))
}

// handleClearAll deletes Sentry structured data.
//
//	DELETE /api/sentry/all                      every project, everything (as before)
//	DELETE /api/sentry/all?project=shop         only the project "shop"
//	DELETE /api/sentry/all?keep_pinned=1        keep errors whose event is pinned
//
// Both parameters combine.
func handleClearAll(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		keepPinned, _ := strconv.ParseBool(q.Get("keep_pinned"))
		if err := clearSentry(db, strings.TrimSpace(q.Get("project")), keepPinned); err != nil {
			apiError(w, err.Error(), http.StatusInternalServerError)
			return
		}
		apiJSON(w, map[string]any{"status": true})
	}
}

// clearSentry deletes the Sentry rows of one project (every project when
// project is empty), keeping errors whose event is pinned when keepPinned is
// set. Foreign keys are not enforced, so child rows are deleted explicitly.
func clearSentry(db *sql.DB, project string, keepPinned bool) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if project == "" && !keepPinned {
		for _, t := range []string{
			"sentry_breadcrumbs", "sentry_exceptions", "sentry_error_events",
			"sentry_spans", "sentry_transactions", "sentry_traces",
			"sentry_logs", "sentry_metrics",
		} {
			if _, err := tx.Exec("DELETE FROM " + t); err != nil {
				return err
			}
		}
		return tx.Commit()
	}

	var where string // " WHERE project_id = ?" or ""
	var args []any
	if project != "" {
		where, args = " WHERE project_id = ?", []any{project}
	}

	errWhere := where
	if keepPinned {
		pinned := "event_id NOT IN (SELECT uuid FROM events WHERE is_pinned = 1)"
		if errWhere == "" {
			errWhere = " WHERE " + pinned
		} else {
			errWhere += " AND " + pinned
		}
	}

	// Traces are shared by trace_id across projects: remember the ones this
	// clear touches, then recount or drop them once their spans are gone.
	stmts := []string{
		`CREATE TEMP TABLE IF NOT EXISTS sentry_clear_traces (trace_id TEXT PRIMARY KEY)`,
		`DELETE FROM sentry_clear_traces`,
		`INSERT OR IGNORE INTO sentry_clear_traces SELECT trace_id FROM sentry_spans` + where,
		`INSERT OR IGNORE INTO sentry_clear_traces SELECT trace_id FROM sentry_transactions` + where,
		`DELETE FROM sentry_breadcrumbs WHERE error_event_id IN (SELECT id FROM sentry_error_events` + errWhere + `)`,
		`DELETE FROM sentry_exceptions WHERE error_event_id IN (SELECT id FROM sentry_error_events` + errWhere + `)`,
		`DELETE FROM sentry_error_events` + errWhere,
		`DELETE FROM sentry_spans` + where,
		`DELETE FROM sentry_transactions` + where,
		`DELETE FROM sentry_logs` + where,
		`DELETE FROM sentry_metrics` + where,
		`DELETE FROM sentry_traces WHERE trace_id IN (SELECT trace_id FROM sentry_clear_traces)
			AND NOT EXISTS (SELECT 1 FROM sentry_spans s WHERE s.trace_id = sentry_traces.trace_id)
			AND NOT EXISTS (SELECT 1 FROM sentry_transactions t WHERE t.trace_id = sentry_traces.trace_id)`,
		`UPDATE sentry_traces SET
			span_count = (SELECT COUNT(*) FROM sentry_spans s WHERE s.trace_id = sentry_traces.trace_id),
			error_count = (SELECT COUNT(*) FROM sentry_spans s WHERE s.trace_id = sentry_traces.trace_id
				AND s.transaction_id IS NOT NULL AND s.is_error = 1)
			WHERE trace_id IN (SELECT trace_id FROM sentry_clear_traces)`,
		`DELETE FROM sentry_clear_traces`,
	}
	for _, st := range stmts {
		var a []any
		if strings.Contains(st, "?") {
			a = args
		}
		if _, err := tx.Exec(st, a...); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// timeWindow reads the period bounds from the query parameters.
//
//	from, to — unix seconds or RFC3339 ("2026-09-01T10:00:00Z", "2026-09-01");
//	window   — a window relative to now: "24h", "15m", "7d".
//
// A zero result means "unbounded". None of the sentry endpoints could be
// narrowed by time before, so on a stream of tens of thousands of events a day
// the only way to reach a particular hour was to page through the whole list.
func timeWindow(r *http.Request) (from, to time.Time) {
	q := r.URL.Query()
	from = parseWhen(q.Get("from"))
	to = parseWhen(q.Get("to"))

	if from.IsZero() && to.IsZero() {
		if w := strings.TrimSpace(q.Get("window")); w != "" && w != "all" && w != "0" {
			if d, err := httpserver.ParseWindow(w); err == nil && d > 0 {
				from = time.Now().UTC().Add(-d)
			}
		}
	}
	return from, to
}

// parseWhen reads a single bound: unix seconds or one of the ISO forms.
func parseWhen(v string) time.Time {
	v = strings.TrimSpace(v)
	if v == "" {
		return time.Time{}
	}
	if f, err := strconv.ParseFloat(v, 64); err == nil && f > 0 {
		return time.Unix(int64(f), 0).UTC()
	}
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02T15:04:05", "2006-01-02"} {
		if t, err := time.Parse(layout, v); err == nil {
			return t.UTC()
		}
	}
	return time.Time{}
}

// appendTimeConditions adds the period bounds to a WHERE clause.
//
// The stored format differs per column, hence the iso flag:
//   - received_at, first_seen, last_seen — 'YYYY-MM-DD HH:MM:SS' (datetime('now'));
//   - start_ts, end_ts of transactions and spans — ISO with T and Z.
//
// The comparison is done on strings: for both formats lexicographic order
// matches chronological order, and the indexes on these columns are textual.
func appendTimeConditions(conditions []string, args []any, r *http.Request, column string, iso bool) ([]string, []any) {
	from, to := timeWindow(r)
	layout := "2006-01-02 15:04:05"
	if iso {
		layout = "2006-01-02T15:04:05"
	}
	if !from.IsZero() {
		conditions = append(conditions, column+" >= ?")
		args = append(args, from.UTC().Format(layout))
	}
	if !to.IsZero() {
		conditions = append(conditions, column+" <= ?")
		args = append(args, to.UTC().Format(layout))
	}
	return conditions, args
}

// appendProject narrows a query to ?project when it is given; without it every
// project is read, as before.
func appendProject(conditions []string, args []any, r *http.Request, column string) ([]string, []any) {
	if v := strings.TrimSpace(r.URL.Query().Get("project")); v != "" {
		conditions = append(conditions, column+" = ?")
		args = append(args, v)
	}
	return conditions, args
}

// projectClause is " AND <column> = ?" with its argument for ?project, or nothing.
func projectClause(r *http.Request, column string) (string, []any) {
	conds, args := appendProject(nil, nil, r, column)
	if len(conds) == 0 {
		return "", nil
	}
	return " AND " + conds[0], args
}

// tsUTC wraps a time column so the value leaves the API as an ISO string with
// an explicit zone: 2026-09-04T05:33:12Z.
//
// received_at, first_seen and last_seen are written with datetime('now'), which
// is UTC but carries **no zone marker**: "2026-09-04 05:33:12". A browser reads
// such a string as local time, so "last seen" in the UI was off by the viewer's
// UTC offset — in UTC+3 a fresh error showed up as "3 hours ago". Sorting and
// filtering still use the raw column; only the representation changes.
func tsUTC(column string) string {
	// COALESCE covers a value strftime cannot parse (empty string, garbage):
	// the original value goes out instead of NULL.
	return "COALESCE(strftime('%Y-%m-%dT%H:%M:%SZ', " + column + "), " + column + ")"
}

// whereOf builds a WHERE clause from conditions (empty string when there are none).
func whereOf(conditions []string) string {
	if len(conditions) == 0 {
		return ""
	}
	return " WHERE " + strings.Join(conditions, " AND ")
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
