package storage_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"path/filepath"
	"reflect"
	"strconv"
	"testing"

	"github.com/buggregator/server/v2/internal/event"
	"github.com/buggregator/server/v2/internal/storage"
)

// testSearch indexes the "text" field of a test payload; "level" becomes the
// level facet (for the histogram).
func testSearch(_ string, payload json.RawMessage) string {
	var p struct {
		Text string `json:"text"`
	}
	_ = json.Unmarshal(payload, &p)
	return p.Text
}

func testLevel(_ string, payload json.RawMessage) []event.Facet {
	var p struct {
		Level string `json:"level"`
	}
	_ = json.Unmarshal(payload, &p)
	var s event.FacetSet
	s.Add(event.FacetLevel, p.Level)
	return s.List()
}

func setupSearchStore(t *testing.T) (*storage.SQLiteStore, *sql.DB) {
	t.Helper()
	db := setupFacetDB(t)
	if err := storage.InitSearch(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	s := storage.NewSQLiteStore(db)
	s.SetFacetFunc(testLevel)
	s.SetSearchFunc(testSearch)
	return s, db
}

func searchRows(t *testing.T, db *sql.DB) (ids, fts int) {
	t.Helper()
	if err := db.QueryRow(`SELECT COUNT(*) FROM event_search_ids`).Scan(&ids); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM event_search`).Scan(&fts); err != nil {
		t.Fatal(err)
	}
	return ids, fts
}

func TestSearch_Find(t *testing.T) {
	ctx := context.Background()
	s, _ := setupSearchStore(t)
	storeAt(t, s, "a", "monolog", "default", `{"text":"Order 42 failed: OrderFailedException user_id 7"}`, 1)
	storeAt(t, s, "b", "monolog", "default", `{"text":"Payment captured for order 43"}`, 2)
	storeAt(t, s, "c", "monolog", "other", `{"text":"order failed in another project"}`, 3)
	storeAt(t, s, "d", "monolog", "default", `{"text":"Пользователь вошёл, ok"}`, 4)
	storeAt(t, s, "e", "ray", "default", `{}`, 5) // nothing indexed
	storeAt(t, s, "q", "monolog", "default", `{"text":"he said \"hi\" twice"}`, 6)

	tests := []struct {
		q, project string
		want       []string
	}{
		{"order", "", []string{"a", "b", "c"}},   // case-insensitive
		{"FailedExc", "", []string{"a"}},         // substring inside a word
		{"order failed", "", []string{"a", "c"}}, // every word, any order
		{"order", "default", []string{"a", "b"}}, // with other conditions
		{"42", "", []string{"a"}},                // short word: LIKE
		{"user_id", "", []string{"a"}},           // _ is not a LIKE wildcard
		{"ok", "", []string{"d"}},                // short word, not a substring of others
		{"вошёл", "", []string{"d"}},             // non-ASCII
		{`"order`, "", []string{}},               // a quote is a literal character, not query syntax
		{`said "hi"`, "", []string{"q"}},
		{"nothing-like-this", "", []string{}},
	}
	for _, tt := range tests {
		t.Run(tt.q+"/"+tt.project, func(t *testing.T) {
			evs, err := s.FindAll(ctx, event.FindOptions{Search: tt.q, Project: tt.project})
			if err != nil {
				t.Fatal(err)
			}
			if got := uuids(evs); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("q=%q: %v, want %v", tt.q, got, tt.want)
			}
		})
	}
}

func TestSearch_UpdateAndDelete(t *testing.T) {
	ctx := context.Background()
	s, db := setupSearchStore(t)
	storeAt(t, s, "a", "monolog", "", `{"text":"first words"}`, 1)
	storeAt(t, s, "b", "monolog", "", `{"text":"other words"}`, 2)

	if err := s.UpdatePayload(ctx, "a", json.RawMessage(`{"text":"replaced"}`)); err != nil {
		t.Fatal(err)
	}
	for q, want := range map[string][]string{"first": {}, "replaced": {"a"}, "words": {"b"}} {
		evs, _ := s.FindAll(ctx, event.FindOptions{Search: q})
		if got := uuids(evs); !reflect.DeepEqual(got, want) {
			t.Errorf("after update q=%q: %v, want %v", q, got, want)
		}
	}

	if err := s.Delete(ctx, "a"); err != nil {
		t.Fatal(err)
	}
	if ids, fts := searchRows(t, db); ids != 1 || fts != 1 {
		t.Errorf("after delete: %d ids, %d fts rows, want 1 and 1", ids, fts)
	}
	if err := s.DeleteAll(ctx, event.DeleteOptions{}); err != nil {
		t.Fatal(err)
	}
	if ids, fts := searchRows(t, db); ids != 0 || fts != 0 {
		t.Errorf("after delete all: %d ids, %d fts rows, want 0", ids, fts)
	}
}

