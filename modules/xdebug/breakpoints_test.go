package xdebug

import (
	"bufio"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/buggregator/go-buggregator/internal/storage"
)

// bpEngine answers a session driven here and records every command.
type bpEngine struct {
	mu     sync.Mutex
	seen   []string
	nextID int
	hits   map[string]int // engine id → hit_count in breakpoint_list
	ids    []string       // set breakpoints, in order
	lines  map[string]int // continuation → break line
}

func (b *bpEngine) commands(prefix string) []string {
	b.mu.Lock()
	defer b.mu.Unlock()
	var out []string
	for _, c := range b.seen {
		if strings.HasPrefix(c, prefix) {
			out = append(out, c)
		}
	}
	return out
}

func (b *bpEngine) serve(e *fakeEngine) {
	send := func(xml string) { _, _ = e.conn.Write(encodePacket([]byte(xml))) }
	for {
		line, err := readCommand(e.r)
		if err != nil {
			return
		}
		name := cmdNameRe.FindString(line)
		tid := tidRe.FindStringSubmatch(line)[1]
		b.mu.Lock()
		b.seen = append(b.seen, line)
		b.mu.Unlock()
		switch name {
		case "feature_set":
			send(response(name, tid, ` feature="x" success="1"`, ""))
		case "breakpoint_set":
			b.mu.Lock()
			b.nextID++
			id := strconv.Itoa(9000 + b.nextID)
			b.ids = append(b.ids, id)
			b.mu.Unlock()
			send(response(name, tid, ` id="`+id+`" state="enabled" resolved="unresolved"`, ""))
		case "breakpoint_remove":
			send(response(name, tid, "", ""))
		case "breakpoint_list":
			b.mu.Lock()
			var inner string
			for _, id := range b.ids {
				inner += `<breakpoint type="line" id="` + id + `" state="enabled" resolved="resolved" lineno="21" hit_count="` + strconv.Itoa(b.hits[id]) + `"></breakpoint>`
			}
			b.mu.Unlock()
			send(response(name, tid, "", inner))
		case "run", "step_over", "step_into":
			b.mu.Lock()
			n, ok := b.lines[name]
			b.mu.Unlock()
			if ok {
				send(response(name, tid, ` status="break" reason="ok"`, `<xdebug:message filename="file:///app/src/User.php" lineno="`+strconv.Itoa(n)+`"></xdebug:message>`))
			}
		default:
			send(response(name, tid, "", `<error code="4"><message><![CDATA[unimplemented]]></message></error>`))
		}
	}
}

// startBP starts a session driven here whose engine is a bpEngine.
func startBP(t *testing.T, p *Proxy, be *bpEngine) (*fakeEngine, *Session) {
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
	go be.serve(e)
	h := &harness{proxy: p}
	return e, h.waitSession(t, before+1)
}

