package xdebug

import (
	"strings"
	"testing"
	"time"
)

// hereEngine answers a session driven here: continuations go to the
// scripted lines, reads get fixed replies. At the end it closes the engine.
func hereEngine(t *testing.T, e *fakeEngine, lines map[string]int) {
	send := func(xml string) { _, _ = e.conn.Write(encodePacket([]byte(xml))) }
	for {
		line, err := readCommand(e.r)
		if err != nil {
			return
		}
		name := cmdNameRe.FindString(line)
		tid := tidRe.FindStringSubmatch(line)[1]
		switch name {
		case "step_over", "step_into", "step_out", "run":
			if n, ok := lines[name]; ok {
				send(response(name, tid, ` status="break" reason="ok"`,
					`<xdebug:message filename="file:///app/src/User.php" lineno="`+itoa(n)+`"></xdebug:message>`))
			} else {
				send(response(name, tid, ` status="stopping" reason="ok"`, ""))
			}
		case "stop":
			send(response(name, tid, ` status="stopped" reason="ok"`, ""))
			e.conn.Close()
			return
		case "detach":
			send(response(name, tid, ` status="stopping" reason="ok"`, ""))
			e.conn.Close()
			return
		case "stack_get":
			send(response(name, tid, "", `<stack where="App\User->save" level="0" type="file" filename="file:///app/src/User.php" lineno="13"></stack>`))
		case "source":
			send(response(name, tid, ` encoding="base64"`, b64("$a = 1;\n$b = 2;\n")))
		default:
			send(response(name, tid, "", `<error code="4"><message><![CDATA[unimplemented]]></message></error>`))
		}
	}
}

func TestMCPControl_StepContinueOnSessionDrivenHere(t *testing.T) {
	a := newHereHarness(t, ModeAuto)
	cs := connectToolsWith(t, a.proxy, false, true)
	e, s, first := startHere(t, a.proxy)
	breakAt(t, e, first, "run", 12)
	waitFor(t, "break", s.Paused)
	go hereEngine(t, e, map[string]int{"step_over": 13})

	out, isErr := callTool(t, cs, "xdebug_step", map[string]any{})
	if isErr || !strings.Contains(out, `"line": 13`) || !strings.Contains(out, `App\\User->save`) || !strings.Contains(out, `"source"`) {
		t.Fatalf("step: %s", out)
	}
	if in := s.Info(); in.LastContinuationBy == nil || *in.LastContinuationBy != ByAgent || in.LastContinuationAg != "c" {
		t.Fatalf("driver info: %+v", in)
	}

	// No more breaks: the script ends, the driver says stop, the tool reports it.
	out, isErr = callTool(t, cs, "xdebug_continue", map[string]any{"session_id": s.ID})
	if isErr || !strings.Contains(out, `"closed_reason": "stopped"`) {
		t.Fatalf("continue: %s", out)
	}
	if log := s.AgentLog(); len(log) < 2 || log[0].Tool != "xdebug_step" || log[1].Tool != "xdebug_continue" || log[1].Result != "ended: stopped" {
		t.Fatalf("agent log: %+v", log)
	}
}

func TestMCPControl_StopDetachesByDefault(t *testing.T) {
	a := newHereHarness(t, ModeAuto)
	cs := connectToolsWith(t, a.proxy, false, true)
	e, s, first := startHere(t, a.proxy)
	breakAt(t, e, first, "run", 12)
	waitFor(t, "break", s.Paused)
	go hereEngine(t, e, nil)

	if out, isErr := callTool(t, cs, "xdebug_stop", map[string]any{}); isErr || !strings.Contains(out, `"closed_reason": "detached"`) {
		t.Fatalf("stop: %s", out)
	}
	if out, isErr := callTool(t, cs, "xdebug_stop", map[string]any{"session_id": s.ID, "mode": "kill"}); !isErr || !strings.Contains(out, "detach or stop") {
		t.Fatalf("bad mode: %s", out)
	}
}

func TestMCPControl_RefusedWhileIDEDrives(t *testing.T) {
	h := newHarness(t)
	cs := connectToolsWith(t, h.proxy, false, true)
	e, d, s := h.connect(t)
	pause(t, e, d, "1")
	for _, tool := range []string{"xdebug_step", "xdebug_continue", "xdebug_stop"} {
		if out, isErr := callTool(t, cs, tool, map[string]any{"session_id": s.ID}); !isErr || !strings.Contains(out, "IDE drives") {
			t.Fatalf("%s: %s", tool, out)
		}
	}
	d.expectQuiet(100 * time.Millisecond)
}

func TestMCPControl_Wait(t *testing.T) {
	a := newHereHarness(t, ModeAuto)
	cs := connectToolsWith(t, a.proxy, false, true)

	started := time.Now()
	if out, isErr := callTool(t, cs, "xdebug_wait", map[string]any{"timeout_seconds": 1}); !isErr || !strings.Contains(out, "no session paused") {
		t.Fatalf("empty wait: %s", out)
	}
	if time.Since(started) < 900*time.Millisecond {
		t.Fatal("wait returned before its timeout")
	}

	// The request starts while the agent waits.
	type res struct {
		out   string
		isErr bool
	}
	got := make(chan res, 1)
	go func() {
		out, isErr := callTool(t, cs, "xdebug_wait", map[string]any{"timeout_seconds": 10})
		got <- res{out, isErr}
	}()
	e, s, first := startHere(t, a.proxy)
	breakAt(t, e, first, "run", 12)
	go hereEngine(t, e, nil)
	r := <-got
	if r.isErr || !strings.Contains(r.out, `"session_id": "`+s.ID+`"`) || !strings.Contains(r.out, `"driver": "here"`) || !strings.Contains(r.out, `"line": 12`) {
		t.Fatalf("wait: %s", r.out)
	}
}

func TestWaitSeconds(t *testing.T) {
	for _, c := range []struct {
		n    int
		want time.Duration
	}{{0, 30 * time.Second}, {-3, 30 * time.Second}, {5, 5 * time.Second}, {99, 30 * time.Second}} {
		if got := waitSeconds(c.n, 30*time.Second, 30*time.Second); got != c.want {
			t.Errorf("%d: %v", c.n, got)
		}
	}
}
