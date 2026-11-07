package ws

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/buggregator/go-buggregator/internal/auth"
	"nhooyr.io/websocket"
)

// authHub is a hub with auth on, checking tokens of a real TokenStore.
func authHub(t *testing.T) (*Hub, string) {
	t.Helper()
	tokens := auth.NewTokenStore("secret")
	token, err := tokens.Create(auth.User{Provider: "test", Username: "dev"})
	if err != nil {
		t.Fatal(err)
	}
	h := NewHub()
	h.SetProjectProvider(func() []string { return []string{"default"} })
	h.SetTokenValidator(func(tok string) (time.Time, bool) {
		exp, err := tokens.ExpiresAt(tok)
		return exp, err == nil
	})
	return h, token
}

// closedWith reads until the socket closes and returns its close status.
func (c *wsClient) closedWith() websocket.StatusCode {
	c.t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	for {
		if _, _, err := c.conn.Read(ctx); err != nil {
			var ce websocket.CloseError
			if errors.As(err, &ce) {
				return ce.Code
			}
			return -1
		}
	}
}

func TestHub_AuthOn_ConnectRefused(t *testing.T) {
	for name, connect := range map[string]string{
		"no token":     `{"id":1,"connect":{}}`,
		"forged token": `{"id":1,"connect":{"token":"forged"}}`,
	} {
		t.Run(name, func(t *testing.T) {
			h, _ := authHub(t)
			c := dialHub(t, h)
			waitClients(t, h, 1)

			// Before connect: no push and no answer to commands, so the
			// first frame the client reads is the connect error.
			h.Broadcast("events.project.default", "event.received", map[string]any{"secret": 1})
			c.send(`{"id":5,"rpc":{"method":"get:api/events"}}`)

			c.send(connect)
			reply := c.next()
			e, _ := reply["error"].(map[string]any)
			if e == nil || e["code"] != float64(109) || reply["connect"] != nil {
				t.Fatalf("connect reply = %v, want error 109 and no subs", reply)
			}
			if code := c.closedWith(); code != websocket.StatusPolicyViolation {
				t.Errorf("close status = %v, want 1008", code)
			}
		})
	}
}

func TestHub_AuthOn_ValidTokenGetsEvents(t *testing.T) {
	h, token := authHub(t)
	c := dialHub(t, h)
	waitClients(t, h, 1)

	c.send(`{"id":1,"connect":{"token":"` + token + `"}}`)
	reply := c.next()
	subs, _ := reply["connect"].(map[string]any)["subs"].(map[string]any)
	if _, ok := subs["events.project.default"]; !ok {
		t.Fatalf("connect reply = %v, want subs", reply)
	}

	h.Broadcast("events.project.default", "event.received", map[string]any{"n": 1})
	if m := c.next(); pushOf(m)["pub"] == nil {
		t.Fatalf("frame = %v, want the publication", m)
	}
}

// RPC over the socket carries the payload token as X-Auth-Token, so a
// protected route bridged to answers as over HTTP.
func TestMuxRPCHandler_ProtectedRoute(t *testing.T) {
	tokens := auth.NewTokenStore("secret")
	token, _ := tokens.Create(auth.User{Provider: "test", Username: "dev"})
	mux := http.NewServeMux()
	mux.Handle("DELETE /api/events", auth.Middleware(tokens, true)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"status":true}`))
	})))
	rpc := NewMuxRPCHandler(mux)

	for _, tc := range []struct {
		name, data string
		want       float64
	}{
		{"no token", `{}`, 401},
		{"forged token", `{"token":"forged"}`, 401},
		{"valid token", `{"token":"` + token + `"}`, 200},
	} {
		out, err := rpc.HandleRPC("DELETE", "api/events", json.RawMessage(tc.data))
		if err != nil {
			t.Fatal(err)
		}
		var res map[string]any
		json.Unmarshal(out, &res)
		if res["code"] != tc.want {
			t.Errorf("%s: code = %v, want %v", tc.name, res["code"], tc.want)
		}
	}
}

// With auth on, a socket that never sends connect is closed after the
// connect timeout.
func TestHub_AuthOn_ConnectDeadline(t *testing.T) {
	h, _ := authHub(t)
	h.connectTimeout = 100 * time.Millisecond
	c := dialHub(t, h)

	if code := c.closedWith(); code != websocket.StatusPolicyViolation {
		t.Errorf("close status = %v, want 1008 after the connect timeout", code)
	}
}

// A connected socket passes the deadline untouched.
func TestHub_AuthOn_ConnectedSocketOutlivesDeadline(t *testing.T) {
	h, token := authHub(t)
	h.connectTimeout = 100 * time.Millisecond
	c := dialHub(t, h)
	waitClients(t, h, 1)

	c.send(`{"id":1,"connect":{"token":"` + token + `"}}`)
	if reply := c.next(); reply["connect"] == nil {
		t.Fatalf("connect reply = %v", reply)
	}
	time.Sleep(300 * time.Millisecond)
	h.Broadcast("events.project.default", "event.received", map[string]any{"n": 1})
	if m := c.next(); pushOf(m)["pub"] == nil {
		t.Fatalf("frame after deadline = %v, want the publication", m)
	}
}

// With auth on, the socket is closed with 3500 when its token expires.
func TestHub_AuthOn_TokenExpiryCloses(t *testing.T) {
	h := NewHub()
	h.SetTokenValidator(func(tok string) (time.Time, bool) {
		return time.Now().Add(200 * time.Millisecond), tok == "short"
	})
	c := dialHub(t, h)
	waitClients(t, h, 1)

	c.send(`{"id":1,"connect":{"token":"short"}}`)
	if reply := c.next(); reply["connect"] == nil {
		t.Fatalf("connect reply = %v", reply)
	}
	if code := c.closedWith(); code != websocket.StatusCode(3500) {
		t.Errorf("close status = %v, want 3500 at token expiry", code)
	}
}

// With auth off there is no deadline: a socket without connect still gets
// broadcasts, as before.
func TestHub_AuthOff_NoConnectDeadline(t *testing.T) {
	h := NewHub()
	h.connectTimeout = 50 * time.Millisecond
	c := dialHub(t, h)
	waitClients(t, h, 1)

	time.Sleep(200 * time.Millisecond)
	h.Broadcast("events.project.default", "event.received", nil)
	if m := c.next(); pushOf(m)["pub"] == nil {
		t.Fatalf("frame = %v, want the publication", m)
	}
}