func TestBreakpoints_SentAtStartWithState(t *testing.T) {
	a := newHereHarness(t, ModeAuto)
	line, _ := a.proxy.AddBreakpoint(SavedBreakpoint{File: "/app/src/User.php", Line: 20, Enabled: true})
	exc, _ := a.proxy.AddBreakpoint(SavedBreakpoint{Exception: `\Throwable`, Enabled: true})
	off, _ := a.proxy.AddBreakpoint(SavedBreakpoint{File: "/app/src/Off.php", Line: 3, Enabled: false})
	// A condition saved while eval was on; eval is off now: skipped, never sent.
	cond, _ := a.proxy.bps.Add(SavedBreakpoint{File: "/app/src/User.php", Line: 30, Condition: "$id == 7", Enabled: true})

	be := &bpEngine{lines: map[string]int{"run": 12}, hits: map[string]int{"9001": 2}}
	e, s := startBP(t, a.proxy, be)
	waitFor(t, "break", s.Paused)

	sets := be.commands("breakpoint_set")
	if len(sets) != 2 || !strings.Contains(sets[0], "-t line -f file:///app/src/User.php -n 20") || !strings.Contains(sets[1], `-t exception -x "\\Throwable"`) {
		t.Fatalf("breakpoint_set: %q", sets)
	}
	// Breakpoints go out before the script runs.
	be.mu.Lock()
	order := strings.Join(be.seen, "|")
	be.mu.Unlock()
	if strings.Index(order, "breakpoint_set") > strings.Index(order, "run -i") {
		t.Fatalf("run before breakpoints: %s", order)
	}

	// Xdebug resolves the line breakpoint to the next statement.
	e.send(`<notify xmlns="urn:debugger_protocol_v1" name="breakpoint_resolved"><breakpoint type="line" resolved="resolved" filename="file:///app/src/User.php" lineno="21" state="enabled" hit_count="0" hit_value="0" id="9001"></breakpoint></notify>`)
	waitFor(t, "breakpoint.resolved", func() bool { return a.rec.count("breakpoint.resolved") == 1 })

	code, body := a.do(t, "GET", "/api/xdebug/sessions/"+s.ID+"/breakpoints", "")
	if code != http.StatusOK {
		t.Fatalf("session breakpoints: %d %s", code, body)
	}
	for _, want := range []string{
		`"id":` + itoa(int(line.ID)) + `,"type":"line","file":"file:///app/src/User.php","line":20,"enabled":true,"source":"ui"`,
		`"state":"resolved","resolved_line":21,"hit_count":2`,
		`"id":` + itoa(int(exc.ID)) + `,"type":"exception","exception":"\\Throwable"`,
		`"id":` + itoa(int(off.ID)) + `,"type":"line","file":"file:///app/src/Off.php","line":3,"enabled":false`,
		`"state":"disabled"`,
		`"id":` + itoa(int(cond.ID)),
		`"state":"skipped"`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("missing %s in %s", want, body)
		}
	}
}

func TestBreakpoints_LiveChangesWhilePausedAndRunning(t *testing.T) {
	a := newHereHarness(t, ModeAuto)
	be := &bpEngine{lines: map[string]int{"run": 12, "step_over": 13}}
	_, s := startBP(t, a.proxy, be)
	waitFor(t, "break", s.Paused)

	// Paused: a new breakpoint goes out at once.
	code, body := a.do(t, "POST", "/api/xdebug/breakpoints", `{"file":"/app/src/A.php","line":5}`)
	if code != http.StatusCreated || !strings.Contains(body, `"file":"file:///app/src/A.php"`) {
		t.Fatalf("create: %d %s", code, body)
	}
	waitFor(t, "breakpoint_set", func() bool { return len(be.commands("breakpoint_set")) == 1 })
	if a.rec.count("breakpoints.changed") != 1 {
		t.Fatalf("events: %v", a.rec.names())
	}

	// Disable it: removed from the engine.
	id := strings.SplitN(strings.SplitN(body, `"id":`, 2)[1], ",", 2)[0]
	if code, body := a.do(t, "PATCH", "/api/xdebug/breakpoints/"+id, `{"enabled":false}`); code != http.StatusOK || !strings.Contains(body, `"enabled":false`) {
		t.Fatalf("patch: %d %s", code, body)
	}
	waitFor(t, "breakpoint_remove", func() bool { return len(be.commands("breakpoint_remove -i")) == 1 })
	if rm := be.commands("breakpoint_remove")[0]; !strings.HasSuffix(rm, "-d 9001") {
		t.Fatalf("remove: %q", rm)
	}

	// Delete it: nothing left to remove in the engine, the list is empty.
	if code, _ := a.do(t, "DELETE", "/api/xdebug/breakpoints/"+id, ""); code != http.StatusNoContent {
		t.Fatalf("delete: %d", code)
	}
	waitFor(t, "empty", func() bool { return len(s.SessionBreakpoints(a.proxy.bps.List())) == 0 })
}