func TestSearch_Backfill(t *testing.T) {
	ctx := context.Background()
	db, err := storage.Open(filepath.Join(t.TempDir(), "old.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	// An old database: everything up to the facets migration.
	m := storage.NewMigrator(db)
	for _, f := range []string{"2024_01_01_000000_create_events_table.sql", "2024_01_01_000042_create_event_facets.sql"} {
		raw, _ := storage.CoreMigrations.ReadFile("migrations/" + f)
		m.AddSQL("core", f, string(raw))
	}
	if err := m.Run(); err != nil {
		t.Fatal(err)
	}
	const n = 450 // more than two batches
	tx, _ := db.Begin()
	for i := 0; i < n; i++ {
		text := "plain line"
		if i%3 == 0 {
			text = "needle line"
		}
		if _, err := tx.Exec(`INSERT INTO events (uuid, type, payload, timestamp, project) VALUES (?, 'monolog', ?, ?, '')`,
			"old-"+strconv.Itoa(i), `{"text":"`+text+`"}`, strconv.Itoa(i+1)); err != nil {
			t.Fatal(err)
		}
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}

	m = storage.NewMigrator(db)
	if err := m.AddFromFS("core", storage.CoreMigrations, "migrations"); err != nil {
		t.Fatal(err)
	}
	if err := m.Run(); err != nil {
		t.Fatal(err)
	}
	if err := storage.InitSearch(ctx, db); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(storage.SetFacetBackfillPause(0))
	s := storage.NewSQLiteStore(db)
	s.SetSearchFunc(testSearch)
	storeAt(t, s, "new", "monolog", "", `{"text":"needle new"}`, n+1)

	if err := s.BackfillSearch(ctx); err != nil {
		t.Fatal(err)
	}
	evs, err := s.FindAll(ctx, event.FindOptions{Search: "needle"})
	if err != nil {
		t.Fatal(err)
	}
	if len(evs) != n/3+1 {
		t.Errorf("needle: %d events, want %d", len(evs), n/3+1)
	}
	// Done: a second run does nothing (and writes no duplicates).
	if err := s.BackfillSearch(ctx); err != nil {
		t.Fatal(err)
	}
	if ids, fts := searchRows(t, db); ids != n+1 || fts != n+1 {
		t.Errorf("rows: %d ids, %d fts, want %d", ids, fts, n+1)
	}
}

func TestHistogram(t *testing.T) {
	ctx := context.Background()
	s, _ := setupSearchStore(t)
	// Minute 1795000020 (t=…20): two errors, one info; next minute: one warning; no level.
	base := 1795000020.0
	storeAt(t, s, "a", "monolog", "", `{"level":"ERROR","text":"boom"}`, base+1)
	storeAt(t, s, "b", "monolog", "", `{"level":"error","text":"boom again"}`, base+30)
	storeAt(t, s, "c", "monolog", "", `{"level":"info","text":"fine"}`, base+59)
	storeAt(t, s, "d", "monolog", "", `{"level":"warning","text":"hm"}`, base+61)
	storeAt(t, s, "e", "monolog", "", `{"text":"no level"}`, base+62)
	storeAt(t, s, "f", "ray", "", `{"level":"error"}`, base+2)

	got, bucket, err := s.Histogram(ctx, event.FindOptions{Type: "monolog"}, 60)
	if err != nil {
		t.Fatal(err)
	}
	want := []storage.HistogramBucket{
		{T: 1795000020, Count: 3, Levels: map[string]int{"error": 2, "info": 1}},
		{T: 1795000080, Count: 2, Levels: map[string]int{"warning": 1}},
	}
	if bucket != 60 || !reflect.DeepEqual(got, want) {
		t.Errorf("histogram = %d %+v, want 60 %+v", bucket, got, want)
	}

	// Search and filters narrow it like the feed.
	got, _, _ = s.Histogram(ctx, event.FindOptions{Type: "monolog", Search: "boom"}, 60)
	if len(got) != 1 || got[0].Count != 2 {
		t.Errorf("search boom = %+v, want one bucket of 2", got)
	}

	// A window wider than MaxHistogramBuckets buckets gets wider buckets.
	_, bucket, _ = s.Histogram(ctx, event.FindOptions{From: base, To: base + 7*86400}, 60)
	if bucket != 420 { // 7 days / 1440 = 420 s
		t.Errorf("7-day bucket = %d, want 420", bucket)
	}

	// Nothing matches: empty list.
	got, _, err = s.Histogram(ctx, event.FindOptions{Type: "smtp"}, 60)
	if err != nil || got == nil || len(got) != 0 {
		t.Errorf("empty = %v %v, want []", got, err)
	}
}
