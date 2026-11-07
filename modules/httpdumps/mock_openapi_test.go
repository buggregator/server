package httpdumps

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"
)

const petSpec = `openapi: 3.0.3
info:
  title: Shop API
  version: "1.0"
servers:
  - url: https://api.example.com/v1
paths:
  /users/{id}:
    get:
      operationId: getUser
      responses:
        "404":
          description: not found
        "200":
          description: ok
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/User"
    delete:
      responses:
        "204":
          description: deleted
  /users:
    post:
      summary: Create user
      responses:
        "201":
          description: created
          content:
            application/json:
              example: {"id": 7, "email": "new@example.com", "note": "{{ not a template }}"}
  /orders:
    get:
      responses:
        2XX:
          description: ok
          content:
            application/json:
              examples:
                first:
                  $ref: "#/components/examples/Orders"
  /files/{name}.json:
    get:
      responses:
        "200":
          description: ok
  /health:
    get:
      responses:
        default:
          description: ok
          content:
            text/plain:
              schema: {type: string}
components:
  examples:
    Orders:
      value: [{"id": 1, "total": 9.5}]
  schemas:
    Base:
      type: object
      properties:
        createdAt: {type: string, format: date-time}
    User:
      allOf:
        - $ref: "#/components/schemas/Base"
        - type: object
          properties:
            id: {type: integer, example: 1}
            email: {type: string, format: email}
            first_name: {type: string}
            role: {type: string, enum: [admin, user]}
            age: {type: integer, minimum: 18, maximum: 99}
            score: {type: number, minimum: 0, maximum: 5}
            active: {type: boolean}
            country: {type: string, example: NL}
            tags:
              type: array
              minItems: 3
              items: {type: string}
            meta: {type: object, additionalProperties: true}
            manager:
              $ref: "#/components/schemas/User"
`

func importSpec(t *testing.T, spec string, in OpenAPIImport) OpenAPIResult {
	t.Helper()
	res, err := buildOpenAPIRules([]byte(spec), in)
	if err != nil {
		t.Fatal(err)
	}
	return res
}

func ruleBy(t *testing.T, rules []MockRule, method, path string) MockRule {
	t.Helper()
	for _, r := range rules {
		if r.Method == method && r.Path == path {
			return r
		}
	}
	t.Fatalf("no rule %s %s in %+v", method, path, rules)
	return MockRule{}
}

func renderFor(t *testing.T, r MockRule, path string) string {
	t.Helper()
	out, err := renderMockRule(&r, &mockRequest{Method: r.Method, Path: path, Params: mockPathParams(r.Path, path), Header: http.Header{}})
	if err != nil {
		t.Fatalf("%s %s: %v\n%s", r.Method, r.Path, err, r.Body)
	}
	return string(out.Body)
}

