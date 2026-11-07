package xdebug

import (
	"bufio"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"sort"
	"strconv"
	"sync"
	"time"
)

// Proxy accepts Xdebug connections and relays each one to the IDE, or drives
// it here when there is no IDE (see Settings.Mode).
type Proxy struct {
	ideAddr        string
	idleTimeout    time.Duration // a session paused here is released after this long without a command
	dialTimeout    time.Duration
	initTimeout    time.Duration
	commandTimeout time.Duration

	tombstoneTTL  time.Duration
	maxTombstones int  // newest tombstones kept; older ones are dropped
	readTarget    bool // read $_SERVER at the first break for the session title

	mu       sync.Mutex
	settings Settings
	nextID   int
	sessions map[string]*Session // live sessions and tombstones
	tombs    []string            // tombstone ids, oldest first

	// notify receives session.* and agent.* events for the UI; must not block.
	notify func(event string, payload any)
}

func NewProxy(ideAddr string) *Proxy {
	return &Proxy{
		ideAddr:        ideAddr,
		dialTimeout:    2 * time.Second,
		initTimeout:    10 * time.Second,
		commandTimeout: 10 * time.Second,
		tombstoneTTL:   10 * time.Minute,
		maxTombstones:  50,
		readTarget:     true,
		sessions:       map[string]*Session{},
		settings:       Settings{Mode: ModeProxy},
		idleTimeout:    10 * time.Minute,
	}
}

// Settings change at runtime (PUT /api/xdebug/settings); they apply to new
// sessions only.
type Settings struct {
	Mode             string `json:"mode"`                // proxy, auto or standalone
	PauseOnFirstLine bool   `json:"pause_on_first_line"` // a session driven here stops at the first line
}

// Settings returns the current settings.
func (p *Proxy) Settings() Settings {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.settings
}

// SetSettings replaces the settings; the mode must be valid.
func (p *Proxy) SetSettings(st Settings) error {
	if !ValidMode(st.Mode) {
		return fmt.Errorf("mode must be proxy, auto or standalone, got %q", st.Mode)
	}
	p.mu.Lock()
	p.settings = st
	p.mu.Unlock()
	return nil
}

// SetNotify sets the event sink (WebSocket broadcast). Call before serving.
func (p *Proxy) SetNotify(fn func(event string, payload any)) {
	p.notify = fn
}

func (p *Proxy) emit(event string, payload any) {
	if p.notify != nil {
		p.notify(event, payload)
	}
}

// register adds a session and announces it.
func (p *Proxy) register(s *Session) {
	s.notify = p.notify
	p.mu.Lock()
	p.sessions[s.ID] = s
	p.mu.Unlock()
	p.emit("session.created", s.Info())
}

// bury keeps a closed session as a tombstone until the TTL ends or it is
// dismissed. Only the newest maxTombstones are kept: with
// start_with_request=yes and no IDE every request leaves one.
func (p *Proxy) bury(s *Session) {
	p.mu.Lock()
	p.tombs = append(p.tombs, s.ID)
	for len(p.tombs) > p.maxTombstones {
		p.forgetLocked(p.tombs[0])
	}
	p.mu.Unlock()
	time.AfterFunc(p.tombstoneTTL, func() {
		p.mu.Lock()
		if p.sessions[s.ID] == s {
			p.forgetLocked(s.ID)
		}
		p.mu.Unlock()
	})
}

// forgetLocked drops a tombstone. Caller holds p.mu.
func (p *Proxy) forgetLocked(id string) {
	delete(p.sessions, id)
	for i, t := range p.tombs {
		if t == id {
			p.tombs = append(p.tombs[:i], p.tombs[i+1:]...)
			break
		}
	}
}

// Dismiss removes a tombstone. A live session is never touched.
func (p *Proxy) Dismiss(id string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	s, ok := p.sessions[id]
	if !ok {
		return errNoSession
	}
	if !s.Closed() {
		return errSessionLive
	}
	p.forgetLocked(id)
	return nil
}

