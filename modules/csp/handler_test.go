package csp

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
)

// Bodies as Chrome sends them.
const chromeCSPReport = `{"csp-report":{"document-uri":"https://shop.test/cart","referrer":"","violated-directive":"script-src-elem","effective-directive":"script-src-elem","original-policy":"default-src 'self'; report-uri /csp","disposition":"enforce","blocked-uri":"https://cdn.evil.test/x.js","line-number":12,"column-number":7,"source-file":"https://shop.test/cart","status-code":200,"script-sample":""}}`

const chromeReports = `[{"age":10,"body":{"blockedURL":"inline","columnNumber":39,"disposition":"enforce","documentURL":"https://shop.test/","effectiveDirective":"script-src-elem","lineNumber":121,"originalPolicy":"script-src 'self'; report-to csp","referrer":"","sample":"console.log(1)","sourceFile":"https://shop.test/","statusCode":200},"type":"csp-violation","url":"https://shop.test/","user_agent":"Mozilla/5.0 Chrome/131"},{"age":3,"body":{"columnNumber":5,"id":"PrefixedStorageInfo","lineNumber":2,"message":"'window.webkitStorageInfo' is deprecated.","sourceFile":"https://shop.test/app.js"},"type":"deprecation","url":"https://shop.test/","user_agent":"Mozilla/5.0 Chrome/131"}]`

func TestHandler(t *testing.T) {
	h := &handler{}
	tests := []struct {
		name, path, ctype, body string
		match                   bool
		project                 string
		check                   func(t *testing.T, p Payload)
	}{
		{"csp-report on any path", "/some/report", "application/csp-report", chromeCSPReport, true, "", func(t *testing.T, p Payload) {
			r := p.Reports[0]
			if p.Format != "csp-report" || len(p.Reports) != 1 || r.Type != "csp-violation" || r.Directive != "script-src-elem" ||
				r.Blocked != "https://cdn.evil.test/x.js" || r.Line != 12 || r.URL != "https://shop.test/cart" || r.Policy == "" {
				t.Errorf("%+v", p)
			}
		}},
		{"reports+json with project", "/csp/shop", "application/reports+json", chromeReports, true, "shop", func(t *testing.T, p Payload) {
			if p.Format != "reports+json" || len(p.Reports) != 2 {
				t.Fatalf("%+v", p)
			}
			c, d := p.Reports[0], p.Reports[1]
			if c.Blocked != "inline" || c.Sample != "console.log(1)" || c.Line != 121 || c.UserAgent == "" {
				t.Errorf("csp: %+v", c)
			}
			if d.Type != "deprecation" || d.ID != "PrefixedStorageInfo" || !strings.Contains(d.Message, "deprecated") || d.SourceFile == "" {
				t.Errorf("deprecation: %+v", d)
			}
		}},
		{"plain json to /csp", "/csp", "application/json", chromeCSPReport, true, "", func(t *testing.T, p Payload) {
			if p.Format != "csp-report" || p.Reports[0].Directive != "script-src-elem" {
				t.Errorf("%+v", p)
			}
		}},
		{"other json elsewhere", "/api/x", "application/json", `{}`, false, "", nil},
		{"/cspx is not ours", "/cspx", "application/json", `{}`, false, "", nil},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", tc.path, strings.NewReader(tc.body))
			req.Header.Set("Content-Type", tc.ctype)
			if got := h.Match(req); got != tc.match {
				t.Fatalf("match = %v", got)
			}
			if !tc.match {
				return
			}
			in, err := h.Handle(req)
			if err != nil {
				t.Fatal(err)
			}
			if in.Type != "csp" || in.Project != tc.project {
				t.Errorf("incoming %s/%s", in.Type, in.Project)
			}
			var p Payload
			if err := json.Unmarshal(in.Payload, &p); err != nil {
				t.Fatal(err)
			}
			tc.check(t, p)
		})
	}
	if h.Match(httptest.NewRequest("GET", "/csp", nil)) {
		t.Error("GET matched")
	}
	// A request typed for another module stays with it; typed csp is ours.
	for detected, want := range map[string]bool{"http-dump": false, "sentry": false, "csp": true} {
		req := httptest.NewRequest("POST", "/csp", strings.NewReader(chromeCSPReport))
		req.Header.Set("Content-Type", "application/csp-report")
		req.Header.Set("X-Buggregator-Detected-Type", detected)
		if got := h.Match(req); got != want {
			t.Errorf("detected %s: match = %v, want %v", detected, got, want)
		}
	}
	for _, bad := range []string{`{"x":1}`, `[]`, `not json`} {
		req := httptest.NewRequest("POST", "/csp", strings.NewReader(bad))
		if _, err := h.Handle(req); err == nil {
			t.Errorf("%s accepted", bad)
		}
	}
	if s := (&previewMapper{}).ToSearchableText(mustJSON(t, chromeReports)); !strings.Contains(s, "script-src-elem") || !strings.Contains(s, "deprecated") {
		t.Errorf("search text %q", s)
	}
}

func mustJSON(t *testing.T, reports string) json.RawMessage {
	t.Helper()
	p, err := parse("reports+json", []byte(reports))
	if err != nil {
		t.Fatal(err)
	}
	b, _ := json.Marshal(p)
	return b
}

// X-Buggregator-Project alone picks the project; /csp/<project> wins.
func TestHandler_ProjectHeader(t *testing.T) {
	h := &handler{}
	for _, tc := range []struct{ path, header, want string }{
		{"/csp", "shop", "shop"},
		{"/some/report", " shop ", "shop"},
		{"/csp/billing", "shop", "billing"},
		{"/csp", "", ""},
	} {
		req := httptest.NewRequest("POST", tc.path, strings.NewReader(chromeCSPReport))
		req.Header.Set("Content-Type", "application/csp-report")
		if tc.header != "" {
			req.Header.Set("X-Buggregator-Project", tc.header)
		}
		in, err := h.Handle(req)
		if err != nil {
			t.Fatal(err)
		}
		if in.Project != tc.want {
			t.Errorf("%s header %q: project %q, want %q", tc.path, tc.header, in.Project, tc.want)
		}
	}
}
