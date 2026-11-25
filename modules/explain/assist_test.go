package explain

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/buggregator/server/v2/internal/module"
	"github.com/buggregator/server/v2/internal/storage"
)

func sentText(t *testing.T, f map[string]any) string {
	t.Helper()
	return f["parts"].([]any)[0].(map[string]any)["text"].(string)
}

func connected(t *testing.T, m *Module) {
	t.Helper()
	waitFor(t, "sign-in", func() bool { _, s, _ := m.current(); return s == "connected" })
}

func TestAssist_EventThread(t *testing.T) {
	node := newFakeNode(t)
	m, mux, rec := setup(t, Config{URL: node.url(), Token: testToken, MCP: MCP{On: true, ReadOnly: true, Transport: "http"}})
	connected(t, m)

	code, st := call(t, mux, "GET", "/api/assist", "")
	limits, _ := st["limits"].(map[string]any)
	mcp, _ := st["mcp"].(map[string]any)
	if code != 200 || st["status"] != "connected" || limits["text_bytes"] != float64(16384) || limits["messages_per_day"] != float64(100) ||
		mcp["on"] != true || mcp["read_only"] != true || mcp["transport"] != "http" {
		t.Fatalf("status: %d %v", code, st)
	}

	// A monolog event: the server builds its text; the client's text is ignored.
	code, th := call(t, mux, "POST", "/api/assist/threads", `{"context":{"kind":"event","key":"log-1","title":"Log","text":"IGNORED"},"text":"Why?"}`)
	if code != 200 || th["conversation"] != "conv-1" || th["project"] != "shop" || th["kind"] != "event" || th["key"] != "log-1" {
		t.Fatalf("start: %d %v", code, th)
	}
	text := sentText(t, node.sent()[0])
	for _, want := range []string{"Question: Why?", "Payment failed", "app.ERROR", "Event: log-1 (monolog, project shop)",
		"via the `buggregator` MCP connection (read-only): event log-1, project shop"} {
		if !strings.Contains(text, want) {
			t.Errorf("sent text lacks %q:\n%s", want, text)
		}
	}
	for _, bad := range []string{"IGNORED", "hunter2", testToken} {
		if strings.Contains(text, bad) {
			t.Errorf("sent text has %q:\n%s", bad, text)
		}
	}
	msgs := th["messages"].([]any)
	first, second := msgs[0].(map[string]any), msgs[1].(map[string]any)
	if len(msgs) != 2 || first["context"] != true || first["text"] != text || second["text"] != "Why?" || second["context"] != nil {
		t.Fatalf("messages: %v", msgs)
	}

	// The same context again: 409 with the thread; "new" starts over.
	code, out := call(t, mux, "POST", "/api/assist/threads", `{"context":{"kind":"event","key":"log-1"}}`)
	if code != 409 || out["code"] != "assist_exists" || out["thread"].(map[string]any)["conversation"] != "conv-1" {
		t.Fatalf("second start: %d %v", code, out)
	}
	code, th = call(t, mux, "POST", "/api/assist/threads", `{"context":{"kind":"event","key":"log-1"},"new":true}`)
	if code != 200 || th["conversation"] != "conv-2" || len(th["messages"].([]any)) != 1 {
		t.Fatalf("new start: %d %v", code, th)
	}
	if !strings.Contains(sentText(t, node.sent()[1]), "say what stands out") {
		t.Errorf("no default ask without a question:\n%s", sentText(t, node.sent()[1]))
	}
	id := int64(th["id"].(float64))

	// An answer to the old conversation is dropped; the new one lands.
	node.reply("conv-1", "old")
	node.reply("conv-2", "The payment gateway refused order 15.")
	waitFor(t, "the answer", func() bool {
		_, th := call(t, mux, "GET", fmt.Sprintf("/api/assist/threads/%d", id), "")
		return len(th["messages"].([]any)) == 2
	})
	_, th = call(t, mux, "GET", "/api/assist/threads?kind=event&key=log-1", "")
	if last := th["messages"].([]any)[1].(map[string]any); last["text"] != "The payment gateway refused order 15." || last["from"] != "agent" {
		t.Errorf("thread: %v", th)
	}
	b := rec.messages()
	if len(b) != 2 || b[0].channel != "events.project.shop" || b[0].name != EventName || b[0].payload["thread_id"] != float64(id) || b[0].payload["kind"] != "event" {
		t.Fatalf("broadcasts: %+v", b)
	}

	// A follow-up by id.
	code, th = call(t, mux, "POST", fmt.Sprintf("/api/assist/threads/%d/messages", id), `{"text":"token=abcdef123456 and then?"}`)
	if code != 200 || len(th["messages"].([]any)) != 3 {
		t.Fatalf("follow-up: %d %v", code, th)
	}
	sent := node.sent()
	if f := sent[len(sent)-1]; f["conversation"] != "conv-2" || strings.Contains(sentText(t, f), "abcdef123456") {
		t.Errorf("follow-up frame: %v", f)
	}

	// Deleting the event drops its thread.
	m.OnEventsDeleted([]string{"log-1"})
	if code, out := call(t, mux, "GET", "/api/assist/threads?kind=event&key=log-1", ""); code != 200 || out["thread"] != nil {
		t.Errorf("thread after delete: %d %v", code, out)
	}
}

