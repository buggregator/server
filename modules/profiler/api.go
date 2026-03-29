package profiler

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"sort"
	"strconv"

	"github.com/buggregator/go-buggregator/internal/event"
)

func registerAPI(mux *http.ServeMux, db *sql.DB, store event.Store) {
	mux.HandleFunc("GET /api/profiler/{uuid}/summary", handleSummary(db))
	mux.HandleFunc("GET /api/profiler/{uuid}/call-graph", handleCallGraph(db))
	mux.HandleFunc("GET /api/profiler/{uuid}/top", handleTopFunctions(db))
	mux.HandleFunc("GET /api/profiler/{uuid}/flame-chart", handleFlameChart(db))
}

// --- Edge loading ---

type edgeRow struct {
	UUID       string
	Callee     string
	Caller     *string
	ParentUUID *string
	Cost       Metrics
	Diff       Diffs
	Percents   Percentages
}

func loadAllEdges(db *sql.DB, profileUUID string) ([]edgeRow, error) {
	rows, err := db.Query(`SELECT uuid, callee, caller, parent_uuid,
		cpu, wt, ct, mu, pmu,
		d_cpu, d_wt, d_ct, d_mu, d_pmu,
		p_cpu, p_wt, p_ct, p_mu, p_pmu
		FROM profile_edges WHERE profile_uuid = ? ORDER BY "order" ASC`, profileUUID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var edges []edgeRow
	for rows.Next() {
		var e edgeRow
		err := rows.Scan(&e.UUID, &e.Callee, &e.Caller, &e.ParentUUID,
			&e.Cost.CPU, &e.Cost.WallTime, &e.Cost.Calls, &e.Cost.Memory, &e.Cost.PeakMem,
			&e.Diff.CPU, &e.Diff.WallTime, &e.Diff.Calls, &e.Diff.Memory, &e.Diff.PeakMem,
			&e.Percents.CPU, &e.Percents.WallTime, &e.Percents.Calls, &e.Percents.Memory, &e.Percents.PeakMem,
		)
		if err != nil {
			return nil, err
		}
		edges = append(edges, e)
	}
	return edges, rows.Err()
}

func loadPeaks(db *sql.DB, uuid string) (Metrics, string, error) {
	var peaks Metrics
	var name string
	err := db.QueryRow(`SELECT name, cpu, wt, ct, mu, pmu FROM profiles WHERE uuid = ?`, uuid).
		Scan(&name, &peaks.CPU, &peaks.WallTime, &peaks.Calls, &peaks.Memory, &peaks.PeakMem)
	return peaks, name, err
}

// --- Summary ---

func handleSummary(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		uuid := r.PathValue("uuid")
		edges, err := loadAllEdges(db, uuid)
		if err != nil {
			http.Error(w, "not found", 404)
			return
		}

		peaks, _, _ := loadPeaks(db, uuid)

		// Aggregate functions (sum by callee)
		funcs := aggregateFunctions(edges, peaks)

		var slowest, memHotspot, mostCalled map[string]any

		for _, f := range funcs {
			if f.callee == "main()" {
				continue
			}
			if slowest == nil || f.exclWt > slowest["excl_wt"].(int64) {
				slowest = map[string]any{"function": f.callee, "excl_wt": f.exclWt, "p_excl_wt": round1(f.pExclWt)}
			}
			if memHotspot == nil || f.exclMu > memHotspot["excl_mu"].(int64) {
				memHotspot = map[string]any{"function": f.callee, "excl_mu": f.exclMu, "p_excl_mu": round1(f.pExclMu)}
			}
			if mostCalled == nil || f.ct > mostCalled["ct"].(int64) {
				mostCalled = map[string]any{"function": f.callee, "ct": f.ct}
			}
		}

		writeJSON(w, map[string]any{
			"overall_totals":   map[string]any{"cpu": peaks.CPU, "wt": peaks.WallTime, "mu": peaks.Memory, "pmu": peaks.PeakMem, "ct": peaks.Calls},
			"slowest_function": slowest,
			"memory_hotspot":   memHotspot,
			"most_called":      mostCalled,
		})
	}
}

// --- Top Functions ---

