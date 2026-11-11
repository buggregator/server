package mcp

import (
	"database/sql"

	"github.com/buggregator/server/v2/internal/event"
	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

// ToolProvider is implemented by modules that add their own MCP tools.
type ToolProvider interface {
	RegisterMCPTools(server *sdkmcp.Server)
}

// NewServer creates an MCP server with all Buggregator tools registered,
// plus the tools of the given providers.
func NewServer(db *sql.DB, store event.Store, providers ...ToolProvider) *sdkmcp.Server {
	server := sdkmcp.NewServer(&sdkmcp.Implementation{
		Name:    "buggregator",
		Version: "1.0.0",
	}, nil)

	registerEventTools(server, store)
	registerProfilerTools(server, db)
	registerSentryTools(server, store)
	registerVarDumpTools(server, store)
	for _, p := range providers {
		p.RegisterMCPTools(server)
	}

	return server
}
