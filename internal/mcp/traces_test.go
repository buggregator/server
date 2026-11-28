package mcp_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

// TestMCP_SentryTraces replays the sentry-laravel N+1 and queue recordings
// and reads them through sentry_traces_list and sentry_trace_get.
func TestMCP_SentryTraces(t *testing.T) {
	db, store, reg := setupDB(t)
	replaySDK(t, reg, store, "laravel-n1")
	replaySDK(t, reg, store, "laravel-queue")
	session := connectMCP(t, db, store)

	const n1Trace = "919e328d8a5648e58a1c18208d769367"

	type row = traceListRow
	list := func(args map[string]any) []row {
		t.Helper()
		var res struct {
			Data []row `json:"data"`
		}
		if err := json.Unmarshal([]byte(callTool(t, session, "sentry_traces_list", args)), &res); err != nil {
			t.Fatal(err)
		}
		return res.Data
	}

	all := list(map[string]any{})
	if len(all) < 2 {
		t.Fatalf("all traces = %+v, want the N+1 request and the queue traces", all)
	}

	cases := []struct {
		name string
		args map[string]any
		want func([]row) bool
	}{
		{"n_plus_one", map[string]any{"n_plus_one": true}, func(r []row) bool {
			return len(r) == 1 && r[0].TraceID == n1Trace && r[0].NPlusOneGroups == 2
		}},
		{"name matches any case", map[string]any{"name": strings.ToUpper(nameOf(all, n1Trace))}, func(r []row) bool {
			return len(r) == 1 && r[0].TraceID == n1Trace
		}},
		{"name no match", map[string]any{"name": "no-such-route"}, func(r []row) bool { return len(r) == 0 }},
		{"min duration low keeps traces", map[string]any{"min_duration_ms": 1}, func(r []row) bool { return len(r) > 0 }},
		{"min duration too high", map[string]any{"min_duration_ms": 10_000_000}, func(r []row) bool { return len(r) == 0 }},
		{"limit", map[string]any{"limit": 1}, func(r []row) bool { return len(r) == 1 }},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := list(c.args); !c.want(got) {
				t.Errorf("got %+v", got)
			}
		})
	}

	var trace struct {
		Transaction struct {
			Name string `json:"transaction_name"`
		} `json:"transaction"`
		SpanCount int              `json:"span_count"`
		Spans     []map[string]any `json:"spans"`
		NPlusOne  []struct {
			Query          string   `json:"query"`
			Count          int      `json:"count"`
			ExampleSpanIDs []string `json:"example_span_ids"`
		} `json:"n_plus_one"`
	}
	text := callTool(t, session, "sentry_trace_get", map[string]any{"trace_id": n1Trace})
	if err := json.Unmarshal([]byte(text), &trace); err != nil {
		t.Fatal(err)
	}
	if trace.SpanCount < 60 || len(trace.Spans) == 0 {
		t.Errorf("span_count %d, top-level spans %d", trace.SpanCount, len(trace.Spans))
	}
	if len(trace.NPlusOne) != 2 {
		t.Fatalf("n_plus_one = %+v, want 2 groups", trace.NPlusOne)
	}
	for _, g := range trace.NPlusOne {
		if g.Count != 30 || len(g.ExampleSpanIDs) != 3 {
			t.Errorf("group %q: count %d, examples %d", g.Query, g.Count, len(g.ExampleSpanIDs))
		}
	}
	if strings.Contains(text, `"attributes"`) {
		t.Error("trace carries raw span attributes")
	}

	// max_spans cuts the tree, not the counts.
	var cut struct {
		SpanCount int `json:"span_count"`
		Truncated struct {
			Shown, Total int
		} `json:"spans_truncated"`
		NPlusOne []any `json:"n_plus_one"`
	}
	json.Unmarshal([]byte(callTool(t, session, "sentry_trace_get", map[string]any{"trace_id": n1Trace, "max_spans": 5})), &cut)
	if cut.Truncated.Shown != 5 || cut.Truncated.Total != cut.SpanCount || len(cut.NPlusOne) != 2 {
		t.Errorf("max_spans 5: %+v", cut)
	}

	// A queue trace lists its job attempts.
	var queueTrace string
	for _, r := range all {
		if r.TraceID != n1Trace {
			text := callTool(t, session, "sentry_trace_get", map[string]any{"trace_id": r.TraceID})
			if strings.Contains(text, `"attempt"`) {
				queueTrace = r.TraceID
				break
			}
		}
	}
	if queueTrace == "" {
		t.Error("no queue trace lists queue_jobs attempts")
	}

	res, err := session.CallTool(context.Background(), &sdkmcp.CallToolParams{
		Name: "sentry_trace_get", Arguments: map[string]any{"trace_id": "nope"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !res.IsError {
		t.Error("unknown trace: want a tool error")
	}
}

type traceListRow struct {
	TraceID        string `json:"trace_id"`
	Name           string `json:"transaction_name"`
	NPlusOneGroups int    `json:"n_plus_one_groups"`
}

func nameOf(rows []traceListRow, id string) string {
	for _, r := range rows {
		if r.TraceID == id {
			return r.Name
		}
	}
	return ""
}
