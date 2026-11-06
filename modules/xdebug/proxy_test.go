package xdebug

import (
	"bufio"
	"errors"
	"fmt"
	"net"
	"regexp"
	"strings"
	"testing"
	"time"
)

const initXML = `<?xml version="1.0" encoding="iso-8859-1"?>
<init xmlns="urn:debugger_protocol_v1" xmlns:xdebug="https://xdebug.org/dbgp/xdebug" fileuri="file:///app/public/index.php" language="PHP" xdebug:language_version="8.3.0" protocol_version="1.0" appid="42" idekey="PHPSTORM"><engine version="3.3.2"><![CDATA[Xdebug]]></engine></init>`

// tcpPair returns two connected loopback TCP connections.
func tcpPair(t *testing.T) (net.Conn, net.Conn) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	accepted := make(chan net.Conn, 1)
	go func() {
		c, _ := ln.Accept()
		accepted <- c
	}()
	a, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	b := <-accepted
	t.Cleanup(func() { a.Close(); b.Close() })
	return a, b
}

// fakeEngine is the Xdebug side of a session.
type fakeEngine struct {
	t    *testing.T
	conn net.Conn
	r    *bufio.Reader
}

func (e *fakeEngine) send(xml string) {
	e.t.Helper()
	if _, err := e.conn.Write(encodePacket([]byte(xml))); err != nil {
		e.t.Fatal(err)
	}
}

func (e *fakeEngine) read() string {
	e.t.Helper()
	_ = e.conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	line, err := readCommand(e.r)
	if err != nil {
		e.t.Fatalf("engine read: %v", err)
	}
	return line
}

// fakeIDE is the IDE side of a session.
type fakeIDE struct {
	t    *testing.T
	conn net.Conn
	r    *bufio.Reader
}

func (d *fakeIDE) send(line string) {
	d.t.Helper()
	if _, err := d.conn.Write(append([]byte(line), 0)); err != nil {
		d.t.Fatal(err)
	}
}

func (d *fakeIDE) read() string {
	d.t.Helper()
	_ = d.conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	body, err := readPacket(d.r)
	if err != nil {
		d.t.Fatalf("ide read: %v", err)
	}
	return string(body)
}

// expectQuiet asserts the IDE receives nothing within d.
func (d *fakeIDE) expectQuiet(dur time.Duration) {
	d.t.Helper()
	_ = d.conn.SetReadDeadline(time.Now().Add(dur))
	body, err := readPacket(d.r)
	if err == nil {
		d.t.Fatalf("IDE got an unexpected packet: %s", body)
	}
	var ne net.Error
	if !errors.As(err, &ne) || !ne.Timeout() {
		d.t.Fatalf("ide: %v", err)
	}
}

type harness struct {
	proxy *Proxy
	ideLn net.Listener
}

func newHarness(t *testing.T) *harness {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ln.Close() })
	p := NewProxy(ln.Addr().String())
	p.commandTimeout = 2 * time.Second
	return &harness{proxy: p, ideLn: ln}
}

// connect starts a session: Xdebug connects, sends init, the IDE accepts it.
func (h *harness) connect(t *testing.T) (*fakeEngine, *fakeIDE, *Session) {
	t.Helper()
	before := len(h.proxy.Sessions())
	engineSide, proxySide := tcpPair(t)
	done := make(chan struct{})
	go func() { h.proxy.HandleConnection(proxySide); close(done) }()
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

	_ = h.ideLn.(*net.TCPListener).SetDeadline(time.Now().Add(3 * time.Second))
	ideConn, err := h.ideLn.Accept()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ideConn.Close() })
	d := &fakeIDE{t: t, conn: ideConn, r: bufio.NewReader(ideConn)}
	if got := d.read(); got != initXML {
		t.Fatalf("IDE init differs:\n%s", got)
	}
	return e, d, h.waitSession(t, before+1)
}

