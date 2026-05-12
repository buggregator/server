package smtp

import (
	"context"
	"encoding/json"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/buggregator/go-buggregator/internal/event"
)

// MessageFilter holds filter criteria for SMTP message queries.
type MessageFilter struct {
	To              string
	From            string
	Cc              string
	Subject         string
	SubjectContains string
	SubjectRegex    string
	BodyContains    string
	Project         string
	Since           float64 // unix timestamp (seconds)
	Until           float64 // unix timestamp (seconds)
	Limit           int
	Offset          int
	Order           string // "asc" or "desc"
}

// Link represents a URL extracted from a message.
type Link struct {
	URL    string `json:"url"`
	Text   string `json:"text,omitempty"`
	Source string `json:"source"` // "html" or "text"
}

func registerSMTPAPI(mux *http.ServeMux, store event.Store, mod *Module) {
	mux.HandleFunc("GET /api/smtp/cursor", handleCursor())
	mux.HandleFunc("GET /api/smtp/stats", handleStats(store))
	mux.HandleFunc("DELETE /api/smtp/messages", handleDeleteMessages(store))
	// /wait must be registered before the plain /messages pattern.
	mux.HandleFunc("GET /api/smtp/messages/wait", handleMessagesWait(store, mod))
	mux.HandleFunc("GET /api/smtp/messages", handleMessages(store))
	mux.HandleFunc("GET /api/smtp/message/{uuid}/raw", handleRaw(store))
	mux.HandleFunc("GET /api/smtp/message/{uuid}/links", handleLinks(store))
	mux.HandleFunc("GET /api/smtp/message/{uuid}/codes", handleCodes(store))
}

// handleCursor returns the current server time as a cursor token.
// E2E tests grab the cursor before triggering an action, then pass it as
// `since` to the search / wait endpoints to avoid picking up stale messages.
func handleCursor() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cursor := time.Now().UTC().Format(time.RFC3339Nano)
		smtpJSON(w, map[string]string{"cursor": cursor})
	}
}

// handleMessages returns SMTP messages matching the given filter.
func handleMessages(store event.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		f := parseFilter(r)

		events, err := store.FindAll(r.Context(), event.FindOptions{Type: "smtp", Project: f.Project})
		if err != nil {
			smtpError(w, err.Error(), http.StatusInternalServerError)
			return
		}

		filtered := applyFilter(events, f)

		if f.Order == "asc" {
			// Store returns desc; reverse for asc.
			for i, j := 0, len(filtered)-1; i < j; i, j = i+1, j-1 {
				filtered[i], filtered[j] = filtered[j], filtered[i]
			}
		}

		total := len(filtered)

		// Apply offset/limit after sorting.
		if f.Offset > 0 {
			if f.Offset >= len(filtered) {
				filtered = nil
			} else {
				filtered = filtered[f.Offset:]
			}
		}
		if f.Limit > 0 && len(filtered) > f.Limit {
			filtered = filtered[:f.Limit]
		}
		if filtered == nil {
			filtered = []event.Event{}
		}

		smtpJSON(w, map[string]any{
			"data": filtered,
			"meta": map[string]any{
				"total":  total,
				"limit":  f.Limit,
				"offset": f.Offset,
			},
		})
	}
}

// handleMessagesWait long-polls for a matching SMTP message.
// It holds the connection until a match arrives or the timeout expires.
// Returns 200 with the first matching event, or 408 on timeout.
func handleMessagesWait(store event.Store, mod *Module) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		f := parseFilter(r)

		// Parse timeout (default 30 s, max 60 s).
		timeout := 30 * time.Second
		if t := r.URL.Query().Get("timeout"); t != "" {
			if d, err := time.ParseDuration(t); err == nil && d > 0 {
				if d > 60*time.Second {
					d = 60 * time.Second
				}
				timeout = d
			}
		}

		// Subscribe BEFORE checking existing events to avoid the race where a
		// matching event arrives between the check and the wait.
		ch, unsub := mod.subscribe(f)
		defer unsub()

		// Check for an already-stored matching event.
		events, err := store.FindAll(r.Context(), event.FindOptions{Type: "smtp", Project: f.Project})
		if err != nil {
			smtpError(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if matched := applyFilter(events, f); len(matched) > 0 {
			smtpJSON(w, matched[0])
			return
		}

		// Wait for a new matching event.
		ctx, cancel := context.WithTimeout(r.Context(), timeout)
		defer cancel()

		select {
		case ev := <-ch:
			smtpJSON(w, ev)
		case <-ctx.Done():
			smtpError(w, "timeout waiting for message", http.StatusRequestTimeout)
		}
	}
}

