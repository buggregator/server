package sentry

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type cronRun struct {
	Status, Env, Trace, Source, Body string
	Duration                         sql.NullInt64
	ServerMade                       bool
	StartedAt                        string
}

func cronRuns(t *testing.T, db *sql.DB, slug string) []cronRun {
	t.Helper()
	rows, err := db.Query(`SELECT c.status, c.environment, c.trace_id, c.source, c.body, c.duration_ms, c.server_made, c.started_at
		FROM cron_checkins c JOIN cron_monitors m ON m.id = c.monitor_id WHERE m.slug = ? ORDER BY c.started_at, c.id`, slug)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var out []cronRun
	for rows.Next() {
		var r cronRun
		if err := rows.Scan(&r.Status, &r.Env, &r.Trace, &r.Source, &r.Body, &r.Duration, &r.ServerMade, &r.StartedAt); err != nil {
			t.Fatal(err)
		}
		out = append(out, r)
	}
	return out
}

func mustMonitor(t *testing.T, db *sql.DB, project, slug string) cronMonitorRow {
	t.Helper()
	m, err := findCronMonitor(db, project, slug)
	if err != nil {
		t.Fatalf("monitor %s/%s: %v", project, slug, err)
	}
	return m
}

// Check-ins recorded from real SDKs: @sentry/node withMonitor/captureCheckIn
// and sentry-php captureCheckIn (gzip). Each run is one row, opened by
// in_progress and closed by ok/error with the SDK's duration.
func TestCronSDKFixtures(t *testing.T) {
	for _, tc := range []struct {
		sdk      string
		slug     string
		status   string
		duration int64 // ms
		schedule cronSchedule
		tz       string
		margin   int
		maxRun   int
	}{
		{"crons-node", "nightly-report", cronOK, 4, cronSchedule{Type: "crontab", Value: "0 3 * * *"}, "Europe/Berlin", 5, 30},
		{"crons-node", "sync-users", cronError, 1500, cronSchedule{Type: "interval", Value: "10", Unit: "minute"}, "UTC", 1, 30},
		{"crons-php", "backup-db", cronOK, 3200, cronSchedule{Type: "crontab", Value: "*/5 * * * *"}, "UTC", 2, 10},
	} {
		t.Run(tc.sdk+"/"+tc.slug, func(t *testing.T) {
			db, feed := ingestSDK(t, tc.sdk)
			if len(feed) != 0 {
				t.Fatalf("check-ins made %d feed cards", len(feed))
			}
			m := mustMonitor(t, db, "default", tc.slug)
			if m.Schedule != tc.schedule || m.TZ != tc.tz || m.CheckinMargin != tc.margin || m.MaxRuntime != tc.maxRun {
				t.Errorf("monitor config = %+v %s %d %d", m.Schedule, m.TZ, m.CheckinMargin, m.MaxRuntime)
			}
			if m.LastStatus != tc.status || !m.NextExpected.Valid || m.Source != cronSourceSentry {
				t.Errorf("monitor last_status %q next %v source %q", m.LastStatus, m.NextExpected, m.Source)
			}
			runs := cronRuns(t, db, tc.slug)
			if len(runs) != 1 {
				t.Fatalf("runs = %d, want 1: %+v", len(runs), runs)
			}
			r := runs[0]
			if r.Status != tc.status || r.Duration.Int64 != tc.duration || r.Env != "production" || len(r.Trace) != 32 {
				t.Errorf("run = %+v", r)
			}
		})
	}
}

// A closing check-in can arrive before its in_progress (@sentry/node sends
// them concurrently): the late in_progress must not reopen the run.
func TestCronOutOfOrder(t *testing.T) {
	db := setupTestDB(t)
	now := time.Date(2026, 11, 21, 10, 0, 0, 0, time.UTC)
	d := 1.5
	for _, c := range []cronCheckIn{
		{Slug: "sync", CheckInID: "a1", Status: cronError, Duration: &d, Source: cronSourceSentry},
		{Slug: "sync", CheckInID: "a1", Status: cronInProgress, Source: cronSourceSentry},
	} {
		if _, err := recordCheckIn(db, c, now); err != nil {
			t.Fatal(err)
		}
	}
	runs := cronRuns(t, db, "sync")
	if len(runs) != 1 || runs[0].Status != cronError || mustMonitor(t, db, "", "sync").LastStatus != cronError {
		t.Fatalf("runs = %+v", runs)
	}
}

