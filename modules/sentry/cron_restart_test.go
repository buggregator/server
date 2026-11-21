package sentry

import (
	"testing"
	"time"
)

// Review probe: a short restart over a slot. The old process wrote its last
// heartbeat at 10:04:40, was down 10:04:50-10:05:10 (the 10:05 check-in was
// refused), and the new one ticks first at 10:05:40: a 60 s gap.
func TestCronShortRestartIsUnknown(t *testing.T) {
	db := setupTestDB(t)
	t0 := time.Date(2026, 11, 21, 10, 0, 30, 0, time.UTC)
	every5 := &cronConfig{Schedule: &cronSchedule{Type: "crontab", Value: "*/5 * * * *"}}
	recordCheckIn(db, cronCheckIn{Slug: "job", Status: cronOK, Source: cronSourceSentry, Config: every5}, t0)
	old := &cronChecker{db: db}
	old.tick(time.Date(2026, 11, 21, 10, 4, 40, 0, time.UTC))

	restarted := &cronChecker{db: db}
	restarted.tick(time.Date(2026, 11, 21, 10, 5, 40, 0, time.UTC))
	restarted.tick(time.Date(2026, 11, 21, 10, 6, 10, 0, time.UTC))
	runs := cronRuns(t, db, "job")
	if len(runs) != 2 || runs[1].Status != cronUnknown {
		t.Fatalf("slot during a 20 s restart: runs = %+v", runs)
	}
}

// A duration too big for int64 milliseconds is clamped to a year, so the
// run never starts in the future.
func TestCronHugeDuration(t *testing.T) {
	db := setupTestDB(t)
	now := time.Date(2026, 11, 21, 10, 0, 0, 0, time.UTC)
	d := 1e300
	if _, err := recordCheckIn(db, cronCheckIn{Slug: "job", Status: cronOK, Duration: &d, Source: cronSourceSentry}, now); err != nil {
		t.Fatal(err)
	}
	runs := cronRuns(t, db, "job")
	if len(runs) != 1 || runs[0].Duration.Int64 != 365*24*3600*1000 || runs[0].StartedAt != "2025-11-21T10:00:00.000Z" {
		t.Fatalf("runs = %+v", runs)
	}
}
