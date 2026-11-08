package xdebugtrace

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

func mcpSession(t *testing.T, m *Module) *sdkmcp.ClientSession {
	t.Helper()
	server := sdkmcp.NewServer(&sdkmcp.Implementation{Name: "test", Version: "1"}, nil)
	m.RegisterMCPTools(server)
	ct, st := sdkmcp.NewInMemoryTransports()
	ss, err := server.Connect(context.Background(), st, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ss.Close() })
	cs, err := sdkmcp.NewClient(&sdkmcp.Implementation{Name: "c", Version: "1"}, nil).Connect(context.Background(), ct, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { cs.Close() })
	return cs
}

func callMCP(t *testing.T, s *sdkmcp.ClientSession, name string, args map[string]any, out any) bool {
	t.Helper()
	res, err := s.CallTool(context.Background(), &sdkmcp.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	text := res.Content[0].(*sdkmcp.TextContent).Text
	if res.IsError {
		return false
	}
	if out != nil {
		if err := json.Unmarshal([]byte(text), out); err != nil {
			t.Fatalf("%s: %v: %s", name, err, text)
		}
	}
	if strings.Contains(text, "\\u003e") { // "->" must stay readable
		t.Errorf("%s: HTML-escaped output: %s", name, text)
	}
	return true
}

func TestMCPTools(t *testing.T) {
	e := newEnv(t, Limits{})
	in, err := e.upload(t, gz(t, sample), map[string]string{"X-Buggregator-Trace-Url": "GET /users"})
	if err != nil {
		t.Fatal(err)
	}
	s := mcpSession(t, e.mod)

	tools, err := s.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	names := map[string]bool{}
	for _, tl := range tools.Tools {
		names[tl.Name] = true
	}
	for _, n := range []string{"xdebug_traces", "xdebug_trace_summary", "xdebug_trace_calls", "xdebug_trace_call"} {
		if !names[n] {
			t.Errorf("tool %s missing", n)
		}
	}

	var list []map[string]any
	if !callMCP(t, s, "xdebug_traces", nil, &list) || len(list) != 1 || list[0]["uuid"] != in.UUID || list[0]["url"] != "GET /users" {
		t.Fatalf("xdebug_traces = %v", list)
	}

	var sum struct {
		Summary Summary `json:"summary"`
	}
	if !callMCP(t, s, "xdebug_trace_summary", map[string]any{"uuid": in.UUID}, &sum) || sum.Summary.Calls != 6 || sum.Summary.Index != nil {
		t.Errorf("summary = %+v", sum)
	}

	var top Page
	if !callMCP(t, s, "xdebug_trace_calls", map[string]any{"uuid": in.UUID}, &top) || len(top.Nodes) != 1 || top.Nodes[0].Func != "{main}" {
		t.Fatalf("top = %+v", top)
	}
	var lvl Page
	callMCP(t, s, "xdebug_trace_calls", map[string]any{"uuid": in.UUID, "parent": top.Nodes[0].Offset}, &lvl)
	if len(lvl.Nodes) != 1 || lvl.Nodes[0].Func != "App\\Kernel->handle" {
		t.Fatalf("callees of {main} = %+v", lvl.Nodes)
	}
	var page Page
	callMCP(t, s, "xdebug_trace_calls", map[string]any{"uuid": in.UUID, "parent": lvl.Nodes[0].Offset, "limit": 2}, &page)
	if len(page.Nodes) != 2 || page.Next == nil {
		t.Fatalf("page = %+v", page)
	}
	var rest Page
	callMCP(t, s, "xdebug_trace_calls", map[string]any{"uuid": in.UUID, "parent": lvl.Nodes[0].Offset, "from": *page.Next}, &rest)
	if len(rest.Nodes) != 1 || rest.Nodes[0].Func != "App\\Db->query" {
		t.Errorf("rest = %+v", rest.Nodes)
	}

	var found Page
	callMCP(t, s, "xdebug_trace_calls", map[string]any{"uuid": in.UUID, "query": "pdo"}, &found)
	if len(found.Nodes) != 1 || len(found.Nodes[0].Path) != 3 {
		t.Fatalf("search = %+v", found.Nodes)
	}

	var call struct {
		Call    *Node  `json:"call"`
		Callees []Node `json:"callees"`
	}
	callMCP(t, s, "xdebug_trace_call", map[string]any{"uuid": in.UUID, "offset": rest.Nodes[0].Offset}, &call)
	if call.Call == nil || call.Call.Func != "App\\Db->query" || len(call.Callees) != 1 || call.Callees[0].Func != "PDO->query" {
		t.Errorf("call = %+v", call)
	}

	// More callees than a page: the answer names parent and from for
	// xdebug_trace_calls, which then stays inside the call.
	var handle struct {
		Callees []Node `json:"callees"`
		Parent  int64  `json:"parent"`
		From    *int64 `json:"from"`
	}
	callMCP(t, s, "xdebug_trace_call", map[string]any{"uuid": in.UUID, "offset": top.Nodes[0].Offset}, &handle)
	if handle.Parent != top.Nodes[0].Offset || handle.From != nil || len(handle.Callees) != 1 {
		t.Errorf("call {main} = %+v", handle)
	}

	for name, args := range map[string]map[string]any{
		"xdebug_trace_summary": {"uuid": "nope"},
		"xdebug_trace_calls":   {"uuid": in.UUID, "parent": 1},
		"xdebug_trace_call":    {"uuid": in.UUID, "offset": -5},
	} {
		if callMCP(t, s, name, args, nil) {
			t.Errorf("%s %v: no error", name, args)
		}
	}
}

func TestMCPTools_CallPaging(t *testing.T) {
	e := newEnv(t, Limits{})
	in, err := e.upload(t, []byte(bigTrace(60)), nil)
	if err != nil {
		t.Fatal(err)
	}
	s := mcpSession(t, e.mod)
	var top Page
	callMCP(t, s, "xdebug_trace_calls", map[string]any{"uuid": in.UUID}, &top)
	var call struct {
		Callees []Node `json:"callees"`
		Parent  int64  `json:"parent"`
		From    *int64 `json:"from"`
	}
	callMCP(t, s, "xdebug_trace_call", map[string]any{"uuid": in.UUID, "offset": top.Nodes[0].Offset}, &call)
	if len(call.Callees) != 50 || call.From == nil {
		t.Fatalf("first callees = %d, from %v", len(call.Callees), call.From)
	}
	var rest Page
	callMCP(t, s, "xdebug_trace_calls", map[string]any{"uuid": in.UUID, "parent": call.Parent, "from": *call.From}, &rest)
	if len(rest.Nodes) != 10 || rest.Next != nil {
		t.Errorf("rest = %d next %v", len(rest.Nodes), rest.Next)
	}
}