func (h *harness) waitSession(t *testing.T, n int) *Session {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if ss := h.proxy.Sessions(); len(ss) >= n {
			return ss[n-1]
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("session %d not registered", n)
	return nil
}

var tidRe = regexp.MustCompile(`-i (\d+)`)

func tidOf(t *testing.T, line string) string {
	t.Helper()
	m := tidRe.FindStringSubmatch(line)
	if m == nil {
		t.Fatalf("no -i in %q", line)
	}
	return m[1]
}

func response(cmd, tid, attrs, inner string) string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="iso-8859-1"?>
<response xmlns="urn:debugger_protocol_v1" xmlns:xdebug="https://xdebug.org/dbgp/xdebug" command="%s" transaction_id="%s"%s>%s</response>`, cmd, tid, attrs, inner)
}

// pause makes the IDE run the script to a breakpoint.
func pause(t *testing.T, e *fakeEngine, d *fakeIDE, ideTID string) {
	t.Helper()
	d.send("run -i " + ideTID)
	got := e.read()
	e.send(response("run", tidOf(t, got), ` status="break" reason="ok"`,
		`<xdebug:message filename="file:///app/src/User.php" lineno="12"></xdebug:message>`))
	if r := d.read(); !strings.Contains(r, `transaction_id="`+ideTID+`"`) || !strings.Contains(r, `status="break"`) {
		t.Fatalf("IDE run reply: %s", r)
	}
}

func TestProxy_IDECommandsKeepTheirTransactionIDs(t *testing.T) {
	h := newHarness(t)
	e, d, s := h.connect(t)

	if info := s.Info(); info.IDEKey != "PHPSTORM" || info.FileURI != "file:///app/public/index.php" || info.Status != "starting" {
		t.Fatalf("init not parsed: %+v", info)
	}

	// Quoted file name and base64 data contain "-i 1" and must reach the engine untouched.
	cmd := `breakpoint_set -i 7 -t line -f "file:///app/a -i 1.php" -n 3 -- JGEgLWkgMQ==`
	d.send(cmd)
	got := e.read()
	ptid := tidOf(t, got)
	want := `breakpoint_set -i ` + ptid + ` -t line -f "file:///app/a -i 1.php" -n 3 -- JGEgLWkgMQ==`
	if got != want {
		t.Fatalf("engine got %q, want %q", got, want)
	}
	e.send(response("breakpoint_set", ptid, ` state="enabled" id="100"`, ""))
	if r := d.read(); !strings.Contains(r, `transaction_id="7"`) || !strings.Contains(r, `id="100"`) {
		t.Fatalf("IDE reply: %s", r)
	}

	// Notifications and streams pass through as is.
	notify := `<notify xmlns="urn:debugger_protocol_v1" name="breakpoint_resolved"></notify>`
	e.send(notify)
	if r := d.read(); r != notify {
		t.Fatalf("notify: %s", r)
	}

	pause(t, e, d, "8")
	if info := s.Info(); info.Status != "break" || info.File != "file:///app/src/User.php" || info.Line != 12 {
		t.Fatalf("break not tracked: %+v", info)
	}
}

func TestProxy_AgentRepliesNeverReachTheIDE(t *testing.T) {
	h := newHarness(t)
	e, d, s := h.connect(t)
	pause(t, e, d, "1")

	type res struct {
		body []byte
		err  error
	}
	agent := make(chan res, 1)
	go func() {
		b, err := s.Command("stack_get", "-d 0", nil)
		agent <- res{b, err}
	}()

	got := e.read()
	if !strings.HasPrefix(got, "stack_get -i ") {
		t.Fatalf("engine got %q", got)
	}
	// The IDE asks for status while the agent's command is in flight.
	d.send("status -i 2")
	statusCmd := e.read()
	// Answer in reverse order: routing is by transaction id, not by order.
	e.send(response("status", tidOf(t, statusCmd), ` status="break" reason="ok"`, ""))
	e.send(response("stack_get", tidOf(t, got), "", `<stack where="{main}" level="0" type="file" filename="file:///app/src/User.php" lineno="12"></stack>`))

	r := <-agent
	if r.err != nil || !strings.Contains(string(r.body), `lineno="12"`) {
		t.Fatalf("agent: %v %s", r.err, r.body)
	}
	if ide := d.read(); !strings.Contains(ide, `command="status"`) || !strings.Contains(ide, `transaction_id="2"`) {
		t.Fatalf("IDE reply: %s", ide)
	}
	d.expectQuiet(100 * time.Millisecond)
}

func TestProxy_AgentOnlyWhilePaused(t *testing.T) {
	h := newHarness(t)
	e, d, s := h.connect(t)

	if _, err := s.Command("stack_get", "", nil); !errors.Is(err, errNotPaused) {
		t.Fatalf("starting: %v", err)
	}
	pause(t, e, d, "1")

	// The IDE continues: the agent is refused at once, before the engine answers.
	d.send("step_over -i 2")
	e.read()
	if _, err := s.Command("stack_get", "", nil); !errors.Is(err, errNotPaused) {
		t.Fatalf("running: %v", err)
	}

	// Write commands are not on the allow-list.
	for _, c := range []string{"run", "step_into", "breakpoint_set", "feature_set", "property_set", "stop"} {
		if _, err := s.Command(c, "", nil); err == nil {
			t.Fatalf("%s allowed", c)
		}
	}
}

func TestProxy_NoIDEDetaches(t *testing.T) {
	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	addr := ln.Addr().String()
	ln.Close() // nobody listens here

	p := NewProxy(addr)
	engineSide, proxySide := tcpPair(t)
	done := make(chan struct{})
	go func() { p.HandleConnection(proxySide); close(done) }()

	e := &fakeEngine{t: t, conn: engineSide, r: bufio.NewReader(engineSide)}
	e.send(initXML)
	if got := e.read(); !strings.HasPrefix(got, "detach -i ") {
		t.Fatalf("engine got %q, want detach", got)
	}
	<-done
	if n := len(p.Sessions()); n != 0 {
		t.Fatalf("%d sessions left", n)
	}
}

func TestProxy_IDEDisconnectDetaches(t *testing.T) {
	h := newHarness(t)
	e, d, _ := h.connect(t)
	pause(t, e, d, "1")

	d.conn.Close()
	if got := e.read(); !strings.HasPrefix(got, "detach -i ") {
		t.Fatalf("engine got %q, want detach", got)
	}
	e.conn.Close()
	deadline := time.Now().Add(3 * time.Second)
	for len(h.proxy.Sessions()) > 0 && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	if n := len(h.proxy.Sessions()); n != 0 {
		t.Fatalf("%d sessions left", n)
	}
}

func TestProxy_SessionsAreIndependent(t *testing.T) {
	h := newHarness(t)
	e1, d1, s1 := h.connect(t)
	e2, d2, s2 := h.connect(t)
	if s1.ID == s2.ID {
		t.Fatal("same session id")
	}

	if _, err := h.proxy.Session(""); err == nil {
		t.Fatal("no paused session must be an error")
	}
	pause(t, e2, d2, "1")
	if s, err := h.proxy.Session(""); err != nil || s != s2 {
		t.Fatalf("only paused session: %v", err)
	}
	pause(t, e1, d1, "1")
	if _, err := h.proxy.Session(""); err == nil {
		t.Fatal("two paused sessions need session_id")
	}
	if s, err := h.proxy.Session(s1.ID); err != nil || s != s1 {
		t.Fatalf("by id: %v", err)
	}

	// Session 1 ends; session 2 stays usable.
	e1.conn.Close()
	deadline := time.Now().Add(3 * time.Second)
	for len(h.proxy.Sessions()) != 1 && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	if ss := h.proxy.Sessions(); len(ss) != 1 || ss[0] != s2 {
		t.Fatalf("sessions after close: %d", len(ss))
	}
	go func() {
		got := e2.read()
		e2.send(response("status", tidOf(t, got), ` status="break" reason="ok"`, ""))
	}()
	if _, err := s2.Command("status", "", nil); err != nil {
		t.Fatal(err)
	}
}

func TestProxy_AgentCallFailsWhenEngineCloses(t *testing.T) {
	h := newHarness(t)
	e, d, s := h.connect(t)
	pause(t, e, d, "1")

	errc := make(chan error, 1)
	go func() {
		_, err := s.Command("stack_get", "", nil)
		errc <- err
	}()
	e.read()
	e.conn.Close()
	select {
	case err := <-errc:
		if !errors.Is(err, errSessionClosed) {
			t.Fatalf("got %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("agent call hangs after the engine closed")
	}
}

func TestProxy_ConcurrentAgentCallsQueue(t *testing.T) {
	h := newHarness(t)
	e, d, s := h.connect(t)
	pause(t, e, d, "1")

	stop := make(chan struct{})
	defer close(stop)
	go serveEngine(t, e, stop)

	errc := make(chan error, 3)
	for i := 0; i < 3; i++ {
		go func() {
			_, err := s.Command("stack_get", "", nil)
			errc <- err
		}()
	}
	for i := 0; i < 3; i++ {
		if err := <-errc; err != nil {
			t.Fatalf("call %d: %v", i, err)
		}
	}
	d.expectQuiet(100 * time.Millisecond)
}
