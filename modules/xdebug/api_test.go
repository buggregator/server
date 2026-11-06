package xdebug

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

// recorder collects the events the proxy emits for the WebSocket.
type recorder struct {
	mu     sync.Mutex
	events []recEvent
}

type recEvent struct {
	name    string
	payload any
}

func (r *recorder) notify(name string, payload any) {
	r.mu.Lock()
	r.events = append(r.events, recEvent{name, payload})
	r.mu.Unlock()
}

func (r *recorder) names() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []string
	for _, e := range r.events {
		out = append(out, e.name)
	}
	return out
}

func (r *recorder) find(name string, match func(any) bool) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, e := range r.events {
		if e.name == name && (match == nil || match(e.payload)) {
			return true
		}
	}
	return false
}

func waitFor(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %s", what)
		}
		time.Sleep(5 * time.Millisecond)
	}
}

// countingEngine answers like serveEngine and counts the commands it got.
type countingEngine struct {
	mu    sync.Mutex
	seen  []string
	extra map[string]string // command name → inner XML
}

func (c *countingEngine) count(prefix string) int {
	c.mu.Lock()
	defer c.mu.Unlock()
	n := 0
	for _, s := range c.seen {
		if strings.HasPrefix(s, prefix) {
			n++
		}
	}
	return n
}

func (c *countingEngine) serve(t *testing.T, e *fakeEngine) {
	// Runs past the end of the test (cleanup detaches): never fail from here.
	send := func(xml string) { _, _ = e.conn.Write(encodePacket([]byte(xml))) }
	for {
		line, err := readCommand(e.r)
		if err != nil {
			return
		}
		c.mu.Lock()
		c.seen = append(c.seen, line)
		c.mu.Unlock()
		name := cmdNameRe.FindString(line)
		tid := tidRe.FindStringSubmatch(line)[1]
		if inner, ok := c.extra[name]; ok {
			send(response(name, tid, "", inner))
			continue
		}
		switch name {
		case "stack_get":
			send(response(name, tid, "", `<stack where="App\Http\Kernel->handle" level="0" type="file" filename="file:///app/src/Kernel.php" lineno="40"></stack>`+
				`<stack where="Illuminate\Pipeline->then" level="1" type="file" filename="file:///app/vendor/laravel/Pipeline.php" lineno="9"></stack>`))
		case "context_names":
			send(response(name, tid, "", `<context name="Locals" id="0"></context><context name="Superglobals" id="1"></context><context name="User defined constants" id="2"></context>`))
		case "context_get":
			send(response(name, tid, "", `<property name="$id" fullname="$id" type="int"><![CDATA[42]]></property>`))
		case "property_get":
			send(response(name, tid, "", `<property name="$id" fullname="$id" type="int"><![CDATA[42]]></property>`))
		case "source":
			send(response(name, tid, ` encoding="base64"`, b64("<?php\r\n$a = 1;\n")))
		case "breakpoint_list":
			send(response(name, tid, "", `<breakpoint type="line" filename="file:///app/src/Kernel.php" lineno="40" state="enabled" hit_count="2" hit_value="0" id="170001"></breakpoint>`))
		case "eval":
			send(response(name, tid, "", `<property type="int"><![CDATA[7]]></property>`))
		default:
			send(response(name, tid, "", `<error code="4"><message><![CDATA[unimplemented]]></message></error>`))
		}
	}
}

type apiHarness struct {
	*harness
	mux *http.ServeMux
	mod *Module
	rec *recorder
}

func newAPIHarness(t *testing.T, allowEval bool) *apiHarness {
	t.Helper()
	h := newHarness(t)
	rec := &recorder{}
	h.proxy.SetNotify(rec.notify)
	mod := &Module{cfg: Config{Addr: "127.0.0.1:9003", IDEAddr: h.ideLn.Addr().String(), AllowEval: allowEval}, proxy: h.proxy}
	mux := http.NewServeMux()
	mod.registerAPI(mux, func(next http.Handler) http.Handler { return next })
	return &apiHarness{harness: h, mux: mux, mod: mod, rec: rec}
}

