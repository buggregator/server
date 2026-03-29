package profiler

import (
	"math"
	"strconv"
	"strings"
)

// Process takes raw XHProf data and produces processed edges with diffs/percentages.
func Process(incoming *IncomingProfile) (peaks Metrics, edges map[string]Edge) {
	profile := incoming.Profile

	// Extract peaks from main().
	if main, ok := profile["main()"]; ok {
		peaks = main
	}

	// Calculate children sum per parent.
	childrenSum := make(map[string]Metrics)
	for name, values := range profile {
		parent, _ := splitEdgeName(name)
		if parent != nil {
			sum := childrenSum[*parent]
			sum.CPU += values.CPU
			sum.WallTime += values.WallTime
			sum.Memory += values.Memory
			sum.PeakMem += values.PeakMem
			sum.Calls += values.Calls
			childrenSum[*parent] = sum
		}
	}

	// Calculate diffs (exclusive = inclusive - children).
	diffs := make(map[string]Diffs)
	for name, values := range profile {
		_, callee := splitEdgeName(name)
		children := childrenSum[callee]
		diffs[name] = Diffs{
			WallTime: max64(0, values.WallTime-children.WallTime),
			CPU:      max64(0, values.CPU-children.CPU),
			Memory:   max64(0, values.Memory-children.Memory),
			PeakMem:  max64(0, values.PeakMem-children.PeakMem),
			Calls:    max64(0, values.Calls-children.Calls),
		}
	}

	// Build edges with parent resolution.
	type tempEdge struct {
		id     string
		caller *string
		callee string
		cost   Metrics
		diff   Diffs
		parent *string
	}

	edgesTemp := make(map[string]*tempEdge)
	calleeToEdgeID := make(map[string]string)

	id := 1
	for name, values := range profile {
		parent, callee := splitEdgeName(name)
		edgeID := "e" + strconv.Itoa(id)

		edgesTemp[edgeID] = &tempEdge{
			id:     edgeID,
			caller: parent,
			callee: callee,
			cost:   values,
			diff:   diffs[name],
		}

		if _, exists := calleeToEdgeID[callee]; !exists {
			calleeToEdgeID[callee] = edgeID
		}
		id++
	}

	// Resolve parent references.
	for _, edge := range edgesTemp {
		if edge.caller != nil {
			if parentEdgeID, ok := calleeToEdgeID[*edge.caller]; ok {
				p := parentEdgeID
				edge.parent = &p
			}
		}
	}

	// BFS ordering.
	childrenMap := make(map[string][]string)
	var roots []string
	for edgeID, edge := range edgesTemp {
		if edge.parent == nil {
			roots = append(roots, edgeID)
		} else {
			childrenMap[*edge.parent] = append(childrenMap[*edge.parent], edgeID)
		}
	}

	edges = make(map[string]Edge)
	queue := append([]string{}, roots...)

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		te := edgesTemp[current]
		edges[current] = Edge{
			ID:     te.id,
			Caller: te.caller,
			Callee: te.callee,
			Cost:   te.cost,
			Diff:   te.diff,
			Percents: Percentages{
				WallTime: pct(te.cost.WallTime, peaks.WallTime),
				CPU:      pct(te.cost.CPU, peaks.CPU),
				Memory:   pct(te.cost.Memory, peaks.Memory),
				PeakMem:  pct(te.cost.PeakMem, peaks.PeakMem),
				Calls:    pct(te.cost.Calls, peaks.Calls),
			},
			Parent: te.parent,
		}

		queue = append(queue, childrenMap[current]...)
	}

	// Add orphaned edges.
	for edgeID, te := range edgesTemp {
		if _, exists := edges[edgeID]; !exists {
			edges[edgeID] = Edge{
				ID: te.id, Caller: te.caller, Callee: te.callee,
				Cost: te.cost, Diff: te.diff, Parent: te.parent,
			}
		}
	}

	return peaks, edges
}

func splitEdgeName(name string) (*string, string) {
	parts := strings.SplitN(name, "==>", 2)
	if len(parts) == 2 {
		return &parts[0], parts[1]
	}
	return nil, parts[0]
}

func max64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

func pct(value, peak int64) float64 {
	if peak <= 0 || value <= 0 {
		return 0
	}
	return math.Round(float64(value)/float64(peak)*100*1000) / 1000
}
