package profiler

import (
	"database/sql"
	"errors"
	"net/http"
	"sort"
	"strconv"
)

// Compare limits: functions returned by default and at most.
const (
	compareDefaultLimit = 100
	compareMaxLimit     = 1000
)

// handleCompare compares two profiles function by function.
//
//	GET /api/profiler/compare?base=<uuid>&compare=<uuid>[&limit=N]
//
// Each function of either profile gets its exclusive wall time (µs),
// exclusive memory (bytes) and calls in both, and compare − base. A function
// missing from one profile counts 0 there. Biggest wall-time change first
// (then memory, calls, name); main() included, it carries the total change.
func handleCompare(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		baseID, compareID := q.Get("base"), q.Get("compare")
		if baseID == "" || compareID == "" {
			http.Error(w, "base and compare are required", http.StatusBadRequest)
			return
		}
		limit := compareDefaultLimit
		if v := q.Get("limit"); v != "" {
			n, err := strconv.Atoi(v)
			if err != nil || n < 1 {
				http.Error(w, "limit: a positive integer", http.StatusBadRequest)
				return
			}
			limit = min(n, compareMaxLimit)
		}

		base, baseTotals, err := compareSide(db, baseID)
		if err == nil {
			var cmp map[string]FuncStats
			var cmpTotals Metrics
			cmp, cmpTotals, err = compareSide(db, compareID)
			if err == nil {
				writeJSON(w, compareFunctions(base, cmp, baseTotals, cmpTotals, limit))
				return
			}
		}
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "profile not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// compareSide loads one profile's functions by name and its totals.
func compareSide(db *sql.DB, uuid string) (map[string]FuncStats, Metrics, error) {
	peaks, _, err := LoadPeaks(db, uuid)
	if err != nil {
		return nil, peaks, err
	}
	edges, err := LoadAllEdges(db, uuid)
	if err != nil {
		return nil, peaks, err
	}
	byName := map[string]FuncStats{}
	for _, f := range AggregateFunctions(edges, peaks) {
		byName[f.Callee] = f
	}
	return byName, peaks, nil
}

func compareFunctions(base, cmp map[string]FuncStats, baseTotals, cmpTotals Metrics, limit int) map[string]any {
	type row struct {
		name          string
		bWt, bMu, bCt int64
		cWt, cMu, cCt int64
		dWt, dMu, dCt int64
	}
	rows := make([]row, 0, len(base)+len(cmp))
	seen := map[string]bool{}
	add := func(name string) {
		if seen[name] {
			return
		}
		seen[name] = true
		b, c := base[name], cmp[name]
		rows = append(rows, row{
			name: name,
			bWt:  b.ExclWt, bMu: b.ExclMu, bCt: b.Ct,
			cWt: c.ExclWt, cMu: c.ExclMu, cCt: c.Ct,
			dWt: c.ExclWt - b.ExclWt, dMu: c.ExclMu - b.ExclMu, dCt: c.Ct - b.Ct,
		})
	}
	for name := range base {
		add(name)
	}
	for name := range cmp {
		add(name)
	}
	abs := func(v int64) int64 {
		if v < 0 {
			return -v
		}
		return v
	}
	sort.Slice(rows, func(i, j int) bool {
		a, b := rows[i], rows[j]
		switch {
		case abs(a.dWt) != abs(b.dWt):
			return abs(a.dWt) > abs(b.dWt)
		case abs(a.dMu) != abs(b.dMu):
			return abs(a.dMu) > abs(b.dMu)
		case abs(a.dCt) != abs(b.dCt):
			return abs(a.dCt) > abs(b.dCt)
		}
		return a.name < b.name
	})
	total := len(rows)
	if len(rows) > limit {
		rows = rows[:limit]
	}

	functions := make([]map[string]any, 0, len(rows))
	for _, r := range rows {
		functions = append(functions, map[string]any{
			"function":        r.name,
			"base_excl_wt":    r.bWt,
			"compare_excl_wt": r.cWt,
			"diff_excl_wt":    r.dWt,
			"base_excl_mu":    r.bMu,
			"compare_excl_mu": r.cMu,
			"diff_excl_mu":    r.dMu,
			"base_ct":         r.bCt,
			"compare_ct":      r.cCt,
			"diff_ct":         r.dCt,
		})
	}
	totals := func(m Metrics) map[string]any {
		return map[string]any{"cpu": m.CPU, "wt": m.WallTime, "mu": m.Memory, "pmu": m.PeakMem, "ct": m.Calls}
	}
	return map[string]any{
		"functions":      functions,
		"total":          total,
		"base_totals":    totals(baseTotals),
		"compare_totals": totals(cmpTotals),
	}
}