func TestCronSchedule(t *testing.T) {
	berlin, _ := time.LoadLocation("Europe/Berlin")
	for _, tc := range []struct {
		name  string
		s     cronSchedule
		tz    string
		after time.Time
		want  []time.Time
	}{
		{"crontab utc", cronSchedule{Type: "crontab", Value: "*/15 * * * *"}, "",
			time.Date(2026, 1, 1, 10, 7, 0, 0, time.UTC),
			[]time.Time{time.Date(2026, 1, 1, 10, 15, 0, 0, time.UTC), time.Date(2026, 1, 1, 10, 30, 0, 0, time.UTC)}},
		{"descriptor", cronSchedule{Type: "crontab", Value: "@daily"}, "",
			time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC), []time.Time{time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)}},
		{"crontab in zone", cronSchedule{Type: "crontab", Value: "0 3 * * *"}, "Europe/Berlin",
			time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), []time.Time{time.Date(2026, 1, 1, 2, 0, 0, 0, time.UTC)}},
		// 02:30 does not exist on 2026-03-29 in Berlin: no slot that day.
		{"dst gap", cronSchedule{Type: "crontab", Value: "30 2 * * *"}, "Europe/Berlin",
			time.Date(2026, 3, 29, 0, 0, 0, 0, berlin), []time.Time{time.Date(2026, 3, 30, 2, 30, 0, 0, berlin)}},
		// 02:30 happens twice on 2026-10-25 in Berlin: one slot.
		{"dst repeat", cronSchedule{Type: "crontab", Value: "30 2 * * *"}, "Europe/Berlin",
			time.Date(2026, 10, 25, 0, 0, 0, 0, berlin),
			[]time.Time{time.Date(2026, 10, 25, 0, 30, 0, 0, time.UTC), time.Date(2026, 10, 26, 2, 30, 0, 0, berlin)}},
		{"interval", cronSchedule{Type: "interval", Value: "10", Unit: "minute"}, "",
			time.Date(2026, 1, 1, 10, 3, 0, 0, time.UTC),
			[]time.Time{time.Date(2026, 1, 1, 10, 13, 0, 0, time.UTC), time.Date(2026, 1, 1, 10, 23, 0, 0, time.UTC)}},
		{"interval days in zone", cronSchedule{Type: "interval", Value: "1", Unit: "day"}, "Europe/Berlin",
			time.Date(2026, 3, 28, 12, 0, 0, 0, berlin), []time.Time{time.Date(2026, 3, 29, 12, 0, 0, 0, berlin)}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			next, err := tc.s.compile(tc.tz)
			if err != nil {
				t.Fatal(err)
			}
			at := tc.after
			for _, w := range tc.want {
				at = next(at)
				if !at.Equal(w) {
					t.Fatalf("next = %s, want %s", at, w.UTC())
				}
			}
		})
	}
	for _, bad := range []struct {
		s  cronSchedule
		tz string
	}{
		{cronSchedule{Type: "crontab", Value: "61 * * * *"}, ""},
		{cronSchedule{Type: "crontab", Value: "@every 5m"}, ""},
		{cronSchedule{Type: "interval", Value: "0", Unit: "minute"}, ""},
		{cronSchedule{Type: "interval", Value: "5", Unit: "fortnight"}, ""},
		{cronSchedule{Type: "weekly"}, ""},
		{cronSchedule{Type: "crontab", Value: "* * * * *"}, "Mars/Olympus"},
	} {
		if _, err := bad.s.compile(bad.tz); err == nil {
			t.Errorf("%+v %q compiled", bad.s, bad.tz)
		}
	}
}

