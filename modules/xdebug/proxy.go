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

// Proxy accepts Xdebug connections and forwards each one to the IDE.
type Proxy struct {
	ideAddr        string
	dialTimeout    time.Duration
	initTimeout    time.Duration
	commandTimeout time.Duration

	mu       sync.Mutex
	nextID   int
	sessions map[string]*Session
}

func NewProxy(ideAddr string) *Proxy {
	return &Proxy{
		ideAddr:        ideAddr,
		dialTimeout:    2 * time.Second,
		initTimeout:    10 * time.Second,
		commandTimeout: 10 * time.Second,
		sessions:       map[string]*Session{},
	}
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

	ide, err := net.DialTimeout("tcp", p.ideAddr, p.dialTimeout)
	if err != nil {
		slog.Warn("xdebug: IDE is not reachable, detaching so the script runs on", "ide", p.ideAddr, "file", s.FileURI, "err", err)
		s.detach()
		return
	}
	defer ide.Close()
	s.ide = ide

	if err := s.writeIDE(initBody); err != nil {
		slog.Warn("xdebug: cannot send init to IDE", "err", err)
		s.detach()
		return
	}

	p.mu.Lock()
	p.sessions[id] = s
	p.mu.Unlock()
	slog.Info("xdebug: session started", "session", id, "file", s.FileURI, "idekey", s.IDEKey)

	ideDone := make(chan error, 1)
	engineDone := make(chan error, 1)
	go func() { ideDone <- s.forwardIDE(bufio.NewReader(ide)) }()
	go func() { engineDone <- s.forwardEngine(er) }()

	select {
	case err := <-engineDone:
		s.close()
		ide.Close()
		<-ideDone
		slog.Info("xdebug: session ended", "session", id, "err", quietEOF(err))
	case err := <-ideDone:
		// The IDE went away first: let the script finish without a debugger,
		// then give the engine a moment to close its side.
		s.close()
		slog.Info("xdebug: IDE disconnected, detaching", "session", id, "err", quietEOF(err))
		s.detach()
		_ = engine.SetReadDeadline(time.Now().Add(2 * time.Second))
		<-engineDone
	}

	p.mu.Lock()
	delete(p.sessions, id)
	p.mu.Unlock()
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
			return nil, fmt.Errorf("no xdebug session %q (use xdebug_sessions)", id)
		}
		return s, nil
	}
	var paused []*Session
	all := p.Sessions()
	for _, s := range all {
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
		return nil, errors.New("no paused xdebug session: stop at a breakpoint in the IDE first")
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
