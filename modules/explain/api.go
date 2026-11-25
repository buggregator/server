package explain

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/buggregator/server/v2/internal/event"
	"github.com/harnesy/h2a-go"
)

const (
	maxQuestionBytes = 4000
	sendTimeout      = 30 * time.Second
)

type apiError struct {
	Code  string `json:"code"`
	Error string `json:"error"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(v)
}

func writeErr(w http.ResponseWriter, status int, code, msg string) {
	writeJSON(w, status, apiError{Code: code, Error: msg})
}

func (m *Module) registerAPI(mux *http.ServeMux, wrap func(http.Handler) http.Handler) {
	h := func(pattern string, fn http.HandlerFunc) { mux.Handle(pattern, wrap(fn)) }
	h("GET /api/assist", m.handleStatus)
	h("GET /api/assist/threads", m.handleFind)
	h("POST /api/assist/threads", m.handleStart)
	h("GET /api/assist/threads/{id}", m.handleThread)
	h("POST /api/assist/threads/{id}/messages", m.handleMessage)

	// Aliases of the first concept (event threads only), until the UI moves.
	h("GET /api/explain", m.handleStatus)
	h("GET /api/explain/event/{uuid}", m.handleLegacyGet)
	h("POST /api/explain/event/{uuid}", m.handleLegacyStart)
	h("POST /api/explain/event/{uuid}/messages", m.handleLegacyMessage)
}

// handleStatus: whether the panel can talk now, the node's limits and
// whether the agents can read Buggregator over its MCP.
func (m *Module) handleStatus(w http.ResponseWriter, _ *http.Request) {
	c, status, reason := m.current()
	out := map[string]any{
		"enabled": true, "status": status, "to": m.cfg.To,
		"mcp": map[string]any{"on": m.cfg.MCP.On, "read_only": m.cfg.MCP.ReadOnly, "transport": m.cfg.MCP.Transport},
	}
	if reason != "" {
		out["reason"] = reason
	}
	if c != nil {
		wel := c.Welcome()
		out["project"] = wel.Project
		out["limits"] = map[string]int{
			"text_bytes":        wel.Limits.TextBytes,
			"frames_per_minute": wel.Limits.FramesPerMinute,
			"messages_per_day":  wel.Limits.MessagesPerDay,
			"queue":             wel.Limits.Queue,
		}
	}
	writeJSON(w, http.StatusOK, out)
}

// handleFind: ?kind=&key= gives that context's thread, ?recent=N the last
// threads.
func (m *Module) handleFind(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	if q.Has("recent") {
		n, _ := strconv.Atoi(q.Get("recent"))
		list, err := m.threads.recent(n)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, "store", err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"threads": list})
		return
	}
	if q.Get("kind") == "" || q.Get("key") == "" {
		writeErr(w, http.StatusBadRequest, "bad_query", "give kind and key, or recent")
		return
	}
	// A context without a conversation is the usual case when the panel
	// opens, not an error: 200 {"thread": null}, no 404 in the console.
	th, err := m.threads.byKey(q.Get("kind"), q.Get("key"))
	if errors.Is(err, errNoThread) {
		writeJSON(w, http.StatusOK, map[string]any{"thread": nil})
		return
	}
	m.writeThread(w, th, err)
}

func (m *Module) handleThread(w http.ResponseWriter, r *http.Request) {
	id, ok := threadID(w, r)
	if !ok {
		return
	}
	th, err := m.threads.byID(id)
	m.writeThread(w, th, err)
}

func (m *Module) writeThread(w http.ResponseWriter, th Thread, err error) {
	switch {
	case errors.Is(err, errNoThread):
		writeErr(w, http.StatusNotFound, "assist_none", "no conversation for this context yet")
	case err != nil:
		writeErr(w, http.StatusInternalServerError, "store", err.Error())
	default:
		writeJSON(w, http.StatusOK, th)
	}
}

type startRequest struct {
	Context Context `json:"context"`
	Text    string  `json:"text"`
	New     bool    `json:"new"`
}

// handleStart opens a conversation: the context's text (and the person's
// question) goes to the configured address. A context that already has a
// conversation answers 409 with it, unless "new" asks to start over.
func (m *Module) handleStart(w http.ResponseWriter, r *http.Request) {
	var in startRequest
	if !readJSON(w, r, maxPageBytes+maxQuestionBytes+4096, &in) {
		return
	}
	if err := in.Context.check(); err != nil {
		code := "bad_context"
		if errors.Is(err, errContextTooLong) {
			code = "context_too_long"
		}
		writeErr(w, http.StatusBadRequest, code, err.Error())
		return
	}
	question, ok := checkText(w, in.Text, false)
	if !ok {
		return
	}
	m.start(w, r.Context(), in.Context, question, in.New)
}

func (m *Module) start(w http.ResponseWriter, ctx context.Context, c Context, question string, fresh bool) {
	if !fresh {
		th, err := m.threads.byKey(c.Kind, c.Key)
		if err == nil {
			writeJSON(w, http.StatusConflict, map[string]any{"code": "assist_exists", "error": "this context already has a conversation", "thread": th})
			return
		}
		if !errors.Is(err, errNoThread) {
			writeErr(w, http.StatusInternalServerError, "store", err.Error())
			return
		}
	}
	var ev *event.Event
	project := c.Project
	if c.Kind == KindEvent {
		found, err := m.store.FindByUUID(ctx, c.Key)
		if errors.Is(err, event.ErrNotFound) || (err == nil && found == nil) {
			writeErr(w, http.StatusNotFound, "event_not_found", "event not found")
			return
		}
		if err != nil {
			writeErr(w, http.StatusInternalServerError, "store", err.Error())
			return
		}
		ev, project = found, found.Project
	}
	if project == "" {
		project = "default"
	}
	cl := m.connected(w)
	if cl == nil {
		return
	}
	text, err := contextText(c, ev, question, pointer{MCP: m.cfg.MCP.On && m.cfg.MCP.ReadOnly}, cl.Welcome().Limits.TextBytes)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "assist_payload", err.Error())
		return
	}
	acc, ok := m.send(w, ctx, cl, h2a.Message{To: m.cfg.To, Text: text})
	if !ok {
		return
	}
	now := time.Now().UTC()
	first := []Message{{From: "you", Text: text, Context: true, At: now}}
	if question != "" {
		first = append(first, Message{From: "you", Text: question, At: now})
	}
	th, err := m.threads.start(c, project, acc.Conversation, first)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "store", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, th)
}

// handleMessage writes the person's next words into the conversation.
func (m *Module) handleMessage(w http.ResponseWriter, r *http.Request) {
	id, ok := threadID(w, r)
	if !ok {
		return
	}
	var in struct {
		Text string `json:"text"`
	}
	if !readJSON(w, r, maxQuestionBytes+1024, &in) {
		return
	}
	text, ok := checkText(w, in.Text, true)
	if !ok {
		return
	}
	th, err := m.threads.byID(id)
	if err != nil {
		m.writeThread(w, th, err)
		return
	}
	m.followUp(w, r.Context(), th, text)
}

func (m *Module) followUp(w http.ResponseWriter, ctx context.Context, th Thread, text string) {
	c := m.connected(w)
	if c == nil {
		return
	}
	if _, ok := m.send(w, ctx, c, h2a.Message{Conversation: th.Conversation, Text: scrub(text)}); !ok {
		return
	}
	msg := Message{From: "you", Text: text, At: time.Now().UTC()}
	th, err := m.threads.add(th.ID, msg)
	if err != nil {
		m.writeThread(w, th, err)
		return
	}
	m.broadcast(th, msg)
	writeJSON(w, http.StatusOK, th)
}

func (m *Module) handleLegacyGet(w http.ResponseWriter, r *http.Request) {
	th, err := m.threads.byKey(KindEvent, r.PathValue("uuid"))
	m.writeThread(w, th, err)
}

// handleLegacyStart: the first concept's «Explain» always started over.
func (m *Module) handleLegacyStart(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Text string `json:"text"`
	}
	if !readJSON(w, r, maxQuestionBytes+1024, &in) {
		return
	}
	question, ok := checkText(w, in.Text, false)
	if !ok {
		return
	}
	m.start(w, r.Context(), Context{Kind: KindEvent, Key: r.PathValue("uuid")}, question, true)
}

func (m *Module) handleLegacyMessage(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Text string `json:"text"`
	}
	if !readJSON(w, r, maxQuestionBytes+1024, &in) {
		return
	}
	text, ok := checkText(w, in.Text, true)
	if !ok {
		return
	}
	th, err := m.threads.byKey(KindEvent, r.PathValue("uuid"))
	if err != nil {
		m.writeThread(w, th, err)
		return
	}
	m.followUp(w, r.Context(), th, text)
}

func threadID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		writeErr(w, http.StatusNotFound, "assist_none", "no such conversation")
		return 0, false
	}
	return id, true
}

func (m *Module) connected(w http.ResponseWriter) sender {
	c, status, reason := m.current()
	if c == nil {
		msg := "not connected to harnsy (" + status + ")"
		if reason != "" {
			msg += ": " + reason
		}
		writeErr(w, http.StatusServiceUnavailable, "assist_offline", msg)
	}
	return c
}

func (m *Module) send(w http.ResponseWriter, ctx context.Context, c sender, msg h2a.Message) (h2a.Accepted, bool) {
	ctx, cancel := context.WithTimeout(ctx, sendTimeout)
	defer cancel()
	acc, err := c.Send(ctx, msg)
	if err == nil {
		return acc, true
	}
	var he *h2a.Error
	switch {
	case errors.As(err, &he):
		status := http.StatusBadGateway
		switch he.Code {
		case h2a.CodeRate, h2a.CodeDaily:
			status = http.StatusTooManyRequests
		case h2a.CodeNoHolder:
			status = http.StatusServiceUnavailable
		}
		writeErr(w, status, he.Code, err.Error())
	case errors.Is(err, context.DeadlineExceeded):
		writeErr(w, http.StatusGatewayTimeout, "assist_timeout", "harnsy did not accept the message in time")
	default:
		writeErr(w, http.StatusBadGateway, "assist_send", err.Error())
	}
	return h2a.Accepted{}, false
}

// readJSON reads a JSON body of at most max bytes; an empty body is {}.
func readJSON(w http.ResponseWriter, r *http.Request, max int64, v any) bool {
	body, err := io.ReadAll(io.LimitReader(r.Body, max+1))
	if err != nil {
		writeErr(w, http.StatusBadRequest, "bad_body", err.Error())
		return false
	}
	if int64(len(body)) > max {
		writeErr(w, http.StatusRequestEntityTooLarge, "body_too_large", "the request body is too large")
		return false
	}
	if len(strings.TrimSpace(string(body))) == 0 {
		return true
	}
	if err := json.Unmarshal(body, v); err != nil {
		writeErr(w, http.StatusBadRequest, "bad_body", "send a JSON object: "+err.Error())
		return false
	}
	return true
}

// checkText trims the person's words; empty is fine unless required.
func checkText(w http.ResponseWriter, text string, required bool) (string, bool) {
	text = strings.TrimSpace(text)
	if required && text == "" {
		writeErr(w, http.StatusBadRequest, "text_required", "text is required")
		return "", false
	}
	if len(text) > maxQuestionBytes {
		writeErr(w, http.StatusBadRequest, "text_too_long", "text is longer than 4000 bytes")
		return "", false
	}
	return text, true
}