func TestOpenAPI_Build(t *testing.T) {
	res := importSpec(t, petSpec, OpenAPIImport{Project: "shop"})
	if res.Group != "openapi:Shop API" || res.Title != "Shop API" || len(res.Rules) != 5 {
		t.Fatalf("result: %s %s %d rules, skipped %+v", res.Group, res.Title, len(res.Rules), res.Skipped)
	}
	if len(res.Skipped) != 1 || res.Skipped[0].Path != "/v1/files/{name}.json" {
		t.Errorf("skipped = %+v", res.Skipped)
	}

	// Schema → template: path param echoed, formats, enum, ranges, allOf,
	// array length, recursion cut, example kept.
	get := ruleBy(t, res.Rules, "GET", "/v1/users/{id}")
	if get.Name != "getUser" || get.Status != 200 || !get.Template || get.Project != "shop" || get.Headers["Content-Type"] != "application/json" {
		t.Errorf("get = %+v", get)
	}
	var user map[string]any
	body := renderFor(t, get, "/v1/users/42")
	if err := json.Unmarshal([]byte(body), &user); err != nil {
		t.Fatalf("not JSON: %v\n%s", err, body)
	}
	if user["id"] != float64(42) || !strings.Contains(user["email"].(string), "@") || user["country"] != "NL" {
		t.Errorf("user = %v", user)
	}
	if r := user["role"]; r != "admin" && r != "user" {
		t.Errorf("role = %v", r)
	}
	if a := user["age"].(float64); a < 18 || a > 99 {
		t.Errorf("age = %v", a)
	}
	if s := user["score"].(float64); s < 0 || s > 5 {
		t.Errorf("score = %v", s)
	}
	if _, ok := user["active"].(bool); !ok {
		t.Errorf("active = %v", user["active"])
	}
	if tags, _ := user["tags"].([]any); len(tags) != 3 {
		t.Errorf("tags = %v", user["tags"])
	}
	if _, ok := user["createdAt"].(string); !ok {
		t.Errorf("createdAt (from allOf) = %v", user["createdAt"])
	}
	if m, _ := user["meta"].(map[string]any); m == nil {
		t.Errorf("meta = %v", user["meta"])
	}
	depth := 0
	for m := user; m != nil; depth++ {
		m, _ = m["manager"].(map[string]any)
	}
	if depth < 2 || depth > maxSchemaDepth+1 {
		t.Errorf("manager chain depth %d", depth)
	}
	if !strings.HasPrefix(get.Body, "{\n  \"createdAt\"") {
		t.Errorf("body is not indented in spec order:\n%.80s", get.Body)
	}

	// Example → fixed body, "{{" in it stays text.
	post := ruleBy(t, res.Rules, "POST", "/v1/users")
	if post.Template || post.Status != 201 || post.Name != "Create user" || !strings.Contains(post.Body, `"{{ not a template }}"`) {
		t.Errorf("post = %+v", post)
	}
	orders := ruleBy(t, res.Rules, "GET", "/v1/orders")
	if orders.Template || orders.Status != 200 || !strings.Contains(orders.Body, `"total": 9.5`) {
		t.Errorf("orders = %+v", orders)
	}
	if del := ruleBy(t, res.Rules, "DELETE", "/v1/users/{id}"); del.Status != 204 || del.Body != "" || del.Name != "DELETE /v1/users/{id}" {
		t.Errorf("delete = %+v", del)
	}
	if health := ruleBy(t, res.Rules, "GET", "/v1/health"); !health.Template || health.Headers["Content-Type"] != "text/plain" || renderFor(t, health, "/v1/health") == "" {
		t.Errorf("health = %+v", health)
	}

	// Base path override.
	none := ""
	res = importSpec(t, petSpec, OpenAPIImport{BasePath: &none, Group: "g"})
	ruleBy(t, res.Rules, "GET", "/users/{id}")
	if res.Group != "g" {
		t.Errorf("group = %s", res.Group)
	}
}

func TestOpenAPI_31AndJSON(t *testing.T) {
	spec := `{"openapi": "3.1.0", "info": {"title": "T"}, "paths": {"/things/{thingId}": {"get": {"responses": {"200": {"content": {"application/json": {"schema": {
		"type": "object", "properties": {
			"thingId": {"type": "string"},
			"label": {"type": ["string", "null"]},
			"kind": {"const": "thing"},
			"size": {"type": "integer", "examples": [3]},
			"mixed": {"enum": [1, "a", null]},
			"choice": {"oneOf": [{"type": "boolean"}, {"type": "string"}]},
			"note": {"type": "string", "example": "{{ x }}"}
		}}}}}}}}}}`
	res := importSpec(t, spec, OpenAPIImport{})
	r := ruleBy(t, res.Rules, "GET", "/things/{thingId}")
	var got map[string]any
	body := renderFor(t, r, "/things/abc")
	if err := json.Unmarshal([]byte(body), &got); err != nil {
		t.Fatalf("not JSON: %v\n%s", err, body)
	}
	if got["thingId"] != "abc" || got["kind"] != "thing" || got["size"] != float64(3) || got["note"] != "{{ x }}" {
		t.Errorf("got = %v", got)
	}
	if _, ok := got["label"].(string); !ok {
		t.Errorf("label = %v", got["label"])
	}
	if _, ok := got["choice"].(bool); !ok {
		t.Errorf("choice = %v", got["choice"])
	}
	if m := got["mixed"]; m != float64(1) && m != "a" && m != nil {
		t.Errorf("mixed = %v", m)
	}
}

func TestOpenAPI_Errors(t *testing.T) {
	for spec, want := range map[string]string{
		`swagger: "2.0"`:                         "Swagger 2.0",
		`foo: bar`:                               "not an OpenAPI 3",
		`openapi: 3.0.0`:                         "no paths",
		`[1, 2]`:                                 "not an object",
		"openapi: 3.0.0\npaths: {a: &x {b: *x}}": "aliases",
	} {
		if _, err := buildOpenAPIRules([]byte(spec), OpenAPIImport{}); err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%q: %v, want %q", spec, err, want)
		}
	}
}

// Every generator the import may write exists and renders.
func TestOpenAPI_FakeNamesExist(t *testing.T) {
	names := map[string]bool{"word": true, "sentence": true, "date": true, "number 1 2": true, "float64range 0 1.5": true, "bool": true}
	for _, f := range fakeByName {
		names[f] = true
	}
	for _, f := range fakeByFormat {
		names[f] = true
	}
	for n := range names {
		name, args, _ := strings.Cut(n, " ")
		if _, err := renderBody(t, MockRule{Path: "/", Body: `{{ fake "` + name + `" ` + args + ` | json }}`, Template: true}, &mockRequest{}); err != nil {
			t.Errorf("%s: %v", n, err)
		}
	}
}

