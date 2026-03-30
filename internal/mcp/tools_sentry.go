package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/buggregator/go-buggregator/internal/event"
	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

type sentryEventInput struct {
	UUID string `json:"uuid" jsonschema:"Event UUID (use events_list with type='sentry' to find UUIDs)"`
}

func registerSentryTools(server *sdkmcp.Server, store event.Store) {
	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        "sentry_event",
		Description: "Get structured details of a Sentry error event: error message, severity level, exception chain with stack traces, environment, and platform. Returns clean, AI-friendly data stripped of UI-specific fields.",
	}, func(ctx context.Context, req *sdkmcp.CallToolRequest, input sentryEventInput) (*sdkmcp.CallToolResult, any, error) {
		ev, err := store.FindByUUID(ctx, input.UUID)
		if err != nil {
			return nil, nil, fmt.Errorf("event not found: %s", input.UUID)
		}

		if ev.Type != "sentry" {
			return nil, nil, fmt.Errorf("event %s is not a sentry event (type: %s)", input.UUID, ev.Type)
		}

		// Parse the raw Sentry payload.
		var raw map[string]any
		if err := json.Unmarshal(ev.Payload, &raw); err != nil {
			return nil, nil, fmt.Errorf("failed to parse sentry payload: %w", err)
		}

		// Extract relevant fields for AI consumption.
		result := make(map[string]any)

		for _, key := range []string{"event_id", "message", "level", "platform", "environment", "server_name", "timestamp", "logger", "release"} {
			if v, ok := raw[key]; ok && v != nil {
				result[key] = v
			}
		}

		// Extract exception chain with stack traces.
		if exc, ok := raw["exception"].(map[string]any); ok {
			if values, ok := exc["values"].([]any); ok {
				var exceptions []map[string]any
				for _, v := range values {
					if val, ok := v.(map[string]any); ok {
						exception := map[string]any{}
						if t, ok := val["type"]; ok {
							exception["type"] = t
						}
						if v, ok := val["value"]; ok {
							exception["value"] = v
						}
						if st, ok := val["stacktrace"].(map[string]any); ok {
							if frames, ok := st["frames"].([]any); ok {
								var cleanFrames []map[string]any
								for _, f := range frames {
									if frame, ok := f.(map[string]any); ok {
										cleanFrame := make(map[string]any)
										for _, fk := range []string{"filename", "function", "lineno", "colno", "abs_path", "context_line", "module"} {
											if fv, ok := frame[fk]; ok && fv != nil {
												cleanFrame[fk] = fv
											}
										}
										cleanFrames = append(cleanFrames, cleanFrame)
									}
								}
								exception["stacktrace"] = cleanFrames
							}
						}
						exceptions = append(exceptions, exception)
					}
				}
				result["exceptions"] = exceptions
			}
		}

		// Extract tags and contexts if present.
		if tags, ok := raw["tags"].(map[string]any); ok {
			result["tags"] = tags
		}
		if tags, ok := raw["tags"].([]any); ok && len(tags) > 0 {
			result["tags"] = tags
		}

		data, _ := json.MarshalIndent(result, "", "  ")
		return &sdkmcp.CallToolResult{
			Content: []sdkmcp.Content{&sdkmcp.TextContent{Text: string(data)}},
		}, nil, nil
	})
}
