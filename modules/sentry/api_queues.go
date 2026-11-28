package sentry

import (
	"database/sql"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Queue jobs come from sentry_queue_jobs: one row per attempt, grouped into
// a job by message_id (an attempt without one is a job of its own).
//
//	GET /api/sentry/queues/jobs           jobs, newest last attempt first
//	GET /api/sentry/queues/summary        totals and queues for a window
//	GET /api/sentry/queues/jobs/{id}      one job (message id or transaction id): attempts, errors, dispatcher
//
// A job's status is its last attempt's: "ok" or failed (any other status).
// Sentry sends an attempt only when it ends, so running jobs are not seen,
// and a failed job that will be retried cannot be told from one that will not.

// jobFailedSQL is true for a failed attempt row.
const jobFailedSQL = `(status NOT IN ('ok', ''))`

// jobsCTE ranks the attempts of each job; rn = 1 is the last attempt. where
// filters attempts (project, queue, job class), before grouping.
func jobsCTE(where string) string {
	return `WITH a AS (
		SELECT *, COALESCE(message_id, transaction_id) AS k FROM sentry_queue_jobs` + where + `
	), j AS (
		SELECT a.*,
			ROW_NUMBER() OVER (PARTITION BY k ORDER BY attempt DESC, finished_at DESC) AS rn,
			COUNT(*) OVER (PARTITION BY k) AS attempts,
			SUM(CASE WHEN ` + jobFailedSQL + ` THEN 1 ELSE 0 END) OVER (PARTITION BY k) AS failed_attempts,
			MIN(started_at) OVER (PARTITION BY k) AS first_started_at
		FROM a
	)`
}

// jobAttemptFilters reads ?project, ?queue and ?job (exact class name).
func jobAttemptFilters(r *http.Request) (string, []any) {
	conds, args := appendProject(nil, nil, r, "project_id")
	q := r.URL.Query()
	if v := strings.TrimSpace(q.Get("queue")); v != "" {
		conds = append(conds, "queue = ?")
		args = append(args, v)
	}
	if v := strings.TrimSpace(q.Get("job")); v != "" {
		conds = append(conds, "job = ?")
		args = append(args, v)
	}
	return whereOf(conds), args
}

const jobColumns = `k, message_id, job, queue, connection, status, attempts, failed_attempts,
	duration_ms, latency_ms, first_started_at, finished_at, trace_id, transaction_id, project_id`

func scanJob(rows interface{ Scan(...any) error }) (map[string]any, error) {
	var (
		k, job, queue, conn, status, traceID, txnID, project string
		msgID, started, finished                             sql.NullString
		attempts, failedAttempts                             int
		duration, latency                                    sql.NullInt64
	)
	if err := rows.Scan(&k, &msgID, &job, &queue, &conn, &status, &attempts, &failedAttempts,
		&duration, &latency, &started, &finished, &traceID, &txnID, &project); err != nil {
		return nil, err
	}
	return map[string]any{
		"id":              k,
		"message_id":      nullIfEmpty(msgID.String),
		"job":             job,
		"queue":           queue,
		"connection":      conn,
		"status":          status,
		"failed":          status != "ok" && status != "",
		"attempts":        attempts,
		"failed_attempts": failedAttempts,
		"retried":         attempts > 1,
		"duration_ms":     scanNullInt(duration),
		"latency_ms":      scanNullInt(latency),
		"started_at":      nullIfEmpty(started.String),
		"finished_at":     nullIfEmpty(finished.String),
		"trace_id":        traceID,
		"transaction_id":  txnID,
		"project":         project,
	}, nil
}

func handleQueueJobs(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		res, code, err := queueJobsList(db, r)
		if err != nil {
			apiError(w, err.Error(), code)
			return
		}
		apiJSON(w, res)
	}
}

// QueueJobs lists queue jobs for the MCP tool; q takes the parameters of
// GET /api/sentry/queues/jobs.
func QueueJobs(db *sql.DB, q url.Values) (map[string]any, error) {
	res, _, err := queueJobsList(db, &http.Request{URL: &url.URL{RawQuery: q.Encode()}})
	return res, err
}

// QueueJob returns one job with its attempts for the MCP tool; nil when
// there is no such job.
func QueueJob(db *sql.DB, id string) (map[string]any, error) {
	res, code, err := queueJobDetail(db, &http.Request{URL: &url.URL{}}, id)
	if code == http.StatusNotFound {
		return nil, nil
	}
	return res, err
}

