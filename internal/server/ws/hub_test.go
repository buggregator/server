package ws

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"nhooyr.io/websocket"
)

func TestHandleConnect_ServerChannels(t *testing.T) {
	h := NewHub()
	h.SetProjectProvider(func() []string { return []string{"default"} })
	h.AddServerChannel("xdebug")
	reply := h.handleConnect(1)
	for _, ch := range []string{"events.project.default", "xdebug"} {
		if _, ok := reply.Connect.Subs[ch]; !ok {
			t.Errorf("missing server-side subscription %q: %v", ch, reply.Connect.Subs)
		}
	}
}

// wsClient is a bare Centrifugo JSON-protocol client over a real socket.
type wsClient struct {
	t    *testing.T
	conn *websocket.Conn
}

func dialHub(t *testing.T, h *Hub) *wsClient {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(h.HandleUpgrade))
	t.Cleanup(srv.Close)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)
	conn, _, err := websocket.Dial(ctx, "ws"+strings.TrimPrefix(srv.URL, "http"), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close(websocket.StatusNormalClosure, "") })
	return &wsClient{t: t, conn: conn}
}

func (c *wsClient) send(v string) {
	c.t.Helper()
	if err := c.conn.Write(context.Background(), websocket.MessageText, []byte(v)); err != nil {
		c.t.Fatal(err)
	}
}

// next reads one frame within a second; nil when none came.
func (c *wsClient) next() map[string]any {
	c.t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_, data, err := c.conn.Read(ctx)
	if err != nil {
		return nil
	}
	var m map[string]any
	json.Unmarshal(data, &m)
	return m
}

// pushOf is the "push" object of a frame, empty when there is none.
func pushOf(m map[string]any) map[string]any {
	p, _ := m["push"].(map[string]any)
	if p == nil {
		return map[string]any{}
	}
	return p
}

// waitClients waits until the hub has n clients registered.
func waitClients(t *testing.T, h *Hub, n int) {
	t.Helper()
	for i := 0; i < 100; i++ {
		h.mu.RLock()
		got := len(h.clients)
		h.mu.RUnlock()
		if got == n {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("hub has no %d clients", n)
}

// A page that connected before a project existed gets that project's live
// events: the first publication on a new channel comes after a server-side
// subscribe push for it, once per client.
func TestHub_NewProjectReachesConnectedClient(t *testing.T) {
	h := NewHub()
	h.SetProjectProvider(func() []string { return []string{"default"} })
	c := dialHub(t, h)
	waitClients(t, h, 1)

	c.send(`{"id":1,"connect":{}}`)
	reply := c.next()
	subs, _ := reply["connect"].(map[string]any)["subs"].(map[string]any)
	if _, ok := subs["events.project.default"]; !ok || len(subs) != 1 {
		t.Fatalf("connect subs = %v", subs)
	}

	// A known channel: just the publication.
	h.Broadcast("events.project.default", "event.received", map[string]any{"n": 1})
	if m := c.next(); pushOf(m)["pub"] == nil {
		t.Fatalf("default: %v, want a publication", m)
	}

	// A project created after connect: subscribe push, then the publication.
	h.Broadcast("events.project.shop", "event.received", map[string]any{"n": 2})
	first, second := c.next(), c.next()
	if p := pushOf(first); p["channel"] != "events.project.shop" || p["subscribe"] == nil {
		t.Errorf("first frame = %v, want subscribe push for events.project.shop", first)
	}
	if p := pushOf(second); p["channel"] != "events.project.shop" || p["pub"] == nil {
		t.Errorf("second frame = %v, want the shop publication", second)
	}

	// Once subscribed, no second subscribe push.
	h.Broadcast("events.project.shop", "event.received", map[string]any{"n": 3})
	if m := c.next(); pushOf(m)["subscribe"] != nil {
		t.Errorf("repeated subscribe push: %v", m)
	}
	if m := c.next(); m != nil {
		t.Errorf("extra frame: %v", m)
	}
}

// Before its connect reply a client gets no subscribe push: the connect reply
// will carry the channels.
func TestHub_NoSubscribePushBeforeConnect(t *testing.T) {
	h := NewHub()
	c := dialHub(t, h)
	waitClients(t, h, 1)

	h.Broadcast("events.project.shop", "event.received", nil)
	if m := c.next(); m == nil || pushOf(m)["subscribe"] != nil {
		t.Errorf("frame before connect = %v, want only the publication", m)
	}
}