// The checker: a slot past its margin with no check-in is missed, one gap is
// one row, a run longer than max_runtime is timeout, and a slot that fell
// while the server was down is unknown.
func TestCronChecker(t *testing.T) {
	t0 := time.Date(2026, 11, 21, 10, 0, 30, 0, time.UTC)
	every5 := &cronConfig{Schedule: &cronSchedule{Type: "crontab", Value: "*/5 * * * *"}}

	t.Run("missed after margin, one row per gap", func(t *testing.T) {
		db := setupTestDB(t)
		if _, err := recordCheckIn(db, cronCheckIn{Slug: "job", Status: cronOK, Source: cronSourceSentry, Config: every5}, t0); err != nil {
			t.Fatal(err)
		}
		// Slot 10:05, margin 1 min: not missed at 10:05:50.
		checkCrons(db, time.Date(2026, 11, 21, 10, 5, 50, 0, time.UTC), nil)
		if runs := cronRuns(t, db, "job"); len(runs) != 1 {
			t.Fatalf("missed inside the margin: %+v", runs)
		}
		projects, err := checkCrons(db, time.Date(2026, 11, 21, 10, 21, 0, 0, time.UTC), nil)
		if err != nil || len(projects) != 1 || projects[0] != "default" {
			t.Fatalf("projects %v err %v", projects, err)
		}
		runs := cronRuns(t, db, "job")
		if len(runs) != 2 || runs[1].Status != cronMissed || !runs[1].ServerMade || runs[1].StartedAt != "2026-11-21T10:05:00.000Z" {
			t.Fatalf("runs = %+v", runs)
		}
		m := mustMonitor(t, db, "", "job")
		// 10:10, 10:15 passed in the same gap; 10:20 is still inside its margin.
		if m.LastStatus != cronMissed || m.NextExpected.String != "2026-11-21T10:20:00.000Z" {
			t.Fatalf("monitor %q next %q", m.LastStatus, m.NextExpected.String)
		}
	})

	t.Run("in progress is not missed, hung is timeout", func(t *testing.T) {
		db := setupTestDB(t)
		cfg := *every5
		maxRun := 10
		cfg.MaxRuntime = &maxRun
		if _, err := recordCheckIn(db, cronCheckIn{Slug: "job", CheckInID: "r1", Status: cronInProgress, Source: cronSourceSentry, Config: &cfg}, t0); err != nil {
			t.Fatal(err)
		}
		checkCrons(db, t0.Add(8*time.Minute), nil)
		if runs := cronRuns(t, db, "job"); len(runs) != 1 || runs[0].Status != cronInProgress {
			t.Fatalf("runs = %+v", runs)
		}
		checkCrons(db, t0.Add(11*time.Minute), nil)
		runs := cronRuns(t, db, "job")
		if len(runs) != 1 || runs[0].Status != cronTimeout {
			t.Fatalf("runs = %+v", runs)
		}
		// A late ok keeps the timeout and records the end.
		recordCheckIn(db, cronCheckIn{Slug: "job", CheckInID: "r1", Status: cronOK, Source: cronSourceSentry}, t0.Add(12*time.Minute))
		if runs = cronRuns(t, db, "job"); runs[0].Status != cronTimeout || runs[0].Duration.Int64 != 12*60*1000 {
			t.Fatalf("runs = %+v", runs)
		}
	})

	t.Run("downtime is unknown", func(t *testing.T) {
		db := setupTestDB(t)
		recordCheckIn(db, cronCheckIn{Slug: "job", Status: cronOK, Source: cronSourceSentry, Config: every5}, t0)
		c := &cronChecker{db: db}
		c.tick(t0.Add(time.Minute))
		// Down from 10:01:30 to 10:30:00.
		c.tick(time.Date(2026, 11, 21, 10, 30, 0, 0, time.UTC))
		runs := cronRuns(t, db, "job")
		if len(runs) != 2 || runs[1].Status != cronUnknown {
			t.Fatalf("runs = %+v", runs)
		}
		// Up again: the 10:30 slot is missed for real.
		c.tick(time.Date(2026, 11, 21, 10, 30, 30, 0, time.UTC))
		c.tick(time.Date(2026, 11, 21, 10, 31, 30, 0, time.UTC))
		runs = cronRuns(t, db, "job")
		if len(runs) != 3 || runs[2].Status != cronMissed {
			t.Fatalf("runs = %+v", runs)
		}
	})

	t.Run("no schedule is never missed", func(t *testing.T) {
		db := setupTestDB(t)
		recordCheckIn(db, cronCheckIn{Slug: "adhoc", Status: cronOK, Source: cronSourcePing}, t0)
		checkCrons(db, t0.Add(48*time.Hour), nil)
		if runs := cronRuns(t, db, "adhoc"); len(runs) != 1 {
			t.Fatalf("runs = %+v", runs)
		}
	})
}

func newCronMux(t *testing.T, db *sql.DB, now time.Time) *http.ServeMux {
	t.Helper()
	mux := http.NewServeMux()
	(&cronHTTP{db: db, now: func() time.Time { return now }}).register(mux)
	registerCronAPI(mux, db, func(h http.Handler) http.Handler { return h }, nil)
	registerAPI(mux, db, func(h http.Handler) http.Handler { return h })
	return mux
}

func serve(mux *http.ServeMux, method, target, body string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, target, strings.NewReader(body))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	return w
}

