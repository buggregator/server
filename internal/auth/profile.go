package auth

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Limits of the per-user settings (PUT /api/me/settings).
const (
	maxSettingsKeys  = 64
	maxSettingKeyLen = 128
	maxSettingValue  = 16 << 10
	maxSettingsBody  = 256 << 10
)

// auditRetention is how long audit entries are kept.
const auditRetention = 90 * 24 * time.Hour

// openSession saves the user and opens a server-side session for the token
// about to be issued; it fills user.ID and user.SessionID.
func (h *Handlers) openSession(r *http.Request, user *User) error {
	id, err := h.store.Upsert(*user)
	if err != nil {
		return err
	}
	sid, err := h.store.CreateSession(id, r.UserAgent(), remoteIP(r), time.Now().Add(tokenExpiry))
	if err != nil {
		return err
	}
	user.ID, user.SessionID = id, sid
	return nil
}

// audit records an entry when the store is on; a failure is only logged.
func (h *Handlers) audit(r *http.Request, e AuditEntry) {
	if h.store == nil {
		return
	}
	recordAudit(h.store, r, e)
}

func recordAudit(store *UserStore, r *http.Request, e AuditEntry) {
	if e.IP == "" {
		e.IP = remoteIP(r)
	}
	if err := store.Record(e); err != nil {
		slog.Error("audit: record", "action", e.Action, "err", err)
	}
}

func (h *Handlers) isAdmin(u *User) bool {
	if u == nil || u.Provider == "guest" {
		return false
	}
	if len(h.admins) == 0 {
		return true
	}
	return h.admins[strings.ToLower(strings.TrimSpace(u.Email))]
}

// handleLogout ends the token's server-side session: the token stops working
// for the API and the WebSocket. Without sessions it does nothing.
func (h *Handlers) handleLogout(w http.ResponseWriter, r *http.Request) {
	user := UserFromContext(r.Context())
	if h.store != nil && user != nil && user.SessionID != "" {
		if err := h.store.Revoke(user.SessionID); err != nil {
			slog.Error("logout: revoke session", "err", err)
			writeJSONError(w, http.StatusInternalServerError, "logout failed")
			return
		}
		h.audit(r, AuditEntry{UserID: user.ID, UserLabel: labelOf(user), Action: "logout", Status: http.StatusNoContent})
		if h.onLogout != nil {
			h.onLogout()
		}
	}
	w.WriteHeader(http.StatusNoContent)
}

// handleGetSettings returns the signed-in user's settings: {"data": {key: value}}.
func (h *Handlers) handleGetSettings(w http.ResponseWriter, r *http.Request) {
	user := UserFromContext(r.Context())
	if user == nil || user.ID == "" {
		writeJSONError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	values, err := h.store.Settings(user.ID)
	if err != nil {
		slog.Error("settings: read", "err", err)
		writeJSONError(w, http.StatusInternalServerError, "settings read failed")
		return
	}
	writeJSON(w, map[string]any{"data": values})
}

// handlePutSettings merges the body's keys into the user's settings; a null
// value removes the key. Answers with all settings after the change.
func (h *Handlers) handlePutSettings(w http.ResponseWriter, r *http.Request) {
	user := UserFromContext(r.Context())
	if user == nil || user.ID == "" {
		writeJSONError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, maxSettingsBody+1))
	if err != nil || len(body) > maxSettingsBody {
		writeJSONError(w, http.StatusRequestEntityTooLarge, "settings body too large")
		return
	}
	var values map[string]json.RawMessage
	if err := json.Unmarshal(body, &values); err != nil || values == nil {
		writeJSONError(w, http.StatusBadRequest, "body must be a JSON object")
		return
	}
	for k, v := range values {
		if k == "" || len(k) > maxSettingKeyLen {
			writeJSONError(w, http.StatusBadRequest, "setting key must be 1-128 characters")
			return
		}
		if len(v) > maxSettingValue {
			writeJSONError(w, http.StatusRequestEntityTooLarge, "setting "+k+" is larger than 16 KB")
			return
		}
	}
	if err := h.store.PutSettings(user.ID, values, maxSettingsKeys); err != nil {
		if errors.Is(err, ErrTooManySettings) {
			writeJSONError(w, http.StatusBadRequest, "at most 64 settings per user")
			return
		}
		slog.Error("settings: write", "err", err)
		writeJSONError(w, http.StatusInternalServerError, "settings write failed")
		return
	}
	keys := make([]string, 0, len(values))
	for k := range values {
		keys = append(keys, k)
	}
	h.audit(r, AuditEntry{UserID: user.ID, UserLabel: labelOf(user), Action: "settings.update", Target: strings.Join(keys, ","), Status: http.StatusOK})
	h.handleGetSettings(w, r)
}