func queueJobsList(db *sql.DB, r *http.Request) (map[string]any, int, error) {
	limit, offset := pagination(r, 50)
	where, args := jobAttemptFilters(r)

	// Period and status apply to the job, i.e. its last attempt.
	outer := []string{"rn = 1"}
	outer, args = appendTimeConditions(outer, args, r, "finished_at", true)
	switch r.URL.Query().Get("status") {
	case "":
	case "ok":
		outer = append(outer, "NOT "+jobFailedSQL)
	case "failed":
		outer = append(outer, jobFailedSQL)
	case "retried":
		outer = append(outer, "attempts > 1")
	default:
		return nil, http.StatusBadRequest, errors.New("status must be ok, failed or retried")
	}
	base := jobsCTE(where) + ` SELECT ` + jobColumns + ` FROM j WHERE ` + strings.Join(outer, " AND ")

	var total int
	if err := db.QueryRow(`SELECT COUNT(*) FROM (`+base+`)`, args...).Scan(&total); err != nil {
		return nil, http.StatusInternalServerError, err
	}
	rows, err := db.Query(base+` ORDER BY finished_at DESC, k LIMIT ? OFFSET ?`, append(args, limit, offset)...)
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}
	defer rows.Close()
	data := []map[string]any{}
	for rows.Next() {
		if job, err := scanJob(rows); err == nil {
			data = append(data, job)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, http.StatusInternalServerError, err
	}
	return listResponse(data, total, currentPage(offset, limit), limit), http.StatusOK, nil
}

func handleQueueSummary(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Default window: the last hour.
		q := r.URL.Query()
		if q.Get("from") == "" && q.Get("to") == "" && q.Get("window") == "" {
			q.Set("window", "1h")
			r.URL.RawQuery = q.Encode()
		}
		where, args := jobAttemptFilters(r)
		outer, args := appendTimeConditions([]string{"rn = 1"}, args, r, "finished_at", true)
		jobs := jobsCTE(where) + `, last AS (SELECT * FROM j WHERE ` + strings.Join(outer, " AND ") + `)`

		var (
			total, failed, retried, attempts int
			avg                              sql.NullFloat64
		)
		err := db.QueryRow(jobs+` SELECT COUNT(*),
				COALESCE(SUM(CASE WHEN `+jobFailedSQL+` THEN 1 ELSE 0 END), 0),
				COALESCE(SUM(CASE WHEN attempts > 1 THEN 1 ELSE 0 END), 0),
				COALESCE(SUM(attempts), 0),
				(SELECT AVG(a.duration_ms) FROM a WHERE a.k IN (SELECT k FROM last))
			FROM last`, args...).Scan(&total, &failed, &retried, &attempts, &avg)
		if err != nil {
			apiError(w, err.Error(), http.StatusInternalServerError)
			return
		}

		queues := []map[string]any{}
		rows, err := db.Query(jobs+` SELECT queue, COUNT(*), SUM(CASE WHEN `+jobFailedSQL+` THEN 1 ELSE 0 END)
			FROM last GROUP BY queue ORDER BY COUNT(*) DESC, queue`, args...)
		if err != nil {
			apiError(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer rows.Close()
		for rows.Next() {
			var name string
			var n, nf int
			if rows.Scan(&name, &n, &nf) == nil {
				queues = append(queues, map[string]any{"queue": name, "jobs": n, "failed": nf})
			}
		}
		if err := rows.Err(); err != nil {
			apiError(w, err.Error(), http.StatusInternalServerError)
			return
		}

		var avgMS *float64
		if avg.Valid {
			v := float64(int(avg.Float64*10+0.5)) / 10
			avgMS = &v
		}
		apiJSON(w, map[string]any{
			"jobs":            total,
			"failed":          failed,
			"retried":         retried,
			"attempts":        attempts,
			"avg_duration_ms": avgMS,
			"queues":          queues,
		})
	}
}

func handleQueueJob(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		res, code, err := queueJobDetail(db, r, r.PathValue("id"))
		if err != nil {
			apiError(w, err.Error(), code)
			return
		}
		apiJSON(w, res)
	}
}

