//go:build e2e

package mcp_e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

var (
	buggregatorURL = env("BUGGREGATOR_URL", "http://localhost:8000")
	mcpURL         = env("MCP_URL", "http://localhost:8001")
	mcpAuthToken   = env("MCP_AUTH_TOKEN", "e2e-test-token")
)

// authTransport adds Bearer token to all requests.
type authTransport struct {
	token string
	base  http.RoundTripper
}

func (t *authTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("Authorization", "Bearer "+t.token)
	return t.base.RoundTrip(req)
}

func connectMCPClient(t *testing.T) *sdkmcp.ClientSession {
	t.Helper()

	httpClient := &http.Client{
		Transport: &authTransport{token: mcpAuthToken, base: http.DefaultTransport},
		Timeout:   30 * time.Second,
	}

	transport := &sdkmcp.StreamableClientTransport{
		Endpoint:   mcpURL,
		HTTPClient: httpClient,
	}

	client := sdkmcp.NewClient(&sdkmcp.Implementation{
		Name:    "e2e-test",
		Version: "1.0.0",
	}, nil)

	session, err := client.Connect(context.Background(), transport, nil)
	if err != nil {
		t.Fatalf("MCP connect failed: %v", err)
	}
	t.Cleanup(func() { session.Close() })
	return session
}