func TestOpenAPI_API(t *testing.T) {
	env := newMockEnv(t, nil)
	post := func(body any) (int, map[string]any) {
		b, _ := json.Marshal(body)
		w := env.do(t, "POST", "/api/mocks/import/openapi", string(b), nil)
		var out map[string]any
		json.Unmarshal(w.Body.Bytes(), &out)
		return w.Code, out
	}
	count := func() int {
		w := env.do(t, "GET", "/api/mocks", "", nil)
		var resp struct {
			Data []MockRule `json:"data"`
		}
		json.Unmarshal(w.Body.Bytes(), &resp)
		return len(resp.Data)
	}

	env.create(t, `{"path":"/own"}`)
	code, out := post(map[string]any{"spec": petSpec, "project": "shop"})
	if code != 200 || out["group"] != "openapi:Shop API" || len(out["rules"].([]any)) != 5 || len(out["skipped"].([]any)) != 1 {
		t.Fatalf("import: %d %v", code, out)
	}
	first := out["rules"].([]any)[0].(map[string]any)
	if first["source"] != "api" || first["group"] != "openapi:Shop API" {
		t.Errorf("rule = %v", first)
	}
	w := env.do(t, "GET", "/mock/v1/users/5", "", map[string]string{"X-Buggregator-Project": "shop"})
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"id": 5`) {
		t.Errorf("served %d %s", w.Code, w.Body.String())
	}
	// Re-import replaces the group.
	if code, _ := post(map[string]any{"spec": petSpec, "project": "shop"}); code != 200 || count() != 6 {
		t.Errorf("re-import: %d, %d rules", code, count())
	}
	// Editing a rule keeps its group.
	id := first["id"].(string)
	if w := env.do(t, "PUT", "/api/mocks/"+id, `{"path":"/v1/users/{id}","method":"GET","body":"x"}`, nil); w.Code == 200 {
		t.Errorf("PUT of a replaced rule: %d", w.Code) // the first import's rules are gone
	}
	w = env.do(t, "DELETE", "/api/mocks/groups/openapi:Shop%20API", "", nil)
	if !strings.Contains(w.Body.String(), `"deleted":5`) || count() != 1 {
		t.Errorf("delete group: %s, %d rules", w.Body.String(), count())
	}

	// URL import.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/big" {
			w.Write([]byte(strings.Repeat("#", MaxOpenAPISpec+1)))
			return
		}
		w.Write([]byte(petSpec))
	}))
	defer srv.Close()
	if code, out := post(map[string]any{"url": srv.URL + "/spec.yaml", "group": "remote"}); code != 200 || out["group"] != "remote" {
		t.Errorf("url import: %d %v", code, out)
	}
	for _, body := range []map[string]any{
		{"url": "file:///etc/passwd"},
		{"url": srv.URL + "/big"},
		{"spec": petSpec, "url": srv.URL},
		{},
		{"spec": "openapi: 3.0.0\npaths:\n  /a:\n    get: {responses: {}}"},
	} {
		if code, out := post(body); code != 422 {
			t.Errorf("%v: %d %v", body, code, out)
		}
	}
}

func TestOpenAPI_MCP(t *testing.T) {
	env := newMockEnv(t, nil)
	cs := connectMockTools(t, env, true)
	var out struct {
		Group   string           `json:"group"`
		Rules   []map[string]any `json:"rules"`
		Skipped []OpenAPISkip    `json:"skipped"`
	}
	if text, isErr := mockCall(t, cs, "mock_import_openapi", map[string]any{"spec": petSpec, "base_path": ""}, &out); isErr {
		t.Fatalf("import: %s", text)
	}
	if out.Group != "openapi:Shop API" || len(out.Rules) != 5 || len(out.Skipped) != 1 {
		t.Errorf("import = %+v", out)
	}
	if r := ruleBy(t, env.mod.mocks.list(), "GET", "/users/{id}"); r.Source != "mcp" {
		t.Errorf("source = %s", r.Source)
	}
	var del map[string]any
	mockCall(t, cs, "mock_group_delete", map[string]any{"group": "openapi:Shop API"}, &del)
	if del["deleted"] != float64(5) || len(env.mod.mocks.list()) != 0 {
		t.Errorf("group delete = %v", del)
	}
	if tools := toolNames(t, connectMockTools(t, env, false)); tools["mock_import_openapi"] != nil || tools["mock_group_delete"] != nil {
		t.Error("import tools exist with mock_write off")
	}
}

func TestOpenAPI_ConfigFile(t *testing.T) {
	path := t.TempDir() + "/api.yaml"
	os.WriteFile(path, []byte(petSpec), 0o600)
	res, err := OpenAPIRulesFromFile(path, OpenAPIImport{Project: "shop"})
	if err != nil || len(res.Rules) != 5 {
		t.Fatalf("%v, %d rules", err, len(res.Rules))
	}
	env := newMockEnv(t, res.Rules)
	r := ruleBy(t, env.mod.mocks.list(), "GET", "/v1/users/{id}")
	if !r.ReadOnly || r.Source != "config" || r.Group != "openapi:Shop API" {
		t.Errorf("config rule = %+v", r)
	}
	if w := env.do(t, "GET", "/mock/v1/users/3", "", map[string]string{"X-Buggregator-Project": "shop"}); !strings.Contains(w.Body.String(), `"id": 3`) {
		t.Errorf("served %s", w.Body.String())
	}
	if _, err := OpenAPIRulesFromFile(t.TempDir()+"/missing.yaml", OpenAPIImport{}); err == nil {
		t.Error("missing file: no error")
	}
}

// An empty group name never matches the rules made by hand.
func TestMockStore_DeleteGroupEmpty(t *testing.T) {
	env := newMockEnv(t, nil)
	env.create(t, `{"path":"/a"}`)
	env.create(t, `{"path":"/b"}`)
	for _, g := range []string{"", "  "} {
		if _, err := env.mod.mocks.deleteGroup(g); err == nil {
			t.Errorf("deleteGroup(%q): no error", g)
		}
	}
	cs := connectMockTools(t, env, true)
	if _, isErr := mockCall(t, cs, "mock_group_delete", map[string]any{"group": ""}, nil); !isErr {
		t.Error("mock_group_delete with an empty group: no error")
	}
	if w := env.do(t, "DELETE", "/api/mocks/groups/%20", "", nil); w.Code != 422 {
		t.Errorf("REST blank group: %d", w.Code)
	}
	if n := len(env.mod.mocks.list()); n != 2 {
		t.Errorf("%d rules left, want 2", n)
	}
}

// A small spec with a wide self-referencing (or chained) schema must not
// expand exponentially: the operation is skipped, fast and small.
func TestOpenAPI_WideSchemaBounded(t *testing.T) {
	props := func(ref string, n int) string {
		var b strings.Builder
		for i := 0; i < n; i++ {
			b.WriteString(`"p` + string(rune('a'+i)) + `": {"$ref": "#/components/schemas/` + ref + `"}, `)
		}
		return strings.TrimSuffix(b.String(), ", ")
	}
	head := `{"openapi": "3.0.0", "info": {"title": "T"}, "paths": {"/x": {"get": {"responses": {"200": {"content": {"application/json": {"schema": {"$ref": "#/components/schemas/S0"}}}}}}}}, "components": {"schemas": {`
	selfRef := head + `"S0": {"type": "object", "properties": {` + props("S0", 8) + `}}}}}`
	var chain strings.Builder
	chain.WriteString(head)
	for i := 0; i < 12; i++ {
		if i > 0 {
			chain.WriteString(", ")
		}
		chain.WriteString(`"S` + strconv.Itoa(i) + `": {"type": "object", "properties": {` + props("S"+strconv.Itoa(i+1), 8) + `}}`)
	}
	chain.WriteString(`, "S12": {"type": "string"}}}}`)
	allOf := head + `"S0": {"allOf": [` + strings.TrimSuffix(strings.Repeat(`{"$ref": "#/components/schemas/S0"}, `, 8), ", ") + `], "properties": {"a": {"$ref": "#/components/schemas/S0"}}}}}}`

	for name, spec := range map[string]string{"self": selfRef, "chain": chain.String(), "allOf": allOf} {
		var before, after runtime.MemStats
		runtime.GC()
		runtime.ReadMemStats(&before)
		start := time.Now()
		res, err := buildOpenAPIRules([]byte(spec), OpenAPIImport{})
		d := time.Since(start)
		runtime.ReadMemStats(&after)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if d > 2*time.Second || after.TotalAlloc-before.TotalAlloc > 200<<20 {
			t.Errorf("%s: %v, %d MB allocated", name, d, (after.TotalAlloc-before.TotalAlloc)>>20)
		}
		if len(res.Rules) == 1 {
			if len(res.Rules[0].Body) > MaxMockBody {
				t.Errorf("%s: body %d bytes", name, len(res.Rules[0].Body))
			}
		} else if len(res.Skipped) != 1 || !strings.Contains(res.Skipped[0].Reason, "too large") {
			t.Errorf("%s: rules %d, skipped %+v", name, len(res.Rules), res.Skipped)
		}
	}
}

// Names and media types from the spec stay text, never template code.
func TestOpenAPI_SpecTextIsNotTemplate(t *testing.T) {
	spec := `{"openapi": "3.0.0", "info": {"title": "T"}, "paths": {"/x": {"get": {"responses": {"200": {"content": {
		"application/json{{ header \"X-A\" }}": {"schema": {"type": "object", "properties": {
			"{{ bodyRaw }}": {"type": "boolean"},
			"a}}b": {"type": "boolean"},
			"{{": {"type": "boolean"}
		}}}}}}}}}}`
	res := importSpec(t, spec, OpenAPIImport{})
	if len(res.Rules) != 1 {
		t.Fatalf("rules %d, skipped %+v", len(res.Rules), res.Skipped)
	}
	r := res.Rules[0]
	out, err := renderMockRule(&r, &mockRequest{Method: "GET", Path: "/x", Header: http.Header{"X-A": {"leak"}}, Body: []byte("SECRET")})
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(out.Body, &got); err != nil {
		t.Fatalf("not JSON: %v\n%s", err, out.Body)
	}
	for _, k := range []string{"{{ bodyRaw }}", "a}}b", "{{"} {
		if _, ok := got[k].(bool); !ok {
			t.Errorf("key %q missing in %v", k, got)
		}
	}
	if strings.Contains(string(out.Body), "SECRET") {
		t.Error("request body echoed")
	}
	if ct := out.Headers["Content-Type"]; ct != `application/json{{ header "X-A" }}` {
		t.Errorf("Content-Type = %q", ct)
	}
}

// Many operations sharing one heavy schema or example stay bounded as a
// whole import, and no more than MaxMockRulesPerProject rules are built.
func TestOpenAPI_ImportBudget(t *testing.T) {
	var tProps, sProps []string
	for i := 0; i < 100; i++ {
		tProps = append(tProps, `"t`+strconv.Itoa(i)+`": {"type": "string"}`)
	}
	for i := 0; i < 90; i++ {
		sProps = append(sProps, `"s`+strconv.Itoa(i)+`": {"$ref": "#/components/schemas/T"}`)
	}
	paths := func(n int) string {
		out := make([]string, n)
		for i := range out {
			out[i] = `"/p` + strconv.Itoa(i) + `": {"$ref": "#/components/pathItems/P"}`
		}
		return strings.Join(out, ", ")
	}
	spec := func(schemaS string, n int) string {
		return `{"openapi": "3.1.0", "info": {"title": "T"}, "paths": {` + paths(n) + `}, "components": {` +
			`"pathItems": {"P": {"get": {"responses": {"200": {"content": {"application/json": {"schema": {"$ref": "#/components/schemas/S"}}}}}}}}, ` +
			`"schemas": {"T": {"type": "object", "properties": {` + strings.Join(tProps, ", ") + `}}, "S": ` + schemaS + `}}}`
	}
	bigExample := `{"type": "object", "example": {"x": "` + strings.Repeat("a", 900<<10) + `"}}`
	cases := map[string]string{
		"template": spec(`{"type": "object", "properties": {`+strings.Join(sProps, ", ")+`}}`, 2000),
		"example":  spec(bigExample, 2000),
		"cheap":    spec(`{"type": "string", "example": "x"}`, 2000),
	}
	for name, sp := range cases {
		var before, after runtime.MemStats
		runtime.GC()
		runtime.ReadMemStats(&before)
		start := time.Now()
		res, err := buildOpenAPIRules([]byte(sp), OpenAPIImport{})
		d := time.Since(start)
		runtime.ReadMemStats(&after)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		alloc := (after.TotalAlloc - before.TotalAlloc) >> 20
		kept := 0
		for _, r := range res.Rules {
			kept += len(r.Body)
		}
		t.Logf("%s: %v, %d MB allocated, %d rules, %d MB of bodies, %d skipped", name, d, alloc, len(res.Rules), kept>>20, len(res.Skipped))
		if d > 3*time.Second || alloc > 512 || kept > maxImportBytes+MaxMockBody || len(res.Rules) > MaxMockRulesPerProject {
			t.Errorf("%s: not bounded", name)
		}
		if len(res.Rules)+len(res.Skipped) != 2000 {
			t.Errorf("%s: %d rules + %d skipped", name, len(res.Rules), len(res.Skipped))
		}
	}
}
