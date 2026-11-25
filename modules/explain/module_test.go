package explain

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/buggregator/server/v2/internal/event"
	"github.com/buggregator/server/v2/internal/module"
	"github.com/buggregator/server/v2/internal/storage"
	"github.com/buggregator/server/v2/modules/sentry"
	"github.com/coder/websocket"
)

const testToken = "h2a_TESTtokenTESTtokenTESTtoken0123456789"

// fakeNode speaks the node's side of h2a v1, enough for the module: it signs
// the hello in, accepts messages (a new conversation per message without
// one) and sends the replies a test gives it.
type fakeNode struct {
	t   *testing.T
	srv *httptest.Server

	mu       sync.Mutex
	conn     *websocket.Conn
	messages []map[string]any // message frames the service sent
	acked    int64
	seq      int64
	convs    int
	ready    chan struct{}
	drop     int // close this many connections right after the welcome
	hellos   int
}

func newFakeNode(t *testing.T) *fakeNode {
	n := &fakeNode{t: t, ready: make(chan struct{})}
	n.srv = httptest.NewServer(http.HandlerFunc(n.serve))
	t.Cleanup(n.srv.Close)
	return n
}

func (n *fakeNode) url() string { return "ws" + strings.TrimPrefix(n.srv.URL, "http") + "/h2a/v1" }

func (n *fakeNode) serve(w http.ResponseWriter, r *http.Request) {
	c, err := websocket.Accept(w, r, &websocket.AcceptOptions{Subprotocols: []string{"harnsy.h2a.v1"}})
	if err != nil {
		return
	}
	defer c.CloseNow()
	ctx := r.Context()
	var hello map[string]any
	if !n.read(ctx, c, &hello) || hello["type"] != "hello" {
		return
	}
	if hello["token"] != testToken {
		n.write(ctx, c, map[string]any{"type": "error", "code": "h2a_auth", "text": "the token is wrong"})
		_ = c.Close(4401, "h2a_auth")
		return
	}
	n.write(ctx, c, map[string]any{
		"type": "welcome", "v": "1", "project": "App",
		"to":     []map[string]string{{"address": "lead", "name": "Project lead"}},
		"limits": map[string]int{"frames_per_minute": 60, "messages_per_day": 100, "text_bytes": 16384, "frame_bytes": 262144, "queue": 500, "queue_hours": 24, "ping_seconds": 20, "idle_seconds": 60},
	})
	n.mu.Lock()
	n.hellos++
	if n.drop > 0 {
		n.drop--
		n.mu.Unlock()
		// A refusal without a frame's id: Run returns it, a reason that passes.
		n.write(ctx, c, map[string]any{"type": "error", "code": "h2a_frame", "text": "a frame is one JSON object of a known type"})
		_ = c.Close(websocket.StatusNormalClosure, "")
		return
	}
	n.conn = c
	first := n.ready != nil
	ready := n.ready
	n.ready = nil
	n.mu.Unlock()
	if first {
		close(ready)
	}
	for {
		var f map[string]any
		if !n.read(ctx, c, &f) {
			return
		}
		switch f["type"] {
		case "message":
			n.mu.Lock()
			n.messages = append(n.messages, f)
			conv, _ := f["conversation"].(string)
			if conv == "" {
				n.convs++
				conv = "conv-" + string(rune('0'+n.convs))
			}
			n.mu.Unlock()
			n.write(ctx, c, map[string]any{"type": "accepted", "re": f["id"], "conversation": conv})
		case "ack":
			n.mu.Lock()
			n.acked = int64(f["up_to"].(float64))
			n.mu.Unlock()
		case "ping":
			n.write(ctx, c, map[string]any{"type": "pong"})
		}
	}
}

func (n *fakeNode) read(ctx context.Context, c *websocket.Conn, v any) bool {
	_, data, err := c.Read(ctx)
	return err == nil && json.Unmarshal(data, v) == nil
}

func (n *fakeNode) write(ctx context.Context, c *websocket.Conn, v any) {
	b, _ := json.Marshal(v)
	_ = c.Write(ctx, websocket.MessageText, b)
}

// reply sends an agent's answer into a conversation.
func (n *fakeNode) reply(conversation, text string) {
	n.mu.Lock()
	n.seq++
	seq, c := n.seq, n.conn
	n.mu.Unlock()
	n.write(context.Background(), c, map[string]any{
		"type": "message", "seq": seq, "conversation": conversation,
		"from": map[string]string{"project": "App", "role": "Project lead"}, "parts": []map[string]string{{"text": text}},
		"at": "2026-11-25T12:00:00Z",
	})
}

func (n *fakeNode) sent() []map[string]any {
	n.mu.Lock()
	defer n.mu.Unlock()
	return append([]map[string]any(nil), n.messages...)
}

