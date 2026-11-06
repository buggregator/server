package sentry

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/buggregator/go-buggregator/internal/storage"
)

// clearTestDB has the core events table too: keep_pinned reads the pins there.
func clearTestDB(t *testing.T) (*sql.DB, *http.ServeMux) {
	t.Helper()
	db, err := storage.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	m := storage.NewMigrator(db)
	if err := m.AddFromFS("core", storage.CoreMigrations, "migrations"); err != nil {
		t.Fatal(err)
	}
	if err := m.AddFromFS("sentry", migrations, "migrations"); err != nil {
		t.Fatal(err)
	}
	if err := m.Run(); err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	registerAPI(mux, db)
	return db, mux
}

// seedProject stores one error (with an exception and a breadcrumb), one
// transaction with two spans (one failed) in traceID, a standalone v2 span,
// a log and a metric under project.
func seedProject(t *testing.T, db *sql.DB, project, eventID, traceID string) {
	t.Helper()
	payload, _ := json.Marshal(map[string]any{
		"event_id":    eventID,
		"level":       "error",
		"exception":   map[string]any{"values": []map[string]any{{"type": "E", "value": eventID}}},
		"breadcrumbs": map[string]any{"values": []map[string]any{{"message": "crumb"}}},
	})
	var ev ErrorEvent
	json.Unmarshal(payload, &ev)
	if _, err := storeErrorEvent(db, &ev, payload, project); err != nil {
		t.Fatal(err)
	}

	txnPayload, _ := json.Marshal(map[string]any{
		"event_id": "txn-" + eventID, "type": "transaction", "transaction": "GET /",
		"start_timestamp": 1.0, "timestamp": 2.0,
		"contexts": map[string]any{"trace": map[string]any{"trace_id": traceID, "span_id": "root"}},
		"spans": []map[string]any{
			{"span_id": "a-" + eventID, "trace_id": traceID, "start_timestamp": 1.0, "timestamp": 1.5, "status": "ok"},
			{"span_id": "b-" + eventID, "trace_id": traceID, "start_timestamp": 1.0, "timestamp": 1.5, "status": "internal_error"},
		},
	})
	var txn Transaction
	json.Unmarshal(txnPayload, &txn)
	if _, err := storeTransaction(db, &txn, txnPayload, project); err != nil {
		t.Fatal(err)
	}
	if err := storeSpansV2(db, []RawSpan{{SpanID: "v2-" + eventID, TraceID: traceID}}, nil, project); err != nil {
		t.Fatal(err)
	}
	if err := storeLogs(db, []LogRecord{{Level: "info", Body: eventID}}, project); err != nil {
		t.Fatal(err)
	}
	if err := storeMetrics(db, []MetricRecord{{Name: "m", Type: "counter", Value: 1}}, project); err != nil {
		t.Fatal(err)
	}
}

