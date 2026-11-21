package sentry

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// sentryCheckIn is a check_in envelope item, also the JSON body of the cron
// HTTP endpoint.
type sentryCheckIn struct {
	CheckInID     string   `json:"check_in_id"`
	MonitorSlug   string   `json:"monitor_slug"`
	Status        string   `json:"status"`
	Environment   string   `json:"environment"`
	Duration      *float64 `json:"duration"`
	MonitorConfig *struct {
		Schedule *struct {
			Type  string          `json:"type"`
			Value json.RawMessage `json:"value"`
			Unit  string          `json:"unit"`
		} `json:"schedule"`
		CheckinMargin *float64 `json:"checkin_margin"`
		MaxRuntime    *float64 `json:"max_runtime"`
		Timezone      *string  `json:"timezone"`
	} `json:"monitor_config"`
	Contexts struct {
		Trace struct {
			TraceID string `json:"trace_id"`
		} `json:"trace"`
	} `json:"contexts"`
}

func (s sentryCheckIn) checkIn(project, envEnvironment string) cronCheckIn {
	c := cronCheckIn{
		Project:     project,
		Slug:        s.MonitorSlug,
		CheckInID:   strings.ReplaceAll(s.CheckInID, "-", ""),
		Status:      s.Status,
		Environment: s.Environment,
		TraceID:     s.Contexts.Trace.TraceID,
		Source:      cronSourceSentry,
		Duration:    s.Duration,
	}
	if c.Environment == "" {
		c.Environment = envEnvironment
	}
	if mc := s.MonitorConfig; mc != nil {
		cfg := &cronConfig{Timezone: mc.Timezone, CheckinMargin: minutes(mc.CheckinMargin), MaxRuntime: minutes(mc.MaxRuntime)}
		if sc := mc.Schedule; sc != nil {
			cfg.Schedule = &cronSchedule{Type: sc.Type, Value: rawScalar(sc.Value), Unit: sc.Unit}
			if sc.Type == scheduleCrontab {
				cfg.Schedule.Unit = ""
			}
		}
		c.Config = cfg
	}
	return c
}

func minutes(v *float64) *int {
	if v == nil {
		return nil
	}
	n := int(math.Round(*v))
	return &n
}

// rawScalar reads a JSON string or number as text: a crontab value is a
// string, an interval value a number.
func rawScalar(raw json.RawMessage) string {
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return s
	}
	return strings.TrimSpace(string(raw))
}

// handleCheckInItem stores a check_in envelope item. It reports whether a
// check-in was stored.
func (h *handler) handleCheckInItem(item EnvelopeItem, env EnvelopeHeader, project string) bool {
	if h.db == nil {
		return false
	}
	var s sentryCheckIn
	if err := json.Unmarshal(item.Payload, &s); err != nil {
		slog.Warn("sentry: dropped check_in item", "err", err)
		return false
	}
	if _, err := storeCheckIn(h.db, s.checkIn(project, env.Trace.Environment), time.Now()); err != nil {
		slog.Warn("sentry: dropped check_in item", "slug", s.MonitorSlug, "err", err)
		return false
	}
	return true
}

// storeCheckIn records a check-in from an SDK. A config the server cannot
// read (a schedule it does not know) is dropped and the check-in kept: the
// run still happened.
func storeCheckIn(db *sql.DB, c cronCheckIn, now time.Time) (string, error) {
	id, err := recordCheckIn(db, c, now)
	if errors.Is(err, errBadSchedule) {
		slog.Warn("cron: ignored monitor config", "slug", c.Slug, "err", err)
		c.Config = nil
		id, err = recordCheckIn(db, c, now)
	}
	return id, err
}

// cronHTTP serves the open check-in endpoints: Sentry's cron HTTP endpoint and
// the ping URL.
type cronHTTP struct {
	db     *sql.DB
	notify func(project string, kinds []string)
	now    func() time.Time
}

// sentryCron handles GET|POST /api/{project}/cron/{slug}/{key}/ with
// ?status=&check_in_id=&environment=&duration= (seconds), or the same as a
// JSON body with monitor_config. It answers 202 with the check_in_id.
func (c *cronHTTP) sentryCron(w http.ResponseWriter, r *http.Request) {
	project := resolveProject(r.PathValue("project"))
	if project == "" {
		project = (&handler{db: c.db}).projectByKey(r.PathValue("key"))
	}
	var s sentryCheckIn
	if r.Method == http.MethodPost {
		body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if len(bytes.TrimSpace(body)) > 0 {
			if err := json.Unmarshal(body, &s); err != nil {
				http.Error(w, "invalid JSON body", http.StatusBadRequest)
				return
			}
		}
	}
	q := r.URL.Query()
	s.MonitorSlug = r.PathValue("slug")
	if v := q.Get("status"); v != "" {
		s.Status = v
	}
	if v := q.Get("check_in_id"); v != "" {
		s.CheckInID = v
	}
	if v := q.Get("environment"); v != "" {
		s.Environment = v
	}
	if v := q.Get("duration"); v != "" {
		if d, err := strconv.ParseFloat(v, 64); err == nil {
			s.Duration = &d
		}
	}
	id, ok := c.record(w, s.checkIn(project, ""), true)
	if !ok {
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]string{"id": id})
}

