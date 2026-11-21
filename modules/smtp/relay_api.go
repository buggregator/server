package smtp

import (
	"encoding/json"
	"errors"
	"mime"
	"net/http"

	"github.com/buggregator/server/v2/internal/event"
	"github.com/buggregator/server/v2/internal/server/sameorigin"
)

// handleRelayInfo tells the UI whether release is on and where it goes. The
// password is never sent.
//
//	GET /api/smtp/relay
func handleRelayInfo(rl *relay) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c := rl.cfg
		if !c.Enabled() {
			smtpJSON(w, map[string]any{"enabled": false})
			return
		}
		allowed := c.AllowedRecipients
		if allowed == nil {
			allowed = []string{}
		}
		smtpJSON(w, map[string]any{
			"enabled":            true,
			"host":               c.Host,
			"port":               c.port(),
			"tls":                c.tlsMode(),
			"auth":               c.Username != "",
			"from":               c.From,
			"allowed_recipients": allowed,
			"max_recipients":     relayMaxRecipients,
			"local_only":         !c.AuthOn, // without auth only this machine may release
		})
	}
}

// handleRelease sends one caught message to real mailboxes through the
// configured relay. Only the UI itself may call it (same origin, not over
// the WebSocket RPC); with auth on, any signed-in user.
//
//	POST /api/smtp/release/{uuid}  {"to": ["a@example.com"], "rewrite_to": false}
//
// The message goes as captured; rewrite_to replaces its To header with the
// recipients and drops Cc.
func handleRelease(store event.Store, rl *relay) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !sameorigin.Check(r) {
			smtpError(w, "only the Buggregator UI itself can release a message", http.StatusForbidden)
			return
		}
		// Without auth anyone who reaches the port could send mail through
		// the owner's account: only this machine may release then.
		if !rl.cfg.AuthOn && !sameorigin.Local(r) {
			smtpError(w, "without auth, release works only from this machine (localhost); Docker, another machine or a proxy need auth on", http.StatusForbidden)
			return
		}
		if !rl.cfg.Enabled() {
			smtpError(w, errRelayOff.Error(), http.StatusConflict)
			return
		}
		if mt, _, err := mime.ParseMediaType(r.Header.Get("Content-Type")); err != nil || mt != "application/json" {
			smtpError(w, "Content-Type must be application/json", http.StatusUnsupportedMediaType)
			return
		}
		var body struct {
			To        []string `json:"to"`
			RewriteTo bool     `json:"rewrite_to"`
		}
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(&body); err != nil {
			smtpError(w, `body must be {"to": ["user@example.com"], "rewrite_to": false}`, http.StatusBadRequest)
			return
		}
		to, err := rl.recipients(body.To)
		if err != nil {
			smtpError(w, err.Error(), http.StatusUnprocessableEntity)
			return
		}

		ev, err := store.FindByUUID(r.Context(), r.PathValue("uuid"))
		if err != nil {
			smtpError(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if ev == nil || ev.Type != "smtp" {
			smtpError(w, "message not found", http.StatusNotFound)
			return
		}
		var email ParsedEmail
		if err := json.Unmarshal(ev.Payload, &email); err != nil || email.Raw == "" {
			smtpError(w, "the message has no raw source to send", http.StatusUnprocessableEntity)
			return
		}
		from := ""
		if len(email.From) > 0 {
			from = email.From[0].Email
		}
		raw := []byte(email.Raw)
		if body.RewriteTo {
			raw = rewriteTo(raw, to)
		}

		if !rl.allow() {
			smtpError(w, errRelayRateLimited.Error(), http.StatusTooManyRequests)
			return
		}
		if err := rl.send(r.Context(), from, to, raw); err != nil {
			code := http.StatusBadGateway
			if errors.Is(err, errRelayOff) {
				code = http.StatusConflict
			}
			smtpError(w, err.Error(), code)
			return
		}
		smtpJSON(w, map[string]any{"status": true, "to": to})
	}
}
