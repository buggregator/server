package profiler

import (
	"sort"
	"strconv"
	"strings"
)

// How the call graph shows recursion (the "recursion" query parameter and
// response field, as in Trap, buggregator/trap#400): a node per recursion
// level (XHProf names them foo, foo@1, foo@2…), or a node per function.
const (
	RecursionLevels    = "levels"
	RecursionCollapsed = "collapsed"
)

// recursionLevel splits an XHProf function name: "foo@2" is foo at level 2,
// "foo" level 0. Only an all-digit suffix is a level.
func recursionLevel(name string) (string, int) {
	i := strings.LastIndexByte(name, '@')
	if i <= 0 || i == len(name)-1 {
		return name, 0
	}
	n, err := strconv.Atoi(name[i+1:])
	if err != nil || n < 0 || strings.ContainsAny(name[i+1:], "+-") {
		return name, 0
	}
	return name[:i], n
}

// collapseRecursion merges the recursion levels of a function into one node.
// A node costs what its outermost level costs (that takes the deeper levels
// in) and counts the calls of every level; recursive holds, by node name, the
// calls of the deeper levels. The calls between two nodes are summed over
// the levels, a call nested in an outer call of the same pair adding its
// count only, so a recursion is not counted once per level; direct recursion
// is a call of a node to itself.
func collapseRecursion(edges []EdgeRow) (nodes, calls []EdgeRow, recursive map[string]int64) {
	levelNodes, _ := graphLevels(edges)

	var order []string
	groups := make(map[string][]EdgeRow)
	for _, e := range levelNodes {
		base, _ := recursionLevel(e.Callee)
		if _, ok := groups[base]; !ok {
			order = append(order, base)
		}
		groups[base] = append(groups[base], e)
	}
	recursive = make(map[string]int64)
	for _, base := range order {
		group := groups[base]
		outer := group[0]
		_, outerLevel := recursionLevel(outer.Callee)
		var ct int64
		var pct float64
		for _, e := range group {
			if _, l := recursionLevel(e.Callee); l < outerLevel {
				outer, outerLevel = e, l
			}
			ct += e.Cost.Calls
			pct += e.Percents.Calls
		}
		if n := ct - outer.Cost.Calls; n > 0 {
			recursive[base] = n
		}
		outer.Callee = base
		outer.Cost.Calls, outer.Percents.Calls = ct, pct
		nodes = append(nodes, outer)
	}

	type pairKey struct{ caller, callee string }
	var pairs []pairKey
	byPair := make(map[pairKey][]EdgeRow)
	for _, e := range edges {
		if e.Caller == nil {
			continue
		}
		caller, _ := recursionLevel(*e.Caller)
		callee, _ := recursionLevel(e.Callee)
		k := pairKey{caller, callee}
		if _, ok := byPair[k]; !ok {
			pairs = append(pairs, k)
		}
		byPair[k] = append(byPair[k], e)
	}
	for _, k := range pairs {
		pair := byPair[k]
		nested := nestedCalls(pair)
		sum := EdgeRow{UUID: pair[0].UUID, Callee: k.callee}
		for i, e := range pair {
			sum.Cost.Calls += e.Cost.Calls
			sum.Percents.Calls += e.Percents.Calls
			if nested[i] {
				continue
			}
			sum.Cost.CPU += e.Cost.CPU
			sum.Cost.WallTime += e.Cost.WallTime
			sum.Cost.Memory += e.Cost.Memory
			sum.Cost.PeakMem += e.Cost.PeakMem
			sum.Percents.CPU += e.Percents.CPU
			sum.Percents.WallTime += e.Percents.WallTime
			sum.Percents.Memory += e.Percents.Memory
			sum.Percents.PeakMem += e.Percents.PeakMem
		}
		caller := k.caller
		sum.Caller = &caller
		calls = append(calls, sum)
	}
	return nodes, calls, recursive
}

// nestedCalls marks the calls of one caller→callee pair made inside an outer
// call of the same pair, deeper in both recursions: a call whose callee level
// is above the lowest callee level of the calls from a lower caller level.
// A call deeper in one function only (g@1→f@1 beside g→f@1) is apart.
func nestedCalls(pair []EdgeRow) map[int]bool {
	if len(pair) < 2 {
		return nil
	}
	type lv struct{ i, caller, callee int }
	ls := make([]lv, len(pair))
	for i, e := range pair {
		_, ls[i].caller = recursionLevel(*e.Caller)
		_, ls[i].callee = recursionLevel(e.Callee)
		ls[i].i = i
	}
	sort.SliceStable(ls, func(a, b int) bool { return ls[a].caller < ls[b].caller })

	nested := make(map[int]bool)
	const none = int(^uint(0) >> 1)
	min, levelMin, level := none, none, -1
	for _, l := range ls {
		if l.caller != level {
			if levelMin < min {
				min = levelMin
			}
			levelMin, level = none, l.caller
		}
		if min < l.callee {
			nested[l.i] = true
		}
		if l.callee < levelMin {
			levelMin = l.callee
		}
	}
	return nested
}
