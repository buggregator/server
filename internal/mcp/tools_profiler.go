package mcp

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/buggregator/go-buggregator/modules/profiler"
	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

type profilerSummaryInput struct {
	UUID string `json:"uuid" jsonschema:"Profile UUID"`
}

type profilerTopInput struct {
	UUID   string `json:"uuid" jsonschema:"Profile UUID"`
	Metric string `json:"metric,omitempty" jsonschema:"Sort metric: cpu, wt (wall time), mu (memory), pmu (peak memory), ct (calls), excl_cpu, excl_wt, excl_mu, excl_pmu, excl_ct. Default: cpu"`
	Limit  int    `json:"limit,omitempty" jsonschema:"Number of functions to return (default 50, min 5, max 200)"`
}

type profilerCallGraphInput struct {
	UUID       string  `json:"uuid" jsonschema:"Profile UUID"`
	Metric     string  `json:"metric,omitempty" jsonschema:"Metric for filtering: cpu, wt, mu, pmu. Default: cpu"`
	Threshold  float64 `json:"threshold,omitempty" jsonschema:"Upper bound for filtering (0-100). Nodes below this AND below percentage are excluded. Default: 0"`
	Percentage float64 `json:"percentage,omitempty" jsonschema:"Minimum importance threshold (0-100). Nodes above this are always shown. Default: 1"`
}

