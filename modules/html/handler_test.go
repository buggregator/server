package html

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/buggregator/server/v2/internal/event"
)

func req(path, detected, contentType, body string) *http.Request {
	r := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	if detected != "" {
		r.Header.Set("X-Buggregator-Detected-Type", detected)
	}
	if contentType != "" {
		r.Header.Set("Content-Type", contentType)
	}
	return r
}

func TestMatch(t *testing.T) {
	h := &handler{}
	cases := []struct {
		r    *http.Request
		want bool
	}{
		{req("/", "html", "text/html", "<b>x</b>"), true},
		{req("/html", "", "text/html", "<b>x</b>"), true},
		{req("/html/shop", "", "", "<b>x</b>"), true},
		{req("/", "", "text/html", "<b>x</b>"), false},              // untyped: http-dump
		{req("/html", "http-dump", "text/html", "<b>x</b>"), false}, // typed for another module
		{req("/htmlx", "", "text/html", "<b>x</b>"), false},
		{httptest.NewRequest(http.MethodGet, "/html", nil), false},
	}
	for i, c := range cases {
		if got := h.Match(c.r); got != c.want {
			t.Errorf("case %d (%s %s): match %v, want %v", i, c.r.Method, c.r.URL.Path, got, c.want)
		}
	}
}

func TestHandle(t *testing.T) {
	h := &handler{}
	cases := []struct {
		name, path, project, ct, body string
		title, html                   string
	}{
		{"raw html, title from page", "/", "", "text/html; charset=utf-8",
			"<html><head><title> Nightly  &amp; report </title></head><body><h1>OK</h1></body></html>",
			"Nightly & report", "<html><head><title> Nightly  &amp; report </title></head><body><h1>OK</h1></body></html>"},
		{"json with title", "/", "", "application/json", `{"title":"Trap page","html":"<a href=\"http://x\">open</a>"}`,
			"Trap page", `<a href="http://x">open</a>`},
		{"no content type", "/html/shop", "shop", "", "<p>hi</p>", "", "<p>hi</p>"},
		{"latin1 charset", "/", "", "text/html; charset=iso-8859-1", "<p>caf\xe9</p>", "", "<p>café</p>"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			inc, err := h.Handle(req(tc.path, "html", tc.ct, tc.body))
			if err != nil {
				t.Fatal(err)
			}
			var p Payload
			json.Unmarshal(inc.Payload, &p)
			if inc.Type != "html" || inc.Project != tc.project || p.Title != tc.title || p.HTML != tc.html || p.Size != len(tc.html) {
				t.Errorf("got type %q project %q payload %+v", inc.Type, inc.Project, p)
			}
		})
	}

	r := req("/", "html", "text/html", "<b>x</b>")
	r.Header.Set("X-Buggregator-Detected-Project", "billing")
	if inc, _ := h.Handle(r); inc.Project != "billing" {
		t.Errorf("project from html:billing@host: %q", inc.Project)
	}

	for name, r := range map[string]*http.Request{
		"empty":        req("/", "html", "text/html", "   "),
		"bad json":     req("/", "html", "application/json", "{"),
		"json no html": req("/", "html", "application/json", `{"title":"x"}`),
		"too big":      req("/", "html", "text/html", strings.Repeat("a", maxBody+1)),
	} {
		if _, err := h.Handle(r); err == nil {
			t.Errorf("%s accepted", name)
		}
	}
}

func TestPreview(t *testing.T) {
	m := &previewMapper{}
	small, _ := json.Marshal(Payload{Title: "T", HTML: "<style>p{}</style><script>alert(1)</script><p>Hello &amp; <b>bye</b></p>", Size: 10})
	out, _ := m.ToPreview(small)
	var p map[string]any
	json.Unmarshal(out, &p)
	if p["title"] != "T" || p["text"] != "Hello & bye" || p["html"] == nil || p["truncated"] != nil {
		t.Errorf("small preview %s", out)
	}
	big, _ := json.Marshal(Payload{Title: "B", HTML: strings.Repeat("<p>x</p>", previewHTMLMax), Size: 1})
	out, _ = m.ToPreview(big)
	p = nil
	json.Unmarshal(out, &p)
	if p["html"] != nil || p["truncated"] != true || len(p["text"].(string)) > 300*4 {
		t.Errorf("big preview has html or no truncated flag")
	}
	var _ event.SearchTextExtractor = m // found by GET /api/events?q=
	if s := m.SearchText(small); s != "T Hello & bye" {
		t.Errorf("searchable %q", s)
	}
}