func countRows(t *testing.T, db *sql.DB, table, where string, args ...any) int {
	t.Helper()
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM `+table+` `+where, args...).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func clear(t *testing.T, mux *http.ServeMux, query string) {
	t.Helper()
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest("DELETE", "/api/sentry/all"+query, nil))
	if w.Code != http.StatusOK {
		t.Fatalf("clear %q: %d %s", query, w.Code, w.Body.String())
	}
}

var sentryTables = []string{
	"sentry_error_events", "sentry_exceptions", "sentry_breadcrumbs",
	"sentry_transactions", "sentry_spans", "sentry_logs", "sentry_metrics",
}

// Clearing one project used to wipe the Sentry tables of every project.
func TestClearSentry_OneProject(t *testing.T) {
	db, mux := clearTestDB(t)
	seedProject(t, db, "shop", "e-shop", "trace-shop")
	seedProject(t, db, "billing", "e-billing", "trace-billing")
	// A distributed trace: both projects have spans in it.
	seedProject(t, db, "shop", "e-shared-shop", "trace-shared")
	seedProject(t, db, "billing", "e-shared-billing", "trace-shared")

	clear(t, mux, "?project=shop")

	for _, table := range sentryTables {
		var where string
		switch table {
		case "sentry_exceptions", "sentry_breadcrumbs":
			where = "WHERE error_event_id IN (SELECT id FROM sentry_error_events WHERE project_id = 'billing')"
			if got, all := countRows(t, db, table, where), countRows(t, db, table, ""); got == 0 || got != all {
				t.Errorf("%s: %d rows, %d of billing, want only billing's", table, all, got)
			}
			continue
		}
		if n := countRows(t, db, table, "WHERE project_id = 'shop'"); n != 0 {
			t.Errorf("%s: %d shop rows left", table, n)
		}
		if n := countRows(t, db, table, "WHERE project_id = 'billing'"); n == 0 {
			t.Errorf("%s: billing rows deleted", table)
		}
	}

	if n := countRows(t, db, "sentry_traces", "WHERE trace_id = 'trace-shop'"); n != 0 {
		t.Error("trace of shop only is left")
	}
	var spans, errs int
	db.QueryRow(`SELECT span_count, error_count FROM sentry_traces WHERE trace_id = 'trace-shared'`).Scan(&spans, &errs)
	if spans != 3 || errs != 1 {
		t.Errorf("shared trace span_count %d error_count %d, want billing's 3 and 1", spans, errs)
	}
	if n := countRows(t, db, "sentry_traces", "WHERE trace_id = 'trace-billing'"); n != 1 {
		t.Error("billing trace deleted")
	}
}

// Without parameters the call wipes everything, as before.
func TestClearSentry_NoParamsWipesAll(t *testing.T) {
	db, mux := clearTestDB(t)
	seedProject(t, db, "shop", "e-shop", "trace-shop")
	seedProject(t, db, "", "e-default", "trace-default")

	clear(t, mux, "")
	for _, table := range append(sentryTables, "sentry_traces") {
		if n := countRows(t, db, table, ""); n != 0 {
			t.Errorf("%s: %d rows left", table, n)
		}
	}
}

// keep_pinned keeps the errors whose event is pinned, with their exceptions
// and breadcrumbs; the rest of the project goes.
func TestClearSentry_KeepPinned(t *testing.T) {
	db, mux := clearTestDB(t)
	seedProject(t, db, "shop", "e-pinned", "trace-1")
	seedProject(t, db, "shop", "e-loose", "trace-2")
	seedProject(t, db, "billing", "e-billing", "trace-3")
	for _, u := range []string{"e-pinned", "e-loose", "e-billing"} {
		pinned := 0
		if u == "e-pinned" {
			pinned = 1
		}
		if _, err := db.Exec(`INSERT INTO events (uuid, type, payload, timestamp, project, is_pinned) VALUES (?, 'sentry', '{}', '1', 'shop', ?)`, u, pinned); err != nil {
			t.Fatal(err)
		}
	}

	clear(t, mux, "?project=shop&keep_pinned=1")

	if n := countRows(t, db, "sentry_error_events", "WHERE event_id = 'e-pinned'"); n != 1 {
		t.Error("pinned error deleted")
	}
	if n := countRows(t, db, "sentry_error_events", "WHERE event_id = 'e-loose'"); n != 0 {
		t.Error("unpinned error kept")
	}
	if n := countRows(t, db, "sentry_error_events", "WHERE event_id = 'e-billing'"); n != 1 {
		t.Error("other project's error deleted")
	}
	pinnedChildren := "WHERE error_event_id IN (SELECT id FROM sentry_error_events WHERE event_id = 'e-pinned')"
	if countRows(t, db, "sentry_exceptions", pinnedChildren) != 1 || countRows(t, db, "sentry_breadcrumbs", pinnedChildren) != 1 {
		t.Error("pinned error lost its exception or breadcrumb")
	}
	for _, table := range []string{"sentry_transactions", "sentry_spans", "sentry_logs", "sentry_metrics"} {
		if n := countRows(t, db, table, "WHERE project_id = 'shop'"); n != 0 {
			t.Errorf("%s: %d shop rows left", table, n)
		}
	}
}

// Rows stored before the project column existed belong to "default".
func TestClearSentry_MigrationPutsOldRowsInDefault(t *testing.T) {
	db, err := storage.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })

	const newest = "2024_01_01_000030_project_for_sentry_tables.sql"
	entries, _ := migrations.ReadDir("migrations")
	old := storage.NewMigrator(db)
	for _, e := range entries {
		if e.Name() == newest {
			continue
		}
		b, _ := migrations.ReadFile("migrations/" + e.Name())
		old.AddSQL("sentry", e.Name(), string(b))
	}
	if err := old.Run(); err != nil {
		t.Fatal(err)
	}
	for _, q := range []string{
		`INSERT INTO sentry_logs (id, level, body) VALUES ('old', 'info', 'x')`,
		`INSERT INTO sentry_metrics (id, name, type, value) VALUES ('old', 'm', 'counter', 1)`,
		`INSERT INTO sentry_transactions (id, event_id, trace_id, transaction_name, payload) VALUES ('old', 'e', 't', 'GET /', '{}')`,
		`INSERT INTO sentry_spans (id, span_id, trace_id) VALUES ('old', 's', 't')`,
	} {
		if _, err := db.Exec(q); err != nil {
			t.Fatal(err)
		}
	}

	all := storage.NewMigrator(db)
	if err := all.AddFromFS("sentry", migrations, "migrations"); err != nil {
		t.Fatal(err)
	}
	if err := all.Run(); err != nil {
		t.Fatal(err)
	}
	for _, table := range []string{"sentry_logs", "sentry_metrics", "sentry_transactions", "sentry_spans"} {
		if countRows(t, db, table, "WHERE id = 'old' AND project_id = 'default'") != 1 {
			t.Errorf("%s: old row not in default", table)
		}
	}
}
