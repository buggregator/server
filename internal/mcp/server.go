package mcp

import (
	"database/sql"

	"github.com/buggregator/go-buggregator/internal/event"
	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

// NewServer creates an MCP server with all Buggregator tools registered.
func NewServer(db *sql.DB, store event.Store) *sdkmcp.Server {
	server := sdkmcp.NewServer(&sdkmcp.Implementation{
		Name:    "buggregator",
		Version: "1.0.0",
	}, nil)

	registerEventTools(server, store)
	registerProfilerTools(server, db)
	registerSentryTools(server, store)
	registerVarDumpTools(server, store)

	return server
}
