package html

import (
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"regexp"
	"strings"

	stdhtml "html"

	"github.com/buggregator/server/v2/internal/event"
	"golang.org/x/text/encoding/ianaindex"
)

// maxBody bounds one event's markup.
const maxBody = 1 << 20

// maxTitle bounds the title in runes.
const maxTitle = 300

// Payload is the stored event.
type Payload struct {
	Title string `json:"title"` // from JSON "title", else the page's <title>; may be empty
	HTML  string `json:"html"`
	Size  int    `json:"size"` // bytes of html
}

type handler struct{}

// Priority: before http-dump (9999), after the SDK modules.
func (h *handler) Priority() int { return 37 }

// Match: a POST typed html (html@host, X-Buggregator-Event: html, Basic
// Auth), or an untyped POST to /html[/project].
func (h *handler) Match(r *http.Request) bool {
	if r.Method != http.MethodPost {
		return false
	}
	switch r.Header.Get("X-Buggregator-Detected-Type") {
	case "html":
		return true
	case "":
	default:
		return false
	}
	return r.URL.Path == "/html" || strings.HasPrefix(r.URL.Path, "/html/")
}

func (h *handler) Handle(r *http.Request) (*event.Incoming, error) {
	body, err := io.ReadAll(io.LimitReader(r.Body, maxBody+1))
	if err != nil {
		return nil, err
	}
	if len(body) > maxBody {
		return nil, errors.New("html: body over 1 MB")
	}
	p, err := parse(r.Header.Get("Content-Type"), body)
	if err != nil {
		return nil, err
	}
	raw, err := json.Marshal(p)
	if err != nil {
		return nil, err
	}
	project := ""
	if rest, ok := strings.CutPrefix(r.URL.Path, "/html/"); ok {
		project = strings.Trim(rest, "/")
	}
	if project == "" {
		project = r.Header.Get("X-Buggregator-Detected-Project")
	}
	return &event.Incoming{Type: "html", Payload: raw, Project: project}, nil
}

// parse reads JSON {"title","html"} or a raw HTML body (any other content
// type, converted to UTF-8 from its charset).
func parse(contentType string, body []byte) (*Payload, error) {
	mt, params, _ := mime.ParseMediaType(contentType)
	var p Payload
	if mt == "application/json" {
		var in struct {
			Title string `json:"title"`
			HTML  string `json:"html"`
		}
		if err := json.Unmarshal(body, &in); err != nil {
			return nil, errors.New("html: JSON body must be {\"title\": \"…\", \"html\": \"…\"}")
		}
		p.Title, p.HTML = in.Title, in.HTML
	} else {
		p.HTML = string(toUTF8(body, params["charset"]))
	}
	if strings.TrimSpace(p.HTML) == "" {
		return nil, errors.New("html: empty markup")
	}
	if strings.TrimSpace(p.Title) == "" {
		p.Title = pageTitle(p.HTML)
	}
	p.Title = cut(strings.TrimSpace(p.Title), maxTitle)
	p.Size = len(p.HTML)
	return &p, nil
}

var titleTag = regexp.MustCompile(`(?is)<title[^>]*>(.*?)</title>`)

// pageTitle is the text of the markup's <title>, if any.
func pageTitle(markup string) string {
	m := titleTag.FindStringSubmatch(markup)
	if m == nil {
		return ""
	}
	return strings.Join(strings.Fields(stdhtml.UnescapeString(m[1])), " ")
}

var (
	hiddenBlocks = regexp.MustCompile(`(?is)<(script|style|head|template)\b.*?</(script|style|head|template)\s*>`)
	anyTag       = regexp.MustCompile(`(?s)<[^>]*>`)
)

// plainText is the visible text of the markup, whitespace collapsed, at most
// n runes: for the feed card and search.
func plainText(markup string, n int) string {
	s := hiddenBlocks.ReplaceAllString(markup, " ")
	s = anyTag.ReplaceAllString(s, " ")
	s = strings.Join(strings.Fields(stdhtml.UnescapeString(s)), " ")
	return cut(s, n)
}

func cut(s string, n int) string {
	if len(s) <= n {
		return s
	}
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n])
}

func toUTF8(data []byte, charset string) []byte {
	charset = strings.ToLower(strings.TrimSpace(charset))
	if charset == "" || charset == "utf-8" || charset == "us-ascii" {
		return data
	}
	enc, err := ianaindex.MIME.Encoding(charset)
	if err != nil || enc == nil {
		return data
	}
	if out, err := enc.NewDecoder().Bytes(data); err == nil {
		return out
	}
	return data
}
