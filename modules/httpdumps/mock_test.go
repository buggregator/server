package httpdumps

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/buggregator/go-buggregator/internal/module"
	serverhttp "github.com/buggregator/go-buggregator/internal/server/http"
	"github.com/buggregator/go-buggregator/internal/server/ws"
	"github.com/buggregator/go-buggregator/internal/storage"
)

func TestMatchMockPath(t *testing.T) {
	tests := []struct {
		pattern, path string
		want          bool
	}{
		{"/users/me", "/users/me", true},
		{"/users/me", "/users/me/", true},
		{"/users/me", "/users/you", false},
		{"/users/me", "/users", false},
		{"/users/{id}", "/users/42", true},
		{"/users/{id}", "/users/42/posts", false},
		{"/users/{id}", "/users", false},
		{"/users/{id}/posts", "/users/42/posts", true},
		{"/files/*", "/files/a/b.png", true},
		{"/files/*", "/files", true},
		{"/files/*", "/other/a", false},
		{"/*", "/anything/at/all", true},
		{"/", "/", true},
		{"/", "/x", false},
	}
	for _, tt := range tests {
		if got := matchMockPath(tt.pattern, tt.path); got != tt.want {
			t.Errorf("matchMockPath(%q, %q) = %v, want %v", tt.pattern, tt.path, got, tt.want)
		}
	}
}

func TestMockRuleNormalize(t *testing.T) {
	ok := MockRule{Path: "/users/{id}"}
	if err := ok.Normalize(); err != nil {
		t.Fatalf("valid rule: %v", err)
	}
	if ok.Method != "ANY" || ok.Status != 200 || ok.Headers == nil {
		t.Errorf("defaults = %s %d %v, want ANY 200 {}", ok.Method, ok.Status, ok.Headers)
	}

	bad := []struct {
		rule  MockRule
		field string
	}{
		{MockRule{Path: "users"}, "path"},
		{MockRule{Path: "/a/*/b"}, "path"},
		{MockRule{Path: "/a/b*"}, "path"},
		{MockRule{Path: "/a/{id"}, "path"},
		{MockRule{Path: "/a", Method: "FETCH"}, "method"},
		{MockRule{Path: "/a", Status: 99}, "status"},
		{MockRule{Path: "/a", Status: 600}, "status"},
		{MockRule{Path: "/a", DelayMs: MaxMockDelay + 1}, "delay_ms"},
		{MockRule{Path: "/a", Body: strings.Repeat("x", MaxMockBody+1)}, "body"},
		{MockRule{Path: "/a", Headers: map[string]string{"Bad Name": "x"}}, "headers"},
	}
	for _, tt := range bad {
		err := tt.rule.Normalize()
		verr, isV := err.(ValidationError)
		if !isV || verr[tt.field] == "" {
			t.Errorf("%+v: err = %v, want a %s error", tt.rule, err, tt.field)
		}
	}
}

// mockEnv wires the module the way app.go does: registry init (migrations,
// public routes), the real EventService, protected routes, and a stand-in for
// the ingestion pipeline at "/".
type mockEnv struct {
	mux      *http.ServeMux
	mod      *Module
	pipeline []string // paths that reached the pipeline stand-in
}

func newMockEnv(t *testing.T, config []MockRule) *mockEnv {
	t.Helper()
	db, err := storage.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })

	env := &mockEnv{mux: http.NewServeMux(), mod: New(nil, db)}
	env.mod.SetMockRules(config)

	registry := module.NewRegistry()
	registry.Register(env.mod)
	store := storage.NewSQLiteStore(db)
	if err := registry.Init(db, env.mux, store); err != nil {
		t.Fatal(err)
	}
	env.mod.SetEventService(serverhttp.NewEventService(store, ws.NewHub(), registry, nil, db))
	registry.RegisterProtectedRoutes(env.mux, func(h http.Handler) http.Handler { return h })
	env.mux.HandleFunc("GET /api/version", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("version")) })
	env.mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		env.pipeline = append(env.pipeline, r.URL.Path)
		w.Write([]byte("pipeline"))
	})
	return env
}

