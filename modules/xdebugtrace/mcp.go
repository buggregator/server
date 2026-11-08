package xdebugtrace

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/buggregator/go-buggregator/internal/event"
	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

// MCP answers are read by agents: smaller pages than the UI's.
const (
	mcpDefaultLimit = 50
	mcpMaxLimit     = 200
)

type tracesInput struct {
	Project string `json:"project,omitempty" jsonschema:"Only traces of this project key"`
	Limit   int    `json:"limit,omitempty" jsonschema:"How many traces, newest first (default 20, max 100)"`
}

type traceInput struct {
	UUID string `json:"uuid" jsonschema:"Trace event UUID (from xdebug_traces)"`
}

type callsInput struct {
	UUID   string `json:"uuid" jsonschema:"Trace event UUID (from xdebug_traces)"`
	Parent *int64 `json:"parent,omitempty" jsonschema:"offset of a call: list its direct callees. Omit for the top-level calls"`
	From   *int64 `json:"from,omitempty" jsonschema:"next from a previous answer: continue the same list"`
	Query  string `json:"query,omitempty" jsonschema:"Search instead: calls whose function name or file contains this text (case does not matter), in trace order"`
	Limit  int    `json:"limit,omitempty" jsonschema:"Calls per answer (default 50, max 200)"`
}

type callInput struct {
	UUID   string `json:"uuid" jsonschema:"Trace event UUID (from xdebug_traces)"`
	Offset int64  `json:"offset" jsonschema:"offset of the call (from xdebug_trace_calls or xdebug_trace_summary slowest)"`
}

// RegisterMCPTools adds the xdebug_trace* tools.
func (m *Module) RegisterMCPTools(server *sdkmcp.Server) {
	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        "xdebug_traces",
		Description: "List uploaded Xdebug function traces (xdebug.trace_format=1), newest first: script or request, duration in seconds, peak memory, number of calls, and whether the trace was cut at the size limit. Start here, then xdebug_trace_summary.",
	}, func(ctx context.Context, _ *sdkmcp.CallToolRequest, in tracesInput) (*sdkmcp.CallToolResult, any, error) {
		if m.store == nil {
			return nil, nil, errors.New("xdebug-trace module is not initialized")
		}
		limit := in.Limit
		if limit <= 0 {
			limit = 20
		}
		limit = min(limit, 100)
		evs, err := m.store.FindAll(ctx, event.FindOptions{Type: Type, Project: in.Project, Limit: limit})
		if err != nil {
			return nil, nil, err
		}
		out := []map[string]any{}
		for _, ev := range evs {
			var p Payload
			if json.Unmarshal(ev.Payload, &p) != nil || p.Summary == nil {
				continue
			}
			out = append(out, map[string]any{
				"uuid": ev.UUID, "project": ev.Project, "received": ev.Timestamp,
				"name": p.Name, "url": p.URL, "main": p.Summary.Main,
				"duration": p.Summary.Duration, "peak_memory": p.Summary.PeakMemory,
				"calls": p.Summary.Calls, "max_depth": p.Summary.MaxDepth, "truncated": p.Summary.Truncated,
			})
		}
		return textResult(out), nil, nil
	})

	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        "xdebug_trace_summary",
		Description: "Overview of one Xdebug trace: totals, the 20 functions with the most self time (time in the function itself, seconds), and the 20 slowest single calls by inclusive time, each with its offset for xdebug_trace_call. Times are seconds, memory bytes.",
	}, func(ctx context.Context, _ *sdkmcp.CallToolRequest, in traceInput) (*sdkmcp.CallToolResult, any, error) {
		p, err := m.payload(ctx, in.UUID)
		if err != nil {
			return nil, nil, fmt.Errorf("trace not found: %s", in.UUID)
		}
		return textResult(map[string]any{
			"name": p.Name, "url": p.URL, "size": p.Size, "summary": p.Summary,
		}), nil, nil
	})

	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        "xdebug_trace_calls",
		Description: "Walk the call tree of an Xdebug trace one level at a time, or search it. No parent: the top-level calls. parent=<offset>: the direct callees of that call. A non-null next means more: call again with from=next and the same parent. query=<text>: calls whose function or file contains the text. Each call has incl and self time (seconds), memory change, arguments and return value (cut to 512 bytes), the number of direct callees and its offset.",
	}, func(ctx context.Context, _ *sdkmcp.CallToolRequest, in callsInput) (*sdkmcp.CallToolResult, any, error) {
		limit := in.Limit
		if limit <= 0 {
			limit = mcpDefaultLimit
		}
		limit = min(limit, mcpMaxLimit)
		parent, from := int64(-1), int64(-1)
		if in.Parent != nil {
			parent = *in.Parent
		}
		if in.From != nil {
			from = *in.From
		}
		if parent < -1 || from < -1 {
			return nil, nil, errors.New("parent and from are offsets, not negative")
		}
		q := strings.TrimSpace(in.Query)
		page, err := m.withFile(ctx, in.UUID, func(f *os.File) (*Page, error) {
			if q != "" {
				return Search(f, m.limits.MaxSize, q, max(from, 0), limit)
			}
			return Children(f, m.limits.MaxSize, parent, from, limit)
		})
		if err != nil {
			return nil, nil, err
		}
		return textResult(page), nil, nil
	})

	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        "xdebug_trace_call",
		Description: "One call of an Xdebug trace by its offset: function, file and line, arguments, return value, incl and self time, memory change, and its first 50 direct callees. A non-null from means more callees: call xdebug_trace_calls with this parent and from.",
	}, func(ctx context.Context, _ *sdkmcp.CallToolRequest, in callInput) (*sdkmcp.CallToolResult, any, error) {
		if in.Offset < 0 {
			return nil, nil, errors.New("offset must not be negative")
		}
		page, err := m.withFile(ctx, in.UUID, func(f *os.File) (*Page, error) {
			return Children(f, m.limits.MaxSize, in.Offset, -1, mcpDefaultLimit)
		})
		if err != nil {
			return nil, nil, err
		}
		return textResult(map[string]any{
			"call": page.Parent, "callees": page.Nodes, "parent": in.Offset, "from": page.Next, "truncated": page.Truncated,
		}), nil, nil
	})
}

// withFile opens the trace file of uuid and runs scan on it.
func (m *Module) withFile(ctx context.Context, uuid string, scan func(*os.File) (*Page, error)) (*Page, error) {
	p, err := m.payload(ctx, uuid)
	if err != nil {
		return nil, fmt.Errorf("trace not found: %s", uuid)
	}
	f, err := os.Open(filepath.Join(m.dir, p.File))
	if err != nil {
		return nil, fmt.Errorf("trace file is gone: %s", uuid)
	}
	defer f.Close()
	return scan(f)
}

func textResult(v any) *sdkmcp.CallToolResult {
	// No HTML escaping: "->" in PHP names stays readable.
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	_ = enc.Encode(v)
	return &sdkmcp.CallToolResult{Content: []sdkmcp.Content{&sdkmcp.TextContent{Text: strings.TrimRight(buf.String(), "\n")}}}
}