func (a *apiHarness) do(t *testing.T, method, path, body string) (int, string) {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	rr := httptest.NewRecorder()
	a.mux.ServeHTTP(rr, req)
	return rr.Code, rr.Body.String()
}

func TestAPI_ReadsWhilePaused(t *testing.T) {
	a := newAPIHarness(t, false)
	e, d, s := a.connect(t)
	base := "/api/xdebug/sessions/" + s.ID

	if code, body := a.do(t, "GET", "/api/xdebug", ""); code != 200 || !strings.Contains(body, `"enabled":true`) || !strings.Contains(body, `"allow_eval":false`) {
		t.Fatalf("status: %d %s", code, body)
	}
	// Not paused yet.
	if code, body := a.do(t, "GET", base+"/stack", ""); code != 409 || !strings.Contains(body, `"code":"not_paused"`) || !strings.Contains(body, `"state":"starting"`) {
		t.Fatalf("stack before pause: %d %s", code, body)
	}

	pause(t, e, d, "1")
	eng := &countingEngine{}
	go eng.serve(t, e)

	code, body := a.do(t, "GET", "/api/xdebug/sessions", "")
	var infos []Info
	if code != 200 || json.Unmarshal([]byte(body), &infos) != nil || len(infos) != 1 || infos[0].State != "break" || !infos[0].IDEConnected || infos[0].PausedAt == nil {
		t.Fatalf("sessions: %d %s", code, body)
	}

	code, body = a.do(t, "GET", base+"/stack", "")
	var frames []StackFrame
	if code != 200 || json.Unmarshal([]byte(body), &frames) != nil || len(frames) != 2 || frames[0].Vendor || !frames[1].Vendor {
		t.Fatalf("stack: %d %s", code, body)
	}
	if code, body := a.do(t, "GET", base+"/contexts", ""); code != 200 || !strings.Contains(body, `{"id":1,"name":"Superglobals"}`) {
		t.Fatalf("contexts: %d %s", code, body)
	}
	if code, body := a.do(t, "GET", base+"/variables?frame=1&context=0", ""); code != 200 || !strings.Contains(body, `"value":"42"`) {
		t.Fatalf("variables: %d %s", code, body)
	}
	if eng.count("context_get -i ") != 1 {
		t.Fatal("context_get not sent")
	}
	if code, body := a.do(t, "GET", base+"/property?name=$id&page=2", ""); code != 200 || !strings.Contains(body, `"fullname":"$id"`) {
		t.Fatalf("property: %d %s", code, body)
	}
	if code, _ := a.do(t, "GET", base+"/property", ""); code != 400 {
		t.Fatalf("property without name: %d", code)
	}
	if code, _ := a.do(t, "GET", base+"/variables?frame=-1", ""); code != 400 {
		t.Fatalf("bad frame: %d", code)
	}
	if code, body := a.do(t, "GET", base+"/source?file=file:///app/a.php&begin=10&end=11", ""); code != 200 ||
		!strings.Contains(body, `{"line":10,"text":"<?php"}`) || !strings.Contains(body, `{"line":11,"text":"$a = 1;"}`) {
		t.Fatalf("source: %d %s", code, body)
	}
	if code, body := a.do(t, "GET", base+"/breakpoints", ""); code != 200 || !strings.Contains(body, `"line":40`) || !strings.Contains(body, `"hit_count":2`) {
		t.Fatalf("breakpoints: %d %s", code, body)
	}
	if code, body := a.do(t, "POST", base+"/eval", `{"expression":"1"}`); code != 403 || !strings.Contains(body, `"code":"eval_disabled"`) {
		t.Fatalf("eval off: %d %s", code, body)
	}
	if code, _ := a.do(t, "GET", "/api/xdebug/sessions/99/stack", ""); code != 404 {
		t.Fatalf("unknown session: %d", code)
	}

	// UI reads are not agent calls.
	if code, body := a.do(t, "GET", base+"/agent-log", ""); code != 200 || strings.TrimSpace(body) != "[]" {
		t.Fatalf("agent log after UI reads: %s", body)
	}
	if s.Info().AgentCalls != 0 || a.rec.find("agent.call", nil) {
		t.Fatal("UI reads logged as agent calls")
	}
	d.expectQuiet(100 * time.Millisecond)

	// The IDE steps: reads refused, stale state broadcast.
	d.send("step_over -i 2")
	waitFor(t, "running", func() bool { return s.Info().State == "running" })
	if code, body := a.do(t, "GET", base+"/variables", ""); code != 409 || !strings.Contains(body, `"state":"running"`) {
		t.Fatalf("variables while running: %d %s", code, body)
	}
}

