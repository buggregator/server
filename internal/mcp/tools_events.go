package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/buggregator/go-buggregator/internal/event"
	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

type eventsListInput struct {
	Type    string `json:"type,omitempty" jsonschema:"Filter by event type (sentry, profiler, var-dump, ray, inspector, smtp, monolog, http-dump)"`
	Project string `json:"project,omitempty" jsonschema:"Filter by project key"`
	Limit   int    `json:"limit,omitempty" jsonschema:"Maximum number of events to return (default 20, max 100)"`
}

type eventGetInput struct {
	UUID string `json:"uuid" jsonschema:"Event UUID"`
}

type eventDeleteInput struct {
	UUID string `json:"uuid" jsonschema:"Event UUID to delete"`
}

func registerEventTools(server *sdkmcp.Server, store event.Store) {
	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        "events_list",
		Description: "List debugging events captured by Buggregator. Returns event metadata (uuid, type, timestamp, project) without payloads to keep output compact. Use event_get to fetch full details.",
	}, func(ctx context.Context, req *sdkmcp.CallToolRequest, input eventsListInput) (*sdkmcp.CallToolResult, any, error) {
		limit := input.Limit
		if limit <= 0 {
			limit = 20
		}
		if limit > 100 {
			limit = 100
		}

		events, err := store.FindAll(ctx, event.FindOptions{
			Type:    input.Type,
			Project: input.Project,
			Limit:   limit,
		})
		if err != nil {
			return nil, nil, fmt.Errorf("failed to list events: %w", err)
		}

		type eventSummary struct {
			UUID      string  `json:"uuid"`
			Type      string  `json:"type"`
			Timestamp float64 `json:"timestamp"`
			Project   string  `json:"project,omitempty"`
			IsPinned  bool    `json:"is_pinned,omitempty"`
		}

		summaries := make([]eventSummary, len(events))
		for i, ev := range events {
			summaries[i] = eventSummary{
				UUID:      ev.UUID,
				Type:      ev.Type,
				Timestamp: ev.Timestamp,
				Project:   ev.Project,
				IsPinned:  ev.IsPinned,
			}
		}

		data, _ := json.MarshalIndent(summaries, "", "  ")
		return &sdkmcp.CallToolResult{
			Content: []sdkmcp.Content{&sdkmcp.TextContent{Text: string(data)}},
		}, nil, nil
	})

	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        "event_get",
		Description: "Get a complete event by UUID, including the full payload. Use events_list first to find event UUIDs.",
	}, func(ctx context.Context, req *sdkmcp.CallToolRequest, input eventGetInput) (*sdkmcp.CallToolResult, any, error) {
		ev, err := store.FindByUUID(ctx, input.UUID)
		if err != nil {
			return nil, nil, fmt.Errorf("event not found: %s", input.UUID)
		}

		data, _ := json.MarshalIndent(ev, "", "  ")
		return &sdkmcp.CallToolResult{
			Content: []sdkmcp.Content{&sdkmcp.TextContent{Text: string(data)}},
		}, nil, nil
	})

	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        "event_delete",
		Description: "Delete an event by UUID.",
	}, func(ctx context.Context, req *sdkmcp.CallToolRequest, input eventDeleteInput) (*sdkmcp.CallToolResult, any, error) {
		if err := store.Delete(ctx, input.UUID); err != nil {
			return nil, nil, fmt.Errorf("failed to delete event: %w", err)
		}
		return &sdkmcp.CallToolResult{
			Content: []sdkmcp.Content{&sdkmcp.TextContent{Text: fmt.Sprintf("Event %s deleted", input.UUID)}},
		}, nil, nil
	})
}