// HandleConnection implements tcp.ConnectionHandler for one Xdebug connection.
func (p *Proxy) HandleConnection(engine net.Conn) {
	defer engine.Close()
	er := bufio.NewReader(engine)

	_ = engine.SetReadDeadline(time.Now().Add(p.initTimeout))
	initBody, err := readPacket(er)
	if err != nil {
		slog.Warn("xdebug: no init packet", "remote", engine.RemoteAddr(), "err", err)
		return
	}
	_ = engine.SetReadDeadline(time.Time{})

	p.mu.Lock()
	p.nextID++
	id := strconv.Itoa(p.nextID)
	p.mu.Unlock()
	s := newSession(id, engine, initBody, p.commandTimeout)
	s.targetDone = !p.readTarget
	st := p.Settings()

	var ide net.Conn
	if st.Mode != ModeStandalone {
		ide, err = net.DialTimeout("tcp", p.ideAddr, p.dialTimeout)
		if err == nil {
			s.ide = ide
			if err = s.writeIDE(initBody); err != nil {
				ide.Close()
				s.ide = nil
			}
		}
		if err != nil && st.Mode == ModeProxy {
			slog.Warn("xdebug: IDE is not reachable, detaching so the script runs on", "ide", p.ideAddr, "file", s.FileURI, "err", err)
			s.detach()
			p.register(s)
			s.close(ReasonIDEUnreachable)
			p.bury(s)
			return
		}
	}
	if s.ide == nil {
		s.driver = DriverHere
		s.idleTimeout = p.idleTimeout
		p.register(s)
		slog.Info("xdebug: session started, driven here (no IDE)", "session", id, "file", s.FileURI, "mode", st.Mode)
		err := s.driveHere(er, st.PauseOnFirstLine)
		s.close(ReasonEngineClosed)
		slog.Info("xdebug: session ended", "session", id, "err", quietEOF(err))
		p.bury(s)
		return
	}
	defer ide.Close()
	s.driver = DriverIDE

	s.mu.Lock()
	s.ideConnected = true
	s.mu.Unlock()
	p.register(s)
	slog.Info("xdebug: session started", "session", id, "file", s.FileURI, "idekey", s.IDEKey)

	ideDone := make(chan error, 1)
	engineDone := make(chan error, 1)
	go func() { ideDone <- s.forwardIDE(bufio.NewReader(ide)) }()
	go func() { engineDone <- s.forwardEngine(er) }()

	select {
	case err := <-engineDone:
		s.close(ReasonEngineClosed)
		ide.Close()
		<-ideDone
		slog.Info("xdebug: session ended", "session", id, "err", quietEOF(err))
	case err := <-ideDone:
		// The IDE went away first: let the script finish without a debugger,
		// then give the engine a moment to close its side.
		s.close(ReasonIDEClosed)
		slog.Info("xdebug: IDE disconnected, detaching", "session", id, "err", quietEOF(err))
		s.detach()
		_ = engine.SetReadDeadline(time.Now().Add(2 * time.Second))
		<-engineDone
	}

	p.bury(s)
}

// Sessions lists live sessions, oldest first.
func (p *Proxy) Sessions() []*Session {
	p.mu.Lock()
	out := make([]*Session, 0, len(p.sessions))
	for _, s := range p.sessions {
		out = append(out, s)
	}
	p.mu.Unlock()
	sort.Slice(out, func(i, j int) bool {
		a, _ := strconv.Atoi(out[i].ID)
		b, _ := strconv.Atoi(out[j].ID)
		return a < b
	})
	return out
}

// Session finds a session by id; with an empty id, the only paused session.
func (p *Proxy) Session(id string) (*Session, error) {
	if id != "" {
		p.mu.Lock()
		s, ok := p.sessions[id]
		p.mu.Unlock()
		if !ok {
			return nil, fmt.Errorf("%w: %q (use xdebug_sessions)", errNoSession, id)
		}
		return s, nil
	}
	var paused, all []*Session
	for _, s := range p.Sessions() {
		if s.Closed() {
			continue
		}
		all = append(all, s)
		if s.Paused() {
			paused = append(paused, s)
		}
	}
	switch {
	case len(paused) == 1:
		return paused[0], nil
	case len(all) == 0:
		return nil, errors.New("no xdebug sessions: start a PHP request with Xdebug enabled")
	case len(paused) == 0:
		return nil, errors.New("no paused xdebug session: stop at a breakpoint first")
	default:
		return nil, errors.New("several sessions are paused: pass session_id (see xdebug_sessions)")
	}
}

func quietEOF(err error) error {
	if err == nil || errors.Is(err, net.ErrClosed) || err.Error() == "EOF" {
		return nil
	}
	return err
}
