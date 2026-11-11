package httpdumps

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"
)

func renderBody(t *testing.T, rule MockRule, req *mockRequest) (string, error) {
	t.Helper()
	if req.Header == nil {
		req.Header = http.Header{}
	}
	if req.Params == nil {
		req.Params = mockPathParams(rule.Path, req.Path)
	}
	out, err := renderMockRule(&rule, req)
	return string(out.Body), err
}

func TestMockTemplate_Helpers(t *testing.T) {
	req := func() *mockRequest {
		return &mockRequest{
			Method: "POST", Path: "/users/42/files/a/b.png",
			Query:  url.Values{"page": {"3"}, "tag": {"a", "b"}},
			Header: http.Header{"X-Request-Id": {"abc"}},
			Body:   []byte(`{"user":{"email":"a@example.com"},"items":[{"id":7}],"n":1.5,"ok":true}`),
		}
	}
	tests := []struct{ src, want string }{
		{`{{ path "id" }}`, `42`},
		{`{{ path "*" }}`, `a/b.png`},
		{`{{ path "nope" }}`, ``},
		{`{{ query "page" }}`, `3`},
		{`{{ query "limit" | default "20" }}`, `20`},
		{`{{ queryAll "tag" | json }}`, `["a","b"]`},
		{`{{ queryAll "none" | json }}`, `[]`},
		{`{{ header "x-request-id" }}`, `abc`},
		{`{{ body "user.email" | json }}`, `"a@example.com"`},
		{`{{ body "items.0.id" }}`, `7`},
		{`{{ body "items.5.id" | json }}`, `null`},
		{`{{ body "user" | json }}`, `{"email":"a@example.com"}`},
		{`{{ body "ok" }}`, `true`},
		{`{{ body "missing" | default "x" }}`, `x`},
		{`{{ method }} {{ requestPath }}`, `POST /users/42/files/a/b.png`},
		{`{{ pick "only" }}`, `only`},
		{`[{{ range $i := repeat 3 }}{{ if $i }},{{ end }}{{ add $i 1 }}{{ end }}]`, `[1,2,3]`},
		{`{{ json "O'Neil \"Jr\" <b>" }}`, `"O'Neil \"Jr\" <b>"`},
		{`{{ upper "abc" }}{{ lower "DEF" }}`, `ABCdef`},
		{`{{ range $i := repeat (int (query "page")) }}{{ $i }}{{ end }}`, `012`},
		{`{{ add (int (body "n")) 1 }}`, `2`},
		{`{{ if gt (int (body "items.0.id")) 5 }}big{{ end }}`, `big`},
		{`{{ int (body "ok") }}{{ int (query "none") }}{{ int "4.9" }}`, `104`},
	}
	rulePath := "/users/{id}/files/*"
	for _, tt := range tests {
		got, err := renderBody(t, MockRule{Path: rulePath, Body: tt.src, Template: true}, req())
		if err != nil || got != tt.want {
			t.Errorf("%s = %q, %v; want %q", tt.src, got, err, tt.want)
		}
	}

	// bodyRaw echoes; a non-JSON body gives nothing to body.
	r := &mockRequest{Body: []byte("plain text")}
	if got, _ := renderBody(t, MockRule{Path: "/", Body: `{{ bodyRaw }}|{{ body "a" | json }}`, Template: true}, r); got != `plain text|null` {
		t.Errorf("bodyRaw = %q", got)
	}
	// now: RFC 3339 by default, unix seconds, a layout.
	got, err := renderBody(t, MockRule{Path: "/", Body: `{{ now }}|{{ now "unix" }}|{{ now "2006" }}`, Template: true}, &mockRequest{})
	parts := strings.Split(got, "|")
	if err != nil || len(parts) != 3 || !strings.HasSuffix(parts[0], "Z") || len(parts[1]) < 10 || len(parts[2]) != 4 {
		t.Errorf("now = %q, %v", got, err)
	}
}

