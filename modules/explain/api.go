package explain

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
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
	h("GET /api/explain", m.handleStatus)
	h("GET /api/explain/event/{uuid}", m.handleGet)
	h("POST /api/explain/event/{uuid}", m.handleExplain)
	h("POST /api/explain/event/{uuid}/messages", m.handleFollowUp)
}

// handleStatus: whether the button can work now.
func (m *Module) handleStatus(w http.ResponseWriter, _ *http.Request) {
	c, status, reason := m.current()
	out := map[string]any{"enabled": true, "status": status, "to": m.cfg.To}
	if reason != "" {
		out["reason"] = reason
	}
	if c != nil {
		out["project"] = c.Welcome().Project
	}
	writeJSON(w, http.StatusOK, out)
}

func (m *Module) handleGet(w http.ResponseWriter, r *http.Request) {
	th, ok := m.threads.get(r.PathValue("uuid"))
	if !ok {
		writeErr(w, http.StatusNotFound, "explain_none", "no conversation for this event yet")
		return
	}
	writeJSON(w, http.StatusOK, th)
}

// handleExplain starts a conversation: the event's text (and the person's
// question, if any) goes to the configured address.
func (m *Module) handleExplain(w http.ResponseWriter, r *http.Request) {
	question, ok := readText(w, r, false)
	if !ok {
		return
	}
	ev, err := m.store.FindByUUID(r.Context(), r.PathValue("uuid"))
	if errors.Is(err, event.ErrNotFound) || (err == nil && ev == nil) {
		writeErr(w, http.StatusNotFound, "event_not_found", "event not found")
		return
	}
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "store", err.Error())
		return
	}
	if ev.Type != "sentry" {
		writeErr(w, http.StatusBadRequest, "explain_type", "only Sentry errors can be explained for now")
		return
	}
	c := m.connected(w)
	if c == nil {
		return
	}
	text, err := Prompt(ev.Payload, question, c.Welcome().Limits.TextBytes)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "explain_payload", err.Error())
		return
	}
	acc, ok := m.send(w, r.Context(), c, h2a.Message{To: m.cfg.To, Text: text})
	if !ok {
		return
	}
	th := m.threads.start(ev.UUID, ev.Project, acc.Conversation, Message{From: "you", Text: text, At: time.Now().UTC()})
	writeJSON(w, http.StatusOK, th)
}

// handleFollowUp writes the person's next question into the conversation.
func (m *Module) handleFollowUp(w http.ResponseWriter, r *http.Request) {
	text, ok := readText(w, r, true)
	if !ok {
		return
	}
	th, ok := m.threads.get(r.PathValue("uuid"))
	if !ok {
		writeErr(w, http.StatusNotFound, "explain_none", "no conversation for this event yet: explain it first")
		return
	}
	c := m.connected(w)
	if c == nil {
		return
	}
	if _, ok := m.send(w, r.Context(), c, h2a.Message{Conversation: th.Conversation, Text: text}); !ok {
		return
	}
	msg := Message{From: "you", Text: text, At: time.Now().UTC()}
	th, _ = m.threads.add(th.Conversation, msg)
	m.broadcast(th, msg)
	writeJSON(w, http.StatusOK, th)
}

func (m *Module) connected(w http.ResponseWriter) sender {
	c, status, reason := m.current()
	if c == nil {
		msg := "not connected to harnsy (" + status + ")"
		if reason != "" {
			msg += ": " + reason
		}
		writeErr(w, http.StatusServiceUnavailable, "explain_offline", msg)
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
		writeErr(w, http.StatusGatewayTimeout, "explain_timeout", "harnsy did not accept the message in time")
	default:
		writeErr(w, http.StatusBadGateway, "explain_send", err.Error())
	}
	return h2a.Accepted{}, false
}

// readText reads {"text": "..."}; an empty body is fine unless required.
func readText(w http.ResponseWriter, r *http.Request, required bool) (string, bool) {
	body, err := io.ReadAll(io.LimitReader(r.Body, maxQuestionBytes+1024))
	if err != nil {
		writeErr(w, http.StatusBadRequest, "bad_body", err.Error())
		return "", false
	}
	var in struct {
		Text string `json:"text"`
	}
	if len(strings.TrimSpace(string(body))) > 0 {
		if err := json.Unmarshal(body, &in); err != nil {
			writeErr(w, http.StatusBadRequest, "bad_body", "send JSON {\"text\": \"...\"}")
			return "", false
		}
	}
	in.Text = strings.TrimSpace(in.Text)
	if required && in.Text == "" {
		writeErr(w, http.StatusBadRequest, "text_required", "text is required")
		return "", false
	}
	if len(in.Text) > maxQuestionBytes {
		writeErr(w, http.StatusBadRequest, "text_too_long", "text is longer than 4000 bytes")
		return "", false
	}
	return in.Text, true
}