func (e *mockEnv) do(t *testing.T, method, path, body string, hdr map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	r := httptest.NewRequest(method, path, strings.NewReader(body))
	for k, v := range hdr {
		r.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	e.mux.ServeHTTP(w, r)
	return w
}

func (e *mockEnv) create(t *testing.T, rule string) MockRule {
	t.Helper()
	w := e.do(t, "POST", "/api/mocks", rule, nil)
	if w.Code != http.StatusCreated {
		t.Fatalf("create %s: %d %s", rule, w.Code, w.Body.String())
	}
	var r MockRule
	json.Unmarshal(w.Body.Bytes(), &r)
	return r
}

func (e *mockEnv) requests(t *testing.T, query string) []map[string]any {
	t.Helper()
	w := e.do(t, "GET", "/api/mocks/requests"+query, "", nil)
	var resp struct {
		Data []map[string]any `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &resp)
	return resp.Data
}

func TestMockServer_Matching(t *testing.T) {
	env := newMockEnv(t, []MockRule{{Name: "from yaml", Method: "GET", Path: "/health", Body: "yaml"}})
	env.create(t, `{"name":"me","method":"GET","path":"/users/me","status":200,"body":"me"}`)
	env.create(t, `{"name":"user","method":"GET","path":"/users/{id}","status":200,"headers":{"Content-Type":"application/json"},"body":"{\"id\":1}"}`)
	env.create(t, `{"name":"any user","method":"ANY","path":"/users/{id}","status":202,"body":"any"}`)
	env.create(t, `{"name":"files","path":"/files/*","status":201,"body":"file"}`)
	env.create(t, `{"name":"off","path":"/off","enabled":false,"body":"off"}`)
	env.create(t, `{"name":"proj","path":"/proj","project":"shop","body":"shop"}`)
	env.create(t, `{"name":"health override","method":"GET","path":"/health","body":"api"}`)

	basic := "Basic " + base64.StdEncoding.EncodeToString([]byte("mock:shop"))
	tests := []struct {
		name, method, path string
		hdr                map[string]string
		status             int
		body               string
	}{
		{"exact wins by order", "GET", "/mock/users/me", nil, 200, "me"},
		{"param", "GET", "/mock/users/42", nil, 200, `{"id":1}`},
		{"ANY catches other methods", "DELETE", "/mock/users/42", nil, 202, "any"},
		{"wildcard", "PUT", "/mock/files/a/b.png", nil, 201, "file"},
		{"yaml rule first", "GET", "/mock/health", nil, 200, "yaml"},
		{"project via header", "GET", "/mock/proj", map[string]string{"X-Buggregator-Project": "shop"}, 200, "shop"},
		{"project via basic auth", "GET", "/mock/proj", map[string]string{"Authorization": basic}, 200, "shop"},
		{"other project", "GET", "/mock/proj", nil, 404, `{"error":"no mock rule matched GET /proj"}`},
		{"disabled", "GET", "/mock/off", nil, 404, `{"error":"no mock rule matched GET /off"}`},
		{"no rule", "POST", "/mock/nothing", nil, 404, `{"error":"no mock rule matched POST /nothing"}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := env.do(t, tt.method, tt.path, "", tt.hdr)
			if w.Code != tt.status || strings.TrimSpace(w.Body.String()) != tt.body {
				t.Errorf("got %d %q, want %d %q", w.Code, w.Body.String(), tt.status, tt.body)
			}
			if csp := w.Header().Get("Content-Security-Policy"); csp != "sandbox" {
				t.Errorf("Content-Security-Policy = %q, want sandbox", csp)
			}
		})
	}
	if len(env.pipeline) != 0 {
		t.Errorf("mock requests reached the ingestion pipeline: %v", env.pipeline)
	}
}

func TestMockServer_CapturesEveryRequest(t *testing.T) {
	env := newMockEnv(t, nil)
	rule := env.create(t, `{"name":"order","method":"POST","path":"/orders","status":201,"body":"ok"}`)

	env.do(t, "POST", "/mock/orders?x=1", `{"item":"book"}`, map[string]string{"Content-Type": "application/json"})
	env.do(t, "GET", "/mock/missing", "", map[string]string{"X-Buggregator-Project": "shop"})

	all := env.requests(t, "")
	if len(all) != 2 {
		t.Fatalf("captured %d requests, want 2", len(all))
	}
	byURI := map[string]map[string]any{}
	for _, ev := range all {
		p := ev["payload"].(map[string]any)
		byURI[p["request"].(map[string]any)["uri"].(string)] = ev
	}

	matched := byURI["mock/orders"]["payload"].(map[string]any)
	info, _ := matched["mock"].(map[string]any)
	if info == nil || info["rule_id"] != rule.ID || info["status"] != float64(201) {
		t.Errorf("matched request mock = %v, want rule %s status 201", matched["mock"], rule.ID)
	}
	if body := matched["request"].(map[string]any)["body"]; body != `{"item":"book"}` {
		t.Errorf("captured body = %v", body)
	}

	missing := byURI["mock/missing"]
	if v, present := missing["payload"].(map[string]any)["mock"]; !present || v != nil {
		t.Errorf("unmatched request mock = %v (present %v), want null", v, present)
	}
	if missing["project"] != "shop" {
		t.Errorf("unmatched request project = %v, want shop", missing["project"])
	}

	if got := env.requests(t, "?rule_id="+rule.ID); len(got) != 1 {
		t.Errorf("requests for rule = %d, want 1", len(got))
	}
	if got := env.requests(t, "?project=shop"); len(got) != 1 {
		t.Errorf("requests for project shop = %d, want 1", len(got))
	}
}

