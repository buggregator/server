// Package explain sends a Sentry error to the agents of a harnsy project
// over h2a (github.com/harnesy/h2a-go) and shows their answers in
// Buggregator: «Explain this error», then a conversation about it.
//
// A concept: conversations live in memory, the text is not scrubbed beyond
// leaving out the request's headers and body. Off unless a URL and a token
// are set.
package explain

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/buggregator/server/v2/internal/event"
	"github.com/buggregator/server/v2/internal/module"
	"github.com/buggregator/server/v2/internal/server/tcp"
	"github.com/harnesy/h2a-go"
)

// EventName is the WebSocket event of a new line in a thread, on the
// event's project channel.
const EventName = "explain.message"

// Config of the h2a connection.
type Config struct {
	URL   string // wss://<node gateway>/h2a/v1
	Token string // the pass's token (h2a_…), a secret
	To    string // the address the explanations go to; "" = lead
}

// sender is what the module needs of the h2a client.
type sender interface {
	Send(ctx context.Context, m h2a.Message) (h2a.Accepted, error)
	Welcome() h2a.Welcome
}

type Module struct {
	module.BaseModule
	cfg     Config
	store   event.Store
	threads *threads
	bc      module.Broadcaster

	mu     sync.Mutex
	client sender
	status string // connecting, connected, stopped
	reason string // why stopped or last connection error
	cancel context.CancelFunc
	done   chan struct{}
}

func New(cfg Config) *Module {
	if cfg.To == "" {
		cfg.To = "lead"
	}
	return &Module{cfg: cfg, threads: newThreads(), status: "connecting"}
}

func (m *Module) Name() string { return "Explain (harnsy h2a)" }
func (m *Module) Type() string { return "explain" }

func (m *Module) SetBroadcaster(b module.Broadcaster) { m.bc = b }

func (m *Module) RegisterRoutes(_ *http.ServeMux, store event.Store) { m.store = store }

func (m *Module) RegisterProtectedRoutes(mux *http.ServeMux, protect func(http.Handler) http.Handler) {
	m.registerAPI(mux, protect)
}

func (m *Module) OnEventsDeleted(uuids []string) { m.threads.remove(uuids) }

// TCPServers runs the h2a client with the other servers: started after
// startup, stopped on shutdown. It opens no port.
func (m *Module) TCPServers() []tcp.ServerConfig {
	addr := m.cfg.URL
	if u, err := url.Parse(m.cfg.URL); err == nil {
		addr = u.Host // never the token, never a query
	}
	return []tcp.ServerConfig{{Name: "explain-h2a", Address: addr, Starter: m}}
}

// Start connects in the background: harnsy being away never stops Buggregator.
func (m *Module) Start(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	m.cancel = cancel
	m.done = make(chan struct{})
	go m.loop(ctx)
	return nil
}

func (m *Module) Stop() error {
	if m.cancel != nil {
		m.cancel()
		<-m.done
	}
	return nil
}

// loop signs in, then hands every answer to onReply. The SDK reconnects by
// itself; loop only retries a sign-in that failed for a reason that passes.
func (m *Module) loop(ctx context.Context) {
	defer close(m.done)
	if err := checkURL(m.cfg.URL); err != nil {
		m.setStatus(nil, "stopped", err.Error())
		slog.Error("explain: bad h2a URL; module off", "err", err)
		return
	}
	wait := 5 * time.Second
	for ctx.Err() == nil {
		c, err := h2a.Dial(ctx, h2a.Config{
			URL:   m.cfg.URL,
			Token: m.cfg.Token,
			Logf: func(format string, args ...any) {
				slog.Debug("explain: h2a "+format, args...)
			},
		})
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			if final(err) {
				m.setStatus(nil, "stopped", err.Error())
				slog.Error("explain: harnsy refused the pass; not retrying", "err", err)
				return
			}
			m.setStatus(nil, "connecting", err.Error())
			slog.Warn("explain: cannot reach harnsy, retrying", "err", err, "in", wait)
			select {
			case <-ctx.Done():
				return
			case <-time.After(wait):
			}
			if wait < time.Minute {
				wait *= 2
			}
			continue
		}
		wait = 5 * time.Second
		w := c.Welcome()
		m.setStatus(c, "connected", "")
		slog.Info("explain: connected to harnsy", "project", w.Project, "to", m.cfg.To)

		err = c.Run(ctx, m.onReply)
		_ = c.Close()
		if ctx.Err() != nil {
			return
		}
		m.setStatus(nil, "stopped", errString(err))
		slog.Error("explain: h2a connection stopped", "err", err)
		if err != nil && final(err) {
			return
		}
	}
}

// checkURL refuses at once what the SDK would refuse on every try.
func checkURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil {
		return err
	}
	if u.User != nil || u.RawQuery != "" {
		return errors.New("the URL takes no user or query: the token goes in EXPLAIN_H2A_TOKEN")
	}
	host := u.Hostname()
	if u.Scheme == "wss" || (u.Scheme == "ws" && (host == "localhost" || host == "127.0.0.1" || host == "::1")) {
		return nil
	}
	return errors.New("the URL must be wss:// (ws:// only to this machine)")
}

// final: errors that need a person (a new token, the pass on, one copy of
// the service), so retrying would only be refused again.
func final(err error) bool {
	if errors.Is(err, h2a.ErrReplaced) || errors.Is(err, h2a.ErrVersion) {
		return true
	}
	var e *h2a.Error
	if errors.As(err, &e) {
		return e.Code == h2a.CodeAuth || e.Code == h2a.CodeOff
	}
	return false
}

func (m *Module) onReply(_ context.Context, r h2a.Reply) error {
	msg := Message{From: "agent", Role: r.From.Role, Project: r.From.Project, Text: r.Text, At: r.At}
	if msg.At.IsZero() {
		msg.At = time.Now().UTC()
	}
	th, ok := m.threads.add(r.Conversation, msg)
	if !ok {
		// A conversation from before a restart: nothing to show it under.
		slog.Warn("explain: answer to an unknown conversation dropped", "conversation", r.Conversation)
		return nil
	}
	m.broadcast(th, msg)
	return nil
}

func (m *Module) broadcast(th Thread, msg Message) {
	if m.bc == nil {
		return
	}
	m.bc.Broadcast("events.project."+th.Project, EventName, map[string]any{
		"event_uuid":   th.EventUUID,
		"conversation": th.Conversation,
		"message":      msg,
	})
}

func (m *Module) setStatus(c sender, status, reason string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.client, m.status, m.reason = c, status, reason
}

func (m *Module) current() (sender, string, string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.client, m.status, m.reason
}

func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
