package xdebug

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"encoding/xml"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	errNotPaused      = errors.New("session is not paused (the script is running or has not stopped at a breakpoint yet)")
	errSessionClosed  = errors.New("session closed")
	errCommandTimeout = errors.New("timed out waiting for Xdebug")
	errNoSession      = errors.New("no such xdebug session")
	errSessionLive    = errors.New("session is live: only a closed session can be dismissed")
)

// continuations resume the script: after one is sent the engine is running.
var continuations = map[string]bool{
	"run": true, "step_into": true, "step_over": true, "step_out": true, "stop": true, "detach": true,
}

// agentCommands is the read-only allow-list for the agent. eval is gated separately.
var agentCommands = map[string]bool{
	"status": true, "stack_depth": true, "stack_get": true, "context_names": true, "context_get": true,
	"property_get": true, "property_value": true, "source": true, "breakpoint_list": true, "typemap_get": true,
	"eval": true,
}

type pending struct {
	ideTID string      // origin IDE: the IDE's own transaction id
	reply  chan []byte // origin agent
	cont   bool        // a continuation: its reply carries the engine's new status
}

// Session is one Xdebug connection proxied to the IDE.
type Session struct {
	ID        string
	IDEKey    string
	FileURI   string
	Language  string
	Engine    string
	StartedAt time.Time

	engine net.Conn
	ide    net.Conn

	writeMu sync.Mutex // serialises writes to the engine; taken before mu

	mu       sync.Mutex
	nextTID  int
	pending  map[int]pending
	status   string
	conts    int // continuations sent and not answered yet
	file     string
	line     int
	closed   bool
	agentSem chan struct{} // one agent command at a time; others wait their turn
	timeout  time.Duration

	pauseSeq     int // bumped on every transition into break; keys the read cache
	pausedAt     time.Time
	cache        map[string][]byte // UI reads of the current pause
	cacheSeq     int
	ideConnected bool
	closeReason  string
	closedAt     time.Time
	kind         string // http or cli, read once at the first break
	target       string // "GET /checkout?…" or the CLI command line
	targetDone   bool
	agentCalls   int
	agentLog     []AgentCall // ring, newest last
	notify       func(event string, payload any)
}

// Close reasons of a session (tombstone).
const (
	ReasonIDEUnreachable = "ide_unreachable"
	ReasonIDEClosed      = "ide_closed"
	ReasonEngineClosed   = "engine_closed"
	ReasonStopped        = "stopped"
)

const agentLogSize = 200

// Info is a snapshot of a session for listing.
type Info struct {
	ID           string     `json:"id"`
	IDEKey       string     `json:"idekey"`
	State        string     `json:"state"` // starting, break, running, stopping, stopped, detached
	Kind         string     `json:"kind,omitempty"`
	Target       string     `json:"target,omitempty"`
	EntryFile    string     `json:"entry_file"`
	Language     string     `json:"language,omitempty"`
	Engine       string     `json:"engine,omitempty"`
	IDEConnected bool       `json:"ide_connected"`
	LastFile     string     `json:"last_file,omitempty"`
	LastLine     int        `json:"last_line,omitempty"`
	StartedAt    time.Time  `json:"started_at"`
	PausedAt     *time.Time `json:"paused_at,omitempty"`
	AgentCalls   int        `json:"agent_calls"`
	ClosedReason string     `json:"closed_reason,omitempty"`
	ClosedAt     *time.Time `json:"closed_at,omitempty"`
}

func (s *Session) Info() Info {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.infoLocked()
}

func (s *Session) infoLocked() Info {
	in := Info{
		ID: s.ID, IDEKey: s.IDEKey, State: s.stateLocked(), Kind: s.kind, Target: s.target,
		EntryFile: s.FileURI, Language: s.Language, Engine: s.Engine, IDEConnected: s.ideConnected,
		LastFile: s.file, LastLine: s.line, StartedAt: s.StartedAt, AgentCalls: s.agentCalls,
		ClosedReason: s.closeReason,
	}
	if !s.pausedAt.IsZero() {
		t := s.pausedAt
		in.PausedAt = &t
	}
	if s.closed {
		t := s.closedAt
		in.ClosedAt = &t
	}
	return in
}

// stateLocked is the state shown to clients. Caller holds mu.
func (s *Session) stateLocked() string {
	if s.closed {
		if s.closeReason == ReasonIDEUnreachable || s.closeReason == ReasonIDEClosed {
			return "detached"
		}
		return "stopped"
	}
	return s.status
}

// statusEvent is the payload of session.status. Caller holds mu.
func (s *Session) statusEventLocked() map[string]any {
	return map[string]any{"id": s.ID, "state": s.stateLocked(), "file": s.file, "line": s.line, "kind": s.kind, "target": s.target, "pause_seq": s.pauseSeq}
}

