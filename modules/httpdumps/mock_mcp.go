package httpdumps

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

// MCP tools for the mock server (#4493): agents list, write and preview rules
// and read the requests the mock server got. They call the same store as the
// REST API, so the UI shows the agent's rules at once (source "mcp").

// RegisterMCPTools adds the mock_* tools. The write tools exist only when
// mcp.mock_write is on (the default).
func (m *Module) RegisterMCPTools(server *sdkmcp.Server) {
	registerMockTools(server, m.mocks, m, m.mcpMockWrite)
}

// SetMCPMockWrite turns the MCP write tools (create, update, delete, from
// request) on or off. Default: on.
func (m *Module) SetMCPMockWrite(on bool) { m.mcpMockWrite = on }

// mockRequestReader reads captured mock requests (the module's db).
type mockRequestReader interface {
	mockRequests(project, ruleID string, limit int) ([]map[string]any, error)
	mockRequest(uuid string) (map[string]any, error)
}

func (m *Module) mockRequests(project, ruleID string, limit int) ([]map[string]any, error) {
	return listMockRequests(m.db, project, ruleID, limit)
}

func (m *Module) mockRequest(uuid string) (map[string]any, error) {
	var payload string
	var project *string
	err := m.db.QueryRow(`SELECT payload, project FROM events WHERE uuid = ? AND type = 'http-dump'`, uuid).Scan(&payload, &project)
	if err != nil {
		return nil, fmt.Errorf("captured request %s not found", uuid)
	}
	out := map[string]any{"payload": json.RawMessage(payload), "project": ""}
	if project != nil {
		out["project"] = *project
	}
	return out, nil
}

// mockTemplateReference is the helper reference for tool descriptions, built
// from MockHelpers so it never drifts from the code.
func mockTemplateReference() string {
	var b strings.Builder
	b.WriteString(`TEMPLATES. With template=true the body and every header value are Go text/template sources, rendered per request. ` +
		`Write every string into a JSON body through | json (it quotes and escapes); numbers and bools as is. ` +
		`Conditions: {{ if eq (query "status") "paid" }}…{{ else }}…{{ end }}; variables: {{ $id := path "id" }}. ` +
		`Lists: {{ range $i := repeat 3 }}{{ if $i }},{{ end }}{…}{{ end }} — range works only over repeat, queryAll and body; define/template/block are refused. ` +
		`seed (not 0) makes fake data and pick the same on every request; 0 = new data each time. ` +
		`A syntax error fails create/update with the message; a runtime error makes the mock answer 500 {"error":"mock template: …"} and the captured request shows mock.template_error. ` +
		`Always check with mock_rule_preview before saving. Helpers:` + "\n")
	for _, h := range MockHelpers {
		fmt.Fprintf(&b, "- %s — %s Example: %s → %s\n", h.Usage, h.Description, h.Example, h.Result)
	}
	b.WriteString(`Fake generators for fake "name" args…: list them with mock_template_helpers (category or search filter); args fill the generator's params in order.`)
	return b.String()
}

type mockRuleFields struct {
	Project  *string           `json:"project,omitempty" jsonschema:"Project key the rule answers for; empty = every project. A mock request picks its project from the X-Buggregator-Project header (else default)."`
	Name     *string           `json:"name,omitempty" jsonschema:"Short name shown in the UI"`
	Method   *string           `json:"method,omitempty" jsonschema:"GET, POST, PUT, PATCH, DELETE, HEAD, OPTIONS or ANY (default ANY)"`
	Path     *string           `json:"path,omitempty" jsonschema:"Path under /mock: exact segments, {name} for one segment, a trailing * for the rest. E.g. /users/{id} answers /mock/users/42"`
	Status   *int              `json:"status,omitempty" jsonschema:"HTTP status, default 200"`
	Headers  map[string]string `json:"headers,omitempty" jsonschema:"Response headers, e.g. {\"Content-Type\": \"application/json\"}. Values are templates when template=true"`
	Body     *string           `json:"body,omitempty" jsonschema:"Response body, at most 1 MB. A template when template=true"`
	Template *bool             `json:"template,omitempty" jsonschema:"true: body and header values are Go templates (see the tool description)"`
	Seed     *int64            `json:"seed,omitempty" jsonschema:"Not 0: the same fake data on every request"`
	DelayMs  *int              `json:"delay_ms,omitempty" jsonschema:"Delay before the answer, 0-30000 ms"`
	Enabled  *bool             `json:"enabled,omitempty" jsonschema:"Default true; a disabled rule never answers"`
}