// Nothing outside /mock/ changes: SDK ingestion, /api and the UI keep their
// routes even when a rule has the same path.
func TestMockServer_NeverShadowsOtherRoutes(t *testing.T) {
	env := newMockEnv(t, nil)
	env.create(t, `{"path":"/*","body":"mock"}`)
	env.create(t, `{"path":"/api/version","body":"mock"}`)

	if w := env.do(t, "GET", "/api/version", "", nil); w.Body.String() != "version" {
		t.Errorf("/api/version = %q", w.Body.String())
	}
	for _, p := range []string{"/api/1/envelope/", "/api/1/store/", "/", "/settings", "/mocks", "/mock-server"} {
		if w := env.do(t, "POST", p, "{}", nil); w.Body.String() != "pipeline" {
			t.Errorf("%s answered %q, want the pipeline", p, w.Body.String())
		}
	}
	if w := env.do(t, "GET", "/mock/api/version", "", nil); w.Body.String() != "mock" {
		t.Errorf("/mock/api/version = %q, want the mock", w.Body.String())
	}
}

func TestMockServer_Delay(t *testing.T) {
	env := newMockEnv(t, nil)
	env.create(t, `{"path":"/slow","delay_ms":80,"body":"late"}`)

	start := time.Now()
	w := env.do(t, "GET", "/mock/slow", "", nil)
	if took := time.Since(start); took < 80*time.Millisecond {
		t.Errorf("answered after %v, want at least 80ms", took)
	}
	if w.Body.String() != "late" {
		t.Errorf("body = %q", w.Body.String())
	}
	// The request is stored before the delay.
	if len(env.requests(t, "")) != 1 {
		t.Error("delayed request was not captured")
	}
}

func TestMockAPI_CRUD(t *testing.T) {
	env := newMockEnv(t, []MockRule{{Path: "/from-yaml"}})

	a := env.create(t, `{"name":"a","path":"/a"}`)
	b := env.create(t, `{"name":"b","path":"/b"}`)
	if !a.Enabled || a.Method != "ANY" || a.Status != 200 || a.Source != "api" {
		t.Errorf("defaults = %+v", a)
	}

	// Validation.
	w := env.do(t, "POST", "/api/mocks", `{"path":"no-slash","status":42}`, nil)
	var verr struct {
		Errors map[string]string `json:"errors"`
	}
	json.Unmarshal(w.Body.Bytes(), &verr)
	if w.Code != http.StatusUnprocessableEntity || verr.Errors["path"] == "" || verr.Errors["status"] == "" {
		t.Errorf("invalid rule: %d %s", w.Code, w.Body.String())
	}

	// List: config rule first, read-only.
	list := func() []MockRule {
		w := env.do(t, "GET", "/api/mocks", "", nil)
		var resp struct {
			Data []MockRule `json:"data"`
		}
		json.Unmarshal(w.Body.Bytes(), &resp)
		return resp.Data
	}
	rules := list()
	if len(rules) != 3 || rules[0].Source != "config" || !rules[0].ReadOnly || rules[1].ID != a.ID || rules[2].ID != b.ID {
		t.Fatalf("list = %+v", rules)
	}

	// Config rules cannot be changed.
	if w := env.do(t, "PUT", "/api/mocks/"+rules[0].ID, `{"path":"/x"}`, nil); w.Code != http.StatusConflict {
		t.Errorf("PUT config rule = %d, want 409", w.Code)
	}
	if w := env.do(t, "DELETE", "/api/mocks/"+rules[0].ID, "", nil); w.Code != http.StatusConflict {
		t.Errorf("DELETE config rule = %d, want 409", w.Code)
	}

	// Update.
	w = env.do(t, "PUT", "/api/mocks/"+a.ID, `{"name":"a2","method":"get","path":"/a2","status":418,"enabled":false}`, nil)
	var upd MockRule
	json.Unmarshal(w.Body.Bytes(), &upd)
	if w.Code != 200 || upd.Name != "a2" || upd.Method != "GET" || upd.Status != 418 || upd.Enabled || upd.Position != a.Position {
		t.Errorf("update: %d %+v", w.Code, upd)
	}

	// Reorder.
	w = env.do(t, "POST", "/api/mocks/order", `{"ids":["`+b.ID+`"]}`, nil)
	if w.Code != 200 {
		t.Fatalf("order: %d %s", w.Code, w.Body.String())
	}
	if rules := list(); rules[1].ID != b.ID || rules[2].ID != a.ID {
		t.Errorf("after reorder = %s, %s; want b, a", rules[1].Name, rules[2].Name)
	}
	if w := env.do(t, "POST", "/api/mocks/order", `{"ids":["nope"]}`, nil); w.Code != http.StatusNotFound {
		t.Errorf("order with unknown id = %d, want 404", w.Code)
	}

	// Get and delete.
	if w := env.do(t, "GET", "/api/mocks/"+b.ID, "", nil); w.Code != 200 {
		t.Errorf("get = %d", w.Code)
	}
	if w := env.do(t, "DELETE", "/api/mocks/"+b.ID, "", nil); w.Code != 200 {
		t.Errorf("delete = %d", w.Code)
	}
	if w := env.do(t, "GET", "/api/mocks/"+b.ID, "", nil); w.Code != http.StatusNotFound {
		t.Errorf("get deleted = %d, want 404", w.Code)
	}
}