func TestAPI_EvalWhenAllowed(t *testing.T) {
	a := newAPIHarness(t, true)
	e, d, s := a.connect(t)
	pause(t, e, d, "1")
	eng := &countingEngine{}
	go eng.serve(t, e)

	base := "/api/xdebug/sessions/" + s.ID
	if code, body := a.do(t, "POST", base+"/eval", `{"expression":"3+4"}`); code != 200 || !strings.Contains(body, `"value":"7"`) {
		t.Fatalf("eval: %d %s", code, body)
	}
	// Not cached: an expression may change state.
	a.do(t, "POST", base+"/eval", `{"expression":"3+4"}`)
	if n := eng.count("eval "); n != 2 {
		t.Fatalf("eval sent %d times, want 2", n)
	}
	if code, _ := a.do(t, "POST", base+"/eval", `{}`); code != 400 {
		t.Fatalf("empty eval: %d", code)
	}
}

func TestAPI_ReadsAreCachedPerPause(t *testing.T) {
	a := newAPIHarness(t, false)
	e, d, s := a.connect(t)
	pause(t, e, d, "1")
	base := "/api/xdebug/sessions/" + s.ID

	// Serve two reads by hand to keep control over the run reply.
	go func() {
		line := e.read()
		e.send(response("context_get", tidOf(t, line), "", `<property name="$a" fullname="$a" type="int"><![CDATA[1]]></property>`))
	}()
	for i := 0; i < 3; i++ {
		if code, body := a.do(t, "GET", base+"/variables", ""); code != 200 || !strings.Contains(body, `"value":"1"`) {
			t.Fatalf("read %d: %d %s", i, code, body)
		}
	}
	// Only one context_get reached the engine: the next command it sees is the IDE's run.
	d.send("run -i 2")
	runCmd := e.read()
	if !strings.HasPrefix(runCmd, "run -i ") {
		t.Fatalf("engine got %q, want the IDE's run (reads must be cached)", runCmd)
	}
	e.send(response("run", tidOf(t, runCmd), ` status="break" reason="ok"`, `<xdebug:message filename="file:///app/x.php" lineno="3"></xdebug:message>`))
	_ = d.read()

	// A new pause: the cache is fresh.
	go func() {
		line := e.read()
		e.send(response("context_get", tidOf(t, line), "", `<property name="$a" fullname="$a" type="int"><![CDATA[2]]></property>`))
	}()
	if code, body := a.do(t, "GET", base+"/variables", ""); code != 200 || !strings.Contains(body, `"value":"2"`) {
		t.Fatalf("after next pause: %d %s", code, body)
	}
}

