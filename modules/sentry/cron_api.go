package sentry

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"
)

// cronMonitorJSON is a monitor as the API returns it.
type cronMonitorJSON struct {
	Project        string         `json:"project"`
	Slug           string         `json:"slug"`
	Source         string         `json:"source"`
	Schedule       *cronSchedule  `json:"schedule"`
	Timezone       string         `json:"timezone"`
	CheckinMargin  int            `json:"checkin_margin"`
	MaxRuntime     int            `json:"max_runtime"`
	LastStatus     string         `json:"last_status"`
	LastCheckinAt  *string        `json:"last_checkin_at"`
	NextExpectedAt *string        `json:"next_expected_at"`
	CreatedAt      string         `json:"created_at"`
	Recent         []cronRunBrief `json:"recent"`
}

// cronRunBrief is one dot of a monitor's recent runs, newest first.
type cronRunBrief struct {
	Status    string `json:"status"`
	StartedAt string `json:"started_at"`
}

// cronCheckInJSON is a run as the API returns it.
type cronCheckInJSON struct {
	ID          string  `json:"id"`
	Status      string  `json:"status"`
	StartedAt   string  `json:"started_at"`
	FinishedAt  *string `json:"finished_at"`
	DurationMS  *int64  `json:"duration_ms"`
	Environment string  `json:"environment"`
	TraceID     string  `json:"trace_id"`
	Source      string  `json:"source"`
	Body        string  `json:"body"`
	ServerMade  bool    `json:"server_made"`
	ExpectedAt  *string `json:"expected_at"`
}

const cronRecentRuns = 20

func nullStr(ns sql.NullString) *string {
	if !ns.Valid {
		return nil
	}
	return &ns.String
}

func monitorJSON(m cronMonitorRow) cronMonitorJSON {
	j := cronMonitorJSON{
		Project: m.Project, Slug: m.Slug, Source: m.Source, Timezone: m.TZ,
		CheckinMargin: m.CheckinMargin, MaxRuntime: m.MaxRuntime, LastStatus: m.LastStatus,
		LastCheckinAt: nullStr(m.LastCheckinAt), NextExpectedAt: nullStr(m.NextExpected),
		CreatedAt: m.CreatedAt, Recent: []cronRunBrief{},
	}
	if m.Schedule.Type != "" {
		s := m.Schedule
		j.Schedule = &s
	}
	return j
}

func registerCronAPI(mux *http.ServeMux, db *sql.DB, protect func(http.Handler) http.Handler, notify func(string, []string)) {
	route := func(pattern string, h http.HandlerFunc) { mux.Handle(pattern, protect(h)) }
	route("GET /api/crons", handleCronList(db))
	route("GET /api/crons/{slug}/checkins", handleCronCheckIns(db))
	route("PUT /api/crons/{slug}", handleCronUpdate(db, notify))
	route("DELETE /api/crons/{slug}", handleCronDelete(db, notify))
}

// handleCronList: GET /api/crons?project= — monitors with their recent
// runs; without ?project every project.
func handleCronList(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		conds, args := appendProject(nil, nil, r, "project_id")
		rows, err := db.Query(`SELECT `+cronMonitorCols+` FROM cron_monitors`+whereOf(conds)+` ORDER BY project_id, slug`, args...)
		if err != nil {
			apiError(w, err.Error(), http.StatusInternalServerError)
			return
		}
		var ids []int64
		byID := map[int64]int{}
		out := []cronMonitorJSON{}
		for rows.Next() {
			m, err := scanCronMonitor(rows)
			if err != nil {
				rows.Close()
				apiError(w, err.Error(), http.StatusInternalServerError)
				return
			}
			byID[m.ID] = len(out)
			ids = append(ids, m.ID)
			out = append(out, monitorJSON(m))
		}
		rows.Close()
		for _, id := range ids {
			rr, err := db.Query(`SELECT status, started_at FROM cron_checkins WHERE monitor_id = ?
				ORDER BY started_at DESC, id DESC LIMIT ?`, id, cronRecentRuns)
			if err != nil {
				apiError(w, err.Error(), http.StatusInternalServerError)
				return
			}
			for rr.Next() {
				var b cronRunBrief
				if err := rr.Scan(&b.Status, &b.StartedAt); err == nil {
					out[byID[id]].Recent = append(out[byID[id]].Recent, b)
				}
			}
			rr.Close()
		}
		apiJSON(w, map[string]any{"data": out})
	}
}

// monitorOr404 finds the monitor of {slug} in ?project (default "default").
func monitorOr404(w http.ResponseWriter, r *http.Request, db *sql.DB) (cronMonitorRow, bool) {
	m, err := findCronMonitor(db, strings.TrimSpace(r.URL.Query().Get("project")), r.PathValue("slug"))
	if errors.Is(err, sql.ErrNoRows) {
		apiError(w, "cron monitor not found", http.StatusNotFound)
		return m, false
	}
	if err != nil {
		apiError(w, err.Error(), http.StatusInternalServerError)
		return m, false
	}
	return m, true
}

