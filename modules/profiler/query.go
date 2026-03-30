package profiler

import (
	"database/sql"
	"sort"
)

// EdgeRow represents a processed caller→callee relationship from the database.
type EdgeRow struct {
	UUID       string
	Callee     string
	Caller     *string
	ParentUUID *string
	Cost       Metrics
	Diff       Diffs
	Percents   Percentages
}

// LoadAllEdges loads all edges for a profile from the database.
func LoadAllEdges(db *sql.DB, profileUUID string) ([]EdgeRow, error) {
	rows, err := db.Query(`SELECT uuid, callee, caller, parent_uuid,
		cpu, wt, ct, mu, pmu,
		d_cpu, d_wt, d_ct, d_mu, d_pmu,
		p_cpu, p_wt, p_ct, p_mu, p_pmu
		FROM profile_edges WHERE profile_uuid = ? ORDER BY "order" ASC`, profileUUID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var edges []EdgeRow
	for rows.Next() {
		var e EdgeRow
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

// LoadPeaks loads the peak metrics and name for a profile.
func LoadPeaks(db *sql.DB, uuid string) (Metrics, string, error) {
	var peaks Metrics
	var name string
	err := db.QueryRow(`SELECT name, cpu, wt, ct, mu, pmu FROM profiles WHERE uuid = ?`, uuid).
		Scan(&name, &peaks.CPU, &peaks.WallTime, &peaks.Calls, &peaks.Memory, &peaks.PeakMem)
	return peaks, name, err
}

// FuncStats holds aggregated statistics for a function.
type FuncStats struct {
	Callee                                                string
	Ct, CPU, Wt, Mu, Pmu                                 int64
	ExclCPU, ExclWt, ExclMu, ExclPmu, ExclCt             int64
	PCPU, PWt, PMu, PPmu                                  float64
	PExclCPU, PExclWt, PExclMu, PExclPmu                  float64
}

// ToMap converts FuncStats to a map representation.
func (f *FuncStats) ToMap() map[string]any {
	return map[string]any{
		"function": f.Callee,
		"ct": f.Ct, "cpu": f.CPU, "wt": f.Wt, "mu": f.Mu, "pmu": f.Pmu,
		"excl_cpu": f.ExclCPU, "excl_wt": f.ExclWt, "excl_mu": f.ExclMu, "excl_pmu": f.ExclPmu, "excl_ct": f.ExclCt,
		"p_cpu": f.PCPU, "p_wt": f.PWt, "p_mu": f.PMu, "p_pmu": f.PPmu,
		"p_excl_cpu": f.PExclCPU, "p_excl_wt": f.PExclWt, "p_excl_mu": f.PExclMu, "p_excl_pmu": f.PExclPmu,
	}
}

// AggregateFunctions aggregates edge data into per-function statistics.
func AggregateFunctions(edges []EdgeRow, peaks Metrics) []FuncStats {
	agg := make(map[string]*FuncStats)
	for _, e := range edges {
		f, ok := agg[e.Callee]
		if !ok {
			f = &FuncStats{Callee: e.Callee}
			agg[e.Callee] = f
		}
		f.Ct += e.Cost.Calls
		f.CPU += e.Cost.CPU
		f.Wt += e.Cost.WallTime
		f.Mu += e.Cost.Memory
		f.Pmu += e.Cost.PeakMem
		f.ExclCPU += e.Diff.CPU
		f.ExclWt += e.Diff.WallTime
		f.ExclMu += e.Diff.Memory
		f.ExclPmu += e.Diff.PeakMem
		f.ExclCt += e.Diff.Calls
	}

	result := make([]FuncStats, 0, len(agg))
	for _, f := range agg {
		f.PCPU = pctF(f.CPU, peaks.CPU)
		f.PWt = pctF(f.Wt, peaks.WallTime)
		f.PMu = pctF(f.Mu, peaks.Memory)
		f.PPmu = pctF(f.Pmu, peaks.PeakMem)
		f.PExclCPU = pctF(f.ExclCPU, peaks.CPU)
		f.PExclWt = pctF(f.ExclWt, peaks.WallTime)
		f.PExclMu = pctF(f.ExclMu, peaks.Memory)
		f.PExclPmu = pctF(f.ExclPmu, peaks.PeakMem)
		result = append(result, *f)
	}
	return result
}

// SortFunctions sorts function stats by the given metric in descending order.
func SortFunctions(funcs []FuncStats, metric string) {
	sort.Slice(funcs, func(i, j int) bool {
		return GetFuncMetric(&funcs[i], metric) > GetFuncMetric(&funcs[j], metric)
	})
}

// GetFuncMetric returns the value of the specified metric for a function.
func GetFuncMetric(f *FuncStats, metric string) int64 {
	switch metric {
	case "cpu":
		return f.CPU
	case "wt":
		return f.Wt
	case "mu":
		return f.Mu
	case "pmu":
		return f.Pmu
	case "ct":
		return f.Ct
	case "excl_cpu":
		return f.ExclCPU
	case "excl_wt":
		return f.ExclWt
	case "excl_mu":
		return f.ExclMu
	case "excl_pmu":
		return f.ExclPmu
	case "excl_ct":
		return f.ExclCt
	default:
		return f.CPU
	}
}

// GetEdgePercent returns the percentage value for the given metric on an edge.
func GetEdgePercent(e *EdgeRow, metric string) float64 {
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

// GetEdgeMetricValue returns the cost value for the given metric on an edge.
func GetEdgeMetricValue(e *EdgeRow, metric string) int64 {
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

