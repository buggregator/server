package profiler

import (
	"fmt"
	"testing"
)

func flameEdge(caller, callee string, wt int64) EdgeRow {
	e := EdgeRow{UUID: caller + "==>" + callee, Callee: callee, Cost: Metrics{WallTime: wt}}
	if caller != "" {
		e.Caller = &caller
	}
	return e
}

func countFlameNodes(nodes []*flameNode) (total, maxDepth int) {
	var walk func(n *flameNode, depth int)
	walk = func(n *flameNode, depth int) {
		total++
		if depth > maxDepth {
			maxDepth = depth
		}
		for _, c := range n.Children {
			walk(c, depth+1)
		}
	}
	for _, n := range nodes {
		walk(n, 1)
	}
	return total, maxDepth
}

func TestBuildFlameChart_Tree(t *testing.T) {
	edges := []EdgeRow{
		flameEdge("", "main()", 10000),
		flameEdge("main()", "a", 6000),
		flameEdge("main()", "b", 3000),
		flameEdge("a", "c", 2000),
	}
	got := buildFlameChart(edges, "wt", 10000)

	if len(got) != 1 || got[0].Name != "main()" || got[0].Duration != 10 {
		t.Fatalf("root = %+v", got)
	}
	main := got[0]
	if len(main.Children) != 2 {
		t.Fatalf("main children = %d, want 2", len(main.Children))
	}
	a, b := main.Children[0], main.Children[1]
	if a.Name != "a" || a.Start != 0 || b.Name != "b" || b.Start != 6 {
		t.Errorf("a=%s@%v b=%s@%v, want a@0 b@6", a.Name, a.Start, b.Name, b.Start)
	}
	if len(a.Children) != 1 || a.Children[0].Name != "c" {
		t.Errorf("a children = %+v, want [c]", a.Children)
	}
	if b.Children == nil {
		t.Error("leaf children must be an empty slice, not nil")
	}
}

// A lattice of 40 layers, two functions each, every function calling both
// functions of the next layer: 2^40 paths. The old builder copied every
// subtree once per path and ate all memory (#356).
func TestBuildFlameChart_DiamondLattice(t *testing.T) {
	const layers = 40
	name := func(layer, i int) string { return fmt.Sprintf("f%d_%d", layer, i) }

	edges := []EdgeRow{flameEdge("", "main()", 1000), flameEdge("main()", name(0, 0), 1), flameEdge("main()", name(0, 1), 1)}
	for l := 0; l < layers-1; l++ {
		for i := 0; i < 2; i++ {
			for j := 0; j < 2; j++ {
				edges = append(edges, flameEdge(name(l, i), name(l+1, j), 1))
			}
		}
	}

	got := buildFlameChart(edges, "wt", 1000)

	total, depth := countFlameNodes(got)
	if total > len(edges) {
		t.Errorf("nodes = %d, want at most one per edge (%d)", total, len(edges))
	}
	if depth != layers+1 {
		t.Errorf("depth = %d, want %d", depth, layers+1)
	}
}

func TestBuildFlameChart_Recursion(t *testing.T) {
	edges := []EdgeRow{
		flameEdge("", "main()", 100),
		flameEdge("main()", "a", 50),
		flameEdge("a", "b", 40),
		flameEdge("b", "a", 30),
		flameEdge("a", "a", 10),
	}
	total, _ := countFlameNodes(buildFlameChart(edges, "wt", 100))
	if total != len(edges) {
		t.Errorf("nodes = %d, want %d", total, len(edges))
	}
}

func TestBuildFlameChart_Limits(t *testing.T) {
	t.Run("nodes", func(t *testing.T) {
		edges := []EdgeRow{flameEdge("", "main()", 1)}
		for i := 0; i < maxFlameNodes+100; i++ {
			edges = append(edges, flameEdge("main()", fmt.Sprintf("f%d", i), 1))
		}
		total, _ := countFlameNodes(buildFlameChart(edges, "wt", 1))
		if total != maxFlameNodes {
			t.Errorf("nodes = %d, want %d", total, maxFlameNodes)
		}
	})

	t.Run("depth", func(t *testing.T) {
		edges := []EdgeRow{flameEdge("", "f0", 1)}
		for i := 0; i < maxFlameDepth+100; i++ {
			edges = append(edges, flameEdge(fmt.Sprintf("f%d", i), fmt.Sprintf("f%d", i+1), 1))
		}
		_, depth := countFlameNodes(buildFlameChart(edges, "wt", 1))
		if depth != maxFlameDepth+1 {
			t.Errorf("depth = %d, want %d", depth, maxFlameDepth+1)
		}
	})
}
