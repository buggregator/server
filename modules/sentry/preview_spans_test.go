package sentry

import (
	"database/sql"
	"testing"
)

// A span without op (SDKs may omit it) still shows in the preview.
func TestLoadPreviewSpansNullOp(t *testing.T) {
	db := setupTestDB(t)
	for _, q := range []string{
		`INSERT INTO sentry_spans (id, span_id, trace_id, op, duration_ms) VALUES ('a', 's-op', 't1', 'db.query', 20)`,
		`INSERT INTO sentry_spans (id, span_id, trace_id, op, duration_ms) VALUES ('b', 's-null', 't1', NULL, 10)`,
	} {
		if _, err := db.Exec(q); err != nil {
			t.Fatal(err)
		}
	}
	spans := loadPreviewSpans(db, "t1", sql.NullString{}, 3, "", nil)
	if len(spans) != 2 {
		t.Fatalf("spans = %v, want 2", spans)
	}
	if spans[1]["span_id"] != "s-null" || spans[1]["op"] != "" {
		t.Errorf("null-op span = %v, want span_id s-null, op \"\"", spans[1])
	}
}
