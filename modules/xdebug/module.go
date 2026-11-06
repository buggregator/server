// Package xdebug proxies Xdebug (DBGp) sessions to the IDE and lets an MCP
// agent read the same paused session: stack, variables, properties, eval.
// Phase 1 is read-only: the agent never steps, continues or changes breakpoints.
package xdebug

import (
	"github.com/buggregator/go-buggregator/internal/module"
	"github.com/buggregator/go-buggregator/internal/server/tcp"
	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

// Config of the Xdebug proxy.
type Config struct {
	Addr      string // where Xdebug connects (xdebug.client_host:client_port)
	IDEAddr   string // where the IDE listens for debug connections
	AllowEval bool   // expose xdebug_eval to MCP clients
}

type Module struct {
	module.BaseModule
	cfg   Config
	proxy *Proxy
}

func New(cfg Config) *Module {
	return &Module{cfg: cfg, proxy: NewProxy(cfg.IDEAddr)}
}

func (m *Module) Name() string { return "Xdebug proxy" }
func (m *Module) Type() string { return "xdebug" }

func (m *Module) TCPServers() []tcp.ServerConfig {
	return []tcp.ServerConfig{{Name: "xdebug", Address: m.cfg.Addr, Handler: m.proxy}}
}

// RegisterMCPTools adds the read-only xdebug_* tools to the MCP server.
func (m *Module) RegisterMCPTools(server *sdkmcp.Server) {
	registerTools(server, m.proxy, m.cfg.AllowEval)
}