func handleTopFunctions(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		uuid := r.PathValue("uuid")
		limit := 100
		if l := r.URL.Query().Get("limit"); l != "" {
			if v, err := strconv.Atoi(l); err == nil && v >= 10 && v <= 300 {
				limit = v
			}
		}

		edges, err := loadAllEdges(db, uuid)
		if err != nil {
			http.Error(w, "not found", 404)
			return
		}

		peaks, _, _ := loadPeaks(db, uuid)
		funcs := aggregateFunctions(edges, peaks)

		// Sort by metric (default: cpu).
		metric := r.URL.Query().Get("metric")
		if metric == "" {
			metric = "cpu"
		}
		sortFunctions(funcs, metric)

		if len(funcs) > limit {
			funcs = funcs[:limit]
		}

		// Build response.
		functions := make([]map[string]any, len(funcs))
		for i, f := range funcs {
			functions[i] = f.toMap()
		}

		writeJSON(w, map[string]any{
			"schema":         topFunctionsSchema(),
			"overall_totals": map[string]any{"cpu": peaks.CPU, "wt": peaks.WallTime, "mu": peaks.Memory, "pmu": peaks.PeakMem, "ct": peaks.Calls},
			"functions":      functions,
		})
	}
}

// --- Call Graph ---

func handleCallGraph(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		uuid := r.PathValue("uuid")
		edges, err := loadAllEdges(db, uuid)
		if err != nil {
			http.Error(w, "not found", 404)
			return
		}

		peaks, _, _ := loadPeaks(db, uuid)
		metric := r.URL.Query().Get("metric")
		if metric == "" {
			metric = "cpu"
		}

		// Build nodes (one per unique callee) and edges.
		nodeMap := make(map[string]*edgeRow)
		for i := range edges {
			e := &edges[i]
			if _, ok := nodeMap[e.Callee]; !ok {
				nodeMap[e.Callee] = e
			}
		}

		nodes := make([]map[string]any, 0, len(nodeMap))
		for _, e := range nodeMap {
			pct := getPercent(e, metric)
			cost := metricValue(e, metric)
			peakVal := peakMetricValue(peaks, metric)

			name := e.Callee
			if e.Cost.Calls > 0 {
				name = fmt.Sprintf("%s (%dx)", e.Callee, e.Cost.Calls)
			}

			nodes = append(nodes, map[string]any{
				"data": map[string]any{
					"id":   e.UUID,
					"name": name,
					"cost": map[string]any{
						"cpu": e.Cost.CPU, "wt": e.Cost.WallTime, "mu": e.Cost.Memory, "pmu": e.Cost.PeakMem, "ct": e.Cost.Calls,
					},
					"metrics":   map[string]any{"cost": cost, "percents": pct},
					"color":     percentToColor(pct),
					"textColor": textColorFor(pct),
				},
			})
			_ = peakVal
		}

		graphEdges := make([]map[string]any, 0)
		for _, e := range edges {
			if e.Caller == nil {
				continue
			}
			parentEdge := nodeMap[*e.Caller]
			if parentEdge == nil {
				continue
			}
			pct := getPercent(&e, metric)
			graphEdges = append(graphEdges, map[string]any{
				"data": map[string]any{
					"source": parentEdge.UUID,
					"target": nodeMap[e.Callee].UUID,
					"label":  fmt.Sprintf("%.2f%%", pct),
					"color":  percentToColor(pct),
				},
			})
		}

		writeJSON(w, map[string]any{
			"toolbar": callGraphToolbar(),
			"nodes":   nodes,
			"edges":   graphEdges,
		})
	}
}

// --- Flame Chart ---

func handleFlameChart(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		uuid := r.PathValue("uuid")
		edges, err := loadAllEdges(db, uuid)
		if err != nil {
			http.Error(w, "not found", 404)
			return
		}

		metric := r.URL.Query().Get("metric")
		if metric == "" {
			metric = "wt"
		}

		peaks, _, _ := loadPeaks(db, uuid)
		peakVal := float64(peakMetricValue(peaks, metric))
		divisor := metricDivisor(metric)

		// Build tree.
		type flameNode struct {
			Name     string       `json:"name"`
			Start    float64      `json:"start"`
			Duration float64      `json:"duration"`
			Type     string       `json:"type"`
			Children []*flameNode `json:"children"`
			Cost     map[string]any `json:"cost"`
			Color    string       `json:"color"`
		}

		nodesByCallee := make(map[string]*edgeRow)
		childrenMap := make(map[string][]edgeRow) // parent callee → children
		var roots []edgeRow

		for _, e := range edges {
			if _, ok := nodesByCallee[e.Callee]; !ok {
				nodesByCallee[e.Callee] = &e
			}
			if e.Caller == nil {
				roots = append(roots, e)
			} else {
				childrenMap[*e.Caller] = append(childrenMap[*e.Caller], e)
			}
		}

		var buildNode func(e edgeRow, start float64) *flameNode
		buildNode = func(e edgeRow, start float64) *flameNode {
			val := float64(metricValue(&e, metric))
			duration := val / divisor
			pct := 0.0
			if peakVal > 0 {
				pct = val / peakVal * 100
			}

			node := &flameNode{
				Name:     e.Callee,
				Start:    start,
				Duration: duration,
				Type:     "task",
				Cost: map[string]any{
					"cpu": e.Cost.CPU, "wt": e.Cost.WallTime, "mu": e.Cost.Memory, "pmu": e.Cost.PeakMem, "ct": e.Cost.Calls,
				},
				Color: flameColor(pct),
			}

			childStart := start
			for _, child := range childrenMap[e.Callee] {
				childNode := buildNode(child, childStart)
				node.Children = append(node.Children, childNode)
				childStart += childNode.Duration
			}
			if node.Children == nil {
				node.Children = []*flameNode{}
			}

			return node
		}

		result := make([]*flameNode, 0, len(roots))
		start := 0.0
		for _, root := range roots {
			node := buildNode(root, start)
			result = append(result, node)
			start += node.Duration
		}

		writeJSON(w, result)
	}
}

