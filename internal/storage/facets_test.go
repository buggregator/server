package storage_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"testing"

	"github.com/buggregator/server/v2/internal/event"
	"github.com/buggregator/server/v2/internal/storage"
)

// testFacets reads facets straight from a test payload:
// {"host":"web-1","env":"dev","tags":{"k":"v"}}.
func testFacets(_ string, payload json.RawMessage) []event.Facet {
	var p struct {
		Host string            `json:"host"`
		Env  string            `json:"env"`
		Tags map[string]string `json:"tags"`
	}
	_ = json.Unmarshal(payload, &p)
	var s event.FacetSet
	s.Add(event.FacetHost, p.Host)
	s.Add(event.FacetEnv, p.Env)
	for k, v := range p.Tags {
		s.AddTag(k, v)
	}
	return s.List()
}

// setupFacetDB opens a file database with the real core migrations and the
// cleanup trigger.
func setupFacetDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := storage.Open(filepath.Join(t.TempDir(), "facets.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	m := storage.NewMigrator(db)
	if err := m.AddFromFS("core", storage.CoreMigrations, "migrations"); err != nil {
		t.Fatal(err)
	}
	if err := m.Run(); err != nil {
		t.Fatal(err)
	}
	if err := storage.InitFacets(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	return db
}

func storeAt(t *testing.T, s *storage.SQLiteStore, uuid, typ, project, payload string, ts float64) {
	t.Helper()
	if err := s.Store(context.Background(), event.Event{
		UUID: uuid, Type: typ, Project: project, Payload: json.RawMessage(payload), Timestamp: ts,
	}); err != nil {
		t.Fatal(err)
	}
}

func uuids(evs []event.Event) []string {
	out := make([]string, 0, len(evs))
	for _, e := range evs {
		out = append(out, e.UUID)
	}
	sort.Strings(out)
	return out
}

func countRows(t *testing.T, db *sql.DB, uuid string) int {
	t.Helper()
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM event_facets WHERE uuid = ?`, uuid).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func TestFacets_FilterCountDelete(t *testing.T) {
	ctx := context.Background()
	db := setupFacetDB(t)
	s := storage.NewSQLiteStore(db)
	s.SetFacetFunc(testFacets)

	storeAt(t, s, "a", "sentry", "p1", `{"host":"web-1","env":"dev","tags":{"feature":"cart"}}`, 1)
	storeAt(t, s, "b", "sentry", "p1", `{"host":"web-2","env":"dev"}`, 2)
	storeAt(t, s, "c", "monolog", "p1", `{"host":"web-1","env":"prod"}`, 3)
	storeAt(t, s, "d", "smtp", "p1", `{}`, 4) // no facets
	storeAt(t, s, "e", "sentry", "p2", `{"host":"web-1"}`, 5)

	f := func(key string, not bool, values ...string) event.FacetFilter {
		return event.FacetFilter{Key: key, Values: values, Not: not}
	}
	for _, tc := range []struct {
		name string
		opts event.FindOptions
		want []string
	}{
		{"one host", event.FindOptions{Filters: []event.FacetFilter{f("host", false, "web-1")}}, []string{"a", "c", "e"}},
		{"OR within key", event.FindOptions{Project: "p1", Filters: []event.FacetFilter{f("host", false, "web-1", "web-2")}}, []string{"a", "b", "c"}},
		{"AND across keys", event.FindOptions{Filters: []event.FacetFilter{f("host", false, "web-1"), f("env", false, "dev")}}, []string{"a"}},
		{"exclude keeps events without the key", event.FindOptions{Project: "p1", Filters: []event.FacetFilter{f("host", true, "web-1")}}, []string{"b", "d"}},
		{"tag", event.FindOptions{Filters: []event.FacetFilter{f("tag:feature", false, "cart")}}, []string{"a"}},
		{"with type", event.FindOptions{Type: "monolog", Filters: []event.FacetFilter{f("host", false, "web-1")}}, []string{"c"}},
		{"unknown value", event.FindOptions{Filters: []event.FacetFilter{f("host", false, "nope")}}, []string{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			evs, err := s.FindAll(ctx, tc.opts)
			if err != nil {
				t.Fatal(err)
			}
			if got := uuids(evs); !reflect.DeepEqual(got, tc.want) {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}

	// Counts: a key's own filter does not narrow that key.
	counts, err := s.FacetCounts(ctx, event.FindOptions{Project: "p1",
		Filters: []event.FacetFilter{f("host", false, "web-1")}}, 0)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string][]event.FacetCount{
		"host":        {{Value: "web-1", Count: 2}, {Value: "web-2", Count: 1}},
		"env":         {{Value: "dev", Count: 1}, {Value: "prod", Count: 1}},
		"tag:feature": {{Value: "cart", Count: 1}},
	}
	if !reflect.DeepEqual(counts, want) {
		t.Errorf("counts = %v, want %v", counts, want)
	}

	// perKey caps values.
	counts, _ = s.FacetCounts(ctx, event.FindOptions{}, 1)
	if len(counts["host"]) != 1 || counts["host"][0] != (event.FacetCount{Value: "web-1", Count: 3}) {
		t.Errorf("capped host counts = %v", counts["host"])
	}

	// Any delete path removes the facets: the store, and raw SQL (retention,
	// module cleanup).
	if err := s.Delete(ctx, "a"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`DELETE FROM events WHERE uuid = 'b'`); err != nil {
		t.Fatal(err)
	}
	if n := countRows(t, db, "a") + countRows(t, db, "b"); n != 0 {
		t.Errorf("facets left after delete: %d", n)
	}
}

// A merged payload (Ray follow-up) replaces the event's facets.
func TestFacets_UpdatePayload(t *testing.T) {
	ctx := context.Background()
	db := setupFacetDB(t)
	s := storage.NewSQLiteStore(db)
	s.SetFacetFunc(testFacets)

	storeAt(t, s, "r", "ray", "", `{"host":"old"}`, 1)
	if err := s.UpdatePayload(ctx, "r", json.RawMessage(`{"host":"new","env":"dev"}`)); err != nil {
		t.Fatal(err)
	}
	counts, _ := s.FacetCounts(ctx, event.FindOptions{}, 0)
	want := map[string][]event.FacetCount{"host": {{Value: "new", Count: 1}}, "env": {{Value: "dev", Count: 1}}}
	if !reflect.DeepEqual(counts, want) {
		t.Errorf("counts = %v, want %v", counts, want)
	}
	if err := s.UpdatePayload(ctx, "missing", json.RawMessage(`{}`)); err != event.ErrNotFound {
		t.Errorf("missing event: err = %v, want ErrNotFound", err)
	}
}

// Events stored before the facets migration are indexed by the backfill, in
// batches, and a second run does nothing.
func TestFacets_Backfill(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "old.db")
	db, err := storage.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })

	// An old database: only the events migration has run.
	m := storage.NewMigrator(db)
	raw, _ := storage.CoreMigrations.ReadFile("migrations/2024_01_01_000000_create_events_table.sql")
	m.AddSQL("core", "2024_01_01_000000_create_events_table.sql", string(raw))
	if err := m.Run(); err != nil {
		t.Fatal(err)
	}
	// What the old server stored: events without facets, in one transaction
	// (a file database syncs every single insert).
	const n = 1203 // more than two batches
	tx, err := db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < n; i++ {
		host := "web-1"
		if i%3 == 0 {
			host = "web-2"
		}
		if _, err := tx.Exec(`INSERT INTO events (uuid, type, payload, timestamp, project) VALUES (?, 'sentry', ?, ?, '')`,
			"old-"+strconv.Itoa(i), `{"host":"`+host+`"}`, strconv.Itoa(i+1)); err != nil {
			t.Fatal(err)
		}
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}

	// Upgrade: all migrations, then a new event, then the backfill.
	m = storage.NewMigrator(db)
	if err := m.AddFromFS("core", storage.CoreMigrations, "migrations"); err != nil {
		t.Fatal(err)
	}
	if err := m.Run(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(storage.SetFacetBackfillPause(0))
	s := storage.NewSQLiteStore(db)
	s.SetFacetFunc(testFacets)
	storeAt(t, s, "new", "sentry", "", `{"host":"web-3"}`, float64(n+1))

	if err := s.BackfillFacets(ctx); err != nil {
		t.Fatal(err)
	}
	counts, _ := s.FacetCounts(ctx, event.FindOptions{}, 0)
	want := []event.FacetCount{{Value: "web-1", Count: 802}, {Value: "web-2", Count: 401}, {Value: "web-3", Count: 1}}
	if !reflect.DeepEqual(counts["host"], want) {
		t.Errorf("host counts = %v, want %v", counts["host"], want)
	}
	var cursor, cutoff int64
	if err := db.QueryRow(`SELECT cursor, cutoff FROM event_facets_backfill`).Scan(&cursor, &cutoff); err != nil {
		t.Fatal(err)
	}
	if cursor != cutoff || cutoff != n {
		t.Errorf("cursor %d, cutoff %d, want both %d", cursor, cutoff, n)
	}
	if err := s.BackfillFacets(ctx); err != nil { // done: no-op
		t.Fatal(err)
	}
}
