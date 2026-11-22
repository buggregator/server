package profiler

import (
	"container/heap"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"net/http"
	"strconv"
	"strings"
)

// registerAPI adds the /api/profiler routes; protect is the API auth middleware.
func registerAPI(mux *http.ServeMux, db *sql.DB, protect func(http.Handler) http.Handler) {
	route := func(pattern string, h http.HandlerFunc) { mux.Handle(pattern, protect(h)) }
	route("GET /api/profiler/{uuid}/summary", handleSummary(db))
	route("GET /api/profiler/{uuid}/call-graph", handleCallGraph(db))
	route("GET /api/profiler/{uuid}/top", handleTopFunctions(db))
	route("GET /api/profiler/{uuid}/flame-chart", handleFlameChart(db))
	route("GET /api/profiler/compare", handleCompare(db))
}

// --- Edge loading (delegated to query.go) ---

// --- Summary ---

func handleSummary(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		uuid := r.PathValue("uuid")
		edges, err := LoadAllEdges(db, uuid)
		if err != nil {
			http.Error(w, "not found", 404)
			return
		}

		peaks, _, _ := LoadPeaks(db, uuid)

		// Aggregate functions (sum by callee)
		funcs := AggregateFunctions(edges, peaks)

		var slowest, memHotspot, mostCalled map[string]any

		for _, f := range funcs {
			if f.Callee == "main()" {
				continue
			}
			if slowest == nil || f.ExclWt > slowest["excl_wt"].(int64) {
				slowest = map[string]any{"function": f.Callee, "excl_wt": f.ExclWt, "p_excl_wt": round1(f.PExclWt)}
			}
			if memHotspot == nil || f.ExclMu > memHotspot["excl_mu"].(int64) {
				memHotspot = map[string]any{"function": f.Callee, "excl_mu": f.ExclMu, "p_excl_mu": round1(f.PExclMu)}
			}
			if mostCalled == nil || f.Ct > mostCalled["ct"].(int64) {
				mostCalled = map[string]any{"function": f.Callee, "ct": f.Ct}
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

		edges, err := LoadAllEdges(db, uuid)
		if err != nil {
			http.Error(w, "not found", 404)
			return
		}

		peaks, _, _ := LoadPeaks(db, uuid)
		funcs := AggregateFunctions(edges, peaks)

		// Sort by metric (default: cpu).
		metric := r.URL.Query().Get("metric")
		if metric == "" {
			metric = "cpu"
		}
		// order=asc sorts the whole profile before the cut: the lowest N, not
		// the top N reversed.
		SortFunctionsOrder(funcs, metric, r.URL.Query().Get("order") == "asc")

		total := len(funcs)
		if len(funcs) > limit {
			funcs = funcs[:limit]
		}

		// Build response.
		functions := make([]map[string]any, len(funcs))
		for i, f := range funcs {
			functions[i] = f.ToMap()
		}

		writeJSON(w, map[string]any{
			"schema":         topFunctionsSchema(),
			"overall_totals": map[string]any{"cpu": peaks.CPU, "wt": peaks.WallTime, "mu": peaks.Memory, "pmu": peaks.PeakMem, "ct": peaks.Calls},
			"functions":      functions,
			"total":          total,
		})
	}
}

// --- Call Graph ---

func handleCallGraph(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		uuid := r.PathValue("uuid")
		edges, err := LoadAllEdges(db, uuid)
		if err != nil {
			http.Error(w, "not found", 404)
			return
		}

		metric := r.URL.Query().Get("metric")
		if metric == "" {
			metric = "cpu"
		}

		// Parse threshold and percentage params.
		threshold := parseFloat(r.URL.Query().Get("threshold"), 0)   // 0-100, default 0
		percentage := parseFloat(r.URL.Query().Get("percentage"), 10) // 0-100, default 10

		// Build nodes (one per unique callee).
		nodeMap := make(map[string]*EdgeRow)
		for i := range edges {
			e := &edges[i]
			if _, ok := nodeMap[e.Callee]; !ok {
				nodeMap[e.Callee] = e
			}
		}

		// Filter and build nodes.
		registeredUUIDs := make(map[string]bool) // track included nodes for edge filtering
		nodes := make([]map[string]any, 0, len(nodeMap))
		for _, e := range nodeMap {
			pct := GetEdgePercent(e, metric)

			// Filtering logic (matching PHP):
			// Exclude if: NOT important (pct < percentage) AND satisfied (pct <= threshold)
			isImportant := pct >= percentage
			isSatisfied := pct <= threshold
			if !isImportant && isSatisfied {
				continue // skip this node
			}

			// Color: only colored if important (pct >= percentage), otherwise white.
			nodeColor := "#FFFFFF"
			if isImportant {
				nodeColor = nodeColorByPercent(pct)
			}
			txtColor := textColorByLuminance(nodeColor)

			name := e.Callee
			if e.Cost.Calls > 0 {
				name = fmt.Sprintf("%s (%dx)", e.Callee, e.Cost.Calls)
			}

			registeredUUIDs[e.UUID] = true
			nodes = append(nodes, map[string]any{
				"data": map[string]any{
					"id":   e.UUID,
					"name": name,
					"cost": map[string]any{
						"cpu": e.Cost.CPU, "wt": e.Cost.WallTime, "mu": e.Cost.Memory, "pmu": e.Cost.PeakMem, "ct": e.Cost.Calls,
					},
					"metrics":   map[string]any{"cost": GetEdgeMetricValue(e, metric), "percents": pct},
					"color":     nodeColor,
					"textColor": txtColor,
				},
			})
		}

		// Build edges — only between included nodes.
		graphEdges := make([]map[string]any, 0)
		for _, e := range edges {
			if e.Caller == nil {
				continue
			}
			parentEdge := nodeMap[*e.Caller]
			targetEdge := nodeMap[e.Callee]
			if parentEdge == nil || targetEdge == nil {
				continue
			}
			if !registeredUUIDs[parentEdge.UUID] || !registeredUUIDs[targetEdge.UUID] {
				continue
			}

			pct := GetEdgePercent(&e, metric)
			isImportant := pct >= percentage
			edgeColor := "#FFFFFF"
			if isImportant {
				edgeColor = nodeColorByPercent(pct)
			}

			graphEdges = append(graphEdges, map[string]any{
				"data": map[string]any{
					"source": parentEdge.UUID,
					"target": targetEdge.UUID,
					"label":  fmt.Sprintf("%.2f%%", pct),
					"color":  edgeColor,
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

// Hard limits for the flame chart tree. The tree already has at most one node
// per edge, these only guard against a pathological profile.
const (
	maxFlameNodes = 50000
	maxFlameDepth = 512
)

type flameNode struct {
	Name     string         `json:"name"`
	Start    float64        `json:"start"`
	Duration float64        `json:"duration"`
	Type     string         `json:"type"`
	Children []*flameNode   `json:"children"`
	Cost     map[string]any `json:"cost"`
	Color    string         `json:"color"`
}

func handleFlameChart(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		uuid := r.PathValue("uuid")
		edges, err := LoadAllEdges(db, uuid)
		if err != nil {
			http.Error(w, "not found", 404)
			return
		}

		metric := r.URL.Query().Get("metric")
		if metric == "" {
			metric = "wt"
		}

		peaks, _, _ := LoadPeaks(db, uuid)
		tree, nodes, truncated := buildFlameChart(edges, metric, float64(peakMetricValue(peaks, metric)))
		if truncated {
			slog.Warn("profiler: flame chart truncated at node/depth cap",
				"uuid", uuid, "nodes", nodes, "edges", len(edges),
				"max_nodes", maxFlameNodes, "max_depth", maxFlameDepth)
		}
		writeJSON(w, tree)
	}
}

// buildFlameChart turns caller→callee edges into a flame chart tree.
//
// XHProf only records caller→callee pairs, not full call stacks, so a function
// reached from many callers would be copied with its whole subtree once per
// path — exponential on real profiles (#356). Instead each function is
// expanded once, under its heaviest incoming edge among those reachable from
// the roots; its other call sites are shown as leaves. Every edge becomes at
// most one node.
//
// The callee==>* edges of a function sum all its calls, so under one call
// site its children are scaled by that site's share of the function's
// incoming total: children are never wider than their parent (#446).
// It returns the tree, its node count, and whether a cap cut it short.
func buildFlameChart(edges []EdgeRow, metric string, peakVal float64) (tree []*flameNode, count int, truncated bool) {
	divisor := metricDivisor(metric)

	// value is the edge's metric, never below 0 (memory deltas can be).
	value := func(i int) float64 {
		return math.Max(0, float64(GetEdgeMetricValue(&edges[i], metric)))
	}

	childrenMap := make(map[string][]int) // caller → edge indexes
	incoming := make(map[string]float64)  // callee → Σ value of edges into it
	var roots []int
	for i := range edges {
		incoming[edges[i].Callee] += value(i)
		if edges[i].Caller == nil {
			roots = append(roots, i)
		} else {
			childrenMap[*edges[i].Caller] = append(childrenMap[*edges[i].Caller], i)
		}
	}

	// Heaviest reachable edge first: it expands its callee. Ties keep the
	// order edges were reached in.
	expandedBy := make(map[string]int)
	h := &edgeHeap{value: value}
	for _, i := range roots {
		h.add(i)
	}
	for h.Len() > 0 {
		cur := heap.Pop(h).(heapItem).edge
		callee := edges[cur].Callee
		if _, ok := expandedBy[callee]; ok {
			continue
		}
		expandedBy[callee] = cur
		for _, i := range childrenMap[callee] {
			if _, ok := expandedBy[edges[i].Callee]; !ok {
				h.add(i)
			}
		}
	}

	// buildNode draws edge i at value(i)*scale.
	var buildNode func(i int, start, scale float64, depth int) *flameNode
	buildNode = func(i int, start, scale float64, depth int) *flameNode {
		count++
		e := &edges[i]
		width := value(i) * scale
		pct := 0.0
		if peakVal > 0 {
			pct = float64(GetEdgeMetricValue(e, metric)) / peakVal * 100
		}

		node := &flameNode{
			Name:     e.Callee,
			Start:    start,
			Duration: width / divisor,
			Type:     "task",
			Children: []*flameNode{},
			Cost: map[string]any{
				"cpu": e.Cost.CPU, "wt": e.Cost.WallTime, "mu": e.Cost.Memory, "pmu": e.Cost.PeakMem, "ct": e.Cost.Calls,
			},
			Color: flameColor(pct),
		}

		if expandedBy[e.Callee] != i {
			return node
		}
		kids := childrenMap[e.Callee]
		if depth >= maxFlameDepth {
			truncated = truncated || len(kids) > 0
			return node
		}
		// This call site's share of all calls of the function; then never
		// more than the parent in total (data noise).
		childScale := 0.0
		if in := incoming[e.Callee]; in > 0 {
			childScale = width / in
		}
		sum := 0.0
		for _, c := range kids {
			sum += value(c) * childScale
		}
		if sum > width && sum > 0 {
			childScale *= width / sum
		}
		childStart := start
		for _, c := range kids {
			if count >= maxFlameNodes {
				truncated = true
				break
			}
			child := buildNode(c, childStart, childScale, depth+1)
			node.Children = append(node.Children, child)
			childStart += child.Duration
		}
		return node
	}

	tree = make([]*flameNode, 0, len(roots))
	start := 0.0
	for _, i := range roots {
		if count >= maxFlameNodes {
			truncated = true
			break
		}
		node := buildNode(i, start, 1, 0)
		tree = append(tree, node)
		start += node.Duration
	}
	return tree, count, truncated
}

// edgeHeap pops the edge with the largest value; equal values in push order.
type heapItem struct {
	edge int
	val  float64
	seq  int
}

type edgeHeap struct {
	items []heapItem
	value func(int) float64
	seq   int
}

func (h *edgeHeap) add(i int) {
	h.seq++
	heap.Push(h, heapItem{edge: i, val: h.value(i), seq: h.seq})
}

func (h *edgeHeap) Len() int { return len(h.items) }
func (h *edgeHeap) Less(a, b int) bool {
	if h.items[a].val != h.items[b].val {
		return h.items[a].val > h.items[b].val
	}
	return h.items[a].seq < h.items[b].seq
}
func (h *edgeHeap) Swap(a, b int) { h.items[a], h.items[b] = h.items[b], h.items[a] }
func (h *edgeHeap) Push(x any)    { h.items = append(h.items, x.(heapItem)) }
func (h *edgeHeap) Pop() any {
	n := len(h.items)
	it := h.items[n-1]
	h.items = h.items[:n-1]
	return it
}

// --- Helpers ---

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

// nodeColorByPercent returns the 10-step red gradient matching PHP Node::detectNodeColor.
func nodeColorByPercent(pct float64) string {
	switch {
	case pct <= 10:
		return "#FFFFFF"
	case pct <= 20:
		return "#f19797"
	case pct <= 30:
		return "#d93939"
	case pct <= 40:
		return "#ad1e1e"
	case pct <= 50:
		return "#982525"
	case pct <= 60:
		return "#862323"
	case pct <= 70:
		return "#671d1d"
	case pct <= 80:
		return "#540d0d"
	case pct <= 90:
		return "#340707"
	default:
		return "#000000"
	}
}

// textColorByLuminance calculates contrast text color using WCAG luminance.
func textColorByLuminance(hexColor string) string {
	hex := strings.TrimPrefix(hexColor, "#")
	if len(hex) != 6 {
		return "#000000"
	}
	r, _ := strconv.ParseInt(hex[0:2], 16, 64)
	g, _ := strconv.ParseInt(hex[2:4], 16, 64)
	b, _ := strconv.ParseInt(hex[4:6], 16, 64)
	brightness := (r*299 + g*587 + b*114) / 1000
	if brightness > 125 {
		return "#000000"
	}
	return "#FFFFFF"
}

func parseFloat(s string, def float64) float64 {
	if s == "" {
		return def
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return def
	}
	return v
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