// apply sets the given fields on r.
func (f mockRuleFields) apply(r *MockRule) {
	set := func(dst *string, v *string) {
		if v != nil {
			*dst = *v
		}
	}
	set(&r.Project, f.Project)
	set(&r.Name, f.Name)
	set(&r.Method, f.Method)
	set(&r.Path, f.Path)
	set(&r.Body, f.Body)
	if f.Status != nil {
		r.Status = *f.Status
	}
	if f.Headers != nil {
		r.Headers = f.Headers
	}
	if f.Template != nil {
		r.Template = *f.Template
	}
	if f.Seed != nil {
		r.Seed = *f.Seed
	}
	if f.DelayMs != nil {
		r.DelayMs = *f.DelayMs
	}
	if f.Enabled != nil {
		r.Enabled = *f.Enabled
	}
}

type mockIDInput struct {
	ID string `json:"id" jsonschema:"Rule id from mock_rules_list"`
}

type mockListInput struct {
	Project string `json:"project,omitempty" jsonschema:"Only rules that answer this project (its own and those for every project)"`
}

type mockUpdateInput struct {
	ID string `json:"id" jsonschema:"Rule id from mock_rules_list"`
	mockRuleFields
	Position *int `json:"position,omitempty" jsonschema:"New place among the stored rules, 0 = first (matched first after the config rules)"`
}

type mockPreviewRequest struct {
	Method  string            `json:"method,omitempty" jsonschema:"Default: the rule's method, GET for ANY"`
	Path    string            `json:"path,omitempty" jsonschema:"Request path with query string, e.g. /users/42?page=2 (the /mock prefix is optional). Default: the rule's path"`
	Headers map[string]string `json:"headers,omitempty"`
	Body    string            `json:"body,omitempty" jsonschema:"Request body, e.g. a JSON document for body \"a.b\""`
}

type mockPreviewToolInput struct {
	ID string `json:"id,omitempty" jsonschema:"Preview a stored rule; the fields below then override its values without saving"`
	mockRuleFields
	Request mockPreviewRequest `json:"request,omitempty" jsonschema:"The sample request to answer"`
}

type mockRequestsInput struct {
	Project       string `json:"project,omitempty" jsonschema:"Only requests of this project"`
	RuleID        string `json:"rule_id,omitempty" jsonschema:"Only requests answered by this rule"`
	OnlyUnmatched bool   `json:"only_unmatched,omitempty" jsonschema:"Only requests no rule answered (they got 404 and list near rules that almost matched)"`
	Limit         int    `json:"limit,omitempty" jsonschema:"Default 20, at most 200"`
}

type mockFromRequestInput struct {
	RequestUUID string `json:"request_uuid" jsonschema:"uuid of a captured request from mock_requests_list"`
	mockRuleFields
}

type mockHelpersInput struct {
	Category string `json:"category,omitempty" jsonschema:"Only fake generators of this category (e.g. person, address, payment, internet, datetime, number, string, word)"`
	Search   string `json:"search,omitempty" jsonschema:"Only fake generators whose name or description contains this text"`
}

