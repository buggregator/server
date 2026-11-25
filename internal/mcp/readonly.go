package mcp

import sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"

// WriteTools are the tools that change data or drive a debug session:
// a read-only server drops them, so they are neither listed nor callable.
var WriteTools = []string{
	"event_delete",
	"mock_rule_create", "mock_rule_update", "mock_rule_delete",
	"mock_rule_from_request", "mock_import_openapi", "mock_group_delete",
	"xdebug_step", "xdebug_continue", "xdebug_stop", "xdebug_wait",
	"xdebug_breakpoint_set", "xdebug_breakpoint_remove", "xdebug_eval",
}

// ReadTools are the tools a read-only server keeps. Every tool is in one of
// the two lists (a test checks), so a new tool is classified on purpose.
var ReadTools = []string{
	"events_list", "events_facets", "event_get", "sentry_event", "vardump_get", "queue_jobs",
	"profiler_summary", "profiler_top", "profiler_call_graph",
	"mock_rules_list", "mock_rule_get", "mock_rule_preview", "mock_requests_list", "mock_template_helpers",
	"xdebug_sessions", "xdebug_location", "xdebug_stack", "xdebug_variables", "xdebug_property", "xdebug_breakpoints",
	"xdebug_traces", "xdebug_trace_summary", "xdebug_trace_calls", "xdebug_trace_call",
}

// MakeReadOnly removes the write tools from the server.
func MakeReadOnly(server *sdkmcp.Server) {
	server.RemoveTools(WriteTools...)
}