func TestBreakpoints_PendingWhileRunningSentAtNextPause(t *testing.T) {
	a := newHereHarness(t, ModeAuto)
	be := &bpEngine{lines: map[string]int{"run": 12}}
	e, s := startBP(t, a.proxy, be)
	waitFor(t, "break", s.Paused)

	// Step: the engine does not answer yet, so the script runs.
	be.mu.Lock()
	be.lines = map[string]int{}
	be.mu.Unlock()
	if err := s.Continue("step_over", ByUI, ""); err != nil {
		t.Fatal(err)
	}
	waitFor(t, "step sent", func() bool { return len(be.commands("step_over")) == 1 })
	if _, err := a.proxy.AddBreakpoint(SavedBreakpoint{File: "/app/src/B.php", Line: 9, Enabled: true}); err != nil {
		t.Fatal(err)
	}
	time.Sleep(50 * time.Millisecond)
	if n := len(be.commands("breakpoint_set")); n != 0 {
		t.Fatalf("sent while running: %d", n)
	}
	if bps := s.SessionBreakpoints(a.proxy.bps.List()); len(bps) != 1 || bps[0].State != BPPending {
		t.Fatalf("state while running: %+v", bps)
	}

	// The step lands: the breakpoint goes out.
	step := be.commands("step_over")[0]
	e.send(response("step_over", tidOf(t, step), ` status="break" reason="ok"`, `<xdebug:message filename="file:///app/src/User.php" lineno="13"></xdebug:message>`))
	waitFor(t, "breakpoint_set at the pause", func() bool { return len(be.commands("breakpoint_set")) == 1 })
}

func TestBreakpoints_API(t *testing.T) {
	a := newAPIHarness(t, false)
	for _, c := range []struct {
		body string
		code int
		err  string
	}{
		{`{"file":"/app/a.php","line":3,"condition":"$x > 1"}`, http.StatusForbidden, "eval_disabled"},
		{`{"file":"/app/a.php"}`, http.StatusBadRequest, "file and line"},
		{`{"type":"watch","file":"/app/a.php","line":1}`, http.StatusBadRequest, "type must be"},
		{`{"exception":"Foo Bar"}`, http.StatusBadRequest, "class name"},
		{`{"file":"/app/a.php","line":3,"color":"red"}`, http.StatusBadRequest, "bad_request"},
	} {
		if code, body := a.do(t, "POST", "/api/xdebug/breakpoints", c.body); code != c.code || !strings.Contains(body, c.err) {
			t.Errorf("%s: %d %s", c.body, code, body)
		}
	}
	if code, _ := a.doType(t, "POST", "/api/xdebug/breakpoints", "text/plain", `{"file":"/a.php","line":1}`); code != http.StatusUnsupportedMediaType {
		t.Fatalf("text/plain: %d", code)
	}
	if code, body := a.do(t, "POST", "/api/xdebug/breakpoints", `{"exception":"\\RuntimeException"}`); code != http.StatusCreated || !strings.Contains(body, `"type":"exception"`) {
		t.Fatalf("exception: %d %s", code, body)
	}
	if code, body := a.do(t, "GET", "/api/xdebug/breakpoints", ""); code != http.StatusOK || !strings.Contains(body, `"exception":"\\RuntimeException"`) {
		t.Fatalf("list: %d %s", code, body)
	}
	if code, _ := a.do(t, "PATCH", "/api/xdebug/breakpoints/99", `{"enabled":false}`); code != http.StatusNotFound {
		t.Fatalf("patch missing: %d", code)
	}
	if code, _ := a.do(t, "PATCH", "/api/xdebug/breakpoints/1", `{"condition":"$a"}`); code != http.StatusForbidden {
		t.Fatalf("patch condition: %d", code)
	}
	if code, _ := a.do(t, "DELETE", "/api/xdebug/breakpoints/1", ""); code != http.StatusNoContent {
		t.Fatalf("delete: %d", code)
	}
	if code, _ := a.do(t, "DELETE", "/api/xdebug/breakpoints/1", ""); code != http.StatusNotFound {
		t.Fatalf("delete again: %d", code)
	}
}