func sendHTTPEvent(t *testing.T, path string, headers map[string]string, body string) {
	t.Helper()
	url := buggregatorURL + path
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBufferString(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("HTTP event send failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("HTTP event returned status %d", resp.StatusCode)
	}
}

func clearEvents(t *testing.T) {
	t.Helper()
	req, _ := http.NewRequest(http.MethodDelete, buggregatorURL+"/api/events", nil)
	http.DefaultClient.Do(req)
}

func callTool(t *testing.T, session *sdkmcp.ClientSession, name string, args map[string]any) string {
	t.Helper()
	result, err := session.CallTool(context.Background(), &sdkmcp.CallToolParams{
		Name:      name,
		Arguments: args,
	})
	if err != nil {
		t.Fatalf("tool %s failed: %v", name, err)
	}
	if result.IsError {
		t.Fatalf("tool %s returned error", name)
	}
	return result.Content[0].(*sdkmcp.TextContent).Text
}

func TestE2E_MCP_ToolsList(t *testing.T) {
	session := connectMCPClient(t)

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
	t.Logf("Found %d tools", len(result.Tools))
}

func TestE2E_MCP_Profiler(t *testing.T) {
	clearEvents(t)
	session := connectMCPClient(t)

	// Send profiler event via HTTP.
	sendHTTPEvent(t, "/", map[string]string{"X-Profiler-Dump": "1"}, `{
		"profile": {
			"main()": {"wt": 50000, "cpu": 30000, "mu": 1048576, "pmu": 2097152, "ct": 1},
			"main()==>doWork()": {"wt": 40000, "cpu": 20000, "mu": 524288, "pmu": 1048576, "ct": 2},
			"doWork()==>db_query()": {"wt": 30000, "cpu": 15000, "mu": 131072, "pmu": 262144, "ct": 5}
		},
		"app_name": "e2e-test-app",
		"hostname": "e2e-host",
		"date": 1711800000
	}`)

	// Wait a moment for async processing.
	time.Sleep(500 * time.Millisecond)

	// Find profiler event via MCP.
	text := callTool(t, session, "events_list", map[string]any{"type": "profiler"})
	var events []map[string]any
	json.Unmarshal([]byte(text), &events)
	if len(events) == 0 {
		t.Fatal("no profiler events found via MCP")
	}
	uuid := events[0]["uuid"].(string)
	t.Logf("Profiler event UUID: %s", uuid)

	// Test profiler_summary.
	t.Run("summary", func(t *testing.T) {
		text := callTool(t, session, "profiler_summary", map[string]any{"uuid": uuid})
		var summary map[string]any
		json.Unmarshal([]byte(text), &summary)

		if summary["profile_name"] != "e2e-test-app" {
			t.Errorf("expected profile_name=e2e-test-app, got %v", summary["profile_name"])
		}
		if summary["slowest_function"] == nil {
			t.Error("missing slowest_function")
		}
		t.Logf("Summary: %d total functions", int(summary["total_functions"].(float64)))
	})

	// Test profiler_top with different metrics.
	t.Run("top_by_cpu", func(t *testing.T) {
		text := callTool(t, session, "profiler_top", map[string]any{"uuid": uuid, "metric": "cpu", "limit": 5})
		var top map[string]any
		json.Unmarshal([]byte(text), &top)
		if top["sorted_by"] != "cpu" {
			t.Errorf("expected sorted_by=cpu, got %v", top["sorted_by"])
		}
		funcs := top["functions"].([]any)
		t.Logf("Top %d functions by CPU", len(funcs))
	})

	t.Run("top_by_wt", func(t *testing.T) {
		text := callTool(t, session, "profiler_top", map[string]any{"uuid": uuid, "metric": "wt", "limit": 10})
		var top map[string]any
		json.Unmarshal([]byte(text), &top)
		if top["sorted_by"] != "wt" {
			t.Errorf("expected sorted_by=wt, got %v", top["sorted_by"])
		}
	})

	// Test profiler_call_graph.
	t.Run("call_graph", func(t *testing.T) {
		text := callTool(t, session, "profiler_call_graph", map[string]any{
			"uuid": uuid, "metric": "cpu", "percentage": 5,
		})
		var graph map[string]any
		json.Unmarshal([]byte(text), &graph)

		nodeCount := int(graph["node_count"].(float64))
		edgeCount := int(graph["edge_count"].(float64))
		if nodeCount < 3 {
			t.Errorf("expected >= 3 nodes, got %d", nodeCount)
		}
		if edgeCount < 2 {
			t.Errorf("expected >= 2 edges, got %d", edgeCount)
		}
		t.Logf("Call graph: %d nodes, %d edges", nodeCount, edgeCount)
	})
}

func TestE2E_MCP_Sentry(t *testing.T) {
	clearEvents(t)
	session := connectMCPClient(t)

	// Send sentry event via HTTP.
	sendHTTPEvent(t, "/api/default/store", nil, `{
		"event_id": "e2e-sentry-test-001",
		"message": "E2E test error",
		"level": "error",
		"platform": "php",
		"environment": "e2e-testing",
		"server_name": "e2e-host",
		"exception": {
			"values": [{
				"type": "RuntimeException",
				"value": "Something failed in e2e",
				"stacktrace": {
					"frames": [
						{"filename": "app/Service.php", "function": "process", "lineno": 100},
						{"filename": "app/Controller.php", "function": "handle", "lineno": 50}
					]
				}
			}]
		},
		"tags": {"test": "e2e"}
	}`)

	time.Sleep(500 * time.Millisecond)

	// Find sentry event via MCP.
	text := callTool(t, session, "events_list", map[string]any{"type": "sentry"})
	var events []map[string]any
	json.Unmarshal([]byte(text), &events)
	if len(events) == 0 {
		t.Fatal("no sentry events found via MCP")
	}
	uuid := events[0]["uuid"].(string)
	t.Logf("Sentry event UUID: %s", uuid)

	// Test sentry_event tool.
	t.Run("sentry_event", func(t *testing.T) {
		text := callTool(t, session, "sentry_event", map[string]any{"uuid": uuid})
		var ev map[string]any
		json.Unmarshal([]byte(text), &ev)

		if ev["level"] != "error" {
			t.Errorf("level: expected error, got %v", ev["level"])
		}
		if ev["platform"] != "php" {
			t.Errorf("platform: expected php, got %v", ev["platform"])
		}
		if ev["environment"] != "e2e-testing" {
			t.Errorf("environment: expected e2e-testing, got %v", ev["environment"])
		}

		exceptions := ev["exceptions"].([]any)
		if len(exceptions) != 1 {
			t.Fatalf("expected 1 exception, got %d", len(exceptions))
		}
		exc := exceptions[0].(map[string]any)
		if exc["type"] != "RuntimeException" {
			t.Errorf("exception type: expected RuntimeException, got %v", exc["type"])
		}
		frames := exc["stacktrace"].([]any)
		if len(frames) != 2 {
			t.Errorf("expected 2 frames, got %d", len(frames))
		}
		t.Logf("Sentry event: %s - %s", exc["type"], exc["value"])
	})

	// Test event_get returns full payload.
	t.Run("event_get", func(t *testing.T) {
		text := callTool(t, session, "event_get", map[string]any{"uuid": uuid})
		if !strings.Contains(text, "sentry") {
			t.Error("event_get should contain type=sentry")
		}
		if !strings.Contains(text, uuid) {
			t.Error("event_get should contain the UUID")
		}
	})

	// Test event_delete and verify cleanup.
	t.Run("event_delete_and_verify", func(t *testing.T) {
		text := callTool(t, session, "event_delete", map[string]any{"uuid": uuid})
		if !strings.Contains(text, "deleted") {
			t.Errorf("expected delete confirmation, got: %s", text)
		}

		text = callTool(t, session, "events_list", map[string]any{"type": "sentry"})
		var remaining []any
		json.Unmarshal([]byte(text), &remaining)
		if len(remaining) != 0 {
			t.Errorf("expected 0 events after delete, got %d", len(remaining))
		}
	})
}

func TestE2E_MCP_EventFiltering(t *testing.T) {
	clearEvents(t)
	session := connectMCPClient(t)

	// Send both profiler and sentry events.
	sendHTTPEvent(t, "/", map[string]string{"X-Profiler-Dump": "1"},
		`{"profile":{"main()":{"wt":1000,"cpu":500,"mu":1024,"pmu":2048,"ct":1}},"app_name":"filter-test","hostname":"h","date":1}`)
	sendHTTPEvent(t, "/api/default/store", nil,
		fmt.Sprintf(`{"event_id":"filter-sentry-%d","message":"filter test","level":"warning","platform":"go"}`, time.Now().UnixNano()))

	time.Sleep(500 * time.Millisecond)

	// events_list without filter should return both.
	text := callTool(t, session, "events_list", map[string]any{"limit": 50})
	var all []map[string]any
	json.Unmarshal([]byte(text), &all)
	if len(all) < 2 {
		t.Fatalf("expected at least 2 events, got %d", len(all))
	}

	// Filter by type=profiler.
	text = callTool(t, session, "events_list", map[string]any{"type": "profiler"})
	var profilers []map[string]any
	json.Unmarshal([]byte(text), &profilers)
	for _, ev := range profilers {
		if ev["type"] != "profiler" {
			t.Errorf("expected type=profiler, got %v", ev["type"])
		}
	}

	// Filter by type=sentry.
	text = callTool(t, session, "events_list", map[string]any{"type": "sentry"})
	var sentries []map[string]any
	json.Unmarshal([]byte(text), &sentries)
	for _, ev := range sentries {
		if ev["type"] != "sentry" {
			t.Errorf("expected type=sentry, got %v", ev["type"])
		}
	}

	t.Logf("Total: %d, Profilers: %d, Sentries: %d", len(all), len(profilers), len(sentries))
}