func registerMockTools(server *sdkmcp.Server, rules *mockStore, requests mockRequestReader, write bool) {
	ref := mockTemplateReference()

	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name: "mock_rules_list",
		Description: "List the mock server's rules in matching order: config rules (source config, read-only) first, then stored ones by position (source api or mcp). " +
			"Requests to /mock/<path> are answered by the first enabled rule whose method, path and project match.",
	}, func(ctx context.Context, req *sdkmcp.CallToolRequest, in mockListInput) (*sdkmcp.CallToolResult, any, error) {
		out := []MockRule{}
		for _, r := range rules.list() {
			if in.Project == "" || r.Project == "" || r.Project == in.Project {
				out = append(out, r)
			}
		}
		return mockToolResult(map[string]any{"rules": out}), nil, nil
	})

	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        "mock_rule_get",
		Description: "One mock rule with its full body.",
	}, func(ctx context.Context, req *sdkmcp.CallToolRequest, in mockIDInput) (*sdkmcp.CallToolResult, any, error) {
		r, err := rules.get(in.ID)
		if err != nil {
			return nil, nil, err
		}
		return mockToolResult(r), nil, nil
	})

	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name: "mock_rule_preview",
		Description: "Render a mock rule for a sample request without saving anything: returns status, headers, body, matched (would this request hit the rule) and error (template runtime error). " +
			"Give id to preview a stored rule (other fields override it), or the rule fields alone for a new one. Invalid rules return the validation message.\n" + ref,
	}, func(ctx context.Context, req *sdkmcp.CallToolRequest, in mockPreviewToolInput) (*sdkmcp.CallToolResult, any, error) {
		var rule MockRule
		if in.ID != "" {
			r, err := rules.get(in.ID)
			if err != nil {
				return nil, nil, err
			}
			rule = r
		}
		in.apply(&rule)
		p := mockPreviewInput{Rule: rule}
		p.Request.Method, p.Request.Path, p.Request.Headers, p.Request.Body = in.Request.Method, in.Request.Path, in.Request.Headers, in.Request.Body
		if p.Request.Path == "" {
			p.Request.Path = strings.ReplaceAll(strings.ReplaceAll(rule.Path, "*", "x"), "{", "")
			p.Request.Path = strings.ReplaceAll(p.Request.Path, "}", "")
		}
		out, err := previewMock(p)
		if err != nil {
			return nil, nil, err
		}
		out["request_path"] = p.Request.Path
		return mockToolResult(out), nil, nil
	})

	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name: "mock_requests_list",
		Description: "Requests the mock server got, newest first: the request (method, uri, headers, query, body), whether a rule matched, which rule, and what was sent back (status, headers, body up to 64 KB, template_error). " +
			"An unmatched request got 404 and lists near rules that missed it by one condition (disabled, method or project).",
	}, func(ctx context.Context, req *sdkmcp.CallToolRequest, in mockRequestsInput) (*sdkmcp.CallToolResult, any, error) {
		limit := in.Limit
		if limit <= 0 {
			limit = 20
		}
		limit = min(limit, 200)
		fetch := limit
		if in.OnlyUnmatched {
			fetch = 500 // filtered below
		}
		list, err := requests.mockRequests(in.Project, in.RuleID, fetch)
		if err != nil {
			return nil, nil, err
		}
		out := []map[string]any{}
		for _, ev := range list {
			item := compactMockRequest(ev)
			if in.OnlyUnmatched {
				if mock, _ := item["mock"].(map[string]any); mock["matched"] == true {
					continue
				}
			}
			out = append(out, item)
			if len(out) == limit {
				break
			}
		}
		return mockToolResult(map[string]any{"requests": out}), nil, nil
	})

	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        "mock_template_helpers",
		Description: "Reference for mock response templates: every helper with usage and an example, and the fake data generators for fake \"name\" args… with their parameters (filter by category or search; ~250 in all).",
	}, func(ctx context.Context, req *sdkmcp.CallToolRequest, in mockHelpersInput) (*sdkmcp.CallToolResult, any, error) {
		fakes := []MockFake{}
		cats := map[string]bool{}
		search := strings.ToLower(in.Search)
		for _, f := range MockFakes() {
			cats[f.Category] = true
			if in.Category != "" && !strings.EqualFold(f.Category, in.Category) {
				continue
			}
			if search != "" && !strings.Contains(strings.ToLower(f.Name+" "+f.Description), search) {
				continue
			}
			fakes = append(fakes, f)
		}
		categories := make([]string, 0, len(cats))
		for c := range cats {
			categories = append(categories, c)
		}
		sort.Strings(categories)
		return mockToolResult(map[string]any{"helpers": MockHelpers, "fake": fakes, "fake_categories": categories}), nil, nil
	})

	if !write {
		return
	}

	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name: "mock_rule_create",
		Description: "Create a mock rule. Requests to /mock/<path> then get this response; the rule shows in the Buggregator UI at once. " +
			"path is required; defaults: method ANY, status 200, enabled. At most 500 rules per project.\n" + ref,
	}, func(ctx context.Context, req *sdkmcp.CallToolRequest, in mockRuleFields) (*sdkmcp.CallToolResult, any, error) {
		rule := MockRule{Enabled: true}
		in.apply(&rule)
		r, err := rules.create(rule, "mcp")
		if err != nil {
			return nil, nil, err
		}
		return mockToolResult(r), nil, nil
	})

	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name: "mock_rule_update",
		Description: "Change a stored mock rule: only the given fields change. position moves it among the stored rules (0 = first). " +
			"Config rules (source config) are read-only. Templates: see mock_rule_create.",
	}, func(ctx context.Context, req *sdkmcp.CallToolRequest, in mockUpdateInput) (*sdkmcp.CallToolResult, any, error) {
		old, err := rules.get(in.ID)
		if err != nil {
			return nil, nil, err
		}
		if old.ReadOnly {
			return nil, nil, errMockReadOnly
		}
		r := old
		in.apply(&r)
		if r, err = rules.update(in.ID, r); err != nil {
			return nil, nil, err
		}
		if in.Position != nil {
			if err := rules.move(in.ID, *in.Position); err != nil {
				return nil, nil, err
			}
			r, _ = rules.get(in.ID)
		}
		return mockToolResult(r), nil, nil
	})

	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        "mock_rule_delete",
		Description: "Delete a stored mock rule. Config rules (source config) are read-only.",
	}, func(ctx context.Context, req *sdkmcp.CallToolRequest, in mockIDInput) (*sdkmcp.CallToolResult, any, error) {
		if err := rules.delete(in.ID); err != nil {
			return nil, nil, err
		}
		return mockToolResult(map[string]any{"deleted": in.ID}), nil, nil
	})

	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name: "mock_rule_from_request",
		Description: "Create a rule from a captured mock request (e.g. one that got 404): method, path and project come from the request; give status, headers and body (other rule fields override them). " +
			"Templates: see mock_rule_create.",
	}, func(ctx context.Context, req *sdkmcp.CallToolRequest, in mockFromRequestInput) (*sdkmcp.CallToolResult, any, error) {
		ev, err := requests.mockRequest(in.RequestUUID)
		if err != nil {
			return nil, nil, err
		}
		var payload struct {
			Request struct {
				Method string `json:"method"`
				URI    string `json:"uri"`
			} `json:"request"`
			Mock map[string]any `json:"mock"`
		}
		raw, _ := ev["payload"].(json.RawMessage)
		if err := json.Unmarshal(raw, &payload); err != nil || payload.Mock == nil {
			return nil, nil, errors.New("not a request captured by the mock server")
		}
		path := strings.TrimPrefix("/"+strings.TrimPrefix(payload.Request.URI, "/"), MockPrefix)
		if path == "" {
			path = "/"
		}
		rule := MockRule{Method: payload.Request.Method, Path: path, Enabled: true}
		rule.Project, _ = ev["project"].(string)
		in.apply(&rule)
		r, err := rules.create(rule, "mcp")
		if err != nil {
			return nil, nil, err
		}
		return mockToolResult(r), nil, nil
	})

	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name: "mock_import_openapi",
		Description: "Import an OpenAPI 3.0/3.1 spec (YAML or JSON; text in spec, or url to download, http/https, at most 5 MB, 10 s): one rule per operation, in one group. " +
			"Status = the lowest 2xx response; body = the spec's example as is, else a template built from the schema with fake data (edit it afterwards like any rule). " +
			"Path params: a root field named like a param (or id for the last one) echoes it. Paths get servers[0].url's path in front unless base_path is given (\"\" = none). " +
			"Re-importing the same group replaces its rules. Local $refs only; oneOf/anyOf take the first option. Returns the rules and the skipped operations with reasons.",
	}, func(ctx context.Context, req *sdkmcp.CallToolRequest, in mockImportInput) (*sdkmcp.CallToolResult, any, error) {
		res, err := importOpenAPI(ctx, rules, OpenAPIImport(in), "mcp")
		if err != nil {
			return nil, nil, err
		}
		summary := make([]map[string]any, len(res.Rules))
		for i, r := range res.Rules {
			summary[i] = map[string]any{"id": r.ID, "name": r.Name, "method": r.Method, "path": r.Path, "status": r.Status, "template": r.Template}
		}
		return mockToolResult(map[string]any{"group": res.Group, "title": res.Title, "rules": summary, "skipped": res.Skipped}), nil, nil
	})

	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        "mock_group_delete",
		Description: "Delete every stored rule of a group (the group of an OpenAPI import, see the group field of mock_rules_list).",
	}, func(ctx context.Context, req *sdkmcp.CallToolRequest, in mockGroupInput) (*sdkmcp.CallToolResult, any, error) {
		n, err := rules.deleteGroup(in.Group)
		if err != nil {
			return nil, nil, err
		}
		return mockToolResult(map[string]any{"group": in.Group, "deleted": n}), nil, nil
	})
}