func TestBreakpoints_ConditionWithEval(t *testing.T) {
	a := newAPIHarness(t, true)
	a.proxy.allowEval = true
	if code, body := a.do(t, "POST", "/api/xdebug/breakpoints", `{"file":"/app/a.php","line":3,"condition":"$x > 1"}`); code != http.StatusCreated {
		t.Fatalf("%d %s", code, body)
	}
	got := breakpointSetCmd(a.proxy.bps.List()[0])
	if got != "breakpoint_set -t conditional -f file:///app/a.php -n 3 -- "+b64("$x > 1") {
		t.Fatalf("cmd: %s", got)
	}
}

func TestBreakpoints_PersistInSQLite(t *testing.T) {
	db, err := storage.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	mig := storage.NewMigrator(db)
	if err := New(Config{}).RegisterMigrations(mig); err != nil {
		t.Fatal(err)
	}
	if err := mig.Run(); err != nil {
		t.Fatal(err)
	}
	st := &bpStore{}
	if err := st.load(db); err != nil {
		t.Fatal(err)
	}
	b, err := st.Add(SavedBreakpoint{File: "/app/a.php", Line: 3, Enabled: true, Source: "mcp"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := st.Update(b.ID, func(b *SavedBreakpoint) error { b.Line = 4; return nil }); err != nil {
		t.Fatal(err)
	}
	if _, err := st.Add(SavedBreakpoint{Exception: `\Throwable`, Enabled: true}); err != nil {
		t.Fatal(err)
	}
	if err := st.Delete(b.ID + 1); err != nil {
		t.Fatal(err)
	}

	again := &bpStore{}
	if err := again.load(db); err != nil {
		t.Fatal(err)
	}
	list := again.List()
	if len(list) != 1 || list[0].Line != 4 || list[0].File != "file:///app/a.php" || list[0].Source != "mcp" || !list[0].Enabled || list[0].CreatedAt.IsZero() {
		t.Fatalf("reloaded: %+v", list)
	}
}

func TestFiles_SeenAndBreakpoints(t *testing.T) {
	a := newHereHarness(t, ModeAuto)
	_, _ = a.proxy.AddBreakpoint(SavedBreakpoint{File: "/app/src/Billing/Invoice.php", Line: 1, Enabled: true})
	be := &bpEngine{lines: map[string]int{"run": 12}}
	_, s := startBP(t, a.proxy, be)
	waitFor(t, "break", s.Paused)

	if code, body := a.do(t, "GET", "/api/xdebug/files?q=user", ""); code != http.StatusOK || body != "[\"file:///app/src/User.php\"]\n" {
		t.Fatalf("q=user: %d %s", code, body)
	}
	if _, body := a.do(t, "GET", "/api/xdebug/files?q=inv", ""); !strings.Contains(body, "Invoice.php") {
		t.Fatalf("q=inv: %s", body)
	}
	if _, body := a.do(t, "GET", "/api/xdebug/files?q=nothing", ""); body != "[]\n" {
		t.Fatalf("no match: %s", body)
	}
	f := &seenFiles{}
	for i := 0; i < maxSeenFiles+10; i++ {
		f.see("file:///f" + strconv.Itoa(i) + ".php")
	}
	if got := f.find("", 1000, nil); len(got) != maxSeenFiles || got[0] != "file:///f509.php" {
		t.Fatalf("cap: %d %v", len(got), got[:1])
	}
}

func TestMCPBreakpointTools(t *testing.T) {
	a := newHereHarness(t, ModeAuto)
	cs := connectToolsWith(t, a.proxy, false, true)
	out, isErr := callTool(t, cs, "xdebug_breakpoint_set", map[string]any{"file": "/app/src/User.php", "line": 20})
	if isErr || !strings.Contains(out, `"source": "mcp"`) || !strings.Contains(out, `"file": "file:///app/src/User.php"`) {
		t.Fatalf("set: %s", out)
	}
	if out, isErr := callTool(t, cs, "xdebug_breakpoint_set", map[string]any{"file": "/app/a.php", "line": 1, "condition": "$a"}); !isErr || !strings.Contains(out, "allow_eval") {
		t.Fatalf("condition: %s", out)
	}
	be := &bpEngine{lines: map[string]int{"run": 12}}
	_, s := startBP(t, a.proxy, be)
	waitFor(t, "break", s.Paused)
	if out, isErr := callTool(t, cs, "xdebug_breakpoints", map[string]any{"session_id": s.ID}); isErr || !strings.Contains(out, `"state": "unresolved"`) {
		t.Fatalf("list: %s", out)
	}
	if out, isErr := callTool(t, cs, "xdebug_breakpoint_remove", map[string]any{"id": 1}); isErr || !strings.Contains(out, `"removed": 1`) {
		t.Fatalf("remove: %s", out)
	}
	waitFor(t, "breakpoint_remove", func() bool { return len(be.commands("breakpoint_remove")) == 1 })
	if out, _ := callTool(t, cs, "xdebug_breakpoints", map[string]any{}); strings.TrimSpace(out) != "[]" {
		t.Fatalf("after remove: %s", out)
	}
}

// pausedWithBPs starts a session driven here, paused at line 12, with the
// given breakpoints already set in the engine.
func pausedWithBPs(t *testing.T, a *apiHarness, files ...string) (*fakeEngine, *Session, *bpEngine, []SavedBreakpoint) {
	t.Helper()
	var saved []SavedBreakpoint
	for i, f := range files {
		b, err := a.proxy.AddBreakpoint(SavedBreakpoint{File: f, Line: 10 + i, Enabled: true})
		if err != nil {
			t.Fatal(err)
		}
		saved = append(saved, b)
	}
	be := &bpEngine{lines: map[string]int{"run": 12}}
	e, s := startBP(t, a.proxy, be)
	waitFor(t, "break", s.Paused)
	if n := len(be.commands("breakpoint_set")); n != len(files) {
		t.Fatalf("set at start: %d", n)
	}
	return e, s, be, saved
}

// Review #401 (1): a remove the engine did not take (the script moved on)
// must not lose the engine id: the next pause removes it.
func TestBreakpoints_DisableAndDeleteWhileRunningRemovedAtNextPause(t *testing.T) {
	a := newHereHarness(t, ModeAuto)
	e, s, be, saved := pausedWithBPs(t, a, "/app/a.php", "/app/b.php")

	be.mu.Lock()
	be.lines = map[string]int{} // the step is not answered: the script runs
	be.mu.Unlock()
	if err := s.Continue("step_over", ByAgent, "probe"); err != nil {
		t.Fatal(err)
	}
	waitFor(t, "step sent", func() bool { return len(be.commands("step_over")) == 1 })

	if _, err := a.proxy.bps.Update(saved[0].ID, func(b *SavedBreakpoint) error { b.Enabled = false; return nil }); err != nil {
		t.Fatal(err)
	}
	a.proxy.BreakpointsChanged()
	if err := a.proxy.RemoveBreakpoint(saved[1].ID); err != nil {
		t.Fatal(err)
	}
	if n := len(be.commands("breakpoint_remove")); n != 0 {
		t.Fatalf("remove sent while running: %d", n)
	}
	if bps := s.SessionBreakpoints(a.proxy.bps.List()); len(bps) != 1 || bps[0].State != BPPending {
		t.Fatalf("while running: %+v", bps)
	}

	step := be.commands("step_over")[0]
	e.send(response("step_over", tidOf(t, step), ` status="break" reason="ok"`, `<xdebug:message filename="file:///app/src/User.php" lineno="13"></xdebug:message>`))
	waitFor(t, "both removed at the pause", func() bool { return len(be.commands("breakpoint_remove")) == 2 })
	rm := strings.Join(be.commands("breakpoint_remove"), "|")
	if !strings.Contains(rm, "-d 9001") || !strings.Contains(rm, "-d 9002") {
		t.Fatalf("removes: %s", rm)
	}
	waitFor(t, "disabled", func() bool {
		bps := s.SessionBreakpoints(a.proxy.bps.List())
		return len(bps) == 1 && bps[0].State == BPDisabled
	})
	s.mu.Lock()
	left := len(s.bps)
	s.mu.Unlock()
	if left != 1 {
		t.Fatalf("session still tracks %d breakpoints", left)
	}
}

// Review #401 (2): a sync that waited for another applies the current
// list, never an older snapshot.
func TestBreakpoints_WaitingSyncNeverAppliesOldList(t *testing.T) {
	a := newHereHarness(t, ModeAuto)
	_, s, be, _ := pausedWithBPs(t, a)

	s.bpSyncMu.Lock() // a sync in progress
	added := make(chan SavedBreakpoint, 1)
	go func() {
		b, _ := a.proxy.AddBreakpoint(SavedBreakpoint{File: "/app/c.php", Line: 3, Enabled: true})
		added <- b
	}()
	waitFor(t, "saved", func() bool { return len(a.proxy.bps.List()) == 1 })
	removed := make(chan error, 1)
	go func() { removed <- a.proxy.RemoveBreakpoint(a.proxy.bps.List()[0].ID) }()
	waitFor(t, "deleted", func() bool { return len(a.proxy.bps.List()) == 0 })
	s.bpSyncMu.Unlock()
	<-added
	if err := <-removed; err != nil {
		t.Fatal(err)
	}
	if sets, rms := len(be.commands("breakpoint_set")), len(be.commands("breakpoint_remove")); sets != rms {
		t.Fatalf("engine keeps a deleted breakpoint: %d set, %d removed", sets, rms)
	}
}

// Review #401 (3): a breakpoint added to a paused session is in the engine
// before the caller can continue.
func TestBreakpoints_AddThenContinueSendsSetFirst(t *testing.T) {
	a := newHereHarness(t, ModeAuto)
	_, s, be, _ := pausedWithBPs(t, a)
	if _, err := a.proxy.AddBreakpoint(SavedBreakpoint{File: "/app/d.php", Line: 4, Enabled: true}); err != nil {
		t.Fatal(err)
	}
	if err := s.Continue("run", ByAgent, "probe"); err != nil {
		t.Fatal(err)
	}
	waitFor(t, "second run", func() bool { return len(be.commands("run")) == 2 })
	be.mu.Lock()
	order := strings.Join(be.seen, "|")
	be.mu.Unlock()
	if i, j := strings.Index(order, "breakpoint_set"), strings.LastIndex(order, "run -i"); i < 0 || i > j {
		t.Fatalf("order: %s", order)
	}
}

func TestBreakpoints_FilePaths(t *testing.T) {
	for in, want := range map[string]string{
		"/var/www/x.php":    "file:///var/www/x.php",
		`C:\app\src\x.php`:  "file:///C:/app/src/x.php",
		"c:/app/x.php":      "file:///c:/app/x.php",
		"file:///srv/x.php": "file:///srv/x.php",
	} {
		b := SavedBreakpoint{File: in, Line: 1}
		if err := b.normalize(); err != nil || b.File != want {
			t.Errorf("%s: %q %v", in, b.File, err)
		}
	}
	for _, bad := range []string{"src/x.php", "x.php", "./x.php"} {
		b := SavedBreakpoint{File: bad, Line: 1}
		if err := b.normalize(); err == nil {
			t.Errorf("%s accepted", bad)
		}
	}
}