// --- Helpers ---

type funcStats struct {
	callee                                                     string
	ct, cpu, wt, mu, pmu                                       int64
	exclCpu, exclWt, exclMu, exclPmu, exclCt                  int64
	pCpu, pWt, pMu, pPmu                                      float64
	pExclCpu, pExclWt, pExclMu, pExclPmu                      float64
}

func (f *funcStats) toMap() map[string]any {
	return map[string]any{
		"function": f.callee,
		"ct": f.ct, "cpu": f.cpu, "wt": f.wt, "mu": f.mu, "pmu": f.pmu,
		"excl_cpu": f.exclCpu, "excl_wt": f.exclWt, "excl_mu": f.exclMu, "excl_pmu": f.exclPmu, "excl_ct": f.exclCt,
		"p_cpu": f.pCpu, "p_wt": f.pWt, "p_mu": f.pMu, "p_pmu": f.pPmu,
		"p_excl_cpu": f.pExclCpu, "p_excl_wt": f.pExclWt, "p_excl_mu": f.pExclMu, "p_excl_pmu": f.pExclPmu,
	}
}

func aggregateFunctions(edges []edgeRow, peaks Metrics) []funcStats {
	agg := make(map[string]*funcStats)
	for _, e := range edges {
		f, ok := agg[e.Callee]
		if !ok {
			f = &funcStats{callee: e.Callee}
			agg[e.Callee] = f
		}
		f.ct += e.Cost.Calls
		f.cpu += e.Cost.CPU
		f.wt += e.Cost.WallTime
		f.mu += e.Cost.Memory
		f.pmu += e.Cost.PeakMem
		f.exclCpu += e.Diff.CPU
		f.exclWt += e.Diff.WallTime
		f.exclMu += e.Diff.Memory
		f.exclPmu += e.Diff.PeakMem
		f.exclCt += e.Diff.Calls
	}

	result := make([]funcStats, 0, len(agg))
	for _, f := range agg {
		f.pCpu = pctF(f.cpu, peaks.CPU)
		f.pWt = pctF(f.wt, peaks.WallTime)
		f.pMu = pctF(f.mu, peaks.Memory)
		f.pPmu = pctF(f.pmu, peaks.PeakMem)
		f.pExclCpu = pctF(f.exclCpu, peaks.CPU)
		f.pExclWt = pctF(f.exclWt, peaks.WallTime)
		f.pExclMu = pctF(f.exclMu, peaks.Memory)
		f.pExclPmu = pctF(f.exclPmu, peaks.PeakMem)
		result = append(result, *f)
	}
	return result
}

func sortFunctions(funcs []funcStats, metric string) {
	sort.Slice(funcs, func(i, j int) bool {
		return getFuncMetric(&funcs[i], metric) > getFuncMetric(&funcs[j], metric)
	})
}

func getFuncMetric(f *funcStats, metric string) int64 {
	switch metric {
	case "cpu":
		return f.cpu
	case "wt":
		return f.wt
	case "mu":
		return f.mu
	case "pmu":
		return f.pmu
	case "ct":
		return f.ct
	case "excl_cpu":
		return f.exclCpu
	case "excl_wt":
		return f.exclWt
	case "excl_mu":
		return f.exclMu
	case "excl_pmu":
		return f.exclPmu
	case "excl_ct":
		return f.exclCt
	default:
		return f.cpu
	}
}

func getPercent(e *edgeRow, metric string) float64 {
	switch metric {
	case "cpu":
		return e.Percents.CPU
	case "wt":
		return e.Percents.WallTime
	case "mu":
		return e.Percents.Memory
	case "pmu":
		return e.Percents.PeakMem
	default:
		return e.Percents.CPU
	}
}

func metricValue(e *edgeRow, metric string) int64 {
	switch metric {
	case "cpu":
		return e.Cost.CPU
	case "wt":
		return e.Cost.WallTime
	case "mu":
		return e.Cost.Memory
	case "pmu":
		return e.Cost.PeakMem
	case "ct":
		return e.Cost.Calls
	default:
		return e.Cost.CPU
	}
}

