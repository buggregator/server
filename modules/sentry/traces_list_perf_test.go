package sentry

import (
	"database/sql"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// The traces list picks each trace's root from parent_span_id (written at
// store time), never by reading payloads: with 3000 transactions of 30 KB
// it must stay fast. Review of #460 measured 320 ms when the root came from
// json_extract over every payload.
func TestTracesList_LargePayloadsStayFast(t *testing.T) {
	if testing.Short() {
		t.Skip("timing probe")
	}
	db := setupTestDB(t)
	pad := strings.Repeat("x", 30<<10)
	now := time.Now().UTC()

	tx, err := db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3000; i++ {
		traceID := fmt.Sprintf("%032x", i)
		payload := fmt.Sprintf(`{"contexts":{"trace":{"trace_id":%q,"span_id":"%016x"}},"pad":%q}`, traceID, i, pad)
		if _, err := tx.Exec(`INSERT INTO sentry_traces (trace_id, span_count) VALUES (?, 1)`, traceID); err != nil {
			t.Fatal(err)
		}
		if _, err := tx.Exec(`INSERT INTO sentry_transactions (id, event_id, trace_id, transaction_name, start_ts, payload, project_id)
			VALUES (?, ?, ?, 'GET /', ?, ?, 'default')`,
			fmt.Sprint("t", i), fmt.Sprint("e", i), traceID, now.Add(-time.Duration(i)*time.Second).Format(time.RFC3339Nano), payload); err != nil {
			t.Fatal(err)
		}
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}

	h := handleTracesList(db)
	best := time.Hour
	for run := 0; run < 3; run++ {
		rec := httptest.NewRecorder()
		start := time.Now()
		h(rec, httptest.NewRequest(http.MethodGet, "/api/sentry/traces?limit=50", nil))
		if d := time.Since(start); d < best {
			best = d
		}
		if rec.Code != 200 || !strings.Contains(rec.Body.String(), `"total":3000`) {
			t.Fatalf("list: %d %.200s", rec.Code, rec.Body.String())
		}
	}
	t.Logf("GET /api/sentry/traces?limit=50 over 3000 × 30 KB transactions: best of 3 = %v", best)
	if best > 100*time.Millisecond {
		t.Errorf("traces list took %v, want well under 100 ms", best)
	}
}

// Migration 000036 fills parent_span_id of transactions stored before it,
// from the payload: a Sentry transaction's contexts.trace, a segment's own.
func TestMigrationBackfillsParentSpanID(t *testing.T) {
	db := setupTestDB(t)
	rows := map[string]string{
		"root":    `{"contexts":{"trace":{"span_id":"a"}}}`,
		"child":   `{"contexts":{"trace":{"span_id":"b","parent_span_id":"a"}}}`,
		"segment": `{"span_id":"c","parent_span_id":"b"}`,
		"empty":   `{"span_id":"d","parent_span_id":""}`,
		"broken":  `not json`,
	}
	for id, payload := range rows {
		if _, err := db.Exec(`INSERT INTO sentry_transactions (id, event_id, trace_id, transaction_name, payload) VALUES (?, ?, 'tr', 'x', ?)`, id, id, payload); err != nil {
			t.Fatal(err)
		}
	}
	sqlText, err := migrations.ReadFile("migrations/2024_01_01_000036_otlp_source.sql")
	if err != nil {
		t.Fatal(err)
	}
	var update string
	for _, stmt := range strings.Split(string(sqlText), ";") {
		if i := strings.Index(stmt, "UPDATE sentry_transactions"); i >= 0 {
			update = stmt[i:]
		}
	}
	if _, err := db.Exec(update); err != nil {
		t.Fatal(err)
	}
	want := map[string]string{"root": "", "child": "a", "segment": "b", "empty": "", "broken": ""}
	for id, w := range want {
		var got sql.NullString
		db.QueryRow(`SELECT parent_span_id FROM sentry_transactions WHERE id = ?`, id).Scan(&got)
		if got.String != w {
			t.Errorf("%s: parent_span_id %q, want %q", id, got.String, w)
		}
	}
}