func TestMockTemplate_Fake(t *testing.T) {
	tests := []struct {
		src   string
		check func(string) bool
	}{
		{`{{ fake "name" }}`, func(s string) bool { return strings.Contains(s, " ") }},
		{`{{ fake "Name" }}`, func(s string) bool { return s != "" }},
		{`{{ fake "number" 1 10 }}`, func(s string) bool { return len(s) >= 1 && len(s) <= 2 }},
		{`{{ fake "date" | json }}`, func(s string) bool { return strings.HasPrefix(s, `"`) }},
		{`{{ fake "email" }}`, func(s string) bool { return strings.Contains(s, "@") }},
		{`{{ fake "person" | json }}`, func(s string) bool { return json.Valid([]byte(s)) && strings.HasPrefix(s, "{") }},
		{`{{ fake "loremipsumsentence" 5 }}`, func(s string) bool { return len(strings.Fields(s)) == 5 }},
		{`{{ fake "dice" 2 | json }}`, func(s string) bool { return strings.Count(s, ",") == 1 }}, // list param "sides" from its default [6]
		{`{{ fake "randomstring" }}`, nil}, // a required parameter is missing
	}
	for _, tt := range tests {
		got, err := renderBody(t, MockRule{Path: "/", Body: tt.src, Template: true}, &mockRequest{})
		if tt.check == nil {
			if err == nil {
				t.Errorf("%s = %q, want an error", tt.src, got)
			}
			continue
		}
		if err != nil || !tt.check(got) {
			t.Errorf("%s = %q, %v", tt.src, got, err)
		}
	}

	errs := []string{
		`{{ fake "nope" }}`,
		`{{ fake "sql" }}`,        // excluded
		`{{ fake "name" 1 2 3 }}`, // too many arguments
		`{{ fake "loremipsumsentence" 5000 }}`,
		`{{ pick }}`,
		`{{ now "a" "b" }}`,
		`{{ repeat -1 }}`,
		`{{ int "abc" }}`,
		`{{ int (queryAll "x") }}`,
	}
	for _, src := range errs {
		if _, err := renderBody(t, MockRule{Path: "/", Body: src, Template: true}, &mockRequest{}); err == nil {
			t.Errorf("%s: no error", src)
		}
	}
}

func TestMockFakes_Listed(t *testing.T) {
	fakes := MockFakes()
	byName := map[string]MockFake{}
	for _, f := range fakes {
		byName[f.Name] = f
	}
	if _, ok := byName["name"]; !ok || len(fakes) < 100 {
		t.Fatalf("%d fakes, name present: %v", len(fakes), ok)
	}
	for _, n := range []string{"sql", "csv", "json", "template", "regex", "imagepng"} {
		if _, ok := byName[n]; ok {
			t.Errorf("%s is listed", n)
		}
	}
	if p := byName["number"].Params; len(p) != 2 || p[0].Name != "min" || p[1].Name != "max" {
		t.Errorf("number params = %+v", p)
	}
}

func TestMockTemplate_Limits(t *testing.T) {
	// The repeat budget counts every repeat, even ones that write nothing.
	src := `{{ range repeat 200 }}{{ range repeat 100 }}{{ end }}{{ end }}`
	if _, err := renderBody(t, MockRule{Path: "/", Body: src, Template: true}, &mockRequest{}); err == nil ||
		!strings.Contains(err.Error(), "repeat") {
		t.Errorf("repeat budget: %v", err)
	}
	// Output past 1 MB stops the render.
	src = `{{ range repeat 20 }}{{ bodyRaw }}{{ end }}`
	big := &mockRequest{Body: []byte(strings.Repeat("x", 100<<10))}
	if _, err := renderBody(t, MockRule{Path: "/", Body: src, Template: true}, big); err == nil ||
		!strings.Contains(err.Error(), "larger than") {
		t.Errorf("size cap: %v", err)
	}
	// A header with a line break is refused.
	rule := MockRule{Path: "/", Headers: map[string]string{"X-A": "a{{ \"\\n\" }}b"}, Template: true}
	if _, err := renderMockRule(&rule, &mockRequest{Header: http.Header{}}); err == nil {
		t.Error("header with a line break rendered")
	}
}