func TestCronPing(t *testing.T) {
	db := setupTestDB(t)
	now := time.Date(2026, 11, 21, 10, 0, 0, 0, time.UTC)
	mux := newCronMux(t, db, now)

	w := serve(mux, "GET", "/ping/backup-db/start?schedule=0+3+*+*+*&tz=Europe/Berlin&margin=5&project=shop", "")
	if w.Code != 200 || w.Header().Get("X-Check-In-Id") == "" {
		t.Fatalf("start: %d %s", w.Code, w.Body)
	}
	out := strings.Repeat("x", maxPingBody) + "\ndone: 42 tables"
	if w := serve(mux, "POST", "/ping/backup-db?project=shop", out); w.Code != 200 {
		t.Fatalf("ok: %d %s", w.Code, w.Body)
	}
	runs := cronRuns(t, db, "backup-db")
	if len(runs) != 1 || runs[0].Status != cronOK || runs[0].Source != cronSourcePing ||
		len(runs[0].Body) != maxPingBody || !strings.HasSuffix(runs[0].Body, "done: 42 tables") {
		t.Fatalf("runs = %d %+v", len(runs), runs[0].Status)
	}
	m := mustMonitor(t, db, "shop", "backup-db")
	if m.Schedule.Value != "0 3 * * *" || m.TZ != "Europe/Berlin" || m.CheckinMargin != 5 || m.NextExpected.String != "2026-11-22T02:00:00.000Z" {
		t.Fatalf("monitor = %+v", m)
	}

	if w := serve(mux, "POST", "/ping/backup-db/fail?project=shop", "disk full"); w.Code != 200 {
		t.Fatalf("fail: %d", w.Code)
	}
	if runs := cronRuns(t, db, "backup-db"); len(runs) != 2 || runs[1].Status != cronError || runs[1].Body != "disk full" {
		t.Fatalf("runs = %+v", runs[1:])
	}

	for target, code := range map[string]int{
		"/ping/backup-db/pause":             404,
		"/ping/bad%20slug":                  400,
		"/ping/x?schedule=nope":             400,
		"/ping/x?tz=Mars/Olympus":           400,
		"/ping/x?margin=-1":                 400,
		"/ping/" + strings.Repeat("a", 129): 400,
	} {
		if w := serve(mux, "GET", target, ""); w.Code != code {
			t.Errorf("%s: %d, want %d", target, w.Code, code)
		}
	}
}

func TestCronMonitorCap(t *testing.T) {
	db := setupTestDB(t)
	now := time.Now()
	for i := 0; i < maxCronMonitors; i++ {
		if _, err := db.Exec(`INSERT INTO cron_monitors (project_id, slug, source, created_at) VALUES ('default', ?, 'ping', ?)`,
			fmt.Sprintf("m%d", i), cronTime(now)); err != nil {
			t.Fatal(err)
		}
	}
	mux := newCronMux(t, db, now)
	if w := serve(mux, "GET", "/ping/one-more", ""); w.Code != http.StatusTooManyRequests {
		t.Fatalf("over the cap: %d", w.Code)
	}
	// Another project has room; an existing monitor still takes check-ins.
	if w := serve(mux, "GET", "/ping/one-more?project=other", ""); w.Code != 200 {
		t.Fatalf("other project: %d", w.Code)
	}
}

// Sentry's cron HTTP endpoint, as curl sends it.
func TestCronSentryHTTP(t *testing.T) {
	db := setupTestDB(t)
	mux := newCronMux(t, db, time.Date(2026, 11, 21, 10, 0, 0, 0, time.UTC))

	w := serve(mux, "GET", "/api/shop/cron/nightly/abc123/?status=in_progress&environment=staging", "")
	if w.Code != http.StatusAccepted {
		t.Fatalf("in_progress: %d %s", w.Code, w.Body)
	}
	var resp struct{ ID string }
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.ID == "" {
		t.Fatal("no check_in_id in the answer")
	}
	body := `{"status":"ok","monitor_config":{"schedule":{"type":"interval","value":2,"unit":"hour"},"checkin_margin":3}}`
	if w := serve(mux, "POST", "/api/shop/cron/nightly/abc123/?check_in_id="+resp.ID+"&duration=12.5", body); w.Code != http.StatusAccepted {
		t.Fatalf("ok: %d %s", w.Code, w.Body)
	}
	runs := cronRuns(t, db, "nightly")
	if len(runs) != 1 || runs[0].Status != cronOK || runs[0].Duration.Int64 != 12500 || runs[0].Env != "staging" {
		t.Fatalf("runs = %+v", runs)
	}
	m := mustMonitor(t, db, "shop", "nightly")
	if m.Schedule != (cronSchedule{Type: "interval", Value: "2", Unit: "hour"}) || m.CheckinMargin != 3 {
		t.Fatalf("monitor = %+v", m)
	}
	// A numeric DSN project is the SDK's id: the default project.
	if w := serve(mux, "GET", "/api/1/cron/nightly/abc123?status=error", ""); w.Code != http.StatusAccepted {
		t.Fatalf("numeric project: %d", w.Code)
	}
	mustMonitor(t, db, "default", "nightly")
	if w := serve(mux, "GET", "/api/shop/cron/nightly/abc123/?status=paused", ""); w.Code != 400 {
		t.Fatalf("bad status: %d", w.Code)
	}
}

