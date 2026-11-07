package xdebug

import (
	"bufio"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"
)

// Standalone driver: with no IDE (mode standalone, or auto when the IDE is
// not reachable) Buggregator drives the session itself. It sets the features
// an IDE would set, then runs the script (or steps into the first line), and
// lets the UI or an agent step, continue, stop or release it.

// Modes of the proxy (xdebug.mode).
const (
	ModeProxy      = "proxy"      // relay to the IDE; no IDE → detach, the script runs on
	ModeAuto       = "auto"       // relay when the IDE answers, else drive here
	ModeStandalone = "standalone" // never dial the IDE; drive here
)

// ValidMode reports whether m is a known xdebug.mode.
func ValidMode(m string) bool {
	return m == ModeProxy || m == ModeAuto || m == ModeStandalone
}

// Who sent a continuation.
const (
	ByUI    = "ui"
	ByAgent = "agent"
)

// controlCommands are the continuations the UI and agents may send to a
// session driven here.
var controlCommands = map[string]bool{
	"run": true, "step_into": true, "step_over": true, "step_out": true, "stop": true, "detach": true,
}

// standaloneFeatures are what an IDE sets at init; Xdebug's defaults are too
// small for the UI (max_children 32, max_data 1024).
var standaloneFeatures = []string{
	"max_depth -v 1",
	"max_children -v 100",
	"max_data -v 65536",
	"resolved_breakpoints -v 1",
	"notify_ok -v 1",
}

// driveHere runs a session with no IDE until the engine closes.
func (s *Session) driveHere(er *bufio.Reader, pauseOnFirstLine bool) error {
	engineDone := make(chan error, 1)
	go func() { engineDone <- s.forwardEngine(er) }()

	for _, f := range standaloneFeatures {
		if err := s.sendOwn("feature_set -n "+f, false); err != nil {
			return <-engineDone
		}
	}
	first := "run"
	if pauseOnFirstLine {
		first = "step_into"
	}
	if err := s.sendOwn(first, true); err != nil {
		return <-engineDone
	}
	return <-engineDone
}

// sendOwn writes a command of Buggregator's own; its reply is dropped (a
// continuation's reply still sets the status).
func (s *Session) sendOwn(cmd string, cont bool) error {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return errSessionClosed
	}
	tid := s.allocTID()
	s.pending[tid] = pending{own: true, cont: cont}
	var ev map[string]any
	if cont {
		s.conts++
		s.stopIdleLocked()
		if s.status != "running" {
			s.status = "running"
			ev = s.statusEventLocked()
		}
	}
	s.mu.Unlock()
	if ev != nil {
		s.emit("session.status", ev)
	}
	name, rest, _ := strings.Cut(cmd, " ")
	line := name + " -i " + strconv.Itoa(tid)
	if rest != "" {
		line += " " + rest
	}
	return s.writeEngineLocked(line)
}

// Continue sends a continuation (run, step_into, step_over, step_out, stop,
// detach) from the UI or an agent. It does not wait: DBGp answers a
// continuation only at the next break or at the end of the script, and the
// reply moves the session's status. by is ByUI or ByAgent; agent is the
// agent's name, if known.
func (s *Session) Continue(cmd, by, agent string) error {
	if !controlCommands[cmd] {
		return fmt.Errorf("command %q is not a continuation", cmd)
	}
	s.mu.Lock()
	switch {
	case s.closed:
		s.mu.Unlock()
		return errSessionClosed
	case s.driver != DriverHere:
		s.mu.Unlock()
		return errIDEAttached
	case s.status != "break" || s.conts > 0:
		s.mu.Unlock()
		return errNotPaused
	}
	s.lastContBy, s.lastContName = by, ""
	if by == ByAgent {
		s.lastContName = agent
	}
	if cmd == "detach" {
		s.detaching = true
	}
	s.mu.Unlock()
	return s.sendOwn(cmd, true)
}

// afterReply runs after a reply moved the status (forwardEngine). A script
// driven here that reached its end waits in "stopping" for the IDE's word:
// say stop, so the PHP process can finish.
func (s *Session) afterReply(prev string) {
	s.mu.Lock()
	end := s.driver == DriverHere && s.status == "stopping" && prev != "stopping" && s.conts == 0 && !s.closed
	s.mu.Unlock()
	if end {
		_ = s.sendOwn("stop", true)
	}
}

// lastContByLocked is last_continuation_by for the UI: nil until a
// continuation is sent here. Caller holds mu.
func (s *Session) lastContByLocked() *string {
	if s.lastContBy == "" {
		return nil
	}
	v := s.lastContBy
	return &v
}

// idleDeadlineLocked is the idle_deadline for the UI. Caller holds mu.
func (s *Session) idleDeadlineLocked() *time.Time {
	if s.idleDeadline.IsZero() || s.closed {
		return nil
	}
	t := s.idleDeadline
	return &t
}

// armIdleLocked (re)starts the idle guard of a session paused here. Caller
// holds mu.
func (s *Session) armIdleLocked() {
	if s.driver != DriverHere || s.idleTimeout <= 0 || s.closed {
		return
	}
	s.idleDeadline = time.Now().Add(s.idleTimeout)
	if s.idleTimer == nil {
		s.idleTimer = time.AfterFunc(s.idleTimeout, s.idleFire)
	} else {
		s.idleTimer.Reset(s.idleTimeout)
	}
}

// stopIdleLocked stops the idle guard: the script runs, or the session ended.
// Caller holds mu.
func (s *Session) stopIdleLocked() {
	s.idleDeadline = time.Time{}
	s.idleSent = time.Time{}
	if s.idleTimer != nil {
		s.idleTimer.Stop()
	}
}

// touch restarts the idle guard on every command from the UI or an agent
// while the session is paused here, and tells the UI the new deadline (as
// session.idle: a session.status would make clients re-read, and loop).
func (s *Session) touch() {
	s.mu.Lock()
	if s.driver != DriverHere || s.status != "break" || s.conts > 0 || s.idleDeadline.IsZero() {
		s.mu.Unlock()
		return
	}
	s.armIdleLocked()
	var ev map[string]any
	if s.idleDeadline.Sub(s.idleSent) >= 5*time.Second {
		s.idleSent = s.idleDeadline
		ev = map[string]any{"id": s.ID, "idle_deadline": s.idleDeadline}
	}
	s.mu.Unlock()
	if ev != nil {
		s.emit("session.idle", ev)
	}
}

// idleFire releases a session left paused for idle_timeout: a paused script
// holds a PHP-FPM worker, the session lock and open transactions.
func (s *Session) idleFire() {
	s.mu.Lock()
	if s.closed || s.idleDeadline.IsZero() || s.status != "break" || s.conts > 0 {
		s.mu.Unlock()
		return
	}
	if left := time.Until(s.idleDeadline); left > 0 {
		s.idleTimer.Reset(left) // touched since the timer was set
		s.mu.Unlock()
		return
	}
	s.idleReason, s.detaching = true, true
	s.mu.Unlock()
	slog.Info("xdebug: paused session idle, releasing it", "session", s.ID, "idle_timeout", s.idleTimeout)
	_ = s.sendOwn("detach", true)
}
