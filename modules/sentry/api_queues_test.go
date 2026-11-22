package sentry

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

// testdata/sdk/laravel-queue: sentry-laravel 4.31.0, a request dispatching
// two SendReport jobs to the "reports" queue (database driver), then the
// worker: job A fails on attempt 1 (RuntimeException) and succeeds on
// attempt 2, job B succeeds at once.
const (
	queueJobA = "99529e42-5137-4e2a-903b-e6a536167173"
	queueJobB = "852e5b1c-5653-4392-8c7d-6170bb3a54da"
)

func queueAPI(t *testing.T) (*http.ServeMux, *sql.DB) {
	t.Helper()
	db, _ := ingestSDK(t, "laravel-queue")
	mux := http.NewServeMux()
	registerAPI(mux, db, noAuth)
	return mux, db
}

func getQueueJSON(t *testing.T, mux *http.ServeMux, url string, v any) int {
	t.Helper()
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest("GET", url, nil))
	if w.Code == http.StatusOK {
		if err := json.Unmarshal(w.Body.Bytes(), v); err != nil {
			t.Fatalf("%s: %v", url, err)
		}
	}
	return w.Code
}

type queueJobJSON struct {
	ID             string  `json:"id"`
	MessageID      *string `json:"message_id"`
	Job            string  `json:"job"`
	Queue          string  `json:"queue"`
	Connection     string  `json:"connection"`
	Status         string  `json:"status"`
	Failed         bool    `json:"failed"`
	Attempts       int     `json:"attempts"`
	FailedAttempts int     `json:"failed_attempts"`
	Retried        bool    `json:"retried"`
	DurationMS     *int    `json:"duration_ms"`
	LatencyMS      *int    `json:"latency_ms"`
	TraceID        string  `json:"trace_id"`
	AttemptList    []struct {
		Attempt   int    `json:"attempt"`
		Status    string `json:"status"`
		Failed    bool   `json:"failed"`
		GapMS     *int64 `json:"gap_ms"`
		LatencyMS *int   `json:"latency_ms"`
		Errors    []struct {
			ExceptionType  string `json:"exception_type"`
			ExceptionValue string `json:"exception_value"`
		} `json:"errors"`
	} `json:"attempt_list"`
	DispatchedBy *struct {
		TransactionName string `json:"transaction_name"`
		TraceID         string `json:"trace_id"`
	} `json:"dispatched_by"`
}

