package xdebug

import (
	"errors"
	"strings"
	"time"
)

const maxCachedReads = 256

// Read runs a read-only command for the UI. Replies are cached per pause, so
// several open tabs asking the same thing cost one DBGp round trip.
func (s *Session) Read(name, args string) ([]byte, error) {
	key := name + " " + args
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return nil, errSessionClosed
	}
	if s.status != "break" || s.conts > 0 {
		s.mu.Unlock()
		return nil, errNotPaused
	}
	seq := s.pauseSeq
	if s.cacheSeq != seq {
		s.cache = map[string][]byte{}
		s.cacheSeq = seq
	}
	if body, ok := s.cache[key]; ok {
		s.mu.Unlock()
		return body, nil
	}
	s.mu.Unlock()

	body, err := s.Command(name, args, nil)
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	if s.pauseSeq == seq && s.cacheSeq == seq && len(s.cache) < maxCachedReads {
		s.cache[key] = body
	}
	s.mu.Unlock()
	return body, nil
}

// readResponse is Read plus parsing and the DBGp error check.
func (s *Session) readResponse(name, args string) (*xResponse, error) {
	body, err := s.Read(name, args)
	if err != nil {
		return nil, err
	}
	return parseResponse(body)
}

// resolveTarget reads what the request is (HTTP method + URI, or the CLI
// command line) once per session, at the first break.
func (s *Session) resolveTarget() {
	get := func(name string) (string, []Var, error) {
		r, err := s.call("property_get", "-c 1 -m 2048 -n "+quoteArg(name), nil)
		if err != nil {
			return "", nil, err
		}
		if len(r.Props) == 0 {
			return "", nil, errors.New("empty")
		}
		v := toVar(r.Props[0])
		return derefOr(v.Value), v.Children, nil
	}
	retryLater := func(err error) bool {
		return errors.Is(err, errNotPaused) || errors.Is(err, errSessionClosed) || errors.Is(err, errCommandTimeout)
	}

	var kind, target string
	method, _, err := get("$_SERVER['REQUEST_METHOD']")
	switch {
	case err == nil && method != "":
		uri, _, err := get("$_SERVER['REQUEST_URI']")
		if retryLater(err) {
			return
		}
		kind, target = "http", strings.TrimSpace(method+" "+uri)
	case retryLater(err):
		return
	default:
		_, argv, err := get("$_SERVER['argv']")
		if retryLater(err) {
			return
		}
		parts := make([]string, 0, len(argv))
		for _, a := range argv {
			parts = append(parts, derefOr(a.Value))
		}
		kind, target = "cli", strings.Join(parts, " ")
	}

	s.mu.Lock()
	if s.targetDone {
		s.mu.Unlock()
		return
	}
	s.kind, s.target, s.targetDone = kind, target, true
	ev := s.statusEventLocked()
	s.mu.Unlock()
	s.emit("session.status", ev)
}

// AgentCall is one MCP tool call on a session, as shown in the UI's agent log.
type AgentCall struct {
	Seq        int       `json:"seq"`
	SessionID  string    `json:"session_id"`
	At         time.Time `json:"at"`
	Tool       string    `json:"tool"`
	Args       string    `json:"args,omitempty"`
	OK         bool      `json:"ok"`
	Result     string    `json:"result_summary"`
	DurationMS int64     `json:"duration_ms"`
}

// agentStarted announces a tool call before it runs ("reading…" in the UI).
func (s *Session) agentStarted(tool, args string) {
	s.emit("agent.call.started", map[string]any{
		"session_id": s.ID, "tool": tool, "args": args, "at": time.Now(),
	})
}

// agentDone records a finished tool call in the ring and broadcasts it.
func (s *Session) agentDone(tool, args string, started time.Time, ok bool, result string) {
	s.mu.Lock()
	s.agentCalls++
	c := AgentCall{
		Seq: s.agentCalls, SessionID: s.ID, At: started, Tool: tool, Args: args, OK: ok, Result: result,
		DurationMS: time.Since(started).Milliseconds(),
	}
	s.agentLog = append(s.agentLog, c)
	if len(s.agentLog) > agentLogSize {
		s.agentLog = append([]AgentCall(nil), s.agentLog[len(s.agentLog)-agentLogSize:]...)
	}
	s.mu.Unlock()
	s.emit("agent.call", c)
}

// AgentLog returns the session's agent calls, oldest first.
func (s *Session) AgentLog() []AgentCall {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]AgentCall{}, s.agentLog...)
}

// shortArgs keeps log args readable.
func shortArgs(s string) string {
	s = strings.Join(strings.Fields(s), " ")
	if len(s) > 120 {
		return s[:117] + "..."
	}
	return s
}
