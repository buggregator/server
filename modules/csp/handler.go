package csp

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"strings"

	"github.com/buggregator/server/v2/internal/event"
)

// maxBody bounds one report request; browsers send a few KB.
const maxBody = 1 << 20

// Report is one report, the same shape for both formats.
type Report struct {
	// Type: csp-violation for CSP (both formats), else the Reporting API
	// type (deprecation, intervention, crash, network-error, ...).
	Type      string `json:"type"`
	URL       string `json:"url"` // the page that reported
	UserAgent string `json:"user_agent,omitempty"`
	// CSP violations.
	Directive   string `json:"directive,omitempty"` // effective directive: script-src-elem
	Blocked     string `json:"blocked,omitempty"`   // blocked URL, or inline / eval
	Disposition string `json:"disposition,omitempty"`
	SourceFile  string `json:"source_file,omitempty"`
	Line        int    `json:"line,omitempty"`
	Column      int    `json:"column,omitempty"`
	Sample      string `json:"sample,omitempty"`
	Policy      string `json:"policy,omitempty"`
	// Other report types: the message and id (deprecation, intervention).
	Message string `json:"message,omitempty"`
	ID      string `json:"id,omitempty"`
	// Body is the report as the browser sent it.
	Body json.RawMessage `json:"body"`
}

// Payload is the stored event: one request's reports.
type Payload struct {
	Format  string   `json:"format"` // csp-report or reports+json
	Reports []Report `json:"reports"`
}

type handler struct{}

func (h *handler) Priority() int { return 36 }

func reportType(r *http.Request) string {
	mt, _, _ := mime.ParseMediaType(r.Header.Get("Content-Type"))
	switch mt {
	case "application/csp-report":
		return "csp-report"
	case "application/reports+json":
		return "reports+json"
	}
	return ""
}

// Match: a POST with a report content type, or to /csp[/project].
func (h *handler) Match(r *http.Request) bool {
	if r.Method != http.MethodPost {
		return false
	}
	if r.Header.Get("X-Buggregator-Detected-Type") == "csp" || reportType(r) != "" {
		return true
	}
	return r.URL.Path == "/csp" || strings.HasPrefix(r.URL.Path, "/csp/")
}

func (h *handler) Handle(r *http.Request) (*event.Incoming, error) {
	body, err := io.ReadAll(io.LimitReader(r.Body, maxBody+1))
	if err != nil {
		return nil, err
	}
	if len(body) > maxBody {
		return nil, errors.New("csp: report over 1 MB")
	}
	p, err := parse(reportType(r), body)
	if err != nil {
		return nil, err
	}
	raw, err := json.Marshal(p)
	if err != nil {
		return nil, err
	}
	project := ""
	if rest, ok := strings.CutPrefix(r.URL.Path, "/csp/"); ok {
		project = strings.Trim(rest, "/")
	}
	return &event.Incoming{Type: "csp", Payload: raw, Project: project}, nil
}

// parse reads either format; with no content type (a POST to /csp) it
// guesses: an array is reports+json, an object csp-report.
func parse(format string, body []byte) (*Payload, error) {
	if format == "" {
		if strings.HasPrefix(strings.TrimSpace(string(body)), "[") {
			format = "reports+json"
		} else {
			format = "csp-report"
		}
	}
	p := &Payload{Format: format, Reports: []Report{}}
	switch format {
	case "csp-report":
		var doc struct {
			Report json.RawMessage `json:"csp-report"`
		}
		if err := json.Unmarshal(body, &doc); err != nil || len(doc.Report) == 0 {
			return nil, fmt.Errorf("csp: not a csp-report document")
		}
		var c struct {
			DocumentURI        string `json:"document-uri"`
			ViolatedDirective  string `json:"violated-directive"`
			EffectiveDirective string `json:"effective-directive"`
			OriginalPolicy     string `json:"original-policy"`
			BlockedURI         string `json:"blocked-uri"`
			SourceFile         string `json:"source-file"`
			LineNumber         int    `json:"line-number"`
			ColumnNumber       int    `json:"column-number"`
			Disposition        string `json:"disposition"`
			ScriptSample       string `json:"script-sample"`
		}
		if err := json.Unmarshal(doc.Report, &c); err != nil {
			return nil, fmt.Errorf("csp: %w", err)
		}
		dir := c.EffectiveDirective
		if dir == "" {
			dir = c.ViolatedDirective
		}
		p.Reports = append(p.Reports, Report{Type: "csp-violation", URL: c.DocumentURI, Directive: dir, Blocked: c.BlockedURI,
			Disposition: c.Disposition, SourceFile: c.SourceFile, Line: c.LineNumber, Column: c.ColumnNumber,
			Sample: c.ScriptSample, Policy: c.OriginalPolicy, Body: doc.Report})
	case "reports+json":
		var list []struct {
			Type      string          `json:"type"`
			URL       string          `json:"url"`
			UserAgent string          `json:"user_agent"`
			Body      json.RawMessage `json:"body"`
		}
		if err := json.Unmarshal(body, &list); err != nil {
			return nil, fmt.Errorf("csp: not a reports+json list: %w", err)
		}
		if len(list) == 0 {
			return nil, errors.New("csp: empty report list")
		}
		for _, it := range list {
			rep := Report{Type: it.Type, URL: it.URL, UserAgent: it.UserAgent, Body: it.Body}
			if len(rep.Body) == 0 {
				rep.Body = json.RawMessage("{}")
			}
			var b struct {
				EffectiveDirective string `json:"effectiveDirective"`
				BlockedURL         string `json:"blockedURL"`
				Disposition        string `json:"disposition"`
				SourceFile         string `json:"sourceFile"`
				LineNumber         int    `json:"lineNumber"`
				ColumnNumber       int    `json:"columnNumber"`
				Sample             string `json:"sample"`
				OriginalPolicy     string `json:"originalPolicy"`
				Message            string `json:"message"`
				ID                 string `json:"id"`
			}
			_ = json.Unmarshal(rep.Body, &b)
			rep.Directive, rep.Blocked, rep.Disposition = b.EffectiveDirective, b.BlockedURL, b.Disposition
			rep.SourceFile, rep.Line, rep.Column, rep.Sample, rep.Policy = b.SourceFile, b.LineNumber, b.ColumnNumber, b.Sample, b.OriginalPolicy
			rep.Message, rep.ID = b.Message, b.ID
			p.Reports = append(p.Reports, rep)
		}
	}
	return p, nil
}
