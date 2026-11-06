package xdebug

import (
	"strings"
	"testing"
	"time"
)

// A property name with a NUL must not split into a second engine command:
// that would let the read-only agent send a continuation (review of #363).
func TestPropertyNameCannotInjectCommand(t *testing.T) {
	h := newHarness(t)
	e, d, s := h.connect(t)
	pause(t, e, d, "1")

	cs := connectTools(t, h.proxy, false)
	out, isErr := callTool(t, cs, "xdebug_property", map[string]any{"name": "$x\x00run -i 999"})
	if !isErr || !strings.Contains(out, "NUL") {
		t.Fatalf("want a NUL error, got %v %s", isErr, out)
	}
	if _, err := s.Command("context_get", "-d 0\x00stop -i 9", nil); err == nil {
		t.Fatal("Command accepted NUL in args")
	}

	// Nothing reached the engine, and the session is still paused.
	_ = e.conn.SetReadDeadline(time.Now().Add(200 * time.Millisecond))
	if line, err := readCommand(e.r); err == nil {
		t.Fatalf("engine got %q", line)
	}
	if !s.Paused() {
		t.Fatal("session left break")
	}
}

// A reply that arrives after the IDE sent a continuation must not report the
// session paused while the engine runs (review of #363).
func TestStatusReplyAfterContinuation(t *testing.T) {
	h := newHarness(t)
	e, d, s := h.connect(t)
	pause(t, e, d, "1")

	d.send("status -i 2")
	statusCmd := e.read()
	d.send("run -i 3")
	runCmd := e.read() // run is now in flight: the engine runs
	e.send(response("status", tidOf(t, statusCmd), ` status="break" reason="ok"`, ""))
	_ = d.read()

	if s.Paused() {
		t.Fatal("session reports paused while the IDE's run is in flight")
	}
	if _, err := s.Command("stack_get", "", nil); err == nil {
		t.Fatal("agent command sent behind the continuation")
	}

	// The continuation's own reply sets the status.
	e.send(response("run", tidOf(t, runCmd), ` status="break" reason="ok"`,
		`<xdebug:message filename="file:///app/b.php" lineno="7"></xdebug:message>`))
	_ = d.read()
	if !s.Paused() || s.Info().LastLine != 7 {
		t.Fatalf("not paused after run reply: %+v", s.Info())
	}
}

// A quoted value containing " -- " before -i must not hide the tid.
func TestCommandTID_QuotedDashes(t *testing.T) {
	line := `breakpoint_set -t line -f "file:///my -- app/a.php" -i 5 -n 3 -- JA==`
	name, s, e, ok := commandTID(line)
	if !ok || name != "breakpoint_set" || line[s:e] != "5" {
		t.Fatalf("ok=%v name=%q", ok, name)
	}
}

// Two continuations in flight: the first one's reply must not report the
// session paused while the second is queued (review of #363).
func TestTwoContinuationsInFlight(t *testing.T) {
	h := newHarness(t)
	e, d, s := h.connect(t)
	pause(t, e, d, "1")
	d.send("step_over -i 2")
	first := e.read()
	d.send("step_over -i 3")
	second := e.read()
	e.send(response("step_over", tidOf(t, first), ` status="break" reason="ok"`, ""))
	_ = d.read()
	if s.Paused() {
		t.Fatal("paused while the second step_over is in flight")
	}
	e.send(response("step_over", tidOf(t, second), ` status="break" reason="ok"`, ""))
	_ = d.read()
	if !s.Paused() {
		t.Fatal("not paused after the last continuation's reply")
	}
}

// Real Xdebug 3.4 marks plain replies, even errors, with status="break".
// They must not wipe the current location (found in a live run).
func TestPlainBreakReplyKeepsLocation(t *testing.T) {
	h := newHarness(t)
	e, d, s := h.connect(t)
	pause(t, e, d, "1")

	d.send("property_get -i 2 -n $nope")
	got := e.read()
	e.send(response("property_get", tidOf(t, got), ` status="break" reason="ok"`,
		`<error code="300"><message><![CDATA[can not get property]]></message></error>`))
	_ = d.read()
	if in := s.Info(); in.LastFile != "file:///app/src/User.php" || in.LastLine != 12 || in.State != "break" {
		t.Fatalf("location lost: %+v", in)
	}
}
