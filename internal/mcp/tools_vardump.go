package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/buggregator/go-buggregator/internal/event"
	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

type vardumpGetInput struct {
	UUID string `json:"uuid" jsonschema:"Event UUID (use events_list with type='var-dump' to find UUIDs)"`
}

var htmlTagRegex = regexp.MustCompile(`<[^>]*>`)

func registerVarDumpTools(server *sdkmcp.Server, store event.Store) {
	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        "vardump_get",
		Description: "Get a var-dump value with HTML stripped for clean AI consumption. Returns the variable type, label, and plain text representation of the dumped value.",
	}, func(ctx context.Context, req *sdkmcp.CallToolRequest, input vardumpGetInput) (*sdkmcp.CallToolResult, any, error) {
		ev, err := store.FindByUUID(ctx, input.UUID)
		if err != nil {
			return nil, nil, fmt.Errorf("event not found: %s", input.UUID)
		}

		if ev.Type != "var-dump" {
			return nil, nil, fmt.Errorf("event %s is not a var-dump event (type: %s)", input.UUID, ev.Type)
		}

		// Parse the payload.
		var payload struct {
			Payload struct {
				Type     string `json:"type"`
				Value    string `json:"value"`
				Label    string `json:"label"`
				Language string `json:"language"`
			} `json:"payload"`
			Context json.RawMessage `json:"context"`
		}

		if err := json.Unmarshal(ev.Payload, &payload); err != nil {
			// Fallback: try direct structure.
			var direct struct {
				Type     string `json:"type"`
				Value    string `json:"value"`
				Label    string `json:"label"`
				Language string `json:"language"`
			}
			if err2 := json.Unmarshal(ev.Payload, &direct); err2 != nil {
				return nil, nil, fmt.Errorf("failed to parse var-dump payload: %w", err)
			}
			payload.Payload = direct
		}

		// Strip HTML from value.
		cleanValue := stripHTML(payload.Payload.Value)

		result := map[string]any{
			"type":  payload.Payload.Type,
			"value": cleanValue,
		}
		if payload.Payload.Label != "" {
			result["label"] = payload.Payload.Label
		}
		if payload.Payload.Language != "" {
			result["language"] = payload.Payload.Language
		}

		data, _ := json.MarshalIndent(result, "", "  ")
		return &sdkmcp.CallToolResult{
			Content: []sdkmcp.Content{&sdkmcp.TextContent{Text: string(data)}},
		}, nil, nil
	})
}

// stripHTML removes HTML tags and decodes common entities.
func stripHTML(s string) string {
	s = htmlTagRegex.ReplaceAllString(s, "")
	s = strings.ReplaceAll(s, "&lt;", "<")
	s = strings.ReplaceAll(s, "&gt;", ">")
	s = strings.ReplaceAll(s, "&amp;", "&")
	s = strings.ReplaceAll(s, "&quot;", "\"")
	s = strings.ReplaceAll(s, "&#039;", "'")
	s = strings.ReplaceAll(s, "&nbsp;", " ")
	return s
}
