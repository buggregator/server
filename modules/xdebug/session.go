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
}

// Info is a snapshot of a session for listing.
type Info struct {
	ID        string    `json:"id"`
	IDEKey    string    `json:"idekey,omitempty"`
	FileURI   string    `json:"fileuri,omitempty"`
	Language  string    `json:"language,omitempty"`
	Engine    string    `json:"engine,omitempty"`
	Status    string    `json:"status"`
	File      string    `json:"file,omitempty"`
	Line      int       `json:"line,omitempty"`
	StartedAt time.Time `json:"started_at"`
}

func (s *Session) Info() Info {
	s.mu.Lock()
	defer s.mu.Unlock()
	return Info{
		ID: s.ID, IDEKey: s.IDEKey, FileURI: s.FileURI, Language: s.Language, Engine: s.Engine,
		Status: s.status, File: s.file, Line: s.line, StartedAt: s.StartedAt,
	}
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
		pending: map[int]pending{}, status: "starting", timeout: timeout,
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
		if cont {
			s.conts++
			s.status = "running"
		}
		s.mu.Unlock()
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
		if rt.Status != "" && s.conts == 0 {
			s.status = rt.Status
			if rt.Status == "break" {
				s.file, s.line = breakLocation(body)
			}
		}
		s.mu.Unlock()

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

// close fails pending agent calls; it does not close the connections.
func (s *Session) close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return
	}
	s.closed = true
	s.status = "stopped"
	for tid, p := range s.pending {
		if p.reply != nil {
			close(p.reply)
		}
		delete(s.pending, tid)
	}
}

// breakLocation reads <xdebug:message filename lineno> from a break response.
func breakLocation(body []byte) (string, int) {
	var r struct {
		Message struct {
			Filename string `xml:"filename,attr"`
			Lineno   int    `xml:"lineno,attr"`
		} `xml:"message"`
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