func TestCronAPI(t *testing.T) {
	db := setupTestDB(t)
	now := time.Now().UTC()
	mux := newCronMux(t, db, now)
	serve(mux, "GET", "/ping/backup?project=shop", "")
	serve(mux, "GET", "/ping/backup/fail?project=shop", "boom")
	serve(mux, "GET", "/ping/report", "")

	var list struct{ Data []cronMonitorJSON }
	w := serve(mux, "GET", "/api/crons?project=shop", "")
	json.Unmarshal(w.Body.Bytes(), &list)
	if w.Code != 200 || len(list.Data) != 1 || list.Data[0].LastStatus != cronError || len(list.Data[0].Recent) != 2 || list.Data[0].Schedule != nil {
		t.Fatalf("list: %d %s", w.Code, w.Body)
	}

	var runs struct {
		Data []cronCheckInJSON
		Meta struct{ Total int }
	}
	w = serve(mux, "GET", "/api/crons/backup/checkins?project=shop&limit=1", "")
	json.Unmarshal(w.Body.Bytes(), &runs)
	if w.Code != 200 || runs.Meta.Total != 2 || len(runs.Data) != 1 || runs.Data[0].Body != "boom" {
		t.Fatalf("checkins: %d %s", w.Code, w.Body)
	}

	w = serve(mux, "PUT", "/api/crons/backup?project=shop", `{"schedule":{"type":"crontab","value":"0 * * * *"},"timezone":"Asia/Tokyo","max_runtime":5}`)
	var m cronMonitorJSON
	json.Unmarshal(w.Body.Bytes(), &m)
	if w.Code != 200 || m.Schedule == nil || m.Timezone != "Asia/Tokyo" || m.MaxRuntime != 5 || m.NextExpectedAt == nil {
		t.Fatalf("put: %d %s", w.Code, w.Body)
	}
	if w := serve(mux, "PUT", "/api/crons/backup?project=shop", `{"schedule":null}`); w.Code != 200 || strings.Contains(w.Body.String(), `"next_expected_at":"`) {
		t.Fatalf("put null: %d %s", w.Code, w.Body)
	}
	if w := serve(mux, "PUT", "/api/crons/backup?project=shop", `{"schedule":{"type":"crontab","value":"x"}}`); w.Code != 400 {
		t.Fatalf("put bad: %d", w.Code)
	}
	if w := serve(mux, "PUT", "/api/crons/nope", `{}`); w.Code != 404 {
		t.Fatalf("put missing: %d", w.Code)
	}

	// Clearing a project's Sentry data drops its runs, keeps its monitors.
	if w := serve(mux, "DELETE", "/api/sentry/all?project=shop", ""); w.Code != 200 {
		t.Fatalf("clear: %d", w.Code)
	}
	if n := len(cronRuns(t, db, "backup")); n != 0 {
		t.Fatalf("runs after clear: %d", n)
	}
	if n := len(cronRuns(t, db, "report")); n != 1 {
		t.Fatalf("other project's runs after clear: %d", n)
	}
	mustMonitor(t, db, "shop", "backup")

	if w := serve(mux, "DELETE", "/api/crons/backup?project=shop", ""); w.Code != 200 {
		t.Fatalf("delete: %d", w.Code)
	}
	if _, err := findCronMonitor(db, "shop", "backup"); err != sql.ErrNoRows {
		t.Fatalf("monitor after delete: %v", err)
	}
}

func TestCronRetention(t *testing.T) {
	db, _ := clearTestDB(t)
	old := time.Now().Add(-48 * time.Hour)
	recordCheckIn(db, cronCheckIn{Slug: "job", Status: cronOK, Source: cronSourcePing}, old)
	recordCheckIn(db, cronCheckIn{Slug: "job", Status: cronOK, Source: cronSourcePing}, time.Now())
	if _, err := pruneBefore(t.Context(), db, time.Now().Add(-24*time.Hour)); err != nil {
		t.Fatal(err)
	}
	if n := len(cronRuns(t, db, "job")); n != 1 {
		t.Fatalf("runs after prune: %d", n)
	}
}
