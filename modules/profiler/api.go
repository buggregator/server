package profiler

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/buggregator/go-buggregator/internal/event"
)

func registerAPI(mux *http.ServeMux, db *sql.DB, store event.Store) {
	// GET /api/profiler/{uuid}/summary
	mux.HandleFunc("GET /api/profiler/{uuid}/summary", func(w http.ResponseWriter, r *http.Request) {
		uuid := r.PathValue("uuid")
		row := db.QueryRow(`SELECT uuid, name, cpu, wt, ct, mu, pmu FROM profiles WHERE uuid = ?`, uuid)

		var p struct {
			UUID string  `json:"uuid"`
			Name string  `json:"name"`
			Peaks Metrics `json:"peaks"`
		}
		if err := row.Scan(&p.UUID, &p.Name, &p.Peaks.CPU, &p.Peaks.WallTime, &p.Peaks.Calls, &p.Peaks.Memory, &p.Peaks.PeakMem); err != nil {
			http.Error(w, "profile not found", http.StatusNotFound)
			return
		}

		writeJSON(w, p)
	})

	// GET /api/profiler/{uuid}/call-graph
	mux.HandleFunc("GET /api/profiler/{uuid}/call-graph", func(w http.ResponseWriter, r *http.Request) {
		uuid := r.PathValue("uuid")
		edges, err := loadEdges(db, uuid, 0)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, edges)
	})

	// GET /api/profiler/{uuid}/top
	mux.HandleFunc("GET /api/profiler/{uuid}/top", func(w http.ResponseWriter, r *http.Request) {
		uuid := r.PathValue("uuid")
		limit := 100
		if l := r.URL.Query().Get("limit"); l != "" {
			if v, err := strconv.Atoi(l); err == nil && v >= 10 && v <= 300 {
				limit = v
			}
		}

		edges, err := loadEdges(db, uuid, limit)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, edges)
	})

	// GET /api/profiler/{uuid}/flame-chart
	mux.HandleFunc("GET /api/profiler/{uuid}/flame-chart", func(w http.ResponseWriter, r *http.Request) {
		uuid := r.PathValue("uuid")
		edges, err := loadEdges(db, uuid, 0)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, edges)
	})
}

func loadEdges(db *sql.DB, profileUUID string, limit int) ([]Edge, error) {
	query := `SELECT callee, caller, cpu, wt, ct, mu, pmu, d_cpu, d_wt, d_ct, d_mu, d_pmu, p_cpu, p_wt, p_ct, p_mu, p_pmu, parent_uuid
		FROM profile_edges WHERE profile_uuid = ? ORDER BY "order" ASC`
	if limit > 0 {
		query += " LIMIT " + strconv.Itoa(limit)
	}

	rows, err := db.Query(query, profileUUID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var edges []Edge
	id := 1
	for rows.Next() {
		var e Edge
		err := rows.Scan(
			&e.Callee, &e.Caller,
			&e.Cost.CPU, &e.Cost.WallTime, &e.Cost.Calls, &e.Cost.Memory, &e.Cost.PeakMem,
			&e.Diff.CPU, &e.Diff.WallTime, &e.Diff.Calls, &e.Diff.Memory, &e.Diff.PeakMem,
			&e.Percents.CPU, &e.Percents.WallTime, &e.Percents.Calls, &e.Percents.Memory, &e.Percents.PeakMem,
			&e.Parent,
		)
		if err != nil {
			return nil, err
		}
		e.ID = "e" + strconv.Itoa(id)
		id++
		edges = append(edges, e)
	}
	return edges, rows.Err()
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}