// handleDeleteMessages purges SMTP messages, optionally filtered by project.
func handleDeleteMessages(store event.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		project := r.URL.Query().Get("project")
		opts := event.DeleteOptions{Type: "smtp", Project: project}
		if err := store.DeleteAll(r.Context(), opts); err != nil {
			smtpError(w, err.Error(), http.StatusInternalServerError)
			return
		}
		smtpJSON(w, map[string]any{"status": true})
	}
}

// handleStats returns count and last_received_at for SMTP messages.
func handleStats(store event.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		project := r.URL.Query().Get("project")
		events, err := store.FindAll(r.Context(), event.FindOptions{Type: "smtp", Project: project})
		if err != nil {
			smtpError(w, err.Error(), http.StatusInternalServerError)
			return
		}

		var lastReceivedAt *string
		if len(events) > 0 {
			// Events are ordered desc by timestamp; the first is the most recent.
			t := time.Unix(0, int64(events[0].Timestamp*1e9)).UTC().Format(time.RFC3339)
			lastReceivedAt = &t
		}

		smtpJSON(w, map[string]any{
			"count":            len(events),
			"last_received_at": lastReceivedAt,
		})
	}
}

// handleRaw returns the original RFC 822 source of an SMTP message.
func handleRaw(store event.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		uuid := r.PathValue("uuid")
		ev, err := store.FindByUUID(r.Context(), uuid)
		if err != nil {
			smtpError(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if ev == nil || ev.Type != "smtp" {
			smtpError(w, "message not found", http.StatusNotFound)
			return
		}

		var email ParsedEmail
		if err := json.Unmarshal(ev.Payload, &email); err != nil {
			smtpError(w, "failed to parse message", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "message/rfc822")
		w.Write([]byte(email.Raw)) //nolint:errcheck
	}
}

// handleLinks extracts every hyperlink from a message's HTML and text parts.
func handleLinks(store event.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		uuid := r.PathValue("uuid")
		ev, err := store.FindByUUID(r.Context(), uuid)
		if err != nil {
			smtpError(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if ev == nil || ev.Type != "smtp" {
			smtpError(w, "message not found", http.StatusNotFound)
			return
		}

		var email ParsedEmail
		if err := json.Unmarshal(ev.Payload, &email); err != nil {
			smtpError(w, "failed to parse message", http.StatusInternalServerError)
			return
		}

		links := extractLinks(&email)
		smtpJSON(w, map[string]any{"data": links})
	}
}

// handleCodes extracts codes (e.g. OTP digits) from a message using a regex pattern.
// Default pattern matches 4–8 digit sequences.
func handleCodes(store event.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		pattern := r.URL.Query().Get("pattern")
		if pattern == "" {
			pattern = `\b\d{4,8}\b`
		}

		re, err := regexp.Compile(pattern)
		if err != nil {
			smtpError(w, "invalid pattern: "+err.Error(), http.StatusBadRequest)
			return
		}

		uuid := r.PathValue("uuid")
		ev, err := store.FindByUUID(r.Context(), uuid)
		if err != nil {
			smtpError(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if ev == nil || ev.Type != "smtp" {
			smtpError(w, "message not found", http.StatusNotFound)
			return
		}

		var email ParsedEmail
		if err := json.Unmarshal(ev.Payload, &email); err != nil {
			smtpError(w, "failed to parse message", http.StatusInternalServerError)
			return
		}

		seen := make(map[string]bool)
		var codes []string
		for _, s := range []string{email.Text, stripHTMLTags(email.HTML)} {
			for _, m := range re.FindAllString(s, -1) {
				if !seen[m] {
					seen[m] = true
					codes = append(codes, m)
				}
			}
		}
		if codes == nil {
			codes = []string{}
		}

		smtpJSON(w, map[string]any{"data": codes, "pattern": pattern})
	}
}

// parseFilter extracts MessageFilter fields from query parameters.
func parseFilter(r *http.Request) MessageFilter {
	q := r.URL.Query()
	f := MessageFilter{
		To:              q.Get("to"),
		From:            q.Get("from"),
		Cc:              q.Get("cc"),
		Subject:         q.Get("subject"),
		SubjectContains: q.Get("subject_contains"),
		SubjectRegex:    q.Get("subject_regex"),
		BodyContains:    q.Get("body_contains"),
		Project:         q.Get("project"),
		Order:           q.Get("order"),
	}
	if f.Order == "" {
		f.Order = "desc"
	}
	if s := q.Get("since"); s != "" {
		f.Since = parseTimestamp(s)
	}
	if u := q.Get("until"); u != "" {
		f.Until = parseTimestamp(u)
	}
	if l := q.Get("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 {
			f.Limit = n
		}
	}
	if o := q.Get("offset"); o != "" {
		if n, err := strconv.Atoi(o); n >= 0 && err == nil {
			f.Offset = n
		}
	}
	return f
}

// unixFloat converts a time.Time to a float64 unix timestamp with microsecond
// precision (matching the internal event.Event.Timestamp representation).
func unixFloat(t time.Time) float64 {
	return float64(t.UnixMicro()) / 1_000_000
}

// parseTimestamp parses a timestamp from an RFC3339 string, unix-ms integer, or
// plain float (unix seconds). Returns 0 on failure.
func parseTimestamp(s string) float64 {
	if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
		return unixFloat(t)
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return unixFloat(t)
	}
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		// Heuristic: values > 1e12 are milliseconds, otherwise seconds.
		if f > 1e12 {
			return f / 1000
		}
		return f
	}
	return 0
}

// applyFilter filters a slice of events in-memory.
func applyFilter(events []event.Event, f MessageFilter) []event.Event {
	var result []event.Event
	for _, ev := range events {
		if matchesFilter(ev, f) {
			result = append(result, ev)
		}
	}
	return result
}

// matchesFilter returns true if ev satisfies all criteria in f.
func matchesFilter(ev event.Event, f MessageFilter) bool {
	if f.Since > 0 && ev.Timestamp < f.Since {
		return false
	}
	if f.Until > 0 && ev.Timestamp > f.Until {
		return false
	}

	var email ParsedEmail
	if err := json.Unmarshal(ev.Payload, &email); err != nil {
		return false
	}

	if f.To != "" && !matchAddresses(email.To, f.To) {
		return false
	}
	if f.From != "" && !matchAddresses(email.From, f.From) {
		return false
	}
	if f.Cc != "" && !matchAddresses(email.Cc, f.Cc) {
		return false
	}
	if f.Subject != "" && email.Subject != f.Subject {
		return false
	}
	if f.SubjectContains != "" && !strings.Contains(email.Subject, f.SubjectContains) {
		return false
	}
	if f.SubjectRegex != "" {
		re, err := regexp.Compile(f.SubjectRegex)
		if err != nil || !re.MatchString(email.Subject) {
			return false
		}
	}
	if f.BodyContains != "" &&
		!strings.Contains(email.Text, f.BodyContains) &&
		!strings.Contains(email.HTML, f.BodyContains) {
		return false
	}

	return true
}

func matchAddresses(addrs []EmailAddress, query string) bool {
	lq := strings.ToLower(query)
	for _, a := range addrs {
		if strings.Contains(strings.ToLower(a.Email), lq) ||
			strings.Contains(strings.ToLower(a.Name), lq) {
			return true
		}
	}
	return false
}

// extractLinks gathers all unique URLs from a message's HTML and text parts.
func extractLinks(email *ParsedEmail) []Link {
	var links []Link
	seen := make(map[string]bool)
	add := func(l Link) {
		if !seen[l.URL] {
			seen[l.URL] = true
			links = append(links, l)
		}
	}
	for _, l := range extractHTMLLinks(email.HTML) {
		add(l)
	}
	for _, l := range extractTextLinks(email.Text) {
		add(l)
	}
	if links == nil {
		links = []Link{}
	}
	return links
}

var (
	reHTMLLink = regexp.MustCompile(`(?is)<a[^>]+href=["']([^"']+)["'][^>]*>(.*?)</a>`)
	reHTMLTag  = regexp.MustCompile(`<[^>]+>`)
	reTextURL  = regexp.MustCompile(`https?://[^\s<>"']+`)
)

func extractHTMLLinks(htmlContent string) []Link {
	var links []Link
	for _, m := range reHTMLLink.FindAllStringSubmatch(htmlContent, -1) {
		href := m[1]
		text := strings.TrimSpace(reHTMLTag.ReplaceAllString(m[2], ""))
		links = append(links, Link{URL: href, Text: text, Source: "html"})
	}
	return links
}

func extractTextLinks(text string) []Link {
	var links []Link
	for _, m := range reTextURL.FindAllString(text, -1) {
		links = append(links, Link{URL: m, Source: "text"})
	}
	return links
}

func stripHTMLTags(s string) string {
	return reHTMLTag.ReplaceAllString(s, " ")
}

func smtpJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v) //nolint:errcheck
}

func smtpError(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]any{"message": msg, "code": code}) //nolint:errcheck
}
