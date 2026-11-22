package ws

import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/buggregator/server/v2/internal/auth"
	"github.com/buggregator/server/v2/internal/storage"
	"nhooyr.io/websocket"
)

// sessionHub is a hub with auth on and server-side sessions, its RPC bridged
// to a mux with the audited clear route, as app.go wires them.
func sessionHub(t *testing.T) (*Hub, *auth.UserStore, string, string) {
	t.Helper()
	db, err := storage.Open(filepath.Join(t.TempDir(), "bg.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	m := storage.NewMigrator(db)
	if err := m.AddFromFS("core", storage.CoreMigrations, "migrations"); err != nil {
		t.Fatal(err)
	}
	if err := m.Run(); err != nil {
		t.Fatal(err)
	}
	users := auth.NewUserStore(db)
	tokens := auth.NewTokenStore("secret")
	tokens.SetSessions(users)
	uid, _ := users.Upsert(auth.User{Provider: "test", Subject: "1", Email: "ann@acme.com"})
	sid, _ := users.CreateSession(uid, "", "", time.Now().Add(time.Hour))
	token, _ := tokens.Create(auth.User{Provider: "test", Email: "ann@acme.com", ID: uid, SessionID: sid})

	mux := http.NewServeMux()
	mw := auth.AuditMiddleware(users, auth.Middleware(tokens, true))
	mux.Handle("DELETE /api/events", mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})))
	h := NewHub()
	h.SetProjectProvider(func() []string { return []string{"default"} })
	h.SetTokenValidator(func(tok string) (time.Time, bool) {
		exp, err := tokens.ExpiresAt(tok)
		return exp, err == nil
	})
	h.SetRPCHandler(NewMuxRPCHandler(mux))
	return h, users, token, sid
}

// A clear sent over the WebSocket RPC (as the UI does) is audited with the
// socket's peer address and what it cleared, not httptest's 192.0.2.1.
func TestHub_RPCAuditedWithPeerIP(t *testing.T) {
	h, users, token, _ := sessionHub(t)
	c := dialHub(t, h)
	waitClients(t, h, 1)
	c.send(`{"id":1,"connect":{"token":"` + token + `"}}`)
	c.next()
	c.send(`{"id":2,"rpc":{"method":"delete:api/events","data":{"token":"` + token + `","type":"sentry","uuids":["a","b"],"keep_pinned":true}}}`)
	if reply := c.next(); !strings.Contains(toJSON(reply), `"code":204`) {
		t.Fatalf("rpc reply %v", reply)
	}
	entries, _ := users.ListAudit(1, 0)
	if len(entries) != 1 {
		t.Fatal("no audit entry")
	}
	e := entries[0]
	if e.IP != "127.0.0.1" || e.Action != "events.clear" || e.UserLabel != "ann@acme.com" {
		t.Errorf("entry %+v, want ip 127.0.0.1", e)
	}
	if e.Target != "/api/events type=sentry uuids=2 keep_pinned" {
		t.Errorf("target %q", e.Target)
	}
}

// After a logout revokes the session, RecheckTokens closes the open socket
// that connected with it (3500: centrifuge-js does not reconnect).
func TestHub_RecheckTokensClosesRevoked(t *testing.T) {
	h, users, token, sid := sessionHub(t)
	c := dialHub(t, h)
	waitClients(t, h, 1)
	c.send(`{"id":1,"connect":{"token":"` + token + `"}}`)
	c.next()

	// A valid socket stays and keeps answering.
	h.RecheckTokens()
	c.send(`{"id":2,"rpc":{"method":"delete:api/events","data":{"token":"` + token + `"}}}`)
	if reply := c.next(); !strings.Contains(toJSON(reply), `"code":204`) {
		t.Fatalf("valid socket after recheck: %v", reply)
	}
	users.Revoke(sid)
	h.RecheckTokens()
	if code := c.closedWith(); code != websocket.StatusCode(3500) {
		t.Errorf("close %v, want 3500", code)
	}
}

func toJSON(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}
