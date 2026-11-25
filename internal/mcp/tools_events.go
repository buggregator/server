package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/buggregator/server/v2/internal/event"
	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

type eventsListInput struct {
	Type    string `json:"type,omitempty" jsonschema:"Filter by event type (sentry, profiler, var-dump, ray, inspector, smtp, monolog, http-dump, sms, csp, html, broadcast, xdebug-trace)"`
	Project string `json:"project,omitempty" jsonschema:"Filter by project key"`
	Limit   int    `json:"limit,omitempty" jsonschema:"Maximum number of events to return (default 20, max 100)"`
	eventFilterInput
}

// eventFilterInput is the feed filter of events_list and events_facets:
// values of one field are OR-ed, different fields AND-ed.
type eventFilterInput struct {
	Host        []string `json:"host,omitempty" jsonschema:"Only events from these machines (Sentry server_name, Ray/Inspector/profiler hostname, Monolog extra.hostname, request host of dumps and HTTP dumps)"`
	Env         []string `json:"env,omitempty" jsonschema:"Only events of these environments (Sentry environment)"`
	User        []string `json:"user,omitempty" jsonschema:"Only events of these users (Sentry user email, username, id or IP)"`
	Channel     []string `json:"channel,omitempty" jsonschema:"Only events of these channels (Monolog channel, Sentry logger)"`
	Level       []string `json:"level,omitempty" jsonschema:"Only events of these levels (Monolog level name, Sentry level)"`
	Release     []string `json:"release,omitempty" jsonschema:"Only events of these releases (Sentry release)"`
	Tag         []string `json:"tag,omitempty" jsonschema:"Only events with these tags, each as name:value (Sentry and profiler tags)"`
	Trace       []string `json:"trace,omitempty" jsonschema:"Only events of these trace ids: Monolog extra.trace_id or context.trace_id, mails with an X-Trace-Id / sentry-trace / traceparent header, HTTP dumps with a sentry-trace / traceparent header"`
	ExcludeHost []string `json:"exclude_host,omitempty" jsonschema:"Leave out events from these machines"`
	Query       string   `json:"query,omitempty" jsonschema:"Full-text search: every word must occur as a substring, case-insensitive (Monolog message, channel, context, extra)"`
}

// filters converts the input to store filters.
func (in eventFilterInput) filters() []event.FacetFilter {
	var out []event.FacetFilter
	add := func(key string, values []string, not bool) {
		if len(values) > 0 {
			out = append(out, event.FacetFilter{Key: key, Values: values, Not: not})
		}
	}
	add(event.FacetHost, in.Host, false)
	add(event.FacetEnv, in.Env, false)
	add(event.FacetUser, in.User, false)
	add(event.FacetChannel, in.Channel, false)
	add(event.FacetLevel, in.Level, false)
	add(event.FacetRelease, in.Release, false)
	add(event.FacetTrace, in.Trace, false)
	add(event.FacetHost, in.ExcludeHost, true)
	byTag := map[string][]string{}
	var names []string
	for _, t := range in.Tag {
		name, value, ok := strings.Cut(t, ":")
		if !ok || name == "" || value == "" {
			continue
		}
		if _, seen := byTag[name]; !seen {
			names = append(names, name)
		}
		byTag[name] = append(byTag[name], value)
	}
	for _, n := range names {
		add(event.FacetTagPrefix+n, byTag[n], false)
	}
	return out
}

type eventsFacetsInput struct {
	Type    string `json:"type,omitempty" jsonschema:"Only events of this type"`
	Project string `json:"project,omitempty" jsonschema:"Only events of this project key"`
	eventFilterInput
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
			Filters: input.filters(),
			Search:  input.Query,
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
		Name:        "events_facets",
		Description: "List the values events can be filtered by (machines, environments, users, channels, levels, releases, tags as tag:<name>) with event counts, most frequent first. Pass the values to events_list to see only matching events.",
	}, func(ctx context.Context, req *sdkmcp.CallToolRequest, input eventsFacetsInput) (*sdkmcp.CallToolResult, any, error) {
		fc, ok := store.(interface {
			FacetCounts(context.Context, event.FindOptions, int) (map[string][]event.FacetCount, error)
		})
		counts := map[string][]event.FacetCount{}
		if ok {
			var err error
			counts, err = fc.FacetCounts(ctx, event.FindOptions{
				Type:    input.Type,
				Project: input.Project,
				Filters: input.filters(),
				Search:  input.Query,
			}, 20)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to count facets: %w", err)
			}
		}
		data, _ := json.MarshalIndent(counts, "", "  ")
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