// A render is bounded: loops other than repeat/queryAll/body and recursion are
// refused when the rule is saved, and every loop item counts against the budget.
func TestMockTemplate_Bounded(t *testing.T) {
	refused := []string{
		`{{ range 2000 }}{{ range 2000 }}{{ end }}{{ end }}`,
		`{{ range $i := add 1 2 }}{{ end }}`,
		`{{ range $x := now "unix" }}{{ end }}`,
		`{{ if true }}{{ range 5 }}{{ end }}{{ end }}`,
		`{{ with 1 }}{{ else }}{{ range 5 }}{{ end }}{{ end }}`,
		`{{ range repeat 2 }}{{ else }}{{ range 5 }}{{ end }}{{ end }}`,
		`{{ range (repeat 3) }}{{ end }}`,
		`{{ define "a" }}{{ fake "name" }}{{ end }}{{ template "a" }}`,
		`{{ template "x" }}`,
		`{{ block "b" . }}x{{ end }}`,
	}
	for _, src := range refused {
		r := MockRule{Path: "/", Body: src, Template: true}
		if verr, ok := r.Normalize().(ValidationError); !ok || verr["body"] == "" {
			t.Errorf("%s: saved", src)
		}
		if _, err := renderBody(t, r, &mockRequest{}); err == nil {
			t.Errorf("%s: rendered", src)
		}
	}

	req := func() *mockRequest {
		return &mockRequest{Query: url.Values{"tag": {"a", "b"}}, Body: []byte(`{"items":[{"id":1},{"id":2}],"obj":{"k":"v"}}`)}
	}
	allowed := map[string]string{
		`{{ range $i := repeat 3 }}{{ $i }}{{ end }}`:                `012`,
		`{{ range queryAll "tag" }}{{ . }}{{ end }}`:                 `ab`,
		`{{ range $t := queryAll "none" }}x{{ else }}none{{ end }}`:  `none`,
		`{{ range body "items" }}{{ .id }}{{ end }}`:                 `12`,
		`{{ range $k, $v := body "obj" }}{{ $k }}={{ $v }}{{ end }}`: `k=v`,
		`{{ range body "missing" }}x{{ end }}`:                       ``,
	}
	for src, want := range allowed {
		r := MockRule{Path: "/", Body: src, Template: true}
		if err := r.Normalize(); err != nil {
			t.Errorf("%s: %v", src, err)
		}
		if got, err := renderBody(t, r, req()); err != nil || got != want {
			t.Errorf("%s = %q, %v; want %q", src, got, err, want)
		}
	}

	// Ranging over the request's lists counts too.
	tags := url.Values{}
	for range 200 {
		tags.Add("tag", "x")
	}
	src := `{{ range queryAll "tag" }}{{ range queryAll "tag" }}{{ end }}{{ end }}`
	if _, err := renderBody(t, MockRule{Path: "/", Body: src, Template: true}, &mockRequest{Query: tags}); err == nil {
		t.Error("nested range over queryAll is not bounded")
	}
	items := "[" + strings.TrimSuffix(strings.Repeat("1,", 200), ",") + "]"
	src = `{{ range body "" }}{{ range body "" }}{{ end }}{{ end }}`
	if _, err := renderBody(t, MockRule{Path: "/", Body: src, Template: true}, &mockRequest{Body: []byte(items)}); err == nil {
		t.Error("nested range over body is not bounded")
	}
}

// Big templates stay bounded too: loop items cost their body's steps, and
// helper output counts even when it is not written.
func TestMockTemplate_WorkBudget(t *testing.T) {
	big := &mockRequest{Body: []byte(strings.Repeat("x", 100<<10))}
	step := strings.Repeat(`{{ $x := add 1 2 }}`, 200)
	cases := map[string]string{
		`{{ range repeat 10000 }}` + step + `{{ end }}`:                                                              "steps",
		`{{ range repeat 10000 }}{{ $x := fake "loremipsumparagraph" 20 20 50 " " }}{{ end }}`:                       "bytes",
		`{{ range repeat 1000 }}{{ $x := print bodyRaw }}{{ end }}`:                                                  "bytes",
		`{{ range repeat 1000 }}{{ $x := bodyRaw | json }}{{ end }}`:                                                 "bytes",
		`{{ range repeat 1000 }}{{ $x := upper bodyRaw }}{{ $y := html bodyRaw }}{{ end }}`:                          "bytes",
		`{{ range repeat 100 }}{{ range repeat 100 }}` + strings.Repeat(`{{ $x := 1 }}`, 150) + `{{ end }}{{ end }}`: "steps",
	}
	for src, want := range cases {
		start := time.Now()
		_, err := renderBody(t, MockRule{Path: "/", Body: src, Template: true}, big)
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%.60s…: %v, want a %s error", src, err, want)
		}
		if d := time.Since(start); d > 5*time.Second {
			t.Errorf("%.60s…: took %v", src, d)
		}
	}
	// Ordinary loops fit.
	ok := `[{{ range $i := repeat 5000 }}{{ if $i }},{{ end }}{"id": {{ $i }}, "name": {{ fake "firstname" | json }}}{{ end }}]`
	if out, err := renderBody(t, MockRule{Path: "/", Body: ok, Template: true}, &mockRequest{}); err != nil || !json.Valid([]byte(out)) {
		t.Errorf("5000 items: %v", err)
	}
}