func registerProfilerTools(server *sdkmcp.Server, db *sql.DB) {
	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        "profiler_summary",
		Description: "Get a quick overview of a profiling session: total CPU/wall time/memory, the slowest function, the biggest memory consumer, and the most called function. Use events_list with type='profiler' to find profile UUIDs.",
	}, func(ctx context.Context, req *sdkmcp.CallToolRequest, input profilerSummaryInput) (*sdkmcp.CallToolResult, any, error) {
		edges, err := profiler.LoadAllEdges(db, input.UUID)
		if err != nil {
			return nil, nil, fmt.Errorf("profile not found: %s", input.UUID)
		}

		peaks, name, _ := profiler.LoadPeaks(db, input.UUID)
		funcs := profiler.AggregateFunctions(edges, peaks)

		var bestWt, bestMu, bestCt int64
		var slowest, memHotspot, mostCalled map[string]any
		for _, f := range funcs {
			if f.Callee == "main()" {
				continue
			}
			if slowest == nil || f.ExclWt > bestWt {
				bestWt = f.ExclWt
				slowest = map[string]any{"function": f.Callee, "excl_wt_us": f.ExclWt, "percent": f.PExclWt}
			}
			if memHotspot == nil || f.ExclMu > bestMu {
				bestMu = f.ExclMu
				memHotspot = map[string]any{"function": f.Callee, "excl_mu_bytes": f.ExclMu, "percent": f.PExclMu}
			}
			if mostCalled == nil || f.Ct > bestCt {
				bestCt = f.Ct
				mostCalled = map[string]any{"function": f.Callee, "calls": f.Ct}
			}
		}

		result := map[string]any{
			"profile_name": name,
			"totals": map[string]any{
				"cpu_us":          peaks.CPU,
				"wall_time_us":    peaks.WallTime,
				"memory_bytes":    peaks.Memory,
				"peak_memory_bytes": peaks.PeakMem,
				"calls":           peaks.Calls,
			},
			"slowest_function": slowest,
			"memory_hotspot":   memHotspot,
			"most_called":      mostCalled,
			"total_functions":  len(funcs),
		}

		data, _ := json.MarshalIndent(result, "", "  ")
		return &sdkmcp.CallToolResult{
			Content: []sdkmcp.Content{&sdkmcp.TextContent{Text: string(data)}},
		}, nil, nil
	})

	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        "profiler_top",
		Description: "Get top functions from a profiling session sorted by a metric. Returns both inclusive (total) and exclusive (self) metrics with percentages. Useful for finding CPU bottlenecks, memory hogs, or frequently called functions.",
	}, func(ctx context.Context, req *sdkmcp.CallToolRequest, input profilerTopInput) (*sdkmcp.CallToolResult, any, error) {
		edges, err := profiler.LoadAllEdges(db, input.UUID)
		if err != nil {
			return nil, nil, fmt.Errorf("profile not found: %s", input.UUID)
		}

		metric := input.Metric
		if metric == "" {
			metric = "cpu"
		}

		limit := input.Limit
		if limit <= 0 {
			limit = 50
		}
		if limit < 5 {
			limit = 5
		}
		if limit > 200 {
			limit = 200
		}

		peaks, _, _ := profiler.LoadPeaks(db, input.UUID)
		funcs := profiler.AggregateFunctions(edges, peaks)
		profiler.SortFunctions(funcs, metric)

		if len(funcs) > limit {
			funcs = funcs[:limit]
		}

		functions := make([]map[string]any, len(funcs))
		for i, f := range funcs {
			functions[i] = f.ToMap()
		}

		result := map[string]any{
			"sorted_by": metric,
			"count":     len(functions),
			"totals": map[string]any{
				"cpu_us":          peaks.CPU,
				"wall_time_us":    peaks.WallTime,
				"memory_bytes":    peaks.Memory,
				"peak_memory_bytes": peaks.PeakMem,
				"calls":           peaks.Calls,
			},
			"functions": functions,
		}

		data, _ := json.MarshalIndent(result, "", "  ")
		return &sdkmcp.CallToolResult{
			Content: []sdkmcp.Content{&sdkmcp.TextContent{Text: string(data)}},
		}, nil, nil
	})

	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        "profiler_call_graph",
		Description: "Get a filtered call graph showing function relationships. Use threshold and percentage to control which nodes are shown — this filters out noise and shows only significant call paths. Lower percentage = more nodes visible. Ideal for understanding why a specific function is slow.",
	}, func(ctx context.Context, req *sdkmcp.CallToolRequest, input profilerCallGraphInput) (*sdkmcp.CallToolResult, any, error) {
		edges, err := profiler.LoadAllEdges(db, input.UUID)
		if err != nil {
			return nil, nil, fmt.Errorf("profile not found: %s", input.UUID)
		}

		metric := input.Metric
		if metric == "" {
			metric = "cpu"
		}

		threshold := input.Threshold
		percentage := input.Percentage
		if percentage == 0 {
			percentage = 1
		}

		// Build nodes (one per unique callee).
		nodeMap := make(map[string]*profiler.EdgeRow)
		for i := range edges {
			e := &edges[i]
			if _, ok := nodeMap[e.Callee]; !ok {
				nodeMap[e.Callee] = e
			}
		}

		type graphNode struct {
			Function string  `json:"function"`
			CPU      int64   `json:"cpu_us"`
			WallTime int64   `json:"wt_us"`
			Memory   int64   `json:"mu_bytes"`
			PeakMem  int64   `json:"pmu_bytes"`
			Calls    int64   `json:"calls"`
			Percent  float64 `json:"percent"`
		}

		type graphEdge struct {
			From    string  `json:"from"`
			To      string  `json:"to"`
			Percent float64 `json:"percent"`
		}

		registeredCallees := make(map[string]bool)
		var nodes []graphNode

		for _, e := range nodeMap {
			pct := profiler.GetEdgePercent(e, metric)
			isImportant := pct >= percentage
			isSatisfied := pct <= threshold
			if !isImportant && isSatisfied {
				continue
			}

			registeredCallees[e.Callee] = true
			nodes = append(nodes, graphNode{
				Function: e.Callee,
				CPU:      e.Cost.CPU,
				WallTime: e.Cost.WallTime,
				Memory:   e.Cost.Memory,
				PeakMem:  e.Cost.PeakMem,
				Calls:    e.Cost.Calls,
				Percent:  pct,
			})
		}

		var graphEdges []graphEdge
		for _, e := range edges {
			if e.Caller == nil {
				continue
			}
			if !registeredCallees[*e.Caller] || !registeredCallees[e.Callee] {
				continue
			}
			pct := profiler.GetEdgePercent(&e, metric)
			graphEdges = append(graphEdges, graphEdge{
				From:    *e.Caller,
				To:      e.Callee,
				Percent: pct,
			})
		}

		result := map[string]any{
			"metric":     metric,
			"threshold":  threshold,
			"percentage": percentage,
			"nodes":      nodes,
			"edges":      graphEdges,
			"node_count": len(nodes),
			"edge_count": len(graphEdges),
		}

		data, _ := json.MarshalIndent(result, "", "  ")
		return &sdkmcp.CallToolResult{
			Content: []sdkmcp.Content{&sdkmcp.TextContent{Text: string(data)}},
		}, nil, nil
	})
}
