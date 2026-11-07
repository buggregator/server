// Package xdebug proxies Xdebug (DBGp) sessions to the IDE and lets an MCP
// agent and the web UI read the same paused session: stack, variables,
// properties, eval. With no IDE (mode auto or standalone) Buggregator drives
// the session itself: the UI steps, continues, stops or releases it. While an
// IDE is attached, the IDE drives and Buggregator only reads.
package xdebug

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/buggregator/go-buggregator/internal/module"
	"github.com/buggregator/go-buggregator/internal/server/tcp"
	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

// Channel is the WebSocket channel of the xdebug events.
const Channel = "xdebug"

// Config of the Xdebug proxy.
type Config struct {
	Addr      string // where Xdebug connects (xdebug.client_host:client_port)
	IDEAddr   string // where the IDE listens for debug connections
	AllowEval bool   // expose eval to MCP clients and the UI
	// AllowControl exposes the MCP control tools (step, continue, stop, wait).
	AllowControl bool

	Mode             string        // proxy, auto or standalone; empty = auto
	PauseOnFirstLine bool          // a session driven here stops at the first line
	IdleTimeout      time.Duration // release a session paused here after this long without a command; 0 = 10 min, <0 = never
}

// Broadcaster publishes an event on a WebSocket channel (the ws.Hub).
type Broadcaster interface {
	Broadcast(channel, eventName string, payload any)
}

type Module struct {
	module.BaseModule
	cfg    Config
	proxy  *Proxy
	events chan wsEvent
}

type wsEvent struct {
	name    string
	payload any
}

func New(cfg Config) *Module {
	if cfg.Mode == "" {
		cfg.Mode = ModeAuto
	}
	p := NewProxy(cfg.IDEAddr)
	if err := p.SetSettings(Settings{Mode: cfg.Mode, PauseOnFirstLine: cfg.PauseOnFirstLine}); err != nil {
		slog.Warn("xdebug: unknown mode, using auto", "mode", cfg.Mode)
		_ = p.SetSettings(Settings{Mode: ModeAuto, PauseOnFirstLine: cfg.PauseOnFirstLine})
	}
	switch {
	case cfg.IdleTimeout < 0:
		p.idleTimeout = 0
	case cfg.IdleTimeout > 0:
		p.idleTimeout = cfg.IdleTimeout
	}
	return &Module{cfg: cfg, proxy: p}
}

// SetBroadcaster sends session.* and agent.* events to the WebSocket hub.
// Events are queued and sent from one goroutine, so a slow browser never
// holds up the DBGp stream; when the queue is full an event is dropped.
func (m *Module) SetBroadcaster(b Broadcaster) {
	m.events = make(chan wsEvent, 256)
	go func() {
		for ev := range m.events {
			b.Broadcast(Channel, ev.name, ev.payload)
		}
	}()
	m.proxy.SetNotify(func(name string, payload any) {
		select {
		case m.events <- wsEvent{name, payload}:
		default:
			slog.Warn("xdebug: websocket event dropped, queue full", "event", name)
		}
	})
}

func (m *Module) Name() string { return "Xdebug proxy" }
func (m *Module) Type() string { return "xdebug" }

func (m *Module) TCPServers() []tcp.ServerConfig {
	return []tcp.ServerConfig{{Name: "xdebug", Address: m.cfg.Addr, Handler: m.proxy}}
}

// RegisterProtectedRoutes adds the /api/xdebug routes behind the API's auth.
func (m *Module) RegisterProtectedRoutes(mux *http.ServeMux, auth func(http.Handler) http.Handler) {
	m.registerAPI(mux, auth)
}

// RegisterMCPTools adds the xdebug_* tools to the MCP server: reads, the
// control tools when allow_control is on, xdebug_eval when allow_eval is on.
func (m *Module) RegisterMCPTools(server *sdkmcp.Server) {
	registerTools(server, m.proxy, m.cfg.AllowEval, m.cfg.AllowControl)
}

var _ module.ProtectedRoutesRegistrar = (*Module)(nil)