// What the budgets do not price (comparing two 1 MB strings) stops at the
// render deadline.
func TestMockTemplate_Deadline(t *testing.T) {
	old := MaxMockRenderTime
	MaxMockRenderTime = 100 * time.Millisecond
	t.Cleanup(func() { MaxMockRenderTime = old })

	big := &mockRequest{Body: []byte(strings.Repeat("x", 1<<20))}
	src := `{{ range repeat 10000 }}` + strings.Repeat(`{{ if eq (slice bodyRaw 1) (slice bodyRaw 0 (add (len bodyRaw) -1)) }}{{ end }}`, 10) + `{{ end }}`
	start := time.Now()
	_, err := renderBody(t, MockRule{Path: "/", Body: src, Template: true}, big)
	if err == nil || !strings.Contains(err.Error(), "longer than") {
		t.Errorf("err = %v", err)
	}
	if d := time.Since(start); d > 2*time.Second {
		t.Errorf("took %v", d)
	}
	// Nested loops stop too.
	src = `{{ range repeat 100 }}{{ range repeat 99 }}{{ if eq (slice bodyRaw 1) (slice bodyRaw 0 (add (len bodyRaw) -1)) }}{{ end }}{{ end }}{{ end }}`
	if _, err := renderBody(t, MockRule{Path: "/", Body: src, Template: true}, big); err == nil {
		t.Error("nested: no error")
	}
}

func TestMockTemplate_Seed(t *testing.T) {
	src := `{{ fake "name" }} {{ fake "number" 1 1000000 }} {{ pick "a" "b" "c" "d" "e" "f" }}`
	render := func(seed int64) string {
		got, err := renderBody(t, MockRule{Path: "/", Body: src, Template: true, Seed: seed}, &mockRequest{})
		if err != nil {
			t.Fatal(err)
		}
		return got
	}
	if a, b := render(42), render(42); a != b {
		t.Errorf("seed 42: %q != %q", a, b)
	}
	if render(42) == render(43) {
		t.Error("seeds 42 and 43 give the same body")
	}
	if a, b := render(0), render(0); a == b {
		t.Errorf("seed 0 gave the same body twice: %q", a)
	}
}

func TestMockRuleNormalize_Template(t *testing.T) {
	r := MockRule{Path: "/a", Body: `{{ fake "name" `, Template: true}
	if verr, ok := r.Normalize().(ValidationError); !ok || verr["body"] == "" {
		t.Errorf("bad body template: %v", r.Normalize())
	}
	r = MockRule{Path: "/a", Headers: map[string]string{"X-A": `{{ nohelper }}`}, Template: true}
	if verr, ok := r.Normalize().(ValidationError); !ok || verr["headers"] == "" {
		t.Errorf("bad header template: %v", r.Normalize())
	}
	// Template off: the same text is a plain body.
	r = MockRule{Path: "/a", Body: `{{ fake "name" `}
	if err := r.Normalize(); err != nil {
		t.Errorf("plain body: %v", err)
	}
}