func TestQueueJobs_Laravel(t *testing.T) {
	mux, db := queueAPI(t)

	var list struct {
		Data []queueJobJSON      `json:"data"`
		Meta struct{ Total int } `json:"meta"`
	}
	getQueueJSON(t, mux, "/api/sentry/queues/jobs", &list)
	if list.Meta.Total != 2 || len(list.Data) != 2 {
		t.Fatalf("jobs = %d (total %d), want 2: %+v", len(list.Data), list.Meta.Total, list.Data)
	}
	byID := map[string]queueJobJSON{}
	for _, j := range list.Data {
		byID[j.ID] = j
		if j.Job != `App\Jobs\SendReport` || j.Queue != "reports" || j.Connection != "database" {
			t.Errorf("%s: job %q queue %q connection %q", j.ID, j.Job, j.Queue, j.Connection)
		}
	}
	a, b := byID[queueJobA], byID[queueJobB]
	if a.Attempts != 2 || a.FailedAttempts != 1 || !a.Retried || a.Failed || a.Status != "ok" {
		t.Errorf("job A = %+v, want 2 attempts, 1 failed, ok now", a)
	}
	if b.Attempts != 1 || b.Retried || b.Failed {
		t.Errorf("job B = %+v, want 1 ok attempt", b)
	}
	// sentry-laravel sends the latency in seconds: about 1.3 s here.
	if b.LatencyMS == nil || *b.LatencyMS < 500 || *b.LatencyMS > 10000 {
		t.Errorf("job B latency_ms = %v, want ~1300", b.LatencyMS)
	}

	for _, tc := range []struct {
		query string
		want  int
	}{
		{"status=retried", 1}, {"status=failed", 0}, {"status=ok", 2},
		{"queue=reports", 2}, {"queue=emails", 0}, {`job=App\Jobs\SendReport`, 2},
		{"project=other", 0}, {"from=2100-01-01", 0},
	} {
		var l struct{ Data []queueJobJSON }
		getQueueJSON(t, mux, "/api/sentry/queues/jobs?"+strings.ReplaceAll(tc.query, `\`, "%5C"), &l)
		if len(l.Data) != tc.want {
			t.Errorf("?%s: %d jobs, want %d", tc.query, len(l.Data), tc.want)
		}
	}
	if code := getQueueJSON(t, mux, "/api/sentry/queues/jobs?status=running", &list); code != http.StatusBadRequest {
		t.Errorf("status=running: %d, want 400", code)
	}

	var sum struct {
		Jobs, Failed, Retried, Attempts int
		AvgDurationMS                   *float64 `json:"avg_duration_ms"`
		Queues                          []struct {
			Queue        string
			Jobs, Failed int
		}
	}
	getQueueJSON(t, mux, "/api/sentry/queues/summary?window=all", &sum)
	if sum.Jobs != 2 || sum.Failed != 0 || sum.Retried != 1 || sum.Attempts != 3 || sum.AvgDurationMS == nil ||
		len(sum.Queues) != 1 || sum.Queues[0].Queue != "reports" || sum.Queues[0].Jobs != 2 {
		t.Errorf("summary = %+v", sum)
	}
	// Default window is the last hour: move the jobs into it, then out.
	if _, err := db.Exec(`UPDATE sentry_queue_jobs SET finished_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now', '-1 minute')`); err != nil {
		t.Fatal(err)
	}
	getQueueJSON(t, mux, "/api/sentry/queues/summary", &sum)
	if sum.Jobs != 2 {
		t.Errorf("summary, last hour = %+v, want 2 jobs", sum)
	}
	if _, err := db.Exec(`UPDATE sentry_queue_jobs SET finished_at = '2020-01-01T00:00:00Z'`); err != nil {
		t.Fatal(err)
	}
	getQueueJSON(t, mux, "/api/sentry/queues/summary", &sum)
	if sum.Jobs != 0 || sum.Queues == nil {
		t.Errorf("summary, last hour, old jobs = %+v, want 0 jobs", sum)
	}
	if _, err := db.Exec(`UPDATE sentry_queue_jobs SET finished_at = (SELECT end_ts FROM sentry_transactions WHERE id = transaction_id)`); err != nil {
		t.Fatal(err)
	}

	var job queueJobJSON
	if code := getQueueJSON(t, mux, "/api/sentry/queues/jobs/"+queueJobA, &job); code != http.StatusOK {
		t.Fatalf("job A: %d", code)
	}
	if len(job.AttemptList) != 2 {
		t.Fatalf("attempts = %+v", job.AttemptList)
	}
	first, second := job.AttemptList[0], job.AttemptList[1]
	if first.Attempt != 1 || !first.Failed || first.Status != "internal_error" || first.GapMS != nil {
		t.Errorf("attempt 1 = %+v", first)
	}
	if len(first.Errors) != 1 || first.Errors[0].ExceptionType != "RuntimeException" ||
		!strings.Contains(first.Errors[0].ExceptionValue, "Mail server timeout") {
		t.Errorf("attempt 1 errors = %+v", first.Errors)
	}
	if second.Attempt != 2 || second.Failed || len(second.Errors) != 0 || second.GapMS == nil || *second.GapMS < 0 {
		t.Errorf("attempt 2 = %+v", second)
	}
	if job.DispatchedBy == nil || job.DispatchedBy.TransactionName != "/dispatch" || job.DispatchedBy.TraceID != job.TraceID {
		t.Errorf("dispatched_by = %+v, want /dispatch in trace %s", job.DispatchedBy, job.TraceID)
	}

	if code := getQueueJSON(t, mux, "/api/sentry/queues/jobs/nope", &job); code != http.StatusNotFound {
		t.Errorf("unknown job: %d, want 404", code)
	}
}

// The migration fills the table from transactions stored before it, with
// the same values ingest writes.
func TestQueueJobs_MigrationBackfill(t *testing.T) {
	_, db := queueAPI(t)
	const cols = `transaction_id, COALESCE(message_id, ''), trace_id, project_id, job, queue, connection, attempt, status, COALESCE(duration_ms, -1), COALESCE(latency_ms, -1), started_at, finished_at`
	dump := func() []string {
		rows, err := db.Query(`SELECT ` + cols + ` FROM sentry_queue_jobs ORDER BY transaction_id`)
		if err != nil {
			t.Fatal(err)
		}
		defer rows.Close()
		var out []string
		for rows.Next() {
			v := make([]any, 13)
			p := make([]any, 13)
			for i := range v {
				p[i] = &v[i]
			}
			if err := rows.Scan(p...); err != nil {
				t.Fatal(err)
			}
			b, _ := json.Marshal(v)
			out = append(out, string(b))
		}
		return out
	}
	ingested := dump()
	if len(ingested) != 3 {
		t.Fatalf("ingested attempts = %d, want 3", len(ingested))
	}

	if _, err := db.Exec(`DELETE FROM sentry_queue_jobs`); err != nil {
		t.Fatal(err)
	}
	src, err := os.ReadFile("migrations/2024_01_01_000042_create_sentry_queue_jobs.sql")
	if err != nil {
		t.Fatal(err)
	}
	for _, stmt := range strings.Split(string(src), ";") {
		if strings.TrimSpace(stmt) != "" {
			if _, err := db.Exec(stmt); err != nil {
				t.Fatalf("%v\n%s", err, stmt)
			}
		}
	}
	if got := dump(); strings.Join(got, "\n") != strings.Join(ingested, "\n") {
		t.Errorf("backfill\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(ingested, "\n"))
	}
}

func TestQueueJobs_ClearRemovesRows(t *testing.T) {
	_, db := queueAPI(t)
	for _, project := range []string{"other", ""} {
		if err := clearSentry(db, project, false); err != nil {
			t.Fatal(err)
		}
		var n int
		db.QueryRow(`SELECT COUNT(*) FROM sentry_queue_jobs`).Scan(&n)
		want := 3
		if project == "" {
			want = 0
		}
		if n != want {
			t.Errorf("clear project %q: %d rows left, want %d", project, n, want)
		}
	}
	// A project clear drops the rows of that project's transactions.
	_, db = queueAPI(t)
	if err := clearSentry(db, "default", false); err != nil {
		t.Fatal(err)
	}
	var n int
	db.QueryRow(`SELECT COUNT(*) FROM sentry_queue_jobs`).Scan(&n)
	if n != 0 {
		t.Errorf("clear project default: %d rows left, want 0", n)
	}
}
