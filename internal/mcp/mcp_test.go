package mcp_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/buggregator/go-buggregator/internal/event"
	mcpserver "github.com/buggregator/go-buggregator/internal/mcp"
	"github.com/buggregator/go-buggregator/internal/module"
	"github.com/buggregator/go-buggregator/internal/storage"
	"github.com/buggregator/go-buggregator/modules/profiler"
	"github.com/buggregator/go-buggregator/modules/sentry"
	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

func connectMCP(t *testing.T, db *sql.DB, store event.Store) *sdkmcp.ClientSession {
	t.Helper()
	server := mcpserver.NewServer(db, store)
	ct, st := sdkmcp.NewInMemoryTransports()
	ctx := context.Background()

	serverSession, err := server.Connect(ctx, st, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { serverSession.Close() })

	client := sdkmcp.NewClient(&sdkmcp.Implementation{Name: "test", Version: "1.0"}, nil)
	session, err := client.Connect(ctx, ct, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { session.Close() })

	return session
}

func setupDB(t *testing.T) (*sql.DB, *storage.SQLiteStore, *module.Registry) {
	t.Helper()
	db, err := storage.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })

	store := storage.NewSQLiteStore(db)
	mux := http.NewServeMux()
	reg := module.NewRegistry()
	reg.Register(profiler.New())
	reg.Register(sentry.New())
	if err := reg.Init(db, mux, store); err != nil {
		t.Fatal(err)
	}
	return db, store, reg
}

func ingestViaHandler(t *testing.T, reg *module.Registry, store event.Store, req *http.Request) string {
	t.Helper()
	for _, h := range reg.Handlers() {
		if h.Match(req) {
			incoming, err := h.Handle(req)
			if err != nil {
				t.Fatal(err)
			}
			ev := event.NewEvent(incoming)
			if err := store.Store(context.Background(), ev); err != nil {
				t.Fatal(err)
			}
			return ev.UUID
		}
	}
	t.Fatal("no handler matched")
	return ""
}

func callTool(t *testing.T, session *sdkmcp.ClientSession, name string, args map[string]any) string {
	t.Helper()
	result, err := session.CallTool(context.Background(), &sdkmcp.CallToolParams{
		Name:      name,
		Arguments: args,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatalf("tool %s returned error: %v", name, result.Content)
	}
	return result.Content[0].(*sdkmcp.TextContent).Text
}

func TestMCP_ListTools(t *testing.T) {
	db, store, _ := setupDB(t)
	session := connectMCP(t, db, store)

	result, err := session.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}

	expected := []string{
		"events_list", "event_get", "event_delete",
		"profiler_summary", "profiler_top", "profiler_call_graph",
		"sentry_event", "vardump_get",
	}
	found := make(map[string]bool)
	for _, tool := range result.Tools {
		found[tool.Name] = true
	}
	for _, name := range expected {
		if !found[name] {
			t.Errorf("missing tool: %s", name)
		}
	}
}

func TestMCP_EventsListEmpty(t *testing.T) {
	db, store, _ := setupDB(t)
	session := connectMCP(t, db, store)

	text := callTool(t, session, "events_list", map[string]any{"limit": 10})
	var events []any
	json.Unmarshal([]byte(text), &events)
	if len(events) != 0 {
		t.Errorf("expected 0 events, got %d", len(events))
	}
}