// The rule API sits behind the API auth; the mock server itself stays public.
func TestMockAPI_Protected(t *testing.T) {
	db, err := storage.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	mod := New(nil, db)
	registry := module.NewRegistry()
	registry.Register(mod)
	mux := http.NewServeMux()
	if err := registry.Init(db, mux, storage.NewSQLiteStore(db)); err != nil {
		t.Fatal(err)
	}
	deny := func(http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusUnauthorized) })
	}
	registry.RegisterProtectedRoutes(mux, deny)

	for _, req := range []*http.Request{
		httptest.NewRequest("GET", "/api/mocks", nil),
		httptest.NewRequest("POST", "/api/mocks", bytes.NewReader([]byte(`{"path":"/a"}`))),
		httptest.NewRequest("GET", "/api/mocks/requests", nil),
	} {
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Errorf("%s %s = %d, want 401", req.Method, req.URL.Path, w.Code)
		}
	}
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest("GET", "/mock/a", nil))
	body, _ := io.ReadAll(w.Body)
	if w.Code != http.StatusNotFound || !strings.Contains(string(body), "no mock rule matched") {
		t.Errorf("/mock/a = %d %s, want the public mock server's 404", w.Code, body)
	}
}

func TestSetMockRules_SkipsInvalid(t *testing.T) {
	mod := New(nil, nil)
	mod.SetMockRules([]MockRule{{Path: "/ok"}, {Path: "bad"}, {Path: "/also-ok", Status: 204}})
	rules := mod.mocks.list()
	if len(rules) != 2 || rules[0].Path != "/ok" || rules[1].Status != 204 || !rules[1].Enabled || !rules[1].ReadOnly {
		t.Errorf("config rules = %+v", rules)
	}
}

// The security headers keep an HTML mock out of the UI's origin: a rule must
// not be able to weaken or drop them.
func TestMockServer_RuleCannotOverrideSecurityHeaders(t *testing.T) {
	env := newMockEnv(t, nil)
	env.create(t, `{"path":"/page","headers":{"content-security-policy":"sandbox allow-scripts allow-same-origin","X-Content-Type-Options":"","Content-Type":"text/html"},"body":"<script>alert(1)</script>"}`)

	w := env.do(t, "GET", "/mock/page", "", nil)
	if got := w.Header().Values("Content-Security-Policy"); len(got) != 1 || got[0] != "sandbox" {
		t.Errorf("Content-Security-Policy = %q, want [sandbox]", got)
	}
	if got := w.Header().Values("X-Content-Type-Options"); len(got) != 1 || got[0] != "nosniff" {
		t.Errorf("X-Content-Type-Options = %q, want [nosniff]", got)
	}
}

// A client cannot mark an API rule read-only.
func TestMockAPI_UpdateIgnoresReadOnly(t *testing.T) {
	env := newMockEnv(t, nil)
	a := env.create(t, `{"path":"/a","read_only":true}`)
	if a.ReadOnly {
		t.Error("create returned read_only true")
	}
	w := env.do(t, "PUT", "/api/mocks/"+a.ID, `{"path":"/a","read_only":true,"source":"config"}`, nil)
	var upd MockRule
	json.Unmarshal(w.Body.Bytes(), &upd)
	if upd.ReadOnly || upd.Source != "api" {
		t.Errorf("update returned read_only %v source %q, want false api", upd.ReadOnly, upd.Source)
	}
}