// ping handles GET|POST /ping/{slug}[/start|/fail]: ok, in_progress, error.
// The body (the job's output) is kept, its last 10 KB. ?schedule= (crontab),
// ?tz=, ?margin= and ?max_runtime= (minutes) set the monitor's config;
// ?project= or X-Buggregator-Project names the project.
func (c *cronHTTP) ping(w http.ResponseWriter, r *http.Request) {
	status := cronOK
	switch r.PathValue("action") {
	case "":
	case "start":
		status = cronInProgress
	case "fail":
		status = cronError
	default:
		http.NotFound(w, r)
		return
	}
	q := r.URL.Query()
	project := strings.TrimSpace(q.Get("project"))
	if project == "" {
		project = strings.TrimSpace(r.Header.Get("X-Buggregator-Project"))
	}
	ci := cronCheckIn{
		Project:     project,
		Slug:        r.PathValue("slug"),
		CheckInID:   q.Get("check_in_id"),
		Status:      status,
		Environment: q.Get("environment"),
		Source:      cronSourcePing,
	}
	if r.Body != nil {
		body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		ci.Body = bodyTail(body)
	}
	var cfg cronConfig
	set := false
	if v := strings.TrimSpace(q.Get("schedule")); v != "" {
		cfg.Schedule, set = &cronSchedule{Type: scheduleCrontab, Value: v}, true
	}
	if q.Has("tz") {
		tz := q.Get("tz")
		cfg.Timezone, set = &tz, true
	}
	for name, dst := range map[string]**int{"margin": &cfg.CheckinMargin, "max_runtime": &cfg.MaxRuntime} {
		if v := q.Get(name); v != "" {
			n, err := strconv.Atoi(v)
			if err != nil || n < 0 {
				http.Error(w, "invalid "+name, http.StatusBadRequest)
				return
			}
			*dst, set = &n, true
		}
	}
	if set {
		ci.Config = &cfg
	}
	id, ok := c.record(w, ci, false)
	if !ok {
		return
	}
	w.Header().Set("X-Check-In-Id", id)
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	io.WriteString(w, "OK\n")
}

// record stores the check-in and answers the errors. lenient drops a config
// it cannot read (an SDK sent it); a ping URL gets a 400 instead.
func (c *cronHTTP) record(w http.ResponseWriter, ci cronCheckIn, lenient bool) (string, bool) {
	now := time.Now()
	if c.now != nil {
		now = c.now()
	}
	var (
		id  string
		err error
	)
	if lenient {
		id, err = storeCheckIn(c.db, ci, now)
	} else {
		id, err = recordCheckIn(c.db, ci, now)
	}
	switch {
	case err == nil:
	case errors.Is(err, errTooManyMonitors):
		http.Error(w, err.Error(), http.StatusTooManyRequests)
		return "", false
	case errors.Is(err, errBadCheckIn), errors.Is(err, errBadSchedule):
		http.Error(w, err.Error(), http.StatusBadRequest)
		return "", false
	default:
		slog.Warn("cron: cannot store check-in", "err", err)
		http.Error(w, "cannot store check-in", http.StatusInternalServerError)
		return "", false
	}
	if c.notify != nil {
		c.notify(projectKey(ci.Project), []string{"cron"})
	}
	return id, true
}

// register adds the open check-in routes. They take GET too, which the
// ingestion pipeline does not, so they are routes of their own.
func (c *cronHTTP) register(mux *http.ServeMux) {
	for _, method := range []string{http.MethodGet, http.MethodPost} {
		mux.HandleFunc(method+" /ping/{slug}", c.ping)
		mux.HandleFunc(method+" /ping/{slug}/{action}", c.ping)
		mux.HandleFunc(method+" /api/{project}/cron/{slug}/{key}/{$}", c.sentryCron)
		mux.HandleFunc(method+" /api/{project}/cron/{slug}/{key}", c.sentryCron)
	}
}
