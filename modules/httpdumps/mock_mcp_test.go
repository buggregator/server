package httpdumps

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

func connectMockTools(t *testing.T, env *mockEnv, write bool) *sdkmcp.ClientSession {
	t.Helper()
	env.mod.SetMCPMockWrite(write)
	server := sdkmcp.NewServer(&sdkmcp.Implementation{Name: "test", Version: "1"}, nil)
	env.mod.RegisterMCPTools(server)
	ct, st := sdkmcp.NewInMemoryTransports()
	ctx := context.Background()
	ss, err := server.Connect(ctx, st, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ss.Close() })
	cs, err := sdkmcp.NewClient(&sdkmcp.Implementation{Name: "c", Version: "1"}, nil).Connect(ctx, ct, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { cs.Close() })
	return cs
}

// mockCall calls a tool and decodes its JSON answer into out (when not nil).
// It returns the text and whether the tool reported an error.
func mockCall(t *testing.T, cs *sdkmcp.ClientSession, name string, args map[string]any, out any) (string, bool) {
	t.Helper()
	res, err := cs.CallTool(context.Background(), &sdkmcp.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	var sb strings.Builder
	for _, c := range res.Content {
		if tc, ok := c.(*sdkmcp.TextContent); ok {
			sb.WriteString(tc.Text)
		}
	}
	if out != nil && !res.IsError {
		if err := json.Unmarshal([]byte(sb.String()), out); err != nil {
			t.Fatalf("%s: %v in %s", name, err, sb.String())
		}
	}
	return sb.String(), res.IsError
}

func toolNames(t *testing.T, cs *sdkmcp.ClientSession) map[string]*sdkmcp.Tool {
	t.Helper()
	res, err := cs.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]*sdkmcp.Tool{}
	for _, tool := range res.Tools {
		out[tool.Name] = tool
	}
	return out
}

