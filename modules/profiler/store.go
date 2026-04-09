package profiler

import (
	"database/sql"

	"github.com/buggregator/go-buggregator/internal/event"
)

func storeProfile(db *sql.DB, uuid, name string, peaks Metrics, edges map[string]Edge) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec(
		`INSERT INTO profiles (uuid, name, cpu, wt, ct, mu, pmu) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		uuid, name, peaks.CPU, peaks.WallTime, peaks.Calls, peaks.Memory, peaks.PeakMem,
	)
	if err != nil {
		return err
	}

	stmt, err := tx.Prepare(`INSERT INTO profile_edges
		(uuid, profile_uuid, "order", cpu, wt, ct, mu, pmu, d_cpu, d_wt, d_ct, d_mu, d_pmu, p_cpu, p_wt, p_ct, p_mu, p_pmu, callee, caller, parent_uuid)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	order := 0
	for _, edge := range edges {
		edgeUUID := event.GenerateUUID()
		_, err = stmt.Exec(
			edgeUUID, uuid, order,
			edge.Cost.CPU, edge.Cost.WallTime, edge.Cost.Calls, edge.Cost.Memory, edge.Cost.PeakMem,
			edge.Diff.CPU, edge.Diff.WallTime, edge.Diff.Calls, edge.Diff.Memory, edge.Diff.PeakMem,
			edge.Percents.CPU, edge.Percents.WallTime, edge.Percents.Calls, edge.Percents.Memory, edge.Percents.PeakMem,
			edge.Callee, edge.Caller, edge.Parent,
		)
		if err != nil {
			return err
		}
		order++
	}

	return tx.Commit()
}
