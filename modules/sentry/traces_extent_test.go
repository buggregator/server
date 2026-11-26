package sentry

import (
	"encoding/json"
	"testing"
)

// A request to shop-api (200 ms) queues a job that shop-worker runs after
// the response (150 → 530 ms): the list row spans the whole trace, as its
// preview does (frontend#390, as in Trap).
func TestTracesList_DurationCoversWholeTrace(t *testing.T) {
	db, _ := clearTestDB(t)
	store := func(project string, txn map[string]any) {
		t.Helper()
		payload, _ := json.Marshal(txn)
		var tx Transaction
		if err := json.Unmarshal(payload, &tx); err != nil {
			t.Fatal(err)
		}
		if _, err := storeTransaction(db, &tx, payload, project); err != nil {
			t.Fatal(err)
		}
	}
	const trace = "trace-shop"
	store("shop", map[string]any{
		"event_id": "txn-api", "type": "transaction", "transaction": "POST /orders",
		"start_timestamp": 1000.0, "timestamp": 1000.2,
		"contexts": map[string]any{"trace": map[string]any{"trace_id": trace, "span_id": "api-root"}},
		"spans": []map[string]any{
			{"span_id": "api-db", "parent_span_id": "api-root", "trace_id": trace, "op": "db.query", "start_timestamp": 1000.01, "timestamp": 1000.05, "status": "ok"},
			{"span_id": "api-queue", "parent_span_id": "api-root", "trace_id": trace, "op": "queue.publish", "start_timestamp": 1000.06, "timestamp": 1000.07, "status": "ok"},
		},
	})
	store("worker", map[string]any{
		"event_id": "txn-worker", "type": "transaction", "transaction": "process-order",
		"start_timestamp": 1000.15, "timestamp": 1000.53,
		"contexts": map[string]any{"trace": map[string]any{"trace_id": trace, "span_id": "worker-root", "parent_span_id": "api-queue"}},
		"spans": []map[string]any{
			{"span_id": "worker-http", "parent_span_id": "worker-root", "trace_id": trace, "op": "http.client", "start_timestamp": 1000.2, "timestamp": 1000.52, "status": "internal_error"},
		},
	})

	tests := []struct {
		name              string
		query             string
		root, receivedAt  string
		duration          float64
		spans, errors     float64
		firstPreviewStart float64 // the longest preview span's offset
	}{
		{"whole trace", "", "POST /orders", "1970-01-01T00:16:40Z", 530, 3, 1, 200},
		{"shop only", "?project=shop", "POST /orders", "1970-01-01T00:16:40Z", 200, 2, 0, 10},
		{"worker only", "?project=worker", "process-order", "1970-01-01T00:16:40.15Z", 380, 1, 1, 50},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, body := apiGet(t, db, "/api/sentry/traces"+tt.query)
			rows := listData(t, body)
			if code != 200 || len(rows) != 1 {
				t.Fatalf("%d, %d rows: %v", code, len(rows), body)
			}
			row := rows[0]
			if row["transaction_name"] != tt.root {
				t.Errorf("root = %v, want %s", row["transaction_name"], tt.root)
			}
			if row["duration_ms"] != tt.duration {
				t.Errorf("duration_ms = %v, want %v", row["duration_ms"], tt.duration)
			}
			if row["span_count"] != tt.spans || row["error_count"] != tt.errors {
				t.Errorf("span_count %v error_count %v, want %v %v", row["span_count"], row["error_count"], tt.spans, tt.errors)
			}
			if got, _ := parseRFC3339(row["received_at"].(string)); got.UnixMilli() != mustTime(t, tt.receivedAt) {
				t.Errorf("received_at = %v, want the root start %s", row["received_at"], tt.receivedAt)
			}
			preview, _ := row["preview_spans"].([]any)
			if len(preview) == 0 {
				t.Fatal("no preview spans")
			}
			if first := preview[0].(map[string]any); first["start_offset_ms"] != tt.firstPreviewStart {
				t.Errorf("longest preview span offset = %v, want %v", first["start_offset_ms"], tt.firstPreviewStart)
			}
			for _, p := range preview {
				s := p.(map[string]any)
				start, dur := s["start_offset_ms"].(float64), s["duration_ms"].(float64)
				if start < 0 || start+dur > tt.duration {
					t.Errorf("preview span %v at %v+%v ms is outside [0, %v]", s["span_id"], start, dur, tt.duration)
				}
			}
		})
	}

	// The trace detail is unchanged: the root transaction's own duration.
	_, detail := apiGet(t, db, "/api/sentry/traces/"+trace)
	if d := detail["transaction"].(map[string]any)["duration_ms"]; d != float64(200) {
		t.Errorf("detail duration_ms = %v, want 200", d)
	}
}

func mustTime(t *testing.T, s string) int64 {
	t.Helper()
	v, err := parseRFC3339(s)
	if err != nil {
		t.Fatal(err)
	}
	return v.UnixMilli()
}
