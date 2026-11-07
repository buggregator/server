package xdebug

import (
	"bufio"
	"encoding/json"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"
)

// newHereHarness is an API harness whose IDE address answers nobody.
func newHereHarness(t *testing.T, mode string) *apiHarness {
	t.Helper()
	a := newAPIHarness(t, false)
	a.ideLn.Close() // nobody listens: auto falls back to driving here
	if err := a.proxy.SetSettings(Settings{Mode: mode}); err != nil {
		t.Fatal(err)
	}
	return a
}

// startHere connects an engine to a session driven here, answers the
// feature_set commands and returns the first continuation the driver sends.
func startHere(t *testing.T, p *Proxy) (*fakeEngine, *Session, string) {
	t.Helper()
	before := len(p.Sessions())
	engineSide, proxySide := tcpPair(t)
	done := make(chan struct{})
	go func() { p.HandleConnection(proxySide); close(done) }()
	t.Cleanup(func() {
		engineSide.Close()
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Error("HandleConnection did not return")
		}
	})
	e := &fakeEngine{t: t, conn: engineSide, r: bufio.NewReader(engineSide)}
	e.send(initXML)
	for _, f := range standaloneFeatures {
		got := e.read()
		if !strings.HasPrefix(got, "feature_set -i ") || !strings.HasSuffix(got, " -n "+f) {
			t.Fatalf("engine got %q, want feature_set %s", got, f)
		}
		e.send(response("feature_set", tidOf(t, got), ` feature="x" success="1"`, ""))
	}
	first := e.read()
	h := &harness{proxy: p}
	return e, h.waitSession(t, before+1), first
}

func breakAt(t *testing.T, e *fakeEngine, cmdLine, cmd string, line int) {
	t.Helper()
	e.send(response(cmd, tidOf(t, cmdLine), ` status="break" reason="ok"`,
		`<xdebug:message filename="file:///app/src/User.php" lineno="`+itoa(line)+`"></xdebug:message>`))
}

func itoa(n int) string { b, _ := json.Marshal(n); return string(b) }

func TestStandalone_AutoWithoutIDEDrivesHere(t *testing.T) {
	a := newHereHarness(t, ModeAuto)
	e, s, first := startHere(t, a.proxy)
	if !strings.HasPrefix(first, "run -i ") {
		t.Fatalf("first continuation %q, want run", first)
	}
	if in := s.Info(); in.Driver != DriverHere || in.IDEConnected || in.State != "running" || in.LastContinuationBy != nil {
		t.Fatalf("info: %+v", in)
	}

	breakAt(t, e, first, "run", 12)
	waitFor(t, "break", s.Paused)
	in := s.Info()
	if in.LastLine != 12 || in.IdleDeadline == nil || time.Until(*in.IdleDeadline) < 9*time.Minute {
		t.Fatalf("paused info: %+v", in)
	}

	// Step over from the UI: sent without waiting for the reply.
	code, body := a.do(t, "POST", "/api/xdebug/sessions/"+s.ID+"/step-over", "{}")
	if code != http.StatusOK || !strings.Contains(body, `"state":"running"`) || !strings.Contains(body, `"last_continuation_by":"ui"`) || !strings.Contains(body, `"idle_deadline":null`) {
		t.Fatalf("step-over: %d %s", code, body)
	}
	step := e.read()
	if !strings.HasPrefix(step, "step_over -i ") {
		t.Fatalf("engine got %q", step)
	}
	// A second continuation while running is refused.
	if code, body := a.do(t, "POST", "/api/xdebug/sessions/"+s.ID+"/run", "{}"); code != http.StatusConflict || !strings.Contains(body, `"not_paused"`) {
		t.Fatalf("run while running: %d %s", code, body)
	}
	breakAt(t, e, step, "step_over", 13)
	waitFor(t, "second break", func() bool { return s.Paused() && s.Info().LastLine == 13 })
	if !a.rec.find("session.status", func(p any) bool {
		m := p.(map[string]any)
		return m["line"] == 13 && m["driver"] == DriverHere && *(m["last_continuation_by"].(*string)) == ByUI
	}) {
		t.Fatalf("no session.status for line 13: %v", a.rec.names())
	}

	// Reads work as with an IDE.
	go func() {
		got := e.read()
		e.send(response("stack_get", tidOf(t, got), "", `<stack where="{main}" level="0" type="file" filename="file:///app/src/User.php" lineno="13"></stack>`))
	}()
	if code, body := a.do(t, "GET", "/api/xdebug/sessions/"+s.ID+"/stack", ""); code != http.StatusOK || !strings.Contains(body, `"line":13`) {
		t.Fatalf("stack: %d %s", code, body)
	}

	// Release: the script runs on; the engine closes and the tombstone says detached.
	if code, body := a.do(t, "POST", "/api/xdebug/sessions/"+s.ID+"/detach", "{}"); code != http.StatusOK {
		t.Fatalf("detach: %d %s", code, body)
	}
	if got := e.read(); !strings.HasPrefix(got, "detach -i ") {
		t.Fatalf("engine got %q", got)
	}
	e.conn.Close()
	waitFor(t, "close", s.Closed)
	if in := s.Info(); in.State != "detached" || in.ClosedReason != ReasonDetached || in.IdleDeadline != nil {
		t.Fatalf("tombstone: %+v", in)
	}
}