// handleAudit lists the audit log, newest first: ?limit= (1-500, default
// 100), ?before=<id> for the next page. Only admins (auth.admin_emails;
// none set: every signed-in user).
func (h *Handlers) handleAudit(w http.ResponseWriter, r *http.Request) {
	if !h.isAdmin(UserFromContext(r.Context())) {
		writeJSONError(w, http.StatusForbidden, "only admins may read the audit log")
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	before, _ := strconv.ParseInt(r.URL.Query().Get("before"), 10, 64)
	entries, err := h.store.ListAudit(limit, before)
	if err != nil {
		slog.Error("audit: list", "err", err)
		writeJSONError(w, http.StatusInternalServerError, "audit read failed")
		return
	}
	resp := map[string]any{"data": entries}
	if len(entries) > 0 {
		resp["next_before"] = entries[len(entries)-1].ID
	}
	writeJSON(w, resp)
}

// skipAudit reports routes that change nothing worth a record: a preview,
// debugger stepping (eval, which runs code, is kept), and the routes that
// record themselves.
func skipAudit(r *http.Request) bool {
	switch r.Method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return true
	}
	switch r.Pattern {
	case "POST /api/mocks/preview", "PUT /api/me/settings", "POST /api/auth/logout":
		return true
	case "POST /api/xdebug/sessions/{id}/eval":
		return false
	}
	// Debugger stepping: POST /api/xdebug/sessions/{id}/<command>.
	return r.Method == http.MethodPost && strings.HasPrefix(strings.TrimPrefix(r.Pattern, r.Method+" "), "/api/xdebug/sessions/{id}/")
}

// auditActions names the common routes; others are recorded by pattern.
var auditActions = map[string]string{
	"DELETE /api/events":        "events.clear",
	"DELETE /api/event/{uuid}":  "event.delete",
	"DELETE /api/sentry/all":    "sentry.clear",
	"DELETE /api/smtp/messages": "smtp.clear",
}

// AuditMiddleware wraps an auth middleware so every non-GET request on a
// protected route (clears, deletes, settings, mocks…) is written to the
// audit log with the user, the route and the answer's status.
func AuditMiddleware(store *UserStore, authMiddleware func(http.Handler) http.Handler) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if skipAudit(r) {
				next.ServeHTTP(w, r)
				return
			}
			target := r.URL.RequestURI()
			if r.Pattern == "DELETE /api/events" {
				target += clearSummary(r)
			}
			rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(rec, r)
			action := auditActions[r.Pattern]
			if action == "" {
				action = r.Pattern
			}
			if action == "" {
				action = r.Method + " " + r.URL.Path
			}
			e := AuditEntry{Action: action, Target: target, Status: rec.status}
			if u := UserFromContext(r.Context()); u != nil {
				e.UserID, e.UserLabel = u.ID, labelOf(u)
			}
			recordAudit(store, r, e)
		})
		return authMiddleware(inner)
	}
}

// clearSummary describes what a clear removes, from its JSON body (type,
// project, how many selected uuids, keep_pinned), and puts the body back for
// the handler: " type=sentry uuids=3 keep_pinned". Empty for "clear all".
func clearSummary(r *http.Request) string {
	if r.Body == nil {
		return ""
	}
	raw, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	r.Body.Close()
	r.Body = io.NopCloser(bytes.NewReader(raw))
	if err != nil {
		return ""
	}
	var body struct {
		Type       string   `json:"type"`
		Project    string   `json:"project"`
		UUIDs      []string `json:"uuids"`
		KeepPinned bool     `json:"keep_pinned"`
	}
	if json.Unmarshal(raw, &body) != nil {
		return ""
	}
	var b strings.Builder
	if body.Type != "" {
		b.WriteString(" type=" + body.Type)
	}
	if body.Project != "" {
		b.WriteString(" project=" + body.Project)
	}
	if len(body.UUIDs) > 0 {
		b.WriteString(" uuids=" + strconv.Itoa(len(body.UUIDs)))
	}
	if body.KeepPinned {
		b.WriteString(" keep_pinned")
	}
	return b.String()
}

// PruneLoop deletes audit entries older than 90 days and closed sessions,
// now and then once a day, until done is closed.
func PruneLoop(store *UserStore, done <-chan struct{}) {
	prune := func() {
		now := time.Now()
		if err := store.Prune(now.Add(-auditRetention), now); err != nil {
			slog.Error("audit: prune", "err", err)
		}
	}
	prune()
	t := time.NewTicker(24 * time.Hour)
	defer t.Stop()
	for {
		select {
		case <-done:
			return
		case <-t.C:
			prune()
		}
	}
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (s *statusRecorder) WriteHeader(code int) {
	s.status = code
	s.ResponseWriter.WriteHeader(code)
}

func (s *statusRecorder) Unwrap() http.ResponseWriter { return s.ResponseWriter }

// safeReturnTo accepts only a path of this server ("/#/events/…" or
// "/events/…"): no scheme, no host, no "//" or backslash a browser would
// read as another host.
func safeReturnTo(p string) (string, bool) {
	if p == "" || len(p) > 2048 || p[0] != '/' {
		return "", false
	}
	dec, err := url.PathUnescape(p)
	if err != nil {
		return "", false
	}
	for _, s := range []string{p, dec} {
		if strings.HasPrefix(s, "//") || strings.ContainsAny(s, "\\\r\n\t\x00") {
			return "", false
		}
	}
	u, err := url.Parse(p)
	if err != nil || u.Scheme != "" || u.Host != "" || u.User != nil {
		return "", false
	}
	return p, true
}

func labelOf(u *User) string {
	switch {
	case u == nil:
		return "guest"
	case u.Email != "":
		return u.Email
	case u.Username != "":
		return u.Username
	}
	return "guest"
}

func remoteIP(r *http.Request) string {
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return host
	}
	return r.RemoteAddr
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func writeJSONError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{"message": msg, "code": status})
}
