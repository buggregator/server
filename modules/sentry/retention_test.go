package sentry

import (
	"context"
	"testing"
	"time"
)

// Deleting an event card removes its structured error; other projects' rows stay.
func TestOnEventsDeleted(t *testing.T) {
	db, _ := clearTestDB(t)
	seedProject(t, db, "shop", "e-shop", "trace-shop")
	seedProject(t, db, "billing", "e-billing", "trace-billing")
	m := &Module{db: db}

	m.OnEventsDeleted([]string{"e-shop", "unknown"})

	if n := countRows(t, db, "sentry_error_events", "WHERE event_id = 'e-shop'"); n != 0 {
		t.Errorf("e-shop error rows = %d, want 0", n)
	}
	if n := countRows(t, db, "sentry_exceptions", ""); n != 1 {
		t.Errorf("exceptions = %d, want 1 (billing)", n)
	}
	if n := countRows(t, db, "sentry_breadcrumbs", ""); n != 1 {
		t.Errorf("breadcrumbs = %d, want 1 (billing)", n)
	}
	if n := countRows(t, db, "sentry_error_events", "WHERE event_id = 'e-billing'"); n != 1 {
		t.Errorf("e-billing error rows = %d, want 1", n)
	}
	// Traces, logs and metrics are not tied to a card.
	if n := countRows(t, db, "sentry_logs", ""); n != 2 {
		t.Errorf("logs = %d, want 2", n)
	}
}

func TestPruneBefore(t *testing.T) {
	db, _ := clearTestDB(t)
	seedProject(t, db, "shop", "e-pinned", "trace-a") // transactions/spans start in 1970
	seedProject(t, db, "shop", "e-orphan", "trace-b")
	if _, err := db.Exec(`INSERT INTO events (uuid, type, payload, timestamp, project, is_pinned)
		VALUES ('e-pinned', 'sentry', '{}', '1.0', 'shop', 1)`); err != nil {
		t.Fatal(err)
	}
	m := &Module{db: db}
	ctx := context.Background()

	// An hour ago: the 1970 transactions and spans go; the v2 spans without a
	// start time stay with their trace (written now), as do logs, metrics, errors.
	n, err := m.PruneBefore(ctx, time.Now().Add(-time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	for table, want := range map[string]int{
		"sentry_transactions": 0, "sentry_spans": 2, "sentry_traces": 2,
		"sentry_logs": 2, "sentry_metrics": 2, "sentry_error_events": 2,
	} {
		if got := countRows(t, db, table, ""); got != want {
			t.Errorf("after 1h cutoff %s = %d, want %d", table, got, want)
		}
	}
	if n != 2+4 { // 2 transactions with 2 spans each
		t.Errorf("deleted = %d, want 6", n)
	}

	// A cutoff in the future: the rest of the traces, logs, metrics and the
	// error without a card go; the error of the pinned card stays.
	if _, err := m.PruneBefore(ctx, time.Now().Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	for table, want := range map[string]int{
		"sentry_spans": 0, "sentry_traces": 0,
		"sentry_logs": 0, "sentry_metrics": 0, "sentry_error_events": 1,
		"sentry_exceptions": 1, "sentry_breadcrumbs": 1,
	} {
		if got := countRows(t, db, table, ""); got != want {
			t.Errorf("after future cutoff %s = %d, want %d", table, got, want)
		}
	}
	if got := countRows(t, db, "sentry_error_events", "WHERE event_id = 'e-pinned'"); got != 1 {
		t.Error("the pinned card's error was deleted")
	}
}