func TestMockServer_Template(t *testing.T) {
	env := newMockEnv(t, []MockRule{{Path: "/cfg/{id}", Body: `{"id": {{ path "id" }}}`, Template: true}})
	rule := env.create(t, `{"name":"user","method":"POST","path":"/users/{id}","template":true,"seed":7,
		"headers":{"X-Id":"{{ path \"id\" }}","Content-Type":"application/json"},
		"body":"{\"id\": {{ path \"id\" }}, \"email\": {{ body \"email\" | json }}, \"name\": {{ fake \"name\" | json }}}"}`)
	if !rule.Template || rule.Seed != 7 {
		t.Fatalf("created = %+v", rule)
	}

	reqBody := `{"email":"a@example.com","pad":"` + strings.Repeat("x", 100) + `"}`
	w := env.do(t, "POST", "/mock/users/42", reqBody, nil)
	var got map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil || w.Code != 200 ||
		got["id"] != float64(42) || got["email"] != "a@example.com" || got["name"] == "" || w.Header().Get("X-Id") != "42" {
		t.Fatalf("response %d %s %v", w.Code, w.Body.String(), w.Header())
	}
	// Seeded: the same body again.
	if w2 := env.do(t, "POST", "/mock/users/42", reqBody, nil); w2.Body.String() != w.Body.String() {
		t.Errorf("seeded bodies differ: %s / %s", w.Body.String(), w2.Body.String())
	}
	// The captured request still has the whole body the template read.
	ev := env.requests(t, "")[0]
	if b, _ := ev["payload"].(map[string]any)["request"].(map[string]any)["body"].(string); b != reqBody {
		t.Errorf("captured body = %q", b)
	}

	// Config rule with a template.
	if w := env.do(t, "GET", "/mock/cfg/5", "", nil); w.Body.String() != `{"id": 5}` {
		t.Errorf("config rule = %q", w.Body.String())
	}

	// HEAD: headers rendered, no body.
	env.create(t, `{"path":"/h","method":"HEAD","template":true,"headers":{"X-M":"{{ method }}"},"body":"{{ method }}"}`)
	if w := env.do(t, "HEAD", "/mock/h", "", nil); w.Body.Len() != 0 || w.Header().Get("X-M") != "HEAD" {
		t.Errorf("HEAD: %q %v", w.Body.String(), w.Header())
	}

	// Template off: {{ is served as is.
	env.create(t, `{"path":"/plain","body":"{{ fake \"name\" }}"}`)
	if w := env.do(t, "GET", "/mock/plain", "", nil); w.Body.String() != `{{ fake "name" }}` {
		t.Errorf("plain = %q", w.Body.String())
	}
}

func TestMockServer_TemplateRuntimeError(t *testing.T) {
	env := newMockEnv(t, nil)
	env.create(t, `{"name":"bad","path":"/bad","status":201,"template":true,"headers":{"X-A":"1"},"body":"{{ fake \"nope\" }}"}`)
	w := env.do(t, "GET", "/mock/bad", "", nil)
	if w.Code != http.StatusInternalServerError || !strings.Contains(w.Body.String(), `mock template:`) ||
		w.Header().Get("X-A") != "" || w.Header().Get("Content-Security-Policy") != "sandbox" {
		t.Fatalf("response %d %s %v", w.Code, w.Body.String(), w.Header())
	}
	info := mockField(t, env.requests(t, "")[0])
	if e, _ := info["template_error"].(string); !strings.Contains(e, "nope") || info["status"] != float64(500) {
		t.Errorf("mock = %v", info)
	}
}

func TestMockAPI_TemplateFields(t *testing.T) {
	env := newMockEnv(t, nil)

	// A parse error is a 422 on the body field.
	w := env.do(t, "POST", "/api/mocks", `{"path":"/a","template":true,"body":"{{ if }}"}`, nil)
	if w.Code != http.StatusUnprocessableEntity || !strings.Contains(w.Body.String(), `"body"`) {
		t.Errorf("bad template: %d %s", w.Code, w.Body.String())
	}

	// PUT without template/seed (an older client) keeps them; with them, sets them.
	rule := env.create(t, `{"path":"/a","template":true,"seed":9,"body":"{{ method }}"}`)
	put := func(body string) MockRule {
		w := env.do(t, "PUT", "/api/mocks/"+rule.ID, body, nil)
		if w.Code != http.StatusOK {
			t.Fatalf("put %s: %d %s", body, w.Code, w.Body.String())
		}
		var r MockRule
		json.Unmarshal(w.Body.Bytes(), &r)
		return r
	}
	if r := put(`{"path":"/a","body":"{{ method }}"}`); !r.Template || r.Seed != 9 {
		t.Errorf("kept = %+v", r)
	}
	if r := put(`{"path":"/a","body":"{{ method }}","template":false,"seed":0}`); r.Template || r.Seed != 0 {
		t.Errorf("cleared = %+v", r)
	}
	if w := env.do(t, "GET", "/api/mocks/"+rule.ID, "", nil); !strings.Contains(w.Body.String(), `"template":false`) {
		t.Errorf("stored = %s", w.Body.String())
	}
}