type broadcast struct {
	channel, name string
	payload       map[string]any
}

type recorder struct {
	mu  sync.Mutex
	got []broadcast
}

func (r *recorder) Broadcast(channel, name string, payload any) {
	b, _ := json.Marshal(payload)
	var m map[string]any
	_ = json.Unmarshal(b, &m)
	r.mu.Lock()
	r.got = append(r.got, broadcast{channel, name, m})
	r.mu.Unlock()
}

func (r *recorder) all() []broadcast {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]broadcast(nil), r.got...)
}

// setup stores a real SDK error event and runs the module against the node.
func setup(t *testing.T, cfg Config) (*Module, *http.ServeMux, *recorder) {
	t.Helper()
	db, err := storage.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	store := storage.NewSQLiteStore(db)
	if err := module.NewRegistry().Init(db, http.NewServeMux(), store); err != nil {
		t.Fatal(err)
	}
	if err := store.Store(context.Background(), event.Event{UUID: "ev-1", Type: "sentry", Project: "default", Payload: sdkEvent(t, "node/02-event.json"), Timestamp: 1}); err != nil {
		t.Fatal(err)
	}
	if err := store.Store(context.Background(), event.Event{UUID: "dump-1", Type: "var-dump", Project: "default", Payload: json.RawMessage(`{}`), Timestamp: 1}); err != nil {
		t.Fatal(err)
	}

	m := New(cfg)
	rec := &recorder{}
	m.SetBroadcaster(rec)
	mux := http.NewServeMux()
	m.RegisterRoutes(mux, store)
	m.RegisterProtectedRoutes(mux, func(h http.Handler) http.Handler { return h })
	if err := m.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = m.Stop() })
	return m, mux, rec
}

func call(t *testing.T, mux *http.ServeMux, method, path, body string) (int, map[string]any) {
	t.Helper()
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest(method, path, strings.NewReader(body)))
	var out map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &out)
	return w.Code, out
}

func waitFor(t *testing.T, what string, ok func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for !ok() {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %s", what)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestExplain_Conversation(t *testing.T) {
	node := newFakeNode(t)
	m, mux, rec := setup(t, Config{URL: node.url(), Token: testToken})
	waitFor(t, "sign-in", func() bool { _, s, _ := m.current(); return s == "connected" })

	code, st := call(t, mux, "GET", "/api/explain", "")
	if code != 200 || st["status"] != "connected" || st["project"] != "App" || st["to"] != "lead" {
		t.Fatalf("status: %d %v", code, st)
	}

	// Explain: the error's text goes to the lead, a new conversation.
	code, th := call(t, mux, "POST", "/api/explain/event/ev-1", `{"text":"Why?"}`)
	if code != 200 || th["conversation"] != "conv-1" {
		t.Fatalf("explain: %d %v", code, th)
	}
	sent := node.sent()
	if len(sent) != 1 || sent[0]["to"] != "lead" {
		t.Fatalf("node got %v", sent)
	}
	text := sent[0]["parts"].([]any)[0].(map[string]any)["text"].(string)
	if !strings.Contains(text, "Order 15 not found") || !strings.Contains(text, "Question: Why?") {
		t.Errorf("sent text:\n%s", text)
	}

	// The agent answers: it lands in the thread and on the project channel.
	node.reply("conv-1", "loadOrder throws for a missing order; check the id first.")
	waitFor(t, "the answer", func() bool {
		_, th := call(t, mux, "GET", "/api/explain/event/ev-1", "")
		return len(th["messages"].([]any)) == 2
	})
	b := rec.all()
	if len(b) != 1 || b[0].channel != "events.project.default" || b[0].name != EventName || b[0].payload["event_uuid"] != "ev-1" {
		t.Fatalf("broadcasts: %+v", b)
	}
	msg := b[0].payload["message"].(map[string]any)
	if msg["from"] != "agent" || msg["role"] != "Project lead" || !strings.HasPrefix(msg["text"].(string), "loadOrder throws") {
		t.Errorf("broadcast message: %v", msg)
	}
	waitFor(t, "the ack", func() bool { node.mu.Lock(); defer node.mu.Unlock(); return node.acked == 1 })

	// A follow-up goes into the same conversation.
	code, th = call(t, mux, "POST", "/api/explain/event/ev-1/messages", `{"text":"And how do I test it?"}`)
	if code != 200 || len(th["messages"].([]any)) != 3 {
		t.Fatalf("follow-up: %d %v", code, th)
	}
	sent = node.sent()
	if len(sent) != 2 || sent[1]["conversation"] != "conv-1" || sent[1]["to"] != nil {
		t.Fatalf("follow-up frame: %v", sent[len(sent)-1])
	}

	// Deleting the event drops its thread.
	m.OnEventsDeleted([]string{"ev-1"})
	if code, _ := call(t, mux, "GET", "/api/explain/event/ev-1", ""); code != 404 {
		t.Errorf("thread after delete: %d", code)
	}
}

func TestExplain_Refusals(t *testing.T) {
	node := newFakeNode(t)
	m, mux, _ := setup(t, Config{URL: node.url(), Token: testToken})
	waitFor(t, "sign-in", func() bool { _, s, _ := m.current(); return s == "connected" })

	tests := []struct {
		name, method, path, body string
		code                     int
		errCode                  string
	}{
		{"unknown event", "POST", "/api/explain/event/nope", "", 404, "event_not_found"},
		{"not sentry", "POST", "/api/explain/event/dump-1", "", 400, "explain_type"},
		{"follow-up before explain", "POST", "/api/explain/event/ev-1/messages", `{"text":"hi"}`, 404, "explain_none"},
		{"follow-up empty", "POST", "/api/explain/event/ev-1/messages", `{"text":"  "}`, 400, "text_required"},
		{"bad json", "POST", "/api/explain/event/ev-1", `nope`, 400, "bad_body"},
		{"no thread", "GET", "/api/explain/event/ev-1", "", 404, "explain_none"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, out := call(t, mux, tt.method, tt.path, tt.body)
			if code != tt.code || out["code"] != tt.errCode {
				t.Errorf("got %d %v, want %d %s", code, out, tt.code, tt.errCode)
			}
		})
	}
	if n := len(node.sent()); n != 0 {
		t.Errorf("node got %d messages, want none", n)
	}
}