func TestStandalone_PauseOnFirstLine(t *testing.T) {
	a := newHereHarness(t, ModeStandalone)
	if err := a.proxy.SetSettings(Settings{Mode: ModeStandalone, PauseOnFirstLine: true}); err != nil {
		t.Fatal(err)
	}
	_, _, first := startHere(t, a.proxy)
	if !strings.HasPrefix(first, "step_into -i ") {
		t.Fatalf("first continuation %q, want step_into", first)
	}
}

// Standalone never dials the IDE, even when one listens.
func TestStandalone_ModeNeverDialsIDE(t *testing.T) {
	a := newAPIHarness(t, false)
	if err := a.proxy.SetSettings(Settings{Mode: ModeStandalone}); err != nil {
		t.Fatal(err)
	}
	_, s, _ := startHere(t, a.proxy)
	_ = a.ideLn.(*net.TCPListener).SetDeadline(time.Now().Add(200 * time.Millisecond))
	if c, err := a.ideLn.Accept(); err == nil {
		c.Close()
		t.Fatal("IDE was dialed in standalone mode")
	}
	if s.Info().Driver != DriverHere {
		t.Fatalf("driver: %+v", s.Info())
	}
}

// At the end of the script Xdebug waits in "stopping" for the IDE's word.
func TestStandalone_ScriptEndIsStopped(t *testing.T) {
	a := newHereHarness(t, ModeAuto)
	e, s, first := startHere(t, a.proxy)
	e.send(response("run", tidOf(t, first), ` status="stopping" reason="ok"`, ""))
	stop := e.read()
	if !strings.HasPrefix(stop, "stop -i ") {
		t.Fatalf("engine got %q, want stop", stop)
	}
	e.send(response("stop", tidOf(t, stop), ` status="stopped" reason="ok"`, ""))
	e.conn.Close()
	waitFor(t, "close", s.Closed)
	if in := s.Info(); in.State != "stopped" || in.ClosedReason != ReasonStopped {
		t.Fatalf("tombstone: %+v", in)
	}
}

