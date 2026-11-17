package profiler

import (
	"fmt"
	"math"
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
	got, _, truncated := buildFlameChart(edges, "wt", 10000)
	if truncated {
		t.Error("small tree reported as truncated")
	}

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

	got, nodes, truncated := buildFlameChart(edges, "wt", 1000)
	if truncated {
		t.Error("lattice reported as truncated")
	}

	total, depth := countFlameNodes(got)
	if total != nodes {
		t.Errorf("reported nodes = %d, counted %d", nodes, total)
	}
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
	tree, _, _ := buildFlameChart(edges, "wt", 100)
	total, _ := countFlameNodes(tree)
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
		tree, _, truncated := buildFlameChart(edges, "wt", 1)
		total, _ := countFlameNodes(tree)
		if !truncated {
			t.Error("truncated = false, want true")
		}
		if total != maxFlameNodes {
			t.Errorf("nodes = %d, want %d", total, maxFlameNodes)
		}
	})

	t.Run("depth", func(t *testing.T) {
		edges := []EdgeRow{flameEdge("", "f0", 1)}
		for i := 0; i < maxFlameDepth+100; i++ {
			edges = append(edges, flameEdge(fmt.Sprintf("f%d", i), fmt.Sprintf("f%d", i+1), 1))
		}
		tree, _, truncated := buildFlameChart(edges, "wt", 1)
		_, depth := countFlameNodes(tree)
		if !truncated {
			t.Error("truncated = false, want true")
		}
		if depth != maxFlameDepth+1 {
			t.Errorf("depth = %d, want %d", depth, maxFlameDepth+1)
		}
	})
}

// checkInsideParents fails when a child starts before its parent, ends past
// it, or overlaps the sibling before it.
func checkInsideParents(t *testing.T, tree []*flameNode) {
	t.Helper()
	const eps = 1e-9
	var walk func(n *flameNode)
	walk = func(n *flameNode) {
		end := n.Start
		for _, c := range n.Children {
			if c.Duration < 0 || c.Start < end-eps || c.Start+c.Duration > n.Start+n.Duration+eps {
				t.Errorf("child %s [%v,%v] outside parent %s [%v,%v] or over its sibling",
					c.Name, c.Start, c.Start+c.Duration, n.Name, n.Start, n.Start+n.Duration)
			}
			end = c.Start + c.Duration
			walk(c)
		}
	}
	for _, r := range tree {
		walk(r)
	}
}

func findFlame(nodes []*flameNode, path ...string) *flameNode {
	var n *flameNode
	for _, name := range path {
		n = nil
		for _, c := range nodes {
			if c.Name == name {
				n = c
				break
			}
		}
		if n == nil {
			return nil
		}
		nodes = n.Children
	}
	return n
}

// #446: x is called from a (10) and b (70); x==>y (65) sums both. y used to
// be drawn 65 wide under a 10-wide x, over b's subtree.
func TestBuildFlameChart_MultiCallerChildrenFitParent(t *testing.T) {
	edges := []EdgeRow{
		flameEdge("", "main()", 100),
		flameEdge("main()", "a", 20),
		flameEdge("main()", "b", 80),
		flameEdge("a", "x", 10),
		flameEdge("b", "z", 5),
		flameEdge("b", "x", 70),
		flameEdge("x", "y", 65),
		flameEdge("z", "w", 4),
	}
	tree, _, _ := buildFlameChart(edges, "wt", 100)
	checkInsideParents(t, tree)

	// x is expanded under its heaviest caller b; under a it is a leaf.
	if n := findFlame(tree, "main()", "a", "x"); n == nil || len(n.Children) != 0 {
		t.Errorf("a/x = %+v, want a leaf", n)
	}
	y := findFlame(tree, "main()", "b", "x", "y")
	if y == nil {
		t.Fatal("b/x/y missing")
	}
	// y gets x's share under b: 65 * 70/80.
	if want := 65.0 * 70 / 80 / 1000; math.Abs(y.Duration-want) > 1e-12 {
		t.Errorf("y width = %v, want %v", y.Duration, want)
	}
}

func TestBuildFlameChart_RecursionFitsParents(t *testing.T) {
	// Smarty-like: render@N recursion and a helper called at every level.
	edges := []EdgeRow{flameEdge("", "main()", 1000), flameEdge("main()", "render", 900)}
	prev := "render"
	for i := 1; i <= 8; i++ {
		cur := fmt.Sprintf("render@%d", i)
		edges = append(edges, flameEdge(prev, cur, int64(900-100*i)), flameEdge(prev, "escape", 20))
		prev = cur
	}
	edges = append(edges, flameEdge("escape", "htmlspecialchars", 150))
	tree, _, _ := buildFlameChart(edges, "wt", 1000)
	checkInsideParents(t, tree)
}

func TestBuildFlameChart_NegativeMemoryClamped(t *testing.T) {
	e := func(caller, callee string, mu int64) EdgeRow {
		r := flameEdge(caller, callee, 0)
		r.Cost.Memory = mu
		return r
	}
	edges := []EdgeRow{e("", "main()", 1000), e("main()", "a", -300), e("main()", "b", 400), e("a", "c", 50)}
	tree, _, _ := buildFlameChart(edges, "mu", 1000)
	checkInsideParents(t, tree)
	if a := findFlame(tree, "main()", "a"); a == nil || a.Duration != 0 {
		t.Errorf("a = %+v, want width 0", a)
	}
	if b := findFlame(tree, "main()", "b"); b == nil || b.Start != 0 {
		t.Errorf("b = %+v, want start 0 after a zero-width a", b)
	}
	if n := findFlame(tree, "main()", "a"); n != nil && n.Cost["mu"] != int64(-300) {
		t.Errorf("tooltip cost mu = %v, want the raw -300", n.Cost["mu"])
	}
}