func TestAssist_PageThread(t *testing.T) {
	node := newFakeNode(t)
	// MCP on but writable: no pointer, agents are not invited to it.
	m, mux, _ := setup(t, Config{URL: node.url(), Token: testToken, MCP: MCP{On: true}})
	connected(t, m)

	code, th := call(t, mux, "POST", "/api/assist/threads",
		`{"context":{"kind":"page","key":"/sentry?project=shop","title":"Sentry · shop","text":"12 errors in 1h; top: OrderNotFound x9. DSN https://abc123@o1.ingest.sentry.io/1"}}`)
	if code != 200 || th["project"] != "default" || th["title"] != "Sentry · shop" {
		t.Fatalf("start: %d %v", code, th)
	}
	text := sentText(t, node.sent()[0])
	for _, want := range []string{"looking at this page", "Page: Sentry · shop", "OrderNotFound x9", "Page route: /sentry?project=shop"} {
		if !strings.Contains(text, want) {
			t.Errorf("sent text lacks %q:\n%s", want, text)
		}
	}
	if strings.Contains(text, "abc123") || strings.Contains(text, "MCP") {
		t.Errorf("sent text:\n%s", text)
	}

	// A page context names its project: the thread goes on its channel.
	code, th = call(t, mux, "POST", "/api/assist/threads", `{"context":{"kind":"page","key":"/","project":"shop","text":"home"}}`)
	if code != 200 || th["project"] != "shop" {
		t.Fatalf("page with project: %d %v", code, th)
	}

	code, out := call(t, mux, "GET", "/api/assist/threads?recent=10", "")
	list, _ := out["threads"].([]any)
	if code != 200 || len(list) != 2 || list[0].(map[string]any)["key"] != "/" || list[0].(map[string]any)["last_from"] != "you" {
		t.Fatalf("recent: %d %v", code, out)
	}
}

