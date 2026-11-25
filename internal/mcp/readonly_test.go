package mcp_test

import (
	"context"
	"net/http"
	"slices"
	"testing"

	mcpserver "github.com/buggregator/server/v2/internal/mcp"
	"github.com/buggregator/server/v2/internal/module"
	"github.com/buggregator/server/v2/internal/storage"
	"github.com/buggregator/server/v2/modules/httpdumps"
	"github.com/buggregator/server/v2/modules/xdebug"
	"github.com/buggregator/server/v2/modules/xdebugtrace"
	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

// fullServer builds the MCP server with every tool provider and every
// optional tool on (mock writes, xdebug control and eval).
func fullServer(t *testing.T) *sdkmcp.Server {
	t.Helper()
	db, err := storage.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	store := storage.NewSQLiteStore(db)
	dumps := httpdumps.New(nil, db)
	mods := []module.Module{
		dumps,
		xdebug.New(xdebug.Config{Addr: "127.0.0.1:0", AllowEval: true, AllowControl: true}),
		xdebugtrace.New(t.TempDir(), xdebugtrace.Limits{}),
	}
	reg := module.NewRegistry()
	for _, m := range mods {
		reg.Register(m)
	}
	if err := reg.Init(db, http.NewServeMux(), store); err != nil {
		t.Fatal(err)
	}
	var providers []mcpserver.ToolProvider
	for _, m := range mods {
		providers = append(providers, m.(mcpserver.ToolProvider))
	}
	return mcpserver.NewServer(db, store, providers...)
}

func toolNames(t *testing.T, server *sdkmcp.Server) []string {
	t.Helper()
	ct, st := sdkmcp.NewInMemoryTransports()
	ctx := context.Background()
	ss, err := server.Connect(ctx, st, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ss.Close() })
	cs, err := sdkmcp.NewClient(&sdkmcp.Implementation{Name: "test", Version: "1.0"}, nil).Connect(ctx, ct, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { cs.Close() })
	res, err := cs.ListTools(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, tool := range res.Tools {
		names = append(names, tool.Name)
	}
	return names
}

// Every tool is either a read or a write tool: a new tool must be put in one
// list, so read-only mode never lets an unclassified write through.
func TestTools_AllClassified(t *testing.T) {
	names := toolNames(t, fullServer(t))
	for _, n := range names {
		if !slices.Contains(mcpserver.ReadTools, n) && !slices.Contains(mcpserver.WriteTools, n) {
			t.Errorf("tool %q is in neither ReadTools nor WriteTools", n)
		}
	}
	for _, n := range append(append([]string{}, mcpserver.ReadTools...), mcpserver.WriteTools...) {
		if !slices.Contains(names, n) {
			t.Errorf("listed tool %q does not exist", n)
		}
	}
}

func TestMakeReadOnly(t *testing.T) {
	server := fullServer(t)
	mcpserver.MakeReadOnly(server)
	names := toolNames(t, server)
	for _, n := range mcpserver.WriteTools {
		if slices.Contains(names, n) {
			t.Errorf("write tool %q still listed", n)
		}
	}
	if len(names) != len(mcpserver.ReadTools) {
		t.Errorf("%d tools left, want the %d read tools", len(names), len(mcpserver.ReadTools))
	}

	// A write tool is refused when called by name.
	ct, st := sdkmcp.NewInMemoryTransports()
	ctx := context.Background()
	ss, err := server.Connect(ctx, st, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer ss.Close()
	cs, err := sdkmcp.NewClient(&sdkmcp.Implementation{Name: "test", Version: "1.0"}, nil).Connect(ctx, ct, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer cs.Close()
	res, err := cs.CallTool(ctx, &sdkmcp.CallToolParams{Name: "event_delete", Arguments: map[string]any{"uuid": "x"}})
	if err == nil && (res == nil || !res.IsError) {
		t.Errorf("event_delete answered on a read-only server: %+v", res)
	}
}