func TestExplain_WrongTokenStops(t *testing.T) {
	node := newFakeNode(t)
	m, mux, _ := setup(t, Config{URL: node.url(), Token: "h2a_wrong"})
	waitFor(t, "the refusal", func() bool { _, s, _ := m.current(); return s == "stopped" })

	code, out := call(t, mux, "POST", "/api/explain/event/ev-1", "")
	if code != 503 || out["code"] != "explain_offline" || !strings.Contains(out["error"].(string), "h2a_auth") {
		t.Errorf("explain while stopped: %d %v", code, out)
	}
	_, st := call(t, mux, "GET", "/api/explain", "")
	if strings.Contains(st["reason"].(string), "h2a_wrong") {
		t.Errorf("status leaks the token: %v", st)
	}
}

func TestExplain_NodeAwayDoesNotBlock(t *testing.T) {
	// Nothing listens there: Start returns at once, the module keeps trying.
	m, mux, _ := setup(t, Config{URL: "ws://127.0.0.1:1/h2a/v1", Token: testToken})
	code, out := call(t, mux, "POST", "/api/explain/event/ev-1", "")
	if code != 503 || out["code"] != "explain_offline" {
		t.Errorf("explain with harnsy away: %d %v", code, out)
	}
	if _, s, _ := m.current(); s != "connecting" {
		t.Errorf("status %q, want connecting", s)
	}
}

// The routes must sit beside Sentry's (POST /api/{project}/cron/{slug}/{key}
// once made http.ServeMux panic at startup).
func TestExplain_RoutesBesideSentry(t *testing.T) {
	db, err := storage.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	store := storage.NewSQLiteStore(db)
	reg := module.NewRegistry()
	reg.Register(sentry.New())
	reg.Register(New(Config{URL: "wss://node.example/h2a/v1", Token: testToken}))
	mux := http.NewServeMux()
	if err := reg.Init(db, mux, store); err != nil {
		t.Fatal(err)
	}
	reg.RegisterProtectedRoutes(mux, func(h http.Handler) http.Handler { return h })
}

// A node that drops each connection right after the welcome gets a backoff,
// not a tight loop, and the status says connecting while it retries.
func TestExplain_DroppedConnectionsBackOff(t *testing.T) {
	old := retryBase
	retryBase = 100 * time.Millisecond
	t.Cleanup(func() { retryBase = old })

	node := newFakeNode(t)
	node.drop = 1 << 30
	m, _, _ := setup(t, Config{URL: node.url(), Token: testToken})
	time.Sleep(1500 * time.Millisecond)
	node.mu.Lock()
	hellos := node.hellos
	node.drop = 0
	node.mu.Unlock()
	// Retries after 100, 200, 400, 800 ms: a handful, never hundreds.
	if hellos < 3 || hellos > 8 {
		t.Errorf("%d sign-ins in 1.5 s", hellos)
	}
	if _, s, _ := m.current(); s == "stopped" {
		t.Errorf("status %q while reconnecting", s)
	}
	waitFor(t, "a lasting connection", func() bool { _, s, _ := m.current(); return s == "connected" })
}
