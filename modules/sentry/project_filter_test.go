package sentry

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// Traces, logs and metrics carry a project since migration 000030, but their
// reads ignored ?project: every project's page showed all of them.
func TestSentryReads_FilterByProject(t *testing.T) {
	db, mux := clearTestDB(t)
	seedProject(t, db, "shop", "e-shop", "trace-shared")
	seedProject(t, db, "billing", "e-billing", "trace-shared")
	seedProject(t, db, "billing", "e-billing-2", "trace-billing")

	// Spans with a peer, recent enough for the service map's window.
	now := float64(time.Now().Unix())
	for _, p := range []string{"shop", "billing", "billing"} {
		var span RawSpan
		json.Unmarshal([]byte(fmt.Sprintf(`{"span_id":"db-%s","trace_id":"trace-shared","op":"db.query",
			"start_timestamp":%f,"timestamp":%f,"data":{"server.address":"mysql-%s"}}`, p, now-1, now, p)), &span)
		if err := storeSpansV2(db, []RawSpan{span}, nil, p); err != nil {
			t.Fatal(err)
		}
	}

	get := func(t *testing.T, path string) (int, map[string]any) {
		t.Helper()
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, httptest.NewRequest("GET", path, nil))
		var body map[string]any
		json.Unmarshal(w.Body.Bytes(), &body)
		return w.Code, body
	}
	total := func(t *testing.T, path string) int {
		t.Helper()
		code, body := get(t, path)
		if code != http.StatusOK {
			t.Fatalf("%s: %d", path, code)
		}
		data, _ := body["data"].([]any)
		return len(data)
	}

	lists := []struct {
		path               string
		all, shop, billing int
	}{
		{"/api/sentry/traces", 3, 1, 2},
		{"/api/sentry/logs", 3, 1, 2},
		{"/api/sentry/metrics", 3, 1, 2},
	}
	for _, tt := range lists {
		t.Run(tt.path, func(t *testing.T) {
			for q, want := range map[string]int{"": tt.all, "?project=shop": tt.shop, "?project=billing": tt.billing, "?project=none": 0} {
				if got := total(t, tt.path+q); got != want {
					t.Errorf("%s%s: %d items, want %d", tt.path, q, got, want)
				}
			}
		})
	}

	t.Run("counts", func(t *testing.T) {
		for q, want := range map[string]float64{"": 3, "?project=shop": 1, "?project=billing": 2} {
			_, body := get(t, "/api/sentry/counts"+q)
			for _, k := range []string{"exceptions", "traces", "logs", "metrics"} {
				if body[k] != want {
					t.Errorf("counts%s %s = %v, want %v", q, k, body[k], want)
				}
			}
		}
	})

	t.Run("trace detail", func(t *testing.T) {
		spans := func(q string) int {
			code, body := get(t, "/api/sentry/traces/trace-shared"+q)
			if code != http.StatusOK {
				t.Fatalf("trace-shared%s: %d", q, code)
			}
			s, _ := body["spans"].([]any)
			return len(s)
		}
		// seedProject: 2 transaction spans + 1 v2 span; db spans: shop 1, billing 2.
		if got := spans(""); got != 9 {
			t.Errorf("all spans = %d, want 9", got)
		}
		if got := spans("?project=shop"); got != 4 {
			t.Errorf("shop spans = %d, want 4", got)
		}
		if code, _ := get(t, "/api/sentry/traces/trace-billing?project=shop"); code != http.StatusNotFound {
			t.Errorf("billing-only trace for shop: %d, want 404", code)
		}
	})

	t.Run("traces list preview spans", func(t *testing.T) {
		_, body := get(t, "/api/sentry/traces?project=shop")
		data, _ := body["data"].([]any)
		if len(data) != 1 {
			t.Fatalf("shop traces = %d", len(data))
		}
		for _, s := range data[0].(map[string]any)["preview_spans"].([]any) {
			if id := s.(map[string]any)["span_id"].(string); id == "a-e-billing" || id == "b-e-billing" || id == "v2-e-billing" || id == "db-billing" {
				t.Errorf("shop preview has billing span %s", id)
			}
		}
	})

	t.Run("service map", func(t *testing.T) {
		targets := func(q string) map[string]bool {
			_, body := get(t, "/api/sentry/service-map"+q)
			out := map[string]bool{}
			edges, _ := body["edges"].([]any)
			for _, e := range edges {
				out[e.(map[string]any)["target"].(string)] = true
			}
			return out
		}
		if got := targets(""); !got["mysql-shop"] || !got["mysql-billing"] {
			t.Errorf("all edges = %v", got)
		}
		if got := targets("?project=shop"); !got["mysql-shop"] || got["mysql-billing"] {
			t.Errorf("shop edges = %v, want only mysql-shop", got)
		}
	})
}