func TestAssist_Refusals(t *testing.T) {
	node := newFakeNode(t)
	m, mux, _ := setup(t, Config{URL: node.url(), Token: testToken})
	connected(t, m)
	long := strings.Repeat("x", maxPageBytes+1)
	tests := []struct {
		name, method, path, body string
		code                     int
		errCode                  string
	}{
		{"bad kind", "POST", "/api/assist/threads", `{"context":{"kind":"tab","key":"x"}}`, 400, "bad_context"},
		{"no key", "POST", "/api/assist/threads", `{"context":{"kind":"page"}}`, 400, "bad_context"},
		{"page text too long", "POST", "/api/assist/threads", `{"context":{"kind":"page","key":"/","text":"` + long + `"}}`, 400, "context_too_long"},
		{"unknown event", "POST", "/api/assist/threads", `{"context":{"kind":"event","key":"nope"}}`, 404, "event_not_found"},
		{"question too long", "POST", "/api/assist/threads", `{"context":{"kind":"page","key":"/"},"text":"` + strings.Repeat("q", maxQuestionBytes+1) + `"}`, 400, "text_too_long"},
		{"bad json", "POST", "/api/assist/threads", `nope`, 400, "bad_body"},
		{"no query", "GET", "/api/assist/threads", "", 400, "bad_query"},
		{"bad id", "GET", "/api/assist/threads/abc", "", 404, "assist_none"},
		{"unknown id", "POST", "/api/assist/threads/99/messages", `{"text":"hi"}`, 404, "assist_none"},
		{"empty follow-up", "POST", "/api/assist/threads/99/messages", `{"text":" "}`, 400, "text_required"},
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

	// No conversation for a context yet: 200 {"thread": null}.
	code, out := call(t, mux, "GET", "/api/assist/threads?kind=page&key=/x", "")
	if th, ok := out["thread"]; code != 200 || !ok || th != nil {
		t.Errorf("unknown context: %d %v", code, out)
	}
}

// A conversation outlives a restart: the answer that comes after it lands
// in the stored thread.
func TestAssist_SurvivesRestart(t *testing.T) {
	node := newFakeNode(t)
	db, store := setupDB(t)
	cfg := Config{URL: node.url(), Token: testToken}
	m, mux, _ := startModule(t, db, store, cfg)
	connected(t, m)
	code, th := call(t, mux, "POST", "/api/assist/threads", `{"context":{"kind":"event","key":"ev-1"}}`)
	if code != 200 {
		t.Fatalf("start: %d %v", code, th)
	}
	if err := m.Stop(); err != nil {
		t.Fatal(err)
	}

	node.mu.Lock()
	node.ready = make(chan struct{})
	ready := node.ready
	node.mu.Unlock()
	m2, mux2, _ := startModule(t, db, store, cfg)
	connected(t, m2)
	<-ready
	node.reply("conv-1", "after restart")
	waitFor(t, "the answer after restart", func() bool {
		_, th := call(t, mux2, "GET", "/api/assist/threads?kind=event&key=ev-1", "")
		msgs, _ := th["messages"].([]any)
		return len(msgs) == 2 && msgs[1].(map[string]any)["text"] == "after restart"
	})
}

func TestAssist_PageThreadsCapped(t *testing.T) {
	db, _ := setupDB(t)
	m := New(Config{})
	reg := module.NewRegistry()
	reg.Register(m)
	if err := reg.Init(db, http.NewServeMux(), storage.NewSQLiteStore(db)); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < maxPageThreads+5; i++ {
		if _, err := m.threads.start(Context{Kind: KindPage, Key: fmt.Sprintf("/p%d", i)}, "default", "lead", fmt.Sprintf("c%d", i), []Message{{From: "you", Text: "x"}}); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := m.threads.start(Context{Kind: KindEvent, Key: "ev-1"}, "default", "lead", "ce", []Message{{From: "you", Text: "x"}}); err != nil {
		t.Fatal(err)
	}
	var pages, msgs int
	db.QueryRow(`SELECT COUNT(*) FROM assist_threads WHERE kind = 'page'`).Scan(&pages)
	db.QueryRow(`SELECT COUNT(*) FROM assist_messages`).Scan(&msgs)
	if pages != maxPageThreads || msgs != maxPageThreads+1 {
		t.Errorf("page threads %d, messages %d; want %d, %d", pages, msgs, maxPageThreads, maxPageThreads+1)
	}
	if _, err := m.threads.byKey(KindPage, "/p0"); err != errNoThread {
		t.Errorf("oldest page thread kept: %v", err)
	}

	th, _ := m.threads.byKey(KindPage, "/p54")
	for i := 0; i < maxMessages+10; i++ {
		if _, err := m.threads.add(th.ID, Message{From: "you", Text: fmt.Sprint(i)}); err != nil {
			t.Fatal(err)
		}
	}
	th, _ = m.threads.byID(th.ID)
	if len(th.Messages) != maxMessages || th.Messages[len(th.Messages)-1].Text != fmt.Sprint(maxMessages+9) {
		t.Errorf("%d messages kept, last %q", len(th.Messages), th.Messages[len(th.Messages)-1].Text)
	}
}

func TestAssist_Recipients(t *testing.T) {
	two := []map[string]string{{"address": "lead", "name": "Project lead"}, {"address": "role:7", "name": "Go developer", "about": "server code"}}
	tests := []struct {
		name        string
		to          []map[string]string // the welcome's list; nil = the fake node's default (lead only)
		cfgTo       string
		wantDefault string
		bodyTo      string
		code        int
		wantSentTo  string
		errCode     string
	}{
		{"default is the configured address", two, "lead", "lead", "", 200, "lead", ""},
		{"configured address not in the list: the first", two, "role:9", "lead", "", 200, "lead", ""},
		{"picked from the list", two, "lead", "lead", "role:7", 200, "role:7", ""},
		{"not in the list", two, "lead", "lead", "role:99", 400, "", "bad_recipient"},
		{"node sends no list: only the configured one", []map[string]string{}, "role:3", "role:3", "role:3", 200, "role:3", ""},
		{"node sends no list: others refused", []map[string]string{}, "role:3", "role:3", "lead", 400, "", "bad_recipient"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node := newFakeNode(t)
			if tt.to != nil {
				node.to = tt.to
			}
			m, mux, _ := setup(t, Config{URL: node.url(), Token: testToken, To: tt.cfgTo})
			connected(t, m)

			_, st := call(t, mux, "GET", "/api/assist", "")
			if st["default"] != tt.wantDefault || len(st["recipients"].([]any)) != len(node.to) {
				t.Fatalf("status: %v", st)
			}
			body := `{"context":{"kind":"page","key":"/"}`
			if tt.bodyTo != "" {
				body += `,"to":"` + tt.bodyTo + `"`
			}
			code, th := call(t, mux, "POST", "/api/assist/threads", body+"}")
			if code != tt.code || (tt.errCode != "" && th["code"] != tt.errCode) {
				t.Fatalf("start: %d %v", code, th)
			}
			sent := node.sent()
			if tt.code != 200 {
				if len(sent) != 0 {
					t.Errorf("node got %v", sent)
				}
				return
			}
			if len(sent) != 1 || sent[0]["to"] != tt.wantSentTo || th["to"] != tt.wantSentTo {
				t.Fatalf("sent %v, thread to %v; want %s", sent, th["to"], tt.wantSentTo)
			}
			// A follow-up stays in the conversation: no new address.
			id := int64(th["id"].(float64))
			if code, _ := call(t, mux, "POST", fmt.Sprintf("/api/assist/threads/%d/messages", id), `{"text":"more","to":"lead"}`); code != 200 {
				t.Fatalf("follow-up: %d", code)
			}
			if f := node.sent()[1]; f["to"] != nil || f["conversation"] != "conv-1" {
				t.Errorf("follow-up frame: %v", f)
			}
			_, out := call(t, mux, "GET", "/api/assist/threads?recent=5", "")
			if row := out["threads"].([]any)[0].(map[string]any); row["to"] != tt.wantSentTo {
				t.Errorf("recent row: %v", row)
			}
		})
	}
}

// The person adds a post to the pass: the node sends a new welcome on the
// live connection; the status and the open panels get the new list without
// a restart.
func TestAssist_RecipientsFollowNewWelcome(t *testing.T) {
	node := newFakeNode(t)
	m, mux, rec := setup(t, Config{URL: node.url(), Token: testToken})
	connected(t, m)
	waitFor(t, "the first status broadcast", func() bool { return len(rec.all()) == 1 })
	if b := rec.all()[0]; b.channel != StatusChannel || b.name != StatusEventName || len(b.payload["recipients"].([]any)) != 1 {
		t.Fatalf("first broadcast: %+v", b)
	}

	node.welcomeAgain([]map[string]string{{"address": "lead", "name": "Project lead"}, {"address": "role:213", "name": "Go developer"}})
	waitFor(t, "the new list", func() bool {
		_, st := call(t, mux, "GET", "/api/assist", "")
		r, _ := st["recipients"].([]any)
		return len(r) == 2
	})
	waitFor(t, "the status broadcast", func() bool { return len(rec.all()) == 2 })
	b := rec.all()[1]
	if r := b.payload["recipients"].([]any); b.channel != StatusChannel || len(r) != 2 || r[1].(map[string]any)["address"] != "role:213" || b.payload["default"] != "lead" {
		t.Errorf("broadcast: %+v", b)
	}
	// The new address can be picked at once.
	if code, th := call(t, mux, "POST", "/api/assist/threads", `{"context":{"kind":"page","key":"/"},"to":"role:213"}`); code != 200 {
		t.Errorf("start with the new address: %d %v", code, th)
	}
	// The same list again: no broadcast.
	node.welcomeAgain([]map[string]string{{"address": "lead", "name": "Project lead"}, {"address": "role:213", "name": "Go developer"}})
	time.Sleep(200 * time.Millisecond)
	if n := len(rec.all()); n != 2 {
		t.Errorf("%d broadcasts after the same list", n)
	}
}
