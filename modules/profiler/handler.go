package profiler

import (
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/buggregator/go-buggregator/internal/event"
)

type handler struct {
	db *sql.DB
}

func (h *handler) Priority() int { return 40 }

func (h *handler) Match(r *http.Request) bool {
	if r.Method != http.MethodPost {
		return false
	}
	return r.Header.Get("X-Profiler-Dump") != "" ||
		strings.HasSuffix(r.URL.Path, "/profiler/store")
}

func (h *handler) Handle(r *http.Request) (*event.Incoming, error) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	defer r.Body.Close()

	var incoming IncomingProfile
	if err := json.Unmarshal(body, &incoming); err != nil {
		return nil, err
	}

	// Process the profile (compute diffs, percentages, edges).
	peaks, edges := Process(&incoming)

	// Store in profiler-specific tables.
	uuid := event.GenerateUUID()
	if err := storeProfile(h.db, uuid, incoming.AppName, peaks, edges); err != nil {
		return nil, err
	}

	// Build event payload.
	payload := map[string]any{
		"app_name":    incoming.AppName,
		"hostname":    incoming.Hostname,
		"date":        incoming.Date,
		"tags":        incoming.Tags,
		"peaks":       peaks,
		"total_edges": len(edges),
	}
	b, _ := json.Marshal(payload)

	return &event.Incoming{
		UUID:    uuid,
		Type:    "profiler",
		Payload: json.RawMessage(b),
	}, nil
}

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