type mockImportInput struct {
	Spec     string  `json:"spec,omitempty" jsonschema:"The spec text, YAML or JSON"`
	URL      string  `json:"url,omitempty" jsonschema:"Or where to download the spec (http/https)"`
	Project  string  `json:"project,omitempty" jsonschema:"Project of the rules; empty = every project"`
	Group    string  `json:"group,omitempty" jsonschema:"Group name; default openapi:<info.title>, or openapi:<project>:<info.title> with a project"`
	BasePath *string `json:"base_path,omitempty" jsonschema:"Prefix for every path; default the path of servers[0].url, empty string = none"`
}

type mockGroupInput struct {
	Group string `json:"group" jsonschema:"Group name, e.g. openapi:Petstore"`
}

// compactMockRequest keeps what an agent needs of a captured request.
func compactMockRequest(ev map[string]any) map[string]any {
	var payload struct {
		ReceivedAt string         `json:"received_at"`
		Request    map[string]any `json:"request"`
		Mock       map[string]any `json:"mock"`
	}
	raw, _ := ev["payload"].(json.RawMessage)
	_ = json.Unmarshal(raw, &payload)
	req := map[string]any{}
	for _, k := range []string{"method", "uri", "headers", "query", "body"} {
		req[k] = payload.Request[k]
	}
	return map[string]any{
		"uuid": ev["uuid"], "project": ev["project"], "received_at": payload.ReceivedAt,
		"request": req, "mock": payload.Mock,
	}
}

// mockToolResult renders v as indented JSON; no HTML escaping, so templates
// and bodies read as written.
func mockToolResult(v any) *sdkmcp.CallToolResult {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	_ = enc.Encode(v)
	return &sdkmcp.CallToolResult{Content: []sdkmcp.Content{&sdkmcp.TextContent{Text: strings.TrimRight(buf.String(), "\n")}}}
}
