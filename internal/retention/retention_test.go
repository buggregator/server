package retention

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"testing"
	"time"

	"github.com/buggregator/server/v2/internal/event"
	"github.com/buggregator/server/v2/internal/storage"
)

var now = time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)

func setup(t *testing.T, policy Policy) (*Cleaner, *storage.SQLiteStore, *[]string, *[]string) {
	t.Helper()
	db, err := storage.Open(":memory:")
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
	store := storage.NewSQLiteStore(db)
	var hooked, broadcast []string
	store.OnDelete(func(uuids []string) { hooked = append(hooked, uuids...) })
	c := New(db, store, policy, func(uuid, project string) { broadcast = append(broadcast, uuid+"@"+project) })
	c.now = func() time.Time { return now }
	return c, store, &hooked, &broadcast
}

// put stores an event age ago; pinned when asked.
func put(t *testing.T, s *storage.SQLiteStore, uuid string, age time.Duration, pinned bool) {
	t.Helper()
	ts := float64(now.Add(-age).UnixNano()) / 1e9
	if err := s.Store(context.Background(), event.Event{UUID: uuid, Type: "smtp", Payload: json.RawMessage(`{}`),
		Timestamp: ts, Project: "default", IsPinned: pinned}); err != nil {
		t.Fatal(err)
	}
}

func left(t *testing.T, s *storage.SQLiteStore) []string {
	t.Helper()
	all, err := s.FindAll(context.Background(), event.FindOptions{})
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, e := range all {
		out = append(out, e.UUID)
	}
	sort.Strings(out)
	return out
}

func TestMaxAge(t *testing.T) {
	c, s, hooked, broadcast := setup(t, Policy{MaxAge: 24 * time.Hour})
	put(t, s, "new", time.Hour, false)
	put(t, s, "old", 48*time.Hour, false)
	put(t, s, "old-pinned", 48*time.Hour, true)

	n, err := c.RunOnce(context.Background())
	if err != nil || n != 1 {
		t.Fatalf("RunOnce = %d, %v; want 1", n, err)
	}
	if got := fmt.Sprint(left(t, s)); got != "[new old-pinned]" {
		t.Errorf("left = %s", got)
	}
	// Through the store: the OnDelete hooks (attachments, module data) ran.
	if fmt.Sprint(*hooked) != "[old]" || fmt.Sprint(*broadcast) != "[old@default]" {
		t.Errorf("hooked = %v, broadcast = %v", *hooked, *broadcast)
	}
}

func TestMaxEvents(t *testing.T) {
	c, s, _, _ := setup(t, Policy{MaxEvents: 2})
	for i := range 5 {
		put(t, s, fmt.Sprintf("e%d", i), time.Duration(i)*time.Minute, false) // e0 newest
	}
	put(t, s, "pinned", time.Hour, true) // never deleted, not counted

	n, err := c.RunOnce(context.Background())
	if err != nil || n != 3 {
		t.Fatalf("RunOnce = %d, %v; want 3", n, err)
	}
	if got := fmt.Sprint(left(t, s)); got != "[e0 e1 pinned]" {
		t.Errorf("left = %s", got)
	}
}

// More victims than one batch.
func TestMaxEventsManyBatches(t *testing.T) {
	c, s, _, _ := setup(t, Policy{MaxEvents: 10})
	for i := range batch*2 + 10 {
		put(t, s, fmt.Sprintf("e%04d", i), time.Duration(i)*time.Second, false)
	}
	n, err := c.RunOnce(context.Background())
	if err != nil || n != batch*2 {
		t.Fatalf("RunOnce = %d, %v; want %d", n, err, batch*2)
	}
	if got := len(left(t, s)); got != 10 {
		t.Errorf("left %d, want 10", got)
	}
}

func TestPolicyOff(t *testing.T) {
	if (Policy{}).On() {
		t.Fatal("zero policy is on")
	}
	c, s, _, _ := setup(t, Policy{})
	put(t, s, "old", 1000*time.Hour, false)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	c.Run(ctx) // returns at once without touching anything
	if got := fmt.Sprint(left(t, s)); got != "[old]" {
		t.Errorf("left = %s", got)
	}
}