func TestMockAPI_Preview(t *testing.T) {
	env := newMockEnv(t, nil)
	preview := func(body string) (int, map[string]any) {
		w := env.do(t, "POST", "/api/mocks/preview", body, nil)
		var out map[string]any
		json.Unmarshal(w.Body.Bytes(), &out)
		return w.Code, out
	}

	code, out := preview(`{"rule":{"path":"/users/{id}","method":"POST","status":201,"template":true,
		"headers":{"X-Q":"{{ query \"q\" }}"},"body":"{{ path \"id\" }}-{{ body \"a\" }}-{{ header \"X-H\" }}-{{ method }}"},
		"request":{"path":"/mock/users/5?q=z","headers":{"X-H":"h"},"body":"{\"a\":1}"}}`)
	if code != 200 || out["body"] != "5-1-h-POST" || out["status"] != float64(201) || out["matched"] != true ||
		out["error"] != nil || out["headers"].(map[string]any)["X-Q"] != "z" {
		t.Errorf("preview: %d %v", code, out)
	}

	code, out = preview(`{"rule":{"path":"/x","template":true,"body":"{{ fake \"nope\" }}"},"request":{"path":"/y"}}`)
	if code != 200 || out["status"] != float64(500) || out["matched"] != false || !strings.Contains(out["error"].(string), "nope") {
		t.Errorf("runtime error: %d %v", code, out)
	}

	code, out = preview(`{"rule":{"path":"/x","body":"{{ plain"},"request":{"path":"/x"}}`)
	if code != 200 || out["body"] != "{{ plain" {
		t.Errorf("template off: %d %v", code, out)
	}

	if code, _ = preview(`{"rule":{"path":"/x","template":true,"body":"{{ if }}"},"request":{"path":"/x"}}`); code != 422 {
		t.Errorf("parse error: %d", code)
	}
	if code, _ = preview(`{"rule":{"path":"/x"},"request":{"path":"x"}}`); code != 422 {
		t.Errorf("bad request path: %d", code)
	}
}

func TestMockAPI_Helpers(t *testing.T) {
	env := newMockEnv(t, nil)
	w := env.do(t, "GET", "/api/mocks/helpers", "", nil)
	var out struct {
		Helpers []MockHelper `json:"helpers"`
		Fake    []MockFake   `json:"fake"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil || w.Code != 200 || len(out.Helpers) != len(MockHelpers) || len(out.Fake) == 0 {
		t.Fatalf("helpers: %d %v", w.Code, err)
	}
}

// Every documented helper exists, and every example parses and runs.
func TestMockHelpers_ExamplesRun(t *testing.T) {
	funcs := (&renderState{req: &mockRequest{}}).funcs()
	req := func() *mockRequest {
		return &mockRequest{Method: "POST", Path: "/users/42", Query: url.Values{"page": {"3"}, "tag": {"a"}},
			Header: http.Header{"X-Request-Id": {"abc"}}, Body: []byte(`{"user":{"email":"a@example.com"}}`)}
	}
	for _, h := range MockHelpers {
		if _, ok := funcs[h.Name]; !ok {
			t.Errorf("helper %s is documented but not defined", h.Name)
		}
		if _, err := renderBody(t, MockRule{Path: "/users/{id}", Body: h.Example, Template: true}, req()); err != nil {
			t.Errorf("example of %s: %v", h.Name, err)
		}
	}
	documented := map[string]bool{}
	for _, h := range MockHelpers {
		documented[h.Name] = true
	}
	for name := range funcs {
		if !documented[name] && !hiddenFuncs[name] {
			t.Errorf("helper %s is defined but not documented", name)
		}
	}
}
