package mcp

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/buggregator/server/v2/modules/sentry"
	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

type sentryTracesListInput struct {
	Project       string `json:"project,omitempty" jsonschema:"Project key"`
	Name          string `json:"name,omitempty" jsonschema:"Only traces whose root transaction name contains this text, any case (a route like /orders or a job class)"`
	MinDurationMS int    `json:"min_duration_ms,omitempty" jsonschema:"Only traces whose root transaction took at least this many milliseconds"`
	NPlusOne      bool   `json:"n_plus_one,omitempty" jsonschema:"Only traces with an N+1 query group (the same SQL repeated in one transaction); looks through the newest 1000 traces"`
	Window        string `json:"window,omitempty" jsonschema:"Only traces that started within this period: 15m, 1h, 24h, 7d"`
	Limit         int    `json:"limit,omitempty" jsonschema:"Traces to return (default 20, max 100)"`
}

type sentryTraceGetInput struct {
	TraceID  string `json:"trace_id" jsonschema:"Trace id (from sentry_traces_list, or trace_id of a Sentry error event)"`
	Project  string `json:"project,omitempty" jsonschema:"Only this project's part of a distributed trace"`
	MaxSpans int    `json:"max_spans,omitempty" jsonschema:"Spans to include, earliest first (default 300, max 2000); N+1 groups and counts always cover the whole trace"`
}

func registerTraceTools(server *sdkmcp.Server, db *sql.DB) {
	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name: "sentry_traces_list",
		Description: "Sentry performance traces (requests, commands, queue jobs sent with tracing on), newest first, one row per trace: " +
			"root transaction name, op, status, whole-trace duration, span and error counts, services and the number of N+1 query groups. " +
			"Filter by route/name, minimum duration or n_plus_one=true to find slow requests and repeated queries. " +
			"Then call sentry_trace_get with a trace_id.",
	}, func(ctx context.Context, req *sdkmcp.CallToolRequest, in sentryTracesListInput) (*sdkmcp.CallToolResult, any, error) {
		res, err := sentry.TracesForAgent(db, sentry.TracesQuery{
			Project: in.Project, Name: in.Name, MinDurationMS: in.MinDurationMS,
			NPlusOne: in.NPlusOne, Window: in.Window, Limit: in.Limit,
		})
		if err != nil {
			return nil, nil, err
		}
		return jsonResult(res)
	})

	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name: "sentry_trace_get",
		Description: "One Sentry trace: the root transaction, the request timeline as a span tree (op, description such as the SQL or URL, " +
			"start offset and duration in ms, errors, code place file:line when the SDK sent it), N+1 groups (normalized query, count, total ms, " +
			"origin file:line), the errors raised in the trace and the queue job attempts that ran in it.",
	}, func(ctx context.Context, req *sdkmcp.CallToolRequest, in sentryTraceGetInput) (*sdkmcp.CallToolResult, any, error) {
		if in.TraceID == "" {
			return nil, nil, fmt.Errorf("trace_id is required")
		}
		res, err := sentry.TraceForAgent(db, in.TraceID, in.Project, in.MaxSpans)
		if err != nil {
			return nil, nil, err
		}
		if res == nil {
			return nil, nil, fmt.Errorf("trace not found: %s", in.TraceID)
		}
		return jsonResult(res)
	})
}

func jsonResult(v any) (*sdkmcp.CallToolResult, any, error) {
	data, _ := json.MarshalIndent(v, "", "  ")
	return &sdkmcp.CallToolResult{
		Content: []sdkmcp.Content{&sdkmcp.TextContent{Text: string(data)}},
	}, nil, nil
}