func peakMetricValue(p Metrics, metric string) int64 {
	switch metric {
	case "cpu":
		return p.CPU
	case "wt":
		return p.WallTime
	case "mu":
		return p.Memory
	case "pmu":
		return p.PeakMem
	case "ct":
		return p.Calls
	default:
		return p.CPU
	}
}

func metricDivisor(metric string) float64 {
	switch metric {
	case "mu", "pmu":
		return 1024.0
	default:
		return 1000.0
	}
}

func pctF(val, peak int64) float64 {
	if peak <= 0 || val <= 0 {
		return 0
	}
	return math.Round(float64(val)/float64(peak)*100*100) / 100
}

func round1(v float64) float64 {
	return math.Round(v*10) / 10
}

func percentToColor(pct float64) string {
	if pct <= 5 {
		return "#FFFFFF"
	}
	r := int(math.Min(255, pct*2.55))
	return fmt.Sprintf("#FF%02X%02X", 255-r, 255-r)
}

func textColorFor(pct float64) string {
	if pct > 50 {
		return "#FFFFFF"
	}
	return "#000000"
}

func flameColor(pct float64) string {
	switch {
	case pct <= 10:
		return "#B3E5FC"
	case pct <= 20:
		return "#81D4FA"
	case pct <= 30:
		return "#4FC3F7"
	case pct <= 40:
		return "#29B6F6"
	case pct <= 50:
		return "#FFCDD2"
	case pct <= 60:
		return "#FFB2B2"
	case pct <= 70:
		return "#FF9E9E"
	case pct <= 80:
		return "#FF8989"
	case pct <= 90:
		return "#FF7474"
	default:
		return "#FF5F5F"
	}
}

func topFunctionsSchema() []map[string]any {
	return []map[string]any{
		{"key": "function", "label": "Function", "description": "Function that was called", "sortable": false, "values": []map[string]any{{"key": "function", "format": "string"}}},
		{"key": "ct", "label": "CT", "description": "Calls", "sortable": true, "values": []map[string]any{{"key": "ct", "format": "number"}}},
		{"key": "cpu", "label": "CPU", "description": "CPU Time (ms)", "sortable": true, "values": []map[string]any{{"key": "cpu", "format": "ms"}, {"key": "p_cpu", "format": "percent", "type": "sub"}}},
		{"key": "excl_cpu", "label": "CPU excl.", "description": "CPU Time exclusions (ms)", "sortable": true, "values": []map[string]any{{"key": "excl_cpu", "format": "ms"}, {"key": "p_excl_cpu", "format": "percent", "type": "sub"}}},
		{"key": "wt", "label": "WT", "description": "Wall Time (ms)", "sortable": true, "values": []map[string]any{{"key": "wt", "format": "ms"}, {"key": "p_wt", "format": "percent", "type": "sub"}}},
		{"key": "excl_wt", "label": "WT excl.", "description": "Wall Time exclusions (ms)", "sortable": true, "values": []map[string]any{{"key": "excl_wt", "format": "ms"}, {"key": "p_excl_wt", "format": "percent", "type": "sub"}}},
		{"key": "mu", "label": "MU", "description": "Memory Usage (bytes)", "sortable": true, "values": []map[string]any{{"key": "mu", "format": "bytes"}, {"key": "p_mu", "format": "percent", "type": "sub"}}},
		{"key": "excl_mu", "label": "MU excl.", "description": "Memory Usage exclusions (bytes)", "sortable": true, "values": []map[string]any{{"key": "excl_mu", "format": "bytes"}, {"key": "p_excl_mu", "format": "percent", "type": "sub"}}},
		{"key": "pmu", "label": "PMU", "description": "Peak Memory Usage (bytes)", "sortable": true, "values": []map[string]any{{"key": "pmu", "format": "bytes"}, {"key": "p_pmu", "format": "percent", "type": "sub"}}},
		{"key": "excl_pmu", "label": "PMU excl.", "description": "Peak Memory Usage exclusions (bytes)", "sortable": true, "values": []map[string]any{{"key": "excl_pmu", "format": "bytes"}, {"key": "p_excl_pmu", "format": "percent", "type": "sub"}}},
	}
}

func callGraphToolbar() []map[string]any {
	return []map[string]any{
		{"label": "CPU", "metric": "cpu", "description": "CPU time in ms."},
		{"label": "Wall time", "metric": "wt", "description": "Wall time in ms."},
		{"label": "Memory", "metric": "mu", "description": "Memory usage in bytes."},
		{"label": "Memory peak", "metric": "pmu", "description": "Memory peak usage in bytes."},
	}
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}