// handleCronCheckIns: GET /api/crons/{slug}/checkins?project=&page=&limit= —
// runs newest first.
func handleCronCheckIns(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		m, ok := monitorOr404(w, r, db)
		if !ok {
			return
		}
		limit, offset := pagination(r, 50)
		var total int
		if err := db.QueryRow(`SELECT COUNT(*) FROM cron_checkins WHERE monitor_id = ?`, m.ID).Scan(&total); err != nil {
			apiError(w, err.Error(), http.StatusInternalServerError)
			return
		}
		rows, err := db.Query(`SELECT check_in_id, status, started_at, finished_at, duration_ms, environment, trace_id,
			source, body, server_made, expected_at FROM cron_checkins WHERE monitor_id = ?
			ORDER BY started_at DESC, id DESC LIMIT ? OFFSET ?`, m.ID, limit, offset)
		if err != nil {
			apiError(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer rows.Close()
		out := []cronCheckInJSON{}
		for rows.Next() {
			var (
				c                  cronCheckInJSON
				finished, expected sql.NullString
				dur                sql.NullInt64
			)
			if err := rows.Scan(&c.ID, &c.Status, &c.StartedAt, &finished, &dur, &c.Environment, &c.TraceID,
				&c.Source, &c.Body, &c.ServerMade, &expected); err != nil {
				apiError(w, err.Error(), http.StatusInternalServerError)
				return
			}
			c.FinishedAt, c.ExpectedAt = nullStr(finished), nullStr(expected)
			if dur.Valid {
				c.DurationMS = &dur.Int64
			}
			out = append(out, c)
		}
		apiJSON(w, listResponse(out, total, currentPage(offset, limit), limit))
	}
}

// cronUpdate is the PUT body. Absent fields keep their value; "schedule":
// null removes the schedule.
type cronUpdate struct {
	Schedule      json.RawMessage `json:"schedule"`
	Timezone      *string         `json:"timezone"`
	CheckinMargin *int            `json:"checkin_margin"`
	MaxRuntime    *int            `json:"max_runtime"`
}

// handleCronUpdate: PUT /api/crons/{slug}?project= — set the schedule, time
// zone, margin and max runtime (a ping monitor has no other way to get them).
// The next slot counts from now. Answers the monitor.
func handleCronUpdate(db *sql.DB, notify func(string, []string)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		m, ok := monitorOr404(w, r, db)
		if !ok {
			return
		}
		var u cronUpdate
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<16)).Decode(&u); err != nil {
			apiError(w, "invalid JSON body", http.StatusBadRequest)
			return
		}
		cfg := &cronConfig{Timezone: u.Timezone, CheckinMargin: u.CheckinMargin, MaxRuntime: u.MaxRuntime}
		if len(u.Schedule) > 0 {
			s := &cronSchedule{}
			if string(u.Schedule) != "null" {
				var raw struct {
					Type  string          `json:"type"`
					Value json.RawMessage `json:"value"`
					Unit  string          `json:"unit"`
				}
				if err := json.Unmarshal(u.Schedule, &raw); err != nil {
					apiError(w, "invalid schedule", http.StatusBadRequest)
					return
				}
				s = &cronSchedule{Type: raw.Type, Value: rawScalar(raw.Value), Unit: raw.Unit}
			}
			cfg.Schedule = s
		}
		m.apply(cfg)
		if (u.CheckinMargin != nil && *u.CheckinMargin < 0) || (u.MaxRuntime != nil && *u.MaxRuntime <= 0) {
			apiError(w, "checkin_margin must be >= 0 and max_runtime > 0", http.StatusBadRequest)
			return
		}
		if _, err := m.Schedule.compile(m.TZ); err != nil {
			apiError(w, err.Error(), http.StatusBadRequest)
			return
		}
		var next any
		if n, ok := m.next(time.Now()); ok {
			next = cronTime(n)
		}
		if _, err := db.Exec(`UPDATE cron_monitors SET schedule_type = ?, schedule_value = ?, schedule_unit = ?,
			timezone = ?, checkin_margin = ?, max_runtime = ?, next_expected_at = ? WHERE id = ?`,
			m.Schedule.Type, m.Schedule.Value, m.Schedule.Unit, m.TZ, m.CheckinMargin, m.MaxRuntime, next, m.ID); err != nil {
			apiError(w, err.Error(), http.StatusInternalServerError)
			return
		}
		m, err := findCronMonitor(db, m.Project, m.Slug)
		if err != nil {
			apiError(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if notify != nil {
			notify(m.Project, []string{"cron"})
		}
		apiJSON(w, monitorJSON(m))
	}
}

// handleCronDelete: DELETE /api/crons/{slug}?project= — the monitor and its
// runs. The next check-in creates it again.
func handleCronDelete(db *sql.DB, notify func(string, []string)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		m, ok := monitorOr404(w, r, db)
		if !ok {
			return
		}
		err := inTx(r.Context(), db, func(tx *sql.Tx) error {
			if _, err := tx.Exec(`DELETE FROM cron_checkins WHERE monitor_id = ?`, m.ID); err != nil {
				return err
			}
			_, err := tx.Exec(`DELETE FROM cron_monitors WHERE id = ?`, m.ID)
			return err
		})
		if err != nil {
			apiError(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if notify != nil {
			notify(m.Project, []string{"cron"})
		}
		apiJSON(w, map[string]any{"status": true})
	}
}