func (s *Session) emit(event string, payload any) {
	if s.notify != nil {
		s.notify(event, payload)
	}
}

// Closed reports whether the session is a tombstone.
func (s *Session) Closed() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.closed
}

// Paused reports whether the engine is stopped at a breakpoint.
func (s *Session) Paused() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.status == "break" && !s.closed
}

type initPacket struct {
	IDEKey   string `xml:"idekey,attr"`
	FileURI  string `xml:"fileuri,attr"`
	Language string `xml:"language,attr"`
	Engine   struct {
		Version string `xml:"version,attr"`
		Name    string `xml:",chardata"`
	} `xml:"engine"`
}

func newSession(id string, engine net.Conn, initBody []byte, timeout time.Duration) *Session {
	s := &Session{
		ID: id, StartedAt: time.Now(), engine: engine,
		pending: map[int]pending{}, status: "starting", timeout: timeout, cache: map[string][]byte{},
		agentSem: make(chan struct{}, 1),
	}
	var ip initPacket
	if err := unmarshalXML(initBody, &ip); err == nil {
		s.IDEKey, s.FileURI, s.Language = ip.IDEKey, ip.FileURI, ip.Language
		if ip.Engine.Name != "" {
			s.Engine = ip.Engine.Name + " " + ip.Engine.Version
		}
	}
	return s
}

// forwardIDE runs the IDE → engine direction until the IDE closes.
func (s *Session) forwardIDE(r *bufio.Reader) error {
	for {
		line, err := readCommand(r)
		if err != nil {
			return err
		}
		name, start, end, ok := commandTID(line)
		if !ok {
			// No transaction id: forward as is; the engine answers with an error the IDE can match.
			if err := s.writeEngine(line); err != nil {
				return err
			}
			continue
		}
		s.writeMu.Lock()
		s.mu.Lock()
		if s.closed {
			s.mu.Unlock()
			s.writeMu.Unlock()
			return errSessionClosed
		}
		tid := s.allocTID()
		cont := continuations[name]
		s.pending[tid] = pending{ideTID: line[start:end], cont: cont}
		var ev map[string]any
		if cont {
			s.conts++
			if s.status != "running" {
				s.status = "running"
				ev = s.statusEventLocked()
			}
		}
		s.mu.Unlock()
		if ev != nil {
			s.emit("session.status", ev)
		}
		err = s.writeEngineLocked(line[:start] + strconv.Itoa(tid) + line[end:])
		s.writeMu.Unlock()
		if err != nil {
			return err
		}
	}
}

// forwardEngine runs the engine → IDE direction until the engine closes.
func (s *Session) forwardEngine(r *bufio.Reader) error {
	for {
		body, err := readPacket(r)
		if err != nil {
			return err
		}
		rt, err := parseRoot(body)
		if err != nil {
			slog.Warn("xdebug: unparsable packet from engine, forwarding", "session", s.ID, "err", err)
			if err := s.writeIDE(body); err != nil {
				return err
			}
			continue
		}
		if rt.Name != "response" || rt.TransactionID == "" {
			// init, stream, notify: the IDE's.
			if err := s.writeIDE(body); err != nil {
				return err
			}
			continue
		}
		tid, _ := strconv.Atoi(rt.TransactionID)
		s.mu.Lock()
		p, found := s.pending[tid]
		delete(s.pending, tid)
		if found && p.cont {
			s.conts--
		}
		// While a continuation is in flight the engine is running: a late reply
		// to an earlier command (e.g. status) or to an earlier continuation must
		// not report it paused again. conts is already decremented for this
		// reply, so the last continuation's reply sets the status.
		var ev map[string]any
		resolve := false
		if rt.Status != "" && s.conts == 0 {
			prev, prevFile, prevLine := s.status, s.file, s.line
			s.status = rt.Status
			if rt.Status == "break" {
				// Xdebug also puts status="break" on plain replies (even errors);
				// only a reply that names a location moves the current line.
				if f, l := breakLocation(body); f != "" {
					s.file, s.line = f, l
				}
				if prev != "break" {
					s.pauseSeq++
					s.pausedAt = time.Now()
					resolve = !s.targetDone
				}
			}
			// A read or eval reply during a pause repeats status="break": no
			// event for it, or a client that re-reads on break loops.
			if prev != s.status || s.file != prevFile || s.line != prevLine {
				ev = s.statusEventLocked()
			}
		}
		s.mu.Unlock()
		if ev != nil {
			s.emit("session.status", ev)
		}
		if resolve {
			go s.resolveTarget()
		}

		switch {
		case !found:
			slog.Debug("xdebug: response for unknown transaction dropped", "session", s.ID, "tid", rt.TransactionID)
		case p.reply != nil:
			p.reply <- body // buffered; never reaches the IDE
		default:
			if err := s.writeIDE(rt.withTID(body, p.ideTID)); err != nil {
				return err
			}
		}
	}
}