func TestAgentLogAndEvents(t *testing.T) {
	a := newAPIHarness(t, false)
	e, d, s := a.connect(t)
	// register() lists the session before it emits the event.
	waitFor(t, "session.created", func() bool {
		return a.rec.find("session.created", func(p any) bool { return p.(Info).ID == s.ID && p.(Info).IDEConnected })
	})
	pause(t, e, d, "1")
	if !a.rec.find("session.status", func(p any) bool {
		m := p.(map[string]any)
		return m["state"] == "break" && m["line"] == 12
	}) {
		t.Fatalf("no session.status break: %v", a.rec.names())
	}
	eng := &countingEngine{}
	go eng.serve(t, e)

	cs := connectTools(t, a.proxy, false)
	if _, isErr := callTool(t, cs, "xdebug_stack", map[string]any{}); isErr {
		t.Fatal("stack failed")
	}
	if _, isErr := callTool(t, cs, "xdebug_property", map[string]any{"name": "$nope\x00x"}); !isErr {
		t.Fatal("NUL name accepted")
	}

	code, body := a.do(t, "GET", "/api/xdebug/sessions/"+s.ID+"/agent-log", "")
	var log []AgentCall
	if code != 200 || json.Unmarshal([]byte(body), &log) != nil || len(log) != 2 {
		t.Fatalf("agent log: %d %s", code, body)
	}
	if log[0].Tool != "xdebug_stack" || !log[0].OK || log[0].Result != "2 frames" || log[0].Seq != 1 {
		t.Fatalf("first entry: %+v", log[0])
	}
	if log[1].Tool != "xdebug_property" || log[1].OK || !strings.HasPrefix(log[1].Result, "refused — ") {
		t.Fatalf("second entry: %+v", log[1])
	}
	if s.Info().AgentCalls != 2 {
		t.Fatalf("agent_calls = %d", s.Info().AgentCalls)
	}
	if !a.rec.find("agent.call.started", nil) || !a.rec.find("agent.call", func(p any) bool { return p.(AgentCall).Tool == "xdebug_stack" }) {
		t.Fatalf("agent events: %v", a.rec.names())
	}

	// The IDE goes away: session.closed with the reason, a tombstone stays.
	d.conn.Close()
	waitFor(t, "session.closed", func() bool {
		return a.rec.find("session.closed", func(p any) bool { return p.(map[string]any)["reason"] == ReasonIDEClosed })
	})
	if code, body := a.do(t, "GET", "/api/xdebug/sessions/"+s.ID, ""); code != 200 || !strings.Contains(body, `"state":"detached"`) || !strings.Contains(body, `"closed_reason":"ide_closed"`) {
		t.Fatalf("tombstone: %d %s", code, body)
	}
}

func TestAgentLogIsARing(t *testing.T) {
	s := &Session{ID: "1"}
	for i := 0; i < agentLogSize+5; i++ {
		s.agentDone("xdebug_stack", "", time.Now(), true, "1 frame")
	}
	log := s.AgentLog()
	if len(log) != agentLogSize || log[0].Seq != 6 || log[len(log)-1].Seq != agentLogSize+5 {
		t.Fatalf("ring: len %d, first %d", len(log), log[0].Seq)
	}
}

func TestAPI_DismissOnlyTombstones(t *testing.T) {
	a := newAPIHarness(t, false)
	e, _, s := a.connect(t)
	path := "/api/xdebug/sessions/" + s.ID

	if code, body := a.do(t, "DELETE", path, ""); code != 409 || !strings.Contains(body, "session_live") {
		t.Fatalf("dismiss live: %d %s", code, body)
	}
	e.conn.Close()
	waitFor(t, "close", s.Closed)
	if code, _ := a.do(t, "DELETE", path, ""); code != 204 {
		t.Fatalf("dismiss tombstone: %d", code)
	}
	if code, _ := a.do(t, "GET", path, ""); code != 404 {
		t.Fatalf("after dismiss: %d", code)
	}
}

func TestTombstoneExpires(t *testing.T) {
	h := newHarness(t)
	h.proxy.tombstoneTTL = 50 * time.Millisecond
	e, _, s := h.connect(t)
	e.conn.Close()
	waitFor(t, "close", s.Closed)
	waitFor(t, "tombstone expiry", func() bool { return len(h.proxy.Sessions()) == 0 })
}