func TestStandalone_IdleSessionIsReleased(t *testing.T) {
	a := newHereHarness(t, ModeAuto)
	a.proxy.idleTimeout = 300 * time.Millisecond
	e, s, first := startHere(t, a.proxy)
	breakAt(t, e, first, "run", 12)
	waitFor(t, "break", s.Paused)

	// A read resets the deadline.
	time.Sleep(150 * time.Millisecond)
	d1 := *s.Info().IdleDeadline
	go func() {
		got := e.read()
		e.send(response("stack_get", tidOf(t, got), "", ""))
	}()
	if _, err := s.Command("stack_get", "", nil); err != nil {
		t.Fatal(err)
	}
	if d2 := *s.Info().IdleDeadline; !d2.After(d1) {
		t.Fatalf("deadline not moved: %v → %v", d1, d2)
	}

	started := time.Now()
	if got := e.read(); !strings.HasPrefix(got, "detach -i ") {
		t.Fatalf("engine got %q, want detach", got)
	}
	if time.Since(started) < 200*time.Millisecond {
		t.Fatal("released before the moved deadline")
	}
	e.conn.Close()
	waitFor(t, "close", s.Closed)
	if in := s.Info(); in.State != "detached" || in.ClosedReason != ReasonIdleDetached {
		t.Fatalf("tombstone: %+v", in)
	}
}

func TestControl_RefusedWhileIDEDrives(t *testing.T) {
	a := newAPIHarness(t, false)
	e, d, s := a.connect(t)
	pause(t, e, d, "1")
	if in := s.Info(); in.Driver != DriverIDE || in.IdleDeadline != nil {
		t.Fatalf("info: %+v", in)
	}
	for _, r := range []string{"run", "step-into", "step-over", "step-out", "stop", "detach"} {
		if code, body := a.do(t, "POST", "/api/xdebug/sessions/"+s.ID+"/"+r, "{}"); code != http.StatusConflict || !strings.Contains(body, `"ide_attached"`) {
			t.Fatalf("%s: %d %s", r, code, body)
		}
	}
	// Only JSON: a cross-site form must not step a session.
	if code, _ := a.doType(t, "POST", "/api/xdebug/sessions/"+s.ID+"/run", "application/x-www-form-urlencoded", "a=1"); code != http.StatusUnsupportedMediaType {
		t.Fatalf("form post: %d", code)
	}
	d.expectQuiet(100 * time.Millisecond)
}

func TestAPI_Settings(t *testing.T) {
	a := newAPIHarness(t, false)
	a.proxy.settings = Settings{Mode: ModeAuto}
	if code, body := a.do(t, "GET", "/api/xdebug", ""); code != http.StatusOK || !strings.Contains(body, `"mode":"auto"`) || !strings.Contains(body, `"idle_timeout_seconds":600`) {
		t.Fatalf("status: %d %s", code, body)
	}
	if code, body := a.do(t, "PUT", "/api/xdebug/settings", `{"pause_on_first_line":true}`); code != http.StatusOK || !strings.Contains(body, `"mode":"auto"`) || !strings.Contains(body, `"pause_on_first_line":true`) {
		t.Fatalf("put: %d %s", code, body)
	}
	if code, _ := a.do(t, "PUT", "/api/xdebug/settings", `{"mode":"ide"}`); code != http.StatusBadRequest {
		t.Fatalf("bad mode: %d", code)
	}
	if code, _ := a.doType(t, "PUT", "/api/xdebug/settings", "text/plain", `{"mode":"proxy"}`); code != http.StatusUnsupportedMediaType {
		t.Fatalf("text/plain: %d", code)
	}
	if st := a.proxy.Settings(); st.Mode != ModeAuto || !st.PauseOnFirstLine {
		t.Fatalf("settings: %+v", st)
	}
}

func TestNew_ModeDefaults(t *testing.T) {
	cases := []struct {
		cfg  Config
		mode string
		idle time.Duration
	}{
		{Config{}, ModeAuto, 10 * time.Minute},
		{Config{Mode: "proxy", IdleTimeout: time.Minute}, ModeProxy, time.Minute},
		{Config{Mode: "bogus", IdleTimeout: -1}, ModeAuto, 0},
	}
	for _, c := range cases {
		m := New(c.cfg)
		if st := m.proxy.Settings(); st.Mode != c.mode || m.proxy.idleTimeout != c.idle {
			t.Errorf("%+v: mode %s idle %v", c.cfg, st.Mode, m.proxy.idleTimeout)
		}
	}
}
