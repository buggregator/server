package xdebug

import (
	"bufio"
	"encoding/base64"
	"errors"
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
		if err := s.sendOwn("feature_set -n "+f, false, nil); err != nil {
			return <-engineDone
		}
	}
	if s.bpSource != nil {
		list, eval := s.bpSource()
		s.syncBreakpoints(list, eval)
	}
	first := "run"
	if pauseOnFirstLine {
		first = "step_into"
	}
	if err := s.sendOwn(first, true, nil); err != nil {
		return <-engineDone
	}
	return <-engineDone
}

// canSendLocked: the engine reads commands now (at the start or paused, no
// continuation in flight). Caller holds mu.
func (s *Session) canSendLocked() bool {
	return !s.closed && s.conts == 0 && (s.status == "break" || s.status == "starting")
}

// ownCallReply sends a command of Buggregator's own (breakpoints) and waits
// for its reply. Refused unless the engine reads commands now.
func (s *Session) ownCallReply(cmd string) ([]byte, error) {
	reply := make(chan []byte, 1)
	s.writeMu.Lock()
	s.mu.Lock()
	if !s.canSendLocked() {
		s.mu.Unlock()
		s.writeMu.Unlock()
		return nil, errNotPaused
	}
	tid := s.allocTID()
	s.pending[tid] = pending{reply: reply}
	s.mu.Unlock()
	name, rest, _ := strings.Cut(cmd, " ")
	line := name + " -i " + strconv.Itoa(tid)
	if rest != "" {
		line += " " + rest
	}
	err := s.writeEngineLocked(line)
	s.writeMu.Unlock()
	if err != nil {
		s.forget(tid)
		return nil, err
	}
	t := time.NewTimer(s.timeout)
	defer t.Stop()
	select {
	case body, ok := <-reply:
		if !ok {
			return nil, errSessionClosed
		}
		if _, err := parseResponse(body); err != nil {
			return nil, err
		}
		return body, nil
	case <-t.C:
		s.forget(tid)
		return nil, errCommandTimeout
	}
}

func (s *Session) ownCall(cmd string) error {
	_, err := s.ownCallReply(cmd)
	return err
}

func b64Arg(v string) string { return base64.StdEncoding.EncodeToString([]byte(v)) }

// errSkip from a sendOwn precondition: nothing to send, not an error.
var errSkip = errors.New("skip")

// sendOwn writes a command of Buggregator's own; its reply is dropped (a
// continuation's reply still sets the status). check runs under mu in the
// same critical section that counts the continuation and allocates its tid,
// so two callers can never both pass a "paused" check; it may also set the
// session's flags. A failed write clears detaching and idleReason again.
func (s *Session) sendOwn(cmd string, cont bool, check func() error) error {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return errSessionClosed
	}
	if check != nil {
		if err := check(); err != nil {
			s.mu.Unlock()
			if errors.Is(err, errSkip) {
				return nil
			}
			return err
		}
	}
	tid := s.allocTID()
	s.pending[tid] = pending{own: true, cont: cont}
	var ev map[string]any
	if cont {
		s.conts++
		s.stopIdleLocked()
		// From a pause (or the start) the script runs; the stop that ends a
		// script in "stopping" leaves the status alone (no flicker).
		if s.status == "break" || s.status == "starting" {
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
	err := s.writeEngineLocked(line)
	if err != nil {
		s.mu.Lock()
		s.detaching, s.idleReason = false, false
		s.mu.Unlock()
	}
	return err
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
	return s.sendOwn(cmd, true, func() error {
		switch {
		case s.driver != DriverHere:
			return errIDEAttached
		case s.status != "break" || s.conts > 0:
			return errNotPaused
		}
		s.lastContBy, s.lastContName = by, ""
		if by == ByAgent {
			s.lastContName = agent
		}
		s.detaching = cmd == "detach"
		return nil
	})
}

// afterReply runs after a reply moved the status (forwardEngine). A script
// driven here that reached its end waits in "stopping" for the IDE's word:
// say stop, so the PHP process can finish.
func (s *Session) afterReply(prev string) {
	s.mu.Lock()
	end := s.driver == DriverHere && s.status == "stopping" && prev != "stopping"
	s.mu.Unlock()
	if !end {
		return // cheap path: no writeMu on every reply of a relayed session
	}
	_ = s.sendOwn("stop", true, func() error {
		if s.driver != DriverHere || s.status != "stopping" || s.conts > 0 {
			return errSkip
		}
		return nil
	})
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
	released := false
	_ = s.sendOwn("detach", true, func() error {
		if s.idleDeadline.IsZero() || s.status != "break" || s.conts > 0 {
			return errSkip
		}
		if left := time.Until(s.idleDeadline); left > 0 {
			s.idleTimer.Reset(left) // touched since the timer was set
			return errSkip
		}
		s.idleReason, s.detaching, released = true, true, true
		return nil
	})
	if released {
		slog.Info("xdebug: paused session idle, released it", "session", s.ID, "idle_timeout", s.idleTimeout)
	}
}