// Command sends one read-only command for the agent and waits for its reply.
// args is the part after "-i <tid>"; data is sent base64-encoded after "--".
func (s *Session) Command(name, args string, data []byte) ([]byte, error) {
	if !agentCommands[name] {
		return nil, fmt.Errorf("command %q is not allowed", name)
	}
	// NUL ends a DBGp command: inside args it would smuggle a second command
	// (e.g. "run") past the allow-list. data is base64 and cannot carry one.
	if strings.ContainsRune(args, 0) {
		return nil, errors.New("arguments must not contain NUL")
	}
	t := time.NewTimer(s.timeout)
	defer t.Stop()
	select {
	case s.agentSem <- struct{}{}:
		defer func() { <-s.agentSem }()
	case <-t.C:
		return nil, errCommandTimeout
	}

	reply := make(chan []byte, 1)
	s.writeMu.Lock()
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		s.writeMu.Unlock()
		return nil, errSessionClosed
	}
	if s.status != "break" {
		s.mu.Unlock()
		s.writeMu.Unlock()
		return nil, errNotPaused
	}
	tid := s.allocTID()
	s.pending[tid] = pending{reply: reply}
	s.mu.Unlock()

	line := name + " -i " + strconv.Itoa(tid)
	if args != "" {
		line += " " + args
	}
	if data != nil {
		line += " -- " + base64.StdEncoding.EncodeToString(data)
	}
	err := s.writeEngineLocked(line)
	s.writeMu.Unlock()
	if err != nil {
		s.forget(tid)
		return nil, err
	}

	t.Reset(s.timeout)
	select {
	case body, ok := <-reply:
		if !ok {
			return nil, errSessionClosed
		}
		return body, nil
	case <-t.C:
		s.forget(tid)
		return nil, errCommandTimeout
	}
}

func (s *Session) forget(tid int) {
	s.mu.Lock()
	delete(s.pending, tid)
	s.mu.Unlock()
}

// allocTID returns the next proxy transaction id. Caller holds mu.
func (s *Session) allocTID() int {
	s.nextTID++
	return s.nextTID
}

func (s *Session) writeEngine(line string) error {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	return s.writeEngineLocked(line)
}

func (s *Session) writeEngineLocked(line string) error {
	_, err := s.engine.Write(append([]byte(line), 0))
	return err
}

// writeIDE is called only from forwardEngine, the IDE connection's single writer.
func (s *Session) writeIDE(body []byte) error {
	_, err := s.ide.Write(encodePacket(body))
	return err
}

// detach lets the script run on without a debugger.
func (s *Session) detach() {
	s.writeMu.Lock()
	s.mu.Lock()
	tid := s.allocTID()
	s.status = "running"
	s.mu.Unlock()
	_ = s.engine.SetWriteDeadline(time.Now().Add(time.Second))
	_ = s.writeEngineLocked("detach -i " + strconv.Itoa(tid))
	s.writeMu.Unlock()
}

// close fails pending agent calls and turns the session into a tombstone
// with a reason; it does not close the connections.
func (s *Session) close(reason string) {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return
	}
	if reason == ReasonEngineClosed && (s.status == "stopping" || s.status == "stopped") {
		reason = ReasonStopped
	}
	s.closed = true
	s.closeReason = reason
	s.closedAt = time.Now()
	s.ideConnected = false
	s.cache = map[string][]byte{}
	for tid, p := range s.pending {
		if p.reply != nil {
			close(p.reply)
		}
		delete(s.pending, tid)
	}
	ev := map[string]any{"id": s.ID, "reason": reason, "state": s.stateLocked()}
	s.mu.Unlock()
	s.emit("session.closed", ev)
}

// breakLocation reads <xdebug:message filename lineno> from a break response.
func breakLocation(body []byte) (string, int) {
	var r struct {
		Message struct {
			Filename string `xml:"filename,attr"`
			Lineno   int    `xml:"lineno,attr"`
		} `xml:"https://xdebug.org/dbgp/xdebug message"` // not <error><message>
	}
	if err := unmarshalXML(body, &r); err != nil {
		return "", 0
	}
	return r.Message.Filename, r.Message.Lineno
}

func unmarshalXML(body []byte, v any) error {
	dec := xml.NewDecoder(bytes.NewReader(body))
	dec.CharsetReader = passthroughCharset
	return dec.Decode(v)
}