func queueJobDetail(db *sql.DB, r *http.Request, id string) (map[string]any, int, error) {
	pc, pa := projectClause(r, "project_id")
	where := ` WHERE (message_id = ? OR (message_id IS NULL AND transaction_id = ?))` + pc
	args := append([]any{id, id}, pa...)

	job, err := scanJob(db.QueryRow(jobsCTE(where)+` SELECT `+jobColumns+` FROM j WHERE rn = 1`, args...))
	if err == sql.ErrNoRows {
		return nil, http.StatusNotFound, errors.New("job not found")
	}
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}

	type attempt struct {
		data       map[string]any
		start, end time.Time
	}
	var attempts []*attempt
	traces := map[string]bool{}
	rows, err := db.Query(`SELECT transaction_id, trace_id, attempt, status, duration_ms, latency_ms, started_at, finished_at
		FROM sentry_queue_jobs`+where+` ORDER BY started_at, attempt`, args...)
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}
	for rows.Next() {
		var (
			txnID, traceID, status string
			n                      int
			duration, latency      sql.NullInt64
			started, finished      sql.NullString
		)
		if rows.Scan(&txnID, &traceID, &n, &status, &duration, &latency, &started, &finished) != nil {
			continue
		}
		a := &attempt{data: map[string]any{
			"transaction_id": txnID,
			"trace_id":       traceID,
			"attempt":        n,
			"status":         status,
			"failed":         status != "ok" && status != "",
			"duration_ms":    scanNullInt(duration),
			"latency_ms":     scanNullInt(latency),
			"started_at":     nullIfEmpty(started.String),
			"finished_at":    nullIfEmpty(finished.String),
			"gap_ms":         nil,
			"errors":         []map[string]any{},
		}}
		a.start, _ = parseRFC3339(started.String)
		a.end, _ = parseRFC3339(finished.String)
		if len(attempts) > 0 {
			if prev := attempts[len(attempts)-1]; !prev.end.IsZero() && !a.start.IsZero() {
				a.data["gap_ms"] = a.start.Sub(prev.end).Milliseconds()
			}
		}
		attempts = append(attempts, a)
		traces[traceID] = true
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}

	// Errors: an error of the attempt's trace captured while it ran.
	for traceID := range traces {
		erows, err := db.Query(`SELECT e.event_id, e.event_ts, se.exception_type, se.exception_value, se.stacktrace
			FROM sentry_error_events e
			LEFT JOIN sentry_exceptions se ON se.error_event_id = e.id AND se.position = 0
			WHERE e.trace_id = ?`, traceID)
		if err != nil {
			continue
		}
		for erows.Next() {
			var eventID string
			var ts, excType, excValue, stack sql.NullString
			if erows.Scan(&eventID, &ts, &excType, &excValue, &stack) != nil {
				continue
			}
			t, err := parseRFC3339(ts.String)
			if err != nil {
				continue
			}
			for _, a := range attempts {
				if a.data["trace_id"] == traceID && !t.Before(a.start) && !t.After(a.end.Add(time.Second)) {
					a.data["errors"] = append(a.data["errors"].([]map[string]any), map[string]any{
						"id":              eventID,
						"exception_type":  scanNullString(excType),
						"exception_value": scanNullString(excValue),
						"frame":           topAppFrame(stack.String),
						"event_ts":        ts.String,
					})
					break
				}
			}
		}
		err = erows.Err()
		erows.Close()
		if err != nil {
			return nil, http.StatusInternalServerError, err
		}
	}

	list := make([]map[string]any, 0, len(attempts))
	for _, a := range attempts {
		list = append(list, a.data)
	}
	job["attempt_list"] = list
	job["dispatched_by"] = loadDispatcher(db, job)
	return job, http.StatusOK, nil
}

// loadDispatcher finds the transaction that queued the job: its queue.publish
// span carries the same messaging.message.id. sentry-laravel continues the
// dispatcher's trace in the job, so the span is in one of the job's traces.
// Only spans of a stored transaction are searched: a standalone streamed
// (span v2) queue.publish span has no transaction_id, and the job then has
// no dispatched_by.
func loadDispatcher(db *sql.DB, job map[string]any) any {
	msgID, _ := job["message_id"].(*string)
	if msgID == nil {
		return nil
	}
	var txnID, traceID, name string
	var startTS sql.NullString
	err := db.QueryRow(`SELECT t.id, t.trace_id, t.transaction_name, t.start_ts
		FROM sentry_spans s JOIN sentry_transactions t ON t.id = s.transaction_id
		WHERE s.trace_id IN (SELECT trace_id FROM sentry_queue_jobs WHERE message_id = ?)
			AND s.op = 'queue.publish'
			AND json_extract(s.attributes, '$."messaging.message.id"') = ?
		ORDER BY s.start_ts LIMIT 1`, *msgID, *msgID).Scan(&txnID, &traceID, &name, &startTS)
	if err != nil {
		return nil
	}
	return map[string]any{
		"transaction_id":   txnID,
		"trace_id":         traceID,
		"transaction_name": name,
		"start_ts":         nullIfEmpty(startTS.String),
	}
}