func TestTargetIsReadAtFirstBreak(t *testing.T) {
	tests := []struct {
		name       string
		answer     func(e *fakeEngine, line, tid string)
		kind, want string
	}{
		{
			name: "http",
			answer: func(e *fakeEngine, line, tid string) {
				switch {
				case strings.Contains(line, "REQUEST_METHOD"):
					e.send(response("property_get", tid, "", `<property name="REQUEST_METHOD" type="string" size="3" encoding="base64"><![CDATA[`+b64("GET")+`]]></property>`))
				case strings.Contains(line, "REQUEST_URI"):
					e.send(response("property_get", tid, "", `<property name="REQUEST_URI" type="string" size="17" encoding="base64"><![CDATA[`+b64("/checkout?step=2")+`]]></property>`))
				}
			},
			kind: "http", want: "GET /checkout?step=2",
		},
		{
			name: "cli",
			answer: func(e *fakeEngine, line, tid string) {
				switch {
				case strings.Contains(line, "REQUEST_METHOD"):
					e.send(response("property_get", tid, "", `<error code="300"><message><![CDATA[can not get property]]></message></error>`))
				case strings.Contains(line, "argv"):
					e.send(response("property_get", tid, "", `<property name="argv" type="array" children="1" numchildren="2">`+
						`<property name="0" type="string" encoding="base64"><![CDATA[`+b64("artisan")+`]]></property>`+
						`<property name="1" type="string" encoding="base64"><![CDATA[`+b64("queue:work")+`]]></property></property>`))
				}
			},
			kind: "cli", want: "artisan queue:work",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newHarness(t)
			h.proxy.readTarget = true
			e, d, s := h.connect(t)

			d.send("run -i 1")
			runCmd := e.read()
			e.send(response("run", tidOf(t, runCmd), ` status="break" reason="ok"`,
				`<xdebug:message filename="file:///app/t.php" lineno="6"></xdebug:message>`))
			_ = d.read()
			for !s.Info().targetSet() {
				line := e.read()
				if !strings.HasPrefix(line, "property_get -i ") || !strings.Contains(line, "-c 1") {
					t.Fatalf("unexpected engine command %q", line)
				}
				tt.answer(e, line, tidOf(t, line))
				time.Sleep(10 * time.Millisecond)
			}
			if in := s.Info(); in.Kind != tt.kind || in.Target != tt.want || in.LastFile != "file:///app/t.php" || in.LastLine != 6 {
				t.Fatalf("info after target: %+v", in)
			}
			d.expectQuiet(100 * time.Millisecond)
		})
	}
}

func (in Info) targetSet() bool { return in.Kind != "" }

func TestAPI_RoutesAreBehindAuth(t *testing.T) {
	mod := New(Config{Addr: "127.0.0.1:0", IDEAddr: "127.0.0.1:1"})
	mux := http.NewServeMux()
	deny := func(http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusUnauthorized) })
	}
	mod.RegisterProtectedRoutes(mux, deny)
	for _, rt := range []struct{ method, path string }{
		{"GET", "/api/xdebug"}, {"GET", "/api/xdebug/sessions"}, {"GET", "/api/xdebug/sessions/1"},
		{"DELETE", "/api/xdebug/sessions/1"}, {"GET", "/api/xdebug/sessions/1/stack"},
		{"GET", "/api/xdebug/sessions/1/contexts"}, {"GET", "/api/xdebug/sessions/1/variables"},
		{"GET", "/api/xdebug/sessions/1/property"}, {"GET", "/api/xdebug/sessions/1/source"},
		{"GET", "/api/xdebug/sessions/1/breakpoints"}, {"POST", "/api/xdebug/sessions/1/eval"},
		{"GET", "/api/xdebug/sessions/1/agent-log"},
	} {
		rr := httptest.NewRecorder()
		mux.ServeHTTP(rr, httptest.NewRequest(rt.method, rt.path, nil))
		if rr.Code != http.StatusUnauthorized {
			t.Errorf("%s %s: %d, want 401", rt.method, rt.path, rr.Code)
		}
	}
}