func TestMCP_ProfilerEndToEnd(t *testing.T) {
	db, store, reg := setupDB(t)

	// Ingest profiler event.
	body := `{"profile":{"main()":{"wt":10000,"cpu":5000,"mu":4096,"pmu":8192,"ct":1},"main()==>doWork()":{"wt":8000,"cpu":4000,"mu":2048,"pmu":4096,"ct":2},"doWork()==>db_query()":{"wt":6000,"cpu":3000,"mu":1024,"pmu":2048,"ct":5}},"app_name":"test-app","hostname":"test-host","date":1711800000}`
	req := httptest.NewRequest(http.MethodPost, "/", io.NopCloser(strings.NewReader(body)))
	req.Header.Set("X-Profiler-Dump", "1")
	req.Header.Set("Content-Type", "application/json")
	uuid := ingestViaHandler(t, reg, store, req)

	session := connectMCP(t, db, store)

	t.Run("events_list_contains_profiler", func(t *testing.T) {
		text := callTool(t, session, "events_list", map[string]any{"type": "profiler"})
		if !strings.Contains(text, uuid) {
			t.Errorf("events_list missing profiler uuid %s", uuid)
		}
	})

	t.Run("profiler_summary", func(t *testing.T) {
		text := callTool(t, session, "profiler_summary", map[string]any{"uuid": uuid})
		var summary map[string]any
		json.Unmarshal([]byte(text), &summary)

		if summary["profile_name"] != "test-app" {
			t.Errorf("expected profile_name=test-app, got %v", summary["profile_name"])
		}
		if summary["slowest_function"] == nil {
			t.Error("missing slowest_function")
		}
		if summary["memory_hotspot"] == nil {
			t.Error("missing memory_hotspot")
		}
		if summary["most_called"] == nil {
			t.Error("missing most_called")
		}
	})

	t.Run("profiler_top", func(t *testing.T) {
		text := callTool(t, session, "profiler_top", map[string]any{"uuid": uuid, "metric": "wt", "limit": 5})
		var top map[string]any
		json.Unmarshal([]byte(text), &top)

		if top["sorted_by"] != "wt" {
			t.Errorf("expected sorted_by=wt, got %v", top["sorted_by"])
		}
		funcs := top["functions"].([]any)
		if len(funcs) == 0 {
			t.Error("expected functions")
		}
		// First function should have highest wall time.
		first := funcs[0].(map[string]any)
		if first["function"] == nil {
			t.Error("missing function name")
		}
	})

	t.Run("profiler_call_graph", func(t *testing.T) {
		text := callTool(t, session, "profiler_call_graph", map[string]any{"uuid": uuid, "metric": "cpu", "percentage": 1})
		var graph map[string]any
		json.Unmarshal([]byte(text), &graph)

		nodeCount := int(graph["node_count"].(float64))
		edgeCount := int(graph["edge_count"].(float64))
		if nodeCount < 3 {
			t.Errorf("expected at least 3 nodes, got %d", nodeCount)
		}
		if edgeCount < 2 {
			t.Errorf("expected at least 2 edges, got %d", edgeCount)
		}

		// Verify nodes contain function names.
		nodes := graph["nodes"].([]any)
		funcNames := make(map[string]bool)
		for _, n := range nodes {
			node := n.(map[string]any)
			funcNames[node["function"].(string)] = true
		}
		for _, name := range []string{"main()", "doWork()", "db_query()"} {
			if !funcNames[name] {
				t.Errorf("missing node: %s", name)
			}
		}
	})
}

func TestMCP_SentryEndToEnd(t *testing.T) {
	db, store, reg := setupDB(t)

	// Ingest sentry event.
	body := `{"event_id":"sentry-test-001","message":"Something broke","level":"error","platform":"php","environment":"staging","exception":{"values":[{"type":"RuntimeException","value":"Division by zero","stacktrace":{"frames":[{"filename":"app/Math.php","function":"divide","lineno":15},{"filename":"app/Controller.php","function":"calculate","lineno":42}]}}]}}`
	req := httptest.NewRequest(http.MethodPost, "/api/default/store", io.NopCloser(strings.NewReader(body)))
	req.Header.Set("Content-Type", "application/json")
	uuid := ingestViaHandler(t, reg, store, req)

	session := connectMCP(t, db, store)

	t.Run("sentry_event", func(t *testing.T) {
		text := callTool(t, session, "sentry_event", map[string]any{"uuid": uuid})
		var ev map[string]any
		json.Unmarshal([]byte(text), &ev)

		if ev["level"] != "error" {
			t.Errorf("expected level=error, got %v", ev["level"])
		}
		if ev["platform"] != "php" {
			t.Errorf("expected platform=php, got %v", ev["platform"])
		}
		if ev["environment"] != "staging" {
			t.Errorf("expected environment=staging, got %v", ev["environment"])
		}

		exceptions := ev["exceptions"].([]any)
		exc := exceptions[0].(map[string]any)
		if exc["type"] != "RuntimeException" {
			t.Errorf("expected RuntimeException, got %v", exc["type"])
		}
		if exc["value"] != "Division by zero" {
			t.Errorf("expected 'Division by zero', got %v", exc["value"])
		}
		frames := exc["stacktrace"].([]any)
		if len(frames) != 2 {
			t.Fatalf("expected 2 frames, got %d", len(frames))
		}
	})

	t.Run("event_get", func(t *testing.T) {
		text := callTool(t, session, "event_get", map[string]any{"uuid": uuid})
		var ev map[string]any
		json.Unmarshal([]byte(text), &ev)
		if ev["type"] != "sentry" {
			t.Errorf("expected type=sentry, got %v", ev["type"])
		}
	})

	t.Run("event_delete", func(t *testing.T) {
		text := callTool(t, session, "event_delete", map[string]any{"uuid": uuid})
		if !strings.Contains(text, "deleted") {
			t.Errorf("expected delete confirmation, got %s", text)
		}

		// Verify gone.
		text = callTool(t, session, "events_list", map[string]any{"type": "sentry"})
		var events []any
		json.Unmarshal([]byte(text), &events)
		if len(events) != 0 {
			t.Errorf("expected 0 events after delete, got %d", len(events))
		}
	})
}