func TestMockMCP_Tools(t *testing.T) {
	env := newMockEnv(t, []MockRule{{Name: "yaml", Path: "/health"}})
	cs := connectMockTools(t, env, true)

	tools := toolNames(t, cs)
	for _, name := range []string{"mock_rules_list", "mock_rule_get", "mock_rule_preview", "mock_requests_list", "mock_template_helpers",
		"mock_rule_create", "mock_rule_update", "mock_rule_delete", "mock_rule_from_request"} {
		if tools[name] == nil {
			t.Errorf("tool %s missing", name)
		}
	}
	// The write and preview tools carry the whole helper reference.
	for _, name := range []string{"mock_rule_create", "mock_rule_preview"} {
		for _, h := range MockHelpers {
			if !strings.Contains(tools[name].Description, h.Usage) {
				t.Errorf("%s description lacks %s", name, h.Usage)
			}
		}
	}

	// Create a template rule: source mcp, visible in the REST API.
	var rule MockRule
	if text, isErr := mockCall(t, cs, "mock_rule_create", map[string]any{
		"name": "order", "method": "GET", "path": "/orders/{id}", "template": true, "seed": 3,
		"headers": map[string]string{"Content-Type": "application/json"},
		"body":    `{"id": {{ path "id" }}, "customer": {{ fake "name" | json }}}`,
	}, &rule); isErr {
		t.Fatalf("create: %s", text)
	}
	if rule.Source != "mcp" || !rule.Enabled || !rule.Template || rule.Seed != 3 {
		t.Errorf("created = %+v", rule)
	}
	if w := env.do(t, "GET", "/api/mocks/"+rule.ID, "", nil); !strings.Contains(w.Body.String(), `"source":"mcp"`) {
		t.Errorf("REST: %s", w.Body.String())
	}
	w := env.do(t, "GET", "/mock/orders/7", "", nil)
	var got map[string]any
	if json.Unmarshal(w.Body.Bytes(), &got); got["id"] != float64(7) || got["customer"] == "" {
		t.Errorf("served %d %s", w.Code, w.Body.String())
	}

	// A template error is refused with the message.
	if text, isErr := mockCall(t, cs, "mock_rule_create", map[string]any{"path": "/x", "template": true, "body": "{{ if }}"}, nil); !isErr || !strings.Contains(text, "body") {
		t.Errorf("bad template: %v %s", isErr, text)
	}

	// Preview: stored rule by id, with an override, and a new rule.
	var prev map[string]any
	mockCall(t, cs, "mock_rule_preview", map[string]any{"id": rule.ID, "request": map[string]any{"path": "/orders/9"}}, &prev)
	if !strings.Contains(prev["body"].(string), `"id": 9`) || prev["matched"] != true {
		t.Errorf("preview by id: %v", prev)
	}
	mockCall(t, cs, "mock_rule_preview", map[string]any{"path": "/q", "template": true, "body": `{{ query "a" }}-{{ fake "nope" }}`,
		"request": map[string]any{"path": "/q?a=1"}}, &prev)
	if prev["status"] != float64(500) || !strings.Contains(prev["error"].(string), "nope") {
		t.Errorf("preview error: %v", prev)
	}
	mockCall(t, cs, "mock_rule_preview", map[string]any{"path": "/users/{id}", "template": true, "body": `{{ path "id" }}`}, &prev)
	if prev["body"] != "id" || prev["request_path"] != "/users/id" {
		t.Errorf("preview with default request: %v", prev)
	}

	// Update: only the given fields change; position moves it.
	second := MockRule{}
	mockCall(t, cs, "mock_rule_create", map[string]any{"path": "/second"}, &second)
	var upd MockRule
	if text, isErr := mockCall(t, cs, "mock_rule_update", map[string]any{"id": second.ID, "status": 201, "position": 0}, &upd); isErr {
		t.Fatalf("update: %s", text)
	}
	if upd.Status != 201 || upd.Path != "/second" || upd.Source != "mcp" || upd.Position != 0 {
		t.Errorf("updated = %+v", upd)
	}
	var list struct {
		Rules []MockRule `json:"rules"`
	}
	mockCall(t, cs, "mock_rules_list", nil, &list)
	if len(list.Rules) != 3 || list.Rules[0].Source != "config" || list.Rules[1].ID != second.ID || list.Rules[2].ID != rule.ID {
		t.Errorf("list order = %+v", list.Rules)
	}
	// Config rules are read-only.
	if text, isErr := mockCall(t, cs, "mock_rule_update", map[string]any{"id": "config-1", "status": 500}, nil); !isErr || !strings.Contains(text, "read-only") {
		t.Errorf("config update: %v %s", isErr, text)
	}
	if _, isErr := mockCall(t, cs, "mock_rule_delete", map[string]any{"id": "config-1"}, nil); !isErr {
		t.Error("config rule deleted")
	}
	// The UI editing an agent's rule keeps its source.
	if w := env.do(t, "PUT", "/api/mocks/"+second.ID, `{"path":"/second","status":202}`, nil); !strings.Contains(w.Body.String(), `"source":"mcp"`) {
		t.Errorf("PUT: %s", w.Body.String())
	}
	if _, isErr := mockCall(t, cs, "mock_rule_delete", map[string]any{"id": second.ID}, nil); isErr {
		t.Error("delete failed")
	}

	// A request no rule answers → listed as unmatched → a rule made from it.
	env.do(t, "POST", "/mock/payments/charge", `{"amount": 5}`, map[string]string{"X-Buggregator-Project": "shop"})
	var reqs struct {
		Requests []map[string]any `json:"requests"`
	}
	mockCall(t, cs, "mock_requests_list", map[string]any{"only_unmatched": true}, &reqs)
	if len(reqs.Requests) != 1 {
		t.Fatalf("unmatched = %v", reqs.Requests)
	}
	un := reqs.Requests[0]
	if r := un["request"].(map[string]any); r["method"] != "POST" || r["body"] != `{"amount": 5}` || un["project"] != "shop" {
		t.Errorf("unmatched request = %v", un)
	}
	var fromReq MockRule
	if text, isErr := mockCall(t, cs, "mock_rule_from_request", map[string]any{"request_uuid": un["uuid"], "status": 201, "body": `{"ok": true}`}, &fromReq); isErr {
		t.Fatalf("from request: %s", text)
	}
	if fromReq.Method != "POST" || fromReq.Path != "/payments/charge" || fromReq.Project != "shop" || fromReq.Status != 201 || fromReq.Source != "mcp" {
		t.Errorf("from request = %+v", fromReq)
	}
	if w := env.do(t, "POST", "/mock/payments/charge", "", map[string]string{"X-Buggregator-Project": "shop"}); w.Code != 201 {
		t.Errorf("rule from request answers %d", w.Code)
	}
	mockCall(t, cs, "mock_requests_list", map[string]any{"rule_id": fromReq.ID}, &reqs)
	if len(reqs.Requests) != 1 || reqs.Requests[0]["mock"].(map[string]any)["matched"] != true {
		t.Errorf("by rule = %v", reqs.Requests)
	}

	// Helpers reference with a fake filter.
	var helpers struct {
		Helpers    []MockHelper `json:"helpers"`
		Fake       []MockFake   `json:"fake"`
		Categories []string     `json:"fake_categories"`
	}
	mockCall(t, cs, "mock_template_helpers", map[string]any{"search": "email"}, &helpers)
	if len(helpers.Helpers) != len(MockHelpers) || len(helpers.Fake) == 0 || len(helpers.Categories) < 5 {
		t.Errorf("helpers: %d helpers, %d fakes, %d categories", len(helpers.Helpers), len(helpers.Fake), len(helpers.Categories))
	}
	for _, f := range helpers.Fake {
		if !strings.Contains(strings.ToLower(f.Name+" "+f.Description), "email") {
			t.Errorf("fake %s does not match the search", f.Name)
		}
	}
}

func TestMockMCP_ReadOnly(t *testing.T) {
	env := newMockEnv(t, nil)
	tools := toolNames(t, connectMockTools(t, env, false))
	for _, name := range []string{"mock_rule_create", "mock_rule_update", "mock_rule_delete", "mock_rule_from_request"} {
		if tools[name] != nil {
			t.Errorf("%s exists with mock_write off", name)
		}
	}
	for _, name := range []string{"mock_rules_list", "mock_rule_get", "mock_rule_preview", "mock_requests_list", "mock_template_helpers"} {
		if tools[name] == nil {
			t.Errorf("%s missing with mock_write off", name)
		}
	}
}

func TestMockStore_RuleLimit(t *testing.T) {
	env := newMockEnv(t, nil)
	// Fill the project with SQL: 500 creates reload the cache 500 times.
	for i := 0; i < MaxMockRulesPerProject; i++ {
		if _, err := env.mod.db.Exec(`INSERT INTO mock_rules (id, project, name, method, path, status, headers, body, delay_ms, enabled, position, created_at, updated_at)
			VALUES (?, 'shop', '', 'ANY', '/a', 200, '{}', '', 0, 1, ?, '', '')`, fmt.Sprintf("r%d", i), i); err != nil {
			t.Fatalf("rule %d: %v", i, err)
		}
	}
	w := env.do(t, "POST", "/api/mocks", `{"path":"/a","project":"shop"}`, nil)
	if w.Code != http.StatusUnprocessableEntity || !strings.Contains(w.Body.String(), "rules per project") {
		t.Errorf("rule 501: %d %s", w.Code, w.Body.String())
	}
	// Another project still has room.
	env.create(t, `{"path":"/a","project":"billing"}`)
}
